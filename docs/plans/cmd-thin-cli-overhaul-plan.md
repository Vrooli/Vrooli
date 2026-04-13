# Thin `cmd/` Overhaul Plan

**Status:** Phase 2 complete
**Owner:** Matthew Halloran
**Scope:** Project-level Go command surface under `cmd/`, the command/application boundary under `internal/`, and the related test surface for `vrooli` and `vrooli-api`
**Out of Scope:** Scenario-internal CLIs/APIs under `scenarios/*/`, except where project-level command contracts or shared test utilities must be aligned
**Target:** A greenfield-quality command architecture where `cmd/` is a thin composition root, `internal/` owns application behavior, command metadata/help/output contracts are declared once, and all remaining legacy, compatibility, and dead code paths are removed

---

## 0. For agents picking this up later

If you are resuming this work later, read this section first.

- **What this plan is:** A full architectural cleanup and boundary-hardening plan for the project-level Go CLI after the Bash-to-Go migration became functionally complete.
- **What triggered it:** The migration succeeded in behavior, but too much parsing, help text, compatibility translation, and orchestration logic still lives in `cmd/vrooli`, which is making the code verbose, inconsistent, and difficult to evolve safely.
- **What “done” means:** Not “the current commands still pass.” The target is a professional end state with:
  - thin `cmd/` binaries
  - explicit application-layer boundaries in `internal/`
  - centralized command metadata and user-visible contracts
  - shared test seams and fixtures
  - zero legacy or compatibility dispatch in the final runtime path
  - zero dead code retained “just in case”
- **How to validate progress:** Run the validation matrix in Section 11 after each phase, not just `go test ./cmd/vrooli`.
- **How to resume work:** Start from the first unchecked item in the phased checklist. If code and plan drift, the repo is authoritative; update this plan rather than working around stale text.
- **Phase 0 completion note:** On April 13, 2026 this plan was reviewed against the current `cmd/vrooli` and `internal/` implementation and promoted from a draft proposal to the authoritative target architecture for the overhaul work.

---

## 1. Why this overhaul is necessary

The Go migration moved behavior into the Go module, but not yet enough architecture.

Today, `cmd/vrooli` still acts as all of the following at once:

- binary entry point
- command registry
- argument parser
- usage/help text owner
- command execution coordinator
- compatibility router
- subprocess translation layer
- response-format policy layer
- partial application layer

That is the opposite of the intended end state. The command binary should be the thinnest layer in the system.

The current shape has four concrete costs:

1. **Responsibility drift**
   - Command semantics are split across `cmd/vrooli`, `internal/cli/*`, and `internal/*` domain packages.
   - Similar commands are implemented with different local patterns, increasing maintenance cost.

2. **Contract drift**
   - Usage text, error strings, JSON envelope keys, and human output messages are hard-coded in many places.
   - Tests often assert those literals directly, which means contract changes require broad, noisy edits.

3. **Transitional architecture becoming permanent**
   - Compatibility and subprocess bridges that were acceptable during migration are still mixed into the normal command path.
   - This makes it hard to see the real ownership boundary for a feature.

4. **Test drag**
   - There is solid coverage, but too much of it is concentrated in large command-package integration tests rather than layered test seams.
   - Shared fixtures already exist in `packages/testkit-go`, but the command surface still carries too much custom test setup.

---

## 2. Current state summary

This section records the current state observed during the April 13, 2026 review.

### What is already good

- The project-level Go path is real and in use.
- The main project-level command suite is native and no longer depends on the deleted Bash bootstrap path.
- `internal/` already contains meaningful domain packages:
  - `internal/project`
  - `internal/orchestrator`
  - `internal/resources`
  - `internal/setup`
  - `internal/maintenance`
  - `internal/cli/*`
- The test surface is broad and currently green for:
  - `go test ./cmd/vrooli/... ./internal/...`

### What is not yet good enough

- `cmd/vrooli` is still too large and too responsible.
- Command plumbing is duplicated through multiple descriptor/action abstractions.
- Help/usage text is spread through many files.
- User-facing strings and JSON envelopes are inconsistently owned.
- Some commands still depend on compatibility shims and subprocess translation that live in `cmd/`.
- Tests are good functionally, but not yet organized around professional seams and declarative contracts.

### Current hotspots

At the time of review:

- `cmd/vrooli` is about `12.5k` LOC in aggregate.
- Some of the largest command-layer files are:
  - `cmd/vrooli/scenario_actions.go`
  - `cmd/vrooli/package_commands.go`
  - `cmd/vrooli/scenario_template.go`
  - `cmd/vrooli/scenario_logs.go`
  - `cmd/vrooli/resource_actions.go`
  - `cmd/vrooli/scenario_lifecycle_commands.go`
  - `cmd/vrooli/resource_commands.go`
- The current command layer includes overlapping abstractions:
  - `commandDescriptor`
  - `appSubcommandDescriptor`
  - `resourceSubcommandDescriptor`
  - `commandAction`
  - `boundCommandAction`
  - `run*WithApp` wrappers
- Inline `Usage:` / `usage:` strings and ad hoc human output messages are spread across `cmd/vrooli`, `internal/cli/*`, `internal/project`, `internal/setup`, and resource control/rendering layers.

### Representative examples of lingering transitional architecture

- `cmd/vrooli/resource_commands.go`
  - still contains fallback behavior for legacy resource invocation
- `cmd/vrooli/scenario_external_commands.go`
  - translates requirements, completeness, and test-genie subprocess behavior in the command layer
- `cmd/vrooli/cleanup_command.go`
  - re-expresses `orphans` and `locks` behavior as a synthetic command wrapper in `cmd/`
- `cmd/vrooli/info_command.go`
  - owns command help text and output contract logic directly in the binary package
- `internal/setup/setup.go`
  - carries multiple responsibilities and should be decomposed into smaller use-case modules

---

## 3. Problem statement

The main architectural problem is not “there is too much code in `cmd/vrooli`.” That is only a symptom.

The actual problem is:

> The project-level command system does not yet have a single, disciplined command architecture with clean separation between command declaration, application use cases, domain services, compatibility adapters, and presentation contracts.

Because of that, the codebase currently experiences:

- repeated parse-run-render loops
- repeated command registration patterns
- repeated help/usage declarations
- repeated output envelope choices
- repeated fallback-routing choices
- blurred responsibility between command package and application/domain packages

---

## 4. Target end state

This section is the intended final architecture. Every phase in the checklist must move toward this shape.

### 4.1 Core architectural rule

`cmd/` must be a composition root only.

That means:

- `cmd/vrooli`
  - startup
  - global flag capture
  - buildinfo/staleness wiring
  - logger construction
  - dependency graph construction
  - command runner invocation
  - exit code mapping
- `cmd/vrooli-api`
  - same principle for the API binary

It must **not** own:

- domain use cases
- command-family parsing rules
- usage/help text definitions
- response rendering logic
- compatibility routing decisions
- subprocess translation rules
- feature-specific business logic

### 4.2 Internal layering

The project-level command stack should resolve into layers like this:

1. **Composition**
   - `cmd/vrooli`
   - `cmd/vrooli-api`

2. **Command system**
   - command tree definitions
   - parse + validate + usage/help metadata
   - runner orchestration
   - output selection policy

3. **Application use cases**
   - scenario start/stop/status/list
   - resource list/status/control
   - project status/doctor/stop/lifecycle
   - package list/validate/refresh/audit
   - contract validate/show/resolve/match
   - info/context loading

4. **Domain/infrastructure services**
   - orchestrator
   - lifecycle
   - resources
   - maintenance
   - runtime
   - buildinfo
   - ports/process/network

5. **External adapters**
   - shell/process invocation
   - filesystem
   - HTTP
   - remaining third-party tool invocation

### 4.3 Declarative command ownership

Each command family must have one authoritative home for:

- subcommand names
- summaries
- usage/help text
- supported flags/options
- JSON/human output contract
- parsing rules
- validation rules

There must not be duplicate command declarations across multiple files.

### 4.4 Contract ownership

The following contracts must be declared once and reused:

- CLI help/usage strings
- JSON envelope keys
- stable human-facing status messages
- repo-contract paths and filenames
- scenario/resource/package command names
- error codes and categories

Tests should import or derive from the same declarations instead of repeating raw literals.

### 4.5 Legacy and compatibility target

The final runtime path must contain:

- no legacy fallbacks
- no compatibility dispatch
- no dead shims
- no obsolete wrapper commands retained for internal convenience

If a compatibility behavior is still required temporarily during implementation, it must:

- live in an explicitly temporary package or file
- be called out in this plan
- have a checklist item for deletion

The final state is not “legacy but isolated.” The final state is **no legacy path at all**.

### 4.6 Test target

The test surface must be layered:

- unit tests for parsing and rendering
- use-case tests with fake interfaces
- focused command integration tests
- a smaller number of binary-level smoke tests

Shared fixtures should prefer `packages/testkit-go` and `packages/testkit-go/vrooli` over bespoke per-file setup where practical.

### 4.7 Authority of this plan

Until this overhaul is complete, this document is the authoritative architecture target for project-level CLI cleanup.

That means:

- new CLI cleanup work must move the codebase toward this document, not away from it
- if implementation reveals the plan is wrong or incomplete, update this document before normalizing the code around a competing architecture
- if code and plan disagree temporarily during a phase, treat the mismatch as intentional only when the relevant checklist item explicitly allows it
- do not preserve existing command-package structure solely because it already exists

### 4.8 Dependency rules

These rules freeze the intended package boundaries for the overhaul. Exact package names may shift slightly, but the dependency directions must not.

Allowed dependency flow:

1. `cmd/*`
   - may depend on:
     - `internal/bootstrap`
     - `internal/buildinfo`
     - `internal/config`
     - `internal/logx`
     - `internal/cli/commandtree`
     - the top-level internal command runner package introduced during implementation
   - must not depend directly on:
     - feature-specific command-family packages for ad hoc execution
     - domain services for feature behavior
     - compatibility helpers

2. `internal/cli/*`
   - may depend on:
     - `internal/cli/commandtree`
     - `internal/cliout`
     - `internal/app/*`
     - stable contract and error packages
   - must not depend on:
     - `cmd/*`
     - direct shell/process invocation except through injected app or adapter seams

3. `internal/app/*`
   - may depend on:
     - domain and infrastructure packages such as `internal/project`, `internal/resources`, `internal/orchestrator`, `internal/lifecycle`, `internal/maintenance`, `internal/runtime`, `internal/ports`, and `internal/process`
     - external adapters behind interfaces
   - must not depend on:
     - `cmd/*`
     - human-output rendering packages except where returning stable contract types shared with CLI renderers

4. Domain and infrastructure packages under `internal/`
   - must not depend on `cmd/*`
   - must not depend on feature-specific CLI packages
   - should not know about command-line parsing, help text, or presentation policy

5. Temporary compatibility packages, if any are introduced
   - must live under explicitly temporary locations
   - must only be depended on by the phase currently deleting them
   - must not become stable dependencies of `cmd/*`, `internal/cli/*`, or `internal/app/*`

Architectural litmus test:

- if a change adds command parsing, help text, fallback routing, or human-output decisions to `cmd/*`, it violates the target architecture
- if a change adds feature orchestration to `internal/cli/*` that belongs in a reusable use case, it violates the target architecture
- if a change leaves a compatibility path in the runtime without a named deletion phase, it violates the target architecture

### 4.9 Temporary command surfaces and deletion intent

The following surfaces are intentionally temporary. They exist only because the migration is not yet fully consolidated.

| Surface | Current location | Why it is temporary | Owner | Delete in |
| --- | --- | --- | --- | --- |
| Duplicated top-level/scenario command registry types and maps | `cmd/vrooli/command_registry.go`, `cmd/vrooli/command_actions.go`, `cmd/vrooli/command_sets.go` | Transitional command framework duplicated across command families | Matthew Halloran | Phase 1 |
| Top-level parse/run/render helpers in the binary package | `cmd/vrooli/top_level_actions.go`, `cmd/vrooli/top_level_native_commands.go` | Command-family behavior still lives in `cmd/` instead of internal CLI packages | Matthew Halloran | Phase 2 |
| Synthetic `cleanup` wrapper command | `cmd/vrooli/cleanup_command.go` | Convenience wrapper around `orphans` and `locks`, implemented outside the normal declarative command model | Matthew Halloran | Phase 2 |
| Binary-owned `info` command contract and rendering | `cmd/vrooli/info_command.go` | Context-info command still owns help/output logic directly in `cmd/` | Matthew Halloran | Phase 2 |
| Scenario command-family parsing and orchestration in `cmd/` | `cmd/vrooli/scenario_actions.go`, `cmd/vrooli/scenario_commands.go`, `cmd/vrooli/scenario_lifecycle_commands.go`, `cmd/vrooli/scenario_logs.go`, `cmd/vrooli/scenario_template.go` | Command family not yet extracted into internal CLI and app layers | Matthew Halloran | Phase 3 |
| Scenario subprocess translation in the command layer | `cmd/vrooli/scenario_external_commands.go` | Legacy-style subprocess translation still sits in `cmd/` | Matthew Halloran | Phase 3 and Phase 8 |
| Resource command-family parsing and fallback dispatch in `cmd/` | `cmd/vrooli/resource_actions.go`, `cmd/vrooli/resource_commands.go` | Command family not yet extracted and still contains fallback behavior | Matthew Halloran | Phase 4 and Phase 8 |
| Package command-family parsing and orchestration in `cmd/` | `cmd/vrooli/package_commands.go`, related package command files in `cmd/vrooli/` | Package governance commands have not yet moved to the unified internal command architecture | Matthew Halloran | Phase 5 |

### 4.10 Currently required compatibility behavior to track until deletion

The following behaviors exist today and cannot be treated as permanent architecture. They are recorded here so they can be deleted deliberately rather than forgotten.

| Behavior | Current implementation | Why it still exists today | Replacement target | Delete in |
| --- | --- | --- | --- | --- |
| Unknown `vrooli resource <subcommand>` fallback dispatch into `resources.Controller.Run` | `cmd/vrooli/resource_commands.go` via `runLegacyResourceInvocationWithApp` | Not every resource behavior has been consolidated into one declarative command path yet | First-class native resource command declarations in internal CLI and app layers | Phase 4 and Phase 8 |
| Scenario requirements/completeness/test-genie subprocess argument translation in the command layer | `cmd/vrooli/scenario_external_commands.go` | Transitional bridge to still-external scenario utilities | Native app use cases or explicitly bounded adapter packages below the command layer | Phase 3 and Phase 8 |
| Synthetic `cleanup orphans` -> `orphans kill` and `cleanup locks` -> `locks clean` mapping | `cmd/vrooli/cleanup_command.go` | Transitional convenience command preserved during migration | Remove `cleanup` entirely or re-declare it through the same unified command metadata system | Phase 2 |
| Binary-local ownership of info manifest defaults and rendering contract | `cmd/vrooli/info_command.go` | Context-info behavior was migrated functionally but not architecturally | `internal/app/contextinfo` plus declarative internal CLI command definitions | Phase 2 |
| Rootless command exceptions encoded through binary-local descriptor flags | `cmd/vrooli/app.go`, `cmd/vrooli/command_registry.go` | Root-policy decisions are still coupled to the old command descriptor system | Root requirement policy encoded in the unified command tree metadata | Phase 1 |

---

## 5. Design principles for implementation

These principles are mandatory while executing the checklist.

### 5.1 Thin command binaries

Every refactor should make `cmd/vrooli` smaller, not move logic around inside it.

### 5.2 One command framework

There should be one command execution pattern for the CLI surface, not separate bespoke patterns for top-level, scenario, resource, and package commands.

### 5.3 No string drift

If a string is part of a stable contract, it must have one declarative home.

### 5.4 Compatibility is temporary only

No permanent “adapter forever” posture. Anything temporary must be tracked until deletion.

### 5.5 Shared seams over test copy-paste

If a test fixture pattern appears more than twice, consider promoting it into `testkit-go` or shared test helpers.

### 5.6 Prefer deletion over preservation

If a wrapper, shim, alias, or compatibility branch no longer serves the target architecture, delete it instead of preserving it “for safety.”

---

## 6. Proposed target package layout

The exact names may vary slightly, but the responsibilities must match.

```text
/cmd/
  vrooli/
    main.go
    app.go                # composition root only
  vrooli-api/
    main.go

/internal/
  app/
    project/
    scenario/
    resource/
    package/
    contract/
    contextinfo/
  cli/
    commandtree/
    topcli/
    scenariocli/
    resourcecli/
    packagecli/
    contractcli/
    projectcli/
  cliout/
  compat/                # temporary only during implementation, must be empty or deleted at end
  bootstrap/
  buildinfo/
  control/
  lifecycle/
  maintenance/
  network/
  orchestrator/
  packagegov/
  ports/
  process/
  project/
  resources/
  runtime/
  scenario/
  secrets/
  setup/
  shell/
```

Notes:

- `internal/app/*` owns application use cases.
- `internal/cli/commandtree` owns shared command metadata/parsing/help execution machinery.
- `internal/cli/*cli` packages own command-family declarations and response mapping, not `cmd/vrooli`.
- Existing domain packages like `internal/project`, `internal/resources`, and `internal/orchestrator` remain, but should stop carrying command-presentation concerns.
- A temporary `internal/compat/` package may be introduced during the transition, but it must be empty or removed by the end.

---

## 7. Workstreams

This program should be executed as coordinated workstreams rather than random file cleanup.

### Workstream A: Command framework consolidation

Goal:
- Replace the current duplicated command descriptor/action/set plumbing with one reusable command system.

Primary files currently affected:
- `cmd/vrooli/command_registry.go`
- `cmd/vrooli/command_actions.go`
- `cmd/vrooli/command_sets.go`
- `cmd/vrooli/app_handlers.go`

### Workstream B: Command-family extraction out of `cmd/vrooli`

Goal:
- Move scenario/resource/package/top-level/contract command-family declarations, parsing, help, and render orchestration into `internal/cli/*`.

Primary files currently affected:
- `cmd/vrooli/scenario_actions.go`
- `cmd/vrooli/scenario_commands.go`
- `cmd/vrooli/scenario_lifecycle_commands.go`
- `cmd/vrooli/scenario_external_commands.go`
- `cmd/vrooli/resource_actions.go`
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/package_commands.go`
- `cmd/vrooli/top_level_actions.go`
- `cmd/vrooli/top_level_native_commands.go`
- `cmd/vrooli/contract_commands.go`
- `cmd/vrooli/info_command.go`

### Workstream C: Application-service formalization

Goal:
- Create explicit application-layer use cases so command handlers are thin request adapters rather than direct coordinators of controllers/services.

Primary packages affected:
- `internal/project`
- `internal/orchestrator`
- `internal/resources`
- `internal/setup`
- new `internal/app/*`

### Workstream D: Contract centralization

Goal:
- Centralize help text, stable output strings, JSON envelope keys, error identifiers, and repo-contract paths.

Primary files affected:
- `cmd/vrooli/command_help.go`
- `cmd/vrooli/command_errors.go`
- `internal/cli/*`
- `internal/project`
- `internal/resources/control`
- `internal/setup`

### Workstream E: Legacy/compatibility elimination

Goal:
- Remove every remaining runtime legacy path and dead compatibility shim.

Primary files affected:
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/scenario_external_commands.go`
- any temporary `internal/compat/*`
- any dead wrapper helpers uncovered during refactor

### Workstream F: Test architecture cleanup

Goal:
- Reduce monolithic command-package tests, standardize fixtures, and align tests to declarative contracts.

Primary files affected:
- `cmd/vrooli/main_test.go`
- `cmd/vrooli/test_helpers_test.go`
- command-family test files
- `packages/testkit-go/vrooli`

---

## 8. Phased checklist

The phases below are intentionally strict. Do not skip deletion phases.

## Phase 0: Freeze the target architecture

Objective:
- Make the end state explicit before changing code.

Checklist:
- [x] Confirm this plan is the current authoritative architecture target for project-level CLI cleanup.
- [x] Add a short dependency rule section to this plan if package moves during implementation require clarification.
- [x] Identify any command surface that is intentionally temporary and mark it with an owner plus deletion phase.
- [x] Record any currently required compatibility behavior that cannot be deleted immediately.

Validation:
- [x] Plan reviewed against current repo state.
- [x] No implementation starts without a clear target package map and deletion intent.

Definition of done:
- The future boundary model is explicit enough that contributors can classify any code as composition, command, app, domain, adapter, or dead code.

## Phase 1: Build a unified command framework

Objective:
- Replace duplicated command registry and action abstractions with one command system.

Checklist:
- [x] Introduce a single shared command-spec abstraction under `internal/cli/commandtree`.
- [x] Support:
  - [x] name
  - [x] aliases
  - [x] summary
  - [x] hidden/internal visibility
  - [x] root requirement policy
  - [x] parse/validate
  - [x] execute
  - [x] render
  - [x] usage/help metadata
- [x] Move generic command execution helpers out of `cmd/vrooli`.
- [x] Remove duplicate descriptor types and execution helpers from `cmd/vrooli`.
- [x] Keep exit-code and top-level process concerns in `cmd/vrooli` only.

Validation:
- [x] Unit tests for the new command framework.
- [x] Existing command-family tests still pass after adapter migration.
- [x] `go test ./cmd/vrooli/... ./internal/...`

Definition of done:
- There is exactly one command execution pattern for the project-level CLI.

## Phase 2: Extract top-level command declarations from `cmd/vrooli`

Objective:
- Move top-level command declaration/parsing/help/render orchestration into internal command packages.

Checklist:
- [x] Extract `status`, `doctor`, `stop`, `orphans`, `locks`, `diagnose-port`, `setup`, `develop`, `build`, `deploy`, `clean`, `backup`, `restore`, `info`, `cleanup`, and `contract` command declarations into `internal/cli/*`.
- [x] Move top-level usage/help strings into declarative command metadata.
- [x] Remove parse/render helpers from:
  - [x] `cmd/vrooli/top_level_actions.go`
  - [x] `cmd/vrooli/top_level_native_commands.go`
  - [x] `cmd/vrooli/cleanup_command.go`
  - [x] `cmd/vrooli/info_command.go`
- [x] Convert `cleanup` from an ad hoc wrapper into a normal declarative command or remove it if redundant with first-class maintenance commands.
- [x] Ensure top-level help is generated from the same command tree metadata used for dispatch.

Validation:
- [x] Focused tests for top-level commands and help output.
- [x] Existing `status`, `doctor`, `stop`, `cleanup`, and `info` command tests updated and green.

Definition of done:
- `cmd/vrooli` no longer owns top-level command-family behavior beyond composition.

## Phase 3: Extract scenario command family from `cmd/vrooli`

Objective:
- Move scenario command semantics out of `cmd/vrooli` and into dedicated internal command/application packages.

Checklist:
- [x] Extract scenario command declarations and parsers into `internal/cli/scenariocli`.
- [x] Extract scenario application use cases into `internal/app/scenario` or equivalent.
- [x] Move scenario-specific help text to declarative metadata.
- [x] Move scenario output contract mapping out of `cmd/vrooli`.
- [x] Refactor:
  - [x] `scenario list`
  - [x] `scenario info`
  - [x] `scenario status`
  - [x] `scenario start`
  - [x] `scenario stop`
  - [x] `scenario restart`
  - [x] `scenario start-all`
  - [x] `scenario stop-all`
  - [x] `scenario setup`
  - [x] `scenario test`
  - [x] `scenario port`
  - [x] `scenario open`
  - [x] `scenario logs`
  - [x] `scenario template`
  - [x] `scenario generate`
  - [x] `scenario requirements`
  - [x] `scenario completeness`
  - [x] `scenario ui-smoke`
  - [x] `scenario heal-from-sandbox`
- [x] Eliminate scenario command-specific wrappers like `runScenario*CommandWithApp` where the framework can bind directly.

Validation:
- [x] Unit tests for scenario parsers and renderers.
- [x] Focused integration tests for scenario command family.
- [x] `go test ./cmd/vrooli/... ./internal/...`

Definition of done:
- `cmd/vrooli` does not contain scenario command request types, parse functions, or scenario-specific dispatch glue.

## Phase 4: Extract resource command family from `cmd/vrooli`

Objective:
- Move resource command declarations and orchestration out of the binary package and remove legacy fallback behavior.

Checklist:
- [x] Extract resource command declarations/parsers/help into `internal/cli/resourcecli`.
- [x] Introduce explicit resource application use cases in `internal/app/resource` or equivalent.
- [x] Move enable/disable/install/start/stop/list/status/info/deprecate/archive/restore flows behind app services.
- [x] Remove `runLegacyResourceInvocationWithApp`.
- [x] Remove any “unknown subcommand means fallback” resource behavior from the final runtime path.
- [x] Ensure resource archive/blueprint/template subcommands are first-class declarative commands, not local wrapper indirections.
- [x] Centralize resource command usage/help strings.

Validation:
- [x] Resource command parser/render tests.
- [x] Resource command integration tests.
- [x] Resource domain tests remain green.

Definition of done:
- Resource command execution has no runtime legacy fallback path.

## Phase 5: Extract package and contract command families

Objective:
- Finish the same architectural treatment for package governance and repo contract surfaces.

Checklist:
- [x] Move package command declarations/help/parsing to internal command packages.
- [x] Introduce package application use cases for list/info/dependents/validate/build/generate/refresh/audit.
- [x] Reduce direct shell/process coordination inside package commands by isolating it behind app services or explicit adapters.
- [x] Move contract command declarations/help/parsing to internal command packages.
- [x] Ensure contract validation/show/resolve/match use the same command framework and output contract conventions.

Validation:
- [x] Package command tests green.
- [x] Contract command tests green.
- [x] `go test ./cmd/vrooli/... ./internal/...`

Definition of done:
- Package and contract commands follow the same architecture as top-level, scenario, and resource commands.

## Phase 6: Formalize application use cases and slim domain controllers

Objective:
- Stop using `cmd` or command packages as mini application layers.

Checklist:
- [x] Introduce explicit application-layer packages for project/scenario/resource/package/contract/context info use cases.
- [x] Move multi-step orchestration logic from command handlers into app services.
- [x] Keep domain controllers focused:
  - [x] `internal/project` for project domain orchestration
  - [x] `internal/orchestrator` for scenario orchestration primitives
  - [x] `internal/resources` for resource domain orchestration
  - [x] `internal/setup` for setup/develop primitives
- [x] Ensure app services depend on interfaces, not concrete command-layer types.
- [x] Introduce narrow DTOs/requests/responses where needed so command packages do not depend on internal domain structs arbitrarily.

Validation:
- [x] Use-case tests with fakes for app services.
- [x] Existing domain tests remain green.

Definition of done:
- Command packages adapt requests into app services; they do not coordinate core behavior themselves.

## Phase 7: Centralize contracts and eliminate string drift

Objective:
- Remove scattered string ownership and duplicated contracts.

Checklist:
- [x] Centralize command usage/help text with command metadata.
- [x] Centralize error codes/categories and stable hints where part of the CLI contract.
- [x] Centralize JSON envelope conventions for success/data/list/report responses.
- [x] Centralize repo-contract-related filenames and path keys where still duplicated.
- [x] Replace hard-coded test literals with shared declarations where appropriate.
- [x] Review human-facing output in:
  - [x] `internal/cli/projectcli`
  - [x] `internal/cli/resourcecli`
  - [x] `internal/cli/scenariocli`
  - [x] `internal/setup`
  - [x] `internal/project`
  - [x] `internal/resources/control`

Validation:
- [x] Contract-focused tests for usage/help output.
- [x] JSON contract tests for representative commands.
- [x] Golden tests where stable text is intentional.

Definition of done:
- Stable command and output contracts are declared once and reused by code and tests.

## Phase 8: Eliminate all legacy, compatibility, and dead code

Objective:
- Delete the migration leftovers rather than preserving them in cleaned-up form.

Checklist:
- [x] Delete any remaining compatibility wrappers in `cmd/vrooli`.
- [x] Delete any remaining fallback routing based on unknown commands/subcommands.
- [x] Delete any temporary `internal/compat/*` packages created during the migration, or reduce them to zero runtime usage and remove them.
- [x] Delete dead request/response aliases in `cmd/vrooli`.
- [x] Delete unused helper functions that only existed for transitional glue.
- [x] Delete dead tests that only validate removed compatibility behavior.
- [x] Remove comments that describe obsolete migration paths as if they are still active.

Validation:
- [x] Search confirms no remaining runtime legacy markers in project-level CLI paths.
- [x] Search confirms no remaining dead wrapper functions in `cmd/vrooli`.
- [x] Full command suite remains green.

Definition of done:
- The final runtime path has no compatibility mode, no fallback shim, and no dead glue retained.

## Phase 9: Rebuild the test architecture around professional seams

Objective:
- Make the test suite match the cleaned architecture.

Checklist:
- [x] Split `cmd/vrooli/main_test.go` into command-family or concern-focused suites.
- [x] Promote common repo/manifest/fixture helpers into `packages/testkit-go/vrooli` where they are generally useful.
- [x] Add parser-level tests for command metadata and request decoding.
- [x] Add renderer-level tests for human/JSON contract outputs.
- [x] Add app-service tests using narrow fake interfaces.
- [x] Keep a smaller binary-level integration surface in `cmd/vrooli`.
- [x] Replace direct literal assertions with shared contract helpers where practical.

Validation:
- [x] `go test ./cmd/vrooli/... ./internal/...`
- [ ] `go test ./...`
  Note: blocked in this environment because `data/resources/btcpay/postgres` is not readable by the current user, so the Go tool cannot traverse `./...`.

Definition of done:
- The bulk of command correctness is validated below the binary package, with focused integration tests above it.

## Phase 10: Final polish and architectural verification

Objective:
- Verify the codebase now reflects the intended end state and not just passing tests.

Checklist:
- [x] Review `cmd/vrooli` and confirm it is composition-root thin.
- [x] Review `cmd/vrooli-api` for the same principle.
- [x] Review package boundaries for forbidden dependencies from domain/app layers back into `cmd`.
- [x] Review all command families for a single declarative metadata source.
- [x] Review app services for coherent ownership and interface seams.
- [x] Review tests for excessive literal duplication or bespoke setup that should live in shared testkit.
- [x] Update this plan with completion notes and any post-overhaul follow-up items.

Validation:
- [ ] Full validation matrix green.
  Note: `go test ./...` is still blocked in this environment because `data/resources/btcpay/postgres` is unreadable to the current user, so the Go tool cannot traverse the full repo tree.
- [x] Manual architecture audit complete.

Definition of done:
- The system looks intentionally designed, not historically accreted.

Completion notes:
- Deleted residual alias and wrapper files from `cmd/vrooli`: `project_commands.go`, `command_sets.go`, and `scenario_runtime_types.go`.
- Deleted unused scenario lifecycle helper functions that no longer participated in runtime dispatch.
- Replaced `cmd`-local commandtree wrapper helpers with direct `internal/cli/commandtree` usage in remaining command-family roots.
- Confirmed `cmd/vrooli` now primarily contains:
  composition root bootstrap in `main.go` and `app.go`
  command registration and binding in `command_registry.go`, `command_bindings.go`, and narrow family root adapters
  minimal runtime-only subprocess helpers where the binary must own process launching or OS integration
  tests
- Confirmed `cmd/vrooli-api` is similarly thin: startup/bootstrap plus HTTP handler shims that delegate into `internal/api`, with no application logic owned by the binary package.
- Confirmed no `internal/*` runtime package depends on `cmd/vrooli`; remaining string references to `cmd/vrooli` are build-target and fingerprint metadata, not package imports.

---

## 9. Specific deletion targets

These are not guaranteed to survive unchanged. They are called out because they are likely transitional or redundant today and should be challenged aggressively.

Candidate deletion or collapse targets:

- `cmd/vrooli/command_registry.go`
- `cmd/vrooli/command_actions.go`
- `cmd/vrooli/command_sets.go`
- `cmd/vrooli/app_handlers.go`
- `cmd/vrooli/top_level_actions.go`
- `cmd/vrooli/top_level_native_commands.go`
- `cmd/vrooli/cleanup_command.go`
- `cmd/vrooli/scenario_actions.go`
- `cmd/vrooli/scenario_commands.go`
- `cmd/vrooli/scenario_lifecycle_commands.go`
- `cmd/vrooli/scenario_external_commands.go`
- `cmd/vrooli/resource_actions.go`
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/package_commands.go`
- `cmd/vrooli/info_command.go`
- ad hoc local wrapper helpers like `run*CommandWithApp` where direct framework binding becomes possible

Targeted runtime behavior to delete:

- unknown resource command fallback to legacy invocation
- command-layer subprocess translation as a permanent architectural pattern
- duplicated help text ownership
- duplicated parse-run-render boilerplate
- any compatibility path that exists only because the migration was incremental

---

## 10. Risks and controls

### Risk: Large refactor causes behavioral drift

Control:
- phase by command family
- preserve command contracts with explicit tests
- use golden or contract tests for stable outputs

### Risk: “Move code, same architecture” refactor

Control:
- reject changes that only relocate files without removing responsibility from `cmd/`
- require deletion of old glue in the same phase that introduces the replacement

### Risk: Temporary compatibility becomes permanent again

Control:
- all temporary compatibility code must have an explicit deletion phase
- no compatibility package may remain in active runtime use at program end

### Risk: Tests become even more fragmented

Control:
- organize tests by layer
- promote common fixtures into `testkit-go`
- reduce giant binary-package test files as part of the work, not afterward

---

## 11. Validation matrix

Run these repeatedly throughout implementation.

### Baseline validation

- [ ] `go test ./cmd/vrooli/... ./internal/...`
- [ ] `go test ./...`

### Focused CLI validation

- [ ] `vrooli --help`
- [ ] `vrooli --version`
- [ ] `vrooli info`
- [ ] `vrooli status --json`
- [ ] `vrooli doctor --json`
- [ ] `vrooli stop --json`
- [ ] `vrooli cleanup --help`
- [ ] `vrooli contract validate --json`
- [ ] `vrooli package list --json`
- [ ] `vrooli resource list --json`
- [ ] `vrooli resource status --json`
- [ ] `vrooli scenario list --json`
- [ ] `vrooli scenario status --json`
- [ ] `vrooli scenario start-all --json`
- [ ] `vrooli scenario stop-all --json`

### Behavioral validation

- [ ] Top-level lifecycle help does not accidentally execute lifecycle phases.
- [ ] JSON output remains machine-readable and consistent across command families.
- [ ] Human help output is generated from metadata, not hard-coded per command.
- [ ] No command requires a compatibility shim to dispatch correctly.

### Architecture validation

- [ ] `cmd/vrooli` contains only composition-root concerns.
- [ ] No `internal/*` package depends on `cmd/vrooli`.
- [ ] No runtime legacy/compatibility dispatch remains.
- [ ] Shared testkit utilities cover common fixture patterns.

### Optional higher-level validation

- [ ] `make validate`

If `make validate` is not green due to unrelated tracks, record the precise blocker in this plan rather than silently ignoring it.

---

## 12. Definition of done

This effort is complete only when all of the following are true:

- [ ] `cmd/vrooli` is thin and only composes dependencies plus invokes a shared internal command runner.
- [ ] Command declarations, help text, and parsing rules live in internal command packages, not in the binary package.
- [ ] Explicit application use-case packages own command orchestration behavior.
- [ ] Domain packages are focused on domain responsibilities, not command presentation.
- [ ] Stable strings, error identifiers, JSON envelopes, and contract paths are centrally declared.
- [ ] Tests use shared fixtures and utilities where practical, especially via `packages/testkit-go`.
- [ ] Large command-package test files are decomposed into professional seams.
- [ ] No runtime legacy path remains.
- [ ] No compatibility fallback remains.
- [ ] No dead wrapper code remains.
- [ ] The full validation matrix is green.

If any of those are false, the effort is not done.

---

## 13. Immediate next step

The recommended first implementation slice is:

1. Build `internal/cli/commandtree` as the single command framework.
2. Migrate top-level command declaration/help/dispatch to that framework.
3. Delete the old registry/action/set plumbing from `cmd/vrooli`.

That slice sets the pattern for every later command family and prevents more cleanup work from being layered onto the current command architecture.
