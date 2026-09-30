/**
 * usePages Hook
 *
 * Manages page state for multi-tab recording sessions.
 * Handles fetching pages, listening to WebSocket events,
 * and switching between pages.
 */

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useWebSocketMessage } from '@/contexts/WebSocketContext';
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
  faviconUrl?: string;
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
  faviconUrl?: string;
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
    faviconUrl: typeof value.faviconUrl === 'string' ? value.faviconUrl : undefined,
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
    faviconUrl: typeof value.faviconUrl === 'string' ? value.faviconUrl : undefined,
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
}: UsePagesOptions): UsePagesReturn {
  const pages = useSessionStore(s => s.pages);
  const activePageId = useSessionStore(s => s.activePageId);
  const storeSessionId = useSessionStore(s => s.sessionId);
  const isValidated = useSessionStore(s => s.isValidated);
  const sessionId = isValidated ? storeSessionId : propSessionId;
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const apiUrl = getApiBase();

  const createdCallback = useRef(onPageCreated);
  createdCallback.current = onPageCreated;
  const owner = useRef({sessionId});
  if (owner.current.sessionId !== sessionId) owner.current = {sessionId};

  // A creation receipt and its callback describe one page. Preserve terminal
  // closure if a delayed creation observation arrives after the user closed it.
  const observeCreatedPage = useCallback((page: Page, selected?: string) => {
    const state = useSessionStore.getState();
    const existing = state.pages.get(page.id);
    if (existing?.status === 'closed' || (existing && selected === undefined)) return;
    const next = new Map(state.pages);
    next.set(page.id, page);
    useSessionStore.setState({pages: next, activePageId: selected ?? state.activePageId});
    state.updatePageColorMap([...next.values()]);
    if (!existing) createdCallback.current?.(page);
  }, []);

  // Every async page operation retains its original UI session admission.
  const request = useCallback(async (
    suffix: string, init: RequestInit | undefined, apply: (payload: unknown) => void,
  ): Promise<void> => {
    const admitted = owner.current;
    if (!sessionId || admitted.sessionId !== sessionId) return;
    const owns = () => owner.current === admitted;
    if (!init) setIsLoading(true);
    setError(null);
    try {
      const response = await fetch(`${apiUrl}/recordings/live/${sessionId}/pages${suffix}`, init);
      const payload: unknown = await response.json().catch(() => null);
      if (!owns()) return;
      if (!response.ok) throw new Error(parseErrorMessage(payload) ?? `Browser tab request failed: ${response.statusText}`);
      apply(payload);
    } catch (err) {
      if (!owns()) return;
      const message = err instanceof Error ? err.message : 'Browser tab request failed';
      setError(message);
      console.error('[usePages]', message);
    } finally {
      if (owns() && !init) setIsLoading(false);
    }
  }, [apiUrl, sessionId]);

  const refreshPages = useCallback(() => request('', undefined, payload => {
    const data = parsePagesResponse(payload);
    if (!data || data.pages.some(page => page.sessionId !== sessionId)) throw new Error('Invalid pages response');
    useSessionStore.setState({pages: new Map(data.pages.map(page => [page.id, page])), activePageId: data.activePageId || null});
    useSessionStore.getState().updatePageColorMap(data.pages);
  }), [request, sessionId]);

  useEffect(() => {
    owner.current = {sessionId};
    useSessionStore.setState({pages: new Map(), activePageId: null, pageColorMap: new Map()});
    setError(null);
    setIsLoading(false);
    if (sessionId && isValidated) void refreshPages();
    return () => {owner.current = {sessionId: null};};
  }, [sessionId, isValidated, refreshPages]);

  // Mutate the canonical store and notify consumers outside React state updaters.
  useWebSocketMessage((lastMessage) => {
    if (!sessionId) return;
    const message = parsePageEventMessage(lastMessage);
    if (message?.session_id === sessionId) {
      const event = message.event;
      const state = useSessionStore.getState();
      const existing = state.pages.get(event.pageId);
      if (event.type === 'page_created') {
        observeCreatedPage({id: event.pageId, sessionId, url: event.url || '', title: event.title || '',
          faviconUrl: event.faviconUrl, openerId: event.openerId, isInitial: false, status: 'active', createdAt: event.timestamp});
      } else if (event.type === 'page_navigated' && existing?.status === 'active') {
        const url = event.url ?? existing.url;
        const title = event.title ?? existing.title;
        const faviconUrl = event.faviconUrl ?? (url === existing.url ? existing.faviconUrl : '');
        state.updatePage(event.pageId, {url, title, faviconUrl});
      } else if (event.type === 'page_closed' && existing?.status === 'active') {
        state.updatePage(event.pageId, {status: 'closed', closedAt: event.timestamp});
      }
    }
    const selection = parsePageSwitchMessage(lastMessage);
    if (selection?.session_id === sessionId) useSessionStore.getState().setActivePageId(selection.active_page_id || null);
  });

  const switchToPage = useCallback((pageId: string) => request(`/${pageId}/activate`, {method: 'POST'}, () => {
    useSessionStore.getState().setActivePageId(pageId);
  }), [request]);

  const closePage = useCallback((pageId: string) => request(`/${pageId}/close`, {method: 'POST'}, payload => {
    const selected = isRecord(payload) && typeof payload.activePageId === 'string' ? payload.activePageId
      : isRecord(payload) && typeof payload.active_page_id === 'string' ? payload.active_page_id : null;
    if (selected === null) throw new Error('Invalid close-page response');
    const state = useSessionStore.getState();
    const wasOpen = state.pages.get(pageId)?.status === 'active';
    if (wasOpen) state.updatePage(pageId, {status: 'closed', closedAt: new Date().toISOString()});
    state.setActivePageId(selected || null);
  }), [request]);

  const createPage = useCallback((url?: string) => request('', {
    method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({url: url || 'about:blank'}),
  }, payload => {
    const page = isRecord(payload) ? parsePage(payload.page) : null;
    if (!page || page.sessionId !== sessionId || !isRecord(payload) || payload.activePageId !== page.id) {
      throw new Error('Invalid created-page response');
    }
    observeCreatedPage(page, page.id);
  }), [request, sessionId, observeCreatedPage]);

  const pageList = useMemo(() => [...pages.values()].sort((a, b) =>
    new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()), [pages]);
  const openPages = useMemo(() => pageList.filter(page => page.status === 'active'), [pageList]);
  return {
    pages, pageList, openPages, activePageId, activePage: activePageId ? pages.get(activePageId) ?? null : null,
    isLoading, error, switchToPage, closePage, createPage, refreshPages,
    openPageCount: openPages.length, hasMultiplePages: openPages.length > 1,
  };
}
