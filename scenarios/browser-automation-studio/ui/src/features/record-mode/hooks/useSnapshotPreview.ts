import { useCallback, useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

interface SnapshotPreviewOptions {
  url: string | null;
  enabled?: boolean;
  sessionId?: string | null;
}

interface SnapshotPreviewState {
  imageUrl: string | null;
  isLoading: boolean;
  error: string | null;
  lastUpdated: number | null;
  refresh: () => Promise<void>;
}

/**
 * Lightweight snapshot fetcher that reuses the existing preview-screenshot API.
 */
export function useSnapshotPreview({ url, enabled = true, sessionId }: SnapshotPreviewOptions): SnapshotPreviewState {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);
  const inFlightRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    if (!url || !enabled) {
      return;
    }

    // Cancel any previous request
    if (inFlightRef.current) {
      inFlightRef.current.abort();
    }

    const abortController = new AbortController();
    inFlightRef.current = abortController;

    setIsLoading(true);
    setError(null);

    try {
      const config = await getConfig();
      const endpoint = sessionId
        ? `${config.API_URL}/recordings/live/${sessionId}/screenshot`
        : `${config.API_URL}/preview-screenshot`;

      const body = sessionId ? JSON.stringify({}) : JSON.stringify({ url });

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        signal: abortController.signal,
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || `Failed to fetch snapshot (${response.status})`);
      }

      const data = await response.json() as { screenshot?: string };
      if (!data?.screenshot) {
        throw new Error('No screenshot returned from preview endpoint');
      }

      setImageUrl(data.screenshot);
      setLastUpdated(Date.now());
    } catch (err) {
      if ((err as Error).name === 'AbortError') {
        return;
      }
      const message = err instanceof Error ? err.message : 'Failed to fetch snapshot';
      setError(message);
      logger.error('Snapshot preview failed', { component: 'useSnapshotPreview', url }, err);
    } finally {
      setIsLoading(false);
      inFlightRef.current = null;
    }
  }, [url, enabled, sessionId]);

  useEffect(() => {
    if (enabled && url) {
      void refresh();
    }
    return () => {
      if (inFlightRef.current) {
        inFlightRef.current.abort();
      }
    };
  }, [enabled, url, refresh]);

  return { imageUrl, isLoading, error, lastUpdated, refresh };
}

export type { SnapshotPreviewState };
