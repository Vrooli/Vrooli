# Conversation Tracking

Web-console parses `claude` and `codex` sessions into a structured message feed that lives alongside the raw terminal, so you can search, re-play, TTS, and cursor-track conversations without losing any of the terminal fidelity. This guide covers how the feature decides which session a given claude/codex invocation belongs to, and where that attribution breaks down.

## What gets tracked

Two independent paths feed the conversation store:

- **Codex rollouts** — each session gets a private `CODEX_HOME` (`api/session.go:927-931`). When `codex` runs inside that session it writes rollout JSONL files under `$CODEX_HOME/sessions/YYYY/MM/DD/`. `CodexTailer` (`api/codex_tailer.go`) polls every 2s across all live sessions, tails any new rollout, and routes `output_text` items to the owning session's conversation store.
- **Claude Stop hooks** — Claude Code fires a Stop hook after each assistant response. The hook script (`lib/claude-stop-hook.sh`) reads `WC_WEB_CONSOLE_SESSION_ID` from its shell env, includes it in the POST payload, and the server (`api/tts_hook_handler.go`) routes the event to that session.

Attribution is always path-or-env-bound — never inferred from PTY output text.

## Works from

- **Shortcut launch** — clicking `claude` / `codex` in the Terminal Launcher.
- **Plain shell, then typing `claude` or `codex` later.** The three attribution env vars (`WC_WEB_CONSOLE_SESSION_ID`, `CODEX_HOME`, `WC_CODEX_SESSIONS_DIR`) are injected on every PTY spawn, so launching the CLI mid-session is equivalent to launching via shortcut. Useful when you want to pass flags like `claude --continue`, `codex --model gpt-5`, or `cd` into a project first.
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

## Debugging attribution silently failing

If conversation events are missing:

1. `make logs` — look for `conversation-event: ... code=conversation_target_missing`. If you see that, the hook fired but the payload lacked a valid session id.
2. Enable the opt-in shell-side warning by exporting `WC_HOOK_WARN_UNATTRIBUTED=1` in the shell that runs claude. The hook scripts will then write to stderr when `WC_WEB_CONSOLE_SESSION_ID` is unset (normally silent to avoid noise for project-wide hook installations).
3. Check that the project's `.claude/settings.json` contains the Stop hook. Reconcile via `wc::register_tts_hook` if not.
4. For codex, check that a rollout file exists under `<session CODEX_HOME>/sessions/YYYY/MM/DD/`. The CodexTailer won't create one; codex itself has to.

## See also

- `docs/guides/TTS_TROUBLESHOOTING.md` — full TTS pipeline troubleshooting
- `docs/internal/SEAMS.md` — hook delivery chain and dual-trigger architecture
- `api/mid_session_conversation_test.go` — regression tests that lock in the attribution invariants described here
