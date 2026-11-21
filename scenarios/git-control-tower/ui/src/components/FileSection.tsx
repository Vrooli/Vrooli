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
  const virtualItems = useMemo<VirtualItem[]>(
    () => blocks.flatMap((block): VirtualItem[] => {
      if (block.type === "header") return [{ type: "custom", block, key: block.key }];
      const { section } = block;
      if (section.files.length === 0) return [];
      return [
        { type: "header", section, key: `${section.key}:header` },
        ...(section.expanded
          ? section.files.map((file): VirtualItem => ({ type: "file", section, file, key: `${section.key}:file:${file}` }))
          : []),
      ];
    }),
    [blocks],
  );
  // The virtualizer keys its measurement cache by getItemKey identity. Keep
  // it stable during scrolling so a viewport update does not remeasure N rows.
  const getItemKey = useCallback((index: number) => virtualItems[index]?.key ?? index, [virtualItems]);
  const estimateSize = useCallback((index: number) => {
    const item = virtualItems[index];
    return item?.type === "custom" ? item.block.estimateSize
      : item?.type === "header" ? (isMobile ? 56 : 36)
      : (isMobile ? 48 : 32);
  }, [virtualItems, isMobile]);
  const rowVirtualizer = useVirtualizer({
    count: virtualItems.length,
    getScrollElement,
    getItemKey,
    estimateSize,
    overscan: 8,
    // Changes content starts below the optional mobile selection toolbar.
    scrollMargin: scrollRef.current?.querySelector<HTMLElement>("[data-testid=mobile-selection-toolbar]")?.offsetHeight ?? 0,
  });

  useEffect(() => {
    if (!scrollToFile) return;
    const itemIndex = virtualItems.findIndex((item) =>
      item.type === "file" && item.file === scrollToFile,
    );
    // The parent may still be opening the destination's collapsed section.
    if (itemIndex < 0) return;
    rowVirtualizer.scrollToIndex(itemIndex, { align: "center" });
    const frame = requestAnimationFrame(() => onScrollComplete?.());
    return () => cancelAnimationFrame(frame);
  }, [onScrollComplete, rowVirtualizer, scrollToFile, virtualItems]);

  return (
    <div ref={listRef} data-change-rows className="relative min-w-0" style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
      {rowVirtualizer.getVirtualItems().map((virtualItem) => {
        const item = virtualItems[virtualItem.index];
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
              <div className={`flex items-center gap-2 w-full px-2 py-1.5 rounded hover:bg-slate-800/50 transition-colors ${isMobile ? "min-h-11" : ""}`}>
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
                  <span className={`font-medium text-slate-400 uppercase tracking-wider ${isMobile ? "text-sm" : "text-xs"}`}>
                    {section.title}
                  </span>
                </button>
                <div className="ml-auto flex items-center gap-2">
                  <LineStats stats={section.changeStats} compact onClick={section.onStatsClick} />
                  <span className={`text-slate-600 ${isMobile ? "text-sm" : "text-xs"}`}>{section.files.length}</span>
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
