# Workout Generator — Design Spec

## Overview

A stateless workout-of-the-day generator running as a Go web app in a Docker container. It reads a structured training program and personal records from JSON config files, randomly selects exercises from defined pools, calculates target weights as percentages of 1RM, and renders the workout for copy-paste into Obsidian.

## Goals

- Replace an LLM prompt-based workflow with a deterministic, self-hosted app
- Generate daily workouts (strength + endurance reference) from a 7-day training cycle
- Allow exercise swapping within a slot's pool after generation
- Output Obsidian-compatible plain text for clipboard copy
- Keep it simple: Go server-side rendering, minimal JS (clipboard only)

## Non-Goals

- Logging actuals — the user logs in Obsidian
- History or persistence — the app is stateless
- Pace/HR calculations for endurance — the user's Garmin handles execution
- PR management in the app — the user edits `prs.json` directly

## Data Model

### `prs.json`

Flat map of exercise name to estimated 1RM in pounds:

```json
{
  "Incline Bench Press": 181,
  "Machine Chest Press": 227,
  "Preacher Curls": 74,
  "Tricep Pushdown": 77
}
```

- Keys must match exercise names used in `program.json`
- Values are integers (lbs)
- User maintains this file manually and checks it into the repo

### `program.json`

The 7-day program structure. Top-level is an object keyed by day number (string "1" through "7"):

```json
{
  "1": {
    "label": "Saturday",
    "name": "Upper Body Hypertrophy — Push Primary",
    "strength": [ ...slots ],
    "endurance": { ...endurance }
  }
}
```

#### Strength Slot Schema

Each slot in the `strength` array:

```json
{
  "name": "Secondary Push",
  "method": "ME",
  "sets": 3,
  "reps": { "min": 1, "max": 5 },
  "load_pct": { "min": 0.90, "max": 1.00 },
  "warmup_sets": 2,
  "choices": ["Larsen Press", "Incline Bench Press", "Close-Grip Bench Press"],
  "superset_group": null
}
```

Fields:
- `name`: Display label describing the slot's role (e.g., "Secondary Push", "Focused Pull/Arms")
- `method`: One of ME, DE, SKILL, HYP — determines rep/load ranges
- `sets`: Number of work sets
- `reps`: Min/max rep range for work sets
- `load_pct`: Min/max percentage of 1RM for target weight
- `warmup_sets`: Number of ramping warm-up sets before work sets (typically only for slot 1 ME/DE). 0 or omitted means no warm-up sets.
- `choices`: List of exercise names from this pool (must exist in `prs.json`)
- `superset_group`: String identifier to group slots into a superset (e.g., "arms"). Slots sharing the same `superset_group` are displayed together as a superset. Null means standalone.

#### Method-to-Loading Map (Reference)

| Method | Reps | Load % 1RM | RIR | Sets |
|--------|------|------------|-----|------|
| ME | 1-5 | 90-100% | None | 1-3 |
| DE | 2-4 | 70-80% | 3-4 | 4-6 |
| SKILL | 3-5 | 75-85% | 3-4 | 3-5 |
| HYP | 6-12 | ~65-80% | 0-2 | 3-4 |

These ranges are encoded into each slot's `reps` and `load_pct` fields in `program.json`. The method field is used for display (e.g., "Hypertrophy focus. Aim for 0-2 RIR.") and for determining warm-up set progression.

#### Endurance Schema

```json
{
  "type": "Sprint+MLSS",
  "description": "Combined Sprint Option 2 + MLSS+ Option 1.\n\nSprint: 2 rounds of 4x200m @ >vVO2...\n\nMLSS+: 6 rounds of 15s @130%, 45s @105%, 1min @VT1"
}
```

Fields:
- `type`: Category label (Sprint+MLSS, NT, VT1, LSD, None, Rest)
- `description`: Plain text description of the endurance workout. This is a reference display — the detailed pacing is programmed into the user's Garmin watch.

For days with no endurance component, set `type` to "None" and `description` to "No endurance component — strength training only."

## Architecture

### Endpoints

| Route | Method | Purpose |
|---|---|---|
| `GET /` | | Day picker — 7 buttons, one per day of the cycle |
| `GET /workout` | | Generate workout for a given day |

#### `/workout` Query Parameters

| Param | Required | Description |
|---|---|---|
| `day` | Yes | Day number 1-7 |
| `seed` | No | Deterministic seed for exercise selection. If omitted, a new random seed is generated. |
| `s{N}` | No | Override for slot N (0-based). Value is the exercise name. Multiple allowed (e.g., `s0=Incline+Bench+Press&s3=Preacher+Curls`). |

`seed` preserves all random selections. Slot overrides (`s0`, `s1`, etc.) are additive — swapping one exercise preserves any prior swaps already in the URL. This ensures swapping one exercise doesn't re-roll or reset the others.

### Generation Logic

1. Parse `day` param, load program day from `program.json`
2. If no `seed`, generate a random one (e.g., 8-char hex string)
3. For each strength slot:
   - If this slot has an override (`s{index}` param present), use the specified exercise
   - Otherwise, seed a PRNG with `hash(seed + slot_index)` and pick randomly from `choices`
4. For the selected exercise, look up its 1RM in `prs.json`
5. Calculate target weight: `1RM * load_pct.min` (use the lower bound as the working target, rounded to nearest 5 lbs)
6. For warm-up sets (if `warmup_sets > 0`): generate ramping sets progressing toward the work weight
7. Render via Go `html/template`

### Warm-Up Set Ramp Logic

For ME/DE exercises with `warmup_sets > 0`, generate ramp-up sets starting at ~50% of work weight and progressing linearly. Example with 2 warm-up sets targeting 163 lbs work weight:
- Warm-up 1: ~75% of work weight -> ~120 lbs, 5 reps
- Warm-up 2: ~85% of work weight -> ~140 lbs, 3 reps
- Work sets: 163 lbs, method-prescribed reps

All weights rounded to nearest 5 lbs.

### Page Layout

The `/workout` page has three zones:

1. **Header** — "Day 1 — Saturday: Upper Body Hypertrophy — Push Primary" with "Re-roll All" link (drops seed param to generate fresh) and "Back" link to day picker
2. **Workout** — Rendered strength exercises with targets, swap controls, then endurance description
3. **Output** — A read-only `<textarea>` containing the Obsidian-formatted plain text with a "Copy" button

#### Strength Exercise Display

Each exercise slot renders:
- Exercise name (bold) with the slot label (e.g., "Secondary Push — ME")
- A dropdown/select of other choices in the pool, each linking to the swap URL
- Warm-up sets (if any) with target weight and reps
- Work sets with target weight, reps, and method note (e.g., "Hypertrophy focus. Aim for 0-2 RIR.")

Superset groups are visually grouped with a "Superset" header.

#### Swap Control

Each exercise slot shows a small `<select>` element. Selecting a different exercise navigates to a URL that includes all current overrides plus the new one:
```
/workout?day=1&seed=abc123&s2=Other+Exercise&s0=Already+Swapped
```
Full page reload. Server re-renders everything with the same seed and all accumulated overrides preserved.

#### Obsidian Output Format

The `<textarea>` contains plain text matching the user's established format:

```
Day 1 — Saturday
Lift: Upper Body Hypertrophy — Push Primary

Incline Bench Press: Ramping
Warm-up Set 1
target: 75% 1RM x 5
actual:
Warm-up Set 2
target: 85% 1RM x 3
actual:
Work Set 1
target: 90% 1RM (163 lbs) x 3
actual:
Work Set 2
target: 90% 1RM (163 lbs) x 3
actual:
Work Set 3
target: 90% 1RM (163 lbs) x 3
actual:

Machine Chest Press: Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).
Work Set 1
target: 70% 1RM (159 lbs) x 10
actual:
Work Set 2
target: 70% 1RM (159 lbs) x 10
actual:
Work Set 3
target: 70% 1RM (159 lbs) x 10
actual:

Superset: Focused Push/Pull Arms
Tricep Pushdown: Hypertrophy focus. Aim for 0-2 RIR.
Work Set 1
target: 70% 1RM (54 lbs) x 12
actual:
...

Preacher Curls: Hypertrophy focus. Aim for 0-2 RIR.
Work Set 1
target: 75% 1RM (56 lbs) x 8
actual:
...

---
Endurance: NT Option 3
1x1000m @95% with 3min rest
1x800m @95% with 3min rest
4x400m @95% with 1:30 rest
4x200m @95% with 1min rest

User notes:
```

The `actual:` fields are left blank. The `User notes:` section at the bottom is an empty area for the user to fill in after copying.

### File Structure

```
uha-hyp-5k/
├── main.go              # HTTP server, handlers, generation logic
├── templates/
│   ├── index.html       # Day picker page
│   └── workout.html     # Workout display + textarea output
├── data/
│   ├── program.json     # 7-day program structure
│   └── prs.json         # Exercise PRs
├── Dockerfile           # Multi-stage build
├── go.mod
└── README.md
```

### Docker

- **Build stage:** `golang:1.22-alpine`, compile static binary
- **Runtime stage:** `alpine`, copy binary + templates
- `data/` is mounted as a volume so JSON files can be updated without rebuilding the image
- Exposed port: 8080 (configurable via env var)
- Templates are embedded in the binary via Go `embed` package — no need to copy templates separately at runtime

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o workout-generator .

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/workout-generator .
EXPOSE 8080
CMD ["./workout-generator"]
```

Run with:
```bash
docker run -p 8080:8080 -v $(pwd)/data:/app/data workout-generator
```

### CSS / Styling

Minimal, clean styling. No framework. Inline `<style>` block in the templates. The workout page should be readable on a phone (responsive, single column). The textarea should be full-width and tall enough to show the full workout without scrolling.

### JS

One function: copy the textarea contents to clipboard on button click.

```javascript
function copyWorkout() {
  const textarea = document.getElementById('workout-output');
  textarea.select();
  navigator.clipboard.writeText(textarea.value);
}
```

## Athlete Constraints Encoded in Config

These are user-specific constraints baked into `program.json`:

- **No Skull Crushers** — not included in any slot's `choices` array
- **Equipment availability** — only exercises doable with available equipment are listed in `choices`
- **Specialty bar exclusions** — exercises requiring Swiss bar, Safety bar, etc. are not included

## Edge Cases

- **Exercise not in `prs.json`**: If a selected exercise has no PR entry, display "No PR recorded — set target manually" instead of a calculated weight.
- **Single choice in pool**: If a slot has only one exercise option, no swap control is shown.
- **Invalid swap**: If an `s{N}` exercise name doesn't match any choice in that slot, ignore the override and pick randomly for that slot.
- **Day 7 (Rest)**: Show "Rest Day — no training" with no strength or endurance sections.
