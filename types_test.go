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
	if s.Name != "Secondary Push" { t.Errorf("expected name 'Secondary Push', got %q", s.Name) }
	if s.Method != "ME" { t.Errorf("expected method 'ME', got %q", s.Method) }
	if s.Sets != 3 { t.Errorf("expected sets 3, got %d", s.Sets) }
	if s.Reps.Min != 1 || s.Reps.Max != 5 { t.Errorf("expected reps 1-5, got %d-%d", s.Reps.Min, s.Reps.Max) }
	if s.LoadPct.Min != 0.90 || s.LoadPct.Max != 1.00 { t.Errorf("expected load_pct 0.90-1.00, got %f-%f", s.LoadPct.Min, s.LoadPct.Max) }
	if s.WarmupSets != 2 { t.Errorf("expected warmup_sets 2, got %d", s.WarmupSets) }
	if len(s.Choices) != 2 { t.Errorf("expected 2 choices, got %d", len(s.Choices)) }
	if s.SupersetGroup != nil { t.Errorf("expected nil superset_group, got %v", s.SupersetGroup) }
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
	if err := json.Unmarshal([]byte(raw), &s); err != nil { t.Fatalf("unmarshal failed: %v", err) }
	if s.SupersetGroup == nil || *s.SupersetGroup != "arms" { t.Errorf("expected superset_group 'arms', got %v", s.SupersetGroup) }
}

func TestProgramDayUnmarshal(t *testing.T) {
	raw := `{
		"label": "Saturday",
		"name": "Upper Body Hypertrophy",
		"strength": [{"name": "Secondary Push","method": "ME","sets": 3,"reps": {"min": 1, "max": 5},"load_pct": {"min": 0.90, "max": 1.00},"warmup_sets": 2,"choices": ["Incline Bench Press"],"superset_group": null}],
		"endurance": {"type": "Sprint+MLSS","description": "Sprint + MLSS combo"}
	}`
	var day ProgramDay
	if err := json.Unmarshal([]byte(raw), &day); err != nil { t.Fatalf("unmarshal failed: %v", err) }
	if day.Label != "Saturday" { t.Errorf("expected label 'Saturday', got %q", day.Label) }
	if len(day.Strength) != 1 { t.Errorf("expected 1 strength slot, got %d", len(day.Strength)) }
	if day.Endurance == nil || day.Endurance.Type != "Sprint+MLSS" { t.Errorf("expected endurance type 'Sprint+MLSS', got %v", day.Endurance) }
}

func TestPRsUnmarshal(t *testing.T) {
	raw := `{"Incline Bench Press": 181, "Machine Chest Press": 227}`
	var prs PRs
	if err := json.Unmarshal([]byte(raw), &prs); err != nil { t.Fatalf("unmarshal failed: %v", err) }
	if prs["Incline Bench Press"] != 181 { t.Errorf("expected 181, got %d", prs["Incline Bench Press"]) }
	if prs["Machine Chest Press"] != 227 { t.Errorf("expected 227, got %d", prs["Machine Chest Press"]) }
}
