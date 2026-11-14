import { CaptureAttachmentPreview } from "../capture/capture-attachment-preview";
import type { CaptureAttachment } from "../../hooks/useIndexedDBAttachments";

interface AttachmentPreviewTrayProps {
  attachments: CaptureAttachment[];
  onRemove: (id: string) => void;
}

export function AttachmentPreviewTray({ attachments, onRemove }: AttachmentPreviewTrayProps) {
  return <CaptureAttachmentPreview attachments={attachments} onRemove={onRemove} />;
}
