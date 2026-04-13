# testkit-go

`testkit-go` is the canonical Go test-support package for Vrooli.

Its purpose is to centralize reusable Go test infrastructure so migrated project-level tests stop hand-rolling:

- repo fixtures
- repo-contract fixtures
- project/scenario/resource manifests
- JSON/file/executable writers
- compatibility fixture setup for intentionally-supported legacy surfaces

## Scope

`testkit-go` should contain reusable Go test fixture construction and validation helpers that are shared across:

- the root Go module
- `packages/repo-contract-go`
- `packages/cli-core`
- other Go packages that need repo-aware Vrooli fixtures

## Non-goals

`testkit-go` should not contain:

- production code
- package-specific business assertions
- broad assertion helper wrappers
- real integration orchestration that belongs in consumer packages
- fake language-neutral abstractions that are not yet justified

## Design principles

- Prefer typed builders for valid fixtures.
- Keep malformed fixture helpers explicit and separate from valid builders.
- Keep compatibility helpers isolated so migration debt stays visible.
- Preserve clear ownership boundaries:
  - `repo-contract-go` owns repo contract semantics.
  - `testkit-go` owns reusable Go test fixture construction.
  - consumers own behavior assertions.

## Package structure

- root package `packages/testkit-go`
  - cycle-safe base helpers
  - repo fixture construction
  - repo-contract setup
  - support docs
  - file, executable, JSON, and relative path writers
- `packages/testkit-go/vrooli`
  - Vrooli-specific typed fixture helpers that depend on root-module `internal/*` packages
  - project/scenario/resource manifest builders
  - compatibility fixture helpers for legacy shell-era artifacts still intentionally covered

This split is intentional. Lower-level packages and external sibling modules such as `packages/repo-contract-go` can depend on the root package without importing Vrooli domain types and creating test-time import cycles.

## Basic usage

Cycle-safe repo and file helpers live in the root package:

```go
fixture := testkitgo.NewRepoFixture(t)
testkitgo.WriteRepoContract(t, fixture.Root, "scenarios")
testkitgo.WriteRepoContractExceptions(t, fixture.Root)
testkitgo.WriteJSON(t, filepath.Join(fixture.Root, ".vrooli", "settings.json"), map[string]any{
	"mode": "test",
})
```

Vrooli-domain manifest fixtures live under `vrooli`:

```go
manifest := testkitvrooli.ScenarioServiceManifest(
	"alpha",
	testkitvrooli.WithDisplayName("Alpha"),
	testkitvrooli.WithPorts(map[string]scenario.Port{
		"api": {EnvVar: "API_PORT", Range: "18080-18090"},
	}),
)
testkitvrooli.WriteScenarioService(t, fixture.Root, "alpha", manifest)
```

## Adoption rule

When a Go test needs a valid Vrooli repo fixture or valid manifest fixture, the default path should be `testkit-go`, not local handwritten JSON or duplicated repo setup.

Raw JSON/string fixtures should remain only for:

- negative tests validating malformed input handling
- tiny package-local edge cases not worth promoting into shared test infrastructure

See [PLAN.md](/home/matthalloran8/Vrooli/packages/testkit-go/PLAN.md) for the phased implementation and migration checklist.
