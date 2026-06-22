# Package Governance

Vrooli scenarios stay intentionally independent. They do not join the root pnpm workspace, and shared packages are only supported through explicit, manifest-driven adoption rules.

## Rules

- Every package root under `packages/` must declare [`packages/<name>/.vrooli/package.json`](../packages/api-base/.vrooli/package.json).
- `vrooli package ...` is the canonical operator surface for package discovery, validation, build/generate, refresh, and audit.
- `scenario-stack-governor` exposes scenario-scoped package governance enforcement through the `PACKAGE_GOVERNANCE_SCENARIO_ADOPTION` external rule consumed by `scenario-auditor`.
- Real scenarios must not use workspace-star dependencies for shared package adoption.
- Shared package propagation must not rely on scenario-local `postinstall` copy/symlink hacks.
- Only packages marked `scenario_adoptable` may be consumed by governed external consumers such as scenarios, templates, or resources.
- Leaf shared Go packages must not take on new local governed package dependencies unless that coupling is explicitly allowed by policy.

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
   - Go consumers must build as the main module under `GOWORK=off`; dependency-module `replace` directives do not count as sufficient
3. Do not use workspace-star dependencies.
4. Do not add shared-package copy/symlink logic to `postinstall`.

### Leaf shared Go package policy

Some shared Go packages are governed as leaves. They are intended to be
directly reusable by consumers without forcing those consumers to adopt a wider
local package graph.

Current leaf packages:

- `cli-core`
- `repo-contract-go`

Implications:

- `cli-core` must stay self-contained relative to other governed local Go
  packages, except for explicitly allowed leaf-safe dependencies such as
  `repo-contract-go`.
- If a shared package only needs a wire-format DTO or a tiny contract shape, it
  should usually define that decode shape locally instead of importing another
  governed local package just for convenience.
- Consumer correctness is validated from the consumer module's point of view,
  not from the dependency package's own module context.

### Refreshing package consumers

1. Run `vrooli package dependents <name>` to inspect the affected consumers.
2. Run `vrooli package refresh <name> <consumer|all>` to apply the package's governed refresh behavior.
3. Use `--no-restart` when you want setup/rebuild propagation without restarting running consumers.
4. Re-run `vrooli package validate --all` after large migrations or manifest changes.

### Reconciling in-repo go.mod replaces

Go does not propagate a dependency's `replace` directives to downstream main
modules. So whenever a shared Go package takes on a **new in-repo module edge**
(for example a leaf package importing `packages/proto`), every surface that
transitively requires that module must independently declare its own local
`replace` — otherwise `go build` (and therefore `vrooli scenario restart`) fails
with a `missing go.sum entry` error that only surfaces at restart time.

After editing a shared Go package, reconcile consumers with the single
SDA-owned command instead of hand-editing each `go.mod`:

```bash
# dry run: report surfaces missing a local replace (whole fleet)
scenario-dependency-analyzer deps reconcile --all

# apply the missing replaces + go mod tidy for one scenario
scenario-dependency-analyzer deps reconcile --scenario <name> --apply
```

`deps reconcile` is dry-run by default, idempotent, and safety-scoped: it only
adds a local `replace` for a module that resolves unambiguously to one in-repo
module directory; third-party dependencies stay under approved-dependencies
governance and are never touched. The same detection runs automatically as the
Test Genie dependencies phase (an ERROR finding,
`dependency.gomod.replace.missing`), so the gap fails CI before a human hits a
broken restart.

Refresh behavior is consumer-type-aware:
- real scenario consumers can run setup and optional restart flows
- Go consumers such as scenario CLIs/APIs or resources rebuild where appropriate
- template consumers are reported explicitly and never treated as runnable scenarios

## CI And Validation

- `make validate-package-governance` is the canonical repo-level validation target for package manifests, package refresh coverage, and governance drift.
- `make validate-go-cli-consumers` is the canonical isolated-build check for scenario/resource CLI Go modules and must stay green alongside package governance.
- CI runs that target directly, which means package governance must stay green at the CLI level and through the `scenario-stack-governor` rule surface.

## Why This Exists

This model preserves scenario independence across languages, frameworks, and package managers while still allowing disciplined reuse. The package manifest declares what a package is, who may adopt it, how it is refreshed, and how governance is validated.
