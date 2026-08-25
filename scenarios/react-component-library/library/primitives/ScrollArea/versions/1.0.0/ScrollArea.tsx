/** @vrooliComponentSource primitives.scroll-area */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import { forwardRef, type HTMLAttributes } from "react";

export const ScrollArea = forwardRef<
  HTMLDivElement,
  HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
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
));
