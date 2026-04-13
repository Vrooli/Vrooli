# Package Governance

Vrooli scenarios stay intentionally independent. They do not join the root pnpm workspace, and shared packages are only supported through explicit, manifest-driven adoption rules.

## Rules

- Every package root under `packages/` must declare [`packages/<name>/.vrooli/package.json`](../packages/api-base/.vrooli/package.json).
- `vrooli package ...` is the canonical operator surface for package discovery, validation, build/generate, refresh, and audit.
- `scenario-stack-governor` exposes scenario-scoped package governance enforcement through the `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION` external rule consumed by `scenario-auditor`.
- Real scenarios must not use `workspace:*` for shared package adoption.
- Shared package propagation must not rely on scenario-local `postinstall` copy/symlink hacks.
- Only packages marked `scenario_adoptable` may be consumed by scenarios or templates.

## Supported Adoption Modes

- `file_dependency`
  - For isolated JS/TS consumers using relative `file:` dependencies.
- `go_module_replace`
  - For isolated Go consumers using local `replace` directives.
- `generated_artifact`
  - For generated outputs governed by a source package such as `packages/proto`.

## Common Commands

```bash
vrooli package list
vrooli package info api-base
vrooli package dependents iframe-bridge
vrooli package validate --all
vrooli package refresh api-base all
vrooli package audit --all
```

## CI And Validation

- `make validate-package-governance` is the canonical repo-level validation target for package manifests, package refresh coverage, and governance drift.
- CI runs that target directly, which means package governance must stay green at the CLI level and through the `scenario-stack-governor` rule surface.

## Why This Exists

This model preserves scenario independence across languages, frameworks, and package managers while still allowing disciplined reuse. The package manifest declares what a package is, who may adopt it, how it is refreshed, and how governance is validated.
