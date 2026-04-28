import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Sidebar } from "../../surfaces/graph/components/Sidebar";
import { SettingsDrawer } from "../../surfaces/graph/components/SettingsDrawer";
import { useGraphUIStore } from "../../surfaces/graph/stores/graph-ui-store";
import { buildActivityNodeId } from "../../surfaces/graph/lib/node-id-parser";
import { buildFeed } from "../../lib/feed";
import { defaultQueryOptions } from "../../lib";
import { settingsService } from "../../services";
import { useAgentActivitiesStore, useBacklogStore, useCaptureStore, useExecutionStore } from "../../stores";
import { useGovernanceStore } from "../../stores/governance-store";
import { useCapturePolling } from "../../hooks/useCapturePolling";
import { useStorePolling } from "../../hooks/useStorePolling";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { AppShellContext } from "./AppShellContext";
import { commandPostPath, detailPathFromNodeId, graphPath } from "../routes/route-paths";
import type { FeedbackItem, MaturityItem } from "../../lib/feed";

export function AppShell() {
  const navigate = useNavigate();
  const shellRef = useRef<HTMLDivElement>(null);
  const isMobile = useIsMobile();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const captures = useCaptureStore((s) => s.captures);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const refreshGovernance = useGovernanceStore((s) => s.refreshGovernance);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setSidebarCollapsed = useGraphUIStore((s) => s.setSidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const { resizeHandleProps, size: sidebarWidth } = useResizablePanel({
    containerRef: shellRef,
    minSize: 260,
    maxSize: 480,
    defaultSize: 320,
    adjacentMinSize: 420,
    handleWidth: 6,
    storageKey: "swarm-manager.sidebar.width.v1",
  });

  useCapturePolling();

  useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

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

  const closeSidebarOnMobile = useCallback(() => {
    if (isMobile) setSidebarCollapsed(true);
  }, [isMobile, setSidebarCollapsed]);

  const navigateToNode = useCallback(
    (nodeId: string, fallbackLens: "topology" | "operations" = "topology") => {
      selectNode(nodeId);
      closeSidebarOnMobile();

      const detailPath = detailPathFromNodeId(nodeId);
      if (detailPath) {
        navigate(detailPath);
        return;
      }

      navigate(graphPath({ lens: fallbackLens, focus: nodeId, select: nodeId }));
    },
    [closeSidebarOnMobile, navigate, selectNode],
  );

  const handleGoHome = useCallback(() => {
    closeSidebarOnMobile();
    navigate(graphPath({ lens: "topology" }));
  }, [closeSidebarOnMobile, navigate]);

  const handleViewActivity = useCallback(
    (activityId: string) => {
      navigateToNode(buildActivityNodeId(activityId), "operations");
    },
    [navigateToNode],
  );

  const handleViewBacklog = useCallback(
    (nodeId: string) => {
      navigateToNode(nodeId, "topology");
    },
    [navigateToNode],
  );

  const shellContext = useMemo(
    () => ({
      openSidebar: () => setSidebarCollapsed(false),
      closeSidebar: () => setSidebarCollapsed(true),
      toggleSidebar,
    }),
    [setSidebarCollapsed, toggleSidebar],
  );

  return (
    <AppShellContext.Provider value={shellContext}>
      <div ref={shellRef} className="flex h-screen min-w-0 bg-slate-950 text-slate-50">
        <Sidebar
          feed={feed}
          onItemClick={(nodeId) => navigateToNode(nodeId)}
          onSettingsOpen={() => setShowSettingsDrawer(true)}
          onViewActivity={handleViewActivity}
          onViewBacklog={handleViewBacklog}
          onGoHome={handleGoHome}
          onOpenCommandPost={() => {
            closeSidebarOnMobile();
            navigate(commandPostPath());
          }}
          desktopWidth={isMobile ? undefined : sidebarWidth}
          resizeHandleProps={isMobile ? undefined : resizeHandleProps}
        />

        <main className="min-w-0 flex-1 overflow-auto">
          <Outlet />
        </main>

        <SettingsDrawer isOpen={showSettingsDrawer} onClose={() => setShowSettingsDrawer(false)} />
      </div>
    </AppShellContext.Provider>
  );
}
