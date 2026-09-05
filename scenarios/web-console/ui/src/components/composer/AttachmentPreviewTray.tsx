import { Loader2, X } from "lucide-react";

/** A staged image attachment awaiting review + upload-on-send. */
export interface ComposerAttachment {
  id: string;
  file: File;
  /** Object-URL (or data-URL) thumbnail for review. */
  previewUrl: string;
  /** Per-item lifecycle used to show an inline spinner during upload. */
  status?: "staged" | "uploading" | "error";
}

interface AttachmentPreviewTrayProps {
  attachments: ComposerAttachment[];
  onRemove: (id: string) => void;
  removeAriaLabel: string;
}

/**
 * Horizontal tray of reviewable image thumbnails shown inside the composer.
 * Attachments are NEVER submitted until send — this tray only stages them.
 * Modeled on swarm-manager's AttachmentPreviewTray, themed with wc-* tokens.
 */
export function AttachmentPreviewTray({ attachments, onRemove, removeAriaLabel }: AttachmentPreviewTrayProps) {
  if (attachments.length === 0) return null;

  return (
    <div data-testid="composer-attachment-tray" className="mb-2 flex gap-2 overflow-x-auto py-1">
      {attachments.map((att) => (
        <div key={att.id} className="group relative shrink-0">
          <div className="h-20 w-20 overflow-hidden rounded-lg border border-wc-default bg-wc-surface-input">
            <img src={att.previewUrl} alt={att.file.name} className="h-full w-full object-cover" />
          </div>
          {att.status === "uploading" && (
            <div className="absolute inset-0 flex items-center justify-center rounded-lg bg-wc-backdrop">
              <Loader2 className="h-4 w-4 animate-spin text-wc-text-primary" />
            </div>
          )}
          <button
            type="button"
            data-testid={`composer-attachment-remove-${att.id}`}
            onClick={() => onRemove(att.id)}
            className="absolute -end-2 -top-2 rounded-full border border-wc-default bg-wc-surface-raised p-1 text-wc-text-secondary transition hover:border-red-500/50 hover:text-red-400"
            aria-label={removeAriaLabel}
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ))}
    </div>
  );
}
