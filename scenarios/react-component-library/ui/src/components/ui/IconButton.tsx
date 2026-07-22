/**
 * @vrooliComponentSource react-component-library:IconButton
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 8a3af1ec-0e94-4b6a-ae8d-2bb53f995f27
 * @vrooliComponentAppliedAt 2026-07-21T03:41:03Z
 * @vrooliComponentSourceSha256 db2c11d9969084739e9c4a6c42d6ef6ab79aed10c42d2a0ceef12359a0f56f4d
 * @vrooliComponentDriftHash db2c11d9969084739e9c4a6c42d6ef6ab79aed10c42d2a0ceef12359a0f56f4d
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
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
