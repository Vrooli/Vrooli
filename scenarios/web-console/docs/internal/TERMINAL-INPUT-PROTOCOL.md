# Terminal Input Protocol

Contract between the browser-side session hook and the server-side
WebSocket handler for every client → server input frame. Lives here
rather than ARCHITECTURE.md so reviewers touching the delivery path
read the exact semantics, not an overview.

Referenced from:

- `ui/src/components/terminal/inputGate.ts`
- `ui/src/hooks/terminal/useStdinStream.ts`
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
| `offset` | `number` | yes | Cumulative UTF-8 byte offset at which this payload starts. The first payload starts at zero. |
| `intent` | `"typing"` \| `"bulk_text"` \| `"named_key"` | yes | Identifies the source intent. `bulk_text` selects tmux paste-buffer delivery; `typing` and `named_key` select literal send-keys delivery. Empty / unknown defaults to `"typing"` on the server for defensive parsing. |

### `hello` (client → server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"hello"` | yes | Starts reconnect reconciliation. |
| `have_through` | `number` | yes | Highest offset the client has already released. The server refuses a value ahead of its accepted prefix. |

### `stdin_ack` (server → client)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"stdin_ack"` | yes | Discriminator. |
| `accepted_through` | `number` | yes | Highest contiguous UTF-8 byte offset accepted by the session. It is monotonic across WebSocket reconnects. |
| `ok` | `boolean` | yes | `true` iff the backend accepted the bytes. |
| `reason` | `string` | when `ok=false` | Typed error code (see below). Human-readable detail in `data`. |
| `data` | `string` | when `ok=false` | Full error text for logging; UI presents `reason` only. |

### `session_ready` mouse fields (server → client)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `mouse_mode_known` | `boolean` | yes | `true` only for a persistent tmux-backed pane that exposes mouse capture. |
| `mouse_mode` | `boolean` | when known | Current tmux mouse-capture state. `false` is the default and means browser-local scrollback remains in control. |

### `mouse_mode` (client ↔ server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"mouse_mode"` | yes | Requests or reports the per-pane tmux mouse-capture state. |
| `data` | `"on"` \| `"off"` \| `"unsupported"` | yes | Client sends `on` or `off`; the server echoes the applied state. Unsupported backends return `unsupported` and do not change terminal input. |
| `ok` | `boolean` | server response | `true` only when the tmux option was changed successfully. |
| `reason` | `string` | on unsupported/rejected | Diagnostic detail; it is not written into the terminal buffer. |

## Input lanes and delivery paths

The complete source-intent vocabulary is `typing`, `bulk_text`, `named_key`,
and `control`. Operator payloads use the reliable stdin lane and settle through
`stdin_ack`; `control` is represented by the separate control frame rather than
an acknowledged stdin frame.
Synthetic terminal bytes use a separate best-effort `control` frame and do not
enter the reliable-input ordering/replay state; reconnect must never replay them.

| Intent | Standard backend (`realPTY`) | Persistent backend (`tmuxPTY`) |
|---|---|---|
| `typing` / `named_key` | `ptmx.Write(data)`. | `tmux send-keys -t <session> -l -- <data>`; any active tmux client mode (copy-mode, command-prompt, menu, prefix-pending) is cancelled first via `send-keys -X cancel`. Payloads > 64 KB fall through to the paste path to dodge argv size limits. |
| `bulk_text` | `ptmx.Write(data)`. | `tmux load-buffer -b <buf> - < data` then `tmux paste-buffer -d -b <buf> -t <session>`; the mode is cancelled first. The `-d` flag deletes the per-call buffer after delivery so buffers never leak. |

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
| `offset_gap` | Server | The client sent a payload after a byte offset the session has not accepted. |
| `unreconcilable` | Server/client | Reconciliation found an offset inside a payload or below the released prefix. No bytes are replayed. |

## Client-side gating

`useStdinStream.send(data, kind)` emits a `stdin` frame only if the
session is ready and the WebSocket is open; otherwise returns
`{sent: false, reason: "not-ready"}` and the caller queues the payload. An
open socket's browser send-buffer high-water mark does not create a second
reliable-input queue: cumulative offsets remain the ordering and replay
barrier. On a fresh `session_ready`, the offline queue is drained in FIFO
order with each entry's original intent preserved.

`useStdinStream` retains sent payload boundaries across a close. It sends
`hello`, compares the server's `accepted_through` with the local released
offset, and replays only the unaccepted suffix. It never retries by timer.
Offscreen pane unmounts retain the same entry boundaries and intent in the
workspace buffer; they are not flattened into one bulk-text string.

## `subscribeInputSettled` contract

Every `send()` that returns `{sent: true, offset}` eventually settles via
`subscribeInputSettled(cb)`: the numeric argument is the covered byte offset,
not a connection-local sequence. The server must never acknowledge beyond the
client write head; such a response is unreconcilable. Paths:

1. `stdin_ack(accepted_through)` covers the payload → `cb(offset, true)`.
2. `stdin_ack(ok=false)` arrives → `cb(offset, false)` with the typed reason.
3. Reconnect reports an offset inside a payload or below the released prefix
   → `cb(offset, false, "unreconcilable")`; no bytes are replayed.

`TerminalContextMenu`'s paste UI keys on this: the menu stays open
showing `Pasting…` until the cb fires, then flashes `Pasted` (ok) or
`Paste failed: <reason>` (not ok) before closing.

## Bug regressions to refuse

These are the shapes the refactor permanently forbids. Each has a
greenfield assertion test that fails at CI time if it reappears:

- `ptmx.Write(` called outside `pty.go` / `pty_tmux.go` — enforced by
  `TestGreenfield_NoRawPtmxWriteOutsidePTYFiles`.
- `PTY.Write(p []byte) (int, error)` declared on the interface — enforced by
  `TestGreenfield_PTYInterfaceHasNoLegacyWrite`.
- Raw PTY resizing outside the lease-gated paths — enforced by
  `TestGreenfield_NoRawSetSizeOutsideGatedPaths`.
- Audio-tools being required for terminal boot — enforced by
  `TestGreenfield_AudioToolsDependencyIsLazyDegraded`.
- Internal audio domains importing handlers — enforced by
  `TestGreenfield_InternalAudioDomainsDoNotImportHandlers`.
- Orchestration bypassing audio ports — enforced by
  `TestGreenfield_OrchestrationRoutesThroughAudioPorts`.
