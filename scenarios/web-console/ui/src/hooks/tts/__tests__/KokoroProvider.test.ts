import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { KokoroProvider } from "../KokoroProvider";

vi.mock("../../../lib/api", () => ({
  synthesizeTTS: vi.fn(),
}));

import { synthesizeTTS } from "../../../lib/api";

const mockSynthesizeTTS = synthesizeTTS as ReturnType<typeof vi.fn>;

class FakeBufferSource {
  buffer: AudioBuffer | null = null;
  onended: (() => void) | null = null;
  connect = vi.fn();
  disconnect = vi.fn();
  stop = vi.fn();
  start = vi.fn((when?: number) => {
    this.startedAt = when;
    setTimeout(() => this.onended?.(), 0);
  });
  startedAt: number | undefined;
}

class FakeAudioContext {
  static instances: FakeAudioContext[] = [];

  state: AudioContextState = "suspended";
  destination = {} as unknown as AudioDestinationNode;
  resume = vi.fn(async () => {
    this.state = "running";
  });
  close = vi.fn(async () => {
    this.state = "closed" as AudioContextState;
  });
  decodeAudioData = vi.fn(async (_audioData: ArrayBuffer) => ({ sampleRate: 24_000 } as AudioBuffer));
  createBufferSource = vi.fn(() => {
    const source = new FakeBufferSource();
    this.lastSource = source;
    return source as unknown as AudioBufferSourceNode;
  });
  lastSource: FakeBufferSource | null = null;

  constructor() {
    FakeAudioContext.instances.push(this);
  }
}

beforeEach(() => {
  FakeAudioContext.instances = [];
  Object.defineProperty(window, "AudioContext", {
    value: FakeAudioContext,
    writable: true,
    configurable: true,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("KokoroProvider", () => {
  it("synthesizes, resumes, decodes, and starts playback from the beginning", async () => {
    const blob = { arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(16)) };
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    await provider.speak("hello world", { voice: "af_heart", rate: 1.2 });

    expect(mockSynthesizeTTS).toHaveBeenCalledWith(
      "hello world",
      "af_heart",
      1.2,
      expect.any(AbortSignal),
    );

    const context = FakeAudioContext.instances[0];
    expect(context).toBeDefined();
    if (!context) throw new Error("expected audio context");
    expect(blob.arrayBuffer).toHaveBeenCalledTimes(1);
    expect(context.resume).toHaveBeenCalledTimes(1);
    expect(context.decodeAudioData).toHaveBeenCalledTimes(1);
    expect(context.createBufferSource).toHaveBeenCalledTimes(1);
    expect(context.lastSource?.connect).toHaveBeenCalledWith(context.destination);
    expect(context.lastSource?.start).toHaveBeenCalledWith(0);
    expect(provider.isSpeaking).toBe(false);
  });

  it("stop() aborts playback and disconnects the active source", async () => {
    const blob = { arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(16)) };
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    const heldSource = new FakeBufferSource();
    heldSource.start = vi.fn();
    Object.defineProperty(window, "AudioContext", {
      value: class extends FakeAudioContext {
        override createBufferSource = vi.fn(() => {
          this.lastSource = heldSource;
          return heldSource as unknown as AudioBufferSourceNode;
        });
      },
      writable: true,
      configurable: true,
    });

    const speakPromise = provider.speak("test");

    await vi.waitFor(() => {
      expect(FakeAudioContext.instances[0]?.createBufferSource).toHaveBeenCalledTimes(1);
    });

    provider.stop();

    const context = FakeAudioContext.instances[0];
    expect(context).toBeDefined();
    if (!context) throw new Error("expected audio context");
    const source = context.lastSource;
    expect(source?.stop).toHaveBeenCalledWith(0);
    expect(source?.disconnect).toHaveBeenCalledTimes(1);
    await expect(speakPromise).rejects.toThrow("The operation was aborted.");
    expect(provider.isSpeaking).toBe(false);
  });

  it("reuses the same audio context across multiple playback requests", async () => {
    const blob = { arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(16)) };
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    await provider.speak("first");
    await provider.speak("second");

    expect(FakeAudioContext.instances).toHaveLength(1);
  });

  it("dispose() stops playback and closes the audio context", async () => {
    const blob = { arrayBuffer: vi.fn().mockResolvedValue(new ArrayBuffer(16)) };
    mockSynthesizeTTS.mockResolvedValue(blob);

    const provider = new KokoroProvider();
    await provider.speak("test");

    provider.dispose();

    const context = FakeAudioContext.instances[0];
    expect(context).toBeDefined();
    if (!context) throw new Error("expected audio context");
    expect(context.close).toHaveBeenCalledTimes(1);
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
