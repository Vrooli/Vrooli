/**
 * TabBar Component
 *
 * Displays browser tabs for multi-tab recording sessions.
 * Shows each open page as a tab with title, favicon placeholder,
 * and activity indicators.
 */

import { useCallback, useMemo } from 'react';
import type { Page } from '../hooks/usePages';

interface TabBarProps {
  /** List of open pages */
  pages: Page[];
  /** Currently active page ID */
  activePageId: string | null;
  /** Callback when a tab is clicked */
  onTabClick: (pageId: string) => void;
  /** Callback when a tab close button is clicked */
  onTabClose?: (pageId: string) => void;
  /** Callback when the new tab button is clicked */
  onCreateTab?: () => void;
  /** Whether the tab bar is loading */
  isLoading?: boolean;
  /** Page ID with recent activity (for activity indicator) */
  recentActivityPageId?: string | null;
  /** Whether to show the tab bar (hidden when only one page) */
  show?: boolean;
}

/** Individual tab component */
function Tab({
  page,
  isActive,
  hasActivity,
  onClick,
  onClose,
}: {
  page: Page;
  isActive: boolean;
  hasActivity: boolean;
  onClick: () => void;
  onClose?: () => void;
}) {
  // Can this tab be closed?
  const canClose = onClose && !page.isInitial;

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
        border-t border-l border-r cursor-pointer
        ${isActive
          ? 'bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 z-10 -mb-px'
          : 'bg-gray-100 dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-750 opacity-80 hover:opacity-100'
        }
      `}
      title={`${page.title}\n${page.url}`}
    >
      {/* Favicon placeholder */}
      <div className={`
        flex-shrink-0 w-4 h-4 rounded flex items-center justify-center text-[10px] font-semibold
        ${isActive
          ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400'
          : 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
        }
      `}>
        {faviconLetter}
      </div>

      {/* Tab title */}
      <span className={`
        flex-1 text-xs font-medium truncate
        ${isActive
          ? 'text-gray-900 dark:text-gray-100'
          : 'text-gray-600 dark:text-gray-400'
        }
      `}>
        {displayTitle}
      </span>

      {/* Activity indicator - only show when no close button visible */}
      {hasActivity && !isActive && !canClose && (
        <span className="absolute top-1 right-1 w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
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

      {/* Active indicator line */}
      {isActive && (
        <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500" />
      )}
    </div>
  );
}

export function TabBar({
  pages,
  activePageId,
  onTabClick,
  onTabClose,
  onCreateTab,
  isLoading = false,
  recentActivityPageId = null,
  show = true,
}: TabBarProps) {
  const handleTabClick = useCallback((pageId: string) => {
    if (pageId !== activePageId) {
      onTabClick(pageId);
    }
  }, [activePageId, onTabClick]);

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

  return (
    <div className="flex items-end px-2 pt-2 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
      {/* Tab list */}
      <div className="flex items-end gap-0.5" role="tablist" aria-label="Browser tabs">
        {pages.map((page) => (
          <Tab
            key={page.id}
            page={page}
            isActive={page.id === activePageId}
            hasActivity={page.id === recentActivityPageId}
            onClick={() => handleTabClick(page.id)}
            onClose={onTabClose ? () => handleTabClose(page.id) : undefined}
          />
        ))}
      </div>

      {/* New tab button */}
      {onCreateTab && (
        <button
          onClick={onCreateTab}
          className="flex-shrink-0 ml-1 mb-0.5 p-1.5 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
          title="Open new tab"
          aria-label="Open new tab"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        </button>
      )}

      {/* Loading indicator */}
      {isLoading && (
        <div className="ml-2 flex items-center gap-1 px-2 py-1 text-xs text-gray-500 dark:text-gray-400">
          <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          <span>Loading...</span>
        </div>
      )}
    </div>
  );
}
