import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { FileList } from "./components/FileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import {
  useHealth,
  useRepoStatus,
  useDiff,
  useSyncStatus,
  useStageFiles,
  useUnstageFiles,
  useCommit,
  useDiscardFiles,
  usePush,
  usePull,
  queryKeys
} from "./lib/hooks";

export default function App() {
  const queryClient = useQueryClient();
  const mainRef = useRef<HTMLDivElement | null>(null);
  const sidebarRef = useRef<HTMLDivElement | null>(null);

  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") return 320;
    const stored = Number(localStorage.getItem("gct.sidebarWidth"));
    return Number.isFinite(stored) && stored > 0 ? stored : 320;
  });
  const [changesHeight, setChangesHeight] = useState(() => {
    if (typeof window === "undefined") return 420;
    const stored = Number(localStorage.getItem("gct.changesHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 420;
  });
  const [changesCollapsed, setChangesCollapsed] = useState(false);
  const [commitCollapsed, setCommitCollapsed] = useState(false);
  const [isResizingSidebar, setIsResizingSidebar] = useState(false);
  const [isResizingSplit, setIsResizingSplit] = useState(false);
  const sidebarResize = useRef<{ left: number; max: number } | null>(null);
  const splitResize = useRef<{ top: number; height: number } | null>(null);
  const sidebarMinWidth = 200;
  const diffMinWidth = 320;

  // Selected file state
  const selectionKey = useCallback(
    (entry: { path: string; staged: boolean }) => `${entry.staged ? "1" : "0"}:${entry.path}`,
    []
  );
  const [selectedFile, setSelectedFile] = useState<string | undefined>();
  const [selectedIsStaged, setSelectedIsStaged] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<Array<{ path: string; staged: boolean }>>([]);
  const lastSelectedKeyRef = useRef<string | null>(null);
  const [confirmingDiscard, setConfirmingDiscard] = useState<string | null>(null);
  const [lastCommitHash, setLastCommitHash] = useState<string | undefined>();
  const [commitError, setCommitError] = useState<string | undefined>();

  // Queries
  const healthQuery = useHealth();
  const statusQuery = useRepoStatus();
  const syncStatusQuery = useSyncStatus();
  const diffQuery = useDiff(selectedFile, selectedIsStaged);

  // Mutations
  const stageMutation = useStageFiles();
  const unstageMutation = useUnstageFiles();
  const commitMutation = useCommit();
  const discardMutation = useDiscardFiles();
  const pushMutation = usePush();
  const pullMutation = usePull();

  const isStaging = stageMutation.isPending || unstageMutation.isPending;
  const isDiscarding = discardMutation.isPending;

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    if (selectedFile) {
      queryClient.invalidateQueries({
        queryKey: queryKeys.diff(selectedFile, selectedIsStaged)
      });
    }
  }, [queryClient, selectedFile, selectedIsStaged]);

  const orderedFiles = useMemo(() => {
    const files = statusQuery.data?.files;
    if (!files) return [] as Array<{ path: string; staged: boolean }>;

    return [
      ...(files.conflicts ?? []).map((path) => ({ path, staged: false })),
      ...(files.staged ?? []).map((path) => ({ path, staged: true })),
      ...(files.unstaged ?? []).map((path) => ({ path, staged: false })),
      ...(files.untracked ?? []).map((path) => ({ path, staged: false }))
    ];
  }, [statusQuery.data?.files]);

  const orderedKeys = useMemo(() => orderedFiles.map((entry) => selectionKey(entry)), [orderedFiles, selectionKey]);
  const orderedKeyToEntry = useMemo(
    () => new Map(orderedFiles.map((entry) => [selectionKey(entry), entry])),
    [orderedFiles, selectionKey]
  );
  const orderedIndexMap = useMemo(
    () => new Map(orderedKeys.map((key, index) => [key, index])),
    [orderedKeys]
  );
  const orderedKeySet = useMemo(() => new Set(orderedKeys), [orderedKeys]);

  const handleSelectFile = useCallback(
    (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => {
      const nextEntry = orderedKeyToEntry.get(selectionKey({ path, staged })) ?? { path, staged };
      const nextKey = selectionKey(nextEntry);
      const lastKey = lastSelectedKeyRef.current;
      const isToggle = event.metaKey || event.ctrlKey;
      const isRange = event.shiftKey && lastKey && orderedIndexMap.has(lastKey);
      let nextSelection: Array<{ path: string; staged: boolean }>;

      if (isRange && orderedIndexMap.has(nextKey)) {
        const start = orderedIndexMap.get(lastKey) ?? 0;
        const end = orderedIndexMap.get(nextKey) ?? 0;
        const [from, to] = start < end ? [start, end] : [end, start];
        nextSelection = orderedKeys
          .slice(from, to + 1)
          .map((key) => orderedKeyToEntry.get(key))
          .filter((entry): entry is { path: string; staged: boolean } => Boolean(entry));
      } else if (isToggle) {
        const hasEntry = selectedFiles.some((entry) => selectionKey(entry) === nextKey);
        if (hasEntry) {
          nextSelection = selectedFiles.filter((entry) => selectionKey(entry) !== nextKey);
        } else {
          nextSelection = [...selectedFiles, nextEntry].sort((a, b) => {
            const aIndex = orderedIndexMap.get(selectionKey(a)) ?? 0;
            const bIndex = orderedIndexMap.get(selectionKey(b)) ?? 0;
            return aIndex - bIndex;
          });
        }
      } else {
        nextSelection = [nextEntry];
      }

      setSelectedFiles(nextSelection);
      lastSelectedKeyRef.current = nextKey;

      if (nextSelection.length === 0) {
        setSelectedFile(undefined);
        setSelectedIsStaged(false);
        return;
      }

      const clickedStillSelected = nextSelection.some((entry) => selectionKey(entry) === nextKey);
      const primary = clickedStillSelected
        ? nextEntry
        : nextSelection[nextSelection.length - 1];
      setSelectedFile(primary.path);
      setSelectedIsStaged(primary.staged);
    },
    [
      orderedIndexMap,
      orderedKeyToEntry,
      orderedKeys,
      selectionKey,
      selectedFiles
    ]
  );

  const handleStageFile = useCallback(
    (path: string) => {
      stageMutation.mutate(
        { paths: [path] },
        {
          onSuccess: () => {
            // If we were viewing this file's unstaged diff, switch to staged
            if (selectedFile === path && !selectedIsStaged) {
              setSelectedIsStaged(true);
            }
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, false)
            });
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, true)
            });
          }
        }
      );
    },
    [stageMutation, queryClient, selectedFile, selectedIsStaged]
  );

  const handleUnstageFile = useCallback(
    (path: string) => {
      unstageMutation.mutate(
        { paths: [path] },
        {
          onSuccess: () => {
            // If we were viewing this file's staged diff, switch to unstaged
            if (selectedFile === path && selectedIsStaged) {
              setSelectedIsStaged(false);
            }
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, false)
            });
            queryClient.invalidateQueries({
              queryKey: queryKeys.diff(path, true)
            });
          }
        }
      );
    },
    [unstageMutation, queryClient, selectedFile, selectedIsStaged]
  );

  const handleStageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files) return;

    const allUnstaged = [
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
    if (allUnstaged.length === 0) return;

    stageMutation.mutate({ paths: allUnstaged });
  }, [stageMutation, statusQuery.data]);

  const handleUnstageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files || (files.staged?.length ?? 0) === 0) return;

    unstageMutation.mutate({ paths: files.staged ?? [] });
  }, [unstageMutation, statusQuery.data]);

  const handleDiscardFile = useCallback(
    (path: string, untracked: boolean) => {
      discardMutation.mutate(
        { paths: [path], untracked },
        {
          onSuccess: () => {
            // If we were viewing this file's diff, clear selection
            if (selectedFile === path) {
              setSelectedFile(undefined);
            }
            setSelectedFiles((prev) =>
              prev.filter((entry) => !(entry.path === path && !entry.staged))
            );
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
          }
        }
      );
    },
    [discardMutation, queryClient, selectedFile]
  );

  const handleCommit = useCallback(
    (
      message: string,
      options: { conventional: boolean; authorName?: string; authorEmail?: string }
    ) => {
      setCommitError(undefined);
      setLastCommitHash(undefined);

      commitMutation.mutate(
        {
          message,
          validate_conventional: options.conventional,
          author_name: options.authorName,
          author_email: options.authorEmail
        },
        {
          onSuccess: (result) => {
            if (result.success && result.hash) {
              setLastCommitHash(result.hash);
              // Clear selection if viewing staged diff
              if (selectedIsStaged) {
                setSelectedFile(undefined);
              }
            } else {
              setCommitError(
                result.error ||
                  result.validation_errors?.join("; ") ||
                  "Commit failed"
              );
            }
          },
          onError: (error) => {
            setCommitError(error.message);
          }
        }
      );
    },
    [commitMutation, selectedIsStaged]
  );

  const handlePush = useCallback(() => {
    pushMutation.mutate({});
  }, [pushMutation]);

  const handlePull = useCallback(() => {
    pullMutation.mutate({});
  }, [pullMutation]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.sidebarWidth", String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.changesHeight", String(changesHeight));
  }, [changesHeight]);

  useEffect(() => {
    if (!orderedKeySet.size) {
      setSelectedFiles([]);
      setSelectedFile(undefined);
      setSelectedIsStaged(false);
      lastSelectedKeyRef.current = null;
      return;
    }

    setSelectedFiles((prev) => prev.filter((entry) => orderedKeySet.has(selectionKey(entry))));
  }, [orderedKeySet, selectionKey]);

  useEffect(() => {
    if (!selectedFile) return;
    const activeKey = selectionKey({ path: selectedFile, staged: selectedIsStaged });
    if (orderedKeySet.has(activeKey)) return;

    if (selectedFiles.length > 0) {
      const fallback = selectedFiles[selectedFiles.length - 1];
      setSelectedFile(fallback.path);
      setSelectedIsStaged(fallback.staged);
    } else {
      setSelectedFile(undefined);
      setSelectedIsStaged(false);
    }
  }, [orderedKeySet, selectedFile, selectedFiles, selectedIsStaged, selectionKey]);

  useEffect(() => {
    if (!sidebarRef.current || typeof ResizeObserver === "undefined") return;

    const minTop = 200;
    const minBottom = 180;
    const clamp = () => {
      if (!sidebarRef.current || changesCollapsed || commitCollapsed) return;
      const height = sidebarRef.current.clientHeight;
      const maxHeight = Math.max(minTop, height - minBottom);
      if (changesHeight > maxHeight) {
        setChangesHeight(maxHeight);
      } else if (changesHeight < minTop) {
        setChangesHeight(Math.min(minTop, maxHeight));
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(sidebarRef.current);
    return () => observer.disconnect();
  }, [changesHeight, changesCollapsed, commitCollapsed]);

  useEffect(() => {
    if (!isResizingSidebar) return;

    const handleMove = (event: MouseEvent) => {
      if (!sidebarResize.current) return;
      const minWidth = sidebarMinWidth;
      const nextWidth = event.clientX - sidebarResize.current.left;
      const clampedWidth = Math.max(minWidth, Math.min(sidebarResize.current.max, nextWidth));
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

  useEffect(() => {
    if (!isResizingSplit) return;

    const handleMove = (event: MouseEvent) => {
      if (!splitResize.current) return;
      const minTop = 200;
      const minBottom = 180;
      const nextHeight = event.clientY - splitResize.current.top;
      const maxHeight = splitResize.current.height - minBottom;
      const clampedHeight = Math.max(minTop, Math.min(maxHeight, nextHeight));
      setChangesHeight(clampedHeight);
    };

    const handleUp = () => {
      setIsResizingSplit(false);
      splitResize.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizingSplit]);

  const handleSidebarResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!mainRef.current) return;
    const rect = mainRef.current.getBoundingClientRect();
    const minDiffWidth = diffMinWidth;
    const minWidth = sidebarMinWidth;
    sidebarResize.current = {
      left: rect.left,
      max: Math.max(minWidth, rect.width - minDiffWidth)
    };
    setIsResizingSidebar(true);
  };

  const handleSplitResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!sidebarRef.current || changesCollapsed || commitCollapsed) return;
    const rect = sidebarRef.current.getBoundingClientRect();
    splitResize.current = { top: rect.top, height: rect.height };
    setIsResizingSplit(true);
  };

  const showSplitHandle = !changesCollapsed && !commitCollapsed;
  const sidebarRows = (() => {
    if (changesCollapsed && commitCollapsed) return "auto 0px auto";
    if (changesCollapsed) return "auto 0px minmax(0, 1fr)";
    if (commitCollapsed) return "minmax(0, 1fr) 0px auto";
    return `minmax(0, ${changesHeight}px) 6px minmax(0, 1fr)`;
  })();

  return (
    <div
      className="h-screen flex flex-col bg-slate-950 text-slate-50"
      data-testid="git-control-tower"
    >
      {/* Status Header */}
      <StatusHeader
        status={statusQuery.data}
        health={healthQuery.data}
        syncStatus={syncStatusQuery.data}
        isLoading={statusQuery.isLoading || healthQuery.isLoading}
        onRefresh={handleRefresh}
        onPush={handlePush}
        onPull={handlePull}
        isPushing={pushMutation.isPending}
        isPulling={pullMutation.isPending}
      />

      {/* Main Content - Split Pane */}
      <div className="flex-1 flex overflow-hidden" ref={mainRef}>
        {/* File List Panel + Commit Panel */}
        <div
          className="flex-shrink-0 border-r border-slate-800 overflow-hidden min-w-0"
          style={{ width: sidebarWidth, minWidth: sidebarMinWidth }}
        >
          <div
            className="h-full min-h-0 min-w-0 grid overflow-hidden"
            style={{ gridTemplateRows: sidebarRows }}
            ref={sidebarRef}
          >
            <div className="min-h-0 min-w-0 overflow-hidden">
              <FileList
                files={statusQuery.data?.files}
                selectedFiles={selectedFiles}
                onSelectFile={handleSelectFile}
                onStageFile={handleStageFile}
                onUnstageFile={handleUnstageFile}
                onDiscardFile={handleDiscardFile}
                onStageAll={handleStageAll}
                onUnstageAll={handleUnstageAll}
                isStaging={isStaging}
                isDiscarding={isDiscarding}
                confirmingDiscard={confirmingDiscard}
                onConfirmDiscard={setConfirmingDiscard}
                collapsed={changesCollapsed}
                onToggleCollapse={() => setChangesCollapsed((prev) => !prev)}
                fillHeight={!changesCollapsed}
              />
            </div>
            <div
              className={`${
                showSplitHandle
                  ? "cursor-row-resize bg-slate-900 hover:bg-slate-800"
                  : "bg-transparent"
              }`}
              onMouseDown={showSplitHandle ? handleSplitResizeStart : undefined}
              aria-hidden="true"
            />
            <div className="min-h-0 min-w-0 border-t border-slate-800 overflow-hidden">
              <CommitPanel
                stagedCount={statusQuery.data?.summary.staged ?? 0}
              onCommit={handleCommit}
              isCommitting={commitMutation.isPending}
              lastCommitHash={lastCommitHash}
              commitError={commitError}
              defaultAuthorName={statusQuery.data?.author?.name}
              defaultAuthorEmail={statusQuery.data?.author?.email}
              collapsed={commitCollapsed}
              onToggleCollapse={() => setCommitCollapsed((prev) => !prev)}
              fillHeight={!commitCollapsed}
            />
            </div>
          </div>
        </div>

        <div
          className="w-1 bg-slate-900 hover:bg-slate-800 cursor-col-resize"
          onMouseDown={handleSidebarResizeStart}
          aria-hidden="true"
        />

        {/* Diff Viewer Panel */}
        <div className="flex-1 overflow-hidden">
          <DiffViewer
            diff={diffQuery.data}
            selectedFile={selectedFile}
            isStaged={selectedIsStaged}
            isLoading={diffQuery.isLoading}
            error={diffQuery.error}
            repoDir={statusQuery.data?.repo_dir}
          />
        </div>
      </div>

      {/* Error Toast for Mutations */}
      {(stageMutation.error ||
        unstageMutation.error ||
        discardMutation.error ||
        pushMutation.error ||
        pullMutation.error) && (
        <div
          className="fixed bottom-4 right-4 px-4 py-3 rounded-lg bg-red-950 border border-red-800 text-red-200 text-sm max-w-md"
          data-testid="error-toast"
        >
          <p className="font-medium">Operation failed</p>
          <p className="text-xs mt-1 text-red-300">
            {(
              stageMutation.error ||
              unstageMutation.error ||
              discardMutation.error ||
              pushMutation.error ||
              pullMutation.error
            )?.message}
          </p>
        </div>
      )}
    </div>
  );
}
