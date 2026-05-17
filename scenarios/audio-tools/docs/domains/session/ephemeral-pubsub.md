# Session Ephemeral Pub/Sub

This document is the canonical architecture reference for the
voice-session abstraction: a duplex audio interaction with
multi-observer pub/sub, barge-in coordination, and a
transport-pluggable boundary.

Read this first when:

- adding a new session transport (currently `Subscribe` via
  Connect-RPC server-streaming; planned WS),
- introducing a new `EventType`,
- changing barge-in semantics or the cancel-hook contract,
- debugging "why did my observer miss an event under load?".

The session domain is intentionally ephemeral — there is no
persistence. Sessions live in an in-process `Registry` and disappear
on process restart. If durable transcript storage becomes a
requirement, that lives in a sibling domain, not here.

## Purpose

`internal/session` (`api/internal/session/session.go:26`) owns:

- session lifecycle (open / close / barge-in),
- subscriber fanout with drop-on-backpressure semantics,
- the typed `SessionEvent` envelope
  (`api/internal/session/events.go:30`),
- barge-in cancellation: a single in-flight assistant action id
  (`inflightID atomic.Value`) and a transport-provided `CancelHook`.

`handlers/session` (`api/handlers/session/handler.go:28`) is the
Connect-RPC translation shim. It owns no session state — every method
resolves the session from `Registry.Get(session_id)` and delegates.

## Inputs

`SessionService` exposes five Connect methods
(`api/handlers/session/handler.go:139`):

| Method | Inputs | Effect |
|---|---|---|
| `OpenSession` | `transport`, `voice`, `language` | Creates a new session, registers it, returns `session_id` + echoed `transport`. |
| `CloseSession` | `session_id`, `reason` | Emits a final `Closed` event, closes observer channels, removes from registry. |
| `SendText` | `session_id`, `text` | Emits `AssistantDelta` + `AssistantFinal` events (no actual TTS — TTS owns its own audio surface). |
| `SendCancel` | `session_id` | Triggers `BargeIn(BargeInExplicit)`. |
| `Subscribe` | `session_id` | Server-streaming: forwards every `SessionEvent` until close or client disconnect. |

The session itself receives events from its transport via
`EmitEvent(SessionEvent)`. Production transports are expected to:

1. Construct a `Session` via `New(Options{...})`
   (`api/internal/session/session.go:53`),
2. Register a `CancelHook` that short-circuits in-flight TTS-out,
3. Drive `EmitEvent` for transcript/assistant/VAD/tool events,
4. Call `MarkInflight(eventID)` / `ClearInflight()` around assistant
   actions so barge-in has a target,
5. Call `BargeIn(reason)` when VAD reports `speech_start` mid-flight.

The currently shipping transport is the WS handler under
`handlers/stt/stream_ws.go` (planned in PRD P0; see
[`../stt/streaming-pipeline.md`](../stt/streaming-pipeline.md)).
`Subscribe` is the read-only fanout transport for observers that
want to attach without driving audio.

## Outputs

Every observer receives `SessionEvent` values
(`api/internal/session/events.go:30`):

| `Type` | Payload field | Emitted by |
|---|---|---|
| `transcript_delta` | `TranscriptDelta{Text, FromSeconds, ToSeconds}` | Transport (during STT) |
| `transcript_final` | `TranscriptFinal{Text, DurationSeconds, SpeakerVerified}` | Transport (on segment commit) |
| `assistant_delta` | `AssistantDelta{Text}` | `SendText` handler; transport (during LLM stream) |
| `assistant_final` | `AssistantFinal{Text, HadAudio}` | `SendText` handler; transport (on assistant turn end) |
| `vad` | `VADEvent{State}` (`speech_start` \| `speech_end`) | Transport |
| `tool` | `ToolEvent{Name, PayloadJSON}` | Transport (tool-use events) |
| `barge_in_cancel` | `BargeInCancel{Reason, CanceledEventID}` | `Session.BargeIn` |
| `closed` | `SessionClosed{Reason}` | `Session.Close` |

Every event carries `EventID`, `SessionID`, and `EmittedAt` — auto-populated
when blank (`api/internal/session/session.go:128`).

The Connect `Subscribe` stream forwards events until `EventClosed` is
seen or the client disconnects (`api/handlers/session/handler.go:108`).
The handler maps to `sessv1` protos via `toProto`
(`api/handlers/session/mappers.go`).

## Internal Chain

```
OpenSession (Connect)               BargeIn flow:
        │                              ▼
        ▼                          transport (WS) detects VAD speech_start
intsession.New(Options)               │
        │                              ▼
        ▼                          session.BargeIn(BargeInVAD)
Registry.Add                          │
        │                              ▼
        ▼                          ┌──────────────────────────────────────┐
return session_id                  │ load inflightID; if empty → no-op    │
                                   │ store("")                            │
Subscribe (Connect, server-stream) │ cancelHook(reason, canceledEventID)  │
        │                          │ EmitEvent(BargeInCancel{...})        │
        ▼                          └──────────────────────────────────────┘
Registry.Get                          │
        │                              │ (fanout to every observer)
        ▼                              ▼
Session.Subscribe(ctx, 64)         observer channels (drop-on-full)
        │
        ▼
for ev := range ch:
    stream.Send(toProto(ev))
    if ev.Type == EventClosed: return
```

Fanout semantics (`api/internal/session/session.go:124`):

- `EmitEvent` acquires the registry RLock and walks every observer
  channel via a non-blocking send (`select { case ch <- evt: default: }`).
- A slow observer drops events; the publisher never blocks. The
  contract is "observers keep up or accept loss" — there is no retry
  queue and no per-observer backpressure protocol.
- Observer buffer size is 64 by default (`Subscribe(ctx, 64)`,
  `api/handlers/session/handler.go:113`); the entry point accepts a
  per-call override.

`Close` is idempotent (`atomic.Bool.Swap`,
`api/internal/session/session.go:178`). A second `Close` is a no-op.
It synthesizes a final `EventClosed` and sends it to every observer
under the write lock before closing the channels.

`Unsubscribe` auto-fires when the subscriber's context cancels — the
goroutine spawned in `Subscribe`
(`api/internal/session/session.go:95`) waits on `ctx.Done()` and
calls `Unsubscribe(key)`. There is no leak when an observer's
context expires.

## Seams

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| Session registry | `*Registry` (`api/internal/session/session.go:201`) | In-process `map[id]*Session` | Tests construct their own registry directly |
| Cancel hook | `Options.CancelHook func(reason, eventID)` | Transport closure that drops the in-flight TTS-out channel | Per-test closure capturing into a slice (`session_test.go`) |
| Observer channel | `chan SessionEvent` | Buffered (default 64) | Tests subscribe and drain manually |
| Transport | Not a Go interface — discipline only | WS handler + Connect `Subscribe` | `OpenSession` accepts `transport="fake"` (`api/handlers/session/handler.go:42`) so tests can open sessions without a real WS |

The fact that "transport" is discipline rather than a typed interface
is intentional — `Session` exposes the primitives transports need
(`EmitEvent`, `MarkInflight`, `ClearInflight`, `BargeIn`) and trusts
them to use them correctly. Codifying a `Transport` interface would
not buy anything; the surface is small enough that the WS handler
just calls the methods.

## Failure Modes

| Cause | Behavior |
|---|---|
| `OpenSession` with blank `transport` | Defaults to `"fake"` (`api/handlers/session/handler.go:42`) — convenience for tests; production transports always set it. |
| `Subscribe` on closed session | `Session.Subscribe` returns `ErrSessionClosed`; handler maps to `CodeResourceExhausted` (`api/handlers/session/handler.go:115`) — current mapping is imprecise (should be `FailedPrecondition` or `NotFound`); see [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md). |
| `CloseSession` / `SendText` / `SendCancel` with unknown id | `Registry.Get` returns `ErrUnknownSession`; handler maps to `CodeNotFound`. |
| `SendText` with empty text | `CodeInvalidArgument`. |
| Slow observer | Events drop silently — by design (drop-on-full). No log line; observers are responsible for noticing missed sequence ids. |
| Multiple `Close` calls | Idempotent; only the first call emits `EventClosed`. |
| Barge-in with no in-flight action | No-op (`api/internal/session/session.go:161`). |
| Cancel hook panics | Not caught — propagates up the call stack of whoever invoked `BargeIn`. Transports are expected to keep hooks panic-free. |
| `EmitEvent` after `Close` | Silently dropped (`closed.Load()` short-circuits). |
| Subscriber limit reached | `ErrTooManySubscribers` is declared but never returned in the current code path; the limit is effectively unbounded (one channel per subscriber, fanout iterates the map). If you need a cap, add it to `Subscribe`. |

## Capacity Notes

Sessions are O(observers) per event for fanout; the `RLock` is held
for the duration of the walk, so a very fat observer list will
serialize concurrent `EmitEvent` calls. In practice observer counts
are 1–3 (the driving transport plus optional debug subscribers); the
data structure is not optimized for hundreds of observers per
session.

Each observer holds a 64-deep channel. At ~100 bytes per event
that's a ~6 KiB per-observer steady-state footprint. There is no
upper bound on the number of sessions or subscribers in the
registry; operators relying on this in a multi-tenant context should
add an admission control layer upstream.

The registry uses a single `RWMutex`; under heavy churn (many
short-lived sessions) the lock can become hot. The current shape is
optimal for the expected workload (a handful of long-lived
sessions); fragmenting the registry by shard is a known future
optimization, not a current need.

There is no persistence — restarting the API drops every session.
Clients that need session continuity must re-open and re-establish
observers. This is by design: voice sessions are inherently
short-lived and the failure mode of a stale resumed session
(replaying old audio) is worse than the failure mode of a clean
restart.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — ephemeral-by-design decision
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — Subscribe error-mapping drift
- [`../../reference/configuration.md`](../../reference/configuration.md) — session-related env vars
- [`../stt/streaming-pipeline.md`](../stt/streaming-pipeline.md) — where the WS transport will hook the session
- `packages/proto/schemas/audio-tools/v1/session/session.proto` — wire shape
