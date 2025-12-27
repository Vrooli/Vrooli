/**
 * ExecutionTabBar Component
 *
 * Displays browser tabs during multi-page workflow execution playback.
 * Shows each page as a tab with title, favicon placeholder, and status indicators.
 * Adapted from recording TabBar for playback context.
 */

import { useMemo, useCallback } from 'react';
import type { ExecutionPage } from '../../store';

/** Color palette for page indicators (consistent colors per page). */
const PAGE_COLORS = [
  { bg: 'bg-blue-500', text: 'text-blue-500', border: 'border-blue-500' },
  { bg: 'bg-emerald-500', text: 'text-emerald-500', border: 'border-emerald-500' },
  { bg: 'bg-purple-500', text: 'text-purple-500', border: 'border-purple-500' },
  { bg: 'bg-orange-500', text: 'text-orange-500', border: 'border-orange-500' },
  { bg: 'bg-pink-500', text: 'text-pink-500', border: 'border-pink-500' },
  { bg: 'bg-cyan-500', text: 'text-cyan-500', border: 'border-cyan-500' },
  { bg: 'bg-yellow-500', text: 'text-yellow-500', border: 'border-yellow-500' },
  { bg: 'bg-rose-500', text: 'text-rose-500', border: 'border-rose-500' },
];

/** Get consistent color for a page based on its index. */
export function getPageColor(pageIndex: number) {
  return PAGE_COLORS[pageIndex % PAGE_COLORS.length];
}

/** Get page color by ID from a list of pages. */
export function getPageColorById(pageId: string, pages: ExecutionPage[]) {
  const index = pages.findIndex((p) => p.id === pageId);
  return index >= 0 ? getPageColor(index) : PAGE_COLORS[0];
}

interface ExecutionTabBarProps {
  /** List of pages in the execution. */
  pages: ExecutionPage[];
  /** Currently active page ID. */
  activePageId: string | null;
  /** Callback when a tab is clicked (read-only mode may not use this). */
  onTabClick?: (pageId: string) => void;
  /** Callback when a tab close button is clicked. */
  onTabClose?: (pageId: string) => void;
  /** Whether the execution is running. */
  isRunning?: boolean;
  /** Page ID currently being executed (for progress indicator). */
  executingPageId?: string | null;
  /** Whether to show the tab bar (hidden when only one page). */
  show?: boolean;
}

/** Individual tab component for execution playback. */
function ExecutionTab({
  page,
  pageIndex,
  isActive,
  isExecuting,
  onClick,
  onClose,
}: {
  page: ExecutionPage;
  pageIndex: number;
  isActive: boolean;
  isExecuting: boolean;
  onClick?: () => void;
  onClose?: () => void;
}) {
  const color = getPageColor(pageIndex);

  // Can this tab be closed?
  const canClose = onClose && !page.isInitial && !page.closed;

  // Truncate title for display
  const displayTitle = useMemo(() => {
    const title = page.title || 'Untitled';
    const maxLength = 20;
    if (title.length <= maxLength) return title;
    return title.slice(0, maxLength - 1) + '…';
  }, [page.title]);

  // Get domain for favicon placeholder
  const domain = useMemo(() => {
    try {
      const url = new URL(page.url);
      return url.hostname.replace(/^www\./, '');
    } catch {
      return '';
    }
  }, [page.url]);

  // First letter of domain for favicon placeholder
  const faviconLetter = domain.charAt(0).toUpperCase() || 'P';

  // Handle close button click
  const handleClose = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation(); // Don't trigger tab click
      e.preventDefault();
      onClose?.();
    },
    [onClose]
  );

  return (
    <div
      role="tab"
      aria-selected={isActive}
      onClick={onClick}
      className={`
        relative group flex items-center gap-2 px-3 py-2 max-w-[200px]
        rounded-t-lg transition-all duration-150 ease-out
        border-t border-l border-r
        ${
          isActive
            ? 'bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 z-10 -mb-px'
            : 'bg-gray-100 dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-750 opacity-80 hover:opacity-100'
        }
        ${page.closed ? 'opacity-50' : ''}
        ${onClick ? 'cursor-pointer' : 'cursor-default'}
      `}
      title={`${page.title}\n${page.url}${page.closed ? '\n(Closed)' : ''}`}
    >
      {/* Color indicator dot */}
      <div className={`flex-shrink-0 w-2 h-2 rounded-full ${color.bg}`} />

      {/* Favicon placeholder */}
      <div
        className={`
        flex-shrink-0 w-4 h-4 rounded flex items-center justify-center text-[10px] font-semibold
        ${
          isActive
            ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400'
            : 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
        }
      `}
      >
        {faviconLetter}
      </div>

      {/* Tab title */}
      <span
        className={`
        flex-1 text-xs font-medium truncate
        ${isActive ? 'text-gray-900 dark:text-gray-100' : 'text-gray-600 dark:text-gray-400'}
        ${page.closed ? 'line-through' : ''}
      `}
      >
        {displayTitle}
      </span>

      {/* Executing indicator (pulsing dot) - only show when no close button visible */}
      {isExecuting && !page.closed && !canClose && (
        <span className="absolute top-1 right-1 w-2 h-2 bg-flow-accent rounded-full animate-pulse" />
      )}

      {/* Close button - appears on hover for closable tabs */}
      {canClose && (
        <button
          onClick={handleClose}
          className={`
            flex-shrink-0 w-4 h-4 rounded-sm flex items-center justify-center
            text-gray-400 hover:text-gray-600 dark:hover:text-gray-300
            hover:bg-gray-200 dark:hover:bg-gray-600
            opacity-0 group-hover:opacity-100 transition-opacity duration-150
            focus:opacity-100 focus:outline-none focus:ring-1 focus:ring-gray-400
            ${isExecuting ? 'ring-1 ring-flow-accent/50' : ''}
          `}
          title="Close tab"
          aria-label={`Close ${page.title || 'tab'}`}
        >
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}

      {/* Initial page badge */}
      {page.isInitial && (
        <span className="flex-shrink-0 text-[9px] px-1 py-0.5 rounded bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400">
          main
        </span>
      )}

      {/* Closed badge */}
      {page.closed && (
        <span className="flex-shrink-0 text-[9px] px-1 py-0.5 rounded bg-red-200 dark:bg-red-900/40 text-red-600 dark:text-red-400">
          closed
        </span>
      )}

      {/* Active indicator line */}
      {isActive && <span className={`absolute bottom-0 left-0 right-0 h-0.5 ${color.bg}`} />}
    </div>
  );
}

export function ExecutionTabBar({
  pages,
  activePageId,
  onTabClick,
  onTabClose,
  isRunning = false,
  executingPageId = null,
  show = true,
}: ExecutionTabBarProps) {
  const handleTabClick = useCallback(
    (pageId: string) => {
      if (onTabClick && pageId !== activePageId) {
        onTabClick(pageId);
      }
    },
    [activePageId, onTabClick]
  );

  const handleTabClose = useCallback(
    (pageId: string) => {
      onTabClose?.(pageId);
    },
    [onTabClose]
  );

  // Don't render if only one page or told to hide
  if (!show || pages.length <= 1) {
    return null;
  }

  // Count open pages
  const openPageCount = pages.filter((p) => !p.closed).length;

  return (
    <div className="flex items-end px-2 pt-2 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
      {/* Tab list */}
      <div className="flex items-end gap-0.5" role="tablist" aria-label="Execution pages">
        {pages.map((page, index) => (
          <ExecutionTab
            key={page.id}
            page={page}
            pageIndex={index}
            isActive={page.id === activePageId}
            isExecuting={isRunning && page.id === executingPageId}
            onClick={onTabClick ? () => handleTabClick(page.id) : undefined}
            onClose={onTabClose ? () => handleTabClose(page.id) : undefined}
          />
        ))}
      </div>

      {/* Running indicator */}
      {isRunning && (
        <div className="ml-2 flex items-center gap-1 px-2 py-1 text-xs text-gray-500 dark:text-gray-400">
          <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          <span>Running...</span>
        </div>
      )}

      {/* Tab count badge */}
      <div className="ml-auto flex-shrink-0 px-2 py-1.5 text-[10px] font-medium text-gray-500 dark:text-gray-400">
        {openPageCount} / {pages.length} tabs
      </div>
    </div>
  );
}

export default ExecutionTabBar;
