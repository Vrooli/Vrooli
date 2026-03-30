/**
 * DetailPageHeader
 *
 * Shared header for all entity detail pages. Provides:
 * - Close (X) button to return to graph
 * - Entity type badge
 * - Title and optional subtitle
 * - Status badge
 * - Optional cross-lens navigation (View History / View Operations)
 * - Action slot for entity-specific buttons
 */

import { type ReactNode } from "react";
import { X, History, Activity as ActivityIcon } from "lucide-react";
import { cn } from "../../lib/utils";
import { StatusBadge } from "./StatusBadge";

interface CrossLensNavProps {
  nodeId: string;
  onDrillToFlow?: (nodeId: string) => void;
  onDrillToOperations?: (nodeId: string) => void;
  activeExecutionStatus?: string;
}

interface DetailPageHeaderProps {
  entityType: string;
  title: string;
  subtitle?: string;
  status?: string;
  onClose: () => void;
  actions?: ReactNode;
  crossLensNav?: CrossLensNavProps;
  className?: string;
}

function CrossLensNav({ nodeId, onDrillToFlow, onDrillToOperations, activeExecutionStatus }: CrossLensNavProps) {
  if (!onDrillToFlow && !onDrillToOperations) return null;

  return (
    <div className="flex items-center gap-2" data-testid="detail-cross-lens-nav">
      {onDrillToFlow && (
        <button
          type="button"
          onClick={() => onDrillToFlow(nodeId)}
          className="flex items-center gap-1.5 rounded-lg bg-slate-700/50 px-3 py-1.5 text-sm font-medium text-slate-100 transition-colors hover:bg-slate-700/70"
          data-testid="detail-drill-flow"
        >
          <History className="h-3.5 w-3.5 text-cyan-400" />
          View History
        </button>
      )}
      {onDrillToOperations && (
        <button
          type="button"
          onClick={() => onDrillToOperations(nodeId)}
          className="flex items-center gap-1.5 rounded-lg bg-slate-700/50 px-3 py-1.5 text-sm font-medium text-slate-100 transition-colors hover:bg-slate-700/70"
          data-testid="detail-drill-operations"
        >
          <ActivityIcon className="h-3.5 w-3.5 text-amber-400" />
          View Operations
          {activeExecutionStatus && (
            <span
              className={cn(
                "ml-0.5 h-2 w-2 rounded-full",
                activeExecutionStatus === "running" || activeExecutionStatus === "starting"
                  ? "animate-pulse bg-cyan-400"
                  : activeExecutionStatus === "needs_review" || activeExecutionStatus === "needs_fixup"
                    ? "bg-amber-400"
                    : activeExecutionStatus === "failed"
                      ? "bg-red-400"
                      : "bg-slate-400",
              )}
            />
          )}
        </button>
      )}
    </div>
  );
}

export function DetailPageHeader({
  entityType,
  title,
  subtitle,
  status,
  onClose,
  actions,
  crossLensNav,
  className,
}: DetailPageHeaderProps) {
  return (
    <header className={cn("border-b border-slate-800", className)} data-testid="detail-page-header">
      {/* Primary row */}
      <div className="flex items-center gap-3 px-4 py-3 md:px-6">
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close detail view"
          data-testid="detail-close"
        >
          <X className="h-5 w-5" />
        </button>

        <span className="rounded-full bg-slate-700/60 px-2 py-0.5 text-xs font-medium uppercase tracking-wider text-slate-400">
          {entityType}
        </span>

        <div className="min-w-0 flex-1">
          <h1 className="truncate text-lg font-semibold text-slate-100">{title}</h1>
          {subtitle && (
            <p className="truncate text-sm text-slate-400">{subtitle}</p>
          )}
        </div>

        {status && <StatusBadge status={status} />}

        {actions && (
          <div className="flex items-center gap-2">
            {actions}
          </div>
        )}
      </div>

      {/* Cross-lens navigation row (only if provided) */}
      {crossLensNav && (
        <div className="border-t border-slate-800/50 px-4 py-2 md:px-6">
          <CrossLensNav {...crossLensNav} />
        </div>
      )}
    </header>
  );
}
