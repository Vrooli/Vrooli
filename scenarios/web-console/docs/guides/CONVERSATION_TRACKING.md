# Conversation Tracking

Web-console parses `claude`, `codex`, `opencode`, and `grok` sessions into a structured message feed that lives alongside the raw terminal, so you can search, re-play, TTS, and cursor-track conversations without losing any of the terminal fidelity. This guide covers how the feature decides which session a given agent invocation belongs to, and where that attribution breaks down.

## What gets tracked

Four independent paths feed the conversation store. All of them converge on the same `AppendUser` / `AppendAssistant` seam and carry a stable `source` string (`claude_hook`, `codex_tailer`, `grok_tailer`, `opencode_api`) so the UI can label each message by runtime.

- **Codex rollouts** (`source=codex_tailer`) — the attributed launcher materializes a private `CODEX_HOME` only when Codex starts. When `codex` runs inside that session it writes rollout JSONL files under `$CODEX_HOME/sessions/YYYY/MM/DD/`. `CodexTailer` (`api/codex_tailer.go`) polls every 2s across all live sessions, tails any new rollout, and routes `output_text` items to the owning session's conversation store.
- **Claude Stop hooks** (`source=claude_hook`) — Claude Code fires a Stop hook after each assistant response. Registration points at the portable `web-console hooks dispatch` CLI command, which reads the JSON payload from stdin and posts it with the session token; no shell, jq, or curl script is required. A two-second Claude transcript tailer is the defense-in-depth path: it tails the session's `~/.claude/projects/<cwd-key>/<agent-session-id>.jsonl` transcript and emits user/assistant text through the same append seam. Source-independent role/text deduplication prevents duplicate events when both paths arrive.
- **Grok transcripts** (`source=grok_tailer`) — the attributed launcher materializes a private `GROK_HOME` only when Grok starts. When `grok` runs inside that session it writes an ACP update stream to `$GROK_HOME/sessions/<url-encoded-cwd>/<session-id>/updates.jsonl`. `GrokTailer` (`api/grok_tailer.go`) polls every 2s, tails each transcript, accumulates `user_message_chunk` / `agent_message_chunk` text per turn, and emits one user + one assistant event at each `turn_completed` boundary. Thought chunks and tool-call lifecycle are parsed but **not** appended to the conversation (natural-language only).
- **OpenCode API** (`source=opencode_api`) — web-console owns a single loopback `opencode serve` instance. `OpenCodeWatcher` (`api/opencode_watcher.go`) subscribes to its `GET /event` SSE stream and, on any session-touching event, reconciles the affected session through the full-history `GET /session/{id}/message` endpoint. OpenCode shares a global storage dir, so the managed server observes sessions started by any `opencode` process; attribution binds a session to a pane by directory + creation time (see below).

Attribution is always path-, env-, or directory-bound — never inferred from PTY output text.

## Works from

- **Shortcut launch** — clicking `claude` / `codex` / `opencode` / `grok` in the Terminal Launcher.
- **Plain shell, then typing the agent later.** The attribution identity (`WC_WEB_CONSOLE_SESSION_ID`) and state root are injected on every PTY spawn. The shared `vrooli-agent-launcher` shim creates the private Codex/Grok home when the agent starts, so launching the CLI mid-session is equivalent to launching via shortcut without creating unused agent state for shell-only panes. Useful when you want to pass flags like `claude --continue`, `codex --model gpt-5`, `grok --model ...`, or `cd` into a project first. OpenCode is the exception: it has no per-pane env isolation (see attribution below).
- **Tmux panes inside a web-console-managed session.** `tmux new-session -e KEY=VAL` propagates the attribution vars into the tmux session env, so panes see the correct values even when the tmux server itself was created by a previous session. See `buildTmuxNewSessionArgs` in `api/pty_tmux.go`.

## Does not work from

These cases are correctly *ignored* — they won't produce events and won't bleed into another session's feed. If you're relying on tracking, avoid them or launch the CLI from a real web-console pane.

- **External terminals** (system terminal, SSH to this host, another shell opened outside web-console). No `WC_WEB_CONSOLE_SESSION_ID`; the claude hook payload arrives with no session id and lands as `conversation_target_missing` in the server log. Codex rollouts land in the default `~/.codex/` tree and are never tailed.
- **Pre-existing tmux servers not managed by web-console.** If you `tmux attach` from a web-console pane into a server that was started before the pane opened, panes in that server inherit the *original* server env, not the pane's. We intentionally don't mutate foreign tmux servers.
- **SSH-to-remote-host from inside a web-console pane.** SSH strips env by default.
- **`claude` invocations in a cwd without `.claude/settings.json` declaring the Stop hook.** No hook is registered, so nothing fires. Make sure the project uses the standard web-console hook installation (`wc::register_tts_hook` / `claude-code` resource reconcile). Per `docs/guides/TTS_TROUBLESHOOTING.md`, web-console deliberately does **not** install a global `~/.claude/settings.json` hook — that would disturb Claude's shared login state.

## How attribution stays scoped

- **Codex** — each session's `CODEX_HOME` is a unique path. Rollouts written by a codex process running in that session land in that path; the tailer iterates `server.sessions.List()` and only scans each session's own directory. Cross-session bleed is structurally impossible.
- **Claude** — the Stop hook carries `WC_WEB_CONSOLE_SESSION_ID` inherited from the shell's env. The server's `appendConversationEvent` rejects any payload whose session id doesn't match a live session (`conversation_target_missing`). Two open tabs with two distinct ids cannot mis-route to each other.
- **Grok** — like Codex, each session's `GROK_HOME` is a unique path (shared `~/.grok` auth/config is symlinked in so login keeps working; only the `sessions/` subtree is isolated). The tailer globs `<session GROK_HOME>/sessions/*/*/updates.jsonl` and attributes by construction. Cross-session bleed is structurally impossible.
- **OpenCode** — there is no per-pane isolation: `opencode` uses a global storage dir shared across processes. The watcher attributes a session to a pane only when it is **mutually unique** — the OpenCode session's `directory` matches a single live opencode-launched pane's cwd (and was created after that pane), and that pane matches a single session. If two opencode panes run in the same directory at once, the watcher *skips* both rather than guess. Once bound, the pane's `agent_session_id` is recorded so the binding survives restart.

## Debugging attribution silently failing

If conversation events are missing:

1. `make logs` — look for `conversation-event: ... code=conversation_target_missing`. If you see that, the hook fired but the payload lacked a valid session id.
2. Enable the opt-in shell-side warning by exporting `WC_HOOK_WARN_UNATTRIBUTED=1` in the shell that runs claude. The hook scripts will then write to stderr when `WC_WEB_CONSOLE_SESSION_ID` is unset (normally silent to avoid noise for project-wide hook installations).
3. Check that the project's `.claude/settings.json` contains the Stop hook. Reconcile via `wc::register_tts_hook` if not.
4. For codex, check that a rollout file exists under `<session CODEX_HOME>/sessions/YYYY/MM/DD/`. The CodexTailer won't create one; codex itself has to.
5. For grok, check that `<session GROK_HOME>/sessions/<encoded-cwd>/<id>/updates.jsonl` exists and that `make logs` shows `grok-tailer:` lines. Note that a turn only appends once `turn_completed` is written, so an in-progress turn shows nothing yet.
6. For opencode, check `make logs` for `opencode-watcher: managed server at http://127.0.0.1:<port>` (capture is disabled if the `opencode` binary is missing) and `opencode-watcher: attributed session ... -> pane ...`. If a session is never attributed, it is almost always the ambiguity guard: another opencode pane shares the same cwd. Run them in distinct directories.

For a recovered Claude pane, the Messages view shows a subtle warning only when it has emitted terminal output after recovery, two minutes have elapsed, and no new conversation event was recorded. This is a degraded-observability signal, not a claim that the agent stopped; check the transcript path and hook configuration before retrying anything.

## Security & privacy

- Conversation events carry user/assistant natural-language text only. Grok thought chunks and tool-call arguments, and OpenCode tool parts, are deliberately **not** appended.
- The managed `opencode serve` binds to `127.0.0.1` only and is stopped on web-console cleanup. No provider auth material (`~/.grok/auth.json`, OpenCode credentials) is read or stored by web-console.

## See also

- `docs/guides/TTS_TROUBLESHOOTING.md` — full TTS pipeline troubleshooting
- `docs/internal/SEAMS.md` — hook delivery chain and dual-trigger architecture
- `api/mid_session_conversation_test.go` — regression tests that lock in the attribution invariants described here
