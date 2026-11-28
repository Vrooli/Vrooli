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
  Copy,
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
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import { getFileStats, filterFileStats, filterCategoryStats } from "../lib/metrics";
import { copyText } from "../lib/clipboard";
import { compactCount, stripPathPrefix } from "../lib/utils";
import { restoreScrollPosition } from "../lib/scrollRestore";
import { experienceSignal } from "../lib/experience";
import { useApprovedChanges, useDiffStats, useProvenance } from "../lib/hooks";
import { buildRunIndex } from "../lib/runAttribution";
import { MobileContext, type FileCategory, type FileListProps, summarizeFileStats, summarizeGroupFileStats, LineStats } from "./FileListTypes";
import { VirtualizedFileSections, type ChangeListSection, type ChangeListBlock } from "./FileSection";
import { groupKindLabel } from "../lib/groupKinds";
import { RunSheet } from "./RunSheet";
import { runHue } from "../lib/runAttribution";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";

export type { GroupingRule, FileCategory, SelectedFileEntry, FileListProps } from "./FileListTypes";

type GroupDrillIn = { groupId: string; directory: string };

function formatCount(value: number): string {
  return value.toLocaleString();
}

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
  onUnstagePaths,
  onDiscardPaths,
  onClearSelection,
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
  const stageableAll = useMemo(
    () => [...new Set([
      ...(files?.conflicts ?? []),
      ...(files?.unstaged ?? []),
      ...(files?.untracked ?? []),
    ])],
    [files?.conflicts, files?.unstaged, files?.untracked],
  );
  const hasUnstaged = stageableAll.length > 0;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);
  const [compactHeader, setCompactHeader] = useState(false);
  const [confirmingGroup, setConfirmingGroup] = useState<string | null>(null);
  const [groupDrillIn, setGroupDrillIn] = useState<GroupDrillIn | null>(null);
  // Scroll memory is keyed by the visible list scope: the overview and each
  // drilled folder. The virtualizer changes its total height after a scope
  // switch, so the restore is deferred until the new height is measurable.
  const scrollPositionsRef = useRef<Map<string, number>>(new Map());
  const currentScrollScopeRef = useRef("overview");
  const scrollScope = groupDrillIn ? `drill:${groupDrillIn.groupId}:${groupDrillIn.directory}` : "overview";
  currentScrollScopeRef.current = scrollScope;
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
  const [groupContextMenu, setGroupContextMenu] = useState<{
    x: number;
    y: number;
    groupId: string;
  } | null>(null);
  const [batchDiscardConfirm, setBatchDiscardConfirm] = useState(false);

  const handleFileContextMenu = useCallback((file: string, event: React.MouseEvent) => {
    setGroupContextMenu(null);
    setContextMenu({
      x: event.clientX,
      y: event.clientY,
      file,
    });
  }, []);

  const handleCloseContextMenu = useCallback(() => {
    setContextMenu(null);
  }, []);

  const handleGroupContextMenu = useCallback((groupId: string, event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    setContextMenu(null);
    setGroupContextMenu({ x: event.clientX, y: event.clientY, groupId });
  }, []);

  const handleCloseGroupContextMenu = useCallback(() => {
    setGroupContextMenu(null);
  }, []);

  const contextMenuItems = useMemo<ContextMenuItem[]>(() => {
    if (!contextMenu) return [];
    const file = contextMenu.file;
    const items: ContextMenuItem[] = [];
    const isStaged = files?.staged?.includes(file) ?? false;
    const isUnstaged = files?.unstaged?.includes(file) ?? false;
    const isUntracked = files?.untracked?.includes(file) ?? false;
    const isConflict = files?.conflicts?.includes(file) ?? false;
    const category: FileCategory = isConflict
      ? "conflicts"
      : isUntracked && !isUnstaged
        ? "untracked"
        : isStaged && !isUnstaged
          ? "staged"
          : "unstaged";

    if (isUnstaged || isUntracked || isConflict) {
      items.push({
        id: "stage",
        label: "Stage file",
        icon: <Plus className="h-4 w-4 text-emerald-300" />,
        onSelect: () => onStageFile(file),
        testId: "context-stage-file",
      });
    }
    if (isStaged) {
      items.push({
        id: "unstage",
        label: "Unstage file",
        icon: <Minus className="h-4 w-4 text-slate-300" />,
        onSelect: () => onUnstageFile(file),
        testId: "context-unstage-file",
      });
    }

    const directory = file.includes("/") ? file.slice(0, file.lastIndexOf("/")) : "";
    const folderPrefix = directory ? `${directory}/` : "";
    const stageableSiblings = [...new Set([
      ...(files?.conflicts ?? []),
      ...(files?.unstaged ?? []),
      ...(files?.untracked ?? []),
    ])].filter((candidate) =>
      candidate !== file &&
      (!directory || candidate.startsWith(folderPrefix)),
    );
    if (onStagePaths && stageableSiblings.length > 0) {
      items.push({
        id: "stage-folder",
        label: `Stage all ${stageableSiblings.length + (isUnstaged || isUntracked || isConflict ? 1 : 0)} files in ${directory || "root"}`,
        icon: <ClipboardCheck className="h-4 w-4 text-emerald-300" />,
        onSelect: () => onStagePaths([
          ...(isUnstaged || isUntracked || isConflict ? [file] : []),
          ...stageableSiblings,
        ]),
        testId: "context-stage-folder",
        separatorBefore: items.length > 0,
      });
    }

    const basename = file.split("/").pop() || file;
    const changedFiles = [
      ...(files?.staged ?? []),
      ...(files?.unstaged ?? []),
      ...(files?.untracked ?? []),
      ...(files?.conflicts ?? []),
    ];
    const sameNameCount = new Set(
      changedFiles.filter((candidate) => (candidate.split("/").pop() || candidate) === basename),
    ).size;
    if (onStageFilesWithSameName && sameNameCount > 1) {
      items.push({
        id: "stage-same-name",
        label: `Stage all changed files named ${basename}`,
        icon: <ClipboardCheck className="h-4 w-4" />,
        onSelect: () => onStageFilesWithSameName(file),
      });
    }

    if (onRevealInTree) items.push({
      id: "reveal",
      label: "Reveal in file tree",
      icon: <FolderTree className="h-4 w-4" />,
      onSelect: () => onRevealInTree(file),
      testId: "reveal-in-tree-action",
      separatorBefore: items.length > 0,
    });

    items.push({
      id: "metrics",
      label: "View file metrics",
      icon: <BarChart3 className="h-4 w-4" />,
      onSelect: () => openFileMetrics(file, category),
      testId: "context-view-metrics",
    });

    const attribution = runIndex?.get(file);
    if (attribution) items.push({
      id: "run",
      label: "Show the run that changed this",
      icon: <History className="h-4 w-4" />,
      onSelect: () => openRun(attribution.runId),
    });

    if (onBlameFile) items.push({
      id: "history",
      label: "View file history",
      icon: <History className="h-4 w-4" />,
      onSelect: () => onBlameFile(file),
    });

    items.push({
      id: "copy-path",
      label: "Copy path",
      icon: <Copy className="h-4 w-4" />,
      onSelect: () => { void copyText(file); },
      testId: "context-copy-path",
    });

    if (onConfirmIgnore && (isUnstaged || isUntracked || isConflict)) {
      items.push({
        id: "ignore",
        label: "Ignore file…",
        icon: <EyeOff className="h-4 w-4" />,
        onSelect: () => onConfirmIgnore(file),
        separatorBefore: true,
      });
    }
    if (onConfirmDiscard && (isUnstaged || isUntracked)) {
      items.push({
        id: "discard",
        label: "Discard changes…",
        icon: <Trash2 className="h-4 w-4" />,
        onSelect: () => onConfirmDiscard(file),
        destructive: true,
        testId: "context-discard-file",
      });
    }

    return items;
  }, [contextMenu, files, onBlameFile, onRevealInTree, onStageFile, onUnstageFile,
    onStageFilesWithSameName, onStagePaths, onConfirmIgnore, onConfirmDiscard,
    runIndex, openRun, openFileMetrics]);

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

  useEffect(() => {
    setBatchDiscardConfirm(false);
  }, [selectedFiles]);

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
      // Reserve room for the status badge, file icon, line stats, primary
      // action, and padding. Narrow rows hide the hover-only secondary actions,
      // so they need less reservation than wide rows.
      const compact = width < 500;
      const chrome = compact ? 140 : 240;
      const usable = Math.max(0, width - chrome);
      const nextMax = Math.max(12, Math.min(110, Math.floor(usable / 7.5)));
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

  // Navigation may target a file inside a group that is not open yet. Open the
  // owning group at the file's parent folder before the virtualizer resolves the
  // destination; flat mode only needs the category section opened.
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

    let sectionScope = "__flat__";
    if (group) {
      const root = (group.root ?? "").replace(/^\/+|\/+$/g, "");
      const relative = stripPathPrefix(scrollToFile, root);
      const directory = relative.includes("/") ? relative.slice(0, relative.lastIndexOf("/")) : "";
      sectionScope = `drill:${group.key}:${directory}`;
      setGroupDrillIn((previous) => (
        previous?.groupId === group.key && previous.directory === directory
          ? previous
          : { groupId: group.key, directory }
      ));
    }

    setSectionExpanded(prev => {
      if (categories.every(category => prev[`${sectionScope}::${category}`] === true)) return prev;
      const next = { ...prev };
      for (const category of categories) next[`${sectionScope}::${category}`] = true;
      try { localStorage.setItem(`gct.collapsedSections.${repoId ?? "default"}`, JSON.stringify(next)); } catch { /* ignore */ }
      return next;
    });
  }, [scrollToFile, fileViewMode, files, groupingActive, resolvedGroups, onScrollComplete, repoId]);

  // Persist the Changes list scroll position per visible scope. The mobile
  // store survives the panel unmount; the map survives every drill transition.
  const handleScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const top = event.currentTarget.scrollTop;
      scrollPositionsRef.current.set(currentScrollScopeRef.current, top);
      if (currentScrollScopeRef.current === "overview" && scrollTopStore) {
        scrollTopStore.current = top;
      }
    },
    [scrollTopStore],
  );

  // Restore the mobile list after its panel remounts. The list may still be
  // loading on the first frame, so the helper retries until the target fits and
  // reports the real position. Skipped when a scrollToFile target is pending so
  // the scroll-into-view path wins instead of fighting the restore.
  useLayoutEffect(() => {
    if (!scrollTopStore || scrollToFile) return;
    const el = scrollAreaRef.current;
    const target = scrollTopStore.current;
    if (!el || target <= 0) return;
    return restoreScrollPosition(el, target, (actual) => {
      scrollTopStore.current = actual;
      experienceSignal("scroll_restore", {
        container: "changes-scroll-region",
        scope: "mobile-changes",
        expected: target,
        actual,
      });
    });
    // Mount-only: the mobile store is the source of truth for this remount.
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
    for (const [categoryIndex, [, paths]] of categories.entries()) {
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
      const allPaths = Object.values(groupedFiles).flat();
      const uniquePaths = [...new Set(allPaths)];
      const stageable = [...new Set([
        ...groupedFiles.unstaged,
        ...groupedFiles.untracked,
        ...groupedFiles.conflicts,
      ])];
      const discardPaths = [...new Set([...groupedFiles.unstaged, ...groupedFiles.untracked])];
      return {
        id: group.key,
        label: group.label,
        kind: group.kind,
        source: group.source,
        displayPrefix: group.root ?? "",
        files: groupedFiles,
        stageable,
        discardCount: discardPaths.length,
        count: uniquePaths.length,
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

  const renderGroupHeader = useCallback((group: (typeof groupedSections)[number], index: number) => {
    const { stageable, discardCount, count: groupCount, runs } = group;
    const discardTracked = group.files.unstaged;
    const discardUntracked = group.files.untracked;
    const stats = fileStatsState === "refreshing" ? group.previewLineStats : group.lineStats;
    const showGroupActions = !mobileSelectionMode;
    const openGroup = () => setGroupDrillIn({ groupId: group.id, directory: "" });

    const previous = groupedSections[index - 1];
    const bandKind = group.source === "contract" ? group.kind : group.source;
    const previousBandKind = previous?.source === "contract" ? previous.kind : previous?.source;
    const showBand = bandKind !== previousBandKind;

    const groupHeader = (
      <div
        className="group/change-group cursor-pointer border-b border-slate-800/60 px-1.5 py-1 transition-colors hover:bg-slate-900/50 active:bg-slate-900/60"
        onClick={openGroup}
        onContextMenu={(event) => handleGroupContextMenu(group.id, event)}
        data-testid={`file-group-header-${group.id}`}
      >
        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2">
          <button
            type="button"
            className={`col-span-2 flex min-w-0 items-center gap-2 rounded-md py-0.5 text-left focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-400 ${isMobile ? "min-h-11" : ""}`}
            onClick={(event) => {
              event.stopPropagation();
              openGroup();
            }}
            aria-label={`Open ${group.label}`}
            data-testid={`file-group-open-${group.id}`}
          >
            <span className={`min-w-0 flex-1 truncate font-semibold text-slate-100 transition-colors group-hover/change-group:text-white group-hover/change-group:underline decoration-slate-500 underline-offset-2 ${isMobile ? "text-sm" : "text-xs"}`} title={group.label}>
              {group.label}
            </span>
            <span className="shrink-0 font-mono text-[10px] tabular-nums text-slate-500" aria-label={`${groupCount} ${groupCount === 1 ? "file" : "files"}`}>
              {formatCount(groupCount)} {groupCount === 1 ? "file" : "files"}
            </span>
            {runs.length > 0 && runIndex && (
              <span className="flex shrink-0 items-center gap-1" data-testid="group-run-dots" aria-label={`${runs.length} agent runs`}>
                {runs.slice(0, 3).map((runId) => <span key={runId} className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: runHue(runId) }} />)}
                {runs.length > 3 && <span className="text-[9px] text-slate-500">+{runs.length - 3}</span>}
              </span>
            )}
          </button>

          <div className="mt-0.5 flex min-w-0 items-center gap-1.5 text-[10.5px] text-slate-500">
            {group.displayPrefix && group.source !== "contract" && (
              <span
                className="truncate-start min-w-0"
                title={group.displayPrefix}
              >
                {group.displayPrefix.replace(/\/+$/, "")}
              </span>
            )}
            {stats ? (
              <button
                type="button"
                className="inline-flex shrink-0 items-center gap-1 rounded font-mono text-[10px] tabular-nums hover:underline decoration-slate-600"
                onClick={(event) => {
                  event.stopPropagation();
                  openGroupMetrics(group.files, group.label);
                }}
                aria-label={`View change metrics for ${group.label}`}
                title={`View change metrics for ${group.label}`}
              >
                <span className="text-emerald-300">+{compactCount(stats.additions)}</span>
                <span className="text-rose-300">−{compactCount(stats.deletions)}</span>
              </button>
            ) : fileStatsState === "loading" || fileStatsState === "refreshing" ? (
              <span className="h-2.5 w-12 animate-pulse rounded-sm bg-slate-800" aria-hidden="true" />
            ) : null}
          </div>

          <div
            className="mt-0.5 flex shrink-0 items-center gap-1"
            onClick={(event) => event.stopPropagation()}
          >
            {showGroupActions && stageable.length > 0 && onStagePaths && (
              <button
                type="button"
                className={`inline-flex items-center gap-1 rounded-md border border-emerald-400/25 bg-emerald-400/[0.08] px-2 text-[11px] font-semibold text-emerald-200 transition-colors hover:border-emerald-400/45 hover:bg-emerald-400/15 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400 ${isMobile ? "h-9" : "h-8"}`}
                onClick={() => onStagePaths(stageable)}
                aria-label={`Stage ${stageable.length} files in ${group.label}`}
                title={`Stage ${stageable.length} files in ${group.label}`}
                data-testid={`file-group-stage-${group.id}`}
              >
                <Plus className="h-3.5 w-3.5" />
                {!compactHeader && <span>Stage</span>}
              </button>
            )}
            {showGroupActions && (
              <IconButton
                size="xs"
                surface="ghost"
                denseTapTarget
                className={isMobile ? "!h-9 !w-9 !min-h-0 !min-w-0 !border-0 !shadow-none !text-slate-400" : "!h-8 !w-8 !min-h-0 !min-w-0 !border-0 !shadow-none !text-slate-400"}
                onClick={(event) => {
                  const rect = event.currentTarget.getBoundingClientRect();
                  setGroupContextMenu({ x: rect.left, y: rect.bottom, groupId: group.id });
                }}
                aria-label={`More actions for ${group.label}`}
                title={`More actions for ${group.label}`}
                data-testid={`file-group-actions-${group.id}`}
              ><MoreHorizontal className="h-4 w-4" /></IconButton>
            )}
          </div>
        </div>
      </div>
    );
    return (
      <>
      {showBand && (
        <div role="region" aria-label={`${groupKindLabel(group.kind, group.source)} group band`} className="relative z-10 flex min-h-[26px] items-center gap-2 border-y border-slate-800/80 bg-slate-950/95 px-2 py-0.5 text-[9.5px] font-semibold uppercase tracking-[0.14em] text-slate-500" data-testid={`file-kind-band-${bandKind || "other"}`}>
          <span>{groupKindLabel(group.kind, group.source)}</span>
          <span className="h-px flex-1 bg-slate-800/80" />
          <span className="font-normal tracking-normal normal-case text-slate-600">{group.bandSummary.groups} groups · {formatCount(group.bandSummary.files)} files</span>
        </div>
      )}
      <div
        key={group.id}
        className="border-b border-slate-800/80"
        data-testid={`file-group-${group.id}`}
      >
        {groupHeader}
        {confirmingGroup === group.id &&
              discardCount > 0 && (
                <div className="flex items-center justify-between gap-2 border-b border-red-900/40 bg-red-950/30 px-3 py-2 text-xs text-red-200">
                  <span>
                    Discard {discardCount} changes in this
                    group?
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      className="min-h-8 px-2 py-1 rounded border border-red-400/40 text-red-100 hover:bg-red-900/30"
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
                      className="min-h-8 px-2 py-1 rounded border border-slate-600 text-slate-200 hover:bg-slate-800/50"
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
  }, [groupedSections, isMobile, runIndex,
    mobileSelectionMode, onStagePaths, compactHeader, onDiscardPaths,
    confirmingGroup, openGroupMetrics,
    setGroupDrillIn, fileStatsState, handleGroupContextMenu]);

  const groupMenuTarget = groupContextMenu
    ? groupedSections.find((candidate) => candidate.id === groupContextMenu.groupId) ?? null
    : null;

  useEffect(() => {
    if (groupContextMenu && !groupMenuTarget) setGroupContextMenu(null);
  }, [groupContextMenu, groupMenuTarget]);

  const groupMenuItems = useMemo<ContextMenuItem[]>(() => {
    if (!groupMenuTarget) return [];
    const items: ContextMenuItem[] = [];
    const staged = groupMenuTarget.files.staged;
    if (!mobileSelectionMode && onStagePaths && groupMenuTarget.stageable.length > 0) {
      items.push({
        id: "stage-group",
        label: `Stage ${groupMenuTarget.stageable.length} files in ${groupMenuTarget.label}`,
        icon: <Plus className="h-4 w-4 text-emerald-300" />,
        onSelect: () => onStagePaths(groupMenuTarget.stageable),
        testId: "group-context-stage",
      });
    }
    if (!mobileSelectionMode && staged.length > 0) {
      items.push({
        id: "unstage-group",
        label: `Unstage ${staged.length} files in ${groupMenuTarget.label}`,
        icon: <Minus className="h-4 w-4 text-slate-300" />,
        onSelect: () => {
          if (onUnstagePaths) onUnstagePaths(staged);
          else staged.forEach((path) => onUnstageFile(path));
        },
        testId: "group-context-unstage",
      });
    }
    items.push({
      id: "open-group",
      label: "Open group",
      icon: <FolderOpen className="h-4 w-4 text-sky-300" />,
      onSelect: () => setGroupDrillIn({ groupId: groupMenuTarget.id, directory: "" }),
      testId: "group-context-open",
      separatorBefore: items.length > 0,
    });
    items.push({
      id: "group-metrics",
      label: "View change metrics",
      icon: <BarChart3 className="h-4 w-4" />,
      onSelect: () => openGroupMetrics(groupMenuTarget.files, groupMenuTarget.label),
    });
    if (onOpenReview && groupMenuTarget.kind === "scenario" && groupMenuTarget.displayPrefix) {
      items.push({
        id: "group-review",
        label: "Open scenario review",
        icon: <ClipboardCheck className="h-4 w-4 text-cyan-300" />,
        onSelect: () => onOpenReview(groupMenuTarget.displayPrefix.split("/")[1] ?? ""),
      });
    }
    if (!mobileSelectionMode && onDiscardPaths && groupMenuTarget.discardCount > 0) {
      items.push({
        id: "discard-group",
        label: `Discard ${groupMenuTarget.discardCount} changes…`,
        icon: <Trash2 className="h-4 w-4" />,
        destructive: true,
        separatorBefore: true,
        onSelect: () => setConfirmingGroup(groupMenuTarget.id),
        testId: "group-context-discard",
      });
    }
    return items;
  }, [groupMenuTarget, mobileSelectionMode, onStagePaths,
    onUnstagePaths, onUnstageFile, onOpenReview, onDiscardPaths,
    openGroupMetrics, setGroupDrillIn]);


  const listBlocks = useMemo<ChangeListBlock[]>(() => {
    const sectionsFor = (groupId: string, categoryFiles: Partial<Record<FileCategory, string[]>>, label?: string, displayPrefix?: string): ChangeListBlock[] => {
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
          displayPrefix,
          fileStats: fileStats?.[statsCategory],
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
            const uniquePaths = [...new Set(allPaths)];
            const stageable = [...new Set([
              ...folderFiles.conflicts,
              ...folderFiles.unstaged,
              ...folderFiles.untracked,
            ])];
            const directory = [groupDrillIn.directory, name].filter(Boolean).join("/");
            return {
              type: "header",
              key: `folder:${group.id}:${directory}`,
              estimateSize: isMobile ? 56 : 50,
              render: () => (
                <div className="flex min-h-11 items-center gap-1 border-b border-slate-800/70 py-1 pl-2 pr-1.5 transition-colors hover:bg-slate-900/40" data-testid="change-folder-row">
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 rounded px-1 py-0.5 text-left text-slate-200 hover:text-white focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-sky-400"
                    onClick={() => setGroupDrillIn({ groupId: group.id, directory })}
                    aria-label={`Open folder ${directory}`}
                    data-testid="change-folder-open"
                  >
                    <FolderOpen className="h-4 w-4 shrink-0 text-sky-300" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[13px] font-medium">{name}</span>
                      <span className="block truncate text-[10px] text-slate-500" data-testid="change-folder-count">
                        {uniquePaths.length} {uniquePaths.length === 1 ? "file" : "files"}
                        {stageable.length > 0 ? ` · ${stageable.length} to stage` : ""}
                      </span>
                    </span>
                    <ChevronRight className="h-4 w-4 shrink-0 text-slate-600" />
                  </button>
                  {stageable.length > 0 && onStagePaths && (
                    <button
                      type="button"
                      className={`inline-flex shrink-0 items-center gap-1 rounded-md border border-emerald-400/20 bg-emerald-400/[0.06] px-1.5 text-[10.5px] font-semibold text-emerald-200 transition-colors hover:border-emerald-400/40 hover:bg-emerald-400/15 ${isMobile ? "h-10" : "h-8"}`}
                      onClick={() => onStagePaths(stageable)}
                      aria-label={`Stage ${stageable.length} changed files in ${directory}`}
                      title={`Stage ${stageable.length} changed files in ${directory}`}
                      data-testid="change-folder-stage"
                    >
                      <Plus className="h-3 w-3" />
                      {!compactHeader && <span>Stage</span>}
                    </button>
                  )}
                </div>
              ),
            };
          });

        const allScopedPaths = categories.flatMap((category) => [
          ...(directFiles[category] ?? []),
          ...[...folders.values()].flatMap((folderFiles) => folderFiles[category]),
        ]);
        const uniqueScopedPaths = [...new Set(allScopedPaths)];
        const stageablePathSet = new Set([
          ...group.files.unstaged,
          ...group.files.untracked,
          ...group.files.conflicts,
        ]);
        const stageable = uniqueScopedPaths.filter((path) => stageablePathSet.has(path));
        const segments = groupDrillIn.directory.split("/").filter(Boolean);
        const breadcrumbRows: ChangeListBlock[] = [{
          type: "header",
          key: `group-drill:${group.id}:${groupDrillIn.directory}`,
          estimateSize: isMobile ? 96 : 72,
          render: () => (
            <div className="border-b border-slate-800 bg-slate-950/95 px-2 py-1.5" data-testid="change-group-drill-header">
              <div className="flex min-h-8 items-center gap-1.5">
                <button
                  type="button"
                  className="inline-flex min-h-8 shrink-0 items-center gap-1 rounded px-1 text-xs font-medium text-sky-300 hover:bg-slate-800/70"
                  onClick={() => {
                    if (!segments.length) setGroupDrillIn(null);
                    else setGroupDrillIn({ groupId: group.id, directory: segments.slice(0, -1).join("/") });
                  }}
                  aria-label={segments.length ? "Go to parent folder" : "Back to all change groups"}
                  data-testid="change-group-drill-back"
                ><ArrowLeft className="h-4 w-4" />{segments.length ? "Back" : "Groups"}</button>
                <nav className="flex min-w-0 flex-1 items-center gap-0.5 overflow-hidden text-[11px] text-slate-400" aria-label="Change location">
                  <button type="button" className="truncate rounded px-0.5 hover:text-slate-200" onClick={() => setGroupDrillIn({ groupId: group.id, directory: "" })}>{group.label}</button>
                  {segments.map((segment, index) => (
                    <span key={`${index}:${segment}`} className="flex min-w-0 items-center gap-0.5">
                      <ChevronRight className="h-3 w-3 shrink-0 text-slate-600" />
                      <button type="button" className="truncate rounded px-0.5 hover:text-slate-200" onClick={() => setGroupDrillIn({ groupId: group.id, directory: segments.slice(0, index + 1).join("/") })}>{segment}</button>
                    </span>
                  ))}
                </nav>
                {stageable.length > 0 && onStagePaths && (
                  <button
                    type="button"
                    className={`inline-flex shrink-0 items-center gap-1 rounded-md border border-emerald-400/25 bg-emerald-400/[0.08] px-2 text-[11px] font-semibold text-emerald-200 hover:border-emerald-400/45 hover:bg-emerald-400/15 ${isMobile ? "h-10" : "h-8"}`}
                    onClick={() => onStagePaths(stageable)}
                    data-testid="change-group-scope-stage"
                  >
                    <Plus className="h-3.5 w-3.5" />Stage {segments.length ? "folder" : "group"}
                  </button>
                )}
              </div>
              <div className="mt-0.5 flex items-center gap-2 pl-1 font-mono text-[10px] tabular-nums text-slate-500">
                <span>{formatCount(uniqueScopedPaths.length)} changed {uniqueScopedPaths.length === 1 ? "file" : "files"}</span>
                {stageable.length > 0 && <span className="text-emerald-300/80">{formatCount(stageable.length)} to stage</span>}
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
          ...sectionsFor(
            `drill:${group.id}:${groupDrillIn.directory}`,
            directFiles,
            `${group.label}/${groupDrillIn.directory}`,
            [root, groupDrillIn.directory].filter(Boolean).join("/"),
          ),
        ];
      }
    }
    return groupedSections.flatMap((group, index): ChangeListBlock[] => group.count === 0 ? [] : [
      { type: "header", key: `group:${group.id}`, render: () => renderGroupHeader(group, index), estimateSize: isMobile ? 84 : 60 },
    ]);
  }, [files, fileStats, fileStatsState, groupedSections, groupingActive, filterPaths,
    isSectionExpanded, toggleSectionCollapse, onStageFile, onUnstageFile,
    handleDiscardUnstaged, handleDiscardUntracked, openAggregateMetrics, openGroupCategoryMetrics,
    isMobile, compactHeader, renderGroupHeader, groupDrillIn, onStagePaths]);

  const deferredListBlocks = useDeferredValue(listBlocks);
  const hasVisibleFiles = useMemo(
    () => listBlocks.some((block) => block.type === "header" || block.section.files.length > 0),
    [listBlocks],
  );

  useLayoutEffect(() => {
    const element = scrollAreaRef.current;
    const previous = previousDrillInRef.current;
    if (!element) return;
    const same = previous?.groupId === groupDrillIn?.groupId && previous?.directory === groupDrillIn?.directory;
    if (same) return;
    previousDrillInRef.current = groupDrillIn;
    const scope = groupDrillIn ? `drill:${groupDrillIn.groupId}:${groupDrillIn.directory}` : "overview";
    const target = scrollPositionsRef.current.get(scope) ?? 0;
    return restoreScrollPosition(element, target, (actual) => {
      scrollPositionsRef.current.set(scope, actual);
      if (scope === "overview" && scrollTopStore) scrollTopStore.current = actual;
      experienceSignal("scroll_restore", {
        container: "changes-scroll-region",
        scope,
        expected: target,
        actual,
      });
    });
  }, [groupDrillIn, scrollTopStore]);

  useEffect(() => {
    if (groupDrillIn && !resolvedGroups?.some((group) => group.key === groupDrillIn.groupId)) {
      setGroupDrillIn(null);
    }
  }, [groupDrillIn, resolvedGroups]);

  const handleCycleViewMode = onCycleViewMode ?? (() => {});
  const totalFilesCount = useMemo(
    () => new Set([
      ...(files?.conflicts ?? []),
      ...(files?.staged ?? []),
      ...(files?.unstaged ?? []),
      ...(files?.untracked ?? []),
    ]).size,
    [files?.conflicts, files?.staged, files?.unstaged, files?.untracked],
  );

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
            <span
              className="shrink-0 rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[10px] tabular-nums text-slate-400"
              aria-label={`${totalFilesCount} changed files`}
            >
              {formatCount(totalFilesCount)}
            </span>
            {compactHeader ? (
              <IconButton
                onClick={(e) => {
                  e.stopPropagation();
                  setMetricsModal({ mode: "aggregate" });
                }}
                aria-label="View change metrics"
                title="View change metrics"
                size="xs"
                surface="ghost"
                denseTapTarget
              >
                <BarChart3 className="h-4 w-4 text-slate-400" />
              </IconButton>
            ) : (
              <LineStats
                stats={totalStats}
                compact
                onClick={() => setMetricsModal({ mode: "aggregate" })}
              />
            )}
          </CardTitle>
          <div className="relative z-20 flex shrink-0 flex-nowrap items-center justify-end gap-0.5">
            {!mobileSelectionMode && hasUnstaged && (
              <button
                type="button"
                onClick={onStageAll}
                className="inline-flex h-8 items-center gap-1 rounded-md border border-emerald-400/20 bg-emerald-400/[0.06] px-1.5 text-[11px] font-semibold text-emerald-200 transition-colors hover:border-emerald-400/40 hover:bg-emerald-400/15 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-400"
                aria-label={`Stage all ${stageableAll.length} changed files`}
                data-testid="stage-all-button"
                title={`Stage all ${stageableAll.length} changed files`}
              >
                <Plus className="h-4 w-4" />
                {!compactHeader && <span>Stage all</span>}
                <span className="font-mono tabular-nums">{compactCount(stageableAll.length)}</span>
              </button>
            )}
            {!mobileSelectionMode && hasStaged && (
              <button
                type="button"
                onClick={onUnstageAll}
                className="inline-flex h-8 items-center gap-1 rounded-md border border-slate-700/80 bg-slate-900/60 px-1.5 text-[11px] font-semibold text-slate-300 transition-colors hover:border-slate-600 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-slate-400"
                aria-label={`Unstage all ${files?.staged?.length ?? 0} staged files`}
                data-testid="unstage-all-button"
                title={`Unstage all ${files?.staged?.length ?? 0} staged files`}
              >
                <Minus className="h-4 w-4" />
                {!compactHeader && <span>Unstage all</span>}
                <span className="font-mono tabular-nums">{compactCount(files?.staged?.length ?? 0)}</span>
              </button>
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
              { id: "agents", label: `Agents ${runIndex?.size ?? 0}` },
              { id: "staged", label: "Staged" },
              { id: "conflicts", label: "Conflicts" },
            ]}
            active={listFilter}
            onChange={(next) => setListFilter(next as typeof listFilter)}
            ariaLabel="Changes filters"
            itemTestId={(item) => `changes-filter-${item}`}
          />
        </div>

        {/* Desktop multi-select toolbar. Selecting rows with click, ctrl or
            shift now exposes the same batch actions the mobile toolbar has. */}
        {!isMobile && !mobileSelectionMode && (selectedFiles?.length ?? 0) > 1 && (() => {
          const unstagedPaths = selectedFiles?.filter((entry) => !entry.staged).map((entry) => entry.path) ?? [];
          const stagedPaths = selectedFiles?.filter((entry) => entry.staged).map((entry) => entry.path) ?? [];
          const discardable = selectedFiles?.filter((entry) => !entry.staged) ?? [];
          const untrackedSet = new Set(files?.untracked ?? []);
          const trackedDiscard = discardable.filter((entry) => !untrackedSet.has(entry.path)).map((entry) => entry.path);
          const untrackedDiscard = discardable.filter((entry) => untrackedSet.has(entry.path)).map((entry) => entry.path);
          return (
            <div
              role="toolbar"
              aria-label="Selected changes"
              data-testid="changes-selection-toolbar"
              className="flex items-center justify-between gap-2 border-b border-sky-400/20 bg-sky-950/20 px-2 py-1.5"
            >
              <div className="flex min-w-0 items-center gap-1.5">
                <span className="grid h-5 min-w-5 place-items-center rounded-full bg-sky-400/15 px-1 font-mono text-[10px] font-semibold text-sky-200">
                  {selectedFiles?.length ?? 0}
                </span>
                <span className="truncate text-[11px] text-slate-300">selected</span>
              </div>
              <div className="flex shrink-0 items-center gap-1">
                {unstagedPaths.length > 0 && (
                  <button
                    type="button"
                    className="inline-flex h-7 items-center gap-1 rounded border border-emerald-400/30 bg-emerald-400/10 px-2 text-[11px] font-semibold text-emerald-200 hover:bg-emerald-400/15"
                    onClick={() => onStagePaths?.(unstagedPaths)}
                    data-testid="selection-stage"
                  >
                    <Plus className="h-3 w-3" />Stage {unstagedPaths.length}
                  </button>
                )}
                {stagedPaths.length > 0 && (
                  <button
                    type="button"
                    className="inline-flex h-7 items-center gap-1 rounded border border-slate-600 bg-slate-800/60 px-2 text-[11px] font-semibold text-slate-200 hover:bg-slate-700"
                    onClick={() => {
                      if (onUnstagePaths) onUnstagePaths(stagedPaths);
                      else stagedPaths.forEach((path) => onUnstageFile(path));
                    }}
                    data-testid="selection-unstage"
                  >
                    <Minus className="h-3 w-3" />Unstage {stagedPaths.length}
                  </button>
                )}
                {(trackedDiscard.length > 0 || untrackedDiscard.length > 0) && (
                  batchDiscardConfirm ? (
                    <>
                      <button
                        type="button"
                        className="inline-flex h-7 items-center rounded border border-red-400/50 bg-red-500/15 px-2 text-[11px] font-semibold text-red-100 hover:bg-red-500/25"
                        onClick={() => {
                          if (trackedDiscard.length > 0) onDiscardPaths?.(trackedDiscard, false);
                          if (untrackedDiscard.length > 0) onDiscardPaths?.(untrackedDiscard, true);
                          setBatchDiscardConfirm(false);
                        }}
                        data-testid="selection-discard-confirm"
                      >
                        Discard {discardable.length}
                      </button>
                      <button
                        type="button"
                        className="inline-flex h-7 items-center rounded border border-slate-600 px-2 text-[11px] text-slate-200 hover:bg-slate-800"
                        onClick={() => setBatchDiscardConfirm(false)}
                      >
                        Cancel
                      </button>
                    </>
                  ) : (
                    <button
                      type="button"
                      className="inline-flex h-7 items-center gap-1 rounded border border-red-400/25 px-2 text-[11px] font-medium text-red-300 hover:bg-red-500/10"
                      onClick={() => setBatchDiscardConfirm(true)}
                      data-testid="selection-discard"
                    >
                      <Trash2 className="h-3 w-3" />Discard
                    </button>
                  )
                )}
                <button
                  type="button"
                  className="grid h-7 w-7 place-items-center rounded text-slate-400 hover:bg-slate-800 hover:text-slate-100"
                  onClick={() => {
                    setBatchDiscardConfirm(false);
                    if (onClearSelection) onClearSelection();
                    else onExitSelectionMode?.();
                  }}
                  aria-label="Clear selection"
                  title="Clear selection"
                  data-testid="selection-clear"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          );
        })()}

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
                    batchDiscardConfirm ? (
                      <>
                        <Button
                          variant="secondary"
                          size="sm"
                          className="h-7 px-2 border-red-400/60 bg-red-500/15 text-red-100 hover:bg-red-500/25"
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
                            setBatchDiscardConfirm(false);
                          }}
                          disabled={isDiscarding}
                          style={{ background: "transparent" }}
                        >
                          Discard {selectedFiles?.filter((e) => !e.staged).length ?? 0}
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          className="h-7 px-2"
                          onClick={() => setBatchDiscardConfirm(false)}
                          style={{ background: "transparent" }}
                        >
                          Cancel
                        </Button>
                      </>
                    ) : (
                      <Button
                        variant="secondary"
                        size="sm"
                        className="h-7 px-2 border-red-400/40 text-red-200 hover:bg-red-900/20"
                        onClick={() => setBatchDiscardConfirm(true)}
                        disabled={isDiscarding}
                        style={{ background: "transparent" }}
                      >
                        Discard
                      </Button>
                    )
                  )}
                </div>
              </div>
            )}
              <div style={{ paddingBottom: isMobile ? 72 : 12 }}>
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
                      onContextMenu={handleFileContextMenu}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onViewMetrics={openFileMetrics}
                      compactActions={compactHeader}
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
                {fileViewMode !== "tree" && files && !loadError && totalFilesCount > 0 && !hasVisibleFiles && listFilter !== "all" && (
                  <div
                    className="flex flex-col items-center justify-center px-4 py-6 text-center"
                    data-testid="filter-empty-state"
                  >
                    <File className="h-6 w-6 text-slate-700 mb-1.5" />
                    <p className="text-[13px] text-slate-400">
                      No files in “{listFilter === "agents" ? "Agents" : listFilter === "staged" ? "Staged" : "Conflicts"}”.
                    </p>
                    <p className="mt-0.5 text-[11px] text-slate-600">
                      Other changes are waiting under All.
                    </p>
                    <button
                      type="button"
                      className="mt-2 inline-flex min-h-8 items-center rounded-md border border-slate-700 px-3 text-xs font-medium text-slate-200 hover:bg-slate-800"
                      onClick={() => setListFilter("all")}
                    >
                      Show all changes
                    </button>
                  </div>
                )}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>

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
              label="View file metrics"
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
                label="Discard changes"
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
      <ContextMenu
        open={Boolean(groupContextMenu)}
        position={groupContextMenu ?? { x: 0, y: 0 }}
        title={groupMenuTarget ? `${groupMenuTarget.label} actions` : "Group actions"}
        closeLabel="Close group actions"
        items={groupMenuItems}
        onOpenChange={(open) => { if (!open) handleCloseGroupContextMenu(); }}
        triggers={[]}
        testId="group-context-menu"
      />

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
