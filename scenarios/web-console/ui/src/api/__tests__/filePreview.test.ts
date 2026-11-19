import { describe, expect, it, vi } from "vitest";

import { create } from "@bufbuild/protobuf";
import { PreviewKind, ResolveResponseSchema, ListArchiveResponseSchema, OpenArchiveEntryResponseSchema } from "@vrooli/proto-types/web-console/v1/file_preview/file_preview_pb";
import type { PreviewModel } from "@vrooli/file-viewer";
import { createSessionPreviewSource, filePreviewClient, previewBlobHref, resolveSessionPath, sessionPreviewSource } from "../filePreview";

const source = createSessionPreviewSource("session");

async function resolveModel(path: string): Promise<PreviewModel> {
  const result = await source.resolve(path);
  if (result.outcome !== "resolved") throw new Error(`expected ${path} to resolve, got ${result.outcome}`);
  return result.model;
}

describe("previewBlobHref", () => {
  it("returns empty for empty input", () => {
    expect(previewBlobHref("")).toBe("");
  });
  it("passes absolute URLs through unchanged", () => {
    expect(previewBlobHref("https://example.com/x")).toBe("https://example.com/x");
  });
  it("keeps a same-origin relative path resolvable (ends with the blob path)", () => {
    const href = previewBlobHref("/api/v1/sessions/s/file-previews/pv/blob");
    expect(href.endsWith("/api/v1/sessions/s/file-previews/pv/blob")).toBe(true);
  });
});

describe("session preview source", () => {
  it("is scoped to its session and offers every capability the host serves", () => {
    expect(source.scope).toBe("session");
    expect(source.capabilities).toEqual({ thumbnails: true, archives: true, htmlRuntime: true, locate: true, parentFolders: true });
  });

  it("hands equal sessions the same object, so identity comparisons hold", () => {
    expect(sessionPreviewSource("a")).toBe(sessionPreviewSource("a"));
    expect(sessionPreviewSource("a")).not.toBe(sessionPreviewSource("b"));
  });
});

describe("spreadsheet preview transport", () => {
  it("decodes the spreadsheet enum as a blob preview without requesting text", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, {
      previewKind: PreviewKind.SPREADSHEET, previewId: "book", basename: "book.xlsx", resolvedPath: "/book.xlsx", canPreview: true, canDownload: true, blobUrl: "/blob", sizeBytes: 100n,
    }));
    try { expect(await resolveModel("/book.xlsx")).toMatchObject({ kind: "spreadsheet", canPreview: true, textContentAvailable: false, blobHref: expect.stringContaining("/blob") }); }
    finally { resolve.mockRestore(); }
  });
});

describe("Word preview transport", () => {
  it("decodes Word documents as blob previews without text transport", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, {
      previewKind: PreviewKind.DOCUMENT, previewId: "doc", basename: "brief.docx", resolvedPath: "/brief.docx", canPreview: true, canDownload: true, blobUrl: "/blob", sizeBytes: 100n,
    }));
    try { expect(await resolveModel("/brief.docx")).toMatchObject({ kind: "document", canPreview: true, textContentAvailable: false, blobHref: expect.stringContaining("/blob") }); }
    finally { resolve.mockRestore(); }
  });
});

describe("publication preview transport", () => {
  it.each([[PreviewKind.DATABASE, "database", "project.sqlite"], [PreviewKind.FONT, "font", "face.woff2"], [PreviewKind.MODEL_3D, "model3d", "model.glb"], [PreviewKind.PRESENTATION, "presentation", "deck.pptx"], [PreviewKind.EBOOK, "ebook", "book.epub"]] as const)("decodes %s as a local blob preview", async (previewKind, kind, basename) => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, { previewKind, previewId: "publication", basename, canPreview: true, canDownload: true, blobUrl: "/blob" }));
    try { expect(await resolveModel(basename)).toMatchObject({ kind, canPreview: true, textContentAvailable: false, blobHref: expect.stringContaining("/blob") }); }
    finally { resolve.mockRestore(); }
  });
});

describe("resolve outcomes", () => {
  const base = { previewKind: PreviewKind.TEXT, inputPath: "notes.md", canPreview: true } as const;
  const match = { resolvedPath: "/a/notes.md", resolutionBasis: "session_cwd", previewKind: PreviewKind.TEXT, canPreview: true, basename: "notes.md" };

  it("reports a relative path that matched several bases as ambiguous", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, { ...base, candidates: [match, { ...match, resolvedPath: "/b/notes.md", resolutionBasis: "project_root" }] }));
    try { expect(await source.resolve("notes.md")).toMatchObject({ outcome: "ambiguous", inputPath: "notes.md", candidates: [{ resolvedPath: "/a/notes.md" }, { resolvedPath: "/b/notes.md" }] }); }
    finally { resolve.mockRestore(); }
  });

  it("reports a relative path that matched nowhere as unresolved, with the folders to browse", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, { ...base, unresolved: true, searchBases: [{ ...match, resolvedPath: "/a", previewKind: PreviewKind.DIRECTORY }] }));
    try { expect(await source.resolve("notes.md")).toMatchObject({ outcome: "unresolved", searchBases: [{ resolvedPath: "/a", kind: "directory" }] }); }
    finally { resolve.mockRestore(); }
  });

  it("names the session's working directory when asked which folder a path means", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, { ...base, candidates: [{ ...match, resolvedPath: "/root", resolutionBasis: "project_root" }, match] }));
    try { expect(await resolveSessionPath("s", ".")).toBe("/a/notes.md"); }
    finally { resolve.mockRestore(); }
  });
});

describe("thumbnails", () => {
  const model = { previewId: "dir", basename: "photos" } as PreviewModel;
  it("asks only for the image types the host can render, naming the child rather than a path", () => {
    expect(source.thumbnailHref?.(model, { entry: "a.png", size: 96.4 })).toContain("/api/v1/sessions/session/file-previews/dir/thumbnail?entry=a.png&size=96");
    expect(source.thumbnailHref?.(model, { entry: "a.webp" })).toBeUndefined();
    expect(source.thumbnailHref?.({ previewId: "file", basename: "shot.JPG" } as PreviewModel)).toContain("/file-previews/file/thumbnail");
  });
});

describe("archive transport", () => {
  it("decodes archive metadata and preserves physical entry IDs", async () => {
    const request = vi.spyOn(filePreviewClient, "listArchive").mockResolvedValue(create(ListArchiveResponseSchema, { format: "ZIP", status: "ready", entries: [{ entryId: "3", name: "same.txt", sizeBytes: 42n, status: "ready" }] }));
    try { expect(await source.listArchive?.({ previewId: "p" } as PreviewModel)).toEqual({ format: "ZIP", status: "ready", entries: [{ id: "3", name: "same.txt", sizeBytes: 42, status: "ready" }] }); } finally { request.mockRestore(); }
  });
  it("marks virtual targets so renderers cannot execute or resolve host resources", async () => {
    const request = vi.spyOn(filePreviewClient, "openArchiveEntry").mockResolvedValue(create(OpenArchiveEntryResponseSchema, { status: "ready", target: { previewId: "virtual", previewKind: PreviewKind.ARCHIVE, archiveMember: true } }));
    try { expect(await source.openArchiveEntry?.({ previewId: "p", entryId: "0" })).toMatchObject({ status: "ready", target: { archiveMember: true, kind: "archive" } }); } finally { request.mockRestore(); }
  });
});
