# Package Adoption And Refresh Governance Implementation Record

**Status:** Implemented and validated

**Last Updated:** 2026-04-13

## Summary

The package adoption and refresh governance rollout is complete.

Vrooli now governs shared packages through a native manifest-driven model with a first-class `vrooli package ...` command surface. The system preserves scenario independence, supports multiple language ecosystems, and enforces package adoption policy through the canonical CLI plus stack-governor and scenario-auditor integration.

## Goals Achieved

- Every package root under `packages/` declares a canonical `.vrooli/package.json` manifest.
- Shared package adoption is explicit and policy-driven.
- `vrooli package ...` is the supported operator surface for package discovery, validation, generation, build, refresh, and audit.
- Scenario adoption rules are enforced through `scenario-stack-governor` and exposed in `scenario-auditor`.
- Legacy Bash refresh helpers have been removed.
- The implementation preserves scenario independence and does not require scenarios to join the root pnpm workspace.

## Canonical Model

### Package manifests

Every governed package root must declare:

- `packages/<name>/.vrooli/package.json`

The manifest is the source of truth for:

- package identity and kind
- scenario adoptability
- allowed consumer classes
- supported adoption modes
- build/generate lifecycle commands
- refresh behavior
- supporting documentation

Schema:

- [package.schema.json](/home/matthalloran8/Vrooli/.vrooli/schemas/package.schema.json:1)

### Supported package kinds

Current governed kinds include:

- `js_runtime`
- `go_runtime`
- `go_cli`
- `go_testkit`
- `schema_or_contract`

### Supported adoption modes

- `file_dependency`
- `go_module_replace`
- `generated_artifact`

### Operator surface

The native command surface is:

```bash
vrooli package list
vrooli package info <name>
vrooli package dependents <name>
vrooli package validate <name|--all>
vrooli package build <name>
vrooli package generate <name>
vrooli package refresh <name> <consumer|all>
vrooli package audit <name|--all>
```

Canonical documentation:

- [docs/package-governance.md](/home/matthalloran8/Vrooli/docs/package-governance.md:1)

## Governance Rules

The enforced policy is:

- scenarios remain intentionally independent from the root workspace
- real scenarios must not use `workspace:*` for shared package adoption
- only packages marked `scenario_adoptable` may be adopted by governed consumers
- consumers may only use adoption modes explicitly declared by the package manifest
- shared-package propagation must not rely on scenario-local package-copy/symlink `postinstall` hacks
- refresh behavior is package-kind-specific rather than pretending every language/runtime behaves the same way

## Package-Class Semantics

### JS/TS runtime packages

Examples:

- `api-base`
- `iframe-bridge`
- `vitest-requirement-reporter`

Governed behavior:

- adopted through isolated `file:` dependencies
- refreshed through package build plus `vrooli scenario setup` on affected consumers
- optionally restart only consumers that were already running

### Go shared packages

Examples:

- `api-core`
- `cli-core`
- `repo-contract-go`

Governed behavior:

- adopted through Go module imports and local `replace` directives
- do not use JS-style reinstall/setup propagation
- use rebuild/restart assistance where appropriate through `vrooli package refresh`

### Generated contract packages

Example:

- `packages/proto`

Governed behavior:

- generated outputs are owned by the source package manifest
- generation happens before refresh when required
- TS consumers use governed local `file:` adoption
- Go consumers use governed local `replace` adoption

## Enforcement Surfaces

### Native validation

Repo-level validation runs through:

```bash
make validate-package-governance
```

This validates:

- package manifest schema and semantics
- CLI package-governance behavior
- stack-governor integration
- package adoption drift detected by `vrooli package validate --all`
- package policy drift detected by `vrooli package audit --all`

### Stack-governor and scenario-auditor

- `scenario-stack-governor` exposes `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION`
- that rule delegates to `vrooli package audit --all`
- `scenario-auditor` consumes the external rule so scenario-scoped enforcement stays aligned with the canonical package-governance engine

Relevant files:

- [scenarios/scenario-stack-governor/api/rules.go](/home/matthalloran8/Vrooli/scenarios/scenario-stack-governor/api/rules.go:73)
- [scenarios/scenario-stack-governor/api/rule_package_governance.go](/home/matthalloran8/Vrooli/scenarios/scenario-stack-governor/api/rule_package_governance.go:15)
- [scenarios/scenario-auditor/api/external_rules_stack_governor.go](/home/matthalloran8/Vrooli/scenarios/scenario-auditor/api/external_rules_stack_governor.go:71)

## Final Decisions

The implementation resolved the open design questions as follows:

1. Package manifests live at `packages/<name>/.vrooli/package.json`.
2. Generated outputs such as `@vrooli/proto-types` are governed by the source package manifest under `packages/proto/.vrooli/package.json`.
3. Native lifecycle-owned hydration replaces scenario-level shared-package propagation hacks.
4. Go package refresh behavior is rebuild/restart assistance rather than JS-style reinstall parity.
5. Templates are validated using the same package-governance rule family as real scenarios, with template-aware allowances where needed.

## Documentation Surfaces

Primary documentation:

- [docs/package-governance.md](/home/matthalloran8/Vrooli/docs/package-governance.md:1)
- [docs/CONTRIBUTING.md](/home/matthalloran8/Vrooli/docs/CONTRIBUTING.md:146)

Package-specific documentation:

- [packages/api-base/README.md](/home/matthalloran8/Vrooli/packages/api-base/README.md:94)
- [packages/api-core/README.md](/home/matthalloran8/Vrooli/packages/api-core/README.md:33)
- [packages/cli-core/README.md](/home/matthalloran8/Vrooli/packages/cli-core/README.md:33)
- [packages/iframe-bridge/README.md](/home/matthalloran8/Vrooli/packages/iframe-bridge/README.md:57)
- [packages/proto/README.md](/home/matthalloran8/Vrooli/packages/proto/README.md:89)
- [packages/testkit-go/README.md](/home/matthalloran8/Vrooli/packages/testkit-go/README.md:65)
- [packages/vitest-requirement-reporter/README.md](/home/matthalloran8/Vrooli/packages/vitest-requirement-reporter/README.md:48)

## Validation Record

Verified as part of rollout completion:

- all current package roots under `packages/` have manifests
- native CLI package commands are available and working
- package validation and audit pass repo-wide
- stack-governor package-governance integration tests pass
- scenario-auditor external-rule integration tests pass through retained validation coverage

Representative validation commands:

```bash
make validate-package-governance
vrooli package list
vrooli package info api-base
vrooli package dependents api-base
vrooli package build api-base
vrooli package generate proto
vrooli package refresh api-base <consumer> --no-restart
vrooli package refresh api-core <consumer> --no-restart
vrooli package audit --all
```

## Long-Term Invariants

Future work must preserve these constraints:

1. Scenario independence is mandatory.
2. Shared package adoption must remain explicit.
3. Package governance must remain language-agnostic at the architecture layer.
4. Language-specific behavior must stay package-kind-specific.
5. Native `vrooli` commands remain the only supported operator path.
6. No return to shell-helper-based governance or undocumented package propagation behavior.
