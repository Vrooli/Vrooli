export type { CaptureAttachment, UseIndexedDBAttachmentsReturn as UseCaptureAttachmentsReturn } from "./useIndexedDBAttachments";
import { useIndexedDBAttachments } from "./useIndexedDBAttachments";
import type { UseIndexedDBAttachmentsReturn } from "./useIndexedDBAttachments";

export function useCaptureAttachments(): UseIndexedDBAttachmentsReturn {
  return useIndexedDBAttachments({ dbName: "swarm-capture-attachments" });
}
