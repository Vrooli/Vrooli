import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KokoroProvider } from "./KokoroProvider";

class FakeAudio extends EventTarget {
  static last: FakeAudio | null = null;
  src = "";
  muted = false;
  paused = true;
  currentTime = 0;
  duration = NaN;
  playbackRate = 1;
  volume = 1;
  play = vi.fn().mockResolvedValue(undefined);
  pause = vi.fn(() => { this.paused = true; });
  load = vi.fn();
  removeAttribute = vi.fn(() => { this.src = ""; });
  constructor() { super(); FakeAudio.last = this; }
}

const blob = () => new Blob([new Uint8Array([1])], { type: "audio/mpeg" });
const metrics = { requestId: "test", synthStartMs: 0, totalChars: 1 };

beforeEach(() => {
  FakeAudio.last = null;
  vi.stubGlobal("Audio", FakeAudio);
  Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn(() => "blob:test") });
  Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
});

afterEach(() => {
  delete (URL as typeof URL & { createObjectURL?: unknown }).createObjectURL;
  delete (URL as typeof URL & { revokeObjectURL?: unknown }).revokeObjectURL;
  vi.unstubAllGlobals();
});

describe("KokoroProvider", () => {
  it("requires an injected runtime and exposes rich playback capabilities", async () => {
    const provider = new KokoroProvider();
    expect(provider.capabilities).toEqual({
      canPause: true,
      canSeek: true,
      canAdjustSpeed: true,
      canAdjustVolume: true,
    });
    await expect(provider.speak("hello")).rejects.toThrow("runtime is not configured");
  });

  it("uses the injected synthesis runtime and completes after audio ends", async () => {
    const synthesizeWithMetrics = vi.fn().mockResolvedValue({ blob: blob(), metrics });
    const provider = new KokoroProvider({ synthesizeWithMetrics });
    const speaking = provider.speak("hello", { voice: "af_heart", rate: 1.1 });
    await Promise.resolve();
    FakeAudio.last?.dispatchEvent(new Event("ended"));
    await speaking;
    expect(synthesizeWithMetrics).toHaveBeenCalledWith(
      "hello",
      "af_heart",
      1.1,
      expect.any(AbortSignal),
      undefined,
    );
    expect(provider.isSpeaking).toBe(false);
  });

  it("continues a sequence after an isolated failed paragraph", async () => {
    const outcomes: string[] = [];
    const synthesizeWithMetrics = vi
      .fn()
      .mockRejectedValueOnce(new Error("first paragraph failed"))
      .mockResolvedValueOnce({ blob: new Blob(), metrics })
      .mockResolvedValueOnce({ blob: new Blob(), metrics });
    const provider = new KokoroProvider({ synthesizeWithMetrics });
    provider.onParagraphOutcome = ({ outcome }) => outcomes.push(outcome);
    await provider.speakSequence(["first", "second"]);
    expect(outcomes).toContain("retried");
    expect(provider.isSpeaking).toBe(false);
  });

  it("replays a batch of cached blobs without synthesis", async () => {
    const provider = new KokoroProvider({
      synthesizeWithMetrics: vi.fn(),
    });
    await provider.speakFromBlobs([new Blob(), new Blob()]);
    expect(provider.isSpeaking).toBe(false);
  });
});
