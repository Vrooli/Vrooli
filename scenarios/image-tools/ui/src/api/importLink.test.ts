import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { importImageLink } from "./importLink";

const makeResponse = (init: { ok?: boolean; status?: number; json?: unknown; blob?: Blob }): Response =>
  ({
    ok: init.ok ?? true,
    status: init.status ?? 200,
    headers: new Headers(),
    json: vi.fn(() => Promise.resolve(init.json ?? {})),
    blob: vi.fn(() => Promise.resolve(init.blob ?? new Blob([]))),
  }) as unknown as Response;

describe("api/importLink", () => {
  beforeEach(() => vi.stubGlobal("fetch", vi.fn()));
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it("posts the URL, fetches the stored image, and returns a File", async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(makeResponse({
        status: 201,
        json: { input_ref: "input/abc.png", format: "png", mime_type: "image/png", width: 2, height: 3 },
      }))
      .mockResolvedValueOnce(makeResponse({ blob: new Blob(["png bytes"], { type: "image/png" }) }));

    const result = await importImageLink("https://images.example/photo.png");

    expect(result).toBeInstanceOf(File);
    expect(result.name).toBe("imported.png");
    expect(result.type).toBe("image/png");
    const [importURL, importInit] = vi.mocked(fetch).mock.calls[0] ?? [];
    if (typeof importURL !== "string" || typeof importInit?.body !== "string") {
      throw new Error("expected the import request to use string URL and body");
    }
    expect(importURL).toContain("/api/v1/import/link");
    expect(JSON.parse(importInit.body)).toEqual({ url: "https://images.example/photo.png" });
    const downloadURL = vi.mocked(fetch).mock.calls[1]?.[0];
    if (typeof downloadURL !== "string") {
      throw new Error("expected the download request to use a string URL");
    }
    expect(downloadURL).toContain("/api/v1/blobs/input/abc.png");
  });
});
