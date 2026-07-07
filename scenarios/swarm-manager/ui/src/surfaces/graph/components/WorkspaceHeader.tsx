/**
 * WorkspaceHeader - the unified top bar shared by the Plan board and the
 * Graph canvas.
 *
 * This is a real, in-flow header row (not a floating overlay): it sits above
 * the active surface as a flex sibling, so neither surface has to reserve
 * dead space to clear it. It deliberately mirrors the sidebar header's
 * treatment — a 40px (`h-10`) row, the same hairline divider, and ghost icon
 * buttons — so the two headers read as one system. Layout:
 *   Left  — sidebar toggle (when collapsed) + Plan/Graph/Stats lens nav
 *   Right — graph controls (canvas only), help (lens-aware), and the
 *           Operations trigger
 *
 * The help button opens the guide for the active surface — the Plan Guide on
 * the board, the Graph Guide on the canvas — via the same lens-aware handler
 * wired up in GraphWorkspace. Graph controls only make sense over the canvas,
 * so that button is omitted on the Plan board.
 *
 * The agents button is the always-visible Operations Center trigger; it reads
 * its count from `useOperationsStore`, so the header does not need activity /
 * stop-run plumbing piped down from `GraphWorkspace`.
 */

import { HelpCircle, Menu, RefreshCw, Settings, SlidersHorizontal } from "lucide-react";
import { OpsTriggerButton } from "../../../components/operations/OpsTriggerButton";
import { cn } from "../../../lib/utils";
import { LensNav } from "./LensNav";
import { GraphNavControls } from "./GraphNavControls";
import type { AppGraphLens } from "../../../app/routes/route-paths";
import type { GraphLens } from "../stores/graph-data-store";
import { useOperationsStore } from "../../../stores/operations-store";
import { selectActiveCount } from "../../../stores/operations-store";
import { usePlanDataStore } from "../../plan/stores/plan-data-store";
import { hasActiveFilters } from "../../plan/lib/plan-url-state";

export interface WorkspaceHeaderProps {
  /** Current active surface or graph data lens */
  lens: AppGraphLens | GraphLens;
  /** Whether the sidebar is collapsed */
  sidebarCollapsed: boolean;
  /** Whether on-screen pan/zoom nav controls are enabled (graph only) */
  showNavControls: boolean;
  onToggleSidebar: () => void;
  onToggleSettings: () => void;
  onToggleHelp: () => void;
  onLensChange: (lens: AppGraphLens) => void;
}

/**
 * Ghost icon button — borderless, transparent until hover — matching the
 * buttons in the sidebar header (`SidebarHeader`).
 */
const ICON_BUTTON_CLASS =
  "rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50";

export function WorkspaceHeader({
  lens,
  sidebarCollapsed,
  showNavControls,
  onToggleSidebar,
  onToggleSettings,
  onToggleHelp,
  onLensChange,
}: WorkspaceHeaderProps) {
  const activeSurface: AppGraphLens = lens === "plan" ? "plan" : lens === "stats" ? "stats" : "graph";
  const isGraphSurface = activeSurface === "graph";
  const isPlanSurface = activeSurface === "plan";
  const helpLabel = activeSurface === "plan" ? "Plan guide" : activeSurface === "stats" ? "Stats guide" : "Graph guide";
  // Nav controls and graph controls are canvas-only affordances; Plan and
  // Stats have nothing to pan or configure, so neither appears there.
  const showNavRow = showNavControls && isGraphSurface;
  const activeAgentCount = useOperationsStore(selectActiveCount);
  const planFilters = useOperationsStore((s) => s.filters);
  const planViewMode = useOperationsStore((s) => s.viewMode);
  const planWindowSeconds = usePlanDataStore((s) => s.windowSeconds);
  const planShowSnoozed = usePlanDataStore((s) => s.showSnoozed);
  const planGoal = usePlanDataStore((s) => s.goal);
  const togglePlanFilterDrawer = usePlanDataStore((s) => s.toggleFilterDrawer);
  const refreshPlanBoard = usePlanDataStore((s) => s.fetchBoard);
  const hasActivePlanFilters = hasActiveFilters({
    filters: { ...planFilters, windowSeconds: planWindowSeconds },
    viewMode: planViewMode,
    showSnoozed: planShowSnoozed,
    goal: planGoal,
  });

  return (
    <header
      className="z-20 flex shrink-0 flex-col"
      data-testid="workspace-header"
    >
      <div className="flex h-10 shrink-0 items-center gap-2 border-b border-slate-200/20 px-3">
        {/* Left: sidebar toggle + lens nav */}
        <div className="flex items-center gap-1.5">
          {sidebarCollapsed && (
            <button
              type="button"
              onClick={onToggleSidebar}
              className={ICON_BUTTON_CLASS}
              aria-label="Open sidebar"
              data-testid="sidebar-toggle-open"
            >
              <Menu className="h-4 w-4" />
            </button>
          )}
          <LensNav activeLens={lens} onLensChange={onLensChange} badges={{ plan: activeAgentCount }} />
        </div>

        {/* Right: graph controls (canvas only) + help + agents trigger */}
        <div className="ml-auto flex items-center gap-1">
          {isPlanSurface && (
            <>
              <button
                type="button"
                onClick={togglePlanFilterDrawer}
                className={cn(
                  ICON_BUTTON_CLASS,
                  hasActivePlanFilters && "text-cyan-400 hover:text-cyan-300",
                )}
                aria-label="Plan filters"
                title="Plan filters"
                data-testid="plan-board-filters"
              >
                <SlidersHorizontal className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => void refreshPlanBoard({ force: true })}
                className={ICON_BUTTON_CLASS}
                aria-label="Refresh plan"
                title="Refresh plan"
                data-testid="plan-board-refresh"
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </>
          )}
          {isGraphSurface && (
            <button
              type="button"
              onClick={onToggleSettings}
              className={ICON_BUTTON_CLASS}
              aria-label="Open graph controls"
              data-testid="settings-gear"
            >
              <Settings className="h-4 w-4" />
            </button>
          )}
          <button
            type="button"
            onClick={onToggleHelp}
            className={ICON_BUTTON_CLASS}
            aria-label={helpLabel}
            title={helpLabel}
            data-testid="help-button"
          >
            <HelpCircle className="h-4 w-4" />
          </button>
          {/* Operations Center trigger — the compact sidebar-header pill so it
              reads as a peer of the ghost buttons. On mobile (sidebar collapsed
              behind the menu) it is the operator's primary entry point, so it is
              always shown; on desktop it hides while the sidebar is open because
              the sidebar header already exposes the same pill. */}
          <OpsTriggerButton
            variant="compact"
            className={sidebarCollapsed ? "" : "md:hidden"}
          />
        </div>
      </div>

      {/* Optional second row: on-screen pan/zoom for TV and accessibility. */}
      {showNavRow && (
        <div className="flex items-center border-b border-slate-200/20 px-3 py-2">
          <GraphNavControls />
        </div>
      )}
    </header>
  );
}
