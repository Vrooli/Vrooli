import { useCallback, useMemo, useRef, useState } from 'react';
import { appService } from '@/services/api';
import type { App } from '@/types';
import { isRunningStatus } from '@/utils/appPreview';
import { logger } from '@/services/logger';

export type PreviewLifecycleAction = 'start' | 'stop' | 'restart';

type LifecycleContext = {
  appId: string;
  action: PreviewLifecycleAction;
};

interface UsePreviewAppLifecycleOptions {
  currentApp: App | null;
  setStatusMessage: (message: string | null) => void;
  controlApp?: (appId: string, action: PreviewLifecycleAction) => Promise<boolean>;
  onSuccess?: (context: LifecycleContext) => void | Promise<void>;
  onFailure?: (context: LifecycleContext) => void | Promise<void>;
  initialMessageForAction?: (action: PreviewLifecycleAction) => string;
  failureMessageForAction?: (action: PreviewLifecycleAction) => string;
  toggleLabels?: {
    start: string;
    stop: string;
  };
  restartLabel?: string;
}

interface PreviewAppLifecycleResult {
  pendingAction: PreviewLifecycleAction | null;
  actionInProgress: boolean;
  isAppRunning: boolean;
  appStatusLabel: string;
  urlStatusClass: string;
  toggleActionLabel: string;
  restartActionLabel: string;
  runAction: (appId: string, action: PreviewLifecycleAction) => Promise<boolean>;
  handleToggleCurrentApp: () => Promise<boolean>;
  handleRestartCurrentApp: () => Promise<boolean>;
}

const defaultInitialMessage = (action: PreviewLifecycleAction): string => (
  action === 'start'
    ? 'Starting application...'
    : action === 'stop'
      ? 'Stopping application...'
      : 'Restarting application...'
);

const defaultFailureMessage = (action: PreviewLifecycleAction): string => (
  `Unable to ${action} the application. Check logs for details.`
);

export function usePreviewAppLifecycle({
  currentApp,
  setStatusMessage,
  controlApp = appService.controlApp,
  onSuccess,
  onFailure,
  initialMessageForAction = defaultInitialMessage,
  failureMessageForAction = defaultFailureMessage,
  toggleLabels = { start: 'Start app', stop: 'Stop app' },
  restartLabel = 'Restart app',
}: UsePreviewAppLifecycleOptions): PreviewAppLifecycleResult {
  const [pendingAction, setPendingAction] = useState<PreviewLifecycleAction | null>(null);

  const isAppRunning = useMemo(
    () => Boolean(currentApp && isRunningStatus(currentApp.status)),
    [currentApp],
  );

  const appStatusLabel = useMemo(() => {
    const rawStatus = (currentApp?.status ?? 'unknown').trim();
    if (rawStatus.length === 0) {
      return 'Unknown';
    }
    return rawStatus.charAt(0).toUpperCase() + rawStatus.slice(1);
  }, [currentApp?.status]);

  const urlStatusClass = useMemo(() => {
    const rawStatus = (currentApp?.status ?? 'unknown').trim().toLowerCase();
    return rawStatus.length > 0 ? rawStatus : 'unknown';
  }, [currentApp?.status]);

  const runAction = useCallback(async (appId: string, action: PreviewLifecycleAction): Promise<boolean> => {
    if (pendingAction) {
      return false;
    }

    setPendingAction(action);
    setStatusMessage(initialMessageForAction(action));

    try {
      const success = await controlApp(appId, action);
      if (!success) {
        setStatusMessage(failureMessageForAction(action));
        if (onFailure) {
          await onFailure({ appId, action });
        }
        return false;
      }

      if (onSuccess) {
        await onSuccess({ appId, action });
      }
      return true;
    } catch (error) {
      logger.error(`[preview-lifecycle] Failed to ${action} app ${appId}`, error);
      setStatusMessage(failureMessageForAction(action));
      if (onFailure) {
        await onFailure({ appId, action });
      }
      return false;
    } finally {
      setPendingAction(null);
    }
  }, [
    controlApp,
    failureMessageForAction,
    initialMessageForAction,
    onFailure,
    onSuccess,
    pendingAction,
    setStatusMessage,
  ]);

  const handleToggleCurrentApp = useCallback(async (): Promise<boolean> => {
    if (!currentApp || pendingAction) {
      return false;
    }
    const action: PreviewLifecycleAction = isRunningStatus(currentApp.status) ? 'stop' : 'start';
    return runAction(currentApp.id, action);
  }, [currentApp, pendingAction, runAction]);

  const handleRestartCurrentApp = useCallback(async (): Promise<boolean> => {
    if (!currentApp || pendingAction || !isRunningStatus(currentApp.status)) {
      return false;
    }
    return runAction(currentApp.id, 'restart');
  }, [currentApp, pendingAction, runAction]);

  const toggleLabelsRef = useRef(toggleLabels);
  toggleLabelsRef.current = toggleLabels;
  const restartLabelRef = useRef(restartLabel);
  restartLabelRef.current = restartLabel;

  return useMemo(() => ({
    pendingAction,
    actionInProgress: pendingAction !== null,
    isAppRunning,
    appStatusLabel,
    urlStatusClass,
    toggleActionLabel: isAppRunning ? toggleLabelsRef.current.stop : toggleLabelsRef.current.start,
    restartActionLabel: pendingAction === 'restart' ? 'Restarting...' : restartLabelRef.current,
    runAction,
    handleToggleCurrentApp,
    handleRestartCurrentApp,
  }), [
    pendingAction,
    isAppRunning,
    appStatusLabel,
    urlStatusClass,
    runAction,
    handleToggleCurrentApp,
    handleRestartCurrentApp,
  ]);
}

export default usePreviewAppLifecycle;
