/**
 * @libraryId react-component-library:EdgeFade
 * @displayName EdgeFade
 * @description
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.edge-fade */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import type { HTMLAttributes } from "react";
import "./EdgeFade.css";

export const EdgeFade = withClassName(function EdgeFade({
  side = "inline-end",
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & { side?: "inline-start" | "inline-end" }) {
  return (
    <div
      data-testid="primitives.edge-fade"
      aria-hidden
      data-rcl-edge-fade
      data-side={side}
      style={style}
      {...props}
    />
  );
});
