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

### R-001: Silent launch failure after protected-sandbox cutover (2026-04-28)
**Symptom**: swarm-manager initiative-feedback runs landed in `RUN_STATUS_NEEDS_REVIEW` after ~134ms with 0 assistant messages and exit code 0. The runner never produced output; the run looked complete.
**Root causes** (four stacked defects):
1. SandboxLauncher posted the *host* merged path as `WorkingDir` to workspace-sandbox `/processes`. Inside the bwrap mount namespace the merged dir is bind-mounted at `/workspace`; the host path does not exist there, so bwrap exited 1 with `Can't chdir to ...: No such file or directory` before claude launched.
2. workspace-sandbox `StreamProcessLogs` raced the wait reaper for fast-failing processes — the SSE stream closed before `RecordExit` ran, so no `event: exit` was emitted.
3. `sandboxLaunchedProcess.finalizeWaitErr` treated missing exit info as success.
4. swarm-manager profile hardcoded `ManualReview=true`, so even silent failures landed in NEEDS_REVIEW.
**Fix**: see `docs/plans/sandbox-launch-and-auto-approve-fixes-plan.md` (committed 2026-04-28). Phase A translates paths at the SandboxLauncher boundary; Phase B adds `WaitForExit` server-side; Phase C surfaces `ErrSandboxNoExitInfo` and emits stderr on success; Phase D adds `validateRunOutcome` to demote silent successes; Phase E removes ManualReview from the swarm-manager profile.
**Affected commits**: `3e8b004704` through `26af7314ab` (Sandboxing auto-approval p1..p5).
