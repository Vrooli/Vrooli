# Claude-Code Runner `execution_error` Diagnostics — Implementation Plan

## 1. Purpose

Close the diagnostics gap exposed by run `66cbfd0a-3467-4b63-9a50-58ad348665b1` (swarm-manager, 2026-04-20), where the claude-code CLI emitted a final `result` event with `is_error: true` but an empty `result` message, causing the runner to persist a bare `{"code":"execution_error"}` event with no cause, subtype, stderr, or turn/token counters. The plan also addresses two contributing environment frictions that the same run surfaced (Bash cwd-reset swallowing build output; unreported multi-minute stream stalls).

Outcome: every future `execution_error` event is **self-diagnosing** (carries `subtype`, `num_turns`, `duration_ms`, captured stderr tail, and the rate-limit flag that was already parsed), plus the agent is steered away from the `cd X && cmd` pattern that reliably loses output under the Claude Code persistent Bash tool.

## 2. Required Reading (run before implementing)

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Additional run-specific evidence:

```bash
agent-manager run events 66cbfd0a-3467-4b63-9a50-58ad348665b1 -json | \
  python3 -c "import json,sys; d=json.load(sys.stdin); [print(e['sequence'], e['event_type']) for e in d['events']]"
agent-manager run get 66cbfd0a-3467-4b63-9a50-58ad348665b1
```

## 3. Problem Statement

In run 66cbfd0a the final persisted error event was:

```json
{"event_type":"RUN_EVENT_TYPE_ERROR","error":{"code":"execution_error"}}
```

followed by a status transition with reason `"Claude Code execution completed"`. No `message`, no `subtype`, no stderr, no turn/token counters. Root-cause classification (max_turns vs context_exhausted vs API error vs CLI crash) was impossible from telemetry alone.

Three concrete defects in `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go`:

| # | Defect | Location | Observed consequence in run 66cbfd0a |
|---|---|---|---|
| D1 | `parseResultEvent` emits the error using only `resultStr` (which was empty) and discards `event.Subtype` plus every numeric counter on the event | lines 1440–1467 | Empty-string error message; no way to distinguish `error_max_turns` from `error_during_execution` |
| D2 | stderr from the claude-code child process is buffered into `errorOutput` but only surfaced when `mp.Wait()` returns an OS-level error. When the CLI self-reports `is_error: true` with exit code 0, stderr is silently dropped | lines 205, 209–215, 295–308 | No stderr trail to inspect for CLI-internal diagnostics |
| D3 | No idle-stream heartbeat; a 1m47s silence between events #120 (02:55:45) and #121 (02:57:32) left the run looking frozen but produced no log. Runner's compaction detection (`isCompactionSummary`, line 1040) only matches `"<summary>"` / `"Summary of"` prefixes and missed whatever the CLI was doing during that gap | lines 224–264 (scanner loop), 1040 | Gap in observability that masks slow-API or internal-compaction behavior |

Separately, agent-side (`scenarios/swarm-manager/api/internal/execution/prompt_builder.go`):

| # | Defect | Location | Observed consequence |
|---|---|---|---|
| D4 | Execution prompt contains no steer away from `cd X && cmd` invocations under Claude Code's persistent Bash tool. When the command produces empty stdout, Claude Code replaces the output with `Shell cwd was reset to <origin>` and the agent cannot distinguish success-with-no-output from failure | `buildExecutionPrompt`, lines 48–100ish | Events 126/128/130/134 all return only the cwd-reset warning; agent retried 5 times before switching to `bash -c '…'` wrapper |

## 4. Scope

**In scope**
- Enrich the `execution_error` event emitted by `parseResultEvent` with `subtype`, `duration_ms`, `num_turns`, session id, and captured stderr tail
- Ensure stderr buffered during the run is attached to the error when the CLI self-reports `is_error: true`
- Emit an idle-stream heartbeat log event after a configurable idle threshold during stream parsing
- Broaden `isCompactionSummary` (or add a sibling detector) to recognize claude-code's auto-compact indicators so compaction events get emitted instead of silent system-context bursts
- Add a short "Bash command patterns" stanza to the swarm-manager execution prompt that instructs agents to avoid `cd X && cmd` in favor of `bash -c 'cd X && cmd; echo "exit=$?"'`
- Unit tests for every new code path; golden-sample tests for the enriched error

**Out of scope**
- Changing the claude-code CLI's own behavior or suppressing the cwd-reset warning at source (upstream)
- Reworking `ErrorEventData` — the existing `Details map[string]interface{}` field already accommodates the new data (see §8)
- Re-engineering the runner's cancellation, timeout, or retry strategy
- Adding prompt content beyond the Bash-pattern guidance (no broader prompt rewrite)
- Populating the agent profile description with prompt-location breadcrumbs (tracked separately as finding A1 — deferred)

## 5. Current Technical Context

Key files and their roles:

| Path | Role | Relevant lines |
|---|---|---|
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code.go` | Claude-Code runner adapter; parses stream-json, emits domain events | 140–349 (Execute loop), 676–720 (buildArgs), 738–810 (stream types), 1225–1500 (parseStreamEvent / parseResultEvent) |
| `scenarios/agent-manager/api/internal/adapters/runner/claude_code_stream_test.go` | Sample-based tests for stream parsing | 1–80 (sample fixtures incl. `result_success`, `result_error` at line 39/42) |
| `scenarios/agent-manager/api/internal/domain/types.go` | `RunEvent`, `ErrorEventData{Code,Message,Details,…}`, `NewErrorEvent` | 893–904 (type), 1030–1038 (constructor) |
| `scenarios/swarm-manager/api/internal/execution/prompt_builder.go` | Builds the execution prompt used by swarm-manager task runs | 26–44 (title helper), 48–230 (`buildExecutionPrompt`) |
| `scenarios/swarm-manager/api/internal/execution/service_test.go` | Prompt builder golden tests | (verify presence of prompt-text assertions before editing) |

Existing helpers we can reuse:
- `ClaudeStreamEvent.Subtype` string at line 742 — already parsed, just dropped on the floor
- `event.DurationMs`, `event.NumTurns`, `event.SessionID` — all already parsed
- `ErrorEventData.Details map[string]interface{}` — already exists; no schema change needed
- `domain.NewLogEvent(runID, level, message)` — already used for debug logs; fine for heartbeat

Verification commands:

```bash
# Confirm parseResultEvent current shape
sed -n '1440,1500p' scenarios/agent-manager/api/internal/adapters/runner/claude_code.go

# Confirm Subtype is already decoded
grep -n 'Subtype' scenarios/agent-manager/api/internal/adapters/runner/claude_code.go

# Confirm ErrorEventData has Details
grep -n 'type ErrorEventData' scenarios/agent-manager/api/internal/domain/types.go
```

## 6. Target End State

When the claude-code CLI emits `{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":60,"duration_ms":450000,"session_id":"…"}` with an empty `result` field, the persisted error event reads:

```json
{
  "event_type": "RUN_EVENT_TYPE_ERROR",
  "error": {
    "code": "execution_error",
    "message": "claude-code terminated with is_error=true (subtype=error_max_turns, turns=60, duration_ms=450000)",
    "retryable": false,
    "details": {
      "subtype": "error_max_turns",
      "num_turns": 60,
      "duration_ms": 450000,
      "session_id": "…",
      "stderr_tail": "<last ~2 KB of stderr, newline-split>",
      "result_text": ""
    }
  }
}
```

Simultaneously:
- The run's event stream contains a `debug` log `stream idle for 30s (last event: <seq/type>)` if any gap ≥ 30s occurs, instead of silent dead air
- swarm-manager execution prompts contain a short Bash-pattern stanza (≤ 6 lines) steering agents to wrap `cd` invocations in `bash -c` with explicit exit-code echo
- All existing tests still pass, and new tests cover the enriched error path and heartbeat emission

## 7. Implementation Strategy (phased)

Phases are ordered so the highest-leverage, lowest-risk change ships first and observability improves even if later phases slip.

### Phase 1 — Enrich `parseResultEvent` error payload (D1 + D2)

**Dependencies:** none. Self-contained change inside `claude_code.go`.

1. Change `parseResultEvent`'s signature internally (or pass additional context) so it can read the buffered stderr and a small struct of stream metadata. Two viable approaches — pick one; recommend (b) for smaller blast radius:
   - **(a)** Refactor so `parseResultEvent` lives on a parse-state struct that already has access to stderr buffer and session id.
   - **(b)** Leave `parseResultEvent` pure: have `Execute` detect `event.IsError` + `state.gotResult` after the scanner loop and augment the error-event details with `errorOutput.String()` before `EventSink.Emit`. Prefer this — it keeps the stream-parsing layer deterministic and testable.
2. Inside `parseResultEvent` (claude_code.go:1458–1467), build the enriched error:
   - `message`: `fmt.Sprintf("claude-code terminated with is_error=true (subtype=%s, turns=%d, duration_ms=%d)", subtype, numTurns, durationMs)` when `resultStr` is empty; otherwise prepend the formatted prefix and include `resultStr` after a `: ` separator.
   - `details`: populate `{"subtype","num_turns","duration_ms","session_id","result_text"}` from the stream event fields that are already decoded.
   - Keep `retryable: false` — the CLI already decided to fail; retries belong upstream.
3. In `Execute` (claude_code.go:224–349), after the scanner loop: if `state.resultIsError` is true, attach a trimmed stderr tail (last 2 KB, UTF-8-safe truncation) into the emitted error event's `Details["stderr_tail"]`. If `EventSink` has already emitted the error event at this point, emit a *follow-up* log event of level `warn` with the stderr tail instead of mutating the prior event (domain events are immutable).
4. Preserve existing rate-limit handling — the `detectRateLimit` branch at lines 1447–1456 stays untouched and still short-circuits.

### Phase 2 — Idle-stream heartbeat (D3)

**Dependencies:** Phase 1 merged (keeps the diff reviewable and independently revertable).

1. In the scanner loop (claude_code.go:224), track `lastEventAt time.Time` updated on every emitted event.
2. Run a background goroutine with `time.Ticker(15 * time.Second)` (threshold configurable via `StreamIdleHeartbeatInterval` constant) that, while the loop is active, emits a `debug` log event `fmt.Sprintf("stream idle for %s (last event: type=%s seq=%d)", elapsed, lastType, lastSeq)` any time `time.Since(lastEventAt) ≥ 30s`.
3. Coordinate with existing `DefaultStreamIdleTimeout` — heartbeat threshold must be strictly less than idle-timeout so we log before we kill.
4. Broaden `isCompactionSummary` (claude_code.go:1040) or add a sibling `isAutoCompactMarker` checking for patterns like `"Auto-compacting conversation"` / `"conversation history has been compacted"` / consecutive `system` events within a 2s window. Emit a `CompactionEvent` (trigger=`"auto"`, summary=`""` if unknown) when detected, so investigators see compaction rather than dead air.

### Phase 3 — swarm-manager Bash-pattern prompt stanza (D4)

**Dependencies:** none (can run in parallel with Phase 1/2; sequence after so the agent benefits from both diagnostic streams simultaneously).

1. In `scenarios/swarm-manager/api/internal/execution/prompt_builder.go`, in `buildExecutionPrompt`, add a new XML stanza just before the closing section (after the current `</execution-context>`) **only when** `p.RunType` is one of `process`, `fixup`, `followup`, `custom` (i.e., always for execution runs).
2. Stanza content (verbatim, keep ≤ 6 non-blank lines):
   ```
   <bash-patterns>
   When running commands that change directory, avoid `cd X && cmd` — Claude
   Code's persistent Bash tool replaces empty output with a cwd-reset warning.
   Prefer: bash -c 'cd X && cmd; echo "exit=$?"'
   or pass absolute paths to the command directly (e.g. `go build ./path/...`).
   </bash-patterns>
   ```
3. Update the golden prompt test in `scenarios/swarm-manager/api/internal/execution/service_test.go` (if one exists for the prompt content) to assert the stanza is present.

### Phase 4 — Documentation

**Dependencies:** Phases 1–3 merged.

1. Add a 1-paragraph note in `scenarios/agent-manager/docs/ARCHITECTURE.md` (or `docs/internal/ERROR-SEMANTICS.md` if one exists) pointing investigators to the enriched `execution_error` `Details` schema.
2. Cross-link this plan from `scenarios/agent-manager/docs/PROBLEMS.md` so the root-cause-visibility gap is tracked as resolved.

## 8. Contract Decisions

**`ErrorEventData.Details` schema for `execution_error` events** (new, additive — no domain-type change):

| Key | Type | Required | Meaning |
|---|---|---|---|
| `subtype` | string | yes (empty if CLI omitted) | e.g. `error_max_turns`, `error_during_execution`, `error` (legacy), `` |
| `num_turns` | int | yes (0 if unknown) | CLI-reported conversation turns at failure |
| `duration_ms` | int | yes (0 if unknown) | CLI-reported wall time |
| `session_id` | string | optional | CLI session id for correlation with `--resume` |
| `result_text` | string | yes (may be empty) | Raw `result` field from the stream event, whatever the CLI sent |
| `stderr_tail` | string | optional | Up to final 2048 bytes of captured stderr, trimmed at UTF-8 boundary |

**Error `message` format:** `claude-code terminated with is_error=true (subtype=%s, turns=%d, duration_ms=%d)[: %s]` — the trailing `: %s` is the `resultStr` if non-empty. This gives humans a readable summary while `Details` carries structured data for telemetry.

**Heartbeat event shape:** existing `domain.NewLogEvent(runID, "debug", message)`. No new event type. Message format: `stream idle for 30s (last event: type=<type> seq=<seq>)`.

**Bash-patterns prompt stanza:** uses the existing XML-tag convention in `buildExecutionPrompt`. New tag: `<bash-patterns>`. Non-empty stanza is always emitted for execution runs; adds ≤ 6 lines to the prompt, well under any token budget.

**Rate-limit branch:** unchanged. If `detectRateLimit` matches, we still emit a `RateLimitEvent` and never hit the enrichment code path.

## 9. Testing Plan

### 9.1 Unit tests (Phase 1)

File: `scenarios/agent-manager/api/internal/adapters/runner/claude_code_stream_test.go`

Add new fixtures adjacent to `result_error` at line 42:

```go
"result_error_max_turns": `{"type":"result","subtype":"error_max_turns","is_error":true,"duration_ms":450000,"num_turns":60,"session_id":"s1","result":"","total_cost_usd":0}`,
"result_error_empty":     `{"type":"result","subtype":"error","is_error":true,"duration_ms":100,"num_turns":0,"session_id":"s2","result":"","total_cost_usd":0}`,
"result_error_during":    `{"type":"result","subtype":"error_during_execution","is_error":true,"duration_ms":5000,"num_turns":12,"session_id":"s3","result":"internal API error","total_cost_usd":0}`,
```

New tests:

- `TestClaudeCodeRunner_ParseResultEvent_ErrorMaxTurns` — decodes `result_error_max_turns`, asserts event is `EventTypeError`, `Code="execution_error"`, `Message` starts with `"claude-code terminated with is_error=true (subtype=error_max_turns, turns=60, duration_ms=450000)"`, and `Details["subtype"]=="error_max_turns"`, `Details["num_turns"]==60`, `Details["duration_ms"]==450000`, `Details["session_id"]=="s1"`.
- `TestClaudeCodeRunner_ParseResultEvent_ErrorEmpty` — regression guard for the run-66cbfd0a scenario: empty result + empty subtype still produces a useful message (not empty string) and populated `Details`.
- `TestClaudeCodeRunner_ParseResultEvent_ErrorDuringExecution_CarriesResultText` — confirms `result_text` is populated with the original payload when non-empty.
- `TestClaudeCodeRunner_ParseResultEvent_RateLimitPrecedence` — feed a result event where `result` matches the rate-limit regex; assert it still becomes a `RateLimitEvent`, not an enriched `execution_error`.
- `TestExecute_StderrTailAttachedOnCLIError` (integration-style, using a fake `mp`/stderr): write a stderr chunk > 2 KB, verify only trailing 2048 bytes appear in `Details["stderr_tail"]`, cut at UTF-8 boundary.

### 9.2 Unit tests (Phase 2)

- `TestExecute_EmitsIdleHeartbeat` — inject a slow stdout (no events for 35s using a fake clock or `time.Sleep` in a short-threshold variant), assert at least one `stream idle for` log event was emitted via the sink.
- `TestIsAutoCompactMarker_DetectsKnownStrings` — table-driven over auto-compact marker strings; assert compaction event is emitted with trigger `"auto"`.

### 9.3 Unit tests (Phase 3)

- `TestBuildExecutionPrompt_ContainsBashPatternsStanza` — for each `RunType` in `{process, fixup, followup, custom}`, assert the generated prompt contains `<bash-patterns>` and the `bash -c 'cd X && cmd; echo "exit=$?"'` line.
- Regression: existing golden-prompt tests must continue to pass (stanza is additive; check for stable-order insertion).

### 9.4 Manual validation

```bash
# Phase 1/2 — agent-manager
cd scenarios/agent-manager/api && go build ./... && go test ./internal/adapters/runner/...

# Phase 3 — swarm-manager
cd scenarios/swarm-manager/api && go build ./... && go test ./internal/execution/...

# Full monorepo smoke
cd /home/matthalloran8/Vrooli && make -C scenarios/agent-manager build
```

Then: re-trigger a swarm-manager execute run and force a failure scenario (e.g., set `max_turns: 2` on a non-trivial task) to confirm the enriched error event shape end-to-end via `agent-manager run events <id> -json | jq '.events[] | select(.event_type=="RUN_EVENT_TYPE_ERROR")'`.

## 10. Rollout / Validation Checklist

- [ ] Phase 1 PR merged; CI green; manual forced-failure shows enriched `Details`
- [ ] Phase 2 PR merged; CI green; manual long-idle run shows ≥ 1 heartbeat event in the log stream
- [ ] Phase 3 PR merged; CI green; a fresh swarm-manager execute run's prompt (visible in run message #0 or via `agent-manager run events`) contains `<bash-patterns>`
- [ ] Run `agent-manager run investigate <any-failed-run>` against a post-merge failed run; confirm the investigator can classify the failure subtype without reading code
- [ ] Phase 4 docs updated; link from `PROBLEMS.md` to this plan file
- [ ] Spot-check 3 recent swarm-manager runs to confirm no prompt-size / token-count regression from Phase 3 stanza

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Enriched `Details` breaks a downstream consumer expecting only `{code,message}` | Low | Medium | `Details` is already on `ErrorEventData`; consumers use JSON-tag-optional decoding. Add a migration note in the PR description; grep for consumers before merging (`grep -rn 'ErrorEventData' scenarios packages`) |
| Heartbeat goroutine leaks if scanner loop exits before the ticker goroutine is stopped | Medium | Low | Use a `context.WithCancel` or `done chan struct{}` pattern; cancel on scanner exit. Covered by `TestExecute_EmitsIdleHeartbeat` cleanup assertion |
| stderr-tail inclusion leaks sensitive tokens into events | Low | High | claude-code stderr is already considered non-sensitive in current deployment; add a simple regex redactor for `Bearer \S+` / `sk-[A-Za-z0-9-_]+` patterns before attaching, as defense-in-depth |
| Prompt-stanza insertion confuses existing prompts that depend on exact section ordering | Low | Medium | Insert at a documented anchor (immediately after `</execution-context>`); update golden tests in same PR |
| False-positive auto-compact detection spams `CompactionEvent`s | Medium | Low | Detector should require ≥2 consecutive `system` events within 2s AND a subsequent assistant message to fire; rate-limit one event per minute |
| Heartbeat threshold (30s) is too chatty for short-timeout test runs | Medium | Low | Make the threshold a runner constant; tests inject a shorter value |

## 12. Non-Goals / Prohibited Patterns

- **Not** reworking `RunnerError` or introducing a new domain event type — `Details` on `ErrorEventData` already covers everything this plan needs
- **Not** retrying the run when an `execution_error` now carries `subtype=error_max_turns` — retry policy is a separate concern; this plan is diagnostics-only
- **Not** adding rich context-usage estimates or token-budget warnings to the prompt — out of scope; handled elsewhere
- **Not** editing claude-code CLI behavior or filing an upstream issue as part of this work
- **Not** changing persisted event schemas in a backward-incompatible way; every new field is additive inside `Details`

## 13. Definition of Done

All must be true:

1. `parseResultEvent` emits `execution_error` events whose `Details` map populates `subtype`, `num_turns`, `duration_ms`, `session_id`, `result_text`, and (when present) `stderr_tail` — verified by the three new fixture tests
2. When `is_error=true` with empty `result`, the persisted `Message` is non-empty and parsable by a human (`"claude-code terminated with is_error=true (subtype=..., turns=..., duration_ms=...)"`)
3. A stream idle period of ≥ 30s produces at least one `debug` log event with message prefix `"stream idle for"`; goroutine exits cleanly on scanner EOF (no leak, asserted in test)
4. swarm-manager execution prompts for `process | fixup | followup | custom` run types contain the `<bash-patterns>` stanza; golden test asserts presence
5. All new unit tests pass locally and in CI; no existing test regresses
6. `scenarios/agent-manager/docs/PROBLEMS.md` links to this plan and marks finding E1/E2/E3 resolved; A1 (prompt-location breadcrumb) remains open but tracked
7. A re-investigated failure run (`agent-manager run investigate`) classifies the failure subtype without the investigator having to read runner source
