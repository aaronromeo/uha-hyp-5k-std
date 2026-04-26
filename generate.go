package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
)

// pickExercise deterministically selects an exercise using SHA256 hash of
// seed+slotIndex to seed a PRNG.
func pickExercise(seed string, slotIndex int, choices []string) string {
	if len(choices) == 1 {
		return choices[0]
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", seed, slotIndex)))
	s := binary.BigEndian.Uint64(h[:8])
	r := rand.New(rand.NewPCG(s, 0))
	return choices[r.IntN(len(choices))]
}

// roundToNearest5 rounds a weight in lbs to the nearest 5-lb increment.
func roundToNearest5(lbs float64) int {
	return int(math.Round(lbs/5.0)) * 5
}

// calcTargetWeight computes the target weight as a percentage of a 1RM,
// rounded to the nearest 5-lb increment.
func calcTargetWeight(oneRM int, pct float64) int {
	return roundToNearest5(float64(oneRM) * pct)
}

// generateWarmupSets creates count ramping warmup sets from ~75% to ~87.5% of
// workWeight. Reps decrease from 5 down to 3.
func generateWarmupSets(count int, workWeight int) []GeneratedSet {
	if count == 0 {
		return nil
	}
	sets := make([]GeneratedSet, count)
	for i := range count {
		pct := 0.75 + (0.125 * float64(i) / float64(max(count-1, 1)))
		weight := roundToNearest5(float64(workWeight) * pct)
		reps := 5 - (2 * i / max(count-1, 1))
		sets[i] = GeneratedSet{
			Label:      fmt.Sprintf("Warm-up Set %d", i+1),
			TargetPct:  pct,
			TargetLbs:  weight,
			TargetReps: reps,
		}
	}
	return sets
}

// methodNote returns a display note for a given training method code.
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

// isValidChoice reports whether name is a non-empty member of choices.
func isValidChoice(name string, choices []string) bool {
	if name == "" {
		return false
	}
	for _, c := range choices {
		if c == name {
			return true
		}
	}
	return false
}

// generateWorkout produces a fully resolved GeneratedWorkout for a program day.
func generateWorkout(dayNum string, day ProgramDay, prs PRs, seed string, overrides map[int]string) GeneratedWorkout {
	exercises := make([]GeneratedExercise, len(day.Strength))

	for i, slot := range day.Strength {
		// Resolve exercise name: use override if present and valid, else pick.
		var exerciseName string
		if overrides != nil {
			if override, ok := overrides[i]; ok && isValidChoice(override, slot.Choices) {
				exerciseName = override
			}
		}
		if exerciseName == "" {
			exerciseName = pickExercise(seed, i, slot.Choices)
		}

		// Look up 1RM.
		oneRM, hasPR := prs[exerciseName]

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

		exercises[i] = GeneratedExercise{
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
		}
	}

	return GeneratedWorkout{
		Day:       dayNum,
		Label:     day.Label,
		Name:      day.Name,
		Seed:      seed,
		Exercises: exercises,
		Endurance: day.Endurance,
	}
}
