# Terminal Input Protocol

Contract between the browser-side session hook and the server-side
WebSocket handler for every client → server input frame. Lives here
rather than ARCHITECTURE.md so reviewers touching the delivery path
read the exact semantics, not an overview.

Referenced from:

- `ui/src/components/terminal/inputGate.ts`
- `ui/src/hooks/terminal/useStdinAck.ts`
- `ui/src/hooks/terminal/useTerminalSession.ts`
- `api/terminal_ws.go` (type constants)
- `api/terminal_ws_input.go` (dispatch)
- `api/pty.go` / `api/pty_tmux.go` (delivery)

## Wire frames

### `stdin` (client → server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"stdin"` | yes | Discriminator. |
| `data` | `string` | yes | Payload. UTF-8 JSON string; multi-byte runes are delivered byte-exact to the PTY. |
| `seq` | `number` | yes | Client-assigned monotonic sequence, per WS connection. Reset to 1 on each (re)open. Echoed by `stdin_ack`. |
| `kind` | `"keystroke"` \| `"paste"` | yes | Discriminates the delivery path on the persistent backend. Empty / unknown defaults to `"keystroke"` on the server for defensive parsing; the UI always sends it explicitly. |

### `stdin_ack` (server → client)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"stdin_ack"` | yes | Discriminator. |
| `seq` | `number` | yes | Echoes the `seq` of the matched `stdin` frame. |
| `ok` | `boolean` | yes | `true` iff the backend accepted the bytes. |
| `reason` | `string` | when `ok=false` | Typed error code (see below). Human-readable detail in `data`. |
| `data` | `string` | when `ok=false` | Full error text for logging; UI presents `reason` only. |

## `InputKind` — delivery paths

| Kind | Standard backend (`realPTY`) | Persistent backend (`tmuxPTY`) |
|---|---|---|
| `keystroke` | `ptmx.Write(data)`. | `tmux send-keys -t <session> -l -- <data>`; any active tmux client mode (copy-mode, command-prompt, menu, prefix-pending) is cancelled first via `send-keys -X cancel`. Payloads > 64 KB fall through to the paste path to dodge argv size limits. |
| `paste` | `ptmx.Write(data)`. | `tmux load-buffer -b <buf> - < data` then `tmux paste-buffer -d -b <buf> -t <session>`; the mode is cancelled first. The `-d` flag deletes the per-call buffer after delivery so buffers never leak. |

`-l` (literal) + explicit cancel + per-call buffers are load-bearing:
- `-l` alone does NOT escape copy-mode in tmux 3.4; it delivers to the
  pane's running program only if the client is in the default mode.
- `paste-buffer -d` alone does NOT escape copy-mode either.
- The fix is the explicit `send-keys -X cancel` pre-step guarded by
  `#{pane_in_mode}` so we don't emit a spurious `not in a mode`
  error on the common path.

## `reason` codes on `stdin_ack.ok=false`

| Code | Source | Semantics |
|---|---|---|
| `tmux_write_failed` | `tmuxPTY.WriteInput` → `tmux send-keys`/`load-buffer`/`paste-buffer` returned non-zero | Transient or terminal tmux failure. UI preserves the draft, surfaces reason, does NOT auto-retry. |
| `pty_closed` | `errors.Is(writeErr, errPTYClosed)` | Session's PTY has been closed (process exited, Close called). UI stops attempting to send; pane should surface a terminated banner. |
| `not_ready` | Client-side only (input gate / useStdinAck) | `session_ready` not yet observed on this WS gen. Never emitted by the server. |
| `invalid_input` | Reserved; not emitted today | Future use for malformed payloads. |

## Client-side gating

`useStdinAck.send(data, kind)` emits a `stdin` frame only if the
session is ready; otherwise returns `{sent: false, reason: "not-ready"}`
and the caller queues the payload. On a fresh `session_ready`, the
queue is drained in FIFO order with each entry's original `kind`
preserved.

`useStdinAck.handleClose` re-enqueues only payloads whose `gen` matches
the current transport generation. Payloads from prior generations
have a committed outcome (their server wrote the bytes before the WS
close, or the shell never saw them — either way, double-delivery is
worse than one-off loss on a flaky network).

## `subscribeInputSettled` contract

Every `send()` that returns `{sent: true, seq}` eventually settles via
`subscribeInputSettled(cb)`: `cb(seq, ok)` fires exactly once. Paths:

1. `stdin_ack(seq, ok=true)` arrives → `cb(seq, true)`.
2. `stdin_ack(seq, ok=false)` arrives → `cb(seq, false)`. Payload is
   automatically re-enqueued.
3. `ACK_TIMEOUT_MS` elapses without an ack → `cb(seq, false)`. Payload
   is automatically re-enqueued.
4. WS close with matching `gen` → `cb(seq, false)`. Payload is
   automatically re-enqueued.

`TerminalContextMenu`'s paste UI keys on this: the menu stays open
showing `Pasting…` until the cb fires, then flashes `Pasted` (ok) or
`Paste failed: <reason>` (not ok) before closing.

## Bug regressions to refuse

These are the shapes the refactor permanently forbids. Each has a
greenfield assertion test that fails at CI time if it reappears:

- `ptmx.Write(` called outside `pty.go` / `pty_tmux.go` — would bypass
  the kind-aware tmux path and reintroduce Bug A.
- `PTY.Write(p []byte) (int, error)` declared on the interface — the
  legacy shape was deleted; its return does not carry the typed kind.
- Code on the stdout handler that does NOT advance `totalBytesRef`
  per-frame — cache drift reintroduces Bug C.
- A second paste handler that calls `submitInput` from anywhere other
  than `TerminalPane.tsx` — bypasses the settlement contract and
  reintroduces Bug B.
- References to any deleted plan filename — indicates a stale
  reviewer/doc cross-link.
