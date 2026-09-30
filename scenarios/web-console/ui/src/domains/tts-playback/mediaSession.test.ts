import { describe, expect, it, vi } from "vitest";
import { setupMediaSession } from "./mediaSession";

describe("setupMediaSession", () => {
  it("registers transport handlers, position state, and clears everything", () => {
    vi.stubGlobal("MediaMetadata", class { constructor(public readonly init: MediaMetadataInit) {} });
    const handlers = new Map<string, (() => void) | null>();
    const mediaSession = {
      metadata: null,
      playbackState: "none" as MediaSessionPlaybackState,
      setActionHandler: vi.fn((action: MediaSessionAction, handler: (() => void) | null) => handlers.set(action, handler)),
      setPositionState: vi.fn(),
    };
    const next = vi.fn();
    const cleanup = setupMediaSession(mediaSession, {
      title: "Reply",
      artist: "agent",
      album: "conversation",
      isPaused: false,
      duration: 20,
      currentTime: 4,
      playbackRate: 1.25,
      handlers: { nexttrack: next },
    });
    expect(mediaSession.metadata).toBeTruthy();
    expect(mediaSession.playbackState).toBe("playing");
    expect(mediaSession.setPositionState).toHaveBeenCalledWith({ duration: 20, playbackRate: 1.25, position: 4 });
    handlers.get("nexttrack")?.();
    expect(next).toHaveBeenCalledOnce();
    cleanup();
    expect(mediaSession.metadata).toBeNull();
    expect(mediaSession.playbackState).toBe("none");
    expect(handlers.get("nexttrack")).toBeNull();
    vi.unstubAllGlobals();
  });
});
