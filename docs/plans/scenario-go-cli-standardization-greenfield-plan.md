# Scenario Go CLI Standardization Greenfield Plan

This plan covers the repo-wide standardization of scenario Go CLIs after the manifest cutover and template redesign.

The objective is not just to make new templates cleaner. The objective is to make scenario Go CLIs across the repo converge on one platform-standard shape:

- manifest-driven CLI contract
- `cli-core` as the shared substrate
- `NewStandardScenarioApp(...)` as the default bootstrap
- built-in health/status where the scenario does not need specialized diagnostic behavior
- domain-package CLI architecture as the default growth path
- human-first output contracts implemented in code, not left as prose-only guidance

This is a **greenfield standardization effort**. It must be treated as a breaking cleanup, not an additive compatibility layer.

## Greenfield Constraint

This work is explicitly **greenfield**.

That means:

- do **not** preserve legacy CLI bootstrap patterns just because existing scenarios use them today
- do **not** add compatibility wrappers, fallback registration paths, or “both old and new” patterns
- do **not** keep old `apiPath(...)`, custom health boilerplate, or manual output helpers around once a scenario is migrated
- do **not** add docs that describe legacy CLI patterns as acceptable alternatives
- do **not** preserve `cmd_<domain>.go` as the preferred long-term architecture for scenarios expected to grow

Agents implementing this plan should make the new standard the only documented and recommended standard.

If a scenario is migrated in a phase below, the migration should remove legacy CLI scaffolding in that scenario rather than layering the new approach on top of it.

## Why This Exists

The repo now has a stronger CLI substrate than it had before:

- scenario CLI behavior is manifest-defined
- `cli-core` has a higher-level `NewStandardScenarioApp(...)` constructor
- `cli-core` exposes direct request helpers: `Get(...)`, `Request(...)`, `GetRoot(...)`, `RequestRoot(...)`
- `cli-core` now has reusable human output renderers:
  - `RenderOperationalReport(...)`
  - `RenderListReport(...)`
  - `RenderMutationReport(...)`
  - `PrintReportJSON(...)`
- built-in `status` now follows the operational output contract by default
- scenario templates now default to domain-package architecture

Without a repo-wide migration plan, the likely failure mode is platform drift:

- templates teach one architecture while existing Go CLIs keep another
- `cli-core` gains standard output/rendering helpers but scenarios keep hand-rolled output
- some CLIs use built-in status while others keep custom health boilerplate for no reason
- agents continue copying older CLI shapes from nearby scenarios instead of the new standard

This plan exists to prevent that split-brain state.

## Related Decisions And Plans

- [Scenario CLI Manifest Decision](/home/matthalloran8/Vrooli/docs/strategy/scenario-cli-manifest-decision.md:1)
- [Scenario CLI Manifest Greenfield Migration Plan](/home/matthalloran8/Vrooli/docs/plans/scenario-cli-manifest-greenfield-migration-plan.md:1)
- [CLI Core README](/home/matthalloran8/Vrooli/packages/cli-core/README.md:1)
- [CLI Steer Skill](/home/matthalloran8/Vrooli/scenarios/prompt-manager/store/skills/packs/core/cli-steer/SKILL.md:1)

## Target End State

At the end of this plan, scenario Go CLIs should converge on this standard:

### Bootstrap

- `main.go` is entrypoint only
- `app.go` contains metadata + `NewStandardScenarioApp(...)`
- standard `status` + `configure` come from `cli-core`
- scenarios only drop to `NewScenarioApp(...)` when they have a real need for lower-level control

### Architecture

- domain packages are the default shape
- top-level CLI package contains only bootstrap and aggregation
- domain registrations are aggregated from `path:cli/domains/domains.go`
- command-rich scenarios prefer `SubcommandGroup`
- CLI-only helper code lives under `path:cli/internal/...`

Target structure:

```text
cli/
├── main.go
├── app.go
├── domains/
│   ├── domains.go
│   ├── <domain>/
│   │   ├── register.go
│   │   ├── list.go
│   │   ├── get.go
│   │   ├── create.go
│   │   ├── update.go
│   │   ├── delete.go
│   │   ├── output.go
│   │   └── types.go
└── internal/
    ├── output/
    ├── flags/
    └── client/
```

### API Access

- use `core.Get(...)` / `core.Request(...)` for versioned API routes
- use `core.GetRoot(...)` / `core.RequestRoot(...)` for root endpoints such as `/health`
- do not keep per-scenario `apiPath(...)` duplication after migration unless a scenario has a true nonstandard API prefix concern

### Output Contracts

Default human output must follow command intent:

- operational commands: `Status -> Triage -> Next Steps`
- list/read commands: `Summary -> Results -> Retrieval Hints`
- mutation commands: `Result -> What Changed -> Next Command`

Render those contracts through `cli-core`, not custom ad hoc formatting, unless the scenario genuinely needs a richer operator surface.

### Status / Health

- use built-in `StandardStatusCommand()` by default
- keep custom `status` only if the scenario has domain-specific diagnostic behavior that materially exceeds generic health reporting
- even custom status should use `RenderOperationalReport(...)` where possible

## Scope

### In Scope

- every scenario with `cli.adapter.kind = "go_module"`
- `cli-core` helpers, docs, and tests needed for convergence
- template standardization and module wiring
- `prompt-manager` skill guidance updates
- repo-wide migration inventory and phased execution

### Out Of Scope

- shell CLI migration to Go, except where already separately planned
- API redesign
- scenario business logic changes unrelated to CLI standardization
- preserving existing CLI output quirks when they conflict with the new standard and are not part of a deliberate operator-facing contract

## Current State Summary

Current Go scenario CLIs fall into several groups:

### Group A: Nonstandard / Hand-Rolled

- `elo-swipe`
- `local-info-scout`

These should be treated as rewrites onto the standard template.

### Group B: Thin Or Near-Template

- `llm-evaluator`
- `web-console`
- `vrooli-onboarding`
- `reference-react-vite`

These are the best first migration wave.

### Group C: Monolithic `app.go` CLIs

- `brand-manager`
- `development-toolchain-validator`
- `git-control-tower`
- `knowledge-observatory`
- `landing-page-business-suite`
- `lifestyle-dashboard`
- `tunnel-manager`
- `workspace-sandbox`

These need substrate convergence first, then domain extraction.

### Group D: Already Domainized Or Domain-Leaning

- `browser-automation-studio`
- `deployment-manager`
- `ecosystem-manager`
- `prompt-manager`
- `scenario-to-cloud`
- `scenario-to-desktop`
- `test-genie`

These mostly need shared-substrate adoption and output-contract convergence, not architecture invention.

### Group E: Split By Commands But Not Yet On The New Standard

- `scenario-completeness-scoring`
- `stream-of-consciousness-analyzer`
- `swarm-manager`
- `visited-tracker`
- `vrooli-events`

## Progress Snapshot (2026-04-15)

The repo is no longer at the untouched-starting-point described above. Current progress against this plan:

- Phase 0 is complete:
  - docs, templates, and `cli-steer` consistently teach the new domain-package + `cli-core` standard
- Phase 1 is complete:
  - `cli-core` now provides `NewStandardScenarioApp(...)`, built-in base commands, request helpers, and human-output contract renderers
- Phase 2 is complete:
  - scenario templates use built-in `status` / `configure`, default to domain-package structure, and compile cleanly as generated modules
- Phase 3 is complete:
  - `llm-evaluator`
  - `web-console`
  - `vrooli-onboarding`
  - `reference-react-vite`
- Phase 4 is complete:
  - completed target batch:
    - `brand-manager`
    - `development-toolchain-validator`
    - `knowledge-observatory`
    - `git-control-tower`
    - `landing-page-business-suite`
    - `lifestyle-dashboard`
    - `tunnel-manager`
    - `workspace-sandbox`
  - completed Phase 4 outcomes:
    - standardized bootstrap on `NewStandardScenarioApp(...)`
    - built-in `status` / `configure` used where appropriate
    - justified richer custom `status` surfaces moved onto `RenderOperationalReport(...)` where retained
    - monolithic `app.go` command registration replaced with domain aggregation
    - human output converged onto `cli-core` operational/list/mutation contracts across migrated command surfaces
  - current exemplars:
    - `brand-manager`
    - `development-toolchain-validator`
    - `workspace-sandbox`
- Phase 5 is complete:
  - completed:
    - `browser-automation-studio`
    - `deployment-manager`
    - `ecosystem-manager`
    - `prompt-manager`
    - `scenario-to-cloud`
    - `scenario-to-desktop`
    - `test-genie`
  - completed Phase 5 outcomes:
    - already-domainized CLIs now use `NewStandardScenarioApp(...)` where appropriate without preserving legacy top-level alias surfaces
    - built-in `status` / `configure` replace scenario-local health/config plumbing when no richer diagnostic surface is justified
    - richer custom `status` surfaces, such as `browser-automation-studio`, now render through `RenderOperationalReport(...)` instead of ad hoc headings
    - command registration is aggregated from `path:cli/domains/...` so the app bootstrap matches the template standard even when domain implementations remain command-rich
    - greenfield migration means legacy alias entrypoints are removed and tests are updated to assert the canonical command surface instead
- Phase 6 is complete:
  - completed target batch:
    - `agent-manager`
    - `prd-control-tower`
    - `scenario-completeness-scoring`
    - `stream-of-consciousness-analyzer`
    - `swarm-manager`
    - `visited-tracker`
    - `vrooli-events`
  - completed Phase 6 outcomes:
    - command-split legacy shapes moved onto `NewStandardScenarioApp(...)`
    - domain packages became the primary registration/organization model
    - repeated `apiPath(...)` and request-helper duplication was removed where migrated
    - human output contracts were adopted broadly while preserving explicitly richer operator UX where justified
- Phase 7 is complete:
  - completed:
    - `elo-swipe`
    - `local-info-scout`
  - completed Phase 7 outcomes:
    - the final hand-rolled Go CLIs were rewritten onto `NewStandardScenarioApp(...)`
    - greenfield command surfaces now use domain aggregation instead of bespoke flag-only bootstrap code
    - install wiring now flows through the shared `cli-core` installer instead of scenario-local binary-copy scripts
    - remaining work is now validation/audit, not bootstrap migration
- Phase 8 is complete:
  - repo-wide CLI `go test ./...` audit passed for:
    - `path:packages/cli-core`
    - every `go_module` scenario CLI (`28/28`)
  - stale example cleanup completed:
    - `cli-steer` now demonstrates `NewStandardScenarioApp(...)` for the canonical stale-detection/bootstrap snippet
  - structural audit completed:
    - `0` `NewScenarioApp(...)` references remain under `path:scenarios/*/cli`, including tests
    - `0` production `apiPath(...)`, `getV1(...)`, or `requestV1(...)` references remain under `path:scenarios/*/cli`
    - docs/templates/skills continue to describe domain packages as the greenfield default rather than flat `cmd_<domain>.go`
  - remaining non-CLI validation caveats:
    - broader scenario-level `vrooli scenario test ...` coverage is still partially blocked by pre-existing non-CLI runtime issues such as:
      - `scenario-to-desktop` hitting the unrelated `ollama` startup/template parsing failure
      - `prompt-manager` reaching the shared `test-genie` execution-plan failure (`failed to build execution plan`)
    - those issues are outside the scope of CLI standardization and no longer block closure of this plan

Repo-wide current state summary:

- `28` scenarios currently declare `cli.adapter.kind = "go_module"`
- `28` already use `NewStandardScenarioApp(...)`
- `0` still use legacy `NewScenarioApp(...)`
- `0` are still hand-rolled custom CLIs

Notes for follow-on agents:

- treat `brand-manager`, `development-toolchain-validator`, and `workspace-sandbox` as the current strongest migration exemplars
- treat `agent-manager`, `prd-control-tower`, `stream-of-consciousness-analyzer`, and `vrooli-events` as current split-command/domain-package exemplars
- do not regress migrated scenarios back toward monolithic `app.go` command implementations or scenario-local request/status scaffolding
- the CLI standardization work is complete; follow-on work should treat remaining failures as scenario/runtime issues, not bootstrap migration tasks

## Non-Negotiable Rules

1. New standard only.
   If a scenario is migrated, it should use the new standard directly. Do not keep both old and new registration/bootstrap styles in the same scenario.

2. `cli-core` first.
   Repeated formatting logic, request helpers, or bootstrap code should be moved into `cli-core` when it is clearly cross-scenario and stable.

3. Domain packages by default.
   `cmd_<domain>.go` is no longer the recommended target architecture. It is acceptable only as a temporary waypoint or for genuinely tiny CLIs.

4. Human-first by default.
   `--json` remains important, but default output is the product surface. The default output contract must be deliberate, consistent, and operator-friendly.

5. No speculative compatibility layers.
   If a scenario needs a custom status or output surface, implement that custom behavior explicitly; do not preserve old code “just in case.”

## Phase 0: Freeze The Standard

Goal: make the target standard explicit before broad migration begins.

Tasks:

1. Confirm `cli-core` surface is the canonical bootstrap:
   - `NewStandardScenarioApp(...)`
   - `StandardBaseCommandGroups(...)`
   - request helpers
   - output renderers

2. Confirm template structure is canonical:
   - `main.go`
   - `app.go`
   - `domains/domains.go`
   - `domains/<domain>/...`
   - `path:internal/...` as needed

3. Confirm CLI steer and template README language reject old shapes as defaults.

4. Confirm repo policy:
   - no fallback bootstrap guidance
   - no “either flat files or domains are equally preferred” language
   - no continued recommendation of custom status boilerplate when built-in status is sufficient

Acceptance criteria:

- docs and skills consistently teach one preferred architecture
- no repo documentation presents the previous flat-file CLI shape as the standard for growing scenarios

## Phase 1: Harden `cli-core` As The Only Shared Substrate

Goal: ensure the shared package is strong enough that scenario migrations remove boilerplate instead of just moving it around.

Tasks:

1. Keep and expand the new output renderers:
   - operational
   - list/read
   - mutation

2. Add tests that lock the human output contracts down:
   - section ordering
   - headings
   - empty-state behavior
   - `--json` parity behavior where applicable

3. Review whether any repeated scenario helpers should move into `cli-core`:
   - request helper wrappers
   - common next-step printing patterns
   - maybe common list row rendering helpers if repetition becomes obvious

4. Ensure built-in status remains stable and human-first.

5. Document all relevant helpers in `path:packages/cli-core/README.md`.

Acceptance criteria:

- all new shared output helpers are tested
- `cli-core` docs describe the supported bootstrap and output-contract surfaces
- no duplicated bootstrap logic is required in new templates

## Phase 2: Lock Templates As The Only Greenfield Source Of Truth

Goal: ensure newly generated scenarios always start from the right architecture.

Tasks:

1. Keep templates on `NewStandardScenarioApp(...)`.

2. Keep templates on built-in `status` / `configure`.

3. Keep templates on domain-package structure by default.

4. Keep template `go.mod` and `go.sum` ready for immediate `go test .` after generation.

5. Keep README guidance explicit about:
   - domain packages
   - request helpers
   - human output contracts
   - `NeedsAPI: true`

6. Optionally add a tiny documented example domain package in a follow-up if agents still struggle to extend generated CLIs correctly.

Acceptance criteria:

- temp-instantiated templates compile without `go mod tidy`
- template docs clearly state the preferred extension model
- no generated scenario needs to invent its own bootstrap or output contract from scratch

## Phase 3: Migrate Easy Thin CLIs First

Goal: create clean, low-risk exemplars for the rest of the repo.

Target batch:

- `llm-evaluator`
- `web-console`
- `vrooli-onboarding`
- `reference-react-vite`

Tasks per scenario:

1. Replace `NewScenarioApp(...)` wiring with `NewStandardScenarioApp(...)`.

2. Remove custom `status` implementation and use built-in status.

3. Remove manual `apiPath(...)` helpers and use `core.Get(...)` / `core.Request(...)`.

4. Move command registration to domain aggregation if the scenario has more than one logical area.

5. Use output renderers for any non-status commands added or retained.

6. Remove now-dead helper structs/functions for status rendering.

Acceptance criteria:

- each scenario compiles and passes its CLI tests
- status output comes from `cli-core`
- no leftover manual status boilerplate remains

## Phase 4: Migrate Medium Monoliths

Goal: converge the most repetitive older Go CLIs onto the new substrate and start architectural cleanup.

Target batch:

- `brand-manager`
- `development-toolchain-validator`
- `knowledge-observatory`
- `git-control-tower`
- `landing-page-business-suite`
- `lifestyle-dashboard`
- `tunnel-manager`
- `workspace-sandbox`

Tasks per scenario:

1. Adopt `NewStandardScenarioApp(...)`.

2. Decide status strategy explicitly:
   - built-in status if generic health is enough
   - custom status only if richer domain diagnostics are truly necessary

3. Replace `APIClient.Get/Request + apiPath(...)` call sites with `core.Get/Request` where behavior matches.

4. Split monolithic `app.go` command logic into domain packages.

5. Migrate human output:
   - mutation commands to `RenderMutationReport(...)`
   - list/read commands to `RenderListReport(...)`
   - diagnostic commands to `RenderOperationalReport(...)`

6. Delete scenario-local helper code that becomes obsolete.

Acceptance criteria:

- no migrated scenario keeps `apiPath(...)` unless it has a documented exception
- domain boundaries are visible in CLI layout
- output rendering is visibly more standardized

## Phase 5: Converge Already-Domainized CLIs

Goal: avoid over-refactoring good architecture while still standardizing the shared substrate.

Target batch:

- `browser-automation-studio`
- `deployment-manager`
- `prompt-manager`
- `scenario-to-cloud`
- `scenario-to-desktop`

Tasks per scenario:

1. Replace bootstrap with `NewStandardScenarioApp(...)` where possible.

2. Keep existing domain package layout when it already expresses the domain well.

3. Replace local request helper duplication with `core.Get/Request` where possible.

4. Standardize status if the current custom status is not materially better than the built-in one.

5. Migrate hand-rolled output sections to shared renderers where it reduces duplication without weakening operator UX.

Acceptance criteria:

- these CLIs keep their existing domain shape where it is already good
- shared boilerplate is reduced substantially
- output contracts are intentionally aligned instead of incidental

Progress update:

- completed:
  - `ecosystem-manager`
  - `scenario-to-cloud`
  - `scenario-to-desktop`
  - `test-genie`
- remaining:
  - `browser-automation-studio`
  - `deployment-manager`
  - `prompt-manager`

## Phase 6: Migrate Command-Split Legacy Shapes

Goal: turn “split by files” CLIs into true domain-architected CLIs.

Target batch:

- `agent-manager`
- `prd-control-tower`
- `scenario-completeness-scoring`
- `stream-of-consciousness-analyzer`
- `swarm-manager`
- `visited-tracker`
- `vrooli-events`

Tasks per scenario:

1. Adopt `NewStandardScenarioApp(...)`.

2. Re-home command files into domain packages where it improves clarity.

3. Keep `swarm-manager` and similarly rich CLIs on custom output only where necessary; otherwise use shared output renderers as building blocks.

4. Standardize any repeated “Next Steps” printing patterns through the new output contract helpers.

5. Keep `SubcommandGroup` as the default registration shape for command-rich domains.

Acceptance criteria:

- scenarios no longer read as transport-verb file dumps
- domain folders express the CLI’s business surface
- shared output contract helpers replace repeated ad hoc section printers where possible

## Phase 7: Rewrite Nonstandard Outliers

Goal: remove the last two Go CLIs that still bypass the platform standard almost entirely.

Target batch:

- `elo-swipe`
- `local-info-scout`

Tasks:

1. Rewrite onto the standard template shape.

2. Replace lifecycle/manual env checks with `cli-core` bootstrap.

3. Re-express commands as API-thin wrappers.

4. Use built-in status and standard output contracts.

5. Add smoke tests for the new CLI surface.

Acceptance criteria:

- these scenarios are no longer special cases
- they compile and behave like first-class scenario Go CLIs

## Phase 8: Validation And Audit Sweep

Goal: prove repo-wide convergence and reject half-migration.

Validation categories:

### A. Shared Package Validation

- `cd packages/cli-core && go test ./...`

### B. Scenario CLI Validation

For each migrated scenario:

- `cd scenarios/<name>/cli && go test ./...`
- run scenario-specific CLI smoke tests if present
- validate `status`
- validate at least one list/read command
- validate at least one mutation command if the CLI has one

### C. Template Validation

- repo-shaped temp instantiation for each template
- placeholder substitution
- `go test .` in the generated CLI module

### D. Structural Audit

Run grep-style audits to confirm convergence:

- no migrated scenario still uses custom status boilerplate when built-in status would suffice
- no migrated scenario still keeps dead `apiPath(...)` helpers after moving to `core.Get/Request`
- no docs describe flat `cmd_<domain>.go` as the preferred architecture for growing scenarios
- `cli-steer` remains aligned with the current template and `cli-core`

### E. Output Contract Audit

For migrated commands, verify:

- default human output is primary
- `--json` exists where machine-readable output is needed
- command output matches the correct contract family

Acceptance criteria:

- repo-standard bootstrap is in use across migrated Go CLIs
- templates and skills match actual implementation
- no migrated scenario depends on retained legacy CLI scaffolding

## Recommended Execution Order

1. Phase 0: freeze the standard
2. Phase 1: harden `cli-core`
3. Phase 2: keep templates and skills aligned
4. Phase 3: easy thin CLIs
5. Phase 4: medium monoliths
6. Phase 5: already-domainized CLIs
7. Phase 6: command-split legacy shapes
8. Phase 7: rewrite outliers
9. Phase 8: validation and audit sweep

## Suggested PR / Batch Strategy

To keep risk manageable, do not migrate all Go CLIs in one PR.

Recommended batching:

- PR 1: `cli-core` + docs + template substrate
- PR 2: thin CLI batch
- PR 3: medium monolith batch A
- PR 4: medium monolith batch B
- PR 5: already-domainized batch A
- PR 6: already-domainized batch B
- PR 7: command-split batch A
- PR 8: command-split batch B
- PR 9: nonstandard outlier rewrites
- PR 10: repo audit + cleanup pass

## Risks

### Risk 1: Half-Migration

Scenario adopts `NewStandardScenarioApp(...)` but keeps old request helpers, old status boilerplate, and old output patterns.

Mitigation:

- require each migrated scenario to remove dead scaffolding
- audit for old helper retention

### Risk 2: Over-Aggressive Status Standardization

Some scenarios have genuinely richer diagnostic workflows.

Mitigation:

- built-in status is the default, not a blind mandate
- keep custom status only with explicit justification
- still use `RenderOperationalReport(...)` when possible

### Risk 3: Structural Churn Without Value

For already-domainized CLIs, over-refactoring could add noise.

Mitigation:

- keep good domain package structures
- migrate substrate and output first, not filenames for their own sake

### Risk 4: Output Standardization Regresses Operator UX

Some command outputs are intentionally richer than the generic renderer.

Mitigation:

- use renderers as building blocks, not straightjackets
- require before/after review for high-touch operator workflows

## Final Acceptance Criteria

This plan is complete only when all of the following are true:

- scenario Go CLIs have one documented preferred bootstrap path
- templates and `cli-steer` teach only the new standard
- `cli-core` owns the shared human output contracts
- built-in status is the default path across the repo unless a scenario has a documented reason to diverge
- migrated scenarios no longer keep legacy bootstrap, status, or request-helper code “for compatibility”
- domain-package architecture is the default documented and implemented direction for growing scenario CLIs
- newly generated scenarios compile and test immediately with the standardized CLI shape

## Guidance For Agents Executing This Plan

When working on a specific scenario:

1. Read the current CLI shape before editing.
2. Decide whether the scenario is:
   - thin / easy
   - monolithic
   - already domainized
   - rewrite candidate
3. Migrate to the standard directly.
4. Remove old scaffolding instead of leaving both patterns in place.
5. Validate the scenario CLI locally.
6. Update any scenario-local CLI docs if command behavior or output contract changed materially.

Do not preserve old bootstrap or output code out of caution. This is a greenfield platform standardization effort. The correct move is to converge on the new standard, validate it, and delete obsolete paths.
