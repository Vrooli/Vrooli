/**
 * WorkspaceHeader - the unified top bar shared by the Plan board and the
 * Graph canvas.
 *
 * This is a real, in-flow header row (not a floating overlay): it sits above
 * the active surface as a flex sibling, so neither surface has to reserve
 * dead space to clear it. It deliberately mirrors the sidebar header's
 * treatment — a 40px (`h-10`) row, the same hairline divider, and ghost icon
 * buttons — so the two headers read as one system. Layout:
 *   Left  — sidebar toggle (when collapsed) + Plan/Graph lens nav
 *   Right — stats, graph controls (canvas only), help (lens-aware), and the
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

import { BarChart3, HelpCircle, Menu, Settings } from "lucide-react";
import { OpsTriggerButton } from "../../../components/operations/OpsTriggerButton";
import { LensNav } from "./LensNav";
import { GraphNavControls } from "./GraphNavControls";
import type { AppGraphLens } from "../../../app/routes/route-paths";
import type { GraphLens } from "../stores/graph-data-store";

export interface WorkspaceHeaderProps {
  /** Current active lens */
  lens: GraphLens;
  /** Whether the sidebar is collapsed */
  sidebarCollapsed: boolean;
  /** Whether on-screen pan/zoom nav controls are enabled (graph only) */
  showNavControls: boolean;
  onToggleSidebar: () => void;
  onToggleStats: () => void;
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
  onToggleStats,
  onToggleSettings,
  onToggleHelp,
  onLensChange,
}: WorkspaceHeaderProps) {
  const isPlan = lens === "plan";
  const helpLabel = isPlan ? "Plan guide" : "Graph guide";
  // Nav controls and graph controls are canvas-only affordances; the Plan
  // board has nothing to pan or configure, so neither appears there.
  const showNavRow = showNavControls && !isPlan;

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
          <LensNav activeLens={lens} onLensChange={onLensChange} />
        </div>

        {/* Right: stats + graph controls (canvas only) + help + agents trigger */}
        <div className="ml-auto flex items-center gap-1">
          <button
            type="button"
            onClick={onToggleStats}
            className={ICON_BUTTON_CLASS}
            aria-label="Open stats"
            data-testid="stats-button"
          >
            <BarChart3 className="h-4 w-4" />
          </button>
          {!isPlan && (
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
