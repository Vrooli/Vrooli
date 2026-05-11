import { describe, expect, it } from "vitest";

import {
  checkAttachmentUploadInvariants,
  transitionAttachmentUpload,
} from "./AttachmentUploadWorkflow";

const file = new File(["hello"], "hello.txt", { type: "text/plain" });

describe("AttachmentUpload workflow", () => {
  it("rejects impossible states", () => {
    expect(() => checkAttachmentUploadInvariants({ status: "succeeded", fileName: "   ", attemptId: "attempt-1" })).toThrow(
      "succeeded upload state requires a file name",
    );
    expect(() => checkAttachmentUploadInvariants({ status: "failed", file, message: " ", attemptId: "attempt-1" })).toThrow(
      "failed upload state requires a message",
    );
    expect(() => checkAttachmentUploadInvariants({ status: "uploading", file, attemptId: " " })).toThrow(
      "uploading upload state requires an attempt id",
    );
  });

  it("ignores stale completion attempts", () => {
    const uploading = transitionAttachmentUpload(
      { status: "selected", file },
      { type: "start", attemptId: "attempt-1" },
    );
    expect(transitionAttachmentUpload(uploading, {
      type: "fail",
      attemptId: "attempt-2",
      message: "network failed",
    })).toEqual(uploading);
    const reselected = transitionAttachmentUpload(uploading, { type: "select", file });
    expect(transitionAttachmentUpload(reselected, {
      type: "succeed",
      attemptId: "attempt-1",
      fileName: file.name,
    })).toEqual(reselected);
    expect(transitionAttachmentUpload({ status: "idle" }, {
      type: "fail",
      attemptId: "attempt-1",
      message: "network failed",
    })).toEqual({ status: "idle" });
  });
});
