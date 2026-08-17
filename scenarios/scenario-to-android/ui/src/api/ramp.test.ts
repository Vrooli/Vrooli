import { afterEach, describe, expect, it, vi } from "vitest";

import { getRampJson, targetsFromPayload } from "./ramp";

describe("api/ramp", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("normalizes both target payload shapes and rejects malformed payloads", () => {
    const target = { id: "emulator", available: true };
    expect(targetsFromPayload([target])).toEqual([target]);
    expect(targetsFromPayload({ targets: [target] })).toEqual([target]);
    expect(targetsFromPayload({ targets: "not-an-array" })).toEqual([]);
    expect(targetsFromPayload(null)).toEqual([]);
  });

  it("returns JSON for successful ramp requests", async () => {
    const fetchSpy = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    vi.stubGlobal("fetch", fetchSpy);

    await expect(getRampJson<{ ok: boolean }>("/android/targets")).resolves.toEqual({ ok: true });
    expect(fetchSpy).toHaveBeenCalledWith(expect.stringContaining("/android/targets"), { cache: "no-store" });
  });

  it("turns HTTP failures into a useful error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: false, status: 503 }));

    await expect(getRampJson("/android/targets")).rejects.toThrow("Android ramp request failed (503)");
  });
});
