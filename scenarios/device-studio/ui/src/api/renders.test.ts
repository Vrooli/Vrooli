import { afterEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { RenderOutputSchema, VideoSampleBindingSchema } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { getRenderOutputContent, getVideoSampleBinding, rendersClient, verifyRenderOutputContent } from "./renders";

describe("render output verification", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("accepts only the expected project/job content URL", async () => {
    const fetch = vi.fn().mockResolvedValue(new Response(new Blob(["png"], { type: "image/png" }), { status: 200, headers: { "Content-Type": "image/png" } }));
    vi.stubGlobal("fetch", fetch);
    await expect(getRenderOutputContent("project-1", "render-2", "/api/v1/projects/project-1/renders/render-2/content")).resolves.toHaveProperty("type", "image/png");
    expect(fetch).toHaveBeenCalledTimes(1);
    await expect(getRenderOutputContent("project-1", "render-2", "/api/v1/projects/project-else/renders/render-2/content")).rejects.toThrow(/does not match/);
    await expect(getRenderOutputContent("project-1", "render-2", "https://example.invalid/out.png")).rejects.toThrow(/project-scoped/);
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("requires matching PNG type, size, digest and decoded dimensions", async () => {
    const bytes = new Uint8Array([1, 2, 3]);
    const blob = new Blob([bytes], { type: "image/png" });
    const output = create(RenderOutputSchema, { mediaType: "image/png", sizeBytes: 3n, widthPixels: 12, heightPixels: 8, sha256: "a".repeat(64) });
    vi.stubGlobal("createImageBitmap", vi.fn().mockResolvedValue({ width: 12, height: 8, close: vi.fn() }));
    const invalidDigest = create(RenderOutputSchema, { ...output, sha256: "not-a-sha256" });
    await expect(verifyRenderOutputContent(blob, invalidDigest)).rejects.toThrow(/digest metadata is invalid/);
    await expect(verifyRenderOutputContent(blob, output)).resolves.toBe(blob);
    vi.stubGlobal("createImageBitmap", vi.fn().mockResolvedValue({ width: 13, height: 8, close: vi.fn() }));
    await expect(verifyRenderOutputContent(blob, output)).rejects.toThrow(/dimensions/);
    await expect(verifyRenderOutputContent(new Blob([bytes], { type: "image/jpeg" }), output)).rejects.toThrow(/not a PNG/);
    await expect(verifyRenderOutputContent(new Blob([bytes, bytes], { type: "image/png" }), output)).rejects.toThrow(/size/);
  });

  it("loads the source-derived sample binding for the exact project, shot and render", async () => {
    const binding = create(VideoSampleBindingSchema, { sourceSha256: "a".repeat(64) });
    const request = vi.spyOn(rendersClient, "getVideoSampleBinding").mockResolvedValue({ binding } as never);
    await expect(getVideoSampleBinding("project-1", "shot-2", "render-3")).resolves.toEqual(binding);
    expect(request).toHaveBeenCalledWith({ projectId: "project-1", shotId: "shot-2", renderId: "render-3" });
    request.mockResolvedValue({} as never);
    await expect(getVideoSampleBinding("project-1", "shot-2", "render-3")).rejects.toThrow(/no verified settings claim/);
  });
});
