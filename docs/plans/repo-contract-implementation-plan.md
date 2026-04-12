# Repo Contract Implementation Plan

**Status:** Draft
**Scope:** Language-agnostic repository contract for Vrooli's future-state cross-platform structure
**Out of scope:** Legacy/transitional project-level Bash structure that is being phased out

---

## 1. Intent

This plan defines how to implement a language-agnostic repository contract for Vrooli so shared packages, platform internals, and repo-aware scenarios stop hard-coding assumptions about project structure.

The contract must describe the **future-state** Vrooli repository shape, not the accidental or transitional shape that exists during migration. In particular, the contract must align with the cross-platform project-level migration described in [project-level-bash-to-go-migration-plan.md](/home/matthalloran8/Vrooli/docs/plans/project-level-bash-to-go-migration-plan.md), and it must **not** normalize deprecated shell-era layout as permanent architecture.

The contract should become the authoritative source for:
- repo root detection
- canonical top-level directories
- canonical scenario layout
- canonical repo-relative glob semantics
- canonical repo-aware bundle/deploy profiles
- selected environment-variable names that are part of the platform contract

The contract should **not** become a dumping ground for all conventions in the repo. Only structural assumptions that are intentionally shared and expected to remain stable belong here.

---

## 2. Authoritative Inputs

The repo contract should be derived from these future-state authorities:

- [project-level-bash-to-go-migration-plan.md](/home/matthalloran8/Vrooli/docs/plans/project-level-bash-to-go-migration-plan.md)
- [resource-cross-platform-migration-plan.md](/home/matthalloran8/Vrooli/docs/plans/resource-cross-platform-migration-plan.md)
- the current Go-native project layout already landed at the project level:
  - root `go.mod`
  - `cmd/`
  - `internal/`
  - `.vrooli/`
  - `scenarios/`
  - `resources/`
  - `packages/`

The contract must treat those sources as higher authority than legacy implementation details still present in the repository.

---

## 3. Explicit Exclusions

The following project-level shell-era paths are transitional debt and must **not** appear in the canonical repo contract:

- `cli/`
- `cli/commands/`
- `cli/lib/`
- `scripts/lib/`
- `scripts/manage.sh`
- project-level shell dispatch assumptions
- project-level shell helper locations

These paths may still exist physically while migration cleanup continues, but they are not part of the future-state repository contract and new consumers must not depend on them.

The contract should also exclude:
- incidental paths used only in tests
- scenario-private folders that are not shared platform assumptions
- implementation detail paths whose structure is not meant to be stable

---

## 4. Desired End State

The end state should have three layers:

### 4.1 Contract Spec

- `.vrooli/repo-contract.json`
- `.vrooli/schemas/repo-contract.schema.json`

This is the language-agnostic source of truth.

### 4.2 Language Adapters

Initial adapter:
- `packages/repo-contract-go`

Future adapters:
- `packages/repo-contract-ts`
- other language adapters as needed

Optional future tooling:
- `vrooli contract ...`

### 4.3 Consumers

Primary consumers:
- `packages/api-core`
- `packages/cli-core`
- selected `internal/*`
- repo-aware scenarios and tools such as:
  - `scenario-to-cloud`
  - `swarm-manager`
  - `workspace-sandbox`
  - `tidiness-manager`
  - `test-genie`
  - `scenario-auditor`
  - `git-control-tower`

---

## 5. Dependency Model

The intended dependency model is:

```text
repo-contract.json/schema
        ↓
packages/repo-contract-go
        ↓
packages/api-core
packages/cli-core
internal/*
        ↓
most scenarios
```

And separately:

```text
packages/repo-contract-go
        ↓
repo-aware scenarios/tools
```

Default rule:
- ordinary scenario runtime logic should usually consume `api-core` or `cli-core`
- repo-aware infrastructure code may consume `repo-contract-go` directly when repository/layout semantics are part of its primary job

Examples of valid direct use:
- bundle composition
- acceptance glob interpretation
- repo root detection
- canonical scenario file resolution
- workspace scope interpretation tied to repository structure

Examples that should stay in higher-level packages:
- storage path policy
- HTTP server lifecycle
- scenario CLI scaffolding
- runtime health and preflight behavior

---

## 6. Contract Scope

Start with a narrow but high-value scope.

### 6.1 Include

- repository root detection contract
- canonical top-level directories
- canonical scenario layout
- canonical resource layout where the future-state structure is stable
- canonical file locations for shared scenario metadata
- root-relative glob semantics
- repo-aware bundle/deploy profiles
- environment variables that are part of the cross-platform platform contract

### 6.2 Exclude

- project-level shell implementation details
- deprecated or transitional paths
- scenario-specific private layouts
- behavioral logic that belongs in code rather than in data
- non-shared conventions that are only used by one implementation

---

## 7. Proposed Contract Shape

An initial contract shape should look roughly like this:

```json
{
  "version": "1.0.0",
  "platform": {
    "mode": "cross_platform_go_native",
    "legacy_project_bash_supported": false
  },
  "root": {
    "markers": {
      "required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"],
      "required_files": ["go.mod"]
    }
  },
  "layout": {
    "project_config_dir": ".vrooli",
    "scenario_dir": "scenarios",
    "resource_dir": "resources",
    "package_dir": "packages",
    "command_dir": "cmd",
    "internal_dir": "internal",
    "docs_dir": "docs"
  },
  "scenario": {
    "required_files": [".vrooli/service.json"],
    "well_known_paths": {
      "service": ".vrooli/service.json",
      "metadata": ".vrooli/metadata.json",
      "docs": "docs",
      "requirements": "requirements",
      "api": "api",
      "ui": "ui",
      "cli": "cli",
      "initialization": "initialization"
    }
  },
  "resource": {
    "manifest": ".vrooli/resource.json",
    "well_known_paths": {
      "docs": "docs",
      "initialization": "initialization"
    }
  },
  "globs": {
    "syntax": "doublestar",
    "root_relative": true,
    "case_sensitive": true,
    "allow_absolute": false
  },
  "sandbox": {
    "env": {
      "repo_root": "VROOLI_ROOT",
      "source_root": "VROOLI_SOURCE_ROOT",
      "sandbox_id": "VROOLI_SANDBOX_ID",
      "sandbox_merged": "VROOLI_SANDBOX_MERGED",
      "sandbox_scope": "VROOLI_SANDBOX_SCOPE"
    }
  },
  "profiles": {
    "mini_vrooli_bundle": {
      "include": [
        ".vrooli",
        "packages",
        "resources/{resources[*]}",
        "scenarios/{scenario}",
        "cmd",
        "internal"
      ],
      "exclude": [
        ".git/**",
        "**/node_modules/**",
        "**/coverage/**",
        "**/data/**",
        "scripts/lib/**",
        "cli/**"
      ]
    }
  }
}
```

Notes:
- This is an illustrative starting shape, not final schema.
- All paths in the spec should use slash-style separators.
- Language adapters should convert to native separators at runtime.

---

## 8. Versioning and Compatibility Rules

The contract should be versioned independently from any one adapter.

### 8.1 Semantic Versioning

- **Patch:** clarifications, documentation additions, non-breaking metadata
- **Minor:** additive paths, additive profile fields, additive metadata
- **Major:** breaking structural changes, renamed fields, changed semantics

### 8.2 Compatibility Policy

- The repo contract is authoritative for the **future-state structure only**
- Legacy project-level Bash paths are not represented in the contract
- Adapters must fail loudly on unsupported contract versions
- Consumers may provide transitional local fallbacks while migrating, but those fallbacks should not alter contract semantics

---

## 9. Phased Implementation Plan

### Phase 0 — Contract Discovery and Boundary Locking

**Goal:** Define what belongs in the contract and what is explicitly out of scope.

Tasks:
1. Inventory structural assumptions from future-state plans and current Go-native project code.
2. Classify assumptions into:
   - canonical future-state contract
   - transitional legacy behavior
   - scenario-private behavior
3. Write the exclusion list for deprecated project-level shell surfaces.
4. Identify the first high-risk consumers to migrate.

Primary investigation targets:
- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go)
- [packages/cli-core/cliutil/sandbox.go](/home/matthalloran8/Vrooli/packages/cli-core/cliutil/sandbox.go)
- [packages/api-core/storage/resolver.go](/home/matthalloran8/Vrooli/packages/api-core/storage/resolver.go)
- [scenarios/scenario-to-cloud/api/bundle/builder.go](/home/matthalloran8/Vrooli/scenarios/scenario-to-cloud/api/bundle/builder.go)
- [scenarios/swarm-manager/api/internal/backlog/types.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/types.go)
- [scenarios/swarm-manager/api/internal/backlog/validate_globs.go](/home/matthalloran8/Vrooli/scenarios/swarm-manager/api/internal/backlog/validate_globs.go)
- [scenarios/tidiness-manager/api/services.go](/home/matthalloran8/Vrooli/scenarios/tidiness-manager/api/services.go)

Deliverables:
- scope document
- exclusions list
- first-migration target list

### Phase 1 — Spec Design

**Goal:** Define the versioned language-agnostic contract format.

Tasks:
1. Draft `.vrooli/schemas/repo-contract.schema.json`
2. Draft `.vrooli/repo-contract.json`
3. Define semantic versioning rules
4. Define normalization rules:
   - slash-normalized paths in spec
   - adapter-native separators at runtime
5. Define explicit compatibility policy for future-state-only structure

Deliverables:
- schema
- initial spec
- versioning policy
- compatibility policy

### Phase 2 — Go Adapter Implementation

**Goal:** Ship `packages/repo-contract-go` as the first adapter.

Core responsibilities:
- load and validate the contract
- resolve repo root
- resolve canonical top-level dirs
- resolve scenario roots
- resolve well-known scenario files
- match root-relative globs
- extract affected scenarios from globs
- resolve named profiles

Suggested package surface:
- `Load(path string) (*Contract, error)`
- `LoadDefault(repoRoot string) (*Contract, error)`
- `FindRepoRoot(start string) (string, error)`
- `ScenarioRoot(repoRoot, scenario string) string`
- `ScenarioFile(repoRoot, scenario, key string) (string, error)`
- `MatchRepoGlob(pattern, relPath string) (bool, error)`
- `AffectedScenarios(patterns []string) []string`
- `Profile(name string) (Profile, error)`

Implementation rules:
- no hidden policy beyond validation and normalization
- use a single glob engine consistently
- treat loaded contract data as immutable
- fail loudly on unsupported versions

Deliverables:
- `packages/repo-contract-go`
- unit tests
- fixture contracts

### Phase 3 — Shared Package Integration

**Goal:** Make shared packages consume the contract where appropriate.

#### `packages/cli-core`

Should use the contract for:
- repo root defaults
- scenario root composition
- canonical scenario file paths

Should keep its own responsibilities for:
- sandbox environment detection
- merged-path redirection
- CLI ergonomics

#### `packages/api-core`

Should use the contract for:
- repo-layout-aware path and file resolution where that is part of a shared API concern

Should keep its own responsibilities for:
- storage policy
- server lifecycle
- preflight/health/database/runtime helpers

Important constraint:
- `api-core` and `cli-core` should not become generic pass-through wrappers for the whole contract.
- They should expose only the slices relevant to their domain.

Deliverables:
- integration plan
- migrated helpers
- reduced duplication in covered path/layout concerns

### Phase 4 — High-Risk Consumer Migration

**Goal:** Remove the most dangerous duplicated layout logic first.

Priority 1:
- `swarm-manager` acceptance glob semantics
- `scenario-to-cloud` bundle/deploy/repo-root rules
- `tidiness-manager` scenario/root resolution
- `workspace-sandbox` project-root defaults that should derive from canonical future-state structure

Priority 2:
- `test-genie`
- `scenario-auditor`
- `git-control-tower`

Examples of concrete migrations:
- Replace the split `filepath.Match` validation and `doublestar` execution in `swarm-manager` with one contract-backed glob policy.
- Replace `scenario-to-cloud` custom repo-root detection and hard-coded include/exclude lists with contract-driven profiles.
- Replace `HOME/Vrooli` and raw string concatenation in `tidiness-manager` with contract-backed path resolution.
- Replace project-root assumptions in `workspace-sandbox` that should derive from the canonical contract.

Deliverables:
- migrated high-risk consumers
- regression coverage
- removal of duplicated layout logic in covered areas

### Phase 5 — Tooling and Validation Surface

**Goal:** Make the contract easy to inspect and hard to drift from.

Proposed commands:
- `vrooli contract validate`
- `vrooli contract show`
- `vrooli contract resolve scenario <name> --file service`
- `vrooli contract match-glob <pattern> <path>`

Validation surfaces:
- schema validity
- semantic validity
- root marker consistency
- profile consistency
- explicit rejection of excluded legacy paths in the contract

Deliverables:
- CLI tooling
- machine-readable output modes
- operator/developer usage docs

### Phase 6 — Documentation and Adoption Rules

**Goal:** Make the contract govern future repo-aware work.

Tasks:
1. Add docs covering:
   - what belongs in the contract
   - what does not
   - how shared packages should consume it
   - when scenarios may depend on `repo-contract-go` directly
2. Update relevant contributor guidance and agent guidance.
3. Add coding rules for covered areas:
   - no new independent repo-root detection logic
   - no new hard-coded canonical scenario path assembly
   - no new duplicated glob semantics in covered consumers

Deliverables:
- contract documentation
- contribution guidance
- adoption rules

---

## 10. Validation Plan

Validation must prove both correctness and drift resistance.

### 10.1 Schema Validation

- validate `repo-contract.json` against `repo-contract.schema.json`
- validate in CI
- optionally reject unknown fields if strictness is desired

### 10.2 Adapter Unit Tests

Test `repo-contract-go` for:
- repo root detection
- path normalization across OS separators
- scenario file resolution
- glob matching semantics
- affected-scenario extraction
- profile expansion

### 10.3 Cross-Platform Path Tests

Run adapter tests on:
- Linux
- macOS
- Windows

Focus:
- separator normalization
- absolute path rejection
- root-relative glob behavior
- case sensitivity semantics

### 10.4 Contract Conformance Tests

Add tests that verify the live repo conforms to the future-state contract:
- required top-level dirs/files exist
- canonical scenario manifests resolve through the contract
- contract profiles reference valid structural roots
- excluded legacy shell paths are absent from the contract

Important:
- do not require transitional legacy files to be physically deleted until their migration plan says so
- do require that the contract itself never points to them

### 10.5 Consumer Integration Tests

For each migrated consumer:
- compare pre- and post-migration behavior where practical
- assert contract-backed behavior matches intended future-state structure
- add regressions for previously duplicated assumptions

Examples:
- `swarm-manager` uses one canonical glob engine/policy for both validation and matching
- `scenario-to-cloud` resolves bundle composition from contract profiles rather than hard-coded lists
- `tidiness-manager` resolves scenario paths through shared contract-backed logic

### 10.6 CI Drift Checks

Add checks that fail when:
- a covered consumer reintroduces hard-coded layout rules
- excluded legacy shell paths get added to the contract
- new repo-aware code bypasses the shared adapter in covered areas

Initial implementation can be grep-based or targeted static checks.

---

## 11. Recommended Migration Order

1. Define scope and exclusions
2. Define schema and initial contract
3. Implement `packages/repo-contract-go`
4. Migrate `swarm-manager` glob semantics
5. Migrate `tidiness-manager` root/scenario resolution
6. Migrate `scenario-to-cloud` repo-root and bundle profile logic
7. Integrate `packages/cli-core`
8. Integrate selected `packages/api-core` path consumers
9. Add CLI tooling and CI drift checks
10. Expand to remaining repo-aware scenarios/tools

This order prioritizes high-risk duplicated assumptions without requiring a full-repo rewrite first.

---

## 12. Decision Rules for Future Additions

Add a structural rule to the repo contract only if all are true:

1. it is shared across multiple consumers
2. it is intended to remain stable in the cross-platform future state
3. drift would be costly
4. it is structural, not merely an incidental implementation detail

If any of those are false, keep it out of the contract.

---

## 13. Recommended Acceptance Criteria

This initiative should be considered complete when:

- the contract spec and schema exist and are versioned
- `packages/repo-contract-go` is implemented and tested cross-platform
- `packages/cli-core` consumes it for repo/scenario path composition
- high-risk repo-aware consumers use it instead of duplicating layout rules
- `swarm-manager` uses one canonical glob engine/policy
- `scenario-to-cloud` bundle composition is profile-driven from the contract
- CI validates the contract and detects drift
- the contract contains no project-level legacy shell structure

---

## 14. Bottom Line

The correct implementation strategy is:
- define the contract against the future Go-native cross-platform architecture
- explicitly exclude transitional shell-era structure
- implement thin language adapters over a declarative spec
- migrate the highest-risk repo-aware consumers first
- add validation that makes drift visible immediately

This gives Vrooli a language-agnostic structural contract without turning any single implementation language into the default source of truth.
