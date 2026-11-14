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

Every directory under `packages/` must carry a loadable `.vrooli/package.json`.
`go test ./internal/packagegov` enforces this, because an unmanifested package
is invisible to the registry and therefore to refresh planning, adoption checks
and the [shared-package baseline](TESTING.md#shared-package-baseline). A Go
package that stays inside the repository module records
`module_boundary.status: intentional_exception` with the reason.

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

Dependency changes go through Scenario Dependency Analyzer. Do not hand-edit
the approved dependency registry, a `go.mod`, or a `go.sum`, and do not run a
raw package manager (`go get`, `go mod tidy`, `pnpm add`, …) to change a
module. Read-only checks such as `go mod tidy -diff` are fine.

```bash
scenario-dependency-analyzer deps reconcile --all-modules            # dry run: every module that would change
scenario-dependency-analyzer deps reconcile --all-modules --apply
scenario-dependency-analyzer deps reconcile --scenario <name> --apply
scenario-dependency-analyzer deps vendor --module packages/proto --preserve googleapis,protovalidate --apply
```

The `deps vendor` gateway is the repository-owned path for regenerating a
committed Go `vendor/` tree; it requires an explicit `--apply`, runs with
`GOWORK=off`, preserves explicitly named non-Go vendor inputs, and reports the
tool output as evidence.

The repository cross-compile gate sets `GOWORK=off` deliberately. A workspace
can hide a module's missing local replacement; the gate must compile each
module against its own `go.mod` so dependency drift remains visible.
Scenario UIs remain isolated projects and shared package adoption remains
explicit through their package manifests and local dependency declarations.

## Go module convergence

Every scenario, resource, and package Go module is its own module. It binds
shared in-repo modules (`packages/*`, scenario APIs, and the repository root
through `replace github.com/vrooli/vrooli => ../../..`) with local `replace`
directives. A replace shares source, not the module graph: Go propagates
neither replaces nor checksums. Each consumer's own `go.mod` and `go.sum` must
therefore record:

- every requirement and checksum its import closure reaches, including those
  that arrive only through a shared module's imports, and
- a local replace for every in-repo module among those requirements.

So a single new import in a shared package can break every consumer that
lacks the matching requirement. On 2026-09-01 one call added to
`packages/api-core` broke 98 of 118 scenario modules. On 2026-11-06 one
`golang.org/x/mod` call in `internal/buildinfo` left about 30 modules short.

`scenario-dependency-analyzer deps reconcile` is the one repair path. For each
selected module it computes the converged `go.mod` and `go.sum` on a scratch
copy, running `go mod tidy` against the copy (`-modfile`). The real files are not
touched until `--apply`.

- It seeds a local replace for every in-repo module reachable through in-repo
  requirements, so tidy never tries to download an unpublished module.
- After tidy, it drops each in-repo replace whose module the tidied `go.mod`
  does not require. Go prunes the module graph and loads only the modules the
  main `go.mod` lists, so those replaces are dead.
- A final tidy verifies the pruned state. If it fails, or records anything new,
  the seeded state is kept.
- `go.sum` is put in canonical order.
- For a module with a committed `vendor/` tree (`packages/proto`,
  `packages/delivery-ramp-go`), it runs `go mod vendor` for the converged
  `go.mod` into a staging directory inside the module. Top-level `vendor/`
  entries that `modules.txt` does not own, such as the `googleapis` and
  `protovalidate` sources, are carried over. On `--apply` the staged tree
  replaces `vendor/` in the same write as `go.mod` and `go.sum`. Go refuses a
  module whose `vendor/modules.txt` disagrees with its requirements or replace
  directives ("inconsistent vendoring"), and vendored copies of locally
  replaced in-repo modules go stale when those modules change. Both count as
  drift and are repaired the same way.

The result agrees with plain `go mod tidy`. Requirements that the module would newly record then
pass the dependency gate:

| Requirement after convergence | Rule |
|---|---|
| Indirect (reached through another module) | Recorded without a new decision. The version comes from minimal version selection over requirements the repository already declares, and the module that requires it directly made the dependency decision. Recorded decisions still bind it: a denial refuses it, and a version outside a recorded approval range is refused with the `deps approved widen-range` command. An unrecorded indirect requirement is transitive, as in validation. |
| Direct (the module's own source imports it) | The same fail-closed gate as `deps install`: recorded, approved for the scenario and surface, and in range. |

The indirect rule is not new policy. Approved-dependency validation already
skips unrecorded Go requirements in the `require_indirect` group as transitive:
`isUnrecordedTransitiveDependency` (`scenarios/scenario-dependency-analyzer/api/internal/dependencygovernance/registry.go:2768`,
applied in `ValidateObserved` at line 644). That behaviour is covered by
`TestValidateObservedDoesNotWarnForUnrecordedIndirectGoDependency` in
`registry_test.go`. The gate (`sync_gate.go`, `GateRequirementSync`) reuses that
rule. It also keeps the recorded checks that validation applies to every
dependency: the denial rule (`DENIED_IN_USE`) and the version range.

A refused module is left untouched. The command exits non-zero and names the
exact `deps approved …` command that would unblock it. Writes are atomic per
file and refuse if `go.mod` or `go.sum` changed after the preview was computed.
Removing unused requirements and dead local replaces, dropping stale checksums, and fixing ordering or
`// indirect` markers are housekeeping: they are applied without a decision.

`--scenario <name>` covers every module in the scenario, including root and
runtime modules. `--module <dir>` converges exactly one module, which may sit
outside the repository (for example a scenario being generated). `--all-modules` covers every buildable module in the
repository. Both skip `docs/`, `testdata/`, `evidence/`, `fixtures/`, `bas/`,
`templates/` and `.vrooli/`, whose modules are retained or synthetic. `--check`
is a dry run that exits non-zero when any file would change.

Drift is reported wherever an agent already looks, and every report names the
reconcile command:

- The test-genie `dependencies` phase reports `dependency.go.tidy` and
  `dependency.gomod.replace.missing` for each Go surface. Both are deterministic
  fixes served by the same convergence.
- `vrooli hygiene` runs `scenario-dependency-analyzer freshness --touched`. It
  maps changed in-repo modules to every consumer surface they reach, so the
  author of a shared-package change sees every consumer it leaves stale.
- When a Go component fails to build, lifecycle setup checks the module
  read-only (`go list -mod=readonly`). If the build failed because `go.mod` or
  `go.sum` lags the import closure, the error says so and names the reconcile
  command, in place of the toolchain's `go get`/`go mod tidy` advice. Because
  `vrooli scenario restart` builds replacement artifacts before it stops the
  running instance, this failure never takes a running scenario down.

### Docs evidence must not join a module

Go compiles every directory under a module root into that module unless the
directory starts with `.` or `_`, is named `testdata`, or holds its own
`go.mod`. Retained source copies kept as evidence under a scenario's `docs/` are
real Go packages, so without a boundary they join the enclosing module (usually
the repository root) and pull their imports into its requirements. On
2026-11-06, evidence under `scenarios/nutrition-planner/docs/` and
`scenarios/workflow-health/docs/` made the repository root require
nutrition-planner, workflow-health, browser-automation-studio, and maturity-go.

The docs-evidence convention fences each documentation tree that holds Go
source with a `go.mod` at the top of its `docs/` directory:

```
module github.com/vrooli/vrooli/scenarios/<scenario>/docs-evidence

go 1.26.0
```

`docs/go.mod` and `scenarios/device-studio/docs/go.mod` are examples. A
documentation tree is a `docs/` directory that holds Markdown. A production
package that is merely named `docs`, such as a CLI `domains/docs`, is not one.
Two checks enforce the convention:

- The test-genie `dependencies` phase reports
  `dependency.gomod.evidence.unfenced` for Go packages under a scenario's
  documentation tree that have no module boundary.
- `deps reconcile` refuses to converge a module that contains such packages, so
  their imports never enter its `go.mod`.

Template Manager converges every Go module it generates the same way. Scenario
and resource generation, template validation, and detemplate finalizers run
`scenario-dependency-analyzer deps reconcile --module <dir> --apply`, not a raw
`go mod tidy`. A template's replace list therefore cannot leave a new scenario
unbuildable or carrying dead replaces after shared packages change. The template
`go.mod` files are kept in the converged shape, so convergence of a fresh
scenario is a no-op.

After changing a shared Go package, run
`scenario-dependency-analyzer deps reconcile --all-modules --apply` once. It
converges every consumer; already converged modules are a no-op.

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
