/**
 * DetailPageHeader
 *
 * Unified header for all entity detail pages. Two-row layout:
 *
 * Row 1: nav button + title (nearly full width for readability)
 * Row 2: status badge + subtitle + primary action + overflow menu
 *
 * The action contract is deliberately narrow so the row can never overflow:
 * at most ONE always-visible primary action; everything else goes in
 * `menuActions`, rendered as an ellipsis menu (a bottom sheet on mobile).
 * Entity-type icons and metadata chips live in the body's "Overview"
 * section, not here.
 *
 * Also provides:
 * - Integrated LensBar for cross-lens navigation
 * - Optional tab bar slot for entity-specific tabs (e.g., backlog info/prompt/files)
 */

import { type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Menu, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { graphPath } from "../../app/routes/route-paths";
import { useAppBack } from "../../app/routes/useAppBack";
import { useAppShell } from "../../app/shell/AppShellContext";
import { useGraphUIStore } from "../../surfaces/graph/stores/graph-ui-store";
import { ActionMenu, type ActionMenuItem } from "../ui/action-menu";
import { StatusBadge } from "./StatusBadge";
import { TitlePopover } from "./TitlePopover";
import { LensBar } from "./LensBar";
import type { LensOption } from "./lens-options";
import type { AppGraphLens } from "../../app/routes/route-paths";
import type { BacklogStatus } from "../../types";

export interface DetailPageHeaderProps {
  /** Human label for the entity kind — names the actions menu ("Backlog actions"). */
  entityType: string;
  title: string;
  subtitle?: string;
  status?: string;
  /** Graph node ID for cross-lens navigation. */
  nodeId: string | null;
  /** Available lens navigation options. */
  lenses: LensOption[];
  /** The single always-visible action button. Anything else belongs in `menuActions`. */
  primaryAction?: ReactNode;
  /** Secondary actions, shown in an ellipsis menu (bottom sheet on mobile). */
  menuActions?: ActionMenuItem[];
  /** Optional testid overrides for the actions menu trigger/panel. */
  menuTriggerTestId?: string;
  menuTestId?: string;
  /** Optional tab bar rendered below the LensBar (e.g., backlog tabs). */
  tabBar?: ReactNode;
  /**
   * Overrides the default lens navigation (graph focus/select deep link).
   * Goal pages use this to scope the plan board via ?goal= instead.
   */
  onDrillToLens?: (nodeId: string, lens: AppGraphLens) => void;
  /** When provided, the status badge becomes clickable for inline status changes. */
  onStatusChange?: (newStatus: BacklogStatus) => void;
  /** Whether a status change is in flight. */
  statusChangePending?: boolean;
  className?: string;
}

export function DetailPageHeader({
  entityType,
  title,
  subtitle,
  status,
  nodeId,
  lenses,
  primaryAction,
  menuActions,
  menuTriggerTestId = "detail-header-actions",
  menuTestId = "detail-header-actions-menu",
  tabBar,
  onStatusChange,
  statusChangePending,
  onDrillToLens,
  className,
}: DetailPageHeaderProps) {
  const goBack = useAppBack();
  const navigate = useNavigate();
  const { openSidebar } = useAppShell();
  // Hide the header hamburger when the sidebar is already visible — its
  // own collapse toggle becomes the single way to dismiss it. Default to
  // `true` (button visible) so we never miss the affordance during the
  // first render before the persisted store finishes hydration.
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed ?? true);

  const handleCloseClick = () => {
    goBack();
  };

  const handleDrillToLens = onDrillToLens ?? ((id: string, lens: AppGraphLens) => {
    navigate(graphPath({ lens, focus: id, select: id }));
  });

  const hasMenu = (menuActions?.length ?? 0) > 0;
  const showActionRow = Boolean(status || subtitle || primaryAction || hasMenu);

  return (
    <header className={cn("border-b border-slate-800", className)} data-testid="detail-page-header">
      {/* Menu button (left, only when the sidebar is collapsed) + two-row content + route close (right) */}
      <div className="flex items-center gap-3 px-4 py-3 md:px-6">
        {sidebarCollapsed && (
          <button
            type="button"
            onClick={openSidebar}
            className="shrink-0 self-center rounded-lg p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
            aria-label="Open sidebar"
            data-testid="page-sidebar-button"
          >
            <Menu className="h-5 w-5" />
          </button>
        )}

        <div className="min-w-0 flex-1">
          {/* Row 1: title (click to view full title + copy) */}
          <h1 className="min-w-0 text-lg font-semibold text-slate-100">
            <TitlePopover
              title={title}
              className="block w-full min-w-0 max-w-full cursor-pointer rounded text-slate-100 transition-colors hover:text-blue-300"
            />
          </h1>

          {/* Row 2: status + subtitle + primary action + overflow menu */}
          {showActionRow && (
            <div className="mt-1 flex items-center gap-2">
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

              {/* Spacer pushes actions to the right */}
              <div className="flex-1" />

              {primaryAction && (
                <div className="flex shrink-0 items-center">{primaryAction}</div>
              )}

              {hasMenu && (
                <ActionMenu
                  items={menuActions ?? []}
                  label={`${entityType} actions`}
                  mobileSheet
                  triggerTestId={menuTriggerTestId}
                  menuTestId={menuTestId}
                  className="h-7 w-7 shrink-0"
                />
              )}
            </div>
          )}
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
