import { useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { ChevronDown, ChevronRight } from "lucide-react";
import type { DiffStats } from "../lib/api";
import { formatPath } from "../lib/utils";
import { MobileContext, type FileCategory, type FileListProps, type FileRowProps, LineStats, getStatusBadge } from "./FileListTypes";
import { FileRow } from "./FileRow";

export type ChangeListSection = {
  key: string;
  title: string;
  category: FileCategory;
  files: string[];
  expanded: boolean;
  icon: React.ReactNode;
  actionIcon: React.ReactNode;
  actionLabel: string;
  onAction: (path: string) => void;
  onToggle: () => void;
  changeStats?: DiffStats;
  metricsState?: "loading" | "refreshing" | "ready" | "unavailable";
  onStatsClick?: () => void;
  onDiscard?: (path: string) => void;
};

export type ChangeListBlock =
  | { type: "section"; section: ChangeListSection }
  | { type: "header"; key: string; render: () => React.ReactNode; estimateSize: number };

interface VirtualizedFileSectionsProps {
  blocks: ChangeListBlock[];
  scrollRef: React.RefObject<HTMLDivElement | null>;
  fileStatuses?: Record<string, string>;
  binaryFiles?: Set<string>;
  runIndex?: FileListProps["runIndex"];
  onOpenRun?: FileListProps["onOpenRun"];
  maxPathChars: number;
  selectedKeySet?: Set<string>;
  selectionKey: FileListProps["selectionKey"];
  onSelectFile: FileListProps["onSelectFile"];
  pendingPaths?: ReadonlySet<string>;
  isDiscarding: boolean;
  isIgnoring: boolean;
  confirmingDiscard: string | null;
  onConfirmDiscard: FileListProps["onConfirmDiscard"];
  confirmingIgnore: string | null;
  onConfirmIgnore: FileListProps["onConfirmIgnore"];
  onIgnore?: FileRowProps["onIgnore"];
  resolvedGroups?: FileListProps["resolvedGroups"];
  onOpenMobileActions?: FileRowProps["onOpenMobileActions"];
  onContextMenu?: FileRowProps["onContextMenu"];
  mobileSelectionMode: boolean;
  onLongPress?: FileRowProps["onLongPress"];
  onMobileTap?: FileRowProps["onMobileTap"];
  onViewMetrics?: FileRowProps["onViewMetrics"];
  scrollToFile?: string;
  onScrollComplete?: () => void;
}

type VirtualItem =
  | { type: "custom"; block: Extract<ChangeListBlock, { type: "header" }>; key: string }
  | { type: "header"; section: ChangeListSection; key: string }
  | { type: "file"; section: ChangeListSection; file: string; key: string };

type VirtualSegment = {
  start: number;
  count: number;
  block: ChangeListBlock;
};

export function VirtualizedFileSections({
  blocks,
  scrollRef,
  fileStatuses,
  binaryFiles,
  runIndex,
  onOpenRun,
  maxPathChars,
  selectedKeySet,
  selectionKey,
  onSelectFile,
  pendingPaths,
  isDiscarding,
  isIgnoring,
  confirmingDiscard,
  onConfirmDiscard,
  confirmingIgnore,
  onConfirmIgnore,
  onIgnore,
  resolvedGroups,
  onOpenMobileActions,
  onContextMenu,
  mobileSelectionMode,
  onLongPress,
  onMobileTap,
  onViewMetrics,
  scrollToFile,
  onScrollComplete,
}: VirtualizedFileSectionsProps) {
  const isMobile = useContext(MobileContext);
  const listRef = useRef<HTMLDivElement>(null);
  // Child layout effects run before the ancestor attaches its ref. Resolve
  // the already-mounted local list's scroller on that first commit too.
  const getScrollElement = useCallback(() => scrollRef.current ?? listRef.current?.closest<HTMLDivElement>("[data-testid=changes-scroll-region]") ?? null, [scrollRef]);
  // Keep only one descriptor per section/header. The old flattened array also
  // allocated a VirtualItem object for every file before the virtualizer could
  // discard offscreen rows, which was costly for large repositories.
  const { segments, virtualCount } = useMemo(() => {
    const segments: VirtualSegment[] = [];
    let virtualCount = 0;
    for (const block of blocks) {
      if (block.type === "section" && block.section.files.length === 0) continue;
      const count = block.type === "header"
        ? 1
        : 1 + (block.section.expanded ? block.section.files.length : 0);
      segments.push({ start: virtualCount, count, block });
      virtualCount += count;
    }
    return { segments, virtualCount };
  }, [blocks]);
  const getVirtualItem = useCallback((index: number): VirtualItem | undefined => {
    let low = 0;
    let high = segments.length;
    while (low < high) {
      const middle = low + Math.floor((high - low) / 2);
      const segment = segments[middle];
      if (segment && segment.start + segment.count <= index) low = middle + 1;
      else high = middle;
    }
    const segment = segments[low];
    if (!segment) return undefined;
    const offset = index - segment.start;
    if (segment.block.type === "header") {
      return { type: "custom", block: segment.block, key: segment.block.key };
    }
    const { section } = segment.block;
    if (offset === 0) return { type: "header", section, key: `${section.key}:header` };
    const file = section.files[offset - 1];
    return file === undefined
      ? undefined
      : { type: "file", section, file, key: `${section.key}:file:${file}` };
  }, [segments]);
  // The virtualizer keys its measurement cache by getItemKey identity. Keep
  // it stable during scrolling so a viewport update does not remeasure N rows.
  const getItemKey = useCallback((index: number) => getVirtualItem(index)?.key ?? index, [getVirtualItem]);
  const estimateSize = useCallback((index: number) => {
    const item = getVirtualItem(index);
    return item?.type === "custom" ? item.block.estimateSize
      : item?.type === "header" ? (isMobile ? 56 : 36)
      : (isMobile ? 48 : 32);
  }, [getVirtualItem, isMobile]);
  const rowVirtualizer = useVirtualizer({
    count: virtualCount,
    getScrollElement,
    getItemKey,
    estimateSize,
    overscan: 8,
    // Changes content starts below the optional mobile selection toolbar.
    scrollMargin: scrollRef.current?.querySelector<HTMLElement>("[data-testid=mobile-selection-toolbar]")?.offsetHeight ?? 0,
  });

  useEffect(() => {
    if (!scrollToFile) return;
    let itemIndex = -1;
    for (const segment of segments) {
      if (segment.block.type !== "section" || !segment.block.section.expanded) continue;
      const fileOffset = segment.block.section.files.indexOf(scrollToFile);
      if (fileOffset >= 0) {
        itemIndex = segment.start + 1 + fileOffset;
        break;
      }
    }
    // The parent may still be opening the destination's collapsed section.
    if (itemIndex < 0) return;
    rowVirtualizer.scrollToIndex(itemIndex, { align: "center" });
    const frame = requestAnimationFrame(() => onScrollComplete?.());
    return () => cancelAnimationFrame(frame);
  }, [onScrollComplete, rowVirtualizer, scrollToFile, segments]);

  return (
    <div ref={listRef} data-change-rows className="relative min-w-0" style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
      {rowVirtualizer.getVirtualItems().map((virtualItem) => {
        const item = getVirtualItem(virtualItem.index);
        if (!item) return null;
        if (item.type === "custom") return (
          <div key={item.key} ref={rowVirtualizer.measureElement} data-index={virtualItem.index}
            className="absolute top-0 left-0 right-0"
            style={{ transform: `translateY(${virtualItem.start - (rowVirtualizer.options?.scrollMargin ?? 0)}px)` }}>
            {item.block.render()}
          </div>
        );
        const { section } = item;

        if (item.type === "header") {
          return (
            <div
              key={item.key}
              ref={rowVirtualizer.measureElement}
              data-index={virtualItem.index}
              data-testid={`file-section-${section.category}`}
              className="absolute top-0 left-0 right-0 mb-1"
              style={{ transform: `translateY(${virtualItem.start - (rowVirtualizer.options?.scrollMargin ?? 0)}px)` }}
            >
              <div className={`flex w-full items-center gap-2 rounded px-2 py-1.5 transition-colors hover:bg-slate-800/50 ${isMobile ? "min-h-11" : ""}`}>
                <button
                  type="button"
                  className={`flex min-w-0 flex-1 items-center gap-2 text-left ${isMobile ? "min-h-11" : ""}`}
                  onClick={section.onToggle}
                  aria-expanded={section.expanded}
                  data-testid={`file-section-toggle-${section.category}`}
                >
                  {section.expanded ? (
                    <ChevronDown className={`text-slate-500 ${isMobile ? "h-4 w-4" : "h-3 w-3"}`} />
                  ) : (
                    <ChevronRight className={`text-slate-500 ${isMobile ? "h-4 w-4" : "h-3 w-3"}`} />
                  )}
                  {section.icon}
                  <span className="truncate text-[11px] font-semibold uppercase tracking-[0.09em] text-slate-400">
                    {section.title}
                  </span>
                </button>
                <div className="ml-auto flex shrink-0 items-center gap-2 tabular-nums">
                  {section.changeStats ? (
                    <LineStats stats={section.changeStats} compact onClick={section.onStatsClick} />
                  ) : section.metricsState === "loading" ? (
                    <span className="h-3 w-14 animate-pulse rounded-sm bg-slate-700/80" role="status" aria-label="Loading change metrics" data-testid="file-stats-loading" />
                  ) : section.metricsState === "refreshing" ? (
                    <span className="h-3 w-14 rounded-sm bg-slate-700/50" role="status" aria-label="Updating change metrics" data-testid="file-stats-refreshing" />
                  ) : section.metricsState === "unavailable" ? (
                    <span className="text-[10px] font-medium text-amber-400" title="Change metrics unavailable">Metrics unavailable</span>
                  ) : null}
                  <span className="min-w-4 text-right text-xs text-slate-500" aria-label={`${section.files.length} files`} title={`${section.files.length} files`}>
                    {section.files.length}
                  </span>
                </div>
              </div>
            </div>
          );
        }

        const { file } = item;
        const isStaged = section.category === "staged";
        const entry = { path: file, staged: isStaged };
        return (
          <div
            key={item.key}
            ref={rowVirtualizer.measureElement}
            data-index={virtualItem.index}
            className="absolute top-0 left-0 right-0"
            style={{ transform: `translateY(${virtualItem.start - (rowVirtualizer.options?.scrollMargin ?? 0)}px)` }}
          >
            <ul className="contents">
              <FileRow
                file={file}
                displayPath={formatPath(file, maxPathChars)}
                badge={getStatusBadge(fileStatuses?.[file], section.category)}
                isSelected={selectedKeySet?.has(selectionKey(entry)) ?? false}
                isStaged={isStaged}
                canDiscard={section.category === "unstaged" || section.category === "untracked"}
                isLoading={pendingPaths?.has(file) ?? false}
                isDiscarding={isDiscarding}
                isIgnoring={isIgnoring}
                isBinary={binaryFiles?.has(file) ?? false}
                runAttribution={runIndex?.get(file)}
                onOpenRun={onOpenRun}
                itemTestId={`file-item-${section.category}`}
                actionTestId={`file-action-${section.category}`}
                discardTestId={`file-discard-${section.category}`}
                ignoreTestId={`file-ignore-${section.category}`}
                actionIcon={section.actionIcon}
                actionLabel={section.actionLabel}
                onSelectFile={onSelectFile}
                onAction={section.onAction}
                onDiscard={section.onDiscard}
                onConfirmDiscard={onConfirmDiscard}
                onIgnore={onIgnore}
                onConfirmIgnore={onConfirmIgnore}
                confirmingDiscard={confirmingDiscard}
                confirmingIgnore={confirmingIgnore}
                resolvedGroups={resolvedGroups}
                onOpenMobileActions={onOpenMobileActions}
                onContextMenu={onContextMenu}
                mobileSelectionMode={mobileSelectionMode}
                onLongPress={onLongPress}
                onMobileTap={onMobileTap}
                onViewMetrics={onViewMetrics}
                category={section.category}
              />
            </ul>
          </div>
        );
      })}
    </div>
  );
}
