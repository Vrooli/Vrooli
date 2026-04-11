# Resource Cross-Platform Migration Plan

**Status:** Proposed
**Owner:** Matthew Halloran
**Scope:** Project-level resource system under `resources/`, `scripts/resources/`, `scripts/lib/resources/`, and the corresponding `vrooli resource` control surface
**Out of Scope:** Scenario internals under `scenarios/*/`, except where scenario template/resource patterns provide precedent
**Primary Goal:** Replace the current shell-first, speculative, high-maintenance resource system with a cross-platform, Go-native control plane built around a small validated resource set, first-class resource blueprints, and managed deprecation/archive workflows

---

## 0. For agents picking this up later

If you are an agent resuming this work in a future session, read this section first.

- **What this plan covers:** The full migration strategy for resources, including architecture, blueprints, deprecation, templates, shared code, phased execution, and cleanup.
- **What this plan does not cover:** Scenario implementation internals. Scenario-level cross-platform work is already on its own track.
- **What problem this is solving:** The current resource system contains many speculative or stale implementations. Migrating all of them to Go would waste effort and preserve low-quality abstractions. We need to separate capability knowledge from supported integrations.
- **What success looks like:** Vrooli ends up with:
  - a small set of actively maintained, cross-platform-capable implemented resources
  - a first-class `resource blueprint` catalog for future capabilities
  - a built-in `deprecate/archive/restore` lifecycle
  - a Go-native resource control plane with platform-aware drivers and templates
- **How to resume work:** Find the first unchecked item in the phase checklist, re-read that phase, and execute only that bounded slice. This plan is intended to be completed piecemeal across multiple conversations.
- **Important bias:** If a resource is speculative, stale, or unused, prefer converting it to a blueprint rather than migrating its code.

---

## 1. Why this matters

### The current problem

Resources are one of the last major parts of Vrooli that do not yet have a credible cross-platform strategy.

At the time of writing:

- project-level orchestration is being migrated from Bash to Go
- scenarios already have a clearer cross-platform path
- resources are still implemented primarily as Bash CLIs and shared Bash libraries

This is a real architectural problem, not just a language problem.

### What is wrong with the current resource system

1. **The control plane is shell-first.**
   - Resource discovery, installation, startup, status, and CLI registration still rely on shell entrypoints and shell frameworks.
   - The new Go resource controller currently shells back into resource CLIs rather than replacing their behavior.

2. **Many resources were created speculatively.**
   - The repo contains a large number of resource implementations that were never validated, never used in real scenarios, or were created under outdated assumptions.
   - Treating all of them as first-class migration targets would create a lot of low-value work.

3. **The current contract couples capability knowledge to executable code.**
   - Today, the only durable way to remember a potentially useful resource is to keep a real implementation in the repo.
   - That creates maintenance load for ideas that should instead be stored as structured knowledge.

4. **Portability assumptions are inconsistent.**
   - Some resources are mostly Docker-backed and portable enough.
   - Some are wrappers around cross-platform CLIs or cloud APIs.
   - Some are Linux-specific or hardware-specific and will never be fully cross-platform.
   - The current system does not classify resources honestly by portability tier.

5. **Templates and shared code are underdeveloped.**
   - Scenario generation already has a clear template model.
   - Resource creation has only a PRD template, not a full implementation template system or driver/archetype strategy.

### Why this is important to Vrooli specifically

Vrooli's resource layer is not just "integrations." It is part of the capability substrate that scenarios and future agents build on.

If the resource system is noisy, stale, and Linux-coupled:

- agents will discover bad integrations and stale patterns
- migration work will be misprioritized toward dead code
- future resource expansion will be slower and more error-prone
- the platform will keep confusing "possible capability" with "supported capability"

The fix is to explicitly model those as separate things:

- **Implemented resources:** supported, validated, executable now
- **Resource blueprints:** structured capability knowledge for future implementation
- **Deprecated resources:** removed from the active surface, recoverable for a time, then purgeable

---

## 2. Ground truth: how resources work now

This section documents the current architecture so later phases do not work from fuzzy memory.

### Current configuration sources

- Project-level enablement lives in `.vrooli/service.json` under `dependencies.resources`
- Resource registry metadata is stored in `.vrooli/resource-registry/*.json`
- Running/installed status tracking currently uses `.vrooli/running-resources.json`
- Each resource usually has:
  - `resources/<name>/cli.sh`
  - `resources/<name>/config/defaults.sh`
  - `resources/<name>/config/runtime.json`
  - `resources/<name>/lib/*.sh`
  - `resources/<name>/test/*`

### Current control surface

The current model is:

1. project-level orchestration determines which resources are enabled
2. shell orchestration invokes `resource-<name>` or `resources/<name>/cli.sh`
3. each resource implements the v2 CLI contract through Bash handlers
4. resource-specific libraries perform the real work

### Current v2 resource contract

All resources are expected to implement:

- `help`
- `info`
- `status`
- `logs`
- `manage install|start|stop|restart|uninstall`
- `test smoke|integration|unit|all`
- `content add|list|get|remove|execute`

This is a good surface contract for users, but its current implementation is shell-native.

### Current portability constraints

There are several categories of current resource behavior:

1. **Docker-backed services**
   - Examples: `postgres`, `redis`, `qdrant`, `browserless`, `vault`
   - These are the best candidates for cross-platform management through Go drivers.

2. **External CLI wrappers**
   - Examples: `claude-code`, `codex`, `gemini`, `ffmpeg`
   - These should not necessarily have custom per-resource code forever; many fit a generic external-tool driver.

3. **Cloud/API wrappers**
   - Examples: `openrouter`, `twilio`, `pushover`
   - These mostly need config, credentials, health checks, and thin action layers.

4. **Native app / host package / GUI / hardware resources**
   - Examples: KiCad-class, OBS-class, hardware-oriented tools
   - Many of these will never be fully equal across Linux/macOS/Windows.

5. **Speculative or stale resources**
   - Created but not validated, not used, or superseded by better approaches
   - These should mostly become blueprints rather than migration targets

### Current template state

Scenarios already have a strong template model:

- `scripts/scenarios/templates/`
- template metadata files such as `template.json`
- standard generation flow

Resources currently do not. There is only:

- `scripts/resources/templates/PRD.md`

That is not sufficient for a maintainable future resource system.

---

## 3. Principles and decisions

These are the strategic decisions this plan assumes.

### Decision 1: Blueprints are first-class

We will introduce **resource blueprints** as the canonical way to preserve resource capability knowledge without keeping stale code in the repo.

Blueprints are not brainstorm scraps. They are structured, reusable implementation seeds.

### Decision 2: Only a small validated resource set remains implemented

The active `resources/` tree should contain only:

- resources used by current project-level workflows
- resources used by active scenarios
- resources with clear strategic value and recent validation

Everything else should either:

- become a blueprint, or
- be deprecated and archived

### Decision 3: Deprecation becomes a built-in lifecycle

Deprecation should not be an informal manual delete.

Vrooli should eventually support:

- archiving deprecated resources outside the repo
- listing deprecated resources
- restoring archived resources during a retention window
- garbage-collecting expired archives

### Decision 4: The resource control plane becomes Go-native

The `vrooli resource` command family should move to Go-native orchestration.

That means:

- typed resource manifests
- typed driver interfaces
- platform-gated implementations where needed
- no shell dependency for core discovery/install/start/stop/status logic

Resource-specific shell logic may temporarily remain behind adapters during migration, but not as the target architecture.

### Decision 5: Templates should mirror the scenario template philosophy

Scenarios already standardize generation around a small number of high-quality templates.

Resources should follow the same rule:

- a small set of canonical resource templates
- archetype-focused, not product-name-focused
- generated from metadata + shared code, not copied from bespoke stale directories

### Decision 6: Be honest about portability tiers

Not every resource should pretend to be equally supported.

Every implemented resource must explicitly declare a portability tier:

- `full`
  - intended to work on Linux, macOS, and Windows
- `partial`
  - Go-native control plane is cross-platform, but install/runtime support is limited to some OSes
- `platform-specific`
  - intentionally limited; other platforms must fail clearly and honestly

---

## 4. Target model

The target system separates capability knowledge, active integrations, and deprecated history.

### 4.1 States

Each resource concept can exist in one of these states:

1. **Blueprint**
   - structured description only
   - not executable
   - used as inspiration and implementation seed

2. **Implemented**
   - scaffolded from a canonical template
   - supported by the Go-native control plane
   - tested and usable

3. **Validated**
   - implemented and proven in a real scenario/project workflow

4. **Maintained**
   - validated and explicitly retained as an active supported resource

5. **Deprecated**
   - removed from active discovery and normal maintenance
   - archived outside git for a retention period
   - represented by metadata and optionally a replacement blueprint

6. **Purged**
   - archive deleted after retention window
   - metadata retained for historical traceability

### 4.2 Core directories

#### In-repo

- `.vrooli/resource-blueprints/`
  - canonical blueprint records
- `.vrooli/schemas/resource-blueprint.schema.json`
  - schema for blueprint records
- `.vrooli/deprecated-resources.json`
  - metadata for deprecated resources
- `resources/`
  - only active implemented resources
- `scripts/resources/templates/`
  - canonical resource templates and generation assets
- `docs/resources/`
  - operator and contributor docs for the new system

#### Outside git / outside repo

- `~/.vrooli/archive/resources/<timestamp>-<resource-name>/`
  - exported code and metadata for deprecated resources

This archive location keeps dead code recoverable without polluting the repo.

### 4.3 Blueprint record model

Each blueprint should be a structured JSON document, with human-readable fields and enough fidelity to scaffold implementation later.

Recommended fields:

- `name`
- `display_name`
- `category`
- `summary`
- `why_it_matters`
- `when_to_use`
- `example_scenarios`
- `integration_kind`
  - `docker-service`
  - `compose-service`
  - `external-cli`
  - `cloud-api`
  - `library`
  - `desktop-app`
  - `hardware`
  - `manual`
- `platform_support`
- `prerequisites`
- `dependencies`
- `suggested_template`
- `implementation_notes`
- `operational_notes`
- `risks`
- `status`
  - `candidate`
  - `validated`
  - `prioritized`
- `replacement_for`
  - list of old resource names if applicable
- `references`
- `last_reviewed`

Blueprints must be easy to browse, search, and promote into real resource implementations.

### 4.4 Deprecation metadata model

`.vrooli/deprecated-resources.json` should track, at minimum:

- `name`
- `deprecated_at`
- `reason`
- `replacement`
  - blueprint name or active resource
- `archive_path`
- `archive_hash`
- `retention_policy_days`
- `restore_supported`
- `purge_after`

This metadata stays in repo. The code does not.

---

## 5. Resource classification and triage approach

Before implementing templates or drivers, we need a rigorous way to decide what remains active.

### 5.1 Evaluation dimensions

Every current resource should be scored on these axes:

1. **Active use**
   - referenced by current project-level workflows
   - referenced by active scenarios
   - used by humans/agents in current practice

2. **Validation**
   - known to work recently
   - covered by real tests
   - documented from actual usage, not speculation

3. **Strategic value**
   - materially expands Vrooli capability
   - likely to be needed again soon

4. **Cross-platform value**
   - worth making portable
   - benefits from a shared driver model

5. **Maintenance cost**
   - shell-heavy
   - host-native
   - brittle install path
   - large surface area relative to value

6. **Duplication or obsolescence**
   - superseded by another resource
   - obsolete due to platform changes
   - redundant with a better current approach

### 5.2 Default classification rules

- **Keep as implemented**
  - active use is clear, validation exists, and strategic value is high

- **Convert to blueprint**
  - no active use, but capability still seems useful later

- **Deprecate**
  - low-value, obsolete, broken, or superseded

### 5.3 Required evidence sources during triage

To reduce guesswork, triage should check:

- references in `.vrooli/service.json`
- references in `scenarios/*/.vrooli/service.json`
- references in scenario docs and tests
- references in project docs/scripts
- evidence of recent runtime validation
- evidence of duplicate capability elsewhere

### 5.4 Expected result of triage

The expected result is that the current large resource tree shrinks substantially.

Likely surviving active implementations will cluster around:

- core containerized infrastructure
- actively used AI/dev tooling
- a few proven external-tool wrappers

Most speculative resources should become blueprints or deprecated archives.

---

## 6. Cross-platform architecture for implemented resources

The resource system should not be rebuilt as 79 bespoke Go packages.

Instead, we should use typed manifests plus a small set of reusable drivers.

### 6.1 Go-native control plane

The future `vrooli resource` system should own:

- discovery
- manifest validation
- install
- uninstall
- start
- stop
- restart
- status
- health checks
- logs
- dependency ordering
- blueprint promotion
- deprecate/archive/restore operations

### 6.2 Resource manifest

Each implemented resource should have a typed manifest, likely JSON, owned by the Go control plane.

Suggested file:

- `resources/<name>/resource.json`

Suggested fields:

- `name`
- `display_name`
- `description`
- `template`
- `driver`
- `portability_tier`
- `platforms`
- `dependencies`
- `ports`
- `health_checks`
- `install`
  - per-platform strategy
- `runtime`
  - env, volumes, data dirs, commands, images
- `capabilities`
  - supported commands / content operations
- `template_version`
- `lifecycle`
  - start order, timeout, retries

This replaces the loose coupling of shell scripts, `runtime.json`, and ad hoc behavior.

### 6.3 Driver model

Canonical Go drivers should be introduced for these archetypes:

1. `docker-service`
   - single-container services
   - examples: postgres, redis, qdrant, browserless

2. `compose-service`
   - multi-container services
   - examples: heavier services needing compose-style orchestration

3. `external-cli`
   - cross-platform or semi-cross-platform command-line tools
   - examples: claude-code, codex, ffmpeg-class tools

4. `cloud-api`
   - resources primarily representing configuration/auth/wrappers around hosted APIs

5. `desktop-app`
   - native applications with platform-specific install logic

6. `manual`
   - resources Vrooli can describe and validate partially, but not fully install/manage everywhere

7. `legacy-adapter`
   - temporary bridge for still-shell-backed resources during migration

### 6.4 Shared Go packages

The following internal packages will likely be needed:

- `internal/resources`
  - controller, manifest loading, high-level orchestration
- `internal/resources/blueprints`
  - blueprint schema, indexing, promotion
- `internal/resources/drivers`
  - driver interfaces and registry
- `internal/resources/drivers/docker`
  - Docker-backed services
- `internal/resources/drivers/compose`
  - compose-backed services
- `internal/resources/drivers/externalcli`
  - installed binary / package-manager / direct executable model
- `internal/resources/drivers/cloudapi`
  - config-only and validation-focused resources
- `internal/resources/archive`
  - archive, restore, garbage collection
- `internal/resources/templates`
  - template rendering and generation
- `internal/resources/health`
  - HTTP/TCP/command health checks
- `internal/resources/platform`
  - platform support checks and install helpers

These should reduce duplication and prevent every resource from owning bespoke lifecycle logic.

### 6.5 Platform support rules

Use build tags where real OS-specific behavior is necessary.

Examples:

- `internal/resources/platform/install_linux.go`
- `internal/resources/platform/install_darwin.go`
- `internal/resources/platform/install_windows.go`

Unsupported paths should fail with typed, honest errors rather than hidden no-ops.

---

## 7. Resource blueprints

Blueprints are central to this migration strategy, not a side feature.

### 7.1 Purpose

Blueprints preserve:

- knowledge of useful tools/services
- likely use cases
- implementation hints
- template choice
- operational considerations

without preserving executable code that nobody uses.

### 7.2 Blueprint UX

Future commands should include:

- `vrooli resource blueprint list`
- `vrooli resource blueprint info <name>`
- `vrooli resource blueprint search <query>`
- `vrooli resource blueprint create <name>`
- `vrooli resource blueprint validate`
- `vrooli resource blueprint promote <name>`

Promotion should scaffold a new implemented resource using the chosen canonical template, seeded from blueprint metadata.

### 7.3 Promotion flow

`blueprint -> implemented resource` should look like:

1. choose blueprint
2. choose or confirm canonical template
3. generate resource skeleton
4. fill in manifest and docs from blueprint fields
5. implement driver-specific details
6. run validation checklist

### 7.4 Blueprint quality bar

Blueprints should be:

- structured enough to guide implementation
- honest about platform support and uncertainty
- lightweight enough that adding one is much cheaper than shipping a new resource

---

## 8. Deprecation, archive, restore, and purge

Deprecation must be treated as a first-class lifecycle.

### 8.1 Goals

- remove dead code from the active repo
- preserve recoverability during a retention window
- keep historical traceability
- enable future cleanup

### 8.2 Proposed archive workflow

When deprecating a resource:

1. export the resource directory to:
   - `~/.vrooli/archive/resources/<timestamp>-<resource-name>/`
2. compute a hash / integrity record
3. add metadata entry to `.vrooli/deprecated-resources.json`
4. optionally generate or update a matching blueprint
5. remove the resource from active repo discovery
6. update docs and template mappings if needed

### 8.3 Restore workflow

During the retention window:

- `vrooli resource restore <name>` should restore the archived resource into a controlled location
- restored resources should come back as `legacy-adapter` or `quarantined` until explicitly promoted again

This prevents accidental resurrection of stale code into the main supported set.

### 8.4 Purge workflow

After retention expires:

- `vrooli resource archive gc` removes old archives from `~/.vrooli/archive/resources/`
- metadata remains in `.vrooli/deprecated-resources.json`
- if the capability still matters, the blueprint remains

### 8.5 Initial retention recommendation

Start with:

- default retention: `90 days`
- overrideable for exceptions

This is long enough for recovery, short enough to avoid permanent graveyard growth.

---

## 9. Template system for implemented resources

Resources need a real template system similar in philosophy to scenarios.

### 9.1 Template philosophy

Do not create one template per vendor/product.

Instead, create a small set of canonical archetype templates.

These templates should match the driver model and minimize duplicated code.

### 9.2 Proposed template location

- `scripts/resources/templates/README.md`
- `scripts/resources/templates/<template-name>/template.json`
- `scripts/resources/templates/<template-name>/README.md`
- `scripts/resources/templates/<template-name>/...template files...`

This mirrors the scenario template layout.

### 9.3 Required template metadata

Each template should include `template.json` with:

- `name`
- `displayName`
- `description`
- `driver`
- `requiredVars`
- `optionalVars`
- `docs`
- `postHooks`
- `platformExpectations`

### 9.4 Canonical templates to include

#### Template 1: `docker-service`

Use for:

- single-service containerized infrastructure
- examples: postgres, redis, qdrant-class resources

Generated structure should include:

- `resource.json`
- `README.md`
- `config/runtime.json` only if transitional compatibility is needed
- health config
- data dirs / volumes
- driver settings
- test scaffold

Expected setup:

- image reference
- env vars
- ports
- volume mappings
- HTTP/TCP/command health checks
- platform note that Docker is prerequisite

#### Template 2: `compose-service`

Use for:

- multi-container resources
- services requiring a coordinated runtime stack

Generated structure should include:

- `resource.json`
- compose manifest or equivalent driver config
- health checks for composed services
- docs for local operator assumptions
- test scaffold

Expected setup:

- service graph
- volumes/networks
- startup order
- readiness criteria

#### Template 3: `external-cli`

Use for:

- tools primarily represented by an installed executable
- examples: claude-code, codex, ffmpeg-class tools

Generated structure should include:

- `resource.json`
- per-platform install rules
- version detection command
- health/probe command
- optional auth/config skeleton
- test scaffold

Expected setup:

- install strategy by OS
- binary detection
- version checks
- config dirs
- health command

#### Template 4: `cloud-api`

Use for:

- hosted services where the main work is auth/config/validation

Generated structure should include:

- `resource.json`
- credential config model
- validation checks
- API health or auth smoke check
- docs for required secrets

Expected setup:

- credential references
- endpoint config
- connection test
- optional webhook/push callback fields

#### Template 5: `desktop-app`

Use for:

- native applications with strong OS-specific behavior
- examples: design/media/GUI tools if they remain active

Generated structure should include:

- `resource.json`
- platform-specific install notes
- detection/validation commands
- explicit unsupported behavior
- manual steps section

Expected setup:

- per-OS install hints
- app path detection
- optional plugin/config path support
- capability limitations

#### Template 6: `manual-resource`

Use for:

- resources that Vrooli can describe and partially validate but not own fully

Generated structure should include:

- `resource.json`
- docs-heavy setup guide
- explicit manual installation checklist
- validation probes

Expected setup:

- prerequisites
- platform caveats
- validation steps
- no promise of full automation

#### Template 7: `legacy-adapter`

Use only during migration.

Purpose:

- wrap an existing Bash resource behind a typed manifest while it awaits either real migration or deprecation

Generated structure should include:

- `resource.json`
- explicit note that it is transitional
- references to old script paths
- deprecation target date

This template must not become a permanent comfort blanket.

### 9.5 Template recommendation rules

`blueprint.suggested_template` should normally point to one of these canonical templates.

Promotion should refuse unknown template types.

### 9.6 Template usage policy

Just like scenarios:

- start from an approved template
- do not create ad hoc template folders casually
- improve canonical templates instead of multiplying variants without discipline

---

## 10. What actual generated resource setup should look like

This section is intentionally concrete so future work does not reinvent skeletons.

### 10.1 Minimal implemented resource layout

For a new Go-native implemented resource:

```text
resources/<name>/
├── README.md
├── resource.json
├── config/
│   ├── defaults.json
│   └── schema.json
├── test/
│   ├── smoke.json
│   ├── integration.json
│   └── fixtures/
└── docs/
    └── OPERATIONS.md
```

Notes:

- `cli.sh` should not be required in the target architecture
- shell files may temporarily exist only for transitional `legacy-adapter` resources
- the Go control plane should interpret `resource.json` plus driver data

### 10.2 Transitional compatibility layout

During migration, some resources may still need:

```text
resources/<name>/
├── README.md
├── resource.json
├── cli.sh
├── config/
│   ├── defaults.sh
│   ├── runtime.json
│   └── schema.json
├── lib/
│   └── *.sh
└── test/
```

These must be explicitly tagged as `legacy-adapter` resources and scheduled either for migration or deprecation.

### 10.3 Shared code vs resource-local code

Shared code belongs in Go packages, not per-resource shell libraries.

Put shared logic in:

- drivers
- health-check packages
- platform install helpers
- archive/blueprint/template services

Keep resource-local code only for:

- true product-specific behavior
- product-specific content operations
- product-specific configuration quirks

Anything repeated across 2+ resources should be considered for shared extraction.

---

## 11. Phased migration strategy

This work is intentionally broken into phases so it can be executed over many conversations.

### Phase 0 — Inventory and decision framework

**Goal:** Establish ground truth before changing architecture.

- [x] Inventory all current resources
- [x] Build usage map from project config, scenarios, docs, and tests
- [x] Build validation map: recently used, test-backed, undocumented, stale
- [x] Assign preliminary classifications: `keep`, `blueprint`, `deprecate`
- [x] Document the active/core resource set that must remain implemented
- [x] Document likely blueprint candidates
- [x] Document likely direct deprecations

Supporting artifacts produced during Phase 0:

- [Resource Phase 0 Inventory](/home/matthalloran8/Vrooli/docs/resources/resource-phase0-inventory.md)
- [Dependency Contract Audit](/home/matthalloran8/Vrooli/docs/resources/dependency-contract-audit.md)
- [Resource Registry Reconciliation](/home/matthalloran8/Vrooli/docs/resources/resource-registry-reconciliation.md)
- [Dependency Contract Validator](/home/matthalloran8/Vrooli/scripts/resources/tools/validate-dependency-contract.sh)

**Deliverable:** A triage inventory with proposed state for every current resource.

**Acceptance:** There is a reviewed table showing which resources survive as implemented and why.

**Status update:** Phase 0 inventory and classification are complete. Current reviewed proposal:

- `29` resources proposed as `keep`
- `52` resources proposed as `blueprint`
- `4` resources proposed as `deprecate`

Important decisions made during Phase 0:

- scenario/resource dependency manifests now use a flat keyed map contract
- `startup_policy` was introduced and wired into scenario dependency startup behavior
- `.vrooli/resource-registry/` is treated as transitional metadata, not a canonical source of truth

### Phase 1 — Blueprint system

**Goal:** Create a first-class structured replacement for speculative resource code.

- [ ] Define blueprint schema
- [ ] Add `.vrooli/resource-blueprints/`
- [ ] Add blueprint docs and operator guidance
- [ ] Add initial commands for listing and viewing blueprints
- [ ] Seed blueprints from a first batch of low-risk speculative resources

**Deliverable:** Resource blueprints exist as a real supported concept.

**Acceptance:** A user can inspect blueprint records without touching `resources/`.

### Phase 2 — Deprecation and archive lifecycle

**Goal:** Make deprecation safe, recoverable, and explicit.

- [ ] Define `.vrooli/deprecated-resources.json`
- [ ] Implement external archive export path under `~/.vrooli/archive/resources/`
- [ ] Implement `deprecate`, `list-deprecated`, `restore`, and archive GC commands
- [ ] Define retention policy and restore semantics
- [ ] Convert first batch of clearly stale resources into `deprecated + blueprint`

**Deliverable:** Dead resources can leave the repo cleanly without losing recoverability.

**Acceptance:** A deprecated resource is no longer active in repo discovery, but can still be restored during retention.

### Phase 3 — Template system

**Goal:** Replace ad hoc resource creation with canonical templates.

- [ ] Create `scripts/resources/templates/<template>/` layout
- [ ] Add template metadata format mirroring scenario templates
- [ ] Implement canonical templates:
  - [ ] `docker-service`
  - [ ] `compose-service`
  - [ ] `external-cli`
  - [ ] `cloud-api`
  - [ ] `desktop-app`
  - [ ] `manual-resource`
  - [ ] `legacy-adapter`
- [ ] Add generation docs and usage examples
- [ ] Add blueprint-to-template recommendation rules

**Deliverable:** New resources are scaffolded from a small high-quality template set.

**Acceptance:** It is no longer necessary to clone an old resource directory to start a new one.

### Phase 4 — Go-native resource manifests and driver interfaces

**Goal:** Create the target control plane architecture.

- [ ] Define `resource.json` schema
- [ ] Define driver interfaces
- [ ] Implement manifest loading and validation in Go
- [ ] Implement driver registry
- [ ] Add typed platform support metadata
- [ ] Add driver-independent health check framework

**Deliverable:** Implemented resources can be described and controlled without depending on shell-first resource structure.

**Acceptance:** At least one prototype resource can run entirely through manifest + driver logic.

### Phase 5 — Migrate the core active resource set

**Goal:** Move the small validated set of important resources to the new model.

Recommended order:

1. containerized infrastructure
   - [ ] postgres
   - [ ] redis
   - [ ] qdrant
   - [ ] browserless
   - [ ] vault if still clearly active
2. active external tooling
   - [ ] claude-code
   - [ ] codex
   - [ ] gemini/openrouter class resources if truly active
3. any additional validated resources justified by current use

For each migrated resource:

- [ ] classify portability tier
- [ ] assign canonical template
- [ ] create `resource.json`
- [ ] implement driver-backed lifecycle
- [ ] remove bespoke duplicated shell logic where possible
- [ ] add validation tests

**Deliverable:** The active resource set is substantially Go-native and cross-platform-aware.

**Acceptance:** `vrooli resource` no longer depends on Bash for the migrated core set.

### Phase 6 — Legacy adapter shrinking

**Goal:** Collapse the old shell estate without forcing premature rewrites.

- [ ] Convert any remaining not-yet-deprecated shell resources into explicit `legacy-adapter` form
- [ ] Add deadlines/owners for each remaining adapter
- [ ] Decide per adapter:
  - [ ] migrate fully
  - [ ] convert to blueprint
  - [ ] deprecate and archive

**Deliverable:** The remaining shell-based resources are a small explicit backlog, not a sprawling default.

**Acceptance:** Every remaining shell resource has an explicit plan and no hidden status.

### Phase 7 — Cleanup and contract hardening

**Goal:** Make the new model the default and remove drift.

- [ ] Make blueprints, templates, and deprecation workflows part of official docs
- [ ] Update contributor guidance
- [ ] Remove obsolete resource framework dependencies
- [ ] Narrow active resource discovery to new manifests
- [ ] Move old contract docs into historical/transitional context if needed
- [ ] Add maintenance checks to keep the active resource set small and validated

**Deliverable:** The new resource architecture is the stable default.

**Acceptance:** Adding a new resource naturally goes through blueprint -> template -> implementation, not ad hoc shell cloning.

---

## 12. Concrete checklist for the first triage pass

This is the first real workstream after this plan lands.

For every current resource:

- [ ] record whether it is enabled in `.vrooli/service.json`
- [ ] record whether any scenario references it
- [ ] record whether any current project workflow uses it
- [ ] record whether there is evidence of recent successful use
- [ ] classify integration kind
- [ ] classify portability tier
- [ ] estimate maintenance cost
- [ ] choose target state:
  - [ ] keep implemented
  - [ ] convert to blueprint
  - [ ] deprecate
- [ ] if kept, assign canonical template
- [ ] if blueprinted, record suggested template
- [ ] if deprecated, record archive plan and replacement blueprint if applicable

---

## 13. Testing and validation strategy

This migration needs tests at the architectural level, not just per resource.

### Required test categories

1. **Blueprint tests**
   - schema validation
   - search/list/info behavior
   - blueprint promotion input validation

2. **Deprecation/archive tests**
   - archive export
   - manifest entry creation
   - restore
   - archive garbage collection

3. **Template generation tests**
   - each canonical template renders valid output
   - generated resources pass manifest/schema validation

4. **Driver tests**
   - docker driver lifecycle
   - external CLI driver detection/install/status
   - cloud API validation logic
   - unsupported platform behavior

5. **Migration tests**
   - legacy adapter compatibility for selected resources
   - blueprint conversion of deprecated resources
   - active resource discovery excludes deprecated items

6. **End-to-end control plane tests**
   - `vrooli resource list`
   - `vrooli resource status`
   - blueprint listing
   - deprecate -> restore cycle

### Validation principle

Tests should validate desired behavior, not preserve accidental shell quirks unless those quirks are explicitly required.

---

## 14. Risks and mitigations

### Risk 1: Keeping too many active resources

If too many resources survive triage, the migration becomes another rewrite of stale code.

**Mitigation:** Bias uncertain resources toward blueprints.

### Risk 2: Blueprints become unstructured notes

If blueprint quality is too low, they will not actually help future implementation.

**Mitigation:** Use a schema and require template recommendation plus implementation notes.

### Risk 3: Deprecation becomes a graveyard with no cleanup

If archives never expire, the problem just moves from git to `~/.vrooli`.

**Mitigation:** Retention windows plus archive GC.

### Risk 4: Legacy adapters become permanent

If the legacy template is too comfortable, shell wrappers will persist forever.

**Mitigation:** Mark them transitional, time-box them, and track a migration/deprecation decision for each.

### Risk 5: Template explosion

If new template folders proliferate the way old resources did, the system becomes inconsistent again.

**Mitigation:** Keep a small canonical template set and require explicit approval for new archetypes.

### Risk 6: Dishonest portability claims

If platform support is not explicit, users will assume more portability than exists.

**Mitigation:** Require portability tier and platform matrix in manifests and status output.

---

## 15. Open questions to settle during implementation

- What exact schema format should `resource.json` use relative to current `runtime.json` and `schema.json` conventions?
- Should the blueprint store be JSON-only, or allow Markdown companions for richer human notes?
- Should archive export include git metadata and diff context, or just files plus metadata JSON?
- Which current resources are definitely part of the long-term active core set?
- How much of the old `content` command surface should be driver-native vs resource-local?
- Should template generation live under `vrooli resource generate`, `vrooli resource template`, or blueprint promotion only?

These are implementation questions, not blockers for starting the phased work.

---

## 16. Immediate next steps

The recommended next slice after this plan is:

1. perform the full resource inventory and triage
2. agree on the initial active/core resource set
3. define the blueprint schema and storage location
4. implement the first blueprint/deprecation lifecycle slice before migrating resource code

This order is intentional. The platform should decide what deserves migration before writing migration code.

---

## 17. Summary

The correct path is not "port every current resource to Go."

The correct path is:

1. separate capability knowledge from maintained implementation
2. keep only validated resources active
3. introduce first-class blueprints
4. introduce first-class deprecation/archive/restore
5. standardize resource generation around a small template set
6. migrate the surviving active resource set onto a Go-native driver-based control plane

If this plan is followed, Vrooli will end up with a cleaner, more honest, and more extensible resource system that is actually worth making cross-platform.
