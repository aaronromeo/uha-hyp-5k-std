# uha-hyp-5k

Workout generator for a hypertrophy-focused training week. Single Go binary, stdlib only, serving a small web UI: pick a day, get that day's workout with weight targets derived from your estimated 1RMs, swap exercises, and copy an Obsidian-formatted note.

## Running

```bash
go run .
```

Serves on `:8080`. Configuration is via environment variables (all applied at startup — restart to change):

| Variable    | Default            | Effect                                                                                                                              |
| ----------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| `PORT`      | `8080`             | Listen port.                                                                                                                          |
| `DATA_DIR`  | `data`             | Directory containing `program.json` and `prs.json`.                                                                                   |
| `WEEK_START`| unset              | Weekday name that day 1 carries (e.g. `Sunday`). Relabels days 1–7 so the week starts there, wrapping around. Unset: labels from `program.json` win. Must be one of the seven full weekday names; anything else aborts startup. |

Example — run a Sunday-start week (runs Sunday→Saturday, rest day lands on Saturday):

```bash
WEEK_START=Sunday go run .
```

`WEEK_START` **relabels** the week; it never reorders workouts. Day 1 always serves the same program content regardless of the week start, so saved permalinks (`/workout?day=N&seed=...`) keep working.

## What you get

- **`/`** — day picker for the week.
- **`/workout?day=N`** — generated workout: warm-up and work sets with target weight/rep ranges computed from PRs, per-slot exercise swap links, a re-roll link (new seed), and a permalink (fixed seed).
- **Obsidian output** — a textarea containing the workout as plain text, header stamped with the generation date: `Day 1 — Saturday - 20260829` (server local time).

Generation is deterministic: same `day` + `seed` + slot overrides always produce the same workout — the Obsidian header's date stamp is the only part that varies (it reflects when you generated it).

## Data files

- **`program.json`** — the training week. Keys `"1"`–`"7"`, each with a `label` (weekday display name), `name`, `strength` slots (`choices`, `sets`, `reps`, `load_pct`, `warmup_sets`, `superset_group`), and optional `endurance` block. Slot order is the workout order; `choices` is the exercise pool per slot.
- **`prs.json`** — estimated 1RMs in lbs, keyed by exercise name. Keys must **exactly** match a name in some slot's `choices` array, or that exercise gets no PR and no weight targets. Values are per-hand for dumbbell moves (never doubled).

## Docker

The image ships only the binary — mount `data/` and set `DATA_DIR`:

```bash
docker build -t workout-generator .
docker run -p 8080:8080 \
  -v "$PWD/data:/app/data" \
  -e WEEK_START=Sunday \
  workout-generator
```

Env vars pass through without image changes. The container runs UTC by default; for local-date stamps set `-e TZ=...` (the alpine base needs `tzdata` installed for named zones; numeric offsets like `TZ=UTC-5` work without it).

## Docs

- `docs/specs/` — feature design specs
- `docs/plans/` — TDD implementation plans
- `docs/research/` — research notes
- `AGENTS.md` — conventions for AI coding agents working in this repo
