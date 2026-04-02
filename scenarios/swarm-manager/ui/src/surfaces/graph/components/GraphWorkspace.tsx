/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Uses HUD-like floating controls instead of a header toolbar:
 * - Top-center: LensNav
 * - Top-right: Settings gear, Help button, Agents dropdown
 * - Bottom-left: Edge legend (rendered inside GraphCanvas)
 * - Bottom-right: MiniMap (rendered inside GraphCanvas)
 */

import { Component, lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, BarChart3, HelpCircle, Menu, MessageSquarePlus, RefreshCw, Settings } from "lucide-react";
import { ErrorDiagnostics } from "../../../components/ui/error-diagnostics";
import { categorizeError, generateUniqueId } from "../../../lib/error-utils";
import { defaultQueryOptions } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { buildFeed } from "../../../lib/feed";
import { settingsService } from "../../../services";
import { useAgentActivitiesStore, useBacklogStore, useCaptureStore } from "../../../stores";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { buildActivityNodeId, parseNodeId } from "../lib/node-id-parser";
import { clearVisualFocus } from "../lib/visual-focus";
import { useDetailSelectionStore } from "../../../stores/detail-selection-store";
import { useDetailUrlSync } from "../../../hooks/useDetailUrlSync";
import { useDetailNavigation } from "../../../hooks/useDetailNavigation";
import { AgentsDropdown } from "../../../components/agents/AgentsDropdown";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { FloatingActionButton } from "../../../components/ui/floating-action-button";
import { PageLoadingState } from "../../../components/ui/loading-states";
import { useCapturePolling } from "../../../hooks/useCapturePolling";
import { useStorePolling } from "../../../hooks/useStorePolling";

import { GraphCanvas } from "./GraphCanvas";
import { GraphNavControls } from "./GraphNavControls";
import { CapturePanel } from "./CapturePanel";

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
import { LensNav } from "./LensNav";
import { Sidebar } from "./Sidebar";
import { SettingsDrawer } from "./SettingsDrawer";
import { StatsPanel } from "./StatsPanel";
import { NodeInspectorPanel } from "./NodeInspectorPanel";
import { GraphHelpPanel } from "./GraphHelpPanel";
import { getGraphNodeLabel } from "../types";
import type { GraphLens } from "../stores/graph-data-store";
import type { FeedbackItem, MaturityItem } from "../../../lib/feed";

function isGraphLens(value: string | null): value is GraphLens {
  return value === "topology" || value === "flow" || value === "operations";
}

/** Canvas error boundary prefix for correlation IDs */
const CANVAS_ERROR_ID_PREFIX = "canvas_err";

interface CanvasErrorBoundaryState {
  hasError: boolean;
  errorId: string | null;
  error: Error | null;
  componentStack: string | null;
  timestamp: string | null;
}

/** Error boundary that wraps only the graph canvas, keeping HUD + sidebar alive. */
class CanvasErrorBoundary extends Component<
  { children: ReactNode },
  CanvasErrorBoundaryState
> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false, errorId: null, error: null, componentStack: null, timestamp: null };
  }

  static getDerivedStateFromError(error: Error): CanvasErrorBoundaryState {
    return {
      hasError: true,
      errorId: generateUniqueId(CANVAS_ERROR_ID_PREFIX),
      error,
      componentStack: null,
      timestamp: new Date().toISOString(),
    };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[GraphCanvas] Render crash:", error.message, info.componentStack);
    this.setState({ componentStack: info.componentStack ?? null });
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-full w-full flex-col items-center justify-center gap-4 overflow-y-auto p-4 text-slate-400">
          <AlertTriangle className="h-10 w-10 text-amber-400" />
          <p className="text-sm">Graph canvas encountered an error.</p>
          <button
            type="button"
            className="flex items-center gap-1.5 rounded-lg border border-slate-600 bg-slate-800 px-3 py-1.5 text-xs text-slate-200 hover:bg-slate-700"
            onClick={() => this.setState({ hasError: false, errorId: null, error: null, componentStack: null, timestamp: null })}
          >
            <RefreshCw className="h-3.5 w-3.5" />
            Retry
          </button>
          {this.state.error && this.state.timestamp && (
            <ErrorDiagnostics
              error={this.state.error}
              componentStack={this.state.componentStack}
              errorId={this.state.errorId}
              category={categorizeError(this.state.error)}
              timestamp={this.state.timestamp}
              compact
            />
          )}
        </div>
      );
    }
    return this.props.children;
  }
}

export function GraphWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [showStatsPanel, setShowStatsPanel] = useState(false);
  const [showHelpPanel, setShowHelpPanel] = useState(false);
  const [showCapturePanel, setShowCapturePanel] = useState(false);

  const searchLens = searchParams.get("lens");
  const urlLens: GraphLens = isGraphLens(searchLens) ? searchLens : "topology";
  const urlSelect = searchParams.get("select");
  const urlFocus = searchParams.get("focus");
  const urlReturnLens = searchParams.get("returnLens");

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const captures = useCaptureStore((s) => s.captures);
  const agentActivities = useAgentActivitiesStore((s) => s.activities);
  const stopRun = useAgentActivitiesStore((s) => s.stopRun);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const lens = useGraphDataStore((s) => s.lens);
  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);
  const nodes = useGraphDataStore((s) => s.nodes);
  const setLens = useGraphDataStore((s) => s.setLens);
  const setNodePulsing = useGraphDataStore((s) => s.setNodePulsing);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);
  const setFocusNode = useGraphDataStore((s) => s.setFocusNode);
  const returnLens = useGraphDataStore((s) => s.returnLens);
  const setReturnLens = useGraphDataStore((s) => s.setReturnLens);
  const focusNodeLabel = useGraphUIStore((s) => s.focusNodeLabel);
  const setFocusNodeLabel = useGraphUIStore((s) => s.setFocusNodeLabel);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const applyLayoutForLens = useGraphUIStore((s) => s.applyLayoutForLens);

  const showNavControls = useGraphDataStore((s) => s.settingsByLens[s.lens].showNavControls);

  const detailSelection = useDetailSelectionStore((s) => s.selection);
  const { openDetail } = useDetailNavigation();

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
  }, [fetchBacklog, fetchCaptures]);

  useStorePolling({
    enabled: true,
    intervalMs: 5000,
    pollFn: () => void refreshActivities(true),
    immediate: true,
  });

  useEffect(() => {
    setLens(urlLens);
    applyLayoutForLens(urlLens);
    void fetchGraph(urlLens);
  }, [applyLayoutForLens, fetchGraph, setLens, urlLens]);

  useEffect(() => {
    setFocusNode(urlFocus ?? null);
    setReturnLens(isGraphLens(urlReturnLens) ? urlReturnLens : null);
  }, [urlFocus, urlReturnLens, setFocusNode, setReturnLens]);

  useEffect(() => {
    if (!focusNodeId) {
      setFocusNodeLabel(null);
      return;
    }
    const node = nodes.find((n) => n.id === focusNodeId);
    if (node) {
      setFocusNodeLabel(getGraphNodeLabel(node));
    }
  }, [focusNodeId, nodes, setFocusNodeLabel]);

  // Sync URL → store only on URL-driven changes. Canvas clicks update the
  // store directly without touching the URL, so we must not deselect when
  // urlSelect is absent — that would race with the canvas click handler.
  const prevUrlSelect = useRef(urlSelect);
  useEffect(() => {
    if (urlSelect === prevUrlSelect.current) {
      return;
    }
    prevUrlSelect.current = urlSelect;

    if (urlSelect) {
      if (urlSelect !== selectedNodeId) {
        selectNode(urlSelect);
      }
    } else {
      if (selectedNodeId) {
        selectNode(null);
      }
    }
  }, [selectedNodeId, selectNode, urlSelect]);

  const handleLensChange = useCallback(
    (newLens: GraphLens) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", newLens);
        return next;
      });
    },
    [setSearchParams],
  );

  const handleReturnToAtlas = useCallback(() => {
    const target = returnLens ?? "topology";
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("lens", target);
      next.delete("focus");
      next.delete("returnLens");
      return next;
    });
  }, [returnLens, setSearchParams]);

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

  const handleDeselectNode = useCallback(() => {
    const cleared = clearVisualFocus();
    selectNode(cleared.selectedNodeId);
    setHighlightState(cleared.highlightState);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("select");
      return next;
    });
  }, [selectNode, setHighlightState, setSearchParams]);

  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onDeselectNode: handleDeselectNode,
    onSettingsToggle: () => setShowSettingsDrawer((prev) => !prev),
    onReturnToAtlas: handleReturnToAtlas,
    focusNodeId,
  });

  // Sync store → URL when selection is cleared (e.g., by pane click in GraphCanvas).
  const prevSelectedNodeId = useRef(selectedNodeId);
  useEffect(() => {
    const prev = prevSelectedNodeId.current;
    prevSelectedNodeId.current = selectedNodeId;
    if (prev !== null && selectedNodeId === null) {
      setSearchParams((p) => {
        if (!p.has("select")) return p;
        const next = new URLSearchParams(p);
        next.delete("select");
        return next;
      });
    }
  }, [selectedNodeId, setSearchParams]);

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
    <div className="flex h-screen bg-slate-950 text-slate-50" data-testid="graph-workspace">
      {/* Sidebar (activity feed) */}
      <Sidebar
        feed={feed}
        onItemClick={handleSidebarItemClick}
        onSettingsOpen={() => setShowSettingsDrawer(true)}
        onViewActivity={handleViewActivity}
        onViewBacklog={handleViewBacklog}
      />

      {/* Main canvas area with HUD overlays */}
      <div className="relative flex-1">
        <CanvasErrorBoundary>
          <GraphCanvas />
        </CanvasErrorBoundary>

        {/* HUD — two rows at top */}
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
                onClick={toggleSidebar}
                className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
                aria-label="Open sidebar"
                data-testid="sidebar-toggle-open"
              >
                <Menu className="h-4 w-4" />
              </button>
            ) : (
              <div />
            )}

            {/* Right: Stats + Settings + Help + Agents */}
            <div className="flex items-center gap-1.5">
              <button
                type="button"
                onClick={() => setShowStatsPanel((prev) => !prev)}
                className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
                aria-label="Open stats"
                data-testid="stats-button"
              >
                <BarChart3 className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => setShowSettingsDrawer((prev) => !prev)}
                className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
                aria-label="Open graph controls"
                data-testid="settings-gear"
              >
                <Settings className="h-4 w-4" />
              </button>
              <button
                type="button"
                onClick={() => setShowHelpPanel((prev) => !prev)}
                className="rounded-lg border border-slate-700/60 bg-slate-900/80 p-2 text-slate-400 transition-colors hover:bg-slate-800/80 hover:text-slate-200"
                aria-label="Graph help"
                data-testid="help-button"
              >
                <HelpCircle className="h-4 w-4" />
              </button>
              {/* Show HUD agents button on mobile always, desktop only when sidebar collapsed */}
              <AgentsDropdown
                activities={agentActivities}
                onViewActivity={handleViewActivity}
                onViewBacklog={handleViewBacklog}
                onStopRun={(runId) => void stopRun(runId)}
                variant="button"
                className={sidebarCollapsed ? "" : "md:hidden"}
              />
            </div>
          </div>

          {/* Row 2: Lens navigation */}
          <LensNav
            activeLens={lens}
            focusNodeId={focusNodeId}
            focusNodeLabel={focusNodeLabel}
            onLensChange={handleLensChange}
            onReturnToAtlas={handleReturnToAtlas}
          />

          {/* Row 3: On-screen pan/zoom for TV and accessibility (toggled via Settings) */}
          {showNavControls && <GraphNavControls />}
        </div>

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

        {/* Detail page overlay — full-page, covers graph when active */}
        {detailSelection && (
          <div className="absolute inset-0 z-40 overflow-y-auto bg-slate-950" data-testid="detail-overlay">
            <Suspense fallback={<PageLoadingState label="Loading details..." />}>
              {detailSelection.entityType === "backlog" && <BacklogDetailsPage />}
              {detailSelection.entityType === "scenario" && <ScenarioDetailsPage />}
              {detailSelection.entityType === "execution" && <ExecutionDetailsPage />}
              {detailSelection.entityType === "initiative" && <InitiativeDetailsPage />}
            </Suspense>
          </div>
        )}
      </div>

      <StatsPanel isOpen={showStatsPanel} onClose={() => setShowStatsPanel(false)} />
      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
  );
}
