/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Uses HUD-like floating controls instead of a header toolbar:
 * - Top-center: Plan/Graph navigation
 * - Top-right: Settings gear, Help button, Agents dropdown
 * - Bottom-left: Edge legend (rendered inside GraphCanvas)
 * - Bottom-right: MiniMap (rendered inside GraphCanvas)
 */

import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { settingsService } from "../../../services";
import { useAgentSessionStore } from "../../../stores";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphSettingsStore } from "../stores/graph-settings-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphStateSync } from "../hooks/useGraphStateSync";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { useQueryClient } from "@tanstack/react-query";
import { GraphCanvas } from "./GraphCanvas";
import { CapturePanel } from "./CapturePanel";
import { PlanBoard } from "../../plan";
import { ClarificationPanel } from "../../../components/backlog/clarification-panel";
import { useSpatialNav } from "../../../hooks/useSpatialNav";
import { SpatialGroup } from "../../../hooks/SpatialGroup";
import { SpatialNavProvider } from "../../../hooks/SpatialNavContext";

import { SettingsDrawer } from "./SettingsDrawer";
import { StatsPanel } from "./StatsPanel";
import { NodeInspectorPanel } from "./NodeInspectorPanel";
import { GraphHelpPanel } from "./GraphHelpPanel";
import { CanvasErrorBoundary } from "./CanvasErrorBoundary";
import { GraphWorkspaceHUD } from "./GraphWorkspaceHUD";
import { GraphActionLauncher } from "./GraphActionLauncher";
import { graphPath, sessionDetailPath } from "../../../app/routes/route-paths";
import { useAppShell } from "../../../app/shell/AppShellContext";
import type { AgentSessionKind } from "../../../types";
import { SESSION_CREATE_TITLES } from "../../../components/session/session-view-model";

export function GraphWorkspace() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [showStatsPanel, setShowStatsPanel] = useState(false);
  const [showHelpPanel, setShowHelpPanel] = useState(false);
  const [showCapturePanel, setShowCapturePanel] = useState(false);
  const [launcherError, setLauncherError] = useState<string | null>(null);
  const [launcherStatus, setLauncherStatus] = useState<string | null>(null);

  const { openSidebar } = useAppShell();

  // --- Graph state sync (URL ↔ store) ---
  const { urlLens: _urlLens, handleLensChange, handleReturnToAtlas, handleDeselectNode } = useGraphStateSync();

  const createSession = useAgentSessionStore((s) => s.createSession);
  const isCreatingSession = useAgentSessionStore((s) => s.isMutating);
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);

  const lens = useGraphDataStore((s) => s.lens);
  const setNodePulsing = useGraphDataStore((s) => s.setNodePulsing);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);

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

  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onDeselectNode: handleDeselectNode,
    onSettingsToggle: () => setShowSettingsDrawer((prev) => !prev),
    onReturnToAtlas: handleReturnToAtlas,
    // Command Post is retired: the shortcut opens the board's decision drawer.
    onToggleCommandPost: () => navigate(`${graphPath({ lens: "plan" })}?drawer=decisions`),
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

  const handleCreateAgentSession = useCallback(
    async (kind: AgentSessionKind) => {
      if (isCreatingSession) return;
      setLauncherError(null);
      setLauncherStatus("Starting session...");
      setShowCapturePanel(false);
      try {
        const session = await createSession({
          kind,
          title: SESSION_CREATE_TITLES[kind],
        });
        navigate(sessionDetailPath(session.id));
      } catch (error) {
        setLauncherError(error instanceof Error ? error.message : "Unable to start agent session.");
      } finally {
        setLauncherStatus(null);
      }
    },
    [createSession, isCreatingSession, navigate],
  );

  return (
    <SpatialNavProvider controllerRef={spatialNav}>
    <div className="flex h-screen bg-slate-950 text-slate-50" data-testid="graph-workspace">
      {/* Main canvas area with HUD overlays */}
      <div className="relative flex-1">
        {/* Plan renders the kanban board; graph modes render the node/edge
            canvas. Both sit under the same HUD. */}
        {lens === "plan" ? (
          <PlanBoard />
        ) : (
          <SpatialGroup controllerRef={spatialNav} mode="passthrough">
            <CanvasErrorBoundary>
              <GraphCanvas />
            </CanvasErrorBoundary>
          </SpatialGroup>
        )}

        {/* HUD — two rows at top */}
        <GraphWorkspaceHUD
          lens={lens}
          sidebarCollapsed={sidebarCollapsed}
          showNavControls={showNavControls}
          onToggleSidebar={openSidebar}
          onToggleStats={() => setShowStatsPanel((prev) => !prev)}
          onToggleSettings={() => setShowSettingsDrawer((prev) => !prev)}
          onToggleHelp={() => setShowHelpPanel((prev) => !prev)}
          onLensChange={handleLensChange}
        />

        {/* Floating panels */}

        <NodeInspectorPanel />
        <GraphHelpPanel isOpen={showHelpPanel} onClose={() => setShowHelpPanel(false)} />

        <GraphActionLauncher
          isBusy={isCreatingSession}
          error={launcherError}
          status={launcherStatus}
          onDismissError={() => setLauncherError(null)}
          onQuickCapture={() => {
            setLauncherError(null);
            setLauncherStatus(null);
            setShowCapturePanel((prev) => !prev);
          }}
          onPlanWork={() => void handleCreateAgentSession("meta_orchestration")}
          onManageSwarm={() => void handleCreateAgentSession("swarm_operations")}
          onAuthorOperatingMode={() => void handleCreateAgentSession("operating_mode_authoring")}
        />

        <CapturePanel isOpen={showCapturePanel} onClose={() => setShowCapturePanel(false)} />

        {/* Single workspace mount for the clarification thread (workshop
            questions answered from the Plan board's decision drawer). */}
        <ClarificationPanel
          onAction={(action) => {
            if (
              action === "invalidate_round" ||
              action === "remove_decision" ||
              action === "update_decision"
            ) {
              void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
            }
          }}
        />
      </div>

      <StatsPanel isOpen={showStatsPanel} onClose={() => setShowStatsPanel(false)} />
      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
    </SpatialNavProvider>
  );
}
