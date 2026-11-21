import { describe, expect, it, vi } from "vitest";

import { createManifestPreviewSource, normalizeManifestPath, type ManifestFile } from "../manifestSource.js";
import { supportsArchives, supportsThumbnails, type PreviewModel } from "../types.js";

const files: ManifestFile[] = [
  { path: "plan.md", url: "/blob/plan", kind: "markdown", mimeType: "text/markdown", sizeBytes: 40 },
  { path: "screens/home.png", url: "/blob/home", kind: "image", mimeType: "image/png", sizeBytes: 900, thumbnailUrl: "/thumb/home" },
  { path: "screens/detail.png", url: "/blob/detail", kind: "image", mimeType: "image/png", sizeBytes: 100 },
  { path: "screens/.draft.png", url: "/blob/draft", kind: "image", mimeType: "image/png", sizeBytes: 5 },
  { path: "data/report.bin", url: "/blob/bin", kind: "unsupported", mimeType: "application/octet-stream", sizeBytes: 7, canDownload: false },
];

async function resolved(source: ReturnType<typeof createManifestPreviewSource>, path: string): Promise<PreviewModel> {
  const result = await source.resolve(path);
  if (result.outcome !== "resolved") throw new Error(`expected ${path} to resolve, got ${result.outcome}`);
  return result.model;
}

describe("normalizeManifestPath", () => {
  it("gives every spelling of a path one canonical form", () => {
    expect(normalizeManifestPath("a/./b/../c")).toBe("/a/c");
    expect(normalizeManifestPath("/a//b/")).toBe("/a/b");
    expect(normalizeManifestPath("..\\x")).toBe("/x");
    expect(normalizeManifestPath("")).toBe("/");
  });
});

describe("createManifestPreviewSource", () => {
  it("identifies its scope and declares only the capabilities it can serve", () => {
    const source = createManifestPreviewSource({ scope: "ask:1", files });
    expect(source.scope).toBe("ask:1");
    expect(source.capabilities).toEqual({ thumbnails: true, archives: false, htmlRuntime: false, locate: false, parentFolders: false });
    expect(supportsThumbnails(source)).toBe(true);
    expect(supportsArchives(source)).toBe(false);
    expect(supportsThumbnails(createManifestPreviewSource({ scope: "ask:2", files: files.slice(0, 1) }))).toBe(false);
  });

  it("resolves a file to its supplied kind, size and URL", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    const model = await resolved(source, "screens/home.png");
    expect(model).toMatchObject({ resolvedPath: "/screens/home.png", basename: "home.png", kind: "image", blobHref: "/blob/home", sizeBytes: 900, canPreview: true, canDownload: true, textContentAvailable: false, listingAvailable: false });
  });

  it("resolves the same file from any spelling of its path and marks only text kinds as readable", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    expect((await resolved(source, "/screens/../plan.md")).resolvedPath).toBe("/plan.md");
    expect((await resolved(source, "plan.md")).textContentAvailable).toBe(true);
    const unsupported = await resolved(source, "data/report.bin");
    expect(unsupported).toMatchObject({ canPreview: false, canDownload: false });
  });

  it("resolves implied folders, including the root, as listable directories", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    expect(await resolved(source, "/")).toMatchObject({ kind: "directory", listingAvailable: true, resolvedPath: "/" });
    expect(await resolved(source, "screens")).toMatchObject({ kind: "directory", resolvedPath: "/screens", basename: "screens" });
  });

  it("reports an unknown path as unresolved with nowhere to browse", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    expect(await source.resolve("missing.md")).toEqual({ outcome: "unresolved", inputPath: "missing.md", searchBases: [] });
  });

  it("reads text from the file's URL with same-origin credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("# Plan"));
    const source = createManifestPreviewSource({ scope: "s", files, fetch: fetchMock });
    const text = await source.readText(await resolved(source, "plan.md"));
    expect(text).toMatchObject({ content: "# Plan", truncated: false, kind: "markdown", resolvedPath: "/plan.md" });
    expect(fetchMock).toHaveBeenCalledWith("/blob/plan", expect.objectContaining({ credentials: "same-origin" }));
  });

  it("cuts text at its inline limit and says so", async () => {
    const source = createManifestPreviewSource({ scope: "s", files, maxTextChars: 4, fetch: vi.fn().mockResolvedValue(new Response("abcdefgh")) });
    expect(await source.readText(await resolved(source, "plan.md"))).toMatchObject({ content: "abcd", truncated: true });
  });

  it("fails readably when the bytes cannot be fetched", async () => {
    const source = createManifestPreviewSource({ scope: "s", files, fetch: vi.fn().mockResolvedValue(new Response("nope", { status: 404 })) });
    await expect(source.readText(await resolved(source, "plan.md"))).rejects.toThrow("Could not read plan.md (404)");
  });

  it("lists a folder with folders first, hiding dotfiles until asked", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    const root = await resolved(source, "/");
    const listing = await source.listDirectory(root, { sort: "dirs_first_name", showHidden: false });
    expect(listing.entries.map((entry) => [entry.name, entry.entryType])).toEqual([["data", "directory"], ["screens", "directory"], ["plan.md", "file"]]);
    expect(listing).toMatchObject({ parentPath: "", nextPageToken: "", truncated: false, totalEntries: 3 });
    expect(listing.entries.find((entry) => entry.name === "screens")?.childCount).toBe(3);

    const screens = await resolved(source, "/screens");
    const visible = await source.listDirectory(screens, { sort: "name", showHidden: false });
    expect(visible.entries.map((entry) => entry.name)).toEqual(["detail.png", "home.png"]);
    expect(visible.parentPath).toBe("/");
    const all = await source.listDirectory(screens, { sort: "size_desc", showHidden: true });
    expect(all.entries.map((entry) => entry.name)).toEqual(["home.png", "detail.png", ".draft.png"]);
  });

  it("offers a thumbnail only for the files that supplied one", async () => {
    const source = createManifestPreviewSource({ scope: "s", files });
    const screens = await resolved(source, "/screens");
    expect(source.thumbnailHref?.(screens, { entry: "home.png" })).toBe("/thumb/home");
    expect(source.thumbnailHref?.(screens, { entry: "detail.png" })).toBeUndefined();
  });

  it("ignores empty and duplicate paths instead of failing", async () => {
    const source = createManifestPreviewSource({ scope: "s", files: [{ ...files[0] as ManifestFile, path: "" }, { ...files[0] as ManifestFile }, { ...files[0] as ManifestFile, url: "/other" }] });
    expect((await resolved(source, "plan.md")).blobHref).toBe("/blob/plan");
  });
});
