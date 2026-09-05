/**
 * @libraryId react-component-library:Separator
 * @displayName Separator
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.separator */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes } from "react";
import "./Separator.css";

export const Separator = withClassName(function Separator({
  orientation = "horizontal",
  style,
  ...props
}: HTMLAttributes<HTMLHRElement> & {
  orientation?: "horizontal" | "vertical";
}) {
  return (
    <hr
      data-testid="primitives.separator"
      aria-orientation={orientation}
      data-orientation={orientation}
      data-rcl-separator
      style={style}
      {...props}
    />
  );
});
