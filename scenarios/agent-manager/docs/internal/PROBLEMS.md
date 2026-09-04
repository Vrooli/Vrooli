# Problems & Known Issues: agent-manager

## Open Issues

### P-013: Some evidence planes are intentionally unavailable without runtime signals (2026-08-04)
**Severity**: Medium
**Description**: Transcript evidence is now fully governed and replayable, but
receipt joins only become confident when the target scenario emits a matching
receipt. Meta-optimization coverage cells also report observed adherence as
unavailable until an Agent Manager adherence reader is configured; they must
not infer adherence from declared skills or from transcript command names.
Optional provider credentials remain absent in the local validation environment,
so model-backed investigation reruns cannot be treated as measured reasoning
quality.
**Mitigation**: Every unavailable surface carries an explicit reason and keeps
its denominator/omitted/unmatched counts. Native and imported receipt
availability remain separate. Deterministic friction and evidence-quality
signals are still validated without inventing model conclusions.
**Status**: Deliberate honesty boundary; add the runtime readers and provider
credentials before claiming those measurements are available.

### R-007: Dead scope-lock code pretended to serialize editors (resolved 2026-09-02)
**Symptom**: On 2026-09-02 one agent session deleted another's tolerance
table and tests within minutes of their creation; nothing on the host could
say which sessions were editing the tree.
**Root cause**: `LockRepository`, `LockManager`, `domain.ScopeLock` and the
`scope_locks` table were unwired dead code: the orchestrator held a `locks`
field no caller ever set, so the lock surface read as a capability while
serializing nothing. The launcher also never sent a working directory, so the
`runs` row could not name a tree.
**Fix**: The lock repository, manager, domain type, validation, table and
tests were deleted. Visibility now lives where the process does: the launcher
records an editor lease (tree, scope, pid, claims) in the control plane's
runtime registry, sends `working_dir` and `scope` at attach, and advisory
claims name an overlapping holder at launch (`docs/reference/agent-sessions.md`).
**Validation**: `TestLauncherSendsWorkingDirAndScope` and
`TestClaimOverlapNamesHolderAndContinues` (`packages/cli-core/cliutil`);
`TestEditorLeaseExpiresOnlyOnProofOfDeath` (`internal/scenarioruntime`);
agent-manager's domain and database suites pass without the lock code.

### R-006: Installed resource policy CLIs lagged the repository schema (resolved 2026-08-04)
**Symptom**: Investigation creation returned HTTP 400 because every
`code.smart` runner candidate failed resource-role preflight with only
`exit status 1` exposed in the API error.
**Root cause**: The installed `resource-codex`, `resource-claude-code`,
`resource-opencode`, and `resource-grok` binaries predated the resource policy
catalog's `model_aliases` field and rejected the current JSON before emitting a
role response. Agent Manager's resolver also discarded the command diagnostic
when classifying the non-zero exit.
**Fix**: Reinstalled all four resource CLIs through `vrooli resource install`,
added a bounded command-diagnostic suffix to resource-resolution errors, and
added a regression test covering the `unknown field "model_aliases"` failure.
**Validation**: All four installed CLIs resolve `code.smart`; the live
investigation run `8e868f7c-7565-4b06-bec9-335081b125b7` passed preflight,
selected Codex `gpt-5.6-sol`, completed with a schema-valid structured result,
and required only the expected manual review. Agent Manager API tests and all
four resource CLI test suites pass.

### P-014: Imported-run persistence nullability defects resolved (2026-08-04)
**Severity**: High (resolved)
**Description**: Full-corpus adoption exposed two empty-value defects that
native runs did not exercise: `runs.canary_arm` and
`invocation_read_model_watermarks.last_event_at` were bound as SQL NULL while
their durable schemas require non-NULL values.
**Mitigation**: Empty canary arms are bound as explicit empty strings, and an
empty projection uses its projection timestamp as the watermark time. Focused
regression tests cover both paths; the governed corpus was re-synced and
projection-refreshed after the fixes.
**Status**: Resolved and validated against the live corpus.

### P-010: Historical analytics are bounded by read-model coverage (2026-08-02)
**Severity**: Medium
**Description**: Stats measures are projection-backed and report validity,
source window, filters, and history floor. Runs older than the available
read-model window cannot be presented as complete history.
**Mitigation**: The Stats page defaults to a seven-day window, displays the
earliest available read-model timestamp and outside-history run count, and
keeps legacy operational health endpoints separate from analytical measures.
Rebuild retained invocation evidence before making historical comparisons.
**Status**: Deliberate durability boundary; full-history reconstruction remains
dependent on retained source events.

### P-011: Subscription charge allocation is not automatic (2026-08-02)
**Severity**: Medium
**Description**: Runner billing declarations and subscription periods are
available, but a subscription fee is not silently allocated across workloads
or runs. Allocation requests without an explicit basis are rejected.
**Mitigation**: Inspect `charge_by_basis`, configure non-overlapping operator
subscription periods, and use an explicit allocation basis once a pricing
allocator is available. Unknown and unpriced consumption remain visible.
**Status**: Deliberate safety boundary; do not infer accounting treatment from
provider labels or token volume.

### P-012: Legacy stats transport remains for operational compatibility (2026-08-02)
**Severity**: Low
**Description**: The Stats page's analytical panels use typed Connect measures,
while the older REST stats handler and operational fallback endpoints remain
in the server for existing clients and health surfaces.
**Mitigation**: New analytics must use the measure registry and evidence
metadata. Legacy endpoints are not authoritative for the Stats page and are
tracked for a later compatibility retirement.
**Status**: Intentional migration seam.

### P-006: Consumption and charge model rollout
**Severity**: Medium
**Description**: Consumption, charge, yield, billing basis, and workload identity are now separate durable facts. Historical rows with retained events can be rebuilt with `agent-manager run replay-invocation-corpus`; rows whose source events were pruned are explicitly reported as unreplayable.
**Mitigation**: Run bounded replay windows and compare `invocation_read_model_runs.total_tokens` with the joined `runs.summary.tokensUsed` oracle. Agent Manager currently starts in documented best-effort mode while the `workspace-sandbox` dependency is unhealthy.
**Status**: Resolved in implementation; final suite and live oracle evidence remain release-validation records.

### P-009: Legacy analytical columns and external goal snapshots retired
**Severity**: Low (migration)
**Description**: The read model no longer stores separate authoritative/estimated/unknown cost columns or Codex goal-token snapshots. Historical event JSON remains readable through normalization, while canonical consumption is projected from usage payloads and run summaries provide the independent reconciliation oracle.
**Mitigation**: Startup rebuilds only the affected SQLite read-model table, copying retained analytical columns before dropping retired fields. Final validation inspects the live schema and checks the token oracle.
**Status**: Resolved in implementation; retain this entry as the migration record.

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

## UX Issues

### Resolved: Import did not match runner-backed conversation workflow (2026-07-30)

The initial import surface required a local file upload and mobile navigation
used a bottom-positioned menu. Import now reads the session locations declared
by coding-agent resource manifests, lets an operator choose a runner and select
saved conversations, and marks conversations already associated with a run.
The header hamburger opens the same primary navigation in an overlay drawer on
mobile; desktop retains the persistent sidebar.

## Work ladder

- Rung: W2
- W0 evidence: the goal `git-control-tower-ai-provenance` directs "Turn Git Control Tower into the primary operator surface for understanding which agent-manager runs produced the current repository state." `OT-P0-012` promises only that "The system shall retain durable invocation facts and provide typed historical run analytics with non-blocking event-capture observability," so it does not own conversation import, unknown-run message discovery, privacy-aware deletion, or federated retrieval. `OT-P0-013` now separately promises attributable conversation recall through API, CLI, UI, and Search Hub while retaining a degraded lexical path. The other goals that name Agent Manager (`ecosystem-intelligence-loop`, `phone-agent`, and `swarm-manager-feature-parity`) neither remove nor contradict that capability.
- W1 evidence: requirement module `MOD-P0-013` links the user-observable import, retrieval-mode, provenance, pagination, privacy, deletion, freshness, reindex, CLI, UI, federation, and skill/program obligations to `OT-P0-013`; `business-health validate scenario agent-manager` and `vrooli scenario requirements validate agent-manager` are the structural gates.
- Remaining W2 work: the new requirements intentionally remain `planned`; each validation note names its exact future test, while `ref` remains empty until that file exists so the registry does not fabricate evidence. Bind each reference only when the named test is implemented. Do not claim W3 until direct, federated, deletion, degradation, accessibility, and operational evidence is live-traced.
- Measured: 2026-09-04

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

### TD-003: Resolved — durable analytics parity and raw aggregate retirement (2026-07-30)
**Resolution:** Every retained statistics question now reads
`invocation_read_model_*` projections. The compatibility summary/drill-down
transport remains for product callers, but it no longer computes from raw
event JSON. Same-snapshot repository coverage includes status, success,
duration, cost/token provenance, runner/profile/model breakdowns, tools,
errors, time-series, and pricing model catalog.

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

### R-006: Stats projection stopped at replay boundary (2026-07-30)
**Symptom**: The Stats page showed no data despite completed Codex runs. A
manual invocation-corpus replay populated historical metrics, but newly
completed executor runs still did not appear.
**Root cause**: The executor's finalization seam persisted terminal runs
directly. It bypassed the shared status-transition helper, which was the only
path that invoked the durable invocation read-model projection.
**Fix**: The executor now invokes a best-effort terminal observer after final
state persistence. Agent Manager wires that observer to the durable projection,
so normal and resumed executor runs update Stats without replay.
**Validation**: API orchestration tests pass. In the live scenario, a completed
Codex smoke run increased the selected profile's Stats count from 1 to 2
without replay; terminal trends showed the new run.

### P-006: Receipt projection policy engine remains externally owned (2026-07-30)
**Status**: Deliberate dependency.
**Detail**: Agent Manager preserves opaque receipt projections and reports
`policy_absent` when Vrooli Events provides no policy version or projected
fields. Enabling scenario-specific projection fields requires the Vrooli Events
receipt projection policy engine; Agent Manager must not infer or hardcode
response keys.

### P-007: Endpoint generation target was unavailable (resolved 2026-07-30)
**Status**: Resolved in Agent Manager.
**Detail**: `api/cmd/gen-endpoints` now derives the endpoint inventory from
the mux route registrations and served Connect descriptors. `make endpoints`
regenerates `.vrooli/endpoints.json` deterministically; the former blocker
`knw-1785387885739969825` is no longer applicable.

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
