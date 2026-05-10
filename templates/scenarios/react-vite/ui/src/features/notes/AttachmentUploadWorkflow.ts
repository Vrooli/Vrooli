export const attachmentUploadStatuses = [
  "idle",
  "selected",
  "uploading",
  "succeeded",
  "failed",
] as const;

export type AttachmentUploadStatus = (typeof attachmentUploadStatuses)[number];

export const attachmentUploadEvents = [
  "select",
  "start",
  "succeed",
  "fail",
  "reset",
] as const;

export type AttachmentUploadEventType = (typeof attachmentUploadEvents)[number];

export type AttachmentUploadState =
  | { readonly status: "idle" }
  | { readonly status: "selected"; readonly file: File }
  | { readonly status: "uploading"; readonly file: File; readonly attemptId: string }
  | { readonly status: "succeeded"; readonly fileName: string; readonly attemptId: string }
  | { readonly status: "failed"; readonly file: File; readonly message: string; readonly attemptId: string };

export type AttachmentUploadEvent =
  | { readonly type: "select"; readonly file: File }
  | { readonly type: "start"; readonly attemptId: string }
  | { readonly type: "succeed"; readonly attemptId: string; readonly fileName: string }
  | { readonly type: "fail"; readonly attemptId: string; readonly message: string }
  | { readonly type: "reset" };

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
  switch (event.type) {
    case "select":
      next = { status: "selected", file: event.file };
      break;
    case "start":
      if (state.status !== "selected" && state.status !== "failed") {
        throw new Error(`cannot start upload from ${state.status}`);
      }
      if (event.attemptId.trim() === "") {
        throw new Error("upload attempt id is required");
      }
      next = { status: "uploading", file: state.file, attemptId: event.attemptId };
      break;
    case "succeed":
      if (state.status !== "uploading" || state.attemptId !== event.attemptId) {
        next = state;
        break;
      }
      next = { status: "succeeded", fileName: event.fileName, attemptId: event.attemptId };
      break;
    case "fail":
      if (state.status !== "uploading" || state.attemptId !== event.attemptId) {
        next = state;
        break;
      }
      next = { status: "failed", file: state.file, message: event.message, attemptId: event.attemptId };
      break;
    case "reset":
      next = initialAttachmentUploadState;
      break;
  }
  checkAttachmentUploadInvariants(next);
  return next;
};

export const canStartAttachmentUpload = (state: AttachmentUploadState): state is StartableAttachmentUploadState =>
  state.status === "selected" || state.status === "failed";
