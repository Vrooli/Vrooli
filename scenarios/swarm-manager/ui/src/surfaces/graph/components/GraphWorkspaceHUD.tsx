/**
 * GraphWorkspaceHUD - Floating controls overlay for the graph workspace.
 *
 * Renders the top HUD rows: sidebar toggle, settings/stats/help/agents
 * buttons, lens navigation, and optional nav controls.
 *
 * The agents button is the always-visible Operations Center trigger; it
 * reads its count from `useOperationsStore`, so the HUD does not need
 * activity / stop-run plumbing piped down from `GraphWorkspace`.
 */

import { BarChart3, HelpCircle, Menu, Settings } from "lucide-react";
import { CommandPostButton } from "../../../components/command-post";
import { OpsTriggerButton } from "../../../components/operations/OpsTriggerButton";
import { LensNav } from "./LensNav";
import { GraphNavControls } from "./GraphNavControls";
import type { GraphLens } from "../stores/graph-data-store";

export interface GraphWorkspaceHUDProps {
  /** Current active lens */
  lens: GraphLens;
  /** Label for the focused node (breadcrumb display) */
  focusNodeLabel: string | null;
  /** Whether the sidebar is collapsed */
  sidebarCollapsed: boolean;
  /** Whether on-screen nav controls are enabled */
  showNavControls: boolean;
  /** Badge count for command post button */
  commandPostBadgeCount: number;
  onToggleSidebar: () => void;
  onToggleCommandPost: () => void;
  onToggleStats: () => void;
  onToggleSettings: () => void;
  onToggleHelp: () => void;
  onLensChange: (lens: GraphLens) => void;
  onReturnToAtlas: () => void;
}

export function GraphWorkspaceHUD({
  lens,
  focusNodeLabel,
  sidebarCollapsed,
  showNavControls,
  commandPostBadgeCount,
  onToggleSidebar,
  onToggleCommandPost,
  onToggleStats,
  onToggleSettings,
  onToggleHelp,
  onLensChange,
  onReturnToAtlas,
}: GraphWorkspaceHUDProps) {
  return (
    <div
      className="pointer-events-auto absolute left-3 right-3 top-3 z-20 flex flex-col gap-1.5"
      data-testid="graph-hud"
    >
      {/* Row 1: Sidebar toggle + (lg: LensNav) + Settings/Help/Agents */}
      <div className="flex min-h-[2.5rem] items-start justify-between">
        {/* Left: Sidebar toggle + lens nav on large screens */}
        <div className="flex items-start gap-1.5">
          {sidebarCollapsed && (
            <button
              type="button"
              onClick={onToggleSidebar}
              className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
              aria-label="Open sidebar"
              data-testid="sidebar-toggle-open"
            >
              <Menu className="h-4 w-4" />
            </button>
          )}
          {/* Lens nav inline on large screens */}
          <div className="hidden lg:block">
            <LensNav
              activeLens={lens}
              focusNodeLabel={focusNodeLabel}
              onLensChange={onLensChange}
              onReturnToAtlas={onReturnToAtlas}
            />
          </div>
        </div>

        {/* Right: Command Post + Stats + Settings + Help + Operations trigger */}
        <div className="flex items-center gap-1.5">
          <CommandPostButton
            count={commandPostBadgeCount}
            onClick={onToggleCommandPost}
          />
          <button
            type="button"
            onClick={onToggleStats}
            className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
            aria-label="Open stats"
            data-testid="stats-button"
          >
            <BarChart3 className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={onToggleSettings}
            className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
            aria-label="Open graph controls"
            data-testid="settings-gear"
          >
            <Settings className="h-4 w-4" />
          </button>
          <button
            type="button"
            onClick={onToggleHelp}
            className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
            aria-label="Graph help"
            data-testid="help-button"
          >
            <HelpCircle className="h-4 w-4" />
          </button>
          {/* Operations Center trigger.
              On mobile (sidebar always collapsed behind the menu) the HUD
              trigger is the operator's primary entry point, so it is
              always shown. On desktop it hides while the sidebar is open
              because the sidebar header already exposes the compact pill. */}
          <OpsTriggerButton
            variant="hud"
            className={sidebarCollapsed ? "" : "md:hidden"}
          />
        </div>
      </div>

      {/* Row 2: Lens navigation (small/medium screens only — inlined into row 1 on lg) */}
      <div className="lg:hidden">
        <LensNav
          activeLens={lens}
          focusNodeLabel={focusNodeLabel}
          onLensChange={onLensChange}
          onReturnToAtlas={onReturnToAtlas}
        />
      </div>

      {/* Row 3: On-screen pan/zoom for TV and accessibility (toggled via Settings) */}
      {showNavControls && <GraphNavControls />}
    </div>
  );
}
