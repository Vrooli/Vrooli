/**
 * @libraryId react-component-library:LayoutGroup
 * @displayName LayoutGroup
 * @description The coordination boundary animating related elements together as layout changes, preserving stable identities and avoiding measurement of unrelated subtrees.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:LayoutGroup
 * @vrooliComponentSourceSlot primitives.layout-group */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes } from "react";

export const LayoutGroup = withClassName(function LayoutGroup({
  children,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div data-testid="motion.layout-group" data-layout-group="true" {...props}>
      {children}
    </div>
  );
});
