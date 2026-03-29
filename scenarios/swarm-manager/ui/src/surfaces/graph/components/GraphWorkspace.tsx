/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Uses HUD-like floating controls instead of a header toolbar:
 * - Top-center: LensNav
 * - Top-right: Settings gear, Help button, Agents dropdown
 * - Bottom-left: Edge legend (rendered inside GraphCanvas)
 * - Bottom-right: MiniMap (rendered inside GraphCanvas)
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Activity, HelpCircle, Settings, Square, X } from "lucide-react";
import { defaultQueryOptions, formatRelativeTime } from "../../../lib";
import { applyTheme, watchSystemTheme } from "../../../lib/theme-utils";
import { buildFeed } from "../../../lib/feed";
import { settingsService } from "../../../services";
import { useAgentActivitiesStore, useBacklogStore, useCaptureStore } from "../../../stores";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { buildActivityNodeId, buildBacklogNodeId } from "../lib/node-id-parser";
import { useGraphKeyboardShortcuts } from "../hooks/useGraphKeyboardShortcuts";
import { useGraphWebSocket } from "../hooks/useGraphWebSocket";
import { GraphCanvas } from "./GraphCanvas";
import { LensNav } from "./LensNav";
import { Sidebar } from "./Sidebar";
import { Inspector } from "./Inspector";
import { SettingsDrawer } from "./SettingsDrawer";
import { GraphHelpPanel } from "./GraphHelpPanel";
import { getGraphNodeLabel } from "../types";
import type { GraphLens } from "../stores/graph-data-store";
import type { FeedbackItem, MaturityItem } from "../../../lib/feed";

function isGraphLens(value: string | null): value is GraphLens {
  return value === "topology" || value === "flow" || value === "operations";
}

export function GraphWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showAgentsDropdown, setShowAgentsDropdown] = useState(false);
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);
  const [showHelpPanel, setShowHelpPanel] = useState(false);

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
  const inspectorOpen = useGraphUIStore((s) => s.inspectorOpen);
  const setInspectorOpen = useGraphUIStore((s) => s.setInspectorOpen);
  const applyLayoutForLens = useGraphUIStore((s) => s.applyLayoutForLens);

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

  useEffect(() => {
    void refreshActivities(true);
    const timer = window.setInterval(() => void refreshActivities(true), 5000);
    return () => window.clearInterval(timer);
  }, [refreshActivities]);

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

  const handleDrillToFlow = useCallback(
    (nodeId: string) => {
      const node = nodes.find((n) => n.id === nodeId);
      if (node) setFocusNodeLabel(getGraphNodeLabel(node));
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", "flow");
        next.set("focus", nodeId);
        next.set("returnLens", lens);
        next.delete("select");
        return next;
      });
    },
    [lens, nodes, setFocusNodeLabel, setSearchParams],
  );

  const handleDrillToOperations = useCallback(
    (nodeId: string) => {
      const node = nodes.find((n) => n.id === nodeId);
      if (node) setFocusNodeLabel(getGraphNodeLabel(node));
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", "operations");
        next.set("focus", nodeId);
        next.set("returnLens", lens);
        next.delete("select");
        return next;
      });
    },
    [lens, nodes, setFocusNodeLabel, setSearchParams],
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
    },
    [selectNode, setSearchParams],
  );

  const selectedNode = useMemo(() => {
    if (!selectedNodeId) return null;
    return nodes.find((node) => node.id === selectedNodeId) ?? null;
  }, [nodes, selectedNodeId]);

  const handleInspectorClose = useCallback(() => {
    setInspectorOpen(false);
    selectNode(null);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("select");
      return next;
    });
  }, [selectNode, setInspectorOpen, setSearchParams]);

  const sortedActiveActivities = useMemo(() => {
    return [...agentActivities].sort(
      (a, b) => new Date(b.requestedAt).getTime() - new Date(a.requestedAt).getTime(),
    );
  }, [agentActivities]);

  useGraphKeyboardShortcuts({
    onLensChange: handleLensChange,
    onInspectorClose: handleInspectorClose,
    onSettingsToggle: () => setShowSettingsDrawer((prev) => !prev),
    onReturnToAtlas: handleReturnToAtlas,
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

  return (
    <div className="flex h-screen bg-slate-950 text-slate-50" data-testid="graph-workspace">
      {/* Sidebar (activity feed) */}
      <Sidebar feed={feed} onItemClick={handleSidebarItemClick} />

      {/* Main canvas area with HUD overlays */}
      <div className="relative flex-1">
        <GraphCanvas />

        {/* HUD bar — single unified row at top */}
        <div
          className="pointer-events-auto absolute left-3 right-3 top-3 z-20 flex items-center justify-between gap-2"
          data-testid="graph-hud"
        >
          {/* Left: Lens navigation */}
          <LensNav
            activeLens={lens}
            focusNodeId={focusNodeId}
            focusNodeLabel={focusNodeLabel}
            onLensChange={handleLensChange}
            onReturnToAtlas={handleReturnToAtlas}
          />

          {/* Right: Settings + Help + Agents */}
          <div className="flex items-center gap-1.5">
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
            <div className="relative">
              <button
                type="button"
                className="flex items-center gap-1.5 rounded-lg border border-slate-700/60 bg-slate-900/80 px-2.5 py-1.5 text-sm text-slate-100 hover:bg-slate-800/80"
                onClick={() => setShowAgentsDropdown((prev) => !prev)}
                data-testid="graph-agents-toggle"
              >
                <Activity className="h-4 w-4 text-cyan-300" />
                <span className="rounded-full bg-cyan-500/20 px-1.5 py-0.5 text-xs text-cyan-200">
                  {sortedActiveActivities.length}
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
                        <p className="text-xs text-slate-400">{sortedActiveActivities.length} active activity item(s)</p>
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
                      {sortedActiveActivities.length === 0 ? (
                        <p className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-6 text-center text-sm text-slate-400">
                          No agents are currently running.
                        </p>
                      ) : (
                        <div className="space-y-2">
                          {sortedActiveActivities.map((activity) => {
                            const backlogNodeId =
                              activity.ownerType === "backlog" &&
                              typeof activity.ownerKind === "string" &&
                              typeof activity.ownerName === "string"
                                ? buildBacklogNodeId(activity.ownerKind, activity.ownerName)
                                : null;

                            return (
                              <div key={activity.activityId} className="rounded-lg border border-slate-800 bg-slate-900/45 p-3">
                                <div className="flex items-start justify-between gap-2">
                                  <div className="min-w-0">
                                    <p className="truncate text-sm font-medium text-slate-100">
                                      {activity.ownerTitle ?? `${activity.ownerType}/${activity.ownerName}`}
                                    </p>
                                    {activity.runId && (
                                      <p className="font-mono text-xs text-cyan-300">{activity.runId}</p>
                                    )}
                                  </div>
                                  <span className="rounded-full bg-cyan-500/15 px-2 py-0.5 text-[11px] text-cyan-200">
                                    {activity.status.replace("_", " ")}
                                  </span>
                                </div>
                                <p className="mt-1 text-xs text-slate-400">
                                  {activity.purpose.replace("_", " ")} • Requested {formatRelativeTime(activity.requestedAt)}
                                </p>
                                <div className="mt-2 flex items-center gap-2">
                                  <button
                                    type="button"
                                    className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                                    onClick={() => {
                                      handleLensChange("operations");
                                      handleSidebarItemClick(buildActivityNodeId(activity.activityId));
                                      setShowAgentsDropdown(false);
                                    }}
                                  >
                                    View Activity
                                  </button>
                                  {backlogNodeId && (
                                    <button
                                      type="button"
                                      className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                                      onClick={() => {
                                        handleLensChange("topology");
                                        handleSidebarItemClick(backlogNodeId);
                                        setShowAgentsDropdown(false);
                                      }}
                                    >
                                      View Backlog
                                    </button>
                                  )}
                                  {activity.runId && (
                                    <button
                                      type="button"
                                      className="rounded border border-red-500/40 bg-red-500/10 px-2 py-1 text-xs text-red-200 hover:bg-red-500/20"
                                      onClick={() => void stopRun(activity.runId ?? "")}
                                      disabled={activity.isStopping}
                                    >
                                      <Square className="mr-1 inline h-3 w-3" />
                                      {activity.isStopping ? "Stopping..." : "Stop"}
                                    </button>
                                  )}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>

        {/* Floating panels */}

        <GraphHelpPanel isOpen={showHelpPanel} onClose={() => setShowHelpPanel(false)} />
        <Inspector
          isOpen={inspectorOpen}
          onClose={handleInspectorClose}
          selectedNode={selectedNode}
          onDrillToFlow={handleDrillToFlow}
          onDrillToOperations={handleDrillToOperations}
        />
      </div>

      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
  );
}
