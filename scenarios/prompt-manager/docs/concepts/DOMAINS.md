# Prompt Manager Domain Model

This is the execution contract for the proto/Connect re-platform. It was produced from the live tree on 2026-08-19 and must be updated before a later slice diverges from it.

## Audit snapshot

- REST registrations: **201** at audit time; **6** after all transport slices retired 195 registrations. Connect mounts and the measures-go substrate are not counted as hand-written REST (the six are two health routes plus GET/PUT budget and discovery-filter configuration pairs).
- Composition root: **1,142** lines.
- API Go source: **101,359** lines live (**101,338** at plan authoring).
- Empty directories: **53**.
- Proto surface: 18 generated domain services spanning skills, experiments, actions, tags, search/discovery, agents/teams/topics, graph, heartbeat/memberflow, testing, metadata, templates, and world configuration.
- CLI manifest: **130** declared commands, **120** generated Connect bindings, and **10** intentional local compatibility/meta commands. The complete proto denominator is either bound or covered by **75** method-specific omissions; runtime composite/multiplexer behavior is documented by **18** exceptions. Nine live measures cover data-bearing stateful domains and six named waivers explain domains without a stable aggregate signal.

## Dependency direction

Transport adapters under `api/handlers/<domain>` depend inward on `api/internal/<domain>`. Domain services may depend on narrow interfaces and shared value packages; persistence implementations under `api/internal/store` implement those interfaces. No domain service imports a handler, generated Connect client, CLI package, or composition root. Cross-domain reads are explicit interfaces, not concrete store assertions. The intended read direction is:

`CLI/UI/program -> generated Connect client -> handler adapter -> domain service -> domain-owned port -> store/resource adapter`

Heartbeat and memberflow remain separate proto packages but form one migration slice because scheduling reads member declarations and memberflow reads heartbeat prompt sections. Graph reads skills/agents/teams/memberflow; AI search reads skills/actions/agents/teams/topics; those dependencies are read ports and must not invert ownership.

## Domain Inventory

This table is the machine-readable architecture authority. Paths describe the
current implementation; planned package changes remain in the slice table
below until their source directories exist.

| Domain | Source Paths |
|---|---|
| actions | `api/internal/actions/**`, `api/handlers/actions/**` |
| agents | `api/internal/agents/**`, `api/handlers/agents/**` |
| aisearch | `api/internal/aisearch/**`, `api/handlers/aisearch/**` |
| discovery | `api/handlers/discovery/**` |
| graph | `api/internal/graph/**`, `api/handlers/graph/**` |
| heartbeat | `api/internal/heartbeat/**`, `api/handlers/heartbeat/**` |
| memberflow | `api/internal/memberflow/**`, `api/handlers/memberflow/**` |
| metadata | `api/internal/ogmeta/**`, `api/handlers/metadata/**`, `api/handlers/ogmeta/**` |
| metrics | `api/internal/metrics/**` |
| search | `api/internal/search/**`, `api/handlers/search/**` |
| skills | `api/internal/skills/**`, `api/handlers/skills/**` |
| tags | `api/internal/tags/**`, `api/handlers/tags/**` |
| teams | `api/internal/teams/**`, `api/handlers/teams/**` |
| templates | `api/internal/templates/**`, `api/handlers/templates/**` |
| testing | `api/internal/testing/**`, `api/handlers/testing/**` |
| topics | `api/internal/topics/**`, `api/handlers/topics/**` |
| worldscale | `api/internal/worldscale/**`, `api/handlers/worldscale/**` |
| worldseats | `api/internal/worldseats/**`, `api/handlers/worldseats/**` |

## Live directory inventory

The dependency column contains direct imports of another top-level API package, including tests. “State” determines whether phase 16 needs a real measure or a named waiver.

| Directory | Go LOC | Files | Direct API dependencies | Role/state | Boundary |
|---|---:|---:|---|---|---|
| `actions` | 4,382 | 12 | `store` | stateful | Action contracts, validation, preview, and governed execution. |
| `agents` | 1,359 | 3 | `store`, `validation` | stateful | Agent identity, files, soul, capabilities, and team membership reads. |
| `aisearch` | 10,828 | 32 | `search`, `skills`, `store` | stateful | Vector-backed search, discovery composition, index reconciliation, budgets, filters, and discovery telemetry. |
| `cmd` | 222 | 2 | `memberflow` | composition | Offline memberflow audit entry points; not an API domain. |
| `docs` | 0 | 0 | — | dead scaffold | Empty legacy scaffold; remove. |
| `finding` | 158 | 2 | — | pure value | Shared typed finding values; no I/O. |
| `graph` | 8,261 | 26 | `memberflow`, `store` | stateful cache/config | Relationship graph, health/config, operating-map projection, index lifecycle. |
| `handlers` | 0 | 0 | — | dead scaffold | Empty target scaffold today; becomes transport adapters in phase 11. |
| `heartbeat` | 16,956 | 60 | `finding`, `internal`, `interop`, `memberflow`, `sourceledger`, `store`, `teamconfig`, `teamcontract` | stateful | Run/task orchestration, schedules, queues, controls, prompts, handoffs, task boards, retention, and bug intake. |
| `internal` | 1,037 | 23 | `store`, `teamconfig`, `teamcontract` | shared substrate | Shared implementation/test substrate; split into named domain packages in phase 11. |
| `interop` | 801 | 3 | `store`, `teamconfig`, `validation` | pure transform | Loss-aware external team-format conversion. |
| `memberflow` | 23,630 | 95 | `finding`, `internal`, `teamcontract` | stateful | Member topic contracts, operating models, rules, objectives, instruments, drain status, orientation cost. |
| `metrics` | 288 | 4 | `internal` | stateful | Skill usage/rating persistence. |
| `ogmeta` | 285 | 3 | — | pure read | Open Graph metadata lookup. |
| `search` | 2,008 | 8 | `skills`, `store` | pure read | Deterministic full-text entity/content search. |
| `skills` | 8,987 | 31 | `metrics`, `store` | stateful | Skill CRUD/read/sync/version/variant behavior and experiment storage-facing types. |
| `sourceledger` | 174 | 1 | — | pure read | Read-only source provenance extraction. |
| `store` | 10,826 | 37 | `finding`, `internal`, `sourceledger`, `teamconfig`, `teamcontract` | persistence substrate | File/SQLite persistence adapters and indexes; never owns transport policy. |
| `tags` | 262 | 6 | `internal` | stateful | Tag taxonomy persistence. |
| `teamconfig` | 658 | 2 | — | pure policy | Portable team configuration policy/value types. |
| `teamcontract` | 1,300 | 3 | `finding` | pure policy | Team/member contract parsing and findings. |
| `teams` | 5,005 | 16 | `internal`, `interop`, `store`, `teamconfig`, `teamcontract`, `validation` | stateful | Team CRUD, membership, roles, files, org chart, messaging, external import/export. |
| `templates` | 180 | 4 | `store` | pure read | Read-only agent file templates. |
| `testing` | 549 | 7 | `internal`, `skills` | stateful | Skill tests and durable result history. |
| `topics` | 1,075 | 3 | `store`, `validation` | stateful | Content taxonomy CRUD, matching, and accumulated skills. |
| `validation` | 92 | 2 | — | pure policy | Shared syntactic validators. |
| `worldscale` | 173 | 2 | `store` | stateful | Persisted world rendering scale configuration. |
| `worldseats` | 405 | 2 | `store` | stateful | Persisted seat placement configuration. |

## Target business domains and proto packages

Each business capability owns one proto package below `packages/proto/schemas/prompt-manager/v1/<domain>`. Shared messages stay in their owning package and are imported by consumers; there is no catch-all `common` package. Persistence/substrate directories do not become services.

| Proto package | Boundary | Read graph position | Durable state |
|---|---|---|---|
| `skills` | Skill CRUD/read/sync, versions, variants, usage/rating facade | Root entity; read by search, AI search, graph, testing | Yes |
| `experiments` | Experiment lifecycle, assignments, receipts, outcomes, reports, promotion | Child of skills; imports stable skill identifiers only | Yes |
| `actions` | Action CRUD, preview, validation, governed execution | Read by AI search/discovery; execution delegates to governed runtime | Yes |
| `tags` | Tag taxonomy | Read by skills/search | Yes |
| `search` | Deterministic full-text entity/content queries | Read-only projection over skills/agents/teams | No |
| `aisearch` | Semantic search and index reconciliation | Projection over skills/actions/agents/teams/topics | Yes: index/config/telemetry |
| `discovery` | Budgeted capability composition, gaps, metrics, skill usage | Reads AI search, topics, telemetry | Yes: call/miss telemetry |
| `agents` | Agent CRUD, soul/files, team membership projection | Root entity; read by teams/graph/AI search/heartbeat | Yes |
| `teams` | Team CRUD, membership, roles, files, org, messages, external exchange | Root aggregate; read by graph/heartbeat/memberflow | Yes |
| `topics` | Content taxonomy CRUD/match/accumulation | Read by discovery/memberflow/AI search | Yes |
| `templates` | Agent-file templates | Leaf read model | No |
| `testing` | Skill model tests and history | Reads skills/model resource | Yes: history |
| `metadata` | Open Graph metadata lookup | Leaf resource read | No |
| `graph` | Relationship/health/operating-map projections and graph config | Reads skills/agents/teams/memberflow | Yes: index/config |
| `heartbeat` | Scheduling, runs, queues, controls, prompts, handoffs, task board, retention, bug intake | Reads teams/agents/memberflow; writes runtime state | Yes |
| `memberflow` | Topic declarations, rules, objectives, instruments, operating models, drain and orientation projections | Reads team contracts and heartbeat prompt section port | Yes |
| `worldscale` | World scale config | Leaf config | Yes |
| `worldseats` | World seat config | Leaf config | Yes |
| `health` | Liveness/readiness | Composition-root projection | No |

`store`, `finding`, `sourceledger`, `teamconfig`, `teamcontract`, `validation`, `interop`, and generic `internal` are implementation/value substrates, not callable domains. `cmd` contains offline audit entry points. Empty `docs` and legacy empty handler/internal scaffolds are removed in phase 11.

## Proto and transport policy

- Packages use `vrooli.prompt_manager.v1.<domain>` and Go packages generated under `packages/proto/gen/go/prompt-manager/v1/<domain>`.
- Services are domain-named (`SkillsService`, `DiscoveryService`, and so on); RPCs are behavior-named and use domain-owned request/response messages.
- Generated types are wire types, not the domain model. Handlers map and delegate; they do not duplicate validation, storage, scheduling, or ranking logic.
- REST is retired operation-by-operation in the owning slice after repository consumer search and live CLI parity. A completed operation has one supported transport.
- CLI code uses generated clients and owns only argument parsing, attribution, and human/JSON rendering.
- Cross-platform behavior uses Go APIs and lifecycle/service discovery; no shell recipes, fixed home paths, or OS-specific separators enter handlers or generated contracts.

## Binding and omission policy

A runtime CLI command must end in exactly one of these states:

1. **Proto-bound** when it represents stable scenario behavior and can be expressed with typed request fields. Reads are run-eligible by default. Writes are run-eligible only when existing authorization, attribution, idempotency, and bounded target semantics make program execution safe.
2. **Explicitly omitted** when it is CLI-local (`help`, `version`, local `configure`, formatting-only `space`), interactive/clipboard behavior, an operator-only destructive or lifecycle action, or cannot yet provide a stable typed contract. Every omission names the exact command, owner, reason, and the condition for removal.
3. **Invalid** when it is runtime-visible but neither declared nor omitted. Phase 16 treats these extra team/graph leaves as denominator defects.

Omission is not a maturity shortcut. Domain CRUD and high-value reads (`skill read`, `skill list`, `discover`) must be bound. Mutating governance reflects actual effects. Alias spellings share the canonical command declaration. A command is never marked callable merely because its transport method exists.

## Slice order

1. **Layout normalization**: move handlers/services without behavior changes; establish inward dependencies and remove 53 empty directories.
2. **Skills + actions + tags — completed**: 22 CLI bindings and 25 RPC methods replaced 26 REST registrations. API-only variant get/update and usage recording have no CLI command; action authoring combines the former preview/create routes. Experiments remain with the later orchestration slice because their receipt/promotion model crosses runtime evidence.
3. **Search + AI search + discovery — completed**: one projection pipeline and shared result/budget concepts makes capability discovery program-callable; 18 REST registrations were retired.
4. **Teams + agents + topics + templates + testing + metadata — completed**: team/agent structure and leaf supporting services now use generated clients. `worldscale` and `worldseats` were included as UI-owned configuration services. The slice added 28 CLI bindings and retired 57 REST registrations; UI-only methods have explicit omission reasons.
5. **Graph + heartbeat + memberflow + experiments**: graph consumes the settled entity contracts; heartbeat/memberflow migrate together because their temporal and prompt contracts are coupled; experiments join because receipts and promotion are orchestration evidence.
6. **Manifest/measures closure**: bind or omit the entire runtime command surface, prove every stateful domain has data-bearing condition evidence, and earn L3.

A later slice may change this order only by updating this document and recording the rejected prior grouping in `docs/internal/DECISIONS.md`.

## CLI command inventory

Canonical leaves are listed; aliases use the same route. Route sets show the current REST ownership before migration. `local` means no scenario API call.

| Command/group | Canonical leaves | Current route mapping | Target |
|---|---|---|---|
| `help`, `version`, `configure` | one flat command each | local | omit with CLI-local reason |
| `status` | flat | `GET /health` | health proto binding or standard-core omission |
| `skill` | list, show, read, add, update, delete, use, sync, rate, versions, revert, variants, add-variant, rm-variant | generated `SkillsService`; usage reporting is `DiscoveryService.GetSkillUsage` | `skills` |
| `action` | list, show, create, update, delete, validate, run | generated `ActionsService` | `actions` |
| `experiment` | list, show, create, start, conclude, holdout, promote, outcomes, report, delete | `/api/v1/experiments*`, `/api/v1/skills/{id}/experiments` | `experiments` |
| `tag` | list, create | generated `TagsService` | `tags` |
| `member` | list, show, create, update, delete | agent/team membership routes | `agents` + `teams` |
| `agent` | list, show, create, update, delete, soul, search | entity REST routes plus generated `SearchService`/`AISearchService` | `agents`/`search` |
| `team` entity | list, show, create, update, delete, add-member, update-member, remove-member, roles | `/api/v1/teams*` | `teams` |
| `team` structure | org-list, org-set, org-remove, message-list, message-send, message-delete, message-clear, responsibilities, heartbeat-instructions, member-context | team/member org, messages, docs, context routes | `teams`/`heartbeat` |
| `team` execution | heartbeat-list, heartbeat, heartbeat-enable, heartbeat-disable, heartbeat-trigger, heartbeat-logs, heartbeat-control, queue-clear, trigger, prompt-preview, prompt-preview-structured, prompt-matrix | heartbeat/run/prompt routes | `heartbeat` |
| `team` knowledge | operating-contract, validate-contract, search, handoff-latest, handoff-history, task-list, task-add, task-update, task-delete, knowledge-add, knowledge-list, knowledge-update, knowledge-delete, bug-capture, bug-repair | operating models, handoffs, tasks, memberflow inbox/knowledge and bug routes | `memberflow`/`heartbeat` |
| `team` maintenance | import-cc, export-cc, retention, prune | team import/export/retention/prune routes | bind safe reads/writes; omit operator-destructive prune unless governance proves safe |
| `heartbeat-control` | status, policy, pause, resume | `/api/v1/heartbeats/control*` | `heartbeat` |
| `topic` | list, show, create, update, delete, skills, search, tree | `/api/v1/topics*` | `topics` |
| `test` | run, history | `/api/v1/skills/{id}/test*` | `testing` |
| `metadata` | fetch | `GET /api/v1/og-metadata` | `metadata` |
| search flats | search, search-status, search-reconcile, search-reconcile-status, search-reconcile-cancel | generated `SearchService` and `AISearchService` | `search`/`aisearch` |
| discovery flats | discover, discovery-gaps, discovery-metrics, skill-usage | generated `DiscoveryService` | `discovery` |
| `graph` core | show, dump, node, regenerate, orphaned-skills, skillless-agents, empty-teams, unaffiliated-agents, cliless-skills, popular, circular-refs, health | `/api/v1/graph*` | `graph` |
| `graph` operating model | topics, runtime, rules, operating-model list/validate/diff/coverage, map, objectives, orientation-cost, instruments, audit, drain-status | topic graph, rules, operating models, objectives/instruments/drain routes; audit also performs local repository reads | bind server projections; explicitly omit local-only audit facets |
| `space` | flat | local generated coverage document | omit as CLI-local projection unless a stable service consumer appears |

The old manifest contains 84 command entries, while the runtime command inventory above includes undeclared team and graph leaves. Phase 16 must reconcile both lists; it may not use 84 as the final denominator.

## Baseline REST route inventory

Generated from the 201 literal `HandleFunc` registrations at audit time. Handler expressions are retained so ownership and removed-count evidence can be checked mechanically. Slices 1–3 have retired 101 registrations and are no longer registered in `api/main.go`: 26 skill/action/tag, 18 search/discovery, and 57 agent/team/topic/supporting-leaf registrations. The table below contains the 100 REST registrations still live after slice 3.

| Method | Route | Handler | Owner |
|---|---|---|---|
| GET | `/health` | `healthHandler` | health |
| GET | `/api/v1/health` | `healthHandler` | health |
| GET | `/api/v1/skills/{id}/experiments` | `experimentHandlers.ListExperimentsBySkill` | experiment |
| GET | `/api/v1/experiments` | `experimentHandlers.ListExperiments` | experiment |
| GET | `/api/v1/experiments/{eid}` | `experimentHandlers.GetExperiment` | experiment |
| POST | `/api/v1/experiments` | `experimentHandlers.CreateExperiment` | experiment |
| PUT | `/api/v1/experiments/{eid}` | `experimentHandlers.UpdateExperiment` | experiment |
| DELETE | `/api/v1/experiments/{eid}` | `experimentHandlers.DeleteExperiment` | experiment |
| POST | `/api/v1/experiments/{eid}/start` | `experimentHandlers.StartExperiment` | experiment |
| POST | `/api/v1/experiments/{eid}/conclude` | `experimentHandlers.ConcludeExperiment` | experiment |
| POST | `/api/v1/experiments/{eid}/outcomes` | `experimentHandlers.RecordOutcome` | experiment |
| GET | `/api/v1/experiments/{eid}/outcomes` | `experimentHandlers.ListOutcomes` | experiment |
| POST | `/api/v1/experiments/{eid}/assignments` | `experimentHandlers.AssignExperiment` | experiment |
| POST | `/api/v1/experiments/{eid}/audit-receipt` | `experimentHandlers.RecordAuditReceipt` | experiment |
| POST | `/api/v1/experiments/{eid}/holdout-receipt` | `experimentHandlers.RecordHoldoutReceipt` | experiment |
| POST | `/api/v1/experiments/{eid}/promote` | `experimentHandlers.PromoteExperiment` | experiment |
| GET | `/api/v1/experiments/{eid}/report` | `experimentHandlers.GetExperimentReport` | experiment |
| GET | `/api/v1/graph` | `graphHandlers.GetGraph` | graph |
| POST | `/api/v1/graph/regenerate` | `graphHandlers.Regenerate` | graph |
| GET | `/api/v1/graph/orphans` | `graphHandlers.GetOrphans` | graph |
| GET | `/api/v1/graph/skillless` | `graphHandlers.GetSkillless` | graph |
| GET | `/api/v1/graph/empty-teams` | `graphHandlers.GetEmptyTeams` | graph |
| GET | `/api/v1/graph/unaffiliated` | `graphHandlers.GetUnaffiliated` | graph |
| GET | `/api/v1/graph/popular` | `graphHandlers.GetPopular` | graph |
| GET | `/api/v1/graph/cycles` | `graphHandlers.GetCycles` | graph |
| GET | `/api/v1/graph/health` | `graphHandlers.GetHealthScores` | graph |
| GET | `/api/v1/graph/health-config` | `graphHandlers.GetHealthConfig` | graph |
| PUT | `/api/v1/graph/health-config` | `graphHandlers.PutHealthConfig` | graph |
| GET | `/api/v1/graph/nodes/{id}` | `graphHandlers.GetNode` | graph |
| GET | `/api/v1/graph/nodes/{id}/edges` | `graphHandlers.GetNodeEdges` | graph |
| GET | `/api/v1/config/budgets` | `aiSearchHandlers.GetBudgetConfig` | aisearch |
| PUT | `/api/v1/config/budgets` | `aiSearchHandlers.PutBudgetConfig` | aisearch |
| GET | `/api/v1/config/discover-filters` | `aiSearchHandlers.GetDiscoverFilterConfig` | aisearch |
| PUT | `/api/v1/config/discover-filters` | `aiSearchHandlers.PutDiscoverFilterConfig` | aisearch |
| GET | `/api/v1/teams/{id}/members/{agentId}/topics` | `memberFlowHandlers.GetMember` | memberflow |
| PUT | `/api/v1/teams/{id}/members/{agentId}/topics` | `memberFlowHandlers.PutMember` | memberflow |
| GET | `/api/v1/teams/{id}/topics` | `memberFlowHandlers.GetTeam` | memberflow |
| GET | `/api/v1/topics/graph` | `memberFlowHandlers.GetGraph` | memberflow |
| GET | `/api/v1/topics/rules` | `memberFlowHandlers.GetRules` | memberflow |
| GET | `/api/v1/objectives` | `memberFlowHandlers.GetObjectives` | memberflow |
| GET | `/api/v1/orientation-cost` | `memberFlowHandlers.GetOrientationCost` | memberflow |
| GET | `/api/v1/instruments` | `memberFlowHandlers.GetInstruments` | memberflow |
| GET | `/api/v1/topics/drain-status` | `memberFlowHandlers.GetDrainStatus` | memberflow |
| GET | `/api/v1/operating-models` | `memberFlowHandlers.GetOperatingModels` | memberflow |
| GET | `/api/v1/operating-models/map` | `graphHandlers.GetOperatingMap` | graph |
| GET | `/api/v1/operating-models/validate` | `memberFlowHandlers.ValidateOperatingModelsHandler` | memberflow |
| GET | `/api/v1/operating-models/diff` | `memberFlowHandlers.DiffOperatingModelsHandler` | memberflow |
| GET | `/api/v1/operating-models/coverage` | `memberFlowHandlers.CoverageOperatingModelsHandler` | memberflow |
| POST | `/api/v1/tasks` | `heartbeatHandlers.CreateTask` | heartbeat |
| POST | `/api/v1/runs` | `heartbeatHandlers.CreateRun` | heartbeat |
| GET | `/api/v1/runs` | `heartbeatHandlers.ListRuns` | heartbeat |
| GET | `/api/v1/heartbeat-attempts` | `heartbeatHandlers.ListHeartbeatAttempts` | heartbeat |
| POST | `/api/v1/runs/investigate` | `heartbeatHandlers.CreateInvestigationRun` | heartbeat |
| POST | `/api/v1/runs/investigation-apply` | `heartbeatHandlers.CreateInvestigationApplyRun` | heartbeat |
| GET | `/api/v1/runs/{runId}` | `heartbeatHandlers.GetRun` | heartbeat |
| POST | `/api/v1/runs/{runId}/retry` | `heartbeatHandlers.RetryRun` | heartbeat |
| GET | `/api/v1/runs/{runId}/events` | `heartbeatHandlers.GetRunEvents` | heartbeat |
| POST | `/api/v1/runs/{runId}/continue` | `heartbeatHandlers.ContinueRun` | heartbeat |
| GET | `/api/v1/heartbeats/control` | `heartbeatHandlers.GetHeartbeatControl` | heartbeat |
| PUT | `/api/v1/heartbeats/control/policy` | `heartbeatHandlers.UpdateHeartbeatControlPolicy` | heartbeat |
| POST | `/api/v1/heartbeats/control/pause` | `heartbeatHandlers.PauseHeartbeatControl` | heartbeat |
| POST | `/api/v1/heartbeats/control/resume` | `heartbeatHandlers.ResumeHeartbeatControl` | heartbeat |
| GET | `/api/v1/heartbeats/running` | `heartbeatHandlers.ListRunning` | heartbeat |
| POST | `/api/v1/heartbeats/running/{teamId}/{agentId}/stop` | `heartbeatHandlers.StopRunning` | heartbeat |
| POST | `/api/v1/prompt-preview` | `heartbeatHandlers.PreviewPrompt` | heartbeat |
| POST | `/api/v1/prompt-preview-structured` | `heartbeatHandlers.PreviewPromptStructured` | heartbeat |
| GET | `/api/v1/teams/{id}/prompt-matrix` | `heartbeatHandlers.PreviewPromptMatrix` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats/control` | `heartbeatHandlers.GetTeamHeartbeatControl` | heartbeat |
| PUT | `/api/v1/teams/{id}/heartbeats/control/policy` | `heartbeatHandlers.UpdateTeamHeartbeatControlPolicy` | heartbeat |
| POST | `/api/v1/teams/{id}/heartbeats/control/pause` | `heartbeatHandlers.PauseTeamHeartbeatControl` | heartbeat |
| POST | `/api/v1/teams/{id}/heartbeats/control/resume` | `heartbeatHandlers.ResumeTeamHeartbeatControl` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats` | `heartbeatHandlers.ListHeartbeats` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats/{agentId}` | `heartbeatHandlers.GetHeartbeat` | heartbeat |
| POST | `/api/v1/teams/{id}/heartbeats/{agentId}` | `heartbeatHandlers.CreateHeartbeat` | heartbeat |
| PUT | `/api/v1/teams/{id}/heartbeats/{agentId}` | `heartbeatHandlers.UpdateHeartbeat` | heartbeat |
| DELETE | `/api/v1/teams/{id}/heartbeats/{agentId}` | `heartbeatHandlers.DeleteHeartbeat` | heartbeat |
| POST | `/api/v1/teams/{id}/heartbeats/{agentId}/trigger` | `heartbeatHandlers.TriggerHeartbeat` | heartbeat |
| POST | `/api/v1/teams/{id}/trigger` | `heartbeatHandlers.TriggerTeam` | heartbeat |
| GET | `/api/v1/teams/{id}/execution-status` | `heartbeatHandlers.GetTeamExecutionStatus` | heartbeat |
| DELETE | `/api/v1/teams/{id}/queue/running/{agentId}` | `heartbeatHandlers.ClearTeamQueueRunning` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats/logs` | `heartbeatHandlers.ListTeamLogs` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats/{agentId}/logs` | `heartbeatHandlers.ListLogs` | heartbeat |
| GET | `/api/v1/teams/{id}/heartbeats/{agentId}/logs/{logId}` | `heartbeatHandlers.GetLog` | heartbeat |
| GET | `/api/v1/teams/{id}/members/{agentId}/responsibilities` | `heartbeatHandlers.GetResponsibilities` | heartbeat |
| PUT | `/api/v1/teams/{id}/members/{agentId}/responsibilities` | `heartbeatHandlers.SetResponsibilities` | heartbeat |
| GET | `/api/v1/teams/{id}/members/{agentId}/heartbeat-instructions` | `heartbeatHandlers.GetHeartbeatInstructions` | heartbeat |
| PUT | `/api/v1/teams/{id}/members/{agentId}/heartbeat-instructions` | `heartbeatHandlers.SetHeartbeatInstructions` | heartbeat |
| GET | `/api/v1/teams/{id}/members/{agentId}/context` | `heartbeatHandlers.GetMemberContext` | heartbeat |
| GET | `/api/v1/teams/{id}/members/{agentId}/handoff` | `heartbeatHandlers.GetLastHandoff` | heartbeat |
| DELETE | `/api/v1/teams/{id}/members/{agentId}/handoff` | `heartbeatHandlers.ClearLastHandoff` | heartbeat |
| GET | `/api/v1/teams/{id}/handoff-history` | `heartbeatHandlers.GetHandoffHistory` | heartbeat |
| DELETE | `/api/v1/teams/{id}/handoff-history` | `heartbeatHandlers.ClearHandoffHistory` | heartbeat |
| GET | `/api/v1/teams/{id}/tasks` | `heartbeatHandlers.GetTaskBoard` | heartbeat |
| POST | `/api/v1/teams/{id}/tasks` | `heartbeatHandlers.AddTask` | heartbeat |
| PATCH/PUT | `/api/v1/teams/{id}/tasks/{taskId}` | `heartbeatHandlers.UpdateTaskHandler` | heartbeat |
| DELETE | `/api/v1/teams/{id}/tasks/{taskId}` | `heartbeatHandlers.DeleteTaskHandler` | heartbeat |
| POST | `/api/v1/teams/{id}/bugs/capture` | `heartbeatHandlers.CaptureBug` | heartbeat |
| PATCH/PUT | `/api/v1/teams/{id}/bugs/{draftId}/capture` | `heartbeatHandlers.RepairBugCapture` | heartbeat |
| GET | `/api/v1/teams/{id}/retention` | `heartbeatHandlers.GetRetention` | heartbeat |
| POST | `/api/v1/teams/{id}/prune` | `heartbeatHandlers.PruneSharedState` | heartbeat |
