# Voice Activation Latency Optimizations

Last updated: 2026-03-31

## Problem Statement

When the user presses the mic button, recording used to take 1.5-2.5 seconds to start due to sequential async operations:

| Operation | Latency | Root cause |
|-----------|---------|------------|
| Capability liveness check | 50-500ms | Blocking HTTP round-trip in startRecording() |
| getUserMedia | 50-300ms | Browser acquires mic hardware, may show permission dialog |
| WebSocket connection | 10-100ms | TCP + WS handshake to Go backend |
| VAD calibration | 500ms | Noise floor sampling (hardcoded VAD_CALIBRATION_MS) |
| AudioContext creation | 0-50ms | Browser creates audio processing context |

Additionally, on mobile, `getUserMedia` switches the OS audio session to "play-and-record" mode, which ducks or pauses other audio apps (YouTube, Spotify) and doesn't resume them after recording stops.

## Optimization Inventory

### 1. Background capability check (always-on)

- **WHERE**: [CODE: ui/src/hooks/useVoiceInput.ts] mount effect + [CODE: ui/src/lib/api.ts#getCapabilitiesLivenessSnapshot]
- **WHAT**: The capabilities liveness check now runs on a 25-second background interval (via `refreshCapabilitiesLiveness()`). `startRecording()` reads the result synchronously from `getCapabilitiesLivenessSnapshot()` instead of awaiting a network call.
- **WHY**: Eliminates 50-500ms of blocking HTTP latency from the mic activation hot path.
- **CONSTRAINT**: The background interval must run inside the server's 30s cache TTL. If the snapshot is null (first activation before mount check resolves), the optimistic Whisper assumption applies.
- **COST**: One extra HTTP request every 25 seconds while voice is enabled.

### 2. Persistent noise floor cache (always-on)

- **WHERE**: [CODE: ui/src/hooks/voice/vad.ts#createVadRefsFromCache] + [CODE: ui/src/hooks/voice/vad.ts#loadNoiseFloorCache]
- **WHAT**: After each recording session, the VAD's current noise floor thresholds are saved to `localStorage` (key: `wc-noise-floor-cache`). On the next recording start, the VAD seeds from this cache and starts directly in `waitingForSpeech` state, skipping the 500ms calibration phase.
- **WHY**: The calibration phase exists to measure ambient noise. If the user hasn't moved to a dramatically different environment, the cached floor is a good approximation.
- **CONSTRAINT**: Cache expires after 24 hours (`VAD_FLOOR_CACHE_MAX_AGE_MS`). A drift guard detects when the live noise floor diverges from the cached floor by >3x within the first 500ms, and resets thresholds from live data immediately (no pause — VAD keeps running).
- **COST**: One `localStorage.getItem()` call on recording start (synchronous, sub-millisecond).
- **FORMAT**: `{ silenceThreshold: number, speechThreshold: number, timestamp: number }`

### 3. Pre-create AudioContext on first gesture (always-on)

- **WHERE**: [CODE: ui/src/hooks/voice/sharedAudioContext.ts]
- **WHAT**: A singleton AudioContext is created on the first user gesture (pointerdown/keydown) anywhere in the app, and shared between the level monitor and audio cues.
- **WHY**: Browsers require a user gesture to create/resume an AudioContext. Creating it on the first interaction (rather than on mic press) saves ~20-50ms. Sharing a single context also reduces the AudioContext count from 2 to 1, leaving more headroom under the browser's 6-8 context limit.
- **CONSTRAINT**: The context is app-lifetime — never closed during normal operation. Individual consumers connect/disconnect their own audio nodes without affecting the context.
- **COST**: One AudioContext in memory (minimal).

### 4. WebSocket pre-connection (always-on)

- **WHERE**: [CODE: ui/src/hooks/voice/VoiceStreamProvider.ts#preConnect]
- **WHAT**: After the mount-time capability check confirms streaming is available, the WebSocket is opened immediately (before the user presses the mic button). `start()` reuses the pre-connected WebSocket instead of opening a new one.
- **WHY**: Eliminates 10-100ms of TCP + WebSocket handshake from the recording start.
- **CONSTRAINT**: A 30-second timeout closes the pre-connected WS if `start()` isn't called, preventing idle connections on the server. If the pre-connected WS errors or times out, `start()` creates a fresh one — the existing `pendingChunks` buffering handles any gap.
- **COST**: One idle WebSocket connection while voice is enabled and streaming is available.

### 5. Low-latency voice mode (opt-in, setting: `lowLatencyVoice`)

- **WHERE**: [CODE: ui/src/hooks/voice/micReadiness.ts] + [CODE: ui/src/stores/useWorkspaceStore.ts#lowLatencyVoice]
- **WHAT**: Pre-warms `getUserMedia` so a `MediaStream` is already available when the user presses the mic button. The stream is injected into the provider via `start(preWarmedStream)`.
- **WHY OPT-IN**: `getUserMedia` activates the OS microphone indicator (red dot on iOS, orange dot on Android, tray icon on desktop). This is a privacy signal that should require user consent.
- **CONSTRAINT**: The pre-warmed stream is "provider-independent" — it is acquired by micReadiness and injected into whichever provider starts. See "Stream injection vs stream acquisition" below.
- **COST**: One active mic stream while the setting is enabled and the tab is visible.

### 6. Page-lifecycle mic cleanup (always-on for ALL mic owners)

- **WHERE**: [CODE: ui/src/audio-integration/hooks/voice/micOwnership.ts#installMicLifecycleCleanup] (privacy backstop) + [CODE: ui/src/audio-integration/hooks/useVoiceCore.ts] (coordinated stop + re-arm).
- **WHAT**: Visibility/lifecycle cleanup is no longer scoped to the low-latency
  pre-warm stream. Every browser mic stream opened by web-console UI is acquired
  through the **mic ownership registry** (one lease per owner: low-latency
  prewarm, active providers, passive wake-word, and the three settings capture
  flows). One central installer reacts to page lifecycle:
  - `visibilitychange` → hidden: release every **non-active-recording** lease
    (passive, prewarm, settings). Each owner's `onRelease` callback resets its
    own state (passive listening flips off; micReadiness goes `released`;
    settings captures cancel without uploading).
  - `pagehide` / `freeze`: release **all** leases. The PWA is closing — privacy
    and hardware release win over preserving a partial recording. MDN notes
    mobile `pagehide` is not fully reliable, so `visibilitychange` is the primary
    session-end signal and `pagehide`/`freeze` are complementary.
  - On becoming visible again, useVoiceCore re-arms passive listening and/or the
    low-latency prewarm (gated on toggles, a loaded template, no active
    recording). A lease release does not re-run React effects, so re-arm is
    explicit.
- **ACTIVE RECORDING POLICY (iOS PWA)**: an active user recording is **stopped**
  on hidden by useVoiceCore (not silently kept open). Whatever was captured is
  finalized and the mic is released; the change is surfaced via a transient
  notice rather than a stuck Dynamic Island indicator.
- **WHY**: (a) Privacy — no mic access in a hidden tab / backgrounded PWA, the
  iOS "mic indicator on while the app looks idle" failure mode. (b) Audio
  ducking — releasing the mic restores normal audio routing on mobile.
- **PLATFORM DIFFERENCES**:
  - **Mobile Safari**: `visibilitychange` fires on app switch, tab switch, and screen lock. Reliable.
  - **Chrome Android**: `visibilitychange` fires on tab switch and app switch. Reliable.
  - **Desktop browsers**: `visibilitychange` fires on tab switch. Does NOT fire when window is on a second monitor (still "visible" per spec). Mic stays active — correct for hands-free use.
- **KEY INVARIANT**: `MediaStreamTrack.stop()` is the only reliable
  application-level mic release; dropping a reference or unmounting React does
  not stop the OS microphone. Lease release is idempotent.

### 7. Prompt mic release (always-on)

- **WHERE**: [CODE: ui/src/hooks/voice/VoiceStreamProvider.ts#retainStream]
- **WHAT**: When `lowLatencyVoice` is disabled (default), mic tracks are stopped immediately after recording finishes. When enabled, the stream is retained for re-use but released promptly (500ms delay) and then re-acquired — minimizing the window where the mic causes audio ducking.
- **WHY**: On mobile, holding a `getUserMedia` stream keeps the OS audio session in "play-and-record" mode, which ducks other audio. The 120ms delay before `provider.stop()` exists to ensure the final MediaRecorder chunk is captured — it should not be reduced.

## Audio Ducking Deep Dive

`getUserMedia({ audio: true })` on mobile switches the OS audio session:
- **iOS**: `AVAudioSessionCategoryPlayAndRecord` — other audio apps duck by ~20dB or pause entirely
- **Android**: `AudioManager.MODE_IN_COMMUNICATION` — similar ducking behavior
- **Desktop**: Generally no ducking (audio sessions are per-window, not system-wide)

Effects:
- Other audio apps (YouTube, Spotify) duck volume or pause
- Bluetooth routing may switch to HFP profile (mono, low quality)
- Speaker output may switch from main speaker to earpiece

Mitigation strategy:
1. Release mic tracks ASAP after recording stops (all providers do this in `stop()`)
2. In low-latency mode, release on `visibilitychange` hidden
3. In low-latency mode, release-then-reacquire after each recording session

**Future**: The `navigator.audioSession` API (Chrome 132+, experimental) allows requesting `type: "play-and-record"` with hint `"playback"`, which may prevent ducking. Track at https://chromestatus.com/feature/5765444243898368. Not implemented yet due to insufficient browser support.

## Stream Injection vs Stream Acquisition

Two ownership models for the MediaStream:

**Stream acquisition (default)**: Each provider calls `getUserMedia` in its own `start()` method. Provider owns the stream lifecycle — it calls `track.stop()` in `stop()`.

**Stream injection (low-latency)**: A pre-warmed stream from `micReadiness.ts` is passed INTO the provider's `start(preWarmedStream)` method. The micReadiness module owns acquisition; the provider uses the stream but does not stop its tracks (`retainStream = true`).

Key invariants:
- If the pre-warmed stream's tracks are ended (browser revoked access), the provider falls back to its own `getUserMedia` call
- The provider checks `track.readyState === "live"` before using an injected stream
- Ownership is tracked by a **lease**, not the `retainStream` flag. A provider
  holds a lease only for a stream it acquired itself (via the mic ownership
  registry); an injected pre-warmed stream's lease stays with micReadiness. On
  `stop()` / `dispose()` the provider releases only its own lease — it never
  stops another owner's tracks. (This also closes a latent leak where a fresh
  `getUserMedia` fallback ran while `retainStream` was still true.)
- The micReadiness module handles the release-then-reacquire cycle after recording finishes

## Audio Cue Contract

Audio cues (rising/falling chimes) signal recording session boundaries to the user. They are scoped to the **logical recording session**, not the **mic hardware lifecycle**. This distinction is critical because low-latency mode pre-warms the mic before recording starts.

### When cues play

| Event | Start cue | Stop cue |
|-------|-----------|----------|
| Recording starts (user pressed mic, provider ready) | Yes | — |
| Recording stops (user pressed stop) | — | Yes |
| VAD auto-stop (silence timeout) | — | Yes |
| VAD no-speech (no speech detected) | — | Yes |
| Abort during startup (stop requested while starting) | Already played | Yes |

### When cues must NOT play

| Event | Reason |
|-------|--------|
| Mic pre-warm (low-latency acquireStream) | Not a recording session |
| Mic release (visibility handler, cleanup) | Not a user-initiated stop |
| Component unmount / app close | Lifecycle event, not recording stop |
| Error recovery / backend fallback | Error, not normal completion |
| Transcription cancellation | Stop cue already played at recording end |
| Wake word passive listening | Different mode, not a recording session |

### Implementation: cue session guard

A `cueSessionActiveRef` in `useVoiceInput.ts` enforces the contract:

1. **Start cue**: Sets `cueSessionActiveRef = true`, then plays the chime
2. **Stop cue**: Checks `cueSessionActiveRef`; if true, sets it to false and plays the chime
3. **Non-stop exits** (cleanup, error, cancel, onResult): Set `cueSessionActiveRef = false` WITHOUT playing the stop cue

This guarantees:
- Cues are always paired (every start has exactly one stop)
- No phantom stop cues on lifecycle events
- No double-stop from redundant `stopRecording()` calls

### Related files

- [CODE: ui/src/hooks/voice/audioCues.ts] — cue playback implementation
- [CODE: ui/src/hooks/useVoiceInput.ts] — cue session guard (`cueSessionActiveRef`)
- [CODE: ui/src/hooks/__tests__/audioCueContract.test.ts] — regression tests

## Expected Latency

| Operation | Before | After (default) | After (low-latency ON) |
|-----------|--------|-----------------|----------------------|
| Capability check | 50-500ms | 0ms (cached) | 0ms |
| getUserMedia | 50-300ms | 50-300ms | 0ms (pre-warmed) |
| WebSocket connect | 10-100ms | 0ms (pre-connected) | 0ms |
| VAD calibration | 500ms | 0ms (cached floor) | 0ms |
| AudioContext | 0-50ms | 0ms (pre-created) | 0ms |
| **Total** | **660-1950ms** | **50-300ms** | **~0ms** |
