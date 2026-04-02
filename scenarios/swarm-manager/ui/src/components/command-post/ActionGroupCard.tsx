/**
 * ActionGroupCard — Small card showing a group count, label, and bulk action button.
 *
 * Clicking the card body filters the "Needs Attention" list to this group.
 * The CTA button triggers the bulk action (e.g., "Run All", "Answer All").
 */

import { Eye, Info, MessageCircle, Play, Sparkles, Tag } from "lucide-react";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";
import type { ActionGroup, ActionGroupId } from "../../lib/command-post-utils";

const GROUP_ICONS: Record<ActionGroupId, React.ElementType> = {
  "needs-workshop": Sparkles,
  "ready-to-run": Play,
  "pending-decisions": MessageCircle,
  "needs-review": Eye,
  "needs-classification": Tag,
};

const GROUP_CTA_LABELS: Record<ActionGroupId, string> = {
  "needs-workshop": "Workshop All",
  "ready-to-run": "Run All",
  "pending-decisions": "Answer All",
  "needs-review": "Review",
  "needs-classification": "Classify",
};

interface ActionGroupCardProps {
  group: ActionGroup;
  isActive: boolean;
  onBulkAction: () => void;
  onFilter: () => void;
}

export function ActionGroupCard({ group, isActive, onBulkAction, onFilter }: ActionGroupCardProps) {
  if (group.count === 0) return null;

  const Icon = GROUP_ICONS[group.id];
  const ctaLabel = GROUP_CTA_LABELS[group.id];

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onFilter}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") onFilter();
      }}
      className={cn(
        "flex cursor-pointer flex-col gap-2 rounded-lg border p-3 transition-colors",
        isActive
          ? "border-cyan-500/50 bg-cyan-500/10"
          : "border-slate-700/60 bg-slate-900/60 hover:border-slate-600",
      )}
      data-testid={`action-group-${group.id}`}
    >
      <div className="flex items-center gap-2">
        <Icon className="h-4 w-4 text-slate-400" />
        <span className="text-2xl font-bold text-slate-100">{group.count}</span>
        <Info className="ml-auto h-3.5 w-3.5 text-slate-600" />
      </div>
      <p className="text-xs text-slate-400">{group.label}</p>
      <Button
        variant="outline"
        size="sm"
        onClick={(e) => {
          e.stopPropagation();
          onBulkAction();
        }}
        className="mt-auto w-full"
        data-testid={`action-group-${group.id}-cta`}
      >
        {ctaLabel}
      </Button>
    </div>
  );
}
