# Resource/Scenario CLI Installed Artifact Unification Greenfield Plan

## Status

- Proposed
- Greenfield
- No compatibility path

## Purpose

This plan defines the clean long-term standard for Go CLIs in Vrooli across both:

- scenarios, which declare CLI contract in `.vrooli/service.json`
- resources, which declare CLI contract in `resource.json`

The goal is not merely to make both kinds of CLIs "similar." The goal is to make them use:

- one shared CLI contract model
- one shared installer/build pipeline
- one shared installed-artifact model
- one clean runtime lookup strategy

This plan explicitly rejects compatibility shims, embedded manifest fallbacks, root-module escape hatches, and other transitional patterns as target architecture.

## Greenfield Constraint

This work is explicitly greenfield.

That means:

- do not preserve old resource-specific or scenario-specific packaging behavior just because it exists today
- do not add compatibility branches for "embedded manifest if installed manifest is missing"
- do not keep root shim Go modules for Go-native resources
- do not keep parallel installer semantics between scenarios and resources
- do not document legacy packaging patterns as acceptable alternatives

If a code path is migrated under this plan, the new standard should become the only recommended and documented standard.

## Why This Exists

At the binary build level, scenarios and resources are already close:

- both use `cli.adapter.kind = "go_module"`
- both point at a `cli/` Go module
- both use the shared `cli-core` installer/builder

Key shared implementation points today:

- shared CLI config shape in [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go:1)
- resources already reuse that CLI config in [internal/resources/manifest/manifest.go](/home/matthalloran8/Vrooli/internal/resources/manifest/manifest.go:1)
- shared installation/discovery path in [internal/cliinstall/cliinstall.go](/home/matthalloran8/Vrooli/internal/cliinstall/cliinstall.go:1)
- shared installer in [packages/cli-core/cmd/cli-installer/main.go](/home/matthalloran8/Vrooli/packages/cli-core/cmd/cli-installer/main.go:1)

So the repo is already partway to convergence.

The remaining gap is not primarily "resource manifests are missing a CLI section." They already reuse the same core `CLIConfig` type. The real gap is that installed CLIs are still treated mostly as standalone binaries, not as fully packaged installed artifacts with companion metadata.

That gap leads directly to architectural distortions like:

- embedding `resource.json` into a binary because the installer only copies the binary
- keeping tiny root Go modules only to make embedded manifest helpers importable
- scenario/resource parity feeling incomplete even when both build through the same installer

## Problem Statement

The current shared installer builds and installs binaries well, but it does not install companion manifest metadata as part of the same atomic installed artifact.

Consequences:

1. Installed CLIs cannot rely on manifest metadata being present beside the binary.
2. Runtime code falls back to source-tree assumptions or embedded manifest bytes.
3. Go-native resources like `sqlite` accumulate structural exceptions.
4. Scenarios and resources share installer code, but not a fully equivalent artifact model.
5. The repo teaches `cli/` as the canonical entrypoint, but runtime packaging still leaks source-layout assumptions.

## Design Decision

The correct long-term model is:

- one shared CLI contract shape for scenarios and resources
- one shared installer implementation
- one shared installed-artifact layout
- no embedded manifest fallback

This does **not** require collapsing `resource.json` into `service.json`.

It **does** require making the `cli` subsection of both manifest systems behave the same way.

## Target End State

### Shared CLI Contract

Both scenario and resource manifests should treat `cli` as a first-class installed artifact contract with the same semantic model:

- `enabled`
- `command`
- `adapter.kind`
- `adapter.module_dir`
- `install`
- `invoke`
- `freshness`
- installed companion artifact declarations

The surrounding manifest remains different:

- scenarios still have scenario lifecycle, ports, dependencies, health, and deployment metadata
- resources still have driver/runtime/platform/orchestration metadata

But the `cli` contract should be conceptually and structurally aligned.

### Shared Installed Artifact Layout

Every installed Go CLI should produce a complete installed artifact set:

- installed binary
- sibling manifest file
- sibling build metadata file

Example scenario install result:

```text
~/.vrooli/bin/test-genie
~/.vrooli/bin/test-genie.manifest.json
~/.vrooli/bin/test-genie.build.meta
```

Example resource install result:

```text
~/.vrooli/bin/resource-sqlite
~/.vrooli/bin/resource-sqlite.manifest.json
~/.vrooli/bin/resource-sqlite.build.meta
```

The binary should not need embedded manifest fallback to operate outside the repo.

### Shared Runtime Lookup Rule

Installed CLIs should resolve manifest metadata in this order:

1. installed sibling manifest beside the executable
2. explicit dev/source override only when intentionally running from source in development

They should **not** rely on:

- embedded `resource.json`
- embedded `service.json`
- source-tree manifests as the installed runtime contract

### Go Module Layout

For Go-native resource CLIs, the canonical structure should be:

```text
resources/<name>/
├── resource.json
├── README.md
├── docs/
├── config/
├── test/
└── cli/
    ├── go.mod
    ├── go.sum
    ├── main.go
    ├── install.sh
    ├── install.ps1
    └── internal/
```

No root shim Go module.
No root `runtime_embed.go`.
No root install wrappers.

Scenario Go CLIs already largely follow the equivalent pattern from their own root.

## Principles

1. **Single source of truth**
   Installed CLI metadata must come from an installed manifest artifact, not duplicated embedding tricks.

2. **One installer path**
   Scenario and resource Go CLIs should reuse the same installer and builder behavior.

3. **One artifact model**
   Scenario CLIs and resource CLIs should be packaged the same way once installed.

4. **No runtime guessing**
   Runtime lookup must be deterministic and explicit.

5. **No structural special cases**
   Go-native resources should not need special root-module exceptions once the packaging model is fixed.

6. **Manifest contract over file-layout folklore**
   Installed behavior should be driven by manifest-declared CLI contract, not inferred source layout.

## Current State Summary

### What Already Converges

- both scenarios and resources can declare `cli.adapter.kind = "go_module"`
- both use the same installer family in `cli-core`
- both are already handled by the same install-manager concepts in `internal/cliinstall`
- resources already reuse the shared `CLIConfig` type from the scenario manifest package

### What Does Not Yet Converge

- installed manifest metadata is not treated as a first-class artifact
- runtime lookup still assumes missing manifest metadata is acceptable
- SQLite still needs a root shim solely to embed manifest bytes
- templates do not yet express a fully packaged installed-artifact story

## Scope

### In Scope

- shared CLI contract shape for scenarios/resources
- `cli-core` installer and builder behavior
- installed artifact naming/location conventions
- runtime manifest lookup for installed Go CLIs
- template updates for scenarios and resources
- migration of Go-native resources already being standardized
- tests and validation required to lock in the new standard

### Out Of Scope

- scenario lifecycle redesign outside the CLI artifact contract
- resource driver redesign unrelated to CLI install/packaging
- shell CLIs that are not being migrated to Go
- preserving old packaging behavior

## Implementation Plan

### Phase 0: Confirm And Freeze The Contract

Define the canonical installed-artifact contract in one place before touching implementation.

Deliverables:

- one documented sibling manifest naming convention
- one documented sibling build metadata naming convention
- one documented rule for runtime manifest lookup
- one documented statement that embedded manifest fallback is not part of the target architecture

Recommended convention:

- binary: `<command>`
- manifest: `<command>.manifest.json`
- build metadata: `<command>.build.meta`

### Phase 1: Extend The Shared CLI Contract Model

Update the shared CLI manifest model in [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go:1) so both scenarios and resources can declare installed companion artifacts explicitly.

Because resources already reuse `scenario.CLIConfig`, this phase should be designed to make the shared type more expressive rather than introducing a resource-only branch.

Goals:

- keep the shared shape minimal
- avoid duplicating CLI contract types between scenarios and resources
- make installed companion metadata a first-class concept instead of hidden installer behavior

The new fields should be generic rather than service/resource-specific.

### Phase 2: Upgrade `cli-core` Installer To Install Companion Manifest Metadata

Update [packages/cli-core/cmd/cli-installer/main.go](/home/matthalloran8/Vrooli/packages/cli-core/cmd/cli-installer/main.go:1).

Required behavior:

- accept a manifest source path
- install the manifest beside the binary using the canonical naming rule
- continue writing build metadata beside the binary
- treat binary + manifest + build meta as one installed artifact transaction

Constraints:

- keep this generic
- do not introduce resource-only code paths
- do not require source-tree-specific assumptions at runtime

Also update:

- [packages/cli-core/install.sh](/home/matthalloran8/Vrooli/packages/cli-core/install.sh:1)
- [packages/cli-core/install.ps1](/home/matthalloran8/Vrooli/packages/cli-core/install.ps1:1)
- `cli-core` README and tests

### Phase 3: Update Shared Install Manager To Pass Manifest Inputs Uniformly

Update [internal/cliinstall/cliinstall.go](/home/matthalloran8/Vrooli/internal/cliinstall/cliinstall.go:1).

Goals:

- scenario install path passes `.vrooli/service.json`
- resource install path passes `resource.json`
- shared manager behavior remains structurally identical
- no scenario/resource branching beyond selecting the correct manifest source path

This phase should be implemented so that scenarios and resources differ only in:

- root path
- manifest source path
- binary name

Everything else should remain shared.

### Phase 4: Update Runtime Lookup For Installed CLIs

Update scenario CLI and resource CLI runtime helpers so installed CLIs look for sibling manifest metadata first.

Rules:

- sibling manifest beside the executable is authoritative for installed binaries
- source-tree manifest resolution may still exist for explicit development mode
- embedded manifest fallback is removed

This phase should produce deterministic runtime behavior that does not depend on whether the repo is available.

### Phase 5: Remove SQLite Shim Architecture

Use SQLite as the first proof that the new packaging model works.

Required cleanup:

- remove root `runtime_embed.go`
- remove root shim `go.mod`
- remove any remaining root-level packaging special cases
- keep only the clean v3 layout rooted at `cli/`

This is the critical proof point because SQLite currently exists in the exact shape this plan is meant to eliminate.

### Phase 6: Update Templates

Update both scenario and resource templates so newly generated Go CLIs follow the same packaging and artifact model.

Scenario template work:

- scenario `service.json` `cli` contract reflects installed companion manifest semantics
- install scripts pass manifest source path to `cli-core`
- docs describe CLI as a packaged installed artifact

Resource template work:

- `resource.json` `cli` contract reflects the same semantics
- install scripts pass manifest source path to `cli-core`
- docs describe the same installed-artifact model

Target result:

- scenarios and resources teach the same CLI packaging story
- only the outer manifest and surrounding lifecycle differ

### Phase 7: Migrate The Already-Touched Resources

After the installer and contract are correct, migrate the resources already standardized around Go-native CLI layouts:

- `sqlite`
- `cloudflare-ai-gateway`
- `k6`
- `opencode`

These should become the first live resource proof set for the new installed-artifact model.

### Phase 8: Repo-Wide Adoption Guidance

Update the relevant docs and guidance so future work does not reintroduce split models.

Required outcomes:

- resource templates describe the final standard, not intermediate variants
- scenario templates do the same
- `cli-core` docs explain installed artifact packaging clearly
- future agents can infer the right pattern from local docs without re-discovering it

## Validation Plan

### Unit Tests: `cli-core`

Add tests that verify:

- binary installs successfully
- sibling manifest installs successfully
- build metadata installs successfully
- install overwrite behavior is deterministic
- Windows and Unix path handling behave correctly

### Unit Tests: Shared Install Manager

Add tests in [internal/cliinstall/cliinstall.go](/home/matthalloran8/Vrooli/internal/cliinstall/cliinstall.go:1) that verify:

- scenario install path passes scenario manifest correctly
- resource install path passes resource manifest correctly
- the shared manager computes installed artifact paths identically
- freshness/install behavior remains aligned

### Template Tests

Strengthen template tests so they assert:

- scenario templates declare the correct CLI artifact model
- resource templates declare the same CLI artifact model
- install wrappers pass manifest inputs to the shared installer
- generated modules validate and build

### End-To-End Tests

For at least one scenario and one resource:

- install the CLI to a temp install directory
- remove repo-root assumptions from runtime
- execute the installed binary outside the source tree
- verify the installed CLI loads sibling manifest metadata successfully

Suggested proof cases:

- one scenario Go CLI from the standard template
- `sqlite` for resources

### Migration Verification

For SQLite and other migrated resources:

- no embedded manifest fallback remains
- no root shim Go module remains
- no root install wrappers remain
- installed CLI still works from outside the repo

## Implementation Order

Recommended execution order:

1. define canonical installed artifact naming and runtime lookup rules
2. extend shared CLI contract shape
3. update `cli-core` installer to install sibling manifest metadata
4. add `cli-core` installer tests
5. update shared install manager to pass manifest paths uniformly
6. update runtime lookup for installed CLIs
7. migrate SQLite fully
8. update scenario and resource templates
9. migrate already-touched Go-native resources
10. add end-to-end validation coverage

This order keeps the work clean:

- shared substrate first
- one real proof point second
- broad rollout third

## Code Quality Bar

This work should optimize for:

- deterministic behavior
- explicit contracts
- minimal duplication
- no scenario/resource special casing beyond manifest source selection
- professional code layout and naming
- tests that validate behavior, not implementation accidents

Specifically:

- do not add compatibility branches
- do not add "temporary" embedded fallback logic
- do not add installer logic that guesses paths from repo layout when manifest input is already available
- do not fork separate installer code paths for scenarios and resources

## Success Criteria

This plan is complete when:

1. scenario and resource Go CLIs use the same installed-artifact model
2. shared installer code installs both binary and manifest companion metadata
3. installed CLIs run correctly outside the source tree
4. SQLite no longer needs root packaging shims
5. templates teach the final standard, not the transitional one
6. the repo has one clear professional answer to "how does a Go CLI get installed and run in Vrooli?"

## Practical Summary

The repo already shares much of the right substrate.

The missing piece is to stop treating installed CLIs as bare binaries and start treating them as complete installed artifacts with companion manifest metadata.

Once that is fixed:

- scenarios and resources can truly converge on one CLI packaging model
- root shim modules go away
- embedded manifest hacks go away
- templates become simpler and more honest
- future Go-native CLIs can be implemented cleanly from the start
