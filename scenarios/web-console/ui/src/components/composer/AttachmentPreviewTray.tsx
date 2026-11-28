import { FileText, Folder, Loader2, Monitor, Music, Play, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { strings } from "../../consts/strings";
import { formatFileSize, type AttachmentKind } from "../../lib/attachments";

interface ComposerAttachmentBase {
  id: string;
  /** Display name (the file or folder's base name). */
  name: string;
  /** Preview category derived from the file type. */
  kind: AttachmentKind;
  /** Original byte size, for the generic file preview (0 when unknown). */
  sizeBytes: number;
  /** Per-item lifecycle used to show an inline spinner during upload. */
  status?: "staged" | "uploading" | "error";
}

/** A file from this device, uploaded as a copy when the message is sent. */
export interface UploadAttachment extends ComposerAttachmentBase {
  source: "upload";
  file: File;
  /** Object-URL (or data-URL) preview for review. */
  previewUrl: string;
}

/** A file or folder already on the host, sent by its absolute path. */
export interface HostAttachment extends ComposerAttachmentBase {
  source: "host";
  path: string;
  isDirectory: boolean;
}

/** A staged attachment awaiting review + send. */
export type ComposerAttachment = UploadAttachment | HostAttachment;

interface AttachmentPreviewTrayProps {
  attachments: ComposerAttachment[];
  onRemove: (id: string) => void;
  removeAriaLabel: string;
  viewAriaLabel: (name: string) => string;
  previewTitle: string;
  closePreviewLabel: string;
  /** Names the badge on attachments that come from the host by path. */
  hostBadgeLabel: string;
}

function IconTile({ icon, name }: { icon: React.JSX.Element; name: string }) {
  return (
    <span className="flex h-full w-full flex-col items-center justify-center gap-1 bg-wc-surface-input p-1 text-wc-text-secondary">
      {icon}
      <span className="line-clamp-2 break-all text-[10px] leading-tight">{name}</span>
    </span>
  );
}

// Host files are never fetched into the tray: the point of attaching by path is
// that nothing moves, so they show an icon tile even when they are images.
function Thumbnail({ attachment }: { attachment: ComposerAttachment }) {
  if (attachment.source === "host") {
    return <IconTile name={attachment.name} icon={attachment.isDirectory ? <Folder className="h-6 w-6 text-wc-accent" aria-hidden /> : <FileText className="h-6 w-6" aria-hidden />} />;
  }
  switch (attachment.kind) {
    case "image":
      return <img src={attachment.previewUrl} alt={attachment.name} className="h-full w-full object-cover" />;
    case "video":
      return (
        <span className="relative block h-full w-full">
          <video src={attachment.previewUrl} muted playsInline preload="metadata" className="h-full w-full object-cover" />
          <span className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/25">
            <Play className="h-5 w-5 text-white" aria-hidden />
          </span>
        </span>
      );
    case "audio":
      return <IconTile name={attachment.name} icon={<Music className="h-6 w-6" aria-hidden />} />;
    default:
      return <IconTile name={attachment.name} icon={<FileText className="h-6 w-6" aria-hidden />} />;
  }
}

function FullPreview({ attachment }: { attachment: ComposerAttachment }) {
  const { t } = useTranslation();
  if (attachment.source === "host") {
    return (
      <div className="flex w-full max-w-md flex-col items-center gap-2 text-center">
        {attachment.isDirectory ? <Folder className="h-10 w-10 text-wc-accent" aria-hidden /> : <FileText className="h-10 w-10 text-wc-text-secondary" aria-hidden />}
        <p className="break-all text-sm text-wc-text-primary">{attachment.name}</p>
        <p className="break-all font-mono text-xs text-wc-text-muted" data-testid="composer-attachment-host-path">{attachment.path}</p>
        <p className="text-xs text-wc-text-muted">{t(strings.composer.hostAttachmentNote)}</p>
      </div>
    );
  }
  switch (attachment.kind) {
    case "image":
      return (
        <img
          src={attachment.previewUrl}
          alt={attachment.name}
          className="max-h-[min(70vh,36rem)] max-w-full object-contain"
        />
      );
    case "video":
      return (
        <video
          src={attachment.previewUrl}
          controls
          autoPlay
          playsInline
          className="max-h-[min(70vh,36rem)] max-w-full rounded-lg bg-black"
        />
      );
    case "audio":
      return (
        <div className="flex w-full max-w-md flex-col items-center gap-3">
          <Music className="h-10 w-10 text-wc-text-secondary" aria-hidden />
          <p className="break-all text-center text-sm text-wc-text-primary">{attachment.name}</p>
          <audio src={attachment.previewUrl} controls className="w-full" />
        </div>
      );
    default:
      return (
        <div className="flex w-full max-w-md flex-col items-center gap-2 text-center">
          <FileText className="h-10 w-10 text-wc-text-secondary" aria-hidden />
          <p className="break-all text-sm text-wc-text-primary">{attachment.name}</p>
          <p className="text-xs text-wc-text-muted">{formatFileSize(attachment.sizeBytes)}</p>
          <p className="text-xs text-wc-text-muted">{t(strings.composer.attachmentPreviewNote)}</p>
        </div>
      );
  }
}

/**
 * Horizontal tray of reviewable attachment previews shown inside the composer.
 * Attachments are NEVER submitted until send — this tray only stages them.
 * Images/video render a thumbnail; audio and other files render an icon tile.
 */
export function AttachmentPreviewTray({
  attachments,
  onRemove,
  removeAriaLabel,
  viewAriaLabel,
  previewTitle,
  closePreviewLabel,
  hostBadgeLabel,
}: AttachmentPreviewTrayProps) {
  const [selected, setSelected] = useState<ComposerAttachment | null>(null);
  if (attachments.length === 0) return null;

  return (
    <>
      <style>{`[data-rcl-responsive-dialog].composer-attachment-preview-layer { z-index: var(--layer-menu, 610); }`}</style>
      <div data-testid="composer-attachment-tray" className="mb-2 flex gap-2 overflow-x-auto py-1">
        {attachments.map((att) => (
          <div key={att.id} className="group relative shrink-0">
            <button
              type="button"
              data-testid={`composer-attachment-preview-${att.id}`}
              aria-label={viewAriaLabel(att.name)}
              onClick={() => { setSelected(att); }}
              className="block h-20 w-20 overflow-hidden rounded-lg border border-wc-default bg-wc-surface-input focus:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent"
            >
              <Thumbnail attachment={att} />
            </button>
            {att.source === "host" && (
              <span
                data-testid={`composer-attachment-host-badge-${att.id}`}
                title={hostBadgeLabel}
                aria-label={hostBadgeLabel}
                className="pointer-events-none absolute bottom-1 start-1 flex items-center gap-0.5 rounded bg-wc-accent/90 px-1 py-0.5 text-[9px] font-medium text-white"
              >
                <Monitor className="h-2.5 w-2.5" aria-hidden />
              </span>
            )}
            {att.status === "uploading" && (
              <div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-lg bg-wc-backdrop">
                <Loader2 className="h-4 w-4 animate-spin text-wc-text-primary" />
              </div>
            )}
            <button
              type="button"
              data-testid={`composer-attachment-remove-${att.id}`}
              onClick={() => { onRemove(att.id); }}
              className="absolute -end-2 -top-2 rounded-full border border-wc-default bg-wc-surface-raised p-1 text-wc-text-secondary transition hover:border-red-500/50 hover:text-red-400"
              aria-label={removeAriaLabel}
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        ))}
      </div>
      {selected ? (
        <ResponsiveDialog
          open
          onClose={() => { setSelected(null); }}
          title={previewTitle}
          closeLabel={closePreviewLabel}
          testId="composer-attachment-preview"
          className="composer-attachment-preview-layer"
          contentPadding="none"
          avoidKeyboard
        >
          <div className="flex min-h-[min(70vh,36rem)] items-center justify-center bg-black/20 p-3">
            <FullPreview attachment={selected} />
          </div>
        </ResponsiveDialog>
      ) : null}
    </>
  );
}
