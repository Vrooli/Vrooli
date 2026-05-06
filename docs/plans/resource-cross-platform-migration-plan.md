# Resource Cross-Platform Migration Plan

**Status:** In Progress
**Owner:** Matthew Halloran
**Scope:** Project-level resource system under `resources/`, `path:scripts/resources/`, `path:scripts/lib/resources/`, and the corresponding `vrooli resource` control surface
**Out of Scope:** Scenario internals under `path:scenarios/*/`, except where scenario template/resource patterns provide precedent
**Primary Goal:** Replace the current shell-first, high-maintenance resource system with a cross-platform, Go-native control plane built around the active retained resource set, first-class resource blueprints, and managed deprecation/archive workflows

---

## 0. For agents picking this up later

If you are an agent resuming this work in a future session, read this section first.

- **What this plan covers:** The full migration strategy for resources, including architecture, blueprints, deprecation, templates, shared code, phased execution, and cleanup.
- **What this plan does not cover:** Scenario implementation internals. Scenario-level cross-platform work is already on its own track.
- **Important status update:** The blueprint cleanup pass has already happened. The speculative and stale implemented resources were already moved out of the active tree via blueprint/deprecation workflows. The resources still present under `resources/` should now be treated as intentionally retained and currently needed.
- **What problem this is solving now:** The remaining problem is not broad triage. It is that the retained active resources are still largely implemented through shell-era CLIs and libraries, and need a clearer manifest-native, Go-owned architecture.
- **What success looks like:** Vrooli ends up with:
  - the current active retained resource set managed through a cross-platform-capable control plane
  - a first-class `resource blueprint` catalog for future capabilities
  - a built-in `literal:deprecate/archive/restore` lifecycle
  - a Go-native resource control plane with platform-aware drivers and templates
- **How to resume work:** Find the first unchecked item in the phase checklist, re-read that phase, and execute only that bounded slice. This plan is intended to be completed piecemeal across multiple conversations.
- **Important bias:** Do not re-open broad resource triage unless requirements have changed. Assume the active `resources/` tree is already the curated set that should receive native migration work.

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

2. **The retained resources still carry shell-era implementation weight.**
   - The previous speculative/stale backlog was already cleaned up through blueprints, deprecation, and archive workflows.
   - The resources that remain are still expensive to maintain because most of them preserve Bash CLIs, Bash libraries, and Bash-shaped contracts beside the new manifests.

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

If the resource system remains shell-heavy and Linux-coupled:

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

Current reality after the blueprint/archive cleanup:

- `resources/` should be read as the active retained resource set, not as an unfiltered backlog of ideas.
- Blueprint preservation and deprecation/archive workflows are already real and should not be treated as future concepts.
- The main migration question is now how to make the retained resources natively manageable, not which resources deserve to exist.

- Project-level enablement lives in `.vrooli/service.json` under `dependencies.resources`
- Resource registry metadata is stored in `.vrooli/resource-registry/*.json`
- Running/installed status tracking currently uses `.vrooli/running-resources.json`
- Each active resource still usually has:
  - `path:resources/<name>/cli.sh`
  - `path:resources/<name>/config/defaults.sh`
  - `path:resources/<name>/config/runtime.json`
  - `path:resources/<name>/lib/*.sh`
  - `path:resources/<name>/test/*`
  - `path:resources/<name>/resource.json`

### Current control surface

The current hybrid model is:

1. project-level orchestration determines which resources are enabled
2. the Go control plane discovers `path:resources/<name>/resource.json`
3. manifest-native drivers handle supported lifecycle operations where implemented
4. unsupported operations still fall back to `resource-<name>` or `path:resources/<name>/cli.sh`
5. resource-specific shell libraries still perform much of the real work

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

5. **Platform-specific or shell-heavy holdouts**
   - Some retained resources will still need manual, desktop, or explicit legacy-adapter treatment rather than a fake "fully native everywhere" story
   - These should be classified honestly instead of hidden behind Bash compatibility forever

### Current template state

Scenarios already have a strong template model:

- `path:templates/scenarios/`
- template metadata files such as `template.json`
- standard generation flow

Resources now do have a canonical template catalog under `path:templates/resources/`, plus a leftover `path:templates/resources/PRD.md` historical seed.

That closes the old "no resource templates exist" gap, but it does not yet mean resources are fully native. The remaining gap is that the templates are still primarily scaffold + manifest assets, while much operational behavior in active resources still lives in shell compatibility surfaces.

---

## 3. Principles and decisions

These are the strategic decisions this plan assumes.

### Decision 1: Blueprints are first-class

We will introduce **resource blueprints** as the canonical way to preserve resource capability knowledge without keeping stale code in the repo.

Blueprints are not brainstorm scraps. They are structured, reusable implementation seeds.

### Decision 2: Only the curated retained resource set remains implemented

The active `resources/` tree should contain only:

- resources used by current project-level workflows
- resources used by active scenarios
- resources with clear strategic value and recent validation

This cleanup has already happened for the previous speculative/stale backlog.

Anything removed from this retained set should either:

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

- `.vrooli/resources/blueprints/`
  - canonical blueprint records
- `.vrooli/schemas/resource-blueprint.schema.json`
  - schema for blueprint records
- `.vrooli/resources/deprecated-resources.json`
  - metadata for deprecated resources
- `resources/`
  - only active implemented resources
- `path:templates/resources/`
  - canonical resource templates and generation assets
- `path:docs/resources/`
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

`.vrooli/resources/deprecated-resources.json` should track, at minimum:

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

This section is now primarily historical context plus a guardrail for future additions. The main triage pass already happened; the remaining active resources are assumed to be intentionally retained.

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
- references in `path:scenarios/*/.vrooli/service.json`
- references in scenario docs and tests
- references in project docs/scripts
- evidence of recent runtime validation
- evidence of duplicate capability elsewhere

### 5.4 Result of triage

The resource tree has already been reduced to the retained active set.

The practical implication for the remaining work is:

- assume current active resources should migrate forward unless a new explicit decision says otherwise
- use blueprint/deprecation/archive workflows only for future removals or newly discovered mistakes
- spend migration effort on native architecture, driver ownership, and template quality rather than repeating broad keep/blueprint/deprecate review

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

- `path:resources/<name>/resource.json`

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

- `path:internal/resources`
  - controller, manifest loading, high-level orchestration
- `path:internal/resources/blueprints`
  - blueprint schema, indexing, promotion
- `path:internal/resources/drivers`
  - driver interfaces and registry
- `path:internal/resources/drivers/docker`
  - Docker-backed services
- `path:internal/resources/drivers/compose`
  - compose-backed services
- `path:internal/resources/drivers/externalcli`
  - installed binary / package-manager / direct executable model
- `path:internal/resources/drivers/cloudapi`
  - config-only and validation-focused resources
- `path:internal/resources/archive`
  - archive, restore, garbage collection
- `path:internal/resources/templates`
  - template rendering and generation
- `path:internal/resources/health`
  - HTTP/TCP/command health checks
- `path:internal/resources/platform`
  - platform support checks and install helpers

These should reduce duplication and prevent every resource from owning bespoke lifecycle logic.

### 6.5 Platform support rules

Use build tags where real OS-specific behavior is necessary.

Examples:

- `path:internal/resources/platform/install_linux.go`
- `path:internal/resources/platform/install_darwin.go`
- `path:internal/resources/platform/install_windows.go`

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
3. add metadata entry to `.vrooli/resources/deprecated-resources.json`
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
- metadata remains in `.vrooli/resources/deprecated-resources.json`
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

- `path:templates/resources/README.md`
- `path:templates/resources/<template-name>/template.json`
- `path:templates/resources/<template-name>/README.md`
- `path:templates/resources/<template-name>/...template files...`

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
- zero Bash files in the template itself
- explicit native environment export model
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
- explicit extension points for custom native commands when the standard driver surface is insufficient

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
- canonical resource templates must be Bash-free and cross-platform by default
- any compatibility surface must live outside the template archetype contract and be explicitly marked transitional

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
- canonical templates should not generate any Bash files
- shell files may temporarily exist only in explicit compatibility or `legacy-adapter` paths during migration
- the Go control plane should interpret `resource.json` plus driver data

### 10.1.1 Native environment contract

One of the old shell-era responsibilities that must be replaced deliberately is environment injection for dependent scenarios.

Historically this was often handled through `config/exports.sh`, sometimes with additional values derived from:

- port registry state
- shell defaults
- secrets resolution
- computed URLs such as `POSTGRES_URL`, `DATABASE_URL`, `REDIS_URL`, `QDRANT_URL`, `OLLAMA_URL`
- ad hoc resource-specific special cases

The target architecture must replace that with a native Go contract. For active resources, environment injection should come from:

- `resource.json`
  - canonical runtime ports, env, credentials references, and resource metadata
- a typed native environment export model
  - explicitly describing which scenario-facing environment variables the resource provides
- native secrets loading
- native computed-value generation
  - for example derived URLs built from host, port, database name, and credentials

This native environment model should be treated as part of the resource contract, not as hidden implementation detail.

### 10.1.2 File-by-file audit rule

Before porting any shell-era resource file into the native model, classify it:

- `still-authoritative`
  - live behavior depends on it today and that behavior must migrate
- `replace-with-native-structure`
  - behavior is still needed, but the file format or location should change
- `compatibility-only`
  - temporary bridge that should move into an isolated compatibility layer
- `delete`
  - historical baggage with no justified future role

This applies especially to:

- `config/exports.sh`
- `config/defaults.sh`
- `config/runtime.json`
- `config/schema.json`
- `config/capabilities.yaml`
- `config/messages.sh`
- resource-local `lib/*.sh`

### 10.2 Transitional compatibility layout

During migration, some resources may still need:

```text
resources/<name>/
├── README.md
├── resource.json
├── compat/
│   ├── README.md
│   ├── bridge.json
│   └── shell/
│       ├── cli.sh
│       ├── config/
│       └── lib/
└── test/
```

Rules:

- compatibility code must be quarantined under an explicit compatibility area instead of blending with the native resource shape
- the native manifest and native control plane remain authoritative
- compatibility behavior must be documented as temporary, removable, and outside the canonical template contract
- resources using compatibility code must carry an explicit migration/removal plan

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

Keep compatibility-only code only for:

- shell-era behavior that is still live and not yet migrated
- short-lived bridges with explicit ownership and removal criteria

Do not let compatibility files become the de facto contract again.

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

- this plan's Phase 0 section, which now serves as the retained historical record
- `path:docs/resources/resource-registry-reconciliation.md` (historical note; file no longer present)
- [Dependency Contract Validator](/home/matthalloran8/Vrooli/scripts/resources/tools/validate-dependency-contract.sh)

**Deliverable:** A triage inventory with proposed state for every current resource.

**Acceptance:** There is a reviewed table showing which resources survive as implemented and why.

**Status update:** Phase 0 inventory and classification are complete. Current reviewed proposal:

- `29` resources proposed as `keep`
- `53` resources proposed as `blueprint`
- `4` resources proposed as `deprecate`

Important decisions made during Phase 0:

- scenario/resource dependency manifests now use a flat keyed map contract
- `startup_policy` was introduced and wired into scenario dependency startup behavior
- `.vrooli/resource-registry/` is treated as transitional metadata, not a canonical source of truth
- stale `node-red` scenario manifest dependencies were removed; remaining Node-RED artifacts are treated as historical prototypes/blueprint material

### Phase 1 — Blueprint system

**Goal:** Create a first-class structured replacement for speculative resource code.

- [x] Define blueprint schema
- [x] Add `.vrooli/resources/blueprints/`
- [x] Add blueprint docs and operator guidance
- [x] Add initial commands for listing and viewing blueprints
- [x] Seed blueprints from a first batch of low-risk speculative resources

**Deliverable:** Resource blueprints exist as a real supported concept.

**Acceptance:** A user can inspect blueprint records without touching `resources/`.

**Status update:** Phase 1 is now inventory-complete rather than seed-only.

- `.vrooli/resources/blueprints/` now covers the full current Phase 0 `blueprint` set
- the Go test suite validates drift between the retained Phase 0 classification and the blueprint store
- operator guidance documents blueprint inspection and validation as a supported workflow
- the Phase 1 closeout validation bundle is:
  - `vrooli resource blueprint validate`
  - `vrooli resource blueprint list`
  - `vrooli resource blueprint info terraform`
  - `vrooli resource blueprint search network`
  - `go test ./internal/resources ./cmd/vrooli`

### Phase 2 — Deprecation and archive lifecycle

**Goal:** Make deprecation safe, recoverable, and explicit.

- [x] Define `.vrooli/resources/deprecated-resources.json`
- [x] Implement external archive export path under `~/.vrooli/archive/resources/`
- [x] Implement `deprecate`, `list-deprecated`, `restore`, and archive GC commands
- [x] Define retention policy and restore semantics
- [x] Convert first batch of clearly stale resources into `deprecated + blueprint`

**Deliverable:** Dead resources can leave the repo cleanly without losing recoverability.

**Acceptance:** A deprecated resource is no longer active in repo discovery, but can still be restored during retention.

**Status update:** Phase 2 is implemented and validated as a native Go workflow.

- `.vrooli/resources/deprecated-resources.json` is now the in-repo metadata source for deprecated resource state
- archived resource state exports to `~/.vrooli/archive/resources/`
- `vrooli resource deprecate <name>`, `list-deprecated`, `restore <name>`, and `archive gc` are implemented
- deprecated resources are excluded from normal `vrooli resource list` / status discovery
- restores are quarantined under `.vrooli/resources/restored/<name>/` instead of silently becoming active again
- the initial deprecation batch is complete for:
  - `autogen-studio`
  - `erpnext`
  - `langchain`
  - `musicgen`
- each deprecated resource now has a matching replacement blueprint record

The focused Phase 2 validation bundle is:

- `go test ./internal/resources`
- `go test ./cmd/vrooli -run 'Test(RunResource(Blueprint(ListCommandJSON|InfoCommandHuman|ValidateCommandHuman|SearchCommandHuman|SearchCommandJSON)|ListDeprecatedCommandJSON|DeprecateCommandHuman|RestoreCommandHuman|ArchiveGCCommandJSON)|ShowResourceHelpIncludesBlueprintCommands)'`
- `go run ./cmd/vrooli resource deprecate autogen-studio`
- `go run ./cmd/vrooli resource list-deprecated`
- `go run ./cmd/vrooli resource restore autogen-studio`
- `go run ./cmd/vrooli resource archive gc`

### Phase 3 — Template system

**Goal:** Replace ad hoc resource creation with canonical templates.

- [x] Create `path:templates/resources/<template>/` layout
- [x] Add template metadata format mirroring scenario templates
- [x] Implement canonical templates:
  - [x] `docker-service`
  - [x] `compose-service`
  - [x] `external-cli`
  - [x] `cloud-api`
  - [x] `desktop-app`
  - [x] `manual-resource`
  - [x] `legacy-adapter`
- [x] Add generation docs and usage examples
- [x] Add blueprint-to-template recommendation rules

**Deliverable:** New resources are scaffolded from a small high-quality template set.

**Acceptance:** It is no longer necessary to clone an old resource directory to start a new one.

**Status update:** Phase 3 is implemented and validated as the canonical scaffold path for new resources.

- `path:templates/resources/` now contains the full canonical template set with shared layout, metadata, docs, and test stubs
- `vrooli resource template list|show|validate|generate` is implemented in the native Go CLI
- blueprints now enforce explicit `integration_kind -> suggested_template` recommendation rules instead of relying on convention
- template validation now checks both manifest correctness and required asset/doc presence
- the Go test suite now validates:
  - generation for every canonical template
  - blueprint-seeded generation across the current archetype set
  - missing required values
  - template/blueprint mismatch rejection
  - overwrite safety through `--force`
  - missing template asset and missing docs reference failures

The focused Phase 3 closeout validation bundle is:

- `go test ./internal/resources ./cmd/vrooli`
- `go run ./cmd/vrooli resource template validate`
- `go run ./cmd/vrooli resource template list`
- `go run ./cmd/vrooli resource template show docker-service`
- `go run ./cmd/vrooli resource template generate --from-blueprint terraform --dry-run`

Phase 4 starts from here. Phase 3 is considered complete once new resource work is expected to go through `blueprint -> template -> implementation` rather than copying a historical `path:resources/<name>/` directory.

### Phase 4 — Go-native resource manifests and driver interfaces

**Goal:** Create the target control plane architecture.

- [x] Define `resource.json` schema
- [x] Define driver interfaces
- [x] Implement manifest loading and validation in Go
- [x] Implement driver registry
- [x] Add typed platform support metadata
- [x] Add driver-independent health check framework

**Deliverable:** Implemented resources can be described and controlled without depending on shell-first resource structure.

**Acceptance:** At least one prototype resource can run entirely through manifest + driver logic.

**Status update:** Phase 4 is implemented and validated as the first manifest-native slice of the resource control plane.

- `.vrooli/schemas/resource.schema.json` now defines the baseline `resource.json` contract for manifest-native resources
- `path:internal/resources` now includes native manifest loading, validation, driver dispatch, platform gating, and shared health-check execution
- `resources.Controller` now distinguishes `manifest-native`, `legacy-adapter`, and `legacy-shell` control modes during discovery and status reporting
- the first native driver registry is in place with a working `docker-service` implementation
- generated resource templates now verify that rendered `resource.json` files pass native manifest validation
- the manifest-native prototype acceptance criterion is covered by repeatable tests using a fake Docker-backed resource lifecycle rather than brittle live-environment assumptions

The focused Phase 4 closeout validation bundle is:

- `go test ./internal/resources -run 'Test(DiscoverMarksManifestNativeResources|StatusForManifestNativeDockerResource|RunManifestNativeDockerLifecycle|StatusForManifestNativeUnsupportedPlatform|StatusForResourceReportsUnavailableCommand|StatusForResourceCategorizesProbeFailures|StatusForResourceParsesStructuredPayload|RunReturnsCategorizedErrors|StartAllUsesBestEffortWhenStatusProbeIsDegraded|StopAllFallsBackWhenStatusProbeIsDegraded)'`
- `go test ./cmd/vrooli -run 'TestRunResource(ListCommandShowsManifestMetadata|StatusCommandShowsManifestMetadata|Blueprint(ListCommandJSON|InfoCommandHuman|ValidateCommandHuman|SearchCommandHuman|SearchCommandJSON)|Template(ListCommandJSON|ShowCommandHuman|ValidateCommandHuman|GenerateCommandHuman)|ListDeprecatedCommandJSON|DeprecateCommandHuman|RestoreCommandHuman|ArchiveGCCommandJSON)|TestShowResourceHelpIncludesBlueprintCommands'`
- `go test ./internal/resources -run 'Test(GenerateResourceTemplate|GenerateResourceTemplateFromBlueprint|GenerateResourceTemplateFromBlueprintRepresentativeArchetypes|GenerateResourceTemplateRejectsBlueprintTemplateMismatch|ValidateResourceTemplates)'`
- `go run ./cmd/vrooli resource template validate`
- `go run ./cmd/vrooli resource list`
- `go run ./cmd/vrooli resource status postgres --fast`

### Phase 5 — Migrate the core active resource set

**Goal:** Move the small validated set of important resources to the new model.

Recommended order:

1. containerized infrastructure
   - [x] postgres
   - [x] redis
   - [x] qdrant
   - [x] browserless
   - [x] vault
2. active external tooling
   - [x] claude-code
   - [x] codex
   - [x] gemini/openrouter class resources if truly active
3. any additional validated resources justified by current use

For each migrated resource:

- [x] classify portability tier
- [x] assign canonical template
- [x] create `resource.json`
- [x] implement driver-backed lifecycle
- [x] remove bespoke duplicated shell logic where possible
- [x] add validation tests

**Deliverable:** The active resource set is substantially Go-native and cross-platform-aware.

**Acceptance:** `vrooli resource` no longer depends on Bash for the migrated core set.

Current status:

- `postgres`, `redis`, `qdrant`, and `browserless` now have native `docker-service` manifests and Go driver-backed lifecycle/status coverage.
- `vault` now has a native `docker-service` manifest and Go-native standard lifecycle/status/log handling, while the retained Vault-specific secret commands still use compatibility shims.
- `litellm`, `minio`, `neo4j`, `questdb`, and `searxng` now have native `docker-service` manifests and Go-native standard lifecycle/status/log handling.
- `ollama` and `unstructured-io` now have native `docker-service` manifests and Go-native standard lifecycle/status/log handling.
- `judge0` now has a native `compose-service` manifest and Go-native standard lifecycle/status/log handling for its multi-container stack.
- `comfyui` now has a native `docker-service` manifest and Go-native standard lifecycle/status/log handling, while workflow/model/GPU-specific commands remain in the shell compatibility surface.
- `home-assistant` now has a native `compose-service` manifest and Go-native standard lifecycle/status/log handling, while automation, backup, component, and voice commands remain in the shell compatibility surface.
- `postgis` now has a native `compose-service` manifest and Go-native standard lifecycle/status/log handling, while GIS import/export and advanced spatial analysis commands remain in the shell compatibility surface.
- `browserless` has been reduced to a thin compatibility surface centered on `status`, `logs`, `screenshot`, and `diagnostics`, while keeping Browserless-shaped `/pressure`, `/function`, and structured status support for `test-genie`.
- `claude-code` and `codex` now have native `external-cli` manifests and Go-native standard lifecycle/status handling.
- `k6`, `opencode`, and `sqlite` now have native `external-cli` manifests and Go-native standard lifecycle/status handling.
- `gemini` and `openrouter` now have native `cloud-api` manifests and Go-native status/configuration checks.
- `twilio` and `cloudflare-ai-gateway` now have native `cloud-api` manifests and Go-native credential/status checks.
- Scenario-facing env contracts are now explicit for the active non-docker resources with live repo consumers:
  - `sqlite`
  - `openrouter`
  - `home-assistant`
  - `whisper`
  - `kokoro`
  - `judge0`
  - `claude-code`
  - `codex`
  - `opencode`
- Those contracts are now resolved natively through `resource.json.environment_exports`, not by shell-era `defaults.sh` inference.
- The current active non-docker resources intentionally left without scenario env contracts are:
  - `cloudflare-ai-gateway`
  - `gemini`
  - `twilio`
  - `k6`
  - `mail-in-a-box`
  - `postgis`
- That omission is intentional for now: the repo audit did not find live scenario dependency consumers for those resources, so adding scenario-facing env exports today would be speculative contract surface rather than migration parity.
- Resource-specific non-standard commands can still fall back to legacy shell entrypoints when the native driver does not own that subcommand yet.
- Phase 5 validation now explicitly covers that the migrated core set uses the native driver path for standard commands even when a legacy `cli.sh` compatibility shim is present.
- Migrated resource `cli.sh` entrypoints now delegate standard lifecycle, status, and logs commands back to `vrooli resource`, leaving only compatibility-only custom subcommands in shell where native ownership is intentionally incomplete.
- Before native `vault` migration, the supported Vault-specific CLI surface was intentionally reduced to the commands active repo code still uses: `status`, `content add|get|remove`, and `secrets check|init|validate|export|create-template`.
- Historical admin, audit, security-monitoring, backup/restore, and migration subcommands were removed rather than carried forward, because there was no non-`vault` code usage to justify migrating them.

### Phase 6 — Legacy adapter shrinking

**Goal:** Collapse the old shell estate without forcing premature rewrites.

- [x] Convert any remaining not-yet-deprecated shell resources into explicit `legacy-adapter` form
- [x] Add deadlines/owners for each remaining adapter
- [x] Decide per adapter:
  - [x] migrate fully
  - [x] convert to blueprint
  - [x] deprecate and archive

**Deliverable:** The remaining shell-based resources are a small explicit backlog, not a sprawling default.

**Acceptance:** Every remaining shell resource has an explicit plan and no hidden status.

**Status update:** Phase 6 is now fully burned down in the active project keep-set.

- The former active shell-backed Phase 0 `keep` adapters now carry native `resource.json` manifests using canonical drivers:
  - `kokoro` -> `compose-service`
  - `mail-in-a-box` -> `compose-service`
  - `sagemath` -> `docker-service`
  - `whisper` -> `compose-service`
- `vrooli resource list` and `vrooli resource status` now show those resources as `manifest-native`.
- Repo-level tests now enforce the stronger invariant that the active keep-set has no remaining project `legacy-adapter` entries.

Current explicit Phase 6 adapter backlog:

- none

The focused Phase 6 closeout validation bundle is:

- `go test ./internal/resources ./cmd/vrooli/...`
- `go run ./cmd/vrooli resource list`
- `go run ./cmd/vrooli resource status`

Phase 6 is considered complete once the active project keep-set has no hidden `legacy-shell` entries and no remaining project `legacy-adapter` backlog.

### Phase 7 — Cleanup and contract hardening

**Goal:** Make the new model the default and remove drift.

- [x] Make blueprints, templates, and deprecation workflows part of official docs
- [x] Update contributor guidance
- [x] Remove obsolete resource framework dependencies
- [x] Narrow active resource discovery to new manifests
- [x] Move old contract docs into historical/transitional context if needed
- [x] Add maintenance checks to keep the active resource set small and validated

**Deliverable:** The new resource architecture is the stable default.

**Acceptance:** Adding a new resource naturally goes through blueprint -> template -> implementation, not ad hoc shell cloning.

**Status update:** Phase 7 is implemented as the default active resource policy and documentation baseline.

- `resources.Controller.Discover()` now exposes only manifest-backed active resources, so plain `path:resources/<name>/` directories without `resource.json` no longer appear in `vrooli resource list` / `status`.
- The active control plane surface is now limited to:
  - `manifest-native` resources
  - explicit `legacy-adapter` resources
  - deprecated resources only through `list-deprecated` / `restore`
- Official resource docs now point new work to blueprints, templates, deprecation, and manifest-backed implementation rather than shell cloning.
- Legacy shell-era framework/contract docs are now marked as historical/transitional instead of competing with the default architecture.
- Repo tests now enforce the Phase 7 invariant that active discovery contains only manifest-backed resources with `manifest-native` or `legacy-adapter` control modes.

Focused Phase 7 validation:

- `go test ./internal/resources`
- targeted `go test ./cmd/vrooli` resource command coverage
- `go run ./cmd/vrooli resource blueprint validate`
- `go run ./cmd/vrooli resource template validate`
- `go run ./cmd/vrooli resource list`
- `go run ./cmd/vrooli resource status`
- `go run ./cmd/vrooli resource list-deprecated`

Validation note: the full `go test ./cmd/vrooli/...` bundle still stalls in `TestRunScenarioStartStopRestartLifecycleCommands`, which appears unrelated to resource migration changes. The resource-focused Phase 7 slice is green.

### Phase 7.5 — Blueprint-only archival cleanup

**Goal:** Remove stale `path:resources/<name>/` implementations for blueprint-backed candidates without misclassifying them as deprecated.

- [x] Add distinct metadata for blueprint-archived resources
- [x] Implement archive / list / restore / GC commands for blueprint-backed archival
- [x] Add safety gates so active resources cannot be archive-to-blueprint candidates
- [x] Keep blueprint-archived resources out of active discovery
- [x] Document the distinction between `blueprint-archived` and `deprecated`

**Deliverable:** Blueprint-only candidates can leave `resources/` cleanly while remaining preserved through structured blueprints.

**Acceptance:** A stale implementation can be archived out of the repo after blueprint preservation, without being mislabeled as deprecated.

**Status update:** Blueprint-only archival is now implemented as a separate Go-native lifecycle.

- `.vrooli/resources/archived-blueprint-resources.json` tracks blueprint-backed archival metadata separately from deprecated resources.
- `vrooli resource archive-to-blueprint <name>` archives and removes an old implementation only when:
  - a matching blueprint exists
  - the resource is not active in root project config
  - no scenario manifests still reference it
- `vrooli resource list-blueprint-archived`, `restore-blueprint <name>`, and `archive gc-blueprints` are implemented.
- blueprint-archived resources remain outside normal `vrooli resource list` / `status` discovery and restore only into quarantined paths.
- The Phase 0 blueprint cleanup pass is complete:
  - all implemented Phase 0 `blueprint` resources have been archived out of `resources/`
  - `52` entries now appear in `vrooli resource list-blueprint-archived`
  - the remaining Phase 0 blueprint records, `node-red` and `parlant`, never had a `path:resources/<name>/` implementation to archive

Focused validation:

- `go test ./internal/resources`
- `go test ./cmd/vrooli -count=1 -run 'TestRunResource'`
- `go run ./cmd/vrooli resource blueprint validate`
- `go run ./cmd/vrooli resource list-blueprint-archived`
- `go run ./cmd/vrooli resource archive gc-blueprints`

---

## 12. Concrete checklist for the remaining active resource migration pass

This replaces the old first-triage checklist. It assumes the active `resources/` tree is already the curated keep-set.

For every current active resource:

- [ ] confirm the manifest driver/archetype is correct
- [ ] confirm portability tier is honest
- [ ] identify which standard commands are already owned by Go
- [ ] identify which commands still fall back to shell compatibility
- [ ] decide whether the remaining shell surface should be:
  - [ ] migrated into a shared driver
  - [ ] migrated into resource-local native code
  - [ ] kept as an explicit compatibility-only custom surface
  - [ ] removed because it is no longer justified
- [ ] remove duplicated shell lifecycle/status/log logic once native ownership exists
- [ ] tighten tests around native driver ownership
- [ ] ensure the resource still maps cleanly to one canonical template
- [ ] capture any repeated pattern that should improve the canonical template rather than staying bespoke

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

- Should the blueprint store be JSON-only, or allow Markdown companions for richer human notes?
- Should archive export include git metadata and diff context, or just files plus metadata JSON?
- Which current resources are definitely part of the long-term active core set?
- How much of the old `content` command surface should be driver-native vs resource-local?
- Should template generation live under `vrooli resource generate`, `vrooli resource template`, or blueprint promotion only?

### Resolved direction: single-manifest resources

Status update:

- `resource.json` is now the single manifest authority for migrated resources.
- `.vrooli/schemas/resource-definitions.json` is now generated from `resource.json`, not `config/schema.json`.
- `path:internal/resources/env/resolver.go` no longer falls back to `resource-definitions.json` for env inference.
- `config/runtime.json` and `config/schema.json` have been removed from active resources.
- canonical templates no longer scaffold `config/defaults.json` or `config/schema.json`.
- repo validation now rejects deprecated sidecar files for single-manifest resources.
- stale scenario manifests still reference removed resources such as `n8n`, `windmill`, `mailpit`, `playwright`, and `system-setup`; those are now visible cleanup work on the scenario side rather than something the resource catalog should mask.

Current repo findings support a stricter direction:

- `resource.json` is already the runtime authority for native lifecycle, driver selection, env exports, ports, install metadata, and health checks.
- `.vrooli/schemas/resource-definitions.json` remains a generated compatibility artifact for scenario authoring and IDE autocomplete, but it is now sourced from `resource.json`.
- the old env fallback and sidecar runtime/schema files have been removed from the active resource set.

The correct path is now Option B:

- `resource.json` becomes the single canonical manifest for both runtime behavior and scenario dependency authoring.
- `config/schema.json` is replaced by a manifest-native `dependency_schema` section.
- `config/runtime.json` is replaced by explicit manifest-native lifecycle and orchestration fields.
- `.vrooli/schemas/resource-definitions.json` stops being authored from `config/schema.json` and is instead generated from `resource.json`, or removed entirely if `service.schema.json` can be generated directly.
- canonical templates stop scaffolding `config/defaults.json` and `config/schema.json`.

### Proposed single-manifest contract

Each resource manifest should converge on a shape like:

```json
{
  "$schema": "../../.vrooli/schemas/resource.schema.json",
  "name": "postgres",
  "display_name": "PostgreSQL Database",
  "description": "Managed PostgreSQL service for project and scenario storage.",
  "template": "docker-service",
  "driver": "docker-service",
  "portability_tier": "full",
  "category": "storage",
  "platforms": {
    "linux": "supported",
    "macos": "supported",
    "windows": "partial"
  },
  "orchestration": {
    "startup_order": 1000,
    "startup_timeout_seconds": 60,
    "startup_time_estimate": "10-30s",
    "recovery_attempts": 5,
    "health_check_retries": 10,
    "health_check_delay_seconds": 3,
    "priority": "critical",
    "dependencies": []
  },
  "dependency_schema": {
    "type": "object",
    "properties": {
      "database": {
        "type": "string",
        "description": "Scenario-specific database name override"
      }
    },
    "additionalProperties": false,
    "examples": [
      {
        "enabled": true,
        "required": true,
        "database": "my_scenario"
      }
    ]
  },
  "ports": [],
  "health_checks": [],
  "runtime": {},
  "environment_exports": {},
  "lifecycle": {},
  "capabilities": {}
}
```

This splits the jobs cleanly:

- `dependency_schema`: what scenarios are allowed to declare under `dependencies.resources.<name>`
- `orchestration`: startup ordering and lifecycle hints previously stored in `config/runtime.json`
- `runtime`, `ports`, `health_checks`, `environment_exports`, `install`, `capabilities`: native control-plane behavior

### What moves where

#### `config/schema.json`

Current role:

- scenario dependency authoring schema
- aggregate input for `resource-definitions.json`
- accidental runtime-default source in old env fallback

Target:

- replace with `resource.json.dependency_schema`

Notes:

- many current schemas are mostly baggage and should collapse to small schema fragments or disappear entirely
- only resource-specific scenario knobs should survive
- generic dependency fields such as `enabled`, `required`, `purpose`, `startup_policy`, and `description` should remain owned by `.vrooli/schemas/resources.schema.json`

#### `config/runtime.json`

Current role:

- shell CLI `info` output metadata
- startup ordering and retry hints

Target:

- replace with `resource.json.orchestration`
- continue using `resource.json.lifecycle` for driver-executed timeouts and retries

Notes:

- `orchestration` should carry repo-level scheduling semantics
- `lifecycle` should carry driver execution semantics

#### `config/defaults.sh`

Current role:

- shell implementation defaults
- sometimes hidden source of runtime env and path values

Target:

- values that matter to native runtime move into:
  - `runtime.env`
  - `ports`
  - `install`
  - `environment_exports`
  - `orchestration`
  - `dependency_schema` defaults where they are scenario-facing
- shell-only values either:
  - move into retained shell compatibility code while migration is active, or
  - are deleted if no longer used

Notes:

- canonical templates should not include any defaults file
- compatibility-only shell code should use a clearly marked compatibility path, not `config/defaults.sh` as a canonical contract

#### `config/exports.sh`

Current role:

- shell-era scenario env export path

Target:

- fully replaced by `resource.json.environment_exports`
- deleted once the native resolver covers all live scenario consumers

#### `config/messages.sh`

Current role:

- shell CLI message text

Target:

- remove from canonical templates
- migrate any still-useful messages into Go/native command rendering or keep them only inside explicit compatibility code

#### `config/capabilities.yaml`

Current role:

- shell-era feature flags and affordance metadata

Target:

- migrate useful data into `resource.json.capabilities`
- delete the YAML files once no consumer remains

#### `config/defaults.json`

Current role:

- template-only scaffold, not used operationally

Target:

- remove from canonical templates and template validation once the single-manifest contract lands

### Generator and schema changes

To make Option B real, the repo must stop generating dependency definitions from `config/schema.json`.

Required changes:

1. Extend `.vrooli/schemas/resource.schema.json` with:
   - `dependency_schema`
   - `orchestration`
2. Extend `path:internal/resources/manifest/manifest.go` with typed structs for those sections.
3. Replace `.vrooli/schemas/build-aggregated-schemas.sh` with a manifest-native generator that:
   - scans `path:resources/*/resource.json`
   - reads `dependency_schema`
   - merges it with `.vrooli/schemas/resources.schema.json#/definitions/resourceConfig`
   - emits the dependency catalog consumed by `.vrooli/schemas/service.schema.json`
4. Keep the output filename either:
   - as `resource-definitions.json` for compatibility, but generated from `resource.json`, or
   - rename it and update `service.schema.json` accordingly

The first option is lower-risk:

```text
same filename
new source of truth
```

### Validation requirements

This migration should not be considered complete unless validation is first-class.

#### Manifest validation

Add validation for:

- malformed `dependency_schema`
- duplicate property names already owned by base `resourceConfig`
- invalid `orchestration` values
- explicit `environment_exports` collisions and missing references

#### Generated schema validation

Validate that:

- every generated dependency schema compiles
- `service.schema.json` still validates scenario manifests
- no resource with `dependency_schema` causes invalid aggregate output

#### Repo cleanup validation

Add checks so canonical resources/templates fail validation if they retain deprecated files after migration:

- `config/schema.json`
- `config/runtime.json`
- `config/defaults.json`
- `config/exports.sh`

This should be strict for canonical templates first, then rolled out to resources by migration phase.

### Cleanup sequence

The cleanup should happen in this order.

1. Add `dependency_schema` and `orchestration` to `resource.json` and schema/types.
2. Implement manifest-native dependency catalog generation.
3. Update `.vrooli/schemas/service.schema.json` to consume the manifest-native aggregate.
4. Remove legacy env fallback to `.vrooli/schemas/resource-definitions.json` from `path:internal/resources/env/resolver.go`.
5. Update resource templates to a single-manifest layout.
6. Remove `config/defaults.json` and `config/schema.json` from template validation.
7. Migrate active resources:
   - move useful `config/runtime.json` fields into `resource.json.orchestration`
   - move useful `config/schema.json` fields into `resource.json.dependency_schema`
   - migrate or delete shell config files
8. Add repo checks that reject deprecated config files for migrated resources.
9. Delete obsolete generator paths and documentation that still describe `config/schema.json` as canonical.

### Template target state

Canonical templates should converge on:

```text
templates/resources/<template>/
  template.json
  README.md
  resource.json
  docs/OPERATIONS.md
  test/smoke.json
  test/integration.json
  optional archetype assets
```

No canonical template should include:

- `config/defaults.json`
- `config/schema.json`
- any Bash file

### Documentation cleanup required

These docs are currently stale and should be updated as part of the migration:

- `path:docs/resources/interface-standards.md`
- `path:docs/resources/resource-templates.md`
- `.vrooli/schemas/README.md`
- any shell-era resource docs that still describe `defaults.sh`, `messages.sh`, or `runtime.json` as canonical contract

### Recommended migration waves

#### Wave 1: contract and generator

- extend `resource.schema.json`
- extend `manifest.go`
- implement manifest-native dependency catalog generator
- update `service.schema.json`
- keep output filename stable if possible

#### Wave 2: templates and validation

- update `path:templates/resources/*`
- update `path:internal/resources/templates.go`
- update template tests
- add validator checks for deprecated files

#### Wave 3: active resource manifests

Migrate the resources whose `config/schema.json` still has meaningful resource-specific knobs:

- `browserless`
- `codex`
- `gemini`
- `home-assistant`
- `judge0`
- `mail-in-a-box`
- `minio`
- `neo4j`
- `ollama`
- `openrouter`
- `postgis`
- `sagemath`
- `unstructured-io`

Resources whose current schema files only restate generic fields should either:

- get a very small `dependency_schema`, or
- drop the section entirely

#### Wave 4: compatibility cleanup

- remove legacy env fallback
- remove `resource-definitions.json` as a `config/schema.json` aggregate
- remove deprecated template files
- begin removing retained shell config files where resource-specific migration is complete

These are implementation questions, not blockers for starting the phased work.

---

## 16. Immediate next steps

The recommended next slice after this plan is:

1. extend `resource.schema.json` and `manifest.go` with `dependency_schema` and `orchestration`
2. replace `config/schema.json` aggregation with a manifest-native dependency catalog generator
3. update `service.schema.json` to consume the manifest-native aggregate
4. remove legacy runtime/env dependence on `resource-definitions.json`
5. update canonical templates and template validation to a zero-Bash, single-manifest shape
6. migrate active resources in waves and then delete deprecated config files

This order is intentional. The contract and generator need to be right before template cleanup and per-resource migration begin.

---

## 17. Summary

The correct path is not "port every current resource to Go."

The correct path is:

1. separate capability knowledge from maintained implementation
2. keep only validated resources active
3. introduce first-class blueprints
4. introduce first-class deprecation/archive/restore
5. standardize resource generation around a small template set
6. consolidate the surviving active resource set onto a single-manifest resource contract
7. migrate the surviving active resource set onto a Go-native driver-based control plane

If this plan is followed, Vrooli will end up with a cleaner, more honest, and more extensible resource system that is actually worth making cross-platform.
