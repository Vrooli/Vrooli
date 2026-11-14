# P2-W7 transport inventory (P2-D11, DEBT P2-13)

Method: every REST route registered by `api/internal/handlers` and `api/internal/wiring/routes.go` (listed from `.vrooli/endpoints.json` plus `runs/quiesce`), its `AgentManagerService` twin, and every caller found by a fleet-wide search (below). Date 2026-10-29.

## Finding that shaped the decision

`NewAgentManagerConnectHandler` embeds `UnimplementedAgentManagerServiceHandler`; **before this pass 38 of the 104 `AgentManagerService` RPCs returned `Unimplemented`** and exactly those are the twins of the busiest REST routes (profiles, tasks, runs, runners, permission policy, purge). EVIDENCE E-P2-5 counted them as "no AM caller"; in fact nothing could call them. A CLI cannot move to a stub, so this pass (a) moved the CLI to every implemented twin, (b) implemented the adapters whose REST route had no caller outside the AM CLI so the REST route could be deleted, and (c) kept REST as the single working transport where the RPC is still a stub and a UI or outside caller exists.

## Consumer search (recorded)

- Fleet grep (`scenarios/*`, `packages/*`, `internal/`, `docs/`, `.vrooli/`, prompt-manager skills/programs) for the path families `/api/v1/(profiles|declarations|workflows|workflow-executions|tasks|runs|import|findings|runners|execution-options|role-policy|permission-policy|validate-path|maintenance|investigations|investigation-settings)`: 1,899 matching lines in 205 files outside this scenario; 120 of those files also name agent-manager. Of those, 71 are Go/TS/JSON/shell sources (the rest are docs and archived evidence) and every Go client was read: agent-inbox, development-toolchain-validator, git-control-tower, knowledge-observatory, meta-optimization-manager, portal, program-runtime, prompt-manager (heartbeat), react-component-library, scenario-auditor, scenario-to-cloud, scenario-to-desktop, swarm-manager, switchboard, system-monitor, test-genie, web-console, workspace-sandbox, vrooli-autoheal, `packages/cli-core/cliutil`, `internal/lifecycle`, `internal/agentsession`.
- Per-route targeted greps for the deleted routes returned **no caller outside `scenarios/agent-manager`** except historical evidence and design documents: `declarations/*` (a program-runtime PROBLEMS.md mention), `workflows/*` (browser-automation-studio has its own unrelated `/api/v1/workflows`), `role-policy/{status,validate,reload,explain}` (test-genie reads only `role-policy/catalog`), `workflow-executions/{id}/runs`, `investigations/{id}`, `investigations/{id}/wait`, `investigations/{id}/cancel` (prompt-manager heartbeat calls only `POST` and `GET /investigations`; system-monitor has its own unrelated `/api/v1/investigations*`).
- AM UI (`ui/src`) REST callers were listed from `hooks/useApiRuns.ts`, `useApiProfilesTasks.ts`, `useApiOperations.ts`, `useOrchestrationSettings.ts`, `lib/api.ts`, `api/episodes.ts`, `pages/InvestigationsPage.tsx`; none of the deleted routes appears there.

## Routes deleted in step 1 (18)

| Route | RPC twin | AM CLI before | AM UI | Outside callers | Decision |
|---|---|---|---|---|---|
| `GET /api/v1/investigations/{id}` | GetInvestigation (already implemented) | RPC (already) | none | none | **Deleted**; CLI uses the procedure |
| `GET /api/v1/role-policy/status` | GetRolePolicyStatus (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `GET /api/v1/workflow-executions/{id}/runs` | ListWorkflowExecutionRuns (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `GET /api/v1/workflows` | ListWorkflowRevisions (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `GET /api/v1/workflows/explain` | ExplainWorkflow (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `GET /api/v1/workflows/revision` | GetWorkflowRevision (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/declarations/plan` | PlanScenarioDeclarations (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/declarations/reconcile-scenario` | ReconcileScenarioDeclarations (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/investigations/{id}/cancel` | CancelInvestigation (already implemented) | RPC (already) | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/investigations/{id}/wait` | WaitInvestigation (already implemented) | RPC (already) | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/role-policy/explain` | ExplainRolePolicy (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/role-policy/reload` | ReloadRolePolicyCatalog (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/role-policy/validate` | ValidateRolePolicyCatalog (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/workflows/plan` | PlanScenarioWorkflows (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/workflows/reconcile-scenario` | ReconcileScenarioWorkflows (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/workflows/reload` | ReloadScenarioWorkflows (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/workflows/simulate` | SimulateWorkflow (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |
| `POST /api/v1/workflows/validate` | ValidateWorkflow (implemented this pass) | REST | none | none | **Deleted**; CLI uses the procedure |

## Routes kept (REST route is still called, or its RPC is a stub)

| Route | RPC twin | AM CLI now | AM UI | Outside callers | Decision |
|---|---|---|---|---|---|
| `GET /api/v1/execution-options` | ListExecutionOptions (impl) | RPC | - | swarm-manager api/internal/agentmanager/execution_options.go | Keep (outside caller) |
| `GET /api/v1/investigations` | ListInvestigations (impl) | RPC | pages/InvestigationsPage.tsx | prompt-manager heartbeat/client.go ListTypedInvestigations | Keep (UI and outside callers) |
| `POST /api/v1/investigations` | StartInvestigation (impl) | RPC | hooks/useApiRuns.ts | prompt-manager heartbeat/client.go | Keep (UI and outside callers) |
| `GET /api/v1/profiles` | ListProfiles (still an Unimplemented stub, see step 2) | REST | hooks/useApiProfilesTasks.ts | git-control-tower agent_manager_client.go, prompt-manager heartbeat | Keep: RPC is a stub, REST is the working transport |
| `POST /api/v1/profiles` | CreateProfile (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | git-control-tower, prompt-manager heartbeat | Keep: REST is the single transport |
| `POST /api/v1/profiles/ensure` | EnsureProfile (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | scenario-auditor, swarm-manager, scenario-to-cloud, scenario-to-desktop | Keep: REST is the single transport |
| `POST /api/v1/profiles/reconcile-scenario` | ReconcileScenarioProfiles (still an Unimplemented stub, see step 2) | REST | - | development-toolchain-validator, git-control-tower, knowledge-observatory, portal, scenario-to-desktop, swarm-manager (+ docs/scripts) | Keep: RPC is a stub, REST is the working transport |
| `DELETE /api/v1/profiles/{id}` | DeleteProfile (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | - | Keep: REST is the single transport |
| `GET /api/v1/profiles/{id}` | GetProfile (still an Unimplemented stub, see step 2) | REST | - | scenario-to-cloud, scenario-to-desktop | Keep: RPC is a stub, REST is the working transport |
| `PUT /api/v1/profiles/{id}` | UpdateProfile (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | - | Keep: REST is the single transport |
| `GET /api/v1/role-policy/catalog` | GetRolePolicyCatalog (implemented this pass) | RPC | hooks/useApiOperations.ts | test-genie api/agentmanager/client.go | Keep (UI and outside callers) |
| `GET /api/v1/runners` | GetRunnerStatus (RPC deleted from proto) | REST | - | test-genie api/agentmanager/client.go | Keep: REST is the single transport |
| `GET /api/v1/runs` | ListRuns (impl) | REST (RPC lacks parent_run_id; follow-up) | hooks/useApiRuns.ts | knowledge-observatory, meta-optimization-manager, prompt-manager | Keep (UI and outside callers) |
| `POST /api/v1/runs` | CreateRun (still an Unimplemented stub, see step 2) | REST | hooks/useApiRuns.ts | agent-inbox, dev-toolchain-validator, git-control-tower, knowledge-observatory, portal, prompt-manager, scenario-auditor, scenario-to-cloud/desktop, swarm-manager, switchboard | Keep: RPC is a stub, REST is the working transport |
| `POST /api/v1/runs/attach` | AttachRun (RPC deleted from proto) | REST | - | packages/cli-core cliutil/agentlauncher.go; agent launchers | Keep: REST is the single transport |
| `POST /api/v1/runs/stop-all` | StopAllRuns (RPC deleted from proto) | REST | - | system-monitor, test-genie | Keep: REST is the single transport |
| `GET /api/v1/runs/tag/{tag}` | GetRunByTag (RPC deleted from proto) | REST | - | scenario-auditor | Keep: REST is the single transport |
| `POST /api/v1/runs/tag/{tag}/stop` | StopRunByTag (RPC deleted from proto) | REST | - | scenario-auditor | Keep: REST is the single transport |
| `DELETE /api/v1/runs/{id}` | DeleteRun (RPC deleted from proto) | REST | hooks/useApiRuns.ts | - | Keep: REST is the single transport |
| `GET /api/v1/runs/{id}` | GetRun (still an Unimplemented stub, see step 2) | REST | hooks/useApiRuns.ts | internal/lifecycle owner_maintenance.go, cli-core, prompt-manager, scenario-auditor, scenario-to-*, system-monitor | Keep: RPC is a stub, REST is the working transport |
| `GET /api/v1/runs/{id}/accounting` | GetRunAccounting (impl) | RPC | - | swarm-manager api/internal/agentmanager/client.go | Keep (outside caller) |
| `POST /api/v1/runs/{id}/approve` | ApproveRun (RPC deleted from proto) | REST | - | knowledge-observatory | Keep: REST is the single transport |
| `POST /api/v1/runs/{id}/detach` | DetachRun (RPC deleted from proto) | REST | - | packages/cli-core cliutil/agentlauncher.go (self-detach) | Keep: REST is the single transport |
| `GET /api/v1/runs/{id}/diff` | GetRunDiff (RPC deleted from proto) | REST | hooks/useApiRuns.ts | knowledge-observatory | Keep: REST is the single transport |
| `GET /api/v1/runs/{id}/events` | GetRunEvents (impl) | REST (follow-up) | hooks/useApiRuns.ts | prompt-manager heartbeat, scenario-to-cloud/desktop, knowledge-observatory, system-monitor | Keep (UI and outside callers) |
| `POST /api/v1/runs/{id}/recover` | RecoverRun (RPC deleted from proto) | REST | - | -  (CLI only; REST handler carries run-initiated lifecycle guard) | Keep: REST is the single transport |
| `POST /api/v1/runs/{id}/reject` | RejectRun (RPC deleted from proto) | REST | - | knowledge-observatory | Keep: REST is the single transport |
| `GET /api/v1/runs/{id}/report` | GetRunReport (impl) | REST (CLI inspection tests mock REST; follow-up) | hooks/useApiRuns.ts | - | Keep until the UI moves to the RPC (hand-over below) |
| `POST /api/v1/runs/{id}/stop` | StopRun (RPC deleted from proto) | REST | hooks/useApiRuns.ts | prompt-manager, scenario-to-cloud/desktop, system-monitor | Keep: REST is the single transport |
| `GET /api/v1/tasks` | ListTasks (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | - | Keep: REST is the single transport |
| `POST /api/v1/tasks` | CreateTask (still an Unimplemented stub, see step 2) | REST | hooks/useApiProfilesTasks.ts | agent-inbox, dev-toolchain-validator, git-control-tower, knowledge-observatory, portal, prompt-manager, scenario-auditor, scenario-to-cloud/desktop | Keep: RPC is a stub, REST is the working transport |
| `DELETE /api/v1/tasks/{id}` | DeleteTask (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | - | Keep: REST is the single transport |
| `GET /api/v1/tasks/{id}` | GetTask (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | scenario-auditor | Keep: REST is the single transport |
| `PUT /api/v1/tasks/{id}` | UpdateTask (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | prompt-manager heartbeat/client.go | Keep: REST is the single transport |
| `POST /api/v1/tasks/{id}/cancel` | CancelTask (RPC deleted from proto) | REST | hooks/useApiProfilesTasks.ts | prompt-manager heartbeat/client.go | Keep: REST is the single transport |
| `GET /api/v1/workflow-executions` | ListWorkflowExecutions (implemented this pass) | RPC | hooks/useApiOperations.ts | swarm-manager workflow_invocation.go (idempotency_key filter the RPC lacks) | Keep (UI and outside callers) |
| `POST /api/v1/workflow-executions` | StartWorkflowExecution (impl) | RPC | - | swarm-manager, program-runtime delegator.go, react-component-library workflows/agent_manager.go | Keep (outside caller) |
| `GET /api/v1/workflow-executions/{id}` | GetWorkflowExecution (impl) | RPC | - | swarm-manager workflow.go; internal/lifecycle owner_maintenance.go | Keep (outside caller) |
| `POST /api/v1/workflow-executions/{id}/cancel` | CancelWorkflowExecution (impl) | RPC | hooks/useApiOperations.ts | swarm-manager workflow_invocation.go; react-component-library | Keep (UI and outside callers) |
| `GET /api/v1/workflow-executions/{id}/result` | GetWorkflowExecutionResult (impl) | RPC | - | swarm-manager, program-runtime, react-component-library | Keep (outside caller) |
| `POST /api/v1/workflow-executions/{id}/signals` | SignalWorkflowExecution (implemented this pass) | RPC | hooks/useApiOperations.ts | swarm-manager workflow_invocation.go | Keep (UI and outside callers) |
| `GET /api/v1/workflow-executions/{id}/trace` | GetWorkflowExecutionTrace (implemented this pass) | RPC | hooks/useApiOperations.ts | internal/lifecycle/owner_maintenance.go | Keep (UI and outside callers) |
| `POST /api/v1/runs/quiesce` | QuiesceScenario (RPC deleted from proto) | REST | - | - | Keep: REST is the single transport |

## Routes with no RPC twin (single transport already)

`findings`, `health`, `import/*`, `investigation-settings*`, `orchestration-settings*`, `identity/*`, `storage/*`, `attachments/upload`, `uploads/*`, `validate-path`, `investigations/{id}/result`, `investigations/{id}/learning-retry`, `runs/{id}/{durability,episodes,invocation-facts,ledger,messages-friction,observed-receipts,await-result,park,wake,partial-approve,sandbox-sync,messages/{event_id}/delete}`, `runs/{cohorts,episode-trend,import-*,backfill-*,investigation-apply,invocation-facts/replay,resume-from-failed,wake-by-key,external/tombstone}`. Nothing to consolidate; their callers are unchanged.


## Step 2: UI moved to the typed client; 12 more REST routes deleted

`ui/src/lib/api.ts` now exports one `agentManager` Connect client (30 s per-call deadline, same as `fetchWithDeadline`), used by the workflow-execution list/trace/cancel/retry/resume/signal, role-policy catalog, permission-policy, runner-probe and purge hooks, and by the effort board. The REST twins with no caller outside the AM CLI/UI were deleted (adapters implemented first):

| Route | RPC twin | AM CLI | AM UI | Outside callers | Decision |
|---|---|---|---|---|---|
| `POST /api/v1/maintenance/purge` | PurgeData (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `GET /api/v1/permission-policy/catalog` | GetPermissionPolicyCatalog (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/permission-policy/doctor` | DoctorPermissionPolicy (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/permission-policy/plan` | PlanPermissionPolicy (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/permission-policy/reconcile` | ReconcilePermissionPolicy (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/permission-policy/reload` | ReloadPermissionPolicyCatalog (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `GET /api/v1/permission-policy/status` | GetPermissionPolicyStatus (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/permission-policy/validate` | ValidatePermissionPolicyCatalog (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/runners/{runner_type}/probe` | ProbeRunner (Unimplemented stub) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/workflow-executions/{id}/advance` | AdvanceWorkflowExecution (implemented this pass) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/workflow-executions/{id}/resume` | ResumeWorkflowExecution (implemented this pass) | RPC | RPC | - | **Deleted**; no outside caller |
| `POST /api/v1/workflow-executions/{id}/retry` | RetryWorkflowExecution (implemented this pass) | RPC | RPC | - | **Deleted**; no outside caller |

Still REST in the UI (no RPC twin, or outside callers keep the route): `profiles`, `tasks`, `runs` CRUD/lifecycle/events/diff/report/ledger/episodes, `investigations` start/list, `investigation-settings`, `orchestration-settings`, `validate-path`, `findings`, `health/*`, `pricing`. These need either a new RPC or are single-transport already; the profiles/tasks/runs RPCs they would use are stubs whose REST twins serve other scenarios.

## Step 3: stub RPCs deleted from `service.proto` (104 -> 82)

Before deleting, every non-generated file in every scenario and package was searched for the RPC name as a Connect path, `AgentManagerService<Name>Procedure`, and a typed-client call, and every Go module that imports the generated client was built (`go vet ./...`, which compiles tests) after regeneration.

Deleted (22), no reference outside `scenarios/agent-manager`: ApproveRun, AttachRun, CancelTask, CreateProfile, DeleteProfile, DeleteRun, DeleteTask, DetachRun, EnsureProfile, GetRunByTag, GetRunDiff, GetRunnerStatus, GetTask, ListTasks, QuiesceScenario, RecoverRun, RejectRun, StopAllRuns, StopRun, StopRunByTag, UpdateProfile, UpdateTask. Their REST twins stay the single transport. (QuiesceScenario/RecoverRun: REST stays for the HTTP-bound run-initiated guards.)

Kept because a search found a live reference, and **implemented** instead of deleted (an RPC that answers `Unimplemented` forever is debt):

| RPC | Reference | Implementation |
|---|---|---|
| GetRun | content-desk `client.go`, git-control-tower `trailer_owners.go` (typed client) | adapter over `svc.GetRun` (`TestGetRunProcedureReturnsTheRunTheRESTRouteReturns`) |
| ReconcileScenarioProfiles | content-desk `client.go` (typed client) | adapter over `profiles.ReconcileScenarioProfiles` |
| GetProfile, ListProfiles | `packages/measures-go/paramschema_test.go` fixtures | adapters over `profiles.GetProfile/ListProfiles` |
| CreateTask | asset-studio `handlers/studio/agent_manager.go` posts the procedure path directly | adapter sharing `authorizeTaskCreate` + `createTask` with the REST handler (verified-orchestrator / run.create check, trusted `created_by`) |
| CreateRun | `scenarios/vrooli-events/api/handlers_capture_policy_test.go` fixture | adapter that runs the REST `CreateRun` handler in process (`callRESTHandler`) with the caller's headers and context, because its single-proof, delegated-child and run-initiated-lifecycle guards read headers and write HTTP responses. Restating them would fork security code |

`AgentOperationMiddleware` keys admission on the path, so `isRunCreationPath` now names both legs on both transports (anonymous switch, standing-grant service path, agent `run.create`); `TestRunCreationProceduresShareTheRouteAdmission` asserts procedure and route are admitted identically. All 104 - 22 = 82 remaining RPCs are implemented.

Stale `google.api.http` annotations were removed from the 30 RPCs whose REST routes were deleted in steps 1-2. The same annotation remains on RPCs whose REST route still exists.

## Step 5: CLI contract and health fixes (2026-10-29)

- `run messages-friction` and `run ledger` already called `EpisodesService.GetSelfReportSpans` / `GetCrossScenarioLedger` through the typed client; the manifest bound them wrongly (`ImportTranscript`, `local`). They are now bound correctly, as is `run import-transcript` (`ImportTranscript`). The unused REST helpers `RunService.Episodes/MessageFriction/Ledger/ImportTranscript` and the REST route `GET /runs/{id}/messages-friction` (no caller in the repo) are deleted. `runs/{id}/episodes` and `runs/{id}/ledger` stay REST for the UI (`api/episodes.ts`), and are the next UI move.
- `effort migrate` and `effort delivery` are declared in `cli/manifest.json`.
- Flags whose bind merely restated the proto field name lost the redundant `bind`; `measures episode-cohort --filter-json` binds to `filter` as `json_inline`.
