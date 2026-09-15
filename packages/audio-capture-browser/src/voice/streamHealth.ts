// DOC: docs/internal/SEAMS.md#mic-ownership-seam
//
// Mic stream health helpers
// =========================
//
// Pure predicates for deciding whether a freshly-acquired mic stream is usable.
// Extracted from the retired low-latency "micReadiness" pre-warm module — the
// pre-warm state machine (which held a mic stream idle to shave getUserMedia
// latency) was removed because holding the mic/audio-session when idle is the
// ducking / iOS-media-wedge anti-pattern. These usability checks remain useful
// for validating a stream the provider just acquired.

/**
 * A track is usable only if it is BOTH live AND not muted.
 *
 * A MediaStreamTrack can sit at `readyState === "live"` while `muted === true`,
 * meaning no audio samples flow. The browser/OS mutes a track after sleep/wake,
 * a default-input-device change, or another app seizing the microphone — and
 * crucially the `"ended"` event does NOT fire for muting. Reusing such a stream
 * records pure silence with no error. Treating muted tracks as unusable forces a
 * fresh getUserMedia, which is what actually recovers the mic.
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
