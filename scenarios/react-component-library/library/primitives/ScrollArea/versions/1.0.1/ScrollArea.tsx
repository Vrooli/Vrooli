/**
 * @libraryId react-component-library:ScrollArea
 * @displayName ScrollArea
 * @description ScrollArea provides a keyboard-reachable overflow boundary.
 * @version 1.0.1
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.scroll-area */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { forwardRef, type HTMLAttributes } from "react";

export const ScrollArea = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={className}
      role="region"
      aria-label={translate("primitives.scroll-area.aria-label.1", "Scrollable content")}
      style={{
        overflow: "auto",
        WebkitOverflowScrolling: "touch",
        ...props.style,
      }}
      {...props}
    />
  ),
);
