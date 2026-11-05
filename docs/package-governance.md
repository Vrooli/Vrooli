# Package Governance

Structure Health is the `claim:package.manifest` sole authority for package structural validation. Its
generated catalog and coverage matrix define package manifests, module roots,
private-import boundaries, and local Go replace requirements:

- [Structure Health rule catalog](../scenarios/structure-health/docs/reference/structure-rules.md)
- [Structure Health coverage matrix](../scenarios/structure-health/docs/reference/structure-rule-coverage.md)

This document owns only package registry and lifecycle operations. The control
plane keeps these read/build/generate/test/refresh verbs because they operate on
the package registry; it does not emit a competing structural verdict.

## Lifecycle

```bash
vrooli package list
vrooli package info <name>
vrooli package dependents <name>
vrooli package build <name>
vrooli package generate <name>
vrooli package test <name>
vrooli package refresh <name> all
```

`package test` is also the freshness boundary for generated owners. Before a
named package target is delegated to Test Genie, the control plane reconciles
all repository-level generated owners, not only the target package's own
manifest. The repository-level path runs each owner's declared, context-free
`lifecycle.generate` or generated-output-owning `lifecycle.build` contract
through the freshness gate. Lifecycle scenario setup performs the same
repository-level preflight before component builds, so a scenario API, CLI, or
UI cannot compile against
a stale shared projection merely because it was started instead of tested.
Scenario-scoped build artifacts remain owned by scenario shared-package setup,
where the requesting scenario is available. The same boundary is used by
`vrooli scenario test`, so Test Genie does not need a private regeneration
implementation. This keeps the workflow generic for protobufs, selectors,
registries, descriptors, runtime bundles, and future generated owners. A
generator or build must be declared in the owner manifest; declaring
`generated_outputs` without either lifecycle command is invalid and fails
closed before a consumer build. Owner commands must declare their output paths,
remain atomic, and be safe to rerun when inputs are unchanged.

Non-interactive refreshes and `--no-restart` must not stop running consumers.
They reconcile package outputs but defer consumer setup when it would require
stopping an active scenario, reporting `running_setup_deferred`. A successful
package refresh therefore does not imply that every running consumer adopted the
new artifact. Use the explicit restart option at a coordinated lifecycle boundary
when consumer setup and restart are required; a package that prohibits restarting
its consumers also leaves their setup deferred. Stopped consumers may run setup.
The lifecycle owner admits this setup only after checking for starting, running,
or stopping registry instances under the scenario lock, across all variants.
It holds the same lock through setup. If a consumer starts after refresh's
initial lookup, admission returns `running_setup_deferred` before setup effects.
A registry read failure fails admission; it does not establish that the consumer
is stopped.

Dependency changes and local Go-module reconciliation go through Scenario
Dependency Analyzer:

```bash
scenario-dependency-analyzer deps reconcile --all
scenario-dependency-analyzer deps reconcile --scenario <name> --apply
scenario-dependency-analyzer deps vendor --module packages/proto --preserve googleapis,protovalidate --apply
```

Do not hand-edit the approved dependency registry or run a raw package manager.
The `deps vendor` gateway is the repository-owned path for regenerating a
committed Go `vendor/` tree; it requires an explicit `--apply`, runs with
`GOWORK=off`, preserves explicitly named non-Go vendor inputs, and reports the
tool output as evidence.

The repository cross-compile gate sets `GOWORK=off` deliberately. A workspace
can hide a module's missing local replacement; the gate must compile each
module against its own `go.mod` so dependency drift remains visible.
Scenario UIs remain isolated projects and shared package adoption remains
explicit through their package manifests and local dependency declarations.

The React Component Library is an example of this contract. Its private
package lives at `packages/react-component-library`, is marked
`scenario_adoptable`, and exposes versioned component, hook, foundation, and
selector subpaths. Consumers must link it through Scenario Dependency
Analyzer and the library's `adoptions link` workflow; do not hand-edit a
scenario's dependency or copy package source into `ui/src`. Ejection is the
explicit, reason-bearing exception and is recorded as `mode=ejected`.

Packages that own a shared resource environment seam may declare resource
names in `package.adoption.owns_resource_environment`. The field is keyed by
canonical resource name, not individual environment variables; validators
derive the variable surface from the resource manifest.

## Shared TypeScript packages

The governed package set includes `react-component-library` at
`packages/react-component-library`. Its manifest is scenario-adoptable with
`adoption_modes: ["file_dependency"]`; consumers use version-pinned package
subpaths through `adoptions link`, and `adoptions eject --reason` is the only
source-materializing exception.

Scenario setup provisions every governed `file:` dependency under `packages/`
before `install-ui-deps`. Each TypeScript package's build lifecycle first
installs its frozen lockfile outside the repository root workspace, then
produces compiled JavaScript and declaration
files from their declared output directory; consumers must not alias an
`@vrooli/*` package into `packages/*/src`. The clean-environment proof lives
in the `Clean shared-package provisioning` job in `.github/workflows/test.yml`.

A package whose build consumes another governed package's generated outputs
declares it in `lifecycle.requires` (package names under `packages/`). Setup
adds those packages even when the scenario does not depend on them directly and
builds them first. Without the declaration the build passes only where the
other package's gitignored outputs already exist; a fresh checkout or a
working-tree node fails it. The installed control-plane CLI decodes these
manifests strictly, so a new manifest key reaches `internal/packagegov`, the
schema, and `make install` before any manifest uses it.

The build-output digest is part of UI freshness, so rebuilding a shared package
invalidates the consumer's installed copy on its next setup. Generated outputs
remain lifecycle-owned artifacts and are not hand-built by scenario operators.

The React Component Library additionally publishes immutable snapshots under
`$HOME/.vrooli/artifacts/react-component-library` (or the configured runtime
artifact root). `current.json` selects a verified snapshot and
`runtime-status.json` records the latest candidate outcome. Its governed build
command declares `artifact_selection`, so lifecycle freshness includes the
selected snapshot identity: draft catalog edits do not invalidate compatible
running consumers, while a newly selected artifact does. A failed candidate
may return a degraded success only when the selected snapshot is intact and
contains every exact export requested by the consumer; missing, corrupt, or
incompatible artifacts remain hard failures.

Provisioning is single-flight at two levels: lifecycle phase entrypoints hold
the scenario lock, and each governed package is protected by a shared-package
lock under `$HOME/.vrooli/state/locks`. This is required because different
scenario consumers can provision the same `file:` dependency concurrently.
Lifecycle-owned package commands receive the setup environment and EOF stdin,
so package-manager prompts fail deterministically and cannot wait on an
operator terminal. Lock waits and command durations are recorded in the
scenario lifecycle log with the package root, process ID, and wait duration.

Shared package lifecycle commands may use the `{scenario}` substitution token
when the command needs to know which scenario requested it. The lifecycle
replaces the token with that scenario name; when there is no requesting
scenario, it removes the `--scenario {scenario}` argument (or its
`--scenario={scenario}` form) so the command remains valid when run directly.
Package manifests should use this token only for requester context, not as a
second package identity.

## Proto artifact selection

`packages/proto` is a governed generated contract package. Its lifecycle
selection is `@vrooli/proto-types`, and scenario setup treats it as a reader of
the selected runtime artifact rather than as an instruction to regenerate
against the mutable shared worktree. The selected artifact is resolved from
`$HOME/.vrooli/artifacts/proto` and materialized into the fixed local module
path only after metadata and output digests pass validation.

Proto generation and lifecycle setup use the same cross-process lock at
`$HOME/.vrooli/locks/proto-generation.lock`. Lifecycle holds that lock through
compatibility-view installation and the consumer build. A failed or cancelled
refresh therefore cannot replace the active or last-known-good record, and a
scenario with an available valid snapshot does not wait for unrelated source
edits to become valid.

The shared-worktree rule is intentional: agents continue to collaborate in one
checkout. Reliability comes from closure digests, private candidates,
immutable snapshots, atomic selection records, reader leases, and explicit
refresh/promotion/rollback operations. A missing valid snapshot is a precise
startup failure; silently falling back to `packages/proto/gen` is not allowed.
