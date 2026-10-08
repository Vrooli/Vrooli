/**
 * CaptureAttachmentPreview — Shows image thumbnails above the quick-capture input.
 *
 * Adapted from agent-manager's AttachmentPreview component, simplified
 * (no upload status tracking since files submit atomically with the capture).
 */

import { X } from "lucide-react";
import type { CaptureAttachment } from "../../hooks/useCaptureAttachments";
import { selectors } from "../../consts/selectors";

interface CaptureAttachmentPreviewProps {
  attachments: CaptureAttachment[];
  onRemove: (id: string) => void;
}

export function CaptureAttachmentPreview({ attachments, onRemove }: CaptureAttachmentPreviewProps) {
  if (attachments.length === 0) return null;

  return (
    <div className="mb-2 flex gap-2 overflow-x-auto px-1 py-1">
      {attachments.map((att) => (
        <div key={att.id} className="group relative shrink-0">
          <div className="h-20 w-20 overflow-hidden rounded-lg border border-slate-700 bg-slate-800/50">
            <img
              src={att.previewUrl}
              alt={att.file.name}
              className="h-full w-full object-cover"
            />
          </div>
          <button
            onClick={() => onRemove(att.id)}
            className="absolute -right-2 -top-2 rounded-full border border-slate-600 bg-slate-800 p-1 opacity-0 transition-opacity hover:border-red-500/50 hover:bg-red-500/20 group-hover:opacity-100"
            data-testid={selectors.agentSessions.composerImagePreviewRemove}
          >
            <X className="h-3 w-3 text-white" />
          </button>
          <div className="pointer-events-none absolute -bottom-1 left-0 right-0 mx-1 truncate rounded bg-slate-900 border border-slate-700 px-1 py-0.5 text-xs text-slate-300 opacity-0 transition-opacity group-hover:opacity-100">
            {att.file.name}
          </div>
        </div>
      ))}
    </div>
  );
}
