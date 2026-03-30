/**
 * StatusBadge
 *
 * Inline status chip for detail pages. Uses the unified status-colors
 * mapping so status colors are consistent between graph nodes and detail views.
 */

import { cn } from "../../lib/utils";
import { getStatusColorClasses } from "../../surfaces/graph/lib/status-colors";

interface StatusBadgeProps {
  status: string;
  size?: "sm" | "md";
  className?: string;
}

export function StatusBadge({ status, size = "sm", className }: StatusBadgeProps) {
  const colors = getStatusColorClasses(status);

  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border font-medium",
        colors.background,
        colors.border,
        colors.text,
        size === "sm" && "px-2.5 py-0.5 text-xs",
        size === "md" && "px-3 py-1 text-sm",
        className,
      )}
      data-testid="status-badge"
    >
      {status.replace(/_/g, " ")}
    </span>
  );
}
