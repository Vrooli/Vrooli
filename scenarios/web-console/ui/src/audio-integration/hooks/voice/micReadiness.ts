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
//   2. **One owner via the registry**: The pre-warmed stream is acquired through
//      the central mic ownership registry under the "low-latency-prewarm" owner.
//      Releasing the lease (here, on OS track-end, or by page-lifecycle
//      emergency cleanup) stops the tracks and drives this module's state back
//      to "released" via the lease `onRelease` callback — so the module never
//      believes it holds a warm stream whose tracks the registry already
//      stopped.
//      DOC: docs/internal/SEAMS.md#mic-ownership-seam
//
//   3. **Stream ownership transfer**: When a pre-warmed stream is injected into
//      a provider via start(preWarmedStream), the provider uses it but does NOT
//      stop its tracks (retainStream=true). After recording finishes, this module
//      may release the stream (for audio ducking mitigation) and then re-acquire
//      it for the next recording session.
//      DOC: docs/internal/VOICE-LATENCY.md#stream-injection-vs-stream-acquisition
//
//   4. **Audio ducking mitigation**: On mobile, holding a getUserMedia stream
//      switches the OS audio session to "play-and-record" mode, which ducks or
//      pauses other audio apps (YouTube, Spotify). We minimize this by releasing
//      the stream promptly after each recording session.
//      DOC: docs/internal/VOICE-LATENCY.md#audio-ducking-deep-dive
//
// Page-visibility lifecycle (release on hidden, re-acquire on visible) is no
// longer owned here: it is one concern across ALL mic owners and lives in the
// central registry (installMicLifecycleCleanup) plus useVoiceCore's coordinated
// re-arm. DOC: docs/internal/VOICE-LATENCY.md#visibility-based-mic-lifecycle

import { acquireMicStream, releaseMicLease, type MicLease } from "./micOwnership";

export type MicReadinessState = "idle" | "acquiring" | "warm" | "released";

let _lease: MicLease | null = null;
let _state: MicReadinessState = "idle";
let _generation = 0;

/**
 * Acquire a mic stream, reusing the existing one if still alive.
 * Returns the live MediaStream.
 */
export async function acquireStream(): Promise<MediaStream> {
  if (_lease && !_lease.released && isStreamUsable(_lease.stream)) {
    return _lease.stream;
  }

  const generation = _generation;
  _state = "acquiring";
  console.info("[voice] Low-latency: pre-warming getUserMedia");
  const start = Date.now();

  let lease: MicLease;
  try {
    lease = await acquireMicStream("low-latency-prewarm", { audio: true }, {
      // Fires when this lease is released by anyone — the OS revoking the
      // device, page-lifecycle emergency cleanup, or releaseStream below. Reset
      // module state so the next use re-acquires instead of reusing dead tracks.
      onRelease: () => {
        if (_lease === lease) {
          _lease = null;
          _state = "released";
        }
      },
    });
  } catch {
    if (generation === _generation) {
      _state = "released";
    }
    throw new Error("Microphone access denied");
  }

  if (generation !== _generation) {
    releaseMicLease(lease, "owner-replaced");
    throw new Error("Microphone pre-warm cancelled");
  }

  _lease = lease;
  _state = "warm";
  console.info("[voice] Low-latency: mic pre-warmed in %dms", Date.now() - start);
  return lease.stream;
}

/**
 * Release the pre-warmed mic stream by stopping all tracks.
 * This stops the OS microphone indicator and frees audio hardware.
 */
export function releaseStream(): void {
  _generation++;
  if (_lease) {
    releaseMicLease(_lease, "manual-stop");
    _lease = null;
  }
  _state = "released";
}

/** Get the current pre-warmed stream, or null if none exists. */
export function getStream(): MediaStream | null {
  return _lease && !_lease.released ? _lease.stream : null;
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
 * a full page reload (which discards the module-scoped stream).
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
  return isStreamUsable(getStream());
}

/** Get the current state of the mic readiness module. */
export function getMicReadinessState(): MicReadinessState {
  return _state;
}

/**
 * Reset all module state. For test cleanup only — not called in production.
 */
export function _resetMicReadiness(): void {
  _generation++;
  if (_lease) {
    releaseMicLease(_lease, "test-reset");
  }
  _lease = null;
  _state = "idle";
}
