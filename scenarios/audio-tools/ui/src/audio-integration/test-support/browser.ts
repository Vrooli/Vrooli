// Shared browser fakes for audio-integration tests. Keep these fixtures
// deliberately small: tests should override only the browser capability they
// are proving instead of rebuilding a subtly different MediaStream shape.

export type MediaTrackState = "live" | "ended";

export function makeMediaStream(
  readyState: MediaTrackState = "live",
  onStop: () => void = () => undefined,
): MediaStream {
  return {
    getTracks: () => [{ readyState, stop: onStop }],
  } as unknown as MediaStream;
}
// HOST DIFFERENCE: audio-tools test support owns this scenario's browser harness.
