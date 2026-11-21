# Where effort evidence lives and how to read it

`scripts/effort_anatomy.py` reads all of these. Use this file when the script cannot answer a
question, or when a format changes. Update it when you verify a new fact.

## Codex

| Artifact | Location |
|---|---|
| Web Console session homes | `~/.vrooli/state/vrooli/web-console/sessions/codex/<home-uuid>/`. Each is a per-session `CODEX_HOME`. `config.toml`, `auth.json` and `skills` are symlinks to `~/.codex/`. |
| Rollouts (one per thread) | `<home>/sessions/YYYY/MM/DD/rollout-<local-time>-<thread-id>.jsonl`. Filenames use local time (UTC−5 here); timestamps inside are UTC. |
| Native goals | `<home>/goals_1.sqlite`, table `thread_goals(thread_id, objective, status, tokens_used, time_used_seconds, created_at_ms, updated_at_ms)`. Status values: `active`, `paused`, `blocked`, `usage_limited`, `budget_limited`, `complete`. Open read-only: `sqlite3 'file:<db>?mode=ro'`. |
| Terminal and `codex exec` sessions | `~/.codex/sessions/` with the same rollout format. |
| Agent Manager runs | `~/.vrooli/state/vrooli/agent-manager/runs/<run-id>/runtime/codex/sessions/` (older runs: `<run-id>/codex/sessions/`), same format; `find <run dir> -name 'rollout-*.jsonl'`. |
| Shared config | `~/.codex/config.toml` (applies to every Web Console session through the symlink). Vrooli's keys are declared in `resources/codex/native-settings.json` and applied by `resource-codex ensure`; `resource-codex settings plan` shows drift and user-owned conflicts. |

Rollout lines are `{"timestamp", "type", "payload"}` (children add `"ordinal"`).

| What | How it appears |
|---|---|
| Thread identity | First line `type: session_meta`: `id`, `cli_version`, `multi_agent_version`; children add `parent_thread_id`, `source.subagent.thread_spawn.{depth, agent_path, agent_role}` and `subagent_history_start_ordinal`. |
| Inherited history | In a `fork_turns=all` child, lines with `ordinal` below `subagent_history_start_ordinal` are the parent's conversation items (no tool events). Exclude them from the child's metrics. |
| Model and effort | `type: turn_context` or `thread_settings_applied` payload `model`, `effort`. |
| Goal objective injection | `response_item` user message containing `<codex_internal_context source="goal">` with `<objective>…</objective>`, plus the continuation and completion-audit text. Only at goal-turn starts. |
| Goal status changes | Three forms. The operator's goal control writes a `response_item` user message `<codex_internal_context source="user_goal"> User set goal status: "active"\|"paused"` and an `event_msg` `thread_goal_updated` with `goal.status`. The agent's own `create_goal` and `update_goal({status: "paused"\|"complete"\|"blocked"})` appear only as the `custom_tool_call` (`name: exec`, the call text in `input`) and write no goal event. The agent cannot set `active`. `drift` tracks all three for the `paused-goal` tripwire. |
| Waits on sub-agents | `response_item` `function_call` `wait_agent` with `timeout_ms` in `arguments`; the matching `function_call_output` says `timed_out` and, when the configured floor applied, "clamped to the minimum of N ms". |
| Turn boundaries | `event_msg` `task_started` (`turn_trigger: goal` for goal turns), `task_complete`, `turn_aborted` (`reason`). |
| Compaction | `event_msg` `item_completed` with `item.type: ContextCompaction`, and a `type: compacted` line whose `replacement_history` is the new context: developer messages, AGENTS.md, an encrypted `compaction` summary and, in a child, its parent's `agent_message` items (encrypted payload). A long goal shows there as `User set the goal: [objective omitted; exceeds the evidence limit]`. |
| What survived a compaction | The `compacted` payload's `retained_context`: `user_messages[]` and `assistant_messages[]` (each `text`, `complete`, `turn_id`, `order`). `complete: false` marks a message cut to its first and last ~1,800 characters around `<truncated omitted_approx_tokens="N" />`. Goal events (`<codex_internal_context source="user_goal">`) and Agent Manager wakes (`[wake] …`) are user messages too. A child always shows `user_messages: []`. To check that the stance survived, find the stance message among the retained user messages of the root thread. |
| Context use | `event_msg` `token_count`: `info.last_token_usage.input_tokens`, `info.model_context_window` (258,400 for the GPT-6 family). |
| File edits | `event_msg` `item_completed` with `item.type: FileChange`, `item.changes{path: {type, unified_diff}}`. Older rollouts (2025, Codex before multi-agent v2) have no FileChange items; edits appear as `apply_patch` text (`*** Begin Patch`) inside `shell` function calls or `apply_patch` custom tool calls, and the analyzer falls back to counting that text (`lines_source: apply_patch`). Shell-based writes (`sed -i`, heredocs, scripts) appear in neither; count them separately. |
| Narration | `event_msg` `item_completed` with `item.type: AgentMessage`, `item.content[].text` (plaintext). Reasoning is encrypted. |
| Sub-agent calls | `response_item` `function_call` with `name` `spawn_agent`, `followup_task`, `send_message`, `wait_agent`; `arguments` is a JSON string. The `message` payload is encrypted. Read the child's rollout to see what it did. |
| Messages between agents | `response_item` `agent_message` with `author`, `recipient` and `Message Type: NEW_TASK | MESSAGE | FINAL_ANSWER`. |
| Child birth instructions | Developer messages: "You are now a newly spawned child agent…", `<multi_agent_role>` (slots, tools) and `<multi_agent_mode>` ("Do not spawn sub-agents unless…"). |

### Multi-agent configuration (`features.multi_agent_v2` in `config.toml`, Codex 0.166)

Fields found in the binary: `max_concurrent_threads_per_session`, `min_wait_timeout_ms`,
`max_wait_timeout_ms`, `default_wait_timeout_ms`, `usage_hint_enabled`, `usage_hint_text`,
`root_agent_usage_hint_text`, `subagent_usage_hint_text`, `subagent_developer_instructions`,
`preserve_fork_prefix`, `multi_agent_mode_hint_text`, `hide_spawn_agent_metadata`,
`expose_spawn_agent_model_overrides`, `wait_agent_enabled`. `[agents]` also has `max_depth`.
The verified effect of each field and the probe results are in
`docs/agent-system/LONG_EFFORT_DESIGN.md` §4; the values are in `resources/codex/native-settings.json`.
`codex features list` shows `multi_agent_v2 … false` even when sessions run v2; read
`multi_agent_version` in `session_meta` instead. Do not assume an unverified field works. To
probe a field, run a short `codex exec` chain with `-c features.multi_agent_v2.<field>=<value>`
and read the resulting rollouts: the root and every child record the `<multi_agent_mode>`
text and developer messages they received.

## Claude Code

| Artifact | Location |
|---|---|
| Session transcript | `~/.claude/projects/<project-slug>/<session-id>.jsonl` |
| Subagent transcripts | `~/.claude/projects/<project-slug>/<session-id>/subagents/agent-<id>.jsonl`, with `agent-<id>.meta.json` (`agentType`, `description`, `spawnDepth`) |

Lines carry `type` (`user`, `assistant`, `system`, …), `timestamp`, `message`. A compaction is
`type: system, subtype: compact_boundary` with `compactMetadata.preTokens`; the summary that
follows has `isCompactSummary: true`. When `resource-claude-code ensure` has run, the
`SessionStart` `compact` hook's re-read reminder appears right after it; its absence after a
compaction means the hook is not installed. Edits are `tool_use` items named `Edit` (`old_string`,
`new_string`) and `Write` (`content`); sub-agents are `Agent` tool uses.

## Effort records outside transcripts

| Effort kind | Records |
|---|---|
| Ecosystem hunt | `vrooli-memory recall wake --scope ecosystem-hunt`; round records by hunt ID |
| Research cohort | `~/.vrooli/research/cohorts/<cohort>/README.md` and `CHECKPOINT.md`; topic runs under `~/.vrooli/research/topics/` |
| Delivery effort | `.vrooli/work/delivery/<effort>/` (`LOG`, `FEEDBACK.md`, `epochs/`) |
| Agent Manager runs | `agent-manager run …`; friction measures through `agent-manager.friction-digest` |
| Friction queue | Agent Manager tool-friction findings |
