# Action Entity Implementation Plan

Status: ready for implementation.

## Implementation Progress

2026-04-30:

- Completed the Phase 1 foundation slice for Action storage:
  - Added `store/schemas/action.schema.json`.
  - Added Action models and `KindAction` to the API store package.
  - Added `FileActionStore` with pack order, list/get/create/update/archive/delete, structural contract validation, history entries, dotted Action ID validation, and unsafe command-shape rejection.
  - Wired `Actions()` / `FileActions()` into `NewFileStore` and initialized `store/actions/packs/{core,local,drafts}` plus `store/actions/_pack-order.json`.
  - Added focused store tests covering CRUD, archive, pack precedence, malformed/invalid file skipping, duplicate IDs, invalid IDs, invalid command forms, timestamps, and revision updates.
  - Added a draft core fixture at `store/actions/packs/core/team.decisions.list/action.json`.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `cd scenarios/prompt-manager && jq . store/schemas/action.schema.json >/dev/null && jq . store/actions/_pack-order.json >/dev/null && jq . store/actions/packs/core/team.decisions.list/action.json >/dev/null`

- Completed the first Phase 2 slice for Action validation:
  - Added the `api/actions/` package with validation response models, a `Service`, HTTP `Handlers`, and a `ControlledCommandResolver` seam.
  - Added `ManifestCommandResolver`, which classifies known `vrooli` root/scenario/resource command paths, known prompt-manager command groups, and manifest-owned scenario/resource CLI binaries without treating first-token ownership as sufficient command certainty.
  - Wired `POST /api/v1/actions/{id}/validate` in `api/main.go`.
  - Added tests for unsafe command forms, unknown subcommands, manifest-owned owner-only certainty, active-vs-draft behavior, recursive validation hooks, default value validation, and the validate handler.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`

- Completed the Phase 3 API CRUD slice, excluding execution:
  - Added Action list/get/create/update/delete/archive handlers in `api/actions/handlers.go`.
  - Added service methods and request/response models for Action CRUD.
  - Wired `GET /api/v1/actions`, `GET /api/v1/actions/{id}`, `POST /api/v1/actions`, `PUT /api/v1/actions/{id}`, and `DELETE /api/v1/actions/{id}` in `api/main.go`.
  - Kept `DELETE` conservative: it archives by default and only hard-deletes with `?hard=true`.
  - Added pack/status/owner/tag filtering for `GET /api/v1/actions`.
  - Mutations run Action validation before persistence and invalidate the graph index after successful changes.
  - Added handler tests for success, not found, malformed JSON, validation failures, and pack/status/tag filters.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`

- Completed an additional Phase 2 validation-hardening slice:
  - Exported shared structural argv validation from the Action store layer and reused it for owner-specific validation hooks.
  - Store validation now rejects invalid `execution.outputMode`, invalid `validation.mode`, and unsafe `validation.argv` contracts before persistence.
  - Action validation now resolves validation hooks through the controlled-command resolver, rejects unknown hooks for active Actions, rejects destructive/admin validation hooks, and catches both `prompt-manager action validate <same-id>` and `prompt-manager actions validate <same-id>` self-recursion.
  - Integer default validation now enforces declared `min` / `max` bounds.
  - Added focused tests for unsafe validation hooks, unknown validation hooks, destructive validation hooks, invalid runtime metadata, and integer default bounds.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`

- Completed another Phase 2 resolver/permission hardening slice:
  - Replaced broad prompt-manager command-group validation with subcommand-level command certainty for current prompt-manager CLI groups, including `action` / `actions` routes planned in this implementation.
  - Added path-specific effect classification for read/write/destructive prompt-manager subcommands so unknown or mutating subcommands no longer inherit generic read certainty.
  - Expanded Action permission declarations and schema with `apiRead`, `apiWrite`, `processStart`, `processStop`, `hostConfigure`, `secretRead`, and `secretWrite`.
  - Permission alignment now enforces API, process, host, and secret permissions in addition to filesystem/network/destructive declarations.
  - Added tests proving prompt-manager subcommand classification, unknown subcommand rejection, and API/process permission failures.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `jq . scenarios/prompt-manager/store/schemas/action.schema.json >/dev/null`

- Completed the first Phase 3 CLI/docs slice:
  - Added `prompt-manager action list`, `prompt-manager action show <id>`, and `prompt-manager action validate <id>` as thin API clients in `cli/actions`.
  - Registered the `action` / `actions` command group in the CLI domain bootstrap.
  - Added CLI unit tests for command registration, list filters, show path routing, validate routing, and invalid validation results.
  - Updated CLI/API parity coverage for the implemented Action routes and marked Action create/update/delete CLI wrappers as `audit-pending`.
  - Updated the Action concept, API reference, CLI reference, docs manifest, and parity audit so they no longer describe storage/API/validate/list/show as purely proposed.
- Verified with:
  - `cd scenarios/prompt-manager/cli && go test ./actions ./domains ./parity`
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`

- Completed the second Phase 3 CLI/docs slice:
  - Added `prompt-manager action create --file=action.json [--pack=...]`, `prompt-manager action update <id> --file=action.json`, and `prompt-manager action delete <id> [--yes] [--hard]` as thin API clients.
  - Kept create/update contract input file-based so the CLI does not duplicate Action schema validation or invent an interactive editor.
  - Added CLI unit tests for create payloads, update routing, archive-by-default delete, and hard-delete routing.
  - Updated CLI reference, Action concept status, and CLI/API parity coverage so Action CRUD/validation routes are covered.
- Verified with:
  - `cd scenarios/prompt-manager/cli && go test ./actions ./parity`
  - `cd scenarios/prompt-manager/cli && go test ./...`
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `git diff --check`

- Completed the first Phase 6 AI search/discovery slice:
  - Added Action vector indexing with a dedicated `AI_SEARCH_ACTION_COLLECTION` defaulting to `prompt-manager-actions`.
  - Wired Action indexing into full reindex, staleness checks, and Action create/update/delete mutation hooks.
  - Added Action text fallback search for environments without reachable Qdrant/Ollama.
  - Extended `/api/v1/discover` with opt-in `type: "skill" | "action" | "all"` while preserving legacy skill-only response shape when `type` is omitted.
  - Extended `prompt-manager discover` with `--type skill|action|all`; the CLI omits `type` for default skill discovery for API compatibility.
  - Updated Action concept, API reference, CLI reference, and configuration docs.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./aisearch ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `cd scenarios/prompt-manager/cli && go test ./discover ./actions ./parity`
  - `cd scenarios/prompt-manager/cli && go test ./...`
  - `git diff --check`

- Completed the first Phase 7 graph integration slice:
  - Added Action graph node type and `action-use` / `action-command` edge kinds.
  - Wired Action store data into graph builder/scanner startup wiring.
  - Action graph node IDs are namespaced as `action:<action-id>` so Actions can coexist with skills, teams, or agents that have the same raw ID.
  - Added `action-command` edges from Action contracts to CLI nodes and made CLI node extraction include Action command targets.
  - Added best-effort `action-use` edges for explicit `action:<id>` and `prompt-manager action run <id>` markdown references when the Action exists.
  - Added Action health factors for contract presence, command declaration, examples, and owner metadata, with default config normalization for existing graph-health configs.
  - Updated Graph, API, configuration, and Action concept docs for implemented graph behavior.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./graph`
  - `cd scenarios/prompt-manager/api && go test ./...`

- Completed the first Phase 8 UI schema/data hooks slice:
  - Added Action Zod schemas and exported Action runtime types from `ui/src/lib/schemas`, including Action contracts, validation checks, command resolution, and mutation envelopes.
  - Added Action API client methods for list/get/create/update/delete/validate in `ui/src/lib/api.ts`.
  - Added cached `actionService` wrappers and `useActionsData` React Query hook with graph health invalidation after mutations.
  - Updated UI discover schemas for opt-in mixed Skill/Action response fields without changing legacy skill-only requirements.
  - Updated UI graph schemas and graph settings/preview seams so Action nodes, `action-use`, and `action-command` edges parse and render instead of being rejected by existing graph clients.
  - Added focused tests for Action schema parsing/rejection, service cache invalidation/search behavior, and hook mutation/validation wiring.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/lib/schemas/action.schema.test.ts src/services/actionService.test.ts src/hooks/useActionsData.test.tsx src/components/graph/GraphNode.test.tsx src/components/graph/GraphSettingsContent.test.tsx src/components/search/SearchResultsList.test.tsx`
  - `cd scenarios/prompt-manager/ui && pnpm test`
  - `git diff --check`

- Completed the first Phase 9 UI surfaces slice:
  - Added an Actions sidebar tab with searchable Action rows and draft Action creation.
  - Added Action URL/selection state through `?action=<id>`.
  - Added an Action detail/editor panel with contract JSON editing, API-backed save, API-backed validation result display, archive/hard-delete controls, summary metadata, and an explicit disabled run panel.
  - Updated discover controls to request `type: skill|action|all` and updated mixed discover rendering so Action results show an Action badge, status, owner, and navigate to the Action panel.
  - Added focused UI tests for Action list behavior, Action detail validation/save/delete states, and mixed discover result rendering.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/components/action/ActionListPanel.test.tsx src/components/action/ActionEditorPanel.test.tsx src/components/search/SearchResultsList.test.tsx`
  - `cd scenarios/prompt-manager/ui && pnpm test`
  - `git diff --check`

- Completed the second Phase 9 UI surfaces slice:
  - Added typed Action contract editing on top of the JSON draft for identity, owner, status, tags, command argv tokens, inputs, outputs, permissions, and examples.
  - Kept JSON as the single draft source of truth: typed field edits patch the JSON draft, and save still goes through the existing API validation path.
  - Added invalid-JSON fallback behavior so typed fields are disabled until the draft is parseable.
  - Added focused UI coverage proving typed contract field edits are included in the saved Action payload.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/components/action/ActionEditorPanel.test.tsx`

- Completed the third Phase 9 UI surfaces slice:
  - Added accessible Action sidebar row labels, selected-row state, and keyboard focus styling.
  - Added a two-step hard-delete guard in the Action editor while preserving archive as the default lifecycle action.
  - Marked invalid Action JSON drafts with `aria-invalid`, made validation results announce with polite live-region semantics, and kept the disabled run control explicitly disabled for assistive technology.
  - Removed nested button semantics from mixed discovery rows by separating row selection from the "go to entity" control.
  - Added focused tests for the hard-delete guard, Action row labeling, Action discover badges, and select-mode navigation separation.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/components/action/ActionListPanel.test.tsx src/components/action/ActionEditorPanel.test.tsx src/components/search/SearchResultsList.test.tsx`

2026-05-01:

- Completed the first Phase 4 execution-governance slice:
  - Added governed Action run request/response models and `Service.Run`.
  - Added typed input/default validation, whole-token placeholder rendering, dry-run support, per-Action timeout capped by service maximum, process-wide concurrency throttling, run-surface eligibility checks, and `execution.runEligible` enforcement.
  - Added a direct `exec.CommandContext` runner with no shell, stdout/stderr caps, truncation flags, JSON output parsing for `execution.outputMode: "json"`, and status envelopes for completed, failed, timed-out, rejected, throttled, and dry-run results.
  - Added bounded file-backed run audit history at `runs.jsonl` with rendered argv, status, exit code, duration, output snippets, truncation flags, error, and validation outcome.
  - Wired `POST /api/v1/actions/{id}/run` in the API while leaving CLI/UI run surfaces disabled.
  - Updated Action concept/API/CLI docs and parity coverage to reflect that only the API run surface exists.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `cd scenarios/prompt-manager/cli && go test ./parity`
  - `jq . scenarios/prompt-manager/cli/parity/coverage.json >/dev/null`
  - `git diff --check`

- Completed the second Phase 4 execution-governance slice:
  - Added `prompt-manager action run <id>` as a thin API client over `POST /api/v1/actions/{id}/run`.
  - Added inline `--input` and file-backed `--input-file` JSON object parsing, with mutual exclusion and preflight shape validation before the API call.
  - Added `--dry-run` support so callers can request API validation and rendered argv without process start.
  - Added text and `--json` output for the run response envelope, while preserving non-zero CLI errors for `failed`, `timed-out`, `rejected`, and `throttled` statuses.
  - Updated CLI reference, API/concept status text, and CLI/API parity coverage for the run route.
- Verified with:
  - `cd scenarios/prompt-manager/cli && go test ./actions ./parity`
  - `cd scenarios/prompt-manager/cli && go test ./...`
  - `jq . scenarios/prompt-manager/cli/parity/coverage.json >/dev/null`
  - `git diff --check`

- Completed the first Phase 11 seed/integration slice:
  - Added a minimal active core seed Action at `store/actions/packs/core/scenario.status.show/action.json`.
  - The seed wraps `vrooli scenario status {{scenario}}` with explicit `filesystemRead` and `processStart` permissions, a scenario-typed input, an example payload, and conservative stdout execution policy.
  - Added an API/domain integration test that loads the real seed contract from the scenario store, validates it with `ManifestCommandResolver`, and exercises API `dryRun` rendering without starting a process or writing audit data into the checked-in fixture directory.
  - Updated the CLI dry-run test fixture to use the real `scenario.status.show` Action and expected rendered argv.
- Verified with:
  - `cd scenarios/prompt-manager/api && go test ./actions ./store`
  - `cd scenarios/prompt-manager/cli && go test ./actions ./parity`
  - `cd scenarios/prompt-manager && jq . store/actions/packs/core/scenario.status.show/action.json >/dev/null`
  - `cd scenarios/prompt-manager && make restart`
  - `API_PORT=18060 prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --dry-run --json`
  - `API_PORT=18060 prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --json`

- Completed the fourth Phase 9 UI surfaces slice:
  - Added Action run request/response schemas to the UI runtime schema layer.
  - Added `runAction` to the UI API client, cached Action service seam, and `useActionsData` hook.
  - Replaced the disabled Action run placeholder with a governed run panel in the Action editor.
  - The run panel supports JSON object input, first-example/default input loading, dry-run, live run, persisted-contract dirty-state blocking, destructive-permission warning text, and response rendering for status, duration, argv, stdout/stderr, parsed output, and API error messages.
  - Added focused tests for the service/hook run seams, dry-run request wiring, malformed input preflight, and disabled run controls while the local contract draft is unsaved.
  - Updated Action concept/API docs so UI run controls are no longer described as deferred.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/lib/schemas/action.schema.test.ts src/services/actionService.test.ts src/hooks/useActionsData.test.tsx src/components/action/ActionListPanel.test.tsx src/components/action/ActionEditorPanel.test.tsx`

- Completed a Phase 10 documentation and verification cleanup slice:
  - Ran the full UI regression and production build after UI Action run enablement.
  - Updated architecture, swarm, docs README, and relations references that still described Actions as proposed even though storage, API, CLI, search, graph, UI, execution, and the seed Action are now implemented.
  - Replaced directory/placeholder code-reference markers in this plan with concrete implementation file references so documentation validation no longer reports Action-plan reference warnings.
  - Updated the rollout checklist to reflect completed Action storage, API, CLI, discovery, graph, UI, docs, and seed coverage.
  - Restarted prompt-manager through the lifecycle and smoke-tested the seeded `scenario.status.show` Action through both CLI dry-run and live run.
  - Ran scenario-level validation through `make test`; it still fails in existing broad scenario phases outside the Action implementation slice: lint timeout after 30s, pre-existing broken documentation links, UI smoke iframe readiness timeout, and a BAS playbook start-url issue. Artifact root: `scenarios/prompt-manager/coverage/logs/20260501-015906-54ae534c`.
- Verified with:
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test`
  - `cd scenarios/prompt-manager/ui && pnpm run build`
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `cd scenarios/prompt-manager/cli && go test ./...`
  - `cd scenarios/prompt-manager && make restart`
  - `API_PORT=18060 prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --dry-run --json`
  - `API_PORT=18060 prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --json`
  - `cd scenarios/prompt-manager && make status`
  - `rg "Actions \\(Proposed\\)|planned contract, not currently implemented|purely proposed|proposed Action layer|proposed Action execution layer|proposed execution domain" scenarios/prompt-manager/docs --glob '!**/plans/action-entity-and-memory-promotion-rfc.md' --glob '!**/plans/action-entity-implementation-plan.md'`
  - `rg "Status: proposed" scenarios/prompt-manager/docs/concepts/ACTIONS.md scenarios/prompt-manager/docs/concepts/ARCHITECTURE.md scenarios/prompt-manager/docs/concepts/SWARM-MODEL.md scenarios/prompt-manager/docs/README.md scenarios/prompt-manager/docs/concepts/RELATIONS.md`

- Continued scenario-level validation cleanup:
  - Cleared current `make lint` failures introduced by the Action implementation by applying `gofumpt`/`golangci-lint --fix` formatting, replacing a redundant resolver struct literal with a type conversion, and fixing Action editor UI lint issues.
  - Cleared hard docs validation failures by replacing stale/broken repo links in prompt-manager PRD/team/skill docs and removing machine-specific absolute home-directory references from meta-optimization run lessons.
  - Restored the Skill editor "More actions" button to use the canonical `selectors.editor.actionsMenu` selector so BAS workflows and UI code share the registry contract again.
  - Made the `skill-save-discard` BAS playbook idempotent by stripping prior `BAS-<timestamp>` suffixes before generating the next temporary saved name; this prevents repeated playbook runs from growing the skill name past validation limits.
  - Verified the lint phase and docs phase through `test-genie execute prompt-manager lint --json` and `test-genie execute prompt-manager docs --json`; docs now pass with warnings only.
  - Rebuilt and restarted prompt-manager through the lifecycle after UI selector changes.
  - Verified isolated `smoke` and `playbooks` phases: `test-genie execute prompt-manager smoke --json` passed, and `test-genie execute prompt-manager playbooks --json` passed with all 6/6 BAS workflows green.
  - Re-ran comprehensive `cd scenarios/prompt-manager && make test`; it still fails only in the full-preset orchestration path because the prompt-manager UI process is unavailable by the smoke/playbooks phases (`net::ERR_CONNECTION_REFUSED` at `http://localhost:21235/`). This does not reproduce when the scenario is healthy and smoke/playbooks are run directly.
- Verified with:
  - `cd scenarios/prompt-manager && make lint`
  - `test-genie execute prompt-manager lint --json`
  - `test-genie execute prompt-manager docs --json`
  - `cd scenarios/prompt-manager/api && go test ./...`
  - `cd scenarios/prompt-manager/ui && pnpm run type-check`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/components/action/ActionEditorPanel.test.tsx`
  - `cd scenarios/prompt-manager/ui && pnpm test -- src/components/editor/SkillEditorPanel.test.tsx src/components/action/ActionEditorPanel.test.tsx`
  - `cd scenarios/prompt-manager && make build`
  - `cd scenarios/prompt-manager && make restart`
  - `test-genie execute prompt-manager smoke --json`
  - `test-genie execute prompt-manager playbooks --json`
  - `cd scenarios/prompt-manager && make test`

Next resume point: If continuing validation work, debug why `test-genie execute prompt-manager --preset comprehensive` loses the prompt-manager UI process before smoke/playbooks, even though `make restart`, isolated smoke, and isolated playbooks pass. Artifact root for the latest comprehensive failure: `scenarios/prompt-manager/coverage/logs/20260501-022859-c9dd11cd`. For Action-specific feature work, continue replacing the temporary command resolver with the operation-contract catalog only when that separate foundation lands.

## Purpose

Fully implement the prompt-manager Action entity described in [DOC: docs/concepts/ACTIONS.md] as a first-class, typed executable wrapper over exactly one Vrooli-controlled CLI command.

This plan covers the implementation mechanism only: schema, file-backed storage, API, CLI, AI search/discovery, UI, graph integration, validation, execution safety, tests, and documentation updates. The separate adoption plan should later update meta-optimization prompts, skills, decision contexts, notebook promotion behavior, and seed rollout policy.

Important dependency: executable Action validation depends on the operation-contract foundation in [DOC: docs/plans/operation-contract-drift-prevention-plan.md]. Do not reduce command validation to trusted first-token checks while that foundation is in progress. Draft Actions may be stored with unresolved or owner-only command certainty, but active runnable Actions should wait for the shared command catalog / controlled-command resolver described in that plan.

## Required Reading

Run these before implementing any phase:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health api-steer cli-steer interoperability-steer react-coherence boundary-of-responsibility-enforcement seam-discovery-and-enforcement utils-unification
```

Also read the prompt-manager Action docs:

```bash
sed -n '1,260p' scenarios/prompt-manager/docs/plans/operation-contract-drift-prevention-plan.md
sed -n '1,260p' scenarios/prompt-manager/docs/concepts/ACTIONS.md
sed -n '1,180p' scenarios/prompt-manager/docs/concepts/MEMORY-PROMOTION.md
sed -n '587,710p' scenarios/prompt-manager/docs/reference/api-endpoints.md
sed -n '937,1005p' scenarios/prompt-manager/docs/reference/cli-commands.md
```

## Greenfield Constraint

This is greenfield work. Do not include compatibility shims, legacy wrappers, deprecated aliases, dead code, unused re-exports, renamed `_unused` variables, or migration code for old Action formats.

There is no existing Action entity to preserve. Implement the clean target contract directly and update all prompt-manager references that still say Actions are "proposed" once the feature is complete.

## Problem Statement

Prompt-manager currently has durable first-class models for Skills, Agents, Teams, Topics, Experiments, Variants, Relations, and graph/search surfaces. It documents Actions, but there is no implemented Action registry or runtime.

The missing implementation causes these gaps:

- deterministic operational work remains trapped in prose skills or prompts
- agents cannot discover exact executable operations through prompt-manager
- there is no typed input/output contract for command-backed operations
- there is no validation gate enforcing Vrooli-controlled command ownership
- discover/search cannot return mixed Skill and Action results
- UI and graph cannot show executable capability nodes

The desired ontology remains:

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in Notebooks.
```

## Scope

In scope:

- `store/actions/packs/{core,local,drafts}/<action-id>/action.json`
- `store/schemas/action.schema.json`
- Go store models and file-backed Action store
- Action validation service with command boundary enforcement
- Action execution service for one argv-shaped command
- REST API routes under `/api/v1/actions`
- prompt-manager CLI `action` group
- AI search indexing and `prompt-manager discover --type skill|action|all`
- UI Action list/detail/editor/run/validate surfaces
- graph node/edge support for Action and Action-to-CLI relationships
- documentation, docs manifest, and reference updates
- focused unit/integration tests and scenario validation

Out of scope for this plan:

- updating meta-optimization prompts or skills to adopt Actions
- creating many production Actions beyond minimal seed fixtures needed for validation
- implementing missing scenario/resource/project CLI commands
- arbitrary shell script execution
- external-tool wrappers over raw `git`, `docker`, `psql`, `curl`, `grep`, etc.
- long-running workflow orchestration beyond running one controlled command

## Current Technical Context

Relevant existing files and seams:

- [CODE: api/main.go] wires stores, handlers, graph, search, AI search, routes, and heartbeat execution.
- [CODE: api/store/store.go] creates file-backed entity stores and store directories.
- [CODE: api/store/models.go] contains shared store entity models.
- [CODE: api/store/skill_store.go] is the closest file-backed pack store pattern.
- [CODE: api/skills/handlers.go] is the closest CRUD handler pattern with AI/graph invalidation hooks.
- [CODE: api/aisearch/service.go] owns skill/topic discovery and multi-entity AI search.
- [CODE: api/aisearch/index.go] owns entity indexing into Qdrant.
- [CODE: cli/domains/domains.go] registers CLI command groups.
- [CODE: cli/discover/discover.go] currently says discover returns skills only.
- [CODE: ui/src/lib/api.ts] is the Zod-validated UI API client.
- [CODE: ui/src/lib/schemas/action.schema.ts] contains Action runtime schemas for API responses.
- [CODE: ui/src/services/skillService.ts] and [CODE: ui/src/hooks/useSkillsData.ts] are the closest UI data patterns.
- [CODE: docs/concepts/ACTIONS.md] is the canonical Action concept contract.
- [CODE: docs/concepts/CAPABILITY-MATCHING.md] reserves "capability" for matching/requirements, not executable Actions.

Useful commands:

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
cd scenarios/prompt-manager/ui && pnpm run type-check && pnpm test
cd scenarios/prompt-manager && make test
vrooli scenario restart prompt-manager
vrooli scenario status prompt-manager
```

## Target End State

At completion:

- Actions are implemented as first-class prompt-manager entities.
- Action contracts are JSON-schema-validated and stored per entity.
- Action IDs support dotted domain names such as `scenario.ui.screenshot`.
- Each Action wraps exactly one argv-shaped Vrooli-controlled command.
- The API can list, show, create, update, delete/archive, validate, and run Actions.
- The CLI can list, show, create/update where appropriate, validate, and run Actions.
- Discover can return Skills, Actions, or both with a type discriminator.
- UI users can browse, inspect, edit, validate, and run Actions with typed inputs.
- Graph output includes Action nodes and Action-to-CLI edges.
- Tests prove invalid command forms are rejected.
- Documentation no longer calls Actions purely "proposed" after implementation.

## Contract Decisions

### Entity Naming

Use `Action`, not `Capability`. Prompt-manager already uses capabilities for agent/skill matching.

### ID Shape

Action IDs should support dotted namespaces:

```text
scenario.ui.screenshot
team.decisions.list
skill.health.audit
```

Do not reuse skill ID validation, which only supports kebab-case. Add an Action-specific validator:

```text
^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$
```

Reject empty segments, uppercase letters, underscores, path separators, shell metacharacters, and IDs over a documented max length.

### Storage Shape

Use a pack-based layout parallel to skills:

```text
store/actions/
  _pack-order.json
  packs/
    core/
      scenario.ui.screenshot/
        action.json
        history.jsonl
    local/
    drafts/
```

Store only `action.json` for the entity. Do not create a markdown body file for Actions; prose belongs in descriptions, examples, docs, or skills.

### Minimum `action.json`

Implement a strict schema based on [DOC: docs/concepts/ACTIONS.md]:

```json
{
  "kind": "action",
  "schemaVersion": 1,
  "id": "scenario.ui.screenshot",
  "name": "Take Scenario Screenshot",
  "description": "Capture a screenshot of a running scenario UI.",
  "status": "active",
  "owner": {
    "type": "scenario",
    "id": "prompt-manager"
  },
  "command": {
    "argv": ["vrooli", "scenario", "screenshot", "{{scenario}}", "--viewport", "{{viewport}}"]
  },
  "inputs": {
    "scenario": { "type": "string", "required": true },
    "viewport": {
      "type": "string",
      "enum": ["desktop", "mobile"],
      "default": "desktop"
    }
  },
  "outputs": {
    "imagePath": {
      "type": "file",
      "description": "Path to the generated screenshot."
    }
  },
  "permissions": {
    "filesystemWrite": true,
    "localhostNetwork": true
  },
  "examples": [
    {
      "description": "Capture a desktop screenshot of prompt-manager.",
      "input": {
        "scenario": "prompt-manager",
        "viewport": "desktop"
      }
    }
  ],
  "validation": {
    "mode": "contract"
  }
}
```

Recommended additional fields:

- `tags`: searchable labels
- `pack`: runtime-only, not persisted
- `createdAt`, `updatedAt`, `revision`: same file-store posture as other entities
- `execution`: optional timeout, working directory policy, output capture policy

Do not model validation as a self-referential command such as `prompt-manager action validate <same-id>`. If an owning CLI needs a command-specific validation hook, model it separately from prompt-manager's contract validation and reject hooks that recursively invoke Action validation for the same Action.

### Command Boundary

Command contracts are argv-shaped, never shell-shaped.

This boundary should be implemented through the shared operation/command catalog and controlled-command resolver from [DOC: docs/plans/operation-contract-drift-prevention-plan.md] when available. Until that plan lands, keep Action command validation conservative: allow draft persistence for owner-known commands if useful, but do not mark an Action active/runnable unless the command path, arguments, permissions, run surface, and output mode are contract-backed.

Allowed command ownership is dynamic and must be validated through a reusable resolver, not a hard-coded static list. The resolver should compose checks for:

- project lifecycle commands such as `vrooli ...`
- the prompt-manager CLI such as `prompt-manager ...`
- scenario-owned CLIs where `argv[0]` matches a discovered scenario slug/folder name and the scenario contract marks that CLI as owned by Vrooli
- resource-owned CLIs where resource metadata exposes a Vrooli-owned wrapper command

Do not merely allow any command whose first token is `vrooli` or `prompt-manager`. The validation service must classify the command target, subcommand, permission class, mutability, and execution eligibility through one central command-ownership resolver. Reuse existing scenario health or CLI-detection seams where they already determine whether a command belongs to a scenario, but keep Action validation behind an explicit `IsControlledCommand`-style API rather than duplicating detection logic.

Rejected forms:

- `sh`, `bash`, `zsh`, `fish`, `python`, `node` as arbitrary script runners
- raw `git`, `docker`, `psql`, `curl`, `grep`, `rg`
- shell metacharacters and separators such as `|`, `&&`, `;`, `>`, `<`, `` ` ``, `$()`
- multiline arguments
- empty argv
- commands with path separators in argv[0]

If a target requires an external tool, create or improve a Vrooli-controlled CLI first.

### Input Validation Boundary

Action execution is argv-only, so shell metacharacters inside rendered input values are not shell-executed by themselves. The runtime still must validate inputs by declared type before rendering placeholders:

- `string`: max length, optional regex, optional enum, no multiline values unless explicitly allowed
- `number`/`integer`: numeric bounds when declared
- `boolean`: strict boolean parsing
- `file`/`path`: no path traversal, no absolute paths unless explicitly allowed, and permission declaration required
- entity references such as scenario/team/action IDs: validate against the relevant entity ID shape and, where practical, existence

Reject shell metacharacters in static command tokens and in command strings that would imply shell interpretation. For placeholder values, rely on declared input type constraints plus per-field regex/enum/path policy so valid values are not rejected just because they contain punctuation that would only be dangerous in a shell.

### Execution Governance

Running an Action through the API turns prompt-manager into a controlled command broker. The first implementation must include explicit governance:

- default timeout and per-Action timeout maximum
- concurrency limits for Action runs, at least process-wide
- stdout/stderr byte caps with truncation flags in the response
- structured run audit history recording action ID, argv after rendering, caller context when available, status, exit code, duration, timestamps, and validation outcome
- permission enforcement before execution, not just permission display
- clear distinction between read-only, filesystem-write, localhost-network, external-network, and destructive/mutating actions
- optional API/UI run eligibility so high-risk Actions can remain discoverable but not runnable from every surface

Store enough execution history to debug failures without storing unbounded logs or secrets.

### Execution Semantics

Action execution should:

- resolve the Action through the store/API
- validate the contract before running
- validate and default typed inputs
- render placeholders only from declared inputs
- run exactly one command without a shell
- capture stdout, stderr, exit code, duration, and structured output if declared
- fail closed when validation fails
- enforce timeout
- enforce permission and concurrency policy
- append a bounded audit/history entry
- return a typed result envelope

Initial output envelope:

```json
{
  "actionId": "scenario.ui.screenshot",
  "status": "completed",
  "exitCode": 0,
  "durationMs": 1234,
  "stdout": "...",
  "stderr": "",
  "output": {
    "imagePath": "/tmp/prompt-manager-screenshot.png"
  }
}
```

Do not parse arbitrary stdout into output fields unless the Action declares an explicit output mode. Prefer JSON stdout from the owning CLI for structured outputs.

## Implementation Strategy

### Phase 1 - Schema, Models, and Store

Deliverables:

- Add [CODE: store/schemas/action.schema.json].
- Add Action models to [CODE: api/store/models.go] or a focused [CODE: api/actions/models.go] if handler/API shapes should not leak into generic store models.
- Add [CODE: api/store/action_store.go] with pack-order handling parallel to [CODE: api/store/skill_store.go].
- Extend [CODE: api/store/store.go] with Action directories, `NewFileActionStore`, `Actions()`, and `FileActions()`.
- Add tests for list/get/create/update/delete/archive behavior, pack precedence, timestamps, revision handling, malformed files, duplicate IDs, and dotted IDs.

Implementation notes:

- Prefer a concrete `FileActionStore` plus narrow interfaces consumed by actions/search/graph.
- Keep schema validation reusable and testable; do not duplicate ad hoc validators in API, CLI, and UI.
- Add `_pack-order.json` for actions rather than reusing skill pack order implicitly.

Acceptance:

- `go test ./store` passes from `scenarios/prompt-manager/api`.
- A fixture Action can be loaded from `store/actions/packs/core/<id>/action.json`.
- Invalid IDs and invalid command contracts are rejected before persistence.

### Phase 2 - Action Domain Service and Validation

Deliverables:

- Add [CODE: api/actions/service.go] and related package files.
- Implement Action CRUD request/response models.
- Implement validation service for:
  - schema fields
  - dynamic command ownership resolver
  - placeholder/input consistency
  - input type/default/enum checks
  - output declaration sanity
  - permission declaration completeness
  - validation hook safety, including no self-recursive Action validation
- Implement `POST /api/v1/actions/{id}/validate` logic without executing target operations.

Implementation notes:

- Keep command validation in one reusable package, e.g. `api/actions/command_validation.go`, backed by an explicit controlled-command resolver that can consult project, scenario, and resource command metadata.
- Action handlers should depend on interfaces, not concrete stores, following [CODE: api/skills/handlers.go].
- Validation response should be structured enough for CLI and UI to show individual checks.

Acceptance:

- Unit tests cover every rejected command form listed in this plan.
- Valid project, prompt-manager, scenario-owned, and resource-owned commands pass validation when their ownership source recognizes them.
- Commands with only a trusted first token but an unsafe or unknown subcommand fail validation.
- Self-recursive validation hooks fail validation.
- Validation errors are deterministic, specific, and usable by CLI/UI.

### Phase 3 - API Routes and Mutation Hooks

Deliverables:

- Add routes in [CODE: api/main.go]:
  - `GET /api/v1/actions`
  - `GET /api/v1/actions/{id}`
  - `POST /api/v1/actions`
  - `PUT /api/v1/actions/{id}`
  - `DELETE /api/v1/actions/{id}`
  - `POST /api/v1/actions/{id}/validate`
  - `POST /api/v1/actions/{id}/run`
- Wire Action handlers with AI indexer and graph invalidator hooks.
- Add handler tests covering success, not found, invalid JSON, validation failures, and pack/status filters.

Implementation notes:

- Static routes must precede parameterized routes where applicable.
- Do not put Action-specific logic in [CODE: api/main.go]; main should only wire dependencies and routes.
- Keep error semantics consistent with existing prompt-manager handlers while improving specificity where possible.

Acceptance:

- `go test ./actions ./store` passes.
- API tests verify JSON response shapes match [DOC: docs/reference/api-endpoints.md].

### Phase 4 - Execution Runtime

Deliverables:

- Implement execution service behind an interface in [CODE: api/actions/service.go].
- Add `POST /api/v1/actions/{id}/run`.
- Support typed input validation, default application, placeholder rendering, timeout, command execution without shell, and output envelope.
- Add dry-run or validation-only behavior if the API/CLI contract needs it; do not fake execution.

Implementation notes:

- Use `exec.CommandContext` or equivalent direct argv execution.
- Never concatenate argv into a shell string.
- Resolve working directory deliberately. Prefer repo root for project-level commands and let CLIs resolve scenario paths themselves.
- Cap stdout/stderr length to avoid huge API responses; return truncation flags if truncation occurs.
- Enforce run concurrency and permission policy before starting a process.
- Write bounded audit/history entries for success, failure, timeout, and validation rejection.
- If an Action declares JSON output, parse stdout into `output`; otherwise return stdout/stderr and leave `output` empty.

Acceptance:

- Tests prove shell metacharacters are treated as invalid, not executed.
- Tests prove timeout cancellation.
- Tests prove declared defaults are applied.
- Tests prove permissions and run eligibility are enforced before execution.
- Tests prove audit/history entries are written without storing unbounded output.
- A safe fixture command can run end to end in tests without external resources.

### Phase 5 - CLI Action Group

Deliverables:

- Add [CODE: cli/actions/actions.go].
- Register it in [CODE: cli/domains/domains.go].
- Implement:
  - `prompt-manager action list [--pack=core|local|drafts] [--status=active|draft|archived] [--owner=...] [--tag=...] [--json]`
  - `prompt-manager action show <id> [--json]`
  - `prompt-manager action validate <id> [--json]`
  - `prompt-manager action run <id> --input='{"key":"value"}' [--input-file=payload.json] [--json]`
- Consider `create`, `update`, and `delete` only if the API CRUD surface is ready and the CLI can keep the command ergonomic. If included, use explicit JSON/file inputs rather than interactive prompts for complex contracts.

Implementation notes:

- CLI should be a thin API client. Do not duplicate validation or execution logic in the CLI.
- Human-readable output should emphasize action ID, validation status, command owner, permissions, and next steps.
- JSON output should mirror API responses.

Acceptance:

- `go test ./...` passes from `scenarios/prompt-manager/cli`.
- `prompt-manager action --help` and subcommand help work without API where possible.
- Invalid input JSON produces a clear CLI error before API call.

### Phase 6 - AI Search and Discover Integration

Deliverables:

- Add Action vector collection configuration, e.g. `AI_SEARCH_ACTION_COLLECTION` defaulting to `prompt-manager-actions`.
- Add Action indexing functions in [CODE: api/aisearch/index.go].
- Add Action search result models in [CODE: api/aisearch/models.go].
- Extend [CODE: api/aisearch/service.go] `Discover` request to support `types: ["skill", "action"]`.
- Extend `prompt-manager discover` with `--type skill|action|all`.
- Preserve existing skill discovery behavior when `--type` is omitted.

Implementation notes:

- Treat Action as greenfield, but treat existing discover API/CLI behavior as compatibility-preserving. Existing skill-only clients should not need response parsing changes when `types`/`--type` is omitted.
- Implement Action indexing/search first, then mixed discover after skill-only compatibility tests are locked.
- Mixed discover results need a discriminator:
  - `type: "skill"` or `type: "action"`
  - `id`, `name`, `description`, `score`, `source`, `contentChars`
  - `readCommand` for skills and `runCommand` or `showCommand` for Actions
- For budget math, Actions should count their compact contract size, not skill body size.
- Keep topic-to-skill discovery intact; topics do not need to accumulate Actions in the first implementation unless the Action docs explicitly require it later.

Acceptance:

- Existing `prompt-manager discover "debugging" --json` remains skill-compatible.
- Existing human-readable `prompt-manager discover "debugging"` output remains skill-compatible when `--type` is omitted.
- `prompt-manager discover "list team decisions" --type action --json` returns only Actions.
- `--type all` can return mixed results with type discriminators.
- Search gracefully degrades when Qdrant/Ollama are unavailable.

### Phase 7 - Graph Integration

Deliverables:

- Add Action node type to graph models.
- Scan Action contracts from the Action store.
- Add `action-command` edges from Action to CLI target.
- Add optional `action-use` edges when skills or agent markdown references Action IDs or `prompt-manager action run <id>`.
- Update graph health scoring to recognize Actions as healthy when:
  - contract validates
  - command target is controlled
  - examples exist
  - owner exists or is intentionally project-level
- Add graph tests for Action node and edge generation.

Implementation notes:

- Prevent graph identity collisions by using namespaced graph node IDs such as `action:<action-id>` and `cli:<command-target>`, or by updating graph APIs/storage to key by `(type, id)` everywhere. Do not add raw Action IDs to the graph if they can collide with skill/team/agent/CLI IDs.
- Do not overfit scanner regexes. Prefer structured Action store data for Action nodes.
- Action references in prose can be best-effort, but Action contracts are authoritative.

Acceptance:

- Tests prove an Action and Skill with the same raw ID do not collide in graph nodes, edges, lookups, or health scores.
- `prompt-manager graph regenerate` includes Action nodes.
- Graph popovers/API node details can show Action type and validation health.

### Phase 8 - UI Schema, Data Hooks, and API Client

Deliverables:

- Add [CODE: ui/src/lib/schemas/action.schema.ts] with Zod schemas mirroring API responses.
- Export Action schemas/types from [CODE: ui/src/lib/schemas/index.ts].
- Add API client methods in [CODE: ui/src/lib/api.ts].
- Add [CODE: ui/src/services/actionService.ts] with cache/invalidation behavior parallel to skill service.
- Add [CODE: ui/src/hooks/useActionsData.ts].
- Add tests for schema parsing, service behavior, and hook mutation invalidation.

Implementation notes:

- Use Zod defaults for nil/nullable Go slices and optional objects.
- Do not reuse Skill types for Actions; the entities have different responsibilities.
- Keep client-side validation helpful but subordinate to API validation.

Acceptance:

- `pnpm run type-check` passes.
- Action schemas reject malformed API responses in tests.

### Phase 9 - UI Surfaces

Deliverables:

- Add Action browse/list surface using existing dense operational UI patterns.
- Add Action detail/editor panel:
  - identity: id, name, description, status, tags, owner
  - command argv editor
  - input schema editor
  - output schema editor
  - permissions editor
  - examples editor
  - validation result panel
- Add Action run panel:
  - typed JSON input editor
  - example picker
  - validate before run
  - result envelope with stdout/stderr/output/exit code/duration
- Update search/discover UI components to render mixed result types.
- Update selectors manifest if new test selectors are added.

Implementation notes:

- Keep operational UI dense, restrained, and scannable. This is a tool surface, not a landing page.
- Use icons for validate/run/copy where available from lucide-react.
- Avoid cards nested inside cards.
- Text must fit on mobile and desktop; long command argv lines should wrap or use horizontal code scrolling intentionally.
- Do not add decorative visual clutter.

Acceptance:

- UI tests cover list/detail validation states and mixed discover result rendering.
- New UI is keyboard-accessible for validate/run/copy flows.
- No overlapping text or layout shift in core Action panels.

### Phase 10 - Documentation and Reference Cleanup

Deliverables:

- Update [DOC: docs/concepts/ACTIONS.md] from "proposed" to implemented once the feature ships.
- Update [DOC: docs/reference/api-endpoints.md] with actual endpoint behavior and response examples.
- Update [DOC: docs/reference/cli-commands.md] with actual command help.
- Update [DOC: docs/reference/configuration.md] with Action-related env vars/config.
- Update [DOC: docs/concepts/GRAPH.md] to mark Action graph nodes implemented.
- Update [DOC: docs/concepts/HEARTBEATS.md] only if discover guidance is implemented.
- Add concrete `[CODE: path/to/file]` references for new implementation files.
- Register any new docs in [CODE: docs/manifest.json].

Implementation notes:

- Do not duplicate detailed API/CLI contracts in multiple places. Reference canonical docs.
- Keep concept docs focused on ontology and responsibilities; put exact flags/endpoints in reference docs.

Acceptance:

- `rg "Actions \\(Proposed\\)|Status: proposed|planned contract, not currently implemented" scenarios/prompt-manager/docs` returns no stale Action implementation claims after ship, except historical RFC language that is intentionally marked as pre-implementation.

### Phase 11 - Seed Minimal Actions

Deliverables:

- Add a tiny set of core Action fixtures only after validation/execution works.
- Prefer low-risk prompt-manager/project commands:
  - `scenario.status.show` wrapping `vrooli scenario status {{scenario}}`
  - `scenario.test.run` wrapping `vrooli scenario test {{scenario}}`
  - `team.decisions.list` wrapping a prompt-manager team decision-list command only after that CLI command exists
- Each seed Action must have examples and validation.

Implementation notes:

- Do not seed Actions for commands that do not exist.
- Do not create scenario screenshot seed unless a controlled screenshot CLI exists.
- Treat `team.decisions.list` as a future seed candidate unless `prompt-manager team decision-list` exists at implementation time.
- Keep seed set small; broad adoption belongs to the later adoption plan.

Acceptance:

- Seed Actions validate.
- At least one seed Action can run in a local dev environment without requiring paid/external services.

## Testing Plan

Backend:

```bash
cd scenarios/prompt-manager/api
go test ./store ./actions ./aisearch ./graph
go test ./...
```

CLI:

```bash
cd scenarios/prompt-manager/cli
go test ./...
prompt-manager action --help
prompt-manager discover "list team decisions" --type all --json
```

UI:

```bash
cd scenarios/prompt-manager/ui
pnpm run type-check
pnpm test
pnpm run build
```

Scenario lifecycle:

```bash
cd scenarios/prompt-manager
make test
make restart
make status
make logs
```

Manual API smoke checks after restart:

```bash
prompt-manager action list --json
prompt-manager action show scenario.status.show --json
prompt-manager action validate scenario.status.show --json
prompt-manager discover "list pending team decisions" --type all --json
```

If a runnable seed Action exists:

```bash
prompt-manager action run scenario.status.show --input='{"scenario":"prompt-manager"}' --json
```

## Rollout and Validation Checklist

- [x] Action schema added and validated by tests.
- [x] FileActionStore supports pack precedence and dotted IDs.
- [x] API CRUD endpoints implemented and tested.
- [x] Validation rejects shell and raw external commands.
- [ ] Validation uses a dynamic controlled-command resolver for project, prompt-manager, scenario, and resource commands.
- [x] Execution uses argv without shell interpretation.
- [x] Execution enforces timeout, concurrency, permissions, output caps, and audit/history recording.
- [x] CLI action group implemented and tested.
- [x] AI search indexes Actions.
- [x] Discover supports `--type skill|action|all`.
- [x] UI can browse, validate, edit, and run Actions.
- [x] Graph contains Action nodes and command edges.
- [x] Docs updated from proposed to implemented.
- [x] Minimal seed Action validates and supports API/CLI dry-run.
- [ ] Scenario tests pass through `make test` or `vrooli scenario test prompt-manager`.
- [x] Scenario restarts cleanly through lifecycle.

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Action runtime becomes arbitrary shell execution | Enforce argv-only commands, no shell, controlled-command ownership checks, static command token validation, and explicit rejection tests |
| Trusted first tokens accidentally allow unsafe subcommands | Validate through a dynamic controlled-command resolver that classifies ownership, subcommand, mutability, permissions, and execution eligibility |
| API run endpoint becomes an ungoverned command broker | Enforce timeout, concurrency, permission, output cap, and audit/history policy before process execution |
| Logic leaks into Action JSON | Keep Action schema declarative; branching belongs in owning CLI |
| Validation duplicated across API/CLI/UI | Centralize validation in API/domain service; CLI/UI display validation results |
| Discover breaks existing skill workflows | Make `--type` optional and default to current skill-compatible behavior until mixed discovery is fully tested |
| Graph nodes collide across entity types | Namespace graph node IDs or key graph APIs by `(type, id)` and test same raw ID across Action and Skill |
| Qdrant collection coupling gets messy | Use separate Action collection and shared indexing helpers where practical |
| UI adds a large parallel editor with drift | Reuse shared form primitives and Zod schemas, but keep Action-specific domain components |
| Graph scanner relies on fragile prose parsing | Use structured Action store for Action nodes; only prose references are best-effort |
| Execution output gets too large | Add stdout/stderr caps and truncation metadata |
| Long-running commands hang API requests | Enforce timeouts and cancellation |

## Non-Goals and Prohibited Patterns

Do not:

- implement compatibility with any preexisting Action format
- call the entity Capability
- wrap raw external tools directly
- execute through `sh -c` or any shell
- support pipelines, command separators, or conditional syntax in Action contracts
- encode branching/routing logic in `action.json`
- create a generic script runner
- validate command ownership through only a hard-coded first-token allowlist
- define validation hooks that recursively call `prompt-manager action validate <same-id>`
- add broad adoption prompt changes in this implementation plan
- seed many Actions before validation, search, and UI are solid
- seed Actions for commands that do not exist
- leave docs saying Actions are only proposed once the implementation is done

## Definition of Done

This work is done when:

- Action is a first-class prompt-manager entity across store, API, CLI, UI, search, graph, and docs.
- Every Action wraps exactly one validated Vrooli-controlled argv command.
- Controlled-command validation works for project, prompt-manager, scenario-owned, and resource-owned commands without hard-coding the scenario list in the plan.
- Invalid command forms are rejected by tests and by runtime validation.
- Runtime execution is governed by timeout, concurrency, permission, output cap, and audit/history policy.
- `prompt-manager action list/show/validate/run` works against the API.
- `prompt-manager discover --type all` can return mixed Skill and Action results with a type discriminator.
- The UI provides professional Action management and execution surfaces.
- Graph output includes Action nodes and Action-to-CLI edges.
- All new docs and code references are registered and accurate.
- `go test ./...` passes in API and CLI modules.
- `pnpm run type-check`, `pnpm test`, and `pnpm run build` pass in UI.
- `vrooli scenario test prompt-manager` passes, or any failure is documented with exact failing phase and reason.
- `vrooli scenario restart prompt-manager` succeeds and health checks pass.
- No compatibility shims, legacy aliases, dead code, or placeholder implementation remains.
