package main

import (
	"fmt"
	"strconv"
	"testing"
)

func makeTestProgram() Program {
	p := Program{}
	for i := 1; i <= 7; i++ {
		p[strconv.Itoa(i)] = ProgramDay{
			Label: fmt.Sprintf("original-%d", i),
			Name:  fmt.Sprintf("Workout %d", i),
		}
	}
	return p
}

func TestApplyWeekStart(t *testing.T) {
	cases := []struct {
		name      string
		weekStart string
		want      []string
	}{
		{"sunday start", "Sunday", []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}},
		{"saturday start matches current data", "Saturday", []string{"Saturday", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}},
		{"monday start", "Monday", []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := applyWeekStart(makeTestProgram(), c.weekStart)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for i, wantLabel := range c.want {
				day := got[strconv.Itoa(i+1)]
				if day.Label != wantLabel {
					t.Errorf("day %d label = %q, want %q", i+1, day.Label, wantLabel)
				}
				if day.Name != fmt.Sprintf("Workout %d", i+1) {
					t.Errorf("day %d name changed: %q", i+1, day.Name)
				}
			}
		})
	}
}

func TestApplyWeekStart_EmptyIsNoOp(t *testing.T) {
	program := makeTestProgram()
	got, err := applyWeekStart(program, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= 7; i++ {
		key := strconv.Itoa(i)
		if got[key].Label != program[key].Label {
			t.Errorf("day %s label changed: %q != %q", key, got[key].Label, program[key].Label)
		}
	}
}

func TestApplyWeekStart_DoesNotMutateInput(t *testing.T) {
	program := makeTestProgram()
	if _, err := applyWeekStart(program, "Sunday"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program["1"].Label != "original-1" {
		t.Errorf("input program mutated: day 1 label = %q", program["1"].Label)
	}
}

func TestApplyWeekStart_RejectsInvalidValues(t *testing.T) {
	for _, bad := range []string{"Sat", "monday", "1", "Caturday", "Sunday "} {
		if _, err := applyWeekStart(makeTestProgram(), bad); err == nil {
			t.Errorf("expected error for WEEK_START=%q", bad)
		}
	}
}

func TestApplyWeekStart_PassesThroughNonDayKeys(t *testing.T) {
	program := makeTestProgram()
	program["8"] = ProgramDay{Label: "Extra Day", Name: "Whatever"}
	got, err := applyWeekStart(program, "Sunday")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["8"].Label != "Extra Day" {
		t.Errorf("key outside 1-7 should pass through unchanged, got %q", got["8"].Label)
	}
}
