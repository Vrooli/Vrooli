import { useCallback, useMemo, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { StorageStateResponse } from '../types/types';

export interface UseStorageStateResult {
  storageState: StorageStateResponse | null;
  loading: boolean;
  error: string | null;
  fetchStorageState: (profileId: string) => Promise<void>;
  clear: () => void;
}

export function useStorageState(): UseStorageStateResult {
  const [storageState, setStorageState] = useState<StorageStateResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStorageState = useCallback(async (profileId: string) => {
    setLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/storage`);
      if (!response.ok) {
        throw new Error(`Failed to fetch storage state (${response.status})`);
      }
      const data = (await response.json()) as StorageStateResponse;
      setStorageState(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load storage state';
      setError(message);
      logger.error(message, { component: 'useStorageState', action: 'fetchStorageState' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setStorageState(null);
    setError(null);
  }, []);

  return useMemo(
    () => ({
      storageState,
      loading,
      error,
      fetchStorageState,
      clear,
    }),
    [storageState, loading, error, fetchStorageState, clear]
  );
}
