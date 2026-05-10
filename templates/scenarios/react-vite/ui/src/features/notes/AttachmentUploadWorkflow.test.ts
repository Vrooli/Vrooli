import { describe, expect, it } from "vitest";

import {
  assertWorkflowSpecConformance,
  assertFormalArtifactFresh,
  assertFormalTransitionsReplay,
  assertFormalTracesReplay,
  assertTransitionMatrix,
  replayTraces,
  type FormalArtifact,
  type MatrixRow,
  type Trace,
  type WorkflowSpec,
} from "../../test-utils";
import formalArtifact from "./AttachmentUploadWorkflow.formal.generated.json";
import spec from "./AttachmentUploadWorkflow.spec.json";
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
      return { status: "uploading", file };
    case "succeeded":
      return { status: "succeeded", fileName: file.name };
    case "failed":
      return { status: "failed", file, message: "network failed" };
  }
};

const eventFor = (event: AttachmentUploadEventType): AttachmentUploadEvent => {
  switch (event) {
    case "select":
      return { type: "select", file };
    case "start":
      return { type: "start" };
    case "succeed":
      return { type: "succeed", fileName: file.name };
    case "fail":
      return { type: "fail", message: "network failed" };
    case "reset":
      return { type: "reset" };
  }
};

const transitionStatus = (
  status: AttachmentUploadStatus,
  event: AttachmentUploadEventType,
): AttachmentUploadStatus => transitionAttachmentUpload(stateFor(status), eventFor(event)).status;

const matrix = [
  { from: "idle", event: "select", to: "selected" },
  { from: "idle", event: "start", to: "idle", wantError: true },
  { from: "idle", event: "succeed", to: "idle", wantError: true },
  { from: "idle", event: "fail", to: "idle", wantError: true },
  { from: "idle", event: "reset", to: "idle" },

  { from: "selected", event: "select", to: "selected" },
  { from: "selected", event: "start", to: "uploading" },
  { from: "selected", event: "succeed", to: "selected", wantError: true },
  { from: "selected", event: "fail", to: "selected", wantError: true },
  { from: "selected", event: "reset", to: "idle" },

  { from: "uploading", event: "select", to: "selected" },
  { from: "uploading", event: "start", to: "uploading", wantError: true },
  { from: "uploading", event: "succeed", to: "succeeded" },
  { from: "uploading", event: "fail", to: "failed" },
  { from: "uploading", event: "reset", to: "idle" },

  { from: "succeeded", event: "select", to: "selected" },
  { from: "succeeded", event: "start", to: "succeeded", wantError: true },
  { from: "succeeded", event: "succeed", to: "succeeded", wantError: true },
  { from: "succeeded", event: "fail", to: "succeeded", wantError: true },
  { from: "succeeded", event: "reset", to: "idle" },

  { from: "failed", event: "select", to: "selected" },
  { from: "failed", event: "start", to: "uploading" },
  { from: "failed", event: "succeed", to: "failed", wantError: true },
  { from: "failed", event: "fail", to: "failed", wantError: true },
  { from: "failed", event: "reset", to: "idle" },
] as const satisfies readonly MatrixRow<AttachmentUploadStatus, AttachmentUploadEventType>[];

const traces = [
  {
    name: "successful_upload",
    initial: "idle",
    steps: [
      { event: "select", want: "selected" },
      { event: "start", want: "uploading" },
      { event: "succeed", want: "succeeded" },
    ],
  },
  {
    name: "failed_upload_then_retry",
    initial: "idle",
    steps: [
      { event: "select", want: "selected" },
      { event: "start", want: "uploading" },
      { event: "fail", want: "failed" },
      { event: "start", want: "uploading" },
      { event: "succeed", want: "succeeded" },
    ],
  },
  {
    name: "completion_before_start_rejected",
    initial: "selected",
    steps: [
      { event: "succeed", want: "selected", wantError: true },
    ],
  },
] as const satisfies readonly Trace<AttachmentUploadStatus, AttachmentUploadEventType>[];

describe("AttachmentUpload workflow", () => {
  it("covers every status/event pair", () => {
    assertTransitionMatrix(attachmentUploadStatuses, attachmentUploadEvents, matrix, transitionStatus);
  });

  it("replays representative traces", () => {
    replayTraces(traces, transitionStatus);
  });

  it("conforms to the declarative workflow spec", () => {
    assertWorkflowSpecConformance(
      spec as WorkflowSpec,
      attachmentUploadStatuses,
      attachmentUploadEvents,
      matrix,
      traces,
    );
  });

  it("replays generated formal model artifacts", () => {
    const artifact = formalArtifact as FormalArtifact;
    assertFormalArtifactFresh(artifact, {
      modelPath: "ui/src/features/notes/AttachmentUploadWorkflow.qnt",
    });
    assertFormalTransitionsReplay(artifact, attachmentUploadStatuses, attachmentUploadEvents, transitionStatus);
    assertFormalTracesReplay(artifact, attachmentUploadStatuses, attachmentUploadEvents, transitionStatus);
  });

  it("rejects impossible states", () => {
    expect(() => checkAttachmentUploadInvariants({ status: "succeeded", fileName: "   " })).toThrow(
      "succeeded upload state requires a file name",
    );
    expect(() => checkAttachmentUploadInvariants({ status: "failed", file, message: " " })).toThrow(
      "failed upload state requires a message",
    );
  });
});
