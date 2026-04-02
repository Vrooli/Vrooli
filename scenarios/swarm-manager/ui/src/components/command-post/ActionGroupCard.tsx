/**
 * ActionGroupCard — Small card showing a group count, label, and bulk action button.
 *
 * Props-driven leaf component. Icon is selected by group ID.
 */

import { AlertTriangle, Eye, MessageCircle, Play, Tag } from "lucide-react";
import { Button } from "../ui/button";
import type { ActionGroup, ActionGroupId } from "../../lib/command-post-utils";

const GROUP_ICONS: Record<ActionGroupId, React.ElementType> = {
  "ready-to-run": Play,
  "pending-decisions": MessageCircle,
  "needs-review": Eye,
  failed: AlertTriangle,
  "needs-classification": Tag,
};

const GROUP_CTA_LABELS: Record<ActionGroupId, string> = {
  "ready-to-run": "Run All",
  "pending-decisions": "Answer All",
  "needs-review": "Review",
  failed: "Triage",
  "needs-classification": "Classify",
};

interface ActionGroupCardProps {
  group: ActionGroup;
  onBulkAction: () => void;
}

export function ActionGroupCard({ group, onBulkAction }: ActionGroupCardProps) {
  if (group.count === 0) return null;

  const Icon = GROUP_ICONS[group.id];
  const ctaLabel = GROUP_CTA_LABELS[group.id];

  return (
    <div
      className="flex min-w-[140px] flex-col gap-2 rounded-lg border border-slate-700/60 bg-slate-900/60 p-3"
      data-testid={`action-group-${group.id}`}
    >
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-slate-400" />
        <span className="text-2xl font-bold text-slate-100">{group.count}</span>
      </div>
      <p className="text-xs text-slate-400">{group.label}</p>
      <Button
        variant="outline"
        size="sm"
        onClick={onBulkAction}
        className="mt-auto w-full"
        data-testid={`action-group-${group.id}-cta`}
      >
        {ctaLabel}
      </Button>
    </div>
  );
}
