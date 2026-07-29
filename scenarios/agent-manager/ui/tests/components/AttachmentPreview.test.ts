import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { AttachmentPreview } from "../../src/components/AttachmentPreview.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

test("AttachmentPreview renders image, PDF, progress/error/success states, and removes the selected attachment", () => {
  const onRemove = vi.fn();
  const image = new File(["image"], "photo.png", { type: "image/png" });
  const pdf = new File(["pdf"], "notes.pdf", { type: "application/pdf" });
  const { rerender } = renderWithProviders(createElement(AttachmentPreview, {
    attachments: [
      { id: "pending", file: image, type: "image", previewUrl: "data:image/png;base64,AA", uploadStatus: "pending" },
      { id: "error", file: pdf, type: "pdf", uploadStatus: "error", error: "Upload failed" },
      { id: "done", file: image, type: "image", previewUrl: "data:image/png;base64,BB", uploadStatus: "uploaded" },
    ],
    onRemove,
    isUploading: true,
  }));

  assert.equal(screen.getAllByTestId("attachment-thumbnail").length, 3);
  assert.equal(screen.getAllByAltText("photo.png").length, 2);
  assert.ok(screen.getByText("PDF"));
  assert.ok(screen.getByText("Upload failed"));
  fireEvent.click(screen.getAllByTestId("attachment-remove-button")[1]!);
  assert.deepEqual(onRemove.mock.calls, [["error"]]);

  rerender(createElement(AttachmentPreview, { attachments: [], onRemove, isUploading: false }));
  assert.equal(screen.queryByTestId("attachment-preview-container"), null);
});
