# AGENTS.md

Workout generator web app. Single Go package, stdlib only (`go.mod` has zero dependencies), no CI, no lint config.

## Commands

- Run: `go run .` — serves on `:8080` (override with `PORT`), reads `data/` (override with `DATA_DIR`).
- Full check: `go build ./... && go vet ./... && go test ./... && gofmt -l .`
- Single test: `go test -run TestName .` (no fixtures or services needed; handler tests read `templates/` from disk, so run from repo root).

## Gotchas

- **Restart to see changes.** `data/*.json` is loaded once at startup and `templates/*.html` is `go:embed`'d at compile time — neither is watched.
- **Generation is deterministic:** same `day` + `seed` + slot overrides → same workout (sha256 of `seed-slotIndex` seeds a PCG PRNG in `generate.go`). Pass `?seed=` to reproduce; tests depend on fixed seeds.
- **`prs.json` keys must exactly match exercise names in `data/program.json`** (`choices` arrays) or the exercise gets no PR and no weight targets. Values are estimated 1RMs in lbs; per-hand for dumbbell moves (e.g. lateral raises), never doubled.
- **Adding a dependency touches the Dockerfile** — it copies only `go.mod` (no `go.sum` exists).
- **The Docker image ships only the binary.** Mount `data/` into the container and set `DATA_DIR`, or it exits at startup.
- `uha-hyp-5k` in the repo root is a local build artifact, not source.

## Conventions

- Feature work follows the docs pipeline: design spec in `docs/specs/`, then a TDD implementation plan in `docs/plans/`, then code. See existing files there for format.
