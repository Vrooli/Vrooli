# Voice Latency Optimizations

Canonical explanation of the latency optimizations in the swarm-manager voice
input stack. The voice code under `ui/src/audio-integration/hooks/**` carries
`// DOC:` markers pointing at the section anchors below; this document is the
target those markers reference.

## Purpose Of This Document

Pressing the mic button and starting to speak should feel instantaneous. A
naive implementation pays several serial costs on the critical start path:

- creating and resuming an `AudioContext` (~20-50ms, and it only succeeds inside
  a user gesture),
- calling `getUserMedia` to acquire a microphone stream (~50-300ms, with a
  permission prompt on first use),
- opening the transcription WebSocket (~10-100ms),
- calibrating the voice-activity-detection (VAD) noise floor (~500ms).

Each optimization documented here moves one of those costs **off** the critical
start path — by doing the work ahead of time, caching it, or reusing it across
sessions. The optimizations are independent; "low-latency voice" mode in
settings enables the ones that hold hardware (mic pre-warm, visibility
lifecycle, stream injection), while the rest (AudioContext pre-create, WebSocket
pre-connect, capability check, noise-floor cache) apply whenever voice is on.

## WebSocket Pre-Connection

The Whisper streaming backend is reached over a WebSocket. Opening that socket
costs roughly 10-100ms. Rather than pay it when the user presses the mic button,
the socket is opened ahead of time.

- After the mount-time capability check confirms a healthy Whisper backend with
  streaming available, `useVoiceCore` instantiates a `VoiceStreamProvider` and
  calls `preConnect(language)`
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- `VoiceStreamProvider.preConnect` opens the WebSocket, flags it
  `isPreConnectedWs`, and arms a timeout
  ([CODE: ui/src/audio-integration/hooks/voice/VoiceStreamProvider.ts#preConnect]).
  It is a no-op if a socket is already open or connecting.
- `start()` reuses the pre-connected socket instead of opening a new one. The
  `isPreConnectedWs` flag is consumed on the first `start()` so a later session
  does not accidentally reuse a stale socket.
- An idle pre-connection is not held forever: if `start()` is not called within
  `PRE_CONNECT_TIMEOUT_MS` (30s), the timer closes the socket to free the
  server-side connection. A failed pre-connect (`onerror`) simply clears the
  socket and falls back to connecting on demand.

## Background Capability Check

The mic button is shown optimistically the moment voice is enabled, before the
backend has been probed, so the user never waits on a health check to start
speaking.

- On mount, `useVoiceCore` immediately sets `supported: true` /
  `backend: "whisper"` (the optimistic default) and only *then* runs the
  capability probe in the background
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- When the probe resolves, it records whether streaming is available and marks
  `capCheckResolvedRef` so `startRecording` knows the snapshot is trustworthy.
  If the probe never resolves before the user starts, the optimistic default is
  used.
- A best-effort re-probe is also kicked off on each `startRecording`. It is
  **not** awaited on the critical start path; its job is to let a stale
  "unhealthy" state recover to "healthy" on a subsequent start. Repeated
  failures past `CAP_CHECK_FAIL_THRESHOLD` trigger the fallback to the browser's
  Web Speech API.

## Pre-Create AudioContext On First Gesture

Browsers start an `AudioContext` in the `suspended` state unless it is created
or resumed inside a user-gesture handler, and they cap a page at roughly 6-8
contexts. Both constraints are handled by a single shared singleton that is
warmed on the very first gesture anywhere in the app.

- `ensureAudioContextOnGesture()` installs one-shot capture-phase `pointerdown`
  and `keydown` listeners; the first gesture creates and resumes the shared
  context, then the listeners self-remove
  ([CODE: ui/src/audio-integration/hooks/voice/sharedAudioContext.ts#ensureAudioContextOnGesture]).
  `pointerdown` is used over `click` because it fires earlier, giving the
  context a few extra milliseconds to initialize. `useVoiceCore` calls this on
  mount.
- `getSharedAudioContext()` returns the one app-lifetime context, recreating it
  only if it was closed. By the time the mic button is pressed it is normally
  already `running`, eliminating ~20-50ms from the start path.
- The context is shared by every audio consumer — level monitoring and audio
  cues both connect their own nodes to it instead of creating separate contexts,
  keeping the total at one and staying well under the browser limit.
- `startLevelMonitor` and `audioCues` both treat a non-`running` context as a
  safety-net case and resume it; the primary resume still happens inside the
  start gesture. The context is closed only by `closeSharedAudioContext()`,
  which exists for tests — in production it is never closed.

## Audio Cue Contract

Audio cues (a soft two-note start chime and a stop chime) are decoupled from the
microphone hardware lifecycle. The contract: cues play **only** while the user
is actively recording or listening, and a start cue is always followed by
exactly one stop cue for the same session.

- `cueSessionActiveRef` in `useVoiceCore` is the guard. The start cue sets it
  true; the stop cue checks it before playing and sets it false
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- All non-stop exit paths — mic pre-warm, visibility release, cleanup/dispose,
  error recovery, backend fallback, cancellation, wake-word passive listening —
  clear the guard **without** playing a stop cue. This is why mic pre-warm and
  visibility releases are silent, and why an error never plays a misleading
  "done" chime.
- Cues are pure Web Audio API oscillators with soft gain envelopes (no audio
  files) scheduled on the shared `AudioContext`, so cue playback does not create
  an extra context
  ([CODE: ui/src/audio-integration/hooks/voice/audioCues.ts]).

## Visibility-Based Mic Lifecycle

In low-latency mode a pre-warmed mic stream is held between recordings. A Page
Visibility handler releases that stream when the tab is hidden and re-acquires
it when the tab is visible again — for privacy (no background-tab mic access),
to stop the OS mic indicator, and to restore normal audio routing on mobile.

- `installVisibilityHandler` listens for `visibilitychange`
  ([CODE: ui/src/audio-integration/hooks/voice/micReadiness.ts#installVisibilityHandler]).
  On hidden it releases the pre-warmed stream; on visible it re-acquires it if
  low-latency is still enabled and no recording is active.
- An **active recording is never interrupted** by a visibility change: the
  handler checks `isRecordingActive()` first and keeps the mic if recording.
- `useVoiceCore` installs this handler (and the initial pre-warm) only while
  low-latency voice is on, and tears it down — releasing the stream — when
  low-latency is turned off
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- Platform note: mobile fires `visibilitychange` on app switch, tab switch, and
  screen lock (mic released — correct). Desktop fires it on tab switch/minimize
  but **not** when the window is merely on a second monitor (still "visible"),
  so the mic stays active there, which is correct for hands-free use on a
  secondary display.

## Stream Injection vs Stream Acquisition

The single largest start-path cost is `getUserMedia` (~50-300ms, plus the
first-use permission prompt). Low-latency mode avoids it by **injecting** a
pre-warmed stream into the provider instead of having the provider **acquire**
a fresh one.

- `micReadiness.ts` owns a module-scoped pre-warmed `MediaStream` with an
  `idle → acquiring → warm → released` state machine. It is a standalone module
  with no React dependency so it is testable and readable synchronously
  ([CODE: ui/src/audio-integration/hooks/voice/micReadiness.ts]).
- On `startRecording`, if low-latency is on and the pre-warmed stream is alive,
  `useVoiceCore` passes it to `provider.start(preWarmedStream)` and sets
  `retainStream = true` on a `VoiceStreamProvider`
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- Each provider's `start(preWarmedStream?)` injects the stream only if all its
  tracks are still `live`; otherwise it falls back to acquiring a fresh stream
  (the browser may have revoked access). This applies to the streaming provider
  ([CODE: ui/src/audio-integration/hooks/voice/VoiceStreamProvider.ts#start]),
  the one-shot Whisper provider
  ([CODE: ui/src/audio-integration/hooks/voice/WhisperProvider.ts#start]), and
  the Web Speech provider
  ([CODE: ui/src/audio-integration/hooks/voice/WebSpeechProvider.ts#start]).
- Ownership: an injected stream is used but **not** stopped by the provider when
  `retainStream` is true — `micReadiness.ts` keeps owning its lifecycle. Without
  a pre-warmed stream the provider acquires and owns one normally.

## Audio Ducking Deep Dive

On mobile, holding an open `getUserMedia` stream switches the OS audio session
into "play-and-record" mode, which ducks or pauses other audio apps (YouTube,
Spotify). The mitigation is to hold the mic for as little time as possible while
still keeping it warm enough for low latency.

- When `retainStream` is true, the provider does **not** stop the stream's
  tracks on `stop()` — the mic readiness module manages the stream instead
  ([CODE: ui/src/audio-integration/hooks/voice/VoiceStreamProvider.ts]).
- In low-latency mode, `useVoiceCore` runs a release-then-reacquire cycle when a
  turn finishes: it calls `releaseMicStream()` immediately to stop ducking, then
  re-acquires after ~500ms so the stream is warm for the next recording
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- The visibility handler is the other release point: hiding the tab releases the
  mic and restores normal audio routing (see
  [Visibility-Based Mic Lifecycle](#visibility-based-mic-lifecycle)).

## Persistent Noise Floor Cache

VAD calibrates a noise floor before it can reliably distinguish speech from
silence — roughly 500ms of listening on a cold start. Persisting that floor
across sessions lets a recording skip the calibration phase.

- The thresholds (`silenceThreshold`, `speechThreshold`) plus a timestamp are
  cached in `localStorage` under `wc-noise-floor-cache`
  ([CODE: ui/src/audio-integration/hooks/voice/vad.ts]).
- On `startRecording` with VAD enabled, `useVoiceCore` loads the cache and, if
  it is younger than `VAD_FLOOR_CACHE_MAX_AGE_MS` (24h), seeds the VAD from it
  to skip the cold calibration
  ([CODE: ui/src/audio-integration/hooks/useVoiceCore.ts]).
- Seeding is not blind trust. The sliding-window adaptation keeps running and
  self-corrects if the environment changed, and a drift guard in `vadTick`
  resets thresholds from live data if the live floor diverges from the cached
  baseline by more than `VAD_FLOOR_DRIFT_FACTOR` (3x) during the early
  calibration window.

## Cross-References

- [CODE: ui/src/audio-integration/hooks/useVoiceCore.ts] — orchestrates every
  optimization on the recording lifecycle.
- [CODE: ui/src/audio-integration/hooks/voice/micReadiness.ts] — pre-warmed mic
  stream and visibility lifecycle.
- [CODE: ui/src/audio-integration/hooks/voice/sharedAudioContext.ts] — shared
  `AudioContext` singleton and gesture pre-create.
- [CODE: ui/src/audio-integration/hooks/voice/VoiceStreamProvider.ts] — WebSocket
  pre-connection, stream injection, ducking-aware stream retention.
- [CODE: ui/src/audio-integration/hooks/voice/vad.ts] — persistent noise-floor
  cache and drift guard.
- [CODE: ui/src/audio-integration/hooks/voice/audioCues.ts] — audio cue contract.
