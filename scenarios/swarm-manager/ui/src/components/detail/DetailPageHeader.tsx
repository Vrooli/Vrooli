/**
 * DetailPageHeader
 *
 * Shared header for all entity detail pages. Provides:
 * - Close (X) button to return to graph
 * - Entity type badge
 * - Title and optional subtitle
 * - Status badge
 * - Action slot for entity-specific buttons
 *
 * Cross-lens navigation is handled by the separate LensBar component,
 * which each detail page renders below its tab bar.
 */

import { type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import { StatusBadge } from "./StatusBadge";

interface DetailPageHeaderProps {
  entityType: string;
  title: string;
  subtitle?: string;
  status?: string;
  onClose: () => void;
  actions?: ReactNode;
  className?: string;
}

export function DetailPageHeader({
  entityType,
  title,
  subtitle,
  status,
  onClose,
  actions,
  className,
}: DetailPageHeaderProps) {
  return (
    <header className={cn("border-b border-slate-800", className)} data-testid="detail-page-header">
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
    </header>
  );
}
