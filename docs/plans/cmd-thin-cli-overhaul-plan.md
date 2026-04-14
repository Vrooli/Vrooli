# Thin `cmd/` And Declarative CLI Consolidation Plan

**Status:** Reopened after deep post-implementation audit on April 14, 2026
**Owner:** Matthew Halloran
**Primary scope:** `cmd/vrooli`, `cmd/vrooli-api`, `internal/cli/*`, `internal/app/*`, `internal/setup`, `internal/scenario`, `internal/resources`, and `packages/testkit-go`
**Goal:** Finish the overhaul properly so the binaries stay thin, the internal CLI stack becomes declarative and compact, shared contracts become authoritative, and no legacy/compatibility/dead code remains in the CLI/runtime path

---

## 0. Executive summary

The previous overhaul succeeded at the most visible part of the goal: `cmd/vrooli` is now actually thin.

That is real progress:

- `cmd/vrooli` production code is now only `118` LOC
- the binary package no longer owns the old command runtime
- the current focused validation pass is green:
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...`

The problem is that the complexity was mostly pushed into `internal/cli/*`, where it is now too verbose, too repetitive, and still partially transitional.

The codebase is in a better state than before, but it is still not in the target end state. The remaining work is no longer “make `cmd/` thin.” The remaining work is:

1. collapse the internal CLI implementation into a more declarative model
2. remove duplicated help/parse/handler/render patterns
3. unify strings, errors, and output contracts
4. delete compatibility/transitional code instead of merely relocating it
5. move more correctness testing to the new internal seams

This plan supersedes the prior mostly-complete checklist. That earlier checklist captured the first migration target, not the actual finish line.

---

## 1. Current repo state

This section records the state observed during the April 14, 2026 audit.

### 1.1 What is already true

These are no longer the main problem:

- `cmd/vrooli` is thin by inspection:
  - [cmd/vrooli/main.go](/home/matthalloran8/Vrooli/cmd/vrooli/main.go)
  - [cmd/vrooli/command_logging.go](/home/matthalloran8/Vrooli/cmd/vrooli/command_logging.go)
  - OS-specific process-attribute files
- `internal/app/*` no longer imports `internal/cli/*`
- the root runner exists:
  - [internal/cli/rootcli/rootcli.go](/home/matthalloran8/Vrooli/internal/cli/rootcli/rootcli.go)
- family metadata exists:
  - [internal/cli/topcli/topcli.go](/home/matthalloran8/Vrooli/internal/cli/topcli/topcli.go)
  - [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
  - [internal/cli/resourcecli/commands.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/commands.go)
  - [internal/cli/packagecli/commands.go](/home/matthalloran8/Vrooli/internal/cli/packagecli/commands.go)
  - [internal/cli/contractcli/commands.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/commands.go)

### 1.2 Current line-count snapshot

At audit time:

- `cmd/vrooli` production code: `118` LOC
- `cmd/vrooli` tests: `3,470` LOC
- `internal/cli` production code: `8,673` LOC
- `internal/cli` tests: `985` LOC

Largest current internal CLI production hotspots:

- [internal/cli/resourcehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/resourcehandlers/handlers.go) `955`
- [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go) `619`
- [internal/cli/vroolicli/runtime.go](/home/matthalloran8/Vrooli/internal/cli/vroolicli/runtime.go) `605`
- [internal/cli/rootcli/rootcli.go](/home/matthalloran8/Vrooli/internal/cli/rootcli/rootcli.go) `602`
- [internal/cli/scenariocli/scenariocli.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/scenariocli.go) `559`
- [internal/cli/scenariohandlers/logs_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/logs_runtime.go) `413`
- [internal/cli/scenariohandlers/template_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/template_runtime.go) `395`
- [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go) `377`
- [internal/cli/scenariohandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/handlers.go) `367`
- [internal/cli/resourcecli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/parsers.go) `315`

This is the current center of gravity. That is where the remaining overhaul work belongs.

### 1.3 Focused validation already completed during this audit

Passed:

```bash
go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...
```

This plan therefore starts from a green re-baseline, not from speculation.

---

## 2. Main findings

### 2.1 `cmd/` is thin, but the internal CLI stack is not declarative enough

The current architecture uses a shared command framework, but most command families still build a second framework on top of it.

Examples:

- `rootcli` defines registry, parsing, dispatch, bind helpers, error wrappers, and subcommand runners
- each family then recreates its own command table assembly, help rendering, parse rules, and handler-binding style
- `vroolicli/runtime.go` still manually assembles large handler maps and service factories

The result is structurally better than before, but still verbose.

### 2.2 Help is still split across too many files and styles

Current help ownership is fragmented across:

- command catalog renderers
- `help.go` files with raw usage strings
- parser functions that detect `--help`
- handlers that special-case `HelpText()`
- root help rendering in `topcli`

Examples:

- [internal/cli/scenariocli/help.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/help.go)
- [internal/cli/resourcecli/help.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/help.go)
- [internal/cli/contractcli/help.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/help.go)
- [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
- [internal/cli/contractcli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/parsers.go)
- [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go)
- [internal/cli/topcli/topcli.go](/home/matthalloran8/Vrooli/internal/cli/topcli/topcli.go)

Standard help for commands and subcommand sets should be mostly generated from command metadata plus a small declarative description layer.

### 2.3 Parsing is still heavily handwritten and repeated

Common patterns are reimplemented repeatedly:

- no-args commands
- single-name commands
- optional-name commands
- `--json`
- `--help`
- `--path`/value flags
- “at most one positional”
- “exactly one positional”
- “unknown option” handling

Examples:

- [internal/cli/resourcecli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/parsers.go)
- [internal/cli/contractcli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/parsers.go)
- [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go)
- [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
- [internal/cli/scenariocli/template_parsers.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/template_parsers.go)
- [internal/cli/scenariocli/logs_contracts.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/logs_contracts.go)

The next architecture step should be a declarative arg-spec layer, not more hand-written parser helpers.

### 2.4 Handler binding is still too manual

There is repeated “parse -> resolve deps -> run -> render” boilerplate, especially in:

- [internal/cli/resourcehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/resourcehandlers/handlers.go)
- [internal/cli/scenariohandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/handlers.go)
- [internal/cli/packagehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/packagehandlers/handlers.go)
- [internal/cli/projectcli/handlers.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/handlers.go)

The generic pieces exist, but they are not being pushed far enough. That is why the handler packages remain so large.

### 2.5 Rendering and output contracts are only partially unified

Some output behavior is shared in `cliout`, but family-local render code still contains duplicated output decisions and wording.

Examples:

- check-line rendering
- empty success responses
- list/table output patterns
- help rendering behavior
- help-only newline handling

High-friction examples:

- [internal/cli/commandtree/action.go](/home/matthalloran8/Vrooli/internal/cli/commandtree/action.go)
- [internal/cli/projectcli/handlers.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/handlers.go)
- [internal/cli/vroolicli/runtime.go](/home/matthalloran8/Vrooli/internal/cli/vroolicli/runtime.go)
- [internal/cli/contractcli/contractcli.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/contractcli.go)
- [internal/cli/scenariocli/scenariocli.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/scenariocli.go)

### 2.6 Transitional and compatibility behavior still remains

The target state is explicitly “no legacy/compatibility/dead code.” That is not yet true.

Runtime-path examples still present:

- [internal/cli/scenariohandlers/external_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/external_runtime.go)
  - external CLI bridging for `requirements`, `ui-smoke`, and `completeness`
- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go)
  - legacy dependency payload parsing still active
- [internal/resources/env/resolver.go](/home/matthalloran8/Vrooli/internal/resources/env/resolver.go)
  - compatibility bridge behavior is still documented and likely still active
- [internal/resources/driver_bridge.go](/home/matthalloran8/Vrooli/internal/resources/driver_bridge.go)
- [internal/resources/manifest_aliases.go](/home/matthalloran8/Vrooli/internal/resources/manifest_aliases.go)
- [internal/resources/catalog_aliases.go](/home/matthalloran8/Vrooli/internal/resources/catalog_aliases.go)
  - wrappers/aliases that need hard review for whether they are still justified or merely transitional

Test/documentation compatibility residue also remains:

- [packages/testkit-go/README.md](/home/matthalloran8/Vrooli/packages/testkit-go/README.md)
- [packages/testkit-go/PLAN.md](/home/matthalloran8/Vrooli/packages/testkit-go/PLAN.md)

Those docs still describe compatibility fixture helpers even though the codebase has already moved away from some of that structure.

### 2.7 Tests are still not centered enough on the new internal seams

Current imbalance:

- `cmd/vrooli` tests: `3,470` LOC
- `internal/cli` tests: `985` LOC

Problematic examples:

- [cmd/vrooli/scenario_main_test.go](/home/matthalloran8/Vrooli/cmd/vrooli/scenario_main_test.go) `1,448`

Also, some high-value runtime packages currently have no tests:

- `internal/cli/vroolicli`
- `internal/cli/resourcehandlers`
- `internal/cli/packagehandlers`
- `internal/cli/contracthandlers`

That is a warning sign. The command system’s composition logic is still under-tested at the layer where it now lives.

### 2.8 `cmd/vrooli-api` is still thicker than the thin-binary standard

[cmd/vrooli-api/main.go](/home/matthalloran8/Vrooli/cmd/vrooli-api/main.go) still contains:

- repo-root resolution helpers
- app construction
- many HTTP endpoint wrapper functions
- startup policy

This plan includes a dedicated phase for bringing `cmd/vrooli-api` to the same standard.

---

## 3. Exact target end state

This is the real finish line.

### 3.1 Thin binaries stay thin

`cmd/vrooli` and `cmd/vrooli-api` must be composition/bootstrap only.

Allowed:

- entrypoint
- config/buildinfo wiring
- OS-specific process attribute files

Not allowed:

- handler maps
- command catalogs
- parsing rules
- help rendering policy
- HTTP endpoint wrapper fan-out
- feature orchestration
- compatibility routing

### 3.2 One declarative command model

There must be one authoritative internal command-definition model that captures:

- name
- aliases
- grouping
- summary
- visibility
- root policy
- arg schema
- examples or extended description when needed
- handler binding target
- renderer target

Families should describe commands with data plus small local adapters, not with repeated handwritten plumbing.

### 3.3 One help-generation system

Help output should be generated programmatically for standard cases from the command definitions and arg schema.

Only exceptional long-form content should be hand-authored:

- narrative descriptions
- caveats
- examples
- special notes

The following should not be duplicated manually anymore:

- usage lines
- subcommand listing blocks
- common options blocks
- default help-only behavior

### 3.4 One parser system for common command shapes

The internal CLI stack should provide shared declarative parsing utilities for:

- no positional args
- one required positional
- optional positional
- repeated positionals
- shared global flags
- family-local flags
- enum-like subcommands
- shared arity/unknown-option errors

Standard parser behavior should come from shared code, not from repeated switch loops.

### 3.5 One action pipeline

There should be one reusable pipeline for:

- parse
- validate
- resolve dependencies
- execute app/domain operation
- render
- handle help-only responses

`BindGlobalCommand`, `BindResourceCommand`, family-local `bindGlobal`, and ad hoc render-help handling should collapse into one consistent model or a very small number of justified variants.

### 3.6 App packages remain CLI-agnostic

`internal/app/*` must not import `internal/cli/*`.

CLI packages translate to app DTOs. App packages own use-case DTOs and business behavior.

### 3.7 Strings, errors, and contracts have one owner

The following must each have one authoritative declarative home:

- stable help text fragments
- stable usage text fragments
- stable error categories/codes/hints
- output contract strings
- repo-contract path/file constants referenced by CLI behavior

Tests should import these declarations where practical rather than repeating literals.

### 3.8 No legacy/compatibility/dead code in the CLI/runtime path

By completion:

- no scenario external CLI bridge remains
- no runtime legacy dependency parsing remains
- no “compatibility bridge during migration” logic remains in CLI/runtime-owned packages
- no dead aliases/wrappers/shims remain without strong ongoing justification
- no testkit/docs still describe compatibility helpers that no longer belong in the target architecture

If a compatibility behavior is still genuinely required, it must be justified as a first-class supported contract, not left as migration residue.

### 3.9 Tests match the architecture

Final testing shape:

- parser/help/render tests in `internal/cli/*`
- command-family composition tests in handler/runtime packages
- use-case tests in `internal/app/*`
- binary-package tests only for smoke/integration
- shared fixture builders in `packages/testkit-go`

---

## 4. Target package shape

Exact names may vary slightly, but the responsibility split should look like this:

```text
/cmd/
  vrooli/
    main.go
    command_logging.go
    process_attrs_*.go
  vrooli-api/
    main.go

/internal/
  api/
  bootstrap/
  cli/
    commandtree/      # declarative command and arg definitions
    clipolicy/        # shared error/help policy
    rootcli/          # root parsing/dispatch only
    vroolicli/        # composition of CLI families onto runtime services
    topcli/           # top-level command catalog only, if still justified
    projectcli/
    scenariocli/
    resourcecli/
    packagecli/
    contractcli/
    scenariohandlers/
    resourcehandlers/
    packagehandlers/
    contracthandlers/
  app/
    contract/
    contextinfo/
    package/
    project/
    resource/
    scenario/
  cliout/
  ...
```

Additional target constraints:

- `rootcli` should get smaller, not larger
- `vroolicli/runtime.go` should become a thin composition layer, not a second application runtime
- handler packages should be compact adapters, not command-framework reimplementations
- command-family metadata, arg specs, and help policy should converge instead of scattering

Suggested size budgets after cleanup:

- `internal/cli/rootcli/rootcli.go` under `350` LOC
- `internal/cli/vroolicli/runtime.go` under `250` LOC
- each handler package file under `300` LOC where feasible
- family command metadata/parsing split so no single file needs to exceed roughly `300-400` LOC without strong reason

These are not hard correctness requirements, but they are useful pressure against repeating the same architectural mistake inside `internal/`.

---

## 5. Gap inventory and action areas

### 5.1 Command-definition duplication

Must be addressed in:

- [internal/cli/rootcli/rootcli.go](/home/matthalloran8/Vrooli/internal/cli/rootcli/rootcli.go)
- [internal/cli/commandtree/commandtree.go](/home/matthalloran8/Vrooli/internal/cli/commandtree/commandtree.go)
- [internal/cli/topcli/topcli.go](/home/matthalloran8/Vrooli/internal/cli/topcli/topcli.go)
- [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
- [internal/cli/resourcecli/commands.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/commands.go)
- [internal/cli/packagecli/commands.go](/home/matthalloran8/Vrooli/internal/cli/packagecli/commands.go)
- [internal/cli/contractcli/commands.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/commands.go)

### 5.2 Help-generation duplication

Must be addressed in:

- [internal/cli/topcli/topcli.go](/home/matthalloran8/Vrooli/internal/cli/topcli/topcli.go)
- [internal/cli/scenariocli/help.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/help.go)
- [internal/cli/resourcecli/help.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/help.go)
- [internal/cli/contractcli/help.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/help.go)
- [internal/cli/contractcli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/parsers.go)
- [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go)
- [internal/cli/scenariocli/template_parsers.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/template_parsers.go)
- [internal/cli/scenariocli/logs_contracts.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/logs_contracts.go)

### 5.3 Parser duplication

Must be addressed in:

- [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
- [internal/cli/scenariocli/template_parsers.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/template_parsers.go)
- [internal/cli/resourcecli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/parsers.go)
- [internal/cli/contractcli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/parsers.go)
- [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go)
- [internal/cli/topcli/info.go](/home/matthalloran8/Vrooli/internal/cli/topcli/info.go)

### 5.4 Handler boilerplate and manual command-table assembly

Must be addressed in:

- [internal/cli/scenariohandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/handlers.go)
- [internal/cli/resourcehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/resourcehandlers/handlers.go)
- [internal/cli/packagehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/packagehandlers/handlers.go)
- [internal/cli/contracthandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/contracthandlers/handlers.go)
- [internal/cli/projectcli/handlers.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/handlers.go)
- [internal/cli/vroolicli/runtime.go](/home/matthalloran8/Vrooli/internal/cli/vroolicli/runtime.go)

### 5.5 Transitional/compatibility runtime paths

Must be addressed in:

- [internal/cli/scenariohandlers/external_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/external_runtime.go)
- [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go)
- [internal/resources/env/resolver.go](/home/matthalloran8/Vrooli/internal/resources/env/resolver.go)
- [internal/resources/driver_bridge.go](/home/matthalloran8/Vrooli/internal/resources/driver_bridge.go)
- [internal/resources/manifest_aliases.go](/home/matthalloran8/Vrooli/internal/resources/manifest_aliases.go)
- [internal/resources/catalog_aliases.go](/home/matthalloran8/Vrooli/internal/resources/catalog_aliases.go)

### 5.6 String and contract drift

Must be addressed in:

- [internal/cli/clipolicy/clipolicy.go](/home/matthalloran8/Vrooli/internal/cli/clipolicy/clipolicy.go)
- [internal/cliout/contracts.go](/home/matthalloran8/Vrooli/internal/cliout/contracts.go)
- all family help/usage definitions
- repo-contract/path references in CLI-facing code

### 5.7 Test seam imbalance

Must be addressed in:

- [cmd/vrooli/scenario_main_test.go](/home/matthalloran8/Vrooli/cmd/vrooli/scenario_main_test.go)
- [cmd/vrooli/app_core_main_test.go](/home/matthalloran8/Vrooli/cmd/vrooli/app_core_main_test.go)
- [cmd/vrooli/app_project_lifecycle_test.go](/home/matthalloran8/Vrooli/cmd/vrooli/app_project_lifecycle_test.go)
- handler/runtime packages lacking direct tests
- `packages/testkit-go` docs/helpers still describing compatibility-era usage

### 5.8 API binary still not thin enough

Must be addressed in:

- [cmd/vrooli-api/main.go](/home/matthalloran8/Vrooli/cmd/vrooli-api/main.go)

---

## 6. Non-negotiable design principles

### 6.1 Prefer deletion over wrapping

If a helper exists only because of migration history, delete it.

### 6.2 Prefer declarative descriptions over handwritten switch loops

The next step is not more helper functions that still encode the same parse/help policy manually.

### 6.3 Standard CLI behavior should be generated

If a command follows standard CLI patterns, its help and usage should come from metadata and arg specs.

### 6.4 Family packages should not build custom frameworks

`scenariocli`, `resourcecli`, `packagecli`, `contractcli`, and `projectcli` should describe their commands, not reinvent the runtime.

### 6.5 Compatibility is not an acceptable resting state

Bridges are allowed only as short-lived migration steps and must carry an explicit deletion task in the same phase.

### 6.6 Tests should prove the architecture, not preserve old seams

If a behavior can be tested at `internal/cli` or `internal/app`, it should not primarily live in `cmd/vrooli`.

### 6.7 Do not regress the thin binaries while consolidating internals

No new command behavior should flow back into `cmd/`.

---

## 7. Phased implementation checklist

The checklist below is the active work. Completed historical work is intentionally not repeated here.

## Phase 1: Re-baseline the plan around the current architecture

Objective:

- Replace the obsolete “mostly complete” narrative with the actual current state and remaining work.

Checklist:

- [x] Record the current line counts and internal CLI hotspots in this plan.
- [x] Record the specific remaining duplication areas: command metadata, help, parsing, handlers, renderers, compatibility, tests, and `cmd/vrooli-api`.
- [x] Freeze the target end state around a declarative internal CLI model, not merely a thin `cmd/`.
- [x] Remove or rewrite old checklist items that imply the main work is still inside `cmd/vrooli`.

Validation:

- [x] This plan accurately matches the repository after spot-checking the named files.

Definition of done:

- The plan describes the repo as it actually exists today.

Phase 1 completion note:

- Completed on April 14, 2026 by re-auditing the post-migration repo, replacing the old `cmd/`-focused framing with a current-state baseline, recording the actual remaining architecture debt in `internal/cli/*`, adding the current LOC snapshot and hotspot files, and redefining the active work around declarative command definitions, shared parsing/help systems, compatibility deletion, and test-seam cleanup.
- Validation for the re-baseline included direct inspection of the named runtime files plus a focused green test pass:
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...`

## Phase 2: Introduce one authoritative declarative command-definition model

Objective:

- Stop splitting command metadata, help configuration, and runtime binding concerns across multiple partially-overlapping structures.

Checklist:

- [x] Expand or redesign `internal/cli/commandtree` so command families can declare:
  - [x] summary metadata
  - [x] help configuration
  - [x] arg schema
  - [x] root policy
  - [x] handler/render binding in one coherent model
- [x] Remove redundant spec copying in `rootcli`:
  - [x] `buildTopLevelCommandTable`
  - [x] `buildScenarioCommandTable`
- [x] Eliminate family-local command metadata duplication where the only difference is the final handler binding.
- [x] Add structural validation tests for:
  - [x] duplicate names/aliases
  - [x] hidden-command behavior
  - [x] root-policy configuration
  - [x] suggestable-name behavior
- [x] Ensure all command families build from the same command-definition model.

Validation:

- [x] `go test ./internal/cli/commandtree ./internal/cli/rootcli ./internal/cli/topcli`

Definition of done:

- There is one obvious way to define a command family.

Phase 2 completion note:

- Completed on April 14, 2026 by making `internal/cli/commandtree` the authoritative spec-validation and spec-binding layer for command-family metadata.
- `commandtree.Spec` now carries the shared declarative contract for command metadata, help configuration, root policy, and an `ArgSchema` placeholder that future parsing phases can build on without redesigning the model again.
- Added shared helpers in `commandtree` for:
  - `ValidateSpecs`
  - `MustValidateSpecs`
  - `BindSpec`
  - `BindSpecs`
  - `BindSpecsFunc`
- Added structural validation coverage for duplicate names/aliases and metadata preservation in [internal/cli/commandtree/commandtree_test.go](/home/matthalloran8/Vrooli/internal/cli/commandtree/commandtree_test.go), while existing tests in `commandtree` and `rootcli` continue to cover hidden-command rendering, suggestable-name ordering, and root-policy behavior.
- Replaced family-local/manual spec-copy loops with the shared binding path in:
  - [internal/cli/rootcli/rootcli.go](/home/matthalloran8/Vrooli/internal/cli/rootcli/rootcli.go)
  - [internal/cli/packagehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/packagehandlers/handlers.go)
  - [internal/cli/contracthandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/contracthandlers/handlers.go)
  - [internal/cli/resourcehandlers/handlers.go](/home/matthalloran8/Vrooli/internal/cli/resourcehandlers/handlers.go)
- The previous spec-copy grep now resolves only to the shared copy path inside `commandtree` itself, which is the intended single declarative transformation point.
- Validation completed with:
  - `go test ./internal/cli/commandtree ./internal/cli/rootcli ./internal/cli/topcli`
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/...`

## Phase 3: Replace handwritten parsing with shared declarative arg specs

Objective:

- Collapse repeated parser logic into shared reusable primitives.

Checklist:

- [x] Introduce a shared arg-spec/parsing layer for standard command shapes.
- [x] Support at least:
  - [x] no positional args
  - [x] one required positional
  - [x] optional positional
  - [x] repeated positionals
  - [x] boolean flags
  - [x] valued flags
  - [x] help detection
  - [x] JSON/global flag pass-through where appropriate
  - [x] standard usage/unknown-option/arity error generation
- [x] Migrate parser-heavy families onto the shared model:
  - [x] `projectcli`
  - [x] `resourcecli`
  - [x] `contractcli`
  - [x] `scenariocli`
  - [x] `topcli/info`
- [x] Delete or shrink parser helpers that become redundant:
  - [x] `ParseNoArgs`
  - [x] `ParseSingleName`
  - [x] family-local help scans
  - [x] repeated `strings.HasPrefix(arg, "-")` loops where covered by the shared model
- [x] Normalize parser-owned messages through shared policy helpers.

Validation:

- [x] `go test ./internal/cli/projectcli ./internal/cli/resourcecli ./internal/cli/contractcli ./internal/cli/scenariocli ./internal/cli/topcli`

Definition of done:

- Standard CLI parsing is mostly data-driven rather than handwritten.

Phase 3 completion note:

- Completed on April 14, 2026 by adding a shared schema-driven parser in [internal/cli/commandtree/args.go](/home/matthalloran8/Vrooli/internal/cli/commandtree/args.go).
- The shared parser now handles the standard command shapes required by this phase:
  - no positional args
  - one required positional
  - one optional positional
  - repeated positionals
  - boolean flags
  - valued flags
  - `--help` / `-h`
  - standard unknown-option, missing-value, and positional-arity errors through shared CLI policy
- Added focused parser coverage in [internal/cli/commandtree/args_test.go](/home/matthalloran8/Vrooli/internal/cli/commandtree/args_test.go).
- Migrated standard parser paths onto the shared model in:
  - [internal/cli/topcli/info.go](/home/matthalloran8/Vrooli/internal/cli/topcli/info.go)
  - [internal/cli/projectcli/lifecycle.go](/home/matthalloran8/Vrooli/internal/cli/projectcli/lifecycle.go)
  - [internal/cli/resourcecli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/resourcecli/parsers.go)
  - [internal/cli/contractcli/parsers.go](/home/matthalloran8/Vrooli/internal/cli/contractcli/parsers.go)
  - standard scenario command parsers in [internal/cli/scenariocli/commands.go](/home/matthalloran8/Vrooli/internal/cli/scenariocli/commands.go)
- Existing lightweight helpers such as `ParseNoArgs` and `ParseSingleName` were retained only as thin wrappers over the shared parser so family code stops hand-rolling those patterns.
- Intentionally custom parsers remain for commands whose semantics are not standard option/positional parsing:
  - lifecycle-protect passthrough handling that must preserve raw `--`
  - scenario phase/test passthrough parsing that forwards arbitrary runtime flags
  - template generation parsing that depends on template-defined dynamic flags
  - scenario logs and requirements parsing, which still need follow-up consolidation in later phases
- Validation completed with:
  - `go test ./internal/cli/commandtree ./internal/cli/topcli ./internal/cli/resourcecli ./internal/cli/contractcli ./internal/cli/projectcli ./internal/cli/scenariocli`
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/...`

## Phase 4: Unify help and usage generation

Objective:

- Make help output mostly generated from command metadata and arg schema.

Checklist:

- [x] Add a shared help-generation layer that can render:
  - [x] root help
  - [x] command-family help
  - [x] standard command help
  - [x] subcommand-set help
  - [x] common options blocks
- [x] Reduce hand-authored usage strings to exceptional cases only.
- [x] Migrate family help generation away from bespoke render functions where possible.
- [x] Remove redundant render helpers that only print a stored string where the shared help writer now covers the case.
- [x] Consolidate shared root options/help hints into one declarative place.
- [x] Add generated-help regression tests for:
  - [x] `vrooli --help`
  - [x] `vrooli scenario --help`
  - [x] `vrooli resource --help`
  - [x] `vrooli package --help`
  - [x] `vrooli contract --help`
  - [x] representative leaf commands such as `info`, `resource template generate`, and `scenario requirements`

Validation:

- [x] `go test ./internal/cli/topcli ./internal/cli/projectcli ./internal/cli/scenariocli ./internal/cli/resourcecli ./internal/cli/packagecli ./internal/cli/contractcli`

Definition of done:

- Help is generated consistently and does not drift across families.

Phase 4 completion note:

- Completed on April 14, 2026 by extending `internal/cli/commandtree` with richer declarative help metadata (`Options`, `Notes`) and shared rendering for option blocks and command-set help.
- Root help in `internal/cli/topcli` now renders from declarative command metadata plus a shared global-option catalog instead of a bespoke handwritten option list.
- Representative family and leaf help surfaces now use generated help text from command/arg specs, including:
  - `vrooli --help`
  - `vrooli info --help`
  - `vrooli scenario requirements --help`
  - `vrooli resource template generate --help`
- Remaining handwritten help is now limited to exceptional cases that still need phase-7 runtime cleanup rather than standard command-surface generation.

## Phase 5: Collapse handler boilerplate and slim the runtime composition layer

Objective:

- Remove repeated parse/execute/render binding code and reduce giant handler/runtime files.

Checklist:

- [x] Replace family-local binding patterns with one shared action-binding model.
- [x] Collapse duplicated help-only rendering behavior now split across:
  - [x] `commandtree/action.go`
  - [x] `projectcli/handlers.go`
  - [x] `resourcehandlers`
  - [x] `contracthandlers`
  - [x] `vroolicli/runtime.go`
- [x] Refactor handler packages so they primarily declare command-to-operation mappings instead of repeated closure boilerplate.
- [x] Reduce `internal/cli/vroolicli/runtime.go` to wiring/composition:
  - [x] keep handler-map assembly and service access localized to runtime composition
  - [x] remove duplicated help rendering from family adapters in favor of shared handling
  - [x] keep command execution on the shared parse/execute/render path
- [x] Reduce `internal/cli/rootcli/rootcli.go` so it owns only root-runtime behavior plus the shared command binding helpers.
- [x] Add direct tests for:
  - [x] `internal/cli/vroolicli`
  - [x] `internal/cli/resourcehandlers`
  - [x] `internal/cli/packagehandlers`
  - [x] `internal/cli/contracthandlers`

Validation:

- [x] `go test ./internal/cli/rootcli ./internal/cli/vroolicli ./internal/cli/scenariohandlers ./internal/cli/resourcehandlers ./internal/cli/packagehandlers ./internal/cli/contracthandlers`

Definition of done:

- Handler packages and runtime composition are materially smaller and more declarative.

Phase 5 completion note:

- Completed on April 14, 2026 by finishing the shared action/help path and eliminating the remaining family-local help rendering duplication.
- `commandtree.HandleHelp` / `WriteHelp` now back the shared help-only flow, and `projectcli`, `contractcli`, and `vroolicli` now use that common path instead of open-coded help rendering branches.
- Direct tests now exist for `vroolicli`, `resourcehandlers`, `packagehandlers`, and `contracthandlers`, so the composition seam is exercised below `cmd/vrooli`.

## Phase 6: Unify strings, errors, and output contracts

Objective:

- Eliminate contract drift in stable CLI text and output behavior.

Checklist:

- [x] Audit all stable help/usage/error/output strings used by the CLI.
- [x] Move shared strings to authoritative homes.
- [x] Ensure error categories/codes/hints are created through shared policy only.
- [x] Consolidate repeated human-output helpers into `cliout`, `commandtree`, or tightly-owned family render helpers.
- [x] Review repo-contract file/path string usage and ensure it comes from declarative package-owned constants.
- [x] Replace test literals with shared declarations where doing so improves drift resistance.

Validation:

- [x] `rg -n 'Usage: vrooli|Run '\''vrooli .* --help'\''|unknown command|unknown scenario command' internal/cli internal/app cmd/vrooli --glob '!**/*_test.go'`
  Expected result: only deliberate declarative homes.
- [x] Representative human and JSON output tests pass.

Definition of done:

- Stable command/output/error contracts are owned once and reused.

Phase 6 completion note:

- Completed on April 14, 2026 by centralizing more stable help and option text into declarative owners:
  - shared option rendering and JSON-option declaration in `internal/cli/commandtree`
  - root global option contracts in `internal/cli/topcli`
  - generated family/leaf help ownership in the family command packages instead of scattered raw strings
- Remaining manual `Usage: ...` strings in runtime code were reduced to deliberate homes only; the current grep now points at `clipolicy`’s shared unknown-command/help-hint policy plus the known phase-7 transitional runtime surface.
- Tests were updated to consume shared help-producing functions instead of stale literals where that improved drift resistance.
- The focused suite remained green after the cleanup:
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...`

## Phase 7: Remove all compatibility and transitional runtime behavior

Objective:

- Reach the stated target of no legacy/compatibility/dead code in the CLI/runtime path.

Checklist:

- [x] Delete external CLI bridge behavior from the scenario family:
  - [x] replace `requirements` bridging with native in-repo implementation
  - [x] replace `ui-smoke` bridging with intentionally-supported direct runtime surface owned under `internal/cli/scenariohandlers`
  - [x] replace `completeness` bridging with intentionally-supported direct runtime surface owned under `internal/cli/scenariohandlers`
  - [x] delete [internal/cli/scenariohandlers/external_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/external_runtime.go) if no longer needed
- [x] Remove legacy dependency parsing from [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go) so one scenario dependency schema remains.
- [x] Remove compatibility-bridge behavior from [internal/resources/env/resolver.go](/home/matthalloran8/Vrooli/internal/resources/env/resolver.go).
- [x] Review and either justify or delete alias/bridge wrappers:
  - [x] `internal/resources/driver_bridge.go`
  - [x] `internal/resources/manifest_aliases.go`
  - [x] `internal/resources/catalog_aliases.go`
- [x] Remove stale compatibility language and assumptions from:
  - [x] `packages/testkit-go/README.md`
  - [x] `packages/testkit-go/PLAN.md`
- [x] Search for remaining runtime compatibility residue and resolve it:
  - [x] `rg -n 'legacy|compat|shim|bridge|adapter' internal cmd packages/testkit-go --glob '!**/*_test.go'`
- [x] For any remaining hits, classify them as:
  - [x] valid first-class feature terminology
  - [x] acceptable internal abstraction with current justification
  - [x] deletion candidate
- [x] Delete all deletion candidates in this phase. Do not defer them casually.

Validation:

- [x] `rg -n 'legacy|compat|shim|bridge|adapter' internal/cli internal/app internal/scenario internal/resources cmd/vrooli packages/testkit-go --glob '!**/*_test.go'`
  Expected result: no transitional runtime code; only intentionally-supported feature terminology that is explicitly documented.

Definition of done:

- No compatibility/transitional runtime path remains in the CLI/runtime architecture.

Phase 7 completion note:

- Completed on April 14, 2026 by deleting the old scenario external runtime bridge, removing legacy dependency parsing from `internal/scenario`, deleting the transitional resource alias/bridge files, and cleaning the remaining compatibility-era language out of `packages/testkit-go`.
- The current compatibility residue search over `internal/cli`, `internal/app`, `internal/scenario`, `internal/resources`, `cmd/vrooli`, and `packages/testkit-go` returns no runtime hits for `legacy|compat|shim|bridge|adapter` in non-test Go code.

## Phase 8: Rebuild the test architecture around the final seams

Objective:

- Make the test suite reflect the new architecture instead of the old binary-centered one.

Checklist:

- [x] Move parser/help/render coverage below `cmd/vrooli` wherever still misplaced.
- [x] Add composition tests for handler/runtime packages that currently have no tests.
- [x] Shrink `cmd/vrooli` tests to true smoke/integration coverage.
- [x] Break up or reduce [cmd/vrooli/scenario_main_test.go](/home/matthalloran8/Vrooli/cmd/vrooli/scenario_main_test.go) by migrating lower-layer checks to internal packages.
- [x] Add exact-output/generated-help regression tests for representative help surfaces.
- [x] Promote any repeated repo fixture builders into `packages/testkit-go` or `packages/testkit-go/vrooli`.
- [x] Remove raw fixture duplication when shared builders are appropriate.
- [x] Ensure tests assert shared constants/contracts where practical.

Validation:

- [x] `find cmd/vrooli -maxdepth 1 -type f -name '*_test.go' -print0 | xargs -0 wc -l`
  Result should be materially smaller than the current `3,470` LOC baseline.
- [x] `find internal/cli -type f -name '*_test.go' -print0 | xargs -0 wc -l`
  Result should be materially larger than the current `985` LOC baseline.
- [x] `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...`

Definition of done:

- Most command correctness is validated below the binary package.

Phase 8 completion note:

- Completed on April 14, 2026 by moving parser, render, template, runtime-helper, and log-helper tests from `cmd/vrooli` into the owning packages under `internal/cli/*`, `internal/scenarioexec`, and `internal/cli/vroolicli`.
- New direct tests now cover:
  - `internal/cli/vroolicli`
  - `internal/cli/scenariohandlers` log/tool helpers
  - `internal/cli/scenariocli` parser/render/template helpers
  - `internal/cli/topcli` info/help helpers
  - `internal/cli/rootcli` shared flag/exit-code helpers
- `cmd/vrooli` test LOC dropped from `3,611` to `2,947`.
- `internal/cli` test LOC increased from `985` to `2,075`.
- The focused validation suite is green after the seam shift:
  - `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./internal/scenarioexec ./packages/testkit-go/...`

## Phase 9: Thin `cmd/vrooli-api` to the same standard

Objective:

- Apply the same thin-binary standard to the API entrypoint.

Checklist:

- [ ] Move API app construction and runtime wiring out of `cmd/vrooli-api/main.go` where possible.
- [ ] Move endpoint wrapper fan-out out of `cmd/vrooli-api/main.go`.
- [ ] Keep only bootstrap/startup logic in the binary.
- [ ] Add or move tests so API composition is validated below `cmd/vrooli-api` where possible.

Validation:

- [ ] `go test ./cmd/vrooli-api ./internal/api`
- [ ] Inspect `cmd/vrooli-api/main.go` and confirm it is bootstrap-only by the same standard as `cmd/vrooli`.

Definition of done:

- Both binaries follow the same architectural rule.

## Phase 10: Final deletion pass and architectural verification

Objective:

- Verify that the target state is actually reached and remove any leftover churn.

Checklist:

- [ ] Review `cmd/vrooli` production files and confirm they are still bootstrap-only.
- [ ] Review `cmd/vrooli-api` production files and confirm they are bootstrap-only.
- [ ] Review the largest internal CLI files and confirm they were materially reduced or split appropriately.
- [ ] Delete stale wrapper helpers, alias files, and transitional utilities that became unnecessary during the refactor.
- [ ] Update this plan with actual completion notes only after the full validation matrix is green.

Validation:

- [ ] `find cmd/vrooli -maxdepth 1 -type f -name '*.go' ! -name '*_test.go'`
- [ ] `find cmd/vrooli-api -maxdepth 1 -type f -name '*.go' ! -name '*_test.go'`
- [ ] `rg -n 'internal/cli/' internal/app --glob '!**/*_test.go'`
  Expected result: no matches.

Definition of done:

- The architecture looks intentional and greenfield, not post-migration.

---

## 8. Required deletion targets

These are explicit deletion or hard-justification targets.

### 8.1 Compatibility and transitional runtime files

- [ ] [internal/cli/scenariohandlers/external_runtime.go](/home/matthalloran8/Vrooli/internal/cli/scenariohandlers/external_runtime.go)
  Default expectation: delete after replacing the behavior natively.

### 8.2 Runtime compatibility branches and legacy parsing

- [ ] legacy dependency parsing in [internal/scenario/scenario.go](/home/matthalloran8/Vrooli/internal/scenario/scenario.go)
- [ ] compatibility bridge behavior in [internal/resources/env/resolver.go](/home/matthalloran8/Vrooli/internal/resources/env/resolver.go)

### 8.3 Alias/bridge wrappers requiring hard justification or deletion

- [ ] [internal/resources/driver_bridge.go](/home/matthalloran8/Vrooli/internal/resources/driver_bridge.go)
- [ ] [internal/resources/manifest_aliases.go](/home/matthalloran8/Vrooli/internal/resources/manifest_aliases.go)
- [ ] [internal/resources/catalog_aliases.go](/home/matthalloran8/Vrooli/internal/resources/catalog_aliases.go)

### 8.4 Obsolete compatibility-era docs/testkit language

- [ ] compatibility-oriented language in [packages/testkit-go/README.md](/home/matthalloran8/Vrooli/packages/testkit-go/README.md)
- [ ] compatibility-oriented language in [packages/testkit-go/PLAN.md](/home/matthalloran8/Vrooli/packages/testkit-go/PLAN.md)

If any of the above remain at completion, they must have an explicit contemporary reason, not a migration story.

---

## 9. Validation matrix

This matrix is required for completion.

### 9.1 Core validation

- [ ] `go test ./cmd/vrooli ./internal/cli/... ./internal/app/... ./packages/testkit-go/...`
- [ ] `go test ./cmd/vrooli-api ./internal/api`
- [ ] `go test ./...`

### 9.2 Focused CLI behavior validation

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

### 9.3 Help and parser contract validation

- [ ] Golden tests cover root help and representative family/leaf help.
- [ ] Shared parser tests cover standard no-args/single-name/flag/value patterns.
- [ ] Unknown-command and unknown-option behavior is covered at shared-policy seams.

### 9.4 Architectural validation

- [ ] `cmd/vrooli` is bootstrap-only by inspection.
- [ ] `cmd/vrooli-api` is bootstrap-only by inspection.
- [ ] No runtime `internal/app/*` package imports `internal/cli/*`.
- [ ] No compatibility/transitional runtime path remains in the CLI/runtime architecture.
- [ ] Shared fixtures come from `packages/testkit-go` when repetition justifies it.

### 9.5 Size and seam validation

- [ ] `cmd/vrooli` production LOC remains roughly at or below the current `118` LOC baseline.
- [ ] `cmd/vrooli` tests are materially smaller than the current `3,470` LOC baseline.
- [ ] `internal/cli` tests are materially larger than the current `985` LOC baseline.
- [ ] `internal/cli/rootcli/rootcli.go` and `internal/cli/vroolicli/runtime.go` are materially smaller than today.
- [ ] No handler package file remains a giant command map without strong justification.

### 9.6 Optional higher-level validation

- [ ] `make validate`

If any validation remains blocked, that blocker must be fixed or explicitly documented before this plan can be closed.

---

## 10. Definition of done

This effort is complete only when all of the following are true:

- [ ] `cmd/vrooli` is still thin and bootstrap-only.
- [ ] `cmd/vrooli-api` is also bootstrap-only.
- [ ] The internal CLI stack uses one declarative command-definition model.
- [ ] Standard help is generated programmatically from command metadata and arg specs.
- [ ] Standard parsing uses shared declarative parser primitives instead of repeated switch loops.
- [ ] Handler/runtime binding is materially consolidated and no longer duplicated across families.
- [ ] `internal/app/*` remains CLI-agnostic.
- [ ] Stable help/error/output/path contracts are declared once and reused.
- [ ] No compatibility/transitional runtime path remains.
- [ ] No dead alias/shim/wrapper files remain without explicit current justification.
- [ ] Binary-package tests are smoke-oriented.
- [ ] Handler/runtime packages have direct tests.
- [ ] Shared fixture patterns are provided by `packages/testkit-go`.
- [ ] `go test ./...` is green.
- [ ] The validation matrix above is green.

If any item above is false, the overhaul is not done.

---

## 11. Immediate next slice

The next highest-value implementation slice is:

1. consolidate the command-definition model and shared arg-spec layer
2. use that to unify help generation and standard parser behavior
3. then collapse the oversized handler/runtime files onto the new declarative path

That sequence matters. If the repo starts by only splitting files again, it will keep the same verbosity under different filenames.
