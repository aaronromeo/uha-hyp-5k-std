# Week Start + Obsidian Date Stamp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Relabel the seven program days so day 1 starts the week on a `WEEK_START` env-var weekday, and append a compact generation-date stamp (` - 20260829`) to the Obsidian output header.

**Architecture:** Two independent changes. (1) `formatObsidianText` gains a `ts time.Time` parameter — the App struct owns an injectable `now func() time.Time` (real clock in `main`, fixed clock in tests), keeping the formatter pure. (2) A new pure function `applyWeekStart` in a new file `weekstart.go` rewrites day labels for keys `"1"`–`"7"` before the `App` is constructed in `main`. Handlers, templates, and types are untouched.

**Tech Stack:** Go 1.25 stdlib only. No new dependencies.

**Spec:** `.scratch/week-start-date-stamp/spec.md`

## Global Constraints

- Go 1.25, stdlib only — zero dependencies in `go.mod`; do not add any.
- En-dash (U+2013, `–`) is for numeric ranges only. The Obsidian header's day–label separator is an em-dash (U+2014, `—`), unchanged. The date stamp separator is a **plain ASCII hyphen with spaces** (` - `) — never an en/em-dash.
- Exact output bytes are asserted by tests (`"Day 1 — Saturday - 20260829"`).
- `gofmt -l .` has a pre-existing dirty baseline of exactly five files: `generate.go`, `generate_test.go`, `handlers_test.go`, `types.go`, `types_test.go`. Do not reformat them wholesale; do not add new violations. New files (`weekstart.go`, `weekstart_test.go`) must be gofmt-clean.
- Run all commands from the repo root (handler tests read `templates/` from disk).
- Full check: `go build ./... && go vet ./... && go test ./... && gofmt -l .` — then `rm -f uha-hyp-5k` (the build artifact `go build` drops in the repo root; never commit it).
- Commit style: capitalized imperative subject, no conventional-commit prefix (e.g. `Add generation date stamp to Obsidian output`).

## Conventions used in this plan

- **"Find … replace with …"** blocks identify edits by exact source text. Match the find block character-for-character including whitespace. If a find block does not match, **stop and ask** — do not guess.
- Where a find block occurs multiple times, the step says to replace **all** occurrences.
- **Do not commit partial work.** Commits happen only at explicit "Commit" steps.

---

## File Structure

**Created:**
- `weekstart.go` — the `WEEK_START` relabel transform (pure function).
- `weekstart_test.go` — table tests for the relabel.

**Modified:**
- `generate.go` — `formatObsidianText` signature + stamped header line.
- `handlers.go` — `App.now` field; pass `a.now()` to the formatter.
- `main.go` — wire `now: time.Now`; apply `WEEK_START` after loading the program.
- `generate_test.go` — fixed test date; updated call sites and assertions.
- `handlers_test.go` — fixed clock in `newTestApp`; new date-stamp handler test.
- `AGENTS.md` — document `WEEK_START` on the Run line; caveat on the determinism bullet.

**Untouched:** `types.go` struct shapes, `templates/*.html`, `data/*.json`, `Dockerfile`.

---

### Task 1: Generation date stamp in Obsidian output

**Files:**
- Modify: `generate.go` (import block; `formatObsidianText` at ~line 183)
- Modify: `handlers.go` (import block; `App` struct at ~line 15; call site at ~line 71)
- Modify: `main.go` (import block; `App` literal at ~line 41)
- Modify: `generate_test.go` (imports; new `testDate` var; three call sites; one assertion)
- Modify: `handlers_test.go` (imports; `newTestApp`; new test at end of file)
- Modify: `AGENTS.md` (determinism bullet)

**Interfaces:**
- Consumes: `formatObsidianText(w GeneratedWorkout) string` (current), `App` struct (current).
- Produces: `formatObsidianText(w GeneratedWorkout, ts time.Time) string` — header line becomes `Day <day> — <label> - <YYYYMMDD>`; `App.now func() time.Time`; package-level test var `testDate` (2026-08-29 00:00 UTC) declared in `generate_test.go`, visible to `handlers_test.go` (same package).

- [ ] **Step 1: Update the formatter tests (RED part 1 — test side)**

In `generate_test.go`, find:

```go
import (
	"testing"
)
```

replace with:

```go
import (
	"testing"
	"time"
)
```

Then find:

```go
// ---- formatObsidianText ----

func TestFormatObsidianText(t *testing.T) {
```

replace with:

```go
// ---- formatObsidianText ----

var testDate = time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC)

func TestFormatObsidianText(t *testing.T) {
```

Then find (occurs **three** times — in `TestFormatObsidianText`, `TestFormatObsidianTextMissingPR`, `TestFormatObsidianTextSuperset`; replace **all**):

```go
	text := formatObsidianText(w)
```

replace with:

```go
	text := formatObsidianText(w, testDate)
```

Then find:

```go
		"Day 1 — Saturday",
```

replace with:

```go
		"Day 1 — Saturday - 20260829",
```

(Note: the first `—` is U+2014 em-dash; the ` - ` before the stamp is plain ASCII hyphen.)

- [ ] **Step 2: Update the handler tests (RED part 2 — test side)**

In `handlers_test.go`, find:

```go
import (
	"net/http"
	"net/http/httptest"
	"testing"
)
```

replace with:

```go
import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)
```

Then find:

```go
	return &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
	}
```

replace with:

```go
	return &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
		now:       func() time.Time { return testDate },
	}
```

Then append this test at the end of `handlers_test.go`:

```go
func TestWorkoutHandlerDateStamp(t *testing.T) {
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
						Choices: []string{"Incline Bench Press"},
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
	if !containsStr(rr.Body.String(), "Day 1 — Saturday - 20260829") {
		t.Errorf("expected stamped header in Obsidian output\n\nBody:\n%s", rr.Body.String())
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./...`
Expected: **compile failure** — errors like `not enough arguments in call to formatObsidianText` (generate.go call sites via handlers.go) and `unknown field 'now' in struct literal of type App`. This is the RED state.

- [ ] **Step 4: Implement**

In `generate.go`, find:

```go
import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
)
```

replace with:

```go
import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"
)
```

Then find:

```go
// formatObsidianText renders a GeneratedWorkout as plain text for Obsidian.
func formatObsidianText(w GeneratedWorkout) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Day %s — %s\n", w.Day, w.Label))
```

replace with:

```go
// formatObsidianText renders a GeneratedWorkout as plain text for Obsidian.
// The header line ends with the generation date, formatted compactly (YYYYMMDD).
func formatObsidianText(w GeneratedWorkout, ts time.Time) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Day %s — %s - %s\n", w.Day, w.Label, ts.Format("20060102")))
```

In `handlers.go`, find:

```go
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)
```

replace with:

```go
import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)
```

Then find:

```go
// App holds the loaded config and parsed templates.
type App struct {
	program   Program
	prs       PRs
	templates *template.Template
}
```

replace with:

```go
// App holds the loaded config and parsed templates.
type App struct {
	program   Program
	prs       PRs
	templates *template.Template
	now       func() time.Time
}
```

Then find:

```go
	workout := generateWorkout(dayNum, day, a.prs, seed, overrides)
	obsidianText := formatObsidianText(workout)
```

replace with:

```go
	workout := generateWorkout(dayNum, day, a.prs, seed, overrides)
	obsidianText := formatObsidianText(workout, a.now())
```

In `main.go`, find:

```go
import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)
```

replace with:

```go
import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)
```

Then find:

```go
	app := &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
	}
```

replace with:

```go
	app := &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
		now:       time.Now,
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./...`
Expected: `ok github.com/aaron-romeo/uha-hyp-5k` — all tests pass.

- [ ] **Step 6: Verify end-to-end with a live server**

Run (uses `/tmp/opencode`, port 8099 to avoid clashing with anything on 8080):

```bash
go build -o /tmp/opencode/wg . && (PORT=8099 /tmp/opencode/wg &) && sleep 1 && curl -s "http://localhost:8099/workout?day=1&seed=fixed" | grep -o "Day 1 — Saturday - [0-9]\{8\}"; pkill -f /tmp/opencode/wg
```

Expected: `Day 1 — Saturday - <today's date>` where `<today's date>` is the server's local date, e.g. `20260827`. (The label is `Saturday` because `WEEK_START` is unset — `data/program.json` labels win.)

- [ ] **Step 7: Update AGENTS.md determinism bullet**

Find:

```markdown
- **Generation is deterministic:** same `day` + `seed` + slot overrides → same workout (sha256 of `seed-slotIndex` seeds a PCG PRNG in `generate.go`). Pass `?seed=` to reproduce; tests depend on fixed seeds. Slot overrides are `?s0=Exercise+Name`, `?s1=…` by slot index.
```

replace with:

```markdown
- **Generation is deterministic:** same `day` + `seed` + slot overrides → same workout (sha256 of `seed-slotIndex` seeds a PCG PRNG in `generate.go`). Pass `?seed=` to reproduce; tests depend on fixed seeds. Slot overrides are `?s0=Exercise+Name`, `?s1=…` by slot index. The Obsidian header's ` - YYYYMMDD` stamp is the generation date (server local time; tests use fixed clocks), so same-seed output differs across days only in that stamp.
```

- [ ] **Step 8: Full check and commit**

Run: `go build ./... && go vet ./... && go test ./... && gofmt -l . && rm -f uha-hyp-5k`
Expected: tests pass; `gofmt -l .` lists exactly the five baseline files (`generate.go`, `generate_test.go`, `handlers_test.go`, `types.go`, `types_test.go`) — no new entries.

```bash
git add generate.go handlers.go main.go generate_test.go handlers_test.go AGENTS.md
git commit -m "Add generation date stamp to Obsidian output"
```

---

### Task 2: WEEK_START environment variable (relabel)

**Files:**
- Create: `weekstart.go`
- Create: `weekstart_test.go`
- Modify: `main.go` (wire `applyWeekStart` after `loadProgram`)
- Modify: `AGENTS.md` (Run line)

**Interfaces:**
- Consumes: `Program` (`map[string]ProgramDay`), `ProgramDay.Label` from `types.go`. Nothing from Task 1.
- Produces: `applyWeekStart(program Program, weekStart string) (Program, error)` — empty string returns the input unchanged; a valid weekday name (exact match, one of Sunday/Monday/Tuesday/Wednesday/Thursday/Friday/Saturday) relabels keys `"1"`–`"7"` so key `"1"` gets that weekday and key N the (N−1)-th following weekday, wrapping; anything else returns an error. Keys outside `"1"`–`"7"` and all non-Label fields pass through; the input map is never mutated.

- [ ] **Step 1: Write the failing tests**

Create `weekstart_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./...`
Expected: **compile failure** — `undefined: applyWeekStart`. This is the RED state.

- [ ] **Step 3: Implement**

Create `weekstart.go`:

```go
package main

import (
	"fmt"
	"strconv"
	"strings"
)

// weekdays lists the seven canonical weekday names in time.Weekday order
// (Sunday = 0).
var weekdays = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// applyWeekStart returns a copy of program with the labels of days "1"-"7"
// rewritten so that day "1" carries weekStart and each following day the next
// weekday, wrapping around. An empty weekStart returns the program unchanged
// (labels in the data win). Any other value that is not one of the seven
// canonical weekday names is an error. Keys outside "1"-"7" and every other
// field are passed through untouched.
func applyWeekStart(program Program, weekStart string) (Program, error) {
	if weekStart == "" {
		return program, nil
	}
	start := -1
	for i, name := range weekdays {
		if name == weekStart {
			start = i
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("WEEK_START must be one of: %s", strings.Join(weekdays, ", "))
	}
	relabeled := make(Program, len(program))
	for key, day := range program {
		n, err := strconv.Atoi(key)
		if err != nil || n < 1 || n > 7 {
			relabeled[key] = day
			continue
		}
		day.Label = weekdays[(start+n-1)%7]
		relabeled[key] = day
	}
	return relabeled, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: `ok github.com/aaron-romeo/uha-hyp-5k` — all tests pass.

- [ ] **Step 5: Wire into main.go**

In `main.go`, find:

```go
	program, err := loadProgram(dataDir + "/program.json")
	if err != nil {
		log.Fatalf("failed to load program.json: %v", err)
	}
```

replace with:

```go
	program, err := loadProgram(dataDir + "/program.json")
	if err != nil {
		log.Fatalf("failed to load program.json: %v", err)
	}

	program, err = applyWeekStart(program, os.Getenv("WEEK_START"))
	if err != nil {
		log.Fatalf("failed to apply WEEK_START: %v", err)
	}
```

- [ ] **Step 6: Verify end-to-end with a live server**

Run:

```bash
go build -o /tmp/opencode/wg . && (PORT=8099 WEEK_START=Sunday /tmp/opencode/wg &) && sleep 1 && curl -s "http://localhost:8099/" | grep -o "Day [1-7] — [A-Za-z]*" && curl -s "http://localhost:8099/workout?day=1&seed=fixed" | grep -o "Day 1 — [A-Za-z]* - [0-9]\{8\}"; pkill -f /tmp/opencode/wg
```

Expected output (the week runs Sunday→Saturday, rest day lands on Saturday):

```
Day 1 — Sunday
Day 2 — Monday
Day 3 — Tuesday
Day 4 — Wednesday
Day 5 — Thursday
Day 6 — Friday
Day 7 — Saturday
Day 1 — Sunday - <today's date>
```

Then verify invalid values abort startup:

```bash
WEEK_START=Sun /tmp/opencode/wg; echo "exit: $?"
```

Expected: prints `failed to apply WEEK_START: WEEK_START must be one of: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday` and `exit: 1`.

- [ ] **Step 7: Update AGENTS.md Run line**

Find:

```markdown
- Run: `go run .` — serves on `:8080` (override with `PORT`), reads `data/` (override with `DATA_DIR`).
```

replace with:

```markdown
- Run: `go run .` — serves on `:8080` (override with `PORT`), reads `data/` (override with `DATA_DIR`). `WEEK_START=<weekday>` relabels days 1–7 so day 1 starts the week on that weekday (unset: labels from `data/program.json` win); takes effect at restart.
```

- [ ] **Step 8: Full check and commit**

Run: `go build ./... && go vet ./... && go test ./... && gofmt -l . && rm -f uha-hyp-5k`
Expected: tests pass; `gofmt -l .` lists exactly the five baseline files plus nothing new (`weekstart.go` and `weekstart_test.go` must NOT appear).

```bash
git add weekstart.go weekstart_test.go main.go AGENTS.md
git commit -m "Add WEEK_START env var for configurable week start"
```

---

## Self-Review

- **Spec coverage:** Stories 1–5, 7–9 (relabel semantics, precedence, validation, restart) → Task 2. Stories 10–13 (stamp format, generation date, ASCII separator, HTML untouched) → Task 1. Stories 6 (workouts stay attached to slots) and 8 (picks unaffected) are guaranteed by design — `applyWeekStart` touches only `Label`; asserted by the `Name` check in `TestApplyWeekStart` and by Task 2 Step 6 showing day 1 still serves Upper Push. Stories 14–15 (env-var pattern, Docker passthrough) → Task 2 Step 5 follows the `PORT`/`DATA_DIR` pattern; no Dockerfile change exists to make. Stories 16–17 (parameter injection, testable clock) → Task 1 design. Story 18 (pure relabel function) → `weekstart.go`. No gaps.
- **Placeholder scan:** none — every step carries exact code, commands, and expected output.
- **Type consistency:** `formatObsidianText(w GeneratedWorkout, ts time.Time) string` and `App.now func() time.Time` match across Task 1 steps; `applyWeekStart(Program, string) (Program, error)` matches between Task 2 test and implementation; `testDate` declared once in `generate_test.go`, consumed in `handlers_test.go` (same package).
