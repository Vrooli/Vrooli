// Individual health check result card
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:UI-EVENTS-001] [REQ:HEAL-ACTION-001]
import { Clock, AlertTriangle, CheckCircle2, XCircle, Info } from "lucide-react";
import { memo, useCallback } from "react";
import type { MouseEvent } from "react";
import { ActionButtons, StatusIcon } from "../../../shared/components";
import { Card } from "../../../shared/ui/primitives";
import { Notice } from "../../../shared/ui/composites";
import { type ActionLog, type HealthResult, type SubCheck, type CheckCategory } from "../../../lib/api";
import { selectors } from "../../../consts/selectors";
import { formatRelativeTime } from "../../../lib/utils";

interface EnrichedCheck extends HealthResult {
  title?: string;
  description?: string;
  importance?: string;
  category?: CheckCategory;
  intervalSeconds?: number;
  autoHealIssue?: ActionLog;
}

interface CheckCardProps {
  check: EnrichedCheck;
  onInfoClick?: (checkId: string) => void;
  mobileListItem?: boolean;
}

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h`;
}

function CheckCardImpl({ check, onInfoClick, mobileListItem = false }: CheckCardProps) {
  const score = check.metrics?.score;
  const subChecks = check.metrics?.subChecks ?? [];
  const hasSubChecks = subChecks.length > 0;
  const hasScore = score !== undefined;
  const isNonOk = check.status !== "ok";

  // Use title if available, fall back to checkId
  const displayTitle = check.title || check.checkId;

  const handleInfoClick = useCallback((e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onInfoClick?.(check.checkId);
  }, [check.checkId, onInfoClick]);

  return (
    <Card
      variant="default"
      className={`min-w-0 w-full p-4 ${onInfoClick ? "hover:bg-surface-overlay/60" : ""} ${
        mobileListItem
          ? "rounded-none border-x-0 border-t-0 bg-transparent shadow-none first:rounded-t-lg first:border-t last:rounded-b-lg last:border-b sm:rounded-lg sm:border sm:border-border-default/70 sm:bg-surface-elevated/70 sm:shadow-panel"
          : ""
      }`}
      data-testid={selectors.checkCard}
    >
      <div className="flex min-w-0 items-start gap-3">
        <StatusIcon status={check.status} />
        <div className="flex-1 min-w-0">
          {/* Header row: Title + Info icon + timing info */}
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <div className="min-w-0">
                <h3 className="truncate font-medium text-text-primary" title={check.checkId}>
                  {displayTitle}
                </h3>
                {check.title && (
                  <span className="break-all font-mono text-xs text-text-muted/80">{check.checkId}</span>
                )}
              </div>
              {onInfoClick && (
                <button
                  onClick={handleInfoClick}
                  className="shrink-0 rounded p-1 text-text-muted transition-colors hover:bg-surface-overlay/70 hover:text-accent-primary"
                  title="View details"
                  aria-label="View check details"
                >
                  <Info size={14} />
                </button>
              )}
            </div>
            <div className="hidden shrink-0 items-center gap-3 text-xs text-text-muted md:flex">
              {hasScore && (
                <span className="flex items-center gap-1" title="Health score">
                  <span className={`font-medium ${score >= 80 ? "text-accent-success" : score >= 50 ? "text-accent-warning" : "text-accent-danger"}`}>
                    {score}%
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
          {check.description && <p className="mt-0.5 break-words text-xs text-text-muted">{check.description}</p>}

          {/* Message (from check result) */}
          <p className="mt-1 break-words text-sm text-text-primary/90">{check.message}</p>

          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-muted md:hidden">
            {hasScore && (
              <span className={score >= 80 ? "text-accent-success" : score >= 50 ? "text-accent-warning" : "text-accent-danger"}>
                Score {score}%
              </span>
            )}
            {check.intervalSeconds && (
              <span className="flex items-center gap-1">
                <Clock size={12} />
                {formatInterval(check.intervalSeconds)}
              </span>
            )}
            <span title={new Date(check.timestamp).toLocaleString()}>{formatRelativeTime(check.timestamp)}</span>
            <span>{Math.round(check.duration / 1000000)}ms</span>
          </div>

          {/* Importance notice - shown when status is not ok */}
          {isNonOk && check.importance ? (
            <Notice tone="warning" className="mt-2 flex min-w-0 items-start gap-2 p-2">
              <AlertTriangle size={14} className="mt-0.5 shrink-0 text-accent-warning" />
              <p className="min-w-0 flex-1 break-words text-xs text-accent-warning">{check.importance}</p>
            </Notice>
          ) : null}

          {check.autoHealIssue ? <HealingIssueNotice issue={check.autoHealIssue} /> : null}

          {/* Sub-checks - displayed as structured checklist */}
          {hasSubChecks && (
            <div className="mt-2 space-y-1">
              {subChecks.map((sc, idx) => (
                <SubCheckRow key={idx} subCheck={sc} />
              ))}
            </div>
          )}

          {/* Recovery actions (for resource checks) - always visible */}
          <ActionButtons checkId={check.checkId} category={check.category} />
        </div>
      </div>
    </Card>
  );
}

export const CheckCard = memo(CheckCardImpl);

// Renders a single sub-check as a pass/fail indicator
function SubCheckRow({ subCheck }: { subCheck: SubCheck }) {
  const Icon = subCheck.passed ? CheckCircle2 : XCircle;
  const colorClass = subCheck.passed ? "text-accent-success" : "text-accent-danger";

  return (
    <div className="flex min-w-0 items-center gap-2 text-xs">
      <Icon size={12} className={colorClass} />
      <span className={`min-w-0 break-words ${subCheck.passed ? "text-text-muted" : "text-text-primary"}`}>
        {subCheck.name}
      </span>
      {subCheck.detail && (
        <span className="min-w-0 break-words text-text-muted/80">- {subCheck.detail}</span>
      )}
    </div>
  );
}

function HealingIssueNotice({ issue }: { issue: ActionLog }) {
  const skipped = issue.actionId === "autoheal-skip";
  return (
    <Notice tone={skipped ? "warning" : "danger"} className="mt-2 flex min-w-0 items-start gap-2 p-2">
      {skipped ? (
        <AlertTriangle size={14} className="mt-0.5 shrink-0 text-accent-warning" />
      ) : (
        <XCircle size={14} className="mt-0.5 shrink-0 text-accent-danger" />
      )}
      <div className="min-w-0 flex-1">
        <p className={`text-xs font-medium ${skipped ? "text-accent-warning" : "text-accent-danger"}`}>
          {skipped ? "Auto-heal skipped" : "Auto-heal failed"}
        </p>
        <p className="mt-0.5 break-words text-xs text-text-muted">
          {issue.message || issue.error || `Action ${issue.actionId} did not complete`}
        </p>
        <p className="mt-0.5 text-[11px] text-text-muted/80" title={new Date(issue.timestamp).toLocaleString()}>
          Last recovery outcome: {formatRelativeTime(issue.timestamp)}
        </p>
      </div>
    </Notice>
  );
}
