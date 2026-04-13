# Host Requirements Declaration And Runtime Redesign Plan

**Status:** In Progress (`Phases 0-3` complete, `Phase 5` implemented, later cleanup/enforcement pending)

**Last Updated:** 2026-04-12

## Why This Plan Exists

Vrooli's setup story is currently in an awkward transitional state:

- Top-level `vrooli setup` is now native Go via `internal/setup.RunSetup`.
- The native runtime layer only knows about a small hardcoded tool list (`docker`, `go`, `node`, `python`, `helm`).
- Old shell setup logic under `scripts/lib/setup.sh` used to install a much wider set of tools, helpers, and machine safeguards.
- Ownership of tool requirements is mostly implicit and scattered across shell scripts, resource scripts, scenario package scripts, and historical setup behavior.
- Some old shell setup surfaces are already stale, partially deleted, or no longer on the main execution path.
- There is no single declarative source of truth for:
  - which host tools are required,
  - which scenarios/resources require them,
  - which machine safeguards should be enabled,
  - which setup actions are skipped and why.

This creates multiple problems:

- setup installs too little in some cases and too much in others;
- specialized tools look "core" only because old shell setup used to install them globally;
- resource/scenario owners cannot cleanly declare their actual host requirements;
- minimal or subset installs are harder than they should be;
- setup logging is not explicit enough about declared vs. unsupported vs. skipped;
- shell compatibility helpers (`scripts/lib/setup.sh`, `setup-conditions`, `ui-guard.sh`, `cli-install.sh`, old deps installers, etc.) remain mixed into the architecture;
- cleanup decisions are risky because requirements are not modeled explicitly.

## Goal

Replace the current hardcoded and partially shell-derived setup/runtime model with a native, manifest-driven host requirements system that:

- allows the root project, scenarios, and resources to declare host tools and host safeguards explicitly;
- keeps the core/default setup surface intentionally small;
- supports subset installs cleanly;
- makes promotion from declared to core a declaration-only change;
- handles OS/platform differences explicitly and predictably;
- produces clear setup plans and outcomes;
- removes legacy shell setup/tooling approaches once parity is proven.

## Non-Goals

- Re-creating `scripts/lib/setup.sh` behavior in Go one-to-one.
- Preserving historical `deps/`, `runtimes/`, `system/`, or `network/` directory taxonomy as a first-class architecture.
- Keeping old shell compatibility surfaces indefinitely.
- Solving every resource migration problem in this plan; resource runtime/control migration remains its own track. This plan owns host requirement declaration, resolution, setup execution, and cleanup of old host setup approaches.

## Desired End State

When this plan is complete:

- host requirements are declared in manifests, not inferred from stale shell setup;
- `vrooli setup` resolves requirements from root + selected resources + selected scenarios;
- setup prints a clear plan and result summary for install/apply/skip decisions;
- tools and safeguards are implemented as native registry entries with shared helpers;
- OS support and non-applicability are explicit;
- root "core" requirements are small and easy to change;
- resource/scenario-specific requirements are owned by those manifests;
- shell-era host setup code and stale setup references are deleted;
- tests validate both resolution correctness and visible setup behavior;
- the codebase is materially cleaner than today.

## Design Principles

1. Declarations over hidden behavior.
2. Ownership at the narrowest responsible layer.
3. Small core, explicit extensions.
4. One native resolution path.
5. Clear install/apply/skip reasoning.
6. Cross-platform honesty over fake portability.
7. Deletion is part of the plan, not an afterthought.

## Current State Summary

### Current Native Setup Path

Today, top-level setup is native:

- `make setup` invokes the installed `vrooli` binary.
- `cmd/vrooli` dispatches top-level `setup` to `internal/setup.RunSetup`.
- `internal/setup.RunSetup`:
  - validates host support,
  - creates project/home state directories,
  - marks shell scripts executable,
  - resolves manifest-owned host requirements,
  - calls `internal/runtime.EnsureRequirements(...)`,
  - configures git,
  - optionally installs resources via `internal/resources.Controller`.

### Current Native Runtime Limitations

`internal/runtime` currently:

- consumes manifest-owned host requirement resolutions from `internal/hostreq`;
- dispatches through a native registry for tools and safeguards;
- supports explicit support classification (`supported`, `unsupported`, `not_applicable`, `manual_only`);
- still needs broader setup UX/reporting work and later-phase migration/cleanup.

### Current Shell-era Setup Debt

Relevant shell-era setup surfaces still present or still referenced:

- `scripts/lib/setup.sh`
- `scripts/lib/setup-conditions/*`
- `scripts/lib/utils/setup.sh`
- `scripts/lib/ui-guard.sh`
- `scripts/lib/utils/cli-install.sh`
- various old `scripts/lib/deps/*` installers/helpers
- historical safeguard scripts such as `remote_session_protect.sh`

Relevant stale or transitional references still in-tree:

- root `.vrooli/service.json` still contains a shell `base-setup` step pointing at `scripts/lib/setup.sh`;
- `internal/lifecycle` still shells out to `scripts/lib/setup-conditions/<type>-check.sh` for unknown setup condition types;
- many scenario package scripts still reference `scripts/lib/ui-guard.sh`;
- many scenario CLI install scripts still source `scripts/lib/utils/cli-install.sh`;
- tests and fixtures still assert against old shell setup references.

## Proposed Architecture

### 1. New Requirement Types

Introduce two explicit declaration types:

- `hostTools`
- `hostSafeguards`

These are intentionally separate:

- a host tool is an installable executable or host dependency such as `docker`, `node`, `stripe`, `ffmpeg`, `buf`;
- a host safeguard is a host-level protection or policy application such as `remote_session_protection` or future GPU/container/runtime guardrails.

### 2. Declaration Locations

Allow declarations in:

- root project manifest: `.vrooli/service.json`
- scenario manifests: `scenarios/*/.vrooli/service.json`
- resource manifests: `resources/*/resource.json`

The same conceptual model should be used everywhere, with ownership determined by manifest location.

### 3. Native Resolution Layer

Add a dedicated native resolver package, e.g.:

- `internal/hostreq`

Responsibilities:

- load declarations from root/resource/scenario manifests;
- support setup selectors such as:
  - core only,
  - enabled resources,
  - explicit resource list,
  - explicit scenario list,
  - future profiles;
- merge and deduplicate requirements;
- track provenance for each requirement;
- classify each requirement for the current host.

### 4. Native Runtime Execution Layer

Evolve `internal/runtime` into a registry-driven execution layer.

Recommended package layout:

- `internal/runtime/`
- `internal/runtime/tools/`
- `internal/runtime/safeguards/`

Each tool/safeguard gets its own implementation file.

Examples:

- `internal/runtime/tools/docker.go`
- `internal/runtime/tools/stripe.go`
- `internal/runtime/safeguards/remote_session_protection.go`

### 5. Shared Runtime Helpers

Provide shared helpers so per-tool/per-safeguard implementations stay thin:

- host and package-manager detection
- package-name mapping
- command existence/version probing
- install/apply helpers
- sudo handling
- dry-run execution
- structured logging/reporting
- OS support checks
- systemd/sysctl/gui presence checks

### 6. Native Setup Planning And Logging

`vrooli setup` should first resolve a plan, then execute it.

The user-visible setup output must clearly distinguish:

- `install` / `apply`
- `already_present`
- `skip_not_declared`
- `skip_unsupported`
- `skip_not_applicable`
- `manual_action_required`

Each result should show the reason and provenance.

### 7. Cleanup Philosophy

The end-state architecture should not preserve old shell setup helpers as hidden dependencies.

Once parity is proven, remove:

- shell-based tool installers and helpers whose responsibility has moved to native runtime;
- shell-based safeguard setup whose responsibility has moved to native safeguards;
- stale shell setup references in root/scenario/resource manifests;
- shell-era setup condition infrastructure where native equivalents exist;
- scenario/resource helper surfaces that can be replaced by native or manifest-native flows.

## Proposed Declaration Shape

This plan does not require the exact final JSON shape to be locked immediately, but it should converge on something close to:

```json
{
  "hostTools": [
    {
      "name": "docker",
      "required": true,
      "reason": "local containerized resources",
      "when": ["setup", "develop"]
    }
  ],
  "hostSafeguards": [
    {
      "name": "remote_session_protection",
      "required": true,
      "reason": "protect GUI VPS sessions under memory pressure"
    }
  ]
}
```

Minimum useful attributes:

- `name`
- `required`
- `reason`

Likely useful attributes:

- `when`
- `notes`
- `manual`
- `platforms`

Provenance should be computed by the resolver, not declared redundantly.

## Ownership Rules

- Root project manifest declares only truly core/default requirements.
- Scenario manifests declare scenario-owned requirements.
- Resource manifests declare resource-owned requirements.
- Resource/scenario declarations should not be duplicated in root just to "make setup work."
- Promotion from declared to core should be a manifest move, not a code-path rewrite.

## Initial Classification Guidance

Likely `core` or root-default candidates:

- `docker`
- `git`
- `curl`
- `jq`
- likely `node`
- likely `go`

Likely declared tool candidates:

- `python`
- `helm`
- `tmux`
- `yq`
- `stripe`
- `vault`
- `buf`
- `sqlite`
- `ffmpeg`
- `shellcheck`
- `bats`
- `lychee`
- `ast-grep`
- `js-yaml`
- `ajv`
- `Xvfb`
- `xdotool`
- `x11vnc`
- `websockify`
- `openbox`

Likely safeguard candidates:

- `remote_session_protection`
- future GPU/container host guardrails

These classifications must be validated by actual usage audit during implementation, not frozen by this document.

## Initial Ownership Matrix

This is the first-pass ownership map to make phase 0 execution-ready. It is intentionally opinionated, but every entry must still be validated during the audit before the classification is treated as final.

| Name | Type | Initial Classification | Likely Owner | Why It Exists | Notes |
| --- | --- | --- | --- | --- | --- |
| `docker` | tool | core | root project | local containerized resources and resource runtime | keep in the first native wave |
| `git` | tool | core | root project | repo operations and root bootstrap hygiene | likely probe/configure, not heavy installer ownership |
| `curl` | tool | core | root project | shell/resource HTTP flows and diagnostics | still widely assumed |
| `jq` | tool | core | root project | shell/resource JSON parsing | still widely assumed |
| `node` | tool | core candidate | root project | many scenario UIs and JS-based tooling | verify with audit before freezing |
| `go` | tool | core candidate | root project | Go-native project/scenario tooling | verify with audit before freezing |
| `python` | tool | declared candidate | scenario/resource-specific | Python-based scenarios and helper flows | may stay core if audit shows broad requirement |
| `helm` | tool | declared | scenario/resource-specific | Kubernetes packaging/deploy flows | not core by default |
| `tmux` | tool | declared candidate | root or scenario-specific | detachable sessions and operator workflows | audit current real usage |
| `yq` | tool | declared candidate | resource/scenario-specific | YAML inspection/manipulation | audit whether root setup still needs it |
| `stripe` | tool | declared | payment-related scenarios | local webhook/payment tooling | explicit mention: landing-manager and payment scenarios |
| `vault` | tool | delete | none | host Vault CLI is retired; use `resource-vault` instead | do not reintroduce host `vault` declarations |
| `buf` | tool | declared | protobuf/codegen owners | schema/codegen workflows | not a universal prerequisite |
| `sqlite` | tool | declared | sqlite-using resources/scenarios | sqlite DB operations | not global by default |
| `ffmpeg` | tool | declared | media/screen-recording owners | capture/transcoding workflows | heavy, specialized |
| `shellcheck` | tool | declared | test/dev profile | shell linting | likely profile-driven |
| `bats` | tool | declared | test/dev profile | shell test execution | likely profile-driven |
| `lychee` | tool | declared | docs/test profile | link validation | likely profile-driven |
| `ast-grep` | tool | declared | dev/tooling profile | structural search/refactor | likely profile-driven |
| `js-yaml` | tool | declared | YAML-manipulating JS flows | node-based YAML CLI | likely replace with direct package usage where possible |
| `ajv` | tool | declared | schema validation owners | JSON schema validation CLI | likely profile-driven or scenario-specific |
| `Xvfb` | tool | declared | browser/headless UI owners | virtual display for headless UI tests | Linux-specific |
| `xdotool` | tool | declared | desktop/browser automation owners | X11 UI automation | Linux/X11-specific |
| `x11vnc` | tool | declared | remote desktop/browserless owners | VNC session exposure | specialized host feature |
| `websockify` | tool | declared | remote desktop/browserless owners | VNC to WebSocket bridge | specialized host feature |
| `openbox` | tool | declared | virtual desktop owners | lightweight window manager for virtual desktop flows | specialized host feature |
| `remote_session_protection` | safeguard | declared safeguard candidate | deployment profile, scenario-to-cloud, or desktop-VPS profile | protect GUI/remote Linux hosts from memory/session failure | should be modeled as safeguard, not tool |
| `gpu_container_support` | safeguard | future declared safeguard | GPU-capable resource/deployment owners | container GPU runtime guardrails/setup | do not add until a real owner and validation exist |

## Initial Adoption Waves

To avoid trying to port every historical setup behavior at once, implement in these waves:

### Wave 1: Foundation And Existing Native Core

- `docker`
- `git`
- `curl`
- `jq`
- `node`
- `go`
- current host/package-manager detection

### Wave 2: First Real Declared Tools

- `python`
- `helm`
- `tmux`
- `yq`
- `stripe`
- `buf`
- `sqlite`

### Wave 3: Dev/Test And Specialized Desktop Tooling

- `shellcheck`
- `bats`
- `lychee`
- `ast-grep`
- `js-yaml`
- `ajv`
- `ffmpeg`
- `Xvfb`
- `xdotool`
- `x11vnc`
- `websockify`
- `openbox`

### Wave 4: Safeguards

- `remote_session_protection`
- future host guardrails with explicit owners

## Ownership Audit Deliverables

Phase 0 should produce a concrete audit artifact that adds, at minimum, these columns for every candidate:

- `name`
- `type`
- `current path or assumption`
- `current live consumers`
- `proposed owner`
- `proposed classification`
- `native implementation needed`
- `delete/replace/defer decision`

The implementation work should not begin migrating large batches of tools until that audit artifact exists.

## Phase Plan

## Phase 0: Ground Truth And Contract Design

### Goals

- inventory current host tools/safeguards and live consumers;
- define final architecture, schema, and execution boundaries;
- prevent implementation drift.

### Tasks

- [x] Audit current host tool and safeguard usage across:
  - root setup,
  - scenario manifests,
  - resource manifests,
  - shell scripts,
  - package scripts,
  - tests and CI targets.
- [x] Build a source-of-truth map:
  - tool/safeguard name,
  - current implicit owner,
  - current install path,
  - intended future owner,
  - future classification (`core`, declared tool, declared safeguard, delete).
- [x] Finalize manifest schema additions for:
  - root service schema,
  - scenario service schema,
  - resource schema.
- [x] Finalize CLI setup selection semantics:
  - `--resources enabled|none|list`
  - scenario selection support
  - future profile support
- [x] Finalize reporting/result model for setup planning and execution.
- [x] Decide exact package naming and directory structure for:
  - resolver,
  - tool registry,
  - safeguard registry,
  - shared helpers.

### Phase 0 Outputs

- Approved audit artifact:
  [host-requirements-phase0-audit.md](/home/matthalloran8/Vrooli/docs/plans/host-requirements-phase0-audit.md)
- Approved declaration contract:
  top-level `hostTools` and `hostSafeguards` arrays on root/scenario/resource manifests
- Approved selector contract:
  keep `--resources enabled|none|<csv>`, add `--scenarios none|all|<csv>`, default scenarios to `none`
- Approved ownership rule:
  keep root core intentionally small; scenario/resource-specific tools must not be promoted to root by convenience
- Approved early decisions:
  `python` remains root-owned only for `development`, `tmux` and `yq` are not core, `remote_session_protection` is a host-profile safeguard, and host `vault` is retired in favor of `resource-vault`

### Acceptance

- [x] Schema shape approved.
- [x] Runtime package layout approved.
- [x] Audit table exists in the plan or an adjacent reference artifact.
- [x] No unresolved ambiguity about root vs scenario vs resource ownership rules.

## Phase 1: Schema And Resolver Foundation

### Goals

- add declarations to manifests and schemas;
- build a native resolver that merges requirements and records provenance.

### Tasks

- [x] Extend `.vrooli/schemas/service.schema.json` with `hostTools` and `hostSafeguards`.
- [x] Extend `.vrooli/schemas/resource.schema.json` with `hostTools` and `hostSafeguards`.
- [x] Extend native manifest structs/parsers:
  - `internal/scenario`
  - `internal/resources`
  - root project manifest loading if separate handling is needed
- [x] Add a new native resolver package:
  - load root/scenario/resource declarations,
  - merge/dedupe,
  - validate declaration shape,
  - attach provenance.
- [x] Add unit tests for:
  - schema acceptance,
  - manifest parsing,
  - merge behavior,
  - duplicate handling,
  - provenance reporting,
  - selector behavior.

### Acceptance

- [x] Root, scenario, and resource manifests can declare requirements.
- [x] Resolver returns deterministic merged requirements.
- [x] Resolver tests are green.

## Phase 2: Native Runtime Registry Redesign

### Goals

- replace the hardcoded runtime tool list with a registry-driven model;
- add first-class safeguard support.

### Tasks

- [x] Refactor `internal/runtime` to separate:
  - host detection,
  - resolution input,
  - tool inspection,
  - tool install,
  - safeguard applicability,
  - safeguard application.
- [x] Add shared runtime helper layer for:
  - command probing,
  - package-manager abstraction,
  - package name mapping,
  - install command execution,
  - sudo policies,
  - dry-run support,
  - structured result capture.
- [x] Create registry interfaces for tools and safeguards.
- [x] Implement one file per tool/safeguard.
- [x] Preserve current `docker`/`go`/`node`/`python`/`helm` support through the new registry.
- [x] Add native implementations for the first new declared targets chosen from the audit.
- [x] Implement support classification:
  - supported,
  - unsupported,
  - not applicable,
  - manual only.
- [x] Add unit tests for:
  - host detection,
  - package-manager mapping,
  - tool/safeguard support checks,
  - dry-run behavior,
  - install/apply result classification.

### Acceptance

- [x] `internal/runtime` no longer depends on a hardcoded `toolSpecs()` list as the core architecture.
- [x] Tools and safeguards are registry-driven.
- [x] Per-tool/per-safeguard implementations are small and focused.

## Phase 3: Setup Planning, Logging, And UX

### Goals

- make `vrooli setup` resolve and execute a visible plan;
- make skips explicit and understandable.

### Tasks

- [x] Update `internal/setup` to:
  - request declarations from the resolver,
  - build a full setup plan,
  - execute tools and safeguards through the runtime registry.
- [x] Add a visible setup summary before execution.
- [x] Add clear per-item reporting during and after execution.
- [x] Distinguish:
  - not declared,
  - already present,
  - unsupported on this OS,
  - not applicable on this host,
  - manual install required,
  - successfully installed/applied,
  - failed.
- [x] Ensure `--dry-run` prints an accurate plan without mutating the host.
- [x] Ensure errors remain actionable and concise.
- [x] Add tests for:
  - plan rendering,
  - dry-run output,
  - unsupported/not-applicable reporting,
  - setup with root-only declarations,
  - setup with scenario/resource-selected declarations.

### Acceptance

- [x] `vrooli setup` output explains what happened and why.
- [x] Dry-run is trustworthy.
- [x] Setup logging is materially better than current behavior.

## Phase 4: Migrate Live Owners To Declarations

### Goals

- move live requirements out of implicit shell behavior into explicit ownership declarations.

### Tasks

- [ ] Add root declarations for the final approved core/default tool set.
- [ ] Update resources to declare their host tools/safeguards.
- [ ] Update scenarios to declare their host tools/safeguards.
- [ ] Replace historical implicit assumptions with explicit declarations.
- [ ] Introduce a temporary migration report or validator to detect:
  - obvious shell references to undeclared tools,
  - declared tools with no registry implementation,
  - root overreach where specialized tools were incorrectly made core.
- [ ] Add fixture-based integration tests covering:
  - root-only requirements,
  - resource-owned requirements,
  - scenario-owned requirements,
  - merged/deduped requirements,
  - unsupported platform behavior.

### Acceptance

- [ ] Major live tool/safeguard consumers are explicitly declared.
- [ ] Setup behavior can be explained from manifests alone.

## Phase 5: Restore Or Re-Implement Missing Safeguards And Specialized Tools

### Goals

- bring back genuinely useful capabilities in the new model, without reviving shell-era architecture.

### Tasks

- [x] Decide which deleted historical capabilities should return as:
  - native tool implementations,
  - native safeguard implementations,
  - declarations only,
  - permanent deletion.
- [x] Re-implement `remote_session_protection` natively if audit confirms continued value.
- [x] Add native tool entries for high-value declared tools such as:
  - `stripe`
  - other audited live needs
- [x] Explicitly reject or defer low-value historical setup features with rationale, including the retired host `vault` CLI path.
- [x] Add platform/applicability tests for safeguards.

### Acceptance

- [x] Important missing capabilities are restored in the new architecture.
- [x] No deleted shell-era feature is reintroduced casually without explicit ownership and tests.

## Phase 6: Remove Old Host Setup Approaches

### Goals

- delete stale and superseded setup/tooling surfaces.

### Tasks

- [ ] Remove stale root `base-setup` reference to `scripts/lib/setup.sh`.
- [ ] Delete `scripts/lib/setup.sh` once no live path depends on it.
- [ ] Delete obsolete `scripts/lib/deps/*` tool installers/helpers whose responsibilities moved native or were intentionally retired.
- [ ] Delete obsolete safeguard shell scripts that were replaced natively or intentionally retired.
- [ ] Replace `scripts/lib/setup-conditions/*` external setup checkers with native equivalents or explicit deprecation.
- [ ] Remove or replace stale scenario/resource shell helpers that this redesign makes obsolete.
- [ ] Remove tests, fixtures, and docs that anchor old shell setup behavior.
- [ ] Update validation targets to fail on reintroduction of deleted setup surfaces.

### Acceptance

- [ ] No production setup path depends on `scripts/lib/setup.sh`.
- [ ] No active host requirement depends on shell-era installer helpers.
- [ ] Setup conditions no longer depend on shell check scripts for native-owned behavior.

## Phase 7: Scenario And Resource Helper Cleanup

### Goals

- clean up adjacent shell helper surfaces that should not survive the redesign unchanged.

### Tasks

- [ ] Replace or redesign `scripts/lib/ui-guard.sh`.
- [ ] Replace or redesign `scripts/lib/utils/cli-install.sh`.
- [ ] Remove or modernize scenario package/CLI references to those helpers.
- [ ] Decide whether these become:
  - native CLI subcommands,
  - manifest-driven setup/develop behavior,
  - scenario-local scripts,
  - or are deleted entirely.
- [ ] Update scaffolding/templates so new scenarios/resources do not reintroduce old helper patterns.

### Acceptance

- [ ] New scenarios/resources do not depend on old shared shell helper patterns.
- [ ] Existing consumers are migrated or intentionally quarantined behind explicit compatibility boundaries.

## Phase 8: Validation, Docs, And Enforcement

### Goals

- make the new architecture durable and hard to regress.

### Tasks

- [ ] Add unit tests for resolver, runtime registry, setup planner, and manifest parsing.
- [ ] Add fixture-based integration tests for setup selection and logging.
- [ ] Add platform-oriented tests for unsupported/not-applicable behavior.
- [ ] Add validation targets that check:
  - no stale shell setup references,
  - no undeclared registry-owned requirements in manifests,
  - no registry entry without tests,
  - no reintroduction of deleted setup scripts.
- [ ] Write architecture docs for:
  - how to declare a host tool,
  - how to declare a host safeguard,
  - how to add a new registry implementation,
  - how setup selection works,
  - how OS support and applicability are modeled.
- [ ] Update contributor guidance and templates.

### Acceptance

- [ ] The new host requirements system is documented and enforceable.
- [ ] CI or validation targets protect against regression.

## Validation Matrix

The implementation must validate all of the following:

- [ ] root-only core setup works on supported Linux hosts
- [ ] setup with `--resources none` installs only root/core requirements
- [ ] setup with selected resources installs merged tool/safeguard requirements
- [ ] setup with selected scenarios installs merged tool/safeguard requirements
- [ ] duplicate declarations dedupe cleanly
- [ ] dry-run prints the plan without mutating host state
- [ ] unsupported OS items are reported explicitly
- [ ] not-applicable safeguards are reported explicitly
- [ ] already-installed tools are reported explicitly
- [ ] manual-only items are surfaced with actionable guidance
- [ ] removed shell setup paths are no longer executed
- [ ] setup remains green for current project-level native flow

## Code Quality Requirements

The implementation must be:

- professional and easy to navigate;
- documented at the package and type level where behavior is non-obvious;
- explicit about OS/platform behavior;
- tested with unit and fixture-based integration coverage;
- designed so adding or promoting a tool/safeguard is primarily a declaration change plus a focused registry implementation;
- free of hidden shell fallbacks for native-owned responsibilities.

## Cleanup Targets

This plan is not complete until the following are either deleted or intentionally replaced:

- [ ] stale shell `base-setup` path in root manifest
- [ ] `scripts/lib/setup.sh`
- [ ] obsolete `scripts/lib/deps/*`
- [ ] obsolete shell safeguard scripts
- [ ] shell `setup-conditions` infrastructure for native-owned behavior
- [ ] stale tests/docs asserting old setup behavior
- [ ] old helper patterns that reintroduce hidden host setup assumptions

## Phase 0 Decisions Resolved Early

- [x] exact JSON shape for declarations
  `hostTools` and `hostSafeguards` are approved as top-level arrays with required `name`, `required`, and `reason` fields plus bounded optional metadata
- [x] whether root core declarations live directly in `.vrooli/service.json` or in a nested setup-specific section
  root declarations live directly in `.vrooli/service.json`
- [x] whether setup profile declarations are needed in phase 1 or can land later
  profile selectors are deferred until after the base resolver exists
- [x] exact CLI selector surface for scenario selection
  use `--scenarios none|all|<csv>`, defaulting to `none`
- [x] whether `python` remains core by default
  root-owned for `development`; not a universal requirement for `production` or `minimal`
- [x] whether `tmux` and `yq` are core or declared after audit
  both are explicit declared tools, not root core
- [x] whether `remote_session_protection` should be host-profile-driven, scenario-declared, or both
  host-profile-driven first; not scenario-owned in the initial model

## Recommended Execution Order

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. Phase 5
7. Phase 6
8. Phase 7
9. Phase 8

## Definition Of Done

This plan is done only when:

- declarations drive setup behavior;
- the runtime layer is registry-driven and supports safeguards;
- setup output is explicit and trustworthy;
- major live tools/safeguards are owned declaratively;
- old shell setup architecture is removed;
- templates/docs/tests enforce the new model;
- the resulting code is clearly cleaner than the current mixed native/shell setup state.
