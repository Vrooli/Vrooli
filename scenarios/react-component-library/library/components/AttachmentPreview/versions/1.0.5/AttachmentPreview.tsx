/**
 * @libraryId react-component-library:AttachmentPreview
 * @displayName AttachmentPreview
 * @description The message- or form-oriented attachment with upload progress, retry, cancellation, scanning state, preview, removal, and accessible status for each file independently.
 * @version 1.0.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:AttachmentPreview
 * @vrooliComponentSourceSlot media.attachment-preview */
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
  [data-rcl-attachment-preview] { display: grid; gap: var(--space-2xs); min-inline-size: 0; color: var(--color-foreground); }
  [data-rcl-attachment-preview-file] { min-inline-size: 0; }
  [data-rcl-attachment-preview-progress] { display: grid; gap: var(--space-3xs); padding-inline: var(--space-sm); }
  [data-rcl-attachment-preview-progress-label] { display: flex; justify-content: space-between; gap: var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
  [data-rcl-attachment-preview-status] { display: flex; flex-wrap: wrap; align-items: flex-start; gap: var(--space-2xs); padding: var(--space-xs) var(--space-sm); border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-muted-foreground); font: var(--text-body); }
  [data-rcl-attachment-preview-status][data-tone="error"] { border-color: color-mix(in srgb, var(--color-danger) 42%, var(--color-border)); background: color-mix(in srgb, var(--color-danger) 7%, var(--color-surface)); color: var(--color-danger); }
  [data-rcl-attachment-preview-status][data-tone="offline"] { border-color: color-mix(in srgb, var(--color-warning) 42%, var(--color-border)); background: color-mix(in srgb, var(--color-warning) 8%, var(--color-surface)); color: var(--color-foreground); }
  [data-rcl-attachment-preview-status-copy] { min-inline-size: 0; flex: 1 1 auto; }
  [data-rcl-attachment-preview-status] > [data-rcl-attachment-preview-actions] { flex: 1 1 100%; }
  [data-rcl-attachment-preview-actions] { display: flex; flex-wrap: wrap; gap: var(--space-2xs); margin-block-start: var(--space-2xs); }
  [data-rcl-attachment-preview-action] { min-block-size: var(--tap-target-min); padding-inline: var(--space-sm); border: 1px solid currentColor; border-radius: var(--radius-control); background: transparent; color: inherit; font: var(--text-label); cursor: pointer; }
  [data-rcl-attachment-preview-action][data-primary="true"] { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-primary-foreground); }
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
      <StyleSheet
        libraryId="react-component-library:AttachmentPreview"
        version="1.0.4"
        css={styles}
      />
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
