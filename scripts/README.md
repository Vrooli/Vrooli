# Scripts Directory

Repo-root orchestration lives in the Go control plane (`cmd/vrooli`, `cmd/vrooli-api`, `internal/*`, plus the root `Makefile`). Use `make setup`, `make dev`, `make build`, or `vrooli --help` to drive it. Nothing in the Go control plane or the `Makefile` shells out to this directory.

What remains here is **legacy bash on its way out**, kept only because a named consumer still needs it. Do not add new scripts; port to Go instead.

## Contents

- `scripts/lib/` — shared shell helpers (`log`, `var`, `format`, `flow`, `trash`, …) sourced by the resource `lib/` scripts that have not been ported yet.
- `scripts/resources/lib/` — resource-side shell frameworks (status, health, docker, backup, credentials) sourced by the same resource scripts.
- `scripts/resources/common.sh` + `common/config-manager.js` — service.json read/write helpers used by `claude-code`, `minio`, and `postgres`.
- `scripts/resources/populate/` — scenario content population; invoked from the `lifecycle.setup` phase of ~29 scenario `service.json` files. No Go successor yet.
- `scripts/resources/tests/lib/` — integration-test helpers sourced by four resources' `test/integration-test.sh`.
- `scripts/resources/port_registry.{sh,json}` — legacy port allocation table. Superseded in practice: `resource.json` `ports` takes precedence in `internal/resources/env`, and the JSON is only a fallback. Slated for removal.
- `scripts/package-vrooli-release.sh` — release packaging, invoked by `.github/workflows/vrooli-release.yml`.
- `scripts/emergency-watchdog.sh`, `scripts/perf/` — operator tools referenced from runbooks.
- `scripts/validate-go-cli-consumers.sh` — standalone validator.

## Retirement path

The bulk of this directory exists to serve `resources/*/lib/*.sh`. All 25 resources now ship a Go CLI (`cli.adapter.kind: go_module`), and only the five coding-agent resources still declare a shell entrypoint (`source .../lib/install.sh`) in `resource.json`. Once those are ported and the orphaned resource bash is removed, most of `scripts/lib/` and `scripts/resources/lib/` becomes unreferenced and can be deleted.

Resource structure and validation are governed by `.vrooli/schemas/resource.schema.json` (enforced by `internal/resources/validate.go`), not by anything in this directory.
