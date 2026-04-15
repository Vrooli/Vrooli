# testkit-go Re-Layering Implementation Plan

**Status:** Phase 13 complete on April 14, 2026  
**Primary scope:** `packages/testkit-go`, `packages/repo-contract-go`, and migrated Go test consumers in `internal/`, `packages/`, and `cmd/`  
**Goal:** Re-layer shared Go test infrastructure so the base layer is dependency-bottom, fixture construction is more canonical, import-cycle pressure is reduced, and duplicated test setup continues to shrink without replacing it with wrappers, compatibility shims, or helper sprawl.

---

## 0. Why this plan exists

The first `testkit-go` rollout was useful and materially improved the repo:

- shared repo fixture creation exists
- shared file/JSON writers exist
- typed manifest builders exist
- many duplicated local helpers were removed

That progress is real.

The remaining problem is architectural: the current helper graph is still only partially layered.

Most importantly:

- root `packages/testkit-go` had a dependency on `internal/repocontractmeta`
- the former `packages/testkit-go/vrooli` umbrella package aggregated helpers that pulled in multiple unrelated `internal/*` domains
- many same-package tests still rely on a broad helper package because the shared fixtures are not yet split by dependency boundary

If we keep growing the current structure, `testkit-go` becomes a junk drawer. This plan exists to prevent that.

---

## 1. Desired end state

At the end of this work:

- the base shared Go testkit layer is dependency-bottom and imports no `internal/*`
- broad umbrella helper packages are gone
- shared helper packages are narrow and ownership is obvious
- exported-behavior tests can move to external `foo_test` packages more easily
- valid fixtures are usually canonical and typed
- malformed fixtures remain explicit and localized
- duplicated package-local fixture helpers are reduced further
- unit tests and smoke tests are easier to distinguish

The target state is intentionally boring. Test infrastructure should be obvious and stable, not clever.

---

## 2. Hard rules

These rules are non-negotiable for the entire migration.

1. Root shared testkit must not import `internal/*`.
2. No new umbrella helper package is allowed.
3. No compatibility facade or forwarding wrapper is allowed for deprecated helper paths.
4. Shared helpers must encode contracts, not incidental current implementation details.
5. Local helpers stay local unless there is clear multi-package reuse.
6. Migration is not complete until replaced helper layers are deleted.
7. Unit and smoke tests must be explicitly separated when the distinction matters.

---

## 3. Package map target

This is the target direction for the shared test-support stack.

### 3.1 Base dependency-bottom layer

This layer must remain safe for any Go test consumer in the repo.

Candidate structure:

```text
packages/testkit-go/
  doc.go
  repo.go
  files.go
  package_boundary_test.go
  adoption_hygiene_test.go
```

Responsibilities:

- repo fixture bootstrapping
- contract cloning/writing
- file, executable, JSON, and malformed JSON writers
- path-safe helper utilities
- hygiene tests that enforce layering rules

Forbidden in this layer:

- `internal/*` imports
- runtime domain fixture builders
- process/runtime-specific helpers
- compatibility wrappers

### 3.2 Narrow domain fixture layers

Any richer shared helper layer must be split by dependency shape rather than by generic “Vrooli helpers”.

Implemented structure:

```text
packages/testkit-go/scenariofixture/
packages/testkit-go/processfixture/
packages/testkit-go/resourcefixture/
packages/testkit-go/packagefixture/
```

The exact split may change slightly during implementation, but the following rule may not:

- no replacement for `packages/testkit-go/vrooli` as one broad catch-all helper package

### 3.3 Package-local fixture helpers

These remain allowed when they are:

- package-specific
- unexported-internals-aware
- not reusable enough to justify shared API surface

The goal is not to eliminate all local test helpers. The goal is to stop duplicating canonical reusable ones.

---

## 4. Helper admission rubric

A helper may be promoted into shared infrastructure only if all of these are true:

- it is reused, or clearly about to be reused, across multiple packages
- it represents a stable contract fixture
- it improves readability at the call site
- it does not hide behavior that matters to the test reader
- it reduces net maintenance cost

A helper must remain local if any of these are true:

- it is only used by one package
- it depends on package-private details
- it mainly renames or forwards to another helper
- it represents a one-off scenario or quirky edge case

---

## 5. Unit vs smoke boundary policy

This migration is not only about fixture packaging. It is also about test quality.

The repository should explicitly distinguish:

- deterministic unit tests using seams and fixtures
- smoke tests that intentionally exercise real OS/process/network/shell behavior

Rules:

- logic-heavy tests should prefer seams over real subprocesses or `/proc`
- smoke tests should remain small and intentional
- one real-behavior smoke test is better than many incidental real-behavior tests

This policy matters most in:

- `internal/lifecycle`
- `internal/resources`
- `internal/setup`
- `internal/process`

---

## 6. Baseline metrics captured for this plan

These counts were measured at Phase 0 start and must move in the right direction over time.

- package-local test helper declarations matching fixture/setup-style patterns: `75`
- imports of `packages/testkit-go/vrooli` from Go test files: `17`
- raw or malformed JSON fixture hotspots matched by a broad search: `70`
- same-package Go test files in `internal/`, `packages/`, and `cmd/`: `110`

These are directional metrics, not perfect truth. They are good enough to judge whether the refactor is shrinking or expanding test debt.

---

## 7. Phase plan

## Phase 0: Baseline and guardrails

**Objective:** prevent the re-layering from creating more debt than it removes.

### Tasks

- [x] Replace the root `packages/testkit-go` dependency on `internal/repocontractmeta`
- [x] Define the target package map and migration rules in a single authoritative plan
- [x] Capture baseline metrics for helper duplication and broad-helper usage
- [x] Add a hygiene test that forbids `internal/*` imports from root `packages/testkit-go` production files
- [ ] Add a hygiene test that forbids deprecated helper import paths once consumer migration begins
- [x] Record the initial consumer migration wave ordering in the execution notes below

### Phase 0 deliverables

- a strict implementation plan
- an enforced root-package import boundary
- baseline counts for the current test-helper landscape

### Exit criteria

- the base shared layer has a guardrail test protecting its dependency boundary
- the implementation plan is authoritative enough to drive the remaining phases

---

## Phase 1: Dependency-bottom extraction

**Objective:** make the base shared helper layer permanently safe.

### Tasks

- [x] audit every root `packages/testkit-go` symbol and classify it as:
  - [x] base-layer-safe
  - [x] domain-specific and must move
  - [x] obsolete and should delete
- [x] move any remaining contract/path constants needed by shared fixtures into a cycle-safe home
- [x] ensure root `packages/testkit-go` is limited to repo/file/JSON/malformed/path-safe helpers
- [x] verify no root-package production file imports `internal/*`

### Exit criteria

- root `packages/testkit-go` is cleanly below runtime code in the import graph

### Phase 1 audit result

The root exported surface has been explicitly audited and locked down with a hygiene test.

**Base-layer-safe and retained**

- `RepoFixture`, `RepoFixtureOption`, `NewRepoFixture`, `WithScenarioDir`
- `ProjectRoot`, `WriteRepoContract`, `WriteScenarioStub`, `WriteResourceStub`
- `WriteFile`, `WriteExecutable`, `WriteExecutableOnPath`
- `WriteJSON`, `WriteJSONMode`, `WriteRawJSON`, `WriteMalformedJSON`
- `ReadJSONFile`, `ReadJSONFileInto`
- `WriteRelativeFile`, `WriteRelativeExecutable`, `WriteRelativeMalformedJSON`
- `WaitForFile`, `ReserveFreePort`

**Obsolete and deleted from the base layer**

- `ContainsString`

Reason: it was a generic slice utility, not canonical repo/file/JSON fixture support. Keeping helpers like that in the root package is how the base layer drifts back into a junk drawer.

**Domain-specific and intentionally not promoted into the root layer**

- everything currently under `packages/testkit-go/vrooli`

Reason: those helpers depend on root-module `internal/*` packages and belong in the Phase 2 decomposition, not in the dependency-bottom base layer.

---

## Phase 2: Domain helper decomposition

**Objective:** delete the umbrella helper model.

### Tasks

- [x] inventory `packages/testkit-go/vrooli`
- [x] split helpers by dependency shape and ownership
- [x] create narrow domain-specific fixture packages as justified
- [x] migrate consumers off the umbrella package
- [x] delete `packages/testkit-go/vrooli` after migration completes

### Exit criteria

- no broad shared helper package remains

### Phase 2 decomposition result

The former umbrella package was decomposed into:

- `scenariofixture` for project/scenario manifests and scenario-oriented composite fixtures
- `resourcefixture` for resource manifests, registry fixtures, CLI shims, and Docker/test runtime helpers
- `processfixture` for process record fixtures
- `packagefixture` for package-governance and Node package fixtures

The old import path `github.com/vrooli/vrooli/packages/testkit-go/vrooli` is now forbidden by hygiene tests. No compatibility facade was kept.

---

## Phase 3: Consumer migration wave 1

**Objective:** move the easiest and highest-value consumers first.

### Priority targets

- `packages/repo-contract-go`
- `internal/scenario`
- `internal/process`
- `packages/api-core/scenario`
- any exported-behavior suites that can move to external `foo_test`

### Tasks

- [x] replace valid handwritten repo/layout/manifest fixtures with canonical builders
- [x] delete obsolete local helpers immediately after adoption
- [x] convert easy same-package suites to external packages where appropriate

### Exit criteria

- high-value duplicated fixture logic drops measurably

### Phase 3 migration result

This phase concentrated on the highest-value low-risk adopters:

- `packages/repo-contract-go` now derives its canonical valid contract fixture from the shared repo-contract fixture instead of restating the full contract document inline.
- `packages/api-core/scenario` now uses an external `scenario_test` package and a contract-derived repo fixture helper instead of a handwritten repo tree.
- `internal/scenario` kept its same-package tests, but repeated repo-contract fixture setup was collapsed behind a package-local helper and dead helper code was removed.
- `internal/process` was intentionally left same-package because the suite still exercises unexported seam variables directly. That is a legitimate boundary exception, not unfinished migration.

---

## Phase 4: Consumer migration wave 2

**Objective:** consolidate heavier packages without turning shared infrastructure into a dumping ground.

### Priority targets

- `internal/resources`
- `internal/setup`
- `internal/api`
- `internal/project`
- fixture portions of `internal/lifecycle`

### Tasks

- [x] centralize only the clearly reusable contract fixtures
- [x] replace raw valid JSON fixtures with typed or canonical builders where justified
- [x] keep one-off package-specific setup local

### Exit criteria

- these suites are using clearer shared fixtures with less raw valid JSON and fewer duplicated writers

### Phase 4 migration result

This phase focused on the heavier fixture-driven suites without turning the shared test layer into a dumping ground.

- `resourcefixture` now owns a realistic external-CLI resource fixture helper that writes the manifest, the resource-local CLI script, and the binary on `PATH`.
- `internal/api` and `internal/project` no longer carry misleading local helpers that accepted raw status JSON strings even though the real external-CLI driver boundary only depends on the manifest and the binary lookup.
- `internal/setup` fixture construction was tightened so the canonical project fixture path reuses the shared project-service writer instead of duplicating valid setup manifest bootstrapping.
- the fixture portion of `internal/lifecycle` dropped a redundant local port-registry wrapper and now calls the shared resource fixture helper directly.
- `internal/setup` no longer carries a brittle repo-drift assertion on `INSTALL_DIR` assignment syntax; the canonical install-dir contract is now asserted independent of `=` vs `:=`.

---

## Phase 5: Seam and boundary cleanup

**Objective:** make major test suites read as either unit tests or smoke tests, not both.

### Tasks

- [x] split mixed suites, especially `internal/lifecycle`, into unit-focused and smoke-focused coverage
- [x] add seams where logic-heavy tests still rely on real runtime behavior unnecessarily
- [x] keep a small explicit smoke slice for real Linux/process/network boundaries

### Exit criteria

- major mixed suites have clearer intent and less accidental integration behavior

### Phase 5 cleanup result

This phase focused on making the lifecycle suite read like two different classes of tests instead of one oversized mixed file.

- `internal/lifecycle/lifecycle_smoke_test.go` now owns the Linux/process/listener smoke coverage:
  - start/stop/restart lifecycle
  - source-freshness rebuild detection against a real started scenario
  - dependency startup behavior that intentionally exercises real lifecycle process management
  - `listeningPIDs` live-listener smoke coverage
- `internal/lifecycle/lifecycle_test.go` now reads as the seam-oriented unit suite for:
  - setup condition evaluation
  - health timeout/clock behavior
  - orphan cleanup signaling
  - runtime-port and strict-health interpretation
  - custom-path loading and local replace parsing
- dependency startup logic now uses the injected `readScenarioRecords` seam instead of a hard-wired package function, and the unit suite has an explicit test that proves the injected record reader is honored.

The lifecycle package still has meaningful smoke coverage, but it is now intentional and isolated instead of mixed into the general unit suite.

---

## Phase 6: Deletion and hardening

**Objective:** remove all transitional residue.

### Tasks

- [x] delete obsolete helper files and deprecated imports
- [x] update docs to describe final usage rules
- [x] rerun the baseline metrics and compare them against Phase 0

### Exit criteria

- no compatibility wrapper residue remains
- duplication and broad-helper usage are down relative to baseline

### Phase 6 hardening result

This phase removed the remaining transitional residue and locked in the final usage rules.

- the obsolete inventory-driven resource tests were deleted because they treated a migration-plan document as a runtime contract
- the remaining migrated suites no longer import the deleted umbrella helper path
- the README now explicitly states that planning and inventory documents are not valid test contracts

### Phase 6 metric snapshot

These counts are still directional, but they now clearly show net simplification in the migrated surface area.

- package-local fixture/setup helper declarations across the main migrated consumers: `38` versus the Phase 0 broad baseline of `75`
- imports of `packages/testkit-go/vrooli` from Go test files: `0` versus the Phase 0 baseline of `17`
- broad raw/malformed JSON fixture hotspots (`WriteRawJSON`, `WriteMalformedJSON`, `{broken`): `13` versus the Phase 0 broad baseline of `70`
- same-package Go test files across the main migrated consumers: `31`

The same-package count is intentionally reported for the migrated consumer set rather than the entire repository because Phase 5 added explicit smoke/unit file splits, which can increase file count while still improving boundary quality.

---

## Phase 7: Template fixture canonicalization

**Objective:** remove the remaining handwritten valid template-manifest JSON from shared test fixtures and the next highest-value consumer.

### Tasks

- [x] add typed scenario-template fixture builders in `scenariofixture`
- [x] add typed resource-template fixture builders in `resourcefixture`
- [x] replace handwritten valid template JSON in shared fixture helpers with canonical writers
- [x] migrate `internal/resources/templates_test.go` off raw valid `template.json` strings
- [x] remove the stale empty `packages/testkit-go/vrooli` directory so the deleted umbrella package is gone on disk, not just unused in imports

### Exit criteria

- shared testkit packages no longer hand-roll valid template-manifest JSON
- the next template-heavy consumer is using typed shared builders instead of raw `template.json` strings

### Phase 7 result

This phase tightened the shared fixture layer itself before expanding adoption further.

- `scenariofixture` now has typed scenario-template manifest builders, and `WriteScenarioTemplateFixture` writes canonical JSON through shared writers instead of embedding raw valid JSON strings.
- `resourcefixture` now owns typed resource-template manifest builders for tests that need `templates/resources/<name>/template.json` fixtures without importing `internal/resources`.
- `internal/resources/templates_test.go` now uses the shared typed resource-template manifest builder plus canonical base-layer file/JSON writers instead of handwritten `template.json` strings and a local `writeTestFile` helper.
- the stale empty `packages/testkit-go/vrooli` directory was removed, so the deleted umbrella package no longer survives as filesystem residue.

### Phase 7 metric snapshot

These are scoped follow-up checks for the new phase rather than replacements for the Phase 6 broad baseline comparison.

- handwritten raw valid `template.json` fixture strings in `packages/testkit-go` and `internal/resources/templates_test.go`: `0`
- filesystem presence of `packages/testkit-go/vrooli`: removed

---

## Phase 8: Package-local resource helper consolidation

**Objective:** finish the next internal cleanup pass by deleting pure forwarding helpers in `internal/resources` while keeping genuinely package-shared helpers local to that package.

### Tasks

- [x] replace single-file forwarding helpers in `internal/resources/resources_test.go` with direct calls to `testkit-go` and `resourcefixture`
- [x] remove forwarding-only helpers from `internal/resources/schema_artifacts_test.go`
- [x] collapse repeated package-wide resource manifest/config writers into `internal/resources/test_helpers_test.go`
- [x] keep package-specific seam helpers local instead of promoting them into shared testkit packages

### Exit criteria

- one-file forwarding helpers are gone from the `internal/resources` suite
- package-wide helpers that remain have clear ownership and are shared across the package, not hidden inside one test file

### Phase 8 result

This phase continued the adoption work without widening the shared API surface.

- `internal/resources/resources_test.go` no longer carries one-file forwarding wrappers for project resource config, resource manifests, resource CLI scripts, or generic relative file writes; those call sites now use `testscenario`, `testresource`, and base `testkit-go` directly.
- `internal/resources/schema_artifacts_test.go` now uses the shared scenario/resource fixture writers directly instead of local pass-through helpers.
- package-wide helpers that are still useful across multiple `internal/resources` test files now live in `internal/resources/test_helpers_test.go`, which is the right boundary for helpers like `writeResourceConfig`, `writeResourceManifest`, and `writeEnvManifestFixture`.
- package-specific seam helpers such as the PATH and `docker` shim installers remain local to `resources_test.go` because they encode package-global seam wiring, not generic fixture construction.

### Phase 8 metric snapshot

- pure forwarding helper functions removed from `internal/resources/resources_test.go` and `internal/resources/schema_artifacts_test.go`: `6`
- remaining `internal/resources` package-local shared helpers are centralized in `internal/resources/test_helpers_test.go` instead of scattered across multiple files

---

## Phase 9: Package-local setup helper consolidation

**Objective:** apply the same package-boundary cleanup to `internal/setup` by moving genuinely package-wide test fixtures into a package-local helper file and deleting stale or dead test drift.

### Tasks

- [x] move package-wide project/onboarding fixture builders out of `internal/setup/setup_test.go`
- [x] centralize repeated setup-marker creation behind a package-local helper
- [x] delete dead setup test helpers that no longer have any consumers
- [x] repair stale repo-contract drift assertions encountered during validation

### Exit criteria

- `internal/setup/setup_test.go` no longer carries buried package-wide fixture builders
- repeated setup marker writes are centralized
- package validation passes against the current repo contract

### Phase 9 result

This phase kept the same architectural rule: canonical reusable setup fixtures should either come from shared testkit packages or from a small package-local helper file when they encode setup-package specifics.

- `internal/setup/test_helpers_test.go` now owns the package-wide setup fixtures: project service fixture creation, onboarding fixture creation, and setup-marker writes.
- `internal/setup/setup_test.go` lost the buried fixture block at the bottom of the file and now reads more directly as a behavior suite.
- the dead helper `writeProjectFixtureWithManifest` was removed instead of being preserved “just in case”.
- `TestRepoMaintainsCanonicalInstallContract` now asserts the current repo contract through the Makefile install target and canonical install directory, rather than a stale `install-cli` lifecycle step that no longer exists in `.vrooli/service.json`.

### Phase 9 metric snapshot

- package-wide helper functions moved out of `internal/setup/setup_test.go`: `4`
- dead setup-specific helper functions deleted: `1`
- repeated setup-marker writes replaced with a package-local helper: `4`

---

## Phase 10: API test helper simplification

**Objective:** finish the next small consumer cleanup by removing the last one-off file-writing shim from `internal/api` and using the dependency-bottom shared writer directly.

### Tasks

- [x] replace local one-off file-writing helpers in `internal/api/app_test.go` with base `testkit-go` writers
- [x] delete the unused helper stack once the call site is migrated
- [x] confirm `internal/api` still has only API-specific local helpers after cleanup

### Exit criteria

- `internal/api` does not carry bespoke generic file-writing helpers
- tests use the canonical dependency-bottom writer directly where that is sufficient

### Phase 10 result

This was a smaller phase than the `resources` and `setup` cleanup passes, but it follows the same rule: do not keep a local helper when the shared base-layer writer already expresses the test intent cleanly.

- `internal/api/app_test.go` now writes log fixtures through `testkit-go.WriteFile` instead of a private `osWriteFileAll` helper stack.
- the local helper trio `osMkdirAll`, `osWriteFileAll`, and `ensureParentDir` was deleted because it was only supporting one call site and duplicated base-layer behavior.
- the remaining local helpers in `internal/api` are behavior-oriented response decoding helpers, not generic fixture-writing wrappers.

### Phase 10 metric snapshot

- one-off generic file-writing helper stacks removed from `internal/api`: `1`
- remaining occurrences of `osMkdirAll`, `osWriteFileAll`, and `ensureParentDir` in `internal/api` tests: `0`

---

## Phase 11: Lifecycle helper boundary cleanup

**Objective:** finish the remaining high-value consumer cleanup by moving shared lifecycle test fixtures out of the smoke file and using the dependency-bottom writer for generic lifecycle test files.

### Tasks

- [x] move shared lifecycle fixture builders into a package-local helper file used by both unit and smoke suites
- [x] remove generic file-writing calls where `testkit-go` base writers already express the intent
- [x] keep the explicit smoke/unit file split intact while reducing fixture duplication

### Exit criteria

- shared lifecycle fixtures no longer live inside `lifecycle_smoke_test.go`
- smoke and unit suites share the same package-local lifecycle fixture boundary
- lifecycle validation still passes with the smoke slice enabled

### Phase 11 result

This phase kept the lifecycle smoke/unit split intact while cleaning up the remaining fixture ownership drift.

- `internal/lifecycle/test_helpers_test.go` now owns the package-local lifecycle fixtures shared by both suites: `writeLifecycleFixture`, `writeLifecycleFixtureManifest`, and `lifecycleFixtureManifest`.
- `internal/lifecycle/lifecycle_smoke_test.go` no longer hides shared fixture builders at the bottom of the smoke file; it now focuses on explicit Linux/process smoke coverage.
- `internal/lifecycle/lifecycle_test.go` and `internal/lifecycle/lifecycle_smoke_test.go` now route generic test file creation through `testkit-go` base writers where the tests were only creating ordinary files or executables.
- the smoke and unit suites still remain explicitly separated; this phase reduced helper drift without collapsing the boundary back together.

### Phase 11 metric snapshot

- shared lifecycle fixture helpers moved out of the smoke file into a package-local helper file: `3`
- remaining generic `os.WriteFile` calls in lifecycle tests after cleanup: `0`

---

## Phase 12: Cross-cutting validation and drift cleanup

**Objective:** run the minimum migrated-surface validation set end-to-end, fix any test drift uncovered by the broader run, and confirm the cleaned fixture boundaries hold across package edges.

### Tasks

- [x] run the root-module migrated validation subset
- [x] run `packages/repo-contract-go` from its own module root
- [x] run `make validate-repo-contract`
- [x] fix repo-contract check fixture drift discovered by the broader validation run
- [x] align setup/lifecycle tests with the canonical `projectstate` marker paths
- [x] delete obsolete tests that no longer correspond to any active validator rule

### Exit criteria

- the minimum migrated validation set passes end-to-end
- repo-contract validation and its test fixture surface are aligned with the current live contract
- setup/lifecycle tests assert the canonical typed state-marker paths rather than stale legacy locations

### Phase 12 result

This phase validated the migration as a coherent whole instead of as isolated package cleanups.

- the root-module migrated subset passed across `internal/process`, `internal/scenario`, `internal/lifecycle`, `internal/setup`, `internal/resources`, `internal/api`, `internal/project`, `internal/repocontractcheck`, `cmd/vrooli`, `cmd/vrooli-api`, and `packages/testkit-go`.
- `packages/repo-contract-go` passed from its own module root, which is the correct invocation model for that package.
- `make validate-repo-contract` passed.
- `internal/repocontractcheck` fixture repos now include the minimum `docs/repo-contract.md` surface needed to isolate adoption-rule failures from docs-alignment failures.
- the obsolete AGENTS-based repocontractcheck test was removed because no current validator rule enforces AGENTS guidance.
- `internal/setup` and `internal/lifecycle` tests now assert the canonical `projectstate` marker paths rather than relying on legacy `data/.setup-complete` and sibling marker paths.

### Phase 12 validation snapshot

Validated successfully with:

```bash
go test ./packages/testkit-go/... ./internal/process ./internal/scenario ./internal/lifecycle ./internal/setup ./internal/resources ./internal/api ./internal/project ./internal/repocontractcheck ./cmd/vrooli ./cmd/vrooli-api
(cd packages/repo-contract-go && go test ./...)
make validate-repo-contract
```

---

## Phase 13: Remaining hotspot sweep and boundary confirmation

**Objective:** finish the remaining low-risk hotspot cleanup while explicitly preserving local helpers where import cycles or module boundaries make sharing counterproductive.

### Tasks

- [x] remove remaining pure forwarding helpers in cycle-safe root-module suites
- [x] confirm same-package internal tests do not import shared fixture packages that depend back on the package under test
- [x] confirm separate-module tests do not gain unnecessary new dependencies on `packages/testkit-go`
- [x] document the final rule for when helpers should remain package-local

### Exit criteria

- root-module hotspot suites use shared fixtures directly where the dependency graph allows it
- cycle-prone same-package tests keep package-local helpers instead of forcing shared-fixture imports
- separate-module tests stay self-contained unless a new cross-module dependency is clearly justified

### Phase 13 result

This phase closed the remaining cleanup pass by removing low-value wrappers where the architecture allowed it and by explicitly preserving local helpers where sharing would have made the codebase worse.

- `internal/app/package/service_integration_test.go` now uses `packagefixture`, `resourcefixture`, and `scenariofixture` directly instead of routing through local forwarding helpers.
- `internal/orchestrator/orchestrator_test.go` now uses shared scenario and process fixtures directly instead of keeping local wrapper functions for process records and scenario services.
- `internal/packagegov/packagegov_test.go` was intentionally kept package-local for package-manifest fixtures because importing `packagefixture` from a same-package `internal/packagegov` test creates an import cycle through `internal/packagegov`.
- `internal/scenario/scenario_test.go` likewise keeps its scenario service helper local because `scenariofixture` depends on `internal/scenario`, so same-package tests cannot import it without recreating the cycle this migration is trying to remove.
- `packages/api-core/scenario/scenario_test.go` was intentionally left self-contained after verifying that importing `packages/testkit-go` there would introduce a new separate-module dependency and `go.sum` churn for minimal cleanup benefit.

The important architectural result is that the migration now has a clearer stopping rule: shared fixtures should be used directly when they reduce duplication without worsening dependencies, but same-package cycle pressure and separate-module boundaries are valid reasons to keep helpers local.

### Phase 13 validation snapshot

Validated successfully with:

```bash
go test ./internal/app/package ./internal/packagegov ./internal/orchestrator ./internal/scenario
(cd packages/api-core && go test ./scenario)
```

---

## 14. Initial migration wave ordering

This is the initial recommended execution order after Phase 0.

1. root `packages/testkit-go` classification and cleanup
2. `packages/repo-contract-go`
3. `internal/process`
4. `internal/scenario`
5. `packages/api-core/scenario`
6. `internal/resources`
7. `internal/setup`
8. `internal/api`
9. `internal/lifecycle`

This order is intentional:

- it starts with the dependency-bottom layer
- then fixes the packages most sensitive to import-cycle pressure
- then moves into the larger fixture-heavy suites

---

## 15. Validation requirements

At the end of each phase, run the relevant subset. The final minimum validation set is:

```bash
go test ./packages/testkit-go/...
go test ./packages/repo-contract-go/...
go test ./internal/process ./internal/scenario ./internal/lifecycle ./internal/setup ./internal/resources ./internal/api ./internal/project ./internal/repocontractcheck
go test ./cmd/vrooli ./cmd/vrooli-api
make validate-repo-contract
```

If additional consumers are migrated, expand the validation set accordingly.

---

## 16. Definition of done

This re-layering is done only when:

- root shared test infrastructure is dependency-bottom
- broad helper aggregation is gone
- valid fixtures are mostly canonical and typed
- malformed fixtures are explicit
- obsolete helper paths are deleted, not wrapped
- duplicated package-local fixture helpers are reduced
- the resulting structure is simpler to explain than the one it replaced
