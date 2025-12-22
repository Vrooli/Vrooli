import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { StatusHeader } from "./components/StatusHeader";
import { FileList } from "./components/FileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import { GitHistory } from "./components/GitHistory";
import { GroupingSettingsModal } from "./components/GroupingSettingsModal";
import type { GroupingRule } from "./components/FileList";
import {
  useHealth,
  useRepoStatus,
  useRepoHistory,
  useDiff,
  useSyncStatus,
  useApprovedChanges,
  useApprovedChangesPreview,
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
  const [historyHeight, setHistoryHeight] = useState(() => {
    if (typeof window === "undefined") return 200;
    const stored = Number(localStorage.getItem("gct.historyHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 200;
  });
  const [changesCollapsed, setChangesCollapsed] = useState(false);
  const [historyCollapsed, setHistoryCollapsed] = useState(false);
  const [commitCollapsed, setCommitCollapsed] = useState(false);
  const [isResizingSidebar, setIsResizingSidebar] = useState(false);
  const [isResizingSplit, setIsResizingSplit] = useState(false);
  const [isResizingHistory, setIsResizingHistory] = useState(false);
  const [historyLimit, setHistoryLimit] = useState(50);
  const historyMaxLimit = 200;
  const [historySearch, setHistorySearch] = useState("");
  const [historyScopeFilter, setHistoryScopeFilter] = useState<string | null>(null);
  const [historyWorkingSetOnly, setHistoryWorkingSetOnly] = useState(false);
  const [isHistoryFiltersOpen, setIsHistoryFiltersOpen] = useState(false);
  const sidebarResize = useRef<{ left: number; max: number } | null>(null);
  const splitResize = useRef<{ top: number; height: number } | null>(null);
  const historyResize = useRef<{ bottom: number } | null>(null);
  const sidebarMinWidth = 200;
  const diffMinWidth = 320;
  const [groupingEnabled, setGroupingEnabled] = useState(false);
  const [groupingRules, setGroupingRules] = useState<GroupingRule[]>([]);
  const [groupingLoadedKey, setGroupingLoadedKey] = useState<string | null>(null);
  const [groupingDefaultsPending, setGroupingDefaultsPending] = useState(false);
  const [isGroupingSettingsOpen, setIsGroupingSettingsOpen] = useState(false);

  useEffect(() => {
    if (!groupingEnabled || groupingRules.length === 0) {
      setHistoryScopeFilter(null);
    }
  }, [groupingEnabled, groupingRules.length]);

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
  const [commitMessage, setCommitMessage] = useState("");

  // Queries
  const healthQuery = useHealth();
  const statusQuery = useRepoStatus();
  const historyNeedsDetails = Boolean(
    historySearch.trim() ||
      historyScopeFilter ||
      historyWorkingSetOnly ||
      (groupingEnabled && groupingRules.length > 0)
  );
  const historyQuery = useRepoHistory(historyLimit, historyNeedsDetails);
  const syncStatusQuery = useSyncStatus();
  const approvedChangesQuery = useApprovedChanges();
  const diffQuery = useDiff(selectedFile, selectedIsStaged);

  // Mutations
  const stageMutation = useStageFiles();
  const unstageMutation = useUnstageFiles();
  const commitMutation = useCommit();
  const discardMutation = useDiscardFiles();
  const pushMutation = usePush();
  const pullMutation = usePull();
  const approvedPreviewMutation = useApprovedChangesPreview();

  const isStaging = stageMutation.isPending || unstageMutation.isPending;
  const isDiscarding = discardMutation.isPending;
  const repoDir = statusQuery.data?.repo_dir;
  const repoKey = useMemo(
    () => (repoDir ? encodeURIComponent(repoDir) : "unknown"),
    [repoDir]
  );

  // Handlers
  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.health });
    queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
    queryClient.invalidateQueries({
      queryKey: queryKeys.repoHistory(historyLimit, historyNeedsDetails)
    });
    queryClient.invalidateQueries({ queryKey: queryKeys.syncStatus });
    queryClient.invalidateQueries({ queryKey: queryKeys.approvedChanges });
    if (selectedFile) {
      queryClient.invalidateQueries({
        queryKey: queryKeys.diff(selectedFile, selectedIsStaged)
      });
    }
  }, [historyLimit, historyNeedsDetails, queryClient, selectedFile, selectedIsStaged]);

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

  const workingSetPaths = useMemo(() => {
    const files = statusQuery.data?.files;
    if (!files) return [] as string[];
    return [
      ...(files.staged ?? []),
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
  }, [statusQuery.data?.files]);

  const approvedPendingPaths = useMemo(() => {
    const files = approvedChangesQuery.data?.files ?? [];
    return files
      .filter((file) => file.status === "pending" && file.relativePath)
      .map((file) => file.relativePath);
  }, [approvedChangesQuery.data?.files]);

  const approvedPendingSet = useMemo(
    () => new Set(approvedPendingPaths),
    [approvedPendingPaths]
  );

  const createGroupingRule = useCallback(
    (label: string, prefix: string, mode: GroupingRule["mode"] = "prefix"): GroupingRule => {
      return {
        id: `group-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        label,
        prefix,
        mode: mode ?? "prefix"
      };
    },
    []
  );
  const normalizeGroupingRules = useCallback(
    (rawRules: GroupingRule[]) => {
      return rawRules
        .map((rule, index) => {
          const prefix = typeof rule?.prefix === "string" ? rule.prefix : "";
          if (!prefix.trim()) return null;
          const label =
            typeof rule?.label === "string" && rule.label.trim()
              ? rule.label.trim()
              : prefix.trim();
          const mode = rule?.mode === "segment" ? "segment" : "prefix";
          const id =
            typeof rule?.id === "string" && rule.id.trim()
              ? rule.id.trim()
              : `group-${Date.now()}-${index}`;
          return { id, label, prefix, mode } as GroupingRule;
        })
        .filter((rule): rule is GroupingRule => Boolean(rule));
    },
    []
  );

  const approvedStagedPaths = useMemo(() => {
    const staged = statusQuery.data?.files?.staged ?? [];
    return staged.filter((path) => approvedPendingSet.has(path));
  }, [approvedPendingSet, statusQuery.data?.files?.staged]);

  const canUseApprovedMessage =
    approvedStagedPaths.length > 0 &&
    approvedStagedPaths.length === (statusQuery.data?.files?.staged ?? []).length;

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
  const selectedKeySet = useMemo(
    () => new Set(selectedFiles.map((entry) => selectionKey(entry))),
    [selectedFiles, selectionKey]
  );

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
      const selectedUnstaged = selectedFiles.filter((entry) => !entry.staged).map((entry) => entry.path);
      const shouldStageSelection =
        selectedUnstaged.length > 1 &&
        selectedUnstaged.some((selectedPath) => selectedPath === path);
      const pathsToStage = shouldStageSelection ? selectedUnstaged : [path];

      stageMutation.mutate(
        { paths: pathsToStage },
        {
          onSuccess: () => {
            // If we were viewing this file's unstaged diff, switch to staged
            if (selectedFile === path && !selectedIsStaged) {
              setSelectedIsStaged(true);
            }
            pathsToStage.forEach((stagedPath) => {
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(stagedPath, false)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(stagedPath, true)
              });
            });
          }
        }
      );
    },
    [stageMutation, queryClient, selectedFile, selectedIsStaged, selectedFiles]
  );

  const handleUnstageFile = useCallback(
    (path: string) => {
      const selectedStaged = selectedFiles.filter((entry) => entry.staged).map((entry) => entry.path);
      const shouldUnstageSelection =
        selectedStaged.length > 1 &&
        selectedStaged.some((selectedPath) => selectedPath === path);
      const pathsToUnstage = shouldUnstageSelection ? selectedStaged : [path];

      unstageMutation.mutate(
        { paths: pathsToUnstage },
        {
          onSuccess: () => {
            // If we were viewing this file's staged diff, switch to unstaged
            if (selectedFile === path && selectedIsStaged) {
              setSelectedIsStaged(false);
            }
            pathsToUnstage.forEach((unstagedPath) => {
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(unstagedPath, false)
              });
              queryClient.invalidateQueries({
                queryKey: queryKeys.diff(unstagedPath, true)
              });
            });
          }
        }
      );
    },
    [unstageMutation, queryClient, selectedFile, selectedIsStaged, selectedFiles]
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

  const handleStagePaths = useCallback(
    (paths: string[]) => {
      if (paths.length === 0) return;
      stageMutation.mutate({ paths });
    },
    [stageMutation]
  );

  const handleStageApproved = useCallback(() => {
    const suggestedMessage = approvedChangesQuery.data?.suggestedMessage ?? "";
    if (approvedPendingPaths.length === 0) return;

    stageMutation.mutate(
      { paths: approvedPendingPaths },
      {
        onSuccess: () => {
          if (suggestedMessage) {
            setCommitMessage(suggestedMessage);
          }
        }
      }
    );
  }, [approvedChangesQuery.data?.suggestedMessage, approvedPendingPaths, stageMutation]);

  const handleUnstageAll = useCallback(() => {
    const files = statusQuery.data?.files;
    if (!files || (files.staged?.length ?? 0) === 0) return;

    unstageMutation.mutate({ paths: files.staged ?? [] });
  }, [unstageMutation, statusQuery.data]);

  const handleDiscardPaths = useCallback(
    (paths: string[], untracked: boolean) => {
      if (paths.length === 0) return;
      discardMutation.mutate(
        { paths, untracked },
        {
          onSuccess: () => {
            if (selectedFile && paths.includes(selectedFile)) {
              setSelectedFile(undefined);
            }
            setSelectedFiles((prev) =>
              prev.filter((entry) => entry.staged || !paths.includes(entry.path))
            );
            queryClient.invalidateQueries({ queryKey: queryKeys.repoStatus });
          }
        }
      );
    },
    [discardMutation, queryClient, selectedFile]
  );

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
              setCommitMessage("");
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

  const handleUseApprovedMessage = useCallback(() => {
    if (!canUseApprovedMessage) return;

    approvedPreviewMutation.mutate(
      { paths: approvedStagedPaths },
      {
        onSuccess: (result) => {
          if (result.available && result.suggestedMessage) {
            setCommitMessage(result.suggestedMessage);
          }
        }
      }
    );
  }, [approvedPreviewMutation, approvedStagedPaths, canUseApprovedMessage]);

  const handlePush = useCallback(() => {
    pushMutation.mutate({});
  }, [pushMutation]);

  const handlePull = useCallback(() => {
    pullMutation.mutate({});
  }, [pullMutation]);

  const handleLoadMoreHistory = useCallback(() => {
    setHistoryLimit((prev) => Math.min(historyMaxLimit, prev + 50));
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.sidebarWidth", String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.changesHeight", String(changesHeight));
  }, [changesHeight]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("gct.historyHeight", String(historyHeight));
  }, [historyHeight]);

  useEffect(() => {
    if (!repoDir) return;
    if (groupingLoadedKey === repoKey) return;

    const enabledKey = `gct.grouping.${repoKey}.enabled`;
    const rulesKey = `gct.grouping.${repoKey}.rules`;
    const storedEnabled = localStorage.getItem(enabledKey);
    const storedRules = localStorage.getItem(rulesKey);
    setGroupingEnabled(storedEnabled === "true");
    if (storedRules) {
      try {
        const parsed = JSON.parse(storedRules) as GroupingRule[];
        setGroupingRules(Array.isArray(parsed) ? normalizeGroupingRules(parsed) : []);
        setGroupingDefaultsPending(false);
      } catch {
        setGroupingRules([]);
        setGroupingDefaultsPending(true);
      }
    } else {
      setGroupingRules([]);
      setGroupingDefaultsPending(true);
    }
    setGroupingLoadedKey(repoKey);
  }, [repoDir, repoKey, groupingLoadedKey, normalizeGroupingRules]);

  useEffect(() => {
    if (!repoDir || groupingLoadedKey !== repoKey) return;
    const enabledKey = `gct.grouping.${repoKey}.enabled`;
    const rulesKey = `gct.grouping.${repoKey}.rules`;
    localStorage.setItem(enabledKey, String(groupingEnabled));
    localStorage.setItem(rulesKey, JSON.stringify(groupingRules));
  }, [repoDir, repoKey, groupingLoadedKey, groupingEnabled, groupingRules]);

  useEffect(() => {
    if (!groupingDefaultsPending || !repoDir) return;
    const files = statusQuery.data?.files;
    if (!files) return;
    const allFiles = [
      ...(files.staged ?? []),
      ...(files.unstaged ?? []),
      ...(files.untracked ?? []),
      ...(files.conflicts ?? [])
    ];
    const hasScenarios = allFiles.some((path) => path.startsWith("scenarios/"));
    const hasResources = allFiles.some((path) => path.startsWith("resources/"));
    if (hasScenarios || hasResources) {
      const defaults: GroupingRule[] = [];
      if (hasScenarios) defaults.push(createGroupingRule("Scenarios", "scenarios/", "segment"));
      if (hasResources) defaults.push(createGroupingRule("Resources", "resources/", "segment"));
      setGroupingRules(defaults);
    }
    setGroupingDefaultsPending(false);
  }, [groupingDefaultsPending, repoDir, statusQuery.data?.files, createGroupingRule]);

  useEffect(() => {
    if (groupingRules.length === 0 && groupingEnabled) {
      setGroupingEnabled(false);
    }
  }, [groupingRules, groupingEnabled]);

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

    const minChanges = 200;
    const minHistory = 140;
    const dividerHeight = 6;
    const minBottom = 180;
    const clamp = () => {
      if (!sidebarRef.current || commitCollapsed) return;
      const height = sidebarRef.current.clientHeight;
      const minTop =
        changesCollapsed && historyCollapsed
          ? minChanges
          : changesCollapsed
            ? minHistory
            : historyCollapsed
              ? minChanges
              : minChanges + minHistory + dividerHeight;
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
  }, [changesHeight, changesCollapsed, historyCollapsed, commitCollapsed]);

  useEffect(() => {
    if (changesCollapsed || historyCollapsed) return;
    const minChanges = 200;
    const minHistory = 140;
    const maxHistory = Math.max(minHistory, changesHeight - minChanges);
    if (historyHeight > maxHistory) {
      setHistoryHeight(maxHistory);
    } else if (historyHeight < minHistory) {
      setHistoryHeight(Math.min(minHistory, maxHistory));
    }
  }, [changesHeight, historyHeight, changesCollapsed, historyCollapsed]);

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

  useEffect(() => {
    if (!isResizingHistory) return;

    const handleMove = (event: MouseEvent) => {
      if (!historyResize.current) return;
      const minHistory = 140;
      const minChanges = 200;
      const nextHeight = historyResize.current.bottom - event.clientY;
      const maxHeight = Math.max(minHistory, changesHeight - minChanges);
      const clampedHeight = Math.max(minHistory, Math.min(maxHeight, nextHeight));
      setHistoryHeight(clampedHeight);
    };

    const handleUp = () => {
      setIsResizingHistory(false);
      historyResize.current = null;
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
  }, [isResizingHistory, changesHeight]);

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
    if (!sidebarRef.current || commitCollapsed || (changesCollapsed && historyCollapsed)) return;
    const rect = sidebarRef.current.getBoundingClientRect();
    splitResize.current = { top: rect.top, height: rect.height };
    setIsResizingSplit(true);
  };

  const handleHistoryResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!sidebarRef.current || changesCollapsed || historyCollapsed) return;
    const rect = sidebarRef.current.getBoundingClientRect();
    historyResize.current = { bottom: rect.top + changesHeight };
    setIsResizingHistory(true);
  };

  const topCollapsed = changesCollapsed && historyCollapsed;
  const showSplitHandle = !commitCollapsed && !topCollapsed;
  const sidebarRows = (() => {
    if (topCollapsed && commitCollapsed) return "auto 0px auto";
    if (topCollapsed) return "auto 0px minmax(0, 1fr)";
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
              <div
                className="h-full min-h-0 min-w-0 grid"
                style={{
                  gridTemplateRows:
                    !changesCollapsed && !historyCollapsed
                      ? `minmax(0, 1fr) 6px minmax(0, ${historyHeight}px)`
                      : "minmax(0, 1fr)"
                }}
              >
                <div className="min-h-0 min-w-0">
                  <FileList
                    files={statusQuery.data?.files}
                    selectedFiles={selectedFiles}
                    selectedKeySet={selectedKeySet}
                    selectionKey={selectionKey}
                    syncStatus={
                      syncStatusQuery.data
                        ? {
                            ahead: syncStatusQuery.data.ahead,
                            behind: syncStatusQuery.data.behind,
                            canPush: syncStatusQuery.data.can_push,
                            canPull: syncStatusQuery.data.can_pull,
                            warning: syncStatusQuery.data.safety_warnings?.join("; ")
                          }
                        : undefined
                    }
                    approvedChanges={
                      approvedChangesQuery.data
                        ? {
                            available: approvedChangesQuery.data.available,
                            committableFiles: approvedChangesQuery.data.committableFiles,
                            warning: approvedChangesQuery.data.warning
                          }
                        : undefined
                    }
                    approvedPaths={approvedPendingSet}
                    onStageApproved={handleStageApproved}
                    isStagingApproved={isStaging}
                    onPush={handlePush}
                    onPull={handlePull}
                    isPushing={pushMutation.isPending}
                    isPulling={pullMutation.isPending}
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
                    groupingEnabled={groupingEnabled}
                    groupingRules={groupingRules}
                    onToggleGrouping={() => setGroupingEnabled((prev) => !prev)}
                    onOpenGroupingSettings={() => setIsGroupingSettingsOpen(true)}
                    onStagePaths={handleStagePaths}
                    onDiscardPaths={handleDiscardPaths}
                  />
                </div>
                <div
                  className={`${
                    !changesCollapsed && !historyCollapsed
                      ? "cursor-row-resize bg-slate-900 hover:bg-slate-800"
                      : "bg-transparent"
                  }`}
                  onMouseDown={
                    !changesCollapsed && !historyCollapsed ? handleHistoryResizeStart : undefined
                  }
                  aria-hidden="true"
                />
                <div className="min-h-0 min-w-0">
                  <GitHistory
                    lines={historyQuery.data?.lines}
                    entries={historyQuery.data?.entries}
                    isLoading={historyQuery.isLoading}
                    error={historyQuery.error}
                    collapsed={historyCollapsed}
                    onToggleCollapse={() => setHistoryCollapsed((prev) => !prev)}
                    height={historyHeight}
                    onLoadMore={handleLoadMoreHistory}
                    isFetching={historyQuery.isFetching}
                    hasMore={
                      (historyQuery.data?.lines?.length ?? 0) >= historyLimit &&
                      historyLimit < historyMaxLimit
                    }
                    searchQuery={historySearch}
                    onSearchQueryChange={setHistorySearch}
                    scopeFilter={historyScopeFilter}
                    onScopeFilterChange={setHistoryScopeFilter}
                    groupingEnabled={groupingEnabled}
                    groupingRules={groupingRules}
                    workingSetPaths={workingSetPaths}
                    workingSetOnly={historyWorkingSetOnly}
                    onWorkingSetOnlyChange={setHistoryWorkingSetOnly}
                    filtersOpen={isHistoryFiltersOpen}
                    onOpenFilters={() => setIsHistoryFiltersOpen(true)}
                    onCloseFilters={() => setIsHistoryFiltersOpen(false)}
                  />
                </div>
              </div>
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
                commitMessage={commitMessage}
                onCommitMessageChange={setCommitMessage}
                canUseApprovedMessage={canUseApprovedMessage}
                onUseApprovedMessage={handleUseApprovedMessage}
                isUsingApprovedMessage={approvedPreviewMutation.isPending}
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
      <GroupingSettingsModal
        isOpen={isGroupingSettingsOpen}
        repoDir={repoDir}
        groupingEnabled={groupingEnabled}
        onToggleGrouping={() => setGroupingEnabled((prev) => !prev)}
        rules={groupingRules}
        onChangeRules={setGroupingRules}
        onClose={() => setIsGroupingSettingsOpen(false)}
      />
    </div>
  );
}
