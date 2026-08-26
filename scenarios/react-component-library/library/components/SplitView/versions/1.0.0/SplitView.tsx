/** @vrooliComponentSource navigation.split-view */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { CSSProperties, ReactNode } from "react";

const styles = `
[data-rcl-split-view] { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: var(--space-md, 1rem); align-items: start; min-inline-size: 0; }
[data-rcl-split-view-region] { min-inline-size: 0; }
@media (max-width: 48rem) { [data-rcl-split-view] { grid-template-columns: 1fr; } }
@media (forced-colors: active) { [data-rcl-split-view-region] { forced-color-adjust: auto; } }
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
    <div data-testid="navigation.split-view"
      data-rcl-split-view
      data-split-view
      className={className}
      style={style}
    >
      <style
        data-rcl-split-view-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <section data-rcl-split-view-region aria-label={primaryLabel}>
        {primary}
      </section>
      <section data-rcl-split-view-region aria-label={secondaryLabel}>
        {secondary}
      </section>
    </div>
  );
});
