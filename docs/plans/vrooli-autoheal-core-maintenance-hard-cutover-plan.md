# Vrooli Autoheal Core Maintenance Hard-Cutover Plan

## Goal

Rebuild the `vrooli-autoheal` integration points that drifted during the bash-to-Go migration so the scenario consumes only current Go-native Vrooli contracts.

This is a hard cutover:

- No backward-compatibility parsing
- No legacy `scenario_data` support
- No local orphan or stale-lock heuristics in autoheal
- No fallback implementation for orphan or lock handling
- Core maintenance and lock APIs are authoritative

## Scope

This plan covers:

- Scenario status parsing
- Orphan-process monitoring and cleanup
- Stale-lock monitoring and cleanup
- Port diagnostics integration
- Test and documentation realignment

This plan does not include unrelated autoheal feature work.

## Core Decisions

### 1. Scenario status contract is Go-native only

`vrooli-autoheal` must parse only the current Go CLI JSON emitted by:

- `vrooli scenario status <name> --json`

Authoritative fields:

- `success`
- `scenario.status`
- `scenario.health_status`
- other current fields needed for diagnostics

Old payload shapes such as `scenario_data.*` are removed from autoheal entirely.

### 2. Core maintenance commands are authoritative

`vrooli-autoheal` must not classify orphan processes or stale locks locally.

Instead it must consume:

- `vrooli orphans --json`
- `vrooli locks --json`
- `vrooli cleanup orphans`
- `vrooli cleanup locks`
- `vrooli diagnose-port`

No local fallback is permitted for these flows.

### 3. Autoheal becomes a thin evaluator over core state

For orphans and stale locks, autoheal should:

- fetch authoritative state from core commands
- summarize, score, and surface it
- invoke core cleanup commands for healing

It should not:

- scan `/proc` independently for orphan classification
- parse lock files independently for stale-lock truth
- mutate lock files directly

## Target Architecture

Create a dedicated contract/integration layer inside `scenarios/vrooli-autoheal/api/internal/integrations/vrooli`.

Suggested files:

- `scenario_status.go`
- `resource_status.go`
- `maintenance.go`
- `command_runner.go`

Responsibilities:

- own command invocation for current Vrooli contracts
- own JSON DTOs for current command output
- validate required fields
- return typed results to checks and healers

Non-responsibilities:

- business scoring
- health severity decisions
- action policy

Checks and healers should depend on typed integration results, not raw JSON blobs.

## Implementation Phases

### Phase 1: Scenario Status Hard Cutover

#### Objectives

- remove legacy `scenario_data` parsing
- parse only the current `scenario` shape
- make scenario health evaluation depend on the current Go CLI contract

#### Work

- Replace the legacy DTO in `scenarios/vrooli-autoheal/api/internal/checks/vrooli/scenario.go`
- Add a canonical typed model for current scenario status JSON in the new integration package
- Refactor scenario check execution to call the integration layer instead of parsing inline
- Keep malformed or incomplete payloads as explicit parse/contract failures
- Preserve current business semantics for healthy/degraded/unhealthy/stopped, but only over the new contract

#### Deliverables

- no `scenario_data` references in autoheal production code
- no inline JSON parsing in scenario checks outside the integration package

### Phase 2: Maintenance Command Integration

#### Objectives

- remove duplicated orphan and stale-lock truth models
- consume only core maintenance commands

#### Work

- Add typed command adapters for:
  - `vrooli orphans --json`
  - `vrooli locks --json`
  - `vrooli diagnose-port`
- Refactor `vrooli-orphans` to score authoritative orphan output from core
- Refactor `vrooli-stale-locks` to score authoritative lock output from core
- Remove direct filesystem and `/proc` truth paths from these checks

#### Deliverables

- orphan check is a thin wrapper over `vrooli orphans --json`
- stale-lock check is a thin wrapper over `vrooli locks --json`

### Phase 3: Healing Hard Cutover

#### Objectives

- move cleanup authority to core commands
- eliminate local cleanup mutation paths

#### Work

- Refactor orphan cleanup actions to invoke:
  - `vrooli cleanup orphans`
- Refactor stale-lock cleanup actions to invoke:
  - `vrooli cleanup locks`
- Refactor diagnostics to invoke:
  - `vrooli diagnose-port`
- Remove local:
  - orphan kill logic
  - lock-file delete logic
  - stale-lock mutation through state-reader abstractions

#### Deliverables

- no orphan cleanup path in autoheal directly kills processes except through core maintenance commands
- no stale-lock cleanup path in autoheal directly removes lock files

### Phase 4: Package Boundary Cleanup

#### Objectives

- make the codebase easier to reason about
- prevent drift from reappearing

#### Work

- Remove obsolete local contract structs and helpers
- Remove now-redundant lock/state parsing from autoheal where no longer used
- Keep scenario/resource checks focused on evaluation and severity mapping
- Keep command contract parsing isolated to integrations
- Add package docs for the new integration layer

#### Deliverables

- cleaner ownership boundaries
- reduced duplicated concepts

### Phase 5: Test Rebuild

#### Objectives

- align tests with current Go contracts
- catch contract drift early

#### Work

- Replace old `scenario_data` fixtures with current `scenario` payloads
- Add parser contract tests for:
  - healthy running scenario
  - degraded running scenario
  - unhealthy running scenario
  - stopped scenario
  - malformed payload
  - missing required fields
- Add maintenance integration adapter tests for:
  - `vrooli orphans --json`
  - `vrooli locks --json`
  - `vrooli cleanup orphans`
  - `vrooli cleanup locks`
  - `vrooli diagnose-port`
- Add check-level regression tests proving:
  - healthy running scenarios are not marked stopped
  - orphan and stale-lock counts mirror core output exactly
  - cleanup actions delegate to core commands only
- Add integration-style command-mocking tests for end-to-end check and heal flows
- Add smoke coverage where feasible against the real `vrooli` command in controlled environments

#### Deliverables

- no tests assert legacy contracts
- regression coverage exists for the drift that caused this work

### Phase 6: Documentation Rewrite

#### Objectives

- make ownership explicit
- prevent future duplicated heuristics

#### Work

- Update autoheal docs to state:
  - scenario health is based on current Go CLI scenario status contracts
  - orphan and stale-lock authority lives in core maintenance commands
  - no backward compatibility exists for the pre-Go status schema
  - no fallback orphan/lock implementation exists in autoheal
- Update architecture and check reference docs
- Add a short contract-ownership section for future maintainers

#### Deliverables

- docs match implementation
- docs explicitly describe the hard cutover

## File-Level Change Areas

### Add

- `scenarios/vrooli-autoheal/api/internal/integrations/vrooli/scenario_status.go`
- `scenarios/vrooli-autoheal/api/internal/integrations/vrooli/resource_status.go`
- `scenarios/vrooli-autoheal/api/internal/integrations/vrooli/maintenance.go`
- `scenarios/vrooli-autoheal/api/internal/integrations/vrooli/command_runner.go`
- integration package tests alongside each file

### Refactor

- `scenarios/vrooli-autoheal/api/internal/checks/vrooli/scenario.go`
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli/resource.go`
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli/orphan.go`
- `scenarios/vrooli-autoheal/api/internal/checks/vrooli/stale_locks.go`
- `scenarios/vrooli-autoheal/api/internal/healing/healers/scenario_healer.go`
- `scenarios/vrooli-autoheal/api/internal/healing/healers/resource_healer.go`
- any shared action strategy files that should route to the integration layer

### Remove Or Retire

- legacy scenario-status DTOs based on `scenario_data`
- local orphan truth heuristics no longer used
- local stale-lock truth and deletion paths no longer used
- tests that encode obsolete CLI schemas

## Test Strategy

### Unit Tests

- integration package DTO parsing
- integration package error handling
- check scoring over typed integration results
- healer/action dispatch over typed command responses

### Regression Tests

- current scenario JSON payload does not misclassify running scenarios
- orphan results equal core output
- lock results equal core output
- cleanup actions shell out only to core maintenance commands

### Integration Tests

- mocked command execution across full scenario-check flow
- mocked command execution across orphan/lock check flow
- mocked command execution across healing flow

### Smoke Tests

Use targeted smoke coverage where the environment permits:

- `vrooli scenario status <name> --json`
- `vrooli orphans --json`
- `vrooli locks --json`

## Documentation Requirements

Update:

- `scenarios/vrooli-autoheal/docs/reference/checks/scenario-check.md`
- orphan and stale-lock check references
- architecture documentation
- troubleshooting documentation where maintenance behavior is described

Each updated doc should clearly state:

- current source of truth
- command contract used
- no compatibility guarantee for old schema shapes

## Rollout Order

1. Add the new integration package and current scenario-status parser.
2. Cut scenario checks over to the new parser.
3. Update scenario-check tests to current payloads.
4. Add maintenance command adapters for orphans, locks, and diagnose-port.
5. Replace orphan and stale-lock checks with core-backed implementations.
6. Replace cleanup actions with direct core command delegation.
7. Remove obsolete local parsing and mutation code.
8. Update docs.
9. Run targeted and full autoheal test suites.

## Definition Of Done

- No `scenario_data` references remain in autoheal production code or tests.
- Scenario health checks consume only the current Go CLI scenario-status contract.
- Orphan checks consume only core maintenance command output.
- Stale-lock checks consume only core maintenance command output.
- Orphan cleanup delegates only to `vrooli cleanup orphans`.
- Stale-lock cleanup delegates only to `vrooli cleanup locks`.
- Port diagnostics delegate only to `vrooli diagnose-port`.
- No local fallback exists for orphan or stale-lock truth or healing.
- Tests cover current contracts and regression cases.
- Documentation reflects the hard-cut architecture and ownership model.

## Non-Goals

- Supporting old bash-era or transitional schemas
- Preserving local orphan/stale-lock heuristics as backup logic
- Broad refactors outside the affected check/healing/integration surfaces
