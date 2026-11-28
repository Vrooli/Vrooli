import { afterEach, describe, expect, it, vi } from "vitest";

import { createCanonicalPcmCapture } from "@vrooli/audio-capture-browser";

class FakeNode {
  readonly connect = vi.fn();
  readonly disconnect = vi.fn();
}

class FakeAudioWorkletNode extends FakeNode {
  static instances: FakeAudioWorkletNode[] = [];
  readonly port: { onmessage: ((event: MessageEvent<Float32Array>) => void) | null } = { onmessage: null };

  constructor() {
    super();
    FakeAudioWorkletNode.instances.push(this);
  }
}

// Minimal structural stand-in for the deprecated AudioProcessingEvent the
// ScriptProcessor fallback receives. Using the local shape keeps the test off
// the deprecated DOM type while asserting the same inputBuffer contract.
interface FakeAudioProcessingEvent {
  inputBuffer: { getChannelData: () => Float32Array };
}

describe("createCanonicalPcmCapture", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("falls back to ScriptProcessor when an attached AudioWorklet never emits PCM", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("AudioWorkletNode", FakeAudioWorkletNode);
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:pcm-capture"),
      revokeObjectURL: vi.fn(),
    });

    const source = new FakeNode();
    FakeAudioWorkletNode.instances = [];
    const gain = new FakeNode() as FakeNode & { gain: { value: number } };
    gain.gain = { value: 1 };
    const processor = new FakeNode() as FakeNode & {
      onaudioprocess: ((event: FakeAudioProcessingEvent) => void) | null;
    };
    const createScriptProcessor = vi.fn(() => processor);
    const context = {
      sampleRate: 16_000,
      audioWorklet: { addModule: vi.fn(() => Promise.resolve()) },
      createMediaStreamSource: vi.fn(() => source),
      createGain: vi.fn(() => gain),
      createScriptProcessor,
      destination: new FakeNode(),
    } as unknown as AudioContext;
    const onFrame = vi.fn();

    const capture = await createCanonicalPcmCapture(context, {} as MediaStream, onFrame);

    await vi.advanceTimersByTimeAsync(750);

    expect(FakeAudioWorkletNode.instances).toHaveLength(1);
    const [worklet] = FakeAudioWorkletNode.instances;
    if (!worklet) throw new Error("AudioWorklet capture should have been attached");
    expect(worklet.disconnect).toHaveBeenCalledOnce();
    expect(createScriptProcessor).toHaveBeenCalledOnce();
    const processFrame = processor.onaudioprocess;
    if (!processFrame) throw new Error("ScriptProcessor fallback should receive a callback");
    processFrame({
      inputBuffer: { getChannelData: () => new Float32Array([0.25, -0.25]) },
    });
    expect(onFrame).toHaveBeenCalledWith(new Float32Array([0.25, -0.25]), 16_000);

    capture.stop();
    expect(processor.disconnect).toHaveBeenCalledOnce();
  });
});
