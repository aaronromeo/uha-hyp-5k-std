# AGENTS.md

Workout generator web app. Single Go package, stdlib only (`go.mod` has zero dependencies), no CI, no lint config.

## Commands

- Run: `go run .` — serves on `:8080` (override with `PORT`), reads `data/` (override with `DATA_DIR`).
- Full check: `go build ./... && go vet ./... && go test ./... && gofmt -l .`
- `gofmt -l .` has a pre-existing dirty baseline: it lists `generate.go`, `types.go`, and three test files (space-indent/alignment drift). You didn't break it — don't mass-reformat inside feature diffs, just don't add new violations.
- Single test: `go test -run TestName .` (no fixtures or services needed; handler tests read `templates/` from disk, so run from repo root).

## Gotchas

- **Restart to see changes.** `data/*.json` is loaded once at startup and `templates/*.html` is `go:embed`'d at compile time — neither is watched.
- **Generation is deterministic:** same `day` + `seed` + slot overrides → same workout (sha256 of `seed-slotIndex` seeds a PCG PRNG in `generate.go`). Pass `?seed=` to reproduce; tests depend on fixed seeds. Slot overrides are `?s0=Exercise+Name`, `?s1=…` by slot index. The Obsidian header's ` - YYYYMMDD` stamp is the generation date (server local time; tests use fixed clocks), so same-seed output differs across days only in that stamp.
- **User-facing ranges use a real en-dash (U+2013, `–`), never a hyphen** — e.g. `"165–180 lbs x 1–5"`. Tests assert the exact bytes; see the convention note in `docs/plans/` if editing range formatting.
- **`prs.json` keys must exactly match exercise names in `data/program.json`** (`choices` arrays) or the exercise gets no PR and no weight targets. Values are estimated 1RMs in lbs; per-hand for dumbbell moves (e.g. lateral raises), never doubled.
- **Adding a dependency touches the Dockerfile** — it copies only `go.mod` (no `go.sum` exists).
- **The Docker image ships only the binary.** Mount `data/` into the container and set `DATA_DIR`, or it exits at startup.
- `go build` drops `uha-hyp-5k` in the repo root (named after the module, not the `-std` directory). It's a local build artifact, not source, and isn't gitignored — don't commit it.

## Conventions

- Feature work follows the docs pipeline: design spec in `docs/specs/`, then a TDD implementation plan in `docs/plans/`, then code. See existing files there for format.

## Agent skills

### Issue tracker

Issues and specs live as local markdown under `.scratch/<feature-slug>/`. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context. See `docs/agents/domain.md`.
