/**
 * @libraryId react-component-library:Separator
 * @displayName Separator
 * @description The semantic divider supporting orientation, inset, and both decorative and grouping roles, so structure can be expressed without adding containers.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Separator
 * @vrooliComponentSourceSlot primitives.separator */
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
