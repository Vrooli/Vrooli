export interface MediaSessionController {
  metadata: MediaMetadata | null;
  playbackState: MediaSessionPlaybackState;
  setActionHandler(action: MediaSessionAction, handler: (() => void) | null): void;
  setPositionState(details: MediaPositionState): void;
}

export interface MediaSessionSetup {
  title: string;
  artist: string;
  album: string;
  isPaused: boolean;
  duration: number | null;
  currentTime: number;
  playbackRate: number;
  handlers: Partial<Record<MediaSessionAction, () => void>>;
}

const MEDIA_ACTIONS: MediaSessionAction[] = [
  "play",
  "pause",
  "stop",
  "seekbackward",
  "seekforward",
  "previoustrack",
  "nexttrack",
];

/** Configure lock-screen/headphone transport and return a complete cleanup. */
export function setupMediaSession(mediaSession: MediaSessionController, setup: MediaSessionSetup): () => void {
  try {
    mediaSession.metadata = new MediaMetadata({ title: setup.title, artist: setup.artist, album: setup.album });
  } catch {
    // Metadata construction is unavailable in some embedded browsers.
  }
  for (const action of MEDIA_ACTIONS) {
    try {
      mediaSession.setActionHandler(action, setup.handlers[action] ?? null);
    } catch {
      // Unsupported actions are normal across browser implementations.
    }
  }
  try {
    mediaSession.playbackState = setup.isPaused ? "paused" : "playing";
  } catch {
    // Some webviews expose handlers but reject playbackState writes.
  }
  if (setup.duration && Number.isFinite(setup.duration) && setup.duration > 0) {
    try {
      mediaSession.setPositionState({
        duration: setup.duration,
        playbackRate: setup.playbackRate || 1,
        position: Math.min(setup.duration, Math.max(0, setup.currentTime)),
      });
    } catch {
      // Invalid position state is browser-specific and must not affect audio.
    }
  }
  return () => {
    for (const action of MEDIA_ACTIONS) {
      try {
        mediaSession.setActionHandler(action, null);
      } catch {
        // Unsupported actions are normal across browser implementations.
      }
    }
    try { mediaSession.metadata = null; } catch { /* best effort */ }
    try { mediaSession.playbackState = "none"; } catch { /* best effort */ }
  };
}
