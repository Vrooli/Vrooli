import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { test, vi } from "vitest";
import { useAttachments } from "../../src/hooks/useAttachments.js";

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
