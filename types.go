package main

type Program map[string]ProgramDay

type ProgramDay struct {
	Label    string     `json:"label"`
	Name     string     `json:"name"`
	Strength []Slot     `json:"strength"`
	Endurance *Endurance `json:"endurance,omitempty"`
}

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

type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type RangeF struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Endurance struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type PRs map[string]int

type GeneratedWorkout struct {
	Day       string
	Label     string
	Name      string
	Seed      string
	Exercises []GeneratedExercise
	Endurance *Endurance
}

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

type GeneratedSet struct {
	Label      string
	TargetPct  float64
	TargetLbs  int
	TargetReps int
}
