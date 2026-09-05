import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { getDocument, listCollections, listDocuments, queryCorpus } from "./documentManager";

describe("document-manager Connect helpers", () => {
  let fetchSpy: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    fetchSpy = vi.fn().mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ ok: true }), { status: 200 })));
    vi.stubGlobal("fetch", fetchSpy);
  });

  afterEach(() => vi.unstubAllGlobals());

  it("posts the corpus and intake requests with their proto fields", async () => {
    await listCollections();
    await listDocuments();
    await queryCorpus("anchor", "collection-1");
    await getDocument("document-1");

    expect(fetchSpy).toHaveBeenCalledTimes(4);
    expect(JSON.parse(fetchSpy.mock.calls[2]![1].body as string)).toMatchObject({
      text: "anchor",
      collection_id: "collection-1",
      limit: 20,
    });
    expect(JSON.parse(fetchSpy.mock.calls[3]![1].body as string)).toEqual({ id: "document-1" });
  });

  it("uses an empty collection id and reports non-2xx responses", async () => {
    await queryCorpus("anchor");
    expect(JSON.parse(fetchSpy.mock.calls[0]![1].body as string)).toMatchObject({ collection_id: "" });

    fetchSpy.mockResolvedValueOnce(new Response("{}", { status: 503 }));
    await expect(listCollections()).rejects.toThrow("API request failed (503)");
  });

  it("normalizes protojson omitted repeated fields for empty services", async () => {
    fetchSpy
      .mockResolvedValueOnce(new Response("{}", { status: 200 }))
      .mockResolvedValueOnce(new Response("{}", { status: 200 }))
      .mockResolvedValueOnce(new Response("{}", { status: 200 }));

    await expect(listCollections()).resolves.toEqual({ collections: [] });
    await expect(listDocuments()).resolves.toEqual({ documents: [] });
    await expect(queryCorpus("empty")).resolves.toEqual({ results: [], partial: false });
  });
});
