import { describe, expect, it, vi } from "vitest";
import { useTtsPlaybackIntentStore } from "./store";

describe("tts playback intent store", () => {
  it("persists intent and selected target updates", () => {
    const store = useTtsPlaybackIntentStore.getState();
    store.setPlaybackIntent("paused");
    store.setSelectedTarget({ sessionId: "s1", eventId: "e1" });
    expect(useTtsPlaybackIntentStore.getState()).toMatchObject({
      playbackIntent: "paused",
      selectedTarget: { sessionId: "s1", eventId: "e1" },
    });
    store.setSelectedTarget(null);
    store.setPlaybackIntent("continuous");
    expect(useTtsPlaybackIntentStore.getState().selectedTarget).toBeNull();
  });

  it("uses the in-memory persistence fallback when window storage is unavailable", async () => {
    vi.resetModules();
    vi.stubGlobal("window", undefined);
    const { useTtsPlaybackIntentStore: isolatedStore } = await import("./store");
    isolatedStore.getState().setPlaybackIntent("paused");
    isolatedStore.getState().setSelectedTarget({ sessionId: "ssr", eventId: "ee" });
    expect(isolatedStore.getState()).toMatchObject({
      playbackIntent: "paused",
      selectedTarget: { sessionId: "ssr", eventId: "ee" },
    });
    vi.unstubAllGlobals();
  });
});
