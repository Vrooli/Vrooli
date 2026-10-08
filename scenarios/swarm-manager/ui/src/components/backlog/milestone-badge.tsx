/**
 * Milestone Badge
 *
 * Small pill showing which milestone a backlog item belongs to.
 */

import { memo } from "react";

interface MilestoneBadgeProps {
  milestone?: string;
}

export const MilestoneBadge = memo(function MilestoneBadge({ milestone }: MilestoneBadgeProps) {
  if (!milestone) return null;

  const label = milestone.length > 20 ? milestone.slice(0, 18) + "\u2026" : milestone;

  return (
    <span
      className="inline-flex items-center rounded-full bg-blue-500/15 px-2 py-0.5 text-[10px] font-medium text-blue-400"
      title={`Milestone: ${milestone}`}
    >
      {label}
    </span>
  );
});
