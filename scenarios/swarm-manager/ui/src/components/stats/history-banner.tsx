/**
 * HistoryBanner — shown at the top of the Stats panel when the event-log
 * history is shorter than the largest lookback window users see in the UI
 * (30 days). Prevents misreading aggregate windows as long-term trends.
 */

import { Info } from "lucide-react";
import type { HistoryWindow } from "../../types/stats";

const LARGEST_WINDOW_DAYS = 30;

interface HistoryBannerProps {
  history: HistoryWindow;
  testId?: string;
}

export function HistoryBanner({ history, testId }: HistoryBannerProps) {
  if (!history.has_history) {
    return null;
  }
  if (history.history_days >= LARGEST_WINDOW_DAYS) {
    return null;
  }
  const days = Math.max(1, Math.round(history.history_days));
  return (
    <div
      className="mb-3 flex items-start gap-2 rounded-md border border-slate-700/50 bg-slate-900/40 px-3 py-2 text-xs text-slate-400"
      data-testid={testId}
    >
      <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" />
      <span>
        Event history covers {days} day{days === 1 ? "" : "s"}. The 30-day
        windows below equal the full history so far — metrics will stabilize as
        more events are recorded.
      </span>
    </div>
  );
}
