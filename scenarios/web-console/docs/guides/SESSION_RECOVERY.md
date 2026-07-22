# Session Recovery

Web-console persists every `backend=persistent` pane behind a tmux session. The persisted **DB row** is what makes recovery possible — once it is gone, the on-disk codex/claude history has no DB pointer back to the pane it belonged to.

There are two recovery modes:

- **Normal restart recovery (automatic).** API restarts but the host and tmux server stay alive. `Recover()` re-attaches to surviving `wc-<id>` tmux sessions, status stays `live`, the user sees no interruption.
- **Crash recovery (one CLI call).** Host reboot, tmux scope kill, OOM, or manual `tmux -L wc kill-server`. The tmux session is gone but the DB row is preserved with `status='awaiting_recovery'`, agent identity (`agent_type`, `agent_session_id`, `cwd`, `last_rollout_path`), and `orphaned_at`. Run `web-console session recover <id>` to spawn a fresh persistent pane, copy the per-session `CODEX_HOME`, carry over the prior conversation history, and paste the resume command.

This guide covers the supported (CLI-driven) flow. The legacy manual procedure lives in the appendix for the case where the API itself is broken.

## Quick Triage

```bash
vrooli scenario status web-console
web-console session list                # live sessions
web-console session list-recoverable    # rows whose tmux session died
```

If `list-recoverable` is empty after a crash, either nothing was running on the persistent backend (check with `web-console session list` against a known-good run) or the API's startup `Recover()` log line says `awaiting_recovery=0`. The startup log lives at `~/.vrooli/logs/scenarios/web-console/vrooli.develop.web-console.start-api.log` and looks like:

```
recovery: recovered=2 awaiting_recovery=1 orphaned_tmux=0 (awaiting_recovery rows preserved for explicit recovery via /api/v1/sessions/recoverable)
```

`awaiting_recovery=N` is the count you can recover. If this prints `awaiting_recovery=0` after a crash you remember had agents in it, jump to the appendix.

## Recover Codex / Claude Panes

```bash
web-console session list-recoverable
# Recoverable sessions: 3
# Recoverable:
#   8a10aa94 | agent=codex | session=019dfdf4 | orphaned=2026-05-06T19:08Z
#   815af6c6 | agent=codex | session=019dfe84 | orphaned=2026-05-06T19:08Z
#   d458939a | agent=claude | session=019dfde5 | orphaned=2026-05-06T19:08Z

web-console session recover 8a10aa94-29d1-472d-a910-7a5548a7ca35
# Recovered 8a10aa94 -> 30563c58
# Agent: codex
# Pasted: codex --yolo resume 019dfdf4-a8d6-7e10-aee2-6b6bd6d495a6
# CODEX_HOME copied: true
```

What `recover` does, in order:

1. Validates the row is `awaiting_recovery` and has enough agent identity to reattach (codex requires nothing more; claude requires a non-empty `agent_session_id`).
2. Creates a fresh `backend=persistent` pane inheriting `shell`, `cols`, `rows`, and `policy` from the orphan.
3. For codex panes, `rsync -a $CODEX_HOME($old_id)/ $CODEX_HOME($new_id)/` copies the rollout history into the new pane's path.
4. Copies the orphan's conversation history (`conversation_sessions` cursor + all `conversation_events`) onto the new session id, preserving sequence numbers and per-event playback/consumption state so the **messages view is populated on reattach** instead of starting empty. Best-effort: a copy failure is logged but does not abort recovery.
5. Pastes the appropriate resume command into the new pane:
   - `codex --yolo resume <agent_session_id>` (or `--last` if id is empty)
   - `claude --resume <agent_session_id> --dangerously-skip-permissions`
6. Marks the orphan row `dismissed` with `recovered_into=<new_session_id>`.

Recovery is idempotent on `X-Idempotency-Key`. The on-disk `CODEX_HOME` of the orphan is preserved even after dismiss; only the DB row transitions to `dismissed`.

To drop an orphan you don't want to recover (UI noise reduction, on-disk state is kept):

```bash
web-console session dismiss 8a10aa94-29d1-472d-a910-7a5548a7ca35
```

## UI Surface

When the API has any `awaiting_recovery` rows, the workspace shows a compact amber summary above the tab strip. Select **View** to open a drawer containing each orphan's pane name, header color, group, agent, and working directory. **Reattach all** processes rows sequentially and continues after an individual failure; every row uses a stable `X-Idempotency-Key`, so an ambiguous network retry cannot create a duplicate replacement pane. Claude rows without a known `agent_session_id` render with `Reattach` disabled — `claude --resume` against the wrong project is unsafe and not auto-guessed.

Recovery moves the original workspace-pane record to the replacement session in the same SQLite transaction: name, header color, group, ordering, active state, theme, font size, and message-view capability are retained. The replacement inherits the original working directory as well.

## Storage Locations

| Data | Path |
|------|------|
| SQLite database | `~/.local/share/vrooli/web-console/web-console.db` |
| Per-pane `CODEX_HOME` | `~/.local/state/vrooli/web-console/sessions/codex/<web-console-session-id>/` |
| Per-pane Claude state, when present | `~/.local/state/vrooli/web-console/sessions/claude/<web-console-session-id>/` |
| Claude project history | `~/.claude/projects/<project-key>/*.jsonl` |
| API startup log | `~/.vrooli/logs/scenarios/web-console/vrooli.develop.web-console.start-api.log` |
| Web-console tmux socket | `tmux -L wc ...` |
| Tmux session name | `wc-<web-console-session-id>` |

## Status state machine

```
            create (UI/CLI)
                  │
                  ▼
                live ─── delete by user ──► gone
                  │
                  │ tmux missing on Recover()
                  ▼
         awaiting_recovery
                  │
        ┌─────────┼──────────────┐
        │         │              │
   recover    dismiss     tmux reappears + Recover()
        │         │              │
        ▼         ▼              ▼
   dismissed dismissed         live
```

Only `Recover()` and the recovery endpoints transition status. Application code never deletes a `detached=1` row directly.

## How agent identity is captured

The recovery endpoints rely on `agent_type` + `agent_session_id` being on the row. Three independent populators put them there:

- **Codex tailer** — when `CodexTailer` first opens a rollout under a session's `CODEX_HOME`, it parses the `session_meta` first line and stores `agent_type=codex`, `agent_session_id=<payload.id>`, `cwd=<payload.cwd>`, `last_rollout_path`.
- **Claude Stop hook** — `POST /api/v1/hooks/stop` carries `web_console_session_id`, `session_id` (claude's own UUID), and `cwd`. The handler stores `agent_type=claude`, `agent_session_id=<session_id>`, `cwd`.
- **Launch metadata** — `CreateSessionRequest` accepts optional `launch_command` and `agent_type`. The UI launcher sends them when the user picks a shortcut; they populate the row up front (the tailer/hook will refine `agent_session_id` once the agent emits something). When Create sets `execute_launch_command=true`, the server pastes `launch_command` into the fresh session's stdin using the **same paste seam** recovery uses to resume an agent (a `{"type":"stdin","kind":"paste"}` write), so a programmatic caller can launch a command headlessly with no browser attached. Recovery's own resume path is unchanged.

If a pane runs an agent we don't classify (custom command), `agent_type` stays `none` and the row appears in `list-recoverable` with `recoverable=false`. The on-disk state is still preserved; the row just can't be auto-resumed and the appendix below applies.

## Appendix: Manual fallback (for when the CLI itself is broken)

This is the historical procedure documented for the 2026-05-01 incident. Use it only when `web-console session recover` is unavailable (e.g., the API panics during startup so the recovery routes never bind). It is intentionally verbose — every step is independently verifiable.

### A.1 Find lost session ids without the CLI

If the API can serve some routes but the CLI is broken:

```bash
api_port=$(vrooli scenario port web-console API_PORT)
curl -sS "http://127.0.0.1:$api_port/api/v1/sessions/recoverable" | jq
```

If the API is fully down, walk the codex state tree:

```bash
for d in ~/.local/state/vrooli/web-console/sessions/codex/*/sessions; do
  id="${d%/sessions}"; id="${id##*/}"
  latest=$(find "$d" -name '*.jsonl' 2>/dev/null | sort | tail -1)
  [ -z "$latest" ] && continue
  echo "$id $(head -1 "$latest" | jq -r '.payload.id // empty')"
done
```

### A.2 Recreate the pane manually

```bash
api="http://127.0.0.1:$api_port"
new_id="$(
  curl -sS -X POST "$api/api/v1/sessions" \
    -H 'Content-Type: application/json' \
    -d '{"shell":"/bin/bash","cols":120,"rows":36,"backend":"persistent","policy":{"mode":"never"}}' |
  jq -r '.id'
)"

old_id="<old-web-console-session-id>"
old_home="$HOME/.local/state/vrooli/web-console/sessions/codex/$old_id"
new_home="$HOME/.local/state/vrooli/web-console/sessions/codex/$new_id"
rsync -a "$old_home/" "$new_home/"

curl -sS -X PUT "$api/api/v1/workspace/panes/$new_id" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Restored Codex","supports_messages_view":true}' | jq
```

### A.3 Paste the resume command

The simplest path is a Python WebSocket client; see the script template at the end of this section. Codex resume:

```bash
codex_session=<from A.1>
echo "codex --yolo resume $codex_session"
```

For Claude, prefer `claude --resume <id>` over `--continue` — guessing the latest project is unsafe.

```python
# resume_paste.py
import json, websocket
ws = websocket.create_connection(f"ws://127.0.0.1:{api_port}/api/v1/sessions/{new_id}/ws")
while True:
    msg = json.loads(ws.recv())
    if msg.get("type") == "session_ready":
        break
ws.send(json.dumps({"type": "stdin", "data": "codex --yolo resume <id>\n", "kind": "paste"}))
ws.close()
```

### A.4 Copy conversation events (manual fallback only)

> The CLI `web-console session recover` now copies conversation history automatically (step 4 above). This manual SQL is only needed in the broken-API fallback where you recreated the pane by hand in A.2.

If the lost pane had assistant messages already captured by the server, copy them into the new pane's session id:

```bash
db="$HOME/.local/share/vrooli/web-console/web-console.db"
cp "$db" "$db.pre-session-recovery-$(date -u +%Y%m%dT%H%M%SZ).bak"

sqlite3 "$db" <<SQL
BEGIN;
INSERT OR IGNORE INTO conversation_sessions (
    session_id, last_sequence, last_seen_sequence, last_listened_sequence, created_at, updated_at
) SELECT '<new-id>', last_sequence, last_seen_sequence, last_listened_sequence, created_at,
         strftime('%Y-%m-%dT%H:%M:%fZ','now')
  FROM conversation_sessions WHERE session_id = '<old-id>';

INSERT OR IGNORE INTO conversation_events (
    id, session_id, source, role, text, speech_paragraphs, original_speech_paragraphs,
    summarized, created_at, sequence, delivery_state, tts_state, consumption_state
) SELECT lower(hex(randomblob(16))), '<new-id>', source, role, text,
         speech_paragraphs, original_speech_paragraphs, summarized, created_at, sequence,
         delivery_state, tts_state, consumption_state
  FROM conversation_events WHERE session_id = '<old-id>';
COMMIT;
SQL
```

### A.5 Verify

```bash
web-console session list                    # new pane appears
tmux -L wc list-sessions                    # wc-<new-id> alive
tmux -L wc capture-pane -pt "wc-<new-id>" -S -80
```

## Incident Pattern From 2026-05-01

During the 2026-05-01 host crash, the API started cleanly but `Recover()` deleted ten persisted session metadata rows because no matching tmux session survived. Eight Codex panes had recoverable rollout histories; two had server-side conversation events but no rollout. Recovery required manually recreating panes, rsync-ing each `CODEX_HOME`, and pasting `codex --yolo resume` over a WebSocket. Post-hardening (this version of the doc), that whole sequence collapses to:

```bash
for id in $(web-console session list-recoverable --json | jq -r '.[].id'); do
  web-console session recover "$id"
done
```

Claude panes are still gated on a non-empty `agent_session_id` because guessing is unsafe.
