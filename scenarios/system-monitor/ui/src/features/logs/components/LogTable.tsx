import { VariableSizeList, type ListOnScrollProps } from 'react-window';
import { useCallback, useRef, useState } from 'react';
import type { LogEntry } from '../types';
import { LogRow } from './LogRow';

interface LogTableProps {
  entries: LogEntry[];
  height?: number;
  rowHeight?: number;
  onScrollTopChange?: (atTop: boolean) => void;
}

const TOP_THRESHOLD = 8; // px

export const LogTable = ({
  entries,
  height = 480,
  rowHeight = 22,
  onScrollTopChange,
}: LogTableProps) => {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const listRef = useRef<VariableSizeList>(null);
  const compactViewport = typeof window !== 'undefined' && window.matchMedia('(max-width: 48rem)').matches;
  const collapsedHeight = compactViewport ? Math.max(rowHeight * 3, 72) : Math.max(rowHeight * 2, 42);
  const expandedHeight = compactViewport ? Math.max(rowHeight * 6, 150) : Math.max(rowHeight * 4, 96);
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
        <div className="text-sm text-muted">No log entries match the current filters.</div>
      </div>
    );
  }

  const handleScroll = ({ scrollOffset }: ListOnScrollProps) => {
    if (onScrollTopChange) {
      onScrollTopChange(scrollOffset <= TOP_THRESHOLD);
    }
  };

  return (
    <div className="log-table">
      <VariableSizeList
        ref={listRef}
        height={height}
        itemCount={entries.length}
        itemSize={index => expandedIds.has(entries[index]?.cursor ?? String(index)) ? expandedHeight : collapsedHeight}
        width="100%"
        onScroll={handleScroll}
      >
        {({ index, style }) => {
          const entry = entries[index];
          if (!entry) return null;
          const id = entry.cursor ?? String(index);
          return <LogRow entry={entry} style={style} expanded={expandedIds.has(id)} onToggleExpanded={() => toggleExpanded(id)} />;
        }}
      </VariableSizeList>
    </div>
  );
};
