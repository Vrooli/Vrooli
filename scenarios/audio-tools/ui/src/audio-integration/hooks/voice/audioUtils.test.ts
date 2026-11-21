// Unit tests for audioUtils.ts:
//   AudioRingBuffer (pure circular-buffer logic),
//   downsample (pure linear-interpolation),
//   createAudioFilterChain (AudioContext wiring),
//   createPassiveCapturePipeline (AudioContext wiring + ring-buffer write).

import { describe, expect, it, vi, beforeEach } from "vitest";

import {
  AudioRingBuffer,
  downsample,
  createAudioFilterChain,
  createPassiveCapturePipeline,
} from "./audioUtils";

// ---------------------------------------------------------------------------
// Fake AudioContext helpers
// ---------------------------------------------------------------------------

function makeNode() {
  return {
    connect: vi.fn().mockReturnThis(),
    disconnect: vi.fn(),
    type: "",
    frequency: { value: 0, setValueAtTime: vi.fn() },
    Q: { value: 0 },
    gain: { value: 0 },
    fftSize: 0,
  };
}

interface FakeAudioCtx {
  state: AudioContextState;
  sampleRate: number;
  currentTime: number;
  destination: ReturnType<typeof makeNode>;
  createBiquadFilter: ReturnType<typeof vi.fn>;
  createGain: ReturnType<typeof vi.fn>;
  createAnalyser: ReturnType<typeof vi.fn>;
  createMediaStreamDestination: ReturnType<typeof vi.fn>;
  createScriptProcessor: ReturnType<typeof vi.fn>;
  createOscillator: ReturnType<typeof vi.fn>;
  createMediaStreamSource: ReturnType<typeof vi.fn>;
  resume: ReturnType<typeof vi.fn>;
  close: ReturnType<typeof vi.fn>;
}

function makeFakeAudioContext(sampleRate = 48000): FakeAudioCtx {
  const destination = makeNode();
  // eslint-disable-next-line @typescript-eslint/no-deprecated -- mirrors the deprecated ScriptProcessor handler used in createPassiveCapturePipeline
  const scriptNode = { ...makeNode(), onaudioprocess: null as null | ((e: AudioProcessingEvent) => void) };
  const mediaDestNode = { ...makeNode(), stream: {} as MediaStream };

  return {
    state: "running" satisfies AudioContextState,
    sampleRate,
    currentTime: 0,
    destination,
    createBiquadFilter: vi.fn().mockImplementation(() => makeNode()),
    createGain: vi.fn().mockImplementation(() => makeNode()),
    createAnalyser: vi.fn().mockImplementation(() => makeNode()),
    createMediaStreamDestination: vi.fn().mockReturnValue(mediaDestNode),
    createScriptProcessor: vi.fn().mockReturnValue(scriptNode),
    createOscillator: vi.fn().mockImplementation(() => makeNode()),
    createMediaStreamSource: vi.fn().mockImplementation(() => makeNode()),
    resume: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
  };
}

function makeSource() {
  return { connect: vi.fn(), disconnect: vi.fn() };
}

// ---------------------------------------------------------------------------
// AudioRingBuffer
// ---------------------------------------------------------------------------

describe("AudioRingBuffer", () => {
  it("constructs with correct capacity and sample rate", () => {
    const rb = new AudioRingBuffer(2, 48000);
    expect(rb.capacity).toBe(96000);
    expect(rb.sampleRate).toBe(48000);
    expect(rb.totalWritten).toBe(0);
  });

  it("rounds up capacity for non-integer durations", () => {
    const rb = new AudioRingBuffer(0.1, 16000);
    expect(rb.capacity).toBe(1600);
  });

  it("write increments totalWritten", () => {
    const rb = new AudioRingBuffer(1, 16000);
    rb.write(new Float32Array([1, 2, 3]));
    expect(rb.totalWritten).toBe(3);
  });

  it("extractLast returns 0-length array when nothing written", () => {
    const rb = new AudioRingBuffer(1, 16000);
    expect(rb.extractLast(100).length).toBe(0);
  });

  it("extractLast returns all samples when fewer written than requested", () => {
    const rb = new AudioRingBuffer(1, 16000);
    rb.write(new Float32Array([0.1, 0.2]));
    const out = rb.extractLast(100);
    expect(out.length).toBe(2);
    expect(out[0]).toBeCloseTo(0.1);
    expect(out[1]).toBeCloseTo(0.2);
  });

  it("extractLast returns the last N samples (no wrap)", () => {
    const rb = new AudioRingBuffer(1, 8);
    rb.write(new Float32Array([1, 2, 3, 4, 5]));
    const out = rb.extractLast(3);
    expect(Array.from(out)).toEqual([3, 4, 5]);
  });

  it("extractLast handles wrap-around correctly", () => {
    // capacity=5, write 7 samples (wraps by 2)
    const rb = new AudioRingBuffer(1, 5);
    rb.write(new Float32Array([1, 2, 3, 4, 5])); // fills buffer, writePos=0
    rb.write(new Float32Array([6, 7]));           // overwrites first 2, writePos=2
    // Buffer is now [6,7,3,4,5], writePos=2
    // extractLast(3): last 3 = [5,6,7] → wait, let's reason through:
    // writePos=2 means next write goes to index 2
    // last 3 = indices (2-3+5)%5=4, 0, 1 → [5,6,7]
    const out = rb.extractLast(3);
    expect(out.length).toBe(3);
    expect(Array.from(out)).toEqual([5, 6, 7]);
  });

  it("write handles input larger than capacity", () => {
    const rb = new AudioRingBuffer(1, 3); // capacity=3
    const big = new Float32Array([1, 2, 3, 4, 5, 6, 7]); // 7 > 3
    rb.write(big);
    expect(rb.totalWritten).toBe(7);
    // Should keep only last 3 samples: [5,6,7]
    const out = rb.extractLast(3);
    expect(Array.from(out)).toEqual([5, 6, 7]);
  });

  it("write splits correctly when wrapping within capacity", () => {
    const rb = new AudioRingBuffer(1, 6); // capacity=6
    rb.write(new Float32Array([1, 2, 3, 4])); // writePos=4
    rb.write(new Float32Array([5, 6, 7, 8])); // wraps: writes [5,6] at 4,5 then [7,8] at 0,1
    expect(rb.totalWritten).toBe(8);
    // extractLast(4) should be [5,6,7,8]
    const out = rb.extractLast(4);
    expect(Array.from(out)).toEqual([5, 6, 7, 8]);
  });

  it("mark and extractSinceMark work correctly", () => {
    const rb = new AudioRingBuffer(1, 16);
    rb.write(new Float32Array([1, 2, 3]));
    const m = rb.mark();
    rb.write(new Float32Array([4, 5]));
    const out = rb.extractSinceMark(m);
    expect(out.length).toBe(2);
    expect(Array.from(out)).toEqual([4, 5]);
  });

  it("extractSinceMark returns empty when no new data", () => {
    const rb = new AudioRingBuffer(1, 16);
    rb.write(new Float32Array([1, 2]));
    const m = rb.mark();
    expect(rb.extractSinceMark(m).length).toBe(0);
  });

  it("extractSinceMark caps at capacity", () => {
    const rb = new AudioRingBuffer(1, 4); // capacity=4
    const m = rb.mark(); // 0
    rb.write(new Float32Array([1, 2, 3, 4, 5, 6])); // written 6, but capacity=4
    const out = rb.extractSinceMark(m);
    expect(out.length).toBe(4); // capped at capacity
  });

  it("reset clears totalWritten and writePos", () => {
    const rb = new AudioRingBuffer(1, 8);
    rb.write(new Float32Array([1, 2, 3]));
    rb.reset();
    expect(rb.totalWritten).toBe(0);
    expect(rb.extractLast(5).length).toBe(0);
  });

  it("extractLast respects capacity limit", () => {
    const rb = new AudioRingBuffer(1, 5);
    rb.write(new Float32Array([1, 2, 3, 4, 5]));
    // Requesting more than capacity returns capacity worth
    const out = rb.extractLast(100);
    expect(out.length).toBe(5);
  });
});

// ---------------------------------------------------------------------------
// downsample
// ---------------------------------------------------------------------------

describe("downsample", () => {
  it("returns the same buffer when rates are equal", () => {
    const buf = new Float32Array([1, 2, 3]);
    const out = downsample(buf, 16000, 16000);
    expect(out).toBe(buf); // identity — same reference
  });

  it("throws when trying to upsample", () => {
    const buf = new Float32Array([1, 2, 3]);
    expect(() => downsample(buf, 16000, 44100)).toThrow("Cannot upsample");
  });

  it("downsamples from 48kHz to 16kHz (3:1 ratio)", () => {
    // Simple input: flat signal at 0.5
    const buf = new Float32Array(48);
    buf.fill(0.5);
    const out = downsample(buf, 48000, 16000);
    expect(out.length).toBe(16);
    // Flat signal should remain flat after downsampling
    for (const v of out) {
      expect(v).toBeCloseTo(0.5, 5);
    }
  });

  it("downsamples from 44100 to 16000", () => {
    const buf = new Float32Array(441).fill(1.0);
    const out = downsample(buf, 44100, 16000);
    // Expected output length = ceil(441 / (44100/16000)) = ceil(441 / 2.75625) = ceil(160.02...) = 161
    expect(out.length).toBeGreaterThan(0);
    expect(out[0]).toBeCloseTo(1.0, 4);
  });

  it("performs linear interpolation at boundary samples", () => {
    // Rising ramp: [0, 1, 2] at 30000Hz, downsample to 10000Hz (ratio=3)
    const buf = new Float32Array([0, 1, 2]);
    const out = downsample(buf, 30000, 10000);
    // At i=0: srcIndex=0 → floor=0, ceil=1, frac=0 → 0*(1)+1*0 = 0
    // At i=1: srcIndex=3 → but buf.length=3, srcCeil = min(4,2)=2, frac=3-3=0 → 2*(1) = 2
    // Actually ratio=3, so srcIndex for i=0 is 0, for i=1 is 3 (out of bounds)
    expect(out.length).toBe(1); // ceil(3/3) = 1
    expect(out[0]).toBeCloseTo(0, 5);
  });

  it("handles single-sample buffer", () => {
    const buf = new Float32Array([0.9]);
    const out = downsample(buf, 48000, 16000);
    expect(out.length).toBe(1);
    expect(out[0]).toBeCloseTo(0.9, 5);
  });

  it("handles empty buffer", () => {
    const buf = new Float32Array(0);
    const out = downsample(buf, 48000, 16000);
    expect(out.length).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// createAudioFilterChain
// ---------------------------------------------------------------------------

describe("createAudioFilterChain", () => {
  let ctx: FakeAudioCtx;
  let source: ReturnType<typeof makeSource>;

  beforeEach(() => {
    ctx = makeFakeAudioContext();
    source = makeSource();
  });

  it("returns analyser, filteredStream, and nodes array", () => {
    const result = createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(result).toHaveProperty("analyser");
    expect(result).toHaveProperty("filteredStream");
    expect(result).toHaveProperty("nodes");
    expect(result.nodes.length).toBeGreaterThan(0);
  });

  it("creates a highpass and lowpass biquad filter", () => {
    createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(ctx.createBiquadFilter).toHaveBeenCalledTimes(2);
  });

  it("creates a gain node for silent monitoring path", () => {
    createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(ctx.createGain).toHaveBeenCalledTimes(1);
  });

  it("creates an analyser node", () => {
    createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(ctx.createAnalyser).toHaveBeenCalledTimes(1);
  });

  it("creates a media stream destination", () => {
    createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(ctx.createMediaStreamDestination).toHaveBeenCalledTimes(1);
  });

  it("connects source to filter chain", () => {
    createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    expect(source.connect).toHaveBeenCalledTimes(1);
  });

  it("returns 5 nodes in the nodes array", () => {
    const { nodes } = createAudioFilterChain(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
    );
    // highpass, lowpass, destination, analyser, silentGain
    expect(nodes.length).toBe(5);
  });
});

// ---------------------------------------------------------------------------
// createPassiveCapturePipeline
// ---------------------------------------------------------------------------

describe("createPassiveCapturePipeline", () => {
  let ctx: FakeAudioCtx;
  let source: ReturnType<typeof makeSource>;
  let ringBuffer: AudioRingBuffer;

  beforeEach(() => {
    ctx = makeFakeAudioContext();
    source = makeSource();
    ringBuffer = new AudioRingBuffer(3, 16000);
  });

  it("returns analyser and nodes array", () => {
    const result = createPassiveCapturePipeline(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
      ringBuffer,
    );
    expect(result).toHaveProperty("analyser");
    expect(result).toHaveProperty("nodes");
  });

  it("creates a ScriptProcessorNode for PCM capture", () => {
    createPassiveCapturePipeline(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
      ringBuffer,
    );
    expect(ctx.createScriptProcessor).toHaveBeenCalledWith(4096, 1, 1);
  });

  it("returns 5 nodes in the nodes array", () => {
    const { nodes } = createPassiveCapturePipeline(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
      ringBuffer,
    );
    // highpass, lowpass, analyser, processor, silentGain
    expect(nodes.length).toBe(5);
  });

  it("sets onaudioprocess handler that writes to ring buffer", () => {
    createPassiveCapturePipeline(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
      ringBuffer,
    );
    const processor = ctx.createScriptProcessor.mock.results[0]!.value;
    expect(processor.onaudioprocess).toBeTypeOf("function");

    // Simulate an audio processing event
    const inputData = new Float32Array([0.1, 0.2, 0.3]);
    const outputData = new Float32Array(3);
    const fakeEvent = {
      inputBuffer: { getChannelData: vi.fn().mockReturnValue(inputData) },
      outputBuffer: { getChannelData: vi.fn().mockReturnValue(outputData) },
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- AudioProcessingEvent matches deprecated ScriptProcessor
    } as unknown as AudioProcessingEvent;

    processor.onaudioprocess(fakeEvent);

    // Ring buffer should now have 3 samples
    expect(ringBuffer.totalWritten).toBe(3);
    const out = ringBuffer.extractLast(3);
    // Float32Array has limited precision — use toBeCloseTo
    expect(out[0]).toBeCloseTo(0.1, 5);
    expect(out[1]).toBeCloseTo(0.2, 5);
    expect(out[2]).toBeCloseTo(0.3, 5);
  });

  it("pass-through copies input to output buffer", () => {
    createPassiveCapturePipeline(
      ctx as unknown as AudioContext,
      source as unknown as MediaStreamAudioSourceNode,
      ringBuffer,
    );
    const processor = ctx.createScriptProcessor.mock.results[0]!.value;

    const inputData = new Float32Array([0.5, 0.6]);
    const outputData = new Float32Array(2);
    const fakeEvent = {
      inputBuffer: { getChannelData: vi.fn().mockReturnValue(inputData) },
      outputBuffer: { getChannelData: vi.fn().mockReturnValue(outputData) },
    // eslint-disable-next-line @typescript-eslint/no-deprecated -- AudioProcessingEvent matches deprecated ScriptProcessor
    } as unknown as AudioProcessingEvent;

    processor.onaudioprocess(fakeEvent);
    // Float32Array precision — use closeTo
    expect(outputData[0]).toBeCloseTo(0.5, 5);
    expect(outputData[1]).toBeCloseTo(0.6, 5);
  });
});
