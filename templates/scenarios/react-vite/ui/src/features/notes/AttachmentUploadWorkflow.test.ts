import { describe, expect, it } from "vitest";

import {
  assertFormalTransitionsReplay,
  assertFormalTracesReplay,
  transitionFromReplayAdapter,
  type FormalArtifact,
} from "../../test-utils";
import { assertFormalArtifactFreshFromFiles } from "../../test-utils/modeltest/formal.node";
import formalArtifact from "./AttachmentUploadWorkflow.formal.generated.json";
import {
  attachmentUploadFormalExpectation,
  attachmentUploadReplayFixtureContract,
  type AttachmentUploadEventFixtureMap,
  type AttachmentUploadStateFixtureMap,
} from "./AttachmentUploadWorkflow.generated";
import {
  attachmentUploadEvents,
  attachmentUploadStatuses,
  checkAttachmentUploadInvariants,
  transitionAttachmentUpload,
} from "./AttachmentUploadWorkflow";

const file = new File(["hello"], "hello.txt", { type: "text/plain" });
const attemptId = "attempt-1";
const networkFailedMessage = "network failed";

const stateFor = {
  idle: () => ({ status: "idle" }),
  selected: () => ({ status: "selected", file }),
  uploading: () => ({ status: "uploading", file, attemptId }),
  succeeded: () => ({ status: "succeeded", fileName: file.name, attemptId }),
  failed: () => ({ status: "failed", file, message: networkFailedMessage, attemptId }),
} satisfies AttachmentUploadStateFixtureMap;

const eventFor = {
  select: () => ({ type: "select", file }),
  start: () => ({ type: "start", attemptId }),
  succeed: () => ({ type: "succeed", attemptId, fileName: file.name }),
  fail: () => ({ type: "fail", attemptId, message: networkFailedMessage }),
  reset: () => ({ type: "reset" }),
} satisfies AttachmentUploadEventFixtureMap;

const transitionStatus = transitionFromReplayAdapter({
  states: attachmentUploadReplayFixtureContract.states,
  events: attachmentUploadReplayFixtureContract.events,
  stateFor,
  eventFor,
  statusOf: (state) => state.status,
  transition: transitionAttachmentUpload,
});

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
    assertFormalArtifactFreshFromFiles(artifact, attachmentUploadFormalExpectation);
    assertFormalTransitionsReplay(artifact, attachmentUploadReplayFixtureContract.states, attachmentUploadReplayFixtureContract.events, transitionStatus);
    assertFormalTracesReplay(artifact, attachmentUploadReplayFixtureContract.states, attachmentUploadReplayFixtureContract.events, transitionStatus);
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
