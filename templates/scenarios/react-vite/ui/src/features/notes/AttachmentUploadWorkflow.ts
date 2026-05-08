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
  | { readonly status: "uploading"; readonly file: File }
  | { readonly status: "succeeded"; readonly fileName: string }
  | { readonly status: "failed"; readonly file: File; readonly message: string };

export type AttachmentUploadEvent =
  | { readonly type: "select"; readonly file: File }
  | { readonly type: "start" }
  | { readonly type: "succeed"; readonly fileName: string }
  | { readonly type: "fail"; readonly message: string }
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
      next = { status: "uploading", file: state.file };
      break;
    case "succeed":
      if (state.status !== "uploading") {
        throw new Error(`cannot complete upload from ${state.status}`);
      }
      next = { status: "succeeded", fileName: event.fileName };
      break;
    case "fail":
      if (state.status !== "uploading") {
        throw new Error(`cannot fail upload from ${state.status}`);
      }
      next = { status: "failed", file: state.file, message: event.message };
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
