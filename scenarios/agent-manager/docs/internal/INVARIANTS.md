# Invariants

These statements are **normative**: every agent making code changes is expected to preserve them. If a refactor cannot preserve an invariant, the refactor is wrong (or the invariant has changed and this document needs to change first).

Invariants are paired with the test that pins them — a regression that violates an invariant should fail that test, not just leak past code review.

## I1. Run-mode is a function of `SandboxConfig.Mode`

**Statement.** `SandboxConfig.Mode` is the single source of truth for whether a run is sandboxed. The only function that translates `SandboxConfig` to `RunMode` is `domain.DeriveRunMode`. Callers that need to override the derived mode pass an explicit `req.RunMode` (highest priority) or `req.ForceInPlace` (only honored when policy permits). No other input may decide RunMode.

**Mapping (the entire decision):**

- `SandboxModeOff` → `RunModeInPlace` (explicit no-sandbox)
- `SandboxModeUnspecified` → effective `Protected` → `RunModeSandboxed`
- `SandboxModeTracking` → `RunModeSandboxed`
- `SandboxModeProtected` → `RunModeSandboxed`
- nil `SandboxConfig` → `RunModeInPlace` (treated as Off; in practice the orchestrator always populates a non-nil cfg)

**Why a `SandboxMode` enum and not a parallel `bool`.** Earlier iterations of this code carried a separate boolean field on the run config and on the agent profile that nominally answered the same question ("is this run sandboxed?"). The consequence of having two answers to one question was that the bool's Go zero-value (`false`) silently overrode the safe default whenever a caller forgot to set it — turning the "sandbox by default" invariant into a silent in-place fallback. The visible failure mode was sandboxed runs whose `cwd` recorded as the canonical repo, agent edits hitting the canonical repo directly, and an audit trail (the workspace-sandbox merged dir) that stayed empty.

A `SandboxMode` enum has no such pit. `Effective()` resolves an unspecified value to `Protected` — which `DeriveRunMode` treats as sandboxed — so a default-constructed config still gets the safe answer. There is no second field that can disagree.

**The lesson, as a rule for new fields:** when a Go bool's zero value would silently invert a safety-relevant decision, the bool is the wrong shape. Use an enum where the zero value is either explicitly invalid (caller must choose) or maps to the safe default.

**Tests:**
- `internal/domain/decisions_test.go::TestDeriveRunMode` — every Mode value + nil cfg.
- `internal/domain/types_test.go::TestRunConfig_ApplyProfile/preserves cfg.SandboxConfig when profile does not specify one` — profile-pointer-clobber regression gate.
- `internal/orchestration/integration/sandbox_cwd_contract_test.go` — end-to-end: sandboxed run records a sandbox-managed `cwd`, never the canonical repo.

## I2. `spawn.Dispatcher.Enqueue` is the only entry point for starting a run

**Statement.** Every run begins by passing through `spawn.Dispatcher.Enqueue`. `CreateRun` and `ResumeRun` both go through it. `go executeRun(...)` (or any other unbounded goroutine spawn for run execution) is forbidden.

**Why.** Codex's bootstrap window (SQLite WAL contention + rollout-file open race + in-memory writer registration) burst-fails silently when N>1 starts overlap. Heartbeat-driven callers can fire several `CreateRun` calls on the same tick. Without a single serialization choke point, every caller would have to know to back off — which is fragile and was the root cause of multiple "runs stop emitting after initial events" production incidents.

The dispatcher caps the *startup* window's concurrency (default `MaxStartingConcurrency=1`); already-running runs proceed in parallel. Backpressure surfaces on every `CreateRunResponse` via `queue_depth`, `active_count`, `starting_count`.

**Tests:**
- `internal/orchestration/spawn/dispatcher_test.go` — unit: capacity, panic safety, idempotent `Started()`, `MinSpacing`.
- `internal/orchestration/integration/spawn_serialization_test.go` — concurrent-burst gate.

## I24. Codec-pipe continuation homes are explicit and durable

**Statement.** Codex and Grok codec-pipe execution never inherit `CODEX_HOME`
or `GROK_HOME`. Each run receives an Agent-Manager-owned home below its durable
run directory, and every execute/continue turn for that run uses the same home.
Credential seed files are removed only after terminal execution; rollout files
remain available for replay and diagnosis.

**Why.** The sandbox overlay is ephemeral across turns. Inheriting a
web-console home can be read-only, while relying on sandbox `$HOME` loses the
Codex rollout required by `exec resume`.

**Tests:** `internal/orchestration/session_home_test.go` and
`internal/orchestration/continue_env_identity_test.go`.

## I25. Storage roots are explicit or request-routed

No production code resolves Agent Manager run-state storage from ambient
process state. A root is injected at construction or selected from the request
context by routed file storage; unit tests use an explicit temporary root.

**Tests:** `internal/runstate/root_test.go` and `internal/archtest/boundary_test.go`.

## I26. Declared controls name a real runtime consumer

Every exported `domain.RunConfig` and grouped `config.Levers` field is
classified against a named non-test runtime consumer. Retention controls name
their reconciler sweepers explicitly, so loading and validating a lever alone
cannot be mistaken for implementing it.

**Tests:** `internal/contracts/run_control_test.go`.

## I27. Codec capabilities agree with installed CLI surfaces

Codec capability declarations and generated argv must agree with the installed
runner's help output. A missing optional binary is a named skip; an available
binary that contradicts a declaration fails conformance.

**Tests:** `internal/adapters/runner/codecs/cli_help_conformance_test.go`.

## I28. Terminal runs persist sandbox-reported attribution

When a sandbox apply or checkpoint completes, the terminal run persists the
sandbox's applied-file count, applied byte total, diff artifact path, and
non-empty commit hash. Agent Manager does not recount the workspace; the
sandbox result is authoritative. Tracking-mode interactive runs use the same
finalization path, while Protected interactive runs remain rejected.

**Tests:** `internal/orchestration/phases/finalize_test.go` and
`internal/orchestration/run_executor_lifecycle_test.go`.

## I29. Codec controls have one translation seam

`Codec.ControlArgs` is the only runner-specific translation of model, effort,
allowed tools, and denied tools. Codec-pipe execution and interactive launch
both consume its output. The interactive package must not switch on
`domain.RunnerType` to recreate a translation.

**Tests:** `internal/orchestration/interactive/launch_test.go` and
`internal/archtest/boundary_test.go`.

## I30. Declared effort values are evidence-backed

A codec advertises only effort levels with a declared native mapping. Where a
CLI publishes an accepted value domain, conformance parses it and rejects
extra declarations; where it does not, the declaration is deliberately narrow
and documented. Resolving an unmapped effort emits the existing
unsupported-effort warning instead of silently dropping the control.

**Tests:** `internal/adapters/runner/codecs/cli_help_conformance_test.go` and
`internal/adapters/runner/codecs/capabilities_conformance_test.go`.

## I3. `Codec.ClassifyTerminalError` is the only codec-side error classifier

**Statement.** When a run exits non-zero, `core.Runner` calls `Codec.ClassifyTerminalError(stderr, exitCode)` and stores the typed `*domain.RunnerError` (if any) on `ExecuteResult.TerminalError`. `phases.ExecuteAgent` lifts that into `EmitFailureEvent`'s typed-error branch, where it lands on the run timeline as the typed `ErrorCode` rather than `INTERNAL`.

There is no `Detect…` proliferation. New failure shapes are added by extending the switch inside the codec, not by adding more boolean predicates next to it.

**Why.** Operators consume run-timeline error codes; a single `INTERNAL` bucket for everything codex/claude/opencode might fail with hides recoverable failures (e.g. session expired on resume) inside the same code as true internal errors. Distinct codes let the UI and downstream callers route correctly — and let regression tests pin the classification of known stderr fixtures.

**Codec → code mapping (current):**

| Codec    | Pattern                                          | ErrorCode                          |
|----------|--------------------------------------------------|------------------------------------|
| codex    | `record_rollout_items` + `thread … not found`    | `RUNNER_SESSION_STATE_LOST`        |
| codex    | `thread … not found` (no rollout-writer context) | `RUNNER_SESSION_EXPIRED`           |
| claude   | `session` + `not found`                          | `RUNNER_SESSION_EXPIRED`           |
| opencode | `session` + (`not found`\|`expired`\|`invalid`)  | `RUNNER_SESSION_EXPIRED`           |

**Tests:**
- `internal/adapters/runner/codecs/codex_test.go`, `claude_test.go`, `opencode_test.go` — pattern-by-pattern classification.
- `internal/orchestration/phases/emitters_test.go::TestEmitFailureEvent_TypedRunnerError_PreservesCode` — the regression gate against `INTERNAL` leakage.

## I4. Lifecycle events are emitted through `obs/events.go` only

**Statement.** Every spawn → exit → finalize transition lands on the run timeline as an `EventTypeLifecycle` event with a typed `LifecycleEventData` payload. Helpers in `internal/orchestration/obs/events.go` are the *only* construction site — direct construction of `LifecycleEventData` outside `obs/` is a contract violation.

**Why.** A small fixed taxonomy gives operators (and the UI) a stable spine to debug against. Ad-hoc `NewSystemEvent("spawn started", …)` calls would put the same information on the timeline under different shapes, defeating any tooling that filters or aggregates by lifecycle phase.

**Taxonomy:** `spawn_enqueued`, `spawn_started`, `runner_acquired`, `runner_exited`, `finalize_started`, `finalize_completed`. Adding a new transition means adding a constant and a helper, not a one-off site.

**Tests:**
- `internal/orchestration/obs/events_test.go` — every helper emits exactly one event of the expected type; nil-sink safety.

## I5. Structured slog with stable keys

**Statement.** All orchestration / runner / phases / codecs logging goes through `obs.Logger()` (or its context-scoped variants `obs.L`, `obs.Component`, `obs.RunCtx`). All dynamic values are key/value pairs using the `KeyXxx` constants declared in `internal/orchestration/obs/log.go`. `log.Printf` is forbidden in this layer.

**Why.** Operators consume logs via `slog`-format-aware tooling (text in dev, JSON in prod). Stable keys are a contract: dashboards, log filters, and incident-response queries all depend on `runID`, `runMode`, `phase`, etc. being spelled the same way every time. `fmt.Sprintf` log messages defeat this — they put the same information into a different position on every log line.

**Tests:**
- `internal/orchestration/obs/log_test.go` — key uniqueness, format selection, RunCtx threading.

## I6. Auto-approve gates are `ManualReview` / `AutoApply` / `ApplyOnFailure`

**Statement.** `phases.ApplyAtRunEnd` gates only on the three `SandboxConfig` fields:

- `cfg.ManualReview == true` → defer to operator (run lands in `NeedsReview`).
- `cfg.GetAutoApply() == false` → skip apply.
- `cfg.GetApplyOnFailure() == false` (when outcome is failure) → skip apply.

There is **no empty-diff branch** anywhere — code or docs.

**Why.** The product semantics are "if the operator opted in to auto-apply, apply; otherwise hold for review." Making "diff is empty" a fourth gate adds a special case that the operator has to reason about for every run, with no product value (an empty diff applies to nothing — let the apply step be a no-op rather than a special case). Empty-diff display in the UI is independent and lives in `cli/cmd_runs.go` / the run-detail view.

**Tests:**
- `internal/orchestration/phases/finalize_test.go` covers each gate path.

## I7. Runner status is separate from sandbox finalization

**Statement.** `Run.Status` represents runner/turn lifecycle. Post-run sandbox apply/checkpoint outcome is recorded in `Run.FinalizationStatus`, `Run.FinalizationError`, and `Run.FinalizedAt`. A failed checkpoint after runner completion must not leave the run in `RunStatusRunning`.

**Why.** Agents can emit multiple assistant messages before a turn ends, so messages are not terminal. Conversely, once the runner process returns a terminal result, sandbox checkpoint failure is infrastructure finalization, not active runner execution. Mixing those concerns blocks valid follow-up messages and creates impossible states such as `status=running`, `phase=completed`, `ended_at != nil`.

**Tests:**
- `internal/domain/run_actions_test.go::TestRunActionsFor_ContinueReason`
- `internal/orchestration/phases/finalize_test.go::TestApplyAtRunEnd_FailurePreservesSandbox`

## I8. Greenfield rule

**Statement.** When a function, field, or file is replaced, it is deleted in the same commit that introduces the replacement. No `// DEPRECATED:`, `// LEGACY:`, `// TODO: remove`, `// formerly:`, or `// removed:` comments. No re-exports from old locations. No "fallback" code paths kept "for callers that haven't migrated."

**Why.** Compat shims accumulate without expiry. Two implementations of the same concept inflate the cognitive surface of the code, and "// TODO: remove" comments rot — what was supposed to be temporary becomes permanent. The explicit alternative is to migrate every caller in the same commit; the rule of thumb is that if a phase ends with both an old and a new implementation present, the phase is not done.

**The exception.** Proto schemas reserve the wire ID *and* the field name when a field is removed; this is required by the protocol to prevent silent reuse. Reservation comments are present-tense ("reserved to prevent reuse"), not narrative ("was removed in Phase 1").

## I9. Global permission reconciliation is explicit, resource-owned, and auditable

**Statement.** Agent Manager may reconcile only the active whole-document
portable permission catalog after explicit human authorization. Resource CLIs
are the sole owners of native syntax and native-file writes; Agent Manager
persists only catalog-qualified, per-resource reconciliation metadata. A
resource failure or unavailable resource never becomes a global success, while
an optional unavailable resource does not by itself make readiness unhealthy.

**Why.** Global desired permissions must remain inspectable without duplicating
resource-specific configuration engines or letting a detected agent mutate
native policy silently. Treating a partial multi-resource apply as success
would conceal drift. Conversely, making every optional resource failure a
readiness outage would misrepresent the catalog's enforcement contract.

**Tests:**
- `internal/permissionpolicy/service_test.go::TestReconcileRequiresExplicitAuthorization`
- `internal/permissionpolicy/service_test.go::TestReconcilePersistsPartialFailureInDeterministicOrder`
- `internal/permissionpolicy/service_test.go::TestReadinessRequiresCurrentHardEnforcementEvidence`
- `internal/permissionpolicy/audit_store_test.go::TestSQLiteAuditStoreRoundTripsOnlyReconcileMetadata`

## I10. Role selection is portable; execution uses an immutable resolved snapshot

**Statement.** New profile and run-create intent names a portable `roleRef`,
never a concrete runner or model. Agent Manager resolves the ordered
cross-runner candidate list through resource-owned role policy before
execution, persists that result as the execution policy snapshot, and retries
or resumes only from the stored snapshot. Current role policy must never
rewrite a historical run's concrete candidate evidence.

**Why.** Resource-owned model inventories can change independently of an
in-flight or historical run. Re-resolving those rows against mutable policy
would make fallback behavior and audit evidence non-reproducible, while
accepting runner/model inputs would duplicate resource authority in consumer
configuration.

**Tests:**
- `internal/rolepolicy/state_test.go` — strict activation, atomic reload, and previous-revision retention.
- `internal/rolepolicy/resolution_test.go` — portable role resolution and immutable candidate construction.
- `internal/orchestration/phases/execute_test.go::TestPolicySnapshotFallbackUsesPersistedCandidatesAcrossRunners` — fallback execution never mutates the stored snapshot.
- `internal/orchestration/phases/execute_test.go::TestPolicySnapshotResumeStartsAtPersistedCandidate` — resume continues from stored evidence rather than current policy.

## I11. Workflow agent identity is explicit per attempt

**Statement.** Every run node attempt creates a distinct fresh Run with its
own node-local profile or role. Only a continue node may reuse conversation
state, and it names a completed ancestor run node; no current/latest Run is
inferred.

**Why.** Silent conversation reuse couples unrelated slices, defeats bounded
context, and makes restart or fan-out ancestry ambiguous.

**Tests:** `internal/workflowruntime/engine_test.go` covers distinct loop Run
ids and named continuation; `internal/workflowcatalog/catalog_test.go` covers
invalid continuation sources and order.

## I12. Workflow side effects follow durable intent

**Statement.** Input/prompt snapshots and a stable attempt idempotency key are
committed before Run creation or continuation. Completion evidence, journal
entries, budget usage, and edge movement commit atomically under execution CAS.

**Why.** A crash at either side-effect boundary must recover by observation,
never by guessing whether an agent turn was sent.

**Tests:** `internal/workflowruntime/engine_test.go::TestRecoveryReusesPersistedDispatchIntentExactlyOnce`
and `internal/database/repository_workflow_test.go::TestWorkflowExecutionRepositoryCASAndJournalSurviveReload`.

## I13. Results are turn-scoped

**Statement.** When provider turn identity is present, only events belonging to
the latest durable turn may compete for the canonical RunResult. Earlier turns
remain historical evidence but cannot make a continued handoff ambiguous.

**Tests:** `internal/orchestration/run_result_test.go`.

## I14. Wait signals are visit-scoped

**Statement.** Every traversal of a wait node creates a new correlation key,
resume token, and deadline. A signal committed for an earlier visit cannot
satisfy a later traversal of the same node.

**Tests:** `internal/workflowruntime/engine_test.go::TestRevisitedWaitRequiresVisitScopedSignal`.

## I15. Abnormal terminals require child cleanup evidence

**Statement.** Cancellation remains non-terminal while child cleanup is
incomplete. Failed, budget-exhausted, and cancelled executions are recoverable
until the current retry generation has a durable cleanup disposition.

**Tests:** `internal/workflowruntime/engine_test.go::TestCancellationRemainsRecoverableUntilChildCleanupSucceeds` and repository recovery tests.

## I16. Fan-out and concurrency are distinct budgets

**Statement.** `maxConcurrency` limits active member dispatch, not total fork
membership. `any` and `quorum` joins stop durable losers as soon as their
threshold is satisfied and fail as soon as success is impossible.

**Tests:** `internal/workflowruntime/engine_test.go::TestParallelFanoutWiderThanConcurrencyDispatchesInBatches` and `TestParallelAnyJoinDoesNotWaitForHungLoser`.

## I17. The declared-run doctrine has one canonical location and no legacy fallback

**Statement.** Every programmatic (non-chat) agent interaction where code composes the prompt and
code consumes the output is a scenario-owned workflow declaration. Scenario-owned profiles and
workflows live **only** under `.vrooli/agent-manager/`, declared through the single
`dependencies.scenarios.agent-manager.config.declarations` block and discriminated per file by
`schemaVersion`. The retired `.vrooli/agent-profiles/` / `.vrooli/agent-workflows/` directories and
the legacy `config.profiles` / `config.workflows` blocks are rejected at reconcile and flagged by
conformance; no reader code for them remains anywhere but the two designated rejection sites. A
scenario keeps exactly two ends in its own code (typed input snapshot, typed result application); a
one-run feature is still a workflow file, spelled with single-node sugar that canonicalizes to a
digest identical to its explicit form — never a new registrable entity kind.

**Why.** Parallel representations of the same behavior are how the failed swarm-manager
operating-modes implementation grew four overlapping layers. One directory, one config block, one
reconcile entry point, and a hard cutover keep the declaration layer from drifting into dialects. A
dual-read window would silently keep the old glue alive; the rejection is deliberate and
user-mandated.

**Tests:**
- `internal/orchestration/declaration_reconcile_test.go::TestReadScenarioDeclarationConfig` — legacy blocks and old-directory sources rejected with actionable diagnostics.
- `internal/conformance/service_test.go::TestValidateRejectsLegacyLayoutDirectoriesAndBlocks` and `TestRealDeclaringScenariosCleanOnUnifiedLayout` — conformance rejects the old layout and every real declaring scenario is clean on the new one.
- `internal/orchestration/legacy_layout_guard_test.go::TestNoLegacyDeclarationLayoutReaders` — the retired directory literals appear only in the two designated rejection files.
- `internal/workflowcatalog/catalog_test.go::TestSingleNodeSugarCanonicalizesToExplicitDigest` / `TestSingleNodeSugarRoundTripsThroughParse` — sugar is digest-identical to the explicit form.

## I18. Workflow registration validation reaches profile parity

**Statement.** CEL edge conditions compile against the engine's shared workflow CEL environment and
run/continue prompt placeholders cross-check against declared bindings **at reconcile and validate**,
not mid-execution. A blocking diagnostic (CEL compile/type error, unbound placeholder) withholds the
digest so nothing registers; an unused binding is a warning that still registers. The same shared
validators back the mutating reconcile and the read-only conformance surface so the two cannot accept
different manifests.

**Why.** Before this parity, a workflow could register with a CEL or placeholder defect and only
explode mid-run, unlike profiles which validated at registration. Catching it at reconcile with a
precise node/edge path is the difference between a registration error and a production incident.

**Tests:**
- `internal/workflowcatalog/catalog_test.go::TestValidateRejectsMalformedEdgeCondition`, `TestValidateRejectsNonBoolEdgeCondition`, `TestValidateRejectsUnboundPromptPlaceholder`, `TestValidateWarnsOnUnusedBinding`.
- `internal/orchestration/declaration_reconcile_test.go::TestReconcileScenarioDeclarationsWorkflowWithholdOnRegistrationDefect` — the defects are caught at reconcile, not only at file validate.

## I19. promptRef is resolved and digest-pinned at reconcile

**Statement.** A `run`/`continue` node names exactly one of `promptTemplate` or `promptRef`. A
`promptRef` resolves against prompt-manager **at reconcile, before the digest**: the resolved content
is embedded into `promptTemplate` and pinned with provenance (skill id, revision, variant, content
hash). A changed skill resolves to different content, a different digest, and a new revision on the
next reconcile — never a silent behavior change under a fixed digest. A resolution failure (missing
skill, prompt-manager unreachable, no source client) withholds the whole atomic workflow batch, never
a partially-resolved revision.

**Why.** Digest pinning is the catalog's identity mechanism; a revision's behavior must be immutable
even if the referenced skill later changes. Embedding at reconcile makes provenance auditable and
keeps execution-time free of prompt-manager coupling.

**Tests:**
- `internal/orchestration/workflow_promptref_test.go::TestPromptRefResolvesAndPinsProvenance`, `TestPromptRefChangedSkillProducesNewRevision`, `TestPromptRefResolutionFailureWithholdsRevision`, `TestPromptRefMissingSkillWithholdsRevision`, `TestPromptRefRequiresSourceClient`.

## I20. The completion nudge is an idempotent trigger, never a scheduler

**Statement.** When a run belonging to a workflow attempt reaches terminal, the orchestrator resolves
the owning execution (`ExecutionIDForRun`) and enqueues one deduplicated drive on the in-process
`WorkflowNudger`; a child-workflow terminal nudges its parent. The drive re-reads durable state and is
guarded by the engine's optimistic-version CAS, so a nudge racing an explicit `Advance` or a second
nudge is safe (the loser rereads and exits at the fixpoint). The nudge is an optimization over the
crash-safe pull loop; `RecoverWorkflowExecutions` remains the durable backstop that progresses an
execution whose run finished while agent-manager was down. No consumer polls `AdvanceWorkflowExecution`
in the normal flow.

**Why.** The fleet's 500ms–5s poll loops existed because nothing pushed completion. The nudge pushes
the existing pull loop without becoming a new scheduler, preserving the crash-safe, pull-based core
chosen by the reliable-results hardening amendment.

**Tests:**
- `internal/orchestration/workflow_nudge_test.go` — dedupe, concurrent drain (-race), wait-registry notify.
- `internal/orchestration/workflow_nudge_integration_test.go::TestCompletionNudgeAdvancesRunNodeWithoutConsumerAdvance`, `TestCompletionNudgeToleratesConcurrentExplicitAdvance`, `TestCompletionNudgeProgressesAcrossSimulatedRestart`.

## I21. The blocking wait never mutates the execution

**Statement.** `WaitWorkflowExecution(executionId, timeout)` long-polls server-side until the
execution is terminal or the deadline passes, mirroring test-genie's `WaitRun`. It only reads
execution state: **cancelling the waiter never cancels the execution**, and the deadline yields a
`timed_out` result with the execution unchanged, never a hang past the deadline. The execution is
driven independently by the engine, the nudge, and the reconciler backstop.

**Why.** This is the documented adoption pattern that replaces consumer pollers. If the wait could
cancel or otherwise perturb the execution, a client's timeout or disconnect would corrupt server-owned
state — the same coupling test-genie's run-wait contract avoids.

**Tests:**
- `internal/orchestration/workflow_wait_test.go::TestWaitWorkflowExecutionReturnsImmediatelyForTerminal`, `TestWaitWorkflowExecutionBlocksUntilTerminal`, `TestWaitWorkflowExecutionRespectsDeadline`, `TestWaitWorkflowExecutionSurvivesConcurrentWaiters`, `TestWaitWorkflowExecutionCancelDoesNotCancelExecution`.

## I22. Self-registration bypasses only the dependency gate

**Statement.** Agent Manager registers its own declarations from
`scenarios/agent-manager/.vrooli/agent-manager/` at startup with owner `agent-manager`, through the
**same** shared validators and ownership rules as any other scenario — it bypasses only the
requirement to declare a dependency on itself, never a validator. A broken self source fails in
isolation and never blocks agent-manager readiness (per-source isolation for profiles; atomic withhold
for the workflow batch).

**Why.** Reconcile otherwise enforces `owner == declaring scenario` and a declared dependency, which
agent-manager cannot satisfy for itself. The seam must not become a second, weaker validation path, or
agent-manager's own declarations (the investigation workflow) could register something a dependent
scenario could not.

**Tests:**
- `internal/orchestration/declaration_reconcile_test.go::TestReconcileSelfDeclarationsBypassesDependencyGate`, `TestReconcileSelfDeclarationsEnforcesValidators`, `TestReconcileSelfDeclarationsIsolatesPerSourceFailure`.

## I23. Tool restrictions survive policy fallback

**Statement.** Before each policy candidate launches, Agent Manager checks that
the selected runner enforces a declared `allowedTools` list. Under enforced
policy an unsupported candidate is skipped and recorded; advisory policy is
the only path that may continue without native enforcement. Claude Code also
passes canonical `deniedTools` through `--disallowedTools` on execute and
continuation.

**Why.** Validating only the initially selected runner lets cross-runner
fallback silently remove an operator-visible safety control.

**Tests:**
- `internal/orchestration/phases/execute_test.go::TestToolRestrictionCandidateReasonRejectsUnsupportedEnforcedFallback`.
- `internal/adapters/runner/codecs/claude_test.go::TestClaudeBuildArgsTranslatesCanonicalDeniedTools` and `TestClaudeBuildContinueArgsKeepsCanonicalDeniedTools`.

## I25. Default terminal sandbox disposal is reachable

**Statement.** A default sandbox lifecycle deletes on the closed `terminal`
event vocabulary. Every configured default lifecycle event is emitted by
`LifecycleEventForStatus`; manual review and explicit lifecycle configuration
remain the only ways to preserve a sandbox.

**Why.** A configuration value that no terminal transition can emit silently
accumulates overlays, state rows, and repository-local database growth.

**Tests:** `internal/contracts/run_control_test.go::TestDefaultLifecycleVocabularyIsEmittable`
and `internal/orchestration/sandbox_config_test.go::TestNormalizeSandboxConfig_ManualReviewSkipsDefaultLifecycle`.

## How to add an invariant here

Additional `OT-P2-001` constraints stay in the architecture decision table
until their executable enforcement tests land. Promote them here only with
those tests; an invariant without a failing test is not yet an invariant.

1. The statement must be checkable. "Don't do X" without a test that fails when someone does X is not an invariant — it's wishful thinking.
2. Pair the statement with the test that pins it. If the test doesn't exist yet, add it before adding the invariant.
3. Pair the statement with a "why" paragraph that explains the cost of violating it. Future agents need to know whether to preserve the invariant or revisit it as the codebase evolves.
4. If the invariant subsumes a previous assumption from `ASSUMPTIONS.md`, move the entry — don't leave both.

## Related

- [`SEAMS.md`](SEAMS.md) — what is mockable, where the seam lives, and the test that exercises it.
- [`ASSUMPTIONS.md`](ASSUMPTIONS.md) — facts the code assumes about the world; weaker than invariants because they may stop holding.
- [`TEMPORAL-FLOWS.md`](TEMPORAL-FLOWS.md) — every cadence and timing-sensitive contract, paired with its lever and test.
- [`../reference/configuration.md`](../reference/configuration.md) — every adjustable threshold (the control surface).
