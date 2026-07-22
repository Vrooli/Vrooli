/**
 * @libraryId react-component-library:IconButton
 * @displayName IconButton
 * @description Accessible icon-only action control.
 * @version 1.0.0
 * @tags ["button","icon","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  "aria-label": string;
  children: ReactNode;
}

/** A compact accessible action; its label is required because the icon is decorative. */
export function IconButton({ children, className = "", type = "button", ...props }: IconButtonProps) {
  return <button type={type} title={props.title ?? props["aria-label"]} className={`inline-flex min-h-11 min-w-11 items-center justify-center rounded-control text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-60 ${className}`} {...props}>{children}</button>;
}
