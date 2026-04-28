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
closes that gap. See
`docs/plans/claude-code-runner-execution-error-diagnostics-plan.md` for
design rationale.

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
