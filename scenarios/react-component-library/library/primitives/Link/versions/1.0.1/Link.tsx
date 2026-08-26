/**
 * @libraryId react-component-library:Link
 * @displayName Link
 * @description Link is the accessible navigation primitive with a stable semantic hook.
 * @version 1.0.1
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.link */
import { forwardRef, type AnchorHTMLAttributes } from "react";

export const Link = forwardRef<HTMLAnchorElement, AnchorHTMLAttributes<HTMLAnchorElement>>(
  ({ children, style, ...props }, ref) => (
    <a
      data-testid="primitives.link"
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
  ),
);
