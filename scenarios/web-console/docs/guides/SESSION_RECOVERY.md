# Session Recovery

Web-console has two recovery modes:

- **Normal restart recovery**: persistent sessions are tmux-backed. If the API restarts but the host and tmux server stay alive, startup recovery re-attaches to the existing `wc-<session-id>` tmux sessions.
- **Crash recovery**: if the host crashes or reboots, tmux sessions are gone. The terminal processes cannot be re-attached, but web-console can usually recreate panes and resume Codex or Claude from durable agent history.

This guide covers both paths, with emphasis on full crash recovery.

## Storage Locations

Default local paths:

| Data | Path |
|------|------|
| SQLite database | `~/.local/share/vrooli/web-console/web-console.db` |
| API startup log | `~/.vrooli/logs/scenarios/web-console/vrooli.develop.web-console.start-api.log` |
| Web-console tmux socket | `tmux -L wc ...` |
| Tmux session name | `wc-<web-console-session-id>` |
| Per-pane Codex home | `~/.local/state/vrooli/web-console/sessions/codex/<web-console-session-id>` |
| Per-pane Claude state, when present | `~/.local/state/vrooli/web-console/sessions/claude/<web-console-session-id>` |
| Claude project history | `~/.claude/projects/<project-key>/*.jsonl` |

The database stores session metadata, workspace panes, conversation events, and Codex rollout checkpoints. The tmux server stores live terminal processes. A reboot destroys the tmux server, so crash recovery must use the database plus agent history.

## Quick Triage

Find the API port:

```bash
vrooli scenario port web-console API_PORT
```

List active sessions known to the API:

```bash
curl -sS "http://127.0.0.1:<API_PORT>/api/v1/sessions" | jq
```

List surviving persistent tmux sessions:

```bash
tmux -L wc list-sessions -F '#{session_name}'
```

Interpretation:

- If API sessions and `wc-...` tmux sessions exist, startup recovery worked.
- If the API has no old sessions and tmux has no `wc-...` sessions, this was a full crash/reboot recovery case.
- If tmux has `wc-...` sessions but the API does not, restart the API once before doing manual recovery. Startup recovery should re-register them.

## Find Lost Session IDs

After a full crash, the startup log usually records which persisted metadata rows were cleaned because no matching tmux session survived:

```bash
rg 'orphan|cleaned|recovery' ~/.vrooli/logs/scenarios/web-console/vrooli.develop.web-console.start-api.log
```

Those session IDs are the old web-console pane IDs. Use them to look for matching Codex or Claude history.

## Recover Codex Panes

Each web-console pane gets its own `CODEX_HOME`. If Codex wrote rollout files before the crash, they are normally still under:

```bash
~/.local/state/vrooli/web-console/sessions/codex/<old-web-console-session-id>/sessions/YYYY/MM/DD/*.jsonl
```

Extract the Codex session ID from the latest rollout:

```bash
old_id="<old-web-console-session-id>"
codex_home="$HOME/.local/state/vrooli/web-console/sessions/codex/$old_id"

rg --files "$codex_home/sessions" -g '*.jsonl' |
  sort |
  tail -1 |
  xargs jq -r 'select(.session_id != null) | .session_id' |
  tail -1
```

Create a new persistent pane:

```bash
api="http://127.0.0.1:<API_PORT>"

new_id="$(
  curl -sS -X POST "$api/api/v1/sessions" \
    -H 'Content-Type: application/json' \
    -d '{"shell":"/bin/bash","cols":120,"rows":36,"backend":"persistent","policy":{"mode":"never"}}' |
  jq -r '.id'
)"
```

Copy the old Codex home into the new pane before launching Codex:

```bash
old_home="$HOME/.local/state/vrooli/web-console/sessions/codex/$old_id"
new_home="$HOME/.local/state/vrooli/web-console/sessions/codex/$new_id"

rsync -a "$old_home/" "$new_home/"
```

Optionally rename the pane in the workspace:

```bash
curl -sS -X PUT "$api/api/v1/workspace/panes/$new_id" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Restored Codex","supportsMessagesView":true}' |
  jq
```

Open the session WebSocket and paste a resume command into the terminal:

```bash
codex_session="<codex-session-id>"
command="codex --yolo resume $codex_session"
```

Any WebSocket client can send the command. For example, with Python:

```python
import json
import websocket

api_port = "<API_PORT>"
web_console_session = "<new-web-console-session-id>"
command = "codex --yolo resume <codex-session-id>\n"

ws = websocket.create_connection(
    f"ws://127.0.0.1:{api_port}/api/v1/sessions/{web_console_session}/ws"
)

while True:
    message = json.loads(ws.recv())
    if message.get("type") == "session_ready":
        break

ws.send(json.dumps({"type": "stdin", "data": command, "kind": "paste"}))
ws.close()
```

If Codex shows an update prompt before resuming, choose the skip option, then re-send the resume command if needed.

When a pane has a Codex home but no rollout `history.jsonl`, there is no exact Codex session ID to target. In that case use:

```bash
codex --yolo resume --last
```

That can only resume whatever Codex considers the latest session in that copied home.

## Recover Conversation Events

If the terminal process could not be resumed but web-console had already captured assistant messages, copy the old conversation rows to the new pane ID. Back up the database first:

```bash
db="$HOME/.local/share/vrooli/web-console/web-console.db"
cp "$db" "$db.pre-session-recovery-$(date -u +%Y%m%dT%H%M%SZ).bak"
```

Then duplicate conversation state:

```sql
-- Run with: sqlite3 ~/.local/share/vrooli/web-console/web-console.db

BEGIN;

INSERT OR IGNORE INTO conversation_sessions (
    session_id,
    last_sequence,
    last_seen_sequence,
    last_listened_sequence,
    created_at,
    updated_at
)
SELECT
    '<new-web-console-session-id>',
    last_sequence,
    last_seen_sequence,
    last_listened_sequence,
    created_at,
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM conversation_sessions
WHERE session_id = '<old-web-console-session-id>';

INSERT OR IGNORE INTO conversation_events (
    id,
    session_id,
    source,
    role,
    text,
    speech_paragraphs,
    original_speech_paragraphs,
    summarized,
    created_at,
    sequence,
    delivery_state,
    tts_state,
    consumption_state
)
SELECT
    lower(hex(randomblob(16))),
    '<new-web-console-session-id>',
    source,
    role,
    text,
    speech_paragraphs,
    original_speech_paragraphs,
    summarized,
    created_at,
    sequence,
    delivery_state,
    tts_state,
    consumption_state
FROM conversation_events
WHERE session_id = '<old-web-console-session-id>';

COMMIT;
```

Only do this after creating the replacement pane. It preserves the old pane's message feed under the new active session ID so Messages view, unread state, and TTS cursor state can use it again.

## Recover Claude Panes

Claude recovery needs a stronger mapping than Codex because Claude stores project histories globally under `~/.claude/projects/`. Do not start the newest Claude history just because it is recent; it may belong to a different terminal.

Use one of these signals before resuming Claude:

- A web-console pane title or metadata that identifies the Claude task.
- A per-pane Claude state directory under `~/.local/state/vrooli/web-console/sessions/claude/<old-web-console-session-id>`.
- A Claude JSONL session whose cwd, timestamps, and prompts match the lost pane.

Once mapped, recreate a persistent pane as above and launch one of:

```bash
claude --resume <claude-session-id> --dangerously-skip-permissions
claude --continue --dangerously-skip-permissions
```

Prefer `--resume <id>` when you have an exact session ID. Use `--continue` only when you have verified the target project's latest Claude session is the intended one.

## Verify Recovery

Check API state:

```bash
curl -sS "$api/api/v1/sessions" | jq '.sessions[] | {id, backend, status, policy}'
```

Check tmux state:

```bash
tmux -L wc list-sessions -F '#{session_name}'
```

Check process state:

```bash
ps -ef | rg 'codex|claude|tmux attach-session|wc-'
```

Capture a pane to confirm it is attached and displaying the resumed agent:

```bash
tmux -L wc capture-pane -pt "wc-<new-web-console-session-id>" -S -80
```

Finally, reload the browser UI and confirm:

- The restored panes appear in the workspace.
- Each pane uses the `persistent` backend.
- Agent TUIs are interactive.
- Messages view shows any copied conversation history.

## Incident Pattern From 2026-05-01

During the 2026-05-01 crash recovery, the API started cleanly but removed ten persisted session metadata rows because the host crash had destroyed the tmux server. Eight old Codex panes had recoverable rollout histories and were recreated with exact `codex --yolo resume <session-id>` commands. Two panes had web-console conversation history but no Codex rollout history, so they were recreated with `codex --yolo resume --last` and their conversation rows were copied from the old pane IDs to the new ones.

No Claude pane was resumed during that incident because the available Claude histories did not have a reliable active-pane mapping. That is intentional: guessing can attach the wrong Claude conversation to the wrong workspace pane.

## Prevention Work

The manual process above works, but the product should eventually automate it. Useful improvements:

- Preserve orphaned persistent-session metadata after a full reboot instead of deleting it immediately.
- Store the launch command, agent type, cwd, and agent conversation ID on each workspace pane.
- Add a recovery endpoint or CLI command that can recreate panes from old metadata and copied Codex homes.
- Add a first-class conversation-copy operation so recovery does not require direct SQLite edits.
- Surface "recoverable after crash" panes in the UI when old agent history exists but tmux does not.
