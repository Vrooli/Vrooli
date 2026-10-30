# Captured runner trace fixtures

The JSONL files in this directory are sanitized captures of provider-native
CLI output. They preserve event type, message/turn identity, parent/origin
fields, completion reason, terminal markers, token/cost shape, and ordering
needed to test codec normalization and final-output selection. Prompts,
workspace paths, credentials, user content, and installation identifiers are
replaced with inert fixture values.

| Runner | Execute evidence | Continue/recovery evidence | Terminal evidence |
|---|---|---|---|
| Claude Code | `claude_trace.jsonl` | `claude_ondisk_trace.jsonl` exercises on-disk replay using the same provider message shape | `result/subtype=success`, assistant message id, `stop_reason=end_turn` when present |
| Codex | `codex_trace.jsonl` | `codex_rollout_trace.jsonl` exercises persisted rollout replay | `turn.completed` or rollout `task_complete` with `turn_id`; `agent_message` item id identifies the candidate |
| OpenCode | `opencode_trace.jsonl` | The same line-oriented format is replayable; continuation retains `sessionID` | `step_finish` reason/text and session id |
| Grok | `grok_trace.jsonl` | `grok_resume_trace.jsonl` is an observed resumed turn | `end` with stop reason, session id, and request id after accumulated text deltas |

`codex_goal_updated.jsonl` is a sanitized native Codex TUI goal-event capture;
the source rollout and capture date are recorded in its consuming test.

`final_output_cases.json` composes those observed field shapes into adversarial
selection cases. Composed cases are marked `composition`; they do not claim the
exact combined conversation was captured. This keeps provenance honest while
allowing deterministic tests for interleaved child output, post-handoff
chatter, missing terminal metadata, and equal-evidence ambiguity. Recovery
tests must feed the same raw lines through transcript parsers and expect the
same selection as live decode.

Fixture maintenance rules:

1. Never copy a live transcript wholesale. Retain only fields required by a
   codec or selection invariant and replace content with inert text.
2. Record whether a fixture is an observed trace or a composition of observed
   event shapes.
3. A provider field that is absent stays absent; tests may not fabricate
   support merely to make a resolver decisive.
4. Every new resolver rule needs at least one selecting case and one
   ambiguity/abstention case.
5. Run `go test ./internal/adapters/runner/codecs/...` before committing a
   transcript fixture. The committed-fixture gate rejects credentials,
   passwords, tokens, API keys, and absolute home paths. Use
   `go run ./cmd/harvest-replay` for sourced corpus candidates, then inspect
   the redacted result before commit.

`codex_tool_returns_e30.jsonl` replays the fourteen paired custom returns from
Nooch E30 run `650c0416-6afd-416a-be82-5b1838ecf90d`, observed October 10,
2026 06:10:52 UTC. Original rollout SHA-256:
`cb7e3273eb0d056b8909f0f82e11426c4bd57e53960fc64d8cc6280b9ed7acc2`.
Retained semantic receipt SHA-256:
`4f0ef92d8886e50e55338c20d17a59c14f51bd4d669caa8a171318aa5aee884e`.
This sanitized fixture preserves timestamps, call identities, block order/type,
first-block completion headers, absent exit metadata and the observed
`Unknown command: read`. Invocation and remaining output text are redacted;
quota/private content is excluded. It proves the incident return shape, not
byte-identical history or command success.

`codex_command_executions.jsonl` is an observed web-console rollout trace
(session `01a182ed-c43e-7be0-9406-10c15448b4a7`, 2026-10-28, used by
`codex_command_execution_test.go`): three `exec` calls with their
`event_msg` `item_completed` `CommandExecution` items, outputs and usage
frames. It keeps a parallel `Promise.allSettled` call with one failed
command, two commands that complete after their exec call returned, and a
script that builds its command list at run time. Home paths are rewritten to
`/workspace`; command output keeps its first line and size, padded with dots;
exec outputs keep their completion header and size; reasoning, quota and
client metadata are dropped.

Usage-accounting fixtures (observed traces, trimmed and redacted on
2026-10-29; used by `usage_accounting_test.go` and the orchestration
usage regressions). Only usage-bearing records and their turn boundaries are
kept; message, tool and prompt content is replaced with `[redacted]`, quota
metadata is dropped, and token counters are unchanged:

- `codex_usage_rollout.jsonl`: the first six turns of the run-scoped rollout
  of codec-pipe run `aee95b01-f8d3-4de7-8c8c-1c6b19a36f2a` (session
  `01a18449-34cc-7291-ad6f-10ff3e83ad11`), including one repeated
  `token_count` frame and one compaction. Source SHA-256
  `cd4ba276ccd7b3bbe207109b18aae9a7d9a73b45cacb1f2b73d80beb8167c55c`.
- `codex_usage_exec.jsonl`: the `thread.started`/`turn.started`/
  `turn.completed` records of the same run's exec stream through its sixth
  `turn.completed`. The first invocations stop before reporting. Source SHA-256
  `0adb69c1e7ac301fc4499459efca05aaefb8427cebc6a6998cc3e95d8d14c21a`.
- `claude_usage_ondisk.jsonl`: the first 40 assistant messages of an on-disk
  Claude Code session transcript, with repeated per-block usage. Source SHA-256
  `d7992553f5486f64fb25de71b1e997f3b65598e5b8ed01c8be7317a542c0d54f`.
- `claude_usage_subagent.jsonl`: a complete Claude Code sub-agent
  (`isSidechain`) transcript whose early content blocks carry partial output
  usage. Source SHA-256
  `df6b51acc298a6e59d1140eb30c50c9136b2535d1b52fe5c0d874dd65c237a75`.
