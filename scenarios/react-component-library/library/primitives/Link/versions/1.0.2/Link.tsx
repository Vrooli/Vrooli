/**
 * @libraryId react-component-library:Link
 * @displayName Link
 * @version 1.0.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.link */
import { forwardRef, type AnchorHTMLAttributes } from "react";
import "./Link.css";

export const Link = forwardRef<HTMLAnchorElement, AnchorHTMLAttributes<HTMLAnchorElement>>(
  ({ children, style, ...props }, ref) => (
    <a
      data-testid="primitives.link"
      ref={ref}
      data-link="true"
      data-rcl-link
      style={style}
      {...props}
    >
      {children}
    </a>
  ),
);
