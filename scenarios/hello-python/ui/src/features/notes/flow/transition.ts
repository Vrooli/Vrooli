import type {
  AttachmentUploadEvent,
  AttachmentUploadStatus,
  AttachmentUploadState,
} from "./generated/runtime";
import {
  isAttachmentUploadEventValid,
  nextAttachmentUploadStatus,
} from "./generated/runtime";

export {
  attachmentUploadEvents,
  attachmentUploadStatuses,
  isAttachmentUploadEventValid,
  nextAttachmentUploadStatus,
  transitionAttachmentUploadStatus,
} from "./generated/runtime";
export type {
  AttachmentUploadEvent,
  AttachmentUploadEventType,
  AttachmentUploadState,
  AttachmentUploadStatus,
} from "./generated/runtime";

export type StartableAttachmentUploadState = Extract<
  AttachmentUploadState,
  { readonly status: "selected" } | { readonly status: "failed" }
>;

export const initialAttachmentUploadState: AttachmentUploadState = { status: "idle" };

export const attachmentUploadStatusOf = (state: AttachmentUploadState): AttachmentUploadStatus =>
  state.status;

export const checkAttachmentUploadInvariants = (state: AttachmentUploadState): void => {
  if (state.status === "succeeded" && state.fileName.trim() === "") {
    throw new Error("succeeded upload state requires a file name");
  }
  if (state.status === "failed" && state.message.trim() === "") {
    throw new Error("failed upload state requires a message");
  }
  if ((state.status === "uploading" || state.status === "succeeded" || state.status === "failed")
    && state.attemptId.trim() === "") {
    throw new Error(`${state.status} upload state requires an attempt id`);
  }
};

export const transitionAttachmentUpload = (
  state: AttachmentUploadState,
  event: AttachmentUploadEvent,
): AttachmentUploadState => {
  let next: AttachmentUploadState;
  let matchesFormalStatus = true;
  switch (event.type) {
    case "select":
      next = { status: "selected", file: event.file };
      break;
    case "start":
      if (!isAttachmentUploadEventValid(state.status, event.type) || (state.status !== "selected" && state.status !== "failed")) {
        throw new Error(`cannot start upload from ${state.status}`);
      }
      if (event.attemptId.trim() === "") {
        throw new Error("upload attempt id is required");
      }
      next = { status: "uploading", file: state.file, attemptId: event.attemptId };
      break;
    case "succeed":
      if (!isAttachmentUploadEventValid(state.status, event.type) || state.status !== "uploading" || state.attemptId !== event.attemptId) {
        next = state;
        matchesFormalStatus = state.status !== "uploading";
        break;
      }
      next = { status: "succeeded", fileName: event.fileName, attemptId: event.attemptId };
      break;
    case "fail":
      if (!isAttachmentUploadEventValid(state.status, event.type) || state.status !== "uploading" || state.attemptId !== event.attemptId) {
        next = state;
        matchesFormalStatus = state.status !== "uploading";
        break;
      }
      next = { status: "failed", file: state.file, message: event.message, attemptId: event.attemptId };
      break;
    case "reset":
      next = initialAttachmentUploadState;
      break;
  }
  const expectedStatus = nextAttachmentUploadStatus(state.status, event.type);
  if (matchesFormalStatus && expectedStatus !== next.status) {
    throw new Error(`attachment upload transition drift: ${state.status}/${event.type} produced ${next.status}, want ${expectedStatus}`);
  }
  checkAttachmentUploadInvariants(next);
  return next;
};

export const canStartAttachmentUpload = (state: AttachmentUploadState): state is StartableAttachmentUploadState =>
  state.status === "selected" || state.status === "failed";
