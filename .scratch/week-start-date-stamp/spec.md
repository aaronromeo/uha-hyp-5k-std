# Spec: Week-Start Configuration + Obsidian Date Stamp

Status: ready-for-agent
Date: 2026-08-27

## Problem Statement

The training week is fixed to start on Saturday — the weekday labels are baked into the program data, so shifting the week (e.g. to a Sunday start, with the rest day landing on Saturday) means hand-editing the data file. Separately, the Obsidian output header (`Day 1 — Saturday`) carries no date, so a generated workout note cannot be anchored to the day it was actually generated.

## Solution

Two changes, both opt-in and restart-time:

1. **Week start via environment variable.** A `WEEK_START` environment variable names the weekday that slot 1 carries; each following slot gets the next weekday, wrapping around the seven-day cycle. Unset, nothing changes — the labels in the program data keep winning. Workouts never move: this relabels the slots, it does not reorder the program.
2. **Generation-date stamp in the Obsidian output.** The Obsidian header line gains a compact date stamp (`YYYYMMDD`) for the moment the workout was generated: `Day 1 — Saturday - 20260829`. The HTML page is unchanged.

## User Stories

1. As a lifter, I want the week to start on a configurable weekday, so that the app's week aligns with my actual training schedule.
2. As a lifter running a Sunday-start week, I want slot 1 labeled Sunday through slot 7 labeled Saturday, so that every slot's label matches the day I train it.
3. As a lifter running a Sunday-start week, I want my rest day to land on Saturday, so that the week ends with rest.
4. As a lifter, I want the week start controlled by an environment variable, so that I can change it by restarting the process instead of editing program data.
5. As a lifter, I want an unset `WEEK_START` to preserve today's labels exactly, so that opting in is optional and the default behavior is byte-identical to the current app.
6. As a lifter, I want the workouts to stay attached to their slots when the week start changes, so that changing labels never reorders or re-keys my program.
7. As a lifter, I want `?day=N` to keep serving the same workout after a week-start change, so that my saved permalinks stay valid.
8. As a lifter, I want exercise selection to be unaffected by the week start, so that the same seed keeps producing the same picks and my reroll/override workflow is unchanged.
9. As a lifter, I want an invalid `WEEK_START` value to abort startup with a clear message, so that a typo fails loudly instead of silently mislabeling my week.
10. As a lifter, I want the Obsidian output header to end with a compact generation-date stamp (`20260829`), so that each note is anchored to the date I generated it.
11. As a lifter, I want the stamp to be the generation moment in server local time, so that generating on training day yields the training date.
12. As a lifter, I want the date stamp separated from the label by a plain hyphen, so that it is visually distinct from the em-dash in the title and easy to strip or parse.
13. As a lifter, I want the HTML page (title, header, links) unchanged by the date stamp, so that the UI stays exactly as familiar.
14. As an operator, I want `WEEK_START` to follow the same env-var pattern as the port and data-dir settings, so that configuration stays uniform.
15. As an operator deploying the Docker image, I want to set the week start with an environment flag at run time, so that no rebuild is needed and the data mount stays the source of program content.
16. As a developer, I want the Obsidian formatter to receive the generation timestamp as a parameter, so that it stays a pure function I can test with a fixed date.
17. As a developer, I want the app's clock injectable, so that handler-level tests can assert the stamped header without sleeping or flaking across midnight.
18. As a developer, I want the relabel logic as a pure function, so that I can table-test all seven starting weekdays plus the unset no-op without any I/O.

## Implementation Decisions

- **Semantics: relabel, not rotate.** `WEEK_START` names the weekday for slot 1; slot N gets the (N−1)-th following weekday, wrapping. Slot-to-workout mapping, day keys, URLs, and seed-derived exercise picks are all untouched. (The user's shift table — Monday→Tuesday, …, Sunday→Monday — defines this transform.)
- **Precedence.** `WEEK_START` unset → labels in the program data win (backwards compatible). Set → derived labels replace the program data's labels for all seven slots; the data's label field is ignored while the variable is set.
- **Validation.** The value must be one of the seven canonical full weekday names (English, exact match). Anything else aborts startup, matching the app's existing fail-fast startup posture for bad data. Abbreviations and integers are rejected.
- **Where it hooks in.** Read and validate at startup, following the existing env-var pattern; apply the relabel to the loaded program once, before the app struct is constructed. Program data loads once at startup, so a change requires a restart — same as today.
- **Date stamp source.** The stamp is the server's local time at generation, formatted compactly (`20060102`). The Obsidian formatter takes the timestamp as an explicit parameter; the app object owns an injectable "now" function — real clock in production, fixed clock in tests.
- **Stamp placement and separator.** First line of the Obsidian output becomes `Day 1 — Saturday - 20260829`: the existing em-dash between day number and label is preserved; the stamp is appended with a plain ASCII ` - ` separator. The `Lift:` line and everything after is unchanged. Obsidian output only — no HTML changes.
- **Determinism.** Same seed + same day + same generation date → identical bytes. Same seed on different dates differs only in the stamp. Exercise selection is unchanged because the picker never sees labels or dates.
- **Docker.** No image changes required; the new variable passes through exactly like the existing ones. The container runs UTC by default — an operator wanting local dates sets `TZ` at run time (documented caveat: the alpine base needs `tzdata` for named zones; numeric offsets work without it). No tz handling is implemented in this feature.

## Testing Decisions

- Tests assert external behavior — rendered bytes and startup outcomes — never internals.
- **Obsidian formatter:** existing exact-byte/contains assertions are updated to pass a fixed timestamp and expect the stamped first line. Prior art: the existing formatter test's must-contain list.
- **Handler:** a test with a fixed clock asserts the stamp appears in the rendered page's textarea; the real clock is never consulted in tests.
- **Relabel:** table-driven tests covering (a) each starting weekday producing the expected seven labels, (b) the unset case being a no-op, (c) slot order/workout content unchanged. Prior art: table style used in the generator tests.
- **Validation:** startup abort on an invalid value, following the existing bad-data startup test posture.
- Full check per repo convention: build, vet, test, and gofmt — respecting the documented pre-existing gofmt dirty baseline (no new violations).

## Out of Scope

- Rotating which workout serves which day number, or re-keying the program data.
- Date stamp in the HTML page, page titles, or URLs.
- A `?date=` query parameter for backdated notes.
- Timezone handling, tzdata installation, or any calendar/"today" detection (e.g. highlighting the current day on the picker).
- Supporting non-weekday labels (e.g. "Arm Day") — validation rejects them; accommodating them is a separate feature.
- Editing `program.json` label fields by hand — the env var replaces that workflow.
- Any change to exercise selection, PR data, or weight targets.

## Further Notes

- Research note with full current-state analysis: `docs/research/2026-08-26-week-start-env-var.md`.
- Key resolved ambiguity: "rotate vs relabel" — the user's shift table (Monday→Tuesday, …) settles on relabel; each slot keeps its workout, labels shift by one under `WEEK_START=Sunday`. Rest day consequently lands on Saturday — confirmed desired.
- Second resolved ambiguity: prose said `YYYY-MM-DD` but the example showed `20260829`; the user confirmed **compact**.
- Per repo docs pipeline, the next artifact after this spec is a TDD implementation plan in `docs/plans/`, then code.
