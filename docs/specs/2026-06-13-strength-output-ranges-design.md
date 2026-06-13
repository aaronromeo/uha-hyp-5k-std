# Strength Workout Generator: Range Output Design

**Date:** 2026-06-13
**Status:** Approved
**Related:** `docs/specs/2026-04-26-workout-generator-design.md`

## Problem

The strength workout generator currently outputs a single weight and a single
rep count per set, taken from the lower bound of the prescribed ranges in
`data/program.json`. The program data already defines both a `Reps.Min/Max`
range and a `LoadPct.Min/Max` range, but only `Min` is propagated through the
output. Users see "165 lbs x 3" when the prescription is actually "90–100% 1RM
(165–180 lbs) x 1–5". The discarded upper bound is information the user needs
to decide how to load and rep out each set.

## Goal

Show users the full prescribed weight and rep range for each work set, in both
the HTML page and the Obsidian-format textarea.

## Non-Goals

- Changing the warmup ramp algorithm.
- Changing the `program.json` or `prs.json` schemas. Source data already has
  the ranges we need.
- Endurance workout output. The change is confined to `GeneratedSet` and the
  strength path.
- Collapse-when-equal display logic for degenerate ranges. We deliberately
  always render both bounds; if warmup display (which is always degenerate)
  becomes a problem in practice, that's a future iteration.

## Design

### Data model

Today `GeneratedSet` (in `types.go`) holds a single weight, percentage, and rep
count per set: `TargetPct`, `TargetLbs`, `TargetReps`. Rename each to its
`Min` form and add a `Max` counterpart:

```go
type GeneratedSet struct {
    Label         string
    TargetPctMin  float64   // was TargetPct
    TargetPctMax  float64   // NEW
    TargetLbsMin  int       // was TargetLbs
    TargetLbsMax  int       // NEW
    TargetRepsMin int       // was TargetReps
    TargetRepsMax int       // NEW
}
```

Renaming the existing fields is required because the bare name `TargetLbs` is
no longer accurate — it's now the lower bound of a range, not a single value.
Every reference to the old field names in `generate.go`, `templates/workout.html`,
and the tests must be updated as part of this change.

Semantics:

- **Work sets:** `Min`/`Max` come from `slot.LoadPct.Min`/`slot.LoadPct.Max`
  and `slot.Reps.Min`/`slot.Reps.Max`. Weight bounds are
  `calcTargetWeight(oneRM, pct)` at each end, rounded to the nearest 5 lbs.
- **Warmup sets:** degenerate range — `Max == Min` for all three fields. The
  warmup ramp algorithm is unchanged.
- **Missing PR:** weight `Min` and `Max` both stay 0 (unchanged); rep range is
  still populated on the struct, but renderers suppress the target per current
  behavior — the Obsidian path emits a blank `target:` line, and the HTML path
  shows an em-dash (`—`).

### Generator logic (`generate.go`)

Work-set generation at `generate.go:104-122` becomes:

```go
targetWeightMin := 0
targetWeightMax := 0
if hasPR {
    targetWeightMin = calcTargetWeight(oneRM, slot.LoadPct.Min)
    targetWeightMax = calcTargetWeight(oneRM, slot.LoadPct.Max)
}

warmupSets := generateWarmupSets(slot.WarmupSets, targetWeightMin)

workSets := make([]GeneratedSet, slot.Sets)
for j := range slot.Sets {
    workSets[j] = GeneratedSet{
        Label:         fmt.Sprintf("Work Set %d", j+1),
        TargetPctMin:  slot.LoadPct.Min,
        TargetPctMax:  slot.LoadPct.Max,
        TargetLbsMin:  targetWeightMin,
        TargetLbsMax:  targetWeightMax,
        TargetRepsMin: slot.Reps.Min,
        TargetRepsMax: slot.Reps.Max,
    }
}
```

The warmup helper (`generate.go:35-54`) is updated to populate `Max := Min` for
each new field on each warmup set. The ramp algorithm itself is untouched.
Warmups continue to be derived from `targetWeightMin` (the bottom of the
prescribed range) — they ramp toward the lighter end of the work weight, which
is the conservative choice.

### Presentation: Obsidian text (`generate.go:149-223`)

Current format (`generate.go:185, 195`):

```
target: 90% 1RM (165 lbs) x 3
```

New format — always show both bounds, even when equal:

```
target: 90–100% 1RM (165–180 lbs) x 1–5
```

A single shared helper formats a numeric range as `min–max` (en-dash, U+2013)
so warmups and work sets use the same code path. Warmups render as
`75–75% 1RM (125–125 lbs) x 5–5` — the "always range" tradeoff. If this looks
noisy in practice, the helper can grow a `collapseWhenEqual` mode later; the
data already supports it.

The missing-PR path is unchanged: still emits `target:` (blank) when `HasPR`
is false.

### Presentation: HTML template (`templates/workout.html:85-103`)

The template currently renders the per-set target as `{{.TargetLbs}} lbs x
{{.TargetReps}}`. Since those fields are being renamed to `TargetLbsMin` /
`TargetRepsMin` (and joined by `Max` counterparts), the template must be
rewritten to render the range — `165–180 lbs x 1–5` — in the same format as
the Obsidian output.

A new template helper, `formatSetTarget(set GeneratedSet) string`, is
registered in the `template.FuncMap` in `handlers.go` alongside the existing
`textareaRows` func. It produces the `165–180 lbs x 1–5` substring used inside
the template. This keeps the template thin and shares formatting logic with
the Obsidian path so the two outputs cannot drift.

Template becomes:

```html
<span class="set-target">{{if $ex.HasPR}}{{formatSetTarget .}}{{else}}—{{end}}</span>
```

The same `<span class="set-target">…</span>` block in the warmup section is
updated identically.

### Shared formatting

Both renderers use the same underlying range-formatting helper to guarantee
identical output. The Obsidian formatter produces the full
`target: 90–100% 1RM (165–180 lbs) x 1–5` line; the HTML helper produces just
the `165–180 lbs x 1–5` substring. Both call into one lower-level
`formatRange(min, max int) string` (and a float variant for percentages) so
the en-dash usage and number formatting cannot diverge.

## Testing

### Updates to existing tests (`generate_test.go`)

- `TestGenerateWarmupSets_Two` (line 82) — assert each warmup set has
  `Max == Min` for all three fields.
- `TestGenerateWorkout_Basic` (line 105) — assert work sets carry
  `slot.LoadPct.Max` and `slot.Reps.Max`, and that
  `TargetLbsMax == calcTargetWeight(oneRM, LoadPct.Max)`.
- `TestGenerateWorkout_MissingPR` (line 221) — assert
  `TargetLbsMin == 0 && TargetLbsMax == 0`, but rep range `Min`/`Max` are
  still populated from the slot.
- `TestFormatObsidianText` (line 289) — assert the new range format string
  appears (e.g. `90–100% 1RM (165–180 lbs) x 1–5`).
- `TestFormatObsidianTextMissingPR` (line 348) — unchanged behavior; assert
  the blank `target:` line is still emitted.

### New tests

- `TestFormatSetTarget` — unit test for the new helper covering: work-set
  range, degenerate range (warmup rendering as `125–125 lbs x 5–5`), and any
  edge cases.
- `TestWorkoutHandler_RendersWeightAndReps` in `handlers_test.go` — assert
  the rendered HTML body contains the expected range string for a known
  seed+day (e.g. a substring like `165–180 lbs x 1–5`). The existing handler
  tests only check that the exercise name and seed parameter appear in the
  response — this closes the coverage gap by exercising the new template
  helper end-to-end and guarding against template-field-name regressions.

## Risks

- **Display noise from degenerate ranges on warmups.** The choice to always
  render both bounds means warmups show `125–125 lbs x 5–5`. Real risk this
  looks ugly enough to want to change later. Easy to revisit by adding a
  `collapseWhenEqual` mode to the helper.
- **Template helper registration.** The new `formatSetTarget` func must be
  added to `template.FuncMap` in `handlers.go`. A missed registration causes
  a template parse error at startup, which the existing handler tests catch
  immediately.
- **En-dash character.** Source files are UTF-8 Go; no compatibility issue.
  Flagged because the renderer outputs U+2013, not a hyphen.

## Rollout

Single change, single commit. No data migration. No backward compatibility
concerns — `GeneratedSet` is an in-memory struct produced and consumed in the
same process.
