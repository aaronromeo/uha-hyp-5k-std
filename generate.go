package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
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

// formatIntRange formats a pair of integers as "min–max" using an en-dash.
// Always renders both bounds.
func formatIntRange(minVal, maxVal int) string {
    return fmt.Sprintf("%d–%d", minVal, maxVal)
}

// formatFloatPctRange formats fractional percentages (0.90, 1.00) as
// integer-percent ranges like "90–100".
func formatFloatPctRange(minVal, maxVal float64) string {
    return fmt.Sprintf("%d–%d", int(minVal*100), int(maxVal*100))
}

// formatLbsRepsRange returns the "165–180 lbs x 1–5" substring.
func formatLbsRepsRange(s GeneratedSet) string {
    return fmt.Sprintf("%s lbs x %s", formatIntRange(s.TargetLbsMin, s.TargetLbsMax), formatIntRange(s.TargetRepsMin, s.TargetRepsMax))
}

// formatSetTargetLine returns the full target line like
// "90–100% 1RM (165–180 lbs) x 1–5" used by Obsidian output.
func formatSetTargetLine(s GeneratedSet) string {
    // Format as: "90–100% 1RM (165–180 lbs) x 3–5"
    lbsRange := formatIntRange(s.TargetLbsMin, s.TargetLbsMax)
    repsRange := formatIntRange(s.TargetRepsMin, s.TargetRepsMax)
    return fmt.Sprintf("%s%% 1RM (%s lbs) x %s", formatFloatPctRange(s.TargetPctMin, s.TargetPctMax), lbsRange, repsRange)
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
            Label:         fmt.Sprintf("Warm-up Set %d", i+1),
            TargetPctMin:  pct,
            TargetPctMax:  pct,
            TargetLbsMin:  weight,
            TargetLbsMax:  weight,
            TargetRepsMin: reps,
            TargetRepsMax: reps,
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

// formatObsidianText renders a GeneratedWorkout as plain text for Obsidian.
func formatObsidianText(w GeneratedWorkout) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Day %s — %s\n", w.Day, w.Label))
	b.WriteString(fmt.Sprintf("Lift: %s\n", w.Name))

	openSuperset := ""

	for i, ex := range w.Exercises {
		currentGroup := ""
		if ex.SupersetGroup != nil {
			currentGroup = *ex.SupersetGroup
		}

		if currentGroup != "" && currentGroup != openSuperset {
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

		if ex.HasPR {
			b.WriteString(fmt.Sprintf("%s: %s\n", ex.ExerciseName, ex.MethodNote))
		} else {
			b.WriteString(fmt.Sprintf("%s: %s — No PR recorded, set target manually\n", ex.ExerciseName, ex.MethodNote))
		}

		for _, s := range ex.WarmupSets {
			b.WriteString(fmt.Sprintf("%s\n", s.Label))
            if ex.HasPR {
                b.WriteString(fmt.Sprintf("target: %s\n", formatSetTargetLine(s)))
            } else {
                b.WriteString("target:\n")
            }
			b.WriteString("actual:\n")
		}

		for _, s := range ex.WorkSets {
			b.WriteString(fmt.Sprintf("%s\n", s.Label))
            if ex.HasPR {
                b.WriteString(fmt.Sprintf("target: %s\n", formatSetTargetLine(s)))
            } else {
                b.WriteString("target:\n")
            }
			b.WriteString("actual:\n")
		}

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
