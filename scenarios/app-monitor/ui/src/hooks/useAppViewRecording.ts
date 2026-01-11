import { useEffect, useRef } from 'react';
import { appService } from '@/services/api';
import { matchesAppIdentifier } from '@/utils/appPreview';
import type { App } from '@/types';
import { useRecentAppsStore } from '@/state/recentAppsStore';

/**
 * Records app views and updates view statistics (count, timestamps).
 * Automatically debounces rapid view changes to avoid excessive API calls.
 */
interface UseAppViewRecordingOptions {
  /** Current app ID being viewed */
  appId: string | null;
  /** Current app data for local history */
  appSnapshot?: App | null;
  /** Callback to update apps in the global store */
  setAppsState: (updater: (prev: App[]) => App[]) => void;
  /** Callback to update the current app state */
  setCurrentApp: React.Dispatch<React.SetStateAction<App | null>>;
}

export const useAppViewRecording = ({
  appId,
  appSnapshot,
  setAppsState,
  setCurrentApp,
}: UseAppViewRecordingOptions): void => {
  const lastRecordedViewRef = useRef<{ id: string | null; timestamp: number }>({ id: null, timestamp: 0 });
  const appSnapshotRef = useRef<App | null>(appSnapshot ?? null);
  const recordRecentApp = useRecentAppsStore(state => state.recordAppView);

  useEffect(() => {
    appSnapshotRef.current = appSnapshot ?? null;
  }, [appSnapshot]);

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
    const snapshot = appSnapshotRef.current;
    recordRecentApp({
      id: snapshot?.id ?? appId,
      name: snapshot?.name,
      scenario_name: snapshot?.scenario_name,
      status: snapshot?.status,
      view_count: snapshot?.view_count,
      last_viewed_at: snapshot?.last_viewed_at ?? new Date().toISOString(),
      completeness_score: snapshot?.completeness_score,
      completeness_classification: snapshot?.completeness_classification,
    });

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
  }, [appId, recordRecentApp, setAppsState, setCurrentApp]);
};
