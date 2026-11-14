import { describe, expect, it, vi } from "vitest";

import { create } from "@bufbuild/protobuf";
import { PreviewKind, ResolveResponseSchema, ListArchiveResponseSchema, OpenArchiveEntryResponseSchema } from "@vrooli/proto-types/web-console/v1/file_preview/file_preview_pb";
import { filePreviewClient, resolveFilePreview, previewBlobHref, listArchive, openArchiveEntry } from "../filePreview";

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

describe("spreadsheet preview transport", () => {
  it("decodes the spreadsheet enum as a blob preview without requesting text", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, {
      previewKind: PreviewKind.SPREADSHEET, previewId: "book", basename: "book.xlsx", resolvedPath: "/book.xlsx", canPreview: true, canDownload: true, blobUrl: "/blob", sizeBytes: 100n,
    }));
    try { expect(await resolveFilePreview("session", "/book.xlsx")).toMatchObject({ kind: "spreadsheet", canPreview: true, textContentAvailable: false, blobUrl: "/blob" }); }
    finally { resolve.mockRestore(); }
  });
});

describe("Word preview transport", () => {
  it("decodes Word documents as blob previews without text transport", async () => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, {
      previewKind: PreviewKind.DOCUMENT, previewId: "doc", basename: "brief.docx", resolvedPath: "/brief.docx", canPreview: true, canDownload: true, blobUrl: "/blob", sizeBytes: 100n,
    }));
    try { expect(await resolveFilePreview("session", "/brief.docx")).toMatchObject({ kind: "document", canPreview: true, textContentAvailable: false, blobUrl: "/blob" }); }
    finally { resolve.mockRestore(); }
  });
});

describe("publication preview transport", () => {
  it.each([[PreviewKind.DATABASE, "database", "project.sqlite"], [PreviewKind.FONT, "font", "face.woff2"], [PreviewKind.MODEL_3D, "model3d", "model.glb"], [PreviewKind.PRESENTATION, "presentation", "deck.pptx"], [PreviewKind.EBOOK, "ebook", "book.epub"]] as const)("decodes %s as a local blob preview", async (previewKind, kind, basename) => {
    const resolve = vi.spyOn(filePreviewClient, "resolve").mockResolvedValue(create(ResolveResponseSchema, { previewKind, previewId: "publication", basename, canPreview: true, canDownload: true, blobUrl: "/blob" }));
    try { expect(await resolveFilePreview("session", basename)).toMatchObject({ kind, canPreview: true, textContentAvailable: false, blobUrl: "/blob" }); }
    finally { resolve.mockRestore(); }
  });
});


describe("archive transport", () => {
  it("decodes archive metadata and preserves physical entry IDs", async () => {
    const request=vi.spyOn(filePreviewClient,"listArchive").mockResolvedValue(create(ListArchiveResponseSchema, {format:"ZIP",status:"ready",entries:[{entryId:"3",name:"same.txt",sizeBytes:42n,status:"ready"}]}));
    try {expect(await listArchive("s","p")).toEqual({format:"ZIP",status:"ready",entries:[{id:"3",name:"same.txt",sizeBytes:42,status:"ready"}]});} finally {request.mockRestore();}
  });
  it("marks virtual targets so renderers cannot execute or resolve host resources", async () => {
    const request=vi.spyOn(filePreviewClient,"openArchiveEntry").mockResolvedValue(create(OpenArchiveEntryResponseSchema,{status:"ready",target:{previewId:"virtual",previewKind:PreviewKind.ARCHIVE,archiveMember:true}}));
    try {expect(await openArchiveEntry("s",{previewId:"p",entryId:"0"})).toMatchObject({status:"ready",target:{archiveMember:true,kind:"archive"}});} finally {request.mockRestore();}
  });
});
