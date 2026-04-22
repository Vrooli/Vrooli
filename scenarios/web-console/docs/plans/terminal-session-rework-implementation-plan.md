# Terminal Session Rework — Implementation Plan

## 1. Purpose

Rework the web-console terminal-session stack (both standard and persistent modes) to eliminate three classes of user-visible bugs that have recurred across multiple emergency patches (through `web-console fixes p4`) and to clean up the tech debt those patches accumulated. The goal is a sharp-edged, testable session layer where:

- Every input path shares one state-aware gate.
- Every source of terminal rendering knows whether we are in the alternate screen buffer.
- The hook that talks to the WebSocket is three small modules instead of one 754-line file.
- Defensive double-strips of ANSI sequences are removed; there is exactly one place each escape is handled.

This is greenfield work. No compat shims, no "legacy path" branches, no `// removed` stubs. When we replace a module, we delete the old one.

## 2. Greenfield Constraint (HARD RULE)

**This plan does not produce any compatibility layers, feature flags, fallback paths, or deprecated re-exports.**

Concretely this forbids:

- `if (newInputGate) { ... } else { oldPath() }` toggles.
- Keeping the client-side mode-2026 strip "just in case an old server sends it."
- Leaving `useTerminalSocket.ts` in place beside the new split hooks.
- Wrappers that forward to the new API so old imports still compile.

When a file is replaced, its old path is deleted in the same commit. When a constant moves, every call site is updated. Definition of Done (§13) re-asserts this.

## 3. Required Reading

Future agent resuming this plan must run, in order:

```bash
prompt-manager skill read implementation-plan-authoring scientific-debugging
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also read before coding:

- `scenarios/web-console/docs/concepts/ARCHITECTURE.md` (sections: "terminal-io", "terminal-history-caching")
- `scenarios/web-console/docs/internal/SEAMS.md` (session lifecycle, WS seam, toolbar keys)
- `scenarios/web-console/docs/internal/ERROR-SEMANTICS.md` (sync-warning coalescing, WS error protocol)
- `scenarios/web-console/docs/plans/persistent-terminal-input-protection-plan.md` — prior work in the same area; context on current seq/ack design.

## 4. Problem Statement

Three user-reported symptoms, all rooted in the same architectural gap (absent state-aware terminal layer):

### 4.1 Symptom A — "Paste is lost until I Ctrl+C"

Pastes submitted via the mobile toolbar Send button or xterm's own paste path sometimes appear not to arrive. Hitting Ctrl+C in the session "unblocks" them. Reproduced in practice repeatedly.

**Root cause (confirmed by code inspection):**
- `ui/src/hooks/useTerminalSocket.ts:279-291` (`trySendStdin`) checks only `sessionReadyRef.current` and `ws.readyState === OPEN`. It does not consult `terminal.modes` or any application-level state.
- `ui/src/components/MobileToolbar.tsx:215,255,285` calls `onInput` (→ `sendInput` → `trySendStdin`), bypassing xterm entirely.
- `ui/src/hooks/useTerminalSocket.ts:691-709` (xterm.onData) also calls `trySendStdin` directly.
- Bytes reach the PTY. If a TUI is holding stdin, the kernel buffers until the TUI reads again. There is no client-side signal, so the user perceives the paste as lost.

### 4.2 Symptom B — Alt-buffer output duplication ("footer duplication" screenshot)

When Claude Code (or any alt-buffer TUI) is active and output coalescing trims the buffer, the tmux status line is re-rendered inline with Claude Code's live output, producing the striped duplication pattern in the reported screenshot.

**Root cause (confirmed by code inspection):**
- `api/session.go:489-497` trims the coalesced pending buffer when over `offlineBufferMax` and sets `pendingTrimmed = true`.
- `api/session.go:553-560` fires `s.pty.SetSize(s.Cols, s.Rows)` (SIGWINCH) unconditionally to "recover" screen state — every time `pendingTrimmed` is set, no rate limit, no alt-buffer awareness.
- `grep -rn "1049\|alternate\|altBuffer" api/ ui/src/` returns one match: a comment at `api/session.go:556`. The code has never tracked alt-buffer state.
- SIGWINCH during live alt-buffer rendering makes tmux redraw its status bar while Claude Code is redrawing the pane, producing the striping.

### 4.3 Symptom C — Legacy input/rendering layer tech debt

Side effects of prior emergency fixes:

- `ui/src/hooks/useTerminalSocket.ts:504` strips `\x1b\[\?2026(?:[hl]|\$p)` from server data, even though `api/ansi_responder.go:114-130` already strips server-side. Comment in the file itself calls this "belt-and-suspenders."
- `ui/src/lib/localEcho.ts:98-111` writes `\b \b` sequences directly into the terminal on prediction mismatch. Visible flicker on fast-typing users.
- `ui/src/hooks/useTerminalSocket.ts:526` sets `hasCachedStateRef = false` after first `history_end`, so a cache-reset on stale offset can only fire once per hook lifetime.
- `useTerminalSocket.ts` is 754 LOC mixing WS transport, seq/ack, session handshake, history replay, and ANSI sanitization.

### 4.4 Symptom D — Secondary stdin double-send on reconnect mid-write

If `ws.onclose` fires while a PTY write is in flight, `clearPendingAcks(true)` (`useTerminalSocket.ts:353-367,606`) re-enqueues the payload. On reconnect's `session_ready`, `flushPendingInput` sends it again. The server's late ack for the first send is ignored (`useTerminalSocket.ts:481-485`). Net effect: same command executed twice.

## 5. Scope

### In scope

- `ui/src/hooks/useTerminalSocket.ts` — split into `useTerminalTransport`, `useStdinAck`, `useTerminalSession`.
- `ui/src/lib/localEcho.ts` — replace mismatch-erase behavior with prediction drop + reset; opt-out via session flag.
- `ui/src/components/MobileToolbar.tsx`, `TerminalPane.tsx`, `useTerminalTouch.ts` — route all input through the new input gate.
- `ui/src/components/terminal/` — new directory for the extracted modules.
- `api/session.go` — add alt-buffer tracking (CSI `?1049h/l` observer) and gate/rate-limit SIGWINCH recovery.
- `api/terminal_ws.go` — adjust to whatever contract changes result from the split (expected: none in wire protocol, but remove the `MsgTypeSessionReady`-gate duplication if both sides converge).
- `api/ansi_responder.go` — keep; client-side duplicate strip is deleted.
- Tests added for every behavior change; old tests updated in place (no dead tests).

### Out of scope

- Changing the WebSocket wire protocol (stdin/stdout/stdin_ack/session_ready message types stay as-is).
- Desktop-only UI concerns unrelated to sessions.
- Voice mode, TTS, conversation events — untouched.
- Persistent-mode server-side tmux management (ProbeReady, reattach retries) — stays as-is.
- Migration guides; there are no external consumers of these internal modules.

## 6. Current Technical Context

### 6.1 Frontend (UI) key files

| File | LOC | Role |
|---|---|---|
| `ui/src/components/TerminalPane.tsx` | 767 | xterm.js init, lifecycle, paste handler, cache restore |
| `ui/src/hooks/useTerminalSocket.ts` | 754 | WS, seq/ack, session_ready gate, history replay, sanitization |
| `ui/src/hooks/useTerminalTouch.ts` | 596 | gestures, mouse-mode routing |
| `ui/src/components/MobileToolbar.tsx` | ~500 | command bar, send status, settlement |
| `ui/src/lib/localEcho.ts` | 125 | predictive echo + reconciliation |

### 6.2 Backend (API) key files

| File | LOC | Role |
|---|---|---|
| `api/session.go` | ~900 | fanout, history buffer, coalescing, SIGWINCH recovery, readLoop |
| `api/terminal_ws.go` | 382 | WS upgrade, input loop, output forwarder, ProbeReady gate, session_ready emission |
| `api/pty.go` / `api/pty_tmux.go` | 250 / 400 | backend abstraction, ProbeReady impl |
| `api/ansi_responder.go` | 153 | server-side DA/DECRQM replies + sanitizeForClient |

### 6.3 Key invariants today

- Server emits `session_ready` after `ProbeReady` completes (timeout 3s).
- Client waits for `session_ready` before sending any `stdin`.
- Every stdin has a seq; every stdin gets a stdin_ack with ok.
- History replay: on connect, server streams buffered history up to current `total_bytes`, emits `history_end`, then live stdout. Client sends `history_offset=N` query param to resume from N.

### 6.4 What is NOT currently tracked

- Whether the PTY has entered the alternate screen buffer (CSI `?1049h`) and not yet exited it (CSI `?1049l`).
- Whether xterm.js is in any application-blocking UI state on the client.
- Whether the kernel PTY input buffer is draining (i.e., the foreground process is actively reading).

## 7. Target End State

After this plan:

1. A single `TerminalInputGate` in `ui/src/components/terminal/inputGate.ts` is the only path to `ws.send({type: "stdin", ...})`. Every input source (xterm.onData, MobileToolbar toolbar keys, MobileToolbar textarea submit, clipboard paste, key-combo picker, voice transcription insert, image-upload injected stdin) routes through it. The gate returns a typed result: `{status: "sent"} | {status: "queued", reason: "not-ready" | "ws-closed" | "paused-by-xterm-mode"} | {status: "rejected", reason: "disposed"}`. Callers that care (MobileToolbar) show status; callers that don't (xterm.onData) silently queue.

2. `useTerminalSocket.ts` is deleted. Replaced by three hooks in `ui/src/hooks/terminal/`:
   - `useTerminalTransport` — WebSocket only. Raw JSON frames in/out. ~150 LOC.
   - `useStdinAck` — seq allocation, ack tracking, re-enqueue on timeout, settlement subscribers. ~200 LOC.
   - `useTerminalSession` — orchestrator. session_ready gate, history replay, reconnection policy, cache offset. ~250 LOC. Uses the other two. Exposes: `{send, subscribeInputSettled, subscribePendingInput, getPendingSnapshot, connectionState}`.

3. `localEcho.ts` reconciliation no longer writes `\b \b` to the terminal. On mismatch, predictions are dropped and the server output is passed through unchanged. Local echo is disabled by default for any session that has ever entered the alt-buffer (alt-buffer signal propagates client-side via a new dedicated channel described in §9.5).

4. `api/session.go` tracks alt-buffer state per session by scanning outgoing bytes for `\x1b[?1049h` (enter) and `\x1b[?1049l` (exit). The SIGWINCH recovery in `FlushPending` (`session.go:553-560`) is replaced by: (a) never fire SIGWINCH while in alt buffer; (b) otherwise rate-limit to at most once per second; (c) only fire when trimming actually discarded bytes the client hadn't already received. The alt-buffer state is broadcast to clients via a new WS message `pty_state` so client-side decisions (local echo) can react.

5. Client-side duplicate strip of mode-2026 sequences at `useTerminalSocket.ts:504` is deleted. The server responder is the single source of truth.

6. `hasCachedStateRef` is replaced by a boolean stored per-connection, not per-hook-lifetime — so multiple reconnects each evaluate cache validity independently.

7. Stdin double-send on reconnect mid-write is prevented by a write barrier: the client does not re-enqueue a payload whose seq has a pending stdin_ack unless the reconnect establishes a fresh session (new WS generation). A per-connection `wsGen` counter distinguishes reconnect-to-same-session from reconnect-to-new-session.

8. Full observability: a single `__wc_terminal_debug` window object exposes `{gate: {...}, transport: {...}, pendingAcks, pendingInput, altBufferActive, coalesceEvents}` for manual repro and test introspection.

## 8. Implementation Strategy (Phased)

Each phase ends with a green test suite. Phases have explicit ordering: later phases depend on earlier phases' types/contracts.

### Phase 0 — Preconditions and probes

- [ ] Add `__wc_terminal_debug` shim in `ui/src/components/terminal/debug.ts`. Populated by later phases. Test: exists at module load.
- [ ] Add `api/session_altbuf_test.go` with a failing test asserting alt-buffer tracking on `\x1b[?1049h` / `\x1b[?1049l`. Implementation lands in Phase 2.
- [ ] Add a client-side integration test harness in `ui/src/__tests__/terminal-integration.test.tsx` that mounts `TerminalPane` with a fake WebSocket factory. This is the integration seam for all subsequent tests.

**Exit criteria:** New test harnesses exist and fail with clear messages.

### Phase 1 — Extract `TerminalInputGate`

- [ ] Create `ui/src/components/terminal/inputGate.ts`:
  - Interface: `createInputGate({send: (data: string) => SendResult, terminalRef: React.MutableRefObject<Terminal | null>})`.
  - Method: `submit(data: string, source: "xterm" | "toolbar-key" | "toolbar-submit" | "paste" | "voice" | "upload"): GateResult`.
  - `GateResult = {status: "sent", seq} | {status: "queued", reason} | {status: "rejected", reason}`.
  - Always rejects empty data.
  - Returns `queued:not-ready` when session not ready.
  - Returns `queued:paused-by-xterm-mode` when terminal is in any of: `modes.mouseTrackingMode !== "none"` and source is `"paste"` (so paste doesn't silently feed a mouse-tracking TUI), or when an externally-settable "paused" flag is on.
- [ ] Unit tests: `ui/src/__tests__/inputGate.test.ts` covers each source × each gate state.
- [ ] Wire MobileToolbar, xterm.onData, clipboard paste, voice insert, and image-upload inject through the gate. Delete their current direct calls to `sendInput`/`trySendStdin`.
- [ ] Update MobileToolbar's status UI to map `GateResult.reason` to distinct pill states (not-ready, paused, sending, queued, failed). Keep existing `sent` / `queued` / `failed` visible labels; reason shown as tooltip + `data-reason` attribute for tests.

**Exit criteria:**
- `grep -rn "trySendStdin\|sendInput(" ui/src/` returns only `inputGate.ts` and its tests.
- All existing MobileToolbar tests pass.
- New test: paste via MobileToolbar while `terminal.modes.mouseTrackingMode === "buttonEventTracking"` returns `queued:paused-by-xterm-mode` and the pill shows "Paused".

### Phase 2 — Alt-buffer tracking + SIGWINCH gating (server)

- [ ] Add `api/pty_state.go` with a `PTYStateTracker` type that scans bytes for `\x1b[?1049h` / `\x1b[?1049l` and exposes `IsAltBuffer() bool` + `LastTransitionAt() time.Time`. Unit-tested against ANSI fixtures including split-across-reads cases.
- [ ] Integrate into `Session`: `broadcast()` updates the tracker before `appendHistory`.
- [ ] Replace `api/session.go:553-560`:
  - Guard with `if !s.ptyState.IsAltBuffer() && timeSinceLastRecovery >= 1s { ... }`.
  - Track `lastRecoveryAt` per session.
  - Remove comment referring to "alternate screen buffer" recovery; add Why-comment citing this plan file.
- [ ] Add new WS message `MsgTypePTYState` (`type: "pty_state"`, `altBuffer: bool`). Emitted on every alt-buffer transition. Client uses this in Phase 3.
- [ ] Tests:
  - `api/session_altbuf_test.go` (failing in Phase 0) now passes.
  - `api/session_sigwinch_recovery_test.go` — verify SIGWINCH is suppressed during alt buffer, rate-limited elsewhere, skipped when no bytes were trimmed.
  - Update `api/session_test.go` to cover the new `pty_state` emission.

**Exit criteria:**
- Alt-buffer test suite green.
- `grep -n "SetSize.*Cols.*Rows" api/session.go` shows exactly one site, gated.

### Phase 3 — Split `useTerminalSocket` into three hooks

- [ ] Create `ui/src/hooks/terminal/useTerminalTransport.ts`. WebSocket open/close/reconnect + message bus. Exposes: `{sendJson, subscribe(type, cb), connectionState, wsGen}`.
- [ ] Create `ui/src/hooks/terminal/useStdinAck.ts`. Owns seq allocation, ack tracking, ack timeout, pending-input queue, inputSettled subscribers, pendingChanged subscribers. Takes `useTerminalTransport`'s `sendJson` and `wsGen`. Implements the write barrier described in §7.7: a stdin is re-enqueued on reconnect only if the new `wsGen !== generationAtSend`.
- [ ] Create `ui/src/hooks/terminal/useTerminalSession.ts`. Orchestrator. session_ready gate (ACKs from server), history replay buffer with `history_end`/`total_bytes`, cache reset on stale offset, alt-buffer state from `pty_state` messages. Exposes: `{gate: TerminalInputGate, subscribeInputSettled, subscribePendingInput, getPendingSnapshot, altBuffer, connectionState}`.
- [ ] Delete `ui/src/hooks/useTerminalSocket.ts` and update all importers (`TerminalPane.tsx`, tests). No re-export shim.
- [ ] Delete `useTerminalSocket.hook.test.ts` and re-author three focused test files (one per new hook), each smaller.
- [ ] Also delete the client-side mode-2026 strip in the stdout handler (was `useTerminalSocket.ts:504`). The new `useTerminalSession` does not re-introduce it. Add a test asserting the client does **not** strip — if a raw `\x1b[?2026$p` ever leaks past the server, the test fails loudly (regression surface, not silent compensation).

**Exit criteria:**
- `useTerminalSocket.ts` deleted; `find ui/src -name "useTerminalSocket*"` returns nothing.
- All three new hooks pass their unit tests.
- `TerminalPane.tsx` compiles with the new imports.
- `ui/src/__tests__/terminal-integration.test.tsx` exercises the full integrated stack with fake WS and passes.

### Phase 4 — Rebuild local echo

- [ ] Rewrite `ui/src/lib/localEcho.ts`:
  - On mismatch, drop all predictions and return unmodified server data. Never write `\b \b`.
  - Accept a `disabled` signal from `useTerminalSession` (set true whenever `altBuffer === true`; stays disabled until session replaces).
  - Keep MAX_PREDICTION_AGE_MS auto-reset and MAX_PENDING_PREDICTIONS cap.
- [ ] Update `localEcho.test.ts` with new mismatch behavior. Add tests for alt-buffer-disabled path.

**Exit criteria:**
- No `\b \b` in the source tree outside deliberate control-char tests.
- Local echo test suite green.
- Integration test: user types 5 chars, TUI enters alt buffer, subsequent user keystrokes are not predicted (no visible double-render).

### Phase 5 — Double-send on reconnect write barrier

- [ ] Implement the `wsGen` barrier in `useStdinAck`. Pending acks are tagged with their `wsGen`. On `ws.onclose`, re-enqueue only if the payload's `wsGen === currentWsGen` AND the close reason is abnormal. Payloads whose acks are legitimately in-flight on the same generation wait one more ack-timeout cycle.
- [ ] Test: simulate WS close mid-write, then reconnect, assert only one PTY write is observed by the fake server.
- [ ] Test: simulate WS close abnormally and a fresh `session_ready` (new generation); assert payload is re-sent exactly once.

**Exit criteria:**
- Double-send test passes.
- `api/terminal_ws_test.go` integration: with the real WS handler and a fake client that drops connection mid-write, the PTY receives the payload exactly once.

### Phase 6 — Observability + cleanup pass

- [ ] `__wc_terminal_debug` fully populated: gate decisions, ack map snapshot, pending queue, alt-buffer state, last 16 coalesce events with server timestamps.
- [ ] Delete any dead code flagged during the refactor (expected: unused types in old `useTerminalSocket.ts`, stale `WS_ERROR_RECOVERY` entries that no longer apply).
- [ ] Update `scenarios/web-console/docs/internal/SEAMS.md`: new seams for the gate and the three hooks.
- [ ] Update `scenarios/web-console/docs/concepts/ARCHITECTURE.md#terminal-io`: new architecture diagram, alt-buffer behavior, write-barrier rules.

**Exit criteria:**
- `grep -rn "TODO\|FIXME\|XXX\|HACK" ui/src/components/terminal/ ui/src/hooks/terminal/ api/session.go api/terminal_ws.go api/pty_state.go` returns nothing plan-related.
- Docs updated.

## 9. Contract Decisions

### 9.1 Input gate contract (client)

- Every stdin byte sent to the server passes through `TerminalInputGate.submit(data, source)`.
- Return type is a discriminated union; callers never receive `boolean`. This is why `onInput: (data: string) => boolean` in `MobileToolbar.tsx:46` changes to `onInput: (data: string) => GateResult`.
- Empty data is rejected with `{status: "rejected", reason: "empty"}`. Callers must not enqueue empties.

### 9.2 Stdin/stdin_ack (WS protocol)

- Unchanged on the wire. `{type: "stdin", data, seq}` → `{type: "stdin_ack", seq, ok}`.
- New invariant: a given `seq` on the server is answered with `stdin_ack` exactly once. Client tolerates duplicate acks (ignore unknown seq) but they must not occur in nominal operation.

### 9.3 session_ready (WS protocol)

- Emitted exactly once per WebSocket connection. Unchanged.
- After session_ready, the client's `wsGen` counter increments. This is the gate for re-enqueue decisions.

### 9.4 history_end (WS protocol)

- Unchanged. `total_bytes` is authoritative for client cache offset.
- Client invariant changes: `hasCachedStateRef` (per-hook) becomes `hasCachedStateForConnection` (per-`wsGen`), so each reconnect evaluates cache validity independently.

### 9.5 pty_state (WS protocol — NEW)

- Server → client only.
- `{type: "pty_state", altBuffer: boolean}`.
- Emitted on every transition. Not emitted in steady state.
- On reconnect, server sends the current state exactly once after `history_end` so a fresh client knows where the session currently is.

### 9.6 SIGWINCH recovery (server internal)

- Only fires from `FlushPending` when **all** of:
  - `pendingTrimmed == true` AND actual bytes were discarded (not just snap-to-boundary).
  - `!ptyState.IsAltBuffer()`.
  - `time.Since(lastRecoveryAt) >= sigwinchCooldown` (default 1s, configurable via `session.sigwinch_cooldown` config key).

### 9.7 Local echo (client internal)

- Disabled whenever `altBuffer === true` OR session configuration sets `localEcho = "off"`.
- Mismatch behavior: drop predictions, no `\b \b` written. Ever.

## 10. Testing Plan (Automated)

Verification is via automated tests. No manual repro checklists.

### 10.1 Frontend unit tests (vitest, already in place)

- `inputGate.test.ts` — 18+ cases covering 6 sources × (ready/not-ready/paused/rejected) matrix.
- `useTerminalTransport.test.ts` — connect, reconnect, message bus subscribe, generation counter.
- `useStdinAck.test.ts` — seq allocation, ack timeout, re-enqueue rules including the wsGen write barrier.
- `useTerminalSession.test.ts` — session_ready gating, history replay, cache invalidation across multiple reconnects, pty_state propagation to local echo.
- `localEcho.test.ts` — drop-on-mismatch, no `\b \b` emitted, alt-buffer disables echo.

### 10.2 Frontend integration test (vitest)

- `terminal-integration.test.tsx` — mounts `TerminalPane` with a fake WebSocket and a script of server frames. Scenarios:
  1. Paste via MobileToolbar while pre-session_ready → pill shows "queued"; flushes on session_ready.
  2. Paste via MobileToolbar while mouse-tracking mode active → gate returns `paused-by-xterm-mode`; pill shows "Paused".
  3. Server sends `pty_state altBuffer=true`; subsequent keystrokes do not render local echo.
  4. Coalesced output with trim while alt buffer active; server does **not** send SIGWINCH (verified via a server-side counter exposed in `__wc_terminal_debug`).
  5. WS drops mid-stdin_ack; reconnect establishes new wsGen; payload is re-sent exactly once.

### 10.3 Backend unit tests (go test)

- `session_altbuf_test.go` — enter/exit on single read, split across reads, nested (redundant 1049h), idempotent exit.
- `session_sigwinch_recovery_test.go` — suppressed during alt buffer, rate-limited to 1 per second, skipped when no bytes trimmed.
- `pty_state_broadcast_test.go` — transitions emitted as WS messages; steady-state does not emit.

### 10.4 Backend integration tests (go test with testcontainers-style fake PTY)

- `terminal_ws_test.go` — extended: client reconnects mid-write, assert PTY writes observed by fake PTY == 1.
- Existing `claude_startup_integration_test.go` — must still pass unchanged. If it fails, regress the change.
- Existing `ansi_responder_integration_test.go` — must still pass (server strip is still authoritative).

### 10.5 Static-assertion tests

- `find_test.sh` (or a Go test invoking `git grep`) — fails if any of the following reappear:
  - `\\x1b\\[\\?2026` in `ui/src/` (client must never strip again).
  - `useTerminalSocket` import anywhere.
  - `trySendStdin` or `sendInput` function definition outside `inputGate.ts`.
  - A raw `s.pty.SetSize` call in `session.go` outside the gated recovery path and the `Resize` method.

## 11. Rollout / Validation Checklist

- [ ] All phases' exit criteria met.
- [ ] `cd scenarios/web-console/ui && pnpm test` green.
- [ ] `cd scenarios/web-console/ui && pnpm run lint` green.
- [ ] `cd scenarios/web-console/ui && pnpm run typecheck` green.
- [ ] `cd scenarios/web-console/api && go build ./... && go test ./... -timeout 600s` green.
- [ ] `cd scenarios/web-console/api && gofumpt -l .` shows no diff.
- [ ] `cd scenarios/web-console/api && golangci-lint run` green.
- [ ] User reloads the web-console in their browser (they do this; plan does not touch a running scenario — see §12 risks).
- [ ] Static-assertion tests (§10.5) green.
- [ ] Docs updated (`SEAMS.md`, `ARCHITECTURE.md`).
- [ ] No files in the repo still reference the deleted modules.

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Splitting `useTerminalSocket.ts` accidentally drops behavior (e.g., a subtle retry) | Phase 0's integration harness is built **before** the split and covers connect/reconnect/history/resume; any drop surfaces as a red test. |
| Alt-buffer tracking misses an exit (`?1049l`) and stays stuck in "alt buffer" state | `PTYStateTracker` test covers split-across-reads and redundant transitions. Also: add a client-side watchdog that logs if `altBuffer=true` persists >60s without fresh `stdout`. |
| `pty_state` message type deployed to a client that predates it | Non-issue: greenfield, no old clients. But the change should ship client+server together in the same commit. |
| SIGWINCH rate-limit of 1s is too aggressive for a legitimately slow client that catches up after a trim | `sigwinchCooldown` is configurable via `session.sigwinch_cooldown`. Default 1s covers observed trim frequency. |
| Removing the client mode-2026 strip exposes a server bug where a non-sanitized path emits it | §10.5's negative test ensures we'd catch it immediately, and the server test covers all `broadcast` paths. |
| Mid-write double-send barrier (wsGen) causes a genuine write loss if a server silently drops a payload without closing WS | Write barrier is only active on reconnect-to-same-generation. If the ack never arrives, the normal ack-timeout path still fires (2s default), and re-enqueue happens as today. |
| Plan assumes plain restart-the-running-dev-server workflow | Per user's feedback on not restarting the scenario Claude Code is running inside, the plan only writes code to disk. User restarts the web-console UI/API themselves after each phase. |

## 13. Non-goals / Prohibited Patterns

- **No compatibility shims.** Deleted files are not re-exported. Deleted functions have no wrappers. If a call site would break, it's updated in the same commit.
- **No feature flags for the rework.** Gate, alt-buffer tracking, and split hooks are all-on immediately.
- **No manual test steps in lieu of automated tests.** Every acceptance criterion is an automated test (§10).
- **No mocking the PTY in an end-to-end test** — Vrooli convention (see `feedback_scenario_url_resolution.md` and the LPBS testcontainers pattern). The `terminal_ws_test.go` integration uses a fake-PTY seam, but `claude_startup_integration_test.go` continues to use a real PTY.
- **No commenting out code.** Dead code is deleted.
- **No TODO / FIXME / XXX comments for follow-ups.** If follow-up is needed, add a task. (TaskCreate)
- **No emojis** in source, comments, or tests.
- **No git commits, reverts, or resets** by the implementing agent (honors `feedback_no_git_mutations.md`). User makes all commits.
- **No restart of the web-console scenario** by the implementing agent (honors `feedback_no_restart_active_scenario.md`). User restarts after each phase.
- **Local echo does not compensate for unknown server-side conditions.** If prediction fails, drop and move on — do not invent corrective output.

## 14. Definition of Done

All must be true before the plan is considered complete:

1. §11 rollout checklist fully green.
2. Static-assertion tests (§10.5) enforce greenfield constraints in CI.
3. `grep -rn "useTerminalSocket" scenarios/web-console/` returns zero matches except historical git history.
4. `grep -rn "\\\\b \\\\b" scenarios/web-console/ui/src/lib/localEcho.ts` returns zero matches.
5. `grep -rn "2026" scenarios/web-console/ui/src/` returns zero matches (server stays authoritative).
6. Integration test `terminal-integration.test.tsx` covers the five scenarios listed in §10.2 and passes.
7. Backend tests cover alt-buffer tracking, SIGWINCH gating, and `pty_state` broadcast, all green.
8. A user reproducing the original paste-lost scenario on their phone sees a "Paused" pill (not a silent lost payload) when xterm is in a mode-incompatible state. Verified via the integration test (no manual verification claimed).
9. A user reproducing the original alt-buffer duplication scenario no longer sees tmux status-line striping; verified via the backend SIGWINCH test (which asserts zero SIGWINCH calls during alt-buffer coalesce trim).
10. No deprecated, compat, legacy, migration-bridge, or fallback code exists in any file touched by this plan.

---

**Plan file:** `scenarios/web-console/docs/plans/terminal-session-rework-implementation-plan.md`
**Owner:** unassigned (claim via TaskUpdate)
**Created:** 2026-04-22
