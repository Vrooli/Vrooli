/**
 * DetailPageHeader
 *
 * Unified header for all entity detail pages. Two-row layout:
 *
 * Row 1: nav button + title (nearly full width for readability)
 * Row 2: entity type badge + status + subtitle + primary action
 *
 * Also provides:
 * - Integrated LensBar for cross-lens navigation
 * - Optional tab bar slot for entity-specific tabs (e.g., backlog info/prompt/files)
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-2
 */

import { type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Menu, X, type LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";
import { graphPath } from "../../app/routes/route-paths";
import { useAppBack } from "../../app/routes/useAppBack";
import { useAppShell } from "../../app/shell/AppShellContext";
import { StatusBadge } from "./StatusBadge";
import { TitlePopover } from "./TitlePopover";
import { LensBar } from "./LensBar";
import type { LensOption } from "./lens-options";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";
import type { BacklogStatus } from "../../types";

export interface DetailPageHeaderProps {
  entityType: string;
  /** Optional icon rendered inside the entity type badge. */
  entityIcon?: LucideIcon;
  title: string;
  subtitle?: string;
  status?: string;
  /** Graph node ID for cross-lens navigation. */
  nodeId: string | null;
  /** Available lens navigation options. */
  lenses: LensOption[];
  /** Entity-specific action buttons (rendered in the metadata row). */
  actions?: ReactNode;
  /** Optional metadata chips rendered beside the entity/status badges. */
  metadata?: ReactNode;
  /** Optional tab bar rendered below the LensBar (e.g., backlog tabs). */
  tabBar?: ReactNode;
  /** When provided, the status badge becomes clickable for inline status changes. */
  onStatusChange?: (newStatus: BacklogStatus) => void;
  /** Whether a status change is in flight. */
  statusChangePending?: boolean;
  className?: string;
}

export function DetailPageHeader({
  entityType,
  entityIcon: EntityIcon,
  title,
  subtitle,
  status,
  nodeId,
  lenses,
  actions,
  metadata,
  tabBar,
  onStatusChange,
  statusChangePending,
  className,
}: DetailPageHeaderProps) {
  const goBack = useAppBack();
  const navigate = useNavigate();
  const { openSidebar } = useAppShell();

  const handleCloseClick = () => {
    goBack();
  };

  const handleDrillToLens = (id: string, lens: GraphLens) => {
    navigate(graphPath({ lens, focus: id, select: id }));
  };

  return (
    <header className={cn("border-b border-slate-800", className)} data-testid="detail-page-header">
      {/* Menu button (left) + two-row content + route close (right) */}
      <div className="flex items-center gap-3 px-4 py-3 md:px-6">
        <button
          type="button"
          onClick={openSidebar}
          className="shrink-0 self-center rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Open sidebar"
          data-testid="page-sidebar-button"
        >
          <Menu className="h-5 w-5" />
        </button>

        <div className="min-w-0 flex-1">
          {/* Row 1: title (click to view full title + copy) */}
          <h1 className="min-w-0 text-lg font-semibold text-slate-100">
            <TitlePopover
              title={title}
              className="block w-full min-w-0 max-w-full cursor-pointer rounded text-slate-100 transition-colors hover:text-blue-300"
            />
          </h1>

          {/* Row 2: entity badge + status + subtitle + actions */}
          <div className="mt-1 flex items-center gap-2">
            {EntityIcon ? (
              <span className="inline-flex h-6 shrink-0 items-center rounded-full bg-slate-700/60 px-1.5" title={entityType}>
                <EntityIcon className="h-3.5 w-3.5 text-slate-400" />
              </span>
            ) : (
              <span className="inline-flex h-6 shrink-0 items-center rounded-full bg-slate-700/60 px-2 text-xs font-medium uppercase tracking-wider text-slate-400">
                {entityType}
              </span>
            )}

            {status && (
              <StatusBadge
                status={status}
                size="sm"
                onStatusChange={onStatusChange}
                statusChangePending={statusChangePending}
              />
            )}

            {subtitle && (
              <p className="min-w-0 truncate text-sm text-slate-400">{subtitle}</p>
            )}

            {metadata && (
              <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                {metadata}
              </div>
            )}

            {/* Spacer pushes actions to the right */}
            <div className="flex-1" />

            {actions && (
              <div className="flex shrink-0 items-center gap-2">
                {actions}
              </div>
            )}
          </div>
        </div>

        <button
          type="button"
          onClick={handleCloseClick}
          className="shrink-0 self-start rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close page"
          data-testid="detail-nav-button"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* LensBar: cross-lens navigation */}
      {nodeId && lenses.length > 0 && (
        <LensBar
          nodeId={nodeId}
          lenses={lenses}
          onDrillToLens={handleDrillToLens}
          className="border-t border-slate-800/50"
        />
      )}

      {/* Optional tab bar (entity-specific, e.g. backlog info/prompt/files) */}
      {tabBar}
    </header>
  );
}
