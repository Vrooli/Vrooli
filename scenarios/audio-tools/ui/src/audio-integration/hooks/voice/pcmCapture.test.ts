// Unit tests for pcmCapture.ts — createScriptProcessorPcmCapture factory.
//
// The factory wires a ScriptProcessorNode on the shared AudioContext and
// returns a PcmCapture with a stop() method. We mock ./sharedAudioContext
// to avoid creating a real AudioContext in jsdom.

import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

vi.mock("./sharedAudioContext", () => ({
  getSharedAudioContext: vi.fn(),
}));

import { getSharedAudioContext } from "./sharedAudioContext";
import { createCanonicalPcmCapture, createScriptProcessorPcmCapture } from "./pcmCapture";

// ---------------------------------------------------------------------------
// Fake AudioContext + MediaStream
// ---------------------------------------------------------------------------

function makeNode() {
  return {
    connect: vi.fn(),
    disconnect: vi.fn(),
  };
}

function makeScriptProcessorNode() {
  return {
    connect: vi.fn(),
    disconnect: vi.fn(),
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- AudioProcessingEvent mirrors the deprecated ScriptProcessor handler
    onaudioprocess: null as null | ((e: AudioProcessingEvent) => void),
  };
}

function makeSourceNode() {
  return {
    connect: vi.fn(),
    disconnect: vi.fn(),
  };
}

interface FakeAudioCtx {
  state: AudioContextState;
  sampleRate: number;
  destination: object;
  createMediaStreamSource: ReturnType<typeof vi.fn>;
  createScriptProcessor: ReturnType<typeof vi.fn>;
  createGain: ReturnType<typeof vi.fn>;
}

function makeFakeAudioContext(sampleRate = 48000): FakeAudioCtx {
  return {
    state: "running" satisfies AudioContextState,
    sampleRate,
    destination: {},
    createMediaStreamSource: vi.fn().mockImplementation(makeSourceNode),
    createScriptProcessor: vi.fn().mockImplementation(makeScriptProcessorNode),
    createGain: vi.fn().mockImplementation(() => ({
      ...makeNode(),
      gain: { value: 0 },
    })),
  };
}

function makeFakeStream(): MediaStream {
  return {} as unknown as MediaStream;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

let ctx: FakeAudioCtx;

beforeEach(() => {
  ctx = makeFakeAudioContext();
  vi.mocked(getSharedAudioContext).mockReturnValue(ctx as unknown as AudioContext);
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("createScriptProcessorPcmCapture", () => {

  it("falls back to ScriptProcessor when AudioWorklet is unavailable", async () => {
    const capture = await createCanonicalPcmCapture(makeFakeStream(), vi.fn());
    expect(capture.stop).toBeTypeOf("function");
    expect(ctx.createScriptProcessor).toHaveBeenCalledWith(4096, 1, 1);
  });

  it("returns a PcmCapture with a stop() function", () => {
    const capture = createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    expect(capture).toHaveProperty("stop");
    expect(typeof capture.stop).toBe("function");
  });

  it("creates a MediaStreamSource from the provided stream", () => {
    const stream = makeFakeStream();
    createScriptProcessorPcmCapture(stream, vi.fn());
    expect(ctx.createMediaStreamSource).toHaveBeenCalledWith(stream);
  });

  it("creates a ScriptProcessorNode with buffer size 4096", () => {
    createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    expect(ctx.createScriptProcessor).toHaveBeenCalledWith(4096, 1, 1);
  });

  it("creates a gain node for silent keep-alive path", () => {
    createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    expect(ctx.createGain).toHaveBeenCalledOnce();
  });

  it("installs onaudioprocess handler on the processor", () => {
    createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
    expect(processor.onaudioprocess).toBeTypeOf("function");
  });

  it("calls onFrame with a copy of the PCM data and the context sampleRate", () => {
    const onFrame = vi.fn();
    createScriptProcessorPcmCapture(makeFakeStream(), onFrame);

    const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
    const inputData = new Float32Array([0.1, 0.2, 0.3]);
    const fakeEvent = {
      inputBuffer: { getChannelData: vi.fn().mockReturnValue(inputData) },
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- AudioProcessingEvent matches the deprecated ScriptProcessor handler
    } as unknown as AudioProcessingEvent;

    processor.onaudioprocess!(fakeEvent);

    expect(onFrame).toHaveBeenCalledOnce();
    const [samples, sampleRate] = onFrame.mock.calls[0]!;
    expect(sampleRate).toBe(48000);
    // The samples passed to onFrame should be a COPY, not the same reference
    expect(samples).not.toBe(inputData);
    // Float32Array has limited precision — use closeTo
    const arr = samples as Float32Array;
    expect(arr[0]).toBeCloseTo(0.1, 5);
    expect(arr[1]).toBeCloseTo(0.2, 5);
    expect(arr[2]).toBeCloseTo(0.3, 5);
  });

  it("delivers a fresh copy each callback so consumers can retain samples safely", () => {
    const frames: Float32Array[] = [];
    const onFrame = vi.fn().mockImplementation((s: Float32Array) => frames.push(s));
    createScriptProcessorPcmCapture(makeFakeStream(), onFrame);

    const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
    const inputData = new Float32Array([1, 2, 3]);
    const fakeEvent = {
      inputBuffer: { getChannelData: vi.fn().mockReturnValue(inputData) },
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- AudioProcessingEvent matches the deprecated ScriptProcessor handler
    } as unknown as AudioProcessingEvent;

    processor.onaudioprocess!(fakeEvent);
    // Mutate the input to verify the delivered frame is independent
    inputData[0] = 99;
    processor.onaudioprocess!(fakeEvent);

    expect(frames[0]![0]).toBe(1); // first frame is a snapshot
    expect(frames[1]![0]).toBe(99); // second frame gets the updated value
  });

  it("wires source → processor → silentGain → destination", () => {
    createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    const source = ctx.createMediaStreamSource.mock.results[0]!.value as ReturnType<typeof makeSourceNode>;
    const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
    const gain = ctx.createGain.mock.results[0]!.value as ReturnType<typeof makeNode> & { gain: { value: number } };

    expect(source.connect).toHaveBeenCalledWith(processor);
    expect(processor.connect).toHaveBeenCalledWith(gain);
    expect(gain.connect).toHaveBeenCalledWith(ctx.destination);
  });

  it("gain value is 0 (silent keep-alive)", () => {
    createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
    const gain = ctx.createGain.mock.results[0]!.value as { gain: { value: number } };
    expect(gain.gain.value).toBe(0);
  });

  describe("stop()", () => {
    it("clears onaudioprocess handler", () => {
      const capture = createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
      capture.stop();
      const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
      expect(processor.onaudioprocess).toBeNull();
    });

    it("disconnects source, processor, and silentGain", () => {
      const capture = createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
      capture.stop();

      const source = ctx.createMediaStreamSource.mock.results[0]!.value as ReturnType<typeof makeSourceNode>;
      const processor = ctx.createScriptProcessor.mock.results[0]!.value as ReturnType<typeof makeScriptProcessorNode>;
      const gain = ctx.createGain.mock.results[0]!.value as ReturnType<typeof makeNode>;

      expect(source.disconnect).toHaveBeenCalledOnce();
      expect(processor.disconnect).toHaveBeenCalledOnce();
      expect(gain.disconnect).toHaveBeenCalledOnce();
    });

    it("does not throw when nodes are already disconnected", () => {
      const source = makeSourceNode();
      source.disconnect = vi.fn().mockImplementation(() => {
        throw new Error("already disconnected");
      });
      ctx.createMediaStreamSource = vi.fn().mockReturnValue(source);

      const capture = createScriptProcessorPcmCapture(makeFakeStream(), vi.fn());
      expect(() => capture.stop()).not.toThrow();
    });
  });
});
