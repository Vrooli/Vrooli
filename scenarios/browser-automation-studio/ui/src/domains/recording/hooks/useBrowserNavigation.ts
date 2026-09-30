/**
 * useBrowserNavigation Hook
 *
 * Encapsulates browser navigation state and handlers for the recording session.
 * Extracted from RecordingSession.tsx to reduce component complexity.
 *
 * Features:
 * - URL state management
 * - Back/Forward/Refresh navigation
 * - Navigation stack for right-click popup
 * - Multi-step navigation (delta-based)
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import type { NavigationStackData } from '../capture/BrowserChrome';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseNavigationState = (
  value: unknown
): { url?: string; can_go_back?: boolean; can_go_forward?: boolean } => {
  if (!isRecord(value)) return {};
  const url = typeof value.url === 'string' ? value.url : undefined;
  const canGoBack = typeof value.can_go_back === 'boolean' ? value.can_go_back : undefined;
  const canGoForward = typeof value.can_go_forward === 'boolean' ? value.can_go_forward : undefined;
  return { url, can_go_back: canGoBack, can_go_forward: canGoForward };
};

const parseNavigationStackData = (value: unknown): NavigationStackData | null => {
  if (!isRecord(value)) return null;
  const parseEntry = (entry: unknown): NavigationStackData['current'] => {
    if (!isRecord(entry)) return null;
    const url = typeof entry.url === 'string' ? entry.url : null;
    const title = typeof entry.title === 'string' ? entry.title : null;
    const timestamp = typeof entry.timestamp === 'string' ? entry.timestamp : undefined;
    if (!url || title === null) return null;
    return { url, title, timestamp };
  };

  const parseEntries = (entries: unknown): NavigationStackData['backStack'] => {
    if (!Array.isArray(entries)) return [];
    return entries.map(parseEntry).filter((entry): entry is NavigationStackData['backStack'][number] => entry !== null);
  };

  return {
    backStack: parseEntries(value.back_stack),
    current: parseEntry(value.current),
    forwardStack: parseEntries(value.forward_stack),
  };
};

interface UseBrowserNavigationOptions {
  /** Session ID for API calls */
  sessionId: string | null;
  /** Canonical page selected when the user issues a command. */
  pageId: string | null;
  /** Selected validated page location; undefined while awaiting admission. */
  observedUrl?: string;
  /** Initial URL (e.g., from template) */
  initialUrl?: string;
}

interface NavigationIntent {
  url: string;
  sessionId: string | null;
  pageId: string | null;
  /** Opaque ready-page lifetime at submission; returning to its ID is a new lifetime. */
  scope?: object;
}

interface UseBrowserNavigationReturn {
  /** Current preview URL */
  previewUrl: string;
  /** Observe a URL without commanding navigation. */
  setPreviewUrl: (url: string) => void;
  /** Submit an explicit navigation, waiting for session admission if needed. */
  handleNavigate: (url: string) => void;
  isInitialNavigationComplete: boolean;
  navigationError: string | null;
  /** Whether browser can go back */
  canGoBack: boolean;
  /** Whether browser can go forward */
  canGoForward: boolean;
  /** Refresh token - increment to trigger frame refresh */
  refreshToken: number;
  /** Navigate browser back */
  handleGoBack: () => Promise<void>;
  /** Navigate browser forward */
  handleGoForward: () => Promise<void>;
  /** Refresh the current page */
  handleRefresh: () => Promise<void>;
  /** Observe capabilities without replacing the URL draft. */
  refreshNavigationState: () => Promise<void>;
  /** Fetch navigation stack for right-click popup */
  handleFetchNavigationStack: () => Promise<NavigationStackData | null>;
  /** Navigate multiple steps back/forward (negative = back, positive = forward) */
  handleNavigateToIndex: (delta: number) => Promise<void>;
  /** Update navigation state from API response */
  updateNavigationState: (data: { url?: string; can_go_back?: boolean; can_go_forward?: boolean }) => void;
}

export function useBrowserNavigation({
  sessionId,
  pageId,
  initialUrl = '',
  observedUrl,
}: UseBrowserNavigationOptions): UseBrowserNavigationReturn {
  const [previewUrl, setPreviewUrl] = useState(initialUrl);
  const [canGoBack, setCanGoBack] = useState(false);
  const [canGoForward, setCanGoForward] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);
  const [request, setRequest] = useState<NavigationIntent | null>(() => initialUrl ? { url: initialUrl, sessionId, pageId } : null);
  const capabilityRead = useRef<AbortController | null>(null);
  const admitted = useRef<{ request: NavigationIntent; scope: object } | null>(null);
  const [isInitialNavigationComplete, setIsInitialNavigationComplete] = useState(!initialUrl);
  const [navigationError, setNavigationError] = useState<string | null>(null);
  const owner = useMemo(() => ({ sessionId, pageId, disposed: false, requests: new Set<AbortController>() }), [sessionId, pageId]);

  useLayoutEffect(() => {
    owner.disposed = false;
    setCanGoBack(false);
    setCanGoForward(false);
    return () => {
      owner.disposed = true;
      for (const controller of owner.requests) controller.abort();
      owner.requests.clear();
    };
  }, [owner]);

  useEffect(() => {
    if (observedUrl !== undefined) setPreviewUrl(observedUrl);
  }, [observedUrl, owner]);

  const updateNavigationState = useCallback((data: ReturnType<typeof parseNavigationState>) => {
    capabilityRead.current?.abort();
    if (data.url !== undefined) setPreviewUrl(data.url);
    if (data.can_go_back !== undefined) setCanGoBack(data.can_go_back);
    if (data.can_go_forward !== undefined) setCanGoForward(data.can_go_forward);
  }, []);

  // Every command/read retains the same page lifetime through config, transport and parsing.
  const send = useCallback(async (endpoint: string, body?: Record<string, unknown>, controller = new AbortController()): Promise<unknown | null> => {
    const current = () => !owner.disposed && !controller.signal.aborted;
    if (!owner.sessionId || !owner.pageId || !current()) return null;
    owner.requests.add(controller);
    setNavigationError(null);
    try {
      const config = await getConfig();
      if (!current()) return null;
      const query = body === undefined ? `?page_id=${encodeURIComponent(owner.pageId)}` : '';
      const response = await fetch(`${config.API_URL}/recordings/live/${owner.sessionId}/${endpoint}${query}`, {
        signal: controller.signal,
        ...(body === undefined ? {} : {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ ...body, page_id: owner.pageId }),
        }),
      });
      if (!response.ok) throw new Error(response.status === 409
        ? 'The selected tab changed. Try again.' : `Navigation failed (${response.status}). Try again.`);
      const data: unknown = await response.json();
      return current() ? data : null;
    } catch (error) {
      if (current()) setNavigationError(error instanceof Error ? error.message : 'Navigation failed. Try again.');
      return null;
    } finally {
      owner.requests.delete(controller);
    }
  }, [owner]);

  const execute = useCallback(async (endpoint: string, body: Record<string, unknown> = {}, steps = 1, controller = new AbortController()) => {
    for (let i = 0; i < steps; i++) {
      const data = await send(endpoint, body, controller);
      if (data === null || owner.disposed || controller.signal.aborted) return false;
      updateNavigationState(parseNavigationState(data));
      if (endpoint === 'reload') setRefreshToken(token => token + 1);
    }
    return true;
  }, [send, owner, updateNavigationState]);

  const handleNavigate = useCallback((url: string) => {
    if (!url || owner.disposed) return;
    setPreviewUrl(url);
    setRequest({ url, sessionId: owner.sessionId, pageId: owner.pageId,
      scope: owner.sessionId && owner.pageId ? owner : undefined });
  }, [owner]);

  useEffect(() => {
    if (!sessionId || !pageId || !request) return;
    if ((request.sessionId && request.sessionId !== sessionId) || (request.pageId && request.pageId !== pageId)) return;
    if (request.scope && request.scope !== owner) return;
    // A queued launch binds once; selection changes cannot replay it on another tab.
    if (admitted.current?.request === request && admitted.current.scope !== owner) return;
    admitted.current = { request, scope: owner };
    const controller = new AbortController();
    void execute('navigate', { url: request.url }, 1, controller).then(success => {
      if (success && !owner.disposed && !controller.signal.aborted) setIsInitialNavigationComplete(true);
    });
    return () => controller.abort();
  }, [sessionId, pageId, request, execute, owner]);

  const handleGoBack = useCallback(async () => { await execute('go-back'); }, [execute]);
  const handleGoForward = useCallback(async () => { await execute('go-forward'); }, [execute]);
  const handleRefresh = useCallback(async () => { await execute('reload'); }, [execute]);
  const handleNavigateToIndex = useCallback(async (delta: number) => {
    if (delta !== 0) await execute(delta < 0 ? 'go-back' : 'go-forward', {}, Math.abs(delta));
  }, [execute]);
  const refreshNavigationState = useCallback(async () => {
    capabilityRead.current?.abort();
    const controller = new AbortController();
    capabilityRead.current = controller;
    const data = parseNavigationState(await send('navigation-state', undefined, controller));
    if (owner.disposed || controller.signal.aborted) return;
    capabilityRead.current = null;
    setCanGoBack(data.can_go_back ?? false);
    setCanGoForward(data.can_go_forward ?? false);
  }, [send, owner]);
  const handleFetchNavigationStack = useCallback(async () => {
    const data = await send('navigation-stack');
    return owner.disposed ? null : parseNavigationStackData(data);
  }, [send, owner]);

  return {
    previewUrl, setPreviewUrl, handleNavigate, isInitialNavigationComplete, navigationError,
    canGoBack, canGoForward, refreshToken, handleGoBack, handleGoForward, handleRefresh,
    handleFetchNavigationStack, handleNavigateToIndex, updateNavigationState, refreshNavigationState,
  };
}
