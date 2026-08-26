/** @vrooliComponentSource primitives.edge-fade */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { HTMLAttributes } from "react";
import "./EdgeFade.css";

export const EdgeFade = withClassName(function EdgeFade({
  side = "inline-end",
  style,
  ...props
}: HTMLAttributes<HTMLDivElement> & { side?: "inline-start" | "inline-end" }) {
  return (
    <div data-testid="primitives.edge-fade"
      aria-hidden
      data-rcl-edge-fade
      data-side={side}
      style={style}
      {...props}
    />
  );
});
