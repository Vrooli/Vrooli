import { Profiler, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Outlet, useNavigate } from "react-router-dom";
import { onProfilerRender } from "../../lib/profiler";
import { useQuery } from "@tanstack/react-query";
import { Sidebar } from "../../surfaces/graph/components/Sidebar";
import { SettingsDrawer } from "../../surfaces/graph/components/SettingsDrawer";
import { useGraphUIStore } from "../../surfaces/graph/stores/graph-ui-store";
import { buildFeed } from "../../lib/feed";
import { defaultQueryOptions } from "../../lib";
import { settingsService } from "../../services";
import { useAgentActivitiesStore, useAgentSessionStore, useBacklogStore, useCaptureStore, useExecutionStore } from "../../stores";
import { useGovernanceStore } from "../../stores/governance-store";
import { useOperationsStore } from "../../stores/operations-store";
import { useAgentSessionPolling } from "../../hooks/useAgentSessionPolling";
import { useCapturePolling } from "../../hooks/useCapturePolling";
import { useStorePolling } from "../../hooks/useStorePolling";
import { useResizablePanel } from "../../hooks/useResizablePanel";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { BacklogItemsProvider } from "../../components/backlog/backlog-items-context";
import { AppShellContext } from "./AppShellContext";
import { detailPathFromNodeId, graphPath } from "../routes/route-paths";
import type { FeedbackItem, MaturityItem } from "../../lib/feed";

export function AppShell() {
  const navigate = useNavigate();
  const shellRef = useRef<HTMLDivElement>(null);
  const sidebarAsideRef = useRef<HTMLElement>(null);
  const isMobile = useIsMobile();
  const [showSettingsDrawer, setShowSettingsDrawer] = useState(false);

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const captures = useCaptureStore((s) => s.captures);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const refreshGovernance = useGovernanceStore((s) => s.refreshGovernance);
  const refreshOperations = useOperationsStore((s) => s.refresh);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setSidebarCollapsed = useGraphUIStore((s) => s.setSidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const { resizeHandleProps, size: sidebarWidth } = useResizablePanel({
    containerRef: shellRef,
    targetRef: sidebarAsideRef,
    minSize: 260,
    maxSize: 480,
    defaultSize: 320,
    adjacentMinSize: 420,
    handleWidth: 6,
    storageKey: "swarm-manager.sidebar.width.v1",
  });

  useCapturePolling();
  useAgentSessionPolling();

  useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  useEffect(() => {
    void fetchBacklog();
    void fetchCaptures();
    void fetchExecutions();
    void fetchSessions(undefined, { force: true });
  }, [fetchBacklog, fetchCaptures, fetchExecutions, fetchSessions]);

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

  // P8 — keep the Operations Center trigger pill (sidebar header + graph
  // HUD) live with the latest agent count. The Operations Center page
  // mounts its own faster polling via `useOperationsPolling`; the page's
  // internal serialization makes the dual-poll a no-op while the page is
  // open. 8s is slow enough to keep idle traffic light yet fast enough
  // that operators see new activity within ~one tick of it spawning.
  useStorePolling({
    enabled: true,
    intervalMs: 8000,
    pollFn: () => void refreshOperations(),
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
    (nodeId: string, fallbackLens: "plan" | "focus" = "focus") => {
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
    navigate(graphPath({ lens: "plan" }));
  }, [closeSidebarOnMobile, navigate]);

  const handleSidebarItemClick = useCallback(
    (nodeId: string) => navigateToNode(nodeId),
    [navigateToNode],
  );

  const handleOpenSettings = useCallback(() => setShowSettingsDrawer(true), []);

  const handleOpenCommandPost = useCallback(() => {
    closeSidebarOnMobile();
    // Command Post is retired; decisions live in the Plan board's drawer.
    navigate(`${graphPath({ lens: "plan" })}?drawer=decisions`);
  }, [closeSidebarOnMobile, navigate]);

  const handleCloseSettingsDrawer = useCallback(() => setShowSettingsDrawer(false), []);

  const shellContext = useMemo(
    () => ({
      openSidebar: () => setSidebarCollapsed(false),
      closeSidebar: () => setSidebarCollapsed(true),
      toggleSidebar,
    }),
    [setSidebarCollapsed, toggleSidebar],
  );

  return (
    <Profiler id="AppShell" onRender={onProfilerRender}>
      <AppShellContext.Provider value={shellContext}>
        <BacklogItemsProvider items={backlogItems}>
          <div ref={shellRef} className="flex h-screen min-w-0 bg-slate-950 text-slate-50">
            <Profiler id="Sidebar" onRender={onProfilerRender}>
              <Sidebar
                feed={feed}
                onItemClick={handleSidebarItemClick}
                onSettingsOpen={handleOpenSettings}
                onGoHome={handleGoHome}
                onOpenCommandPost={handleOpenCommandPost}
                onOpenAgentSession={closeSidebarOnMobile}
                desktopWidth={isMobile ? undefined : sidebarWidth}
                resizeHandleProps={isMobile ? undefined : resizeHandleProps}
                asideRef={sidebarAsideRef}
              />
            </Profiler>

            <main className="min-w-0 flex-1 overflow-auto">
              <Profiler id="Outlet" onRender={onProfilerRender}>
                <Outlet />
              </Profiler>
            </main>

            <SettingsDrawer isOpen={showSettingsDrawer} onClose={handleCloseSettingsDrawer} />
          </div>
        </BacklogItemsProvider>
      </AppShellContext.Provider>
    </Profiler>
  );
}
