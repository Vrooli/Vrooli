# Problems & Known Issues: agent-manager

## Open Issues

### Planned maintenance and startup readiness — 2026-09-12

The composition root now registers the maintenance schema, installs one durable
admission gate in orchestration, and mounts `/api/v1/maintenance/admission` using
the existing routed SQL middleware and canonical verified-human owner authority.
New runs, continuations, resting-review resumes, failed-run replacements,
workflow starts/retries and new attachments are fenced. Accepted replays,
pending recovery, parked wakes and durable active-workflow descendants retain
their existing execution owner. Force flags cannot bypass maintenance.

GET returns a stable closed revision and nested durable/physical inventory.
Terminal status does not exclude a live executor (including the false-FAILED
USI PID315576 evidence); resting review without an executor or finalization is
not active work. History/accounting is retained. Missing-root evidence remains
unknown until a control-plane reader proves complete descendant/group exclusion.
The production adapter now calls `vrooli runtime executor-scope --json` with
JSON stdin containing durable run IDs, current/legacy tags, PID/PGID and optional
start/end timestamps. It requires exit 0, `schemaVersion=executor-scope-v1`,
complete coverage and an explicit `absent` verdict for exclusion. Present scoped
PIDs remain visible; missing, contradictory, partial or unknown evidence blocks
drain. Indirect dependency restart exclusion and host remediation remain in the
control plane, not AM.

The revised owner source contract permits 16,384 references / 8 MiB input and
output with one bounded 4-second indexed host scan. AM mirrors these bounds and
makes at most one scan per inventory attachment, with one
identity-only SQL hydration; it does not scan once per historic PID or SQL page.
The history-scale regression also exposed repeated SQL-page sorting consuming
the five-second attachment deadline. Candidate classification now streams one
SQL result into bounded references and samples, closing it before hydration and
the host read. The attachment deadline is unchanged.
Read-only SQL on 2026-09-12 found 10,811 retained physical references (10,095
attached, 715 codec-pipe, 1 interactive). Imported history has no retained physical
references at this observation cut. Managed codec-pipe/interactive history alone
has 716 references, so excluding imported rows could not resolve the old 128-ref
capacity gap. All 10,811 references fit the revised bound. The USI durable run
now records running PID1957076 at this later cut; no inventory operation changes
its accounting or executor. Positive false-FAILED PID315576 residue remains a
regression fixture. Main must coordinate adoption of both owner and AM binaries;
an old 128-ref owner command still cannot qualify this history. AM reports any
capacity violation and never discards rows or loops over host scans.
Root command installation and lifecycle bootstrap are separate
main/control-plane owner operations.

Fresh recovery uses the same client through
`maintenance.ExcludeExecutor(ctx context.Context, run *domain.Run) error`.
It returns nil only for complete exact-scope absence. Positive, unknown, partial,
unavailable or cancelled evidence returns an error. The continuation owner holds
its admission claim across this observation and dispatch; this read grants no
lifecycle authority and contains no private host walker.

Resume nonblockingly takes
`<owner-home>/.vrooli/state/locks/scenario-agent-manager.lock` before the gate
mutex and holds it through revision-CAS persistence. GET never takes that lock.
The mounted handler advertises `lifecycleInterlock: "scenario-lock-v1"` only with
this interlock installed. The root lifecycle bridge requires a positive stable
revision, `closed=true`, `drained=true`, zero `admitting` and `remaining`, and
empty `inventory.work`, `inventory.executors`, and `inventory.unknown` with
`inventory.remaining=0`. No absent or partial projection is an empty proof.

Human owner commands after endpoint bootstrap (not executed by this worker):

```bash
agent-manager maintenance status --json
agent-manager maintenance begin --reason "planned AM rollout" --local-owner --json
agent-manager maintenance drain --timeout 120s --local-owner --json
```

Only after the complete root proof succeeds may the owner perform its separately
authorized normal lifecycle adoption, for example `make -C scenarios/agent-manager
restart`. After startup readiness is true, reopen explicitly with the retained
closed revision:

```bash
agent-manager maintenance status --json
agent-manager maintenance resume --revision <closed-revision> --local-owner --json
```

Timeout, unknown physical scope or an unavailable API is not restart authority.
An unavailable old API cannot bootstrap its own new endpoint; that recovery is
the main/control-plane owner's separate decision. Do not restart AM or one of
its dependency parents from an admitted agent. No force, accounting purge,
credential elevation or process termination is part of this protocol.

Startup recovery runs asynchronously with per-step 30-second contexts. `/health`
is HTTP200 with `readiness=false` while initializing; `/api/v1/health` and the
Connect Health RPC remain unavailable until ready. Rebuildable historical
accounting/statistics and other-scenario declaration failures remain health
findings, not permanent global readiness failures. Required ownership and AM
self-declaration recovery remain readiness-gated. Expected security lifecycle
refusals are operating findings, not liveness failures; the refusal guard stays
unchanged. These distinctions do not claim useful recovered-work progress or
complete unknown accounting under the effort recovery completion gate.

Deterministic maintenance tests cover concurrent admission, preserving work on
timeout, abrupt-process restart, resting review/history, false-FAILED executors,
unknown physical scopes, root-lock ordering/persistence, and initializing versus
ready. Live lifecycle adoption and complete host-scope exclusion are separate
owner integration obligations, not implied by these local tests.

Focused validation checkpoint (2026-09-12 15:51 UTC): the following checks pass
after main restored generated AM/PM contracts. The historical-readiness regression
was red before the classification repair. No shared owner was restarted.

```bash
cd scenarios/agent-manager/api
go test ./internal/maintenance -race -count=1 -timeout=60s
go test . -run '^TestRouter(MountsDurableMaintenance|ReportsInitializing)' -race -count=1 -timeout=60s
go test ./internal/orchestration -run '^(TestMaintenance|TestContinuationAccepted|TestParkRun_AndWake|TestWakeRun_Concurrent)' -race -count=1 -timeout=60s
go test ./internal/wiring -run '^TestLifecycleRefusals' -race -count=1 -timeout=60s
go build ./...
cd ../cli
go test . -run '^TestMaintenanceCLI' -race -count=1 -timeout=60s
go build ./...
```

An additional adjacent workflow-launcher race check failed in `SendHeartbeat`
versus `runFromDomain` during starting-state persistence. It is retained through
the report-bug skill as `knw-1789228281794385886`, not hidden by the focused green
selection. That owner repair is separate from maintenance admission.

Follow-up regressions reproduced a CLI evidence loss: the shared client returns
503 bodies through `APIError.RawResponse`, while maintenance read only the nil
response bytes. The CLI now displays the bounded error-envelope state (closed
revision, inventory and unknown evidence) in JSON and human output, retaining a
nonzero exit. Missing state never becomes a fabricated open revision zero.
Decode/output is capped at 256 KiB.

The imported-history regression also reproduced a false logical blocker.
Explicit `execution_mode=imported` is a read-only projection even if historical
status is `unknown` or `running`; it must not count as admitted work or contribute
retained historical PIDs to AM's physical inventory. An imported row claiming
active finalization is an ownership contradiction and must remain explicit
unknown, not empty proof. Managed and legacy unknowns, positive managed terminal
executor evidence, and managed active finalization remain in scope. Tests retain
every historical row. A private workflow continuation
now has a focused test through successful mock-runner completion while admission
stays closed; a new explicit continuation of the same run remains refused.

Final owner-client checkpoint: the complete maintenance package passes with
`-race`, including 16,384 exact executor identities and a 10,811-reference SQL
inventory under the unchanged observation deadline. Router, scoped admission /
continuation, wiring, and CLI maintenance race regressions pass; API and CLI
`go build ./...` pass. Red-to-green evidence covers imported retained PIDs, the
old 128-reference client limit, repeated SQL-page deadline exhaustion, and
canonical CLI help. Help now shows required begin reason (1-512 bytes), resume
revision, drain timeout bounds, and optional explicit local-owner authentication.

At main's 16:36 live cut, three `swarm-manager/goal-session-drain` executions
remain cancelling: `391b1e6d-3b4b-40ef-9256-356cfb3183c9`,
`e0353446-ad61-40ab-8551-362281c0d8cc`, and
`f5bb29d2-6f27-427c-bb44-57685a24eb2a`. Read-only joins show one dispatched
fresh-run attempt each, `child_finalization_pending`, terminal children with
PID0, and incomplete parent accounting / unmeasured charge. The owning path is
`orchestration/workflow_execution.go::RecoverWorkflowExecutions` through
`cleanupWorkflowChildren`, then
`workflowruntime/engine.go::recordCleanupDisposition` and
`engine_cleanup_accounting.go::rebuildOrdinaryUsage`. Operator cancellation
explicitly requires original accounting even during recovery. This is an owner
accounting reconciliation boundary, not authority to hide cancelling workflows
in maintenance or fabricate zero usage. No workflow or run rows were changed.

#### Terminal receipt filtering correction — 2026-09-12

Scoped W3 repair, following the owner's instruction to supersede only samples
before an authoritative receipt. Two hypotheses were confirmed independently:
the projection's `i != selectedIndex` predicate removed later usage/charges, and
the meter's latched terminal-authority flag accepted later unfinished usage as
settled even after that filtering predicate was corrected. The regression was
red for later usage, unpriced charge and positive metered charge; correcting
only the predicate left the later-usage assertion red.

The projection now suppresses only earlier samples in the same invocation.
Later usage withdraws terminal authority until another authoritative receipt;
later unpriced charges remain unknown, positive charges remain measured, and
invalid later accounting remains an error. Original event objects and history
are unchanged. No codec, maintenance-safety policy or live state was changed
for this correction. Strict cancellation accounting remains required.

Focused regression and adjacent original-receipt recovery validation:

```bash
cd scenarios/agent-manager/api
go test ./internal/orchestration -run '^(TestWorkflowTerminalReceipt|TestWorkflowReceipt|TestContinuationReceipt|TestWorkflowMeter|TestWorkflowRecoveryReconcilesOriginalNativeTerminalReceipt)' -race -count=1 -timeout=90s
go build ./...
```

### OpenCode provider/session isolation and supervision recovery — 2026-09-12

The managed USI failure was not evidence of exhausted OpenRouter credit.
Interactive OpenCode explicitly selected OpenRouter through global `model` and
`small_model`; managed USI explicitly selected the Go subscription. Private
`XDG_DATA_HOME` isolated sessions but originally omitted the provider-auth link.
The earlier link repair is retained. This follow-up honors a selected inherited
data root without borrowing a different store's login, refreshes only the link,
and reads failure logs from the affected run's data directory. Credential bytes
remain in their owner store, and terminal skill cleanup preserves sessions.
Focused credential-root, replay and log-classification regressions pass.

Normal lifecycle adoption completed under
`startop-dfc66e8e0833bf53dbe40f3be080d12b` at 13:44:39 UTC. USI subsequently
launched fresh attempt 41, run `45871a66-b5a7-4af0-b9a6-4897e94771f4`, with the
pinned Go model and nine successful tool calls at the observation cut. Its live
report had not populated actual-model or final usage; do not treat those fields
as zero or a completed qualification. Old unusable sessions were not continued.

Supervision now observes bounded declared resolution-source hashes and legacy
operator answers, separates observation roles from business runtime and permits
explicitly granted failed-run CONTINUE with a session and recovery hypothesis.
It does not synthesize a new session, start a shell driver, or derive grants from
files. Missing-session pre-admission qualification and recurring legacy-driver
control/grant integration remain owner work. See the cross-owner contract and
`docs/agent-system/effort-supervision-validation.md` for exact limits.

Restart also reconfirmed slow synchronous historical recovery before HTTP
readiness (`main.go::startRecovery`); terminal-accounting warnings were retained,
not erased to make startup green. Prior investigation already recorded this
startup ordering in imported run `2b1afd7c-d98f-42ce-af96-8a9c949ef5b9`, event
`71b80d01-2d20-4c96-b17b-abb9630c2ac3`. Reuse that evidence for owner repair;
do not infer this turn introduced the historical recovery debt.

### Continuation credential generation — 2026-09-12

Effort-supervision's disposable directive was delivered but its target's signed
acknowledgment was refused as unauthenticated. A realistic continuation regression
reproduced `identity token has been revoked`: minting replaced the credential hash
but retained the previous turn's revocation timestamp. Fresh-run assessment worked,
which falsified a broad Codex shell-environment failure.

The identity phase now gives each mint a distinct generation, persists its hash
with cleared prior-generation revocation, and preserves owner/scopes. Terminal
continuations revoke that fresh credential; parked handling remains separate.
Tests prove old credentials remain invalid, the fresh credential works during the
turn, terminal use is rejected, and same-second issuance cannot revive an old token.
Focused continuation/park/verification, identity/phase suites and race regressions
pass. See `continue_env_identity_test.go` and `phases/identity_test.go`.

Status: repaired and live-qualified for the bounded continuation case. After
normal lifecycle adoption, run `3795a4d0-0ef7-45d4-a093-58c1361d7e5d` accepted
the original directive `25755827-b98e-4a95-9e76-e5deda8aa23d` using its own
signed identity (revision 4, `ACCEPTED`). No operator token or second directive
was substituted. Failure, repair and qualification limits remain in
`docs/agent-system/effort-supervision-validation.md`; this is not general crash
recovery or autonomous business-steering certification.

### Swarm engagement budget qualification — 2026-09-08

Prior evidence: Swarm's contract-development readiness review identified
post-child token accounting as insufficient for approved aggregate ceilings
(`scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md`). This pass tested two
competing causes: missing child usage versus admission paths ignoring known usage.

Three deterministic regressions failed before the fix: an invalid structured
result at the token limit scheduled a repair; a node with a 50-turn/1000-second
allowance received it despite only 10 turns/10 seconds remaining; and an exact
token-limit result could lead into another agent node. Directly supplying usage
confirmed control-flow/admission gaps, not missing metering, in those cases.

The sequential interpreter now checks known spend before new work and before
repair, retains actual overrun usage, and clips supported child turn/time limits
to the remaining workflow allowance. Valid exact-limit completion is distinct
from permission to start more work. An owner-issued `WorkflowEngagementGrant`
is now admitted only when it narrows the pinned declaration, persisted with the
execution, reapplied after restart, and protected against idempotency mutation.
The grant is exposed in the operator execution projection. The entire
`internal/workflowruntime` package passes, including `-race`; see
`engine_budget_admission_test.go` and `engagement_grant_test.go`.

Still unqualified: hard in-flight token/charge ceilings, unknown/live usage,
concurrent-child reservations, nested workflow allowance inheritance, Swarm
per-engagement dispatch fencing, native-goal/fallback continuation, and
provider-owned terminal evidence. The existing workflow revision budget is not
by itself a Swarm engagement reservation. Do not enable contract-development
launch based solely on this repair. These changes were not activated by
restarting Agent Manager's shared live service.

Scoped Test Genie unit run `20260909-002357-03cecc07` returned FAIL, reporting
execution-readiness, architecture and coverage findings. This pass does not
attribute those broader findings to the admission repair or claim a clean
scenario certificate from the passing workflow-runtime tests.

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

- Proposed plan-triggered investigation scope (2026-09-05): W0 contract extension required before implementation. The operator requested reusable, meaningful investigations with explicit stale/missing evidence and memory improvement. `OT-P0-012` promises "durable invocation facts" and `OT-P0-014` promises "Durable Plan-Family Supervision"; neither explicitly defines a reusable finite investigation with clean, inconclusive, and actionable outcomes. This is a scoped design finding, not a new full-scenario gate verdict; the prior recall validation below remains unchanged.
- Investigation evidence: `.vrooli/agent-manager/investigate.json` requires nonempty categories and recommendations and couples diagnosis to approval/apply. `api/internal/orchestration/run_report.go` checks watermark presence, not coverage against the report event boundary. `api/internal/orchestration/investigation_findings.go` derives evidence-quality flags from keywords and associates each recommendation with every source run. `CreateInvestigationRun` generates a fresh internal idempotency key on every call. These source findings require targeted behavioral validation before repair claims.
- Live investigation-tool observation (2026-09-05): `agent-manager run report 2f952a9c-f8fc-4df8-a1cd-ff9edf6e6cd1` reported 28 unresolved Bash calls and 28 successes under an unnamed tool. `agent-manager run investigate --help` exposed no option details. The historical example run `8e868f7c-7565-4b06-bec9-335081b125b7` returned 404. No investigator was launched and no scenario suite was run for this exploratory assessment.

- Rung: W3 / R4 validated for searchable conversation history
- W0 evidence: target `OT-P0-013` defines durable, attributable prior-conversation recall as an Agent Manager capability, distinct from distilled memory and repository provenance.
- W1 evidence: requirements `REQ-P0-015` through `REQ-P0-026` specify canonical projection, text/regex/semantic/hybrid retrieval, stable identity, privacy deletion, bounded context, lifecycle recovery, Search Hub federation, governed program reuse, UI/CLI surfaces, telemetry, and retention behavior.
- W2 evidence: comprehensive Test Genie run `20260905-054133-a6dcfb61` produced the requirements-sync snapshot; Agent Manager requirement validation now passes at L3 with the conversation module complete and linked to executable tests. Search Hub owns the federated mapping/routing evidence and Meta Optimization Manager owns answer-space row 37.
- W3 / R4 evidence: the promoted live projection has exact catalog/FTS parity, optional semantic degradation preserves lexical service, event-specific append reconciliation is bounded, cold observational counts do not block retrieval, the governed recall program completed successfully, final direct/federated overlay evals both pass 3/3, and Meta Optimization Manager reports `answer/37` as `NOW` with condition `ok`.
- Measured: 2026-09-05

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


### 2026-09-05 — Supervision empirical calibration and model identity remain open

**Symptom:** `friction-digest` leaves supervision coverage unknown and cannot
claim live policy improvement without assessed candidate decisions.

**Cause:** Owner coverage now counts retained outcomes, assessments, decisions,
actions, applied actions, observed children and families before response sampling.
That population does not establish the number of eligible but never-watched
children. Candidate creation freezes the evaluator artifact; first inference
binds effective provider, model and applied parameters. A mutable provider model
alias still does not identify underlying weights or a route-catalog revision.

**Completion boundary:** Add an owner measure of eligible/watched child intervals
and missed terminal observations. Extend the effective inference identity with
an authoritative route/model revision when AI Gateway exposes one; current
provider/model/parameter drift already fails closed. Collect operator-reviewed
positive and negative cases and a bounded
candidate rollout before promotion. Fixtures establish behavior, not empirical
completion benefit. Keep coverage, unassessed labels, and absent baselines null.

**Owner:** Agent Manager supervision, consuming AI Gateway attribution.
**References:** `docs/reference/configuration.md`, `skills/agent-manager-improve/SKILL.md`,
`.vrooli/program-runtime/friction-digest.py`.

## Work ladder

- Rung: W0/W3 targeted extension (2026-09-06); full scenario certification remains open.
- Evidence: Named goals include Agent Manager investigation surfaces, while the approved plan adds a caller-neutral durable investigation contract, subject-specific finding edges, and bounded programs. Focused implementation tests pass; the full scenario gate was not used as evidence because the shared checkout has unrelated inherited failures.
- Targeted W3 evidence (2026-09-06): investigation contract coverage, typed orchestration attribution, handler lifecycle, prose-quality conservatism, verification-aware reread detection, and caller-supplied domain-evidence allow-listing pass. The active `agent-manager/investigate-typed` workflow is revision 1.2.0 with typed diagnosis fields and bounded evidence references.
- Learning boundary (2026-09-06): completed typed results now own one stable Memory attempt and expose a diagnosis-preserving pending retry. Live Memory completion and independent advice-benefit measurement remain unassessed.
- Owner-suite evidence (2026-09-06): run `20260906-054134-20a28539` failed broad inherited/shared checks, including manifest/schema, UI, dependency, documentation, performance, unit-placement, storage, security, measures, proto, template, and event-declaration checks. It is retained as a certification limitation, not collapsed into the targeted result.
- Blocker: The approved plan's PRD/requirements extension and full owner certification are not yet complete. The execution baseline receipt is partial because source identity changed before producer work began.
- Measured: 2026-09-06

## 2026-09-14 — Supervisor friction findings preserve mitigation boundaries

Recurring friction is detected from durable invocation episodes and routed through
the existing `report-friction` / meta-optimization intake. Findings now retain a
candidate workaround, its acceptance impact, and a suspected owner as structured
metadata. The recurring publisher explicitly records that its candidate is not an
approved workaround; owner review remains required before reuse or promotion.

Focused evidence: findings, Prompt Manager client, orchestration, and legacy
finding-column migration tests pass. The broader package run passes the changed
findings/database/orchestration/supervision/prompt-manager packages; an unrelated
retained-run configuration fixture still fails in the broader handlers package.
