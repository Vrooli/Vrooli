/**
 * GraphWorkspace - Main layout component replacing MainLayout.
 *
 * Renders: header (title, lens switcher, gear, agents dropdown) + sidebar + canvas + inspector.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Activity, Settings, Square, X } from "lucide-react";
import { cn, defaultQueryOptions, formatRelativeTime } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { buildFeed } from "../../../lib/feed";
import { settingsService } from "../../../services";
import {
  useAgentRunsStore,
  useBacklogStore,
  useCaptureStore,
  useExecutionStore,
  useScenariosStore,
} from "../../../stores";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { assembleGraphData } from "../lib/graph-assembler";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { GraphCanvas } from "./GraphCanvas";
import { LensSwitcher } from "./LensSwitcher";
import { Sidebar } from "./Sidebar";
import { Inspector } from "./Inspector";
import { SettingsDrawer } from "./SettingsDrawer";
import type { GraphLens } from "../stores/graph-data-store";
import type { FeedbackItem, MaturityItem } from "../../../lib/feed";

export function GraphWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showAgentsDropdown, setShowAgentsDropdown] = useState(false);
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);

  // URL-driven state.
  const urlLens = (searchParams.get("lens") as GraphLens) ?? "topology";
  const urlSelect = searchParams.get("select");

  // Existing stores.
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchScenarios = useScenariosStore((s) => s.fetchScenarios);
  const scenarios = useScenariosStore((s) => s.scenarios);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const executions = useExecutionStore((s) => s.items);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const captures = useCaptureStore((s) => s.captures);
  const agentRuns = useAgentRunsStore((s) => s.runs);
  const stopRun = useAgentRunsStore((s) => s.stopRun);
  const refreshActiveRuns = useAgentRunsStore((s) => s.refreshActiveRuns);

  // Graph stores.
  const lens = useGraphDataStore((s) => s.lens);
  const setLens = useGraphDataStore((s) => s.setLens);
  const setGraphData = useGraphDataStore((s) => s.setGraphData);
  const nodes = useGraphDataStore((s) => s.nodes);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const inspectorOpen = useGraphUIStore((s) => s.inspectorOpen);
  const setInspectorOpen = useGraphUIStore((s) => s.setInspectorOpen);

  // Settings (for theme).
  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  // Theme application.
  useEffect(() => {
    const theme = settings?.theme ?? "dark";
    applyTheme(theme);
    if (theme === "system") {
      return watchSystemTheme(() => applyTheme("system"));
    }
    return undefined;
  }, [settings?.theme]);

  // Data fetching.
  useEffect(() => {
    void fetchBacklog();
    void fetchScenarios();
    void fetchExecutions();
    void fetchCaptures();
  }, [fetchBacklog, fetchScenarios, fetchExecutions, fetchCaptures]);

  // Agent run polling.
  useEffect(() => {
    void refreshActiveRuns();
    const timer = window.setInterval(() => void refreshActiveRuns(), 5000);
    return () => window.clearInterval(timer);
  }, [refreshActiveRuns]);

  // Sync URL lens to store.
  useEffect(() => {
    if (urlLens !== lens) {
      setLens(urlLens);
    }
  }, [urlLens, lens, setLens]);

  // Sync URL select to store.
  useEffect(() => {
    if (urlSelect && urlSelect !== selectedNodeId) {
      selectNode(urlSelect);
    }
  }, [urlSelect, selectedNodeId, selectNode]);

  // Assemble graph data from existing stores.
  useEffect(() => {
    const { nodes: assembledNodes, edges } = assembleGraphData(
      backlogItems,
      scenarios,
      executions,
      captures,
      agentRuns,
    );
    setGraphData(assembledNodes, edges);
  }, [backlogItems, scenarios, executions, captures, agentRuns, setGraphData]);

  // Lens change handler.
  const handleLensChange = useCallback(
    (newLens: GraphLens) => {
      setLens(newLens);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", newLens);
        return next;
      });
    },
    [setLens, setSearchParams],
  );

  // Feed data (for sidebar).
  const feed = useMemo(() => {
    const feedbackItems: FeedbackItem[] = [];
    const maturityItems: MaturityItem[] = [];
    return buildFeed(captures, backlogItems, feedbackItems, maturityItems);
  }, [captures, backlogItems]);

  // Handle sidebar item click.
  const handleSidebarItemClick = useCallback(
    (nodeId: string) => {
      selectNode(nodeId);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("select", nodeId);
        return next;
      });
    },
    [selectNode, setSearchParams],
  );

  // Find selected node object.
  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;
    return nodes.find((n) => n.id === selectedNodeId) ?? null;
  }, [selectedNodeId, nodes]);

  // Handle inspector close.
  const handleInspectorClose = useCallback(() => {
    setInspectorOpen(false);
    selectNode(null);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("select");
      return next;
    });
  }, [setInspectorOpen, selectNode, setSearchParams]);

  // Active agent runs for dropdown.
  const sortedActiveRuns = useMemo(() => {
    return [...agentRuns]
      .filter((run) => ["pending", "starting", "running", "needs_review"].includes(run.status))
      .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
  }, [agentRuns]);

  const formatDuration = (seconds?: number): string => {
    if (!seconds || seconds <= 0) return "Unknown";
    if (seconds < 60) return `${Math.round(seconds)}s`;
    const minutes = Math.floor(seconds / 60);
    const remainder = Math.round(seconds % 60);
    if (minutes < 60) return `${minutes}m ${remainder}s`;
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    return `${hours}h ${remainingMinutes}m`;
  };

  // Keyboard shortcuts.
  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onInspectorClose: handleInspectorClose,
    onSettingsToggle: () => setShowSettingsDrawer((v) => !v),
  });

  // WebSocket: connect only when Operations lens is active.
  const handleNodePulse = useCallback((nodeId: string) => {
    // Set pulsing flag on node data — GraphNode reads this to apply the CSS animation class.
    useGraphDataStore.setState((state) => ({
      nodes: state.nodes.map((n) =>
        n.id === nodeId ? { ...n, data: { ...n.data, pulsing: true } } : n,
      ),
    }));
    // Clear pulsing flag after the animation duration (2s).
    window.setTimeout(() => {
      useGraphDataStore.setState((state) => ({
        nodes: state.nodes.map((n) =>
          n.id === nodeId ? { ...n, data: { ...n.data, pulsing: false } } : n,
        ),
      }));
    }, 2000);
  }, []);

  useGraphWebSocket({
    enabled: lens === "operations",
    onNodePulse: handleNodePulse,
  });

  return (
    <div className="flex h-screen flex-col bg-slate-950 text-slate-50" data-testid="graph-workspace">
      {/* Header */}
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-slate-200/20 px-3 md:px-4 gap-2" data-testid="graph-header">
        <div className="flex items-center gap-2 md:gap-4 min-w-0">
          <h1 className="hidden md:block text-lg font-semibold whitespace-nowrap">Swarm Manager</h1>
          <LensSwitcher activeLens={lens} onLensChange={handleLensChange} />
        </div>
        <div className="flex items-center gap-1.5 md:gap-2 shrink-0">
          {/* Settings gear */}
          <button
            type="button"
            onClick={() => setShowSettingsDrawer((v) => !v)}
            className="rounded-lg p-2 text-slate-400 hover:text-slate-200 hover:bg-slate-800/50 transition-colors"
            aria-label="Open settings"
            data-testid="settings-gear"
          >
            <Settings className="h-5 w-5" />
          </button>
          {/* Agent-run dropdown */}
          <div className="relative">
            <button
              type="button"
              className="flex items-center gap-2 rounded-lg border border-slate-700/80 bg-slate-900/45 px-3 py-1.5 text-sm text-slate-100 hover:bg-slate-800/70"
              onClick={() => setShowAgentsDropdown((prev) => !prev)}
              data-testid="graph-agents-toggle"
            >
              <Activity className="h-4 w-4 text-cyan-300" />
              <span className="hidden sm:inline">Agents</span>
              <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-200">
                {sortedActiveRuns.length}
              </span>
            </button>
            {showAgentsDropdown && (
              <>
                <button
                  type="button"
                  className="fixed inset-0 z-40 cursor-default bg-transparent"
                  aria-label="Close agents dropdown"
                  onClick={() => setShowAgentsDropdown(false)}
                />
                <div
                  className="absolute right-0 top-12 z-50 w-[calc(100vw-2rem)] max-w-[360px] rounded-lg border border-slate-700/80 bg-slate-950 shadow-xl"
                  data-testid="graph-agents-dropdown"
                >
                  <div className="flex items-center justify-between border-b border-slate-800 px-3 py-2">
                    <div>
                      <p className="text-sm font-semibold text-slate-100">Agents running</p>
                      <p className="text-xs text-slate-400">{sortedActiveRuns.length} active run(s)</p>
                    </div>
                    <button
                      type="button"
                      className="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                      onClick={() => setShowAgentsDropdown(false)}
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                  <div className="max-h-80 overflow-y-auto p-2">
                    {sortedActiveRuns.length === 0 ? (
                      <p className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-6 text-center text-sm text-slate-400">
                        No agents are currently running.
                      </p>
                    ) : (
                      <div className="space-y-2">
                        {sortedActiveRuns.map((run) => (
                          <div key={run.runId} className="rounded-lg border border-slate-800 bg-slate-900/45 p-3">
                            <div className="flex items-start justify-between gap-2">
                              <div className="min-w-0">
                                <p className="truncate text-sm font-medium text-slate-100">
                                  {run.backlogTitle ?? `${run.backlogKind}/${run.backlogName}`}
                                </p>
                                <p className="font-mono text-xs text-cyan-300">{run.runId}</p>
                              </div>
                              <span className="rounded-full bg-cyan-500/15 px-2 py-0.5 text-[11px] text-cyan-200">
                                {run.status.replace("_", " ")}
                              </span>
                            </div>
                            <p className="mt-1 text-xs text-slate-400">
                              Spawned {formatRelativeTime(run.createdAt)} • Duration {formatDuration(run.durationSeconds)}
                            </p>
                            <div className="mt-2 flex items-center gap-2">
                              {run.backlogKind && run.backlogName && (
                                <button
                                  type="button"
                                  className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                                  onClick={() => {
                                    handleSidebarItemClick(`${run.backlogKind}/${run.backlogName}`);
                                    setShowAgentsDropdown(false);
                                  }}
                                >
                                  Select
                                </button>
                              )}
                              <button
                                type="button"
                                className="rounded border border-red-500/40 bg-red-500/10 px-2 py-1 text-xs text-red-200 hover:bg-red-500/20"
                                onClick={() => void stopRun(run.runId)}
                                disabled={run.isStopping}
                              >
                                <Square className="mr-1 inline h-3 w-3" />
                                {run.isStopping ? "Stopping..." : "Stop"}
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </header>

      {/* Body: sidebar + canvas */}
      <div className="flex flex-1 overflow-hidden">
        <Sidebar feed={feed} onItemClick={handleSidebarItemClick} />
        <div className="relative flex-1">
          <GraphCanvas />
          <Inspector
            isOpen={inspectorOpen}
            onClose={handleInspectorClose}
            selectedNode={selectedNode}
          />
        </div>
      </div>

      {/* Settings drawer */}
      <SettingsDrawer
        isOpen={showSettingsDrawer}
        onClose={() => setShowSettingsDrawer(false)}
      />
    </div>
  );
}
