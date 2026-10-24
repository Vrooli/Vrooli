import { Info } from "lucide-react";
import type { HistoryCoverage, HistoryWindow } from "./HistoryWindow";

const LARGEST_WINDOW_DAYS = 30;

interface HistoryBannerProps {
  history?: HistoryWindow;
  coverage?: HistoryCoverage;
  testId?: string;
}

export function HistoryBanner({ history, coverage, testId }: HistoryBannerProps) {
  if (coverage?.historyFloor) {
    return (
      <div className="mb-3 flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-muted-foreground" data-testid={testId}>
        <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-500" />
        <span>Durable statistics cover history from {new Date(coverage.historyFloor).toLocaleDateString()}. {coverage.outsideHistoryRunCount.toLocaleString()} older run{coverage.outsideHistoryRunCount === 1 ? " is" : "s are"} outside the retained analytical read model.</span>
      </div>
    );
  }
  if (!history || !history.has_history) {
    return null;
  }
  if (history.history_days >= LARGEST_WINDOW_DAYS) {
    return null;
  }
  const days = Math.max(1, Math.round(history.history_days));
  return (
    <div
      className="mb-3 flex items-start gap-2 rounded-md border border-border/60 bg-card/40 px-3 py-2 text-xs text-muted-foreground"
      data-testid={testId}
    >
      <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>
        Event history covers {days} day{days === 1 ? "" : "s"}. The 30-day
        windows below equal the full history so far — metrics will stabilize as
        more events are recorded.
      </span>
    </div>
  );
}
