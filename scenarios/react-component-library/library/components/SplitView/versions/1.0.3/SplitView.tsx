/**
 * @libraryId react-component-library:SplitView
 * @displayName SplitView
 * @version 1.0.3
 * @tags ["navigation","layout","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.split-view */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { CSSProperties, ReactNode } from "react";

const styles = `
[data-rcl-split-view] { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: var(--space-md, 24px); align-items: start; min-inline-size: 0; }
[data-rcl-split-view-region] { min-inline-size: 0; }
@media (max-width: 48rem) { [data-rcl-split-view] { grid-template-columns: 1fr; } }

`;

export const SplitView = withClassName(function SplitView({
  primary,
  secondary,
  primaryLabel = "Primary content",
  secondaryLabel = "Secondary content",
  className,
  style,
}: {
  primary?: ReactNode;
  secondary?: ReactNode;
  primaryLabel?: string;
  secondaryLabel?: string;
  className?: string;
  style?: CSSProperties;
}) {
  return (
    <div
      data-testid="navigation.split-view"
      data-rcl-split-view
      data-split-view
      className={className}
      style={style}
    >
      <StyleSheet name="splitview-1-0-1-1" css={styles} />
      <section data-rcl-split-view-region aria-label={primaryLabel}>
        {primary}
      </section>
      <section data-rcl-split-view-region aria-label={secondaryLabel}>
        {secondary}
      </section>
    </div>
  );
});
