import { describe, expect, it, vi } from "vitest";
import { createCaptureProvider, ensurePcmStreamProvider } from "./providerOrchestration";

describe("capture provider orchestration", () => {
  it("selects the streaming provider only when the capability allows it", () => {
    const pcm = { kind: "pcm" } as never;
    const whisper = { kind: "whisper" } as never;
    const pcmStream = vi.fn(() => pcm);
    const httpWhisper = vi.fn(() => whisper);

    expect(createCaptureProvider("whisper", true, { pcmStream, whisper: httpWhisper })).toBe(pcm);
    expect(createCaptureProvider("whisper", false, { pcmStream, whisper: httpWhisper })).toBe(whisper);
    expect(pcmStream).toHaveBeenCalledOnce();
    expect(httpWhisper).toHaveBeenCalledOnce();
  });

  it("does not construct a provider when the backend is unavailable", () => {
    const pcmStream = vi.fn(() => ({}) as never);
    const whisper = vi.fn(() => ({}) as never);
    expect(createCaptureProvider("none", true, { pcmStream, whisper })).toBeNull();
    expect(pcmStream).not.toHaveBeenCalled();
    expect(whisper).not.toHaveBeenCalled();
  });

  it("retains the warmed provider via controller ensure", () => {
    const provider = {} as never;
    const ensure = vi.fn((factory: () => never) => factory());
    expect(ensurePcmStreamProvider({ ensure }, () => provider)).toBe(provider);
    expect(ensure).toHaveBeenCalledOnce();
  });
});
