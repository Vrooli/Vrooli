/**
 * usePages Hook
 *
 * Manages page state for multi-tab recording sessions.
 * Handles fetching pages, listening to WebSocket events,
 * and switching between pages.
 */

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { getApiBase } from '@/config';
import { useSessionStore } from '../stores';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

/** Page status in the recording session */
export type PageStatus = 'active' | 'closed';

/** Page event types from the server */
export type PageEventType = 'page_created' | 'page_navigated' | 'page_closed';

/** Page representing a browser tab within a recording session */
export interface Page {
  id: string;
  sessionId: string;
  url: string;
  title: string;
  openerId?: string;
  isInitial: boolean;
  status: PageStatus;
  createdAt: string;
  closedAt?: string;
}

/** Page lifecycle event */
export interface PageEvent {
  id: string;
  type: PageEventType;
  pageId: string;
  url?: string;
  title?: string;
  openerId?: string;
  timestamp: string;
}

/** API response for listing pages */
interface PagesResponse {
  pages: Page[];
  activePageId: string;
}

/** WebSocket message for page events */
interface PageEventMessage {
  type: 'page_event';
  session_id: string;
  event: PageEvent;
  timestamp: string;
}

/** WebSocket message for page switch */
interface PageSwitchMessage {
  type: 'page_switch';
  session_id: string;
  active_page_id: string;
  timestamp: string;
}

const pageEventTypes = new Set<PageEventType>(['page_created', 'page_navigated', 'page_closed']);

const parsePage = (value: unknown): Page | null => {
  if (!isRecord(value)) return null;
  const id = typeof value.id === 'string' ? value.id : null;
  const sessionId =
    typeof value.sessionId === 'string'
      ? value.sessionId
      : typeof value.session_id === 'string'
        ? value.session_id
        : null;
  if (!id || !sessionId) return null;
  const url = typeof value.url === 'string' ? value.url : '';
  const title = typeof value.title === 'string' ? value.title : '';
  const openerId =
    typeof value.openerId === 'string'
      ? value.openerId
      : typeof value.opener_id === 'string'
        ? value.opener_id
        : undefined;
  const isInitial =
    typeof value.isInitial === 'boolean'
      ? value.isInitial
      : typeof value.is_initial === 'boolean'
        ? value.is_initial
        : false;
  const status: PageStatus = value.status === 'closed' ? 'closed' : 'active';
  const createdAt =
    typeof value.createdAt === 'string'
      ? value.createdAt
      : typeof value.created_at === 'string'
        ? value.created_at
        : new Date().toISOString();
  const closedAt =
    typeof value.closedAt === 'string'
      ? value.closedAt
      : typeof value.closed_at === 'string'
        ? value.closed_at
        : undefined;

  return {
    id,
    sessionId,
    url,
    title,
    openerId,
    isInitial,
    status,
    createdAt,
    closedAt,
  };
};

const parsePagesResponse = (value: unknown): PagesResponse | null => {
  if (!isRecord(value)) return null;
  const pages = Array.isArray(value.pages)
    ? value.pages.map(parsePage).filter((page): page is Page => page !== null)
    : [];
  const activePageId =
    typeof value.activePageId === 'string'
      ? value.activePageId
      : typeof value.active_page_id === 'string'
        ? value.active_page_id
        : '';
  return { pages, activePageId };
};

const parsePageEvent = (value: unknown): PageEvent | null => {
  if (!isRecord(value)) return null;
  if (typeof value.type !== 'string' || !pageEventTypes.has(value.type as PageEventType)) return null;
  const pageId =
    typeof value.pageId === 'string'
      ? value.pageId
      : typeof value.page_id === 'string'
        ? value.page_id
        : null;
  if (!pageId) return null;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();
  const id = typeof value.id === 'string' ? value.id : `${pageId}-${timestamp}`;

  return {
    id,
    type: value.type as PageEventType,
    pageId,
    url: typeof value.url === 'string' ? value.url : undefined,
    title: typeof value.title === 'string' ? value.title : undefined,
    openerId:
      typeof value.openerId === 'string'
        ? value.openerId
        : typeof value.opener_id === 'string'
          ? value.opener_id
          : undefined,
    timestamp,
  };
};

const parsePageEventMessage = (value: unknown): PageEventMessage | null => {
  if (!isRecord(value) || value.type !== 'page_event') return null;
  if (typeof value.session_id !== 'string') return null;
  const event = parsePageEvent(value.event);
  if (!event) return null;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();
  return {
    type: 'page_event',
    session_id: value.session_id,
    event,
    timestamp,
  };
};

const parsePageSwitchMessage = (value: unknown): PageSwitchMessage | null => {
  if (!isRecord(value) || value.type !== 'page_switch') return null;
  if (typeof value.session_id !== 'string' || typeof value.active_page_id !== 'string') return null;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();
  return {
    type: 'page_switch',
    session_id: value.session_id,
    active_page_id: value.active_page_id,
    timestamp,
  };
};

const parseErrorMessage = (value: unknown): string | null => {
  if (!isRecord(value)) return null;
  if (typeof value.error === 'string') return value.error;
  if (typeof value.message === 'string') return value.message;
  return null;
};

interface UsePagesOptions {
  /** Session ID to track pages for */
  sessionId: string | null;
  /** Callback when a new page is created */
  onPageCreated?: (page: Page) => void;
  /** Callback when a page is closed */
  onPageClosed?: (pageId: string) => void;
  /** Callback when the active page changes */
  onActivePageChanged?: (pageId: string) => void;
  /** Callback when a page navigates (URL/title changed). isActive indicates if this is the current active page. */
  onPageNavigated?: (pageId: string, url: string, title: string, isActive: boolean) => void;
}

interface UsePagesReturn {
  /** Map of all pages by ID */
  pages: Map<string, Page>;
  /** Array of all pages (sorted by creation time) */
  pageList: Page[];
  /** Array of only open pages */
  openPages: Page[];
  /** Current active page ID */
  activePageId: string | null;
  /** Current active page object */
  activePage: Page | null;
  /** Whether pages are being loaded */
  isLoading: boolean;
  /** Error message if any */
  error: string | null;
  /** Switch to a different page */
  switchToPage: (pageId: string) => Promise<void>;
  /** Close a page (user-initiated) */
  closePage: (pageId: string) => Promise<void>;
  /** Create a new page (user-initiated) */
  createPage: (url?: string) => Promise<void>;
  /** Refresh pages from the server */
  refreshPages: () => Promise<void>;
  /** Number of open pages */
  openPageCount: number;
  /** Whether multiple pages are open */
  hasMultiplePages: boolean;
}

export function usePages({
  sessionId: propSessionId,
  onPageCreated,
  onPageClosed,
  onActivePageChanged,
  onPageNavigated,
}: UsePagesOptions): UsePagesReturn {
  const [pages, setPages] = useState<Map<string, Page>>(new Map());
  const [activePageId, setActivePageId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const { lastMessage } = useWebSocket();
  const apiUrl = getApiBase();

  // Read session validation state from store
  const storeSessionId = useSessionStore((s) => s.sessionId);
  const isValidated = useSessionStore((s) => s.isValidated);

  // Use store session ID if validated, otherwise fall back to prop
  const sessionId = isValidated ? storeSessionId : propSessionId;

  // Get store actions for syncing page state
  const storeSetPages = useSessionStore((s) => s.setPages);
  const storeAddPage = useSessionStore((s) => s.addPage);
  const storeUpdatePage = useSessionStore((s) => s.updatePage);
  const storeSetActivePageId = useSessionStore((s) => s.setActivePageId);
  const storeUpdatePageColorMap = useSessionStore((s) => s.updatePageColorMap);

  // Keep callbacks in refs to avoid stale closures
  const onPageCreatedRef = useRef(onPageCreated);
  const onPageClosedRef = useRef(onPageClosed);
  const onActivePageChangedRef = useRef(onActivePageChanged);
  const onPageNavigatedRef = useRef(onPageNavigated);
  onPageCreatedRef.current = onPageCreated;
  onPageClosedRef.current = onPageClosed;
  onActivePageChangedRef.current = onActivePageChanged;
  onPageNavigatedRef.current = onPageNavigated;

  // Initial fetch when session changes
  const refreshPages = useCallback(async () => {
    if (!sessionId) {
      setPages(new Map());
      setActivePageId(null);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiUrl}/recordings/live/${sessionId}/pages`);
      if (!response.ok) {
        throw new Error(`Failed to fetch pages: ${response.statusText}`);
      }

      const payload: unknown = await response.json();
      const data = parsePagesResponse(payload);
      if (!data) {
        throw new Error('Invalid pages response');
      }

      const pageMap = new Map<string, Page>();
      data.pages.forEach((page) => {
        pageMap.set(page.id, page);
      });

      setPages(pageMap);
      setActivePageId(data.activePageId || null);

      // Sync to store
      storeSetPages(pageMap);
      storeSetActivePageId(data.activePageId || null);
      storeUpdatePageColorMap(data.pages);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch pages';
      setError(message);
      console.error('[usePages] Error fetching pages:', err);
    } finally {
      setIsLoading(false);
    }
  }, [apiUrl, sessionId, storeSetPages, storeSetActivePageId, storeUpdatePageColorMap]);

  // Fetch pages when session changes and is validated
  useEffect(() => {
    if (sessionId && isValidated) {
      void refreshPages();
    } else {
      setPages(new Map());
      setActivePageId(null);
    }
  }, [sessionId, isValidated, refreshPages]);

  // Keep track of active page in a ref for use in callbacks without stale closures
  const activePageIdRef = useRef(activePageId);
  activePageIdRef.current = activePageId;

  // Handle WebSocket messages for page events
  useEffect(() => {
    if (!lastMessage || !sessionId) return;

    const eventMessage = parsePageEventMessage(lastMessage);
    if (eventMessage && eventMessage.session_id === sessionId) {
      const event = eventMessage.event;

      setPages((prev) => {
        const next = new Map(prev);

        switch (event.type) {
          case 'page_created': {
            const newPage: Page = {
              id: event.pageId,
              sessionId,
              url: event.url || '',
              title: event.title || '',
              openerId: event.openerId,
              isInitial: false,
              status: 'active',
              createdAt: event.timestamp,
            };
            next.set(event.pageId, newPage);

            // Sync to store
            storeAddPage(newPage);

            // Notify callback
            if (onPageCreatedRef.current) {
              onPageCreatedRef.current(newPage);
            }
            break;
          }

          case 'page_navigated': {
            const existing = next.get(event.pageId);
            if (existing) {
              const newUrl = event.url || existing.url;
              const newTitle = event.title || existing.title;
              next.set(event.pageId, {
                ...existing,
                url: newUrl,
                title: newTitle,
              });

              // Sync to store
              storeUpdatePage(event.pageId, { url: newUrl, title: newTitle });

              // Notify callback with isActive flag
              if (onPageNavigatedRef.current) {
                const isActive = event.pageId === activePageIdRef.current;
                onPageNavigatedRef.current(event.pageId, newUrl, newTitle, isActive);
              }
            }
            break;
          }

          case 'page_closed': {
            const closing = next.get(event.pageId);
            if (closing) {
              next.set(event.pageId, {
                ...closing,
                status: 'closed',
                closedAt: event.timestamp,
              });

              // Sync to store
              storeUpdatePage(event.pageId, { status: 'closed', closedAt: event.timestamp });

              // Notify callback
              if (onPageClosedRef.current) {
                onPageClosedRef.current(event.pageId);
              }
            }
            break;
          }
        }

        return next;
      });
    }

    // Handle page switch events
    const switchMessage = parsePageSwitchMessage(lastMessage);
    if (switchMessage && switchMessage.session_id === sessionId) {
      const newActivePageId = switchMessage.active_page_id;
      setActivePageId(newActivePageId);

      // Sync to store
      storeSetActivePageId(newActivePageId);

      // Notify callback
      if (onActivePageChangedRef.current) {
        onActivePageChangedRef.current(newActivePageId);
      }
    }
  }, [lastMessage, sessionId, storeAddPage, storeUpdatePage, storeSetActivePageId]);

  // Switch active page
  const switchToPage = useCallback(async (pageId: string) => {
    if (!sessionId) {
      console.warn('[usePages] Cannot switch page: no session');
      return;
    }

    try {
      const response = await fetch(
        `${apiUrl}/recordings/live/${sessionId}/pages/${pageId}/activate`,
        { method: 'POST' }
      );

      if (!response.ok) {
        const payload: unknown = await response.json().catch(() => null);
        const message = parseErrorMessage(payload) ?? `Failed to switch page: ${response.statusText}`;
        throw new Error(message);
      }

      // Optimistically update state (will also receive WebSocket confirmation)
      setActivePageId(pageId);

      // Notify callback
      if (onActivePageChangedRef.current) {
        onActivePageChangedRef.current(pageId);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to switch page';
      setError(message);
      console.error('[usePages] Error switching page:', err);
    }
  }, [apiUrl, sessionId]);

  // Close a page (user-initiated)
  const closePage = useCallback(async (pageId: string) => {
    if (!sessionId) {
      console.warn('[usePages] Cannot close page: no session');
      return;
    }

    try {
      const response = await fetch(
        `${apiUrl}/recordings/live/${sessionId}/pages/${pageId}/close`,
        { method: 'POST' }
      );

      if (!response.ok) {
        const payload: unknown = await response.json().catch(() => null);
        const message = parseErrorMessage(payload) ?? `Failed to close page: ${response.statusText}`;
        throw new Error(message);
      }

      const resultPayload: unknown = await response.json();
      const activePageIdResult =
        isRecord(resultPayload) && typeof resultPayload.activePageId === 'string'
          ? resultPayload.activePageId
          : isRecord(resultPayload) && typeof resultPayload.active_page_id === 'string'
            ? resultPayload.active_page_id
            : null;

      // Optimistically update state (will also receive WebSocket confirmation)
      setPages((prev) => {
        const next = new Map(prev);
        const closing = next.get(pageId);
        if (closing) {
          next.set(pageId, {
            ...closing,
            status: 'closed',
            closedAt: new Date().toISOString(),
          });
        }
        return next;
      });

      // Update active page if it changed
      if (activePageIdResult) {
        setActivePageId(activePageIdResult);
        if (onActivePageChangedRef.current) {
          onActivePageChangedRef.current(activePageIdResult);
        }
      }

      // Notify callback
      if (onPageClosedRef.current) {
        onPageClosedRef.current(pageId);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to close page';
      setError(message);
      console.error('[usePages] Error closing page:', err);
    }
  }, [apiUrl, sessionId]);

  // Create a new page (user-initiated)
  const createPage = useCallback(async (url?: string) => {
    if (!sessionId) {
      console.warn('[usePages] Cannot create page: no session');
      return;
    }

    try {
      const response = await fetch(
        `${apiUrl}/recordings/live/${sessionId}/pages`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: url || 'about:blank' }),
        }
      );

      if (!response.ok) {
        const payload: unknown = await response.json().catch(() => null);
        const message = parseErrorMessage(payload) ?? `Failed to create page: ${response.statusText}`;
        throw new Error(message);
      }

      // The page will be added via WebSocket event, no need to update state here
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create page';
      setError(message);
      console.error('[usePages] Error creating page:', err);
    }
  }, [apiUrl, sessionId]);

  // Derived values
  const pageList = useMemo(() => {
    const list = Array.from(pages.values());
    // Sort by creation time
    list.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
    return list;
  }, [pages]);

  const openPages = useMemo(
    () => pageList.filter((p) => p.status === 'active'),
    [pageList]
  );

  const activePage = useMemo(
    () => (activePageId ? pages.get(activePageId) ?? null : null),
    [pages, activePageId]
  );

  const openPageCount = openPages.length;
  const hasMultiplePages = openPageCount > 1;

  return {
    pages,
    pageList,
    openPages,
    activePageId,
    activePage,
    isLoading,
    error,
    switchToPage,
    closePage,
    createPage,
    refreshPages,
    openPageCount,
    hasMultiplePages,
  };
}
