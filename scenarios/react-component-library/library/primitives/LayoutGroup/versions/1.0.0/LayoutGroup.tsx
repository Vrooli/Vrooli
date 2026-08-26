/** @vrooliComponentSource primitives.layout-group */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

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
