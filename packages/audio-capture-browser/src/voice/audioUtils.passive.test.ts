import { afterEach, describe, expect, it, vi } from "vitest";
import { AudioRingBuffer, createPassiveCapturePipeline } from "./audioUtils";

function fakeNode() {
  return { connect: vi.fn(), disconnect: vi.fn() };
}

function fakeContext() {
  const source = fakeNode();
  const highpass = Object.assign(fakeNode(), { frequency: { value: 0 }, Q: { value: 0 }, type: "" });
  const lowpass = Object.assign(fakeNode(), { frequency: { value: 0 }, Q: { value: 0 }, type: "" });
  const analyser = Object.assign(fakeNode(), { fftSize: 0 });
  const gain = Object.assign(fakeNode(), { gain: { value: 1 } });
  return {
    context: {
      audioWorklet: { addModule: vi.fn().mockResolvedValue(undefined) },
      createBiquadFilter: vi.fn().mockReturnValueOnce(highpass).mockReturnValueOnce(lowpass).mockImplementation(() => Object.assign(fakeNode(), { frequency: { value: 0 }, Q: { value: 0 }, type: "" })),
      createAnalyser: vi.fn(() => analyser), createGain: vi.fn(() => gain),
      createScriptProcessor: vi.fn(() => Object.assign(fakeNode(), { onaudioprocess: null })),
      destination: fakeNode(), sampleRate: 48_000,
    } as unknown as AudioContext,
    source: source as unknown as MediaStreamAudioSourceNode,
    highpass, lowpass, analyser, gain,
  };
}

describe("passive capture pipeline", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });

  it("reuses one worklet module across pipelines on the same context", async () => {
    const { context, source, highpass, lowpass, analyser, gain } = fakeContext();
    const worklets: ReturnType<typeof fakeNode>[] = [];
    const createObjectURL = vi.fn(() => "blob:passive-worklet");
    const revokeObjectURL = vi.fn();
    vi.stubGlobal("URL", { ...URL, createObjectURL, revokeObjectURL });
    vi.stubGlobal("AudioWorkletNode", vi.fn(() => {
      const worklet = Object.assign(fakeNode(), { port: { onmessage: null as ((event: MessageEvent<Float32Array>) => void) | null } });
      worklets.push(worklet);
      return worklet;
    }));
    const ring = new AudioRingBuffer(1, 48_000);

    const pipeline = await createPassiveCapturePipeline(context, source, ring);
    const secondPipeline = await createPassiveCapturePipeline(context, source, ring);
    const worklet = worklets[0] as ReturnType<typeof fakeNode> & { port: { onmessage: ((event: MessageEvent<Float32Array>) => void) | null } };
    worklet.port.onmessage?.({ data: new Float32Array([0.25, -0.5]) } as MessageEvent<Float32Array>);

    expect(ring.extractSinceMark(0)).toEqual(new Float32Array([0.25, -0.5]));
    expect(context.audioWorklet?.addModule).toHaveBeenCalledOnce();
    expect((context as unknown as { createScriptProcessor: ReturnType<typeof vi.fn> }).createScriptProcessor).not.toHaveBeenCalled();
    expect(source.connect).toHaveBeenCalledWith(highpass);
    expect(highpass.connect).toHaveBeenCalledWith(lowpass);
    expect(lowpass.connect).toHaveBeenCalledWith(analyser);
    expect(lowpass.connect).toHaveBeenCalledWith(worklet);
    expect(worklet.connect).toHaveBeenCalledWith(gain);
    expect(pipeline.nodes).toContain(worklet as unknown as AudioNode);
    expect(secondPipeline.nodes).toContain(worklets[1] as unknown as AudioNode);
    expect(worklets).toHaveLength(2);
    expect(revokeObjectURL).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:passive-worklet");
  });

  it("falls back to ScriptProcessor when worklet loading fails", async () => {
    const { context, source, lowpass } = fakeContext();
    (context.audioWorklet!.addModule as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("blocked"));
    vi.stubGlobal("AudioWorkletNode", vi.fn());
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:passive-fallback"), revokeObjectURL: vi.fn() });
    const ring = new AudioRingBuffer(1, 48_000);

    const pipeline = await createPassiveCapturePipeline(context, source, ring);
    const processor = (context.createScriptProcessor as ReturnType<typeof vi.fn>).mock.results[0]?.value as {
      onaudioprocess: ((event: AudioProcessingEvent) => void) | null; connect: ReturnType<typeof vi.fn>;
    };
    const input = new Float32Array([0.1, 0.2]);
    const output = new Float32Array(2);
    processor.onaudioprocess?.({ inputBuffer: { getChannelData: () => input }, outputBuffer: { getChannelData: () => output } } as unknown as AudioProcessingEvent);

    expect(lowpass.connect).toHaveBeenCalledWith(processor);
    expect(output).toEqual(input);
    expect(ring.extractSinceMark(0)).toEqual(input);
    expect(pipeline.nodes).toContain(processor as unknown as AudioNode);
  });
});
