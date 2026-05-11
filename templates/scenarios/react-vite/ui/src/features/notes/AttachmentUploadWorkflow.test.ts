import { describe, expect, it } from "vitest";

import {
  assertFormalArtifactFresh,
  assertFormalTransitionsReplay,
  assertFormalTracesReplay,
  type FormalArtifact,
} from "../../test-utils";
import formalArtifact from "./AttachmentUploadWorkflow.formal.generated.json";
import {
  attachmentUploadEvents,
  attachmentUploadStatuses,
  checkAttachmentUploadInvariants,
  transitionAttachmentUpload,
  type AttachmentUploadEvent,
  type AttachmentUploadEventType,
  type AttachmentUploadState,
  type AttachmentUploadStatus,
} from "./AttachmentUploadWorkflow";

const file = new File(["hello"], "hello.txt", { type: "text/plain" });

const stateFor = (status: AttachmentUploadStatus): AttachmentUploadState => {
  switch (status) {
    case "idle":
      return { status: "idle" };
    case "selected":
      return { status: "selected", file };
    case "uploading":
      return { status: "uploading", file, attemptId: "attempt-1" };
    case "succeeded":
      return { status: "succeeded", fileName: file.name, attemptId: "attempt-1" };
    case "failed":
      return { status: "failed", file, message: "network failed", attemptId: "attempt-1" };
  }
};

const eventFor = (event: AttachmentUploadEventType): AttachmentUploadEvent => {
  switch (event) {
    case "select":
      return { type: "select", file };
    case "start":
      return { type: "start", attemptId: "attempt-1" };
    case "succeed":
      return { type: "succeed", attemptId: "attempt-1", fileName: file.name };
    case "fail":
      return { type: "fail", attemptId: "attempt-1", message: "network failed" };
    case "reset":
      return { type: "reset" };
  }
};

const transitionStatus = (
  status: AttachmentUploadStatus,
  event: AttachmentUploadEventType,
): AttachmentUploadStatus => transitionAttachmentUpload(stateFor(status), eventFor(event)).status;

describe("AttachmentUpload workflow", () => {
  it("covers every status/event pair", () => {
    assertFormalTransitionsReplay(
      formalArtifact as FormalArtifact,
      attachmentUploadStatuses,
      attachmentUploadEvents,
      transitionStatus,
    );
  });

  it("replays representative traces", () => {
    assertFormalTracesReplay(
      formalArtifact as FormalArtifact,
      attachmentUploadStatuses,
      attachmentUploadEvents,
      transitionStatus,
    );
  });

  it("replays generated formal model artifacts", () => {
    const artifact = formalArtifact as FormalArtifact;
    assertFormalArtifactFresh(artifact, {
      contractPath: "ui/src/features/notes/AttachmentUploadWorkflow.flow.json",
      modelPath: "ui/src/features/notes/AttachmentUploadWorkflow.qnt",
      generatorPath: "tools/temporal-model",
      invariants: [
        "TypeOK",
        "TerminalClosure",
        "IllegalTransitionsPreserveState",
        "AllDeclaredTransitionsCovered",
        "StaleCompletionIsIgnored",
      ],
    });
    assertFormalTransitionsReplay(artifact, attachmentUploadStatuses, attachmentUploadEvents, transitionStatus);
    assertFormalTracesReplay(artifact, attachmentUploadStatuses, attachmentUploadEvents, transitionStatus);
  });

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
