/** @vrooliComponentSource data-display.diff-viewer */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import type { CSSProperties } from "react";

const styles = `
[data-rcl-diff-viewer] { display: grid; gap: var(--space-2xs, .35rem); min-inline-size: 0; }
[data-rcl-diff-viewer] figcaption { color: var(--color-muted-foreground, #64748b); font: var(--text-overline, 700 .68rem/1.1 system-ui, sans-serif); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-diff-viewer-row] { display: grid; grid-template-columns: minmax(6rem, 10rem) minmax(0, 1fr); gap: var(--space-sm, .75rem); align-items: start; min-inline-size: 0; padding: var(--space-xs, .625rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-inline-start: 3px solid var(--color-border-strong, #94a3b8); border-radius: var(--radius-control, .625rem); background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-diff-viewer-row="removed"] { border-inline-start-color: var(--color-danger, #dc2626); background: color-mix(in srgb, var(--color-danger, #dc2626) 6%, var(--color-surface-raised, #fff)); }
[data-rcl-diff-viewer-row="added"] { border-inline-start-color: var(--color-success, #16803c); background: color-mix(in srgb, var(--color-success, #16803c) 6%, var(--color-surface-raised, #fff)); }
[data-rcl-diff-viewer-label] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-diff-viewer-value] { min-inline-size: 0; overflow-wrap: anywhere; color: var(--color-foreground, #0f172a); font: var(--text-body, 400 .9rem/1.45 system-ui, sans-serif); }
[data-rcl-diff-viewer-value] del, [data-rcl-diff-viewer-value] ins { text-decoration-thickness: .12em; text-underline-offset: .16em; }
@media (max-width: 38rem) { [data-rcl-diff-viewer-row] { grid-template-columns: 1fr; gap: var(--space-3xs, .2rem); } }

`;

export function DiffViewer({
  before = "",
  after = "",
  className,
  style,
}: {
  before?: string;
  after?: string;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <figure
      data-rcl-diff-viewer
      aria-label="Diff"
      className={className}
      style={style}
    >
      <StyleSheet name="diffviewer-1-0-0-1" css={styles} />
      <figcaption>Version comparison</figcaption>
      <div data-rcl-diff-viewer-row="removed">
        <span data-rcl-diff-viewer-label>Previous version</span>
        <span data-rcl-diff-viewer-value>
          <del>{before || "No value"}</del>
        </span>
      </div>
      <div data-rcl-diff-viewer-row="added">
        <span data-rcl-diff-viewer-label>Current version</span>
        <span data-rcl-diff-viewer-value>
          <ins>{after || "No value"}</ins>
        </span>
      </div>
    </figure>
  );
}
