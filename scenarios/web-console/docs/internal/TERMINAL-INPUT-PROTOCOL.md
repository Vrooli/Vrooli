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
| `offset` | `number` | yes | Cumulative UTF-8 byte offset at which this payload starts within this WebSocket connection. The first payload starts at zero. |
| `intent` | `"typing"` \| `"bulk_text"` \| `"named_key"` | yes | Identifies the source intent. `bulk_text` selects tmux paste-buffer delivery; `typing` and `named_key` select literal send-keys delivery. Empty / unknown defaults to `"typing"` on the server for defensive parsing. |

### `hello` (client → server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"hello"` | yes | Starts reconnect reconciliation. |
| `have_through` | `number` | yes | Highest offset the client has already released in this connection's offset space. The server refuses a value ahead of its accepted prefix. |
| `rendered_through` | `number` | no | Output cursor rendered by this client before reconnect. Sent with `want_resume` to request a delta instead of a snapshot. |
| `want_resume` | `boolean` | no | Requests output replay from `rendered_through`. Older clients omit this field and receive the normal snapshot. |

The stdin offset is connection-scoped. `session_ready.accepted_through` is
therefore the starting point for that connection, while `stdin_ack` advances
only that connection's prefix. Input accepted by another viewer never makes a
new viewer's first offset unreconcilable.

### `stdin_ack` (server → client)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"stdin_ack"` | yes | Discriminator. |
| `accepted_through` | `number` | yes | Highest contiguous UTF-8 byte offset accepted for this WebSocket connection. It starts at zero for a new connection and is independent of other viewers. |
| `ok` | `boolean` | yes | `true` iff the backend accepted the bytes. |
| `reason` | `string` | when `ok=false` | Typed error code (see below). Human-readable detail in `data`. |
| `data` | `string` | when `ok=false` | Full error text for logging; UI presents `reason` only. |

### Output resume frames (server → client)

`stdout.output_cursor` is the exclusive end cursor of that frame and
`history_end.output_cursor` is the cursor covered by the initial replay. A
reconnecting client sends `hello{want_resume:true,rendered_through:C}`. If `C`
is an exact retained frame boundary, the server sends only the frames after
`C`, then `history_end`; no reset is sent. If the cursor has fallen out of the
bounded ring or lands inside a frame, the server sends `resync`, a complete
snapshot, and `history_end`. The snapshot remains the authoritative fallback.

### `presence` (server → client)

Presence is independent of grid dimensions and carries `viewerCount`,
`leader`, `leaderDevice`, and `holdsLease`. A size change cannot discard a
lease transfer or make a single viewer look like a follower.

### `session_ready` mouse fields (server → client)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `mouse_mode_known` | `boolean` | yes | `true` only for a persistent tmux-backed pane that exposes mouse capture. |
| `mouse_mode` | `boolean` | when known | Current tmux mouse-capture state. `false` is the default; it does **not** mean browser-local scrollback is in control — see "Who owns scrolling" below. |

### `mouse_mode` (client ↔ server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"mouse_mode"` | yes | Requests or reports the per-pane tmux mouse-capture state. |
| `data` | `"on"` \| `"off"` \| `"unsupported"` | yes | Client sends `on` or `off`; the server echoes the applied state. Unsupported backends return `unsupported` and do not change terminal input. |
| `ok` | `boolean` | server response | `true` only when the tmux option was changed successfully. |
| `reason` | `string` | on unsupported/rejected | Diagnostic detail; it is not written into the terminal buffer. |

### `scroll` (client ↔ server)

| Field | Type | Required | Semantics |
|---|---|---|---|
| `type` | `"scroll"` | yes | Asks the backend to move its own scrollback view. |
| `lines` | `number` | yes | Terminal rows. Negative scrolls back toward older output, positive scrolls forward toward live output. `0` is ignored without a reply. |
| `ok` | `boolean` | server response | `true` when the backend scrolled; the reply echoes `lines`. |
| `data` | `"unsupported"` | on rejection | The backend owns no history of its own; the pane keeps real client-side scrollback. |
| `reason` | `string` | on rejection | Diagnostic detail; it is never written into the terminal buffer. |

Like `control`, this frame is best-effort: it carries no offset, never enters
the reliable-input offset space, and is never replayed after a reconnect. A
scroll position is a view of the present, not a piece of the input stream.

## Who owns scrolling

Three different things can move a pane's view, and exactly one of them is
correct at any moment. Picking the wrong one is not a cosmetic bug: xterm's
fallback delivers cursor keys as **real stdin**.

| Terminal state | Owner | Mechanism |
|---|---|---|
| The program requested mouse tracking | The program | xterm encodes the wheel as a mouse report; the program scrolls itself. |
| No mouse tracking, xterm holds scrollback | The browser | xterm scrolls its own viewport using `scrollSensitivity`. |
| No mouse tracking, no scrollback | The **server** | The client sends a `scroll` frame; the backend walks its own history. |

The third row is not an edge case, it is every persistent pane. **A tmux client
emits `\x1b[?1049h` the moment it attaches**, so the browser terminal sits in
the alternate screen buffer for the entire life of the session, whatever the
pane runs. Two consequences follow, and both have bitten us:

- `terminal.scrollLines()` is a guaranteed no-op on such a pane. "Not in mouse
  tracking mode" therefore never implies "local scrollback is in control".
- xterm's built-in wheel handler, seeing no scrollback, translates each wheel
  notch into an `ESC [ A` / `ESC [ B` cursor key and submits it through
  `onData` as ordinary input. In an interactive agent whose composer binds
  Up/Down to message history, scrolling silently rewrites the operator's draft.

`useTerminalWheel` therefore claims the wheel through
`attachCustomWheelEventHandler` for the third row only, and defers to xterm for
the first two.

The same `\x1b[?1049h` is why `echo_state.in_alt_buffer` is sampled from the
pane's own `#{alternate_on}` rather than from the server-side emulator: the
emulator reads the attach stream, so it reports the alternate buffer for every
tmux session regardless of what the pane runs. Predictive echo is gated on that
flag, so reading it from the emulator disabled prediction for every persistent
pane.

## Input lanes and delivery paths

The complete source-intent vocabulary is `typing`, `bulk_text`, `named_key`,
and `control`. Operator payloads use the reliable stdin lane and settle through
`stdin_ack`; `control` is represented by the separate control frame rather than
an acknowledged stdin frame.
Synthetic terminal bytes use a separate best-effort `control` frame and do not
enter the reliable-input offset space or replay state. They are enqueued behind
stdin and ANSI-responder bytes in one per-session ordered writer, but have no
acknowledgement; reconnect must never replay them.

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
| `input_queue_full` | Server | The bounded per-session ordered input lane is full. The payload was not written; the client may retry or retain it in the pending-input UI. |

## Client-side gating

`useStdinStream.send(data, kind)` emits a `stdin` frame only if the
session is ready and the WebSocket is open; otherwise returns
`{sent: false, reason: "not-ready"}` and the caller queues the payload. An
open socket's browser send-buffer high-water mark does not create a second
reliable-input queue: cumulative offsets remain the ordering and replay
barrier. On a fresh `session_ready`, the offline queue is drained in FIFO
order with each entry's original intent preserved.

`useStdinStream` retains sent payload boundaries across a close. On a new
WebSocket it clears the connection-level desynchronization latch, rebases the
unaccepted payloads, sends `hello` with a zero connection offset, and replays
only the unaccepted suffix. The terminal session also sends the last rendered
output cursor so a covered reconnect receives a delta. It never retries by
timer.
Offscreen pane unmounts retain the same entry boundaries and intent in the
workspace buffer; they are not flattened into one bulk-text string.

## `subscribeInputSettled` contract

Every `send()` that returns `{sent: true, offset}` eventually settles via
`subscribeInputSettled(cb)`: the numeric argument is the covered byte offset
within the current connection. The server must never acknowledge beyond the
client write head; such a response is unreconcilable. Paths:

1. `stdin_ack(accepted_through)` covers the payload → `cb(offset, true)`.
2. `stdin_ack(ok=false)` arrives → `cb(offset, false)` with the typed reason.
3. Reconnect reports an offset inside a payload or below the released prefix
   → the pane-status channel receives `input-desynced`; the per-payload
   callback is not invoked because no individual payload was rejected.

Connection-level reconciliation failures are reported as the
`input-desynced` pane status with recovery text. They do not masquerade as a
payload rejection and do not permanently disable input: a new connection
clears the latch and establishes a fresh offset space.

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
