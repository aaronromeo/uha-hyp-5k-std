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

// loadTemplatesFromDir parses templates from a directory.
func loadTemplatesFromDir(dir string) (*template.Template, error) {
	return template.ParseGlob(dir + "/*.html")
}
