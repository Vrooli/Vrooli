/**
 * @libraryId react-component-library:FilePreview
 * @displayName FilePreview
 * @description A safe, typed file summary with status, metadata, preview fallback, and keyboard-operable actions.
 * @version 1.0.3
 * @tags ["media","files","status","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.2";

/** @vrooliComponentSource react-component-library:FilePreview */
import { useId, type CSSProperties } from "react";

export type FilePreviewStatus = "loading" | "success" | "error";

export interface FilePreviewProps {
  name: string;
  mimeType?: string;
  sizeBytes?: number;
  status?: FilePreviewStatus;
  statusMessage?: string;
  thumbnailUrl?: string;
  thumbnailAlt?: string;
  onOpen?: () => void;
  onDownload?: () => void;
  onRemove?: () => void;
  openLabel?: string;
  downloadLabel?: string;
  removeLabel?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-file-preview] { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: var(--space-sm, 16px); width: 100%; box-sizing: border-box; padding: var(--space-sm, 16px); border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-flat, none); }
[data-rcl-file-preview][data-status="error"] { border-color: var(--color-danger-border, color-mix(in srgb, var(--color-danger) 38%, var(--color-border))); }
[data-rcl-file-preview-thumb] { display: grid; place-items: center; inline-size: 3.25rem; block-size: 3.25rem; overflow: hidden; border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); color: var(--color-primary, #2563eb); }
[data-rcl-file-preview-thumb] img { width: 100%; height: 100%; object-fit: cover; }
[data-rcl-file-preview-glyph] { display: grid; place-items: center; inline-size: 2.1rem; block-size: 2.1rem; border: 1px solid currentColor; border-radius: .45rem; font-size: 10px; font-weight: 900; letter-spacing: .04em; }
[data-rcl-file-preview-content] { min-width: 0; }
[data-rcl-file-preview-name] { display: block; overflow: hidden; color: inherit; font-size: var(--font-size-sm, 14px); font-weight: 750; line-height: 1.35; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-file-preview-meta] { display: flex; flex-wrap: wrap; gap: 4px 10px; margin-top: 4px; color: var(--color-muted-foreground, #64748b); font-size: 12px; line-height: 1.4; }
[data-rcl-file-preview-status] { display: inline-flex; align-items: center; gap: 5px; margin-top: 7px; color: var(--color-muted-foreground, #64748b); font-size: 12px; }
[data-rcl-file-preview-status]::before { content: ""; inline-size: 7px; block-size: 7px; border-radius: 50%; background: currentColor; }
[data-rcl-file-preview][data-status="success"] [data-rcl-file-preview-status] { color: var(--color-success-foreground, color-mix(in srgb, var(--color-success) 76%, var(--color-foreground))); }
[data-rcl-file-preview][data-status="error"] [data-rcl-file-preview-status] { color: var(--color-danger-foreground, color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))); }
[data-rcl-file-preview][data-status="loading"] [data-rcl-file-preview-status]::before { animation: rcl-file-preview-pulse 1.2s ease-in-out infinite; }
[data-rcl-file-preview-actions] { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2xs, 8px); }
[data-rcl-file-preview-actions] button { min-block-size: 2.75rem; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); padding: 0 var(--space-xs, 12px); background: transparent; color: inherit; font: inherit; font-size: 12px; font-weight: 750; cursor: pointer; }
[data-rcl-file-preview-actions] button:hover { background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-file-preview-actions] button[data-primary="true"] { border-color: var(--color-primary, #2563eb); background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #ffffff); }
[data-rcl-file-preview-actions] button[data-danger="true"] { border-color: var(--color-danger-border, color-mix(in srgb, var(--color-danger) 38%, var(--color-border))); color: var(--color-danger-foreground, color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))); }
@keyframes rcl-file-preview-pulse { 50% { opacity: .35; } }
@media (max-width: 520px) { [data-rcl-file-preview] { grid-template-columns: auto minmax(0, 1fr); align-items: start; } [data-rcl-file-preview-actions] { grid-column: 1 / -1; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); width: 100%; } [data-rcl-file-preview-actions] button { width: 100%; } }

`;

function formatBytes(bytes?: number) {
  if (bytes === undefined || !Number.isFinite(bytes)) return "Size unavailable";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(1)} GB`;
}

function extension(name: string, mimeType?: string) {
  const suffix = name.split(".").pop()?.slice(0, 4).toUpperCase();
  if (suffix && suffix !== name.toUpperCase()) return suffix;
  return mimeType?.split("/").pop()?.slice(0, 4).toUpperCase() || "FILE";
}

export const FilePreview = withClassName(function FilePreview({
  name,
  mimeType,
  sizeBytes,
  status = "success",
  statusMessage,
  thumbnailUrl,
  thumbnailAlt,
  onOpen,
  onDownload,
  onRemove,
  openLabel = "Open",
  downloadLabel = "Download",
  removeLabel = "Remove",
  className,
  style,
}: FilePreviewProps) {
  const statusId = useId();
  const statusText =
    statusMessage ??
    {
      loading: "Preparing preview",
      success: "Ready to view",
      error: "Preview unavailable",
    }[status];
  return (
    <article
      data-rcl-file-preview
      data-status={status}
      className={className}
      style={style}
      aria-describedby={statusId}
    >
      <StyleSheet name="filepreview-1-0-2-1" css={styles} />
      <div data-rcl-file-preview-thumb aria-hidden={thumbnailUrl ? undefined : "true"}>
        {thumbnailUrl ? (
          <img src={thumbnailUrl} alt={thumbnailAlt ?? ""} />
        ) : (
          <span data-rcl-file-preview-glyph>{extension(name, mimeType)}</span>
        )}
      </div>
      <div data-rcl-file-preview-content>
        <span data-rcl-file-preview-name title={name}>
          {name}
        </span>
        <div data-rcl-file-preview-meta>
          <span>{mimeType ?? "Unknown type"}</span>
          <span>{formatBytes(sizeBytes)}</span>
        </div>
        <span
          id={statusId}
          data-rcl-file-preview-status
          role={status === "error" ? "alert" : "status"}
        >
          {statusText}
        </span>
      </div>
      <div data-rcl-file-preview-actions>
        {onOpen && (
          <button
            data-testid="media.file-preview"
            type="button"
            data-primary="true"
            onClick={onOpen}
          >
            {openLabel}
          </button>
        )}
        {onDownload && (
          <button data-testid="media.file-preview" type="button" onClick={onDownload}>
            {downloadLabel}
          </button>
        )}
        {onRemove && (
          <button
            data-testid="media.file-preview"
            type="button"
            data-danger="true"
            onClick={onRemove}
          >
            {removeLabel}
          </button>
        )}
      </div>
    </article>
  );
});
