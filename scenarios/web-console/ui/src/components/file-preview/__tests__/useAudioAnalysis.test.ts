import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAudioAnalysis } from "../useAudioAnalysis";
import { decodeWaveform, MAX_AUDIO_ANALYSIS_BYTES, readAudioBytes, readAudioTags } from "../audioAnalysis";
import type { PreviewModel } from "../types";

vi.mock("../audioAnalysis", async (original) => ({ ...await original<typeof import("../audioAnalysis")>(), readAudioBytes: vi.fn(), decodeWaveform: vi.fn(), readAudioTags: vi.fn() }));

const model: PreviewModel = {
  previewId: "audio-1", inputPath: "/song", resolvedPath: "/song", basename: "song", resolutionBasis: "absolute", kind: "audio", mimeType: "audio/mp4", sizeBytes: 1024,
  canPreview: true, canDownload: true, supportsRange: true, textContentAvailable: false, listingAvailable: false, blobUrl: "/audio", blobHref: "/audio", expiresMs: 0, warnings: [],
};
const createUrl = vi.fn(() => "blob:artwork");
const revokeUrl = vi.fn();
beforeEach(() => {
  vi.stubGlobal("URL", class extends URL { static createObjectURL = createUrl; static revokeObjectURL = revokeUrl; });
  createUrl.mockClear(); revokeUrl.mockClear();
  vi.mocked(readAudioBytes).mockReset().mockResolvedValue(new ArrayBuffer(2));
  vi.mocked(decodeWaveform).mockReset().mockResolvedValue([0.2, 1]);
  vi.mocked(readAudioTags).mockReset().mockResolvedValue({ title: "Track", artist: "Artist", artwork: new Blob(["image"], { type: "image/png" }) });
});
afterEach(() => { cleanup(); vi.useRealTimers(); vi.unstubAllGlobals(); });

describe("optional audio analysis lifecycle", () => {
  it("does not fetch large, long or unknown-duration files", () => {
    const { rerender } = renderHook(({ duration, sizeBytes }) => useAudioAnalysis({ ...model, sizeBytes }, duration), { initialProps: { duration: 0, sizeBytes: 1024 } });
    rerender({ duration: 601, sizeBytes: 1024 });
    rerender({ duration: 120, sizeBytes: MAX_AUDIO_ANALYSIS_BYTES + 1 });
    rerender({ duration: Infinity, sizeBytes: 1024 });
    expect(readAudioBytes).not.toHaveBeenCalled();
  });
  it("keeps artwork available when waveform decoding fails and revokes it on teardown", async () => {
    vi.mocked(decodeWaveform).mockRejectedValue(new Error("Codec unavailable"));
    const { result, unmount } = renderHook(() => useAudioAnalysis(model, 120));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current).toMatchObject({ title: "Track", artist: "Artist", artwork: "blob:artwork", peaks: [] });
    unmount();
    expect(revokeUrl).toHaveBeenCalledWith("blob:artwork");
  });
  it("keeps the waveform when embedded metadata parsing fails", async () => {
    vi.mocked(readAudioTags).mockRejectedValue(new Error("Malformed tags"));
    const { result } = renderHook(() => useAudioAnalysis(model, 120));
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.peaks).toEqual([0.2, 1]);
    expect(result.current.artwork).toBeUndefined();
  });
  it("does not repeat analysis when a variable-bitrate duration estimate changes", async () => {
    const { result, rerender } = renderHook(({ duration }) => useAudioAnalysis(model, duration), { initialProps: { duration: 120 } });
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    rerender({ duration: 121 });
    expect(readAudioBytes).toHaveBeenCalledTimes(1);
  });
  it("aborts a stale fetch and does not create artwork when a late decode finishes after unmount", async () => {
    let finish: (peaks: number[]) => void = () => {};
    vi.mocked(decodeWaveform).mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
    const { unmount } = renderHook(() => useAudioAnalysis(model, 120));
    await waitFor(() => { expect(decodeWaveform).toHaveBeenCalled(); });
    const signal = vi.mocked(readAudioBytes).mock.calls[0]?.[1];
    unmount();
    expect(signal?.aborted).toBe(true);
    await act(async () => { finish([1]); await Promise.resolve(); });
    expect(createUrl).not.toHaveBeenCalled();
  });
  it("leaves the loading state when optional decoding exceeds its deadline", async () => {
    vi.useFakeTimers();
    vi.mocked(decodeWaveform).mockImplementationOnce(() => new Promise(() => {}));
    const { result } = renderHook(() => useAudioAnalysis(model, 120));
    await act(async () => { await Promise.resolve(); });
    expect(result.current.loading).toBe(true);
    await act(async () => { await vi.advanceTimersByTimeAsync(20_001); });
    expect(result.current.loading).toBe(false);
  });
});
