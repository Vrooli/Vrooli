import { useState, useCallback, useMemo, useRef, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { SandboxList } from "./components/SandboxList";
import { SandboxDetail } from "./components/SandboxDetail";
import { FileTree } from "./components/FileTree";
import { CreateSandboxDialog } from "./components/CreateSandboxDialog";
import { SettingsDialog } from "./components/SettingsDialog";
import { CommitPendingDialog } from "./components/CommitPendingDialog";
import { LaunchAgentDialog, type LaunchConfig } from "./components/LaunchAgentDialog";
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
  useApproveSandbox,
  useRejectSandbox,
  useDiscardFiles,
  useExecCommand,
  useStartProcess,
  queryKeys,
} from "./lib/hooks";
import { computeStats, type Sandbox, type CreateRequest } from "./lib/api";
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

  // URL parameters for deep-linking
  const urlParams = useMemo(() => getUrlParams(), []);
  const [deepLinkProcessed, setDeepLinkProcessed] = useState(false);

  // Local state
  const [selectedSandbox, setSelectedSandbox] = useState<Sandbox | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [commitDialogOpen, setCommitDialogOpen] = useState(false);
  const [launchDialogOpen, setLaunchDialogOpen] = useState(false);

  // Review mode state (lifted from SandboxDetail for sidebar coordination)
  const [isReviewMode, setIsReviewMode] = useState(false);
  const [selectedFileIds, setSelectedFileIds] = useState<string[]>([]);
  const [selectedHunks, setSelectedHunks] = useState<HunkSelection[]>([]);

  // Sidebar resize state
  const SIDEBAR_MIN_WIDTH = 200;
  const DETAIL_MIN_WIDTH = 400;
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") return 320;
    const stored = Number(localStorage.getItem("wsb.sidebarWidth"));
    return Number.isFinite(stored) && stored > 0 ? stored : 320;
  });
  const [isResizingSidebar, setIsResizingSidebar] = useState(false);
  const mainRef = useRef<HTMLDivElement | null>(null);
  const sidebarResize = useRef<{ start: number; max: number } | null>(null);

  // Queries
  const healthQuery = useHealth();
  const sandboxesQuery = useSandboxes();
  const diffQuery = useDiff(selectedSandbox?.id);

  // Deep-link sandbox query - only fetch if we have a sandbox ID in URL params
  const deepLinkSandboxQuery = useSandbox(
    !deepLinkProcessed && urlParams.sandboxId ? urlParams.sandboxId : undefined
  );

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
        setIsReviewMode(true);
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

  // Mutations
  const createMutation = useCreateSandbox();
  const deleteMutation = useDeleteSandbox();
  const stopMutation = useStopSandbox();
  const startMutation = useStartSandbox();
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
  }, []);

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
    startMutation.mutate(selectedSandbox.id, {
      onSuccess: (updated) => {
        setSelectedSandbox(updated);
      },
    });
  }, [selectedSandbox, startMutation]);

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
        setSelectedSandbox(null);
      },
    });
  }, [selectedSandbox, deleteMutation]);

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

  // Sidebar resize handler
  const handleSidebarResizeStart = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      event.preventDefault();
      if (!mainRef.current) return;

      const rect = mainRef.current.getBoundingClientRect();
      sidebarResize.current = {
        start: rect.left,
        max: Math.max(SIDEBAR_MIN_WIDTH, rect.width - DETAIL_MIN_WIDTH),
      };
      setIsResizingSidebar(true);
    },
    []
  );

  // Sidebar resize mouse events
  useEffect(() => {
    if (!isResizingSidebar) return;

    const handleMove = (event: MouseEvent) => {
      if (!sidebarResize.current) return;
      const nextWidth = event.clientX - sidebarResize.current.start;
      const clampedWidth = Math.max(
        SIDEBAR_MIN_WIDTH,
        Math.min(sidebarResize.current.max, nextWidth)
      );
      setSidebarWidth(clampedWidth);
    };

    const handleUp = () => {
      setIsResizingSidebar(false);
      sidebarResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingSidebar]);

  // Persist sidebar width to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("wsb.sidebarWidth", String(sidebarWidth));
  }, [sidebarWidth]);

  // Constrain sidebar width when viewport shrinks
  useEffect(() => {
    if (!mainRef.current || typeof ResizeObserver === "undefined") return;

    const clamp = () => {
      if (!mainRef.current) return;
      const width = mainRef.current.clientWidth;
      const maxSidebar = Math.max(SIDEBAR_MIN_WIDTH, width - DETAIL_MIN_WIDTH);
      if (sidebarWidth > maxSidebar) {
        setSidebarWidth(maxSidebar);
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(mainRef.current);
    return () => observer.disconnect();
  }, [sidebarWidth]);

  // Keep selected sandbox in sync with list updates
  const sandboxes = sandboxesQuery.data?.sandboxes || [];

  // Update selected sandbox from list if it was updated
  const selectedFromList = selectedSandbox
    ? sandboxes.find((sb) => sb.id === selectedSandbox.id)
    : null;

  // Use the list version if available (more up-to-date)
  const currentSandbox = selectedFromList || selectedSandbox;

  // Extract existing reserved paths from active sandboxes for conflict detection
  const existingReservedPaths = useMemo(() => {
    const paths = new Set<string>();
    sandboxes
      .filter((sb) => sb.status === "active" || sb.status === "creating" || sb.status === "stopped")
      .forEach((sb) => {
        const reserved = sb.reservedPaths?.length ? sb.reservedPaths : [sb.reservedPath || sb.scopePath];
        reserved.forEach((p) => p && paths.add(p));
      });
    return Array.from(paths);
  }, [sandboxes]);

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
          className="flex-shrink-0 border-r border-slate-800 overflow-hidden"
          style={{ width: sidebarWidth }}
        >
          {isReviewMode && currentSandbox ? (
            <FileTree
              diff={diffQuery.data}
              sandboxPath={(() => {
                if (
                  currentSandbox.noLock &&
                  (!currentSandbox.reservedPaths || currentSandbox.reservedPaths.length === 0) &&
                  !currentSandbox.reservedPath
                ) {
                  return "No lock";
                }
                const reserved = currentSandbox.reservedPaths?.length
                  ? currentSandbox.reservedPaths
                  : [currentSandbox.reservedPath || currentSandbox.scopePath || "/"];
                return reserved[0] || "/";
              })()}
              selectedHunks={selectedHunks}
              onFileClick={handleScrollToFile}
              onExitReview={handleExitReviewMode}
            />
          ) : (
            <SandboxList
              sandboxes={sandboxes}
              selectedId={currentSandbox?.id}
              onSelect={handleSelectSandbox}
              isLoading={sandboxesQuery.isLoading}
            />
          )}
        </div>

        {/* Sidebar Resize Handle */}
        <div
          className="w-1 bg-slate-900 hover:bg-slate-700 cursor-col-resize flex-shrink-0"
          onMouseDown={handleSidebarResizeStart}
        />

        {/* Detail Panel - Right Panel */}
        <div className="flex-1 min-w-0 overflow-hidden">
          <SandboxDetail
            sandbox={currentSandbox || undefined}
            diff={diffQuery.data}
            isDiffLoading={diffQuery.isLoading}
            diffError={diffQuery.error}
            onStop={handleStop}
            onStart={handleStart}
            onApprove={handleApprove}
            onOverrideAcceptance={handleOverrideAcceptance}
            onReject={handleReject}
            onDelete={handleDelete}
            onDiscardFile={handleDiscardFile}
            onLaunchAgent={() => setLaunchDialogOpen(true)}
            onApproveSelected={handleApproveSelected}
            isApproving={approveMutation.isPending}
            isRejecting={rejectMutation.isPending}
            isStopping={stopMutation.isPending}
            isStarting={startMutation.isPending}
            isDeleting={deleteMutation.isPending}
            isDiscarding={discardMutation.isPending}
            // Review mode state (lifted)
            isReviewMode={isReviewMode}
            onReviewModeChange={setIsReviewMode}
            selectedFileIds={selectedFileIds}
            onSelectedFileIdsChange={setSelectedFileIds}
            selectedHunks={selectedHunks}
            onSelectedHunksChange={setSelectedHunks}
          />
        </div>
      </div>

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

      {/* Error Toast */}
      {(createMutation.error ||
        deleteMutation.error ||
        stopMutation.error ||
        startMutation.error ||
        approveMutation.error ||
        rejectMutation.error ||
        discardMutation.error ||
        execMutation.error ||
        startProcessMutation.error) && (
        <div
          className="fixed bottom-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md z-50"
          data-testid={SELECTORS.errorToast}
        >
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">
            {(
              createMutation.error ||
              deleteMutation.error ||
              stopMutation.error ||
              startMutation.error ||
              approveMutation.error ||
              rejectMutation.error ||
              discardMutation.error ||
              execMutation.error ||
              startProcessMutation.error
            )?.message}
          </p>
        </div>
      )}
    </div>
  );
}
