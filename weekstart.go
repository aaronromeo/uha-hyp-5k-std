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
