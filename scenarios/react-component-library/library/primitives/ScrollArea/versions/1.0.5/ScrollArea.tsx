/**
 * @libraryId react-component-library:ScrollArea
 * @displayName ScrollArea
 * @description
 * @version 1.0.5
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.scroll-area */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { forwardRef, type HTMLAttributes } from "react";
import "./ScrollArea.css";

export const ScrollArea = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, style, ...props }, ref) => {
    const libraryStrings = useStrings();
    return (
      <div
        data-testid="primitives.scroll-area"
        ref={ref}
        className={className}
        role="region"
        aria-label={libraryStrings(
          "primitives.scroll-area.scrollable-content",
          "Scrollable content",
        )}
        data-rcl-scroll-area
        style={style}
        {...props}
      />
    );
  },
);
