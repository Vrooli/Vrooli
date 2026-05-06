# Problems & Known Issues: agent-manager

## Open Issues

### P-001: workspace-sandbox Availability
**Severity**: High
**Description**: agent-manager requires workspace-sandbox to be running for all sandbox operations. If workspace-sandbox is unavailable, all sandboxed runs fail.
**Mitigation**: Health checks on workspace-sandbox before run creation; graceful degradation messaging.
**Status**: Design consideration - will be addressed in implementation.

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

## Technical Debt

### TD-001: Template README Cleanup
**Description**: Generated README.md is template boilerplate, needs replacement with scenario-specific content.
**Priority**: Should be done during initial development.

### TD-002: UI Placeholder
**Description**: UI is minimal scaffold from template; dashboard will need full implementation.
**Priority**: Deferred to OT-P2-007.

## Resolved Incidents

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
