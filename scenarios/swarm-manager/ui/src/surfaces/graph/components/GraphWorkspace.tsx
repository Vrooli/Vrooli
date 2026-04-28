/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Uses HUD-like floating controls instead of a header toolbar:
 * - Top-center: LensNav
 * - Top-right: Settings gear, Help button, Agents dropdown
 * - Bottom-left: Edge legend (rendered inside GraphCanvas)
 * - Bottom-right: MiniMap (rendered inside GraphCanvas)
 */

import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { MessageSquarePlus } from "lucide-react";
import { defaultQueryOptions } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { settingsService } from "../../../services";
import { useAgentActivitiesStore } from "../../../stores";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphSettingsStore } from "../stores/graph-settings-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { buildActivityNodeId } from "../lib/node-id-parser";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphStateSync } from "../hooks/useGraphStateSync";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";

import { GraphCanvas } from "./GraphCanvas";
import { CapturePanel } from "./CapturePanel";
import { useCommandPostBadgeCount } from "../../../hooks/useCommandPostBadgeCount";
import { useSpatialNav } from "../../../hooks/useSpatialNav";
import { SpatialGroup } from "../../../hooks/SpatialGroup";
import { SpatialNavProvider } from "../../../hooks/SpatialNavContext";

import { SettingsDrawer } from "./SettingsDrawer";
import { StatsPanel } from "./StatsPanel";
import { NodeInspectorPanel } from "./NodeInspectorPanel";
import { GraphHelpPanel } from "./GraphHelpPanel";
import { CanvasErrorBoundary } from "./CanvasErrorBoundary";
import { GraphWorkspaceHUD } from "./GraphWorkspaceHUD";
import { commandPostPath, detailPathFromNodeId } from "../../../app/routes/route-paths";
import { useAppShell } from "../../../app/shell/AppShellContext";

export function GraphWorkspace() {
  const navigate = useNavigate();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [showStatsPanel, setShowStatsPanel] = useState(false);
  const [showHelpPanel, setShowHelpPanel] = useState(false);
  const [showCapturePanel, setShowCapturePanel] = useState(false);

  const commandPostBadgeCount = useCommandPostBadgeCount();
  const { openSidebar } = useAppShell();

  // --- Graph state sync (URL ↔ store) ---
  const { urlLens: _urlLens, handleLensChange, handleReturnToAtlas, handleDeselectNode } = useGraphStateSync();

  const agentActivities = useAgentActivitiesStore((s) => s.activities);
  const stopRun = useAgentActivitiesStore((s) => s.stopRun);
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);

  const lens = useGraphDataStore((s) => s.lens);
  const setNodePulsing = useGraphDataStore((s) => s.setNodePulsing);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);
  const focusNodeLabel = useGraphUIStore((s) => s.focusNodeLabel);
  const selectNode = useGraphUIStore((s) => s.selectNode);

  const showNavControls = useGraphSettingsStore((s) => s.settingsByLens[s.activeLens].showNavControls);

  const spatialNav = useSpatialNav();

  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    const theme = settings?.theme ?? "dark";
    applyTheme(theme);
    if (theme === "system") {
      return watchSystemTheme(() => applyTheme("system"));
    }
    return undefined;
  }, [settings?.theme]);

  const handleSidebarItemClick = useCallback(
    (nodeId: string) => {
      selectNode(nodeId);

      const detailPath = detailPathFromNodeId(nodeId);
      if (detailPath) navigate(detailPath);
    },
    [navigate, selectNode],
  );

  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onDeselectNode: handleDeselectNode,
    onSettingsToggle: () => setShowSettingsDrawer((prev) => !prev),
    onReturnToAtlas: handleReturnToAtlas,
    onToggleCommandPost: () => navigate(commandPostPath()),
    focusNodeId,
  });

  const handleNodePulse = useCallback(
    (nodeId: string) => {
      setNodePulsing(nodeId, true);
      window.setTimeout(() => {
        setNodePulsing(nodeId, false);
      }, 2000);
    },
    [setNodePulsing],
  );

  useGraphWebSocket({
    enabled: true,
    lens,
    onNodePulse: handleNodePulse,
  });

  const handleViewActivity = useCallback(
    (activityId: string) => {
      handleLensChange("operations");
      handleSidebarItemClick(buildActivityNodeId(activityId));
    },
    [handleLensChange, handleSidebarItemClick],
  );

  const handleViewBacklog = useCallback(
    (nodeId: string) => {
      handleLensChange("topology");
      handleSidebarItemClick(nodeId);
    },
    [handleLensChange, handleSidebarItemClick],
  );

  return (
    <SpatialNavProvider controllerRef={spatialNav}>
    <div className="flex h-screen bg-slate-950 text-slate-50" data-testid="graph-workspace">
      {/* Main canvas area with HUD overlays */}
      <div className="relative flex-1">
        {/* Graph canvas — passthrough for panning/zooming */}
        <SpatialGroup controllerRef={spatialNav} mode="passthrough">
          <CanvasErrorBoundary>
            <GraphCanvas />
          </CanvasErrorBoundary>
        </SpatialGroup>

        {/* HUD — two rows at top */}
        <GraphWorkspaceHUD
          lens={lens}
          focusNodeLabel={focusNodeLabel}
          sidebarCollapsed={sidebarCollapsed}
          showNavControls={showNavControls}
          commandPostBadgeCount={commandPostBadgeCount}
          agentActivities={agentActivities}
          maxConcurrent={settings?.maxConcurrentExecutions}
          onToggleSidebar={openSidebar}
          onToggleCommandPost={() => navigate(commandPostPath())}
          onToggleStats={() => setShowStatsPanel((prev) => !prev)}
          onToggleSettings={() => setShowSettingsDrawer((prev) => !prev)}
          onToggleHelp={() => setShowHelpPanel((prev) => !prev)}
          onLensChange={handleLensChange}
          onReturnToAtlas={handleReturnToAtlas}
          onViewActivity={handleViewActivity}
          onViewBacklog={handleViewBacklog}
          onStopRun={(runId) => void stopRun(runId)}
        />

        {/* Floating panels */}

        <NodeInspectorPanel />
        <GraphHelpPanel isOpen={showHelpPanel} onClose={() => setShowHelpPanel(false)} />

        <FloatingActionButton
          icon={<MessageSquarePlus className="h-5 w-5" />}
          label="New capture"
          onClick={() => setShowCapturePanel((prev) => !prev)}
          data-testid="capture-fab"
        />

        <CapturePanel isOpen={showCapturePanel} onClose={() => setShowCapturePanel(false)} />
      </div>

      <StatsPanel isOpen={showStatsPanel} onClose={() => setShowStatsPanel(false)} />
      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
    </SpatialNavProvider>
  );
}
