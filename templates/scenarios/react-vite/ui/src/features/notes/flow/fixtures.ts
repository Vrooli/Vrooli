import type { AttachmentUploadFormalReplayFixtures } from "./generated/replay.helper";

const file = new File(["hello"], "hello.txt", { type: "text/plain" });
const attemptId = "attempt-1";
const networkFailedMessage = "network failed";

export const attachmentUploadFormalFixtures = {
  stateFor: {
    idle: () => ({ status: "idle" }),
    selected: () => ({ status: "selected", file }),
    uploading: () => ({ status: "uploading", file, attemptId }),
    succeeded: () => ({ status: "succeeded", fileName: file.name, attemptId }),
    failed: () => ({ status: "failed", file, message: networkFailedMessage, attemptId }),
  },
  eventFor: {
    select: () => ({ type: "select", file }),
    start: () => ({ type: "start", attemptId }),
    succeed: () => ({ type: "succeed", attemptId, fileName: file.name }),
    fail: () => ({ type: "fail", attemptId, message: networkFailedMessage }),
    reset: () => ({ type: "reset" }),
  },
} satisfies AttachmentUploadFormalReplayFixtures;
