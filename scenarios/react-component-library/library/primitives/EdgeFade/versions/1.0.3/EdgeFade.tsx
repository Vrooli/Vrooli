/**
 * @libraryId react-component-library:EdgeFade
 * @displayName EdgeFade
 * @description The pointer-transparent gradient mask indicating clipped or scrollable content without intercepting interaction beneath it.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:EdgeFade
 * @vrooliComponentSourceSlot primitives.edge-fade */
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
