import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { listVersionLedger } from "./versionLedger";

describe("api/versionLedger", () => {
  const fetchSpy = vi.fn();

  beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchSpy.mockReset();
  });

  it("reads version rows through the lifecycle endpoint", async () => {
    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ rows: [{ libraryId: "library:Chart", version: "1.0.0" }] }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    );

    await expect(listVersionLedger("library:Chart")).resolves.toMatchObject([
      { libraryId: "library:Chart", version: "1.0.0" },
    ]);
    expect(fetchSpy).toHaveBeenCalledWith(
      expect.stringContaining("VersionLifecycleService/ListVersionLedger"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ libraryId: "library:Chart" }),
      }),
    );
  });

  it("returns an empty list when the response omits rows and decodes failures", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200 }));
    await expect(listVersionLedger("library:Missing")).resolves.toEqual([]);

    fetchSpy.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: "unavailable", message: "ledger unavailable" }), {
        status: 503,
      }),
    );
    await expect(listVersionLedger("library:Missing")).rejects.toThrow("ledger unavailable");
  });
});
