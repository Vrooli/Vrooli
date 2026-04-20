# Vrooli Emulator Embed Protocol

This document defines the cross-iframe contract the emulator UI emits to a
host page via `@vrooli/iframe-bridge`. A host that embeds the emulator UI in
an `<iframe>` can subscribe to these events to track session lifecycle
without re-querying the emulator API.

## Scope

The emulator UI emits **outbound** lifecycle events only. It does **not**
accept inbound control commands. If a host needs to drive the emulator (for
example to create a session on behalf of a user), it does so through the
emulator HTTP API, not through the bridge.

## Message envelope

All messages from the emulator UI to the host use a single envelope shape:

```ts
interface BridgeMessage {
  v: 1;                       // protocol version
  t: "SESSION";               // channel
  event: SessionEvent;        // payload (see below)
}
```

A host's `window` receives these messages via `window.addEventListener('message', ...)`.
The emulator calls `parent.postMessage(message, parentOrigin)`; it uses the
origin from `VITE_PARENT_ORIGIN` when set, falls back to
`new URL(document.referrer).origin`, and otherwise uses `"*"`.

## Events

Four event types are emitted. All four share the same `SessionEventPayload`
shape so hosts can render a session card without branching on type.

```ts
type SessionEventType =
  | "session.created"
  | "session.state_changed"
  | "session.error"
  | "session.destroyed";

interface SessionEventPayload {
  sessionId: string;
  status:
    | "disconnected"
    | "connecting"
    | "connected"
    | "reconnecting"
    | "failed";
  createdAt: string;          // ISO-8601, server-assigned create time
  backend: string;            // PlatformBackend identifier (e.g., "linux")
  resolution: { width: number; height: number };
  error?: { code?: string; message: string };
}

interface SessionEvent {
  type: SessionEventType;
  payload: SessionEventPayload;
}
```

### `session.created`

Fired once after a successful `POST /api/v1/sessions` response. `status` is
the initial connection status (typically `"connecting"`).

### `session.state_changed`

Fired on every connection-state transition. For terminal transitions (e.g.,
`failed`), `payload.error` may be populated.

### `session.error`

Fired on any caught error during a session lifecycle. `payload.error.message`
is always populated; `payload.error.code` is optional.

### `session.destroyed`

Fired once after a successful `DELETE /api/v1/sessions/:id`. `status` is
`"disconnected"`.

## What is **not** in the payload

- **Live metrics** (CPU, memory, FPS) — fetch via the emulator API on demand.
- **Internal URLs** (VNC WebSocket URLs, capture file paths) — not part of
  the cross-iframe contract.

The contract is *identifier + summary*: enough for a host to render a
session card without re-querying, and nothing more.

## Versioning

The envelope carries `v: 1`. Any breaking change (renamed or removed
fields, changed value semantics) bumps the version. Hosts that receive an
unexpected version should ignore the message.
