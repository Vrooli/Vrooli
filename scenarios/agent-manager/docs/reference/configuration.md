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

`Levers` is constructed via `config.DefaultLevers()` at startup. The orchestration service applies persisted `OrchestrationSettings` from `config/orchestration.json`, then injects the merged values into run execution and the reconciler. The settings API and `agent-manager settings orchestration` command update that same runtime store.

Validation happens at construction (`Levers.Validate()`); invalid values fail fast at startup rather than producing strange runtime behavior.

## Runtime orchestration settings

The persisted run-execution settings are authoritative at run creation. A
profile or inline run request may select smaller limits, but a request above a
global ceiling is refused with both requested and ceiling values; it is never
silently truncated. The accepted values are copied into `run.resolvedConfig`,
which is the value the executor enforces and the run read APIs report.

| JSON field | Unit | Default | Protects |
|---|---:|---:|---|
| `runExecution.runTimeoutMinutes` | minutes | 120 | Agent work duration for one turn; global wall-clock ceiling, sized to the largest declared fleet profile. |
| `runExecution.maxTurns` | turns | 1000 | Agent autonomy; global conversation-turn ceiling, sized to the largest declared fleet profile. |
| `healthDetection.heartbeatIntervalSeconds` | seconds | 60 | Executor liveness; timer-driven and independent of agent output. |
| `healthDetection.staleThresholdSeconds` | seconds without an executor heartbeat | 1000 | Marks an executor stale before destructive recovery. |
| `healthDetection.maxRecoveryAgeSeconds` | seconds without an executor heartbeat | 1100 | Reaps an unrecoverable live child process. This measured liveness threshold is not an agent work timeout. |
| `healthDetection.reconcilerIntervalSeconds` | seconds | 30 | How frequently stale executor state is examined. |

`heartbeatIntervalSeconds < staleThresholdSeconds < maxRecoveryAgeSeconds` is
required. Settings updates that violate this ordering are rejected.

## Role-policy catalog

Portable coding intent is declared in `config/role-policy-catalog.json`. Each
role contains an ordered list of `(runner, resourceRole)` references only;
Agent Manager does not keep a concrete model inventory here. At run creation,
each resource resolves its role through its own `policy resolve --json`
protocol. The persisted execution snapshot records concrete model/fallback
choices, resource policy path and digest, provenance, permission-enforcement
posture, and unavailable-candidate diagnostics.

| Variable | Default | Effect |
|---|---|---|
| `AGENT_MANAGER_ROLE_POLICY_CATALOG_PATH` | Repository-resolved `scenarios/agent-manager/config/role-policy-catalog.json` | Selects the portable role catalog loaded and reported by Agent Manager. |

The role catalog is a readiness dependency. A failed reload keeps the prior
active revision, and only subsequent runs see a successful new revision.

For a deploy that may require binary rollback, back up the Agent Manager SQLite
database before starting the new binary. Profiles store portable `roleRef`
intent only; historical run snapshots remain the audit source for their
concrete runner/model choices. Evolving local development data is a deliberate,
one-shot maintenance operation performed while the scenario is stopped, never
a startup side effect. Catalog rollback restores the last known-good role
document, then uses `agent-manager role-policy validate` and `agent-manager
role-policy reload`; a rejected reload leaves the active digest unchanged.

## Desired-permission catalog

Global coding-agent permission intent is declared separately in
`config/permission-policy-catalog.json`. Each rule has a stable ID, portable
`bash` matcher, action, rationale, owner, target scope, and an explicit
hard-enforcement requirement. This is not a profile or workspace-sandbox
policy: it is the desired state later projected by each resource's own
`permissions plan|reconcile` adapter.

| Variable | Default | Effect |
|---|---|---|
| `AGENT_MANAGER_PERMISSION_POLICY_CATALOG_PATH` | Repository-resolved `scenarios/agent-manager/config/permission-policy-catalog.json` | Selects the desired-permissions catalog. |

The catalog records exact-byte digests and reloads atomically; a rejected
reload preserves the active revision. Agent Manager may write a temporary
portable document to invoke a resource CLI, but it never reads or writes a
resource's native permission files. Global native desired permissions,
role/fallback selection, and per-run workspace-sandbox/profile restrictions
are distinct safety layers.

Agent Manager activates this catalog at startup and retains the last
reconciliation metadata in SQLite. That evidence contains catalog digests,
resource-reported fingerprints, native target paths, enforcement posture, and
per-resource outcomes; it never contains copies of native resource files. A
reconcile requires the shared explicit-human authorization signal and runs in
deterministic runner/scope order. If any resource is unavailable or fails,
the result remains auditable but is never reported as global success.

The Settings **Permissions** tab is a compact operator surface for the same
whole-document workflow: inspect status and declared rules, validate/reload,
plan/doctor, then reconcile only after explicitly confirming authorization.
It deliberately has no per-rule editor and never exposes native resource
syntax as an Agent Manager input.

Readiness treats required hard-enforcement rules more strictly than optional
resources. A catalog with no such rule remains ready even if an optional
resource is unavailable. When a rule requires hard enforcement, current
reconciliation evidence for the active catalog digest must show a native or
hook-backed resource; missing or stale evidence, or a missing enforcing
candidate, makes `/health` degraded with an actionable operator message.

Runner credentials are optional startup inputs. Agent Manager may start with
every runner unavailable; it records runner readiness through probes and
selects or rejects a runner only when a run is requested. In particular, Codex
can authenticate through its signed-in CLI session, so `OPENAI_API_KEY` is an
optional API-key alternative rather than a scenario-start prerequisite.

## Agent-conformance validation

`agent-conformance` is a read-only Test Genie phase declared by Agent Manager.
It applies only to scenarios that declare Agent Manager or own an agent-profile
file. Its L0–L4 maturity ladder checks a declared dependency that is not
explicitly disabled; every
scenario-owned profile file is declared; `profileKey` ownership and portable
`roleRef` inputs; role catalog resolution; and, at L3, narrowly detected
direct coding-agent executable spawns.
Consumers that request a portable role directly at runtime may have no
scenario-owned profile source. Direct runner/model/policy fields are rejected
as legacy inputs.

At the top rung, conformance also reads Agent Manager's permission-policy
readiness. A required hard-enforcement rule with stale, failed, or unsupported
native projection produces a blocking permission-posture finding; the phase
does not reconcile or write native permission files itself.

When `dependencies.scenarios.agent-manager.config.profiles` is present, its
schema is strict: it must declare `reconcile`, a valid `mode`, and at least one
target-relative `sources` entry. Unknown fields, duplicate or empty sources,
and disabled dependencies are rejected by both dry-run reconciliation and
conformance validation. Omit `config` entirely when a consumer makes only a
direct portable role request and owns no profile source.

The validation provider never reconciles a profile, changes permission policy,
starts a target, or writes target source. Direct-spawn detection is gating after
fleet conformance established that its narrow executable-construction patterns
do not produce false positives. Correct a finding in the owning scenario, verify with
`agent-manager profile reconcile-scenario --scenario <scenario> --dry-run`,
then rerun the phase. User-owned SQLite state is never rewritten during
startup; back it up and perform any required local-data evolution deliberately
while the scenario is stopped.

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

## Workflow

The completion-nudge queue that drives workflow executions forward as their runs finish, so no consumer polls. See [`../internal/TEMPORAL-FLOWS.md`](../internal/TEMPORAL-FLOWS.md) (the "Workflow completion nudge + blocking wait" flow).

| Field | Default | What it controls |
|---|---|---|
| `NudgeWorkers` | 4 | Concurrent goroutines draining the nudge queue. Distinct executions drive in parallel; concurrent drives of one execution are CAS-safe. Range 1–32. |
| `NudgeDriveTimeout` | 5m | Bound on a single nudge-triggered drive; detached from any request context. Range 5s–10m. |

The `WaitWorkflowExecution` RPC timeout is per-request (`timeout_seconds`, 0 = block until terminal), not a lever.

## Workflow-runtime controls

Workflow budgets are authored controls owned by the scenario that owns the
definition. There is intentionally no implicit runtime default: every value is
required, finite, and positive, so omission is a validation error. Agent
Manager owns immutable safety ceilings; changing a ceiling is a contract review
rather than an environment tweak. This keeps a reconciled definition equally
bounded after restart and on every deployment.

| Authored field | Default | Valid range | Authoritative owner |
|---|---:|---:|---|
| `wallTimeSeconds` | none (required) | 1–86,400 | Scenario workflow author |
| `maxTurns` | none (required) | 1–1,000 | Scenario workflow author |
| `maxTokens` | none (required) | 1–10,000,000 | Scenario workflow author |
| `maxChargeMicroUsd` | none (required) | >0–10,000,000,000 (micro-USD) | Scenario workflow author |
| `maxNodeAttempts` | none (required) | 1–10,000 | Scenario workflow author |
| `maxChildren` | none (required) | 1–1,000 | Scenario workflow author |
| `maxConcurrency` | none (required) | 1–64 | Scenario workflow author |
| `maxRecursion` | none (required) | 1–16 | Scenario workflow author |
| `maxRetries` | none (required) | 1–100 | Scenario workflow author |
| `maxWaitSeconds` | none (required) | 1–86,400 | Scenario workflow author |
| edge `maxTraversals` | 0 only for acyclic edges; required for cycle edges | 0–10,000 | Scenario workflow author |

Runtime semantics are fixed: wall time is measured from the durable execution
creation timestamp; wait time consumes neither turns nor tokens; parallel
dispatch never exceeds `maxConcurrency`; child consumption rolls into the
parent budget tree; retry and recursion checks occur before dispatch; and an
exceeded bound produces a typed terminal reason instead of silent truncation.
Catalog validation rejects a definition above any ceiling before activation.

| Control group | Safety rule |
|---|---|
| Result selection | No confidence percentage; weak evidence abstains and extraction output is always locally revalidated. |
| Bindings | Selected items and serialized bytes are bounded; overflow fails with a typed diagnostic and full transcript selection is unsupported. |
| Structured output | Source, schema, candidate, depth, and diagnostics are bounded; secrets are redacted from diagnostics. |

The V1 structured-output bounds are currently contract constants: 32 KiB
canonical schema, depth 32, 64 KiB selected source, 64 KiB candidate, and 8 KiB
diagnostic message. Unknown keywords are rejected. Remote/dynamic references,
recursive schemas, unevaluated keywords, and custom validation code are not
supported. These are safety boundaries rather than runtime operator knobs;
changing them requires a ResultSpec contract review.

`extractionMode` defaults to `deterministic_only`. A caller may explicitly set
`constrained_fallback`; its blank role defaults to the portable
`extract.structured` role. Agent Manager core does not select an Ollama model or
consumer-specific categories. If no extractor adapter is wired, fallback
degrades to the durable `abstained` outcome.

Node-local execution choices are authored contract data, not operator tunables:
a node must say `fresh_run` with its own profile/role or `continue` with a named
prior conversation. There is deliberately no workflow-wide default profile or
implicit conversation-reuse switch.

## Scenario-owned workflow sources

Consumers declare workflow desired state alongside profiles in their Agent
Manager dependency configuration:

```json
{
  "profiles": {"reconcile": true, "mode": "update_if_unmodified", "sources": [".vrooli/agent-profiles/default.json"]},
  "workflows": {"reconcile": true, "sources": [".vrooli/agent-workflows/review.json"]}
}
```

Workflow paths must be unique, relative, symlink-contained regular files no
larger than 256 KiB. `agent-manager workflow validate --file <path>` validates
one document without catalog access. `workflow plan` and
`workflow reconcile-scenario --dry-run` validate all manifest sources without
writes. `workflow reconcile-scenario` and `workflow reload` atomically activate
the complete valid source set; a failed reload preserves the prior revision.
`workflow list`, `get`, and `explain` expose provenance and definitions by
owner/key or digest.

## Retention split

`Storage.EventRetentionDays` controls only the raw `run_events` log (30 days
in production, 7 in development). Those rows can contain full tool output and
are reclaimed in bounded batches by the reconciler. Durable invocation facts,
run summaries, errors, friction episodes, self-report spans, and watermarks
are not retention targets and remain queryable after their source events are
removed.

An event is eligible only after the same atomic projection transaction marks
its run `projection_complete`. If that marker is absent or incomplete, the
reconciler keeps the event regardless of age; retention can therefore lag but
cannot erase evidence before its derived projection commits.

## Adding a new lever

1. Pick the section that owns the behavior. If none fits, add a new section type.
2. Add the field with a doc comment that names the trade-off (higher vs lower) and the valid range.
3. Wire it into `DefaultLevers()` with the production-realistic default that matches the literal it replaces.
4. Add a `Validate()` clause if the field has bounds.
5. Replace every literal use of the old constant with `levers.<Section>.<Field>`.
6. Update this document.

The greenfield rule applies: literal constants are deleted in the same commit that introduces the lever — no `// was 60 * time.Minute` comments left behind.
