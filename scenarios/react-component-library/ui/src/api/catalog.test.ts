import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { describeCapabilities, getCatalogCoverage, listCatalogNextWork } from "./catalog";

describe("api/catalog helpers", () => {
  const fetchSpy = vi.fn();

  beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchSpy.mockReset();
  });

  it("reads coverage and ranked next work through the generated Connect client", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ report: { rows: [], totals: {}, byDomain: [], byPriority: [], maturity: { total: 0, atOrAboveTarget: 0, byRung: {} } } }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(getCatalogCoverage()).resolves.toMatchObject({ maturity: { total: 0 } });

    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ rows: [], maturity: { total: 0, atOrAboveTarget: 0, byRung: {} } }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(listCatalogNextWork(8)).resolves.toMatchObject({ rows: [] });
  });

  it("reports missing coverage and capability registry failures", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({}), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(getCatalogCoverage()).rejects.toThrow("coverage was not returned");

    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ definitions: [], states: [{ id: "agent-manager", status: "available" }] }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(describeCapabilities()).resolves.toMatchObject({ states: [{ id: "agent-manager" }] });

    fetchSpy.mockResolvedValueOnce(new Response("unavailable", { status: 503 }));
    await expect(describeCapabilities()).rejects.toThrow("capability registry returned 503");
  });
});
