/**
 * TabBar Component
 *
 * Displays browser tabs for multi-tab recording sessions.
 * Shows each open page as a tab with title, favicon placeholder,
 * and activity indicators.
 */

import { useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react';
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
}

/** Individual tab component */
function Tab({
  page,
  isActive,
  hasActivity,
  tabIndex,
  onClick,
  onClose,
}: {
  page: Page;
  isActive: boolean;
  hasActivity: boolean;
  tabIndex: number;
  onClick: () => void;
  onClose?: () => void;
}) {
  // Can this tab be closed? (all tabs are closable now)
  const canClose = !!onClose;

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

  const faviconUrl = page.faviconUrl;
  const [failedFavicon, setFailedFavicon] = useState<string | null>(null);
  const showFavicon = faviconUrl && faviconUrl !== failedFavicon;

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
    <div className="relative group flex shrink-0 max-w-[200px]" data-page-id={page.id}>
      <button
        type="button"
        role="tab"
        aria-selected={isActive}
        aria-label={page.title || 'Untitled'}
        aria-keyshortcuts={canClose ? 'Delete' : undefined}
        tabIndex={tabIndex}
        onClick={onClick}
        className={`
          relative flex items-center gap-2 px-3 py-2 min-w-0 text-left
          rounded-t-lg transition-all duration-150 ease-out
          border-t border-l border-r cursor-pointer
          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500
          ${canClose ? 'pr-8' : ''}
          ${isActive
            ? 'bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 z-10 -mb-px'
            : 'bg-gray-100 dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-750 opacity-80 hover:opacity-100'
          }
        `}
        title={`${page.title}\n${page.url}`}
      >
        {/* Favicon - show image if available, otherwise show letter */}
        {showFavicon ? (
          <img
            src={faviconUrl}
            alt=""
            className="flex-shrink-0 w-4 h-4 rounded object-contain"
            onError={() => setFailedFavicon(faviconUrl ?? null)}
          />
        ) : (
          <span aria-hidden="true" className={`
            flex-shrink-0 w-4 h-4 rounded flex items-center justify-center text-[10px] font-semibold
            ${isActive
              ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400'
              : 'bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
            }
          `}>
            {faviconLetter}
          </span>
        )}

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

        {/* Active indicator line */}
        {isActive && (
          <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500" />
        )}
      </button>

      {/* Close button - appears on hover for closable tabs */}
      {canClose && (
        <button
          type="button"
          tabIndex={-1}
          onClick={handleClose}
          className={`
            absolute right-2 top-1/2 -translate-y-1/2 z-20 w-4 h-4 rounded-sm flex items-center justify-center
            text-gray-400 hover:text-gray-600 dark:hover:text-gray-300
            hover:bg-gray-200 dark:hover:bg-gray-600
            opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity duration-150
            focus:opacity-100 focus:outline-none focus:ring-1 focus:ring-gray-400
          `}
          title="Close tab (Delete)"
          aria-label={`Close ${page.title || 'tab'}`}
        >
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}

/** Placeholder tab shown when no pages are open */
function NewTabPlaceholder({ onClick }: { onClick?: () => void }) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={true}
      onClick={onClick}
      disabled={!onClick}
      className={`
        relative group flex items-center gap-2 px-3 py-2 max-w-[200px]
        rounded-t-lg transition-all duration-150 ease-out
        border-t border-l border-r cursor-pointer
        focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500
        disabled:cursor-default
        bg-white dark:bg-gray-900 border-gray-200 dark:border-gray-700 z-10 -mb-px
      `}
      title="New Tab"
    >
      {/* Plus icon */}
      <span aria-hidden="true" className="flex-shrink-0 w-4 h-4 rounded flex items-center justify-center bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400">
        <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
      </span>

      {/* Tab title */}
      <span className="flex-1 text-xs font-medium truncate text-gray-900 dark:text-gray-100">
        New Tab
      </span>

      {/* Active indicator line */}
      <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-500" />
    </button>
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
}: TabBarProps) {
  const tabListRef = useRef<HTMLDivElement>(null);
  const lastFocusedTab = useRef<{id: string | null; index: number} | null>(null);
  const [focusedPageId, setFocusedPageId] = useState<string | null>(null);
  const keyboardPageId = pages.find(page => page.id === focusedPageId)?.id
    ?? pages.find(page => page.id === activePageId)?.id ?? pages[0]?.id;

  useLayoutEffect(() => {
    const previous = lastFocusedTab.current;
    if (!previous || (previous.id === null && pages.length === 0)) return;
    const index = pages.findIndex(page => page.id === previous.id);
    if (index >= 0) { previous.index = index; return; }
    const tabs = tabListRef.current?.querySelectorAll<HTMLButtonElement>('[role="tab"]');
    tabs?.[Math.min(previous.index, tabs.length - 1)]?.focus();
  }, [pages]);

  const handleTabClick = useCallback((pageId: string) => {
    if (pageId !== activePageId) {
      onTabClick(pageId);
    }
  }, [activePageId, onTabClick]);

  // Always show tabs - either real pages or a "New Tab" placeholder
  const hasPages = pages.length > 0;

  return (
    <div className="flex items-center px-2 py-1.5 bg-gray-50 dark:bg-gray-800/50 border-b border-gray-200 dark:border-gray-700 overflow-x-auto min-h-[42px]">
      {/* Tab list */}
      <div ref={tabListRef} className="flex items-end gap-0.5" role="tablist" aria-label="Browser tabs"
        onFocusCapture={event => {
          const id = (event.target as HTMLElement).closest<HTMLElement>('[data-page-id]')?.dataset.pageId;
          lastFocusedTab.current = {id: id ?? null, index: id ? pages.findIndex(page => page.id === id) : 0};
          setFocusedPageId(id ?? null);
        }}
        onBlurCapture={event => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null) && event.target.isConnected) {
            lastFocusedTab.current = null;
            setFocusedPageId(null);
          }
        }}
        onKeyDown={event => {
          const target = event.target as HTMLElement;
          if (target.getAttribute('role') !== 'tab' || event.altKey || event.ctrlKey || event.metaKey) return;
          const tabs = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]'));
          const index = tabs.indexOf(target as HTMLButtonElement);
          const destinations: Record<string, number> = {
            ArrowLeft: (index + tabs.length - 1) % tabs.length,
            ArrowRight: (index + 1) % tabs.length,
            Home: 0, End: tabs.length - 1,
          };
          const destination = destinations[event.key];
          if (destination !== undefined) {
            event.preventDefault();
            tabs[destination]?.focus();
          } else if (event.key === 'Delete' && onTabClose && pages[index]) {
            event.preventDefault();
            onTabClose(pages[index].id);
          }
        }}>
        {hasPages ? (
          // Show actual page tabs
          pages.map((page) => (
            <Tab
              key={page.id}
              page={page}
              isActive={page.id === activePageId}
              hasActivity={page.id === recentActivityPageId}
              tabIndex={page.id === keyboardPageId ? 0 : -1}
              onClick={() => handleTabClick(page.id)}
              onClose={onTabClose ? () => onTabClose(page.id) : undefined}
            />
          ))
        ) : (
          // Show "New Tab" placeholder when no pages
          <NewTabPlaceholder onClick={onCreateTab} />
        )}
      </div>

      {/* New tab button - only show when there are actual pages */}
      {onCreateTab && hasPages && (
        <button
          type="button"
          onClick={onCreateTab}
          className="flex-shrink-0 ml-1 p-1.5 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
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
