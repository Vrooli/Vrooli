// Individual health check result card
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001] [REQ:HEAL-ACTION-001]
import { Clock, AlertTriangle, CheckCircle2, XCircle, Info } from "lucide-react";
import { StatusIcon } from "./StatusIcon";
import { ActionButtons } from "./ActionButtons";
import { type HealthResult, type SubCheck, type CheckCategory } from "../lib/api";
import { selectors } from "../consts/selectors";

interface EnrichedCheck extends HealthResult {
  title?: string;
  description?: string;
  importance?: string;
  category?: CheckCategory;
  intervalSeconds?: number;
}

interface CheckCardProps {
  check: EnrichedCheck;
  onInfoClick?: (checkId: string) => void;
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

export function CheckCard({ check, onInfoClick }: CheckCardProps) {
  const hasSubChecks = check.metrics?.subChecks && check.metrics.subChecks.length > 0;
  const hasScore = check.metrics?.score !== undefined;
  const isNonOk = check.status !== "ok";

  // Use title if available, fall back to checkId
  const displayTitle = check.title || check.checkId;

  const handleCardClick = () => {
    if (onInfoClick) {
      onInfoClick(check.checkId);
    }
  };

  return (
    <div
      className={`rounded-lg border border-white/10 bg-white/5 p-4 transition-colors ${
        onInfoClick ? "hover:bg-white/[0.07] cursor-pointer" : ""
      }`}
      data-testid={selectors.checkCard}
      onClick={handleCardClick}
      role={onInfoClick ? "button" : undefined}
      tabIndex={onInfoClick ? 0 : undefined}
      onKeyDown={(e) => {
        if (onInfoClick && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault();
          onInfoClick(check.checkId);
        }
      }}
    >
      <div className="flex items-start gap-3">
        <StatusIcon status={check.status} />
        <div className="flex-1 min-w-0">
          {/* Header row: Title + Info icon + timing info */}
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <div className="min-w-0">
                <h3 className="font-medium text-slate-200 truncate" title={check.checkId}>
                  {displayTitle}
                </h3>
                {check.title && (
                  <span className="text-xs text-slate-600 font-mono">{check.checkId}</span>
                )}
              </div>
              {onInfoClick && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onInfoClick(check.checkId);
                  }}
                  className="p-1 rounded hover:bg-white/10 text-slate-500 hover:text-blue-400 transition-colors flex-shrink-0"
                  title="View details"
                  aria-label="View check details"
                >
                  <Info size={14} />
                </button>
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

          {/* Recovery actions (for resource checks) - always visible */}
          <ActionButtons checkId={check.checkId} category={check.category} />
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
        <span className="text-slate-500">- {subCheck.detail}</span>
      )}
    </div>
  );
}
