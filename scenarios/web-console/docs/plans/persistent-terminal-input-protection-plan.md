# Persistent Terminal Input Protection — Implementation Plan

## 1. Purpose

Close the race window that allows mobile-toolbar input to be silently lost when sent to a **persistent** (tmux-backed) terminal session. Today the client's "was it sent?" check relies solely on `ws.readyState === OPEN`, which is true well before the backend is actually ready to relay stdin into tmux — so the toolbar clears the draft (`clearDraft()` in `MobileToolbar.tsx`) even when the bytes never reach the PTY. This plan adds a backend acknowledgment for stdin, gates success on a `session_ready` signal, hardens client send error-handling, adds observability on the server write path, and surfaces pending input visibly in the UI.

## 2. Greenfield Constraint

**This is greenfield work.** Do not add compatibility shims, wire-format fallbacks, dual code paths, `// removed` comments, or `_unused` renames. The UI and API ship together in one scenario — old clients will not hit new servers. Remove the old optimistic path entirely once the replacement is in place.

## 3. Required Reading

```bash
prompt-manager skill read scientific-debugging seam-discovery-and-enforcement signal-and-feedback-surface-design bugfix-scope test
```

These cover: diagnosing race conditions rigorously (scientific-debugging), introducing new seams without spraying coupling (seam-discovery-and-enforcement), the UX/observability surface for "pending/queued/lost" signals (signal-and-feedback-surface-design), keeping the diff scoped (bugfix-scope), and regression-test discipline (test).

## 4. Problem Statement

**Observed:** Users occasionally lose long messages submitted through the mobile-toolbar text box when the active pane is a persistent terminal. The same toolbar, same code path, never loses input when the active pane is a standard (non-persistent) session.

**Root cause (confirmed via code read):**

1. `useTerminalSocket.sendInput` (`scenarios/web-console/ui/src/hooks/useTerminalSocket.ts:181-192`) returns `true` iff `ws.readyState === WebSocket.OPEN`. `ws.send()` is not wrapped in try/catch and `bufferedAmount` is not checked.
2. `MobileToolbar.submitCommand` (`scenarios/web-console/ui/src/components/MobileToolbar.tsx:219-226`) calls `clearDraft()` the instant `sendInput` returns `true`.
3. On the server, `handleTerminalWS` (`scenarios/web-console/api/terminal_ws.go:102-260`) accepts the WebSocket, runs `sess.Subscribe()`, streams history chunks to the client, and **only then** enters the input loop at line 260. The inner input loop calls `sess.Write(...)` (line 277) without any sync point confirming the tmux attach pipeline is fully wired.
4. For **persistent** sessions specifically, `tmuxPTYFactory` (`scenarios/web-console/api/pty_tmux.go:239-304`) spawns `tmux attach-session` under a PTY — a 50–500 ms asynchronous handshake. During this window the `attach` process's PTY master accepts writes (so `sess.Write()` succeeds silently) but the tmux server has not yet connected the pipe through to the real session, and those bytes are discarded.
5. Pending-queue replay (`useTerminalSocket.ts:217-225`) flushes queued stdin on WebSocket re-open without any check that the backend PTY is actually ready.

The standard backend's PTY is synchronous (`pty.go:92-103`), so it has no equivalent window.

## 5. Scope

**In scope:**
- New wire message `session_ready` (server → client) emitted after the input loop starts reading AND, for persistent sessions, after the tmux attach pipeline has confirmed readiness.
- New wire message `stdin_ack` (server → client) echoed after `sess.Write()` returns successfully, keyed by a client-assigned sequence number.
- Client-side gating: `sendInput()` returns `false` until `session_ready` is received; draft is only cleared on matching `stdin_ack` (with timeout → treat as failure).
- Hardened `ws.send()` (try/catch + `bufferedAmount` threshold).
- Server logging + metrics counter for any stdin accepted on a not-yet-ready session (so we can see if the race still exists after the fix).
- Pending-input pill in the UI near the mobile toolbar showing N queued/unsent bytes with a tooltip listing ages.

**Out of scope:**
- Rearchitecting the PTY subscribe/history/input sequencing. We add a seam; we do not re-order the existing bring-up.
- Voice input path (separate seam, not reported as broken).
- Cross-tab or multi-connection concurrency semantics.
- Changes to conversation event / TTS ack protocol (already has its own ack).

## 6. Current Technical Context

| File | Role | Key lines |
|---|---|---|
| `scenarios/web-console/ui/src/hooks/useTerminalSocket.ts` | WebSocket lifecycle, `sendInput`, pending queue | 161–170, 172–192, 217–225, 269–293, 484–502 |
| `scenarios/web-console/ui/src/components/MobileToolbar.tsx` | Toolbar submit + draft preservation | 186–231, 280 |
| `scenarios/web-console/ui/src/components/TerminalPane.tsx` | `sendInput` via `useImperativeHandle` | 337–374 |
| `scenarios/web-console/ui/src/hooks/useSessionManager.ts` | `sendToActiveTerminal` dispatcher | 248–257 |
| `scenarios/web-console/ui/src/components/Workspace.tsx` | `handleSendToTerminal` wrapper | 318–323 |
| `scenarios/web-console/api/terminal_ws.go` | WS upgrade, Subscribe, output fwd, input loop | 102–260, 260–317; message types 24–72 |
| `scenarios/web-console/api/backend_registry.go` | Backend dispatch (`standard`, `persistent`) | 11–18, 153–177 |
| `scenarios/web-console/api/pty.go` | Standard PTY factory (synchronous) | 92–103 |
| `scenarios/web-console/api/pty_tmux.go` | Persistent PTY factory (async attach) | 239–304; Write 59–67 |
| `scenarios/web-console/api/session.go` | Subscribe/history replay/readLoop | 33, 186–259, 554–674, 1174–1199 |
| `scenarios/web-console/ui/src/consts/backend-options.ts` | Shared backend ID constants | 1–24 |

## 7. Target End State

- `sendInput(data)` returns `true` only when the server has confirmed both `session_ready` and, for the specific sent payload, a matching `stdin_ack` within the timeout. Otherwise returns `false` and the toolbar preserves the draft.
- `ws.send` failures (exceptions or `bufferedAmount` above a high-water mark) are treated as `sendInput === false`.
- The UI shows a persistent "N unsent" pill whenever the pending queue is non-empty, with per-item ages.
- Server metric `stdin_before_ready_total` stays at 0 in steady state; any non-zero value is a logged warning with session ID and backend.
- Identical observable behavior for standard and persistent backends from the user's perspective — both now reliably preserve drafts on any failure.

## 8. Implementation Strategy

Phased so each phase is independently testable and revertable. Do each phase fully (code + tests + typecheck) before starting the next.

### Phase 1 — Wire protocol additions (server + shared types)

- In `scenarios/web-console/api/terminal_ws.go`:
  - Add constants: `MsgTypeSessionReady = "session_ready"`, `MsgTypeStdinAck = "stdin_ack"`.
  - Extend `TerminalMessage` with `Seq int64 \`json:"seq,omitempty"\`` and `Ok bool \`json:"ok,omitempty"\`` (serde-omit zero values, consistent with existing fields).
- Mirror in `scenarios/web-console/ui/src/hooks/useTerminalSocket.ts`:
  - Extend the `TerminalMessage` union literal (line 21) with `"session_ready" | "stdin_ack"` and add optional `seq?: number; ok?: boolean` fields on the interface.
- No behavior yet — pure type additions. Typecheck both sides must pass before Phase 2.

### Phase 2 — Server: emit `session_ready` and `stdin_ack`

- In `handleTerminalWS` (`terminal_ws.go`):
  - Emit `session_ready` as the very first message the input loop sends after it has done a successful "zero-byte probe" of the PTY — a short `sess.ProbeReady(ctx, timeout)` helper that writes 0 bytes (or a no-op ioctl) and confirms the PTY is accepting writes. For the standard backend this returns immediately. For tmux, add `ProbeReady` on `tmuxPTY` that polls `tmux list-clients -t <target>` (or equivalent) until attached or times out (default 3 s, fail hard with a typed error to the client).
  - In the stdin case of the input loop, call `sess.Write(...)`; on success, write back `{type: "stdin_ack", seq: msg.Seq, ok: true}`; on error, `{ok: false, data: "<reason>"}` **and** keep the existing error propagation.
  - Add counter `stdinBeforeReadyTotal` on `s.metrics`; log+increment if a stdin message arrives before `session_ready` was emitted (should never happen — but we'll see it if the sequencing ever regresses).
- In `session.go` — add `ProbeReady(ctx context.Context) error` on the `Session` interface; implement for standard PTY (return nil) and for tmux PTY (wrap existing `tmuxAttach` handshake).
- Keep `sess.Write` error path untouched — it already sends `MsgTypeError` and terminates the input loop. The new ack just adds a positive signal.

### Phase 3 — Client: seq, ack map, timeout, `session_ready` gate

- In `useTerminalSocket.ts`:
  - Add `sessionReadyRef = useRef(false)`, `nextSeqRef = useRef(1)`, `pendingAcksRef = useRef<Map<number, {data: string; timer: number; resolve: (ok: boolean) => void}>>(new Map())`.
  - On `onopen`: reset all three. Do **not** flush pending input yet.
  - On `message.type === "session_ready"`: set `sessionReadyRef = true`; flush pending queue through the new `sendInput` path (each queued item gets its own seq + timer + ack wait).
  - On `message.type === "stdin_ack"`: resolve the matching map entry (`ok` = true|false). Clear the timer.
  - Rewrite `sendInput(data)`:
    - If `!sessionReadyRef.current` → enqueue, return `false`.
    - Assign `seq = nextSeqRef++`, wrap `ws.send(...)` in try/catch; if throw or `bufferedAmount > WS_SEND_HIGH_WATER` (e.g. 1 MiB) → enqueue, return `false`.
    - Return `true` **synchronously** for the optimistic signal (existing callers expect a boolean immediately), but **also** register a pending-ack timer (default 2 s) that, on timeout, re-enqueues the payload, decrements any UI "sent" counter, and fires a new event `onInputLost(seq, data)`.
  - Because the toolbar's existing `if (sent) clearDraft()` semantics are now insufficient (we want to keep the draft until the ack lands), **change the toolbar contract**: `onInput` must return `Promise<boolean>` that resolves on ack/timeout. Alternatively, expose a second callback `onInputSettled(ok)` and have the toolbar keep the draft visible with a subtle "sending…" state until `onInputSettled`. Choose the callback variant — it keeps `sendInput` synchronous for the xterm `onData` path that cannot await.
- In `TerminalPane.tsx` imperative handle: expose both `sendInput(data): boolean` and `subscribeToInputSettled(cb): () => void`.
- In `MobileToolbar.tsx`:
  - Replace the immediate `clearDraft()` with: mark input as "sending" (disable button, keep text visible), subscribe once for the next settlement, clear on `ok === true`, restore editing + show "Send failed — retry" on `ok === false`.
  - Keep the existing `"queued"` status path for the `sendInput === false` case.

### Phase 4 — Hardened send + pending pill

- Wrap the `ws.send` at `useTerminalSocket.ts:175` and `:223` and `:498` in try/catch; on throw or `bufferedAmount >= WS_SEND_HIGH_WATER` treat as failure (push to pending, do not emit ack-wait).
- Add a React subscription (selector on `pendingInputRef` length + per-item timestamps) and render a small pill above the MobileToolbar: `"⏳ 2 unsent (oldest 3s)"`. Clicking the pill opens a disclosure with raw payloads (truncated) and a "Retry now" button (which force-flushes the queue).
- Stored in a new hook `usePendingInputStatus(sessionId)` so other surfaces (e.g. session list) can show the pill too.

### Phase 5 — Observability

- Server: expose `stdin_before_ready_total` on the existing `/metrics` endpoint. Add a structured log line `ws[%s] stdin before session_ready — backend=%s` on every increment.
- Client: debug-only console warning + a one-line telemetry event (`input_ack_timeout`) when the ack timer fires. No PII — sequence number and payload length only.

## 9. Contract Decisions

- **Wire additions only, no removals.** `stdin_ack` and `session_ready` are additive. The server still emits `history_end` exactly as today.
- **Seq is opaque to the server** — the server echoes whatever seq the client sent. The client is free to reuse or skip numbers; only the map-match matters.
- **Ack timeout default: 2000 ms.** Rationale: tmux attach p99 is ~500 ms in our environment; 2 s leaves 4× headroom for network jitter without making users wait visibly long for a "failed" signal. Exposed as `VITE_WC_ACK_TIMEOUT_MS` for ops override.
- **`bufferedAmount` high-water default: 1 MiB.** Above this the browser is more likely to silently drop; we refuse the send and queue.
- **`session_ready` failure (tmux attach timeout ≥ 3 s):** server emits `{type: "error", data: "session_not_ready"}` and closes. Client's existing error handler already surfaces this.
- **No protocol version bump** — additive under a greenfield UI/API pair deployed together.

## 10. Testing Plan

All tests must be automated. No manual test checklists.

**Client (`scenarios/web-console/ui/`, Vitest):**
- `useTerminalSocket.hook.test.ts`:
  - `sendInput` returns `false` and enqueues when `session_ready` has not arrived.
  - `sendInput` returns `true`, assigns increasing seq, fires `inputSettled(true)` on matching `stdin_ack`.
  - Ack timeout → `inputSettled(false)` + payload re-enqueued.
  - `ws.send` throw → `false`, payload enqueued, no ack-wait registered.
  - `bufferedAmount` above high-water → `false`, enqueued.
  - Pending queue flushes on `session_ready`, each flushed item gets its own ack-wait.
- `MobileToolbar.test.tsx`:
  - Draft preserved during `sending` state, cleared on `ok === true`.
  - Draft restored + "Send failed" shown on `ok === false`.
  - "N unsent" pill appears when queue non-empty; disappears when drained.
- Integration smoke: `useTerminalSocket.test.ts` reconnect scenario — stdin sent before `session_ready` on the new socket stays in the queue, flushes exactly once after `session_ready`.

**Server (`scenarios/web-console/api/`, Go):**
- `terminal_ws_test.go` (new cases):
  - Standard backend: input loop emits `session_ready` immediately; every stdin receives a matching `stdin_ack` with `ok: true`.
  - Persistent backend (mock `tmuxPTY.ProbeReady` with a controllable signal): `session_ready` is emitted **only after** ProbeReady returns; stdin received before ProbeReady increments `stdin_before_ready_total`.
  - Write error from `sess.Write` produces `stdin_ack` with `ok: false` and still sends `MsgTypeError`.
  - ProbeReady timeout surfaces `error` and closes the WS.
- `session_test.go`:
  - `ProbeReady` happy path for both backends.
  - `ProbeReady` timeout path for tmux (use a stub attach that never signals ready).

**End-to-end (existing scenario test harness):**
- Extend `vrooli scenario test web-console` with a new case that drives a tmux-backed session, sends a 10 KiB payload immediately on first connect, and asserts it arrives intact at the PTY.

## 11. Rollout / Validation Checklist

- [ ] All Phase 1–5 unit + integration tests pass.
- [ ] `cd scenarios/web-console/api && go build ./... && go test ./... -timeout 300s` clean.
- [ ] `cd scenarios/web-console/ui && npm run typecheck && npm test` clean.
- [ ] `cd scenarios/web-console/api && golangci-lint run ./...` clean — fix **all** issues in modified files, including pre-existing ones.
- [ ] `cd scenarios/web-console/ui && npm run lint` clean — fix **all** issues in modified files, including pre-existing ones.
- [ ] `gofumpt -w` applied to all modified Go files.
- [ ] `vrooli scenario restart web-console` (run by the user — this plan does not auto-restart; write code to disk and stop).
- [ ] After user restart: `curl -s http://localhost:<port>/health` returns healthy.
- [ ] After user restart: `curl -s http://localhost:<port>/metrics | rg stdin_before_ready_total` returns the new counter (0 expected).
- [ ] Scenario test: `vrooli scenario test web-console` passes including new cases.

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Ack timeout hides slow-but-successful sends, making users retry and double-submit | Client de-dupes pending queue by seq; on late ack arrival, the in-flight "retry" is cancelled before send. |
| `ProbeReady` adds latency to session open | Only runs once per WS connect; tmux handshake already had to happen before the user could usefully type; this just gives the client a visible signal. Ship with metric + dashboard on p50/p99 of ready time. |
| Greenfield protocol deploy: old UI assets cached in CDN hit new API | Scenario UI and API are versioned together and deployed atomically; cache-bust the UI bundle on release. |
| The ack mechanism itself becomes a new silent-drop surface (server emits ack, client misses it) | Client clears pending timers on WS close/reconnect and re-enqueues anything unacked, so a dropped ack becomes a timeout + retry, not a lost message. |
| xterm direct `onData` path (`useTerminalSocket.ts:484-502`) has no UI to "retry from" — the char is already echoed | Out of scope for this plan per Section 5, but we log `input_ack_timeout` so we can size the real exposure before deciding. The toolbar path is where users send long messages, which is the reported pain point. |
| Change in `onInput` return semantics breaks voice-input or other callers | Grep `sendInput\|onInput` in `ui/src` during Phase 3 and update every caller; type system enforces this via the imperative handle signature change. |

## 13. Non-goals / Prohibited Patterns

- **No** fallback path that treats a missing `session_ready` as "assume ready after N ms." Either the server sends it or we refuse to send.
- **No** optimistic draft-clear "if we've been connected > X ms." That is the current bug.
- **No** re-sending on a different WebSocket generation without re-acking — sequence numbers are scoped per connection.
- **No** changes to the xterm `onData` direct keystroke path in this plan.
- **No** compat shim that accepts stdin without a seq. Every client send must have a seq; server may assign one if missing and log a warning (greenfield — client always sends it).

## 14. Definition of Done

- `sendInput` contract in `useTerminalSocket.ts` is gated on `session_ready`; returns `false` otherwise; `ws.send` is try/caught with `bufferedAmount` guard.
- Server emits `session_ready` exactly once per WS connection, after `ProbeReady` succeeds.
- Every stdin message produces exactly one `stdin_ack` (ok true or false) OR a terminal `error` message.
- MobileToolbar retains the draft until ack arrives; shows "sending" during the wait, "Send failed — retry" on failure, clears only on success.
- Pending-input pill visible whenever queue is non-empty.
- Metric `stdin_before_ready_total` wired to `/metrics`.
- All automated tests listed in Section 10 pass locally.
- User restarts `web-console` and confirms via scenario test that the end-to-end path works; reproduces no loss on persistent sessions across 100 reconnect/send iterations.
