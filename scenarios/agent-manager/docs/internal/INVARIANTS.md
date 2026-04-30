# Invariants

These statements are **normative**: every agent making code changes is expected to preserve them. If a refactor cannot preserve an invariant, the refactor is wrong (or the invariant has changed and this document needs to change first).

Invariants are paired with the test that pins them — a regression that violates an invariant should fail that test, not just leak past code review.

## I1. Run-mode is a function of `SandboxConfig.Mode`

**Statement.** `SandboxConfig.Mode` is the single source of truth for whether a run is sandboxed. The only function that translates `SandboxConfig` to `RunMode` is `domain.DeriveRunMode`. Callers that need to override the derived mode pass an explicit `req.RunMode` (highest priority) or `req.ForceInPlace` (only honored when policy permits). No other input may decide RunMode.

**Mapping (the entire decision):**

- `SandboxModeOff` → `RunModeInPlace` (explicit no-sandbox)
- `SandboxModeUnspecified` → effective `Tracking` → `RunModeSandboxed`
- `SandboxModeTracking` → `RunModeSandboxed`
- `SandboxModeProtected` → `RunModeSandboxed`
- nil `SandboxConfig` → `RunModeInPlace` (treated as Off; in practice the orchestrator always populates a non-nil cfg)

**Why a `SandboxMode` enum and not a parallel `bool`.** Earlier iterations of this code carried a separate boolean field on the run config and on the agent profile that nominally answered the same question ("is this run sandboxed?"). The consequence of having two answers to one question was that the bool's Go zero-value (`false`) silently overrode the safe default whenever a caller forgot to set it — turning the "sandbox by default" invariant into a silent in-place fallback. The visible failure mode was sandboxed runs whose `cwd` recorded as the canonical repo, agent edits hitting the canonical repo directly, and an audit trail (the workspace-sandbox merged dir) that stayed empty.

A `SandboxMode` enum has no such pit. `Effective()` resolves an unspecified value to `Tracking` — which `DeriveRunMode` treats as sandboxed — so a default-constructed config still gets the safe answer. There is no second field that can disagree.

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

## I7. Greenfield rule

**Statement.** When a function, field, or file is replaced, it is deleted in the same commit that introduces the replacement. No `// DEPRECATED:`, `// LEGACY:`, `// TODO: remove`, `// formerly:`, or `// removed:` comments. No re-exports from old locations. No "fallback" code paths kept "for callers that haven't migrated."

**Why.** Compat shims accumulate without expiry. Two implementations of the same concept inflate the cognitive surface of the code, and "// TODO: remove" comments rot — what was supposed to be temporary becomes permanent. The explicit alternative is to migrate every caller in the same commit; the rule of thumb is that if a phase ends with both an old and a new implementation present, the phase is not done.

**The exception.** Proto schemas reserve the wire ID *and* the field name when a field is removed; this is required by the protocol to prevent silent reuse. Reservation comments are present-tense ("reserved to prevent reuse"), not narrative ("was removed in Phase 1").

## How to add an invariant here

1. The statement must be checkable. "Don't do X" without a test that fails when someone does X is not an invariant — it's wishful thinking.
2. Pair the statement with the test that pins it. If the test doesn't exist yet, add it before adding the invariant.
3. Pair the statement with a "why" paragraph that explains the cost of violating it. Future agents need to know whether to preserve the invariant or revisit it as the codebase evolves.
4. If the invariant subsumes a previous assumption from `ASSUMPTIONS.md`, move the entry — don't leave both.

## Related

- [`SEAMS.md`](SEAMS.md) — what is mockable, where the seam lives, and the test that exercises it.
- [`ASSUMPTIONS.md`](ASSUMPTIONS.md) — facts the code assumes about the world; weaker than invariants because they may stop holding.
- [`TEMPORAL-FLOWS.md`](TEMPORAL-FLOWS.md) — every cadence and timing-sensitive contract, paired with its lever and test.
- [`../reference/configuration.md`](../reference/configuration.md) — every adjustable threshold (the control surface).
