---
name: update-prs
description: Use when asked to update data/prs.json, recalculate PRs / estimated 1RMs, or process a strength workout log (training-log note, Epley formula) into the repo's PR data.
---

# update-prs

Recalculate estimated 1RMs in `data/prs.json` from a workout log. The user supplies the log path (typically a zettelkasten note under `/Users/aaron/workspace/zettelkasten/`).

## Formula

`1RM = weight × (1 + 0.0333 × reps)` (Epley), rounded to the nearest whole number.

## Steps

1. Read the workout log and `data/prs.json`.
2. Parse each exercise's `actual:` lines. Accept `185*8`, `185x8`, `185 x 8`, `185 * 8`. **Skip:** warm-up sets, empty/incomplete entries (`175*`, `actual:`), and bodyweight-only sets (`BW+25*8`) — no bodyweight is recorded anywhere in this repo, so BW sets are unusable.
3. Per exercise, keep the set with the **highest** computed 1RM (usually the highest-rep heavy set, not the heaviest weight).
4. Normalize exercise names with the mapping table below.
5. Merge into `data/prs.json`: per key, keep the higher of new/existing. Add keys that appear in the log but not the JSON. Only touch keys this run computed — leave all others byte-identical (some keys are stale duplicates, e.g. `Lateral Raise` next to `Dumbbell Lateral Raise`; that is not yours to fix).
6. Preserve format: alphabetical keys, 2-space indent, integer values.
7. Verify: `go build ./... && go vet ./... && go test ./...` from the repo root. `prs.json` keys must exactly match exercise names in `data/program.json` `choices` — a typo silently orphans the PR.
8. Ship: branch from `origin/main` (so the PR carries only this data change), commit, push, `gh pr create`. If `gh` fails with `401 Bad credentials`, the `GITHUB_TOKEN` env var on this machine is stale — rerun as `env -u GITHUB_TOKEN gh ...` to use the keyring token.

## Name normalization

Map log names to JSON keys. Anything unmapped: use the log's name as a new key.

- "Lateral Raise" / "Lateral Raises" / "Dumbbell Lateral Raise" → `Dumbbell Lateral Raise` (**per-hand** weight, never doubled)
- "Bulgarian Split Squat" → `Bulgarian Split Squat` (per-hand dumbbell weight)
- "Spider Curls (Dumbbell)" / "Spider Curls" → `Spider Curls` (per-hand)
- "Seated DB Press" → `Seated DB Press` (per-hand)
- "Preacher Curls (Barbell)" / "Preacher Curl" → `Preacher Curls`
- "Triceps Pushdowns" → `Tricep Pushdown`
- "Dumbbell Romanian Deadlift" / "Dumbbell RDL" → `Dumbbell RDL`
- "Single-Leg Leg Press" → `Single-Leg Press`
- "GHD Back Extension" / "Back Extension (Machine)" / "Machine Back Extension" → `GHD Back Extension` (BW-loaded only — skip, see step 2)
- "Rear Delt Machine" / "Machine Rear Delt Fly" → `Rear Delt Machine`
- "Bench Press" → `Bench Press` (unless explicitly close-grip → `Close-Grip Bench Press`)
- "DB Pullover" / "Dumbbell Pullover" → `DB Pullover`
- "Smith Machine Incline Press" / "Smith Machine Press" → `Smith Machine Press`
- "Barbell Split Squat" → `Barbell Split Squat` (total barbell weight, not per-side)

## Worked example

Log line `actual: 225*10` on Romanian Deadlift: `225 × (1 + 0.0333 × 10) = 299.9 → 300`. Existing `Romanian Deadlift: 234` → update to `300`. If the log's best had been 220, the JSON would stay at 234.
