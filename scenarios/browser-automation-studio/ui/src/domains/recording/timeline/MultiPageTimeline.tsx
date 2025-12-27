/**
 * MultiPageTimeline Component
 *
 * Displays a unified timeline of actions and page events with:
 * - Page color coding for visual distinction
 * - Filtering by page
 * - Legend showing all pages with their colors
 * - Real-time updates via WebSocket
 */

import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { MultiPageTimelineEntry } from './MultiPageTimelineEntry';
import type { TimelineEntry, PageColor } from '../hooks/useTimeline';
import type { Page } from '../hooks/usePages';

interface MultiPageTimelineProps {
  /** Timeline entries to display */
  entries: TimelineEntry[];
  /** All pages in the session */
  pages: Page[];
  /** Map of page ID to color class */
  pageColorMap: Map<string, PageColor>;
  /** Currently selected entry ID */
  selectedEntryId?: string;
  /** Callback when an entry is selected */
  onSelectEntry?: (entry: TimelineEntry) => void;
  /** Whether the timeline is loading */
  isLoading?: boolean;
  /** Whether the timeline is live (receiving updates) */
  isLive?: boolean;
  /** Whether to auto-scroll to new entries */
  autoScroll?: boolean;
  /** Custom class name */
  className?: string;
}

/** Page filter option */
type PageFilter = 'all' | string;

export function MultiPageTimeline({
  entries,
  pages,
  pageColorMap,
  selectedEntryId,
  onSelectEntry,
  isLoading = false,
  isLive = false,
  autoScroll = true,
  className = '',
}: MultiPageTimelineProps) {
  const [filterPageId, setFilterPageId] = useState<PageFilter>('all');
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const lastEntryCountRef = useRef(entries.length);

  // Filter entries by selected page
  const filteredEntries = useMemo(() => {
    if (filterPageId === 'all') return entries;
    return entries.filter((e) => e.pageId === filterPageId);
  }, [entries, filterPageId]);

  // Build page map for quick lookup
  const pageMap = useMemo(() => {
    const map = new Map<string, Page>();
    pages.forEach((p) => map.set(p.id, p));
    return map;
  }, [pages]);

  // Auto-scroll to bottom when new entries arrive
  useEffect(() => {
    if (autoScroll && entries.length > lastEntryCountRef.current && scrollContainerRef.current) {
      const container = scrollContainerRef.current;
      // Only scroll if already near bottom (within 100px)
      const isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 100;
      if (isNearBottom) {
        container.scrollTop = container.scrollHeight;
      }
    }
    lastEntryCountRef.current = entries.length;
  }, [entries.length, autoScroll]);

  // Handle entry selection
  const handleSelectEntry = useCallback((entry: TimelineEntry) => {
    if (onSelectEntry) {
      onSelectEntry(entry);
    }
  }, [onSelectEntry]);

  // Get display name for page in filter dropdown
  const getPageDisplayName = useCallback((page: Page) => {
    if (page.title) {
      const maxLen = 25;
      return page.title.length > maxLen ? page.title.slice(0, maxLen - 1) + '…' : page.title;
    }
    try {
      return new URL(page.url).hostname;
    } catch {
      return 'Unknown page';
    }
  }, []);

  const hasMultiplePages = pages.length > 1;
  const openPages = useMemo(() => pages.filter((p) => p.status === 'active'), [pages]);

  return (
    <div className={`flex flex-col h-full ${className}`}>
      {/* Header with filter */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
            Timeline
          </span>
          <span className="text-xs text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">
            {filteredEntries.length}
          </span>
          {isLive && (
            <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
              <span className="w-1.5 h-1.5 bg-green-500 rounded-full animate-pulse" />
              Live
            </span>
          )}
        </div>

        {/* Page filter dropdown */}
        {hasMultiplePages && (
          <select
            value={filterPageId}
            onChange={(e) => setFilterPageId(e.target.value)}
            className="text-xs bg-gray-100 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded px-2 py-1 text-gray-700 dark:text-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="all">All pages</option>
            {pages.map((page) => (
              <option key={page.id} value={page.id}>
                {page.isInitial ? '(main) ' : ''}{getPageDisplayName(page)}
              </option>
            ))}
          </select>
        )}
      </div>

      {/* Legend (when showing all pages and multiple pages exist) */}
      {filterPageId === 'all' && hasMultiplePages && (
        <div className="flex flex-wrap gap-2 px-3 py-2 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
          {openPages.map((page) => (
            <button
              key={page.id}
              onClick={() => setFilterPageId(page.id)}
              className="flex items-center gap-1.5 text-xs hover:bg-gray-100 dark:hover:bg-gray-700 px-1.5 py-0.5 rounded transition-colors"
              title={`Filter to ${page.title || page.url}`}
            >
              <div className={`w-2 h-2 rounded-full ${pageColorMap.get(page.id) || 'bg-gray-400'}`} />
              <span className="text-gray-600 dark:text-gray-400 max-w-[100px] truncate">
                {page.isInitial && (
                  <span className="text-gray-400 dark:text-gray-500 mr-0.5">(main)</span>
                )}
                {getPageDisplayName(page)}
              </span>
            </button>
          ))}
        </div>
      )}

      {/* Timeline entries */}
      <div
        ref={scrollContainerRef}
        className="flex-1 overflow-y-auto px-2 py-2 space-y-1"
      >
        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <svg className="w-5 h-5 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          </div>
        ) : filteredEntries.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-gray-500 dark:text-gray-400">
            <svg className="w-10 h-10 mb-2 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p className="text-sm">
              {filterPageId === 'all'
                ? 'No actions recorded yet'
                : 'No actions on this page'
              }
            </p>
            {isLive && filterPageId === 'all' && (
              <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">
                Perform actions in the browser to record them
              </p>
            )}
          </div>
        ) : (
          filteredEntries.map((entry) => (
            <MultiPageTimelineEntry
              key={entry.id}
              entry={entry}
              page={pageMap.get(entry.pageId)}
              isSelected={entry.id === selectedEntryId}
              onSelect={() => handleSelectEntry(entry)}
              pageColor={pageColorMap.get(entry.pageId)}
              showPageBadge={filterPageId === 'all' && hasMultiplePages}
            />
          ))
        )}
      </div>

      {/* Filter indicator when filtered */}
      {filterPageId !== 'all' && (
        <div className="px-3 py-2 border-t border-gray-200 dark:border-gray-700 bg-blue-50 dark:bg-blue-900/20">
          <div className="flex items-center justify-between text-xs">
            <span className="text-blue-600 dark:text-blue-400">
              Showing {filteredEntries.length} of {entries.length} entries
            </span>
            <button
              onClick={() => setFilterPageId('all')}
              className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium"
            >
              Show all
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
