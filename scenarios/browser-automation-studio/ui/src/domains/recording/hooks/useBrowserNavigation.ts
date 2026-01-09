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
        const data = await response.json();
        updateNavigationState(data);
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
        const data = await response.json();
        updateNavigationState(data);
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
        const data = await response.json();
        setCanGoBack(data.can_go_back ?? false);
        setCanGoForward(data.can_go_forward ?? false);
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
      const data = await response.json();
      return {
        backStack: data.back_stack || [],
        current: data.current || null,
        forwardStack: data.forward_stack || [],
      };
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
          lastResponse = await response.json();
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
