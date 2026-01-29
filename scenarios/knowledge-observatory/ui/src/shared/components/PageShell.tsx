// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { cn } from "../lib/utils";

export type PageShellVariant = "default" | "full-viewport";

export type PageShellProps = {
  children: ReactNode;
  className?: string;
  /** Layout variant - "full-viewport" removes padding and fills screen height minus header */
  variant?: PageShellVariant;
};

export function PageShell({ children, className, variant = "default" }: PageShellProps) {
  const baseClass = variant === "full-viewport" ? "ko-page-shell-full" : "p-6 ko-stack";
  return <main className={cn(baseClass, className)}>{children}</main>;
}
