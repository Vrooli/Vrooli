# Scripts Directory

Repo-root orchestration lives in the Go control plane (`cmd/vrooli`, `cmd/vrooli-api`, `internal/*`, plus the root `Makefile`). Use `make setup`, `make dev`, `make build`, or `vrooli --help` to drive it. Nothing in the Go control plane or the `Makefile` shells out to this directory.

The remaining shell files are deliberately limited to release packaging, the
emergency watchdog, the performance sampler, and the Go-consumer validator.
Resource lifecycle, configuration, diagnostics, and tests are Go-owned.

## Contents

- `scripts/package-vrooli-release.sh` — release packaging, invoked by `.github/workflows/vrooli-release.yml`.
- `scripts/emergency-watchdog.sh`, `scripts/perf/` — operator tools referenced from runbooks.
- `scripts/validate-go-cli-consumers.sh` — standalone validator.

Resource structure and validation are governed by `.vrooli/schemas/resource.schema.json` (enforced by `internal/resources/validate.go`), not by anything in this directory.
