# Scripts Directory

Repo-root orchestration lives in the Go control plane (`cmd/vrooli`, `cmd/vrooli-api`, `internal/*`, plus the root `Makefile`). Use `make setup`, `make dev`, `make build`, or `vrooli --help` to drive it.

## Contents

- `scripts/resources/` — resource-side shell frameworks, templates, and validators consumed by resource `lib/` scripts.
- `scripts/lib/` — shared shell helpers still sourced by resource and scenario code.
- `scripts/migrate_candidates/` — frozen bash references from master kept as input for deciding what to port; not wired into any runtime path. See `scripts/migrate_candidates/README.md`.
- `scripts/validate-go-cli-consumers.sh` — standalone validator.
