import type { ReactNode } from "react";
import { Badge } from "../primitives/Badge";
import { Tooltip } from "../primitives/Tooltip";

export interface StaleBadgeProps {
  label: ReactNode;
  /** Tooltip content explaining *what* is stale. */
  reason?: ReactNode;
  testId?: string;
}

/**
 * Pre-composed `verdict-stale` badge with a tooltip explaining the stale
 * reason. Surfaces consume this rather than reaching for Badge + Tooltip
 * directly so the surface stays declarative.
 */
export function StaleBadge({ label, reason, testId }: StaleBadgeProps) {
  const badge = (
    <Badge data-testid={testId} variant="verdict-stale">
      {label}
    </Badge>
  );
  if (!reason) return badge;
  return <Tooltip content={reason}>{badge}</Tooltip>;
}
