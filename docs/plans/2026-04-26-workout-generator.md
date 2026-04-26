# Workout Generator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a stateless Go web app that generates randomized daily workouts from JSON config files and outputs Obsidian-compatible text for clipboard copy.

**Architecture:** Single Go binary using `html/template` for server-side rendering. Two JSON config files (`prs.json` for personal records, `program.json` for the 7-day program structure) mounted as a Docker volume. Exercise selection is seeded for deterministic replay; swaps are additive URL params.

**Tech Stack:** Go 1.22+, `html/template`, `net/http`, `embed`, Docker

**Spec:** `docs/specs/2026-04-26-workout-generator-design.md`

---

## File Map

| File | Responsibility |
|------|---------------|
| `main.go` | HTTP server setup, routing, config loading |
| `generate.go` | Workout generation logic: exercise selection, weight calculation, warmup ramp, text formatting |
| `generate_test.go` | Tests for generation logic |
| `handlers.go` | HTTP handlers for `/` and `/workout` |
| `handlers_test.go` | Tests for HTTP handlers |
| `types.go` | Struct definitions for program, slots, PRs, generated workout |
| `templates/index.html` | Day picker page |
| `templates/workout.html` | Workout display + textarea output |
| `data/prs.json` | Personal records (user-maintained) |
| `data/program.json` | 7-day program structure (user-maintained) |
| `Dockerfile` | Multi-stage build |
| `go.mod` | Go module definition |

---

### Task 1: Project Scaffolding and Types

**Files:**
- Create: `go.mod`
- Create: `types.go`
- Create: `types_test.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
go mod init github.com/aaron-romeo/uha-hyp-5k
```

Expected: `go.mod` created with module path.

- [ ] **Step 2: Write type definitions**

Create `types.go` with all struct types that mirror the JSON schemas from the spec:

```go
package main

// Program represents the full 7-day training program.
type Program map[string]ProgramDay

// ProgramDay represents one day in the program.
type ProgramDay struct {
	Label    string        `json:"label"`
	Name     string        `json:"name"`
	Strength []Slot        `json:"strength"`
	Endurance *Endurance   `json:"endurance,omitempty"`
}

// Slot represents one exercise slot in a day's strength workout.
type Slot struct {
	Name          string   `json:"name"`
	Method        string   `json:"method"`
	Sets          int      `json:"sets"`
	Reps          Range    `json:"reps"`
	LoadPct       RangeF   `json:"load_pct"`
	WarmupSets    int      `json:"warmup_sets"`
	Choices       []string `json:"choices"`
	SupersetGroup *string  `json:"superset_group"`
}

// Range represents an integer min/max range.
type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// RangeF represents a float64 min/max range (for percentages).
type RangeF struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// Endurance represents the endurance component of a day.
type Endurance struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// PRs is a map of exercise name to estimated 1RM in pounds.
type PRs map[string]int

// GeneratedWorkout is the full output for one day.
type GeneratedWorkout struct {
	Day       string
	Label     string
	Name      string
	Seed      string
	Exercises []GeneratedExercise
	Endurance *Endurance
}

// GeneratedExercise is one exercise slot after selection and weight calculation.
type GeneratedExercise struct {
	SlotIndex     int
	SlotName      string
	Method        string
	ExerciseName  string
	Choices       []string
	SupersetGroup *string
	WarmupSets    []GeneratedSet
	WorkSets      []GeneratedSet
	HasPR         bool
	MethodNote    string
}

// GeneratedSet is one set (warmup or work) with target info.
type GeneratedSet struct {
	Label      string
	TargetPct  float64
	TargetLbs  int
	TargetReps int
}
```

- [ ] **Step 3: Write test to verify JSON round-trip**

Create `types_test.go`:

```go
package main

import (
	"encoding/json"
	"testing"
)

func TestSlotUnmarshal(t *testing.T) {
	raw := `{
		"name": "Secondary Push",
		"method": "ME",
		"sets": 3,
		"reps": {"min": 1, "max": 5},
		"load_pct": {"min": 0.90, "max": 1.00},
		"warmup_sets": 2,
		"choices": ["Larsen Press", "Incline Bench Press"],
		"superset_group": null
	}`
	var s Slot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.Name != "Secondary Push" {
		t.Errorf("expected name 'Secondary Push', got %q", s.Name)
	}
	if s.Method != "ME" {
		t.Errorf("expected method 'ME', got %q", s.Method)
	}
	if s.Sets != 3 {
		t.Errorf("expected sets 3, got %d", s.Sets)
	}
	if s.Reps.Min != 1 || s.Reps.Max != 5 {
		t.Errorf("expected reps 1-5, got %d-%d", s.Reps.Min, s.Reps.Max)
	}
	if s.LoadPct.Min != 0.90 || s.LoadPct.Max != 1.00 {
		t.Errorf("expected load_pct 0.90-1.00, got %f-%f", s.LoadPct.Min, s.LoadPct.Max)
	}
	if s.WarmupSets != 2 {
		t.Errorf("expected warmup_sets 2, got %d", s.WarmupSets)
	}
	if len(s.Choices) != 2 {
		t.Errorf("expected 2 choices, got %d", len(s.Choices))
	}
	if s.SupersetGroup != nil {
		t.Errorf("expected nil superset_group, got %v", s.SupersetGroup)
	}
}

func TestSlotUnmarshalWithSuperset(t *testing.T) {
	raw := `{
		"name": "Focused Arms",
		"method": "HYP",
		"sets": 3,
		"reps": {"min": 6, "max": 12},
		"load_pct": {"min": 0.65, "max": 0.80},
		"warmup_sets": 0,
		"choices": ["Tricep Pushdown"],
		"superset_group": "arms"
	}`
	var s Slot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if s.SupersetGroup == nil || *s.SupersetGroup != "arms" {
		t.Errorf("expected superset_group 'arms', got %v", s.SupersetGroup)
	}
}

func TestProgramDayUnmarshal(t *testing.T) {
	raw := `{
		"label": "Saturday",
		"name": "Upper Body Hypertrophy",
		"strength": [
			{
				"name": "Secondary Push",
				"method": "ME",
				"sets": 3,
				"reps": {"min": 1, "max": 5},
				"load_pct": {"min": 0.90, "max": 1.00},
				"warmup_sets": 2,
				"choices": ["Incline Bench Press"],
				"superset_group": null
			}
		],
		"endurance": {
			"type": "Sprint+MLSS",
			"description": "Sprint + MLSS combo"
		}
	}`
	var day ProgramDay
	if err := json.Unmarshal([]byte(raw), &day); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if day.Label != "Saturday" {
		t.Errorf("expected label 'Saturday', got %q", day.Label)
	}
	if len(day.Strength) != 1 {
		t.Errorf("expected 1 strength slot, got %d", len(day.Strength))
	}
	if day.Endurance == nil || day.Endurance.Type != "Sprint+MLSS" {
		t.Errorf("expected endurance type 'Sprint+MLSS', got %v", day.Endurance)
	}
}

func TestPRsUnmarshal(t *testing.T) {
	raw := `{"Incline Bench Press": 181, "Machine Chest Press": 227}`
	var prs PRs
	if err := json.Unmarshal([]byte(raw), &prs); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if prs["Incline Bench Press"] != 181 {
		t.Errorf("expected 181, got %d", prs["Incline Bench Press"])
	}
	if prs["Machine Chest Press"] != 227 {
		t.Errorf("expected 227, got %d", prs["Machine Chest Press"])
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run TestSlot -run TestProgram -run TestPRs
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git init
git add go.mod types.go types_test.go
git commit -m "feat: project scaffolding with type definitions and JSON tests"
```

---

### Task 2: Generation Logic

**Files:**
- Create: `generate.go`
- Create: `generate_test.go`

- [ ] **Step 1: Write test for exercise selection with seed determinism**

Create `generate_test.go`:

```go
package main

import (
	"testing"
)

func TestPickExerciseDeterministic(t *testing.T) {
	choices := []string{"Larsen Press", "Incline Bench Press", "Close-Grip Bench Press"}
	seed := "abc123"

	pick1 := pickExercise(seed, 0, choices)
	pick2 := pickExercise(seed, 0, choices)
	if pick1 != pick2 {
		t.Errorf("same seed+slot should produce same pick: got %q and %q", pick1, pick2)
	}

	pick3 := pickExercise(seed, 1, choices)
	// Different slot index may produce different pick (not guaranteed, but seed changes input)
	_ = pick3 // just verify it doesn't panic

	pick4 := pickExercise("other-seed", 0, choices)
	// Different seed may produce different pick
	_ = pick4
}

func TestPickExerciseSingleChoice(t *testing.T) {
	choices := []string{"Leg Press"}
	pick := pickExercise("any-seed", 0, choices)
	if pick != "Leg Press" {
		t.Errorf("single choice should always return that choice, got %q", pick)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run TestPickExercise
```

Expected: FAIL — `pickExercise` not defined.

- [ ] **Step 3: Implement pickExercise**

Create `generate.go`:

```go
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
)

// pickExercise deterministically selects an exercise from choices using seed + slot index.
func pickExercise(seed string, slotIndex int, choices []string) string {
	if len(choices) == 1 {
		return choices[0]
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", seed, slotIndex)))
	s := binary.BigEndian.Uint64(h[:8])
	r := rand.New(rand.NewPCG(s, 0))
	return choices[r.IntN(len(choices))]
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run TestPickExercise
```

Expected: PASS.

- [ ] **Step 5: Write test for weight calculation**

Add to `generate_test.go`:

```go
func TestRoundToNearest5(t *testing.T) {
	cases := []struct {
		input    float64
		expected int
	}{
		{163.0, 165},
		{162.0, 160},
		{162.5, 165},
		{160.0, 160},
		{157.5, 160},
		{0.0, 0},
	}
	for _, tc := range cases {
		got := roundToNearest5(tc.input)
		if got != tc.expected {
			t.Errorf("roundToNearest5(%f) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestCalcTargetWeight(t *testing.T) {
	// 1RM = 181, load_pct.min = 0.90 -> 181 * 0.90 = 162.9 -> round to 165
	got := calcTargetWeight(181, 0.90)
	if got != 165 {
		t.Errorf("expected 165, got %d", got)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestRound|TestCalcTarget"
```

Expected: FAIL — functions not defined.

- [ ] **Step 7: Implement weight calculation**

Add to `generate.go`:

```go
// roundToNearest5 rounds a float to the nearest 5-lb increment.
func roundToNearest5(lbs float64) int {
	return int(math.Round(lbs/5.0)) * 5
}

// calcTargetWeight calculates the target weight as a percentage of 1RM, rounded to nearest 5 lbs.
func calcTargetWeight(oneRM int, pct float64) int {
	return roundToNearest5(float64(oneRM) * pct)
}
```

- [ ] **Step 8: Run test to verify it passes**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestRound|TestCalcTarget"
```

Expected: PASS.

- [ ] **Step 9: Write test for warmup ramp generation**

Add to `generate_test.go`:

```go
func TestGenerateWarmupSets(t *testing.T) {
	sets := generateWarmupSets(2, 165)
	if len(sets) != 2 {
		t.Fatalf("expected 2 warmup sets, got %d", len(sets))
	}
	// Warmup 1: ~75% of 165 = 123.75 -> 125 lbs, 5 reps
	if sets[0].TargetLbs != 125 {
		t.Errorf("warmup 1: expected 125 lbs, got %d", sets[0].TargetLbs)
	}
	if sets[0].TargetReps != 5 {
		t.Errorf("warmup 1: expected 5 reps, got %d", sets[0].TargetReps)
	}
	if sets[0].Label != "Warm-up Set 1" {
		t.Errorf("warmup 1: expected label 'Warm-up Set 1', got %q", sets[0].Label)
	}
	// Warmup 2: ~87.5% of 165 = 144.375 -> 145 lbs, 3 reps
	if sets[1].TargetLbs != 145 {
		t.Errorf("warmup 2: expected 145 lbs, got %d", sets[1].TargetLbs)
	}
	if sets[1].TargetReps != 3 {
		t.Errorf("warmup 2: expected 3 reps, got %d", sets[1].TargetReps)
	}
}

func TestGenerateWarmupSetsZero(t *testing.T) {
	sets := generateWarmupSets(0, 165)
	if len(sets) != 0 {
		t.Errorf("expected 0 warmup sets, got %d", len(sets))
	}
}
```

- [ ] **Step 10: Run test to verify it fails**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run TestGenerateWarmup
```

Expected: FAIL — function not defined.

- [ ] **Step 11: Implement warmup ramp generation**

Add to `generate.go`:

```go
// generateWarmupSets creates ramping warmup sets from ~75% to ~87.5% of work weight.
// Reps decrease as weight increases: 5 reps for first set, decreasing to 3.
func generateWarmupSets(count int, workWeight int) []GeneratedSet {
	if count == 0 {
		return nil
	}
	sets := make([]GeneratedSet, count)
	for i := range count {
		// Progress from 0.75 to ~0.875 linearly across warmup sets
		pct := 0.75 + (0.125 * float64(i) / float64(max(count-1, 1)))
		weight := roundToNearest5(float64(workWeight) * pct)
		reps := 5 - (2 * i / max(count-1, 1)) // 5 reps first, 3 reps last
		sets[i] = GeneratedSet{
			Label:      fmt.Sprintf("Warm-up Set %d", i+1),
			TargetPct:  pct,
			TargetLbs:  weight,
			TargetReps: reps,
		}
	}
	return sets
}
```

- [ ] **Step 12: Run test to verify it passes**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run TestGenerateWarmup
```

Expected: PASS.

- [ ] **Step 13: Write test for full workout generation**

Add to `generate_test.go`:

```go
func TestGenerateWorkout(t *testing.T) {
	day := ProgramDay{
		Label: "Saturday",
		Name:  "Upper Body Hypertrophy — Push Primary",
		Strength: []Slot{
			{
				Name:       "Secondary Push",
				Method:     "ME",
				Sets:       3,
				Reps:       Range{Min: 1, Max: 5},
				LoadPct:    RangeF{Min: 0.90, Max: 1.00},
				WarmupSets: 2,
				Choices:    []string{"Incline Bench Press", "Close-Grip Bench Press"},
			},
			{
				Name:    "Braced Push",
				Method:  "HYP",
				Sets:    3,
				Reps:    Range{Min: 6, Max: 12},
				LoadPct: RangeF{Min: 0.65, Max: 0.80},
				Choices: []string{"Machine Chest Press"},
			},
		},
		Endurance: &Endurance{Type: "Sprint+MLSS", Description: "Sprint + MLSS combo"},
	}
	prs := PRs{
		"Incline Bench Press":  181,
		"Close-Grip Bench Press": 200,
		"Machine Chest Press":  227,
	}

	overrides := map[int]string{}
	w := generateWorkout("1", day, prs, "test-seed", overrides)

	if w.Day != "1" {
		t.Errorf("expected day '1', got %q", w.Day)
	}
	if w.Label != "Saturday" {
		t.Errorf("expected label 'Saturday', got %q", w.Label)
	}
	if w.Seed != "test-seed" {
		t.Errorf("expected seed 'test-seed', got %q", w.Seed)
	}
	if len(w.Exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(w.Exercises))
	}

	// First exercise should have warmup sets
	ex0 := w.Exercises[0]
	if len(ex0.WarmupSets) != 2 {
		t.Errorf("expected 2 warmup sets for slot 0, got %d", len(ex0.WarmupSets))
	}
	if len(ex0.WorkSets) != 3 {
		t.Errorf("expected 3 work sets for slot 0, got %d", len(ex0.WorkSets))
	}
	if !ex0.HasPR {
		t.Error("expected slot 0 to have a PR")
	}

	// Second exercise should have no warmup sets
	ex1 := w.Exercises[1]
	if len(ex1.WarmupSets) != 0 {
		t.Errorf("expected 0 warmup sets for slot 1, got %d", len(ex1.WarmupSets))
	}
	if ex1.ExerciseName != "Machine Chest Press" {
		t.Errorf("expected 'Machine Chest Press' (single choice), got %q", ex1.ExerciseName)
	}

	if w.Endurance == nil || w.Endurance.Type != "Sprint+MLSS" {
		t.Errorf("expected endurance, got %v", w.Endurance)
	}
}

func TestGenerateWorkoutWithOverride(t *testing.T) {
	day := ProgramDay{
		Label: "Saturday",
		Name:  "Test",
		Strength: []Slot{
			{
				Name:    "Secondary Push",
				Method:  "ME",
				Sets:    3,
				Reps:    Range{Min: 1, Max: 5},
				LoadPct: RangeF{Min: 0.90, Max: 1.00},
				Choices: []string{"Incline Bench Press", "Close-Grip Bench Press", "Larsen Press"},
			},
		},
	}
	prs := PRs{
		"Incline Bench Press":    181,
		"Close-Grip Bench Press": 200,
		"Larsen Press":           170,
	}

	overrides := map[int]string{0: "Close-Grip Bench Press"}
	w := generateWorkout("1", day, prs, "any-seed", overrides)

	if w.Exercises[0].ExerciseName != "Close-Grip Bench Press" {
		t.Errorf("expected override to 'Close-Grip Bench Press', got %q", w.Exercises[0].ExerciseName)
	}
}

func TestGenerateWorkoutMissingPR(t *testing.T) {
	day := ProgramDay{
		Label: "Saturday",
		Name:  "Test",
		Strength: []Slot{
			{
				Name:    "Secondary Push",
				Method:  "ME",
				Sets:    3,
				Reps:    Range{Min: 1, Max: 5},
				LoadPct: RangeF{Min: 0.90, Max: 1.00},
				Choices: []string{"Larsen Press"},
			},
		},
	}
	prs := PRs{} // no PR for Larsen Press

	w := generateWorkout("1", day, prs, "seed", map[int]string{})
	if w.Exercises[0].HasPR {
		t.Error("expected HasPR=false for missing PR")
	}
}
```

- [ ] **Step 14: Run test to verify it fails**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestGenerateWorkout"
```

Expected: FAIL — `generateWorkout` not defined.

- [ ] **Step 15: Implement generateWorkout**

Add to `generate.go`:

```go
// methodNote returns a display string for the training method.
func methodNote(method string) string {
	switch method {
	case "ME":
		return "Ramping"
	case "DE":
		return "Dynamic effort. Maximum velocity, 3-4 RIR."
	case "SKILL":
		return "Skill work. Controlled eccentric, fast concentric, 3-4 RIR."
	case "HYP":
		return "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR)."
	default:
		return ""
	}
}

// generateWorkout builds a GeneratedWorkout from a program day, PRs, seed, and any slot overrides.
func generateWorkout(dayNum string, day ProgramDay, prs PRs, seed string, overrides map[int]string) GeneratedWorkout {
	w := GeneratedWorkout{
		Day:       dayNum,
		Label:     day.Label,
		Name:      day.Name,
		Seed:      seed,
		Endurance: day.Endurance,
	}

	for i, slot := range day.Strength {
		var exerciseName string
		if override, ok := overrides[i]; ok && isValidChoice(override, slot.Choices) {
			exerciseName = override
		} else {
			exerciseName = pickExercise(seed, i, slot.Choices)
		}

		oneRM, hasPR := prs[exerciseName]
		targetWeight := 0
		if hasPR {
			targetWeight = calcTargetWeight(oneRM, slot.LoadPct.Min)
		}

		warmupSets := generateWarmupSets(slot.WarmupSets, targetWeight)

		workSets := make([]GeneratedSet, slot.Sets)
		for j := range slot.Sets {
			workSets[j] = GeneratedSet{
				Label:      fmt.Sprintf("Work Set %d", j+1),
				TargetPct:  slot.LoadPct.Min,
				TargetLbs:  targetWeight,
				TargetReps: slot.Reps.Min,
			}
		}

		w.Exercises = append(w.Exercises, GeneratedExercise{
			SlotIndex:     i,
			SlotName:      slot.Name,
			Method:        slot.Method,
			ExerciseName:  exerciseName,
			Choices:       slot.Choices,
			SupersetGroup: slot.SupersetGroup,
			WarmupSets:    warmupSets,
			WorkSets:      workSets,
			HasPR:         hasPR,
			MethodNote:    methodNote(slot.Method),
		})
	}

	return w
}

// isValidChoice checks if an exercise name is in the choices list.
func isValidChoice(name string, choices []string) bool {
	for _, c := range choices {
		if c == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 16: Run tests to verify they pass**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestGenerateWorkout|TestPickExercise|TestRound|TestCalcTarget|TestGenerateWarmup"
```

Expected: All tests PASS.

- [ ] **Step 17: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add generate.go generate_test.go
git commit -m "feat: workout generation logic with exercise selection, weight calc, warmup ramps"
```

---

### Task 3: Obsidian Text Formatter

**Files:**
- Modify: `generate.go`
- Modify: `generate_test.go`

- [ ] **Step 1: Write test for Obsidian text output**

Add to `generate_test.go`:

```go
func TestFormatObsidianText(t *testing.T) {
	w := GeneratedWorkout{
		Day:   "1",
		Label: "Saturday",
		Name:  "Upper Body Hypertrophy — Push Primary",
		Seed:  "abc",
		Exercises: []GeneratedExercise{
			{
				SlotIndex:    0,
				SlotName:     "Secondary Push",
				Method:       "ME",
				ExerciseName: "Incline Bench Press",
				HasPR:        true,
				MethodNote:   "Ramping",
				WarmupSets: []GeneratedSet{
					{Label: "Warm-up Set 1", TargetPct: 0.75, TargetLbs: 125, TargetReps: 5},
					{Label: "Warm-up Set 2", TargetPct: 0.875, TargetLbs: 145, TargetReps: 3},
				},
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
					{Label: "Work Set 2", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
					{Label: "Work Set 3", TargetPct: 0.90, TargetLbs: 165, TargetReps: 3},
				},
			},
			{
				SlotIndex:    1,
				SlotName:     "Braced Push",
				Method:       "HYP",
				ExerciseName: "Machine Chest Press",
				HasPR:        true,
				MethodNote:   "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
					{Label: "Work Set 2", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
					{Label: "Work Set 3", TargetPct: 0.70, TargetLbs: 160, TargetReps: 10},
				},
			},
		},
		Endurance: &Endurance{Type: "Sprint+MLSS", Description: "Sprint + MLSS combo workout"},
	}

	text := formatObsidianText(w)

	// Check key structural elements are present
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

	for _, s := range mustContain {
		if !containsString(text, s) {
			t.Errorf("output missing expected string: %q\n\nFull output:\n%s", s, text)
		}
	}
}

func TestFormatObsidianTextMissingPR(t *testing.T) {
	w := GeneratedWorkout{
		Day:   "1",
		Label: "Saturday",
		Name:  "Test",
		Exercises: []GeneratedExercise{
			{
				SlotIndex:    0,
				SlotName:     "Secondary Push",
				Method:       "ME",
				ExerciseName: "Larsen Press",
				HasPR:        false,
				MethodNote:   "Ramping",
				WorkSets: []GeneratedSet{
					{Label: "Work Set 1"},
				},
			},
		},
	}

	text := formatObsidianText(w)
	if !containsString(text, "No PR recorded") {
		t.Errorf("expected 'No PR recorded' for missing PR\n\nFull output:\n%s", text)
	}
}

func TestFormatObsidianTextSuperset(t *testing.T) {
	arms := "arms"
	w := GeneratedWorkout{
		Day:   "1",
		Label: "Saturday",
		Name:  "Test",
		Exercises: []GeneratedExercise{
			{
				SlotIndex:     0,
				SlotName:      "Focused Push",
				Method:        "HYP",
				ExerciseName:  "Tricep Pushdown",
				SupersetGroup: &arms,
				HasPR:         true,
				MethodNote:    "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
				WorkSets:      []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.70, TargetLbs: 55, TargetReps: 12}},
			},
			{
				SlotIndex:     1,
				SlotName:      "Focused Pull",
				Method:        "HYP",
				ExerciseName:  "Preacher Curls",
				SupersetGroup: &arms,
				HasPR:         true,
				MethodNote:    "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
				WorkSets:      []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.75, TargetLbs: 55, TargetReps: 8}},
			},
		},
	}

	text := formatObsidianText(w)
	if !containsString(text, "Superset: Focused Push/Focused Pull") {
		t.Errorf("expected superset header\n\nFull output:\n%s", text)
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && stringContains(haystack, needle)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestFormatObsidian"
```

Expected: FAIL — `formatObsidianText` not defined.

- [ ] **Step 3: Implement formatObsidianText**

Add to `generate.go`:

```go
import "strings"

// formatObsidianText renders a GeneratedWorkout as plain text for Obsidian.
func formatObsidianText(w GeneratedWorkout) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Day %s — %s\n", w.Day, w.Label))
	b.WriteString(fmt.Sprintf("Lift: %s\n", w.Name))

	// Track which superset groups we've already opened
	openSuperset := ""

	for i, ex := range w.Exercises {
		// Handle superset grouping
		currentGroup := ""
		if ex.SupersetGroup != nil {
			currentGroup = *ex.SupersetGroup
		}

		if currentGroup != "" && currentGroup != openSuperset {
			// Starting a new superset group — collect slot names
			b.WriteString("\n")
			names := supersetSlotNames(w.Exercises, currentGroup)
			b.WriteString(fmt.Sprintf("Superset: %s\n", strings.Join(names, "/")))
			openSuperset = currentGroup
		} else if currentGroup == "" && openSuperset != "" {
			openSuperset = ""
		}

		if currentGroup == "" {
			b.WriteString("\n")
		}

		// Exercise header
		if ex.HasPR {
			b.WriteString(fmt.Sprintf("%s: %s\n", ex.ExerciseName, ex.MethodNote))
		} else {
			b.WriteString(fmt.Sprintf("%s: %s — No PR recorded, set target manually\n", ex.ExerciseName, ex.MethodNote))
		}

		// Warmup sets
		for _, s := range ex.WarmupSets {
			b.WriteString(fmt.Sprintf("%s\n", s.Label))
			if ex.HasPR {
				b.WriteString(fmt.Sprintf("target: %d%% 1RM (%d lbs) x %d\n", int(s.TargetPct*100), s.TargetLbs, s.TargetReps))
			} else {
				b.WriteString("target:\n")
			}
			b.WriteString("actual:\n")
		}

		// Work sets
		for _, s := range ex.WorkSets {
			b.WriteString(fmt.Sprintf("%s\n", s.Label))
			if ex.HasPR {
				b.WriteString(fmt.Sprintf("target: %d%% 1RM (%d lbs) x %d\n", int(s.TargetPct*100), s.TargetLbs, s.TargetReps))
			} else {
				b.WriteString("target:\n")
			}
			b.WriteString("actual:\n")
		}

		// Add blank line between exercises in a superset (except after last)
		if currentGroup != "" && i < len(w.Exercises)-1 {
			nextGroup := ""
			if w.Exercises[i+1].SupersetGroup != nil {
				nextGroup = *w.Exercises[i+1].SupersetGroup
			}
			if nextGroup == currentGroup {
				b.WriteString("\n")
			}
		}
	}

	// Endurance section
	if w.Endurance != nil && w.Endurance.Type != "None" && w.Endurance.Type != "Rest" {
		b.WriteString("\n---\n")
		b.WriteString(fmt.Sprintf("Endurance: %s\n", w.Endurance.Type))
		b.WriteString(w.Endurance.Description)
		b.WriteString("\n")
	}

	b.WriteString("\nUser notes:\n")

	return b.String()
}

// supersetSlotNames collects the SlotName of all exercises in a superset group.
func supersetSlotNames(exercises []GeneratedExercise, group string) []string {
	var names []string
	for _, ex := range exercises {
		if ex.SupersetGroup != nil && *ex.SupersetGroup == group {
			names = append(names, ex.SlotName)
		}
	}
	return names
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestFormatObsidian"
```

Expected: All 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add generate.go generate_test.go
git commit -m "feat: Obsidian text formatter with superset and missing-PR support"
```

---

### Task 4: HTTP Handlers

**Files:**
- Create: `handlers.go`
- Create: `handlers_test.go`

- [ ] **Step 1: Write test for index handler**

Create `handlers_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIndexHandler(t *testing.T) {
	app := &App{
		program: Program{
			"1": ProgramDay{Label: "Saturday", Name: "Upper Push"},
			"2": ProgramDay{Label: "Sunday", Name: "Lower Hinge"},
			"7": ProgramDay{Label: "Friday", Name: "Rest"},
		},
	}

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	app.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !stringContains(body, "Saturday") {
		t.Error("expected 'Saturday' in response")
	}
	if !stringContains(body, "Sunday") {
		t.Error("expected 'Sunday' in response")
	}
}
```

- [ ] **Step 2: Write test for workout handler**

Add to `handlers_test.go`:

```go
func TestWorkoutHandler(t *testing.T) {
	app := &App{
		program: Program{
			"1": ProgramDay{
				Label: "Saturday",
				Name:  "Upper Push",
				Strength: []Slot{
					{
						Name:    "Secondary Push",
						Method:  "ME",
						Sets:    3,
						Reps:    Range{Min: 1, Max: 5},
						LoadPct: RangeF{Min: 0.90, Max: 1.00},
						Choices: []string{"Incline Bench Press"},
					},
				},
				Endurance: &Endurance{Type: "Sprint+MLSS", Description: "Sprint combo"},
			},
		},
		prs: PRs{"Incline Bench Press": 181},
	}

	req := httptest.NewRequest("GET", "/workout?day=1", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !stringContains(body, "Incline Bench Press") {
		t.Error("expected 'Incline Bench Press' in response")
	}
	if !stringContains(body, "seed=") {
		t.Error("expected seed in swap links")
	}
}

func TestWorkoutHandlerMissingDay(t *testing.T) {
	app := &App{program: Program{}, prs: PRs{}}

	req := httptest.NewRequest("GET", "/workout", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWorkoutHandlerInvalidDay(t *testing.T) {
	app := &App{program: Program{}, prs: PRs{}}

	req := httptest.NewRequest("GET", "/workout?day=99", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWorkoutHandlerSeedDeterminism(t *testing.T) {
	app := &App{
		program: Program{
			"1": ProgramDay{
				Label: "Saturday",
				Name:  "Upper Push",
				Strength: []Slot{
					{
						Name:    "Secondary Push",
						Method:  "ME",
						Sets:    1,
						Reps:    Range{Min: 1, Max: 5},
						LoadPct: RangeF{Min: 0.90, Max: 1.00},
						Choices: []string{"Incline Bench Press", "Close-Grip Bench Press", "Larsen Press"},
					},
				},
			},
		},
		prs: PRs{
			"Incline Bench Press":    181,
			"Close-Grip Bench Press": 200,
			"Larsen Press":           170,
		},
	}

	req1 := httptest.NewRequest("GET", "/workout?day=1&seed=fixed", nil)
	rr1 := httptest.NewRecorder()
	app.handleWorkout(rr1, req1)

	req2 := httptest.NewRequest("GET", "/workout?day=1&seed=fixed", nil)
	rr2 := httptest.NewRecorder()
	app.handleWorkout(rr2, req2)

	if rr1.Body.String() != rr2.Body.String() {
		t.Error("same seed should produce identical output")
	}
}

func TestWorkoutHandlerSwapOverride(t *testing.T) {
	app := &App{
		program: Program{
			"1": ProgramDay{
				Label: "Saturday",
				Name:  "Upper Push",
				Strength: []Slot{
					{
						Name:    "Secondary Push",
						Method:  "ME",
						Sets:    1,
						Reps:    Range{Min: 1, Max: 5},
						LoadPct: RangeF{Min: 0.90, Max: 1.00},
						Choices: []string{"Incline Bench Press", "Close-Grip Bench Press"},
					},
				},
			},
		},
		prs: PRs{"Incline Bench Press": 181, "Close-Grip Bench Press": 200},
	}

	req := httptest.NewRequest("GET", "/workout?day=1&seed=test&s0=Close-Grip+Bench+Press", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	// The textarea output should contain the overridden exercise
	if !stringContains(body, "Close-Grip Bench Press") {
		t.Error("expected overridden exercise 'Close-Grip Bench Press' in output")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestIndexHandler|TestWorkoutHandler"
```

Expected: FAIL — `App` and handler methods not defined.

- [ ] **Step 4: Implement handlers**

Create `handlers.go`:

```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

// App holds the loaded config and parsed templates.
type App struct {
	program   Program
	prs       PRs
	templates *template.Template
}

// handleIndex renders the day picker page.
func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	type dayInfo struct {
		Num   string
		Label string
		Name  string
	}

	days := make([]dayInfo, 0, 7)
	for i := 1; i <= 7; i++ {
		num := strconv.Itoa(i)
		if day, ok := a.program[num]; ok {
			days = append(days, dayInfo{Num: num, Label: day.Label, Name: day.Name})
		}
	}

	a.templates.ExecuteTemplate(w, "index.html", days)
}

// handleWorkout generates and renders a workout for the requested day.
func (a *App) handleWorkout(w http.ResponseWriter, r *http.Request) {
	dayNum := r.URL.Query().Get("day")
	if dayNum == "" {
		http.Error(w, "missing 'day' parameter", http.StatusBadRequest)
		return
	}

	day, ok := a.program[dayNum]
	if !ok {
		http.Error(w, fmt.Sprintf("day %q not found in program", dayNum), http.StatusNotFound)
		return
	}

	seed := r.URL.Query().Get("seed")
	if seed == "" {
		seed = generateSeed()
	}

	// Parse slot overrides (s0, s1, s2, ...)
	overrides := map[int]string{}
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "s") && len(key) > 1 {
			idx, err := strconv.Atoi(key[1:])
			if err == nil && len(values) > 0 {
				overrides[idx] = values[0]
			}
		}
	}

	workout := generateWorkout(dayNum, day, a.prs, seed, overrides)
	obsidianText := formatObsidianText(workout)

	data := struct {
		Workout      GeneratedWorkout
		ObsidianText string
		Overrides    map[int]string
	}{
		Workout:      workout,
		ObsidianText: obsidianText,
		Overrides:    overrides,
	}

	a.templates.ExecuteTemplate(w, "workout.html", data)
}

// generateSeed returns a random 8-character hex string.
func generateSeed() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 5: Create minimal templates for tests**

Create `templates/index.html`:

```html
<!DOCTYPE html>
<html>
<head><title>Workout Generator</title></head>
<body>
<h1>Pick Your Day</h1>
{{range .}}
<a href="/workout?day={{.Num}}">Day {{.Num}} — {{.Label}}: {{.Name}}</a><br>
{{end}}
</body>
</html>
```

Create `templates/workout.html`:

```html
<!DOCTYPE html>
<html>
<head><title>Workout</title></head>
<body>
<h1>Day {{.Workout.Day}} — {{.Workout.Label}}: {{.Workout.Name}}</h1>
<a href="/workout?day={{.Workout.Day}}">Re-roll All</a> | <a href="/">Back</a>
{{range .Workout.Exercises}}
<div>
<strong>{{.ExerciseName}}</strong> — {{.SlotName}} ({{.Method}})
{{if gt (len .Choices) 1}}
<select onchange="window.location.href=this.value">
{{$w := $.Workout}}{{$o := $.Overrides}}{{$idx := .SlotIndex}}
{{range .Choices}}
<option value="/workout?day={{$w.Day}}&seed={{$w.Seed}}{{range $k, $v := $o}}&s{{$k}}={{$v}}{{end}}&s{{$idx}}={{.}}" {{if eq . (index $w.Exercises $idx).ExerciseName}}selected{{end}}>{{.}}</option>
{{end}}
</select>
{{end}}
{{range .WarmupSets}}
<div>{{.Label}}: {{.TargetLbs}} lbs x {{.TargetReps}}</div>
{{end}}
{{range .WorkSets}}
<div>{{.Label}}: {{.TargetLbs}} lbs x {{.TargetReps}}</div>
{{end}}
</div>
{{end}}
<textarea id="workout-output" readonly rows="30" cols="60">{{.ObsidianText}}</textarea>
<button onclick="copyWorkout()">Copy</button>
<script>
function copyWorkout() {
  const textarea = document.getElementById('workout-output');
  textarea.select();
  navigator.clipboard.writeText(textarea.value);
}
</script>
</body>
</html>
```

- [ ] **Step 6: Wire up template loading in App for tests**

Add a helper at the bottom of `handlers.go` that tests can use:

```go
// loadTemplatesFromDir parses templates from a directory.
func loadTemplatesFromDir(dir string) (*template.Template, error) {
	return template.ParseGlob(dir + "/*.html")
}
```

Update `handlers_test.go` to include template loading in setup. Add this at the top of the file, after imports:

```go
func newTestApp(program Program, prs PRs) *App {
	tmpl, err := loadTemplatesFromDir("templates")
	if err != nil {
		panic("failed to load templates: " + err.Error())
	}
	return &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
	}
}
```

Then update each test to use `newTestApp` instead of raw `&App{}`. For example, `TestIndexHandler` becomes:

```go
func TestIndexHandler(t *testing.T) {
	app := newTestApp(
		Program{
			"1": ProgramDay{Label: "Saturday", Name: "Upper Push"},
			"2": ProgramDay{Label: "Sunday", Name: "Lower Hinge"},
			"7": ProgramDay{Label: "Friday", Name: "Rest"},
		},
		PRs{},
	)
	// ... rest unchanged
}
```

Apply the same pattern to all other handler tests.

- [ ] **Step 7: Run tests to verify they pass**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -run "TestIndexHandler|TestWorkoutHandler"
```

Expected: All 6 tests PASS.

- [ ] **Step 8: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add handlers.go handlers_test.go templates/
git commit -m "feat: HTTP handlers for day picker and workout generation"
```

---

### Task 5: Main Entrypoint and Config Loading

**Files:**
- Create: `main.go`

- [ ] **Step 1: Implement main.go**

```go
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/*.html
var templateFS embed.FS

func main() {
	dataDir := "data"
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	program, err := loadProgram(dataDir + "/program.json")
	if err != nil {
		log.Fatalf("failed to load program.json: %v", err)
	}

	prs, err := loadPRs(dataDir + "/prs.json")
	if err != nil {
		log.Fatalf("failed to load prs.json: %v", err)
	}

	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	app := &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
	}

	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/workout", app.handleWorkout)

	fmt.Printf("Listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadProgram(path string) (Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Program
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func loadPRs(path string) (PRs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var prs PRs
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go build -o /dev/null .
```

Expected: Compiles with no errors.

- [ ] **Step 3: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add main.go
git commit -m "feat: main entrypoint with config loading and embedded templates"
```

---

### Task 6: Seed Data Files

**Files:**
- Create: `data/prs.json`
- Create: `data/program.json`

These files contain the user's actual training data derived from the prompt in the spec.

- [ ] **Step 1: Create prs.json**

Create `data/prs.json` with all exercises from the user's PR table that appear in the program:

```json
{
  "Larsen Press": 222,
  "Incline Bench Press": 181,
  "Close-Grip Bench Press": 222,
  "JM Press": 222,
  "Seated DB Press": 63,
  "Arnold Press": 63,
  "Smith Machine Press": 207,
  "Machine Chest Press": 227,
  "Chest Supported Row": 95,
  "Lat Pulldown": 246,
  "Close Grip Lat Pulldown": 174,
  "Cable Row": 140,
  "Cable Upright Row": 107,
  "Tricep Pushdown": 77,
  "Lateral Raise": 28,
  "Pec Deck": 227,
  "Preacher Curls": 74,
  "Spider Curls": 28,
  "Rear Delt Machine": 140,
  "Romanian Deadlift": 234,
  "Stiff-Legged Deadlift": 234,
  "Good Morning": 180,
  "GHD Back Extension": 297,
  "Hack Squat": 380,
  "Leg Press": 413,
  "Leg Extension": 161,
  "Barbell Split Squat": 187,
  "Bulgarian Split Squat": 67,
  "Forward Lunge": 187,
  "Reverse Lunge": 187,
  "Kroc Row": 114,
  "T-Bar Row": 163,
  "Meadows Row": 114,
  "DB Pullover": 73,
  "Dumbbell Bench Press": 76,
  "Dumbbell Lateral Raise": 28,
  "Dumbbell Rear Delt Fly": 107,
  "One Arm Dumbbell Row": 114,
  "Trap Bar Deadlift": 262,
  "Trap Bar Romanian Deadlift": 234,
  "Conventional Deadlift": 247,
  "Back Squat": 277,
  "Barbell Bench Press": 222,
  "Barbell Military Press": 111,
  "Dumbbell Overhead Press": 63,
  "Barbell Curl": 76,
  "Seated Row": 192,
  "Decline Bench Press": 196,
  "Machine Shoulder Press": 156,
  "Calf Raise": 196,
  "Goblet Squat": 76,
  "Dumbbell RDL": 80,
  "Upright Row": 107,
  "Face Pulls": 60,
  "Single-Leg Press": 102,
  "Machine Rear Delt Fly": 153,
  "Dumbbell Shrugs": 107
}
```

- [ ] **Step 2: Create program.json with Day 1 (Saturday)**

Create `data/program.json`. This is the full 7-day program. Building it incrementally — start with the complete file:

```json
{
  "1": {
    "label": "Saturday",
    "name": "Upper Body Hypertrophy — Push Primary",
    "strength": [
      {
        "name": "Secondary Push",
        "method": "ME",
        "sets": 3,
        "reps": {"min": 1, "max": 5},
        "load_pct": {"min": 0.90, "max": 1.00},
        "warmup_sets": 2,
        "choices": ["Larsen Press", "Incline Bench Press", "Close-Grip Bench Press", "JM Press", "Seated DB Press", "Arnold Press"],
        "superset_group": null
      },
      {
        "name": "Braced Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 6, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Smith Machine Press", "Machine Chest Press"],
        "superset_group": null
      },
      {
        "name": "Braced Pull",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 6, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Chest Supported Row", "Lat Pulldown", "Close Grip Lat Pulldown", "Cable Row"],
        "superset_group": null
      },
      {
        "name": "Focused Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Tricep Pushdown", "Lateral Raise", "Pec Deck"],
        "superset_group": "arms"
      },
      {
        "name": "Focused Pull",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Preacher Curls", "Spider Curls", "Rear Delt Machine"],
        "superset_group": "arms"
      },
      {
        "name": "Focused Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Dumbbell Lateral Raise", "Pec Deck"],
        "superset_group": null
      }
    ],
    "endurance": {
      "type": "Sprint+MLSS",
      "description": "Combined Sprint + MLSS+ session (target 45-50 min total).\nSprint warm-up, remove Sprint cooldown, remove MLSS+ warm-up, use MLSS+ cooldown."
    }
  },
  "2": {
    "label": "Sunday",
    "name": "Lower Body Hypertrophy — Hinge Primary",
    "strength": [
      {
        "name": "Secondary Hinge",
        "method": "DE",
        "sets": 5,
        "reps": {"min": 2, "max": 4},
        "load_pct": {"min": 0.70, "max": 0.80},
        "warmup_sets": 2,
        "choices": ["Romanian Deadlift", "Stiff-Legged Deadlift", "Good Morning", "Trap Bar Romanian Deadlift"],
        "superset_group": null
      },
      {
        "name": "Secondary Hinge",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Romanian Deadlift", "Stiff-Legged Deadlift", "Good Morning", "Dumbbell RDL"],
        "superset_group": null
      },
      {
        "name": "Braced Hinge",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["GHD Back Extension"],
        "superset_group": "hinge-push"
      },
      {
        "name": "Braced Lower Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Hack Squat", "Leg Press"],
        "superset_group": "hinge-push"
      },
      {
        "name": "Focused Hamstring",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Leg Extension", "Calf Raise"],
        "superset_group": null
      },
      {
        "name": "Braced Push (Asymmetrical)",
        "method": "DE",
        "sets": 4,
        "reps": {"min": 2, "max": 4},
        "load_pct": {"min": 0.70, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Single-Leg Press", "Barbell Split Squat", "Bulgarian Split Squat"],
        "superset_group": null
      }
    ],
    "endurance": {
      "type": "None",
      "description": "No endurance component — strength training only."
    }
  },
  "3": {
    "label": "Monday",
    "name": "Plyometric Warmup + Near-Threshold Run",
    "strength": [],
    "endurance": {
      "type": "NT",
      "description": "NT Option 3:\n1 x 1000m @ 95% with 3-minute rest\n1 x 800m @ 95% with 3-minute rest\n4 x 400m @ 95% with 1:30 rest\n4 x 200m @ 95% with 1-minute rest\n\nPlyometric warmup: 3-4 drills from bounding/ground contact before the run."
    }
  },
  "4": {
    "label": "Tuesday",
    "name": "Upper Body Hypertrophy — Pull Primary",
    "strength": [
      {
        "name": "Secondary Pull",
        "method": "ME",
        "sets": 3,
        "reps": {"min": 1, "max": 5},
        "load_pct": {"min": 0.90, "max": 1.00},
        "warmup_sets": 2,
        "choices": ["Kroc Row", "T-Bar Row", "Meadows Row", "DB Pullover"],
        "superset_group": null
      },
      {
        "name": "Braced Pull",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 6, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Chest Supported Row", "Lat Pulldown", "Close Grip Lat Pulldown", "Cable Row"],
        "superset_group": null
      },
      {
        "name": "Braced Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 6, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Smith Machine Press", "Machine Chest Press"],
        "superset_group": null
      },
      {
        "name": "Focused Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Tricep Pushdown", "Lateral Raise"],
        "superset_group": "arms"
      },
      {
        "name": "Focused Pull",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Preacher Curls", "Spider Curls", "Rear Delt Machine"],
        "superset_group": "arms"
      },
      {
        "name": "Focused Pull",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Rear Delt Machine", "Machine Rear Delt Fly", "Face Pulls"],
        "superset_group": null
      }
    ],
    "endurance": {
      "type": "VT1",
      "description": "25-30 minutes at VT1 pace (7:27/km).\nSimple 5-minute progressive warm-up and 5-minute cool-down."
    }
  },
  "5": {
    "label": "Wednesday",
    "name": "Lower Body Hypertrophy — Push Primary",
    "strength": [
      {
        "name": "Secondary Push",
        "method": "DE",
        "sets": 5,
        "reps": {"min": 2, "max": 4},
        "load_pct": {"min": 0.70, "max": 0.80},
        "warmup_sets": 2,
        "choices": ["Back Squat", "Barbell Split Squat", "Forward Lunge", "Reverse Lunge"],
        "superset_group": null
      },
      {
        "name": "Secondary Hinge",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Romanian Deadlift", "Stiff-Legged Deadlift", "Good Morning", "Dumbbell RDL"],
        "superset_group": null
      },
      {
        "name": "Braced Hinge",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["GHD Back Extension"],
        "superset_group": "hinge-push"
      },
      {
        "name": "Braced Lower Push",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 8, "max": 12},
        "load_pct": {"min": 0.65, "max": 0.80},
        "warmup_sets": 0,
        "choices": ["Hack Squat", "Leg Press"],
        "superset_group": "hinge-push"
      },
      {
        "name": "Focused Quadriceps",
        "method": "HYP",
        "sets": 3,
        "reps": {"min": 10, "max": 15},
        "load_pct": {"min": 0.60, "max": 0.75},
        "warmup_sets": 0,
        "choices": ["Leg Extension", "Calf Raise"],
        "superset_group": null
      },
      {
        "name": "Braced Push (Asymmetrical)",
        "method": "SKILL",
        "sets": 4,
        "reps": {"min": 3, "max": 5},
        "load_pct": {"min": 0.75, "max": 0.85},
        "warmup_sets": 0,
        "choices": ["Single-Leg Press", "Barbell Split Squat", "Bulgarian Split Squat"],
        "superset_group": null
      }
    ],
    "endurance": {
      "type": "None",
      "description": "No endurance component — strength training only."
    }
  },
  "6": {
    "label": "Thursday",
    "name": "Long Slow Distance Run",
    "strength": [],
    "endurance": {
      "type": "LSD",
      "description": "LSD run — select from Options 1-4 (60-100 minutes).\nUse Standalone Endurance Warm-Up (10-min jog, lunges, Cossack squats) and Cool-Down (8-min jog).\nEmphasize mixed terrain."
    }
  },
  "7": {
    "label": "Friday",
    "name": "Rest",
    "strength": [],
    "endurance": {
      "type": "Rest",
      "description": "Rest day — no training."
    }
  }
}
```

- [ ] **Step 3: Verify JSON is valid**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && python3 -m json.tool data/program.json > /dev/null && echo "program.json OK" && python3 -m json.tool data/prs.json > /dev/null && echo "prs.json OK"
```

Expected: Both files valid.

- [ ] **Step 4: Run all tests to verify the full stack works**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v ./...
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add data/
git commit -m "feat: seed data files with full 7-day program and PRs"
```

---

### Task 7: Polished Templates

**Files:**
- Modify: `templates/index.html`
- Modify: `templates/workout.html`

- [ ] **Step 1: Rewrite index.html with styling**

Overwrite `templates/index.html`:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Workout Generator</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; padding: 20px; max-width: 600px; margin: 0 auto; }
    h1 { margin-bottom: 20px; font-size: 1.5rem; }
    .day-list { display: flex; flex-direction: column; gap: 10px; }
    .day-link { display: block; padding: 16px; background: #fff; border: 1px solid #ddd; border-radius: 8px; text-decoration: none; color: #333; transition: background 0.15s; }
    .day-link:hover { background: #e8f4fd; }
    .day-num { font-weight: bold; font-size: 1.1rem; }
    .day-name { color: #666; font-size: 0.9rem; margin-top: 2px; }
    .rest { opacity: 0.5; }
  </style>
</head>
<body>
  <h1>Workout Generator</h1>
  <div class="day-list">
    {{range .}}
    <a href="/workout?day={{.Num}}" class="day-link{{if eq .Name "Rest"}} rest{{end}}">
      <div class="day-num">Day {{.Num}} — {{.Label}}</div>
      <div class="day-name">{{.Name}}</div>
    </a>
    {{end}}
  </div>
</body>
</html>
```

- [ ] **Step 2: Rewrite workout.html with styling and swap controls**

Overwrite `templates/workout.html`. This is the main template — it renders the visual workout, swap dropdowns, and the Obsidian textarea:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Day {{.Workout.Day}} — {{.Workout.Label}}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f5f5f5; color: #333; padding: 20px; max-width: 700px; margin: 0 auto; }
    h1 { font-size: 1.3rem; margin-bottom: 4px; }
    .subtitle { color: #666; font-size: 0.9rem; margin-bottom: 16px; }
    .nav { margin-bottom: 20px; font-size: 0.9rem; }
    .nav a { color: #0066cc; text-decoration: none; margin-right: 16px; }
    .nav a:hover { text-decoration: underline; }
    .exercise { background: #fff; border: 1px solid #ddd; border-radius: 8px; padding: 16px; margin-bottom: 12px; }
    .exercise-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; flex-wrap: wrap; gap: 8px; }
    .exercise-name { font-weight: bold; font-size: 1.05rem; }
    .exercise-meta { font-size: 0.8rem; color: #888; }
    .swap-select { font-size: 0.85rem; padding: 4px 8px; border: 1px solid #ccc; border-radius: 4px; }
    .set { display: flex; justify-content: space-between; padding: 4px 0; font-size: 0.9rem; border-bottom: 1px solid #f0f0f0; }
    .set:last-child { border-bottom: none; }
    .set-label { color: #666; }
    .set-target { font-weight: 500; }
    .warmup .set-target { color: #999; }
    .superset-header { background: #f0f0f0; border-radius: 8px; padding: 8px 16px; margin-bottom: 4px; font-size: 0.9rem; font-weight: 600; color: #555; }
    .superset-group { margin-bottom: 12px; }
    .superset-group .exercise { border-left: 3px solid #0066cc; margin-bottom: 4px; }
    .endurance { background: #fff; border: 1px solid #ddd; border-radius: 8px; padding: 16px; margin-bottom: 12px; }
    .endurance h2 { font-size: 1.1rem; margin-bottom: 8px; }
    .endurance pre { white-space: pre-wrap; font-family: inherit; font-size: 0.9rem; color: #555; }
    .output-section { margin-top: 24px; }
    .output-section h2 { font-size: 1.1rem; margin-bottom: 8px; }
    textarea { width: 100%; font-family: monospace; font-size: 0.85rem; padding: 12px; border: 1px solid #ddd; border-radius: 8px; background: #fff; resize: vertical; }
    .copy-btn { margin-top: 8px; padding: 8px 20px; background: #0066cc; color: #fff; border: none; border-radius: 6px; font-size: 0.9rem; cursor: pointer; }
    .copy-btn:hover { background: #0052a3; }
    .copy-btn.copied { background: #28a745; }
    .no-pr { color: #cc6600; font-style: italic; font-size: 0.85rem; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="/">Back</a>
    <a href="/workout?day={{.Workout.Day}}">Re-roll All</a>
  </div>

  <h1>Day {{.Workout.Day}} — {{.Workout.Label}}</h1>
  <div class="subtitle">{{.Workout.Name}}</div>

  {{- $workout := .Workout -}}
  {{- $overrides := .Overrides -}}
  {{- $openGroup := "" -}}

  {{range $i, $ex := .Workout.Exercises}}
    {{- $currentGroup := "" -}}
    {{- if $ex.SupersetGroup}}{{$currentGroup = deref $ex.SupersetGroup}}{{end -}}

    {{if and (ne $currentGroup "") (ne $currentGroup $openGroup)}}
      {{if ne $openGroup ""}}</div>{{end}}
      <div class="superset-group">
      <div class="superset-header">Superset: {{supersetNames $workout.Exercises $currentGroup}}</div>
    {{else if and (eq $currentGroup "") (ne $openGroup "")}}
      </div>
    {{end}}

    <div class="exercise">
      <div class="exercise-header">
        <div>
          <span class="exercise-name">{{$ex.ExerciseName}}</span>
          <span class="exercise-meta">{{$ex.SlotName}} — {{$ex.Method}}</span>
        </div>
        {{if gt (len $ex.Choices) 1}}
        <select class="swap-select" onchange="window.location.href=this.value">
          {{range $ex.Choices}}
          <option value="{{swapURL $workout.Day $workout.Seed $overrides $ex.SlotIndex .}}"{{if eq . $ex.ExerciseName}} selected{{end}}>{{.}}</option>
          {{end}}
        </select>
        {{end}}
      </div>

      {{if not $ex.HasPR}}
      <div class="no-pr">No PR recorded — set target manually</div>
      {{end}}

      {{if $ex.WarmupSets}}
      <div class="warmup">
        {{range $ex.WarmupSets}}
        <div class="set">
          <span class="set-label">{{.Label}}</span>
          <span class="set-target">{{.TargetLbs}} lbs x {{.TargetReps}}</span>
        </div>
        {{end}}
      </div>
      {{end}}

      <div class="work">
        {{range $ex.WorkSets}}
        <div class="set">
          <span class="set-label">{{.Label}}</span>
          <span class="set-target">{{if $ex.HasPR}}{{.TargetLbs}} lbs x {{.TargetReps}}{{else}}—{{end}}</span>
        </div>
        {{end}}
      </div>

      {{if $ex.MethodNote}}
      <div class="exercise-meta" style="margin-top:8px">{{$ex.MethodNote}}</div>
      {{end}}
    </div>

    {{- if $ex.SupersetGroup}}{{$openGroup = deref $ex.SupersetGroup}}{{else}}{{$openGroup = ""}}{{end -}}
  {{end}}

  {{if ne $openGroup ""}}</div>{{end}}

  {{if and .Workout.Endurance (ne .Workout.Endurance.Type "None") (ne .Workout.Endurance.Type "Rest")}}
  <div class="endurance">
    <h2>Endurance: {{.Workout.Endurance.Type}}</h2>
    <pre>{{.Workout.Endurance.Description}}</pre>
  </div>
  {{end}}

  {{if and .Workout.Endurance (eq .Workout.Endurance.Type "Rest")}}
  <div class="endurance">
    <h2>Rest Day</h2>
    <pre>{{.Workout.Endurance.Description}}</pre>
  </div>
  {{end}}

  <div class="output-section">
    <h2>Obsidian Output</h2>
    <textarea id="workout-output" readonly rows="{{textareaRows .ObsidianText}}">{{.ObsidianText}}</textarea>
    <button class="copy-btn" onclick="copyWorkout(this)">Copy to Clipboard</button>
  </div>

  <script>
  function copyWorkout(btn) {
    var ta = document.getElementById('workout-output');
    ta.select();
    navigator.clipboard.writeText(ta.value).then(function() {
      btn.textContent = 'Copied!';
      btn.classList.add('copied');
      setTimeout(function() {
        btn.textContent = 'Copy to Clipboard';
        btn.classList.remove('copied');
      }, 2000);
    });
  }
  </script>
</body>
</html>
```

- [ ] **Step 3: Add template functions to handlers.go**

The template uses custom functions (`swapURL`, `deref`, `supersetNames`, `textareaRows`). Add them to `handlers.go` and update the template parsing:

Add to `handlers.go`:

```go
import (
	"net/url"
	"strings"
)

// templateFuncs returns the FuncMap for workout templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"swapURL": func(day, seed string, overrides map[int]string, slotIndex int, exercise string) string {
			params := url.Values{}
			params.Set("day", day)
			params.Set("seed", seed)
			// Carry forward all existing overrides
			for k, v := range overrides {
				if k != slotIndex {
					params.Set(fmt.Sprintf("s%d", k), v)
				}
			}
			// Set the new override
			params.Set(fmt.Sprintf("s%d", slotIndex), exercise)
			return "/workout?" + params.Encode()
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
		"supersetNames": func(exercises []GeneratedExercise, group string) string {
			names := supersetSlotNames(exercises, group)
			return strings.Join(names, "/")
		},
		"textareaRows": func(text string) int {
			lines := strings.Count(text, "\n") + 1
			if lines < 10 {
				return 10
			}
			return lines + 2
		},
	}
}
```

Update `loadTemplatesFromDir` in `handlers.go`:

```go
func loadTemplatesFromDir(dir string) (*template.Template, error) {
	return template.New("").Funcs(templateFuncs()).ParseGlob(dir + "/*.html")
}
```

Update `main.go` template parsing to use the FuncMap:

```go
tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templateFS, "templates/*.html")
```

- [ ] **Step 4: Run all tests**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v ./...
```

Expected: All tests PASS (template functions are now available).

- [ ] **Step 5: Smoke test — run the server locally**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go run . &
sleep 1
curl -s http://localhost:8080/ | head -20
curl -s "http://localhost:8080/workout?day=1" | head -40
kill %1
```

Expected: HTML responses with day picker and a generated workout.

- [ ] **Step 6: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add templates/ handlers.go main.go
git commit -m "feat: polished templates with swap controls and Obsidian output"
```

---

### Task 8: Dockerfile

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

- [ ] **Step 1: Create .dockerignore**

```
.git
docs/
*.md
```

- [ ] **Step 2: Create Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o workout-generator .

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/workout-generator .
EXPOSE 8080
CMD ["./workout-generator"]
```

- [ ] **Step 3: Build and run**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && docker build -t workout-generator . && docker run --rm -p 8080:8080 -v "$(pwd)/data:/app/data" workout-generator &
sleep 2
curl -s http://localhost:8080/ | head -5
docker stop $(docker ps -q --filter ancestor=workout-generator)
```

Expected: Container starts, serves the day picker page, stops cleanly.

- [ ] **Step 4: Commit**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k
git add Dockerfile .dockerignore
git commit -m "feat: Dockerfile with multi-stage build"
```

---

### Task 9: End-to-End Verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go test -v -count=1 ./...
```

Expected: All tests PASS.

- [ ] **Step 2: Manual verification — start server and test each day**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go run . &
sleep 1
# Test day picker
curl -s http://localhost:8080/ | grep -c "day-link"
# Test each day generates
for d in 1 2 3 4 5 6 7; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/workout?day=$d")
  echo "Day $d: $STATUS"
done
# Test swap
curl -s "http://localhost:8080/workout?day=1&seed=test&s0=Incline+Bench+Press" | grep -c "Incline Bench Press"
kill %1
```

Expected: All days return 200. Swap returns the overridden exercise.

- [ ] **Step 3: Open in browser and verify**

```bash
cd /Users/aaron.romeo/workspace/uha-hyp-5k && go run .
```

Open `http://localhost:8080` in a browser. Verify:
- Day picker shows all 7 days
- Clicking a day generates a workout with random exercises
- Swap dropdown changes one exercise without re-rolling others
- "Re-roll All" generates a fresh random workout
- Textarea contains properly formatted Obsidian text
- Copy button works
- Day 7 shows rest
- Day 3 and Day 6 show endurance only (no strength)

- [ ] **Step 4: Commit any fixes from manual testing**

If any issues found, fix and commit.
