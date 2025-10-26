import { useCallback, useRef } from 'react';
import { appService } from '@/services/api';
import type { App } from '@/types';
import { isRunningStatus, isStoppedStatus, buildPreviewUrl } from '@/utils/appPreview';
import { logger } from '@/services/logger';

type PreviewOverlay = { type: 'restart' | 'waiting' | 'error'; message: string } | null;

interface UseAppLifecycleOptions {
  currentAppIdentifier: string | null;
  hasCustomPreviewUrl: boolean;
  applyDefaultPreviewUrl: (url: string) => void;
  reloadPreview: () => void;
  setLoading: (loading: boolean) => void;
  setStatusMessage: (message: string | null) => void;
  setPreviewOverlay: (overlay: PreviewOverlay | ((prev: PreviewOverlay) => PreviewOverlay)) => void;
  setAppsState: (updater: (apps: App[]) => App[]) => void;
  setCurrentApp: (updater: (app: App | null) => App | null) => void;
}

export interface UseAppLifecycleReturn {
  executeAppAction: (appToControl: string, action: 'start' | 'stop' | 'restart') => Promise<boolean>;
  handleAppAction: (appToControl: string, action: 'start' | 'stop' | 'restart') => Promise<void>;
  stopLifecycleMonitor: () => void;
}

export const useAppLifecycle = ({
  currentAppIdentifier,
  hasCustomPreviewUrl,
  applyDefaultPreviewUrl,
  reloadPreview,
  setLoading,
  setStatusMessage,
  setPreviewOverlay,
  setAppsState,
  setCurrentApp,
}: UseAppLifecycleOptions): UseAppLifecycleReturn => {
  const restartMonitorRef = useRef<{ cancel: () => void } | null>(null);

  const commitAppUpdate = useCallback((nextApp: App) => {
    setAppsState(prev => {
      const index = prev.findIndex(app => app.id === nextApp.id);
      if (index === -1) {
        return [...prev, nextApp];
      }

      const updated = [...prev];
      updated[index] = nextApp;
      return updated;
    });

    setCurrentApp(prev => {
      if (!prev) {
        return nextApp;
      }
      return prev.id === nextApp.id ? nextApp : prev;
    });
  }, [setAppsState, setCurrentApp]);

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
                  }, 1500);
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

  const executeAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    const actionInProgressMessage = action === 'stop'
      ? 'Stopping application...'
      : action === 'start'
        ? 'Starting application...'
        : 'Restarting application...';
    setStatusMessage(actionInProgressMessage);

    try {
      const success = await appService.controlApp(appToControl, action);
      if (!success) {
        const failureMessage = action === 'start'
          ? `Unable to start application. Check logs for details.`
          : `Unable to ${action} the application. Check logs for details.`;
        setStatusMessage(failureMessage);
        return false;
      }

      const timestamp = new Date().toISOString();
      if (action === 'start' || action === 'stop') {
        const nextStatus: App['status'] = action === 'stop' ? 'stopped' : 'running';
        setAppsState(prev => prev.map(app => (app.id === appToControl ? { ...app, status: nextStatus, updated_at: timestamp } : app)));
        setCurrentApp(prev => (prev && prev.id === appToControl ? { ...prev, status: nextStatus, updated_at: timestamp } : prev));
        setStatusMessage(action === 'stop'
          ? 'Application stopped. Start it again to relaunch the UI preview.'
          : 'Application started. Preview will refresh automatically.');
      } else {
        setStatusMessage('Restart command sent. Waiting for application to return...');
      }

      return true;
    } catch (error) {
      logger.error(`Failed to ${action} app ${appToControl}`, error);
      const failureMessage = action === 'start'
        ? `Unable to start application. Check logs for details.`
        : `Unable to ${action} the application. Check logs for details.`;
      setStatusMessage(failureMessage);
      return false;
    }
  }, [setAppsState, setCurrentApp, setStatusMessage]);

  const handleAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    if (action === 'restart') {
      setPreviewOverlay({ type: 'restart', message: 'Restarting application...' });
      setLoading(true);
      reloadPreview();
    } else if (action === 'start') {
      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to start...' });
      setLoading(true);
    }

    const success = await executeAppAction(appToControl, action);
    if (!success) {
      if (action === 'start') {
        setPreviewOverlay({ type: 'error', message: 'Unable to start application. Check logs for details.' });
        setLoading(false);
      } else if (action === 'restart') {
        setPreviewOverlay({ type: 'error', message: 'Unable to restart the application. Check logs for details.' });
        setLoading(false);
      }
      return;
    }

    if (action === 'start') {
      beginLifecycleMonitor(appToControl, 'start');
    } else if (action === 'restart') {
      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to restart...' });
      beginLifecycleMonitor(appToControl, 'restart');
    } else if (action === 'stop') {
      setPreviewOverlay((prev: PreviewOverlay) => (prev && prev.type === 'waiting' ? null : prev));
      setLoading(false);
    }
  }, [beginLifecycleMonitor, executeAppAction, reloadPreview, setLoading, setPreviewOverlay]);

  return {
    executeAppAction,
    handleAppAction,
    stopLifecycleMonitor,
  };
};
