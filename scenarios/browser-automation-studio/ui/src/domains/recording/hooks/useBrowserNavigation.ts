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

import { useCallback, useState } from 'react';
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
  const parseEntry = (entry: unknown) => {
    if (!isRecord(entry)) return null;
    const url = typeof entry.url === 'string' ? entry.url : null;
    const title = typeof entry.title === 'string' ? entry.title : null;
    const timestamp = typeof entry.timestamp === 'string' ? entry.timestamp : null;
    if (!url || !title || !timestamp) return null;
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
  /** Initial URL (e.g., from template) */
  initialUrl?: string;
}

interface UseBrowserNavigationReturn {
  /** Current preview URL */
  previewUrl: string;
  /** Set the preview URL */
  setPreviewUrl: (url: string) => void;
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
  /** Fetch navigation stack for right-click popup */
  handleFetchNavigationStack: () => Promise<NavigationStackData | null>;
  /** Navigate multiple steps back/forward (negative = back, positive = forward) */
  handleNavigateToIndex: (delta: number) => Promise<void>;
  /** Update navigation state from API response */
  updateNavigationState: (data: { url?: string; can_go_back?: boolean; can_go_forward?: boolean }) => void;
}

export function useBrowserNavigation({
  sessionId,
  initialUrl = '',
}: UseBrowserNavigationOptions): UseBrowserNavigationReturn {
  const [previewUrl, setPreviewUrl] = useState(initialUrl);
  const [canGoBack, setCanGoBack] = useState(false);
  const [canGoForward, setCanGoForward] = useState(false);
  const [refreshToken, setRefreshToken] = useState(0);

  // Update navigation state from API response
  const updateNavigationState = useCallback(
    (data: { url?: string; can_go_back?: boolean; can_go_forward?: boolean }) => {
      if (data.url !== undefined) {
        setPreviewUrl(data.url);
      }
      if (data.can_go_back !== undefined) {
        setCanGoBack(data.can_go_back);
      }
      if (data.can_go_forward !== undefined) {
        setCanGoForward(data.can_go_forward);
      }
    },
    []
  );

  // Navigate browser back
  const handleGoBack = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/go-back`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data: unknown = await response.json();
        updateNavigationState(parseNavigationState(data));
      }
    } catch (err) {
      console.warn('Failed to go back:', err);
    }
  }, [sessionId, updateNavigationState]);

  // Navigate browser forward
  const handleGoForward = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/go-forward`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data: unknown = await response.json();
        updateNavigationState(parseNavigationState(data));
      }
    } catch (err) {
      console.warn('Failed to go forward:', err);
    }
  }, [sessionId, updateNavigationState]);

  // Refresh the current page
  const handleRefresh = useCallback(async () => {
    if (!sessionId) return;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/reload`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      if (response.ok) {
        const data: unknown = await response.json();
        const parsed = parseNavigationState(data);
        setCanGoBack(parsed.can_go_back ?? false);
        setCanGoForward(parsed.can_go_forward ?? false);
        // Trigger a refresh of the frame display
        setRefreshToken((t) => t + 1);
      }
    } catch (err) {
      console.warn('Failed to refresh:', err);
    }
  }, [sessionId]);

  // Fetch navigation stack for right-click popup
  const handleFetchNavigationStack = useCallback(async (): Promise<NavigationStackData | null> => {
    if (!sessionId) return null;
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/navigation-stack`);
      if (!response.ok) return null;
      const data: unknown = await response.json();
      return parseNavigationStackData(data);
    } catch (err) {
      console.warn('Failed to fetch navigation stack:', err);
      return null;
    }
  }, [sessionId]);

  // Navigate multiple steps back/forward
  const handleNavigateToIndex = useCallback(
    async (delta: number) => {
      if (!sessionId || delta === 0) return;
      try {
        const config = await getConfig();
        const endpoint = delta < 0 ? 'go-back' : 'go-forward';
        const steps = Math.abs(delta);

        let lastResponse: { url?: string; can_go_back?: boolean; can_go_forward?: boolean } | null =
          null;

        for (let i = 0; i < steps; i++) {
          const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({}),
          });
          if (!response.ok) break;
          const data: unknown = await response.json();
          lastResponse = parseNavigationState(data);
        }

        if (lastResponse) {
          updateNavigationState(lastResponse);
        }
      } catch (err) {
        console.warn('Failed to navigate to index:', err);
      }
    },
    [sessionId, updateNavigationState]
  );

  return {
    previewUrl,
    setPreviewUrl,
    canGoBack,
    canGoForward,
    refreshToken,
    handleGoBack,
    handleGoForward,
    handleRefresh,
    handleFetchNavigationStack,
    handleNavigateToIndex,
    updateNavigationState,
  };
}
