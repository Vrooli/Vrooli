import { afterEach, describe, expect, it, vi } from "vitest";
import { createCanonicalPcmCapture } from "./pcmCapture";

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
});
