import { describe, it, expect } from "vitest";
import { createAudioFilterChain } from "../audioUtils";

function createMockAudioContext() {
  const connectCalls: Array<{ from: string; to: string }> = [];

  const makeNode = (name: string) => ({
    _name: name,
    connect(target: { _name: string }) {
      connectCalls.push({ from: name, to: target._name });
      return target;
    },
    type: "" as string,
    frequency: { value: 0 },
    Q: { value: 0 },
    fftSize: 0,
    frequencyBinCount: 64,
    stream: { id: "filtered-stream" } as unknown as MediaStream,
  });

  let filterIdx = 0;
  const ctx = {
    createBiquadFilter: () => makeNode(`filter-${filterIdx++}`),
    createMediaStreamDestination: () => makeNode("destination"),
    createAnalyser: () => makeNode("analyser"),
    createGain: () => ({ ...makeNode("silentGain"), gain: { value: 1 } }),
    destination: makeNode("ctx.destination"),
  } as unknown as AudioContext;

  const source = makeNode("source") as unknown as MediaStreamAudioSourceNode;

  return { ctx, source, connectCalls };
}

describe("createAudioFilterChain", () => {
  it("creates highpass filter at 80Hz and lowpass at 8kHz", () => {
    const { ctx, source } = createMockAudioContext();
    const filters: Array<{ type: string; frequency: { value: number }; Q: { value: number } }> = [];
    const origCreate = ctx.createBiquadFilter.bind(ctx);
    ctx.createBiquadFilter = () => {
      const node = origCreate();
      filters.push(node as typeof filters[0]);
      return node as unknown as BiquadFilterNode;
    };

    createAudioFilterChain(ctx, source);

    expect(filters).toHaveLength(2);
    const [hp, lp] = filters;
    expect(hp?.type).toBe("highpass");
    expect(hp?.frequency.value).toBe(80);
    expect(hp?.Q.value).toBeCloseTo(0.707);
    expect(lp?.type).toBe("lowpass");
    expect(lp?.frequency.value).toBe(8000);
    expect(lp?.Q.value).toBeCloseTo(0.707);
  });

  it("chains nodes: source -> highpass -> lowpass -> destination + analyser", () => {
    const { ctx, source, connectCalls } = createMockAudioContext();
    createAudioFilterChain(ctx, source);

    expect(connectCalls).toEqual([
      { from: "source", to: "filter-0" },
      { from: "filter-0", to: "filter-1" },
      { from: "filter-1", to: "destination" },
      { from: "filter-1", to: "analyser" },
      { from: "filter-1", to: "silentGain" },
      { from: "silentGain", to: "ctx.destination" },
    ]);
  });

  it("returns filteredStream and analyser", () => {
    const { ctx, source } = createMockAudioContext();
    const result = createAudioFilterChain(ctx, source);
    expect(result.filteredStream).toBeDefined();
    expect(result.analyser).toBeDefined();
  });

  it("sets analyser fftSize to 128", () => {
    const { ctx, source } = createMockAudioContext();
    const result = createAudioFilterChain(ctx, source);
    expect((result.analyser as unknown as { fftSize: number }).fftSize).toBe(128);
  });

  it("returns nodes array for cleanup (regression: audio node leak)", () => {
    const { ctx, source } = createMockAudioContext();
    const result = createAudioFilterChain(ctx, source);
    // Should return all created nodes (highpass, lowpass, destination, analyser, silentGain)
    // so the caller can disconnect them when done.
    expect(result.nodes).toBeDefined();
    expect(result.nodes).toHaveLength(5);
  });
});
