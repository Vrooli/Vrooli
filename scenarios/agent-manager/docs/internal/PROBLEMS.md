# Problems & Known Issues: agent-manager

## Open Issues

### P-002: Runner Process Stability
**Severity**: Medium
**Description**: Agent runners (claude-code, codex, opencode) may hang, crash, or produce unexpected output. Need robust timeout and cleanup handling.
**Mitigation**: Configurable timeouts per AgentProfile; process group tracking for cleanup; structured error events on failure.
**Status**: Design consideration - timeout enforcement planned for P0.

### P-003: Event Log Growth
**Severity**: Low (initially)
**Description**: Append-only RunEvent logs will grow continuously. Long-running installations may accumulate significant data.
**Mitigation**: Retention policies; archival to cold storage; compression of old events.
**Status**: Deferred to P1/P2 - acceptable for alpha/beta phases.

### P-004: Scope Lock Deadlock Potential
**Severity**: Medium
**Description**: Path-scoped locks could theoretically deadlock if not carefully managed, though single-scope-per-run design minimizes risk.
**Mitigation**: Delegate to workspace-sandbox mutual exclusion; avoid multi-scope transactions.
**Status**: Design consideration - workspace-sandbox handles scope conflicts.

### P-005: Runner Adapter Capability Discovery
**Severity**: Low
**Description**: Different runners support different capabilities (message streaming, tool events, cost reporting). Need runtime capability discovery.
**Mitigation**: RunnerAdapter interface includes capability query method.
**Status**: Planned for P1 (OT-P1-006).

## Deferred Ideas

### D-001: Multi-Project Support
**Description**: Current design assumes single project root. Future may require managing agents across multiple repositories.
**Reason for deferral**: Complexity; not needed for initial Vrooli use cases.
**Consideration**: Design types/APIs to not preclude multi-project.

### D-002: Agent Collaboration
**Description**: Multiple agents working on related tasks within same run, passing context between them.
**Reason for deferral**: Complexity; single-agent runs sufficient for initial use cases.
**Consideration**: Multi-phase runs (OT-P2-001) provide sequential agent chaining.

### D-003: Cost Budgeting
**Description**: Set cost budgets per task/agent; pause or abort when budget exceeded.
**Reason for deferral**: Requires cost tracking (P2) as foundation.
**Consideration**: AgentProfile could include cost limits once tracking implemented.

### D-004: Agent Learning from Feedback
**Description**: Use approval/rejection patterns to improve agent behavior over time.
**Reason for deferral**: Complex ML/feedback loop; out of scope for orchestration layer.
**Consideration**: Event logs provide training data if needed later.

## Resolved Issues

### P-008: Lifecycle incorrectly required Codex API key (resolved 2026-07-29)
**Root cause**: The Codex resource declared `OPENAI_API_KEY` as required, so the
lifecycle credential resolver stopped Agent Manager before runner probes ran.
This contradicted the optional Agent Manager runner dependency and Codex's
signed-in CLI authentication path.
**Resolution**: The descriptor is now optional and a real-manifest regression
test enforces that every Agent Manager runner resource has only optional
credentials. `make start` subsequently completed with Agent Manager healthy and
both API/UI listeners bound without an API key.

## Test Gaps

## Work ladder

- Rung: W3
- Evidence: `OT-P0-012` has the planned, truthfully pending `REQ-P0-014`; `business-health validate scenario agent-manager` and `vrooli scenario requirements validate agent-manager --json` pass, with only inherited orphaned P2 warnings.
- Blocker: obtain a fresh implementation baseline before changing the durable analytics projection.
- Measured: 2026-07-29

### P-007: Unit coverage policy gaps remain after reliability hardening (2026-07-23)
**Severity**: Medium (verification)
**Description**: Unit Health now passes the production-to-testutil boundary and
CLI shared-fixture checks. Current measured coverage is API 75.0% against 75%
and CLI 57.6% against 75%; focused reliability tests are green, but the CLI Go
coverage gates remain open. The UI reports 32.7% and four inherited Vitest
threshold-projection errors (native threshold 0 versus policy 85).
**Mitigation**: Continue behavior-focused tests in the API orchestration and
handler paths, then CLI command result/error paths. UI feature coverage is
outside the reliability plan, so preserve the truthful policy finding rather
than weakening the contract.
**Status**: Open.

### P-006: Comprehensive health suite has unrelated inherited debt (2026-07-11)
**Severity**: Medium (verification), not a role/permission cutover defect.
**Description**: Test Genie run `20260711-155454-0b8cfb01` reached terminal
`FAIL` in broad existing health phases: structure, contracts, UI, API,
dependencies, quality, unit, storage, tidiness, security, measures, proto, and
templates. Its provider-conformance and business phases passed. The docs phase
was unavailable because Knowledge Observatory cannot compile its stale
`AgentProfile.RunnerType`/`Model` adapter usage after the hard cutover; that
external consumer defect is tracked as `knw-1783784244326771566`.
**Mitigation**: Keep role-policy, permission-policy, profile-reconcile, and
agent-conformance validation focused and green; resolve each owning health
provider or scenario debt independently before treating the comprehensive
scenario suite as a release gate for this cutover.
**Status**: Open; not remediated by compatibility restoration.

The role-policy boundary has focused coverage for catalog validation and
atomic activation, profile-to-snapshot resolution, cross-runner fallback,
explicit runner-default launch, unavailable-runner skips, terminal exhaustion,
persisted-candidate restart/resume, and catalog reload during execution.
Repository tests cover the clean role-only profile schema and historical
snapshot round trips. Operator contract tests cover status, catalog inspection,
validation, failed-reload preservation, explanation, and removal of the
whole-document mutation command. Seeded-profile reconciliation validates
scenario-owned `roleRef` files; the broader API suite retains only the
separately tracked app-issue-tracker missing-manifest fixture failure. Unit
Health also reports pre-existing UI policy-projection drift and
requirement-tagging debt outside this plan.

## Technical Debt

### TD-003: Measures and legacy statistics parity remain incomplete
**Description:** The durable invocation read model now owns friction facts and
provides corpus metrics, aggregates, and cohort selection. The legacy
`StatsSummary` aggregate and product statistics surfaces still exist because
only the throughput subset (volume, terminal success, cycle time, cost, and
tokens) has same-snapshot parity coverage. Legacy breakdown, tool-usage,
error-pattern, and time-series questions still lack typed measures and parity.
**Priority:** P0 analytics completion.
**Constraint:** Do not remove the legacy aggregate contract or raw-event
analytics until every documented question has a parity result and its machine
consumer has migrated.

### Resolved: Legacy profile inputs
Profiles store one portable `roleRef`. Database startup applies the current
declarative schema without changing persisted operator data. Historical runs
keep their persisted snapshot (or their honest snapshot-less runner/model
projection) without consulting current policy.

### TD-001: Template README Cleanup
**Description**: Generated README.md is template boilerplate, needs replacement with scenario-specific content.
**Priority**: Should be done during initial development.

### TD-002: UI Placeholder
**Description**: UI is minimal scaffold from template; dashboard will need full implementation.
**Priority**: Deferred to OT-P2-007.

## Resolved Incidents

### R-005: Model-policy hard cutover omitted first-party consumers (2026-07-10)
**Symptom**: Managed `test-genie` startup failed to compile after the generated
agent-manager contract removed `ModelPreset`; prompt-manager's manual JSON
client still compiled but would have sent the now-unknown `model_preset` field.
**Root cause**: The hard-cutover consumer inventory covered agent-manager and
scenario-owned profile JSON, but did not search the entire repository for typed
proto consumers and manual HTTP projections before deleting the generated enum
and field.
**Fix**: Migrated test-genie, scenario-to-cloud, system-monitor,
scenario-to-desktop, and prompt-manager heartbeat adapters to portable
`roleRef` values and updated their contract tests. The supported scenario
profile files use the same role contract.
**Prevention**: Proto hard cutovers require a repo-wide structural consumer
search that includes generated-type imports and manual JSON field projections;
target-scenario compilation alone is not a sufficient consumer matrix.
**Validation**: All affected adapter packages pass focused tests, stale
production references are absent, and test-genie starts healthy through the
managed lifecycle.

### R-004: workspace-sandbox unavailable during sandboxed run setup/finalization (2026-05-19)
**Symptom**: Default sandboxed runs could fail at `sandbox_creating` with `SANDBOX_CREATE` caused by `connect: connection refused` when workspace-sandbox had stopped or was still starting after agent-manager boot. Completed runner turns could also fail post-turn checkpoint/apply when workspace-sandbox became unavailable before finalization.
**Root cause**: Agent-manager bootstrap correctly relies on Vrooli lifecycle to start declared dependencies, but individual run setup did not re-check or recover if workspace-sandbox later became unhealthy. Sandbox create/apply/checkpoint made one HTTP attempt and surfaced terse run summaries.
**Fix**: Fresh sandbox setup now performs a bounded provider health check, invokes the `WorkspaceSandboxEnsurer` seam on run-time unavailability, and retries transient create failures with the same `sandbox:run:{runID}` idempotency key. Post-turn checkpoint/apply retries retryable transport failures after one ensure attempt. The production ensurer delegates startup to `vrooli --no-stale-check scenario start workspace-sandbox`, coalesces same-process ensure calls, and leaves cross-process locking to lifecycle.
**Validation**: `internal/orchestration/phases/setup_test.go`, `internal/orchestration/phases/finalize_test.go`, `internal/orchestration/workspace_sandbox_ensurer_test.go`, and `internal/config/levers_test.go` cover recovery, retry bounds, stable idempotency, and concurrency coalescing.

### R-003: Global run-event streaming and WebSocket subscription races (2026-04-30)
**Symptom**: The UI subscribed to all WebSocket events at app startup, which kept run lists fresh but also streamed and retained full event bodies for unrelated runs. Backend broadcast filtering also read per-client subscription fields while the socket read pump could mutate them.
**Root causes**:
1. `subscribeAll` was used as a coarse substitute for a lightweight list-status subscription.
2. The run event store treated all live `run_event` messages as timeline state, regardless of selected-run subscription intent.
3. WebSocket client subscription state had no single synchronization boundary between fanout and client message handling.
**Fix**: `RUN_STATUS` delivery is now global metadata, full run-event/progress payloads remain subscription-scoped, the UI no longer calls `subscribeAll` by default, the store only tracks live events for subscribed runs, selected-run coordination moved into `useSelectedRunController`, reconnect decisions became explicit/tested, and backend subscription fields are guarded.
**Validation**: `go test -race ./internal/handlers -run 'WebSocketHub|Broadcast|Subscription'`, UI type-check/unit tests, orchestration/domain lifecycle tests, lint, and scenario validation were run during the hardening pass.

### R-002: Split realtime state caused stale run timelines and action flags (2026-04-30)
**Symptom**: `App.tsx` and `RunsPage.tsx` both consumed WebSocket messages and reconciled run events independently. Fixes to one path left another stale path behind, especially around reconnects, terminal status updates, and stop/continue action flags.
**Root causes**:
1. WebSocket subscriptions were sent only when the socket was open, so desired subscriptions could be lost across reconnect.
2. Selected-run events, run snapshots, last sequence, terminal reconciliation, and action hydration lived in component-local state instead of one reducer.
3. Backend append/broadcast and stop/continue status mutation paths had duplicate sequencing and hydration logic.
**Fix**: The realtime event architecture pass introduced durable append-before-broadcast, a shared status transition helper, durable WebSocket subscription intent, and a single UI run event store with REST `after_sequence` gap-fill.
**Validation**: Backend event/lifecycle tests, UI type-check/unit tests, and targeted handler/CLI tests were run during the pass. Full scenario validation remains the final rollout gate.

### R-001: Silent launch failure after protected-sandbox cutover (2026-04-28)
**Symptom**: swarm-manager initiative-feedback runs landed in `RUN_STATUS_NEEDS_REVIEW` after ~134ms with 0 assistant messages and exit code 0. The runner never produced output; the run looked complete.
**Root causes** (four stacked defects):
1. SandboxLauncher posted the *host* merged path as `WorkingDir` to workspace-sandbox `/processes`. Inside the bwrap mount namespace the merged dir is bind-mounted at `/workspace`; the host path does not exist there, so bwrap exited 1 with `Can't chdir to ...: No such file or directory` before claude launched.
2. workspace-sandbox `StreamProcessLogs` raced the wait reaper for fast-failing processes — the SSE stream closed before `RecordExit` ran, so no `event: exit` was emitted.
3. `sandboxLaunchedProcess.finalizeWaitErr` treated missing exit info as success.
4. swarm-manager profile hardcoded `ManualReview=true`, so even silent failures landed in NEEDS_REVIEW.
**Fix**: committed 2026-04-28. The fix translates paths at the SandboxLauncher boundary, adds `WaitForExit` server-side, surfaces `ErrSandboxNoExitInfo` and emits stderr on success, adds `validateRunOutcome` to demote silent successes, and removes ManualReview from the swarm-manager profile.
**Affected commits**: `3e8b004704` through `26af7314ab` (Sandboxing auto-approval p1..p5).
