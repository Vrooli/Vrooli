# Terminal Session Refactor — Implementation Plan

## 1. Purpose

Rebuild the web-console terminal-session layer end-to-end so that it is
**correct, decomposed, tested, and professional** — and in doing so close
three long-recurring user-visible bugs at their root:

- **Bug A — Persistent-mode input loss.** Messages submitted via the
  mobile toolbar, xterm keystrokes, or paste are sometimes swallowed.
  Pressing Ctrl+C "unblocks" further input. Root cause (confirmed):
  `tmuxPTY.Write` writes straight to the `tmux attach-session` PTY
  master (`api/pty_tmux.go:60–68`), so any bytes sent while tmux is in
  a client-interpreted mode (copy-mode, command-prompt, menu,
  prefix-pending) are consumed by tmux as client commands instead of
  being delivered to the pane. The server has no way to see this: the
  kernel accepts the bytes, `stdin_ack.ok=true` is emitted, the UI
  clears the draft, and the user's message is gone.
- **Bug B — Right-click paste unreliable in persistent mode.** The
  context menu's paste path (`ui/src/components/TerminalContextMenu.tsx:33–44`
  → `TerminalPane.tsx:425–429` `handleCtxPaste`) fires-and-forgets into
  the input gate and closes the menu immediately. There is no visible
  feedback when the gate queues or when the resulting `stdin_ack`
  fails. Combined with Bug A's mode-blindness, pastes silently vanish.
- **Bug C — Scrollback duplication after refresh / reconnect.** In both
  persistent and non-persistent modes, reconnecting with a live
  `sessionStorage` cache duplicates the trailing bytes of output.
  Root cause (confirmed): `totalBytesRef` in
  `ui/src/hooks/terminal/useTerminalSession.ts:141,327–328` is only
  updated from `history_end` messages, never from live `stdout`
  frames. If the cache is saved while a session is streaming (after
  `history_end` has fired), the cached `totalBytes` is stale relative
  to the serialized xterm state. On restore, the server resumes from
  the stale offset and replays bytes the xterm cache already contains.

This plan is **greenfield**: replaced files are deleted in the same
commit, no feature flags, no fallback paths, no `// removed` stubs, no
dual implementations. The end state is a smaller, seam-first layer with
automated regression tests that will fail if any of the three bugs
reappears.

## 2. Greenfield Constraint (HARD RULE)

**Do not ship compatibility shims, feature flags, legacy branches,
deprecated re-exports, or dead-code placeholders.**

Concretely forbidden:

- `if (newTmuxInput) { … } else { p.ptmx.Write(…) }` toggles.
- Keeping `useTerminalSocket.ts` in place beside the decomposed hooks.
- `// TODO: remove after N releases` or `_unused` renames of dead
  files.
- Dual paste handlers (one direct to `ws.send`, one through the gate).
- Retaining the stale-`totalBytesRef` update path "just in case."

When a file is replaced, its old path is deleted in the same change.
When a constant or function moves, every call site is updated. The
greenfield assertions test suite (§10.4) is extended to keep
regressions compile-time/test-time failures — not code-review
discussions.

The `docs/plans/terminal-session-rework-implementation-plan.md` and
`terminal-session-rework-phase-2-implementation-plan.md` documents that
previously held partial designs for this work have been deleted. The
stale reference in `api/greenfield_assertions_test.go` (comment at the
top of `TestGreenfield_NoRawSetSizeOutsideGatedPaths`) is updated in
Phase 0 to point at this plan.

## 3. Required Reading

A future agent resuming this plan MUST run, in order:

```bash
prompt-manager skill read implementation-plan-authoring scientific-debugging
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read refactor-scope screaming-architecture-audit signal-and-feedback-surface-design error-semantics-recovery-path-design idempotency-replay-safety-hardening
```

Rationale for each added skill:

| Skill | Why it's load-bearing here |
|---|---|
| `refactor-scope` | Keeps the split within "rework without redesign" — we are not changing features, only shape. |
| `screaming-architecture-audit` | The user explicitly asked for "screaming architecture" — the layer must name its concerns loudly (`broadcast.go`, `history_store.go`, `useHistoryReplay`). |
| `signal-and-feedback-surface-design` | Bug B is a pure feedback-surface gap. Context-menu paste must surface "pending → sent → ok/failed". |
| `error-semantics-recovery-path-design` | `stdin_ack.ok=false` must carry a typed reason and the UI must display it. Tmux command failures must be typed errors, not silent `Write` returns. |
| `idempotency-replay-safety-hardening` | Bug C is a replay-safety failure. The cache offset and the xterm buffer state must be atomically consistent. |

Also read before coding:

- `scenarios/web-console/docs/concepts/ARCHITECTURE.md` §`terminal-io`,
  §`terminal-history-caching`, §`data-flow`.
- `scenarios/web-console/docs/internal/SEAMS.md` §3 (session
  lifecycle, WS seam, toolbar keys).
- `scenarios/web-console/docs/internal/ERROR-SEMANTICS.md` §`sync-warning-coalescing`.
- `scenarios/web-console/api/greenfield_assertions_test.go` — pattern
  for static enforcement of architectural invariants.

## 4. Problem Statement

### 4.1 Current shape (measured)

| File | LOC | Concerns mixed |
|---|---|---|
| `api/session.go` | 1537 | Session struct + PTY lifecycle + broadcast + deliver + coalesce + pending-trim + SIGWINCH recovery + alt-buffer gating + UTF-8 splitting + history replay + tmux re-attach retry + policy + exit signaling. "God object". |
| `api/terminal_ws.go` | 410 | Upgrade + subscribe glue + output forwarder + input loop + keepalive + metrics + WS error protocol. |
| `api/pty_tmux.go` | 405 | Factory + Read + Write + ProbeReady + SetSize + Close + Kill + env filtering. `Write` is mode-blind. |
| `ui/src/hooks/terminal/useTerminalSession.ts` | 493 | Transport composition + session_ready gate + history replay buffer + pty_state → local echo toggle + conversation side-channel + debug probe + xterm.onData wiring + exit/error rendering. |
| `ui/src/components/TerminalPane.tsx` | 773 | Terminal host + context menu + touch + fit + drag-drop + file upload + paste capture + cache restore + handlers galore. |

### 4.2 Bug A — Persistent-mode input swallowed

1. `api/pty_tmux.go:60–68`: `Write(buf) { return p.ptmx.Write(buf) }`.
   No call to `tmux send-keys`, `paste-buffer`, `load-buffer`.
2. The `tmux attach-session` process interprets its stdin as **client
   input**. When the attached client is in copy-mode / command-prompt /
   menu / prefix-pending, bytes are consumed as client commands and
   never reach the pane's program.
3. `grep -rn "send-keys\|send_keys\|paste-buffer\|load-buffer" api/`
   returns zero hits — the server has never used the mode-safe input
   path tmux provides.
4. `stdin_ack` keys on `sess.Write()` returning a non-error, which is
   unrelated to whether tmux delivered the bytes to the pane. The UI
   clears the draft; the user's input is lost.

### 4.3 Bug B — Context-menu paste silent failure

1. `TerminalContextMenu.tsx:33–44` calls `onPaste(text)` then `onClose()`
   immediately.
2. `TerminalPane.tsx:425–429` `handleCtxPaste` calls `submitInput(text,
   "paste")` and drops the `GateResult`.
3. The gate legitimately returns `queued` when `session_ready` is
   false or when xterm is in mouse-tracking mode
   (`inputGate.ts:113–120`). No UI signal surfaces.
4. The gate has no insight into tmux's client mode (Bug A), so even a
   `sent` result can land on a tmux client in copy-mode and be eaten.

### 4.4 Bug C — Scrollback duplication on reconnect

1. `useTerminalSession.ts:141` declares `totalBytesRef = useRef(0)`.
2. `useTerminalSession.ts:327–328` updates it only in the
   `history_end` branch:
   `if (msg.total_bytes !== undefined) totalBytesRef.current = msg.total_bytes;`.
3. Live `stdout` messages write to xterm (line 315) but never
   increment `totalBytesRef`.
4. `ui/src/lib/terminalCache.ts:13–20` persists
   `{serialized, totalBytes, savedAt}` to sessionStorage. The caller
   (in `TerminalPane.tsx`) uses `totalBytesRef.current` for
   `totalBytes`. That value drifts behind the serialized xterm state as
   soon as the first live frame arrives post-history.
5. On reload, `buildSessionWsUrl` query param `history_offset=totalBytes`
   tells the server to resume from the stale offset. `Session.Subscribe`
   (`api/session.go:216–293`) sends the delta after the offset. The
   trailing bytes are written to xterm a second time on top of the
   already-restored serialized state.

### 4.5 Secondary gaps uncovered during analysis

- `session.go`'s `maybeSIGWINCHRecovery` (lines 597–620) is the only
  clean seam; the rest of the 1537-line file is unstructured.
  `TestGreenfield_NoRawSetSizeOutsideGatedPaths` exists but only for
  that one invariant.
- `useTerminalSession.ts` owns an in-hook global window probe
  (`appendOutputProbe`, lines 41–51) that mixes observability with
  protocol handling.
- The `conversation_event` / `conversation_event_update` / TTS paths
  are entangled with terminal orchestration; this plan treats them as
  a side-channel and extracts them intact.

## 5. Scope

### 5.1 In scope

- `api/session.go` — split into `session.go`, `broadcast.go`,
  `history_store.go`. Keep `session_paths.go`, `session_policy.go`,
  `session_store.go`, `session_handlers.go`, `session_defaults_handler.go`
  as-is (clean).
- `api/terminal_ws.go` — split into `terminal_ws.go` (upgrade + handler
  glue), `terminal_ws_input.go`, `terminal_ws_output.go`.
- `api/pty.go` + `api/pty_tmux.go` — add a `WriteInput(data []byte, kind
  InputKind) error` method on the `PTY` interface. `realPTY` delegates to
  `os.File.Write`. `tmuxPTY` implements via `tmux send-keys -l --` (for
  keystrokes) or `tmux load-buffer -` + `paste-buffer -d` (for pastes),
  both of which bypass client-mode interpretation. The existing `Write`
  method is removed from the interface; call sites updated.
- `ui/src/hooks/terminal/useTerminalSession.ts` — decompose into
  `useSessionReady`, `useHistoryReplay`, `useAltBufferState`,
  `useConversationChannel`, `useTerminalCache`. Keep
  `useTerminalTransport` and `useStdinAck` as-is (clean).
- `ui/src/lib/terminalCache.ts` — move cache save/load scheduling into
  `useTerminalCache`; count live bytes correctly.
- `ui/src/components/TerminalContextMenu.tsx` +
  `ui/src/components/TerminalPane.tsx` (`handleCtxPaste`) — integrate
  with `subscribeInputSettled` so paste shows "Pasting…", "Pasted",
  or "Paste failed: <reason>" states before the menu closes.
- `api/terminal_ws.go` message types — add `InputKind` discriminator
  on `stdin` frames (new wire field `kind?: "keystroke" | "paste"`),
  extend `stdin_ack` with typed `reason?: string`. Extend
  `TerminalMessage` in `ui/src/types/terminal.ts`.
- Tests: unit and integration coverage for every new file. Regression
  tests that fail if Bug A, B, or C recurs. Greenfield assertion tests
  that fail if a reviewer re-introduces `ptmx.Write` for tmux input,
  an un-decremented `totalBytesRef`, or a second paste handler
  bypassing the gate.
- Documentation: `docs/concepts/ARCHITECTURE.md` terminal sections
  rewritten to reflect the new shape; `docs/internal/SEAMS.md` updated;
  new `docs/internal/TERMINAL-INPUT-PROTOCOL.md` describes
  `InputKind`, ack semantics, and client-side feedback.

### 5.2 Out of scope

- Changing session persistence / detachable-session lifecycle (the
  SQLite metadata + tmux-backed recovery stays as-is).
- Voice-mode pipeline (`persistent-voice-mode`), speaker verification,
  TTS pre-cache. Conversation side-channel is **extracted but not
  redesigned**.
- Local-echo algorithm redesign (`LocalEchoController` stays; it is
  already isolated).
- xterm.js version upgrade or virtualization changes.
- WS protocol renames. We add fields; we do not rename existing ones.
- Mobile toolbar visual redesign. Its wiring changes (pending pill,
  settled feedback) but layout does not.
- Agent-manager integration. Terminal is scoped to web-console.

## 6. Current Technical Context

| File | Role | Key lines |
|---|---|---|
| `api/session.go` | Session struct + Subscribe + broadcast + deliver + FlushPending + maybeSIGWINCHRecovery + readLoop | 99–169, 216–293, 460–513, 515–549, 563–620, 622–768 |
| `api/pty.go` | PTY interface + realPTY (sync) | 54–114, 141–159 |
| `api/pty_tmux.go` | tmuxPTY factory + Write (mode-blind) + ProbeReady | 39–48, 60–68, 70–104, 239–324 |
| `api/pty_state.go` | Alt-buffer CSI tracker | 11–141 (keep as-is) |
| `api/ansi_responder.go` | Server-side DA1/DSR replies | entire (keep as-is) |
| `api/terminal_ws.go` | WS upgrade + input/output loops | 30–66 (types), 116–410 |
| `api/greenfield_assertions_test.go` | Static invariants | comment at top, `TestGreenfield_NoRawSetSizeOutsideGatedPaths` |
| `ui/src/hooks/terminal/useTerminalSession.ts` | Protocol orchestrator | 130–493 (entire hook) |
| `ui/src/hooks/terminal/useStdinAck.ts` | Seq/ack/pending queue | entire (keep as-is) |
| `ui/src/hooks/terminal/useTerminalTransport.ts` | Raw WS + reconnect | entire (keep as-is) |
| `ui/src/components/terminal/inputGate.ts` | Pure input decision layer | entire (keep as-is) |
| `ui/src/components/TerminalPane.tsx` | Xterm host + paste + context menu | 400–459, 463–475, 710–769 |
| `ui/src/components/TerminalContextMenu.tsx` | Right-click menu | 20–104 |
| `ui/src/lib/terminalCache.ts` | Cache save/load | entire |
| `ui/src/types/terminal.ts` | `TerminalMessage` union | (referenced from hook) |

## 7. Target End State

### 7.1 Screaming architecture (what the directory shouts at you)

```
api/
  session.go              <400 LOC. Session struct, lifecycle, policy glue.
  broadcast.go            Fan-out, deliver, coalesce, pending-trim, SIGWINCH gate.
  history_store.go        Bounded history ring + per-subscriber offset helpers.
  pty.go                  PTY interface (with WriteInput), realPTY impl.
  pty_tmux.go             tmuxPTY impl with mode-safe WriteInput.
  pty_state.go            Alt-buffer CSI tracker (unchanged).
  ansi_responder.go       Server-side DA/DSR replies (unchanged).
  terminal_ws.go          WS upgrade + handler composition.
  terminal_ws_input.go    Client → server input loop.
  terminal_ws_output.go   Server → client output forwarder + keepalive.
  greenfield_assertions_test.go  Static invariants (extended).

ui/src/hooks/terminal/
  useTerminalSession.ts     ~150 LOC. Composition only.
  useTerminalTransport.ts   WS lifecycle (unchanged).
  useStdinAck.ts            Seq/ack/pending queue (unchanged).
  useSessionReady.ts        Tracks session_ready + wsGen at ready.
  useHistoryReplay.ts       history buffer + replay flush + live byte counter.
  useAltBufferState.ts      pty_state → localEcho enable/disable.
  useConversationChannel.ts conversation_event / _update / ack.
  useTerminalCache.ts       Cache save/load, driven by live byte counter.

ui/src/components/terminal/
  inputGate.ts             Pure decision layer (unchanged).
```

### 7.2 Behavioral end state

- **Bug A gone.** Persistent-mode input reaches the pane regardless of
  tmux client mode. `stdin_ack.ok=false, reason="tmux_write_failed"`
  surfaces any real failure; UI preserves the draft.
- **Bug B gone.** Context menu shows `Pasting…` until `stdin_ack` is
  received. On `ok=true` it closes with a `Pasted` flash; on
  `ok=false` it shows the reason for 3 s. Paste goes through
  `submitInput` exclusively.
- **Bug C gone.** `totalBytesRef` advances on every `stdout` frame.
  The cache save ALWAYS writes a `totalBytes` consistent with the
  serialized xterm state. Reload/reconnect never duplicates bytes.
- **Screaming architecture.** Each file names a single concern. The
  longest file in the refactored set is under 500 LOC.
- **High coverage.** Every new file has direct unit tests. Every
  regression has a named test that would have caught it.
- **Professional tone.** No dead code, no `// removed`, no dual
  handlers, no feature flags, no "legacy" paths. The greenfield
  assertion suite fails at CI time if these sneak back.

### 7.3 Wire protocol deltas (additive only)

- `stdin` frames gain `kind?: "keystroke" | "paste"`. Default
  `"keystroke"`. Pastes originate exclusively from the paste path.
- `stdin_ack` frames gain `reason?: string`. Populated when `ok=false`
  with a typed value (e.g., `"tmux_write_failed"`, `"pty_closed"`,
  `"not_ready"`).
- No removals, no renames.

## 8. Contract Decisions

### 8.1 `InputKind` (Go + TS)

```go
// api/pty.go
type InputKind uint8

const (
    InputKindKeystroke InputKind = iota
    InputKindPaste
)

type PTY interface {
    Read(buf []byte) (int, error)
    WriteInput(data []byte, kind InputKind) error
    SetSize(cols, rows uint16) error
    Close() error
    Kill() error
    ProbeReady(ctx context.Context) error
}
```

- `realPTY.WriteInput` delegates to `ptmx.Write(data)` regardless of
  `kind`.
- `tmuxPTY.WriteInput`:
  - `InputKindKeystroke`: invokes
    `tmux send-keys -t <session> -l -- <data>` (literal mode —
    bypasses key-name lookup AND client modes; bytes land directly
    in the pane's stdin).
  - `InputKindPaste`: invokes
    `tmux load-buffer -b <buf> - < data` followed by
    `tmux paste-buffer -d -b <buf> -t <session>`. The `-d` flag
    deletes the buffer after paste. `paste-buffer` auto-cancels
    copy-mode before delivery.
  - Returns a typed error; the WS input handler converts to
    `stdin_ack.ok=false, reason="tmux_write_failed"`.
- The old `PTY.Write` method is removed. All call sites in
  `session.go`, `broadcast.go`, `terminal_ws_input.go` use
  `WriteInput`.

### 8.2 `stdin` / `stdin_ack` (wire)

```ts
// ui/src/types/terminal.ts
type StdinMessage = {
  type: "stdin";
  data: string;
  seq?: number;
  kind?: "keystroke" | "paste"; // default "keystroke"
};

type StdinAckMessage = {
  type: "stdin_ack";
  seq: number;
  ok: boolean;
  reason?:
    | "tmux_write_failed"
    | "pty_closed"
    | "not_ready"
    | "invalid_input";
};
```

### 8.3 History-store cursor API (Go)

```go
// api/history_store.go
type HistoryStore struct { /* ring buffer + startOffset + totalBytes */ }
func (h *HistoryStore) Append(data []byte)
func (h *HistoryStore) SnapshotFrom(offset int64) (data []byte, resumed bool, total int64)
func (h *HistoryStore) TotalBytes() int64
func (h *HistoryStore) StartOffset() int64
```

- `SnapshotFrom(0)` returns full history with `resumed=false`.
- `SnapshotFrom(off)` with `off >= StartOffset() && off <= TotalBytes()`
  returns delta with `resumed=true`.
- Out-of-range offsets return full history with `resumed=false`.
- Subscribe calls `SnapshotFrom` exactly once; appends continue via
  live broadcast.

### 8.4 `useTerminalCache` byte-accounting invariant

- `liveBytesRef.current` is initialized from `history_end.total_bytes`.
- Every `stdout` message (both replay and live) adds `msg.data.length`
  (byte length, not code-point length; data is UTF-8 JSON-decoded)
  to `liveBytesRef.current`.
- Cache saves always use `liveBytesRef.current` as `totalBytes`. Never
  `totalBytesRef` from another source.
- `history_end.total_bytes` on a later replay reconciles the counter
  (assert equal; warn in dev if drift detected).

### 8.5 Context-menu paste settlement

- `handleCtxPaste` resolves to an `await`-able promise via
  `subscribeInputSettled`. Menu stays open, button changes to
  `Pasting…` with a spinner. Resolution:
  - `ok=true` → flash `Pasted` for 600ms then close.
  - `ok=false` → show `Paste failed: <reason>` for 3s, keep menu open,
    keep clipboard text in the toolbar draft so the user can retry.
  - Timeout (5s) → same as `ok=false` with `reason="timeout"`.

## 9. Implementation Strategy

Each phase is independently testable and landable. Do each fully
(code + tests + typecheck + lint) before starting the next. No phase
restarts the running web-console; write code to disk and let the user
restart manually.

### Phase 0 — Prep + dead-code purge (½ day)

- Update the comment at the top of `api/greenfield_assertions_test.go`
  to reference this plan (`docs/plans/terminal-session-refactor-implementation-plan.md`).
- Delete the stale `docs/plans/terminal-session-rework-*` files
  (already done at plan-authoring time; confirm no dangling references
  in code or docs).
- Ensure `messages-pane-overhaul-plan.md`, `persistent-voice-mode-*`,
  `speaker-verification-*`, `tts-audio-precache-*` plans have no
  references to the removed plans; fix if any.
- Exit criteria: `rg "terminal-session-rework" scenarios/web-console/`
  returns zero hits. Typecheck + tests still green.

### Phase 1 — Backend seams (2 days)

1. Introduce `InputKind` enum in `api/pty.go`.
2. Add `PTY.WriteInput(data []byte, kind InputKind) error` to the
   interface. Remove `PTY.Write`.
3. Implement `realPTY.WriteInput` (trivial delegation to the PTY
   master).
4. Implement `tmuxPTY.WriteInput`:
   - Keystroke: shell out `tmux send-keys -t <session> -l --
     "<bytes>"` via existing `tmuxCmd` helper. Use `-- ` to guard
     against leading-dash payloads.
   - Paste: pipe bytes via `tmux load-buffer -b wc-paste- <session>
     -` stdin, then `tmux paste-buffer -d -b wc-paste-<session> -t
     <session>`. Buffer name is per-session to avoid cross-session
     collisions.
   - Both paths return a typed error wrapping stderr and the command
     name.
5. Update `session.go` / future `broadcast.go` / `terminal_ws_input.go`
   call sites to use `WriteInput`.
6. Add `api/pty_tmux_input_mode_test.go` — real tmux spawn (like
   existing `pty_tmux_test.go`), enter copy-mode, send keystroke,
   assert it reaches the pane (via a scripted shell echo) instead of
   being interpreted as a copy-mode motion.
7. Extend `TerminalMessage` in `terminal_ws.go` with `Kind string
   json:"kind,omitempty"` and `Reason string json:"reason,omitempty"`.
   Update `stdin_ack` emission to populate `Reason` on failures.
8. Add greenfield assertion:
   `TestGreenfield_NoRawPtmxWriteOutsideFactory` — regex-grep `api/`
   for `ptmx.Write(` and `.Write(data []byte)` on PTY values outside
   `pty.go`/`pty_tmux.go`.

**Bug A is fixed at the end of Phase 1.**

### Phase 2 — Session decomposition (3 days)

1. Extract `api/broadcast.go` from `session.go`: `broadcast`,
   `deliver`, `FlushPending`, `notifyIfThreshold`,
   `maybeSIGWINCHRecovery`, `snapToCleanBoundary`, `ClientInfo`. The
   receiver becomes a new `*Broadcaster` type that holds client map,
   history store pointer, alt-buffer tracker pointer, and the
   SIGWINCH cooldown clock. `Session` composes it.
2. Extract `api/history_store.go` from `session.go`: `outputHistory`,
   `totalOutputBytes`, `historyStart`, `appendHistory`, SGR-reset
   concerns. Replace ad-hoc slice manipulation in `Subscribe` with
   `HistoryStore.SnapshotFrom`.
3. `session.go` becomes ~350 LOC: Session struct, Create/Close/Policy,
   `Subscribe` (thin wrapper delegating to Broadcaster +
   HistoryStore), `Resize`, `readLoop` (simplified — delegates all
   broadcast to `Broadcaster.Ingest`).
4. Update every test file that reaches into session internals:
   `session_test.go`, `session_altbuf_test.go`,
   `session_recovery_test.go`, `session_policy_test.go`,
   `session_handlers_test.go` are updated in place; no dead test
   files left behind.
5. Add `broadcast_test.go` unit tests for: coalesce under
   backpressure, trim + pendingTrimmed flag, SIGWINCH gating in
   alt-buffer, StateCh transitions.
6. Add `history_store_test.go` unit tests for: ring wrap, start
   offset advance, full-history vs. delta snapshot, out-of-range
   offset falls back to full.
7. Extend `TestGreenfield_NoRawSetSizeOutsideGatedPaths` to cover the
   new `broadcast.go` location (enclosing func check updated).

### Phase 3 — WS handler decomposition (1 day)

1. Split `terminal_ws.go`: keep upgrade + handler glue only
   (~120 LOC). Extract `terminal_ws_input.go` (stdin reader loop,
   `stdin_ack` emit, kind-aware dispatch to `WriteInput`). Extract
   `terminal_ws_output.go` (output forwarder goroutine, keepalive,
   sync_warning emission, history_end sentinel).
2. Delete the merged-in input/output loop code from
   `terminal_ws.go`.
3. Existing `terminal_ws_test.go` splits into
   `terminal_ws_input_test.go` + `terminal_ws_output_test.go`; no new
   test file without moving at least one existing test into it.
4. Add `TestGreenfield_TerminalWsFilesSingleResponsibility` — static
   assertion that `terminal_ws.go` contains no goroutine launch and
   no `case "stdin":` string.

### Phase 4 — Frontend decomposition (2 days)

1. Create `useSessionReady.ts`: owns `sessionReadyRef`,
   `wsGenAtReadyRef`, consumes `session_ready` messages, exposes
   `isSessionReady()` and `currentReadyGen()`.
2. Create `useHistoryReplay.ts`: owns `replayingHistoryRef`,
   `historyBufferRef`, `historyTimeoutIdRef`, `liveBytesRef`.
   Subscribes to `stdout` and `history_end` (routed through the
   composition hook's dispatcher); exposes `liveBytesRef`,
   `flushHistoryBuffer()`, `isReplaying()`. **Increments
   `liveBytesRef` on every `stdout` message.** Reconciles with
   `history_end.total_bytes` and logs (not throws) on drift in dev.
3. Create `useAltBufferState.ts`: consumes `pty_state` messages,
   toggles `LocalEchoController.enabled`, exposes `isAltBuffer()`.
4. Create `useConversationChannel.ts`: consumes `conversation_event`
   and `conversation_event_update`, dispatches to callbacks, sends
   `conversation_event_ack`.
5. Create `useTerminalCache.ts`: owns save/load scheduling. Saves on
   debounce (500ms after last stdout) and on visibility hidden /
   unmount. Uses `liveBytesRef` from `useHistoryReplay` for the
   `totalBytes` field. On mount, loads cached entry and exposes
   `initialSerialized` + `historyOffset` to the terminal host.
6. Rewrite `useTerminalSession.ts` to ~150 LOC composition:
   - Instantiates the component hooks.
   - Wires transport subscribe → per-hook dispatcher (a single switch
     that fans out to each component hook's `handleMessage`).
   - Composes the input gate with `stdin.send` / `stdin.enqueue`.
   - Returns the public `UseTerminalSessionResult`.
7. Delete any code paths in the old `useTerminalSession.ts` not moved
   to a component hook. The `appendOutputProbe` debug stuff moves
   into `useHistoryReplay.ts` (where `stdout` is handled).
8. Add direct unit tests:
   `useSessionReady.test.ts`, `useHistoryReplay.test.ts`,
   `useAltBufferState.test.ts`, `useConversationChannel.test.ts`,
   `useTerminalCache.test.ts`. Cover: byte counting across many
   stdout frames, save-on-hidden, restore-on-mount, stale-cache
   invalidation when server's `start_offset` exceeds the cached
   offset.

**Bug C is fixed at the end of Phase 4.**

### Phase 5 — Feedback surfaces + Bug B polish (1 day)

1. `TerminalContextMenu.tsx`: keep menu open during paste; show
   `Pasting…` / `Pasted` / `Paste failed: <reason>` states. Use
   `subscribeInputSettled` via a new `onPasteSettled(result)` prop.
2. `TerminalPane.tsx` `handleCtxPaste`: wire `submitInput` result
   into a pending-paste map keyed by `seq`. On settlement, resolve.
3. Add pending-input pill near the mobile toolbar driven by
   `getPendingInputSnapshot()`. Shows `N unsent`; on tap, lists the
   age of each pending entry. Remove on empty.
4. Tests:
   - `TerminalContextMenu.test.tsx`: paste UI states (mocked gate
     returns queued → sent → ack ok/failure/timeout).
   - `mobile-toolbar-pending-pill.test.tsx`: pill appears/disappears
     with pending-input changes.

**Bug B is fixed at the end of Phase 5.**

### Phase 6 — Documentation + screaming architecture sweep (½ day)

1. Rewrite `docs/concepts/ARCHITECTURE.md` terminal sections. New
   headings mirror file names: `Session + Broadcaster + HistoryStore`,
   `PTY Backends (real vs tmux)`, `WS I/O Split`, `Client Hook Set`,
   `Cache Consistency Invariant`.
2. Update `docs/internal/SEAMS.md` §3 with the new file layout.
3. Add `docs/internal/TERMINAL-INPUT-PROTOCOL.md`: describes
   `InputKind`, `stdin_ack.reason` values, context-menu settlement
   contract.
4. Run `prompt-manager skill read screaming-architecture-audit`
   checklist across the refactored surface; capture any last
   renames.

## 10. Testing Plan

### 10.1 New Go tests (all automated, no manual steps)

| Test file | Covers |
|---|---|
| `api/pty_tmux_input_mode_test.go` | Spawns real tmux, enters copy-mode, sends keystroke and paste via `WriteInput`, asserts both reach the pane (regression for Bug A). |
| `api/broadcast_test.go` | Coalesce, deliver, FlushPending, SIGWINCH gating, StateCh transitions. |
| `api/history_store_test.go` | Ring wrap, start-offset advance, full vs. delta snapshot, out-of-range fallback. |
| `api/terminal_ws_input_test.go` | Stdin dispatch, `kind` routing, `stdin_ack.ok/false/reason` on tmux command failure. |
| `api/terminal_ws_output_test.go` | Output forwarder, `history_end` sentinel, keepalive, sync_warning. |

### 10.2 New TS tests

| Test file | Covers |
|---|---|
| `__tests__/useSessionReady.test.ts` | session_ready gate, wsGen at ready. |
| `__tests__/useHistoryReplay.test.ts` | history buffering, liveBytesRef increment on every stdout, reconciliation on history_end (regression for Bug C). |
| `__tests__/useAltBufferState.test.ts` | pty_state handling, localEcho toggle. |
| `__tests__/useConversationChannel.test.ts` | event + update dispatch, ack send. |
| `__tests__/useTerminalCache.test.ts` | save-on-hidden, restore-on-mount, totalBytes equals liveBytesRef (Bug C), stale cache invalidation when server start_offset advances past cached offset. |
| `__tests__/TerminalContextMenu.test.tsx` (extend) | paste UI states: pending, sent, ok, failed, timeout (Bug B). |
| `__tests__/terminal-scrollback-dedup.test.tsx` | Integration: JSDOM + mock WS. Connect, live stdout, simulate unmount (save cache), remount, reconnect. Assert no byte duplication in xterm buffer. |

### 10.3 Extended existing tests

- `session_test.go`, `session_altbuf_test.go`,
  `session_recovery_test.go`, `session_policy_test.go`,
  `session_handlers_test.go`: updated in place to reference the new
  types. No dead test files.
- `pty_tmux_test.go`: extended with `InputKind` cases.
- `inputGate.test.ts`, `MobileToolbar.test.tsx`: unchanged except for
  any import-path updates.

### 10.4 Greenfield assertion tests (`api/greenfield_assertions_test.go`)

Extend with:

- `TestGreenfield_NoRawPtmxWriteOutsideFactory`: fails if any file
  outside `pty.go` or `pty_tmux.go` calls `ptmx.Write(` on a PTY
  value.
- `TestGreenfield_NoSecondPasteHandler`: fails if any UI file outside
  `TerminalContextMenu.tsx` or `TerminalPane.tsx`'s
  `handleCtxPaste` / `handlePaste` calls `submitInput(*, "paste")`
  OR if more than one `onPasteCapture` handler exists.
- `TestGreenfield_LiveBytesRefUpdatedOnStdout`: fails if
  `useHistoryReplay.ts` contains a `case "stdout"` branch that does
  not increment `liveBytesRef`.
- `TestGreenfield_NoReferencesToRemovedPlans`: fails if any file in
  `scenarios/web-console/` references the removed plan filenames.

### 10.5 Coverage targets

- `api/broadcast.go`, `api/history_store.go`: ≥ 90% line coverage.
- `api/pty_tmux.go` `WriteInput`: each `InputKind` branch covered.
- `ui/src/hooks/terminal/use*.ts` new hooks: ≥ 85% line coverage
  each.
- Regression tests (Bug A, B, C) must each fail on a single-line
  regression to the fixed code path, not just on a large
  refactoring accident.

### 10.6 CI wiring

All new tests go in existing test trees (`api/*_test.go` and
`ui/src/__tests__/`). Makefile targets are unchanged. No new CI jobs.

## 11. Rollout / Validation Checklist

- [ ] `cd scenarios/web-console/api && go build ./... && go test ./...
      -timeout 300s` passes.
- [ ] `cd scenarios/web-console/ui && npm run typecheck && npm test`
      passes.
- [ ] `rg "terminal-session-rework" scenarios/web-console/` returns
      zero hits.
- [ ] `rg "ptmx\.Write\(" scenarios/web-console/api/ | grep -v
      pty_tmux.go | grep -v pty.go` returns zero hits.
- [ ] `wc -l scenarios/web-console/api/session.go
      scenarios/web-console/ui/src/hooks/terminal/useTerminalSession.ts`
      both under 500 LOC.
- [ ] Greenfield assertion suite green (§10.4).
- [ ] The user restarts web-console manually (not from this agent)
      and confirms:
  - Tmux-in-copy-mode pane accepts toolbar input (Bug A).
  - Right-click paste in persistent mode reliably reaches the shell
    and shows settlement feedback (Bug B).
  - Reload-mid-session does not duplicate trailing bytes in
    scrollback (Bug C).
- [ ] Documentation updates landed: ARCHITECTURE.md, SEAMS.md,
      TERMINAL-INPUT-PROTOCOL.md.

## 12. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| `tmux send-keys -l --` has edge-case handling for NUL bytes or huge payloads. | Medium | `pty_tmux_input_mode_test.go` covers a 1 MB paste, a single-NUL paste, a payload containing `;` `$` `"` `\\`. For >256 KB pastes, use the `load-buffer`+`paste-buffer` path exclusively. |
| `tmux paste-buffer` leaves the per-session buffer around if the process dies mid-call. | Low | Use unique per-call buffer names (`wc-paste-<sessionid>-<seq>`) with `-d` (delete after paste). Add a janitor that clears buffers older than 1 min on session close. |
| Frontend hook decomposition introduces render-order bugs where a component hook sees a message before another hook has initialized its ref. | Medium | The composition hook builds hooks in a fixed order and subscribes to the transport AFTER all hooks have returned. Tests assert subscription ordering. |
| `liveBytesRef` byte counting drifts under JSON-decoding where the server sends UTF-8 escapes. | Low | Count bytes of `TextEncoder.encode(msg.data).byteLength`, not `.length`. Assert in `history_end` that server's `total_bytes` == client's `liveBytesRef` within a small tolerance; log in dev on drift. |
| The `conversation_event` side-channel has hidden coupling we miss during extraction. | Medium | Phase 4 extraction is mechanical; the dispatcher in `useTerminalSession.ts` maps message types to hooks by a 1:1 switch. `useConversationChannel.test.ts` reproduces the existing flow as-is. |
| Splitting `session.go` across 3 files introduces a cross-file mutex race. | Medium | All mutation goes through `Session.mu` which is passed to `Broadcaster` as a reference. `broadcast_test.go` includes a `go test -race` scenario. |
| User cannot restart web-console (agent runs inside it). | High | Every phase's output is a disk-state change only; the plan explicitly does not restart the running scenario. The user restarts manually per the memorized project convention. |

## 13. Non-Goals / Prohibited Patterns

Do not:

- Keep `useTerminalSocket.ts` in place (if any copy exists). The
  split hooks are the only client-side protocol surface.
- Add a `--legacy-tmux-write` CLI flag or env variable.
- Keep `ptmx.Write` as a convenience helper on `tmuxPTY`.
- Dual-write paste via both `onPasteCapture` and `handleCtxPaste`.
- Leave `TODO(remove after …)` comments in favor of deletion.
- Introduce a per-session "paste buffer pool" or similar complexity
  beyond what the `tmux load-buffer`/`paste-buffer` pair requires.
- Change the detachable-sessions persistence schema or SQLite layout.
- Migrate to a new WebSocket protocol version field.
- Add a UI toggle to "disable the mode-safe tmux path" — it is
  always on.
- Add "belt-and-suspenders" strips of ANSI sequences on the client
  for things the server already handles.
- Rename `InputKind` values to match OS-specific paste APIs.
- Couple the cache save cadence to a user setting. It is debounced
  at a fixed 500 ms.

## 14. Definition of Done

All of the following must be true:

1. `api/session.go` is under 500 LOC and names only lifecycle /
   composition concerns. `api/broadcast.go` and
   `api/history_store.go` exist and each names its single concern.
2. `api/pty.go` has `WriteInput(data []byte, kind InputKind) error`.
   `PTY.Write` is gone. `ptmx.Write(` appears only inside
   `pty.go` / `pty_tmux.go`.
3. `api/terminal_ws.go` contains only upgrade + handler glue.
   `terminal_ws_input.go` owns the input loop; `terminal_ws_output.go`
   owns the output forwarder.
4. `ui/src/hooks/terminal/useTerminalSession.ts` is under ~150 LOC
   and only composes component hooks. Each of
   `useSessionReady.ts`, `useHistoryReplay.ts`,
   `useAltBufferState.ts`, `useConversationChannel.ts`,
   `useTerminalCache.ts` exists and has a direct unit test file.
5. `liveBytesRef` is incremented in exactly one place
   (`useHistoryReplay.ts`'s `stdout` handler) and consumed in
   exactly one place (`useTerminalCache.ts`'s save path). A
   greenfield assertion enforces this.
6. Context-menu paste surfaces `Pasting… / Pasted / Paste failed:
   <reason>` states. No paste path bypasses `submitInput`.
7. Persistent-mode input reaches the pane regardless of tmux client
   mode; `pty_tmux_input_mode_test.go` spawns a real tmux and proves
   it.
8. Reloading mid-session does not duplicate trailing bytes;
   `terminal-scrollback-dedup.test.tsx` proves it.
9. `stdin_ack.reason` is populated on every non-ok ack. The UI
   displays it via a typed-error map (not a generic string blit).
10. Greenfield assertion suite (§10.4) is green. `rg
    "terminal-session-rework"` and `rg "ptmx\.Write\("` outside
    `pty*.go` return zero hits.
11. All existing tests that referenced the deleted plans or removed
    APIs are updated in place. No test file is left describing a
    removed behavior.
12. `docs/concepts/ARCHITECTURE.md` terminal sections rewritten;
    `docs/internal/SEAMS.md` updated; `docs/internal/TERMINAL-INPUT-PROTOCOL.md`
    created.
13. No compatibility shims, feature flags, or legacy branches exist
    in the diff. This condition is verified by re-reading §2 after
    the final commit and walking every new/changed file.
