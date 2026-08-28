# Week Start Environment Variable — Research Notes

**Date:** 2026-08-26
**Topic:** Making the Saturday week-start configurable via environment variable

## 1. Summary

The "Saturday week-start" is **not hardcoded in Go source code at all**. The weekday labels (`"Saturday"`, `"Sunday"`, … `"Friday"`) live exclusively in `data/program.json` as the `"label"` field on each day entry keyed by `"1"` through `"7"`. The app has **zero calendar/date logic** — no `time.Now`, no `time.Weekday`, no "today" detection. Day numbers are purely abstract sequential keys (`"1"`–`"7"`) used as map keys in `Program`, URL query parameters, and display labels. An environment variable to change the week start would need to decide between **relabeling** (keeping the same day→program mapping but changing display labels) or **rotating** (changing which program day is shown as "Day 1"). The simplest implementation adds env-var reading in `main.go`, a rotation or relabeling step before the `App` struct is built, and updates to tests.

## 2. Current State — Where "Saturday" Lives Today

### 2.1 `data/program.json` — The single source of weekday labels

Every weekday name originates here. The JSON object is keyed by string `"1"` through `"7"`, and each entry has a `"label"` field:

```
data/program.json:3   "1" → label: "Saturday"
data/program.json:73  "2" → label: "Sunday"
data/program.json:143 "3" → label: "Monday"
data/program.json:152 "4" → label: "Tuesday"
data/program.json:222 "5" → label: "Wednesday"
data/program.json:292 "6" → label: "Thursday"
data/program.json:301 "7" → label: "Friday"
```

### 2.2 `main.go` — No weekday logic

`main.go:16-24` reads only two env vars (`DATA_DIR`, `PORT`) with defaults. It loads `program.json` via `loadProgram`, constructs `App`, and registers handlers. No date/weekday processing occurs.

### 2.3 `handlers.go` — Day numbers are abstract map keys

- `handleIndex` (`handlers.go:22-38`): Iterates `i := 1; i <= 7` (`handlers.go:30`), converts to string with `strconv.Itoa(i)` (`handlers.go:31`), looks up `a.program[num]` (`handlers.go:32`). The iteration order determines the display order on the index page. The `Label` field from `program.json` is passed directly to the template.
- `handleWorkout` (`handlers.go:41-84`): Takes `?day=` query param as a string key, looks it up in `a.program`. The day number string becomes `workout.Day` which flows to `formatObsidianText` and template rendering.

### 2.4 `generate.go` — Day number is display-only

- `generateWorkout` (`generate.go:115-180`): Accepts `dayNum string`, stores it as `GeneratedWorkout.Day` (`generate.go:173`). The day number does **not** participate in exercise selection — `pickExercise` uses only `seed + slotIndex` (`generate.go:18`).
- `formatObsidianText` (`generate.go:183-257`): Renders `w.Day` and `w.Label` into `"Day %s — %s\n"` (`generate.go:186`). This is where "Day 1 — Saturday" comes from.

### 2.5 Templates — Purely display the label

- `templates/index.html:24`: `Day {{.Num}} — {{.Label}}` — renders day number + label from program.json.
- `templates/workout.html:6,47`: `Day {{.Workout.Day}} — {{.Workout.Label}}` — same pattern.
- `templates/workout.html:43-44`: Swap/permalink URLs use `.Workout.Day` as the `?day=` param value.

### 2.6 `types.go` — No weekday type

`Program` is `map[string]ProgramDay` (`types.go:3`). `ProgramDay.Label` is a plain string (`types.go:6`). No enum, no weekday type, no validation of label values.

### 2.7 What does NOT exist

- **No `time.Now` / `time.Weekday` / `time.ParseDuration`** — confirmed by grep (zero matches in all `.go` files).
- **No "today" detection** — the app never figures out what day of the week it actually is. The user manually picks a day from the index page.
- **No calendar alignment** — day `"1"` is always the first item on the index page regardless of the actual current date.

## 3. What an Environment Variable Must Touch

### 3.1 Existing env-var pattern

```
main.go:17-24
```
Pattern: `os.Getenv("VAR")` with a fallback default. A `WEEK_START` var would follow this same pattern. It would need to flow into **program transformation** before the `App` struct is constructed (`main.go:41-45`), since the program data is loaded once at startup and never reloaded (AGENTS.md: "Restart to see changes").

### 3.2 Two distinct semantic designs

The env var could mean two fundamentally different things. Each touches different layers:

#### Option A: Relabel only (display-only change)

The env var changes the **labels** but keeps the same day number → program content mapping. Day `"1"` still serves "Upper Body Hypertrophy — Push Primary" but with a different label.

**Touchpoints:**
1. `main.go` — read `WEEK_START`, compute a label offset, mutate `program[day].Label` for all 7 entries before passing to `App`.
2. `handlers.go` — no changes needed. `handleIndex` and `handleWorkout` read `Label` from `program`, which is already mutated.
3. `generate.go` — no changes needed. `formatObsidianText` reads `w.Label` from the already-mutated program.
4. `templates/index.html`, `templates/workout.html` — no changes needed.
5. **Tests** — `handlers_test.go:39` asserts `body` contains `"Saturday"`. Under relabel with a different start day, this test would need the test fixture's labels to match or the assertion to change. The test creates its own inline `Program` (not from `data/program.json`), so it would need updating if it hardcodes `"Saturday"` as day `"1"`'s label.

#### Option B: Rotate the week (reorder which program day is "Day 1")

The env var rotates which day of the program appears first. If `WEEK_START=Monday`, then the Monday workout (currently day `"3"`) becomes day `"1"` in the UI.

**Touchpoints:**
1. `main.go` — read `WEEK_START`, compute a rotation offset, **re-key the entire Program map**. `"1"` → what was `"3"`, `"2"` → what was `"4"`, etc., with wraparound. Labels from program.json stay intact (Monday still says "Monday").
2. `handlers.go` — no code changes needed if the map is re-keyed in main. The `1..7` iteration still works because the keys are still `"1"` through `"7"`.
3. `generate.go` — **determinism is affected**. `pickExercise` does not use day number, so exercise selection is unchanged. However, `formatObsidianText` renders `"Day %s — %s"` where `%s` is the day number — a user who bookmarked `?day=1&seed=abc` would now get a different workout. **This breaks existing permalinks.**
4. `templates/workout.html:43-44` — Permalink URLs embed `.Workout.Day`. Under rotation, the same URL `?day=1` points to different content. The "Re-roll All" link also uses `?day=` which would be affected.
5. **Tests** — Tests that construct inline `Program` with `"1": {Label: "Saturday", ...}` and request `?day=1` would still work because they control their own program data. But tests asserting specific exercise output for `?day=1` would break if the rotation changes which day's exercises are served.

#### Option C: Both relabel and rotate

Change both the labels AND which content is "Day 1". Most comprehensive but highest risk.

### 3.3 Concrete per-file changes (for Option B — rotation, the most likely intent)

| File | Change |
|------|--------|
| `main.go` | Add `weekStart := os.Getenv("WEEK_START")`, default `"Saturday"`. Parse into a day-of-week value, find its offset from Saturday, rotate the `Program` map keys accordingly. |
| `handlers.go` | No changes (map is pre-rotated). |
| `generate.go` | No changes. |
| `types.go` | No changes. |
| `templates/*.html` | No changes. |
| `Dockerfile` | No changes needed — env vars pass through automatically to `CMD`. No new env var declaration required in Docker (AGENTS.md: env var passthrough works without Dockerfile changes for `PORT`/`DATA_DIR` pattern). |
| `handlers_test.go` | Tests use inline `Program`, unaffected by env var. But add a test that verifies rotation behavior. |
| `generate_test.go` | Unaffected — tests use inline `ProgramDay` with arbitrary labels. |
| `types_test.go` | Unaffected. |
| `AGENTS.md` | Document the new `WEEK_START` env var in the Docker/deployment section. |

### 3.4 Validation needs

The env var value must be validated at startup (fail-fast like `loadProgram` does). A valid value needs to be one of: `"Saturday"`, `"Sunday"`, `"Monday"`, `"Tuesday"`, `"Wednesday"`, `"Thursday"`, `"Friday"` — matching the labels in `data/program.json`. Invalid value → `log.Fatalf`.

## 4. Design Decisions to Make

### D1: Relabel vs. Rotate vs. Both

| Design | What changes | User sees | Permalink stability |
|--------|-------------|-----------|---------------------|
| Relabel only | Labels change, day→content mapping stays | Day 1 still = Saturday's workout, but says "Monday" | Stable — `?day=1` same content |
| Rotate only | Day→content mapping shifts, labels from JSON stay | Day 1 = whatever day starts the week | **Breaks** — `?day=1` = different workout |
| Both | Labels + content both shift | Day 1 = week-start workout, labeled with week-start name | **Breaks** |

**Recommendation:** Rotation is the most likely user intent ("I want my week to start on Monday, so Monday's workout should be first"). But this breaks saved permalinks. The user must decide.

### D2: Env var format

| Format | Example | Pros | Cons |
|--------|---------|------|------|
| Full day name | `WEEK_START=Saturday` | Human-readable, matches program.json labels | Case sensitivity, typos |
| Abbreviation | `WEEK_START=Sat` | Shorter | Ambiguous, needs mapping table |
| Go `time.Weekday` int | `WEEK_START=6` | Machine-parseable | Saturday=6 in Go (not 0 or 1), confusing |
| 1–7 index | `WEEK_START=1` | Matches day numbering | "1" = Saturday in current data, not intuitive |

**Recommendation:** Full day name (`"Saturday"`, `"Monday"`, etc.) — matches what's already in `program.json` labels, zero new vocabulary.

### D3: Does `data/program.json` stay the source of truth for labels?

If rotating: yes, labels come from JSON and move with their content. If relabeling: the env var overrides labels, and program.json labels become unused.

**Open question:** Should the env var be the *only* source of labels (making program.json's `"label"` field ignored), or should program.json provide fallback labels?

### D4: Default when unset

Must be `"Saturday"` for backwards compatibility (current behavior).

### D5: What if program.json labels don't match the expected weekdays?

The app currently has no validation that labels are weekday names. If the user edits program.json to say `{"1": {"label": "Arm Day", ...}}`, a `WEEK_START=Saturday` env var would fail to find `"Saturday"`. The validation strategy must account for this — either validate against program.json's actual labels (find the entry with the matching label, use its position), or validate against a fixed set of 7 day names.

## 5. Constraints & Gotchas

1. **Restart required** — `data/program.json` is loaded once at startup (`main.go:26-28`). Changing `WEEK_START` requires restarting the process. (AGENTS.md: "Restart to see changes.")

2. **Templates are `go:embed`'d** — `main.go:13-14` embeds templates at compile time. No runtime template changes, but templates don't need changes for this feature.

3. **Docker** — The Dockerfile (`Dockerfile:10-12`) copies only the binary. No Dockerfile change needed for env var passthrough (same pattern as `PORT`/`DATA_DIR` which work without Dockerfile modification). Running with `docker run -e WEEK_START=Monday ...` works out of the box.

4. **Determinism** — `pickExercise` uses `sha256(seed + slotIndex)` (`generate.go:18`), independent of day number. Rotation does **not** affect which exercises are picked for a given slot. Only the URL/day number mapping changes.

5. **Permalink stability** — If rotating, existing bookmarks like `?day=1&seed=abc` will serve different content. The user must be aware of this.

6. **Tests** — All tests construct inline `Program` maps (not from `data/program.json`). The `TestIndexHandler` (`handlers_test.go:21-45`) asserts `"Saturday"` appears in the response, but uses a test fixture with `Label: "Saturday"` on day `"1"`. This test is **not affected by env var** because it doesn't go through `main()`. However, new integration-style tests for the rotation logic would be needed.

7. **Docs pipeline** — Per AGENTS.md: feature work follows design spec (`docs/specs/`) → TDD plan (`docs/plans/`) → code. This research note precedes that pipeline.

8. **No `go.mod` changes** — This feature uses only stdlib (same as current app). AGENTS.md: "go.mod has zero dependencies."

## 6. Sources

| File | Claims Supported |
|------|-----------------|
| `main.go:16-52` | Env-var pattern (PORT, DATA_DIR), startup loading, no weekday logic |
| `main.go:54-64` | `loadProgram` reads program.json once |
| `handlers.go:22-38` | `handleIndex` iterates 1..7, uses labels from program |
| `handlers.go:41-84` | `handleWorkout` uses day as map key, no date logic |
| `generate.go:14-22` | `pickExercise` seeds from `seed+slotIndex` only, not day |
| `generate.go:115-180` | `generateWorkout` passes dayNum as display-only field |
| `generate.go:186` | `formatObsidianText` renders `"Day %s — %s"` |
| `types.go:3-6` | `Program` is `map[string]ProgramDay`, `Label` is string |
| `data/program.json:1-309` | All 7 day labels ("Saturday" through "Friday"), keyed "1"–"7" |
| `templates/index.html:22-27` | Renders `Day {{.Num}} — {{.Label}}` |
| `templates/workout.html:6,43-47` | Title and nav use `{{.Workout.Day}}` and `{{.Workout.Label}}` |
| `handlers_test.go:21-45` | Test fixture with `"Saturday"` label on day `"1"`, asserts "Saturday" in body |
| `generate_test.go:340-396` | `TestFormatObsidianText` expects `"Day 1 — Saturday"` |
| `types_test.go:49-61` | Tests `ProgramDay` unmarshal with `"Saturday"` label |
| `Dockerfile:1-12` | Multi-stage build, no env var declarations needed |
| `docs/specs/2026-04-26-workout-generator-design.md:44-54` | Spec defines program.json structure with `"label": "Saturday"` |
| `AGENTS.md` | Restart-to-see-changes, Docker passthrough, docs pipeline, stdlib-only constraint |
