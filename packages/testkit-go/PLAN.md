# testkit-go Implementation Plan

## Purpose

Initialize `packages/testkit-go/` as the canonical Go test-support package for Vrooli and fully migrate the appropriate Go tests onto it so the resulting test architecture is clean, professional, low-drift, and maintainable.

This plan focuses on:

- shared Go test fixture construction
- repo/layout/manifest consolidation
- seam adoption where tests currently rely on real behavior unnecessarily
- cleanup of duplicated helper layers
- documentation and validation

## Desired end state

- `packages/testkit-go/` is the default reusable Go test fixture package.
- Root-module tests and package-module tests no longer duplicate valid repo-contract fixtures, valid service/resource manifests, or common file-writing helpers.
- Test seams exist where logic-heavy tests currently rely on real subprocesses, shell behavior, or fragile filesystem setup.
- Edge-case fixture surfaces are isolated so core tests stay on the canonical contracts.
- Local duplicated test helpers are removed after migration.
- `testkit-go` itself is documented and directly tested.

## Architectural boundaries

### `testkit-go` owns

- reusable Go test fixture construction
- typed builders for valid repo/layout/manifest fixtures
- explicit malformed fixture helpers for negative tests
- common file/JSON/executable writers
- focused fixture construction for edge-case runtime scenarios that still need direct setup

### `testkit-go` does not own

- production logic
- domain behavior assertions for consumer packages
- package-specific test doubles that are not reusable
- broad assertion abstractions
- end-to-end orchestration beyond verifying fixtures are consumable

### Ownership boundaries with adjacent packages

- `packages/repo-contract-go` owns repo contract APIs and semantics.
- `packages/testkit-go` owns shared Go test fixture setup using those semantics.
- Consumer packages own behavior and output assertions.

## Proposed structure

Initial target layout:

```text
packages/testkit-go/
  README.md
  PLAN.md
  repo.go
  repo_test.go
  files.go
  files_test.go
  vrooli/
    manifests.go
    manifests_test.go
    runtime_helpers.go
    runtime_helpers_test.go
```

This split is already justified by import-boundary constraints:

- the root `testkit-go` package must remain cycle-safe for repo/layout/file helpers
- Vrooli-domain fixture helpers that depend on `internal/scenario`, `internal/resources/manifest`, or `internal/process` belong under `packages/testkit-go/vrooli`

That keeps:

- `internal/scenario` tests free to import base repo/file helpers
- `packages/repo-contract-go` free to import shared repo/file helpers
- richer project/scenario/resource fixture helpers available to higher-level consumers without collapsing everything into one cyclic package

## Initial API direction

The first iteration should support:

- `NewRepoFixture(t)`
- repo fixture options for `scenarios` vs `apps`
- support-doc and exceptions setup
- cycle-safe base file/JSON/path helpers
- typed project/scenario/resource manifest builders
- valid JSON writers
- explicit malformed JSON writers
- relative file/executable writers
- focused fixture writers for edge-case runtime artifacts still covered by tests

## Migration targets

### First-wave adopters

These are the highest-value consumers because they currently duplicate repo/layout/manifest test setup:

- `cmd/vrooli/test_helpers_test.go`
- `cmd/vrooli-api/main_test.go`
- `internal/repocontractcheck/checks_test.go`
- `internal/scenario/scenario_test.go`
- `packages/cli-core/cliapp/scenario_app_test.go`
- `packages/repo-contract-go/*_test.go`

### Second-wave adopters

These already use some shared helpers but still contain local duplication or fixture churn:

- `internal/project/project_test.go`
- `internal/api/app_test.go`
- `internal/setup/setup_test.go`
- `internal/resources/resources_test.go`
- `internal/lifecycle/lifecycle_test.go`

## Seams to introduce or standardize

After fixture consolidation, prioritize seam cleanup in packages where tests still rely on real behavior:

- command execution
- process liveness
- process environment reads
- time/clock
- health polling
- filesystem probe boundaries where behavior is not the subject under test

The goal is:

- deterministic unit tests for logic
- explicit smoke tests for real platform/shell behavior

## Documentation requirements

Before the migration is complete:

- `packages/testkit-go/README.md` must describe scope and usage
- `docs/CONTRIBUTING.md` should gain a short section directing Go tests to prefer `testkit-go`
- package doc comments should explain each major area if multiple files/subpackages are introduced

## Validation requirements

At the end of the migration, the following must be green:

```bash
go test ./packages/testkit-go/...
go test ./packages/repo-contract-go/...
go test ./packages/cli-core/...
go test ./internal/scenario ./internal/process ./internal/lifecycle ./internal/setup ./internal/resources ./internal/api ./internal/project ./internal/repocontractcheck ./internal/repocontract ./cmd/vrooli ./cmd/vrooli-api
make validate-repo-contract
```

If additional package groups are touched during migration, expand the validation set accordingly.

## Phased checklist

### Phase 0: Foundation

Goal: establish `testkit-go` as a documented package with a clear scope.

- [x] Create `packages/testkit-go/`
- [x] Add package README with scope, non-goals, and adoption intent
- [x] Add this implementation plan with phased checklist
- [x] Decide whether the first version is one package or a few focused subpackages
- [x] Add package-level Go files so `go test ./packages/testkit-go/...` is meaningful

Acceptance criteria:

- `packages/testkit-go/` exists as a real package area, not just an empty directory
- scope is documented locally

### Phase 1: Repo fixture core

Goal: create one canonical repo fixture API for Go tests.

- [x] Implement canonical repo fixture creation
- [x] Support fixture root and home temp dirs
- [x] Support cloning the live repo contract into a temp fixture repo
- [x] Support patching scenario dir between `scenarios` and `apps`
- [x] Support writing repo-contract exceptions
- [x] Support writing support docs
- [x] Add direct tests for the repo fixture API

Acceptance criteria:

- consumers can build a valid temp repo fixture without hand-writing repo-contract JSON
- `repo_test.go` validates layout creation and scenario-dir overrides

### Phase 2: Typed manifest builders

Goal: centralize valid project/scenario/resource manifest construction.

- [x] Move and refine reusable manifest builders into `testkit-go`
- [x] Support project manifests
- [x] Support scenario manifests with lifecycle, ports, dependencies
- [x] Support resource manifests with portability and runtime fields
- [x] Add explicit malformed manifest helpers for negative tests
- [x] Add unit tests for the manifest builders

Acceptance criteria:

- valid manifest fixtures no longer require raw JSON in migrated suites
- negative tests use clearly-named malformed helpers when raw JSON is still required

### Phase 3: Shared file and JSON helpers

Goal: eliminate duplicated write helpers across migrated Go test suites.

- [x] Add shared file writer
- [x] Add shared executable writer
- [x] Add shared JSON writer with stable formatting and trailing newline
- [x] Add shared raw JSON writer
- [x] Add shared malformed JSON writer helpers
- [x] Add tests for newline, permissions, and malformed-output behavior

Acceptance criteria:

- common `write file/json/executable` helpers no longer need to be redefined per package

### Phase 4: Edge-case fixture layer

Goal: isolate non-canonical runtime fixtures still intentionally covered by tests.

- [x] Add helpers for port registry fixtures
- [x] Add helpers for resource CLI fixtures used by direct runtime tests
- [ ] Add helpers for remaining defaults/config fragments where still needed
- [ ] Add helpers for resource/scenario marker files used by migrated tests
- [x] Add tests proving the edge-case helpers create the intended paths and contents

Acceptance criteria:

- non-canonical fixture setup becomes explicit and localized
- migrated tests stop hand-rolling edge-case files repeatedly

### Phase 5: First-wave adoption

Goal: migrate the highest-value consumers onto `testkit-go`.

- [x] Migrate `cmd/vrooli/test_helpers_test.go`
- [x] Migrate `cmd/vrooli-api/main_test.go`
- [x] Migrate `internal/repocontractcheck/checks_test.go`
- [x] Migrate `internal/scenario/scenario_test.go`
- [x] Migrate `packages/cli-core/cliapp/scenario_app_test.go`
- [x] Migrate `packages/repo-contract-go/*_test.go`
- [x] Remove duplicated local helpers made obsolete by the migration

Acceptance criteria:

- first-wave adopters build valid fixtures through `testkit-go`
- duplicated repo fixture builders are removed from those areas

### Phase 6: Second-wave adoption

Goal: migrate the larger controller/lifecycle/resource suites.

- [x] Migrate `internal/project/project_test.go`
- [x] Migrate `internal/api/app_test.go`
- [x] Migrate `internal/setup/setup_test.go`
- [x] Migrate `internal/resources/resources_test.go`
- [x] Migrate `internal/lifecycle/lifecycle_test.go`
- [x] Remove now-obsolete local helper code

Acceptance criteria:

- second-wave adopters use typed shared fixtures for valid setup
- remaining raw JSON or shell fixture text is limited to negative tests or explicit edge-case coverage

### Phase 7: Seam hardening

Goal: reduce reliance on real behavior for logic coverage.

- [ ] Audit migrated packages for unnecessary use of real subprocesses or shell behavior
- [ ] Add seams for command execution where needed
- [ ] Add seams for time/clock where needed
- [ ] Add seams for environment reads where needed
- [ ] Add seams for process inspection where needed
- [ ] Convert logic-heavy tests to seam-based unit tests
- [ ] Keep a narrow smoke layer for real behavior

Acceptance criteria:

- logic coverage is mostly deterministic
- smoke tests are explicit and small in number

### Phase 8: File organization cleanup

Goal: align the resulting tests with screaming architecture and clear boundaries.

- [ ] Split oversized test files by domain behavior
- [ ] Rename helpers/files whose names no longer match responsibilities
- [ ] Keep edge-case runtime tests separate from core behavior tests
- [ ] Keep owner-level exact-value tests near the owning package

Acceptance criteria:

- test file names reflect the behavior under test
- suites are easier to navigate by responsibility

### Phase 9: Documentation and adoption rules

Goal: make the new testing architecture legible and durable.

- [x] Expand `packages/testkit-go/README.md` with real usage examples
- [x] Add Go testing guidance to `docs/CONTRIBUTING.md`
- [ ] Document when raw JSON is acceptable
- [ ] Document when helpers should stay package-local
- [ ] Document the seam-vs-smoke testing strategy

Acceptance criteria:

- contributors can discover and correctly use `testkit-go` without reverse-engineering existing tests

### Phase 10: Enforcement and final cleanup

Goal: prevent regression into duplicated fixture infrastructure.

- [x] Add a lightweight check or hygiene test for duplicated canonical fixture builders outside approved locations
- [x] Remove dead code left behind in old helper files
- [x] Delete temporary migration wrappers if any remain
- [x] Run the full validation suite
- [ ] Review the resulting diff for lingering duplicated helper patterns

## Current status

As of 2026-04-13:

- Root-module and package-module migrated tests under `internal/`, `cmd/`, and `packages/` no longer import `internal/testfixture` or `internal/testutil` directly.
- `packages/testkit-go` and `packages/testkit-go/vrooli` have direct tests and are the canonical path for shared Go fixture setup.
- Explicit malformed JSON and malformed manifest helpers now exist for negative-path tests, with direct package coverage and initial consumer adoption.
- The former `internal/testfixture` and `internal/testutil` wrappers have been removed from the repo.
- Broad validation has passed for:
  - `go test ./packages/testkit-go ./packages/testkit-go/vrooli`
  - `go test ./internal/scenario ./internal/process ./internal/lifecycle ./internal/setup ./internal/resources ./internal/api ./internal/project ./internal/repocontractcheck ./internal/repocontract ./internal/orchestrator ./internal/hostreq ./internal/hostreqcheck ./cmd/vrooli-api`
  - `(cd packages/repo-contract-go && go test ./...)`
  - `(cd packages/cli-core && go test ./cliapp)`
  - `make validate-repo-contract`
- The highest-value remaining work is now seam hardening and retirement of the remaining edge-case helper assumptions, not additional first-party test adoption.

Acceptance criteria:

- no major duplicated fixture layers remain
- validation is green
- the testing architecture is consistent and documented

## `testkit-go` should have its own tests

This package must be directly tested.

Minimum direct coverage:

- repo fixture creation and scenario-dir overrides
- manifest builder output
- file/JSON/executable writer behavior
- malformed fixture helper behavior
- edge-case helper behavior

Also add at least one higher-level integration-style package test proving a generated repo fixture is consumable by real code such as:

- `repo-contract-go.LoadDefault`
- `internal/scenario.Load`

The point is to treat `testkit-go` as real shared infrastructure, not unvalidated helper code.

## Definition of done

This effort is complete when:

- `packages/testkit-go/` is the canonical Go fixture package
- adopted suites use it consistently
- duplicated helper layers are removed
- seam coverage is improved where real behavior was previously carrying logic tests
- docs clearly define the boundary and usage
- the full validation set passes
