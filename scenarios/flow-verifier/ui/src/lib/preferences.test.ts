import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_SETTINGS, fetchSettings, putSettings, readCache, writeCache } from "./preferences";

describe("preferences", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("read/write cache round trip", () => {
    writeCache({ ...DEFAULT_SETTINGS, theme: "dark" });
    expect(readCache()?.theme).toBe("dark");
  });

  it("readCache merges partial cache with defaults", () => {
    window.localStorage.setItem(
      "flow-verifier.settings.cache.v1",
      JSON.stringify({ theme: "dark" }),
    );
    const c = readCache();
    expect(c?.theme).toBe("dark");
    expect(c?.density).toBe("comfortable");
  });

  it("fetchSettings merges server response with defaults and caches", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ theme: "dark", fontScale: "lg" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    const s = await fetchSettings();
    expect(s.theme).toBe("dark");
    expect(s.fontScale).toBe("lg");
    expect(s.density).toBe("comfortable");
    expect(readCache()?.theme).toBe("dark");
  });

  it("putSettings sends patch and updates cache", async () => {
    const spy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ theme: "light", density: "compact" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );
    const s = await putSettings({ density: "compact" });
    expect(s.density).toBe("compact");
    expect(spy).toHaveBeenCalledOnce();
    const [, init] = spy.mock.calls[0]!;
    expect((init as RequestInit).method).toBe("PUT");
  });
});
