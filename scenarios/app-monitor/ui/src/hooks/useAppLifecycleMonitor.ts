import { useCallback, useRef } from 'react';
import { appService } from '@/services/api';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus } from '@/utils/appPreview';
import { logger } from '@/services/logger';
import { PREVIEW_TIMEOUTS } from '@/components/views/previewConstants';
import type { App } from '@/types';
import type { PreviewOverlayState } from '@/types/preview';

/**
 * Monitors app lifecycle operations (start/restart) and polls for status changes.
 * Provides callbacks to begin monitoring and cancel ongoing monitors.
 */
interface UseAppLifecycleMonitorOptions {
  /** Current app identifier for comparison during polling */
  currentAppIdentifier: string | null;
  /** Whether user has set a custom preview URL (affects auto-URL updates) */
  hasCustomPreviewUrl: boolean;
  /** Callback to apply default preview URL when app becomes ready */
  applyDefaultPreviewUrl: (url: string) => void;
  /** Callback to commit app updates to global state */
  commitAppUpdate: (app: App) => void;
  /** Callback to reload the preview iframe */
  reloadPreview: () => void;
  /** Callback to set loading state */
  setLoading: (loading: boolean) => void;
  /** Callback to set preview overlay state */
  setPreviewOverlay: React.Dispatch<React.SetStateAction<PreviewOverlayState>>;
  /** Callback to set status message */
  setStatusMessage: (message: string | null) => void;
}

export interface UseAppLifecycleMonitorReturn {
  /**
   * Begins monitoring an app's lifecycle after a start/restart command.
   * Polls the API until the app is running or timeout is reached.
   */
  beginLifecycleMonitor: (appIdentifier: string, lifecycle: 'start' | 'restart') => void;
  /**
   * Stops any active lifecycle monitoring.
   */
  stopLifecycleMonitor: () => void;
}

export const useAppLifecycleMonitor = ({
  currentAppIdentifier,
  hasCustomPreviewUrl,
  applyDefaultPreviewUrl,
  commitAppUpdate,
  reloadPreview,
  setLoading,
  setPreviewOverlay,
  setStatusMessage,
}: UseAppLifecycleMonitorOptions): UseAppLifecycleMonitorReturn => {
  const restartMonitorRef = useRef<{ cancel: () => void } | null>(null);

  const stopLifecycleMonitor = useCallback(() => {
    if (restartMonitorRef.current) {
      restartMonitorRef.current.cancel();
      restartMonitorRef.current = null;
    }
  }, []);

  const beginLifecycleMonitor = useCallback((appIdentifier: string, lifecycle: 'start' | 'restart') => {
    stopLifecycleMonitor();

    let cancelled = false;
    const monitoredAppId = appIdentifier;
    restartMonitorRef.current = {
      cancel: () => {
        cancelled = true;
      },
    };

    const poll = async () => {
      const maxAttempts = 30;
      for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
        if (cancelled) {
          return;
        }

        try {
          const fetched = await appService.getApp(appIdentifier);
          if (cancelled) {
            return;
          }

          if (fetched) {
            commitAppUpdate(fetched);

            if (isRunningStatus(fetched.status) && !isStoppedStatus(fetched.status)) {
              const candidateUrl = buildPreviewUrl(fetched);
              if (candidateUrl) {
                // Only update UI state if we're still viewing the same app
                const stillViewing = currentAppIdentifier === monitoredAppId;
                if (stillViewing) {
                  if (!hasCustomPreviewUrl) {
                    applyDefaultPreviewUrl(candidateUrl);
                  }
                  setStatusMessage(lifecycle === 'restart'
                    ? 'Application restarted. Refreshing preview...'
                    : 'Application started. Refreshing preview...');
                  setPreviewOverlay(null);
                  setLoading(false);
                  reloadPreview();
                  window.setTimeout(() => {
                    if (!cancelled && currentAppIdentifier === monitoredAppId) {
                      setStatusMessage(null);
                    }
                  }, PREVIEW_TIMEOUTS.STATUS_MESSAGE_DURATION);
                }
                stopLifecycleMonitor();
                return;
              }
            }
          }
        } catch (error) {
          logger.warn('Lifecycle monitor poll failed', error);
        }

        const delay = attempt < 5 ? 1000 : 2000;
        await new Promise(resolve => window.setTimeout(resolve, delay));
      }

      // Only show error overlay if we're still viewing the same app
      if (!cancelled && currentAppIdentifier === monitoredAppId) {
        setLoading(false);
        setPreviewOverlay({
          type: 'error',
          message: lifecycle === 'restart'
            ? 'Application has not come back online yet. Try refreshing.'
            : 'Application is still starting. Try refreshing in a moment.',
        });
      }
    };

    void poll();
  }, [
    applyDefaultPreviewUrl,
    commitAppUpdate,
    currentAppIdentifier,
    hasCustomPreviewUrl,
    reloadPreview,
    setLoading,
    setPreviewOverlay,
    setStatusMessage,
    stopLifecycleMonitor,
  ]);

  return {
    beginLifecycleMonitor,
    stopLifecycleMonitor,
  };
};
