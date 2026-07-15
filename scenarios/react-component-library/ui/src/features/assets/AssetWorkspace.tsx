import type { ReactNode } from "react";

/**
 * Capability-neutral asset workspace frame. Renderable assets place preview
 * tooling inside it; non-renderable assets use the same responsive shell with
 * Files and Details only.
 */
export function AssetWorkspace({ children, label, testId, className = "" }: { children: ReactNode; label: string; testId: string; className?: string }) {
  return <section data-testid={testId} aria-label={label} className={`relative flex min-h-0 flex-1 flex-col overflow-hidden bg-app-background ${className}`}>{children}</section>;
}
