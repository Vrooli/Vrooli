import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const key = "web-search:live-health";
const snapshot = { query: "current evidence", cached: false, degraded: true, degradedReason: "engine unavailable", resultCount: 2, at: 123456 };

// [REQ:REQ-P0-007] Operations must preserve degradation signals across reloads.
describe("live search health persistence", () => {
  beforeEach(() => { vi.resetModules(); localStorage.clear(); });
  afterEach(() => { cleanup(); vi.restoreAllMocks(); });

  it("restores a complete prior search response", async () => {
    localStorage.setItem(key, JSON.stringify(snapshot));
    const { useLiveSearchHealth } = await import("./liveSearchHealth");
    const { result } = renderHook(useLiveSearchHealth);
    expect(result.current).toEqual(snapshot);
  });

  it.each(["{broken", "null", '"unexpected"', "{}",
    ...["query", "cached", "degraded", "degradedReason", "resultCount", "at"].map((field) => JSON.stringify({ ...snapshot, [field]: null })),
  ])("ignores corrupt or incomplete persisted response %s", async (raw) => {
    localStorage.setItem(key, raw);
    const { useLiveSearchHealth } = await import("./liveSearchHealth");
    const { result } = renderHook(useLiveSearchHealth);
    expect(result.current).toBeNull();
  });

  it("continues publishing health when browser persistence is denied", async () => {
    const { recordLiveSearchHealth, useLiveSearchHealth } = await import("./liveSearchHealth");
    const { result, unmount } = renderHook(useLiveSearchHealth);
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("quota"); });
    act(() => recordLiveSearchHealth(snapshot));
    expect(result.current).toEqual(snapshot);
    unmount();
    expect(() => recordLiveSearchHealth({ ...snapshot, resultCount: 0 })).not.toThrow();
  });
});
