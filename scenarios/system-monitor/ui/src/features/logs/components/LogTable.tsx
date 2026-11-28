import { VariableSizeList, type ListOnScrollProps, type ListChildComponentProps } from 'react-window';
import { useCallback, useEffect, useRef, useState } from 'react';
import type { LogEntry } from '../types';
import { LogRow } from './LogRow';

interface LogTableProps {
  entries: LogEntry[];
  height?: number;
  rowHeight?: number;
  onScrollTopChange?: (atTop: boolean) => void;
  onSelect?: (entry: LogEntry) => void;
  selected?: LogEntry | null;
  scrollToTopRef?: React.RefObject<(() => void) | null>;
  scrollToOffsetRef?: React.RefObject<((offset: number) => void) | null>;
  readScrollOffsetRef?: React.RefObject<(() => number) | null>;
  initialScrollOffset?: number;
  onScrollOffset?: (offset: number) => void;
}

const TOP_THRESHOLD = 8; // px

type RowData = {
  entries: LogEntry[];
  expandedIds: Set<string>;
  toggleExpanded: (id: string) => void;
  onSelect?: (entry: LogEntry) => void;
  selected?: LogEntry | null;
};

const VirtualLogRow = ({ index, style, data }: ListChildComponentProps<RowData>) => {
  const entry = data.entries[index];
  if (!entry) return null;
  const id = entry.cursor ?? String(index);
  return <LogRow entry={entry} style={style} expanded={data.expandedIds.has(id)} onToggleExpanded={() => { data.toggleExpanded(id); }} onSelect={data.onSelect ? () => { data.onSelect?.(entry); } : undefined} selected={data.selected === entry || (!!entry.cursor && entry.cursor === data.selected?.cursor)} />;
};

export const LogTable = ({
  entries,
  height,
  rowHeight = 22,
  onScrollTopChange,
  onSelect,
  selected,
  scrollToTopRef,
  scrollToOffsetRef,
  readScrollOffsetRef,
  initialScrollOffset,
  onScrollOffset,
}: LogTableProps) => {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const listRef = useRef<VariableSizeList>(null);
  const outerRef = useRef<HTMLDivElement>(null);
  if (scrollToTopRef) scrollToTopRef.current = () => listRef.current?.scrollTo(0);
  if (scrollToOffsetRef) scrollToOffsetRef.current = offset => {
    // A hidden list can clamp its DOM scroll position while react-window still
    // holds the old offset; scrollTo alone is then a no-op for the same value.
    if (outerRef.current) outerRef.current.scrollTop = offset;
    listRef.current?.scrollTo(offset);
  };
  if (readScrollOffsetRef) readScrollOffsetRef.current = () => outerRef.current?.scrollTop ?? 0;
  const compactViewport = typeof window !== 'undefined' && window.matchMedia('(max-width: 48rem)').matches;
  const collapsedHeight = compactViewport ? Math.max(rowHeight * 5 + 16, 126) : Math.max(rowHeight * 3, 72);
  const longCollapsedHeight = compactViewport ? Math.max(rowHeight * 6 + 16, 148) : collapsedHeight;
  const expandedHeight = compactViewport ? Math.max(rowHeight * 9 + 16, 214) : Math.max(rowHeight * 8, 176);
  useEffect(() => { listRef.current?.resetAfterIndex(0); }, [compactViewport, rowHeight]);
  const toggleExpanded = useCallback((id: string) => {
    setExpandedIds(previous => {
      const next = new Set(previous);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
    requestAnimationFrame(() => listRef.current?.resetAfterIndex(0));
  }, []);

  if (entries.length === 0) {
    return (
      <div className="card" data-sm-style="sm-style-7b635e08e2">
        <div className="text-sm text-muted">No matching entries for these filters.</div>
      </div>
    );
  }

  const handleScroll = ({ scrollOffset }: ListOnScrollProps) => {
    onScrollOffset?.(scrollOffset);
    if (onScrollTopChange) {
      onScrollTopChange(scrollOffset <= TOP_THRESHOLD);
    }
  };

  return (
    <div className="log-table">
      <div className="log-table__head" aria-hidden="true"><span>Time UTC</span><span>Priority</span><span>Unit / message</span></div>
      <VariableSizeList
        ref={listRef}
        outerRef={outerRef}
        height={height ?? (compactViewport ? 480 : 576)}
        itemCount={entries.length}
        itemSize={index => expandedIds.has(entries[index]?.cursor ?? String(index)) ? expandedHeight : (entries[index]?.message?.length ?? 0) > 180 ? longCollapsedHeight : collapsedHeight}
        itemKey={index => entries[index]?.cursor ?? `${entries[index]?.realtime ?? ''}-${index}`}
        width="100%"
        onScroll={handleScroll}
        initialScrollOffset={initialScrollOffset}
        itemData={{ entries, expandedIds, toggleExpanded, onSelect, selected }}
      >
        {VirtualLogRow}
      </VariableSizeList>
    </div>
  );
};
