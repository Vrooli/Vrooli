/**
 * MultiPageTimelineEntry Component
 *
 * Displays a single timeline entry (action or page event) with page color coding.
 * Used in the multi-page recording timeline.
 */

import { useMemo } from 'react';
import type { TimelineEntry, TimelineAction, PageEventType, PageColor } from '../hooks/useTimeline';
import type { Page } from '../hooks/usePages';

interface MultiPageTimelineEntryProps {
  /** The timeline entry to display */
  entry: TimelineEntry;
  /** Page this entry belongs to */
  page?: Page;
  /** Whether this entry is selected */
  isSelected?: boolean;
  /** Callback when entry is clicked */
  onSelect?: () => void;
  /** Color class for this entry's page */
  pageColor?: PageColor;
  /** Whether to show the page badge */
  showPageBadge?: boolean;
}

/** Format timestamp for display */
function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

/** Truncate text to max length */
function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '…';
}

/** Get icon for action type */
function ActionIcon({ actionType }: { actionType: string }) {
  switch (actionType) {
    case 'click':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
        </svg>
      );
    case 'input':
    case 'type':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
        </svg>
      );
    case 'navigate':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
        </svg>
      );
    case 'scroll':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
        </svg>
      );
    case 'keyboard':
    case 'keypress':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707" />
        </svg>
      );
    default:
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z" />
        </svg>
      );
  }
}

/** Get icon for page event type */
function PageEventIcon({ eventType }: { eventType: PageEventType }) {
  switch (eventType) {
    case 'page_created':
      return (
        <svg className="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
        </svg>
      );
    case 'page_closed':
      return (
        <svg className="w-4 h-4 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      );
    case 'page_navigated':
      return (
        <svg className="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
        </svg>
      );
    default:
      return null;
  }
}

/** Get description for action */
function getActionDescription(action: TimelineAction): string {
  switch (action.actionType) {
    case 'click':
      if (action.payload?.text) {
        return `Click: "${truncate(String(action.payload.text), 30)}"`;
      }
      return `Click ${action.selector?.primary ? truncate(action.selector.primary, 30) : 'element'}`;
    case 'input':
    case 'type':
      if (action.payload?.text) {
        return `Type: "${truncate(String(action.payload.text), 30)}"`;
      }
      return 'Type text';
    case 'navigate':
      return `Navigate to ${truncate(action.url || '', 40)}`;
    case 'scroll':
      const deltaY = action.payload?.deltaY as number | undefined;
      return `Scroll ${deltaY !== undefined && deltaY > 0 ? 'down' : 'up'}`;
    case 'keyboard':
    case 'keypress':
      return `Press ${action.payload?.key || 'key'}`;
    default:
      return action.actionType;
  }
}

/** Get description for page event */
function getPageEventDescription(eventType: PageEventType, url?: string): string {
  switch (eventType) {
    case 'page_created':
      return 'New tab opened';
    case 'page_closed':
      return 'Tab closed';
    case 'page_navigated':
      if (url) {
        try {
          const pathname = new URL(url).pathname;
          return `Navigated to ${truncate(pathname, 30)}`;
        } catch {
          return `Navigated to ${truncate(url, 30)}`;
        }
      }
      return 'Page navigated';
    default:
      return eventType;
  }
}

export function MultiPageTimelineEntry({
  entry,
  page,
  isSelected,
  onSelect,
  pageColor = 'bg-gray-500',
  showPageBadge = true,
}: MultiPageTimelineEntryProps) {
  const isPageEvent = entry.type === 'page_event';
  const pageEvent = entry.pageEvent;
  const action = entry.action;

  // Get display text
  const description = useMemo(() => {
    if (isPageEvent && pageEvent) {
      return getPageEventDescription(pageEvent.type, pageEvent.url);
    }
    if (action) {
      return getActionDescription(action);
    }
    return 'Unknown entry';
  }, [isPageEvent, pageEvent, action]);

  // Get page display name
  const pageDisplayName = useMemo(() => {
    if (!page) return '';
    if (page.title) return truncate(page.title, 15);
    try {
      return new URL(page.url).hostname;
    } catch {
      return 'Unknown page';
    }
  }, [page]);

  if (isPageEvent && pageEvent) {
    // Page event entry - styled differently
    return (
      <div className="flex items-center gap-3 py-2 px-3 rounded-md bg-gray-50 dark:bg-gray-800/30 border border-dashed border-gray-200 dark:border-gray-700">
        {/* Page color indicator */}
        <div className={`w-2 h-2 rounded-full flex-shrink-0 ${pageColor}`} />

        {/* Icon */}
        <PageEventIcon eventType={pageEvent.type} />

        {/* Description */}
        <span className="flex-1 text-sm text-gray-600 dark:text-gray-400">
          {description}
          {pageEvent.url && pageEvent.type !== 'page_closed' && (
            <span className="ml-1 font-mono text-xs text-gray-400 dark:text-gray-500">
              {truncate(pageEvent.url, 40)}
            </span>
          )}
        </span>

        {/* Timestamp */}
        <span className="text-xs text-gray-400 dark:text-gray-500 flex-shrink-0">
          {formatTimestamp(entry.timestamp)}
        </span>
      </div>
    );
  }

  // Action entry
  return (
    <button
      onClick={onSelect}
      className={`
        flex items-center gap-3 py-2 px-3 rounded-md w-full text-left
        transition-colors duration-150
        ${isSelected
          ? 'bg-blue-50 dark:bg-blue-900/30 ring-1 ring-blue-400 dark:ring-blue-500'
          : 'hover:bg-gray-50 dark:hover:bg-gray-800/50'
        }
      `}
    >
      {/* Page color indicator */}
      <div className={`w-2 h-2 rounded-full flex-shrink-0 ${pageColor}`} />

      {/* Action icon */}
      <span className="text-gray-500 dark:text-gray-400 flex-shrink-0">
        {action && <ActionIcon actionType={action.actionType} />}
      </span>

      {/* Description */}
      <div className="flex-1 min-w-0">
        <div className="text-sm text-gray-900 dark:text-gray-100 truncate">
          {description}
        </div>

        {/* Page badge (when showing all pages) */}
        {showPageBadge && page && (
          <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
            {pageDisplayName}
          </div>
        )}
      </div>

      {/* Timestamp */}
      <span className="text-xs text-gray-400 dark:text-gray-500 flex-shrink-0">
        {formatTimestamp(entry.timestamp)}
      </span>
    </button>
  );
}
