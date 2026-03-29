/**
 * GraphWorkspace - Graph-first shell for swarm-manager.
 *
 * Renders: header (title, lens switcher, graph controls, active runs),
 * activity feed, graph canvas, and inspector.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Activity, Settings, Square, X } from "lucide-react";
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
import { LensSwitcher } from "./LensSwitcher";
import { Sidebar } from "./Sidebar";
import { Inspector } from "./Inspector";
import { SettingsDrawer } from "./SettingsDrawer";
import type { GraphLens } from "../stores/graph-data-store";
import type { FeedbackItem, MaturityItem } from "../../../lib/feed";

function isGraphLens(value: string | null): value is GraphLens {
  return value === "topology" || value === "flow" || value === "operations";
}

export function GraphWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showAgentsDropdown, setShowAgentsDropdown] = useState(false);
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);

  const searchLens = searchParams.get("lens");
  const urlLens: GraphLens = isGraphLens(searchLens) ? searchLens : "topology";
  const urlSelect = searchParams.get("select");

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
    if (urlSelect) {
      if (urlSelect !== selectedNodeId) {
        selectNode(urlSelect);
      }
      return;
    }

    if (selectedNodeId) {
      selectNode(null);
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
    <div className="flex h-screen flex-col bg-slate-950 text-slate-50" data-testid="graph-workspace">
      <header
        className="flex h-14 shrink-0 items-center justify-between gap-2 border-b border-slate-200/20 px-3 md:px-4"
        data-testid="graph-header"
      >
        <div className="flex min-w-0 items-center gap-2 md:gap-4">
          <h1 className="hidden whitespace-nowrap text-lg font-semibold md:block">Swarm Manager</h1>
          <LensSwitcher activeLens={lens} onLensChange={handleLensChange} />
        </div>
        <div className="flex shrink-0 items-center gap-1.5 md:gap-2">
          <button
            type="button"
            onClick={() => setShowSettingsDrawer((prev) => !prev)}
            className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800/50 hover:text-slate-200"
            aria-label="Open graph controls"
            data-testid="settings-gear"
          >
            <Settings className="h-5 w-5" />
          </button>
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
      </header>

      <div className="flex flex-1 overflow-hidden">
        <Sidebar feed={feed} onItemClick={handleSidebarItemClick} />
        <div className="relative flex-1">
          <GraphCanvas />
          <Inspector isOpen={inspectorOpen} onClose={handleInspectorClose} selectedNode={selectedNode} />
        </div>
      </div>

      <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
    </div>
  );
}
