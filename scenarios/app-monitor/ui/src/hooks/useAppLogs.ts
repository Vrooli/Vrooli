import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { appService } from '@/services/api';
import type { App, AppLogStream } from '@/types';
import { resolveAppIdentifier } from '@/utils/appPreview';
import { logger } from '@/services/logger';

interface UseAppLogsOptions {
  app?: App | null;
  appId?: string | null;
  active?: boolean;
}

interface UseAppLogsResult {
  identifier: string | null;
  logs: string[];
  streams: AppLogStream[];
  loading: boolean;
  error: string | null;
  lastUpdatedAt: number | null;
  refresh: () => Promise<void>;
  clear: () => void;
}

const deriveIdentifier = (app?: App | null, fallback?: string | null): string | null => {
  if (app) {
    const candidate = resolveAppIdentifier(app) || app.id || app.scenario_name || app.name;
    if (candidate && candidate.trim().length > 0) {
      return candidate.trim();
    }
  }
  if (fallback && fallback.trim().length > 0) {
    return fallback.trim();
  }
  return null;
};

export const useAppLogs = ({ app, appId, active = true }: UseAppLogsOptions): UseAppLogsResult => {
  const [logs, setLogs] = useState<string[]>([]);
  const [streams, setStreams] = useState<AppLogStream[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdatedAt, setLastUpdatedAt] = useState<number | null>(null);
  const isMountedRef = useRef(true);
  const fetchPromiseRef = useRef<Promise<void> | null>(null);

  useEffect(() => {
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  const identifier = useMemo(() => deriveIdentifier(app, appId ?? null), [app, appId]);

  const clear = useCallback(() => {
    if (!isMountedRef.current) {
      return;
    }
    setLogs([]);
    setStreams([]);
    setError(null);
    setLastUpdatedAt(null);
  }, []);

  const refresh = useCallback(async () => {
    if (!identifier) {
      clear();
      return;
    }

    if (fetchPromiseRef.current) {
      return fetchPromiseRef.current;
    }

    const promise = (async () => {
      try {
        if (!isMountedRef.current) {
          return;
        }
        setLoading(true);
        setError(null);
        const result = await appService.getAppLogs(identifier, 'both');
        if (!isMountedRef.current) {
          return;
        }
        const combinedLogs = Array.isArray(result.logs) ? result.logs : [];
        const combinedStreams = Array.isArray(result.streams) ? result.streams : [];
        setLogs(combinedLogs);
        setStreams(combinedStreams);
        setLastUpdatedAt(Date.now());
      } catch (error_) {
        if (!isMountedRef.current) {
          return;
        }
        logger.warn('[useAppLogs] Failed to load logs', error_);
        setError('Unable to load logs for this scenario.');
        setLogs([]);
        setStreams([]);
      } finally {
        if (isMountedRef.current) {
          setLoading(false);
        }
        fetchPromiseRef.current = null;
      }
    })();

    fetchPromiseRef.current = promise;
    await promise;
  }, [clear, identifier]);

  useEffect(() => {
    if (!identifier) {
      clear();
      setLoading(false);
      return;
    }
    clear();
  }, [clear, identifier]);

  useEffect(() => {
    if (!active || !identifier) {
      return;
    }
    void refresh();
  }, [active, identifier, refresh]);

  return {
    identifier,
    logs,
    streams,
    loading,
    error,
    lastUpdatedAt,
    refresh,
    clear,
  };
};

export type { UseAppLogsOptions, UseAppLogsResult };
