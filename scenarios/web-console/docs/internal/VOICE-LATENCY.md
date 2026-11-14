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

<a id="audiocontext-lifecycle"></a>
### 3. AudioContext lifecycle — lazy resume, idle suspend (always-on)

- **WHERE**: [CODE: ui/src/audio-integration/hooks/voice/sharedAudioContext.ts]
- **WHAT**: A single shared AudioContext is used by the level monitor and audio cues. It is created/resumed **lazily, inside the real voice or cue gesture** (mic press, record cue, passive listener) — **never eagerly** on an arbitrary first tap. When it goes idle it is **suspended**: on page background (`visibilitychange:hidden`) and shortly (`armIdleSuspend`, ~1.5s) after a capture turn ends. Any real audio need calls `keepAudioContextAwake()` (cancel pending suspend) and resumes.
- **WHY**: On iOS, creating/resuming an AudioContext activates the app's `AVAudioSession` and **interrupts other apps' audio** (Spotify/YouTube). Eagerly resuming on the first interaction hijacked the audio session even when the user never used voice, and a running-but-idle context kept holding the session (the "web-console stops my music the moment I touch it" report). Saving ~20-50ms of first-press latency is not worth that; we activate the session only while actually capturing or cueing. See [PROBLEMS.md §8b].
- **CONSTRAINT**: Consumers resume on demand, so a suspended-when-idle context is transparent to them. Rebuild-on-closed/interrupted (see `ensureRunningSharedAudioContext`) still heals a wedged context.
- **COST**: A tiny first-press resume cost (previously prepaid); in exchange, background audio is never held hostage.

### 4. WebSocket pre-connection (always-on)

- **WHERE**: [CODE: ui/src/hooks/voice/VoiceStreamProvider.ts#preConnect]
- **WHAT**: After the mount-time capability check confirms streaming is available, the WebSocket is opened immediately (before the user presses the mic button). `start()` reuses the pre-connected WebSocket instead of opening a new one.
- **WHY**: Eliminates 10-100ms of TCP + WebSocket handshake from the recording start.
- **CONSTRAINT**: A 30-second timeout closes the pre-connected WS if `start()` isn't called, preventing idle connections on the server. If the pre-connected WS errors or times out, `start()` creates a fresh one — the existing `pendingChunks` buffering handles any gap.
- **COST**: One idle WebSocket connection while voice is enabled and streaming is available.

### 5. Low-latency voice mode — REMOVED (2026-07)

The opt-in `lowLatencyVoice` setting and its `micReadiness` pre-warm module were
**removed**. It held a `getUserMedia` stream open (idle) after a mic-control
intent to shave the getUserMedia call off the first press. Holding the mic idle
is the exact audio-session/ducking anti-pattern that interrupts other apps' audio
and churns the iOS media session — a plausible contributor to the "mic wedged
until reboot" class. The provider now always acquires (and owns) a **fresh** mic
stream on press; there is no pre-warm, no injected stream, no `retainStream`. If
first-press latency ever needs shaving again, do it without holding hardware idle.

### 6. Page-lifecycle mic cleanup (always-on for ALL mic owners)

- **WHERE**: [CODE: ui/src/audio-integration/hooks/voice/micLifecyclePolicy.ts#decideMicLifecycle] (pure policy) + [CODE: ui/src/audio-integration/hooks/voice/micOwnership.ts#installMicLifecycleCleanup] (privacy backstop) + [CODE: ui/src/audio-integration/hooks/voice/voiceCaptureController.ts] (single-authority capture cleanup) + [CODE: ui/src/audio-integration/hooks/useVoiceCore.ts] (coordinated stop + re-arm + registry-driven self-heal).
- **WHAT**: Visibility/lifecycle cleanup is no longer scoped to the low-latency
  pre-warm stream. Every browser mic stream opened by web-console UI is acquired
  through the **mic ownership registry** (one lease per owner: active providers,
  passive wake-word, and the three settings capture flows). The reaction to each
  page-lifecycle event is decided by the pure
  `decideMicLifecycle({ event, standalonePwa })` policy and applied by the
  ref-counted backstop installer:
  - `visibilitychange` → hidden, **standalone/PWA** (`navigator.standalone` or
    `display-mode: standalone`): release **ALL** leases, including the active
    recording. iOS keeps the OS mic indicator (Dynamic Island) active otherwise,
    even after the JS believes it stopped — this is the failure class the policy
    closes.
  - `visibilitychange` → hidden, **desktop tab**: release every **non-active**
    lease (passive, settings); the controller stops the active recording so
    React state stays honest. Each owner's `onRelease` callback resets its own
    state (passive listening flips off; settings captures cancel without
    uploading). The shared AudioContext is also suspended on hidden (§3).
  - `pagehide` / `freeze`: release **all** leases everywhere. The PWA is closing
    — privacy and hardware release win over preserving a partial recording. MDN
    notes mobile `pagehide` is not fully reliable, so `visibilitychange` is the
    primary session-end signal and `pagehide`/`freeze` are complementary.
  - On becoming visible again, useVoiceCore does **not** re-arm passive
    listening or the audio session by itself. Visibility/focus is not a user
    mic intent; the next mic-control hover/focus/press re-arms passive wake-word
    listening if that setting is enabled.
- **PREPARING / START CANCELLATION**: `getUserMedia` may resolve a live lease
  *after* the tab goes hidden (async startup). The controller stamps a
  generation token at `beginStart()`; the hidden handler calls `cancelStarts()`,
  and when the late start resolves it sees a stale token and shuts the provider
  down (releasing the just-acquired lease) instead of entering the recording
  state. So `preparing` is treated as capture-active for lifecycle safety.
- **REGISTRY-DRIVEN SELF-HEAL**: useVoiceCore subscribes to the registry. If a
  live lease exists that the workflow should not be holding (UI idle/off but a
  provider/passive stream is still live — `selectStaleLeases`), it flips
  the honest `staleLiveMicLease` flag and self-heals via
  `controller.recoverStaleLeases` (releasing the orphan + logging a structured
  invariant violation). The mic button exposes a user "release microphone"
  recovery affordance for the same mismatch.
- **ACTIVE RECORDING POLICY (iOS PWA)**: an active user recording is **stopped**
  on hidden by useVoiceCore (not silently kept open) and, in standalone/PWA, its
  lease is also released by the backstop. Whatever was captured is finalized and
  the mic is released; the change is surfaced via a transient notice rather than
  a stuck Dynamic Island indicator.
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

- **WHERE**: [CODE: ui/src/audio-integration/hooks/voice/VoiceStreamProvider.ts]
- **WHAT**: Mic tracks are stopped immediately after a recording turn finishes (the provider releases its own registry lease), and the shared AudioContext is suspended shortly after (see §3). There is no retained/re-acquired stream — every turn acquires fresh and releases fully.
- **WHY**: On mobile, holding a `getUserMedia` stream (or a running AudioContext) keeps the OS audio session in "play-and-record" mode, which ducks other audio. Releasing both when idle is what stops the ducking. The 120ms delay before `provider.stop()` exists to ensure the final MediaRecorder chunk is captured — it should not be reduced.

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
2. **Never hold the mic or the audio session idle.** No pre-warm; the AudioContext
   is resumed lazily and suspended when idle / on background (§3). The audio
   session is active only while actually capturing or playing a cue.

**Future**: The `navigator.audioSession` API (Chrome 132+, experimental) allows requesting `type: "play-and-record"` with hint `"playback"`, which may prevent ducking. Track at https://chromestatus.com/feature/5765444243898368. Not implemented yet due to insufficient browser support.

## Stream Acquisition

Each provider calls `getUserMedia` (through the mic ownership registry) in its own
`start()` method and owns the resulting stream via a **lease** — it calls
`releaseMicLease` (→ `track.stop()`) in `stop()`/`dispose()`. There is one
ownership model: acquire-fresh-per-turn. The former "stream injection" model
(a pre-warmed stream owned by a separate module and injected via
`start(preWarmedStream)`) was removed with low-latency mode — providers no longer
accept an injected stream and there is no `retainStream` flag.

## Audio Cue Contract

Audio cues (rising/falling chimes) signal recording session boundaries to the user. They are scoped to the **logical recording session**, not the **mic hardware lifecycle** — the two are decoupled so a cue never fires on a non-recording audio event.

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
