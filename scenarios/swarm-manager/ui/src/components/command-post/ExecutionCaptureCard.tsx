/**
 * ExecutionCaptureCard — Minimal feed card for execution and capture items
 * in the Command Post. These items don't have BacklogItem data so they
 * use a simplified presentation.
 */

import { CheckCircle, Clock, Eye } from "lucide-react";
import { Button } from "../ui/button";
import { SnoozePopover } from "./SnoozePopover";
import type { ActionableItem } from "../../lib/command-post-utils";

interface ExecutionCaptureCardProps {
  item: ActionableItem;
  onNavigate: () => void;
  onSnooze: (key: string, expiresAt: number) => void;
}

export function ExecutionCaptureCard({
  item,
  onNavigate,
  onSnooze,
}: ExecutionCaptureCardProps) {
  const isCaptureWithTriage = item.type === "capture" && item.primaryCta === "review";
  const isExecutionReview = item.type === "execution" && item.primaryCta === "review";

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onNavigate}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") onNavigate();
      }}
      className="flex cursor-pointer items-center justify-between gap-2 rounded-lg border border-slate-700/40 bg-slate-900/40 px-3 py-2.5 transition-colors hover:bg-slate-800/60"
      data-testid={`execution-capture-card-${item.key}`}
    >
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-slate-200">{item.title}</p>
        <span className="text-[10px] uppercase tracking-wider text-slate-500">
          {item.type === "execution" ? "Execution" : "Capture"}
        </span>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {isCaptureWithTriage && (
          <Button
            variant="outline"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onNavigate();
            }}
          >
            <CheckCircle className="mr-1 h-3.5 w-3.5" />
            Triage
          </Button>
        )}
        {isExecutionReview && (
          <Button
            variant="outline"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              onNavigate();
            }}
          >
            <Eye className="mr-1 h-3.5 w-3.5" />
            Review
          </Button>
        )}
        <SnoozePopover itemKey={item.key} onSnooze={onSnooze}>
          <Clock className="h-3.5 w-3.5" />
        </SnoozePopover>
      </div>
    </div>
  );
}
