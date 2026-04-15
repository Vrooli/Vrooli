# testkit-go

`testkit-go` is the canonical Go test-support package for Vrooli.

Its purpose is to centralize reusable Go test infrastructure so migrated project-level tests stop hand-rolling:

- repo fixtures
- repo-contract fixtures
- JSON/file/executable writers
- focused malformed fixture setup

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
- tests that assert planning or inventory documents as if they were runtime contracts

## Design principles

- Prefer typed builders for valid fixtures.
- Keep malformed fixture helpers explicit and separate from valid builders.
- Tests should validate code contracts, manifests, and runtime behavior, not the continued presence of migration-plan artifacts.
- Preserve clear ownership boundaries:
  - `repo-contract-go` owns repo contract semantics.
  - `testkit-go` owns reusable Go test fixture construction.
  - consumers own behavior assertions.

## Package structure

- root package `packages/testkit-go`
  - dependency-bottom base helpers
  - repo fixture construction
  - repo-contract setup
  - support docs
  - file, executable, JSON, malformed JSON, and relative path writers
- `packages/testkit-go/scenariofixture`
  - project and scenario manifest builders
  - scenario template manifest builders
  - scenario-specific malformed fixtures
  - composite scenario fixture writers used by setup/lifecycle-style tests
- `packages/testkit-go/resourcefixture`
  - resource manifest builders
  - resource template manifest builders
  - resource runtime shims and registry fixtures
- `packages/testkit-go/processfixture`
  - scenario process-record fixtures
- `packages/testkit-go/packagefixture`
  - package-governance and Node package fixture builders

This split replaces the former `packages/testkit-go/vrooli` umbrella package. Tests now import the narrow fixture package that matches the dependency they actually need.

## Basic usage

Dependency-bottom repo and file helpers live in the root package:

```go
fixture := testkitgo.NewRepoFixture(t)
testkitgo.WriteRepoContract(t, fixture.Root, "scenarios")
testkitgo.WriteJSON(t, filepath.Join(fixture.Root, ".vrooli", "settings.json"), map[string]any{
	"mode": "test",
})
```

Scenario-domain manifest fixtures live under `scenariofixture`:

```go
manifest := testscenario.ScenarioServiceManifest(
	"alpha",
	testscenario.WithDisplayName("Alpha"),
	testscenario.WithPorts(map[string]scenario.Port{
		"api": {EnvVar: "API_PORT", Range: "18080-18090"},
	}),
)
testscenario.WriteScenarioService(t, fixture.Root, "alpha", manifest)
```

## Adoption rule

When a Go test needs canonical repo setup, file/JSON writers, or malformed JSON fixtures, the default path should be the root `testkit-go` package, not local handwritten helpers.

When a Go test needs richer fixtures above the root base layer, it should import the narrow package that owns that fixture family: `scenariofixture`, `resourcefixture`, `processfixture`, or `packagefixture`.

## Package Governance

`testkit-go` is governed, but it is not a scenario-adoptable runtime package.

- Scenario-adoptable: no
- Allowed consumer classes: `internal_platform`
- Supported adoption modes: none for governed external scenario/template/resource consumers
- Refresh strategy: none

This package exists for shared Go test infrastructure inside the platform and related governed packages. Scenarios should not adopt it directly. See [docs/package-governance.md](/home/matthalloran8/Vrooli/docs/package-governance.md:1) for the canonical policy.

Raw JSON/string fixtures should remain only for:

- negative tests validating malformed input handling
- tiny package-local edge cases not worth promoting into shared test infrastructure

See [PLAN.md](/home/matthalloran8/Vrooli/packages/testkit-go/PLAN.md) for the phased implementation and migration checklist.
