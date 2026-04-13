# Package Governance

Vrooli scenarios stay intentionally independent. They do not join the root pnpm workspace, and shared packages are only supported through explicit, manifest-driven adoption rules.

## Rules

- Every package root under `packages/` must declare [`packages/<name>/.vrooli/package.json`](../packages/api-base/.vrooli/package.json).
- `vrooli package ...` is the canonical operator surface for package discovery, validation, build/generate, refresh, and audit.
- `scenario-stack-governor` exposes scenario-scoped package governance enforcement through the `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION` external rule consumed by `scenario-auditor`.
- Real scenarios must not use workspace-star dependencies for shared package adoption.
- Shared package propagation must not rely on scenario-local `postinstall` copy/symlink hacks.
- Only packages marked `scenario_adoptable` may be consumed by governed external consumers such as scenarios, templates, or resources.

## Supported Adoption Modes

- `file_dependency`
  - For isolated JS/TS consumers using relative `file:` dependencies.
- `go_module_replace`
  - For isolated Go consumers using local `replace` directives.
- `generated_artifact`
  - For generated outputs governed by a source package such as `packages/proto`, even when the consumer references the generated artifact through an isolated local dependency path.

## Common Commands

```bash
vrooli package list
vrooli package info api-base
vrooli package dependents iframe-bridge
vrooli package validate --all
vrooli package refresh api-base all
vrooli package audit --all
```

## Workflow

### Adding a governed package

1. Create `packages/<name>/.vrooli/package.json`.
2. Declare the package `kind`, `module_identifiers`, allowed consumers, adoption modes, lifecycle commands, and refresh strategy.
3. Document the package in its README and link that doc from the manifest `docs` field.
4. Run `vrooli package validate <name>` and `make validate-package-governance`.

### Adopting a package in a scenario or resource

1. Only adopt packages whose manifest explicitly allows your consumer class.
2. Use the declared adoption mode only:
   - JS/TS isolated consumers use `file:`
   - Go consumers use governed `replace` directives
   - generated artifacts are adopted through their source package contract
3. Do not use workspace-star dependencies.
4. Do not add shared-package copy/symlink logic to `postinstall`.

### Refreshing package consumers

1. Run `vrooli package dependents <name>` to inspect the affected consumers.
2. Run `vrooli package refresh <name> <consumer|all>` to apply the package's governed refresh behavior.
3. Use `--no-restart` when you want setup/rebuild propagation without restarting running consumers.
4. Re-run `vrooli package validate --all` after large migrations or manifest changes.

Refresh behavior is consumer-type-aware:
- real scenario consumers can run setup and optional restart flows
- Go consumers such as scenario CLIs/APIs or resources rebuild where appropriate
- template consumers are reported explicitly and never treated as runnable scenarios

## CI And Validation

- `make validate-package-governance` is the canonical repo-level validation target for package manifests, package refresh coverage, and governance drift.
- CI runs that target directly, which means package governance must stay green at the CLI level and through the `scenario-stack-governor` rule surface.

## Why This Exists

This model preserves scenario independence across languages, frameworks, and package managers while still allowing disciplined reuse. The package manifest declares what a package is, who may adopt it, how it is refreshed, and how governance is validated.
