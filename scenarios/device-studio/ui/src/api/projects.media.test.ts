import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create, toJson } from "@bufbuild/protobuf";
import { SourceMediaSchema, UploadSourceMediaResponseSchema } from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { getSourceMediaContent, sourceMediaContentURL, uploadSourceMedia, verifySourceMediaContent } from "./projects";

describe("project source media REST operations", () => {
  const fetchMock = vi.fn();
  beforeEach(() => { vi.stubGlobal("fetch", fetchMock); });
  afterEach(() => { vi.unstubAllGlobals(); vi.clearAllMocks(); });

  it("uploads the selected file as multipart data and returns the saved metadata", async () => {
    const file = new File(["image"], "scene plate.png", { type: "image/png" });
    const metadata = create(SourceMediaSchema, { id: "media-1", projectId: "project / one", originalFilename: file.name, mediaType: "image/png", sizeBytes: 5n });
    fetchMock.mockResolvedValueOnce(new Response(JSON.stringify(toJson(UploadSourceMediaResponseSchema, create(UploadSourceMediaResponseSchema, { sourceMedia: metadata }))), { status: 200 }));
    const result = await uploadSourceMedia("project / one", file);
    expect(result.id).toBe("media-1");
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toMatch(/\/api\/v1\/projects\/project%20%2F%20one\/media$/);
    expect((init.body as FormData).get("file")).toBeInstanceOf(File);
    expect(((init.body as FormData).get("file") as File).name).toBe("scene plate.png");
  });

  it("fetches content from an encoded project/media route and checks saved metadata", async () => {
    const bytes = new Blob(["image"], { type: "image/png" });
    fetchMock.mockResolvedValueOnce(new Response(bytes, { status: 200, headers: { "content-type": "image/png" } }));
    const result = await getSourceMediaContent("project/one", "media id");
    expect(result.type).toBe("image/png");
    expect(fetchMock.mock.calls[0]?.[0]).toMatch(/\/api\/v1\/projects\/project%2Fone\/media\/media%20id\/content$/);
    const metadata = create(SourceMediaSchema, { sizeBytes: BigInt(result.size), mediaType: "image/png", sha256: "0000000000000000000000000000000000000000000000000000000000000000" });
    vi.stubGlobal("crypto", { subtle: { digest: vi.fn().mockResolvedValue(new Uint8Array(32).buffer) } });
    await expect(verifySourceMediaContent(result, metadata)).resolves.toBe(result);
    await expect(verifySourceMediaContent(new Blob(["x"], { type: "image/png" }), metadata)).rejects.toThrow(/size did not match/);
    await expect(verifySourceMediaContent(new Blob([new Uint8Array(result.size)], { type: "image/jpeg" }), metadata)).rejects.toThrow(/type did not match/);
  });

  it("builds a routed media URL for native video streaming without fetching a Blob", () => {
    const url = sourceMediaContentURL("project/one", "recording id");
    expect(url).toMatch(/\/api\/v1\/projects\/project%2Fone\/media\/recording%20id\/content$/);
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
