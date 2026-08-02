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
