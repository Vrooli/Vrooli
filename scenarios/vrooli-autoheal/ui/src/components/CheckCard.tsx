// Individual health check result card with history drawer
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001]
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Clock, ChevronDown, ChevronRight, History, Loader2, AlertTriangle, CheckCircle2, XCircle, Info, BookOpen } from "lucide-react";
import { StatusIcon } from "./StatusIcon";
import { fetchCheckHistory, type HealthResult, type HistoryEntry, type SubCheck, type CheckCategory } from "../lib/api";
import { selectors } from "../consts/selectors";
import { navigateToCheckDocs } from "../lib/docs";

interface EnrichedCheck extends HealthResult {
  title?: string;
  description?: string;
  importance?: string;
  category?: CheckCategory;
  intervalSeconds?: number;
}

interface CheckCardProps {
  check: EnrichedCheck;
}

function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}h ago`;
  return date.toLocaleDateString();
}

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h`;
}

type ViewMode = "closed" | "details" | "history";

export function CheckCard({ check }: CheckCardProps) {
  const [viewMode, setViewMode] = useState<ViewMode>("closed");
  const hasDetails = check.details && Object.keys(check.details).length > 0;
  const hasSubChecks = check.metrics?.subChecks && check.metrics.subChecks.length > 0;
  const hasScore = check.metrics?.score !== undefined;
  const isNonOk = check.status !== "ok";

  // Only fetch history when the history view is requested
  const { data: historyData, isLoading: historyLoading } = useQuery({
    queryKey: ["check-history", check.checkId],
    queryFn: () => fetchCheckHistory(check.checkId),
    enabled: viewMode === "history",
    staleTime: 30000,
  });

  const toggleView = (mode: ViewMode) => {
    setViewMode(viewMode === mode ? "closed" : mode);
  };

  // Use title if available, fall back to checkId
  const displayTitle = check.title || check.checkId;

  return (
    <div
      className="rounded-lg border border-white/10 bg-white/5 p-4 hover:bg-white/[0.07] transition-colors"
      data-testid={selectors.checkCard}
    >
      <div className="flex items-start gap-3">
        <StatusIcon status={check.status} />
        <div className="flex-1 min-w-0">
          {/* Header row: Title + timing info */}
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <h3 className="font-medium text-slate-200 truncate" title={check.checkId}>
                {displayTitle}
              </h3>
              {check.title && (
                <span className="text-xs text-slate-600 font-mono">{check.checkId}</span>
              )}
            </div>
            <div className="flex items-center gap-3 text-xs text-slate-500 flex-shrink-0">
              {hasScore && (
                <span className="flex items-center gap-1" title="Health score">
                  <span className={`font-medium ${check.metrics!.score! >= 80 ? "text-emerald-400" : check.metrics!.score! >= 50 ? "text-amber-400" : "text-red-400"}`}>
                    {check.metrics!.score}%
                  </span>
                </span>
              )}
              {check.intervalSeconds && (
                <span className="flex items-center gap-1" title="Check interval">
                  <Clock size={12} />
                  {formatInterval(check.intervalSeconds)}
                </span>
              )}
              <span title={new Date(check.timestamp).toLocaleString()}>
                {formatRelativeTime(check.timestamp)}
              </span>
              <span>{Math.round(check.duration / 1000000)}ms</span>
            </div>
          </div>

          {/* Description (from check metadata) */}
          {check.description && (
            <p className="text-xs text-slate-500 mt-0.5">{check.description}</p>
          )}

          {/* Message (from check result) */}
          <p className="text-sm text-slate-400 mt-1">{check.message}</p>

          {/* Importance notice - shown when status is not ok */}
          {isNonOk && check.importance && (
            <div className="flex items-start gap-2 mt-2 p-2 rounded bg-amber-500/10 border border-amber-500/20">
              <AlertTriangle size={14} className="text-amber-400 mt-0.5 flex-shrink-0" />
              <p className="text-xs text-amber-300">{check.importance}</p>
            </div>
          )}

          {/* Sub-checks - displayed as structured checklist */}
          {hasSubChecks && (
            <div className="mt-2 space-y-1">
              {check.metrics!.subChecks!.map((sc, idx) => (
                <SubCheckRow key={idx} subCheck={sc} />
              ))}
            </div>
          )}

          {/* Action buttons */}
          <div className="flex items-center gap-3 mt-2">
            {hasDetails && (
              <button
                onClick={() => toggleView("details")}
                className={`flex items-center gap-1 text-xs transition-colors ${
                  viewMode === "details" ? "text-blue-400" : "text-slate-500 hover:text-slate-300"
                }`}
              >
                {viewMode === "details" ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                <span>Details</span>
              </button>
            )}
            <button
              onClick={() => toggleView("history")}
              className={`flex items-center gap-1 text-xs transition-colors ${
                viewMode === "history" ? "text-blue-400" : "text-slate-500 hover:text-slate-300"
              }`}
              data-testid={selectors.checkHistory}
            >
              <History size={12} />
              <span>History</span>
            </button>
            <button
              onClick={() => navigateToCheckDocs(check.checkId)}
              className="flex items-center gap-1 text-xs text-slate-500 hover:text-blue-400 transition-colors"
              title="View documentation for this check"
              data-testid="check-learn-more"
            >
              <BookOpen size={12} />
              <span>Learn more</span>
            </button>
          </div>

          {/* Expanded details */}
          {viewMode === "details" && hasDetails && (
            <div className="mt-3 p-3 rounded-lg bg-black/30 text-xs font-mono">
              <pre className="overflow-x-auto whitespace-pre-wrap">{JSON.stringify(check.details, null, 2)}</pre>
            </div>
          )}

          {/* History drawer */}
          {viewMode === "history" && (
            <div className="mt-3 rounded-lg bg-black/30 overflow-hidden">
              <div className="p-2 border-b border-white/10 text-xs text-slate-400 flex items-center gap-2">
                <History size={12} />
                Recent History ({historyData?.count || 0} entries)
              </div>
              {historyLoading ? (
                <div className="p-4 flex items-center justify-center text-slate-500">
                  <Loader2 size={16} className="animate-spin mr-2" />
                  Loading history...
                </div>
              ) : historyData?.history && historyData.history.length > 0 ? (
                <div className="max-h-48 overflow-y-auto divide-y divide-white/5">
                  {historyData.history.map((entry, idx) => (
                    <HistoryRow key={`${entry.timestamp}-${idx}`} entry={entry} />
                  ))}
                </div>
              ) : (
                <div className="p-4 text-center text-sm text-slate-500">
                  No history available
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Renders a single sub-check as a pass/fail indicator
function SubCheckRow({ subCheck }: { subCheck: SubCheck }) {
  const Icon = subCheck.passed ? CheckCircle2 : XCircle;
  const colorClass = subCheck.passed ? "text-emerald-400" : "text-red-400";

  return (
    <div className="flex items-center gap-2 text-xs">
      <Icon size={12} className={colorClass} />
      <span className={subCheck.passed ? "text-slate-400" : "text-slate-300"}>
        {subCheck.name}
      </span>
      {subCheck.detail && (
        <span className="text-slate-500">— {subCheck.detail}</span>
      )}
    </div>
  );
}

function HistoryRow({ entry }: { entry: HistoryEntry }) {
  return (
    <div className="flex items-center gap-3 p-2 hover:bg-white/[0.02]">
      <StatusIcon status={entry.status} size={12} />
      <div className="flex-1 min-w-0">
        <p className="text-xs text-slate-300 truncate">{entry.message}</p>
      </div>
      <span className="text-xs text-slate-500 flex-shrink-0" title={new Date(entry.timestamp).toLocaleString()}>
        {formatRelativeTime(entry.timestamp)}
      </span>
    </div>
  );
}
