/**
 * SidebarHeader — Top bar for the graph sidebar.
 *
 * Shows the app title, a home button (returns to graph view), a compact
 * running-agents badge, a settings gear, and a collapse/close button.
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-3
 */

import { Home, PanelLeft, Settings, X } from "lucide-react";
import { useAgentActivitiesStore } from "../../../../stores";
import { useDetailSelectionStore } from "../../../../stores/detail-selection-store";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { AgentsDropdown } from "../../../../components/agents/AgentsDropdown";

export interface SidebarHeaderProps {
  onSettingsOpen: () => void;
  onCollapse: () => void;
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
}

export function SidebarHeader({
  onSettingsOpen,
  onCollapse,
  onViewActivity,
  onViewBacklog,
}: SidebarHeaderProps) {
  const activities = useAgentActivitiesStore((s) => s.activities);
  const stopRun = useAgentActivitiesStore((s) => s.stopRun);
  const clearSelection = useDetailSelectionStore((s) => s.clearSelection);
  const setSidebarCollapsed = useGraphUIStore((s) => s.setSidebarCollapsed);

  const handleGoHome = () => {
    clearSelection();
    setSidebarCollapsed(true);
  };

  return (
    <div className="flex shrink-0 items-center justify-between border-b border-slate-200/20 px-3 py-2">
      {/* Left: Home button + App title */}
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={handleGoHome}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Go to graph view"
          data-testid="sidebar-home"
        >
          <Home className="h-4 w-4" />
        </button>
        <span className="text-sm font-semibold text-slate-200">Swarm Manager</span>
      </div>

      {/* Right: Agents badge + Settings + Collapse/Close */}
      <div className="flex items-center gap-1">
        <AgentsDropdown
          activities={activities}
          onViewActivity={onViewActivity}
          onViewBacklog={onViewBacklog}
          onStopRun={(runId) => void stopRun(runId)}
          variant="badge"
        />
        <button
          type="button"
          onClick={onSettingsOpen}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Settings"
          data-testid="sidebar-settings"
        >
          <Settings className="h-4 w-4" />
        </button>
        <button
          type="button"
          onClick={onCollapse}
          className="rounded p-1.5 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          aria-label="Collapse sidebar"
          data-testid="sidebar-toggle-close"
        >
          <X className="h-4 w-4 md:hidden" />
          <PanelLeft className="hidden h-4 w-4 md:block" />
        </button>
      </div>
    </div>
  );
}
