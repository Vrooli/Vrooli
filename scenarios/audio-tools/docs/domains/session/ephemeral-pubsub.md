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

The session domain is intentionally process-local. Sessions and a bounded
event replay window live in an in-process `Registry` and disappear on process
restart. The cursor supports reconnects to the same live process, not durable
transcript storage or restart recovery; that belongs in a sibling domain.

## Purpose

`internal/session` (`api/internal/session/session.go:26`) owns:

- session lifecycle (open / close / barge-in),
- non-blocking subscriber fanout with explicit overflow failure and bounded
  process-local replay,
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
| `OpenSession` | `transport`, `voice`, `language` | Creates a new session, registers it, returns `session_id` + echoed `transport`. The initial `session_id` is a valid replay cursor until history rolls over. |
| `CloseSession` | `session_id`, `reason` | Emits a final `Closed` event, closes observer channels, removes from registry. |
| `SendText` | `session_id`, `text` | Emits `AssistantDelta` + `AssistantFinal` events (no actual TTS — TTS owns its own audio surface). |
| `SendCancel` | `session_id` | Triggers `BargeIn(BargeInExplicit)`. |
| `Subscribe` | `session_id`, optional `from_event_id` | Server-streaming: replays retained events after the cursor, then forwards live events. Empty cursor starts at now; an overflow close event includes an anchor to retry from. |

The session itself receives events from its transport via
`EmitEvent(SessionEvent)`. Production transports are expected to:

1. Construct a `Session` via `New(Options{...})`
   (`api/internal/session/session.go:53`),
2. Register a `CancelHook` that short-circuits in-flight TTS-out,
3. Drive `EmitEvent` for transcript/assistant/VAD/tool events,
4. Call `MarkInflight(eventID)` / `ClearInflight(eventID)` around assistant
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
| `audio_output` | `AudioFrame{Bytes, Format, SampleRate, Sequence, IsFinal}` | TTS transport |
| `provider_error` | `ProviderError{Message}` | Provider adapter |
| `barge_in_cancel` | `BargeInCancel{Reason, CanceledEventID}` | `Session.BargeIn` |
| `closed` | `SessionClosed{Reason, ResumeFromEventID}` | `Session.Close` or observer overflow |

Every event carries `EventID`, `SessionID`, and `EmittedAt` — auto-populated
when blank (`api/internal/session/session.go:128`).

The Connect `Subscribe` stream forwards events until `EventClosed` is
seen or the client disconnects (`api/handlers/session/connect_handler.go`).
When `from_event_id` is set, retained events after that ID are queued before
live events. Unknown or expired cursors return `FailedPrecondition`.
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

Fanout semantics (`api/internal/session/session.go`):

- `EmitEvent` takes the observer lock and sends to every observer
  channel without blocking the publisher.
- If an observer's bounded channel is full, Audio Tools detaches that
  observer, discards its queued events, and places a terminal
  `EventClosed` whose reason reports the unavailable event count. This
  applies both during normal fanout and when session shutdown cannot queue
  its final event. The Connect handler sends that terminal signal and returns
  `ResourceExhausted`; the close payload includes `resume_from_event_id` so
  the client has a replay anchor even if it had not yet received an event.
  Other observers remain attached and continue receiving events. Missed events
  are replayed while they remain in the 128-event / 1 MiB session history.
  Oversized events invalidate older cursors instead of bridging a gap. Replay
  is in-memory only and cannot survive an API restart.
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
| Slow observer | That observer receives terminal `EventClosed` with `observer-overflow` and the stream returns `CodeResourceExhausted`; other subscribers continue. Missed events can be replayed from the last client cursor if retained. |
| Multiple `Close` calls | Idempotent; only the first call emits `EventClosed`. A full observer queue receives an overflow terminal event instead of a successful close. |
| Barge-in with no in-flight action | No-op (`api/internal/session/session.go:161`). |
| Cancel hook panics | Not caught — propagates up the call stack of whoever invoked `BargeIn`. Transports are expected to keep hooks panic-free. |
| `EmitEvent` after `Close` | Silently dropped (`closed.Load()` short-circuits). |
| Subscriber limit reached | `Session.Subscribe` returns `ErrTooManySubscribers` after 16 observers; the Connect handler maps this to `CodeResourceExhausted`. |

## Capacity Notes

Sessions are O(observers) per event for fanout; the `RLock` is held
for the duration of the walk, so a very fat observer list will
serialize concurrent `EmitEvent` calls. In practice observer counts
are 1–3 (the driving transport plus optional debug subscribers); the
data structure is not optimized for hundreds of observers per
session.

Each observer holds a bounded event channel (the Connect handler uses 64
slots). At ~100 bytes per event that's a ~6 KiB per-observer steady-state
footprint. A session permits at most 16 observers, bounding its event buffers
to about 96 KiB. Each session also retains up to 128 cloned events or 1 MiB of
estimated payload bytes for reconnect replay. The registry still has no upper bound on session count, so
multi-tenant deployments need session admission control upstream.

Subscription creation and session close share the observer lock, so a racing
subscribe either registers before close and receives channel closure, or gets
`ErrSessionClosed`. Event fanout rechecks the closed state while holding the
read lock to prevent sends to closed channels. Cancellable contexts are watched
only while their observer exists; explicit unsubscribe and close release the
watcher. Background contexts do not start a watcher.

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
