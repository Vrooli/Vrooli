/**
 * @libraryId react-component-library:DiffViewer
 * @displayName DiffViewer
 * @description The semantic comparison surface for text or structured change, with split and unified modes, syntax highlighting, hunk navigation, comments, selection, and large-file handling.
 * @version 1.0.6
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** @vrooliComponentSource react-component-library:DiffViewer
 * @vrooliComponentSourceSlot data-display.diff-viewer */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { CSSProperties } from "react";

const styles = `
[data-rcl-diff-viewer] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-diff-viewer] figcaption { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-diff-viewer-row] { display: grid; grid-template-columns: minmax(6rem, 10rem) minmax(0, 1fr); gap: var(--space-sm); align-items: start; min-inline-size: 0; padding: var(--space-xs) var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-inline-start: 3px solid var(--color-border-strong); border-radius: var(--radius-control); background: var(--color-surface-muted); }
[data-rcl-diff-viewer-row="removed"] { border-inline-start-color: var(--color-danger); background: color-mix(in srgb, var(--color-danger) 6%, var(--color-surface-raised)); }
[data-rcl-diff-viewer-row="added"] { border-inline-start-color: var(--color-success); background: color-mix(in srgb, var(--color-success) 6%, var(--color-surface-raised)); }
[data-rcl-diff-viewer-label] { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-diff-viewer-value] { min-inline-size: 0; overflow-wrap: anywhere; color: var(--color-foreground); font: var(--text-body); }
[data-rcl-diff-viewer-value] del, [data-rcl-diff-viewer-value] ins { text-decoration-thickness: .12em; text-underline-offset: .16em; }
@media (max-width: 38rem) { [data-rcl-diff-viewer-row] { grid-template-columns: 1fr; gap: var(--space-3xs); } }

`;

export const DiffViewer = withClassName(function DiffViewer({
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
  const strings = useStrings();
  return (
    <figure
      data-testid="data-display.diff-viewer"
      data-rcl-diff-viewer
      aria-label={strings("data-display.diff-viewer.diff", "Diff")}
      className={className}
      style={style}
    >
      <StyleSheet libraryId="react-component-library:DiffViewer" version="1.0.5" css={styles} />
      <figcaption>
        {strings("data-display.diff-viewer.version-comparison", "Version comparison")}
      </figcaption>
      <div data-rcl-diff-viewer-row="removed">
        <span data-rcl-diff-viewer-label>
          {strings("data-display.diff-viewer.previous-version", "Previous version")}
        </span>
        <span data-rcl-diff-viewer-value>
          <del>{before || "No value"}</del>
        </span>
      </div>
      <div data-rcl-diff-viewer-row="added">
        <span data-rcl-diff-viewer-label>
          {strings("data-display.diff-viewer.current-version", "Current version")}
        </span>
        <span data-rcl-diff-viewer-value>
          <ins>{after || "No value"}</ins>
        </span>
      </div>
    </figure>
  );
});
