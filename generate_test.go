package main

import (
	"testing"
)

// ---- pickExercise ----

func TestPickExercise_SingleChoice(t *testing.T) {
	result := pickExercise("seed", 0, []string{"Squat"})
	if result != "Squat" {
		t.Errorf("expected 'Squat', got %q", result)
	}
}

func TestPickExercise_Deterministic(t *testing.T) {
	choices := []string{"Squat", "Leg Press", "Hack Squat"}
	r1 := pickExercise("2024-01-01", 0, choices)
	r2 := pickExercise("2024-01-01", 0, choices)
	if r1 != r2 {
		t.Errorf("expected deterministic result, got %q and %q", r1, r2)
	}
}

func TestPickExercise_DifferentSlotsYieldDifferentResults(t *testing.T) {
	// With enough choices, different slots should typically differ
	choices := []string{"Squat", "Leg Press", "Hack Squat", "Bulgarian Split Squat"}
	r0 := pickExercise("2024-01-01", 0, choices)
	r1 := pickExercise("2024-01-01", 1, choices)
	// We can't guarantee they differ, but same seed+slot must be stable
	r0b := pickExercise("2024-01-01", 0, choices)
	r1b := pickExercise("2024-01-01", 1, choices)
	if r0 != r0b {
		t.Errorf("slot 0 not deterministic: %q vs %q", r0, r0b)
	}
	if r1 != r1b {
		t.Errorf("slot 1 not deterministic: %q vs %q", r1, r1b)
	}
}

// ---- roundToNearest5 ----

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
	for _, c := range cases {
		got := roundToNearest5(c.input)
		if got != c.expected {
			t.Errorf("roundToNearest5(%v) = %d, want %d", c.input, got, c.expected)
		}
	}
}

// ---- calcTargetWeight ----

func TestCalcTargetWeight(t *testing.T) {
	// 181 * 0.90 = 162.9 → rounds to 165
	got := calcTargetWeight(181, 0.90)
	if got != 165 {
		t.Errorf("calcTargetWeight(181, 0.90) = %d, want 165", got)
	}
}

// ---- generateWarmupSets ----

func TestGenerateWarmupSets_Zero(t *testing.T) {
	sets := generateWarmupSets(0, 165)
	if sets != nil {
		t.Errorf("expected nil for 0 warmup sets, got %v", sets)
	}
}

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

// ---- generateWorkout ----

func TestGenerateWorkout_Basic(t *testing.T) {
	day := ProgramDay{
		Label: "Monday",
		Name:  "Upper Strength",
		Strength: []Slot{
			{
				Name:       "Primary Push",
				Method:     "ME",
				Sets:       3,
				Reps:       Range{Min: 1, Max: 5},
				LoadPct:    RangeF{Min: 0.90, Max: 1.00},
				WarmupSets: 2,
				Choices:    []string{"Incline Bench Press"},
			},
			{
				Name:       "Arm Work",
				Method:     "HYP",
				Sets:       3,
				Reps:       Range{Min: 8, Max: 12},
				LoadPct:    RangeF{Min: 0.65, Max: 0.75},
				WarmupSets: 0,
				Choices:    []string{"Tricep Pushdown"},
			},
		},
	}
	prs := PRs{
		"Incline Bench Press": 181,
		"Tricep Pushdown":     100,
	}
	seed := "2024-W01-A"
	workout := generateWorkout("A", day, prs, seed, nil)

	if workout.Day != "A" {
		t.Errorf("expected day 'A', got %q", workout.Day)
	}
	if workout.Label != "Monday" {
		t.Errorf("expected label 'Monday', got %q", workout.Label)
	}
	if len(workout.Exercises) != 2 {
		t.Fatalf("expected 2 exercises, got %d", len(workout.Exercises))
	}

	// Exercise 0: ME slot
	ex0 := workout.Exercises[0]
	if ex0.ExerciseName != "Incline Bench Press" {
		t.Errorf("expected 'Incline Bench Press', got %q", ex0.ExerciseName)
	}
	if ex0.HasPR != true {
		t.Errorf("expected HasPR true for ex0")
	}
	if len(ex0.WarmupSets) != 2 {
		t.Errorf("expected 2 warmup sets, got %d", len(ex0.WarmupSets))
	}
	if len(ex0.WorkSets) != 3 {
		t.Errorf("expected 3 work sets, got %d", len(ex0.WorkSets))
	}

	// Exercise 1: HYP slot, single choice
	ex1 := workout.Exercises[1]
	if ex1.ExerciseName != "Tricep Pushdown" {
		t.Errorf("expected 'Tricep Pushdown', got %q", ex1.ExerciseName)
	}
	if len(ex1.WarmupSets) != 0 {
		t.Errorf("expected 0 warmup sets for HYP, got %d", len(ex1.WarmupSets))
	}
}

func TestGenerateWorkout_Override(t *testing.T) {
	day := ProgramDay{
		Label: "Monday",
		Name:  "Upper Strength",
		Strength: []Slot{
			{
				Name:    "Primary Push",
				Method:  "ME",
				Sets:    3,
				Reps:    Range{Min: 1, Max: 5},
				LoadPct: RangeF{Min: 0.90, Max: 1.00},
				Choices: []string{"Incline Bench Press", "Larsen Press"},
			},
		},
	}
	prs := PRs{"Incline Bench Press": 181, "Larsen Press": 200}
	overrides := map[int]string{0: "Larsen Press"}
	workout := generateWorkout("A", day, prs, "seed", overrides)

	if workout.Exercises[0].ExerciseName != "Larsen Press" {
		t.Errorf("expected override 'Larsen Press', got %q", workout.Exercises[0].ExerciseName)
	}
}

func TestGenerateWorkout_InvalidOverrideFallsBack(t *testing.T) {
	day := ProgramDay{
		Label: "Monday",
		Name:  "Upper Strength",
		Strength: []Slot{
			{
				Name:    "Primary Push",
				Method:  "ME",
				Sets:    3,
				Reps:    Range{Min: 1, Max: 5},
				LoadPct: RangeF{Min: 0.90, Max: 1.00},
				Choices: []string{"Incline Bench Press"},
			},
		},
	}
	prs := PRs{"Incline Bench Press": 181}
	overrides := map[int]string{0: "Not A Real Exercise"}
	workout := generateWorkout("A", day, prs, "seed", overrides)

	// Invalid override should fall back to pickExercise
	if workout.Exercises[0].ExerciseName != "Incline Bench Press" {
		t.Errorf("expected fallback 'Incline Bench Press', got %q", workout.Exercises[0].ExerciseName)
	}
}

func TestGenerateWorkout_MissingPR(t *testing.T) {
	day := ProgramDay{
		Label: "Monday",
		Name:  "Upper Strength",
		Strength: []Slot{
			{
				Name:    "Primary Push",
				Method:  "ME",
				Sets:    3,
				Reps:    Range{Min: 1, Max: 5},
				LoadPct: RangeF{Min: 0.90, Max: 1.00},
				Choices: []string{"Unknown Exercise"},
			},
		},
	}
	prs := PRs{} // no PRs
	workout := generateWorkout("A", day, prs, "seed", nil)

	ex := workout.Exercises[0]
	if ex.HasPR != false {
		t.Errorf("expected HasPR false for missing PR")
	}
	// Work sets should still be generated (with 0 weight)
	if len(ex.WorkSets) != 3 {
		t.Errorf("expected 3 work sets, got %d", len(ex.WorkSets))
	}
}

// ---- methodNote ----

func TestMethodNote(t *testing.T) {
	cases := []struct {
		method   string
		contains string
	}{
		{"ME", "Ramping"},
		{"DE", "Dynamic effort. Maximum velocity"},
		{"SKILL", "Skill work. Controlled eccentric"},
		{"HYP", "Hypertrophy focus. Aim for 0-2 Reps in Reserve"},
		{"UNKNOWN", ""},
	}
	for _, c := range cases {
		got := methodNote(c.method)
		if c.contains == "" {
			// any non-crashing result is fine
			continue
		}
		if !containsStr(got, c.contains) {
			t.Errorf("methodNote(%q) = %q, want to contain %q", c.method, got, c.contains)
		}
	}
}

// helper for test
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// ---- formatObsidianText ----

func TestFormatObsidianText(t *testing.T) {
	w := GeneratedWorkout{
		Day: "1", Label: "Saturday",
		Name: "Upper Body Hypertrophy — Push Primary",
		Seed: "abc",
		Exercises: []GeneratedExercise{
			{
				SlotIndex: 0, SlotName: "Secondary Push", Method: "ME",
				ExerciseName: "Incline Bench Press", HasPR: true, MethodNote: "Ramping",
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
				SlotIndex: 1, SlotName: "Braced Push", Method: "HYP",
				ExerciseName: "Machine Chest Press", HasPR: true,
				MethodNote: "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
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
		if !containsStr(text, s) {
			t.Errorf("output missing expected string: %q\n\nFull output:\n%s", s, text)
		}
	}
}

func TestFormatObsidianTextMissingPR(t *testing.T) {
	w := GeneratedWorkout{
		Day: "1", Label: "Saturday", Name: "Test",
		Exercises: []GeneratedExercise{
			{
				SlotIndex: 0, SlotName: "Secondary Push", Method: "ME",
				ExerciseName: "Larsen Press", HasPR: false, MethodNote: "Ramping",
				WorkSets: []GeneratedSet{{Label: "Work Set 1"}},
			},
		},
	}
	text := formatObsidianText(w)
	if !containsStr(text, "No PR recorded") {
		t.Errorf("expected 'No PR recorded' for missing PR\n\nFull output:\n%s", text)
	}
}

func TestFormatObsidianTextSuperset(t *testing.T) {
	arms := "arms"
	w := GeneratedWorkout{
		Day: "1", Label: "Saturday", Name: "Test",
		Exercises: []GeneratedExercise{
			{
				SlotIndex: 0, SlotName: "Focused Push", Method: "HYP",
				ExerciseName: "Tricep Pushdown", SupersetGroup: &arms, HasPR: true,
				MethodNote: "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
				WorkSets: []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.70, TargetLbs: 55, TargetReps: 12}},
			},
			{
				SlotIndex: 1, SlotName: "Focused Pull", Method: "HYP",
				ExerciseName: "Preacher Curls", SupersetGroup: &arms, HasPR: true,
				MethodNote: "Hypertrophy focus. Aim for 0-2 Reps in Reserve (RIR).",
				WorkSets: []GeneratedSet{{Label: "Work Set 1", TargetPct: 0.75, TargetLbs: 55, TargetReps: 8}},
			},
		},
	}
	text := formatObsidianText(w)
	if !containsStr(text, "Superset: Focused Push/Focused Pull") {
		t.Errorf("expected superset header\n\nFull output:\n%s", text)
	}
}

// ---- isValidChoice ----

func TestIsValidChoice(t *testing.T) {
	choices := []string{"Squat", "Leg Press"}
	if !isValidChoice("Squat", choices) {
		t.Error("expected 'Squat' to be valid")
	}
	if isValidChoice("Deadlift", choices) {
		t.Error("expected 'Deadlift' to be invalid")
	}
	if isValidChoice("", choices) {
		t.Error("expected empty string to be invalid")
	}
}
