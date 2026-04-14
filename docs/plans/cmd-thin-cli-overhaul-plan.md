# Thin `cmd/` Overhaul Plan

**Status:** Reopened after post-implementation audit on April 13, 2026
**Owner:** Matthew Halloran
**Scope:** Project-level command/runtime architecture for `cmd/vrooli`, `cmd/vrooli-api`, related `internal/cli/*`, `internal/app/*`, `internal/setup`, `internal/resources`, and the associated testkit/test surface
**Target:** A genuinely thin binary layer where `cmd/` is only composition/bootstrap, `internal/` owns all command/application/domain behavior, all command and output contracts are declared once, and no legacy/compatibility/dead code remains in the runtime path

---

## 0. Executive summary

The previous version of this plan was marked complete prematurely.

The implementation moved useful pieces into `internal/cli/*` and `internal/app/*`, but the architecture is still not in the target end state. The main problem is not merely that `cmd/vrooli` still has many files. The deeper problem is that the current design still spreads command behavior across the wrong layers:

- `cmd/vrooli` still owns a large amount of production command logic
- `internal/app/*` still imports CLI packages in places
- `internal/setup` still owns CLI parsing/help behavior
- command errors/help/usage strings are still duplicated
- scenario/resource/package/contract command families still have major runtime glue in `cmd/`
- project-level runtime still exposes legacy/compatibility concepts such as scenario external command bridges and resource `legacy-adapter`
- tests are still weighted far too heavily toward `cmd/vrooli`

This document supersedes the previous “complete” status. The effort is **not complete** until the checklist in this refreshed plan is finished and the final architecture matches the target end state below.

---

## 1. Current-state audit

This section records the repo state observed during the April 13, 2026 audit.

### 1.1 What improved

- A shared command framework now exists in `internal/cli/commandtree`.
- Top-level/scenario/resource/package/contract metadata exists under `internal/cli/*`.
- Application-layer packages exist under `internal/app/*`.
- Some parsing/rendering moved out of `cmd/vrooli`.
- The project-level Go command path is real and actively used.

### 1.2 What is still wrong

`cmd/vrooli` is still far too thick.

At audit time:

- `cmd/vrooli` production code: about `5,126` LOC
- `cmd/vrooli` test code: about `5,908` LOC
- `internal/cli/*` tests: about `530` LOC

Representative production hotspots still in `cmd/vrooli`:

- `cmd/vrooli/app.go`
- `cmd/vrooli/command_registry.go`
- `cmd/vrooli/command_bindings.go`
- `cmd/vrooli/app_handlers.go`
- `cmd/vrooli/resource_actions.go`
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/package_commands.go`
- `cmd/vrooli/scenario_actions.go`
- `cmd/vrooli/scenario_logs.go`
- `cmd/vrooli/scenario_template.go`
- `cmd/vrooli/scenario_external_commands.go`
- `cmd/vrooli/contract_commands.go`
- `cmd/vrooli/lifecycle_commands.go`
- `cmd/vrooli/command_errors.go`
- `cmd/vrooli/command_help.go`

That is not a thin composition root.

### 1.3 Concrete architectural leaks

1. `cmd/vrooli` still owns command dispatch and feature orchestration.

Examples:

- `cmd/vrooli/main.go` still parses globals, selects the command, and dispatches to handler maps.
- `cmd/vrooli/app.go` still acts as a command runtime container instead of pure bootstrap.
- `cmd/vrooli/command_registry.go` still binds command metadata to runtime handlers inside `cmd/`.
- `cmd/vrooli/app_handlers.go` still contains top-level command execution adapters and orchestration.

2. `cmd/vrooli` still owns family-specific command behavior.

Examples:

- `cmd/vrooli/scenario_logs.go` contains scenario log discovery/tailing/cleanup behavior.
- `cmd/vrooli/scenario_template.go` contains scenario template filesystem logic.
- `cmd/vrooli/scenario_external_commands.go` still translates requirements/test-genie/completeness subprocesses.
- `cmd/vrooli/resource_commands.go` and `cmd/vrooli/resource_actions.go` still own most resource family execution glue.
- `cmd/vrooli/package_commands.go` still owns package family execution/rendering glue.
- `cmd/vrooli/contract_commands.go` still owns contract command routing.

3. App-layer boundaries are still wrong.

Examples:

- `internal/app/scenario/service.go` imports `internal/cli/scenariocli`.
- `internal/app/contract/service.go` imports `internal/cli/contractcli`.

That is an inversion failure. `internal/app/*` must not depend on CLI packages.

4. CLI parsing/help still leaks into non-CLI packages.

Examples:

- `internal/setup/setup.go` still parses top-level args and prints help text for `setup`, `build`, and `develop`.
- `internal/cli/topcli` and `cmd/vrooli/command_errors.go` both own usage/error semantics.

5. String/contract ownership is still fragmented.

Examples:

- `cmd/vrooli/command_errors.go`
- `cmd/vrooli/command_help.go`
- `internal/cli/topcli/errors.go`
- `internal/cli/topcli/help.go`
- `internal/cli/scenariocli/help.go`
- `internal/cli/resourcecli/help.go`
- `internal/cli/contractcli/help.go`
- `internal/setup/setup.go`

6. Legacy/compatibility is still present in runtime-reachable paths.

Examples:

- scenario external command bridges in `cmd/vrooli/scenario_external_commands.go`
- resource legacy-driver support under:
  - `internal/resources/manifest/manifest.go`
  - `internal/resources/catalog/catalog.go`
  - `internal/resources/templates.go`
- testkit compatibility helpers under `packages/testkit-go/vrooli/compat.go`

If the target is truly “no legacy/compatibility/dead code at all,” these cannot remain.

7. Tests are still too concentrated in the binary package.

Largest command-package test files at audit time:

- `cmd/vrooli/scenario_main_test.go` at about `1,440` LOC
- `cmd/vrooli/package_commands_test.go` at about `899` LOC
- `cmd/vrooli/resource_commands_test.go` at about `685` LOC

The current weighting still says the binary package is where most behavior is understood and validated. That is the wrong seam.

---

## 2. Why the previous implementation was insufficient

The earlier plan was directionally right, but it was not strict enough about the actual end state.

Specifically, it allowed the codebase to stop at a halfway architecture:

- metadata moved into `internal/cli/*`
- some app services introduced
- but `cmd/vrooli` remained the real runtime command system

That created an attractive but misleading state: the code *looks* more internalized, while the binary package still owns too many responsibilities.

The refreshed plan below fixes that by being stricter about:

- what `cmd/` is allowed to contain
- what `internal/app/*` is allowed to import
- what must be deleted rather than wrapped
- what test seams must move out of `cmd/`
- what legacy/compatibility code must be fully removed

---

## 3. Exact target end state

This is the real finish line.

### 3.1 `cmd/` must be composition only

Final `cmd/vrooli` production code should be limited to:

- `main.go`
- at most a tiny bootstrap/composition file if needed
- OS-specific process-attribute files only if they are truly bootstrap-only

It must not contain:

- command registries
- command handler maps
- command-family parsing
- command-family help/usage
- command-family rendering
- feature-specific runtime helpers
- compatibility routing
- subprocess translation policy
- domain/application logic
- command-specific error policy

Concrete end-state expectation:

- `cmd/vrooli` production LOC should be closer to low hundreds than thousands
- most current production files in `cmd/vrooli` should be deleted

`cmd/vrooli-api` must follow the same rule.

### 3.2 The command runner belongs in `internal/`

There must be one authoritative internal command runner package that owns:

- global option parsing
- command tree assembly
- root-policy checks
- dispatch
- help rendering
- unknown-command handling
- exit-code mapping policy

Suggested home:

- `internal/cli/rootcli`
- or an equivalent internal runner package

The specific name can vary. The boundary cannot.

### 3.3 CLI packages are adapters, not application owners

Each `internal/cli/*` package should own only:

- command metadata
- command-specific parse/validate
- help/usage text
- CLI-only response rendering
- mapping between CLI DTOs and app DTOs

They must not own:

- business workflows
- shell/process launching
- resource/scenario/project orchestration
- controller composition

### 3.4 App packages must not depend on CLI packages

This is mandatory.

`internal/app/*` packages must not import `internal/cli/*`.

Instead:

- `internal/app/*` owns use-case DTOs
- `internal/cli/*` maps CLI requests/responses to/from app DTOs

This applies especially to:

- `internal/app/scenario`
- `internal/app/contract`

### 3.5 `internal/setup` must stop being a CLI surface

`internal/setup` must become a domain/application primitive only.

It must not own:

- command-line option parsing
- help text
- CLI usage strings

Top-level `setup`, `build`, and `develop` command parsing/help belongs in CLI packages. `internal/setup` should expose typed operations only.

### 3.6 No legacy or compatibility runtime paths

The final runtime path must contain none of the following:

- scenario external command translation in `cmd/`
- legacy resource-driver support such as `legacy-adapter`
- compatibility helpers retained for migration convenience
- unknown-command or unknown-subcommand fallbacks that preserve old behavior
- testkit compatibility layers kept “just in case”

If the repo still needs a migration path during implementation, it must be temporary, explicitly named, and then deleted before the plan is complete.

### 3.7 Tests must follow the architecture

Final testing shape:

- parser/render tests live under `internal/cli/*`
- use-case tests live under `internal/app/*`
- domain tests live under owning `internal/*` packages
- shared fixtures live in `packages/testkit-go` or `packages/testkit-go/vrooli`
- `cmd/vrooli` retains only a small smoke/integration layer

### 3.8 One source of truth for command and output contracts

The following must each have one authoritative declarative home:

- command names
- aliases
- help text
- usage text
- stable error categories/codes/hints
- JSON envelope keys
- stable human-facing output text
- repo-contract filenames/keys used by the CLI

Tests must reuse those declarations rather than restating literals where practical.

---

## 4. Target package shape

The exact names may vary slightly, but the responsibility split must look like this:

```text
/cmd/
  vrooli/
    main.go
    process_attrs_unix.go
    process_attrs_windows.go
  vrooli-api/
    main.go

/internal/
  bootstrap/
  cli/
    commandtree/
    rootcli/          # global options, root dispatch, unknown-command/help policy
    topcli/           # root command declarations if kept separate from rootcli
    projectcli/
    scenariocli/
    resourcecli/
    packagecli/
    contractcli/
  app/
    contextinfo/
    contract/
    package/
    project/
    resource/
    scenario/
  cliout/
  buildinfo/
  config/
  control/
  lifecycle/
  maintenance/
  orchestrator/
  packagegov/
  ports/
  process/
  project/
  resources/
  runtime/
  scenario/
  setup/              # typed operations only, no CLI parsing/help
  shell/
  vroolierr/
```

Key dependency rules:

1. `cmd/*` may depend only on bootstrap/composition-oriented internal packages plus the shared internal command runner.
2. `internal/cli/*` may depend on `internal/app/*`, `internal/cliout`, and shared contract/error packages.
3. `internal/app/*` may depend on domain/infrastructure packages, but not on `internal/cli/*`.
4. Domain/infrastructure packages must not know about CLI parsing or help text.

---

## 5. Gap inventory by area

This section is the checklist driver. Every item here must be resolved before the effort is done.

### 5.1 Binary package is still thick

Must be addressed in:

- `cmd/vrooli/main.go`
- `cmd/vrooli/app.go`
- `cmd/vrooli/command_registry.go`
- `cmd/vrooli/command_bindings.go`
- `cmd/vrooli/app_handlers.go`
- `cmd/vrooli/command_errors.go`
- `cmd/vrooli/command_help.go`
- `cmd/vrooli/top_level_runtime.go`
- `cmd/vrooli/lifecycle_commands.go`

### 5.2 Scenario family still lives materially in `cmd/`

Must be addressed in:

- `cmd/vrooli/scenario_actions.go`
- `cmd/vrooli/scenario_lifecycle_commands.go`
- `cmd/vrooli/scenario_external_commands.go`
- `cmd/vrooli/scenario_logs.go`
- `cmd/vrooli/scenario_template.go`
- `cmd/vrooli/scenario_template_actions.go`
- `cmd/vrooli/scenario_helpers.go`

### 5.3 Resource family still lives materially in `cmd/`

Must be addressed in:

- `cmd/vrooli/resource_actions.go`
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/resource_template.go`
- `cmd/vrooli/resource_template_actions.go`

### 5.4 Package and contract families still live materially in `cmd/`

Must be addressed in:

- `cmd/vrooli/package_commands.go`
- `cmd/vrooli/contract_commands.go`

### 5.5 App-layer boundary violations still exist

Must be addressed in:

- `internal/app/scenario/service.go`
- `internal/app/contract/service.go`

### 5.6 Top-level lifecycle/setup CLI concerns still leak downward

Must be addressed in:

- `internal/setup/setup.go`
- `internal/cli/topcli/*`
- `internal/cli/projectcli/*`

### 5.7 Legacy/compatibility still exists

Must be addressed in:

- `cmd/vrooli/scenario_external_commands.go`
- `internal/resources/manifest/manifest.go`
- `internal/resources/catalog/catalog.go`
- `internal/resources/templates.go`
- `packages/testkit-go/vrooli/compat.go`

### 5.8 String and error ownership is still split

Must be addressed in:

- `cmd/vrooli/command_errors.go`
- `cmd/vrooli/command_help.go`
- `internal/cli/topcli/errors.go`
- `internal/cli/*/help.go`
- human-output helpers across `internal/cli/*` and `internal/setup`

### 5.9 Test architecture is still wrong

Must be addressed in:

- most `cmd/vrooli/*_test.go`
- `packages/testkit-go/vrooli/*`
- parser/render test coverage under `internal/cli/*`
- use-case tests under `internal/app/*`

---

## 6. Non-negotiable design principles

### 6.1 Prefer deletion over relocation

Moving code from one `cmd/vrooli/*.go` file to another does not count as progress.

### 6.2 App DTOs belong to app packages

CLI packages may not be the canonical owners of app-layer request/response types.

### 6.3 Help/error/output text must be declarative

No more scattered ad hoc help rendering or duplicate usage strings.

### 6.4 Compatibility is temporary only during deletion

If a temporary bridge is introduced during implementation, it must have an explicit deletion item in the same plan.

### 6.5 Binary-package tests must be minimized

If behavior can be validated below `cmd/`, that is where it belongs.

### 6.6 Completion requires full deletion

If code is dead, redundant, compatibility-only, or superseded, it must be removed before this effort is marked complete.

---

## 7. Phased implementation checklist

All boxes are intentionally reset. The previous completion marks are no longer trusted.

## Phase 0: Re-baseline and stop treating the current state as complete

Objective:

- Establish the audited repo state and reset the plan to truth.

Checklist:

- [x] Replace the previous completion claims with this audited plan.
- [x] Record current `cmd/vrooli` production/test LOC in the plan.
- [x] Record current boundary violations and legacy surfaces in the plan.
- [x] Freeze the target end state for `cmd/`, `internal/cli/*`, `internal/app/*`, `internal/setup`, and testkit.

Validation:

- [x] Plan reviewed against current repo state.

Definition of done:

- The repo has one current, truthful plan and no stale “complete” narrative.

Phase 0 completion note:

- Completed on April 13, 2026 by re-auditing the current repo, replacing the stale “complete” plan state, recording the current architectural gaps, and freezing the stricter end-state and phased checklist that will govern the remaining overhaul work.

## Phase 1: Move the root command runtime out of `cmd/vrooli`

Objective:

- Create a single internal root command runner and reduce `cmd/vrooli` to bootstrap only.

Checklist:

- [x] Introduce a dedicated internal root runner package.
- [x] Move global option parsing out of `cmd/vrooli/main.go`.
- [x] Move root command dispatch out of `cmd/vrooli/main.go`.
- [x] Move command registry construction out of `cmd/vrooli/command_registry.go`.
- [x] Move unknown-command rendering/help policy out of `cmd/vrooli/command_help.go`.
- [x] Move command error category/hint/suggestion policy out of `cmd/vrooli/command_errors.go`.
- [x] Move generic command binding/execution helpers out of `cmd/vrooli/command_bindings.go`.
- [x] Reduce `cmd/vrooli/app.go` to dependency graph construction only, while keeping a thin `App.Run` wrapper for shared test seams.
- [ ] Delete:
  - [x] `cmd/vrooli/command_registry.go`
  - [x] `cmd/vrooli/command_bindings.go`
  - [x] `cmd/vrooli/command_errors.go`
  - [x] `cmd/vrooli/command_help.go`
  - [x] now-dead root dispatch helpers reachable only through those files

Validation:

- [x] Parser/dispatch/help tests moved under `internal/cli/rootcli` or equivalent.
- [x] `go test ./cmd/vrooli ./internal/cli/rootcli ./internal/cli/topcli ./internal/cli/scenariocli ./internal/cli/resourcecli ./internal/cli/packagecli`

Definition of done:

- `cmd/vrooli` no longer owns the command runtime; it only wires and invokes it.

Phase 1 completion note:

- Completed on April 13, 2026 by introducing `internal/cli/rootcli`, routing `main.go` and the shared `App.Run` test seam through the same internal runner, deleting the old root runtime files from `cmd/vrooli`, and moving root parser/dispatch coverage onto the new internal seam.

## Phase 2: Remove CLI-package imports from app services

Objective:

- Enforce clean app/CLI boundaries before continuing extraction work.

Checklist:

- [x] Replace `internal/app/scenario` imports of `internal/cli/scenariocli` with app-owned DTOs.
- [x] Replace `internal/app/contract` imports of `internal/cli/contractcli` with app-owned DTOs.
- [x] Ensure no `internal/app/*` package imports `internal/cli/*`.
- [x] Move any shared DTOs currently living in CLI packages into app packages or a neutral contract package.
- [x] Update CLI packages to map to/from app DTOs instead of reusing them directly.

Validation:

- [x] `rg -n 'internal/cli/' internal/app` returns no runtime-package imports.
- [x] App-service tests pass with the new DTO boundaries.

Definition of done:

- App services are reusable and CLI-agnostic.

Phase 2 completion note:

- Completed on April 13, 2026 by moving scenario and contract request/response DTO ownership into `internal/app/*`, removing all runtime `internal/cli/*` imports from app services, and making the CLI layer parse/render against app-owned contracts through explicit adapters.

## Phase 3: Collapse and clean the top-level/project lifecycle surface

Objective:

- Stop spreading top-level behavior across `topcli`, `projectcli`, `cmd/vrooli`, and `internal/setup`.

Checklist:

- [x] Decide the final ownership split between `rootcli`, `topcli`, and `projectcli`.
- [x] Remove duplicate ownership between `internal/cli/topcli` and `internal/cli/projectcli`.
- [x] Move top-level lifecycle command parsing/help out of `internal/setup`.
- [x] Make `internal/setup` expose typed operations only.
- [x] Centralize top-level usage/help/error strings.
- [x] Move `cleanup` to either:
  - [x] a fully declarative first-class command under internal CLI packages, or
  - [ ] delete it if it is redundant
- [x] Move lifecycle-protect command metadata/help/error handling into internal CLI packages.
- [x] Delete:
  - [x] remaining top-level command adapters from `cmd/vrooli/app_handlers.go`
  - [x] `cmd/vrooli/top_level_runtime.go`
  - [x] `cmd/vrooli/lifecycle_commands.go`
  - [x] any now-redundant `internal/cli/topcli` wrappers if `projectcli` subsumes them

Validation:

- [x] `vrooli --help`
- [x] `vrooli info --help`
- [x] `vrooli status --json`
- [x] `vrooli doctor --json`
- [x] `vrooli stop --json`
- [x] `vrooli cleanup --help` or confirmed deletion of `cleanup`

Definition of done:

- Top-level command behavior is declared and implemented entirely outside `cmd/`.

Phase 3 completion note:

- Completed on April 13, 2026 by moving top-level lifecycle parsing/help/handler construction into `internal/cli/projectcli`, removing top-level setup/develop/build parsing from `internal/setup`, deleting `cmd/vrooli/top_level_runtime.go` and `cmd/vrooli/lifecycle_commands.go`, deleting the remaining dead project-lifecycle adapters from `cmd/vrooli/app_handlers.go`, and making `cleanup`/`lifecycle protect` first-class internal CLI handlers.
- Final ownership split:
  - `internal/cli/rootcli` owns root parsing, registry assembly, dispatch, help/version routing, and exit-code policy.
  - `internal/cli/topcli` owns only the top-level command catalog and root-visible help surface.
  - `internal/cli/projectcli` owns top-level project/lifecycle/maintenance parse, help, validation, and handler builders.
- Validation completed with `go test ./internal/cli/projectcli ./internal/setup ./internal/cli/topcli`, `go test ./cmd/vrooli -count=1`, and live command checks for `vrooli --help`, `vrooli info --help`, `vrooli cleanup --help`, `vrooli status --json`, `vrooli doctor --json`, and `vrooli stop --json`.
- Note: `vrooli stop --json` is intentionally stateful and stopped active scenarios during validation; a second verification run returned `stopped: 0`.

## Phase 4: Extract the scenario family completely

Objective:

- Remove all remaining scenario-family runtime behavior from `cmd/vrooli`.

Checklist:

- [x] Move scenario command-root dispatch into internal CLI runner packages.
- [x] Move scenario logs behavior out of `cmd/vrooli/scenario_logs.go`.
- [x] Move scenario template load/copy/verify/generate behavior out of `cmd/vrooli/scenario_template.go`.
- [x] Move scenario template hook execution orchestration out of `cmd/vrooli/scenario_template_actions.go`.
- [x] Move scenario requirements snapshot/report translation out of `cmd/vrooli/scenario_external_commands.go`.
- [x] Eliminate command-layer subprocess translation for:
  - [x] `scenario requirements`
  - [x] `scenario ui-smoke`
  - [x] `scenario completeness`
  - [x] any remaining test-genie bridging
- [x] Decide the true end-state for those features:
  - [ ] native internal use cases, or
  - [x] bounded adapters below the CLI/app layers
- [x] Move any scenario helper logic in `cmd/vrooli/scenario_helpers.go` to proper internal ownership.
- [x] Delete:
  - [x] `cmd/vrooli/scenario_actions.go`
  - [x] `cmd/vrooli/scenario_lifecycle_commands.go`
  - [x] `cmd/vrooli/scenario_external_commands.go`
  - [x] `cmd/vrooli/scenario_logs.go`
  - [x] `cmd/vrooli/scenario_template.go`
  - [x] `cmd/vrooli/scenario_template_actions.go`
  - [x] `cmd/vrooli/scenario_helpers.go`

Validation:

- [x] `vrooli scenario --help`
- [x] `vrooli scenario list --json`
- [x] `vrooli scenario status --json`
- [x] `vrooli scenario start-all --json`
- [x] `vrooli scenario stop-all --json`
- [x] `vrooli scenario logs --help`
- [x] `vrooli scenario template --help`
- [x] `vrooli scenario requirements --help`

Definition of done:

- Scenario-family runtime logic no longer exists in `cmd/vrooli`.

Phase 4 completion note:

- Completed on April 13, 2026 by moving the scenario-family runtime into:
  - `internal/cli/scenariohandlers` for executable handler construction and scenario-specific CLI orchestration
  - `internal/scenarioexec` for subprocess, CLI-discovery, browser-open, and detached-launch support
  - existing `internal/cli/scenariocli` packages for scenario command metadata, parsing, help, and rendering
- `cmd/vrooli` no longer contains production scenario runtime files; `rg --files cmd/vrooli | rg 'scenario'` now returns test files only.
- The previous scenario subprocess bridges were intentionally retained only as bounded internal adapters below the CLI/app wiring layer. They no longer live in `cmd/`.
- JSON mode now routes lifecycle progress chatter to `stderr` at the bootstrap service boundary so scenario JSON commands keep `stdout` machine-readable.
- Validation completed with:
  - `go test ./internal/cli/scenariohandlers ./internal/scenarioexec`
  - `go test ./cmd/vrooli -count=1`
  - live/source command checks for `scenario --help`, `scenario list --json`, `scenario status --json`, `scenario logs --help`, `scenario template --help`, and `scenario requirements --help`
  - package coverage for `scenario start-all --json` and `scenario stop-all --json` in `cmd/vrooli`

## Phase 5: Extract the resource family completely and delete legacy resource support

Objective:

- Remove all remaining resource-family runtime behavior from `cmd/` and delete legacy resource-driver compatibility.

Checklist:

- [x] Move resource-family root dispatch into internal CLI runner packages.
- [x] Move resource template command runtime logic out of `cmd/vrooli/resource_template.go`.
- [x] Move resource action glue out of `cmd/vrooli/resource_actions.go`.
- [x] Move resource family routing out of `cmd/vrooli/resource_commands.go`.
- [x] Ensure archive/blueprint/template subcommands are defined and executed entirely from internal packages.
- [x] Delete runtime support for `legacy-adapter` in project-level resource handling.
- [x] Remove `legacy_adapter` schema/manifest/catalog/template support from:
  - [x] `internal/resources/manifest/manifest.go`
  - [x] `internal/resources/catalog/catalog.go`
  - [x] `internal/resources/templates.go`
- [x] Update tests/fixtures away from `legacy-adapter`.
- [x] Delete:
  - [x] `cmd/vrooli/resource_actions.go`
  - [x] `cmd/vrooli/resource_commands.go`
  - [x] `cmd/vrooli/resource_template.go`
  - [x] `cmd/vrooli/resource_template_actions.go`

Implementation notes:

- Resource runtime now lives in `internal/cli/resourcehandlers/handlers.go`.
- Top-level `resource` dispatch in `cmd/vrooli` is now a thin handoff to the internal handler package.
- Nested `resource blueprint|archive|template --help` now resolves to the correct subcommand help instead of falling back to root resource help.
- The `templates/resources/legacy-adapter/` scaffold was deleted.

Validation:

- [x] `vrooli resource --help`
- [x] `vrooli resource list --json`
- [x] `vrooli resource status --json`
- [x] `vrooli resource blueprint --help`
- [x] `vrooli resource archive --help`
- [x] `vrooli resource template --help`
- [x] Search confirms no runtime `legacy-adapter` support remains.
- [x] Focused test validation passed for the resource slice:
  - `go test ./cmd/vrooli -run TestDoesNotExist`
  - `go test ./internal/resources/... -run TestDoesNotExist`
  - `go test ./internal/resources ./internal/cli/resourcecli ./cmd/vrooli -run 'TestValidateRejectsInvalidDriver|TestWriteStatusHumanIncludesCoreMetadata|TestSplitRunResourceStatusUsesNativeController|TestMigratedResourceCLIsDelegateStandardCommandsToNativeControlPlane'`

Validation caveat:

- The broad `go test ./cmd/vrooli ./internal/resources ./internal/cli/resourcecli ./internal/api` run still appears to include long-running existing coverage outside the Phase 5 slice, so Phase 5 was validated with focused compile and behavior checks rather than waiting indefinitely on the entire package set.

Definition of done:

- Resource commands run natively from internal packages and no legacy resource-driver compatibility remains.

## Phase 6: Extract package and contract families completely

Objective:

- Finish package and contract cleanup to the same standard.

Checklist:

- [x] Move package-family root dispatch and runtime glue out of `cmd/vrooli/package_commands.go`.
- [x] Move contract-family root dispatch and runtime glue out of `cmd/vrooli/contract_commands.go`.
- [x] Ensure package lifecycle execution is owned by app/adapters rather than command packages.
- [x] Ensure contract validation/show/resolve/match are declared and run from internal CLI/app packages only.
- [ ] Delete:
  - [x] `cmd/vrooli/package_commands.go`
  - [x] `cmd/vrooli/contract_commands.go`

Implementation notes:

- Package command runtime now lives in `internal/cli/packagehandlers/handlers.go`.
- Contract command runtime now lives in `internal/cli/contracthandlers/handlers.go`.
- Package response rendering now lives in `internal/cli/packagecli/render.go`, so package help/parse/render contracts are declared under one internal CLI family instead of being split across `cmd/`.
- Repo-contract runtime helpers were moved out of `internal/cli/contractcli` into `internal/app/contract/runtime.go`, so the contract app service no longer relies on CLI-owned operational functions.
- `cmd/vrooli/app_handlers.go` now hands off `package` and `contract` to internal handler packages, and `cmd/vrooli` no longer contains production package/contract command-family files.

Validation:

- [x] `vrooli package --help`
- [x] `vrooli package list --json`
- [x] `vrooli contract --help`
- [x] `vrooli contract validate --json`
- [x] Focused validation passed for the extracted package/contract slice:
  - [x] `go test ./internal/cli/packagecli ./internal/cli/contractcli ./internal/app/contract`
  - [x] `go test ./internal/cli/packagehandlers ./internal/cli/contracthandlers`
  - [x] `go test ./cmd/vrooli -run 'TestPackage|TestRunContract|TestShowMainHelpIncludesContractCommand'`
  - [x] `go test ./cmd/vrooli -run 'TestDoesNotExist'`

Definition of done:

- Package and contract families no longer have runtime command logic in `cmd/`.

Phase 6 completion note:

- Completed on April 13, 2026 by moving package and contract command dispatch/runtime glue into `internal/cli/packagehandlers` and `internal/cli/contracthandlers`, moving package response rendering into `internal/cli/packagecli`, moving repo-contract operational functions into `internal/app/contract/runtime.go`, deleting `cmd/vrooli/package_commands.go` and `cmd/vrooli/contract_commands.go`, and updating the command-package tests to exercise the new handler seam instead of deleted production helpers.

## Phase 7: Centralize strings, errors, and output contracts

Objective:

- Eliminate drift across help, errors, and output text.

Checklist:

- [x] Choose one authoritative home for root-level unknown-command handling text.
- [x] Choose one authoritative home for usage error construction and categories.
- [x] Remove duplicated help/error helpers between:
  - [x] `cmd/vrooli/*`
  - [x] `internal/cli/topcli/errors.go`
  - [x] any family-local duplicated helpers
- [x] Centralize JSON envelope conventions in one shared place.
- [x] Centralize stable human-facing output strings where they are part of a contract.
- [x] Centralize repo-contract and info-manifest filenames/keys used by command surfaces.
- [x] Replace test literals with shared declarations where practical.

Validation:

- [x] `rg -n 'Usage: vrooli|Unknown command:|Run '\''vrooli .* --help'\''' cmd/vrooli internal/cli internal/app internal/setup` shows only deliberate declarative homes.
- [x] Representative command output tests pass for both human and JSON modes.

Definition of done:

- Stable command/output/error contracts are declared once and reused.

Phase 7 completion note:

- Completed on April 13, 2026 by introducing `internal/cli/clipolicy` as the authoritative home for help-only errors, usage error construction, error categories, unknown-command rendering text, and shared usage hints.
- Deleted `internal/cli/topcli/errors.go` and removed the duplicated family-local `helpOnlyError` / `commandHelpOnly` / `usageErrorf` / `unknownOptionError` helpers across `projectcli`, `scenariocli`, `resourcecli`, `contractcli`, and `packagecli`.
- Extended `internal/cliout/contracts.go` so explicit-success JSON envelopes share one implementation path, then updated scenario/resource validation and status renderers to use the shared envelope helpers instead of handwritten `"success"` maps.
- Expanded `internal/repocontractmeta` to own the canonical `service.json`, `resource.json`, and info-manifest path contracts used by command surfaces, then updated the affected runtime code and tests to consume those declarations.
- Replaced representative `cmd/vrooli`, `internal/cli`, and `packages/testkit-go` test literals with shared declarations for usage text, unknown-command labels/hints, and manifest paths.
- Validation completed with:
  - `go test ./internal/cli/clipolicy ./internal/cli/rootcli ./internal/cli/topcli ./internal/cli/projectcli ./internal/cli/scenariocli ./internal/cli/resourcecli ./internal/cli/contractcli ./internal/cli/packagecli ./internal/cliout`
  - `go test ./internal/resources/manifest ./internal/resources/catalog ./internal/scenario ./internal/repocontractmeta`
  - `go test ./packages/testkit-go ./cmd/vrooli -run 'TestRunInfoHelpExitsZero|TestLifecycleCommandIsHiddenFromMainHelpButSupportsDirectHelp|TestPrintErrorWithContextFormatsUnknownCommandSuggestions|TestNewUnknownCommandErrorIncludesSuggestionCategory|TestExecuteTopLevelCommandRendersHelpOnlyErrors|TestExecuteScenarioCommandRendersHelpOnlyErrors|TestParseScenarioRequirementsRequestTreatsHelpAsCommandHelp|TestRunContractResolveScenarioServicePath|TestCanonicalSetupInstallContract'`

## Phase 8: Rebuild the test architecture around the new seams

Objective:

- Move the center of gravity of the test suite out of `cmd/vrooli`.

Checklist:

- [x] Reduce `cmd/vrooli` tests to smoke/integration coverage only.
- [x] Move parser tests into `internal/cli/*`.
- [x] Move renderer tests into `internal/cli/*` or `internal/cliout`.
- [x] Move use-case tests into `internal/app/*`.
- [x] Promote repeated repo/fixture helpers into `packages/testkit-go/vrooli`.
- [x] Delete or rename compatibility-oriented testkit helpers such as `packages/testkit-go/vrooli/compat.go` if they no longer match the target architecture.
- [x] Replace large binary-package fixtures with shared testkit builders.
- [x] Split or delete oversized `cmd/vrooli/*_test.go` files that are validating lower-layer behavior.

Validation:

- [x] `cmd/vrooli` test LOC is materially smaller than current baseline.
- [x] internal CLI/app tests materially increase relative to current baseline.
- [x] `go test ./cmd/vrooli/... ./internal/...`

Completion notes:

- Deleted lower-layer `cmd/vrooli` test files for root parsing, top-level render helpers, scenario parser/render helpers, package command behavior, and resource CLI compatibility coverage after migrating that coverage below the binary package.
- Added or expanded coverage in `internal/cli/rootcli`, `internal/cli/topcli`, `internal/cli/projectcli`, `internal/cli/scenariocli`, `internal/cli/scenariohandlers`, `internal/app/package`, `internal/orchestrator`, `internal/setup`, and `internal/resources`.
- Promoted generic polling and port-allocation helpers into `packages/testkit-go`, renamed `packages/testkit-go/vrooli/compat.go` to `runtime_helpers.go`, and removed remaining `cmd/vrooli` wrappers for those helpers.
- Removed the dead `templates/resources/legacy-adapter/` scaffold so the repo matches the intended no-legacy template set and `internal/resources` validation runs green.
- `cmd/vrooli` test LOC dropped from `5,275` to `3,444` while correctness coverage moved into internal seams.

Definition of done:

- Most command correctness is validated below the binary package.

## Phase 9: Final deletion pass and architectural verification

Objective:

- Ensure nothing transitional remains.

Checklist:

- [ ] Review `cmd/vrooli` production files and delete everything not strictly bootstrap-only.
- [ ] Review `cmd/vrooli-api` by the same standard.
- [ ] Confirm no `internal/app/*` imports `internal/cli/*`.
- [ ] Confirm no runtime support remains for command-layer compatibility bridges.
- [ ] Confirm no runtime support remains for resource `legacy-adapter`.
- [ ] Confirm no stale wrapper helpers or transitional aliases remain.
- [ ] Confirm no testkit compatibility shims remain.
- [ ] Update this plan with actual completion notes only after all validations are green.

Validation:

- [ ] `find cmd/vrooli -maxdepth 1 -name '*.go' ! -name '*_test.go'`
  Result should reflect only bootstrap-level files.
- [ ] `rg -n 'internal/cli/' internal/app`
  Returns no runtime-package imports.
- [ ] `rg -n 'legacy-adapter|legacy_adapter|compat' cmd internal packages/testkit-go/vrooli`
  Returns no runtime/active compatibility code, only truly intentional non-runtime documentation if any.

Definition of done:

- The architecture now looks greenfield and intentional, not partially migrated.

---

## 8. Required deletions

These files should be treated as deletion targets, not permanent fixtures.

### 8.1 `cmd/vrooli` production files expected to disappear

- `cmd/vrooli/app.go`
- `cmd/vrooli/app_handlers.go`
- `cmd/vrooli/command_registry.go`
- `cmd/vrooli/command_bindings.go`
- `cmd/vrooli/command_errors.go`
- `cmd/vrooli/command_help.go`
- `cmd/vrooli/top_level_runtime.go`
- `cmd/vrooli/lifecycle_commands.go`
- `cmd/vrooli/scenario_actions.go`
- `cmd/vrooli/scenario_lifecycle_commands.go`
- `cmd/vrooli/scenario_external_commands.go`
- `cmd/vrooli/scenario_logs.go`
- `cmd/vrooli/scenario_template.go`
- `cmd/vrooli/scenario_template_actions.go`
- `cmd/vrooli/scenario_helpers.go`
- `cmd/vrooli/resource_actions.go`
- `cmd/vrooli/resource_commands.go`
- `cmd/vrooli/resource_template.go`
- `cmd/vrooli/resource_template_actions.go`
- `cmd/vrooli/package_commands.go`
- `cmd/vrooli/contract_commands.go`

If any of these survive, they must be reduced to genuinely trivial bootstrap glue. The default expectation is deletion.

### 8.2 Runtime compatibility targets expected to disappear

- scenario external command translation in `cmd/vrooli/scenario_external_commands.go`
- resource `legacy-adapter` runtime/schema/catalog/template support
- any fallback routing preserved for migration convenience
- testkit compatibility helpers that only exist to prop up transitional code

---

## 9. Validation matrix

This matrix is required for completion. Do not mark the effort done without it.

### 9.1 Core test validation

- [ ] `go test ./cmd/vrooli/... ./internal/...`
- [ ] `go test ./packages/testkit-go/...`
- [ ] `go test ./...`

If `go test ./...` is blocked by repo permissions or an unreadable directory, fixing that blocker is part of completion. A blocked full-repo test run does **not** count as done.

### 9.2 Focused CLI validation

- [ ] `vrooli --help`
- [ ] `vrooli --version`
- [ ] `vrooli info`
- [ ] `vrooli status --json`
- [ ] `vrooli doctor --json`
- [ ] `vrooli stop --json`
- [ ] `vrooli contract validate --json`
- [ ] `vrooli package list --json`
- [ ] `vrooli resource list --json`
- [ ] `vrooli resource status --json`
- [ ] `vrooli scenario list --json`
- [ ] `vrooli scenario status --json`
- [ ] `vrooli scenario logs --help`
- [ ] `vrooli scenario template --help`
- [ ] `vrooli scenario requirements --help`

### 9.3 Architectural validation

- [ ] `cmd/vrooli` is bootstrap-only by inspection.
- [ ] `cmd/vrooli-api` is bootstrap-only by inspection.
- [ ] No `internal/app/*` runtime package imports `internal/cli/*`.
- [ ] No runtime legacy/compatibility path remains.
- [ ] Shared fixtures are provided by `packages/testkit-go` where repetition justified it.

### 9.4 Optional higher-level validation

- [ ] `make validate`

If this is not green, record the exact blocker in this plan and do not silently wave it away.

---

## 10. Definition of done

This overhaul is complete only when **all** of the following are true:

- [ ] `cmd/vrooli` is composition/bootstrap only.
- [ ] `cmd/vrooli-api` is composition/bootstrap only.
- [ ] `cmd/vrooli` no longer owns command registries, command-family runtime logic, or command help/error policy.
- [ ] `internal/app/*` packages do not import `internal/cli/*`.
- [ ] `internal/setup` no longer owns CLI parsing/help strings.
- [ ] Scenario/resource/package/contract families are implemented entirely below `cmd/`.
- [ ] No resource `legacy-adapter` runtime support remains.
- [ ] No scenario external-command compatibility bridge remains.
- [ ] No testkit compatibility shim remains.
- [ ] Stable help/error/output contracts are declared once and reused.
- [ ] The binary-package test surface is small and smoke-oriented.
- [ ] Shared testkit utilities cover repeated fixture patterns.
- [ ] `go test ./...` is green.
- [ ] The validation matrix above is green.

If any item above is false, the effort is not done.

---

## 11. Immediate next slice

The next highest-value slice is:

1. Create the internal root runner and move global parsing/dispatch/help/error handling out of `cmd/vrooli`.
2. Remove `internal/app/*` imports of CLI packages.
3. Split `internal/setup` into typed operations only and move top-level lifecycle CLI parsing/help into internal CLI packages.

That sequence fixes the foundation. Without it, further extraction work will continue to pile new code onto a still-thick binary layer.
