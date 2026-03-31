/**
 * DetailPageHeader
 *
 * Unified header for all entity detail pages. Provides:
 * - Mobile: hamburger (Menu) to open sidebar; Desktop: back arrow (ArrowLeft) to close detail
 * - Entity type badge
 * - Title and optional subtitle
 * - Status badge
 * - Action slot for entity-specific buttons
 * - Integrated LensBar for cross-lens navigation
 * - Optional tab bar slot for entity-specific tabs (e.g., backlog info/prompt/files)
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-2
 */

import { type ReactNode } from "react";
import { ArrowLeft, Menu } from "lucide-react";
import { cn } from "../../lib/utils";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { useDetailNavigation } from "../../hooks/useDetailNavigation";
import { StatusBadge } from "./StatusBadge";
import { LensBar } from "./LensBar";
import type { LensOption } from "./lens-options";
import type { GraphLens } from "../../surfaces/graph/stores/graph-data-store";

export interface DetailPageHeaderProps {
  entityType: string;
  title: string;
  subtitle?: string;
  status?: string;
  /** Graph node ID for cross-lens navigation. */
  nodeId: string | null;
  /** Available lens navigation options. */
  lenses: LensOption[];
  /** Entity-specific action buttons (rendered inline on desktop). */
  actions?: ReactNode;
  /** Optional tab bar rendered below the LensBar (e.g., backlog tabs). */
  tabBar?: ReactNode;
  className?: string;
}

export function DetailPageHeader({
  entityType,
  title,
  subtitle,
  status,
  nodeId,
  lenses,
  actions,
  tabBar,
  className,
}: DetailPageHeaderProps) {
  const isMobile = useIsMobile();
  const { closeDetail, openSidebar, drillToLens } = useDetailNavigation();

  const handleNavClick = () => {
    if (isMobile) {
      openSidebar();
    } else {
      closeDetail();
    }
  };

  const handleDrillToLens = (id: string, lens: GraphLens) => {
    drillToLens(id, lens);
  };

  return (
    <header className={cn("border-b border-slate-800", className)} data-testid="detail-page-header">
      {/* Primary row: nav button, type badge, title, status, actions */}
      <div className="flex items-center gap-3 px-4 py-3 md:px-6">
        <button
          type="button"
          onClick={handleNavClick}
          className="rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label={isMobile ? "Open sidebar" : "Close detail view"}
          data-testid="detail-nav-button"
        >
          {isMobile ? (
            <Menu className="h-5 w-5" />
          ) : (
            <ArrowLeft className="h-5 w-5" />
          )}
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
