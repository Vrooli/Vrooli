import { describe, expect, it, vi } from "vitest";
import { WhisperProvider } from "./WhisperProvider";

describe("WhisperProvider lifecycle", () => {
  it("starts empty, exposes no stream, and cleans up retained audio safely", () => {
    const provider = new WhisperProvider();
    expect(provider.getStream()).toBeNull();
    expect(provider.getLastTurnAudio()).toBeNull();
    provider.dropTail();
    provider.stop();
    provider.disposeLastTurn();
    provider.dispose();
    expect(provider.getLastTurnAudio()).toBeNull();
  });

  it("surfaces microphone acquisition failure without creating a recorder", async () => {
    const onError = vi.fn();
    const getUserMedia = vi.fn().mockRejectedValue(new DOMException("denied", "NotAllowedError"));
    Object.defineProperty(navigator, "mediaDevices", {
      value: { getUserMedia },
      configurable: true,
    });
    const provider = new WhisperProvider();
    provider.onError = onError;

    await provider.start();

    expect(getUserMedia).toHaveBeenCalledWith({ audio: true });
    expect(onError).toHaveBeenCalledWith(expect.stringContaining("denied"));
    expect(provider.getStream()).toBeNull();
  });
});
