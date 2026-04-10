import { useCallback, useEffect, useRef, useState } from 'react';
import type { App } from '@/types';
import { logger } from '@/services/logger';
import { locateAppByIdentifier, resolveAppIdentifier } from '@/utils/appPreview';

export const DEFAULT_METADATA_FETCH_COOLDOWN_MS = 1500;

type MetricEvent =
  | 'requested'
  | 'completed'
  | 'failed'
  | 'notFound'
  | 'skippedCooldown'
  | 'skippedInFlight'
  | 'ignoredStale';

export interface PaneMetadataMetrics {
  requested: number;
  completed: number;
  failed: number;
  notFound: number;
  skippedCooldown: number;
  skippedInFlight: number;
  ignoredStale: number;
}

interface UsePaneMetadataOptions {
  paneId: string;
  apps: App[];
  resolvedAppIdentifier: string | null;
  scenarioIdentifierFromUrl: string | null;
  shouldPreferExistingApp: (existing: App | null, incoming: App) => boolean;
  setAppsState: (updater: (currentApps: App[]) => App[]) => void;
  getApp: (identifier: string) => Promise<App | null>;
  setStatusMessage: (message: string | null) => void;
  onResetForMissingIdentifier: () => void;
  cooldownMs?: number;
}

const createInitialMetrics = (): PaneMetadataMetrics => ({
  requested: 0,
  completed: 0,
  failed: 0,
  notFound: 0,
  skippedCooldown: 0,
  skippedInFlight: 0,
  ignoredStale: 0,
});

export const usePaneMetadata = ({
  paneId,
  apps,
  resolvedAppIdentifier,
  scenarioIdentifierFromUrl,
  shouldPreferExistingApp,
  setAppsState,
  getApp,
  setStatusMessage,
  onResetForMissingIdentifier,
  cooldownMs = DEFAULT_METADATA_FETCH_COOLDOWN_MS,
}: UsePaneMetadataOptions) => {
  const [currentApp, setCurrentApp] = useState<App | null>(null);
  const [isMetadataLoading, setIsMetadataLoading] = useState(false);
  const metricsRef = useRef<PaneMetadataMetrics>(createInitialMetrics());

  const activeFetchKeyRef = useRef<string | null>(null);
  const metadataFetchTimestampsRef = useRef<Map<string, number>>(new Map());
  const requestGenerationRef = useRef(0);

  const bumpMetric = useCallback((event: MetricEvent, details?: Record<string, unknown>) => {
    metricsRef.current = {
      ...metricsRef.current,
      [event]: metricsRef.current[event] + 1,
    };
    logger.debug('[preview-pane] Metadata hydration event', {
      paneId,
      event,
      metrics: metricsRef.current,
      ...(details ?? {}),
    });
  }, [paneId]);

  useEffect(() => {
    if (!resolvedAppIdentifier) {
      requestGenerationRef.current += 1;
      activeFetchKeyRef.current = null;
      setCurrentApp(null);
      setIsMetadataLoading(false);
      onResetForMissingIdentifier();
      return;
    }

    const localMatch = locateAppByIdentifier(apps, resolvedAppIdentifier);
    if (localMatch) {
      setCurrentApp((existing) => (
        shouldPreferExistingApp(existing, localMatch) ? existing : localMatch
      ));
      const needsHydration = Boolean(
        localMatch.is_partial || !localMatch.status || localMatch.status === 'unknown',
      );
      if (!needsHydration) {
        setIsMetadataLoading(false);
        return;
      }
    }

    setIsMetadataLoading(true);
    if (!localMatch) {
      setStatusMessage('Loading app metadata...');
    }

    const retainMatchingCurrentApp = (): boolean => {
      let retained = false;
      setCurrentApp((previous) => {
        const stillMatches = Boolean(previous && locateAppByIdentifier([previous], resolvedAppIdentifier));
        if (stillMatches) {
          retained = true;
          return previous;
        }
        return null;
      });
      return retained;
    };

    const fetchIdentifiers = [
      localMatch ? resolveAppIdentifier(localMatch) : null,
      resolvedAppIdentifier,
      scenarioIdentifierFromUrl,
    ].filter((value, index, list): value is string => (
      typeof value === 'string' && value.trim().length > 0 && list.indexOf(value) === index
    ));
    if (fetchIdentifiers.length === 0) {
      setIsMetadataLoading(false);
      return;
    }

    const fetchKey = fetchIdentifiers.join('|');
    if (activeFetchKeyRef.current === fetchKey) {
      bumpMetric('skippedInFlight', { fetchKey });
      return;
    }

    const now = Date.now();
    const lastFetchAt = metadataFetchTimestampsRef.current.get(fetchKey) ?? 0;
    if (now - lastFetchAt < cooldownMs) {
      setIsMetadataLoading(false);
      bumpMetric('skippedCooldown', { fetchKey, elapsedMs: now - lastFetchAt, cooldownMs });
      return;
    }
    metadataFetchTimestampsRef.current.set(fetchKey, now);
    if (metadataFetchTimestampsRef.current.size > 100) {
      metadataFetchTimestampsRef.current.clear();
      metadataFetchTimestampsRef.current.set(fetchKey, now);
    }

    const requestGeneration = requestGenerationRef.current + 1;
    requestGenerationRef.current = requestGeneration;
    activeFetchKeyRef.current = fetchKey;
    bumpMetric('requested', { fetchKey });

    const fetchAppMetadata = async (): Promise<App | null> => {
      for (const identifier of fetchIdentifiers) {
        const fetched = await getApp(identifier);
        if (fetched) {
          return fetched;
        }
      }
      return null;
    };

    fetchAppMetadata()
      .then((fetched) => {
        if (requestGenerationRef.current !== requestGeneration) {
          bumpMetric('ignoredStale', { fetchKey });
          return;
        }
        if (!fetched) {
          const retained = retainMatchingCurrentApp();
          setStatusMessage(retained ? 'App metadata unavailable.' : 'App not found.');
          setIsMetadataLoading(false);
          bumpMetric('notFound', { fetchKey });
          return;
        }
        setCurrentApp(fetched);
        setIsMetadataLoading(false);
        bumpMetric('completed', { fetchKey, appId: fetched.id });
        setAppsState((currentApps) => {
          const index = currentApps.findIndex((app) => (
            app.id === fetched.id
            || app.scenario_name === fetched.scenario_name
            || app.name === fetched.name
          ));
          if (index < 0) {
            return [...currentApps, fetched];
          }
          const updated = [...currentApps];
          updated[index] = { ...updated[index], ...fetched };
          return updated;
        });
      })
      .catch((error) => {
        if (requestGenerationRef.current !== requestGeneration) {
          bumpMetric('ignoredStale', { fetchKey, reason: 'error' });
          return;
        }
        logger.warn('[preview-pane] Failed to load app', error);
        retainMatchingCurrentApp();
        setStatusMessage('Failed to load app metadata.');
        setIsMetadataLoading(false);
        bumpMetric('failed', { fetchKey });
      })
      .finally(() => {
        if (
          requestGenerationRef.current === requestGeneration
          && activeFetchKeyRef.current === fetchKey
        ) {
          activeFetchKeyRef.current = null;
        }
      });

    return undefined;
  }, [
    apps,
    bumpMetric,
    cooldownMs,
    getApp,
    onResetForMissingIdentifier,
    resolvedAppIdentifier,
    scenarioIdentifierFromUrl,
    setAppsState,
    setStatusMessage,
    shouldPreferExistingApp,
  ]);

  return {
    currentApp,
    setCurrentApp,
    isMetadataLoading,
    metrics: metricsRef.current,
  };
};

export default usePaneMetadata;
