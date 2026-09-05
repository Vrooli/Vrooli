import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, test, vi } from "vitest";
import { useAttachments } from "../../src/hooks/useAttachments.js";

afterEach(() => vi.unstubAllGlobals());

test("useAttachments uploads, persists, restores, and removes attachment evidence", async () => {
  const upload = vi.fn(async () => ({ id: "server-1", file_name: "evidence.pdf", content_type: "application/pdf", file_size: 3, storage_path: "/uploads/1", url: "https://example.test/1" }));
  const { result } = renderHook(() => useAttachments(upload));
  await act(async () => { result.current.addAttachment(new File(["pdf"], "evidence.pdf", { type: "application/pdf" })); });
  await waitFor(() => assert.equal(result.current.attachments[0]?.uploadStatus, "uploaded"));
  assert.deepEqual(result.current.getUploadedIds(), ["server-1"]);
  assert.deepEqual(result.current.getPersistedAttachments(), [{ serverId: "server-1", fileName: "evidence.pdf", contentType: "application/pdf", serverPath: "/uploads/1", url: "" }]);
  await act(async () => { result.current.removeAttachment(result.current.attachments[0]!.id); });
  assert.equal(result.current.attachments.length, 0);
  await act(async () => { result.current.restoreAttachments([{ serverId: "server-2", fileName: "screenshot.png", contentType: "image/png", serverPath: "/uploads/2", url: "blob:preview" }]); });
  assert.equal(result.current.allUploaded, true);
  assert.deepEqual(result.current.getUploadedIds(), ["server-2"]);
});

test("useAttachments retains failed uploads as actionable errors", async () => {
  const { result } = renderHook(() => useAttachments(async () => { throw new Error("upload rejected"); }));
  await act(async () => { result.current.addAttachment(new File(["x"], "bad.pdf", { type: "application/pdf" })); });
  await waitFor(() => assert.equal(result.current.attachments[0]?.uploadStatus, "error"));
  assert.equal(result.current.hasErrors, true);
  assert.equal(result.current.allUploaded, false);
  assert.equal(result.current.attachments[0]?.error, "upload rejected");
});

test("useAttachments maps server upload limits and transport failures into durable operator errors", async () => {
  const fetch = vi.fn()
    .mockResolvedValueOnce(new Response(null, { status: 413 }))
    .mockResolvedValueOnce(new Response(null, { status: 415 }))
    .mockResolvedValueOnce(new Response(null, { status: 502 }));
  vi.stubGlobal("fetch", fetch);
  const { result } = renderHook(() => useAttachments());

  await act(async () => {
    result.current.addAttachment(new File(["large"], "large.pdf", { type: "application/pdf" }));
    result.current.addAttachment(new File(["unsupported"], "unsupported.pdf", { type: "application/pdf" }));
    result.current.addAttachment(new File(["outage"], "outage.pdf", { type: "application/pdf" }));
  });

  await waitFor(() => assert.equal(result.current.attachments.filter((attachment) => attachment.uploadStatus === "error").length, 3));
  assert.deepEqual(result.current.attachments.map((attachment) => attachment.error), [
    "File is too large",
    "File type not supported",
    "Failed to upload: 502",
  ]);
  assert.equal(fetch.mock.calls.every(([, init]) => (init as RequestInit).method === "POST"), true);
});

test("useAttachments revokes restored object URLs when evidence is removed or cleared", async () => {
  const revoke = vi.fn();
  vi.stubGlobal("URL", { revokeObjectURL: revoke });
  const { result } = renderHook(() => useAttachments());

  await act(async () => {
    result.current.restoreAttachments([
      { serverId: "image", fileName: "screen.png", contentType: "image/png", serverPath: "/image", url: "blob:image" },
      { serverId: "pdf", fileName: "report.pdf", contentType: "application/pdf", serverPath: "/pdf", url: "https://files.test/report" },
    ]);
  });
  await act(async () => result.current.removeAttachment(result.current.attachments[0]!.id));
  await act(async () => result.current.clearAttachments());

  assert.deepEqual(revoke.mock.calls, [["blob:image"]]);
  assert.equal(result.current.attachments.length, 0);
});

test("useAttachments generates image previews while upload state remains independently usable", async () => {
  let completeUpload!: (response: { id: string; file_name: string; content_type: string; file_size: number; storage_path: string; url: string }) => void;
  const upload = vi.fn(() => new Promise<typeof completeUpload extends (response: infer Response) => void ? Response : never>((resolve) => {
    completeUpload = resolve;
  }));
  const { result } = renderHook(() => useAttachments(upload));

  await act(async () => result.current.addAttachment(new File(["pixel"], "screen.png", { type: "image/png" })));
  await waitFor(() => assert.equal(result.current.attachments[0]?.uploadStatus, "uploading"));
  await waitFor(() => assert.match(result.current.attachments[0]?.previewUrl ?? "", /^data:image\/png/));
  assert.equal(result.current.isUploading, true);

  await act(async () => completeUpload({ id: "screen", file_name: "screen.png", content_type: "image/png", file_size: 5, storage_path: "/screens/1", url: "https://files.test/screen" }));
  await waitFor(() => assert.equal(result.current.allUploaded, true));
  assert.match(result.current.getPersistedAttachments()[0]?.url ?? "", /^data:image\/png/);
});
