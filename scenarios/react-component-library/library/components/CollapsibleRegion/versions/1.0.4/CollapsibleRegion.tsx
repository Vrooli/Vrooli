/**
 * @libraryId react-component-library:CollapsibleRegion
 * @displayName CollapsibleRegion
 * @description A disclosure region that preserves accessible semantics while content enters and exits without clipping artifacts.
 * @version 1.0.4
 * @tags ["motion","layout","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CollapsibleRegion */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { ReactNode } from "react";
export const CollapsibleRegion = withClassName(function CollapsibleRegion({
  open = true,
  children,
}: {
  open?: boolean;
  children?: ReactNode;
}) {
  return (
    <div
      data-testid="motion.collapsible-region"
      data-collapsible-region
      data-open={open}
      aria-hidden={!open || undefined}
      style={{
        overflow: "hidden",
        opacity: open ? 1 : 0,
        transition: "opacity var(--dur-moderate, 280ms) ease",
      }}
    >
      {open ? children : null}
    </div>
  );
});
