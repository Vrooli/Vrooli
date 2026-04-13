# Package Adoption And Refresh Governance Plan

**Status:** Implemented and validated

**Last Updated:** 2026-04-13

## Why This Plan Exists

Implementation status:

- the manifest-driven package governance model is now live in the native `vrooli package ...` command surface;
- all current package roots under `packages/` declare `.vrooli/package.json` manifests;
- package manifest schema validation, semantic validation, dependent discovery, refresh behavior, audit drift detection, and stack-governor/scenario-auditor integration are implemented;
- legacy shell refresh helpers have been removed;
- repo-level validation now runs through `make validate-package-governance`.

Vrooli's shared package story is currently useful but not governed cleanly enough.

Today, the repo has a real architectural goal:

- scenarios must stay as independent as possible;
- scenarios must be free to use different languages, frameworks, package managers, and framework versions;
- shared packages should improve reuse without collapsing scenarios into a monolithic workspace;
- package adoption rules should be explicit, enforceable, and easy to operate.

The current implementation only partially satisfies that goal.

What exists today:

- `packages/` is the canonical shared package root ([docs/repo-contract.md](/home/matthalloran8/Vrooli/docs/repo-contract.md:89)).
- JS/TS scenarios commonly adopt shared packages through `file:` dependencies such as:
  - `@vrooli/api-base`
  - `@vrooli/iframe-bridge`
  - `@vrooli/vitest-requirement-reporter`
  - `@vrooli/proto-types`
- Scenarios are intentionally excluded from the root pnpm workspace ([pnpm-workspace.yaml](/home/matthalloran8/Vrooli/pnpm-workspace.yaml:1)).
- Go shared packages such as `api-core` and `cli-core` are consumed via Go modules and local `replace` directives rather than package-manager installation.
- A Bash helper exists for refreshing some JS/TS package consumers: [scripts/scenarios/tools/refresh-shared-package.sh](/home/matthalloran8/Vrooli/scripts/scenarios/tools/refresh-shared-package.sh:1).

What is still wrong:

- package adoption rules are not declared in a canonical manifest;
- not all packages document adoption and refresh behavior consistently;
- some docs contradict the scenario-isolation model (`workspace:*` guidance still exists in a few places);
- JS/TS package propagation relies on a legacy Bash helper rather than a first-class `vrooli` command;
- some scenarios use ad hoc `postinstall` copy/symlink behavior that is not standardized or centrally governed;
- scenario-auditor does not yet enforce package adoption policy as a first-class rule set;
- there is no explicit package taxonomy distinguishing:
  - scenario-adoptable packages,
  - generated packages,
  - Go runtime packages,
  - test-only packages,
  - internal-only packages.

This plan defines a clean hard cutover to a manifest-driven, native-Go package governance system with no legacy compatibility layer left behind.

## Goal

Implement a fully native package governance and refresh system that:

- preserves scenario independence by design;
- explicitly classifies every package in `packages/`;
- defines which packages are scenario-adoptable and by which kinds of consumers;
- gives operators a first-class `vrooli package ...` surface for discovery, validation, generation, and refresh;
- removes package refresh logic from legacy shell helpers;
- replaces scattered convention with documented and enforced policy;
- supports multi-language scenarios without forcing JavaScript- or Go-specific assumptions onto all scenarios;
- fully validates the new model in tests and scenario-auditor rules before deleting legacy code and docs.

## Non-Goals

- Moving scenarios into the root pnpm workspace.
- Requiring all scenarios to use Node, pnpm, Vite, React, or TypeScript.
- Forcing all scenarios to consume shared packages.
- Preserving Bash helpers or legacy compatibility paths indefinitely.
- Standardizing every language ecosystem in one phase; the architecture must be language-agnostic first, with language-specific policy layered on top.

## Hard Constraints

The design must preserve these invariants:

1. Scenario independence is a hard requirement.
2. Shared package adoption must be explicit, not implicit.
3. The repo must remain future-friendly for scenarios using any language/framework/runtime.
4. Package governance must not assume pnpm as the universal scenario package manager.
5. Native `vrooli` commands must become the only supported operator path for shared package refresh/governance.
6. Greenfield hard cutover is required:
   - no permanent compatibility mode;
   - no dead shell helpers;
   - no parallel “old and new” package governance architecture after rollout completes.

## Current State Summary

### 1. JS/TS Package Adoption Today

Common current pattern:

- scenario `ui/package.json` references repo packages via `file:../../../packages/...`;
- scenario setup/install populates scenario-local `node_modules`;
- some scenarios also copy or symlink package contents in `postinstall`.

Representative examples:

- [templates/scenarios/react-vite/ui/package.json](/home/matthalloran8/Vrooli/templates/scenarios/react-vite/ui/package.json:1)
- [scenarios/swarm-manager/ui/package.json](/home/matthalloran8/Vrooli/scenarios/swarm-manager/ui/package.json:1)
- [scenarios/prompt-manager/ui/package.json](/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/package.json:1)
- [scenarios/browser-automation-studio/ui/package.json](/home/matthalloran8/Vrooli/scenarios/browser-automation-studio/ui/package.json:1)
- [scenarios/scenario-to-desktop/ui/package.json](/home/matthalloran8/Vrooli/scenarios/scenario-to-desktop/ui/package.json:1)
- [scenarios/notification-hub/ui/package.json](/home/matthalloran8/Vrooli/scenarios/notification-hub/ui/package.json:1)

Implication:

- changing a shared JS/TS package does not automatically propagate into every scenario's install tree, because scenarios are intentionally isolated from the root workspace.

### 2. Go Package Adoption Today

Go shared packages follow a different model:

- scenario modules import `github.com/vrooli/api-core`, `github.com/vrooli/cli-core`, or `github.com/vrooli/vrooli/packages/proto`;
- scenario `go.mod` files use local `replace` directives to point at repo-local package sources.

Representative guidance:

- [packages/api-core/README.md](/home/matthalloran8/Vrooli/packages/api-core/README.md:13)
- [packages/cli-core/README.md](/home/matthalloran8/Vrooli/packages/cli-core/README.md:1)
- [packages/proto/README.md](/home/matthalloran8/Vrooli/packages/proto/README.md:89)

Implication:

- Go does not need npm-style “reinstall dependents” behavior;
- it does still need:
  - rebuild/restart for already-running binaries,
  - regeneration for generated code packages like `proto`,
  - clear policy on which packages are scenario-adoptable.

### 3. Existing Refresh Surface

Current helper:

- [scripts/scenarios/tools/refresh-shared-package.sh](/home/matthalloran8/Vrooli/scripts/scenarios/tools/refresh-shared-package.sh:1)

Current documentation references:

- [packages/api-base/README.md](/home/matthalloran8/Vrooli/packages/api-base/README.md:94)
- [packages/iframe-bridge/README.md](/home/matthalloran8/Vrooli/packages/iframe-bridge/README.md:57)
- [scenarios/test-genie/docs/reference/vitest-requirement-reporter.md](/home/matthalloran8/Vrooli/scenarios/test-genie/docs/reference/vitest-requirement-reporter.md:221)

Problems:

- shell helper is not a first-class governance surface;
- docs are incomplete and inconsistent;
- no equivalent native package-governance command tree exists today.

### 4. Existing Contradictions And Drift

Examples of current policy/documentation drift:

- scenarios are explicitly excluded from the workspace ([pnpm-workspace.yaml](/home/matthalloran8/Vrooli/pnpm-workspace.yaml:2));
- yet some docs still mention `workspace:*` adoption for scenario packages ([scenarios/scenario-auditor/README.md](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/README.md:194), [packages/vitest-requirement-reporter/README.md](/home/matthalloran8/Vrooli/packages/vitest-requirement-reporter/README.md:8));
- package adoption is real and common, but not declared in any canonical package manifest;
- scenario-auditor has rules for some UI package usage details, but not broad package-governance enforcement.

## Design Principles

1. Scenario independence over convenience.
2. Manifest-driven governance over convention.
3. Package-type-specific behavior over one-size-fits-all heuristics.
4. Native `vrooli` command surfaces over shell entrypoints.
5. Explicitly supported package adoption over accidental reuse.
6. Language-agnostic architecture with language-specific policy modules.
7. Hard cutover with deletion once parity is proven.
8. Auditor-enforced policy, not tribal knowledge.

## Desired End State

When this plan is complete:

- every package under `packages/` declares its package governance metadata in a canonical manifest;
- `vrooli package ...` is the single operator/developer surface for package governance and refresh;
- JS/TS shared package propagation is native and deterministic;
- generated package flows (especially `proto`) are explicitly modeled rather than hidden behind manual steps;
- scenario-auditor rejects unsupported package adoption patterns;
- templates teach the canonical package adoption pattern and nothing else;
- contradictory docs are gone;
- legacy shell refresh helpers are deleted;
- scenario independence remains intact and is more strongly enforced than today.

## Proposed Architecture

### 1. New Package Manifest

Add a required package manifest for every package under `packages/`.

Recommended path:

- `packages/<name>/.vrooli/package.json`

Rationale:

- matches the repo's existing pattern of manifest-driven governance for scenarios/resources;
- keeps governance metadata separate from language/package-manager metadata;
- allows non-Node packages to participate without inventing fake `package.json` semantics.

Recommended minimum fields:

```json
{
  "$schema": "../../../.vrooli/schemas/package.schema.json",
  "version": "1.0.0",
  "package": {
    "name": "iframe-bridge",
    "kind": "js_runtime",
    "adoption": {
      "scenario_adoptable": true,
      "allowed_consumers": ["scenario_ui"],
      "adoption_modes": ["file_dependency"]
    },
    "lifecycle": {
      "generate": [],
      "build": [
        {
          "name": "build",
          "run": ["pnpm", "--filter", "@vrooli/iframe-bridge", "build"]
        }
      ]
    },
    "refresh": {
      "strategy": "scenario_setup",
      "restart_running_consumers": true
    }
  }
}
```

Additional fields to support as needed:

- `language`
- `module_identifiers`
- `generated_outputs`
- `consumer_discovery`
- `requires_generation`
- `internal_only`
- `test_only`
- `docs`
- `validation`

### 2. Package Taxonomy

Every package must declare one `kind`.

Initial kinds:

- `js_runtime`
  - scenario-adoptable JS/TS runtime package
  - examples: `api-base`, `iframe-bridge`, `vitest-requirement-reporter`
- `generated_typescript`
  - generated TS output package intended for scenario consumption
  - example: `proto-types` represented through `packages/proto`
- `go_runtime`
  - scenario-adoptable Go runtime helper package
  - example: `api-core`
- `go_cli`
  - scenario-adoptable Go CLI package
  - example: `cli-core`
- `go_testkit`
  - Go test-only support package
  - example: `testkit-go`
- `internal_platform`
  - shared package not meant for direct scenario adoption
- `schema_or_contract`
  - schema/contract package with special generation or validation rules

The taxonomy must be policy-bearing:

- `kind` is not documentation only;
- `kind` determines allowed consumers, refresh behavior, generation/build requirements, and auditor checks.

### 3. Consumer Taxonomy

Define explicit consumer classes:

- `scenario_ui`
- `scenario_api`
- `scenario_cli`
- `scenario_test`
- `template_ui`
- `template_api`
- `template_cli`
- `resource_runtime`
- `internal_platform`

This prevents “package may be used anywhere” ambiguity.

### 4. Canonical Adoption Modes

Forbid ad hoc package adoption.

Allow only explicit adoption modes declared by package manifests and enforced by scenario-auditor:

- `file_dependency`
  - local path dependency from isolated scenario package manager manifest
- `go_module_replace`
  - Go module + local replace directive
- `generated_artifact`
  - generated outputs adopted through package-specific commands
- `published_semver`
  - future optional mode for packaged/published distribution

Prohibited end-state examples:

- undocumented `workspace:*` in real scenarios;
- arbitrary package copying/symlinking in `postinstall` when not authorized by package policy;
- using internal-only packages directly from scenarios;
- hidden package adoption through bespoke scripts outside `vrooli`.

### 5. Native `vrooli package` Surface

Add a top-level command group:

```bash
vrooli package <subcommand>
```

Initial subcommands:

- `list`
  - enumerate packages and kinds
- `info <name>`
  - show manifest, package kind, adoption policy, and docs
- `dependents <name>`
  - list scenarios/templates/resources/internal consumers discovered via policy-aware scanners
- `validate [<name>|--all]`
  - validate package manifests and consumer conformance
- `build <name>`
  - run package-declared build lifecycle
- `generate <name>`
  - run package-declared generation lifecycle if applicable
- `refresh <name> [<consumer>|all] [--no-restart]`
  - rebuild/regenerate package and propagate to affected consumers according to package policy
- `audit [--all]`
  - summarize unsupported adoption patterns and documentation drift

The command contract must be type-aware:

- `js_runtime.refresh`:
  - build package
  - discover allowed consumers
  - run `vrooli scenario setup` for affected scenarios
  - restart only scenarios that were already running unless `--no-restart`
- `generated_typescript.refresh`:
  - generate package first
  - then run affected JS/TS consumer refresh
- `go_runtime.refresh`:
  - no scenario setup phase by default
  - identify affected consumers
  - optionally restart affected running scenarios if their runtime binaries are impacted
- `go_cli.refresh`:
  - identify affected CLI consumers
  - trigger reinstall/rebuild of those CLIs or mark them stale through a native mechanism

### 6. Package Resolver And Discovery Layer

Add a native package-governance resolver, recommended package:

- `internal/packagegov`

Responsibilities:

- discover package manifests from `packages/*/.vrooli/package.json`;
- validate package manifest schema;
- understand package kinds and refresh strategies;
- discover dependents across:
  - scenario manifests,
  - template package manifests,
  - `package.json`,
  - `go.mod`,
  - future language-specific dependency manifests;
- classify each consumer into a declared consumer class;
- reject unsupported consumer/package combinations.

This resolver must be policy-aware and not rely on package-manager-specific assumptions except inside package-kind-specific discovery modules.

### 7. Scenario-Auditor Enforcement

Add a new package-governance rule family in `scenario-auditor`.

Recommended rule categories:

- `package-adoption-supported`
  - scenario only consumes packages marked `scenario_adoptable`
- `package-adoption-mode-valid`
  - scenario uses only allowed adoption modes for each package kind
- `package-no-workspace-deps`
  - real scenarios may not use `workspace:*` for shared package adoption
- `package-no-unauthorized-postinstall`
  - scenario package manager hooks may not copy/symlink shared packages except through the sanctioned mechanism
- `package-required-shared-runtime`
  - only when a template/archetype requires a package, not globally
- `package-generated-consumer-valid`
  - generated package consumers declare the correct package and generation contract

This enforcement must distinguish:

- real scenarios;
- templates;
- test fixtures used by scenario-auditor itself.

### 8. Template And Scenario Contract Alignment

Templates must become the canonical greenfield examples.

Required template changes:

- scenario templates must use the canonical package adoption pattern only;
- templates must not rely on custom `postinstall` copy/symlink hacks unless that behavior is part of the new sanctioned package-adoption mechanism;
- template docs must teach `vrooli package refresh`, not shell helpers or `workspace:*`.

### 9. Documentation Contract

Documentation must be aligned to the new package governance model.

Required changes:

- add package governance docs under `docs/`;
- update package READMEs to describe package kind, consumer classes, and refresh semantics;
- remove legacy shell refresh references;
- fix all contradictory `workspace:*` guidance in scenario docs and package docs;
- document scenario independence as the reason for this model, not as an incidental implementation detail.

## Recommended Policy Decisions

These decisions should be made explicit in the implementation, not left implicit.

### Policy 1: Scenario Independence Beats Monorepo Convenience

Decision:

- scenarios stay outside the root pnpm workspace.

Reason:

- preserves framework/version/package-manager independence;
- avoids implicit coupling between scenarios;
- matches the product requirement that scenarios must be able to evolve across language/runtime boundaries.

### Policy 2: Shared Package Adoption Is Opt-In, Not Default

Decision:

- packages are not scenario-adoptable unless their manifest says so.

Reason:

- prevents accidental platform coupling;
- forces package authors to think through API stability and consumer classes.

### Policy 3: Package Kinds Drive Refresh Behavior

Decision:

- refresh semantics are package-kind-specific, not uniform.

Reason:

- JS runtime packages, generated packages, Go runtime packages, and Go CLI packages do not share the same propagation mechanics.

### Policy 4: Ad Hoc `postinstall` Package Copying Is Temporary Debt

Decision:

- custom scenario `postinstall` package propagation is not an allowed end state.

Reason:

- it is invisible governance;
- it is difficult to audit;
- it encourages package-manager-specific hidden behavior;
- it makes future non-Node scenarios harder to reason about.

### Policy 5: Package Governance Must Be Auditor-Enforced

Decision:

- package governance is not complete until scenario-auditor validates it.

Reason:

- otherwise the repo will drift back into convention-based adoption.

## Migration Strategy

This is a hard-cutover plan, but the implementation still needs phased delivery so it can be validated before deletion.

The cutover rule is:

- no package-governance phase is considered complete until all old docs/code for that surface are removed or rewritten to the new model.

## Phased Checklist

### Phase 0 — Package Surface Audit

Goal:

- build the authoritative inventory before introducing new rules.

Deliverables:

- inventory of all `packages/*`;
- inventory of all consumers by package and consumer class;
- inventory of all current package adoption patterns;
- list of packages that are intentionally scenario-adoptable vs. not.

Tasks:

- [ ] Enumerate every package under `packages/`
- [ ] Classify each package into a proposed `kind`
- [ ] Enumerate every consumer across scenarios/templates/resources/internal code
- [ ] Record all current adoption modes (`file:`, `go.mod replace`, custom `postinstall`, generated output, docs-only)
- [ ] Identify every package doc that contradicts scenario isolation
- [ ] Identify every scenario/template using custom package-copy/symlink `postinstall`
- [ ] Decide which packages are intentionally scenario-adoptable in the long-term
- [ ] Decide which packages are internal-only or test-only

Validation:

- [ ] Audit output reviewed and checked into the repo as part of implementation context

### Phase 1 — Manifest Contract Design

Goal:

- define the package manifest and its schema before implementing commands.

Deliverables:

- package manifest schema;
- package kind taxonomy;
- consumer taxonomy;
- refresh strategy taxonomy.

Tasks:

- [ ] Add `.vrooli/schemas/package.schema.json`
- [ ] Document canonical package manifest path and required fields
- [ ] Define the allowed `kind` values and semantics
- [ ] Define `adoption.allowed_consumers`
- [ ] Define `adoption.adoption_modes`
- [ ] Define lifecycle hooks for `generate` and `build`
- [ ] Define `refresh.strategy`
- [ ] Define JSON examples for each package kind
- [ ] Document package-governance rules in a new docs page

Validation:

- [ ] Schema validation tests landed
- [ ] Invalid manifests rejected with clear errors
- [ ] Docs and schema reviewed together for coherence

### Phase 2 — Native Package Resolver

Goal:

- create a single native policy-aware source of truth for package manifests and dependent discovery.

Deliverables:

- `internal/packagegov` or equivalent resolver package.

Tasks:

- [ ] Load and validate package manifests
- [ ] Resolve package kind and refresh strategy
- [ ] Discover JS/TS consumers from `package.json`
- [ ] Discover Go consumers from `go.mod`
- [ ] Discover generated TS consumers for `proto-types`
- [ ] Classify discovered consumers into consumer classes
- [ ] Expose structured dependency graph results
- [ ] Add machine-readable JSON output structures for CLI usage

Validation:

- [ ] Unit tests for each discovery mode
- [ ] Fixtures for positive and negative consumer classification
- [ ] No shell helper dependency in resolver logic

### Phase 3 — `vrooli package` CLI

Goal:

- replace shell discovery/refresh workflows with native CLI commands.

Deliverables:

- `vrooli package list`
- `vrooli package info`
- `vrooli package dependents`
- `vrooli package validate`
- `vrooli package build`
- `vrooli package generate`
- `vrooli package refresh`
- `vrooli package audit`

Tasks:

- [ ] Add command group to `cmd/vrooli`
- [ ] Add human-readable and `--json` output for all package commands
- [ ] Implement package-aware build execution
- [ ] Implement package-aware generation execution
- [ ] Implement refresh semantics per package kind
- [ ] Preserve “restart only already-running scenarios” behavior for scenario refreshes
- [ ] Add dry-run support where useful
- [ ] Add error messages that explain why a package cannot be refreshed in a given mode

Validation:

- [ ] CLI integration tests for all package subcommands
- [ ] Scenario refresh behavior validated against real fixtures
- [ ] JSON output snapshot tests landed

### Phase 4 — Scenario-Auditor Package Governance Rules

Goal:

- enforce package governance in scenario validation.

Deliverables:

- new scenario-auditor rule family for package governance.

Tasks:

- [ ] Add rules forbidding `workspace:*` in real scenarios
- [ ] Add rules validating package adoption only for `scenario_adoptable` packages
- [ ] Add rules validating allowed consumer classes
- [ ] Add rules validating allowed adoption modes
- [ ] Add rules detecting unauthorized package-copy/symlink `postinstall`
- [ ] Add fixtures for allowed and disallowed package adoption
- [ ] Ensure rules distinguish fixtures/testdata from real scenario policy

Validation:

- [ ] Auditor unit tests green
- [ ] Representative good scenarios pass
- [ ] Known bad fixtures fail with precise messages

### Phase 5 — Template Hardening

Goal:

- make greenfield generation align with the new model immediately.

Deliverables:

- updated scenario templates;
- updated package-adoption guidance in template docs.

Tasks:

- [ ] Update `templates/scenarios/*` to the canonical package adoption pattern
- [ ] Remove outdated package adoption guidance from templates
- [ ] Ensure templates do not use prohibited package adoption modes
- [ ] If a sanctioned package-install hydration hook is needed, move it into lifecycle-native behavior instead of scenario-specific ad hoc scripts

Validation:

- [ ] Generated scenario from template passes scenario-auditor package rules
- [ ] Generated scenario behaves correctly with `vrooli package refresh`

### Phase 6 — Package Manifest Adoption

Goal:

- convert all existing packages to the new manifest model.

Deliverables:

- package manifests for every package in `packages/`.

Tasks:

- [ ] Add manifests for `api-base`, `iframe-bridge`, `vitest-requirement-reporter`
- [ ] Add manifest for `proto` with generated TS output contract
- [ ] Add manifests for `api-core`, `cli-core`, `testkit-go`
- [ ] Add manifests for any remaining package roots
- [ ] Update package READMEs to match manifest policy exactly
- [ ] Remove contradictory `workspace:*` guidance

Validation:

- [ ] `vrooli package validate --all` passes
- [ ] Docs reference manifest-governed behavior only

### Phase 7 — Scenario Migration

Goal:

- migrate real scenarios to the approved package adoption model.

Deliverables:

- all real scenarios conform to the new package-governance rules.

Tasks:

- [ ] Migrate all scenarios using unsupported `postinstall` package propagation
- [ ] Normalize package adoption in high-value scenarios first:
  - `swarm-manager`
  - `prompt-manager`
  - `scenario-auditor`
  - `test-genie`
- [ ] Migrate remaining real scenarios
- [ ] Update template-derived scenarios where current package adoption is stale
- [ ] Remove or rewrite docs that teach the old package adoption model

Validation:

- [ ] scenario-auditor package governance rules pass across migrated scenarios
- [ ] targeted `vrooli scenario setup` and `vrooli package refresh` checks pass on representative consumers

### Phase 8 — Legacy Deletion And Hard Cutover

Goal:

- remove all superseded governance surfaces.

Deliverables:

- no legacy shell package-refresh path;
- no contradictory docs;
- no legacy package-governance references.

Tasks:

- [ ] Delete `scripts/scenarios/tools/refresh-shared-package.sh`
- [ ] Remove shell-helper references from package READMEs and docs
- [ ] Remove outdated package propagation instructions from scenario docs
- [ ] Remove any temporary migration exceptions added during rollout
- [ ] Verify no stale `workspace:*` guidance remains for real scenario adoption

Validation:

- [ ] `rg` audit shows no remaining legacy refresh helper references
- [ ] `vrooli package ...` is the only documented governance path

## Validation And Testing Strategy

This plan is not complete without strong validation.

### Required Test Layers

1. Schema tests

- package manifest schema acceptance/rejection cases

2. Resolver tests

- package discovery
- dependent discovery
- consumer classification
- refresh strategy dispatch

3. CLI integration tests

- all `vrooli package` commands
- JSON output
- error paths

4. Scenario-auditor tests

- allowed/disallowed package adoption fixtures
- package policy rule coverage

5. Template tests

- generated scenario passes package governance validation

6. End-to-end lifecycle tests

- refresh a JS runtime package and verify:
  - package build runs
  - dependent scenario setup runs
  - running scenarios restart only when expected
- refresh a generated package and verify generation + propagation
- validate Go package flows where refresh means rebuild/restart rather than scenario setup

### Required Real-Scenario Validation Set

At minimum, validate against:

- `swarm-manager`
- `prompt-manager`
- `scenario-auditor`
- `test-genie`
- one simple template-derived scenario
- one scenario that consumes `@vrooli/proto-types`

## Open Design Questions To Resolve During Implementation

These questions have been resolved for implementation and are no longer open.

1. Package manifests will live at:
   - `packages/<name>/.vrooli/package.json`
2. Generated outputs like `@vrooli/proto-types` will be modeled as:
   - artifacts owned by `packages/proto`, declared in the `packages/proto/.vrooli/package.json` manifest
   - not as independently governed package roots by default
3. The sanctioned replacement for scenario-level package-copy/symlink `postinstall` behavior will be:
   - native lifecycle-owned hydration through `vrooli scenario setup` and related native lifecycle flows
   - not scenario-specific ad hoc package propagation scripts
4. Go package refresh behavior will be:
   - dependency discovery plus targeted rebuild/restart assistance
   - not JS-style scenario setup propagation by default
5. Templates will be validated by:
   - the same scenario-auditor package-governance rules as real scenarios
   - with explicit template-aware allowances only where structurally necessary

## Resolved Decisions

### Decision 1: Canonical Package Manifest Path

Canonical package manifest path:

- `packages/<name>/.vrooli/package.json`

This is required for every governed package root under `packages/`.

Reason:

- keeps governance metadata separate from language-specific package metadata;
- works for Go packages and other non-Node package roots;
- aligns with the existing Vrooli pattern of `.vrooli/` as the canonical governance/config namespace.

### Decision 2: Generated Outputs Are Owned By Their Source Package

Generated outputs such as `@vrooli/proto-types` are governed through their source package manifest:

- `packages/proto/.vrooli/package.json`

The generated TypeScript package under `packages/proto/gen/typescript` is an artifact output, not an independently governed package root by default.

Reason:

- generation ownership belongs to `packages/proto`;
- avoids split governance and duplicated package metadata;
- keeps “generate then refresh consumers” modeled as one package lifecycle.

### Decision 3: Native Lifecycle-Owned Hydration Replaces Scenario `postinstall` Package Propagation

Scenario-level package-copy/symlink `postinstall` behavior is not an allowed end state.

The replacement is:

- lifecycle-owned package hydration in native `vrooli` flows, primarily `vrooli scenario setup`

Reason:

- keeps package propagation visible, auditable, and policy-driven;
- avoids hidden package-manager-specific behavior buried inside scenario manifests;
- preserves scenario independence while centralizing governance in the platform control plane.

Implementation note:

- where plain `file:` dependency resolution is sufficient, no extra hydration should be performed;
- lifecycle-owned hydration exists only for package kinds and consumer combinations that actually require it.

### Decision 4: Go Package Refresh Is Dependency-Aware Rebuild/Restart Assistance

Go package refresh will not pretend to be JS package refresh.

For Go packages, `vrooli package refresh` will provide:

- dependent discovery;
- targeted rebuild/restart assistance for affected running consumers;
- package-kind-specific behavior for Go runtime and Go CLI packages.

It will not, by default:

- run scenario setup for Go consumers as if they were JS package-manager installs.

Reason:

- Go consumers point at source via module/replace semantics rather than isolated install trees;
- the propagation problem is different and should be modeled honestly.

### Decision 5: Templates Share The Same Package Governance Rule System

Templates will be validated using the same package-governance rule family in scenario-auditor as real scenarios.

Template-specific behavior will be handled through explicit rule allowances or fixture modes, not through a separate validator.

Reason:

- one policy engine is cleaner than parallel rule systems;
- template validation should exercise the same architectural expectations as generated scenarios;
- reduces drift between greenfield generation and real scenario enforcement.

## Additional Policy Decisions

The implementation should also treat the following as normative decisions:

- real scenarios must not use `workspace:*` for shared package adoption;
- only packages explicitly marked `scenario_adoptable` may be consumed by scenarios;
- generated package outputs are governed by their source package manifest unless explicitly elevated to standalone package roots in a future plan;
- package refresh behavior must be package-kind-specific and must not collapse distinct language/runtime semantics into one fake universal flow.

## Recommendation

Proceed with this plan.

The current philosophy is sound:

- keep scenarios independent;
- allow package reuse only when explicitly intended;
- avoid monorepo lockstep coupling.

What is missing is a professional governance layer around that philosophy.

The correct long-term move is:

- manifest-driven package classification;
- native `vrooli package` commands;
- auditor-enforced adoption policy;
- removal of Bash-based package refresh logic;
- template-first standardization;
- hard deletion of old governance paths after parity is proven.

That path strengthens scenario independence rather than weakening it.
