import { useState } from "react";
import { MarkdownRenderer } from "@vrooli/react-component-library/markdown-renderer/0";
import { AlertTriangle, Check, ChevronDown, ChevronUp, ExternalLink, Loader2, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib";
import type { ReviewResult } from "../../types";

const REVIEW_DIMENSION_COLORS: Record<string, string> = {
  green: "bg-emerald-500",
  yellow: "bg-amber-500",
  red: "bg-red-500",
  skipped: "bg-slate-500",
};

const CLASSIFICATION_LABELS: Record<string, string> = {
  ready: "Checks passed",
  ready_with_notes: "Passed with notes",
  needs_work: "Issues found",
  not_assessable: "Review inconclusive",
};

export interface ReviewStatusBadgeProps {
  result: ReviewResult;
  onOpenSandbox?: () => void;
  onTriggerReview?: () => void;
}

export function ReviewStatusBadge({ result, onOpenSandbox, onTriggerReview }: ReviewStatusBadgeProps) {
  const [showDimensions, setShowDimensions] = useState(false);
  const hasIssues = result.classification === "needs_work";

  return (
    <div className="space-y-1.5" data-testid="review-status-badge">
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          if ((result.dimensions ?? []).length > 0 || result.summary) setShowDimensions(!showDimensions);
        }}
        className={cn(
          "flex w-full items-center gap-2 rounded-md border px-2 py-1.5 text-xs transition-colors",
          result.classification === "ready" && "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
          result.classification === "ready_with_notes" && "border-amber-500/30 bg-amber-500/10 text-amber-300",
          result.classification === "needs_work" && "border-red-500/30 bg-red-500/10 text-red-300",
          result.classification === "not_assessable" && "border-slate-600 bg-slate-800/50 text-slate-400",
          ((result.dimensions ?? []).length > 0 || result.summary) && "cursor-pointer hover:border-white/20",
        )}
      >
        {result.classification === "ready" && <Check className="h-3.5 w-3.5 shrink-0" />}
        {(result.classification === "ready_with_notes" || result.classification === "needs_work") && (
          <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
        )}
        <span className="flex-1 text-left">
          {CLASSIFICATION_LABELS[result.classification] ?? "Review inconclusive"}
        </span>
        {((result.dimensions ?? []).length > 0 || result.summary) && (
          showDimensions
            ? <ChevronUp className="h-3 w-3 shrink-0 text-slate-500" />
            : <ChevronDown className="h-3 w-3 shrink-0 text-slate-500" />
        )}
      </button>

      {showDimensions && (
        <div className="space-y-1 rounded-md bg-slate-800/50 px-2.5 py-2">
          {(result.dimensions ?? []).map((dim) => (
            <div key={dim.name} className="flex items-center gap-2 text-xs" data-testid={`review-dim-${dim.name}`}>
              <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", REVIEW_DIMENSION_COLORS[dim.status] ?? "bg-slate-500")} />
              <span className="text-slate-300">{dim.name}</span>
              {dim.details && <MarkdownRenderer content={`— ${dim.details}`} className="prose-sm-slate text-slate-500" />}
            </div>
          ))}
          {result.summary && (
            <MarkdownRenderer content={result.summary} className="prose-sm-slate mt-1 text-[11px] leading-relaxed text-slate-400" />
          )}
        </div>
      )}

      <div className="flex gap-2">
        {hasIssues && onOpenSandbox && (
          <Button
            size="sm"
            variant="outline"
            className="flex-1 border-red-500/30 text-red-300 hover:bg-red-500/10 hover:text-red-200"
            onClick={(e) => {
              e.stopPropagation();
              onOpenSandbox();
            }}
            data-testid="review-open-sandbox"
          >
            <ExternalLink className="mr-1.5 h-3 w-3" />
            Review in Sandbox
          </Button>
        )}
        {onTriggerReview && (
          <Button
            size="sm"
            variant="outline"
            className="flex-1"
            onClick={(e) => {
              e.stopPropagation();
              onTriggerReview();
            }}
            data-testid="review-rerun-button"
          >
            <RefreshCw className="mr-1.5 h-3 w-3" />
            Re-run Review
          </Button>
        )}
      </div>
    </div>
  );
}

/** Inline indicator shown when a review is in progress. */
export function ReviewValidatingIndicator() {
  return (
    <div
      className="flex items-center gap-2 rounded-md border border-indigo-500/30 bg-indigo-500/10 px-2 py-1.5 text-xs text-indigo-300"
      data-testid="review-validating-indicator"
    >
      <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin" />
      <span>Git Control Tower review in progress&hellip;</span>
    </div>
  );
}

/** Inline indicator shown when a review was skipped. */
export function ReviewSkipIndicator({
  reason,
  onTriggerReview,
}: {
  reason: string;
  onTriggerReview?: () => void;
}) {
  return (
    <div className="flex items-center gap-1.5 rounded-md border border-slate-600 bg-slate-800/50 px-2 py-1.5 text-xs text-slate-400" data-testid="review-skip-indicator">
      <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
      <span className="flex-1">Review skipped: {reason}</span>
      {onTriggerReview && (
        <button
          type="button"
          className="ml-1 inline-flex items-center gap-1 text-indigo-400 hover:text-indigo-300"
          onClick={(e) => {
            e.stopPropagation();
            onTriggerReview();
          }}
          data-testid="review-run-button"
        >
          <RefreshCw className="h-3 w-3" />
          Run Review
        </button>
      )}
    </div>
  );
}
