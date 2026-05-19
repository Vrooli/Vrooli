# Configuration Reference — Levers (Tunables)

Every adjustable threshold in agent-manager lives in a single `config.Levers` struct (`internal/config/levers.go`). New durations, counts, or buffer sizes are added here — never as hard-coded literals scattered through source. This is the **only home for adjustable thresholds**, per the architecture's invariant 4.

The struct splits into fourteen sections; jump to the one that controls the behavior you're tuning.

| Section | Purpose | Audience |
|---|---|---|
| [`Execution`](#execution) | Per-run timeouts, turn limits, event buffering | Operator |
| [`Safety`](#safety) | Accident prevention (sandbox-by-default, etc.) | Operator |
| [`Concurrency`](#concurrency) | Parallelism + resource caps | Operator |
| [`Approval`](#approval) | Review workflow timings | Operator |
| [`Runners`](#runners) | Per-runner availability checks, probe timeouts | Operator |
| [`Server`](#server) | HTTP/WebSocket server tunables | Operator |
| [`Storage`](#storage) | Persistence retention windows | Operator |
| [`Spawn`](#spawn) | Runner-startup serialization + queue depth | Operator |
| [`Sandbox`](#sandbox) | workspace-sandbox availability and retry bounds | Operator |
| [`Observability`](#observability) | Structured-log format and verbosity | Operator |
| [`Heartbeat`](#heartbeat) | Run-lifecycle cadence | Internal |
| [`Recovery`](#recovery) | Transcript-tail and resume-after-restart timing | Internal |
| [`Scanner`](#scanner) | stdout/transcript buffer ceilings | Internal |
| [`Diagnostics`](#diagnostics) | Silent-launch detection, log truncation | Internal |

The "Internal" sections back individual machinery; operators rarely touch them directly. They're tuned via test-fast overrides in unit tests so the production defaults stay realistic.

## Loading order

`Levers` is constructed via `config.DefaultLevers()` at startup. The orchestration service applies any `OrchestrationSettings` overrides from the database via `executorLevers()` in `service.go`, then injects the merged value into `RunExecutor` via `WithLevers(...)` and into `Reconciler` via `WithReconcilerLevers(...)`.

Validation happens at construction (`Levers.Validate()`); invalid values fail fast at startup rather than producing strange runtime behavior.

## Execution

Run-level execution behavior.

| Field | Type | Default | Range | What it controls |
|---|---|---|---|---|
| `DefaultTimeout` | `time.Duration` | 30m | 1m–4h | Maximum execution time for a run when no profile-level timeout is set. |
| `DefaultMaxTurns` | `int` | 100 | 1–1000 | Conversation-turn cap to prevent runaway loops. |
| `EventBufferSize` | `int` | 100 | 10–10000 | Events buffered before flushing. Higher = better throughput, more memory. |
| `EventFlushInterval` | `time.Duration` | 1s | 100ms–30s | How often buffered events flush. Lower = more responsive streaming, more I/O. |

## Safety

Accident prevention; these exist to prevent operator mistakes, not adversarial attacks.

| Field | Type | Default | What it controls |
|---|---|---|---|
| `RequireSandboxByDefault` | `bool` | true | All runs use the overlayfs sandbox unless explicitly overridden. |

(Plus other safety levers — see source for current set.)

## Spawn

Runner-startup serialization. The `spawn.Dispatcher` is the single entry point for starting any run (CreateRun + ResumeRun); these levers control how many runs may be in the codex bootstrap window simultaneously and how the queue behaves under burst load.

The `Dispatcher`'s startup-window cap exists because codex's bootstrap (SQLite WAL contention, rollout-file open race, in-memory writer registration) burst-fails silently when N>1 starts overlap. The default of 1 means strict serialization of *startup*, while *running* runs proceed in parallel. Lift only after the burst-test confirms the runner tolerates parallelism in your environment.

| Field | Default | Range | What it controls |
|---|---|---|---|
| `MaxStartingConcurrency` | 1 | 1–16 | How many runs may be in the codex-bootstrap window simultaneously. The default of 1 is the safe choice; lift after burst-testing. |
| `MinSpacing` | 0 | 0–30s | Minimum delay between two successive slot-acquisition events. Useful when `MaxStartingConcurrency > 1` still produces transient races. Zero disables. |
| `QueueCapacity` | auto | 0 (auto) or 1–1024 | Maximum number of runs queued (not yet started) before `Enqueue` returns `*domain.CapacityExceededError` (HTTP 429, `Resource: "spawn_queue"`). Zero auto-derives from `Concurrency.MaxConcurrentRuns * 2`. |

`CreateRunResponse` carries `queue_depth`, `active_count`, `starting_count` populated from `Dispatcher.Stats()` so UI/CLI callers see backpressure on every accept response — there is no separate stats endpoint.

When the queue is full, callers receive `domain.ErrCodeCapacityRuns`. UI should surface this as transient backpressure, not a hard failure; the caller can retry with backoff.

Tests: `internal/orchestration/spawn/dispatcher_test.go` (unit) + `internal/orchestration/integration/spawn_serialization_test.go` (full orchestrator burst gate).

## Sandbox

Run-time workspace-sandbox availability and retry bounds. These levers apply when a sandboxed run is creating a new sandbox or finalizing post-turn checkpoint/apply. They do not replace Vrooli lifecycle's bootstrap dependency startup for agent-manager.

| Field | Default | Range | What it controls |
|---|---|---|---|
| `AvailabilityCheckTimeout` | 2s | 100ms–10s | Deadline for a single provider health check before deciding workspace-sandbox is unavailable. |
| `EnsureStartTimeout` | 60s | 5s–2m | Bound for lifecycle start plus health polling when workspace-sandbox is unavailable at run time. |
| `EnsurePollInterval` | 500ms | 50ms–5s, less than `EnsureStartTimeout` | Poll cadence while waiting for workspace-sandbox health after lifecycle start. |
| `OperationMaxAttempts` | 4 | 1–8 | Maximum attempts for retryable sandbox create/checkpoint/apply transport failures. |
| `OperationInitialBackoff` | 250ms | 25ms–5s | First retry delay for transient sandbox operation failures. |
| `OperationMaxBackoff` | 2s | initial backoff–30s | Cap on exponential retry delay. |

The production `WorkspaceSandboxEnsurer` delegates startup to `vrooli --no-stale-check scenario start workspace-sandbox`, coalesces same-process ensure calls, and relies on lifecycle's scenario lock for cross-process contention.

## Observability

Structured-logging output. Logging is centralised in `internal/orchestration/obs`; these levers feed `obs.Init` at server startup. The set is deliberately tiny — log shape is a contract, not a control surface.

| Field | Default | Allowed values | What it controls |
|---|---|---|---|
| `LogFormat` | `text` | `text`, `json` | Selects the slog handler. Use `text` in development for human readability; `json` in production for log aggregators. |
| `LogLevel` | `info` | `debug`, `info`, `warn`, `error` | Minimum slog level to emit. Lower = noisier, useful when reproducing a spawn-bootstrap failure; higher = cleaner steady-state logs. |

Stable log keys are declared as constants in `internal/orchestration/obs/log.go` (`KeyRunID`, `KeyRunMode`, `KeyPhase`, …). Adding a new key is a contract change — never log ad-hoc string keys.

## Heartbeat

Run-lifecycle cadence. The single home for the deferred-finalize teardown timeout, run-progress heartbeat interval, agent-idle threshold, and runner-signal grace period.

| Field | Default | What it controls |
|---|---|---|
| `RunHeartbeatInterval` | 15s | How often the executor pings `Run.LastHeartbeat`. The reconciler uses this + StaleThreshold to detect stalled runs. |
| `CheckpointInterval` | 1m | How often the checkpoint store is saved. |
| `TeardownTimeout` | 30s | Bound on `Finalize`'s detached HTTP teardown context. The 2026-04-28 mount-leak gate. |
| `AgentIdleThreshold` | (per-runner) | How long the runner can be silent before idle warnings fire. |
| `RunnerSignalGracePeriod` | (per-runner) | Wait between SIGTERM and SIGKILL when stopping a runner. |

## Recovery

Transcript-tail and resume-after-restart cadence. Drives `Reconciler.startTailer` and `RecoverInFlightRuns`.

| Field | Default | What it controls |
|---|---|---|
| `TranscriptTailInterval` | 100ms | How often the recovery tailer polls the transcript file for new lines. |
| `RunStateRetentionDays` | (operator) | How long completed run state directories are kept on disk. |

## Scanner

Buffer ceilings for stdout and transcript readers. Higher = handles bigger lines without truncation, more memory.

| Field | Default | What it controls |
|---|---|---|
| `StdoutBufferSize` | 10MB | Maximum line length the runner stdout scanner can hold. |
| `TranscriptBufferSize` | 10MB | Same, for transcript replay. |

## Diagnostics

Heuristic windows used by `phases.ValidateRunOutcome` and stderr truncation.

| Field | Default | What it controls |
|---|---|---|
| `LaunchFailedMaxDuration` | 2s | Sub-2s sandboxed runs with zero message events get demoted to `SANDBOX_LAUNCH_FAILED`. Pin this when bwrap chdir failures masquerade as success. |
| `RateLimitTruncate` | 512B | How much error-message text is preserved in rate-limit warnings. |

## Adding a new lever

1. Pick the section that owns the behavior. If none fits, add a new section type.
2. Add the field with a doc comment that names the trade-off (higher vs lower) and the valid range.
3. Wire it into `DefaultLevers()` with the production-realistic default that matches the literal it replaces.
4. Add a `Validate()` clause if the field has bounds.
5. Replace every literal use of the old constant with `levers.<Section>.<Field>`.
6. Update this document.

The greenfield rule applies: literal constants are deleted in the same commit that introduces the lever — no `// was 60 * time.Minute` comments left behind.
