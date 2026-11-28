import { afterEach, describe, expect, it, vi } from "vitest";
import { createAudioWorkletPcmCapture, createCanonicalPcmCapture, preloadCanonicalPcmCapture } from "./pcmCapture";

describe("canonical PCM capture", () => {
  afterEach(() => {
    Object.defineProperty(navigator, "webdriver", { configurable: true, value: undefined });
  });

  it("uses the immediately scheduled graph for webdriver fake media", async () => {
    Object.defineProperty(navigator, "webdriver", { configurable: true, value: true });
    const disconnect = vi.fn();
    const source = { connect: vi.fn(), disconnect };
    const processor = { connect: vi.fn(), disconnect, onaudioprocess: null };
    const gain = { gain: { value: 0 }, connect: vi.fn(), disconnect };
    const addModule = vi.fn(async () => {});
    const context = {
      sampleRate: 48_000,
      audioWorklet: { addModule },
      createMediaStreamSource: vi.fn(() => source),
      createScriptProcessor: vi.fn(() => processor),
      createGain: vi.fn(() => gain),
      destination: {},
    } as unknown as AudioContext;

    const capture = await createCanonicalPcmCapture(context, {} as MediaStream, vi.fn());

    expect(context.createScriptProcessor).toHaveBeenCalledOnce();
    expect(addModule).not.toHaveBeenCalled();
    capture.stop();
  });

  it("preloads the worklet module for a real voice gesture", async () => {
    const addModule = vi.fn(async () => {});
    const context = { audioWorklet: { addModule } } as unknown as AudioContext;
    vi.stubGlobal("AudioWorkletNode", class {});
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:test-vrooli-pcm-capture"),
      revokeObjectURL: vi.fn(),
    });

    await preloadCanonicalPcmCapture(context);

    expect(addModule).toHaveBeenCalledOnce();
    vi.unstubAllGlobals();
  });

  it("falls back when worklet loading stalls and retries the context later", async () => {
    Object.defineProperty(navigator, "webdriver", { configurable: true, value: undefined });
    vi.useFakeTimers();
    const addModule = vi.fn<() => Promise<void>>();
    let resolveLateLoad!: () => void;
    addModule.mockReturnValueOnce(new Promise<void>((resolve) => { resolveLateLoad = resolve; })).mockResolvedValueOnce();
    const context = {
      sampleRate: 48_000,
      audioWorklet: { addModule },
      createMediaStreamSource: vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() })),
      createScriptProcessor: vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn(), onaudioprocess: null })),
      createGain: vi.fn(() => ({ gain: { value: 0 }, connect: vi.fn(), disconnect: vi.fn() })),
      destination: {},
    } as unknown as AudioContext;
    const revokeObjectURL = vi.fn();
    const processorNames: string[] = [];
    const moduleSources: Promise<string>[] = [];
    vi.stubGlobal("AudioWorkletNode", class {
      port = { onmessage: null as ((event: MessageEvent<Float32Array>) => void) | null };
      connect = vi.fn();
      disconnect = vi.fn();
      constructor(_context: AudioContext, name: string) {
        processorNames.push(name);
      }
    });
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn((blob: Blob) => {
        moduleSources.push(Promise.resolve((blob as unknown as { source: string }).source));
        return `blob:test-vrooli-pcm-capture-${moduleSources.length}`;
      }),
      revokeObjectURL,
    });
    vi.stubGlobal("Blob", class {
      readonly source: string;
      constructor(parts: BlobPart[]) {
        this.source = String(parts[0]);
      }
    });

    try {
      const first = createCanonicalPcmCapture(context, {} as MediaStream, vi.fn());
      expect(navigator.webdriver).toBeUndefined();
      expect(addModule).toHaveBeenCalledOnce();
      await vi.advanceTimersByTimeAsync(1_000);
      const fallback = await first;
      expect(context.createScriptProcessor).toHaveBeenCalledOnce();
      expect(revokeObjectURL).toHaveBeenCalledOnce();
      fallback.stop();

      // A timed-out addModule cannot be canceled. Resolve it after fallback to
      // model a slow browser registering its processor while a later gesture
      // retries the same AudioContext.
      resolveLateLoad();
      await Promise.resolve();
      const retry = await createCanonicalPcmCapture(context, {} as MediaStream, vi.fn());
      expect(addModule).toHaveBeenCalledTimes(2);
      const sources = await Promise.all(moduleSources);
      const registeredNames = sources.map((source) => source.match(/registerProcessor\("([^"]+)"/)?.[1]);
      expect(registeredNames[0]).toBeTruthy();
      expect(registeredNames[1]).toBeTruthy();
      expect(registeredNames[1]).not.toBe(registeredNames[0]);
      expect(processorNames).toEqual([registeredNames[1]]);
      retry.stop();
    } finally {
      vi.useRealTimers();
      vi.unstubAllGlobals();
    }
  });

  it("releases the worklet URL when addModule throws synchronously", async () => {
    const addModule = vi.fn(() => { throw new Error("worklet rejected"); });
    const context = {
      sampleRate: 48_000,
      audioWorklet: { addModule },
      createMediaStreamSource: vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() })),
      createScriptProcessor: vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn(), onaudioprocess: null })),
      createGain: vi.fn(() => ({ gain: { value: 0 }, connect: vi.fn(), disconnect: vi.fn() })),
      destination: {},
    } as unknown as AudioContext;
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("AudioWorkletNode", class {});
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:test-vrooli-pcm-capture"),
      revokeObjectURL,
    });

    try {
      const capture = await createCanonicalPcmCapture(context, {} as MediaStream, vi.fn());
      expect(addModule).toHaveBeenCalledOnce();
      expect(revokeObjectURL).toHaveBeenCalledOnce();
      expect(context.createScriptProcessor).toHaveBeenCalledOnce();
      capture.stop();
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it("forwards the transferred worklet frame without copying it again", async () => {
    let workletNode: { port: { onmessage: ((event: MessageEvent<Float32Array>) => void) | null }; connect: ReturnType<typeof vi.fn>; disconnect: ReturnType<typeof vi.fn> } | undefined;
    class FakeAudioWorkletNode {
      port = { onmessage: null as ((event: MessageEvent<Float32Array>) => void) | null };
      connect = vi.fn();
      disconnect = vi.fn();

      constructor() {
        workletNode = this;
      }
    }
    const source = { connect: vi.fn(), disconnect: vi.fn() };
    const gain = { gain: { value: 1 }, connect: vi.fn(), disconnect: vi.fn() };
    const context = {
      sampleRate: 48_000,
      audioWorklet: { addModule: vi.fn(async () => {}) },
      createMediaStreamSource: vi.fn(() => source),
      createGain: vi.fn(() => gain),
      destination: {},
    } as unknown as AudioContext;
    const onFrame = vi.fn();
    vi.stubGlobal("AudioWorkletNode", FakeAudioWorkletNode);
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:test-vrooli-pcm-capture"),
      revokeObjectURL: vi.fn(),
    });

    try {
      const capture = await createAudioWorkletPcmCapture(context, {} as MediaStream, onFrame);
      const transferred = Float32Array.of(0.25, -0.5);
      workletNode?.port.onmessage?.({ data: transferred } as MessageEvent<Float32Array>);

      expect(onFrame).toHaveBeenCalledOnce();
      expect(onFrame).toHaveBeenCalledWith(transferred, 48_000);
      expect(onFrame.mock.calls[0]?.[0]).toBe(transferred);
      capture.stop();
    } finally {
      vi.unstubAllGlobals();
    }
  });
});
