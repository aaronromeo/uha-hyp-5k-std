package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestIndexHandler(t *testing.T) {
	app := newTestApp(
		Program{
			"1": ProgramDay{Label: "Saturday", Name: "Upper Push"},
			"2": ProgramDay{Label: "Sunday", Name: "Lower Hinge"},
			"7": ProgramDay{Label: "Friday", Name: "Rest"},
		},
		PRs{},
	)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	app.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !containsStr(body, "Saturday") {
		t.Error("expected 'Saturday' in response")
	}
	if !containsStr(body, "Sunday") {
		t.Error("expected 'Sunday' in response")
	}
}

func TestWorkoutHandler(t *testing.T) {
	app := newTestApp(
		Program{
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
		PRs{"Incline Bench Press": 181},
	)

	req := httptest.NewRequest("GET", "/workout?day=1", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !containsStr(body, "Incline Bench Press") {
		t.Error("expected 'Incline Bench Press' in response")
	}
	if !containsStr(body, "seed=") {
		t.Error("expected seed in swap links")
	}
}

func TestWorkoutHandlerMissingDay(t *testing.T) {
	app := newTestApp(Program{"1": ProgramDay{Label: "Saturday", Name: "X"}}, PRs{})

	req := httptest.NewRequest("GET", "/workout", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestWorkoutHandlerInvalidDay(t *testing.T) {
	app := newTestApp(Program{"1": ProgramDay{Label: "Saturday", Name: "X"}}, PRs{})

	req := httptest.NewRequest("GET", "/workout?day=99", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWorkoutHandlerSeedDeterminism(t *testing.T) {
	app := newTestApp(
		Program{
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
		PRs{"Incline Bench Press": 181, "Close-Grip Bench Press": 200, "Larsen Press": 170},
	)

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
	app := newTestApp(
		Program{
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
		PRs{"Incline Bench Press": 181, "Close-Grip Bench Press": 200},
	)

	req := httptest.NewRequest("GET", "/workout?day=1&seed=test&s0=Close-Grip+Bench+Press", nil)
	rr := httptest.NewRecorder()
	app.handleWorkout(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !containsStr(body, "Close-Grip Bench Press") {
		t.Error("expected overridden exercise 'Close-Grip Bench Press' in output")
	}
}
