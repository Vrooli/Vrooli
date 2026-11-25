import {
  Profiler,
  memo,
  useDeferredValue,
  useState,
  useEffect,
  useLayoutEffect,
  useRef,
  useMemo,
  useCallback,
} from "react";
import { onProfilerRender } from "../lib/profiler";
import {
  File,
  FilePlus,
  FileX,
  FolderOpen,
  ArrowLeft,
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  EyeOff,
  ChevronDown,
  ChevronRight,
  MoreHorizontal,
  FolderTree,
  History,
  X,
  BarChart3,
  ClipboardCheck,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import { useIsMobile } from "../hooks";
import type { DiffStats, RepoFileStats } from "../lib/api";
import { ViewModeCycleButton } from "./ViewModeCycleButton";
import { ProjectTreeView } from "./ProjectTreeView";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import { SwipeActions, type SwipeAction } from "@vrooli/react-component-library/SwipeActions";
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import { getFileStats, filterFileStats, filterCategoryStats } from "../lib/metrics";
import { useApprovedChanges, useDiffStats, useProvenance } from "../lib/hooks";
import { buildRunIndex } from "../lib/runAttribution";
import { MobileContext, type FileCategory, type FileListProps, summarizeFileStats, summarizeGroupFileStats, LineStats, ChangeStatsRibbon } from "./FileListTypes";
import { VirtualizedFileSections, type ChangeListSection, type ChangeListBlock } from "./FileSection";
import { groupKindLabel } from "../lib/groupKinds";
import { RunSheet } from "./RunSheet";
import { runHue } from "../lib/runAttribution";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";

export type { GroupingRule, FileCategory, SelectedFileEntry, FileListProps } from "./FileListTypes";

type GroupDrillIn = { groupId: string; directory: string };

function FileListImpl({
  files,
  fileStats,
  previewFileStats,
  fileStatsState = fileStats ? "ready" : "loading",
  onRetryFileStats,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onIgnoreFile,
  onStageAll,
  onUnstageAll,
  pendingPaths,
  isDiscarding,
  isIgnoring,
  confirmingDiscard,
  onConfirmDiscard,
  confirmingIgnore,
  onConfirmIgnore,
  collapsed = false,
  onToggleCollapse,
  fillHeight = true,
  fileViewMode = "flat",
  groupingRules = [],
  resolvedGroups,
  groupingAvailable = false,
  onCycleViewMode,
  onStagePaths,
  onDiscardPaths,
  onSelectAnyFile,
  scrollToFile,
  onScrollComplete,
  scrollTopStore,
  onDeletePath,
  onBlameFile,
  onStageFilesWithSameName,
  onRevealInTree,
  onOpenRun,
  repoId,
  onOpenReview,
  mobileSelectionMode = false,
  onEnterSelectionMode,
  onExitSelectionMode,
  onMobileSelectFile,
  fileHotspots,
  runIndex: providedRunIndex,
  loading = false,
  loadError,
  onRetryLoad,
}: FileListProps) {
  const isMobile = useIsMobile();
  // Attribution data belongs to the Changes surface that consumes it. Keeping
  // these polling queries here prevents their refreshes from rerendering App
  // and every unrelated pane.
  const approvedChangesQuery = useApprovedChanges(repoId, !collapsed && !providedRunIndex);
  const provenanceQuery = useProvenance(repoId, !collapsed && !providedRunIndex);
  const derivedRunIndex = useMemo(
    () => buildRunIndex(approvedChangesQuery.data?.files ?? [], provenanceQuery.data?.runGroups ?? []),
    [approvedChangesQuery.data?.files, provenanceQuery.data?.runGroups],
  );
  const runIndex = providedRunIndex ?? derivedRunIndex;
  const hasStaged = (files?.staged?.length ?? 0) > 0;
  const hasUnstaged =
    (files?.unstaged?.length ?? 0) > 0 || (files?.untracked?.length ?? 0) > 0;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);
  const [compactHeader, setCompactHeader] = useState(false);
  const [confirmingGroup, setConfirmingGroup] = useState<string | null>(null);
  const [mobileGroupActionsId, setMobileGroupActionsId] = useState<string | null>(null);
  const [swipeOpenGroupId, setSwipeOpenGroupId] = useState<string | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(() => {
    const storageKey = `gct.collapsedGroups.${repoId ?? "default"}`;
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) return new Set(JSON.parse(stored) as string[]);
    } catch { /* ignore */ }
    return new Set();
  });
  const [groupDrillIn, setGroupDrillIn] = useState<GroupDrillIn | null>(null);
  const overviewScrollTopRef = useRef(0);
  const previousDrillInRef = useRef<GroupDrillIn | null>(null);
  // Persisted subsection (Modified/Untracked/…) expand state, keyed by
  // group.id + category. Only explicitly-toggled sections are stored; sections
  // absent from the map fall back to their per-category default (Untracked
  // starts collapsed). Mirrors the group-collapse mechanism above.
  const [sectionExpanded, setSectionExpanded] = useState<Record<string, boolean>>(() => {
    const storageKey = `gct.collapsedSections.${repoId ?? "default"}`;
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) return JSON.parse(stored) as Record<string, boolean>;
    } catch { /* ignore */ }
    return {};
  });
  const selectedKeys = useMemo(() => selectedKeySet ?? new Set(selectedFiles?.map(selectionKey)), [selectedKeySet, selectedFiles, selectionKey]);
  const binarySet = useMemo(
    () => new Set(files?.binary ?? []),
    [files?.binary],
  );

  // Metrics modal state
  const [metricsModal, setMetricsModal] = useState<{
    mode: "file" | "aggregate";
    stats?: DiffStats;
    path?: string;
    filteredFileStats?: RepoFileStats;
    title?: string;
    category?: FileCategory;
  } | null>(null);
  const openAggregateMetrics = useCallback(
    () => setMetricsModal({ mode: "aggregate" }),
    [],
  );

  const openFileMetrics = useCallback(
    (path: string, category?: FileCategory) => {
      const stats = getFileStats(path, fileStats);
      if (!stats) return;
      setMetricsModal({ mode: "file", stats, path, category });
    },
    [fileStats],
  );

  const openGroupMetrics = useCallback(
    (groupFiles: Record<string, string[]>, groupLabel: string) => {
      const allPaths = [
        ...(groupFiles.conflicts ?? []),
        ...(groupFiles.staged ?? []),
        ...(groupFiles.unstaged ?? []),
        ...(groupFiles.untracked ?? []),
      ];
      const filtered = filterFileStats(allPaths, fileStats);
      setMetricsModal({
        mode: "aggregate",
        filteredFileStats: filtered,
        title: `${groupLabel} — Change Metrics`,
      });
    },
    [fileStats],
  );

  const openGroupCategoryMetrics = useCallback(
    (paths: string[], category: "staged" | "unstaged" | "untracked", groupLabel: string) => {
      const catLabel = category.charAt(0).toUpperCase() + category.slice(1);
      const filtered = filterCategoryStats(paths, category, fileStats);
      setMetricsModal({
        mode: "aggregate",
        filteredFileStats: filtered,
        title: `${catLabel} (${groupLabel})`,
      });
    },
    [fileStats],
  );

  // Fetch enhanced stats (hunks, density, comments) on-demand when file metrics modal is open
  const enhancedQuery = useDiffStats(
    metricsModal?.path,
    metricsModal?.category === "staged",
    metricsModal?.category === "untracked",
    metricsModal !== null && metricsModal.mode === "file",
    repoId,
  );

  // Mobile file actions state
  const [mobileActionFile, setMobileActionFile] = useState<string | null>(null);
  const [openRunId, setOpenRunId] = useState<string | null>(null);
  const [listFilter, setListFilter] = useState<"all" | "agents" | "staged" | "conflicts">("all");
  const openRun = onOpenRun ?? setOpenRunId;
  const attributionEntries = useMemo(() => runIndex ? Array.from(runIndex.entries()) : [], [runIndex]);
  const runFilesById = useMemo(() => {
    const byRun = new Map<string, {
      paths: string[];
      attribution: (typeof attributionEntries)[number][1];
    }>();
    for (const [path, attribution] of attributionEntries) {
      const run = byRun.get(attribution.runId);
      if (run) {
        run.paths.push(path);
      } else {
        byRun.set(attribution.runId, { paths: [path], attribution });
      }
    }
    return byRun;
  }, [attributionEntries]);
  const stagedPathSet = useMemo(() => new Set(files?.staged ?? []), [files?.staged]);
  const runStats = useMemo(() => ({
    ...(fileStats?.untracked ?? {}),
    ...(fileStats?.unstaged ?? {}),
    ...(fileStats?.staged ?? {}),
  }), [fileStats]);
  const mobileActionFileInfo = useMemo(() => {
    if (!mobileActionFile) return null;
    const isStaged = files?.staged?.includes(mobileActionFile) ?? false;
    const isUnstaged = files?.unstaged?.includes(mobileActionFile) ?? false;
    const isUntracked = files?.untracked?.includes(mobileActionFile) ?? false;
    const isConflict = files?.conflicts?.includes(mobileActionFile) ?? false;
    return {
      path: mobileActionFile,
      isStaged,
      isUnstaged,
      isUntracked,
      isConflict,
    };
  }, [mobileActionFile, files]);
  const openRunFiles = openRunId ? runFilesById.get(openRunId) : undefined;

  // Context menu state for right-click file actions.
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    file: string;
  } | null>(null);

  const handleFileContextMenu = useCallback((file: string, event: React.MouseEvent) => {
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      file,
    });
  }, []);

  const handleCloseContextMenu = useCallback(() => {
    setContextMenu(null);
  }, []);

  const contextMenuItems = useMemo<ContextMenuItem[]>(() => {
    if (!contextMenu) return [];
    const items: ContextMenuItem[] = [];
    if (onBlameFile) items.push({
        id: "history",
        label: "View File History",
        icon: <History className="h-4 w-4" />,
        onSelect: () => onBlameFile(contextMenu.file),
    });
    if (onRevealInTree) items.push({
      id: "reveal",
      label: "Reveal in file tree",
      icon: <FolderTree className="h-4 w-4" />,
      onSelect: () => onRevealInTree(contextMenu.file),
      testId: "reveal-in-tree-action",
    });
    const attribution = runIndex?.get(contextMenu.file);
    if (attribution) items.push({
      id: "run",
      label: "Show the run that changed this",
      icon: <History className="h-4 w-4" />,
      onSelect: () => openRun(attribution.runId),
    });
    const basename = contextMenu.file.split("/").pop() || contextMenu.file;
    const changedFiles = [
      ...(files?.staged ?? []),
      ...(files?.unstaged ?? []),
      ...(files?.untracked ?? []),
      ...(files?.conflicts ?? []),
    ];
    const sameNameCount = new Set(
      changedFiles.filter((candidate) => (candidate.split("/").pop() || candidate) === basename),
    ).size;
    if (onStageFilesWithSameName && sameNameCount > 1) items.push({
      id: "stage-same-name",
      label: `Stage all changed files named ${basename}`,
      icon: <ClipboardCheck className="h-4 w-4" />,
      onSelect: () => onStageFilesWithSameName(contextMenu.file),
    });
    return items;
  }, [contextMenu, files, onBlameFile, onRevealInTree, onStageFilesWithSameName, runIndex, openRun]);

  // Mobile long-press enters selection mode; tap in selection mode toggles
  const handleLongPress = useCallback(
    (file: string, staged: boolean) => {
      if (mobileSelectionMode) return;
      onEnterSelectionMode?.(file, staged);
    },
    [mobileSelectionMode, onEnterSelectionMode],
  );

  const handleMobileTap = useCallback(
    (file: string, staged: boolean, mode: "toggle" | "range") => {
      onMobileSelectFile?.(file, staged, mode);
    },
    [onMobileSelectFile],
  );

  // Auto-exit selection mode when all files deselected
  useEffect(() => {
    if (mobileSelectionMode && (selectedFiles?.length ?? 0) === 0) {
      onExitSelectionMode?.();
    }
  }, [mobileSelectionMode, selectedFiles?.length, onExitSelectionMode]);

  // Suppress bottom sheet in selection mode
  const handleOpenMobileActions = useCallback(
    (file: string) => {
      if (mobileSelectionMode) return;
      setMobileActionFile(file);
    },
    [mobileSelectionMode],
  );

  // Selection toolbar: compute action availability
  const selectionHasUnstaged = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles) return false;
    return selectedFiles.some((entry) => !entry.staged);
  }, [mobileSelectionMode, selectedFiles]);
  const selectionHasStaged = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles) return false;
    return selectedFiles.some((entry) => entry.staged);
  }, [mobileSelectionMode, selectedFiles]);
  const selectionHasDiscardable = useMemo(() => {
    if (!mobileSelectionMode || !selectedFiles || !files) return false;
    const unstaged = new Set(files.unstaged ?? []);
    const untracked = new Set(files.untracked ?? []);
    return selectedFiles.some((entry) => !entry.staged && (unstaged.has(entry.path) || untracked.has(entry.path)));
  }, [mobileSelectionMode, selectedFiles, files]);

  const groupingActive = fileViewMode === "grouped" &&
    (resolvedGroups !== undefined ? resolvedGroups.length > 0 : groupingRules.length > 0);
  const totalStats = useMemo(() => {
    if (!files) return undefined;
    const stagedStats = summarizeFileStats(
      files.staged ?? [],
      fileStats?.staged,
    );
    const unstagedStats = summarizeFileStats(
      files.unstaged ?? [],
      fileStats?.unstaged,
    );
    const untrackedStats = summarizeFileStats(
      files.untracked ?? [],
      fileStats?.untracked,
    );
    const summary = { additions: 0, deletions: 0, files: 0 };
    const sources = [stagedStats, unstagedStats, untrackedStats];
    let hasStats = false;
    sources.forEach((source) => {
      if (!source) return;
      hasStats = true;
      summary.additions += source.additions;
      summary.deletions += source.deletions;
      summary.files += source.files;
    });
    return hasStats ? summary : undefined;
  }, [files, fileStats]);
  const handleDiscardUnstaged = useCallback(
    (path: string) => onDiscardFile(path, false),
    [onDiscardFile],
  );
  const handleDiscardUntracked = useCallback(
    (path: string) => onDiscardFile(path, true),
    [onDiscardFile],
  );
  const handleIgnoreFile = useCallback(
    (path: string, level?: "project" | "group", groupDir?: string) => onIgnoreFile(path, level, groupDir),
    [onIgnoreFile],
  );
  const toggleGroupCollapse = useCallback((groupId: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) {
        next.delete(groupId);
      } else {
        next.add(groupId);
      }
      const storageKey = `gct.collapsedGroups.${repoId ?? "default"}`;
      try {
        localStorage.setItem(storageKey, JSON.stringify([...next]));
      } catch { /* ignore */ }
      return next;
    });
  }, [repoId]);

  const isSectionExpanded = useCallback(
    (groupId: string, category: FileCategory, defaultExpanded: boolean) => {
      const key = `${groupId}::${category}`;
      return sectionExpanded[key] ?? defaultExpanded;
    },
    [sectionExpanded],
  );
  const toggleSectionCollapse = useCallback(
    (groupId: string, category: FileCategory, defaultExpanded: boolean) => {
      const key = `${groupId}::${category}`;
      setSectionExpanded((prev) => {
        const current = key in prev ? prev[key] : defaultExpanded;
        const next = { ...prev, [key]: !current };
        const storageKey = `gct.collapsedSections.${repoId ?? "default"}`;
        try {
          localStorage.setItem(storageKey, JSON.stringify(next));
        } catch { /* ignore */ }
        return next;
      });
    },
    [repoId],
  );

  useEffect(() => {
    if (!scrollAreaRef.current || typeof ResizeObserver === "undefined") return;

    const update = () => {
      const width = scrollAreaRef.current?.clientWidth ?? 0;
      // Account for: status badge (~28px), file icon (~22px), action buttons (~80px),
      // badges (binary ~60px), padding (~16px) = ~210px total
      const usable = Math.max(0, width - 180);
      const nextMax = Math.max(12, Math.min(100, Math.floor(usable / 7.5)));
      setMaxPathChars(nextMax);
    };

    // Defer initial measurement to ensure layout is complete after expanding
    const rafId = requestAnimationFrame(update);
    const observer = new ResizeObserver(update);
    observer.observe(scrollAreaRef.current);
    return () => {
      cancelAnimationFrame(rafId);
      observer.disconnect();
    };
    // Re-run when collapsed changes to re-observe the new ScrollArea element
  }, [collapsed]);

  // Track card width to swap +/- stats for a compact icon in the header
  useEffect(() => {
    if (!cardRef.current || typeof ResizeObserver === "undefined") return;
    const update = () => {
      const width = cardRef.current?.clientWidth ?? 0;
      setCompactHeader(width < 500);
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(cardRef.current);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!groupingActive) {
      setConfirmingGroup(null);
    }
  }, [groupingActive]);

  // Navigation may target a virtual row in a collapsed group/category. Open
  // the owning section before the virtualizer resolves the destination.
  useEffect(() => {
    if (!scrollToFile || fileViewMode === "tree") return;
    const categories = (["conflicts", "staged", "unstaged", "untracked"] as const)
      .filter(category => files?.[category]?.includes(scrollToFile));
    const group = groupingActive ? resolvedGroups?.find(candidate => candidate.files.includes(scrollToFile)) : undefined;
    if (!categories.length || (groupingActive && !group)) {
      onScrollComplete?.();
      return;
    }
    setListFilter("all");
    const groupId = group?.key ?? "__flat__";
    if (group) setCollapsedGroups(prev => {
      if (!prev.has(groupId)) return prev;
      const next = new Set(prev);
      next.delete(groupId);
      try { localStorage.setItem(`gct.collapsedGroups.${repoId ?? "default"}`, JSON.stringify([...next])); } catch { /* ignore */ }
      return next;
    });
    setSectionExpanded(prev => {
      if (categories.every(category => prev[`${groupId}::${category}`] === true)) return prev;
      const next = { ...prev };
      for (const category of categories) next[`${groupId}::${category}`] = true;
      try { localStorage.setItem(`gct.collapsedSections.${repoId ?? "default"}`, JSON.stringify(next)); } catch { /* ignore */ }
      return next;
    });
  }, [scrollToFile, fileViewMode, files, groupingActive, resolvedGroups, onScrollComplete, repoId]);

  // Persist the Changes list scroll position (mobile only, when a store is
  // supplied). The store is a ref assignment, so writing on every scroll event
  // is cheap and always captures the latest position (no debounce needed).
  const handleScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      if (scrollTopStore) {
        scrollTopStore.current = event.currentTarget.scrollTop;
      }
    },
    [scrollTopStore],
  );

  // Restore the saved scroll position after the panel content has had two
  // layout frames to settle. The scroller is explicitly owned by Changes, so
  // this never moves the embedding document or host iframe viewport.
  // Skipped when a scrollToFile target is pending so the scroll-into-view path
  // (above) wins instead of fighting the restore.
  useLayoutEffect(() => {
    if (!scrollTopStore || scrollToFile) return;
    const el = scrollAreaRef.current;
    const restore = () => el?.scrollTo({ top: scrollTopStore.current, behavior: "auto" });
    let secondFrame: number | undefined;
    const firstFrame = requestAnimationFrame(() => { secondFrame = requestAnimationFrame(restore); });
    return () => {
      cancelAnimationFrame(firstFrame);
      if (secondFrame !== undefined) cancelAnimationFrame(secondFrame);
    };
    // Mount-only: intentionally not re-running on scrollTopStore/scrollToFile changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filterPaths = useCallback((category: FileCategory, paths?: string[]) => {
    if (listFilter === "all") return paths ?? [];
    return (paths ?? []).filter((path) => {
    if (listFilter === "agents") return runIndex?.has(path) ?? false;
    return category === listFilter;
    });
  }, [listFilter, runIndex]);

  const groupedSections = useMemo(() => {
    if (!groupingActive) return [];
    const categories: Array<[FileCategory, string[]]> = [
      ["conflicts", filterPaths("conflicts", files?.conflicts)],
      ["staged", filterPaths("staged", files?.staged)],
      ["unstaged", filterPaths("unstaged", files?.unstaged)],
      ["untracked", filterPaths("untracked", files?.untracked)],
    ];
    // A single membership lookup per group path avoids probing every category
    // set for every file. A bit mask preserves overlapping categories without
    // allocating a small membership array for each path.
    const categoryMaskByPath = new Map<string, number>();
    for (const [categoryIndex, [category, paths]] of categories.entries()) {
      const bit = 1 << categoryIndex;
      for (const path of paths) {
        categoryMaskByPath.set(path, (categoryMaskByPath.get(path) ?? 0) | bit);
      }
    }
    const visibleGroups = (resolvedGroups ?? []).map((group) => {
      const groupedFiles: Record<FileCategory, string[]> = {
        conflicts: [], staged: [], unstaged: [], untracked: [],
      };
      for (const path of group.files) {
        const mask = categoryMaskByPath.get(path) ?? 0;
        for (const [categoryIndex, [category]] of categories.entries()) {
          if (mask & (1 << categoryIndex)) groupedFiles[category].push(path);
        }
      }
      return {
        id: group.key,
        label: group.label,
        kind: group.kind,
        source: group.source,
        displayPrefix: group.root ?? "",
        files: groupedFiles,
        stageable: [...groupedFiles.unstaged, ...groupedFiles.untracked, ...groupedFiles.conflicts],
        discardCount: groupedFiles.unstaged.length + groupedFiles.untracked.length,
        count: Object.values(groupedFiles).reduce((total, paths) => total + paths.length, 0),
        lineStats: summarizeGroupFileStats(groupedFiles, fileStats),
        previewLineStats: summarizeGroupFileStats(groupedFiles, previewFileStats),
        runs: [...new Set(Object.values(groupedFiles).flatMap(paths => paths.map(path => runIndex?.get(path)?.runId)).filter((id): id is string => Boolean(id)))],
      };
    }).filter((group) =>
      Object.values(group.files).some((paths) => paths.length > 0) || group.id === groupDrillIn?.groupId,
    );

    const bandSummaries = new Map<string, { groups: number; files: number }>();
    for (const group of visibleGroups) {
      const bandKind = (group.source === "contract" ? group.kind : group.source) ?? "other";
      const summary = bandSummaries.get(bandKind) ?? { groups: 0, files: 0 };
      summary.groups += 1;
      summary.files += group.count;
      bandSummaries.set(bandKind, summary);
    }

    return visibleGroups.map((group) => ({
      ...group,
      bandSummary: bandSummaries.get((group.source === "contract" ? group.kind : group.source) ?? "other") ?? { groups: 0, files: 0 },
    }));
  }, [files, fileStats, previewFileStats, groupingActive, resolvedGroups, filterPaths, runIndex, groupDrillIn?.groupId]);

  const mobileGroupActions = groupedSections.find((group) => group.id === mobileGroupActionsId) ?? null;

  const renderGroupHeader = useCallback((group: (typeof groupedSections)[number], index: number) => {
    const { stageable, discardCount, count: groupCount, runs } = group;
    const discardTracked = group.files.unstaged;
    const discardUntracked = group.files.untracked;
    const isGroupCollapsed = collapsedGroups.has(group.id);

    const previous = groupedSections[index - 1];
    const bandKind = group.source === "contract" ? group.kind : group.source;
    const previousBandKind = previous?.source === "contract" ? previous.kind : previous?.source;
    const showBand = bandKind !== previousBandKind;
    const swipeActions: SwipeAction[] = [
      {
        id: "browse",
        label: "Browse",
        icon: <FolderOpen className="h-4 w-4" aria-hidden="true" />,
        onSelect: () => setGroupDrillIn({ groupId: group.id, directory: "" }),
      },
      ...(!mobileSelectionMode && onStagePaths && stageable.length > 0 ? [{
        id: "stage",
        label: "Stage all",
        icon: <Plus className="h-4 w-4" aria-hidden="true" />,
        tone: "primary" as const,
        onSelect: () => onStagePaths(stageable),
      }] : []),
      ...(!mobileSelectionMode && onDiscardPaths && discardCount > 0 ? [{
        id: "discard",
        label: "Discard",
        icon: <Trash2 className="h-4 w-4" aria-hidden="true" />,
        tone: "destructive" as const,
        onSelect: () => setConfirmingGroup(group.id),
      }] : []),
    ];
    const groupHeader = (
        <div
          className={`grid grid-cols-[minmax(0,1fr)_auto] items-center gap-2 px-3 py-2 ${isGroupCollapsed ? "" : "border-b border-slate-800/70"}`}
        >
          <button
            type="button"
            className="flex w-full min-w-0 items-center gap-2 overflow-hidden rounded px-1 -ml-1 text-left transition-colors hover:bg-slate-800/30"
            onClick={() => toggleGroupCollapse(group.id)}
            aria-expanded={!isGroupCollapsed}
            data-testid={`file-group-toggle-${group.id}`}
          >
            {isGroupCollapsed ? (
              <ChevronRight className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
            ) : (
              <ChevronDown className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
            )}
            <div className="min-w-0 flex-1 overflow-hidden">
              <div className={`truncate font-semibold text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`} title={group.label}>
                {group.label}
              </div>
              {group.displayPrefix && group.source !== "contract" && (
                <div className={`truncate text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`} title={group.displayPrefix}>
                  {group.displayPrefix}
                </div>
              )}
            </div>
          </button>
          <div className={`flex shrink-0 flex-nowrap items-center justify-end gap-1 text-slate-500 ${isMobile ? "text-sm" : "text-xs"}`}>
            {isGroupCollapsed && (
              <ChangeStatsRibbon
                stats={fileStatsState === "refreshing" ? group.previewLineStats : group.lineStats}
                label={group.label}
                state={fileStatsState}
                onClick={() => openGroupMetrics(group.files, group.label)}
              />
            )}
            {isGroupCollapsed && runIndex && (() => {
              return runs.length > 0 ? <span className="flex items-center gap-1" data-testid="group-run-dots">{runs.slice(0, 3).map((runId) => <span key={runId} className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: runHue(runId) }} />)}{runs.length > 3 && <span>+{runs.length - 3}</span>}</span> : null;
            })()}
            {onOpenReview && group.kind === "scenario" && group.displayPrefix ? (
              <button
                type="button"
                className="inline-flex h-8 shrink-0 items-center gap-1 whitespace-nowrap rounded px-2 text-xs text-slate-400 transition-colors hover:bg-slate-800/60 hover:text-slate-200"
                onClick={(e) => {
                  e.stopPropagation();
                  onOpenReview(group.displayPrefix?.split("/")[1] ?? "");
                }}
                title="Open scenario review"
              >
                {groupCount !== undefined && <span>{groupCount} files</span>}
                <ClipboardCheck className="h-3.5 w-3.5" />
              </button>
            ) : (
              <button
                type="button"
                className="shrink-0 cursor-pointer whitespace-nowrap px-1 hover:underline decoration-slate-600"
                onClick={(e) => {
                  e.stopPropagation();
                  openGroupMetrics(group.files, group.label);
                }}
              >
                {groupCount} files
              </button>
            )}
            {!isGroupCollapsed && compactHeader && !mobileSelectionMode && (
              <IconButton
                size="xs"
                surface="ghost"
                denseTapTarget
                className="!h-8 !w-8 !min-h-0 !min-w-0 !shrink-0 !border-0 !shadow-none !text-slate-300"
                onClick={() => setMobileGroupActionsId(group.id)}
                aria-label={`Actions for ${group.label}`}
                title={`Actions for ${group.label}`}
                data-testid={`file-group-actions-${group.id}`}
              ><MoreHorizontal className="h-4 w-4" /></IconButton>
            )}
            {!isGroupCollapsed && !compactHeader && (
              <IconButton
                size="xs"
                surface="ghost"
                denseTapTarget
                className="!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !shadow-none !text-sky-300"
                onClick={() => setGroupDrillIn({ groupId: group.id, directory: "" })}
                aria-label={`Open ${group.label} group`}
                title={`Browse ${group.label} folders`}
                data-testid={`file-group-open-${group.id}`}
              ><FolderOpen className="h-4 w-4" /></IconButton>
            )}
            {!compactHeader &&
              !mobileSelectionMode &&
              !isGroupCollapsed &&
              stageable.length > 0 &&
              onStagePaths && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => onStagePaths(stageable)}
                  disabled={false}
                  className="h-7 px-2"
                  title="Stage All"
                  style={{ background: "transparent" }}
                >Stage All</Button>
              )}
            {!compactHeader &&
              !mobileSelectionMode &&
              !isGroupCollapsed &&
              discardCount > 0 &&
              onDiscardPaths && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setConfirmingGroup(group.id)}
                  disabled={isDiscarding}
                  className="h-7 border-red-400/40 px-2 text-red-200 hover:bg-red-900/20"
                  title="Discard All"
                  style={{ background: "transparent" }}
                >Discard All</Button>
              )}
          </div>
        </div>
    );
    return (
      <>
      {showBand && (
        <div role="region" aria-label={`${groupKindLabel(group.kind, group.source)} group band`} className="relative z-10 flex min-h-[30px] items-center gap-2 border-y border-slate-800 bg-slate-950 px-2 py-1 text-[10px] font-semibold uppercase tracking-wider text-slate-500" data-testid={`file-kind-band-${bandKind || "other"}`}>
          <span>{groupKindLabel(group.kind, group.source)}</span>
          <span className="h-px flex-1 bg-slate-800" />
          <span>{group.bandSummary.groups} groups · {group.bandSummary.files} files</span>
        </div>
      )}
      <div
        key={group.id}
        className="border-b border-slate-800/80"
        data-testid={`file-group-${group.id}`}
      >
        {compactHeader && !isGroupCollapsed && !mobileSelectionMode ? (
          <SwipeActions
            actions={swipeActions}
            label={`${group.label} actions`}
            releaseMode="rest"
            open={swipeOpenGroupId === group.id}
            onOpenChange={(open: boolean) => setSwipeOpenGroupId(open ? group.id : null)}
            testId={`file-group-swipe-${group.id}`}
          >
            {groupHeader}
          </SwipeActions>
        ) : groupHeader}
        {!isGroupCollapsed && confirmingGroup === group.id &&
              discardCount > 0 && (
                <div className="flex items-center justify-between gap-2 px-3 py-2 text-xs text-red-200 bg-red-950/30 border-b border-red-900/40">
                  <span>
                    Discard {discardCount} changes in this
                    group?
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      className="px-2 py-1 rounded border border-red-400/40 text-red-100 hover:bg-red-900/30"
                      onClick={() => {
                        if (discardTracked.length > 0) {
                          onDiscardPaths?.(
                            discardTracked,
                            false,
                          );
                        }
                        if (discardUntracked.length > 0) {
                          onDiscardPaths?.(
                            discardUntracked,
                            true,
                          );
                        }
                        setConfirmingGroup(null);
                      }}
                    >
                      Discard
                    </button>
                    <button
                      type="button"
                      className="px-2 py-1 rounded border border-slate-600 text-slate-200 hover:bg-slate-800/50"
                      onClick={() => setConfirmingGroup(null)}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
      </div>
      </>
    );
  }, [groupedSections, collapsedGroups, isMobile, runIndex, onOpenReview,
    mobileSelectionMode, onStagePaths, compactHeader, onDiscardPaths,
    isDiscarding, confirmingGroup, toggleGroupCollapse, openGroupMetrics, setGroupDrillIn]);


  const listBlocks = useMemo<ChangeListBlock[]>(() => {
    const sectionsFor = (groupId: string, categoryFiles: Partial<Record<FileCategory, string[]>>, label?: string): ChangeListBlock[] => {
      const categories: FileCategory[] = ["conflicts", "staged", "unstaged", "untracked"];
      return categories.map((category) => {
        const paths = filterPaths(category, categoryFiles[category]);
        const statsCategory = category === "conflicts" ? "unstaged" : category;
        const section: ChangeListSection = {
          key: `${groupId}:${category}`, category, files: paths,
          title: category === "unstaged" ? "Modified" : category.charAt(0).toUpperCase() + category.slice(1),
          expanded: isSectionExpanded(groupId, category, category !== "untracked"),
          onToggle: () => toggleSectionCollapse(groupId, category, category !== "untracked"),
          icon: category === "conflicts" ? <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
            : category === "staged" ? <FilePlus className="h-3.5 w-3.5 text-emerald-500" />
            : category === "unstaged" ? <FileX className="h-3.5 w-3.5 text-amber-500" />
            : <File className="h-3.5 w-3.5 text-slate-500" />,
          actionIcon: category === "staged" ? <Minus className="h-3 w-3 text-slate-400" /> : <Plus className="h-3 w-3 text-slate-400" />,
          actionLabel: category === "staged" ? "Unstage file" : "Stage file",
          onAction: category === "staged" ? onUnstageFile : onStageFile,
          onDiscard: category === "unstaged" ? handleDiscardUnstaged : category === "untracked" ? handleDiscardUntracked : undefined,
          changeStats: summarizeFileStats(paths, fileStats?.[statsCategory]),
          metricsState: fileStatsState,
          onStatsClick: label ? () => openGroupCategoryMetrics(paths, statsCategory, label) : openAggregateMetrics,
        };
        return { type: "section", section };
      });
    };
    if (!groupingActive) return sectionsFor("__flat__", files ?? {});
    if (groupDrillIn) {
      const group = groupedSections.find((candidate) => candidate.id === groupDrillIn.groupId);
      if (group) {
        const categories: FileCategory[] = ["conflicts", "staged", "unstaged", "untracked"];
        const root = group.displayPrefix.replace(/^\/+|\/+$/g, "");
        const directFiles: Partial<Record<FileCategory, string[]>> = {};
        const folders = new Map<string, Record<FileCategory, string[]>>();

        for (const category of categories) {
          directFiles[category] = [];
          for (const path of group.files[category]) {
            const relative = root && path.startsWith(`${root}/`) ? path.slice(root.length + 1) : path;
            const directoryPrefix = groupDrillIn.directory ? `${groupDrillIn.directory}/` : "";
            if (directoryPrefix && !relative.startsWith(directoryPrefix)) continue;
            const withinDirectory = directoryPrefix ? relative.slice(directoryPrefix.length) : relative;
            const [child, ...remaining] = withinDirectory.split("/");
            if (!child) continue;
            if (remaining.length === 0) {
              directFiles[category]?.push(path);
              continue;
            }
            const childFiles = folders.get(child) ?? {
              conflicts: [], staged: [], unstaged: [], untracked: [],
            };
            childFiles[category].push(path);
            folders.set(child, childFiles);
          }
        }

        const folderRows: ChangeListBlock[] = [...folders.entries()]
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([name, folderFiles]) => {
            const allPaths = categories.flatMap((category) => folderFiles[category]);
            const stageable = [
              ...folderFiles.conflicts,
              ...folderFiles.unstaged,
              ...folderFiles.untracked,
            ];
            const directory = [groupDrillIn.directory, name].filter(Boolean).join("/");
            return {
              type: "header",
              key: `folder:${group.id}:${directory}`,
              estimateSize: isMobile ? 56 : 44,
              render: () => (
                <div className="flex min-h-11 items-center gap-2 border-b border-slate-800/80 px-3 py-1.5" data-testid="change-folder-row">
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 text-left text-sm text-slate-200 hover:text-white"
                    onClick={() => setGroupDrillIn({ groupId: group.id, directory })}
                    aria-label={`Open folder ${directory}`}
                    data-testid="change-folder-open"
                  >
                    <FolderOpen className="h-4 w-4 shrink-0 text-sky-300" />
                    <span className="truncate">{name}</span>
                    <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" />
                  </button>
                  <span className="shrink-0 text-xs text-slate-500">{allPaths.length} files</span>
                  {stageable.length > 0 && onStagePaths && (
                    <IconButton
                      size="xs"
                      surface="ghost"
                      denseTapTarget
                      className="!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !shadow-none !text-emerald-300"
                      onClick={() => onStagePaths(stageable)}
                      aria-label={`Stage ${stageable.length} changed files in ${directory}`}
                      title="Stage folder changes"
                      data-testid="change-folder-stage"
                    ><Plus className="h-4 w-4" /></IconButton>
                  )}
                </div>
              ),
            };
          });

        const allScopedPaths = categories.flatMap((category) => [
          ...(directFiles[category] ?? []),
          ...[...folders.values()].flatMap((folderFiles) => folderFiles[category]),
        ]);
        const stageablePathSet = new Set([
          ...group.files.unstaged,
          ...group.files.untracked,
          ...group.files.conflicts,
        ]);
        const stageable = allScopedPaths.filter((path) => stageablePathSet.has(path));
        const segments = groupDrillIn.directory.split("/").filter(Boolean);
        const breadcrumbRows: ChangeListBlock[] = [{
          type: "header",
          key: `group-drill:${group.id}:${groupDrillIn.directory}`,
          estimateSize: isMobile ? 100 : 76,
          render: () => (
            <div className="border-b border-slate-800 bg-slate-950/95 px-3 py-2" data-testid="change-group-drill-header">
              <div className="flex min-h-9 items-center gap-2">
                <button
                  type="button"
                  className="inline-flex min-h-9 shrink-0 items-center gap-1 rounded px-1 text-sm text-sky-300 hover:bg-slate-800/70"
                  onClick={() => {
                    if (!segments.length) setGroupDrillIn(null);
                    else setGroupDrillIn({ groupId: group.id, directory: segments.slice(0, -1).join("/") });
                  }}
                  aria-label={segments.length ? "Go to parent folder" : "Back to all change groups"}
                  data-testid="change-group-drill-back"
                ><ArrowLeft className="h-4 w-4" />{segments.length ? "Back" : "Changes"}</button>
                <nav className="flex min-w-0 items-center gap-1 overflow-hidden text-xs text-slate-400" aria-label="Change location">
                  <button type="button" className="truncate hover:text-slate-200" onClick={() => setGroupDrillIn({ groupId: group.id, directory: "" })}>{group.label}</button>
                  {segments.map((segment, index) => (
                    <span key={`${index}:${segment}`} className="flex min-w-0 items-center gap-1">
                      <ChevronRight className="h-3 w-3 shrink-0" />
                      <button type="button" className="truncate hover:text-slate-200" onClick={() => setGroupDrillIn({ groupId: group.id, directory: segments.slice(0, index + 1).join("/") })}>{segment}</button>
                    </span>
                  ))}
                </nav>
              </div>
              <div className="mt-1 flex items-center justify-between gap-2 pl-1 text-xs text-slate-500">
                <span>{allScopedPaths.length} changed files</span>
                {stageable.length > 0 && onStagePaths && (
                  <Button
                    variant="secondary"
                    size="sm"
                    className="h-8 px-2 text-emerald-200"
                    onClick={() => onStagePaths(stageable)}
                    data-testid="change-group-scope-stage"
                    style={{ background: "transparent" }}
                  >
                    <Plus className="mr-1 h-3.5 w-3.5" />Stage {segments.length ? "folder" : "group"}
                  </Button>
                )}
              </div>
            </div>
          ),
        }];
        const emptyRow: ChangeListBlock[] = allScopedPaths.length === 0 ? [{
          type: "header",
          key: `group-drill-empty:${group.id}:${groupDrillIn.directory}`,
          estimateSize: 64,
          render: () => (
            <div className="px-4 py-8 text-center text-sm text-slate-500" data-testid="change-group-empty">
              No changed files match this filter.
            </div>
          ),
        }] : [];
        return [
          ...breadcrumbRows,
          ...folderRows,
          ...emptyRow,
          ...sectionsFor(`drill:${group.id}:${groupDrillIn.directory}`, directFiles, `${group.label}/${groupDrillIn.directory}`),
        ];
      }
    }
    return groupedSections.flatMap((group, index): ChangeListBlock[] => group.count === 0 ? [] : [
      { type: "header", key: `group:${group.id}`, render: () => renderGroupHeader(group, index), estimateSize: isMobile ? 80 : 72 },
      ...(collapsedGroups.has(group.id) ? [] : sectionsFor(group.id, group.files, group.label)),
    ]);
  }, [files, fileStats, groupedSections, groupingActive, collapsedGroups, filterPaths,
    isSectionExpanded, toggleSectionCollapse, onStageFile, onUnstageFile,
    handleDiscardUnstaged, handleDiscardUntracked, openAggregateMetrics, openGroupCategoryMetrics,
    isMobile, renderGroupHeader, groupDrillIn, onStagePaths]);

  const deferredListBlocks = useDeferredValue(listBlocks);

  useLayoutEffect(() => {
    const element = scrollAreaRef.current;
    const previous = previousDrillInRef.current;
    if (!element) return;

    if (!previous && groupDrillIn) {
      overviewScrollTopRef.current = element.scrollTop;
      element.scrollTop = 0;
    } else if (previous && !groupDrillIn) {
      element.scrollTop = overviewScrollTopRef.current;
      if (scrollTopStore) scrollTopStore.current = overviewScrollTopRef.current;
    } else if (previous && groupDrillIn && (
      previous.groupId !== groupDrillIn.groupId || previous.directory !== groupDrillIn.directory
    )) {
      element.scrollTop = 0;
    }
    previousDrillInRef.current = groupDrillIn;
  }, [groupDrillIn, scrollTopStore]);

  useEffect(() => {
    if (groupDrillIn && !resolvedGroups?.some((group) => group.key === groupDrillIn.groupId)) {
      setGroupDrillIn(null);
    }
  }, [groupDrillIn, resolvedGroups]);

  const handleCycleViewMode = onCycleViewMode ?? (() => {});
  const totalFilesCount =
    (files?.conflicts?.length ?? 0) +
    (files?.staged?.length ?? 0) +
    (files?.unstaged?.length ?? 0) +
    (files?.untracked?.length ?? 0);

  return (
    <MobileContext.Provider value={isMobile}>
      <Card
        ref={cardRef}
        className={`flex min-w-0 flex-col ${isMobile ? "!m-0 !rounded-none !border-0 !bg-slate-950" : ""} ${fillHeight ? "h-full" : "h-auto"}`}
        data-testid="file-list-panel"
      >
        <CardHeader className="flex-row items-center justify-between gap-2 space-y-0 !border-0 bg-slate-900/90 px-3 py-2 min-w-0">
          <CardTitle className="flex min-w-0 flex-1 items-center gap-2">
            <IconButton
              size="xs"
              surface="ghost"
              denseTapTarget
              className="!h-7 !w-7 !min-h-0 !min-w-0 !border-0 !shadow-none"
              onClick={handleToggleCollapse}
              aria-label={collapsed ? "Expand changes" : "Collapse changes"}
            >
              {collapsed ? (
                <ChevronRight className="h-3 w-3 text-slate-400" />
              ) : (
                <ChevronDown className="h-3 w-3 text-slate-400" />
              )}
            </IconButton>
            <span className="truncate">Changes</span>
            {compactHeader ? (
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  setMetricsModal({ mode: "aggregate" });
                }}
                aria-label="View change metrics"
                size="xs"
                surface="ghost"
                denseTapTarget
              >
                <BarChart3 className="h-4 w-4 text-slate-400" />
              </IconButton>
            ) : (
              <LineStats
                stats={totalStats}
                onClick={() => setMetricsModal({ mode: "aggregate" })}
              />
            )}
          </CardTitle>
          <div className="relative z-20 flex shrink-0 flex-nowrap items-center justify-end gap-0.5">
            {!mobileSelectionMode && hasUnstaged && (
              <IconButton
                onClick={onStageAll}
                disabled={false}
                size="xs"
                surface="ghost"
                denseTapTarget
                className="!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !text-emerald-300"
                aria-label="Stage all"
                data-testid="stage-all-button"
                title="Stage All"
              >
                <Plus className="h-4 w-4" />
              </IconButton>
            )}
            {!mobileSelectionMode && hasStaged && (
              <IconButton
                onClick={onUnstageAll}
                disabled={false}
                size="xs"
                surface="ghost"
                denseTapTarget
                className="!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !text-slate-300"
                aria-label="Unstage all"
                data-testid="unstage-all-button"
                title="Unstage All"
              >
                <Minus className="h-4 w-4" />
              </IconButton>
            )}
            <ViewModeCycleButton
              mode={fileViewMode}
              onCycle={handleCycleViewMode}
              groupingAvailable={groupingAvailable}
              compact={compactHeader}
            />
          </div>
        </CardHeader>

        <div className="border-b border-slate-800 bg-slate-950 px-2" data-testid="run-filter-chip" role="region" aria-label="Changes filters">
          <Tabs
            density="compact"
            items={[
              { id: "all", label: "All" },
              { id: "agents", label: `From agents ${runIndex?.size ?? 0}` },
              { id: "staged", label: "Staged" },
              { id: "conflicts", label: "Conflicts" },
            ]}
            active={listFilter}
            onChange={(next) => setListFilter(next as typeof listFilter)}
            ariaLabel="Changes filters"
            itemTestId={(item) => `changes-filter-${item}`}
          />
        </div>

        {!collapsed && (
          <CardContent className="flex-1 min-w-0 p-0 overflow-hidden">
            <ScrollArea
              className="h-full min-w-0 px-2 pt-2 select-none"
              ref={scrollAreaRef}
              onScroll={handleScroll}
              data-testid="changes-scroll-region"
            >
            {/* Mobile multi-select toolbar */}
            {isMobile && mobileSelectionMode && (selectedFiles?.length ?? 0) > 0 && (
              <div className="z-10 flex items-center justify-between gap-2 px-3 py-2 bg-slate-900/95 backdrop-blur-sm border-b border-slate-700" data-testid="mobile-selection-toolbar">
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    className="p-1 rounded hover:bg-slate-700 transition-colors"
                    onClick={onExitSelectionMode}
                    aria-label="Exit selection mode"
                  >
                    <X className="h-4 w-4 text-slate-400" />
                  </button>
                  <span className="text-sm text-slate-200" aria-live="polite">
                    {selectedFiles?.length ?? 0} file{(selectedFiles?.length ?? 0) !== 1 ? "s" : ""} selected
                  </span>
                </div>
                <div className="flex items-center gap-1">
                  {selectionHasUnstaged && (
                    <Button
                      variant="secondary"
                      size="sm"
                      className="h-7 px-2 border-emerald-500/40 text-emerald-200 hover:bg-emerald-900/20"
                      onClick={() => {
                        const unstaged = selectedFiles?.filter((e) => !e.staged).map((e) => e.path) ?? [];
                        if (unstaged.length > 0) onStagePaths?.(unstaged);
                      }}
                      disabled={false}
                      style={{ background: "transparent" }}
                    >
                      Stage
                    </Button>
                  )}
                  {selectionHasStaged && (
                    <Button
                      variant="secondary"
                      size="sm"
                      className="h-7 px-2"
                      onClick={() => {
                        const staged = selectedFiles?.filter((e) => e.staged).map((e) => e.path) ?? [];
                        staged.forEach((p) => onUnstageFile(p));
                      }}
                      disabled={false}
                      style={{ background: "transparent" }}
                    >
                      Unstage
                    </Button>
                  )}
                  {selectionHasDiscardable && (
                    <Button
                      variant="secondary"
                      size="sm"
                      className="h-7 px-2 border-red-400/40 text-red-200 hover:bg-red-900/20"
                      onClick={() => {
                        const discardable = selectedFiles?.filter((e) => !e.staged) ?? [];
                        const tracked = discardable
                          .filter((e) => !(files?.untracked ?? []).includes(e.path))
                          .map((e) => e.path);
                        const untracked = discardable
                          .filter((e) => (files?.untracked ?? []).includes(e.path))
                          .map((e) => e.path);
                        if (tracked.length > 0) onDiscardPaths?.(tracked, false);
                        if (untracked.length > 0) onDiscardPaths?.(untracked, true);
                      }}
                      disabled={isDiscarding}
                      style={{ background: "transparent" }}
                    >
                      Discard
                    </Button>
                  )}
                </div>
              </div>
            )}
              <div style={{ paddingBottom: 72 }}>
                {fileViewMode === "tree" ? (
                  <ProjectTreeView
                    onSelectFile={(path) => {
                      // Use onSelectAnyFile if provided, otherwise simulate file selection
                      if (onSelectAnyFile) {
                        onSelectAnyFile(path);
                      } else {
                        // Find if file is in changes list
                        const isStaged = files?.staged?.includes(path) ?? false;
                        onSelectFile(path, isStaged, { metaKey: false, ctrlKey: false, shiftKey: false } as React.MouseEvent<HTMLLIElement>);
                      }
                    }}
                    selectedFile={selectedFiles?.[0]?.path}
                    gitStatuses={files?.statuses}
                    scrollToFile={scrollToFile}
                    onScrollComplete={onScrollComplete}
                    onDeletePath={onDeletePath}
                    onBlameFile={onBlameFile}
                    repoId={repoId}
                  />
                ) : (
                    <VirtualizedFileSections
                      blocks={deferredListBlocks}
                      scrollRef={scrollAreaRef}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      runIndex={runIndex}
                      onOpenRun={openRun}
                      maxPathChars={maxPathChars}
                      selectedKeySet={selectedKeys}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      pendingPaths={pendingPaths}
                      isDiscarding={isDiscarding}
                      isIgnoring={isIgnoring}
                      confirmingDiscard={confirmingDiscard}
                      onConfirmDiscard={onConfirmDiscard}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                      onIgnore={handleIgnoreFile}
                      resolvedGroups={resolvedGroups}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile || onStageFilesWithSameName ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onViewMetrics={openFileMetrics}
                      scrollToFile={scrollToFile}
                      onScrollComplete={onScrollComplete}
                    />
                )}

                {loading && !files && (
                  <div role="status" aria-label="Loading repository changes" data-testid="changes-loading" className="space-y-3 p-4 animate-pulse">
                    {[0, 1, 2, 3, 4].map((row) => <div key={row} className="flex items-center gap-3"><div className="h-4 w-4 rounded bg-slate-800" /><div className="h-3 rounded bg-slate-800" style={{ width: `${42 + row * 9}%` }} /></div>)}
                    <span className="sr-only">Loading repository changes…</span>
                  </div>
                )}
                {loadError && !loading && (
                  <div role="alert" data-testid="changes-load-error" className="m-4 rounded-lg border border-rose-900/70 bg-rose-950/30 p-4 text-sm">
                    <p className="text-rose-200">{files ? "Could not refresh repository changes." : "Could not load repository changes."}</p>
                    {files && <p className="mt-1 text-xs text-slate-400">The list may be out of date. Retry to check the current files.</p>}
                    <p className="mt-1 break-words text-xs text-slate-400">{loadError}</p>
                    <Button size="sm" className="mt-3" onClick={onRetryLoad}>Retry</Button>
                  </div>
                )}
                {!files && !loading && !loadError && (
                  <div role="status" data-testid="changes-status-idle" className="m-4 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm text-slate-300">
                    <p>Repository status is not available yet.</p>
                    <Button size="sm" className="mt-3" onClick={onRetryLoad}>Load changes</Button>
                  </div>
                )}
                {/* Empty State - only show for flat/grouped views */}
                {fileViewMode !== "tree" && files && !loadError && totalFilesCount === 0 && (
                  <div
                    className="flex flex-col items-center justify-center py-12 text-center"
                    data-testid="empty-state"
                  >
                    <File className="h-8 w-8 text-slate-700 mb-3" />
                    <p className="text-sm text-slate-500">
                      No changes detected
                    </p>
                    <p className="text-xs text-slate-600 mt-1">
                      Working directory is clean
                    </p>
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

      {/* Keep expanded group actions in one place when the header is narrow. */}
      {compactHeader && mobileGroupActions && (
        <BottomSheet
          isOpen={Boolean(mobileGroupActionsId)}
          onClose={() => setMobileGroupActionsId(null)}
          title={`${mobileGroupActions.label} actions`}
        >
          <div className="space-y-1">
            <BottomSheetAction
              icon={<FolderOpen className="h-5 w-5 text-cyan-300" />}
              label="Browse group folders"
              description="Open the folder view for this group"
              onClick={() => {
                setGroupDrillIn({ groupId: mobileGroupActions.id, directory: "" });
                setMobileGroupActionsId(null);
              }}
            />
            {!mobileSelectionMode && onStagePaths && mobileGroupActions.stageable.length > 0 && (
              <BottomSheetAction
                icon={<Plus className="h-5 w-5 text-emerald-300" />}
                label={`Stage ${mobileGroupActions.stageable.length} files`}
                description="Add all stageable changes in this group"
                variant="success"
                onClick={() => {
                  onStagePaths(mobileGroupActions.stageable);
                  setMobileGroupActionsId(null);
                }}
              />
            )}
            {!mobileSelectionMode && onDiscardPaths && mobileGroupActions.discardCount > 0 && (
              <BottomSheetAction
                icon={<Trash2 className="h-5 w-5 text-red-400" />}
                label={`Discard ${mobileGroupActions.discardCount} changes`}
                description="Review a confirmation before removing these changes"
                variant="danger"
                disabled={isDiscarding}
                onClick={() => {
                  setConfirmingGroup(mobileGroupActions.id);
                  setMobileGroupActionsId(null);
                }}
              />
            )}
          </div>
        </BottomSheet>
      )}

      {/* Mobile file action bottom sheet (suppressed in selection mode) */}
      {isMobile && !mobileSelectionMode && mobileActionFileInfo && (
        <BottomSheet
          isOpen={Boolean(mobileActionFile)}
          onClose={() => setMobileActionFile(null)}
          title={
            mobileActionFileInfo.path.split("/").pop() ||
            mobileActionFileInfo.path
          }
            >
              <div className="space-y-1">
                {onRevealInTree && (
                  <BottomSheetAction
                    icon={<FolderTree className="h-5 w-5 text-cyan-300" />}
                    label="Reveal in file tree"
                    description="Open the full tree and scroll to this file"
                    testId="reveal-in-tree-action"
                    onClick={() => {
                      onRevealInTree(mobileActionFileInfo.path);
                      setMobileActionFile(null);
                    }}
                  />
                  )}
                {mobileActionFileInfo.path && runIndex?.get(mobileActionFileInfo.path) && (
                  <BottomSheetAction
                    icon={<History className="h-5 w-5 text-cyan-300" />}
                    label="Show the run that changed this"
                    description="Open the sandbox run attribution"
                    onClick={() => {
                      openRun(runIndex.get(mobileActionFileInfo.path)?.runId ?? "");
                      setMobileActionFile(null);
                    }}
                  />
                )}
            {/* View metrics */}
            <BottomSheetAction
              icon={<BarChart3 className="h-5 w-5 text-slate-300" />}
              label="View Metrics"
              description="View change metrics for this file"
              onClick={() => {
                const cat: FileCategory = mobileActionFileInfo.isStaged ? "staged"
                  : mobileActionFileInfo.isUntracked ? "untracked"
                  : "unstaged";
                openFileMetrics(mobileActionFileInfo.path, cat);
                setMobileActionFile(null);
              }}
            />

            {/* Stage/Unstage action */}
            {mobileActionFileInfo.isStaged && (
              <BottomSheetAction
                icon={<Minus className="h-5 w-5 text-slate-300" />}
                label="Unstage"
                description="Remove from staged changes"
                onClick={() => {
                  onUnstageFile(mobileActionFileInfo.path);
                  setMobileActionFile(null);
                }}
              />
            )}
            {(mobileActionFileInfo.isUnstaged ||
              mobileActionFileInfo.isUntracked ||
              mobileActionFileInfo.isConflict) && (
              <BottomSheetAction
                icon={<Plus className="h-5 w-5 text-emerald-300" />}
                label="Stage"
                description="Add to staged changes"
                onClick={() => {
                  onStageFile(mobileActionFileInfo.path);
                  setMobileActionFile(null);
                }}
              />
            )}

            {/* Ignore action */}
            {(() => {
              const group = resolvedGroups?.find((candidate) =>
                candidate.source !== "builtin" && candidate.files.includes(mobileActionFileInfo.path),
              );
              if (group) {
                return (
                  <>
                    <BottomSheetAction
                      icon={<EyeOff className="h-5 w-5 text-blue-400" />}
                      label="Ignore (Project)"
                      description="Add to root .gitignore"
                      onClick={() => {
                        onIgnoreFile(mobileActionFileInfo.path, "project");
                        setMobileActionFile(null);
                      }}
                    />
                    <BottomSheetAction
                      icon={<EyeOff className="h-5 w-5 text-amber-300" />}
                          label={`Ignore (${group.label})`}
                          description={`Add to ${group.root ?? ""}.gitignore`}
                      onClick={() => {
                            onIgnoreFile(mobileActionFileInfo.path, "group", group.root ?? "");
                        setMobileActionFile(null);
                      }}
                    />
                  </>
                );
              }
              return (
                <BottomSheetAction
                  icon={<EyeOff className="h-5 w-5 text-amber-300" />}
                  label="Ignore"
                  description="Add to .gitignore"
                  onClick={() => {
                    onIgnoreFile(mobileActionFileInfo.path);
                    setMobileActionFile(null);
                  }}
                />
              );
            })()}

            {/* Discard action - only for unstaged/untracked */}
            {(mobileActionFileInfo.isUnstaged ||
              mobileActionFileInfo.isUntracked) && (
              <BottomSheetAction
                icon={<Trash2 className="h-5 w-5 text-red-400" />}
                label="Discard Changes"
                description={
                  mobileActionFileInfo.isUntracked
                    ? "Delete this file"
                    : "Revert to last commit"
                }
                variant="danger"
                onClick={() => {
                  onDiscardFile(
                    mobileActionFileInfo.path,
                    mobileActionFileInfo.isUntracked,
                  );
                  setMobileActionFile(null);
                }}
              />
            )}
          </div>
        </BottomSheet>
      )}

      {/* Context menu for right-click blame action */}
      <ContextMenu open={Boolean(contextMenu)} position={contextMenu ?? { x: 0, y: 0 }} title="File actions" closeLabel="Close file actions" items={contextMenuItems} onOpenChange={(open) => { if (!open) handleCloseContextMenu() }} triggers={[]} />

      {/* Change metrics modal */}
       <ChangeMetricsModal
        isOpen={metricsModal !== null}
        onClose={() => setMetricsModal(null)}
        mode={metricsModal?.mode ?? "aggregate"}
        stats={metricsModal?.stats}
        filePath={metricsModal?.path}
        fileStats={metricsModal?.filteredFileStats ?? fileStats}
        fileStatsState={fileStatsState}
        onRetryFileStats={onRetryFileStats}
        title={metricsModal?.title}
        enhancedStats={enhancedQuery.stats}
        enhancedLoading={enhancedQuery.isLoading}
        isUntracked={metricsModal?.category === "untracked"}
         fileHotspots={fileHotspots}
       />
       {openRunId && (
         <RunSheet
           runId={openRunId}
           files={openRunFiles?.paths ?? []}
           attribution={openRunFiles?.attribution}
           stats={runStats}
           onClose={() => setOpenRunId(null)}
           onSelectAll={() => {
             for (const path of openRunFiles?.paths ?? []) {
               const staged = stagedPathSet.has(path);
               onSelectFile(path, staged, { metaKey: false, ctrlKey: false, shiftKey: false } as React.MouseEvent<HTMLLIElement>);
             }
             setOpenRunId(null);
           }}
         />
       )}
     </MobileContext.Provider>
  );
}

export const FileList = memo(function FileList(props: FileListProps) {
  return (
    <Profiler id="FileList" onRender={onProfilerRender}>
      <FileListImpl {...props} />
    </Profiler>
  );
});
