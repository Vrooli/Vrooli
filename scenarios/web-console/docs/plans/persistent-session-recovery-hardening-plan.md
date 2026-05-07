# Persistent Session Recovery Hardening — Implementation Plan

## 1. Purpose

Make `backend=persistent` web-console panes actually durable across the failure modes that today silently destroy them, and turn the 8-step manual `codex --yolo resume` recovery procedure documented in [SESSION_RECOVERY.md](../guides/SESSION_RECOVERY.md) into a one-call CLI / one-button UI flow.

Two concrete defects motivate this work:

1. `(*SessionManager).Recover` deletes orphaned persistent-session metadata when the matching `wc-<id>` tmux session is gone (api/session.go:875-878). This irrevocably erases the only DB pointer to the per-session `CODEX_HOME` and conversation history. After a host reboot, a `wc-tmux-server` scope kill, or an OOM-kill of the tmux server, every "persistent" pane row is wiped on the next API start, which is the opposite of persistence.
2. No agent-launch identity is stored on a pane. There is no record of "this pane was running `codex` against rollout `019d…`" or "this pane was running `claude --resume <id>`". Recovery requires walking `~/.local/state/vrooli/web-console/sessions/codex/<old-id>/sessions/YYYY/MM/DD/*.jsonl`, parsing the first `session_meta` line, then rsync-ing the home into a fresh pane. `docs/guides/SESSION_RECOVERY.md` already calls both out under "Prevention Work".

After this plan is executed, persistent panes survive any failure mode short of disk loss, agent context is auto-recovered, and the recovery procedure for tmux-loss is `web-console session recover` instead of seven shell commands and a Python WebSocket script.

## 2. Required Reading

Run before authoring or executing any phase below:

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also read in-repo:

```bash
sed -n '1,160p' scenarios/web-console/docs/guides/SESSION_RECOVERY.md
sed -n '1,40p'  scenarios/web-console/docs/guides/CONVERSATION_TRACKING.md
sed -n '848,990p' scenarios/web-console/api/session.go        # Recover()
sed -n '1,80p'   scenarios/web-console/api/session_store.go    # SessionMetadata + SQLSessionStore
sed -n '1,130p' scenarios/web-console/api/codex_tailer.go      # rollout discovery
cat              scenarios/web-console/initialization/sqlite/schema.sql
```

## 3. Greenfield Constraint (Hard Rule)

Per project standing rules ([feedback_planning_guidelines](../../../.claude/projects/-home-matthalloran8-Vrooli/memory/feedback_planning_guidelines.md), repeated here so this plan is self-contained):

- This is greenfield. Do **not** add migration shims, dual-write paths, or "preserve old behavior under a flag" toggles. Existing rows that lack the new columns get sensible defaults from the migration; that is the whole compatibility story.
- Restart `vrooli scenario restart web-console` before declaring done.
- Use automated tests, not manual walkthroughs, for the acceptance gate.
- All operator surfaces are CLI-first (`web-console session ...`); no raw `curl` / `psql` instructions in user-facing docs. If the CLI is missing a subcommand needed for recovery, add it as part of this plan rather than working around it.

## 4. Problem Statement

### 4.1 Recovery deletes the only thing that proves a session existed

`(*SessionManager).Recover` (api/session.go:873-882):

```go
for id, meta := range metaMap {
    if !tmuxSet[id] {
        // tmux session is gone — clean up stale metadata
        _ = store.Delete(id)
        report.OrphanedMetadata++
        continue
    }
    ...
}
```

The branch fires whenever the tmux server is gone but DB metadata remains — exactly the case where recovery matters most. Today this happens on:

- Host reboot (tmux server is per-`wc` socket, not persisted).
- `systemctl kill wc-tmux-server.scope` or OOM-kill of the tmux server cgroup.
- Manual `tmux -L wc kill-server`.
- Any crash that takes pid 1 of the scope down.

Once the row is `DELETE`d:

- `workspace_panes.session_id` becomes a dangling pointer (no FK), so the pane lingers in the layout but is unrecoverable from API state alone.
- `~/.local/state/vrooli/web-console/sessions/codex/<id>/` is preserved on disk, but nothing in the API knows the directory exists, because the only index is the row that was just deleted.
- Recovery becomes the manual procedure documented in `SESSION_RECOVERY.md` §Recover Codex Panes.

### 4.2 Pane has no record of its agent identity

Today the only fact persisted about an agent running in a pane is `workspace_panes.supports_messages_view` (a boolean). There is no:

- `agent_type` (none / codex / claude)
- `launch_command` (the literal string sent over the WebSocket as the first paste)
- `agent_session_id` (codex session UUID; Claude session UUID)
- `cwd_at_launch`
- `last_rollout_path` / `last_rollout_offset_at_disconnect`

Recovery agents (human or automated) have to:

1. Glob rollout files under the pane's `CODEX_HOME`.
2. `head -1 | jq` to extract `payload.id` and `payload.cwd`.
3. Hope no second rollout was opened later in the same dir.
4. Issue `codex --yolo resume <id>` themselves.

This is fragile and undocumented for Claude entirely (Claude history lives in `~/.claude/projects/<project-key>/*.jsonl` with no per-pane mapping).

## 5. Scope

### In scope

- Stop deleting orphan persistent-session metadata in `Recover()`.
- Add a status field that marks orphaned-but-recoverable rows; surface them via API + CLI + UI.
- Add agent-identity columns to `sessions` (the row that survives orphan, not `workspace_panes`).
- Auto-populate agent-identity from existing signals (codex rollout `session_meta` event, claude Stop hook payload, shortcut launch path).
- Build a `web-console session recover <old-id>` CLI command + matching `POST /api/v1/sessions/{id}/recover` endpoint that performs the full rsync-and-resume flow we currently document as eight manual steps.
- Build a `web-console session list-recoverable` CLI command + matching `GET /api/v1/sessions/recoverable` endpoint.
- Update `SESSION_RECOVERY.md` to lead with the CLI command and demote the manual procedure to "if the CLI fails" appendix.
- Update `docs/manifest.json` if any new doc surface is added.

### Out of scope

- Recreating panes for sessions whose on-disk codex/claude history is also gone (not recoverable by definition).
- Process-level isolation changes for the tmux server (already handled via `wc-tmux-server.scope`).
- Cross-host recovery (single-host only; multi-host is a separate scenario).
- Migrating ancient `backend=standard` rows. Those are correctly destroyed on restart; the fix is to default `auto → persistent` (already true) and educate users via the launcher UI (already done).
- Generic "snapshot a running shell's command history into a launchable script". We only resume agents we know how to resume (codex / claude).

## 6. Current Technical Context

| Concern | Location |
|---|---|
| Recover() with the bug | api/session.go:848-991 |
| Session metadata schema | scenarios/web-console/initialization/sqlite/schema.sql:6-17 + migrations in api/main.go:54-58 |
| `SessionMetadata` Go type | api/session_store.go:11-20 |
| `SessionResponse` JSON | api/session_handlers.go:89-119 |
| Codex rollout watcher | api/codex_tailer.go (whole file) |
| Codex rollout `session_meta` first-line shape | `{type:"session_meta", payload:{id, cwd, ...}}` — confirmed against `~/.local/state/vrooli/web-console/sessions/codex/8a10aa94-…/sessions/2026/05/06/rollout-….jsonl` |
| Claude Stop hook handler | api/hook_prompt_submit_handler.go (and tts_hook_handler.go) — payload carries `WC_WEB_CONSOLE_SESSION_ID` per CONVERSATION_TRACKING.md |
| Per-session env injection | api/session.go:655-665 — already injects `WC_WEB_CONSOLE_SESSION_ID`, `CODEX_HOME`, `WC_CODEX_SESSIONS_DIR` |
| UI launcher → first-paste path | ui/src/hooks/useSessionManager.ts:190-229 (`launchSession` → `pendingCommands.current.set` → `flushPendingCommand` on `terminalReady`) |
| CLI session subcommand registry | scenarios/web-console/cli/domains/session/register.go |
| Existing recovery doc | scenarios/web-console/docs/guides/SESSION_RECOVERY.md |
| Existing recovery tests | api/session_recovery_test.go (`TestRecover_*`) |

## 7. Target End State

After this plan ships:

1. `sessions.detached = 1` rows are **never deleted by `Recover()`**. They transition through statuses: `live`, `awaiting_recovery`, `recovered`, `dismissed`. Only `dismissed` rows are eligible for purge, and only after a TTL.
2. Every persistent session row carries enough metadata to reconstruct the agent it was running, populated automatically as the agent runs.
3. `web-console session list` shows live sessions; `web-console session list-recoverable` shows orphaned-but-recoverable ones.
4. `web-console session recover <session-id>` performs the manual procedure end-to-end, including the WebSocket paste, in a single call.
5. The UI's "Tabs" surface shows recoverable panes with a one-click "Reattach" action that invokes the same endpoint.
6. Test suite covers: orphan preservation, agent metadata population from rollout, agent metadata population from Claude Stop hook, recover endpoint happy path, recover endpoint when codex_session_id is missing (falls back to `--last`).
7. `docs/guides/SESSION_RECOVERY.md` reads "run `web-console session recover <id>`" in the first ~20 lines; the long manual procedure stays as an appendix for the case where the CLI is broken.

## 8. Implementation Strategy (Phased)

Phases are listed in dependency order. Each phase ends with a green test gate.

### Phase 1 — Schema + metadata model

Files: `scenarios/web-console/initialization/sqlite/schema.sql`, `scenarios/web-console/api/main.go` (migrations block), `scenarios/web-console/api/session_store.go`.

1. Extend `sessions` schema:

   ```sql
   CREATE TABLE IF NOT EXISTS sessions (
       id TEXT PRIMARY KEY,
       backend TEXT NOT NULL DEFAULT 'standard',
       shell TEXT NOT NULL DEFAULT '/bin/bash',
       cols INTEGER NOT NULL DEFAULT 80,
       rows INTEGER NOT NULL DEFAULT 24,
       policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never','preset','custom')),
       policy_duration TEXT,
       created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
       detached INTEGER NOT NULL DEFAULT 0,
       -- new:
       status TEXT NOT NULL DEFAULT 'live'
           CHECK(status IN ('live','awaiting_recovery','dismissed')),
       agent_type TEXT NOT NULL DEFAULT 'none'
           CHECK(agent_type IN ('none','codex','claude')),
       launch_command TEXT,
       agent_session_id TEXT,
       cwd TEXT,
       last_rollout_path TEXT,
       last_activity_at TEXT,
       orphaned_at TEXT
   );
   CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
   CREATE INDEX IF NOT EXISTS idx_sessions_agent  ON sessions(agent_type, agent_session_id);
   ```

2. Add idempotent `ALTER TABLE` migrations to `api/main.go:54-58` for each new column. Reuse existing `isDuplicateColumnError` predicate.

3. Extend `SessionMetadata`:

   ```go
   type SessionMetadata struct {
       ID, Backend, Shell        string
       Cols, Rows                uint16
       Policy                    ExpirationPolicy
       Created                   time.Time
       Detached                  bool
       Status                    SessionStatus    // "live"|"awaiting_recovery"|"dismissed"
       AgentType                 AgentType        // "none"|"codex"|"claude"
       LaunchCommand             string
       AgentSessionID            string
       CWD                       string
       LastRolloutPath           string
       LastActivityAt            time.Time        // zero if unknown
       OrphanedAt                time.Time        // zero unless awaiting_recovery
   }
   ```

4. Wire `Save`, `Get`, `List`, `ListDetached`, `scanSessionMetadata`, and `InMemorySessionStore` to all the new fields. Add three focused store methods:

   ```go
   UpdateAgentInfo(id string, info AgentInfo) error            // codex_tailer + claude hook callers
   MarkOrphaned(id string, at time.Time) error                 // called by Recover
   MarkDismissed(id string) error                              // called by recover endpoint after success / explicit dismiss
   ListRecoverable() ([]SessionMetadata, error)                // status='awaiting_recovery'
   ```

Test gate: `go test ./api/... -run TestSessionStore -timeout 60s` green.

### Phase 2 — Stop deleting orphan metadata

Files: `scenarios/web-console/api/session.go`.

Replace the orphan branch in `Recover()` (api/session.go:875-879):

```go
if !tmuxSet[id] {
    if err := store.MarkOrphaned(id, time.Now()); err != nil {
        log.Printf("recovery: failed to mark orphan %s: %v", id, err)
    }
    report.OrphanedMetadata++
    log.Printf("recovery: marked session %s as awaiting_recovery (tmux gone)", id)
    continue
}
```

The success path (after attach) sets `status='live'` and clears `orphaned_at`:

```go
_ = store.MarkLive(id)
```

Add `RecoveryReport.AwaitingRecovery int` so the startup log line stays informative:

```
recovery: recovered=2 awaiting_recovery=1 orphaned_tmux=0
```

Backfill existing rows once on first start of the new code: `UPDATE sessions SET status='live' WHERE status IS NULL OR status=''`. This is part of the migration, not runtime logic.

Test gate: extend `TestRecover_OrphanedMetadata_NoTmuxSession` (api/session_recovery_test.go:36) — assert the row is preserved with `status='awaiting_recovery'` and `orphaned_at` set, instead of asserting `_, err := store.Get(...) == not-found`. Add `TestRecover_AwaitingRecoveryReattachOnNextStart` to verify a session that becomes `awaiting_recovery` and then gets a tmux session created by hand transitions back to `live`.

### Phase 3 — Agent identity capture

Three independent populators feed `UpdateAgentInfo`:

3a. **Codex rollout discovery** — augment `CodexTailer.scanForNewFiles` (api/codex_tailer.go:65-95). When opening a rollout file for the first time, read the first JSONL line; if it parses as `{type:"session_meta", payload:{id, cwd, ...}}`, call:

```go
ct.server.sessionStore.UpdateAgentInfo(sessionID, AgentInfo{
    AgentType:       AgentCodex,
    AgentSessionID:  meta.Payload.ID,
    CWD:             meta.Payload.CWD,
    LastRolloutPath: rolloutPath,
    LastActivityAt:  time.Now(),
})
```

The first-line read is cheap (rollouts are line-oriented; first line ~3KB max) and only happens once per file. Use the existing watcher map (`ct.watchers`) as the dedupe gate.

3b. **Claude Stop hook** — in `api/hook_prompt_submit_handler.go` (or wherever the Stop hook payload lands; confirm via `grep -rn 'WC_WEB_CONSOLE_SESSION_ID' api/`). When a hook payload arrives with `session_id` populated, call `UpdateAgentInfo` with `AgentType=AgentClaude`, `AgentSessionID=<claude session uuid from payload>`, `CWD=<payload cwd>`. The Stop hook already carries enough — see CONVERSATION_TRACKING.md.

3c. **Launch command** — there is no server-side knowledge of the first-paste command today. Add `launch_command` and `agent_type` to `CreateSessionRequest`:

```go
type CreateSessionRequest struct {
    Shell, Backend string
    Cols, Rows     int
    Policy         *CreatePolicySpec
    LaunchCommand  string `json:"launch_command,omitempty"`   // free-form
    AgentType      string `json:"agent_type,omitempty"`       // "codex" | "claude" | "none"
}
```

`handleCreateSession` (api/session_handlers.go:147-205) persists those into the metadata row. The UI already sends the command via `pendingCommands` and pastes it after `terminal_ready`; update `useSessionManager.launchSession` (ui/src/hooks/useSessionManager.ts:190) to also send `launch_command` and a derived `agent_type` (`supportsMessagesViewForCommand` already classifies). This is greenfield: drop `supportsMessagesView` derivation in favor of `agent_type != 'none'`.

Test gate:
- `TestCodexTailer_PopulatesAgentInfoOnFirstRollout`
- `TestClaudeHook_PopulatesAgentInfo`
- `TestCreateSession_PersistsLaunchCommandAndAgentType`
- Update `useSessionManager.test.ts` to assert the `launch_command` payload field is sent.

### Phase 4 — Recovery API + CLI

4a. New endpoint `POST /api/v1/sessions/{id}/recover`. Behavior:

1. Look up `id` in `sessions`. Reject with 404 if missing, 409 if `status != 'awaiting_recovery'`.
2. Create a fresh persistent session (same code path as `handleCreateSession` with `backend=persistent`, copying `shell/cols/rows/policy` from the orphan row).
3. `rsync -a $CODEX_HOME($old_id)/ $CODEX_HOME($new_id)/` if `agent_type='codex'`. The directory copy is bounded (only the per-pane home, which is small) so no streaming required.
4. Issue the appropriate first paste over the new session's PTY without a WebSocket round-trip — the API can call `sess.WriteInput([]byte(cmd), InputKindPaste)` directly:
   - `agent_type='codex'` and `agent_session_id` non-empty: paste `codex --yolo resume <agent_session_id>\n`.
   - `agent_type='codex'` and `agent_session_id` empty: paste `codex --yolo resume --last\n`.
   - `agent_type='claude'` and `agent_session_id` non-empty: paste `claude --resume <agent_session_id> --dangerously-skip-permissions\n`.
   - `agent_type='claude'` and `agent_session_id` empty: return 422 with `claude_session_id_required` (Claude resume by guess is not safe — explicit doc rule from SESSION_RECOVERY.md).
5. Mark old row `status='dismissed'`, write `recovered_into_session_id = <new_id>` for audit.
6. Return `{old_session_id, new_session_id, agent_type, command_sent, copied_codex_home_bytes}`.
7. Optionally copy conversation_events from the old session_id to the new one if `?copy_history=true` (the SQL block in SESSION_RECOVERY.md §Recover Conversation Events). Default false.

The recovery handler must guard against double-recover (idempotency): an `X-Idempotency-Key` header reuse returns the existing `new_session_id`.

4b. New endpoint `GET /api/v1/sessions/recoverable`. Returns rows with `status='awaiting_recovery'`, ordered by `last_activity_at DESC`. Response shape mirrors `SessionResponse` plus `orphaned_at`, `agent_type`, `agent_session_id`, `cwd`, `last_rollout_path`.

4c. New endpoint `DELETE /api/v1/sessions/recoverable/{id}` — explicit dismiss. Sets `status='dismissed'`. Background sweeper (extend `ExpirationSweeper`) purges `dismissed` rows older than 30 days.

4d. CLI subcommands in `cli/domains/session/register.go`:

```go
{Name: "list-recoverable", Description: "List orphaned persistent sessions that can be recovered", Run: ...},
{Name: "recover",          Description: "Recover an orphaned persistent session into a fresh pane",  Run: ...},
{Name: "dismiss",          Description: "Permanently drop an orphaned session row",                  Run: ...},
```

Surface uses default human output per [feedback_cli_default_human_output](../../../.claude/projects/-home-matthalloran8-Vrooli/memory/feedback_cli_default_human_output.md). `recover` prints the new session id and a hint pointing to `web-console session get`.

Test gate:
- `TestHandleRecover_HappyPath_Codex`
- `TestHandleRecover_HappyPath_Codex_NoSessionIdFallsBackToLast`
- `TestHandleRecover_RejectsLiveSession` (409)
- `TestHandleRecover_RejectsClaudeWithoutSessionId` (422)
- `TestHandleRecover_Idempotent`
- `TestListRecoverable_OrdersByLastActivity`
- CLI surface tests in `cli/...` mirroring existing patterns.

### Phase 5 — UI surface

Files: `ui/src/lib/api.ts`, `ui/src/components/Workspace.tsx`, new `ui/src/components/RecoverableSessionsBanner.tsx`.

1. Add typed wrappers `listRecoverableSessions()`, `recoverSession(oldId)`, `dismissRecoverable(oldId)` in `ui/src/lib/api.ts`.
2. On Workspace mount, fetch `listRecoverableSessions()`. If non-empty, show a banner above the tab strip: "N panes are recoverable from a previous session". Banner expands into rows with `Reattach` and `Dismiss` buttons.
3. On `Reattach` success, hand the new session id to the existing pane-activation reconciliation effect (same path as `handleLaunch`).

Test gate: `RecoverableSessionsBanner.test.tsx` + extend `workspace.test.tsx` to mock the recoverable list and assert banner visibility + click → API call wiring.

### Phase 6 — Documentation + manifest

1. Rewrite `scenarios/web-console/docs/guides/SESSION_RECOVERY.md`:
   - Lead with `web-console session list-recoverable` + `web-console session recover <id>`.
   - Demote the manual rsync / WebSocket-paste procedure to an appendix titled "Manual fallback (CLI broken or partial state)".
   - Update the §"Incident Pattern From 2026-05-01" entry to note "post-hardening this would be one CLI call" without rewriting history.
   - Remove §"Prevention Work" — the items there are this plan.
2. Update `scenarios/web-console/docs/manifest.json` only if new docs are added. The existing guide entry stays.
3. Update `scenarios/web-console/README.md` if it has a recovery section (verify: `grep -n -i recover scenarios/web-console/README.md`); otherwise no change.

## 9. Contract Decisions

### 9.1 Status state machine

```
            create
              │
              ▼
            live ─────► (delete by user) ─► gone (row removed)
              │
              │ tmux missing at startup
              ▼
       awaiting_recovery
              │
       ┌──────┼──────────┐
       │      │          │
   recover  dismiss   tmux re-appears
       │      │          │
       ▼      ▼          ▼
  dismissed dismissed   live
       │
   30-day sweep
       ▼
     gone
```

`live` is the only writable status from outside `Recover()` and the recovery endpoints. `dismissed` rows are immutable.

### 9.2 Agent type vocabulary

Closed set: `none`, `codex`, `claude`. New agent runtimes require an explicit code change. We do not parse free-form `launch_command` to derive type at recovery time — it must be persisted at launch or learned from the rollout.

### 9.3 What "recoverable" means

A row qualifies for `GET /api/v1/sessions/recoverable` iff:

- `status='awaiting_recovery'`
- `detached=1`
- `agent_type != 'none'` AND (`agent_session_id != ''` OR an on-disk rollout exists at `last_rollout_path`)

Bash-only orphans (`agent_type='none'`) are not recoverable in the agent sense — there's no session to resume into. They surface in `list-recoverable` only with `--include-bash` to avoid noise.

### 9.4 Idempotency

`POST /sessions/{id}/recover` honors `X-Idempotency-Key` per existing pattern (api/session_handlers.go:21-71). Replays return the same `new_session_id`.

### 9.5 No silent data loss

Even on `dismiss`, the on-disk `CODEX_HOME` is preserved. Only the DB row is removed. A separate command (out of scope for this plan, file as follow-up) can purge stale on-disk homes.

## 10. Testing Plan

All tests are automated. No manual test checklist.

| Layer | File | Cases |
|---|---|---|
| Schema/store | api/session_store_test.go | new columns round-trip, status enum constraint, ListRecoverable filter, MarkOrphaned/MarkLive/MarkDismissed transitions |
| Recovery loop | api/session_recovery_test.go | replace `TestRecover_OrphanedMetadata_NoTmuxSession`; add `TestRecover_AwaitingRecoveryReattachOnNextStart`, `TestRecoveryReport_AwaitingRecoveryCount` |
| Codex tailer | api/codex_tailer_agent_info_test.go (new) | first-rollout populates agent_type/session_id/cwd; subsequent rollouts only update last_activity_at |
| Claude hook | api/hook_prompt_submit_handler_test.go | hook payload populates agent_type='claude' + agent_session_id |
| Create session | api/session_handlers_test.go | launch_command + agent_type round-trip |
| Recover endpoint | api/session_recover_handler_test.go (new) | happy paths (codex/claude), 404, 409, 422, idempotency, --last fallback |
| List recoverable | api/session_recoverable_handler_test.go (new) | ordering, agent_type='none' filter, awaiting_recovery only |
| CLI | cli/domains/session/recover_test.go (new) | flag parsing, JSON output shape, error mapping |
| UI | ui/src/__tests__/RecoverableSessionsBanner.test.tsx (new) | empty list hides banner, populated shows rows, click invokes API |
| Integration | api/session_recovery_e2e_test.go (new) | spawn real tmux session, kill it, restart server, run recover endpoint, assert codex resume command was pasted into new pane (capture via tmux capture-pane) |

Run gate (must all pass before declaring done):

```bash
cd scenarios/web-console
go test ./api/... -timeout 600s
( cd cli && go test ./... -timeout 120s )
( cd ui && yarn test --run )
vrooli scenario test web-console
```

## 11. Rollout / Validation Checklist

This is the post-implementation verification, all automated:

1. `go test`, `yarn test`, and `vrooli scenario test web-console` all green (Phase 10 commands).
2. `vrooli scenario restart web-console`.
3. `web-console session list` shows the live sessions; the count matches `tmux -L wc list-sessions | wc -l`.
4. Run a synthetic crash test (script: `scenarios/web-console/scripts/crash-recovery-smoke.sh`, to be added in Phase 4):
   - Create a persistent pane via the CLI.
   - Paste `codex --yolo` and let it write at least one rollout line.
   - `tmux -L wc kill-server`.
   - `vrooli scenario restart web-console`.
   - Assert `web-console session list-recoverable` shows the orphan with `agent_type=codex` and a non-empty `agent_session_id`.
   - `web-console session recover <old-id>`.
   - Assert `tmux -L wc capture-pane -pt wc-<new-id>` contains the resumed codex header.
5. Smoke script must be idempotent and self-cleaning. Run it twice in a row to prove that.
6. `golangci-lint run ./scenarios/web-console/api/... ./scenarios/web-console/cli/...` clean.
7. `gofumpt -l scenarios/web-console/api scenarios/web-console/cli` empty.

## 12. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Orphan rows accumulate forever | Medium | Disk + UI noise | 30-day sweep on `dismissed`; `list-recoverable` paginates and excludes `none` agent types |
| First-line rollout parse fails on a future codex format change | Low | Agent identity goes unpopulated for new sessions; recovery still works via `--last` fallback | Tolerant parse — catch any error, log once, continue. Don't block the tailer |
| `rsync` copies a too-large CODEX_HOME and blocks the API goroutine | Low | Slow recovery call | Run `rsync` in a goroutine with a 30s timeout; surface `409 recovery_in_progress` if a second call hits a busy entry |
| `WriteInput(InputKindPaste)` to a not-yet-ready PTY drops the resume command | Medium | Pane comes up to a bare bash prompt | Reuse the existing `pendingCommands` queue (UI side) but on the server side. Server-side recover endpoint waits for `session_ready` before pasting; same pattern as the WebSocket session_ready gate (api/terminal_ws.go:327) |
| Claude resume pasted against the wrong cwd | Medium | Wrong project loads | We persist `cwd` and verify the new pane's pwd matches before pasting; if not, prepend `cd <cwd> && ` |
| Idempotency key collision across days | Low | Stale recovery returned | TTL on idempotency cache is already 5min (api/session_handlers.go:33); inherits |
| Migration runs on a prod DB with the columns half-added (if a previous attempt errored) | Low | Startup loops | Existing `isDuplicateColumnError` predicate handles this; new columns each in their own ALTER for granular skip |
| `agent_session_id` populated by tailer races with `recover` call before tailer sees rollout | Low | First recover call uses `--last` fallback, second works correctly | Acceptable — `--last` is correct in this case anyway |

## 13. Non-goals / Prohibited Patterns

- Do **not** add a `migrate_old_to_new_status.sql` shim. The single ALTER + UPDATE in Phase 1 is the migration.
- Do **not** introduce a new `recovery_runs` table to track recover calls. The `sessions.status` field is sufficient.
- Do **not** add a feature flag for the new behavior. Once shipped, orphan-preservation is the only behavior.
- Do **not** parse claude/codex transcript bodies to infer agent type. Use the documented signals (rollout `session_meta`, hook payload, launch metadata).
- Do **not** open a WebSocket from the API to itself to send the resume paste. Call `sess.WriteInput` directly — it's the same code path the WS handler uses.
- Do **not** keep `supportsMessagesView` as a separate boolean once `agent_type` exists. Drop the column in the same schema change (greenfield).
- Do **not** add `curl ...` examples to `SESSION_RECOVERY.md`. Lead with the CLI per project rule [feedback_skills_use_cli_never_api](../../../.claude/projects/-home-matthalloran8-Vrooli/memory/feedback_skills_use_cli_never_api.md).

## 14. Definition of Done

A future agent or human can declare this plan done iff **all** of the following hold:

1. `(*SessionManager).Recover` no longer calls `store.Delete(id)` for missing-tmux orphans. `grep -n 'store.Delete' scenarios/web-console/api/session.go` must show zero call sites in `Recover()`. Confirmed by automated test `TestRecover_PreservesOrphanMetadata`.
2. `sessions` schema includes `status`, `agent_type`, `launch_command`, `agent_session_id`, `cwd`, `last_rollout_path`, `last_activity_at`, `orphaned_at`. Migration is idempotent.
3. `web-console session list-recoverable` and `web-console session recover <id>` exist and pass their automated tests.
4. End-to-end smoke `scenarios/web-console/scripts/crash-recovery-smoke.sh` passes twice in a row from a clean state.
5. `vrooli scenario restart web-console` succeeds; subsequent `web-console session list` is non-empty for any pre-existing live persistent sessions.
6. UI banner appears for recoverable sessions and one click re-attaches a real codex history (covered by `RecoverableSessionsBanner.test.tsx` + the e2e test).
7. `docs/guides/SESSION_RECOVERY.md` opens with the CLI flow; the manual procedure is the appendix.
8. `golangci-lint run` and `gofumpt -l` clean across `api/` and `cli/`.
9. Greenfield rule honored: no `supportsMessagesView` column, no migration shims, no compat flags.
10. `git log --oneline` shows commits scoped per phase, in dependency order, with messages tied to this plan's filename.

When all ten are checked, the plan can be archived under `scenarios/web-console/docs/plans/archive/`.
