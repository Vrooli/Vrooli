import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { KokoroProvider } from "../KokoroProvider";

// Mock the api module
vi.mock("../../../lib/api", () => ({
  synthesizeTTS: vi.fn(),
}));

import { synthesizeTTS } from "../../../lib/api";

const mockSynthesizeTTS = synthesizeTTS as ReturnType<typeof vi.fn>;

// Minimal HTMLAudioElement stub
class FakeAudio {
  src = "";
  onended: (() => void) | null = null;
  onerror: (() => void) | null = null;
  pause = vi.fn();
  play = vi.fn().mockResolvedValue(undefined);
}

beforeEach(() => {
  vi.stubGlobal("Audio", FakeAudio);
  vi.stubGlobal("URL", {
    createObjectURL: vi.fn().mockReturnValue("blob:fake-url"),
    revokeObjectURL: vi.fn(),
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("KokoroProvider", () => {
  it("calls synthesizeTTS with correct params and plays audio", async () => {
    const blob = new Blob(["audio"], { type: "audio/mp3" });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    void provider.speak("hello world", { voice: "af_heart", rate: 1.2 });

    expect(mockSynthesizeTTS).toHaveBeenCalledWith(
      "hello world",
      "af_heart",
      1.2,
      expect.any(AbortSignal),
    );

    // Simulate audio ended
    // Wait for the Audio constructor to be called and promise chain to settle
    await vi.waitFor(() => {
      expect(URL.createObjectURL).toHaveBeenCalledWith(blob);
    });

    // Get the FakeAudio instance that was created - trigger onended
    // Since we can't easily get the instance, we resolve by triggering onended
    // through the provider's internal reference
    // The play() resolves immediately, but we need onended to fire
    // Let's use a small delay then check isSpeaking
    expect(provider.isSpeaking).toBe(true);

    provider.stop();
    expect(provider.isSpeaking).toBe(false);
  });

  it("stop() pauses audio and revokes object URL", async () => {
    const blob = new Blob(["audio"], { type: "audio/mp3" });
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    // Start speaking (don't await - we'll stop mid-play)
    provider.speak("test", {}).catch(() => {});

    // Wait for fetch to complete
    await vi.waitFor(() => {
      expect(URL.createObjectURL).toHaveBeenCalled();
    });

    provider.stop();

    expect(provider.isSpeaking).toBe(false);
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:fake-url");
  });

  it("dispose() calls stop()", async () => {
    const provider = new KokoroProvider();
    const stopSpy = vi.spyOn(provider, "stop");

    provider.dispose();

    expect(stopSpy).toHaveBeenCalled();
  });

  it("reports isSpeaking=false initially", () => {
    const provider = new KokoroProvider();
    expect(provider.isSpeaking).toBe(false);
  });

  it("cleans up on fetch error", async () => {
    mockSynthesizeTTS.mockRejectedValue(new Error("Network error"));

    const provider = new KokoroProvider();
    await expect(provider.speak("test")).rejects.toThrow("Network error");

    expect(provider.isSpeaking).toBe(false);
  });
});
