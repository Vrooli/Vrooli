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
