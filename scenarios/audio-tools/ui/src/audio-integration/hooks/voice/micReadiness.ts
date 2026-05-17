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

/**
 * Acquire a mic stream, reusing the existing one if still alive.
 * Returns the live MediaStream.
 */
export async function acquireStream(): Promise<MediaStream> {
  if (_stream && isStreamAlive()) {
    return _stream;
  }

  _state = "acquiring";
  console.info("[voice] Low-latency: pre-warming getUserMedia");
  const start = Date.now();

  try {
    _stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  } catch {
    _state = "released";
    throw new Error("Microphone access denied");
  }

  // Listen for unexpected track termination (browser/OS revoked access)
  for (const track of _stream.getTracks()) {
    track.addEventListener("ended", () => {
      console.warn("[voice] Low-latency: mic track ended unexpectedly (readyState=%s)", track.readyState);
      _state = "released";
      _stream = null;
    }, { once: true });
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

/** Check whether the pre-warmed stream has all tracks in "live" state. */
export function isStreamAlive(): boolean {
  if (!_stream) return false;
  return _stream.getTracks().every((t) => t.readyState === "live");
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
    } else if (document.visibilityState === "visible") {
      if (opts.isLowLatencyEnabled() && !opts.isRecordingActive()) {
        console.info("[voice] Visibility: tab visible, re-acquiring mic (low-latency=true)");
        acquireStream().catch((err) => {
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
  if (_stream) {
    _stream.getTracks().forEach((t) => t.stop());
  }
  _stream = null;
  _state = "idle";
}
