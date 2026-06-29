// DOC: docs/internal/VOICE-LATENCY.md
//
// Microphone Readiness Module
// ============================
//
// Manages a pre-warmed MediaStream for low-latency voice mode. When the user
// enables "Low-latency voice" in settings, this module keeps a mic stream ready
// so that pressing the mic button skips the getUserMedia call (~50-300ms).
//
// State machine: idle → acquiring → warm → released
//
// Key design decisions:
//
//   1. **Standalone module (no React dependency)**: All state is module-scoped,
//      making it testable without rendering React components. The useVoiceInput
//      hook reads from this module synchronously.
//
//   2. **Stream ownership transfer**: When a pre-warmed stream is injected into
//      a provider via start(preWarmedStream), the provider uses it but does NOT
//      stop its tracks (retainStream=true). After recording finishes, this module
//      may release the stream (for audio ducking mitigation) and then re-acquire
//      it for the next recording session.
//      DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
//
//   3. **Visibility lifecycle**: When the tab becomes hidden, the pre-warmed
//      stream is released (saves resources, stops OS mic indicator). When the
//      tab becomes visible again, it is re-acquired. Active recordings are
//      NEVER interrupted by visibility changes.
//      DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle
//
//   4. **Audio ducking mitigation**: On mobile, holding a getUserMedia stream
//      switches the OS audio session to "play-and-record" mode, which ducks or
//      pauses other audio apps (YouTube, Spotify). We minimize this by releasing
//      the stream promptly after each recording session.
//      DOC: docs/internal/VOICE-LATENCY.md#audio-ducking-deep-dive

export type MicReadinessState = "idle" | "acquiring" | "warm" | "released";

let _stream: MediaStream | null = null;
let _state: MicReadinessState = "idle";
let _generation = 0;

/**
 * Acquire a mic stream, reusing the existing one if still alive.
 * Returns the live MediaStream.
 */
export async function acquireStream(): Promise<MediaStream> {
  if (_stream && isStreamAlive()) {
    return _stream;
  }

  const generation = _generation;
  _state = "acquiring";
  console.info("[voice] Low-latency: pre-warming getUserMedia");
  const start = Date.now();

  let stream: MediaStream;
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  } catch {
    if (generation === _generation) {
      _state = "released";
    }
    throw new Error("Microphone access denied");
  }

  if (generation !== _generation) {
    stream.getTracks().forEach((t) => t.stop());
    throw new Error("Microphone pre-warm cancelled");
  }

  _stream = stream;

  // Listen for unexpected track termination (browser/OS revoked access)
  for (const track of _stream.getTracks()) {
    track.addEventListener("ended", () => {
      console.warn("[voice] Low-latency: mic track ended unexpectedly (readyState=%s)", track.readyState);
      _state = "released";
      _stream = null;
    }, { once: true });
    // Muting (OS/another app seized the device, sleep/wake, device change) does
    // NOT fire "ended" and leaves readyState "live". Log it so a wedged stream
    // is diagnosable; point-of-use validation (isStreamUsable) re-acquires.
    track.addEventListener("mute", () => {
      console.warn("[voice] Low-latency: mic track muted (readyState=%s) — will re-acquire on next use", track.readyState);
    });
    track.addEventListener("unmute", () => {
      console.info("[voice] Low-latency: mic track unmuted (readyState=%s)", track.readyState);
    });
  }

  _state = "warm";
  console.info("[voice] Low-latency: mic pre-warmed in %dms", Date.now() - start);
  return _stream;
}

/**
 * Release the pre-warmed mic stream by stopping all tracks.
 * This stops the OS microphone indicator and frees audio hardware.
 */
export function releaseStream(): void {
  _generation++;
  if (_stream) {
    _stream.getTracks().forEach((t) => t.stop());
    _stream = null;
  }
  _state = "released";
}

/** Get the current pre-warmed stream, or null if none exists. */
export function getStream(): MediaStream | null {
  return _stream;
}

/**
 * A track is usable only if it is BOTH live AND not muted.
 *
 * A MediaStreamTrack can sit at `readyState === "live"` while `muted === true`,
 * meaning no audio samples flow. The browser/OS mutes a track after sleep/wake,
 * a default-input-device change, or another app seizing the microphone — and
 * crucially the `"ended"` event does NOT fire for muting. A retained pre-warmed
 * stream can therefore stay muted indefinitely, and reusing it records pure
 * silence: no transcript, no level meter, and no error (silence is not a
 * failure). Treating muted tracks as unusable forces a fresh getUserMedia,
 * which is what actually recovers the mic. Without this check the only fix was
 * a full page reload (which discards the module-scoped `_stream`).
 */
export function isTrackUsable(track: MediaStreamTrack): boolean {
  return track.readyState === "live" && !track.muted;
}

/** Whether every track of `stream` is live and unmuted (see isTrackUsable). */
export function isStreamUsable(stream: MediaStream | null | undefined): boolean {
  if (!stream) return false;
  const tracks = stream.getTracks();
  return tracks.length > 0 && tracks.every(isTrackUsable);
}

/**
 * Check whether the pre-warmed stream is usable (all tracks live AND unmuted).
 * Named "alive" for historical callers; muted-but-live tracks are NOT alive for
 * reuse purposes — see isTrackUsable.
 */
export function isStreamAlive(): boolean {
  return isStreamUsable(_stream);
}

/** Get the current state of the mic readiness module. */
export function getMicReadinessState(): MicReadinessState {
  return _state;
}

/**
 * Install a Page Visibility API handler that releases the mic when the tab
 * is hidden and re-acquires it when the tab becomes visible.
 *
 * This is critical for two reasons:
 *   1. Privacy — no mic access in background tabs
 *   2. Audio ducking — releasing the mic restores normal audio routing on mobile
 *
 * The handler NEVER releases the mic during active recording. The
 * `isRecordingActive` callback lets the caller (useVoiceInput) provide
 * real-time recording state.
 *
 * Platform behavior:
 *   - Mobile (iOS Safari, Chrome Android): visibilitychange fires on app switch,
 *     tab switch, and screen lock. Mic is released. Correct behavior.
 *   - Desktop: visibilitychange fires on tab switch/minimize. Does NOT fire when
 *     the window is on a second monitor (still "visible"). Mic stays active —
 *     correct for hands-free use on a secondary display.
 *
 * DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle
 *
 * @returns Cleanup function that removes the visibility listener.
 */
export function installVisibilityHandler(opts: {
  isRecordingActive: () => boolean;
  isLowLatencyEnabled: () => boolean;
}): () => void {
  const handler = () => {
    if (document.visibilityState === "hidden") {
      if (opts.isRecordingActive()) {
        console.info("[voice] Visibility: tab hidden during active recording, keeping mic");
        return;
      }
      if (_stream) {
        console.info("[voice] Visibility: tab hidden, releasing mic tracks");
        releaseStream();
      }
    } else {
      // document.visibilityState === "visible"
      if (opts.isLowLatencyEnabled() && !opts.isRecordingActive()) {
        console.info("[voice] Visibility: tab visible, re-acquiring mic (low-latency=true)");
        acquireStream().catch((err: unknown) => {
          console.warn("[voice] Visibility: failed to re-acquire mic:", err);
        });
      }
    }
  };

  document.addEventListener("visibilitychange", handler);
  return () => document.removeEventListener("visibilitychange", handler);
}

/**
 * Reset all module state. For test cleanup only — not called in production.
 */
export function _resetMicReadiness(): void {
  _generation++;
  if (_stream) {
    _stream.getTracks().forEach((t) => t.stop());
  }
  _stream = null;
  _state = "idle";
}
