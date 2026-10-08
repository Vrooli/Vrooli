/**
 * @libraryId react-component-library:Link
 * @displayName Link
 * @description The router-adaptable link primitive handling external-link treatment, active and pending state, prefetch policy, focus behavior, and disabled semantics.
 * @version 1.0.3
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Link
 * @vrooliComponentSourceSlot primitives.link */
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
