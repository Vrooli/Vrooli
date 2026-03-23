/**
 * Renders a single requirement or target as a reviewable card.
 * Mirrors the WorkshopItemCard interaction pattern with approve/flag/comment actions.
 *
 * Uses optimistic local state: clicking Approve/Flag immediately updates the visual
 * appearance while the API call is in flight. The `currentStatus` prop from the server
 * is the source of truth — local overrides are cleared when it changes.
 */
import { useState, useEffect } from "react";
import { CheckCircle2, AlertTriangle, Pencil, Trash2, RotateCcw, Loader2 } from "lucide-react";
import { cn } from "../../lib";
import type { ArchiveRequirement, ArchiveTarget, ReviewStatus } from "../../types";

interface ReviewCardProps {
  item: ArchiveRequirement | ArchiveTarget;
  itemType: "requirement" | "target";
  currentStatus: ReviewStatus;
  comment?: string;
  onApprove: () => void;
  onFlag: (comment: string) => void;
  onUnreview: () => void;
  onEdit: () => void;
  onRemove: () => void;
  disabled?: boolean;
  saving?: boolean;
  error?: string | null;
}

function isTarget(item: ArchiveRequirement | ArchiveTarget): item is ArchiveTarget {
  return "criticality" in item;
}

const CRITICALITY_COLORS: Record<string, string> = {
  P0: "text-red-400 border-red-500/30 bg-red-500/10",
  P1: "text-orange-400 border-orange-500/30 bg-orange-500/10",
  P2: "text-green-400 border-green-500/30 bg-green-500/10",
};

const BORDER_CLASSES: Record<ReviewStatus, string> = {
  approved: "border-emerald-500/40 bg-emerald-500/5",
  flagged: "border-amber-500/40 bg-amber-500/5",
  unreviewed: "border-slate-700 bg-slate-800/30",
};

const STATUS_BADGE: Record<ReviewStatus, { className: string; label: string }> = {
  approved: { className: "bg-emerald-500/20 text-emerald-400", label: "Approved" },
  flagged: { className: "bg-amber-500/20 text-amber-400", label: "Flagged" },
  unreviewed: { className: "bg-slate-700 text-slate-400", label: "Unreviewed" },
};

export function ReviewCard({
  item,
  itemType,
  currentStatus,
  comment: initialComment,
  onApprove,
  onFlag,
  onUnreview,
  onEdit,
  onRemove,
  disabled,
  saving,
  error,
}: ReviewCardProps) {
  const [localComment, setLocalComment] = useState(initialComment ?? "");
  const [showComment, setShowComment] = useState(currentStatus === "flagged");
  // Optimistic local override — cleared when server status changes
  const [optimisticStatus, setOptimisticStatus] = useState<ReviewStatus | null>(null);

  // Clear optimistic override when server status catches up
  useEffect(() => {
    setOptimisticStatus(null);
  }, [currentStatus]);

  const displayStatus = optimisticStatus ?? currentStatus;

  const handleFlag = () => {
    setOptimisticStatus("flagged");
    setShowComment(true);
    onFlag(localComment);
  };

  const handleApprove = () => {
    setOptimisticStatus("approved");
    setShowComment(false);
    onApprove();
  };

  const handleUnreview = () => {
    setOptimisticStatus("unreviewed");
    setShowComment(false);
    setLocalComment("");
    onUnreview();
  };

  const handleCommentChange = (value: string) => {
    setLocalComment(value);
    // Re-fire flag with updated comment
    if (displayStatus === "flagged") {
      onFlag(value);
    }
  };

  const badge = STATUS_BADGE[displayStatus];
  const description = itemType === "requirement"
    ? (item as ArchiveRequirement).description
    : (item as ArchiveTarget).notes;

  return (
    <div className={cn("rounded-md border p-3 transition-colors", BORDER_CLASSES[displayStatus])}>
      <div className="space-y-2">
        {/* Header: badge + ID + title + criticality */}
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-start gap-2 min-w-0">
            <span className={cn(
              "mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium",
              badge.className,
            )}>
              {saving ? (
                <Loader2 className="inline h-3 w-3 animate-spin" />
              ) : (
                badge.label
              )}
            </span>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-xs font-mono text-slate-500">{item.id}</span>
                {isTarget(item) && item.criticality && (
                  <span className={cn(
                    "rounded border px-1.5 py-0.5 text-[10px] font-medium",
                    CRITICALITY_COLORS[item.criticality] ?? "text-slate-400 border-slate-600 bg-slate-700/50",
                  )}>
                    {item.criticality}
                  </span>
                )}
              </div>
              <p className="mt-0.5 text-sm font-medium text-slate-200">{item.title}</p>
            </div>
          </div>
        </div>

        {/* Description/notes */}
        {description && (
          <p className="ml-7 text-xs text-slate-400 line-clamp-3">{description}</p>
        )}

        {/* Error banner */}
        {error && (
          <div className="ml-7 rounded-md border border-red-500/30 bg-red-500/10 px-2.5 py-1.5 text-xs text-red-300">
            {error}
          </div>
        )}

        {/* Action buttons */}
        <div className="ml-7 flex items-center gap-1.5">
          <button
            type="button"
            disabled={disabled || saving}
            onClick={handleApprove}
            className={cn(
              "flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors",
              displayStatus === "approved"
                ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400"
                : "border-slate-600 bg-slate-800/50 text-slate-400 hover:border-emerald-500/40 hover:text-emerald-400",
              (disabled || saving) && "opacity-50 cursor-not-allowed",
            )}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            Approve
          </button>
          <button
            type="button"
            disabled={disabled || saving}
            onClick={handleFlag}
            className={cn(
              "flex items-center gap-1 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors",
              displayStatus === "flagged"
                ? "border-amber-500/40 bg-amber-500/10 text-amber-400"
                : "border-slate-600 bg-slate-800/50 text-slate-400 hover:border-amber-500/40 hover:text-amber-400",
              (disabled || saving) && "opacity-50 cursor-not-allowed",
            )}
          >
            <AlertTriangle className="h-3.5 w-3.5" />
            Flag
          </button>
          {displayStatus !== "unreviewed" && (
            <button
              type="button"
              disabled={disabled || saving}
              onClick={handleUnreview}
              className={cn(
                "flex items-center gap-1 rounded-md border border-slate-600 bg-slate-800/50 px-2.5 py-1.5 text-xs font-medium text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-300",
                (disabled || saving) && "opacity-50 cursor-not-allowed",
              )}
            >
              <RotateCcw className="h-3.5 w-3.5" />
              Reset
            </button>
          )}
          <div className="flex-1" />
          <button
            type="button"
            disabled={disabled}
            onClick={onEdit}
            className={cn(
              "rounded p-1.5 text-slate-500 transition-colors hover:text-slate-300",
              disabled && "opacity-50 cursor-not-allowed",
            )}
            title="Edit"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={onRemove}
            className={cn(
              "rounded p-1.5 text-slate-500 transition-colors hover:text-red-400",
              disabled && "opacity-50 cursor-not-allowed",
            )}
            title="Remove"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Comment field — auto-expands on flag */}
        {showComment && (
          <div className="ml-7">
            <textarea
              className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
              placeholder="Comment (optional)..."
              value={localComment}
              onChange={(e) => handleCommentChange(e.target.value)}
              disabled={disabled}
              rows={2}
            />
          </div>
        )}
      </div>
    </div>
  );
}
