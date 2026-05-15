import { useIndexedDBAttachments, type UseIndexedDBAttachmentsReturn } from "../../hooks/useIndexedDBAttachments";

export function useComposerImageAttachments(dbName: string): UseIndexedDBAttachmentsReturn {
  return useIndexedDBAttachments({ dbName });
}
