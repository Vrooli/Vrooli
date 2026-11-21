# Error Semantics: agent-manager Runner Adapters

This note documents the structured `Details` payload that runner adapters
attach to `execution_error` events, and the accompanying follow-up events
investigators can expect on the event stream.

## Claude Code runner

The Claude Code CLI emits a final `result` stream event at the end of each
run. When `is_error: true`, the adapter emits an `ErrorEventData` with:

| Field | Source | Notes |
|---|---|---|
| `code` | static `"execution_error"` | |
| `message` | synthesized | Always non-empty. Format: `claude-code terminated with is_error=true (subtype=<s>, turns=<n>, duration_ms=<m>)[: <result_text>]` |
| `retryable` | `false` | The CLI already decided to fail; retry policy lives upstream |
| `details.subtype` | `result.subtype` | e.g. `error_max_turns`, `error_during_execution`, `error` |
| `details.num_turns` | `result.num_turns` | Integer — conversation turns at failure |
| `details.duration_ms` | `result.duration_ms` | Integer — wall time reported by CLI |
| `details.session_id` | `result.session_id` | Present when the CLI supplied one |
| `details.result_text` | `result.result` | Raw payload; may be empty string |

A follow-up `warn` log event with message prefix `claude-code stderr tail` is
emitted when buffered stderr is non-empty. Credential-shaped substrings
(`Bearer …`, `sk-…`, `api_key=…`) are redacted to `<redacted>`. The tail is
truncated to 2048 bytes at the nearest UTF-8 rune boundary.

## Idle-stream heartbeat

During the scanner loop, a background goroutine emits a `debug` log with
message prefix `stream idle for` when the stdout pipe has been quiet for
≥ 30 seconds. The event carries a short description of the last event
observed, e.g. `tool_call:Read`. Only one heartbeat fires per idle gap;
the next real event re-arms the warning.

## Auto-compact events

When the CLI emits a `system` stream event whose payload or subtype matches
a known auto-compact marker (`"auto-compacting"`, `"conversation history
has been compacted"`, etc.), the adapter emits a `CompactionEvent` with
`trigger="auto"` instead of a generic "System context received" debug log.

## Why this matters

Prior to the run `66cbfd0a-3467-4b63-9a50-58ad348665b1` investigation
(2026-04-20), an `execution_error` could arrive with just `{"code":
"execution_error"}` — no message, no subtype, no counters. Investigators
had to read runner source to figure out what happened. The schema above
closes that gap.

## Sandbox launch failures (2026-04-28)

Three error codes carry the failure modes that previously masqueraded
as silent successes after the protected-sandbox cutover. All three are
**terminal** — there is no auto-retry path; recovery is operator
inspection of the captured stderr followed by a config fix.

| Category | `ErrorCode` | Trigger | Recovery |
|---|---|---|---|
| Connectivity | `SANDBOX_LAUNCH_FAILED` | Protected-sandbox run with `Success=true`, duration < 2s, and zero `RUN_EVENT_TYPE_MESSAGE` events. The categorizer (`validateRunOutcome` in `internal/orchestration/run_executor.go`) demotes the run to `RUN_STATUS_FAILED` with this code. Typical root cause is bwrap chdir, missing executable, or namespace setup error. | Operator reads `~/.local/share/workspace-sandbox/<sb>/logs/<pid>.stderr.log`, fixes config, re-dispatches. |
| Internal | `RUNNER_TIMEOUT` (existing) | Runner exceeded its `Timeout` budget. | Operator extends timeout or splits the work. |
| Connectivity | `SANDBOX_NO_EXIT_INFO` | Both SSE log streams closed without the workspace-sandbox server emitting `event: exit`. After the `WaitForExit` server-side fix, this is a bug indicator (process became untracked or the connection dropped between exit and notify). | Investigate workspace-sandbox health. |

The matching launcher-side helpers:

- `sandbox.ErrSandboxNoExitInfo` ([CODE: scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go]) — the wrapped error returned by `sandboxLaunchedProcess.Wait` when no `event: exit` is delivered.
- `domain.NewSandboxLaunchFailedError` ([CODE: scenarios/agent-manager/api/internal/domain/errors.go]) — constructs the SandboxError that classifyOutcome routes to FAILED.
- `validateRunOutcome` ([CODE: scenarios/agent-manager/api/internal/orchestration/run_executor.go]) — the categorizer that detects the silent-launch shape.

Stderr captured by the runner now appears on the run timeline as a
warn-level log event even on the success path
(`emitStderrAsWarnOnSuccess` in `claude_code.go`), so operators no
longer need to ssh into the workspace-sandbox host to find launch
diagnostics.

## Workspace-sandbox dependency unavailability (2026-05-19)

Sandbox dependency availability failures are **recoverable only before a
mutation reaches workspace-sandbox**. Agent-manager now distinguishes
run-time workspace-sandbox unavailability from bootstrap dependency
startup:

| Operation | Trigger | Recovery |
|---|---|---|
| `SANDBOX_CREATE` / `operation=create` | Provider health check fails or create transport fails before a response. | Call `WorkspaceSandboxEnsurer`, then retry `Create` with the same `sandbox:run:{runID}` idempotency key up to `Sandbox.OperationMaxAttempts`. |
| `SANDBOX_OPERATION` / `operation=turn_checkpoint` | Post-turn checkpoint transport failure before a response. | Call `WorkspaceSandboxEnsurer` once, then retry within sandbox retry bounds. |
| `SANDBOX_OPERATION` / `operation=apply_at_run_end` | Final apply transport failure before a response. | Retry only provider-marked retryable failures; do not assume response-level apply idempotency. |
| `SANDBOX_OPERATION` / `operation=workspace_sandbox_ensure` | Lifecycle start command failed or workspace-sandbox did not become healthy before `Sandbox.EnsureStartTimeout`. | Surface the lifecycle/health cause in the run summary and inspect workspace-sandbox scenario logs. |

The user-facing summary includes the operation and actionable cause (for
example `connect: connection refused` or a health timeout). Long command
output stays in logs/events, not in `Run.ErrorMsg`.

## Fallback Reason taxonomy (2026-05-07, Phase 2)

Runner and model rejection signals are classified into a typed
`fallback.Reason` (see `internal/fallback/reason.go`) before they reach
the executor. Each Reason carries a deterministic `RecoveryAction` from
`fallback.Recovery(reason)`; the dispatch is exhaustively tested by
`TestReasonRecoveryActionExhaustive`.

| Reason | Trigger | RecoveryAction |
|---|---|---|
| `rate_limit` | HTTP 429, "rate limit exceeded", claude rate-limit envelope | `retry_backoff` |
| `auth_failure` | HTTP 401/403, "invalid api key", "unauthorized" | `escalate_operator` |
| `quota_exhausted` | "quota exceeded", "billing", HTTP 402 | `escalate_operator` |
| `model_deprecated` | "deprecated", "retired", "no longer available", "sunset" | `fallback_to_next` |
| `model_unknown` | "unknown model", "model not found", "invalid model" | `fallback_to_next` |
| `model_unavailable` | Generic runner-rejected-model with no stronger signal | `fallback_to_next` |
| `network_transient` | "connection reset/refused", "timed out", HTTP 502/503/504 | `retry_backoff` |
| `context_length_exceeded` | "context length", "max tokens" | `fallback_to_next` |
| `binary_missing` | "command not found", "no such file" | `fallback_to_next` |
| `probe_timeout` | Health-probe timeout (distinct from runtime timeout) | `retry_backoff` |
| `invalid_flag` | "unknown flag", "unrecognized option" | `escalate_operator` |
| `session_expired` | Codec-recognised "session/thread not found" without state-loss markers | `escalate_operator` |
| `session_state_lost` | Codex `record_rollout_items` / "failed to record rollout items" | `abort` |
| `unknown` | Classifier saw a failure but matched no specific pattern | `abort` |

Per-codec classification lives in `internal/adapters/runner/codecs/{claude,codex,opencode}.go`'s `Classify` method
(structured signals first), with `fallback.TextClassifier` as the
residual safety net. The legacy regex `runner.ClassifyModelError` and
`ModelErrorKind` 3-value enum were deleted in this phase; all callsites
now consume `*fallback.ClassifiedError`.

DOC: `internal/fallback/reason.go` (Reason constants), `internal/fallback/recovery.go` (action map).
