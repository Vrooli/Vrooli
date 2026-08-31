/**
 * @libraryId react-component-library:LayoutGroup
 * @displayName LayoutGroup
 * @description
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.layout-group */
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
