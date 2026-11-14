import { useState, useCallback, useMemo, useRef, useEffect } from "react";
import { MousePointerClick, CheckCircle, Loader2 } from "lucide-react";
import { Button } from "./components/ui/button";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { Sidebar } from "./components/sidebar/Sidebar";
import { useSidebarState } from "./components/sidebar/useSidebarState";
import { SandboxDetail } from "./components/SandboxDetail";
import { ClosedSandboxDetail } from "./components/ClosedSandboxDetail";
import { DiffViewer } from "./components/DiffViewer";
import { FileTree } from "./components/FileTree";
import { CreateSandboxDialog } from "./components/CreateSandboxDialog";
import { SettingsDialog } from "./components/SettingsDialog";
import { CommitPendingDialog } from "./components/CommitPendingDialog";
import { LaunchAgentDialog, type LaunchConfig } from "./components/LaunchAgentDialog";
import { MobileNav, type MobilePanel } from "./components/layout/MobileNav";
import { MobileHeader } from "./components/layout/MobileHeader";
import { useIsMobile } from "./hooks/useViewportSize";
import type { HunkSelection } from "./components/DiffViewer";
import {
  useHealth,
  useSandboxes,
  useSandbox,
  useDiff,
  useCreateSandbox,
  useDeleteSandbox,
  useStopSandbox,
  useStartSandbox,
  useResumeSandbox,
  useApproveSandbox,
  useRejectSandbox,
  useDiscardFiles,
  useExecCommand,
  useStartProcess,
  queryKeys,
} from "./lib/hooks";
import {
  computeStats,
  isHistoryStatus,
  type CreateRequest,
  type DiffArchive,
  type Sandbox,
  type ViewMode,
} from "./lib/api";
import { SELECTORS } from "./consts/selectors";

/**
 * Parse URL parameters for deep-linking support.
 * Supported parameters:
 * - sandbox: Sandbox ID to auto-select
 * - review: Set to "true" to auto-enter review mode
 */
function getUrlParams(): { sandboxId: string | null; autoReview: boolean } {
  const params = new URLSearchParams(window.location.search);
  return {
    sandboxId: params.get("sandbox"),
    autoReview: params.get("review") === "true",
  };
}

export default function App() {
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();

  // URL parameters for deep-linking
  const urlParams = useMemo(() => getUrlParams(), []);
  const [deepLinkProcessed, setDeepLinkProcessed] = useState(false);

  // Mobile panel state
  const [mobileActivePanel, setMobileActivePanel] = useState<MobilePanel>("sandboxes");

  // Local state
  const [selectedSandbox, setSelectedSandbox] = useState<Sandbox | null>(null);
  // When the user clicks an archive row, we keep the raw archive payload
  // so ClosedSandboxDetail can render its archive-specific metadata
  // (snapshotAt, totalBlobBytes, archive_state, runId, …) that isn't on
  // the Sandbox shape.
  const [selectedArchive, setSelectedArchive] = useState<DiffArchive | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [commitDialogOpen, setCommitDialogOpen] = useState(false);
  const [launchDialogOpen, setLaunchDialogOpen] = useState(false);

  // Sidebar tab/filter/sort state. Lifted to App so we can imperatively
  // switch tabs on terminal-status transitions ("selection moved to
  // History" UX) and so the SettingsDialog can clear the History
  // search after retention changes archive contents.
  const [sidebarState, sidebarDispatch] = useSidebarState();
  const [transitionToast, setTransitionToast] = useState<{
    id: number;
    message: string;
  } | null>(null);
  const lastSelectedStatus = useRef<Sandbox["status"] | null>(null);

  // Review mode state (lifted from SandboxDetail for sidebar coordination)
  const [isReviewMode, setIsReviewMode] = useState(false);
  const [selectedFileIds, setSelectedFileIds] = useState<string[]>([]);
  const [selectedHunks, setSelectedHunks] = useState<HunkSelection[]>([]);

  // View mode state for DiffViewer
  const [viewMode, setViewMode] = useState<ViewMode>("diff");

  // Sidebar resize state.
  //
  // During a drag we write the width directly to the sidebar element via a
  // CSS custom property + RAF, bypassing React. Only the settled value at
  // mouseup commits to React state (and to localStorage). This avoids
  // ~30 setState calls per second cascading through the whole App tree —
  // see docs/perf/2026-05-03-history-fileviewer-resize.md F1/F2.
  const SIDEBAR_MIN_WIDTH = 200;
  const DETAIL_MIN_WIDTH = 400;
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") return 320;
    const stored = Number(localStorage.getItem("wsb.sidebarWidth"));
    return Number.isFinite(stored) && stored > 0 ? stored : 320;
  });
  const sidebarWidthRef = useRef(sidebarWidth);
  sidebarWidthRef.current = sidebarWidth;
  const mainRef = useRef<HTMLDivElement | null>(null);
  const sidebarPaneRef = useRef<HTMLDivElement | null>(null);
  const sidebarResize = useRef<{ start: number; max: number; current: number } | null>(null);

  // Queries
  const healthQuery = useHealth();
  const sandboxesQuery = useSandboxes();
  const diffQuery = useDiff(selectedSandbox?.id, viewMode);

  // Deep-link sandbox query - only fetch if we have a sandbox ID in URL params
  const deepLinkSandboxQuery = useSandbox(
    !deepLinkProcessed && urlParams.sandboxId ? urlParams.sandboxId : undefined
  );

  // Track whether we need to wait for diff data before entering review mode
  const pendingAutoReview = useRef(false);

  // Process deep-link when sandbox data is available
  useEffect(() => {
    if (deepLinkProcessed) return;
    if (!urlParams.sandboxId) {
      setDeepLinkProcessed(true);
      return;
    }

    // Wait for the deep-link sandbox query to complete
    if (deepLinkSandboxQuery.isLoading) return;

    if (deepLinkSandboxQuery.data) {
      setSelectedSandbox(deepLinkSandboxQuery.data);
      if (urlParams.autoReview) {
        // Don't enter review mode yet — wait for diff data to check if there are files
        pendingAutoReview.current = true;
      }
    }
    setDeepLinkProcessed(true);
  }, [
    deepLinkProcessed,
    urlParams.sandboxId,
    urlParams.autoReview,
    deepLinkSandboxQuery.isLoading,
    deepLinkSandboxQuery.data,
  ]);

  // Enter review mode once diff data is available (only for deep-link auto-review)
  useEffect(() => {
    if (!pendingAutoReview.current) return;
    if (diffQuery.isLoading) return;
    pendingAutoReview.current = false;
    if ((diffQuery.data?.files?.length ?? 0) > 0) {
      setIsReviewMode(true);
    }
  }, [diffQuery.isLoading, diffQuery.data]);

  // Sync state back to URL params
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (selectedSandbox?.id) params.set("sandbox", selectedSandbox.id);
    else params.delete("sandbox");
    if (isReviewMode) params.set("review", "true");
    else params.delete("review");
    const qs = params.toString();
    const newUrl = qs ? `${window.location.pathname}?${qs}` : window.location.pathname;
    window.history.replaceState(null, "", newUrl);
  }, [selectedSandbox?.id, isReviewMode]);

  // Auto-switch to Changes tab when entering review mode on mobile
  useEffect(() => {
    if (isMobile && isReviewMode) {
      setMobileActivePanel("changes");
    }
  }, [isMobile, isReviewMode]);

  // Mutations
  const createMutation = useCreateSandbox();
  const deleteMutation = useDeleteSandbox();
  const stopMutation = useStopSandbox();
  const startMutation = useStartSandbox();
  const resumeMutation = useResumeSandbox();
  const approveMutation = useApproveSandbox();
  const rejectMutation = useRejectSandbox();
  const discardMutation = useDiscardFiles();
  const execMutation = useExecCommand();
  const startProcessMutation = useStartProcess();

  // Computed stats
  const stats = useMemo(() => {
    if (!sandboxesQuery.data?.sandboxes) return undefined;
    return computeStats(sandboxesQuery.data.sandboxes);
  }, [sandboxesQuery.data?.sandboxes]);

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
    if (selectedSandbox?.id) {
      queryClient.invalidateQueries({ queryKey: queryKeys.diff(selectedSandbox.id) });
    }
  }, [queryClient, selectedSandbox?.id]);

  const handleSelectSandbox = useCallback((sandbox: Sandbox) => {
    setSelectedSandbox(sandbox);
    setSelectedArchive(null);
    if (isMobile) {
      setMobileActivePanel("details");
    }
  }, [isMobile]);

  const handleSelectArchive = useCallback(
    (archive: DiffArchive, asSandbox: Sandbox) => {
      setSelectedSandbox(asSandbox);
      setSelectedArchive(archive);
      if (isMobile) {
        setMobileActivePanel("details");
      }
    },
    [isMobile],
  );

  const handleCreate = useCallback(
    (req: CreateRequest) => {
      createMutation.mutate(req, {
        onSuccess: (newSandbox) => {
          setCreateDialogOpen(false);
          setSelectedSandbox(newSandbox);
        },
      });
    },
    [createMutation]
  );

  const handleStop = useCallback(() => {
    if (!selectedSandbox) return;
    stopMutation.mutate(selectedSandbox.id, {
      onSuccess: (updated) => {
        setSelectedSandbox(updated);
      },
    });
  }, [selectedSandbox, stopMutation]);

  const handleStart = useCallback(() => {
    if (!selectedSandbox) return;
    const mutation = selectedSandbox.status === "checkpointed" ? resumeMutation : startMutation;
    mutation.mutate(selectedSandbox.id, {
      onSuccess: (updated) => {
        setSelectedSandbox(updated);
      },
    });
  }, [selectedSandbox, startMutation, resumeMutation]);

  // Track which sandbox IDs are currently being restarted (stop then start)
  const [restartingIds, setRestartingIds] = useState<Set<string>>(new Set());

  const handleRestartSandbox = useCallback(
    (sandboxId: string) => {
      setRestartingIds((prev) => new Set(prev).add(sandboxId));
      stopMutation.mutate(sandboxId, {
        onSuccess: () => {
          startMutation.mutate(sandboxId, {
            onSettled: () => {
              setRestartingIds((prev) => {
                const next = new Set(prev);
                next.delete(sandboxId);
                return next;
              });
            },
          });
        },
        onError: () => {
          // If stop fails, try starting directly (may already be stopped)
          startMutation.mutate(sandboxId, {
            onSettled: () => {
              setRestartingIds((prev) => {
                const next = new Set(prev);
                next.delete(sandboxId);
                return next;
              });
            },
          });
        },
      });
    },
    [stopMutation, startMutation],
  );

  const handleRestartUnhealthy = useCallback(() => {
    const sandboxes = sandboxesQuery.data?.sandboxes ?? [];
    for (const sb of sandboxes) {
      if (sb.mountHealth && !sb.mountHealth.healthy) {
        handleRestartSandbox(sb.id);
      }
    }
  }, [sandboxesQuery.data?.sandboxes, handleRestartSandbox]);

  const handleApprove = useCallback(() => {
    if (!selectedSandbox) return;
    approveMutation.mutate(
      { id: selectedSandbox.id },
      {
        onSuccess: () => {
          // Refresh the sandbox to get updated status
          queryClient.invalidateQueries({
            queryKey: queryKeys.sandbox(selectedSandbox.id),
          });
          queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
        },
      }
    );
  }, [selectedSandbox, approveMutation, queryClient]);

  const handleOverrideAcceptance = useCallback(() => {
    if (!selectedSandbox) return;
    approveMutation.mutate(
      { id: selectedSandbox.id, options: { overrideAcceptance: true } },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({
            queryKey: queryKeys.sandbox(selectedSandbox.id),
          });
          queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
        },
      }
    );
  }, [selectedSandbox, approveMutation, queryClient]);

  const handleApproveSelected = useCallback(
    (options: {
      hunkRanges?: Array<{ fileId: string; startLine: number; endLine: number }>;
    }) => {
      if (!selectedSandbox || !options.hunkRanges?.length) return;

      approveMutation.mutate(
        {
          id: selectedSandbox.id,
          options: {
            mode: "hunks",
            hunkRanges: options.hunkRanges,
            overrideAcceptance: true, // Selected items bypass acceptance rules
          },
        },
        {
          onSuccess: () => {
            queryClient.invalidateQueries({
              queryKey: queryKeys.sandbox(selectedSandbox.id),
            });
            queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(selectedSandbox.id),
            });
          },
        }
      );
    },
    [selectedSandbox, approveMutation, queryClient]
  );

  const handleReject = useCallback(() => {
    if (!selectedSandbox) return;
    rejectMutation.mutate(
      { id: selectedSandbox.id },
      {
        onSuccess: (updated) => {
          setSelectedSandbox(updated);
        },
      }
    );
  }, [selectedSandbox, rejectMutation]);

  const handleDelete = useCallback(() => {
    if (!selectedSandbox) return;
    deleteMutation.mutate(selectedSandbox.id, {
      onSuccess: () => {
        // Preserve the selection: the sandbox transitions to status =
        // "deleted" with a durable archive row, and the transition
        // effect moves the sidebar to the History tab. The user keeps
        // the same row selected and continues seeing the archived
        // diff via the existing /diff endpoint.
        queryClient.invalidateQueries({ queryKey: ["sandboxes"] });
        if (selectedSandbox.id) {
          queryClient.invalidateQueries({ queryKey: queryKeys.sandbox(selectedSandbox.id) });
          queryClient.invalidateQueries({ queryKey: queryKeys.diff(selectedSandbox.id) });
        }
      },
    });
  }, [selectedSandbox, deleteMutation, queryClient]);

  const handleDiscardFile = useCallback(
    (fileId: string) => {
      if (!selectedSandbox) return;
      discardMutation.mutate({
        sandboxId: selectedSandbox.id,
        fileIds: [fileId],
      });
    },
    [selectedSandbox, discardMutation]
  );

  // Scroll to a specific file in the diff viewer
  const handleScrollToFile = useCallback((filePath: string) => {
    // Find the file element by data-file-path attribute and scroll to it
    const fileElement = document.querySelector(
      `[data-testid="diff-file-item"][data-file-path="${filePath}"]`
    );
    if (fileElement) {
      fileElement.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, []);

  // Exit review mode and clear selections
  const handleExitReviewMode = useCallback(() => {
    setIsReviewMode(false);
    setSelectedFileIds([]);
    setSelectedHunks([]);
  }, []);

  const handleLaunch = useCallback(
    (config: LaunchConfig) => {
      if (!selectedSandbox) return;

      const request = {
        command: config.command,
        args: config.args.length > 0 ? config.args : undefined,
        isolationLevel: config.isolationProfile, // Backend expects isolationLevel but accepts profile IDs
        memoryLimitMB: config.memoryLimitMB,
        cpuTimeSec: config.cpuTimeSec,
        maxProcesses: config.maxProcesses,
        maxOpenFiles: config.maxOpenFiles,
        allowNetwork: config.allowNetwork,
        env: Object.keys(config.env).length > 0 ? config.env : undefined,
        workingDir: config.workingDir !== "/workspace" ? config.workingDir : undefined,
      };

      if (config.mode === "run") {
        startProcessMutation.mutate(
          {
            sandboxId: selectedSandbox.id,
            request: {
              ...request,
              name: config.name,
            },
          },
          {
            onSuccess: () => {
              setLaunchDialogOpen(false);
            },
          }
        );
      } else {
        execMutation.mutate(
          {
            sandboxId: selectedSandbox.id,
            request: {
              ...request,
              timeoutSec: config.timeoutSec,
            },
          },
          {
            onSuccess: () => {
              setLaunchDialogOpen(false);
            },
          }
        );
      }
    },
    [selectedSandbox, execMutation, startProcessMutation]
  );

  // Sidebar resize handler. Installs window-level mousemove/mouseup directly
  // (no isResizing state). Mousemove writes the width straight to the pane
  // element via RAF — React state is touched only on mouseup.
  const handleSidebarResizeStart = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      event.preventDefault();
      if (!mainRef.current || !sidebarPaneRef.current) return;

      const rect = mainRef.current.getBoundingClientRect();
      const startWidth = sidebarPaneRef.current.getBoundingClientRect().width;
      sidebarResize.current = {
        start: rect.left,
        max: Math.max(SIDEBAR_MIN_WIDTH, rect.width - DETAIL_MIN_WIDTH),
        current: startWidth,
      };

      let rafId = 0;
      let pendingWidth = startWidth;

      const applyWidth = () => {
        rafId = 0;
        if (!sidebarPaneRef.current || !sidebarResize.current) return;
        sidebarPaneRef.current.style.width = `${pendingWidth}px`;
        sidebarResize.current.current = pendingWidth;
      };

      const handleMove = (e: MouseEvent) => {
        if (!sidebarResize.current) return;
        const nextWidth = e.clientX - sidebarResize.current.start;
        pendingWidth = Math.max(
          SIDEBAR_MIN_WIDTH,
          Math.min(sidebarResize.current.max, nextWidth)
        );
        if (rafId === 0) rafId = requestAnimationFrame(applyWidth);
      };

      const handleUp = () => {
        if (rafId !== 0) {
          cancelAnimationFrame(rafId);
          applyWidth();
        }
        const settled = sidebarResize.current?.current ?? startWidth;
        sidebarResize.current = null;
        window.removeEventListener("mousemove", handleMove);
        window.removeEventListener("mouseup", handleUp);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        setSidebarWidth(settled);
        try {
          localStorage.setItem("wsb.sidebarWidth", String(settled));
        } catch {
          // localStorage may be disabled or full; the in-memory width is
          // still applied, so we silently skip persistence.
        }
      };

      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
      window.addEventListener("mousemove", handleMove);
      window.addEventListener("mouseup", handleUp);
    },
    []
  );

  // Constrain sidebar width when viewport shrinks. The observer is installed
  // once on mount; the latest width is read from a ref so we don't reattach
  // the observer on every drag step (see F8).
  useEffect(() => {
    if (!mainRef.current || typeof ResizeObserver === "undefined") return;

    const clamp = () => {
      if (!mainRef.current) return;
      const width = mainRef.current.clientWidth;
      const maxSidebar = Math.max(SIDEBAR_MIN_WIDTH, width - DETAIL_MIN_WIDTH);
      if (sidebarWidthRef.current > maxSidebar) {
        setSidebarWidth(maxSidebar);
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(mainRef.current);
    return () => observer.disconnect();
  }, []);

  // Keep selected sandbox in sync with list updates
  const sandboxes = useMemo(
    () => sandboxesQuery.data?.sandboxes || [],
    [sandboxesQuery.data?.sandboxes],
  );

  // Update selected sandbox from list if it was updated
  const selectedFromList = selectedSandbox
    ? sandboxes.find((sb) => sb.id === selectedSandbox.id)
    : null;

  // Use the list version if available (more up-to-date)
  const currentSandbox = selectedFromList || selectedSandbox;

  // Selection-on-transition UX: when the selected sandbox transitions
  // from an active-tab status to a history-tab status, point the
  // sidebar at History and surface a transient toast. The selection
  // itself is preserved so the right panel keeps showing the same
  // sandbox (now via ClosedSandboxDetail).
  useEffect(() => {
    const status = currentSandbox?.status;
    if (!status) {
      lastSelectedStatus.current = null;
      return;
    }
    const prev = lastSelectedStatus.current;
    lastSelectedStatus.current = status;
    if (!prev || prev === status) return;
    if (!isHistoryStatus(prev) && isHistoryStatus(status)) {
      sidebarDispatch({ type: "SET_TAB", tab: "history" });
      const verb = status === "approved" ? "approved" : status === "rejected" ? "rejected" : "deleted";
      setTransitionToast({
        id: Date.now(),
        message: `Sandbox ${verb} — moved to History`,
      });
    }
  }, [currentSandbox?.status, sidebarDispatch]);

  // Auto-dismiss the toast after a short window. The id-based dependency
  // makes successive transitions reset the timer cleanly.
  useEffect(() => {
    if (!transitionToast) return;
    const handle = window.setTimeout(() => setTransitionToast(null), 4000);
    return () => window.clearTimeout(handle);
  }, [transitionToast]);

  // Extract existing reserved paths from active sandboxes for conflict detection
  const existingReservedPaths = useMemo(() => {
    const paths = new Set<string>();
    sandboxes
      .filter(
        (sb) =>
          sb.status === "active" ||
          sb.status === "creating" ||
          sb.status === "stopped" ||
          sb.status === "checkpointed",
      )
      .forEach((sb) => {
        const reserved = sb.reservedPaths?.length ? sb.reservedPaths : [sb.reservedPath || sb.scopePath];
        reserved.forEach((p) => p && paths.add(p));
      });
    return Array.from(paths);
  }, [sandboxes]);

  // Helper to compute sandbox path for FileTree
  const getSandboxPath = useCallback((sandbox: Sandbox) => {
    if (
      sandbox.noLock &&
      (!sandbox.reservedPaths || sandbox.reservedPaths.length === 0) &&
      !sandbox.reservedPath
    ) {
      return "No lock";
    }
    const reserved = sandbox.reservedPaths?.length
      ? sandbox.reservedPaths
      : [sandbox.reservedPath || sandbox.scopePath || "/"];
    return reserved[0] || "/";
  }, []);

  // Diff file count for mobile badge
  const diffFileCount = diffQuery.data?.files?.length ?? 0;

  // Whether the current sandbox can have review/approval actions
  const canReview = currentSandbox?.status === "active" || currentSandbox?.status === "stopped";

  // Shared sandbox detail props
  const sandboxDetailProps = {
    sandbox: currentSandbox || undefined,
    diff: diffQuery.data,
    isDiffLoading: diffQuery.isLoading,
    diffError: diffQuery.error,
    onStop: handleStop,
    onStart: handleStart,
    onApprove: handleApprove,
    onOverrideAcceptance: handleOverrideAcceptance,
    onReject: handleReject,
    onDelete: handleDelete,
    onDiscardFile: handleDiscardFile,
    onLaunchAgent: () => setLaunchDialogOpen(true),
    onApproveSelected: handleApproveSelected,
    isApproving: approveMutation.isPending,
    isRejecting: rejectMutation.isPending,
    isStopping: stopMutation.isPending,
    isStarting: startMutation.isPending || resumeMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isDiscarding: discardMutation.isPending,
    isReviewMode,
    onReviewModeChange: setIsReviewMode,
    selectedFileIds,
    onSelectedFileIdsChange: setSelectedFileIds,
    selectedHunks,
    onSelectedHunksChange: setSelectedHunks,
    viewMode,
    onViewModeChange: setViewMode,
  } as const;

  // Shared elements (dialogs + error toast)
  const sharedElements = (
    <>
      {/* Create Sandbox Dialog */}
      <CreateSandboxDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onCreate={handleCreate}
        isCreating={createMutation.isPending}
        recentSandboxes={sandboxes}
        existingReservedPaths={existingReservedPaths}
        defaultProjectRoot={healthQuery.data?.config?.projectRoot}
      />

      {/* Settings Dialog */}
      <SettingsDialog
        open={settingsDialogOpen}
        onOpenChange={setSettingsDialogOpen}
      />

      {/* Commit Pending Dialog */}
      <CommitPendingDialog
        open={commitDialogOpen}
        onOpenChange={setCommitDialogOpen}
        projectRoot={healthQuery.data?.config?.projectRoot}
      />

      {/* Launch Agent Dialog */}
      {currentSandbox && (
        <LaunchAgentDialog
          open={launchDialogOpen}
          onOpenChange={setLaunchDialogOpen}
          sandbox={currentSandbox}
          onLaunch={handleLaunch}
          isLaunching={execMutation.isPending || startProcessMutation.isPending}
        />
      )}

      {/* Selection-on-transition toast */}
      {transitionToast && (
        <div
          key={transitionToast.id}
          className={`fixed left-1/2 -translate-x-1/2 px-4 py-2 rounded-lg bg-emerald-950/90 border border-emerald-700 text-emerald-100 text-sm max-w-md z-50 shadow-lg ${
            isMobile ? "bottom-24" : "bottom-4"
          }`}
          role="status"
          data-testid="sidebar-transition-toast"
        >
          {transitionToast.message}
        </div>
      )}

      {/* Error Toast */}
      {(createMutation.error ||
        deleteMutation.error ||
        stopMutation.error ||
        startMutation.error ||
        resumeMutation.error ||
        approveMutation.error ||
        rejectMutation.error ||
        discardMutation.error ||
        execMutation.error ||
        startProcessMutation.error) && (
        <div
          className={`fixed right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md z-50 ${isMobile ? "bottom-20 left-4" : "bottom-4"}`}
          data-testid={SELECTORS.errorToast}
        >
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">
            {(
              createMutation.error ||
              deleteMutation.error ||
              stopMutation.error ||
              startMutation.error ||
              resumeMutation.error ||
              approveMutation.error ||
              rejectMutation.error ||
              discardMutation.error ||
              execMutation.error ||
              startProcessMutation.error
            )?.message}
          </p>
        </div>
      )}
    </>
  );

  // ── Mobile Layout ──
  if (isMobile) {
    return (
      <div
        className="h-[100dvh] flex flex-col bg-slate-950 text-slate-50"
        data-testid={SELECTORS.app}
      >
        <MobileHeader
          health={healthQuery.data}
          stats={stats}
          isLoading={healthQuery.isLoading || sandboxesQuery.isLoading}
          onRefresh={handleRefresh}
          onCreateClick={() => setCreateDialogOpen(true)}
          onSettingsClick={() => setSettingsDialogOpen(true)}
          onCommitClick={() => setCommitDialogOpen(true)}
        />

        <main className="flex-1 min-h-0 overflow-hidden pb-16">
          {mobileActivePanel === "sandboxes" && (
            <div className="h-full overflow-y-auto">
              <Sidebar
                sandboxes={sandboxes}
                selectedId={currentSandbox?.id}
                isLoading={sandboxesQuery.isLoading}
                onSelectActive={handleSelectSandbox}
                onSelectHistory={handleSelectArchive}
                onRestartSandbox={handleRestartSandbox}
                onRestartUnhealthy={handleRestartUnhealthy}
                restartingIds={restartingIds}
                state={sidebarState}
                dispatch={sidebarDispatch}
              />
            </div>
          )}

          {mobileActivePanel === "details" && (
            <div className="h-full overflow-y-auto">
              {selectedArchive && currentSandbox ? (
                <ClosedSandboxDetail
                  archive={selectedArchive}
                  diff={diffQuery.data}
                  isDiffLoading={diffQuery.isLoading}
                  diffError={diffQuery.error}
                />
              ) : (
                <SandboxDetail {...sandboxDetailProps} hideDiffViewer />
              )}
            </div>
          )}

          {mobileActivePanel === "changes" && (
            <div className="h-full flex flex-col overflow-hidden">
              {/* Mobile review controls bar */}
              {canReview && currentSandbox && (
                <div className="flex-shrink-0 px-3 py-2 border-b border-slate-800 flex flex-wrap items-center gap-2 bg-slate-900/50">
                  <Button
                    variant={isReviewMode ? "default" : "outline"}
                    size="sm"
                    onClick={() => {
                      const newMode = !isReviewMode;
                      setIsReviewMode(newMode);
                      if (!newMode) {
                        setSelectedFileIds([]);
                        setSelectedHunks([]);
                      }
                    }}
                    data-testid="mobile-review-toggle"
                  >
                    <MousePointerClick className="h-3.5 w-3.5 mr-1.5" />
                    {isReviewMode ? "Exit Review" : "Review"}
                  </Button>

                  {isReviewMode && selectedHunks.length > 0 && (
                    <Button
                      variant="success"
                      size="sm"
                      onClick={() => {
                        handleApproveSelected({
                          hunkRanges: selectedHunks.map((h) => ({
                            fileId: h.fileId,
                            startLine: h.startLine,
                            endLine: h.endLine,
                          })),
                        });
                        setSelectedFileIds([]);
                        setSelectedHunks([]);
                      }}
                      disabled={approveMutation.isPending}
                      data-testid="mobile-approve-selected"
                    >
                      {approveMutation.isPending ? (
                        <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                      ) : (
                        <CheckCircle className="h-3.5 w-3.5 mr-1.5" />
                      )}
                      Approve ({selectedHunks.length})
                    </Button>
                  )}
                </div>
              )}

              {/* File tree (shown inline when review mode is active) */}
              {isReviewMode && currentSandbox && (
                <div className="flex-shrink-0 max-h-[30vh] overflow-y-auto border-b border-slate-800">
                  <FileTree
                    diff={diffQuery.data}
                    sandboxPath={getSandboxPath(currentSandbox)}
                    selectedHunks={selectedHunks}
                    onFileClick={handleScrollToFile}
                    onExitReview={handleExitReviewMode}
                    hideHeader
                  />
                </div>
              )}

              {/* Diff viewer */}
              <div className="flex-1 min-h-0">
                <DiffViewer
                  diff={diffQuery.data}
                  isLoading={diffQuery.isLoading}
                  error={diffQuery.error}
                  showFileActions={canReview && !!handleDiscardFile}
                  onRejectFile={handleDiscardFile}
                  showFileSelection={isReviewMode && canReview}
                  selectedFiles={selectedFileIds}
                  onFileSelectionChange={setSelectedFileIds}
                  showHunkSelection={isReviewMode && canReview}
                  selectedHunks={selectedHunks}
                  onHunkSelectionChange={setSelectedHunks}
                  viewMode={viewMode}
                  onViewModeChange={setViewMode}
                  onEmptyAction={() => setMobileActivePanel("sandboxes")}
                  emptyActionLabel="Go to Sandboxes"
                />
              </div>
            </div>
          )}
        </main>

        <MobileNav
          activePanel={mobileActivePanel}
          onPanelChange={setMobileActivePanel}
          changeCount={diffFileCount}
        />

        {sharedElements}
      </div>
    );
  }

  // ── Desktop Layout ──
  return (
    <div
      className="h-screen flex flex-col bg-slate-950 text-slate-50"
      data-testid={SELECTORS.app}
    >
      {/* Status Header */}
      <StatusHeader
        health={healthQuery.data}
        stats={stats}
        isLoading={healthQuery.isLoading || sandboxesQuery.isLoading}
        onRefresh={handleRefresh}
        onCreateClick={() => setCreateDialogOpen(true)}
        onSettingsClick={() => setSettingsDialogOpen(true)}
        onCommitClick={() => setCommitDialogOpen(true)}
      />

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden" ref={mainRef}>
        {/* Left Panel - Sandbox List or File Tree (in review mode) */}
        <div
          ref={sidebarPaneRef}
          className="flex-shrink-0 border-r border-slate-800 overflow-hidden"
          style={{ width: sidebarWidth }}
        >
          {isReviewMode && currentSandbox ? (
            <FileTree
              diff={diffQuery.data}
              sandboxPath={getSandboxPath(currentSandbox)}
              selectedHunks={selectedHunks}
              onFileClick={handleScrollToFile}
              onExitReview={handleExitReviewMode}
            />
          ) : (
            <Sidebar
              sandboxes={sandboxes}
              selectedId={currentSandbox?.id}
              isLoading={sandboxesQuery.isLoading}
              onSelectActive={handleSelectSandbox}
              onSelectHistory={handleSelectArchive}
              onRestartSandbox={handleRestartSandbox}
              onRestartUnhealthy={handleRestartUnhealthy}
              restartingIds={restartingIds}
              state={sidebarState}
              dispatch={sidebarDispatch}
            />
          )}
        </div>

        {/* Sidebar Resize Handle */}
        <div
          data-testid="sidebar-resize-handle"
          className="w-1 bg-slate-900 hover:bg-slate-700 cursor-col-resize flex-shrink-0"
          onMouseDown={handleSidebarResizeStart}
        />

        {/* Detail Panel - Right Panel */}
        <div className="flex-1 min-w-0 overflow-hidden">
          {selectedArchive && currentSandbox ? (
            <ClosedSandboxDetail
              archive={selectedArchive}
              diff={diffQuery.data}
              isDiffLoading={diffQuery.isLoading}
              diffError={diffQuery.error}
            />
          ) : (
            <SandboxDetail {...sandboxDetailProps} />
          )}
        </div>
      </div>

      {sharedElements}
    </div>
  );
}
