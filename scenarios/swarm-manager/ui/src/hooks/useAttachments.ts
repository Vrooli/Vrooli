export type { CaptureAttachment } from "./useIndexedDBAttachments";
import { useIndexedDBAttachments } from "./useIndexedDBAttachments";
import type { UseIndexedDBAttachmentsReturn } from "./useIndexedDBAttachments";

export type UseAttachmentsReturn = UseIndexedDBAttachmentsReturn;

export function useAttachments(): UseIndexedDBAttachmentsReturn {
  return useIndexedDBAttachments({ dbName: "swarm-clarification-attachments" });
}
