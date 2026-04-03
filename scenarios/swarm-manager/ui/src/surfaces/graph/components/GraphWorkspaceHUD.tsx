/**
 * GraphWorkspaceHUD - Floating controls overlay for the graph workspace.
 *
 * Renders the top HUD rows: sidebar toggle, settings/stats/help/agents buttons,
 * lens navigation, and optional nav controls.
 * Extracted from GraphWorkspace.tsx to reduce component size.
 */

import { BarChart3, HelpCircle, Menu, Settings } from "lucide-react";
import { AgentsDropdown } from "../../../components/agents/AgentsDropdown";
import { CommandPostButton } from "../../../components/command-post";
import { LensNav } from "./LensNav";
import { GraphNavControls } from "./GraphNavControls";
import type { GraphLens } from "../stores/graph-data-store";
import type { AgentActivityRecord } from "../../../stores/agent-activities-store";

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
  /** Agent activities for the dropdown */
  agentActivities: AgentActivityRecord[];
  onToggleSidebar: () => void;
  onToggleCommandPost: () => void;
  onToggleStats: () => void;
  onToggleSettings: () => void;
  onToggleHelp: () => void;
  onLensChange: (lens: GraphLens) => void;
  onReturnToAtlas: () => void;
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
  onStopRun: (runId: string) => void;
}

export function GraphWorkspaceHUD({
  lens,
  focusNodeLabel,
  sidebarCollapsed,
  showNavControls,
  commandPostBadgeCount,
  agentActivities,
  onToggleSidebar,
  onToggleCommandPost,
  onToggleStats,
  onToggleSettings,
  onToggleHelp,
  onLensChange,
  onReturnToAtlas,
  onViewActivity,
  onViewBacklog,
  onStopRun,
}: GraphWorkspaceHUDProps) {
  return (
    <div
      className="pointer-events-auto absolute left-3 right-3 top-3 z-20 flex flex-col gap-1.5"
      data-testid="graph-hud"
    >
      {/* Row 1: Sidebar toggle + Settings/Help/Agents */}
      <div className="flex h-10 items-center justify-between">
        {/* Left: Sidebar toggle (only when collapsed) */}
        {sidebarCollapsed ? (
          <button
            type="button"
            onClick={onToggleSidebar}
            className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
            aria-label="Open sidebar"
            data-testid="sidebar-toggle-open"
          >
            <Menu className="h-4 w-4" />
          </button>
        ) : (
          <div />
        )}

        {/* Right: Command Post + Stats + Settings + Help + Agents */}
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
          {/* Show HUD agents button on mobile always, desktop only when sidebar collapsed */}
          <AgentsDropdown
            activities={agentActivities}
            onViewActivity={onViewActivity}
            onViewBacklog={onViewBacklog}
            onStopRun={(runId) => void onStopRun(runId)}
            variant="button"
            className={sidebarCollapsed ? "" : "md:hidden"}
          />
        </div>
      </div>

      {/* Row 2: Lens navigation */}
      <LensNav
        activeLens={lens}
        focusNodeLabel={focusNodeLabel}
        onLensChange={onLensChange}
        onReturnToAtlas={onReturnToAtlas}
      />

      {/* Row 3: On-screen pan/zoom for TV and accessibility (toggled via Settings) */}
      {showNavControls && <GraphNavControls />}
    </div>
  );
}
