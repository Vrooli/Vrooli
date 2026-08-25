/** @vrooliComponentSource primitives.link */
import { forwardRef, type AnchorHTMLAttributes } from "react";

export const Link = forwardRef<
  HTMLAnchorElement,
  AnchorHTMLAttributes<HTMLAnchorElement>
>(({ children, style, ...props }, ref) => (
  <a data-testid="primitives.link"
    ref={ref}
    data-link="true"
    style={{
      color: "var(--color-primary)",
      textUnderlineOffset: "var(--space-3xs)",
      transition: "color var(--dur-quick) var(--ease-standard)",
      ...style,
    }}
    {...props}
  >
    {children}
  </a>
));
