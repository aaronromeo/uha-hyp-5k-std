# Strength Workout Range Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show the full prescribed weight range (e.g. `165–180 lbs`) and rep range (e.g. `1–5`) for each strength workout set, in both the HTML page and the Obsidian-format textarea.

**Architecture:** Extend `GeneratedSet` from single values (`TargetPct`/`TargetLbs`/`TargetReps`) to paired bounds (`…Min`/`…Max`). Populate `Max` from the existing `slot.LoadPct.Max` and `slot.Reps.Max` in `program.json`. Warmup sets keep their existing ramp algorithm and set `Max == Min` (degenerate range). A new shared range-formatting helper is used by both the Obsidian text formatter and the HTML template, so the two outputs cannot drift.

**Tech Stack:** Go 1.25 stdlib (`html/template`, `net/http`, `embed`). No new dependencies.

**Related spec:** `docs/specs/2026-06-13-strength-output-ranges-design.md`

**Baseline assumption:** The working tree starts clean and tests pass (`go test ./...` → 26 passed). The committed `GeneratedSet` has `Label string; TargetPct float64; TargetLbs int; TargetReps int`. The HTML template references `.TargetLbs` and `.TargetReps` and renders them correctly today.

## Conventions used in this plan

- **"Find … replace with …"** blocks identify edits by giving the exact source text to locate (the *find* block) and the exact text to write in its place (the *replace* block). Match the *find* block character-for-character including whitespace. If the find block does not match, **stop and ask** — do not guess.
- **En-dash character (U+2013, `–`)** appears in user-facing range strings. In Go source it can be written as a literal en-dash inside string literals (the source files are UTF-8). If your editor or tooling is uncertain about UTF-8 round-tripping, the equivalent escape is `"\u2013"`. Either is fine; the bytes that end up on disk must be `0xE2 0x80 0x93`. Do NOT substitute a hyphen (`-`).
- **Em-dash character (U+2014, `—`)** appears in one place: the HTML template's missing-PR branch (`{{else}}—{{end}}`). It is already present in the committed template.
- **Do not commit partial work.** Commits only happen at the explicit "Commit" steps. If a task has multiple steps that leave intermediate states uncompilable, that's expected.
- **If something looks wrong, stop and ask.** Bad work is worse than no work.

---

## File Structure

**Modified files:**
- `types.go` — rename `TargetPct`/`TargetLbs`/`TargetReps` to `…Min` and add `…Max` counterparts.
- `generate.go` — populate the new Max fields in `generateWarmupSets` and `generateWorkout`; add a shared range-formatting helper; update `formatObsidianText` to render ranges.
- `handlers.go` — register a new template helper (`formatSetTarget`) that calls into the shared formatter.
- `templates/workout.html` — replace `{{.TargetLbs}} lbs x {{.TargetReps}}` with `{{formatSetTarget .}}` in two places (warmup and work-set blocks).
- `generate_test.go` — update existing tests to the new field names and assert Max values; add unit tests for the new formatting helpers.
- `handlers_test.go` — add a handler test that asserts the rendered HTML body contains the expected range string.

**New files:** none.

The formatting helpers live in `generate.go` next to `formatObsidianText` because they are part of the same presentation concern. The template helper that bridges to `html/template` lives in `handlers.go` next to the other template funcs.

---

## Task 1: Add `…Max` fields to `GeneratedSet`

**Files:**
- Modify: `types.go:62-67`

**Why first:** Every subsequent task depends on these fields existing. This task only changes the struct definition; the rest of the codebase will fail to compile until later tasks update field references.

- [ ] **Step 1: Edit the struct**

In `types.go`, find:

```go
type GeneratedSet struct {
	Label      string
	TargetPct  float64
	TargetLbs  int
	TargetReps int
}
```

Replace with:

```go
type GeneratedSet struct {
	Label         string
	TargetPctMin  float64
	TargetPctMax  float64
	TargetLbsMin  int
	TargetLbsMax  int
	TargetRepsMin int
	TargetRepsMax int
}
```

- [ ] **Step 2: Verify compile fails (expected)**

Run: `go build ./...`
Expected: build fails with errors about undefined fields `TargetPct`, `TargetLbs`, `TargetReps` in `generate.go` and `generate_test.go`. This is correct — those are fixed in the next tasks. **Do not commit yet.**

---

## Task 2: Update `generate.go` to populate Min/Max fields

**Files:**
- Modify: `generate.go` (three edits: `generateWarmupSets` body, work-set generation in `generateWorkout`, and two field references in `formatObsidianText`).

This task gets the code compiling again with the new field names. The Obsidian format string is *not* changed here — we only swap field references so the binary still builds. The actual range rendering happens in Task 4.

- [ ] **Step 1: Update `generateWarmupSets` body**

In `generate.go`, find:

```go
		sets[i] = GeneratedSet{
			Label:      fmt.Sprintf("Warm-up Set %d", i+1),
			TargetPct:  pct,
			TargetLbs:  weight,
			TargetReps: reps,
		}
```

Replace with:

```go
		sets[i] = GeneratedSet{
			Label:         fmt.Sprintf("Warm-up Set %d", i+1),
			TargetPctMin:  pct,
			TargetPctMax:  pct,
			TargetLbsMin:  weight,
			TargetLbsMax:  weight,
			TargetRepsMin: reps,
			TargetRepsMax: reps,
		}
```

Warmups are a degenerate range — `Max == Min` for all three fields. The ramp algorithm is unchanged.

- [ ] **Step 2: Update work-set generation in `generateWorkout`**

In `generate.go`, find:

```go
		// Calculate target weight using lower bound of load percentage range.
		targetWeight := 0
		if hasPR {
			targetWeight = calcTargetWeight(oneRM, slot.LoadPct.Min)
		}

		// Generate warmup sets.
		warmupSets := generateWarmupSets(slot.WarmupSets, targetWeight)

		// Generate work sets using min reps (the prescribed rep target).
		workSets := make([]GeneratedSet, slot.Sets)
		for j := range slot.Sets {
			workSets[j] = GeneratedSet{
				Label:      fmt.Sprintf("Work Set %d", j+1),
				TargetPct:  slot.LoadPct.Min,
				TargetLbs:  targetWeight,
				TargetReps: slot.Reps.Min,
			}
		}
```

Replace with:

```go
		// Calculate target weight bounds using both ends of the load percentage range.
		targetWeightMin := 0
		targetWeightMax := 0
		if hasPR {
			targetWeightMin = calcTargetWeight(oneRM, slot.LoadPct.Min)
			targetWeightMax = calcTargetWeight(oneRM, slot.LoadPct.Max)
		}

		// Generate warmup sets (ramp toward the lower bound of the work weight).
		warmupSets := generateWarmupSets(slot.WarmupSets, targetWeightMin)

		// Generate work sets carrying the full prescribed range.
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

- [ ] **Step 3: Update the Obsidian formatter's field references (format string UNCHANGED)**

There are two `b.WriteString(fmt.Sprintf("target: …", …))` calls inside `formatObsidianText` — one inside the warmup-sets `for _, s := range ex.WarmupSets` loop, one inside the work-sets `for _, s := range ex.WorkSets` loop. Both reference `s.TargetPct`, `s.TargetLbs`, `s.TargetReps`. The format string itself stays the same; only the field references change.

In `generate.go`, replace **all occurrences** of:

```go
			b.WriteString(fmt.Sprintf("target: %d%% 1RM (%d lbs) x %d\n", int(s.TargetPct*100), s.TargetLbs, s.TargetReps))
```

with:

```go
			b.WriteString(fmt.Sprintf("target: %d%% 1RM (%d lbs) x %d\n", int(s.TargetPctMin*100), s.TargetLbsMin, s.TargetRepsMin))
```

You should make this replacement twice (once for warmups, once for work sets). The two source lines are byte-identical, so a global find-and-replace is safe here.

- [ ] **Step 4: Verify build succeeds (tests will still fail)**

Run: `go build ./...`
Expected: build succeeds (zero output).

Run: `go test ./...`
Expected: test failures in `generate_test.go` referencing undefined `TargetPct`/`TargetLbs`/`TargetReps` fields. That's fixed in Task 3.

---

## Task 3: Update existing tests to the new field names

**Files:**
- Modify: `generate_test.go:82-101` (`TestGenerateWarmupSets_Two`)
- Modify: `generate_test.go:289-346` (`TestFormatObsidianText` literal `GeneratedSet` values)
- Modify: `generate_test.go:365-388` (`TestFormatObsidianTextSuperset` literal `GeneratedSet` values)

This task only renames field references inside test literals so the test file compiles. New behavior assertions (Max values, range rendering) are added in later tasks.

- [ ] **Step 1: Update `TestGenerateWarmupSets_Two`**

In `generate_test.go`, find:

```go
func TestGenerateWarmupSets_Two(t *testing.T) {
	sets := generateWarmupSets(2, 165)
	if len(sets) != 2 {
		t.Fatalf("expected 2 warmup sets, got %d", len(sets))
	}
	// set1: 75% of 165 = 123.75 → 125, 5 reps
	if sets[0].TargetLbs != 125 {
		t.Errorf("set 1 weight: expected 125, got %d", sets[0].TargetLbs)
	}
	if sets[0].TargetReps != 5 {
		t.Errorf("set 1 reps: expected 5, got %d", sets[0].TargetReps)
	}
	// set2: 87.5% of 165 = 144.375 → 145, 3 reps
	if sets[1].TargetLbs != 145 {
		t.Errorf("set 2 weight: expected 145, got %d", sets[1].TargetLbs)
	}
	if sets[1].TargetReps != 3 {
		t.Errorf("set 2 reps: expected 3, got %d", sets[1].TargetReps)
	}
}
```

Replace with:

```go
func TestGenerateWarmupSets_Two(t *testing.T) {
	sets := generateWarmupSets(2, 165)
	if len(sets) != 2 {
		t.Fatalf("expected 2 warmup sets, got %d", len(sets))
	}
	// set1: 75% of 165 = 123.75 → 125, 5 reps
	if sets[0].TargetLbsMin != 125 {
		t.Errorf("set 1 weight: expected 125, got %d", sets[0].TargetLbsMin)
	}
	if sets[0].TargetLbsMax != 125 {
		t.Errorf("set 1 weight max: expected 125 (degenerate), got %d", sets[0].TargetLbsMax)
	}
	if sets[0].TargetRepsMin != 5 {
		t.Errorf("set 1 reps: expected 5, got %d", sets[0].TargetRepsMin)
	}
	if sets[0].TargetRepsMax != 5 {
		t.Errorf("set 1 reps max: expected 5 (degenerate), got %d", sets[0].TargetRepsMax)
	}
	// set2: 87.5% of 165 = 144.375 → 145, 3 reps
	if sets[1].TargetLbsMin != 145 {
		t.Errorf("set 2 weight: expected 145, got %d", sets[1].TargetLbsMin)
	}
	if sets[1].TargetLbsMax != 145 {
		t.Errorf("set 2 weight max: expected 145 (degenerate), got %d", sets[1].TargetLbsMax)
	}
	if sets[1].TargetRepsMin != 3 {
		t.Errorf("set 2 reps: expected 3, got %d", sets[1].TargetRepsMin)
	}
	if sets[1].TargetRepsMax != 3 {
		t.Errorf("set 2 reps max: expected 3 (degenerate), got %d", sets[1].TargetRepsMax)
	}
}
```

- [ ] **Step 2: Update `GeneratedSet` literals in the first exercise of `TestFormatObsidianText`**

In `generate_test.go`, find (inside `TestFormatObsidianText`, the first exercise — note the `Incline Bench Press` exercise name above):

```go
				WarmupSets: []GeneratedSet{
					{Label: "Warm-up Set 1", TargetPct: 0.75, TargetLbs: 125, TargetReps: 5},
					{Label: "Warm-up Set 2", TargetPct: 0.875, TargetLbs: 145, TargetReps: 3},
				},
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
					{Label: "Work Set 2", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
					{Label: "Work Set 3", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
				},
```

Replace with:

```go
				WarmupSets: []GeneratedSet{
					{Label: "Warm-up Set 1", TargetPctMin: 0.75, TargetPctMax: 0.75, TargetLbsMin: 125, TargetLbsMax: 125, TargetRepsMin: 5, TargetRepsMax: 5},
					{Label: "Warm-up Set 2", TargetPctMin: 0.875, TargetPctMax: 0.875, TargetLbsMin: 145, TargetLbsMax: 145, TargetRepsMin: 3, TargetRepsMax: 3},
				},
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPctMin: 0.90, TargetPctMax: 1.00, TargetLbsMin: 165, TargetLbsMax: 180, TargetRepsMin: 1, TargetRepsMax: 5},
					{Label: "Work Set 2", TargetPctMin: 0.90, TargetPctMax: 1.00, TargetLbsMin: 165, TargetLbsMax: 180, TargetRepsMin: 1, TargetRepsMax: 5},
					{Label: "Work Set 3", TargetPctMin: 0.90, TargetPctMax: 1.00, TargetLbsMin: 165, TargetLbsMax: 180, TargetRepsMin: 1, TargetRepsMax: 5},
				},
```

- [ ] **Step 3: Update `GeneratedSet` literals in the second exercise of `TestFormatObsidianText`**

In `generate_test.go`, find (inside `TestFormatObsidianText`, the second exercise — `Machine Chest Press`):

```go
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
					{Label: "Work Set 2", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
					{Label: "Work Set 3", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
				},
```

Replace with:

```go
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPctMin: 0.70, TargetPctMax: 0.80, TargetLbsMin: 160, TargetLbsMax: 180, TargetRepsMin: 6, TargetRepsMax: 12},
					{Label: "Work Set 2", TargetPctMin: 0.70, TargetPctMax: 0.80, TargetLbsMin: 160, TargetLbsMax: 180, TargetRepsMin: 6, TargetRepsMax: 12},
					{Label: "Work Set 3", TargetPctMin: 0.70, TargetPctMax: 0.80, TargetLbsMin: 160, TargetLbsMax: 180, TargetRepsMin: 6, TargetRepsMax: 12},
				},
```

**Important:** the `mustContain` list inside `TestFormatObsidianText` still asserts the *old* format (`target: 75% 1RM (125 lbs) x 5`, etc.) — leave it alone for now. Because Task 2 Step 3 kept the format string identical, this test continues to pass as long as the literal Min values match the old single-value expectations. The `mustContain` list is updated to the new range format in Task 4 Step 10.

- [ ] **Step 4: Update `GeneratedSet` literals in `TestFormatObsidianTextSuperset`**

In `generate_test.go`, find (inside `TestFormatObsidianTextSuperset`, the `Tricep Pushdown` exercise):

```go
				WorkSets: []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.70, TargetLbs: 55, TargetReps: 12}},
```

Replace with:

```go
				WorkSets:   []GeneratedSet{{Label: "Work Set 1", TargetPctMin: 0.70, TargetPctMax: 0.75, TargetLbsMin: 55, TargetLbsMax: 60, TargetRepsMin: 8, TargetRepsMax: 12}},
```

In `generate_test.go`, find (inside `TestFormatObsidianTextSuperset`, the `Preacher Curls` exercise):

```go
				WorkSets: []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.75, TargetLbs: 55, TargetReps: 8}},
```

Replace with:

```go
				WorkSets:   []GeneratedSet{{Label: "Work Set 1", TargetPctMin: 0.75, TargetPctMax: 0.80, TargetLbsMin: 55, TargetLbsMax: 60, TargetRepsMin: 6, TargetRepsMax: 8}},
```

- [ ] **Step 5: Verify build + tests**

Run: `go build ./...`
Expected: success.

Run: `go test ./...`
Expected: all 26 tests pass. The Obsidian formatter still produces the old single-value format using only the Min fields, which is what `TestFormatObsidianText` currently asserts.

- [ ] **Step 6: Commit**

```bash
git add types.go generate.go generate_test.go
git commit -m "Add Min/Max fields to GeneratedSet, plumb through generator

Renames TargetPct/TargetLbs/TargetReps to TargetPctMin/TargetLbsMin/
TargetRepsMin and adds TargetPctMax/TargetLbsMax/TargetRepsMax
counterparts. Work sets carry the full slot.LoadPct and slot.Reps
ranges; warmup sets use a degenerate range (Max == Min). Output
rendering is unchanged in this commit — it will switch to ranges in
follow-up commits."
```

---

## Task 4: Add range-formatting helpers and update Obsidian text output

**Files:**
- Modify: `generate.go` — add two helper funcs near the top of the file (just below `calcTargetWeight`) and update `formatObsidianText` to use a new full-line formatter.

The helpers exist so the Obsidian path and the HTML template produce byte-identical range strings. Both renderers will end up calling `formatLbsRepsRange` (the partial substring like `165–180 lbs x 1–5`). The Obsidian formatter adds the `target: 90–100% 1RM (…)` wrapper around it.

- [ ] **Step 1: Write the failing test for the integer-range helper**

Append to `generate_test.go`:

```go
// ---- formatIntRange / formatFloatPctRange ----

func TestFormatIntRange(t *testing.T) {
	cases := []struct {
		min, max int
		want     string
	}{
		{165, 180, "165–180"},
		{125, 125, "125–125"},
		{0, 0, "0–0"},
		{1, 5, "1–5"},
	}
	for _, c := range cases {
		got := formatIntRange(c.min, c.max)
		if got != c.want {
			t.Errorf("formatIntRange(%d, %d) = %q, want %q", c.min, c.max, got, c.want)
		}
	}
}

func TestFormatFloatPctRange(t *testing.T) {
	cases := []struct {
		min, max float64
		want     string
	}{
		{0.90, 1.00, "90–100"},
		{0.75, 0.75, "75–75"},
		{0.65, 0.80, "65–80"},
	}
	for _, c := range cases {
		got := formatFloatPctRange(c.min, c.max)
		if got != c.want {
			t.Errorf("formatFloatPctRange(%v, %v) = %q, want %q", c.min, c.max, got, c.want)
		}
	}
}
```

The string literal `"165–180"` uses the en-dash U+2013. When writing the test file, ensure the editor preserves UTF-8 and does not substitute a hyphen.

- [ ] **Step 2: Run the failing tests**

Run: `go test ./... -run 'TestFormatIntRange|TestFormatFloatPctRange'`
Expected: FAIL with "undefined: formatIntRange" and "undefined: formatFloatPctRange".

- [ ] **Step 3: Implement the helpers**

Insert these two functions into `generate.go`, immediately after `calcTargetWeight` (around line 33):

```go
// formatIntRange formats a pair of integers as "min–max" using an en-dash
// (U+2013). The range is always rendered with both bounds, even when equal.
func formatIntRange(minVal, maxVal int) string {
	return fmt.Sprintf("%d–%d", minVal, maxVal)
}

// formatFloatPctRange formats a pair of fractional percentages (0.90, 1.00) as
// their integer-percent range "90–100" using an en-dash. The trailing "%" is
// added by callers so they can place it adjacent to "1RM".
func formatFloatPctRange(minVal, maxVal float64) string {
	return fmt.Sprintf("%d–%d", int(minVal*100), int(maxVal*100))
}
```

- [ ] **Step 4: Run the helper tests to verify they pass**

Run: `go test ./... -run 'TestFormatIntRange|TestFormatFloatPctRange' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for the combined target-line helper**

Append to `generate_test.go`:

```go
// ---- formatSetTargetLine ----

func TestFormatSetTargetLine_WorkSet(t *testing.T) {
	s := GeneratedSet{
		TargetPctMin: 0.90, TargetPctMax: 1.00,
		TargetLbsMin: 165, TargetLbsMax: 180,
		TargetRepsMin: 1, TargetRepsMax: 5,
	}
	got := formatSetTargetLine(s)
	want := "90–100% 1RM (165–180 lbs) x 1–5"
	if got != want {
		t.Errorf("formatSetTargetLine = %q, want %q", got, want)
	}
}

func TestFormatSetTargetLine_DegenerateWarmup(t *testing.T) {
	s := GeneratedSet{
		TargetPctMin: 0.75, TargetPctMax: 0.75,
		TargetLbsMin: 125, TargetLbsMax: 125,
		TargetRepsMin: 5, TargetRepsMax: 5,
	}
	got := formatSetTargetLine(s)
	want := "75–75% 1RM (125–125 lbs) x 5–5"
	if got != want {
		t.Errorf("formatSetTargetLine = %q, want %q", got, want)
	}
}
```

- [ ] **Step 6: Run the failing test**

Run: `go test ./... -run 'TestFormatSetTargetLine'`
Expected: FAIL with "undefined: formatSetTargetLine".

- [ ] **Step 7: Implement `formatSetTargetLine` and a helper for the lbs-x-reps substring**

Insert into `generate.go`, just after `formatFloatPctRange`:

```go
// formatLbsRepsRange formats the weight-range and rep-range substring shared
// by the HTML template and the Obsidian text output: e.g. "165–180 lbs x 1–5".
func formatLbsRepsRange(s GeneratedSet) string {
	return fmt.Sprintf("%s lbs x %s",
		formatIntRange(s.TargetLbsMin, s.TargetLbsMax),
		formatIntRange(s.TargetRepsMin, s.TargetRepsMax))
}

// formatSetTargetLine formats the full per-set target line for the Obsidian
// output: "90–100% 1RM (165–180 lbs) x 1–5".
func formatSetTargetLine(s GeneratedSet) string {
	return fmt.Sprintf("%s%% 1RM (%s)",
		formatFloatPctRange(s.TargetPctMin, s.TargetPctMax),
		formatLbsRepsRange(s))
}
```

- [ ] **Step 8: Run the new tests**

Run: `go test ./... -run 'TestFormatSetTargetLine|TestFormatIntRange|TestFormatFloatPctRange' -v`
Expected: PASS.

- [ ] **Step 9: Update `formatObsidianText` to use `formatSetTargetLine`**

In `generate.go`, replace **all occurrences** of:

```go
			b.WriteString(fmt.Sprintf("target: %d%% 1RM (%d lbs) x %d\n", int(s.TargetPctMin*100), s.TargetLbsMin, s.TargetRepsMin))
```

with:

```go
			b.WriteString(fmt.Sprintf("target: %s\n", formatSetTargetLine(s)))
```

There are two matching occurrences (one in the warmup-sets loop, one in the work-sets loop). Both lines are byte-identical, so a global replacement is safe. After the edit, verify the file contains exactly two calls to `formatSetTargetLine(s)`.

- [ ] **Step 10: Update `TestFormatObsidianText` `mustContain` strings**

In `generate_test.go`, find:

```go
	mustContain := []string{
		"Day 1 — Saturday",
		"Lift: Upper Body Hypertrophy — Push Primary",
		"Incline Bench Press: Ramping",
		"Warm-up Set 1",
		"target: 75% 1RM (125 lbs) x 5",
		"actual:",
		"Work Set 1",
		"target: 90% 1RM (165 lbs) x 3",
		"Machine Chest Press: Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
		"target: 70% 1RM (160 lbs) x 10",
		"---",
		"Endurance: Sprint+MLSS",
		"Sprint + MLSS combo workout",
		"User notes:",
	}
```

Replace with:

```go
	mustContain := []string{
		"Day 1 — Saturday",
		"Lift: Upper Body Hypertrophy — Push Primary",
		"Incline Bench Press: Ramping",
		"Warm-up Set 1",
		"target: 75–75% 1RM (125–125 lbs) x 5–5",
		"actual:",
		"Work Set 1",
		"target: 90–100% 1RM (165–180 lbs) x 1–5",
		"Machine Chest Press: Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
		"target: 70–80% 1RM (160–180 lbs) x 6–12",
		"---",
		"Endurance: Sprint+MLSS",
		"Sprint + MLSS combo workout",
		"User notes:",
	}
```

The three changed strings are the `target: …` lines. Note that the `—` separators in `Day 1 — Saturday`, `Upper Body Hypertrophy — Push Primary`, and `Hypertrophy focus` are em-dashes (U+2014) and are unchanged. Only the three `target:` lines use the en-dash (U+2013) for ranges.

- [ ] **Step 11: Run all tests to verify**

Run: `go test ./...`
Expected: PASS. The total test count is now 26 (baseline) + 2 (`TestFormatIntRange`, `TestFormatFloatPctRange`) + 2 (`TestFormatSetTargetLine_WorkSet`, `TestFormatSetTargetLine_DegenerateWarmup`) = 30. If you see a different count, something is off — stop and report.

- [ ] **Step 12: Commit**

```bash
git add generate.go generate_test.go
git commit -m "Render strength workout targets as ranges in Obsidian output

Adds shared range-formatting helpers (formatIntRange,
formatFloatPctRange, formatLbsRepsRange, formatSetTargetLine) used by
the Obsidian text formatter. Output goes from 'target: 90% 1RM (165
lbs) x 3' to 'target: 90–100% 1RM (165–180 lbs) x 1–5'. Warmup sets
render as degenerate ranges (e.g. '125–125 lbs x 5–5')."
```

---

## Task 5: Register `formatSetTarget` template helper and update HTML template

**Files:**
- Modify: `handlers.go:94-126` (`templateFuncs`)
- Modify: `templates/workout.html:90, 100` (two `set-target` `<span>` lines)
- Modify: `handlers_test.go` — add a handler test asserting the range string appears in rendered HTML.

The HTML helper produces the partial substring (`165–180 lbs x 1–5`) so it can plug directly into the template's existing `<span class="set-target">` structure without disturbing the surrounding markup.

- [ ] **Step 1: Write the failing handler test**

Append to `handlers_test.go`:

```go
func TestWorkoutHandlerRendersRange(t *testing.T) {
	app := newTestApp(
		Program{
			"1": ProgramDay{
				Label: "Saturday",
				Name:  "Upper Push",
				Strength: []Slot{
					{
						Name:       "Secondary Push",
						Method:     "ME",
						Sets:       1,
						Reps:       Range{Min: 1, Max: 5},
						LoadPct:    RangeF{Min: 0.90, Max: 1.00},
						WarmupSets: 1,
						Choices:    []string{"Incline Bench Press"},
					},
				},
			},
		},
		PRs{"Incline Bench Press": 181},
	)

	req := httptest.NewRequest("GET", "/workout?day=1&seed=fixed", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()

	// Work set should render the prescribed range:
	//   LoadPct 0.90–1.00 × 181 lbs → 165–180 lbs
	//   Reps 1–5
	if !containsStr(body, "165–180 lbs x 1–5") {
		t.Errorf("expected '165–180 lbs x 1–5' in HTML body\n\nBody:\n%s", body)
	}
}
```

The string `"165–180 lbs x 1–5"` uses en-dashes (U+2013); preserve them.

- [ ] **Step 2: Run the failing test**

Run: `go test ./... -run TestWorkoutHandlerRendersRange -v`
Expected: FAIL. After Tasks 1-3, the struct fields are `TargetLbsMin`/`TargetRepsMin` etc., but `templates/workout.html` still references `.TargetLbs`/`.TargetReps`. Go's `html/template` returns an execution error for missing struct fields, so `handleWorkout` will write an error to the response. The test will fail with either a non-200 status code, an empty body, or a body that does not contain `165–180 lbs x 1–5`. Any of these failures is correct — they all confirm the test exercises the right code path.

- [ ] **Step 3: Add `formatSetTarget` to the template FuncMap**

In `handlers.go`, find:

```go
		"supersetNames": func(exercises []GeneratedExercise, group string) string {
			names := supersetSlotNames(exercises, group)
			return strings.Join(names, "/")
		},
		"textareaRows": func(text string) int {
```

Replace with:

```go
		"supersetNames": func(exercises []GeneratedExercise, group string) string {
			names := supersetSlotNames(exercises, group)
			return strings.Join(names, "/")
		},
		"formatSetTarget": func(s GeneratedSet) string {
			return formatLbsRepsRange(s)
		},
		"textareaRows": func(text string) int {
```

After this edit, `templateFuncs()` should return a map with five entries: `swapURL`, `deref`, `supersetNames`, `formatSetTarget`, `textareaRows`.

- [ ] **Step 4: Update the HTML template**

There are two edits to make in `templates/workout.html`.

**Edit 1 (warmup-sets block):** Find:

```html
          <span class="set-target">{{.TargetLbs}} lbs x {{.TargetReps}}</span>
```

Replace with:

```html
          <span class="set-target">{{formatSetTarget .}}</span>
```

**Edit 2 (work-sets block):** Find:

```html
          <span class="set-target">{{if $ex.HasPR}}{{.TargetLbs}} lbs x {{.TargetReps}}{{else}}—{{end}}</span>
```

Replace with:

```html
          <span class="set-target">{{if $ex.HasPR}}{{formatSetTarget .}}{{else}}—{{end}}</span>
```

The em-dash (`—`, U+2014) in the missing-PR branch is preserved by the replacement.

After both edits, the template should contain exactly two calls to `{{formatSetTarget .}}` and no remaining references to `.TargetLbs` or `.TargetReps`.

- [ ] **Step 5: Run the handler test**

Run: `go test ./... -run TestWorkoutHandlerRendersRange -v`
Expected: PASS.

- [ ] **Step 6: Run the full test suite**

Run: `go test ./...`
Expected: PASS — all existing handler tests still pass (they only check for exercise names and seed presence, which aren't affected by this change).

- [ ] **Step 7: Manual smoke test (optional but recommended)**

Run: `go run . &` (or start the server however you normally do)
Visit: `http://localhost:8080/workout?day=1&seed=smoke`
Expected:
- Warmup sets show something like `125–125 lbs x 5–5` (degenerate range).
- Work sets show something like `165–180 lbs x 1–5` (full range).
- No `<no value>` strings anywhere.

Stop the server when done.

- [ ] **Step 8: Commit**

```bash
git add handlers.go handlers_test.go templates/workout.html
git commit -m "Render strength workout targets as ranges in HTML page

Registers a formatSetTarget template helper that shares the
range-formatting logic with the Obsidian text formatter, so the two
outputs cannot drift. The HTML template now renders work and warmup
sets as e.g. '165–180 lbs x 1–5'. Adds a handler test that asserts
the range string appears in the rendered HTML body."
```

---

## Task 6: Verify and wrap up

- [ ] **Step 1: Full test run**

Run: `go test ./... -v`
Expected: all 31 tests pass — 26 original plus 5 new (`TestFormatIntRange`, `TestFormatFloatPctRange`, `TestFormatSetTargetLine_WorkSet`, `TestFormatSetTargetLine_DegenerateWarmup`, `TestWorkoutHandlerRendersRange`).

- [ ] **Step 2: Vet**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Build production binary**

Run: `go build ./...`
Expected: no output.

- [ ] **Step 4: Final manual check**

Start the server, visit `/`, click into each strength day (1, 2, 4, 5), and confirm:
- Every set shows a range, not a single value.
- Warmup sets render as degenerate ranges (e.g. `125–125 lbs x 5–5`).
- Work sets render the full prescribed range.
- The "Obsidian Output" textarea contains lines like `target: 90–100% 1RM (165–180 lbs) x 1–5`.
- No `<no value>` strings anywhere in the rendered HTML.

- [ ] **Step 5: Confirm clean git state**

Run: `git status`
Expected: working tree clean, branch ahead 3 commits (Task 3, Task 4, Task 5 commits — plus the previously committed spec).

---

## Notes for the executor

- **En-dash character (U+2013)**: used throughout for the range separator. Most editors handle UTF-8 transparently; just don't paste through anything that substitutes a hyphen.
- **Em-dash character (U+2014)**: only appears once, in the missing-PR HTML branch (`{{else}}—{{end}}`). Already present in the committed template.
- **Warmup degenerate range**: by design — the spec accepts the noisier `125–125 lbs x 5–5` display in exchange for shared formatting logic. Do not add a collapse-when-equal special case.
- **No PR behavior**: unchanged. Obsidian emits a blank `target:` line. HTML emits `—`. Both paths short-circuit before calling `formatSetTarget` / `formatSetTargetLine`, so the struct's zero-valued `Target*` fields are never rendered.
- **Test failures from in-progress tasks**: expected and intentional. The plan only commits at the boundaries marked "Step N: Commit", so partial states never land on `main`.
