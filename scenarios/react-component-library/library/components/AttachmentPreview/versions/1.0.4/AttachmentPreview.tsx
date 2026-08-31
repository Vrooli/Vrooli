/**
 * @libraryId react-component-library:AttachmentPreview
 * @displayName AttachmentPreview
 * @description An adapter-driven attachment surface with independent upload, scanning, retry, cancellation, preview, removal, and offline states.
 * @version 1.0.4
 * @tags ["media","files","upload","progress","recovery","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource media.attachment-preview */
import { type CSSProperties, type ReactNode } from "react";
import { FilePreview, type FilePreviewStatus } from "@vrooli/react-component-library/FilePreview/1";
import { Progress } from "@vrooli/react-component-library/Progress/1";

export type AttachmentStatus =
  | "queued"
  | "uploading"
  | "scanning"
  | "success"
  | "error"
  | "offline";

export interface AttachmentPreviewProps {
  name: string;
  mimeType?: string;
  sizeBytes?: number;
  thumbnailUrl?: string;
  thumbnailAlt?: string;
  status?: AttachmentStatus;
  progress?: number;
  statusMessage?: ReactNode;
  errorMessage?: ReactNode;
  onOpen?: () => void;
  onDownload?: () => void;
  onRemove?: () => void;
  onRetry?: () => void;
  onCancel?: () => void;
  openLabel?: string;
  downloadLabel?: string;
  removeLabel?: string;
  retryLabel?: string;
  cancelLabel?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
  [data-rcl-attachment-preview] { display: grid; gap: var(--space-2xs, 8px); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
  [data-rcl-attachment-preview-file] { min-inline-size: 0; }
  [data-rcl-attachment-preview-progress] { display: grid; gap: var(--space-3xs, 4px); padding-inline: var(--space-sm, 16px); }
  [data-rcl-attachment-preview-progress-label] { display: flex; justify-content: space-between; gap: var(--space-sm, 16px); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
  [data-rcl-attachment-preview-status] { display: flex; flex-wrap: wrap; align-items: flex-start; gap: var(--space-2xs, 8px); padding: var(--space-xs, 12px) var(--space-sm, 16px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
  [data-rcl-attachment-preview-status][data-tone="error"] { border-color: color-mix(in srgb, var(--color-danger, #dc2626) 42%, var(--color-border, #cbd5e1)); background: color-mix(in srgb, var(--color-danger, #dc2626) 7%, var(--color-surface, #ffffff)); color: var(--color-danger, #dc2626); }
  [data-rcl-attachment-preview-status][data-tone="offline"] { border-color: color-mix(in srgb, var(--color-warning, #d97706) 42%, var(--color-border, #cbd5e1)); background: color-mix(in srgb, var(--color-warning, #d97706) 8%, var(--color-surface, #ffffff)); color: var(--color-foreground, #0f172a); }
  [data-rcl-attachment-preview-status-copy] { min-inline-size: 0; flex: 1 1 auto; }
  [data-rcl-attachment-preview-status] > [data-rcl-attachment-preview-actions] { flex: 1 1 100%; }
  [data-rcl-attachment-preview-actions] { display: flex; flex-wrap: wrap; gap: var(--space-2xs, 8px); margin-block-start: var(--space-2xs, 8px); }
  [data-rcl-attachment-preview-action] { min-block-size: var(--tap-target-min, 44px); padding-inline: var(--space-sm, 16px); border: 1px solid currentColor; border-radius: var(--radius-control, 0.375rem); background: transparent; color: inherit; font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
  [data-rcl-attachment-preview-action][data-primary="true"] { border-color: var(--color-primary, #2563eb); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #ffffff); }
  @media (max-width: 34rem) { [data-rcl-attachment-preview-progress] { padding-inline: 0; } [data-rcl-attachment-preview-actions] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-attachment-preview-action] { inline-size: 100%; } }
`;

function fileStatus(status: AttachmentStatus): FilePreviewStatus {
  if (status === "success") return "success";
  if (status === "error" || status === "offline") return "error";
  return "loading";
}

function defaultStatus(status: AttachmentStatus) {
  return {
    queued: "Waiting to upload",
    uploading: "Uploading securely",
    scanning: "Scanning for a safe preview",
    success: "Ready to view",
    error: "Upload failed",
    offline: "Waiting for a connection",
  }[status];
}

export const AttachmentPreview = withClassName(function AttachmentPreview({
  name,
  mimeType,
  sizeBytes,
  thumbnailUrl,
  thumbnailAlt,
  status = "success",
  progress,
  statusMessage,
  errorMessage = "We could not finish this upload.",
  onOpen,
  onDownload,
  onRemove,
  onRetry,
  onCancel,
  openLabel = "Open",
  downloadLabel = "Download",
  removeLabel = "Remove",
  retryLabel = "Retry upload",
  cancelLabel = "Cancel",
  className,
  style,
}: AttachmentPreviewProps) {
  const active = status === "queued" || status === "uploading" || status === "scanning";
  const error = status === "error" || status === "offline";
  const hasProgress = status === "uploading" || status === "scanning";
  const boundedProgress =
    typeof progress === "number" && Number.isFinite(progress)
      ? Math.max(0, Math.min(100, progress))
      : undefined;
  return (
    <section
      className={className}
      style={style}
      data-rcl-attachment-preview
      data-status={status}
      aria-label={`${name} attachment`}
    >
      <StyleSheet name="attachmentpreview-1-0-2-1" css={styles} />
      <div data-rcl-attachment-preview-file>
        <FilePreview
          name={name}
          mimeType={mimeType}
          sizeBytes={sizeBytes}
          thumbnailUrl={thumbnailUrl}
          thumbnailAlt={thumbnailAlt}
          status={fileStatus(status)}
          statusMessage={typeof statusMessage === "string" ? statusMessage : defaultStatus(status)}
          onOpen={onOpen}
          onDownload={onDownload}
          onRemove={onRemove}
          openLabel={openLabel}
          downloadLabel={downloadLabel}
          removeLabel={removeLabel}
        />
      </div>
      {hasProgress && (
        <div data-rcl-attachment-preview-progress>
          <div data-rcl-attachment-preview-progress-label>
            <span>{status === "scanning" ? "Safety check" : "Upload progress"}</span>
            <span>
              {boundedProgress === undefined ? "Working…" : `${Math.round(boundedProgress)}%`}
            </span>
          </div>
          <Progress
            value={boundedProgress}
            mode={boundedProgress === undefined ? "indeterminate" : "determinate"}
            label={`${name} ${status} progress`}
            showValue={false}
          />
        </div>
      )}
      {error && (
        <div
          data-rcl-attachment-preview-status
          data-tone={status}
          role={status === "error" ? "alert" : "status"}
          aria-live="polite"
        >
          <span data-rcl-attachment-preview-status-copy>
            {status === "offline"
              ? "You are offline. The attachment will resume when connection returns."
              : errorMessage}
          </span>
          <span data-rcl-attachment-preview-actions>
            {onRetry && (
              <button
                data-testid="media.attachment-preview"
                type="button"
                data-rcl-attachment-preview-action
                data-primary="true"
                onClick={onRetry}
              >
                {retryLabel}
              </button>
            )}
            {onCancel && (
              <button
                data-testid="media.attachment-preview"
                type="button"
                data-rcl-attachment-preview-action
                onClick={onCancel}
              >
                {cancelLabel}
              </button>
            )}
          </span>
        </div>
      )}
      {active && onCancel && (
        <div data-rcl-attachment-preview-actions>
          <button
            data-testid="media.attachment-preview"
            type="button"
            data-rcl-attachment-preview-action
            onClick={onCancel}
          >
            {cancelLabel}
          </button>
        </div>
      )}
    </section>
  );
});
