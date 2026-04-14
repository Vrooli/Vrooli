# Go-Native Scenario and Resource CLI Plan

## Status

Implementation status as of 2026-04-14:

- Phase 1 complete
- Phase 2 complete
- Phase 3 complete
- Phase 4 complete
- Phase 5 implemented for the active control path

Validated locally on Linux with focused Go test coverage for:

- CLI discovery/install/ensure behavior
- stale reinstall behavior via install metadata
- setup wiring
- scenario/resource ensure-install command flow
- active resource control without `cli.sh` fallback

Cross-platform verification remains a follow-up validation task. This document otherwise describes the intended final architecture.

Do not preserve or extend legacy shell compatibility while implementing this plan. Do not add fallback behavior for old shell paths. Do not keep dead code for historical reasons.

## Goal

Unify scenarios and resources around one clean CLI model:

- `vrooli` remains the control plane
- every enabled scenario has an installed Go CLI
- every enabled resource has an installed Go CLI
- installed CLIs are cross-platform Go binaries
- installed CLIs rebuild automatically when stale
- standard lifecycle behavior is Go-native only
- shell is removed from core install/start/stop/restart/status/logs/test functionality

## Baseline

The current codebase already has the core pieces needed for this direction.

### Already Go-native

- Project lifecycle is Go-native in `internal/setup`
- Scenario lifecycle is Go-native in `internal/lifecycle`
- Resource discovery/control is mostly Go-native in `internal/resources`
- Active resources are expected to be `manifest-native`, not `legacy-adapter`
- Scenario CLI install/stale logic already exists in `packages/cli-core`

### Current Gaps

- Scenario CLI installation is not yet enforced consistently from `vrooli setup` and `vrooli scenario start`
- Resource CLIs are still inconsistent and shell-heavy
- Resource standard lifecycle is largely in Go, but direct `resource-<name>` CLIs are not yet standardized as Go binaries
- Some resource install flows still rely on shell installer helpers
- `scripts/lib/resources/` still exists even though it should not remain part of the final architecture

## Architecture Decision

### Final split of responsibilities

Use this boundary:

- `internal/`
  - owns control-plane logic
  - owns scenario lifecycle
  - owns resource lifecycle
  - owns install orchestration policy
  - owns discovery of which CLIs should exist
- `packages/cli-core`
  - owns shared CLI binary concerns
  - owns installer/build logic
  - owns stale detection and auto-rebuild
  - owns shared CLI scaffolding
  - owns shared env/config/runtime helpers for installed CLIs

This is the correct conceptual split.

- `internal/` is platform control-plane code
- `packages/cli-core` is shared reusable CLI runtime/tooling

Do not move generic installer/stale-check logic out of `packages/cli-core` into `internal/`.

If resource CLIs need reusable helpers parallel to scenario helpers, add them to `packages/cli-core` or a sibling shared package, not `internal/`.

## Final Model

### Scenarios

Each supported scenario CLI should follow:

```text
scenarios/<name>/
  .vrooli/service.json
  cli/
    go.mod
    main.go
    install.sh
    install.ps1
```

Scenario CLIs should:

- be Go modules
- use `packages/cli-core`
- install into `~/.vrooli/bin`
- embed build fingerprint, build timestamp, and source root
- auto-rebuild when stale

### Resources

Each active resource CLI should follow:

```text
resources/<name>/
  resource.json
  cli/
    go.mod
    main.go
    install.sh
    install.ps1
```

Resource CLIs should:

- be Go modules
- use the same shared installer/build/stale model as scenario CLIs
- install into `~/.vrooli/bin`
- provide standard lifecycle commands
- call the Go control plane, not shell scripts

Do not keep `resources/<name>/cli.sh` in the final design.

## Behavior Requirements

### `vrooli setup`

`vrooli setup` must:

- install or rebuild `vrooli`
- install CLIs for all enabled resources
- install CLIs for all supported scenario Go CLIs
- place binaries in `~/.vrooli/bin`
- make PATH expectations explicit

Recommended policy:

- install all enabled resource CLIs
- install all scenario Go CLIs in the repo

This is the simplest behavior and best matches operator expectations that CLIs should already be available after setup.

### `vrooli scenario start <name>`

Before lifecycle start:

- ensure the scenario CLI exists in `~/.vrooli/bin`
- rebuild it if stale

This guarantees the scenario CLI is available after start even if setup was skipped or partial.

### `vrooli resource <verb> <name>`

Before resource execution:

- ensure the resource CLI exists in `~/.vrooli/bin`
- rebuild it if stale

This keeps installed resource CLIs deterministic and current.

### Installed CLIs

Installed scenario and resource CLIs must:

- be Go binaries
- be cross-platform
- rebuild automatically when stale
- avoid shell scripts for standard lifecycle behavior

## Resource CLI Design

### Resource CLI runtime

The first clean resource CLI implementation should be thin.

For standard commands, the resource CLI should delegate to the Go control plane through `vrooli`, for example:

- `resource-postgres status` -> `vrooli resource status postgres`
- `resource-postgres start` -> `vrooli resource start postgres`
- `resource-postgres logs` -> `vrooli resource logs postgres`

This keeps behavior entirely Go-native and reuses the existing control plane immediately.

This is preferable to preserving `cli.sh`.

Later, if needed, resource CLIs can call shared Go packages directly instead of invoking `vrooli`, but that should not block the greenfield implementation.

### Standard command set

Every resource CLI should support:

- `info`
- `status`
- `install`
- `uninstall`
- `start`
- `stop`
- `restart`
- `logs`

Optional resource-specific commands may exist, but they are secondary to standard lifecycle consistency.

## Shared CLI Work

### Reuse `packages/cli-core`

Reuse the existing pieces in `packages/cli-core`:

- `cmd/cli-installer`
- `cliutil.StaleChecker`
- shared build fingerprint embedding
- install wrappers
- common CLI app/config/env helpers

Do not invent a separate stale-checking or install mechanism for resource CLIs.

### Extend `packages/cli-core` for resources

Add resource equivalents of the scenario helpers:

- `cliapp.ResourceApp`
- `cliapp.StandardResourceEnv`
- optional resource CLI helpers in `cliutil`

`ResourceApp` should provide:

- standard command wiring
- stale checker integration
- global flags consistent with scenario CLIs
- a clean extension point for resource-specific commands

## New Shared Install Manager

Add a new Go package under `internal/` for installed CLI orchestration.

Suggested package:

- `internal/cliinstall/`

Responsibilities:

- discover installable scenario CLI modules
- discover installable resource CLI modules
- resolve binary names
- check installed binary presence
- invoke `packages/cli-core/cmd/cli-installer`
- decide whether install/reinstall is needed
- expose install/ensure APIs used by setup/start/resource commands

Suggested API surface:

- `InstallScenarioCLI(name)`
- `InstallResourceCLI(name)`
- `InstallAllScenarioCLIs()`
- `InstallEnabledResourceCLIs()`
- `EnsureScenarioCLI(name)`
- `EnsureResourceCLI(name)`

This package should call the installer tool directly. It should not shell out to per-item `install.sh`.

## Implementation Workstreams

### Workstream A: Shared install orchestration

1. Add `internal/cliinstall`
2. Implement scenario CLI discovery
3. Implement resource CLI discovery
4. Implement binary-name resolution rules
5. Implement direct installer invocation
6. Implement ensure/install semantics

### Workstream B: Resource CLI shared runtime

1. Add `cliapp.ResourceApp`
2. Add `cliapp.StandardResourceEnv`
3. Add any shared helper utilities needed for resource CLI delegation
4. Ensure the stale checker path matches scenario CLI behavior

### Workstream C: Resource CLI template

Create one canonical resource CLI template:

- `resources/<name>/cli/go.mod`
- `resources/<name>/cli/main.go`
- `resources/<name>/cli/install.sh`
- `resources/<name>/cli/install.ps1`

The installed binary name must be:

- `resource-<name>`

### Workstream D: Setup auto-install

Wire install manager into `internal/setup` so that:

- `vrooli setup` installs all enabled resource CLIs
- `vrooli setup` installs all supported scenario Go CLIs

### Workstream E: Scenario ensure-install

Wire ensure-install into scenario execution flow so that:

- `vrooli scenario start <name>` ensures scenario CLI exists and is current
- optionally `vrooli scenario test <name>` does the same

### Workstream F: Resource ensure-install

Wire ensure-install into resource command execution so that:

- `vrooli resource <verb> <name>` ensures resource CLI exists and is current

### Workstream G: Active resource migration

Migrate all enabled resources first.

For each enabled resource:

1. add `resources/<name>/cli/` Go module
2. install binary `resource-<name>`
3. route standard commands through `ResourceApp`
4. remove `cli.sh` once parity is complete

Do not try to migrate all historical resources before the active set.

### Workstream H: Shell removal

Once enabled resources are migrated:

1. delete `scripts/lib/resources/`
2. remove active resource `cli.sh` usage
3. remove active control-plane fallback to `cli.sh`
4. remove resource-registry usage if no longer required by live flows
5. remove docs that present shell CLIs as current architecture

## Explicit Non-Goals

These are out of scope for the greenfield implementation:

- no compatibility layer for old `cli.sh`
- no fallback “if shell script exists, use it”
- no preservation of `scripts/lib/resources/`
- no hybrid final design where shell CLIs remain first-class
- no migration of every scenario immediately
- no movement of shared CLI install/stale logic into `internal/`

## Recommended Build Order

### Phase 1: Foundations

1. Define installable CLI discovery rules
2. Add `internal/cliinstall`
3. Extend `packages/cli-core` for resource CLIs
4. Add canonical resource CLI template

### Phase 2: Resource CLI runtime

1. Implement `ResourceApp`
2. Implement standard resource command delegation to `vrooli resource`
3. Add stale rebuild integration
4. Add `install.sh` and `install.ps1` wrappers for resource CLIs

### Phase 3: Auto-install behavior

1. Wire scenario CLI install into `vrooli setup`
2. Wire resource CLI install into `vrooli setup`
3. Wire scenario CLI ensure-install into `vrooli scenario start`
4. Wire resource CLI ensure-install into `vrooli resource ...`

### Phase 4: Active resource migration

1. Migrate all enabled resources to Go CLI modules
2. Remove their `cli.sh`
3. Remove direct use of `scripts/lib/resources/install-resource-cli.sh`

### Phase 5: Core cleanup

1. Delete `scripts/lib/resources/`
2. Remove `cli.sh` fallback from active resource control
3. Remove resource-registry if obsolete
4. Update docs/templates/tests to final-state only

## Validation Plan

### Unit tests

Add or expand tests for:

- installable scenario CLI discovery
- installable resource CLI discovery
- binary name resolution
- install plan generation
- install-if-missing behavior
- rebuild-if-stale behavior
- skip-if-current behavior
- scenario start ensuring CLI installation
- resource command ensuring CLI installation

### Integration tests

Add temp-home integration tests that verify:

- `vrooli setup` installs expected binaries into `~/.vrooli/bin`
- `vrooli scenario start <name>` installs a missing scenario CLI
- `vrooli resource status <name>` installs a missing resource CLI
- stale scenario CLI rebuilds on invocation
- stale resource CLI rebuilds on invocation

### Cross-platform validation

At minimum:

- Linux: full install + invocation + stale rebuild tests
- Windows: install path + build + invocation tests via `install.ps1`
- macOS: build/install smoke in CI if available, otherwise documented local validation checklist

### Acceptance criteria

The implementation is complete only when all of the following are true:

- after `vrooli setup`, all enabled resources have installed Go CLIs in `~/.vrooli/bin`
- after `vrooli setup`, all supported scenario Go CLIs are installed in `~/.vrooli/bin`
- `vrooli scenario start <name>` guarantees the scenario CLI exists afterward
- `vrooli resource <verb> <name>` guarantees the resource CLI exists afterward
- installed scenario/resource CLIs rebuild themselves when stale
- no core lifecycle/install/status/logs path depends on shell scripts
- `scripts/lib/resources/` has been deleted
- active resources no longer depend on `cli.sh`
- docs describe only the Go-native model

## Final Repo Shapes

### Scenario

```text
scenarios/<name>/
  .vrooli/service.json
  cli/
    go.mod
    main.go
    install.sh
    install.ps1
```

### Resource

```text
resources/<name>/
  resource.json
  cli/
    go.mod
    main.go
    install.sh
    install.ps1
```

### Shared

```text
packages/cli-core/
  cliapp/
    scenario_app.go
    resource_app.go
  cliutil/
    stalechecker.go
  cmd/
    cli-installer/
```

### Control plane

```text
internal/
  lifecycle/
  resources/
  cliinstall/
```

## Key Summary

The correct direction is:

- keep control-plane behavior in `internal/`
- keep reusable CLI install/stale/scaffolding in `packages/cli-core`
- make resource CLIs thin Go binaries that reuse the Go control plane
- make scenario/resource auto-install explicit and deterministic
- remove shell from core lifecycle and CLI installation behavior

This plan intentionally targets the clean final state and excludes legacy/compatibility/dead-code preservation.
