import { useEffect, useRef } from 'react';
import { appService } from '@/services/api';
import { matchesAppIdentifier } from '@/utils/appPreview';
import type { App } from '@/types';

/**
 * Records app views and updates view statistics (count, timestamps).
 * Automatically debounces rapid view changes to avoid excessive API calls.
 */
interface UseAppViewRecordingOptions {
  /** Current app ID being viewed */
  appId: string | null;
  /** Callback to update apps in the global store */
  setAppsState: (updater: (prev: App[]) => App[]) => void;
  /** Callback to update the current app state */
  setCurrentApp: React.Dispatch<React.SetStateAction<App | null>>;
}

export const useAppViewRecording = ({
  appId,
  setAppsState,
  setCurrentApp,
}: UseAppViewRecordingOptions): void => {
  const lastRecordedViewRef = useRef<{ id: string | null; timestamp: number }>({ id: null, timestamp: 0 });

  useEffect(() => {
    if (!appId) {
      lastRecordedViewRef.current = { id: null, timestamp: 0 };
      return;
    }

    const now = Date.now();
    const { id: lastId, timestamp } = lastRecordedViewRef.current;
    if (lastId === appId && now - timestamp < 1000) {
      return;
    }

    lastRecordedViewRef.current = { id: appId, timestamp: now };

    void (async () => {
      const stats = await appService.recordAppView(appId);
      if (!stats) {
        return;
      }

      const targets = [appId, stats.scenario_name];

      setAppsState(prev => prev.map(app => {
        if (!targets.some(target => matchesAppIdentifier(app, target))) {
          return app;
        }

        return {
          ...app,
          view_count: stats.view_count,
          last_viewed_at: stats.last_viewed_at ?? app.last_viewed_at ?? null,
          first_viewed_at: stats.first_viewed_at ?? app.first_viewed_at ?? null,
        };
      }));

      setCurrentApp(prev => {
        if (!prev) {
          return prev;
        }

        if (!targets.some(target => matchesAppIdentifier(prev, target))) {
          return prev;
        }

        return {
          ...prev,
          view_count: stats.view_count,
          last_viewed_at: stats.last_viewed_at ?? prev.last_viewed_at ?? null,
          first_viewed_at: stats.first_viewed_at ?? prev.first_viewed_at ?? null,
        };
      });
    })();
  }, [appId, setAppsState, setCurrentApp]);
};
