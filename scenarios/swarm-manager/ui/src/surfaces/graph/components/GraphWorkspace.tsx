/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Uses HUD-like floating controls instead of a header toolbar:
 * - Top-center: LensNav
 * - Top-right: Settings gear, Help button, Agents dropdown
 * - Bottom-left: Edge legend (rendered inside GraphCanvas)
 * - Bottom-right: MiniMap (rendered inside GraphCanvas)
 */

import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { MessageSquarePlus } from "lucide-react";
import { defaultQueryOptions } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { buildFeed } from "../../../lib/feed";
import { settingsService } from "../../../services";
import { useAgentActivitiesStore, useBacklogStore, useCaptureStore, useExecutionStore } from "../../../stores";
import { useGovernanceStore } from "../../../stores/governance-store";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphSettingsStore } from "../stores/graph-settings-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { buildActivityNodeId, parseNodeId } from "../lib/node-id-parser";
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
import { useDetailUrlSync } from "../../../hooks/useDetailUrlSync";
import { useDetailNavigation } from "../../../hooks/useDetailNavigation";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphStateSync } from "../hooks/useGraphStateSync";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";
import { PageLoadingState } from "../../../components/ui/loading-states";
import { useCapturePolling } from "../../../hooks/useCapturePolling";
import { useStorePolling } from "../../../hooks/useStorePolling";

import { GraphCanvas } from "./GraphCanvas";
import { CapturePanel } from "./CapturePanel";
import { CommandPostOverlay } from "../../../components/command-post";
import { useCommandPostBadgeCount } from "../../../hooks/useCommandPostBadgeCount";
import { useSpatialNav } from "../../../hooks/useSpatialNav";
import { SpatialGroup } from "../../../hooks/SpatialGroup";
import { SpatialNavProvider } from "../../../hooks/SpatialNavContext";

const BacklogDetailsPage = lazy(() =>
  import("../../../pages/BacklogDetailsPage").then((m) => ({
    default: m.BacklogDetailsPage,
  })),
);
const ScenarioDetailsPage = lazy(() =>
  import("../../../pages/ScenarioDetailsPage").then((m) => ({
    default: m.ScenarioDetailsPage,
  })),
);
const ExecutionDetailsPage = lazy(() =>
  import("../../../pages/ExecutionDetailsPage").then((m) => ({
    default: m.ExecutionDetailsPage,
  })),
);
const InitiativeDetailsPage = lazy(() =>
  import("../../../pages/InitiativeDetailsPage").then((m) => ({
    default: m.InitiativeDetailsPage,
  })),
);
const CaptureDetailsPage = lazy(() =>
  import("../../../pages/CaptureDetailsPage").then((m) => ({
    default: m.CaptureDetailsPage,
  })),
);
import { Sidebar } from "./Sidebar";
import { SettingsDrawer } from "./SettingsDrawer";
import { StatsPanel } from "./StatsPanel";
import { NodeInspectorPanel } from "./NodeInspectorPanel";
import { GraphHelpPanel } from "./GraphHelpPanel";
import { CanvasErrorBoundary } from "./CanvasErrorBoundary";
import { GraphWorkspaceHUD } from "./GraphWorkspaceHUD";
import type { GraphLens } from "../stores/graph-data-store";
import type { FeedbackItem, MaturityItem } from "../../../lib/feed";

export function GraphWorkspace() {
  const [_searchParams, setSearchParams] = useSearchParams();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [showStatsPanel, setShowStatsPanel] = useState(false);
  const [showHelpPanel, setShowHelpPanel] = useState(false);
  const [showCapturePanel, setShowCapturePanel] = useState(false);
  const [showCommandPost, setShowCommandPost] = useState(false);

  const commandPostBadgeCount = useCommandPostBadgeCount();

  // --- Graph state sync (URL ↔ store) ---
  const { urlLens: _urlLens, handleLensChange, handleReturnToAtlas, handleDeselectNode } = useGraphStateSync();

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const captures = useCaptureStore((s) => s.captures);
  const agentActivities = useAgentActivitiesStore((s) => s.activities);
  const stopRun = useAgentActivitiesStore((s) => s.stopRun);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const refreshGovernance = useGovernanceStore((s) => s.refreshGovernance);
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const lens = useGraphDataStore((s) => s.lens);
  const setNodePulsing = useGraphDataStore((s) => s.setNodePulsing);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);
  const focusNodeLabel = useGraphUIStore((s) => s.focusNodeLabel);
  const selectNode = useGraphUIStore((s) => s.selectNode);

  const showNavControls = useGraphSettingsStore((s) => s.settingsByLens[s.activeLens].showNavControls);

  const detailSelection = useDetailSelectionStore((s) => s.selection);
  const { openDetail } = useDetailNavigation();

  const spatialNav = useSpatialNav();

  useDetailUrlSync();
  useCapturePolling();

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

  useEffect(() => {
    void fetchBacklog();
    void fetchCaptures();
    void fetchExecutions();
  }, [fetchBacklog, fetchCaptures, fetchExecutions]);

  useStorePolling({
    enabled: true,
    intervalMs: 5000,
    pollFn: () => void refreshActivities(true),
    immediate: true,
  });

  useStorePolling({
    enabled: true,
    intervalMs: 15000,
    pollFn: () => void refreshGovernance(),
    immediate: true,
  });

  const feed = useMemo(() => {
    const feedbackItems: FeedbackItem[] = [];
    const maturityItems: MaturityItem[] = [];
    return buildFeed(captures, backlogItems, feedbackItems, maturityItems);
  }, [captures, backlogItems]);

  const handleSidebarItemClick = useCallback(
    (nodeId: string) => {
      selectNode(nodeId);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("select", nodeId);
        return next;
      });

      const parsed = parseNodeId(nodeId);
      if (parsed) {
        const selection = (() => {
          switch (parsed.entityType) {
            case "backlog":
              return parsed.kind && parsed.name
                ? { entityType: "backlog" as const, kind: parsed.kind, name: parsed.name }
                : null;
            case "scenario":
              return parsed.name ? { entityType: "scenario" as const, name: parsed.name } : null;
            case "execution":
              return { entityType: "execution" as const, identifier: parsed.identifier };
            case "initiative":
              return parsed.name ? { entityType: "initiative" as const, name: parsed.name } : null;
            default:
              return null;
          }
        })();

        if (selection) {
          openDetail(selection, { fromSidebar: !sidebarCollapsed });
        }
      }
    },
    [selectNode, setSearchParams, openDetail, sidebarCollapsed],
  );

  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onDeselectNode: handleDeselectNode,
    onSettingsToggle: () => setShowSettingsDrawer((prev) => !prev),
    onReturnToAtlas: handleReturnToAtlas,
    onToggleCommandPost: () => setShowCommandPost((prev) => !prev),
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
      {/* Sidebar (activity feed) — spatial nav for list items */}
      <SpatialGroup controllerRef={spatialNav} mode="spatial">
        <Sidebar
          feed={feed}
          onItemClick={handleSidebarItemClick}
          onSettingsOpen={() => setShowSettingsDrawer(true)}
          onViewActivity={handleViewActivity}
          onViewBacklog={handleViewBacklog}
          onOpenCommandPost={() => setShowCommandPost(true)}
        />
      </SpatialGroup>

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
          onToggleSidebar={toggleSidebar}
          onToggleCommandPost={() => setShowCommandPost((prev) => !prev)}
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

        {/* Capture FAB — hidden when detail overlay is open */}
        {!detailSelection && (
          <FloatingActionButton
            icon={<MessageSquarePlus className="h-5 w-5" />}
            label="New capture"
            onClick={() => setShowCapturePanel((prev) => !prev)}

            data-testid="capture-fab"
          />
        )}

        <CapturePanel isOpen={showCapturePanel} onClose={() => setShowCapturePanel(false)} />

        {/* Command Post overlay */}
        <CommandPostOverlay
          isOpen={showCommandPost}
          onClose={() => setShowCommandPost(false)}
          onNavigateToDetail={(selection) => {
            setShowCommandPost(false);
            openDetail(selection, {});
          }}
          onSwitchLens={(lens) => {
            setShowCommandPost(false);
            handleLensChange(lens as GraphLens);
          }}
        />

        {/* Detail page overlay — full-page, covers graph when active */}
        {detailSelection && (
          <SpatialGroup controllerRef={spatialNav} mode="spatial">
            <div className="absolute inset-0 z-40 overflow-y-auto bg-slate-950" data-testid="detail-overlay">
              <Suspense fallback={<PageLoadingState label="Loading details..." />}>
                {detailSelection.entityType === "backlog" && <BacklogDetailsPage />}
                {detailSelection.entityType === "scenario" && <ScenarioDetailsPage />}
                {detailSelection.entityType === "execution" && <ExecutionDetailsPage />}
                {detailSelection.entityType === "initiative" && <InitiativeDetailsPage />}
                {detailSelection.entityType === "capture" && <CaptureDetailsPage />}
              </Suspense>
            </div>
          </SpatialGroup>
        )}
      </div>

      <StatsPanel isOpen={showStatsPanel} onClose={() => setShowStatsPanel(false)} />
      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
    </SpatialNavProvider>
  );
}
