import type { ReactNode } from "react";
import { cn } from "../lib/utils";

export type PageShellProps = {
  children: ReactNode;
  className?: string;
};

export function PageShell({ children, className }: PageShellProps) {
  return <main className={cn("p-6 ko-stack", className)}>{children}</main>;
}
