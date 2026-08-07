import {
  Profiler,
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
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  EyeOff,
  ChevronDown,
  ChevronRight,
  ShieldCheck,
  History,
  X,
  BarChart3,
  ClipboardCheck,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import { useIsMobile } from "../hooks";
import type { DiffStats, RepoFileStats } from "../lib/api";
import { ViewModeCycleButton } from "./ViewModeCycleButton";
import { ProjectTreeView } from "./ProjectTreeView";
import { ContextMenu, type ContextMenuItem } from "./ContextMenu";
import { ChangeMetricsModal } from "./ChangeMetricsModal";
import { getFileStats, filterFileStats, filterCategoryStats } from "../lib/metrics";
import { useDiffStats } from "../lib/hooks";
import { MobileContext, type FileCategory, type FileListProps, summarizeFileStats, LineStats } from "./FileListTypes";
import { FileSection } from "./FileSection";

export type { GroupingRule, FileCategory, SelectedFileEntry, FileListProps } from "./FileListTypes";

function FileListImpl({
  files,
  fileStats,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  approvedChanges,
  approvedPaths,
  onStageApproved,
  isStagingApproved = false,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  onDiscardFile,
  onIgnoreFile,
  onStageAll,
  onUnstageAll,
  isStaging,
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
  repoId,
  onOpenReview,
  mobileSelectionMode = false,
  onEnterSelectionMode,
  onExitSelectionMode,
  onMobileSelectFile,
  fileHotspots,
}: FileListProps) {
  const isMobile = useIsMobile();
  const hasStaged = (files?.staged?.length ?? 0) > 0;
  const hasUnstaged =
    (files?.unstaged?.length ?? 0) > 0 || (files?.untracked?.length ?? 0) > 0;
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const scrollAreaRef = useRef<HTMLDivElement | null>(null);
  const cardRef = useRef<HTMLDivElement | null>(null);
  const [maxPathChars, setMaxPathChars] = useState(72);
  const [compactHeader, setCompactHeader] = useState(false);
  const [confirmingGroup, setConfirmingGroup] = useState<string | null>(null);
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(() => {
    const storageKey = `gct.collapsedGroups.${repoId ?? "default"}`;
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) return new Set(JSON.parse(stored) as string[]);
    } catch { /* ignore */ }
    return new Set();
  });
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

  // Context menu state for right-click blame action
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
    if (!contextMenu || !onBlameFile) return [];
    return [
      {
        label: "View File History",
        icon: <History className="h-4 w-4" />,
        onClick: () => onBlameFile(contextMenu.file),
      },
    ];
  }, [contextMenu, onBlameFile]);

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
  const showApprovedBanner = Boolean(
    approvedChanges?.available && (approvedChanges.committableFiles ?? 0) > 0,
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
      return key in sectionExpanded ? sectionExpanded[key] : defaultExpanded;
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
      // badges (binary/approved ~100px), padding (~16px) = ~250px total
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

  // Scroll to file when returning from Related Files panel
  useEffect(() => {
    if (!scrollToFile || fileViewMode === "tree") return; // Tree view handles its own scrolling

    const timeoutId = setTimeout(() => {
      const element = document.querySelector(`[data-file-path="${CSS.escape(scrollToFile)}"]`);
      const scroller = scrollAreaRef.current;
      if (element instanceof HTMLElement && scroller) {
        const target = element.getBoundingClientRect();
        const container = scroller.getBoundingClientRect();
        const nextTop = scroller.scrollTop + target.top - container.top - (scroller.clientHeight - target.height) / 2;
        scroller.scrollTo({ top: Math.max(0, nextTop), behavior: "smooth" });
      }
      onScrollComplete?.();
    }, 100);

    return () => clearTimeout(timeoutId);
  }, [scrollToFile, onScrollComplete, fileViewMode]);

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
    const firstFrame = requestAnimationFrame(() => requestAnimationFrame(restore));
    return () => cancelAnimationFrame(firstFrame);
    // Mount-only: intentionally not re-running on scrollTopStore/scrollToFile changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const groupedSections = useMemo(() => {
    if (!groupingActive) return [];
    const categories: Array<[FileCategory, string[] | undefined]> = [
      ["conflicts", files?.conflicts],
      ["staged", files?.staged],
      ["unstaged", files?.unstaged],
      ["untracked", files?.untracked],
    ];
    const categoryByPath = new Map<string, FileCategory>();
    for (const [category, paths] of categories) {
      for (const path of paths ?? []) categoryByPath.set(path, category);
    }
    return (resolvedGroups ?? []).map((group) => {
      const groupedFiles: Record<FileCategory, string[]> = {
        conflicts: [], staged: [], unstaged: [], untracked: [],
      };
      for (const path of group.files) {
        const category = categoryByPath.get(path);
        if (category) groupedFiles[category].push(path);
      }
      return {
        id: group.key,
        label: group.label,
        kind: group.kind,
        displayPrefix: group.root ?? "",
        files: groupedFiles,
      };
    });
  }, [files, groupingActive, resolvedGroups]);

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
        className={`flex flex-col min-w-0 ${fillHeight ? "h-full" : "h-auto"}`}
        data-testid="file-list-panel"
      >
        <CardHeader className="flex-row items-center justify-between space-y-0 py-3 gap-2 min-w-0">
          <CardTitle className="flex items-center gap-2 min-w-0">
            <button
              className="inline-flex items-center justify-center rounded hover:bg-slate-800/70 transition-colors touch-target"
              onClick={handleToggleCollapse}
              aria-label={collapsed ? "Expand changes" : "Collapse changes"}
              type="button"
            >
              {collapsed ? (
                <ChevronRight className="h-3 w-3 text-slate-400" />
              ) : (
                <ChevronDown className="h-3 w-3 text-slate-400" />
              )}
            </button>
            <span className="truncate">Changes</span>
            {compactHeader ? (
              <button
                type="button"
                className="inline-flex items-center justify-center rounded hover:bg-slate-800/70 transition-colors touch-target"
                onClick={(e) => {
                  e.stopPropagation();
                  setMetricsModal({ mode: "aggregate" });
                }}
                aria-label="View change metrics"
              >
                <BarChart3 className="h-3.5 w-3.5 text-slate-400" />
              </button>
            ) : (
              <LineStats
                stats={totalStats}
                onClick={() => setMetricsModal({ mode: "aggregate" })}
              />
            )}
          </CardTitle>
          <div className="flex flex-wrap gap-2 justify-end min-w-0">
            {!mobileSelectionMode && hasUnstaged && (
              <Button
                variant="outline"
                size="sm"
                onClick={onStageAll}
                disabled={isStaging}
                className={compactHeader ? "h-11 w-11 p-0" : "min-w-0 whitespace-normal px-3"}
                data-testid="stage-all-button"
                title="Stage All"
              >
                <Plus className="h-3 w-3" />
                {!compactHeader && <span className="ml-1">Stage All</span>}
              </Button>
            )}
            {!mobileSelectionMode && hasStaged && (
              <Button
                variant="outline"
                size="sm"
                onClick={onUnstageAll}
                disabled={isStaging}
                className={compactHeader ? "h-11 w-11 p-0" : "min-w-0 whitespace-normal px-3"}
                data-testid="unstage-all-button"
                title="Unstage All"
              >
                <Minus className="h-3 w-3" />
                {!compactHeader && <span className="ml-1">Unstage All</span>}
              </Button>
            )}
            <ViewModeCycleButton
              mode={fileViewMode}
              onCycle={handleCycleViewMode}
              groupingAvailable={groupingAvailable}
              compact={compactHeader}
            />
          </div>
        </CardHeader>

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
                      variant="outline"
                      size="sm"
                      className="h-7 px-2 border-emerald-500/40 text-emerald-200 hover:bg-emerald-900/20"
                      onClick={() => {
                        const unstaged = selectedFiles?.filter((e) => !e.staged).map((e) => e.path) ?? [];
                        if (unstaged.length > 0) onStagePaths?.(unstaged);
                      }}
                      disabled={isStaging}
                    >
                      Stage
                    </Button>
                  )}
                  {selectionHasStaged && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 px-2"
                      onClick={() => {
                        const staged = selectedFiles?.filter((e) => e.staged).map((e) => e.path) ?? [];
                        staged.forEach((p) => onUnstageFile(p));
                      }}
                      disabled={isStaging}
                    >
                      Unstage
                    </Button>
                  )}
                  {selectionHasDiscardable && (
                    <Button
                      variant="outline"
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
                    >
                      Discard
                    </Button>
                  )}
                </div>
              </div>
            )}
            {showApprovedBanner && (
              <div className="mx-2 mt-2 mb-1 rounded-md border border-emerald-800/50 bg-emerald-950/20 p-2 text-xs text-emerald-200">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <ShieldCheck className="h-3.5 w-3.5 text-emerald-300" />
                    <span>Approved changes ready</span>
                    <span className="text-emerald-300">
                      {approvedChanges?.committableFiles ?? 0} file
                      {(approvedChanges?.committableFiles ?? 0) !== 1
                        ? "s"
                        : ""}
                    </span>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onStageApproved}
                    disabled={isStagingApproved}
                    className="h-7 px-2"
                    data-testid="stage-approved-button"
                  >
                    Stage approved
                  </Button>
                </div>
                {approvedChanges?.warning && (
                  <div className="mt-1 text-[11px] text-emerald-300/80">
                    {approvedChanges.warning}
                  </div>
                )}
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
                ) : groupingActive ? (
                  groupedSections.map((group) => {
                    const stageable = [
                      ...group.files.unstaged,
                      ...group.files.untracked,
                      ...group.files.conflicts,
                    ];
                    const discardTracked = group.files.unstaged;
                    const discardUntracked = group.files.untracked;
                    const discardCount =
                      discardTracked.length + discardUntracked.length;
                    const groupCount =
                      group.files.conflicts.length +
                      group.files.staged.length +
                      group.files.unstaged.length +
                      group.files.untracked.length;
                    const isGroupCollapsed = collapsedGroups.has(group.id);

                    return (
                      <div
                        key={group.id}
                        className="mb-4 rounded-lg border border-slate-800/80 bg-slate-950/40"
                        data-testid={`file-group-${group.id}`}
                      >
                        <div
                          className={`flex flex-wrap items-center justify-between gap-2 px-3 py-2 ${isGroupCollapsed ? "" : "border-b border-slate-800/70"}`}
                        >
                          <button
                            type="button"
                            className="flex items-center gap-2 min-w-0 hover:bg-slate-800/30 rounded px-1 -ml-1 transition-colors"
                            onClick={() => toggleGroupCollapse(group.id)}
                            data-testid={`file-group-toggle-${group.id}`}
                          >
                            {isGroupCollapsed ? (
                              <ChevronRight className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
                            ) : (
                              <ChevronDown className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
                            )}
                            <div className="min-w-0 text-left">
                              <div className={`font-semibold uppercase tracking-wider text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`}>
                                {group.label}
                              </div>
                              {group.displayPrefix && (
                                <div className={`text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                                  {group.displayPrefix}
                                </div>
                              )}
                            </div>
                          </button>
                          <div className={`flex items-center gap-2 text-slate-500 ${isMobile ? "text-sm" : "text-xs"}`}>
                            {onOpenReview && group.kind === "scenario" && group.displayPrefix ? (
                              <button
                                type="button"
                                className="h-7 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
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
                                className="hover:underline decoration-slate-600 cursor-pointer"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  openGroupMetrics(group.files, group.label);
                                }}
                              >
                                {groupCount} files
                              </button>
                            )}
                            {!mobileSelectionMode &&
                              !isGroupCollapsed &&
                              stageable.length > 0 &&
                              onStagePaths && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => onStagePaths(stageable)}
                                  disabled={isStaging}
                                  className={compactHeader ? "h-11 w-11 p-0" : "h-7 px-2"}
                                  title="Stage All"
                                >
                                  {compactHeader ? <Plus className="h-3 w-3" /> : "Stage All"}
                                </Button>
                              )}
                            {!mobileSelectionMode &&
                              !isGroupCollapsed &&
                              discardCount > 0 &&
                              onDiscardPaths && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => setConfirmingGroup(group.id)}
                                  disabled={isDiscarding}
                                  className={`border-red-400/40 text-red-200 hover:bg-red-900/20 ${compactHeader ? "h-11 w-11 p-0" : "h-7 px-2"}`}
                                  title="Discard All"
                                >
                                  {compactHeader ? <Trash2 className="h-3 w-3" /> : "Discard All"}
                                </Button>
                              )}
                          </div>
                        </div>
                        {!isGroupCollapsed && (
                          <>
                            {confirmingGroup === group.id &&
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
                            <div className="px-2 py-2">
                              <FileSection
                                key={`${group.id}-conflicts`}
                                title="Conflicts"
                                category="conflicts"
                                expanded={isSectionExpanded(group.id, "conflicts", true)}
                                onToggle={() => toggleSectionCollapse(group.id, "conflicts", true)}
                                files={group.files.conflicts}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                pendingPaths={pendingPaths}
                                changeStats={summarizeFileStats(
                                  group.files.conflicts,
                                  fileStats?.unstaged,
                                )}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                resolvedGroups={resolvedGroups}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.conflicts, "unstaged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-staged`}
                                title="Staged"
                                category="staged"
                                expanded={isSectionExpanded(group.id, "staged", true)}
                                onToggle={() => toggleSectionCollapse(group.id, "staged", true)}
                                files={group.files.staged}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <FilePlus className="h-3.5 w-3.5 text-emerald-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onUnstageFile}
                                actionIcon={
                                  <Minus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Unstage file"
                                pendingPaths={pendingPaths}
                                changeStats={summarizeFileStats(
                                  group.files.staged,
                                  fileStats?.staged,
                                )}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                resolvedGroups={resolvedGroups}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.staged, "staged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-unstaged`}
                                title="Modified"
                                category="unstaged"
                                expanded={isSectionExpanded(group.id, "unstaged", true)}
                                onToggle={() => toggleSectionCollapse(group.id, "unstaged", true)}
                                files={group.files.unstaged}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <FileX className="h-3.5 w-3.5 text-amber-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                pendingPaths={pendingPaths}
                                changeStats={summarizeFileStats(
                                  group.files.unstaged,
                                  fileStats?.unstaged,
                                )}
                                onDiscard={handleDiscardUnstaged}
                                isDiscarding={isDiscarding}
                                confirmingDiscard={confirmingDiscard}
                                onConfirmDiscard={onConfirmDiscard}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                resolvedGroups={resolvedGroups}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.unstaged, "unstaged", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                              <FileSection
                                key={`${group.id}-untracked`}
                                title="Untracked"
                                category="untracked"
                                expanded={isSectionExpanded(group.id, "untracked", false)}
                                onToggle={() => toggleSectionCollapse(group.id, "untracked", false)}
                                files={group.files.untracked}
                                fileStatuses={files?.statuses}
                                binaryFiles={binarySet}
                                approvedFiles={approvedPaths}
                                maxPathChars={maxPathChars}
                                icon={
                                  <File className="h-3.5 w-3.5 text-slate-500" />
                                }
                                selectedFiles={selectedFiles}
                                selectedKeySet={selectedKeySet}
                                selectionKey={selectionKey}
                                onSelectFile={onSelectFile}
                                onAction={onStageFile}
                                actionIcon={
                                  <Plus className="h-3 w-3 text-slate-400" />
                                }
                                actionLabel="Stage file"
                                pendingPaths={pendingPaths}
                                changeStats={summarizeFileStats(
                                  group.files.untracked,
                                  fileStats?.untracked,
                                )}
                                defaultExpanded={false}
                                onDiscard={handleDiscardUntracked}
                                isDiscarding={isDiscarding}
                                confirmingDiscard={confirmingDiscard}
                                onConfirmDiscard={onConfirmDiscard}
                                onIgnore={handleIgnoreFile}
                                isIgnoring={isIgnoring}
                                confirmingIgnore={confirmingIgnore}
                                onConfirmIgnore={onConfirmIgnore}
                                resolvedGroups={resolvedGroups}
                                onOpenMobileActions={handleOpenMobileActions}
                                onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                                mobileSelectionMode={mobileSelectionMode}
                                onLongPress={handleLongPress}
                                onMobileTap={handleMobileTap}
                                onStatsClick={() => openGroupCategoryMetrics(group.files.untracked, "untracked", group.label)}
                                onViewMetrics={openFileMetrics}
                              />
                            </div>
                          </>
                        )}
                      </div>
                    );
                  })
                ) : (
                  <>
                    {/* Conflicts - Always show first if any */}
                    <FileSection
                      title="Conflicts"
                      category="conflicts"
                      expanded={isSectionExpanded("__flat__", "conflicts", true)}
                      onToggle={() => toggleSectionCollapse("__flat__", "conflicts", true)}
                      files={files?.conflicts ?? []}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      approvedFiles={approvedPaths}
                      maxPathChars={maxPathChars}
                      icon={
                        <AlertTriangle className="h-3.5 w-3.5 text-red-500" />
                      }
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onStageFile}
                      actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Stage file"
                      pendingPaths={pendingPaths}
                      changeStats={summarizeFileStats(
                        files?.conflicts ?? [],
                        fileStats?.unstaged,
                      )}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                    resolvedGroups={resolvedGroups}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
                    />

                    {/* Staged Changes */}
                    <FileSection
                      title="Staged"
                      category="staged"
                      expanded={isSectionExpanded("__flat__", "staged", true)}
                      onToggle={() => toggleSectionCollapse("__flat__", "staged", true)}
                      files={files?.staged ?? []}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      approvedFiles={approvedPaths}
                      maxPathChars={maxPathChars}
                      icon={
                        <FilePlus className="h-3.5 w-3.5 text-emerald-500" />
                      }
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onUnstageFile}
                      actionIcon={<Minus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Unstage file"
                      pendingPaths={pendingPaths}
                      changeStats={summarizeFileStats(
                        files?.staged ?? [],
                        fileStats?.staged,
                      )}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                    resolvedGroups={resolvedGroups}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
                    />

                    {/* Unstaged Changes */}
                    <FileSection
                      title="Modified"
                      category="unstaged"
                      expanded={isSectionExpanded("__flat__", "unstaged", true)}
                      onToggle={() => toggleSectionCollapse("__flat__", "unstaged", true)}
                      files={files?.unstaged ?? []}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      approvedFiles={approvedPaths}
                      maxPathChars={maxPathChars}
                      icon={<FileX className="h-3.5 w-3.5 text-amber-500" />}
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onStageFile}
                      actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Stage file"
                      pendingPaths={pendingPaths}
                      changeStats={summarizeFileStats(
                        files?.unstaged ?? [],
                        fileStats?.unstaged,
                      )}
                      onDiscard={handleDiscardUnstaged}
                      isDiscarding={isDiscarding}
                      confirmingDiscard={confirmingDiscard}
                      onConfirmDiscard={onConfirmDiscard}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                    resolvedGroups={resolvedGroups}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
                    />

                    {/* Untracked Files */}
                    <FileSection
                      title="Untracked"
                      category="untracked"
                      expanded={isSectionExpanded("__flat__", "untracked", false)}
                      onToggle={() => toggleSectionCollapse("__flat__", "untracked", false)}
                      files={files?.untracked ?? []}
                      fileStatuses={files?.statuses}
                      binaryFiles={binarySet}
                      approvedFiles={approvedPaths}
                      maxPathChars={maxPathChars}
                      icon={<File className="h-3.5 w-3.5 text-slate-500" />}
                      selectedFiles={selectedFiles}
                      selectedKeySet={selectedKeySet}
                      selectionKey={selectionKey}
                      onSelectFile={onSelectFile}
                      onAction={onStageFile}
                      actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
                      actionLabel="Stage file"
                      pendingPaths={pendingPaths}
                      changeStats={summarizeFileStats(
                        files?.untracked ?? [],
                        fileStats?.untracked,
                      )}
                      defaultExpanded={false}
                      onDiscard={handleDiscardUntracked}
                      isDiscarding={isDiscarding}
                      confirmingDiscard={confirmingDiscard}
                      onConfirmDiscard={onConfirmDiscard}
                      onIgnore={handleIgnoreFile}
                      isIgnoring={isIgnoring}
                      confirmingIgnore={confirmingIgnore}
                      onConfirmIgnore={onConfirmIgnore}
                    resolvedGroups={resolvedGroups}
                      onOpenMobileActions={handleOpenMobileActions}
                      onContextMenu={onBlameFile ? handleFileContextMenu : undefined}
                      mobileSelectionMode={mobileSelectionMode}
                      onLongPress={handleLongPress}
                      onMobileTap={handleMobileTap}
                      onStatsClick={openAggregateMetrics}
                      onViewMetrics={openFileMetrics}
                    />
                  </>
                )}

                {/* Empty State - only show for flat/grouped views */}
                {fileViewMode !== "tree" && files && totalFilesCount === 0 && (
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
      <ContextMenu
        isOpen={Boolean(contextMenu)}
        position={contextMenu ?? { x: 0, y: 0 }}
        items={contextMenuItems}
        onClose={handleCloseContextMenu}
      />

      {/* Change metrics modal */}
      <ChangeMetricsModal
        isOpen={metricsModal !== null}
        onClose={() => setMetricsModal(null)}
        mode={metricsModal?.mode ?? "aggregate"}
        stats={metricsModal?.stats}
        filePath={metricsModal?.path}
        fileStats={metricsModal?.filteredFileStats ?? fileStats}
        title={metricsModal?.title}
        enhancedStats={enhancedQuery.stats}
        enhancedLoading={enhancedQuery.isLoading}
        isUntracked={metricsModal?.category === "untracked"}
        fileHotspots={fileHotspots}
      />
    </MobileContext.Provider>
  );
}

export function FileList(props: FileListProps) {
  return (
    <Profiler id="FileList" onRender={onProfilerRender}>
      <FileListImpl {...props} />
    </Profiler>
  );
}
