/**
 * Pending Decision Badge
 *
 * Small pill badges showing why a backlog item needs user attention.
 */

import { memo } from "react";
import type { AttentionReason } from "../../lib/feed";

interface PendingDecisionBadgeProps {
  reasons: AttentionReason[];
}

const REASON_CONFIG: Record<AttentionReason["kind"], { label: (count: number) => string; color: string }> = {
  "pending-decisions": {
    label: (n) => `${n} decision${n !== 1 ? "s" : ""}`,
    color: "bg-amber-500/20 text-amber-400",
  },
  "plan-ready": {
    label: () => "Plan ready",
    color: "bg-emerald-500/20 text-emerald-400",
  },
  "research-complete": {
    label: () => "Review ready",
    color: "bg-green-500/20 text-green-400",
  },
};

export const PendingDecisionBadge = memo(function PendingDecisionBadge({ reasons }: PendingDecisionBadgeProps) {
  if (reasons.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1">
      {reasons.map((reason) => {
        const config = REASON_CONFIG[reason.kind];
        const count = "count" in reason ? reason.count : 0;
        return (
          <span
            key={reason.kind}
            className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${config.color}`}
          >
            {config.label(count)}
          </span>
        );
      })}
    </div>
  );
});
