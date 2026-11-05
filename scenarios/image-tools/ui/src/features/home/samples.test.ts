/**
 * samples tests — the curated demo set is mostly static data, but
 * `loadSampleFile` has real logic (fetch the bundled asset, wrap the bytes as a
 * File the Workspace can adopt, with a PNG type fallback when the blob reports
 * none). Both paths are exercised here.
 */
import { afterEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_SAMPLE, SAMPLES, loadSampleFile, type SampleImage } from "./samples";

describe("samples data", () => {
  it("exposes a non-undefined default sample that is also first in the set", () => {
    expect(DEFAULT_SAMPLE.key).toBe("product");
    expect(SAMPLES[0]).toBe(DEFAULT_SAMPLE);
  });

  it("every sample carries a url, file name, label key, and a mode", () => {
    expect(SAMPLES.length).toBeGreaterThan(0);
    for (const sample of SAMPLES) {
      expect(typeof sample.url).toBe("string");
      expect(sample.fileName.endsWith(".png")).toBe(true);
      expect(typeof sample.labelKey).toBe("string");
      expect(["edit", "enhance", "create", "analyze"]).toContain(sample.mode);
    }
  });
});

describe("loadSampleFile", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetches the asset and wraps the bytes as a named File preserving the blob type", async () => {
    const blob = new Blob([new Uint8Array([1, 2, 3])], { type: "image/jpeg" });
    const fetchMock = vi.fn().mockResolvedValue({ blob: () => Promise.resolve(blob) });
    vi.stubGlobal("fetch", fetchMock);

    const file = await loadSampleFile(DEFAULT_SAMPLE);

    expect(fetchMock).toHaveBeenCalledWith(DEFAULT_SAMPLE.url);
    expect(file).toBeInstanceOf(File);
    expect(file.name).toBe(DEFAULT_SAMPLE.fileName);
    expect(file.type).toBe("image/jpeg");
  });

  it("falls back to image/png when the blob reports no type", async () => {
    const blob = new Blob([new Uint8Array([1])], { type: "" });
    const fetchMock = vi.fn().mockResolvedValue({ blob: () => Promise.resolve(blob) });
    vi.stubGlobal("fetch", fetchMock);

    const sample: SampleImage = { ...DEFAULT_SAMPLE, fileName: "x.png" };
    const file = await loadSampleFile(sample);

    expect(file.type).toBe("image/png");
    expect(file.name).toBe("x.png");
  });
});
