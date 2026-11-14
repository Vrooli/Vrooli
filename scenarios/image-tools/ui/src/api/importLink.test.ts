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
    expect(String(importURL)).toContain("/api/v1/import/link");
    expect(JSON.parse(String(importInit?.body))).toEqual({ url: "https://images.example/photo.png" });
    expect(String(vi.mocked(fetch).mock.calls[1]?.[0])).toContain("/api/v1/blobs/input/abc.png");
  });
});
