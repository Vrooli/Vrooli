import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getCatalogAsset, getComponentExperience, listCatalogAssets, listComponentStories } from "./components";

describe("api/components catalog helpers", () => {
  const fetchSpy = vi.fn();

  beforeEach(() => vi.stubGlobal("fetch", fetchSpy));
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchSpy.mockReset();
  });

  it("forwards catalog and story reads to the generated Components client", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ components: [] }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(listCatalogAssets({ limit: 20, assetKind: 1 })).resolves.toMatchObject({ components: [] });

    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ component: { id: "asset-1" } }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(getCatalogAsset("asset-1")).resolves.toMatchObject({ component: { id: "asset-1" } });

    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ stories: [] }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(listComponentStories({ componentId: "asset-1", version: "1.0.0" })).resolves.toMatchObject({ stories: [] });
  });

  it("returns experience data and rejects a response with no experience contract", async () => {
    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ experience: { componentId: "asset-1", states: [], claims: [], evidence: [] } }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(getComponentExperience("asset-1")).resolves.toMatchObject({ componentId: "asset-1" });

    fetchSpy.mockResolvedValueOnce(new Response(JSON.stringify({ component: { id: "asset-1" } }), { status: 200, headers: { "content-type": "application/json" } }));
    await expect(getComponentExperience("asset-1")).rejects.toThrow("experience was not returned");
  });
});
