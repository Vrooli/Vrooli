/**
 * Review-related indicator components and helpers for operational targets/requirements.
 */

import { CheckCircle2, Circle } from "lucide-react";
import type { ReviewStatus } from "../../types";

// eslint-disable-next-line react-refresh/only-export-components
export function getReviewStatus(item: { review_status?: ReviewStatus }): ReviewStatus {
  return item.review_status ?? "unreviewed";
}

export function ReviewSummary({ reviewed, flagged, total }: { reviewed: number; flagged: number; total: number }) {
  if (total === 0) return null;
  return (
    <div className="flex items-center gap-3 text-xs">
      <span className="text-emerald-400">
        <CheckCircle2 className="mr-1 inline h-3 w-3" />
        {reviewed}/{total} reviewed
      </span>
      {flagged > 0 && (
        <span className="text-amber-400">{flagged} flagged</span>
      )}
    </div>
  );
}

export function ReviewProgressBar({ reviewed, total }: { reviewed: number; total: number }) {
  if (total === 0) return null;
  const pct = Math.round((reviewed / total) * 100);
  return (
    <div className="mb-2">
      <div className="flex items-center justify-between text-xs text-slate-500 mb-1">
        <span>{reviewed}/{total} reviewed</span>
        <span>{pct}%</span>
      </div>
      <div className="h-1 w-full rounded-full bg-slate-700">
        <div className="h-1 rounded-full bg-emerald-500 transition-all" style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

export function StatusIcon({ status }: { status: string }) {
  if (status === "complete") {
    return <CheckCircle2 className="h-4 w-4 text-green-400" />;
  }
  return <Circle className="h-4 w-4 text-slate-500" />;
}

export function ReviewStatusIndicator({ reviewStatus }: { reviewStatus: ReviewStatus }) {
  if (reviewStatus === "approved") {
    return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />;
  }
  if (reviewStatus === "flagged") {
    return <span className="h-3.5 w-3.5 rounded-full border-2 border-amber-400" />;
  }
  return null;
}

// eslint-disable-next-line react-refresh/only-export-components
export function countAllRequirements(groups: { requirements: unknown[]; children: typeof groups }[]): unknown[] {
  const all: unknown[] = [];
  for (const g of groups) {
    all.push(...g.requirements);
    all.push(...countAllRequirements(g.children));
  }
  return all;
}
