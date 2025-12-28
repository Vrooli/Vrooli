import { useCallback, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { StorageStateResponse } from '../types/types';

export interface UseStorageStateResult {
  storageState: StorageStateResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetchStorageState: (profileId: string) => Promise<void>;
  clear: () => void;
  // Delete operations
  clearAllStorage: (profileId: string) => Promise<boolean>;
  clearAllCookies: (profileId: string) => Promise<boolean>;
  deleteCookiesByDomain: (profileId: string, domain: string) => Promise<boolean>;
  deleteCookie: (profileId: string, domain: string, name: string) => Promise<boolean>;
  clearAllLocalStorage: (profileId: string) => Promise<boolean>;
  deleteLocalStorageByOrigin: (profileId: string, origin: string) => Promise<boolean>;
  deleteLocalStorageItem: (profileId: string, origin: string, name: string) => Promise<boolean>;
}

export function useStorageState(): UseStorageStateResult {
  const [storageState, setStorageState] = useState<StorageStateResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const currentProfileId = useRef<string | null>(null);

  const fetchStorageState = useCallback(async (profileId: string) => {
    currentProfileId.current = profileId;
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
    currentProfileId.current = null;
  }, []);

  // Helper for delete operations
  const deleteRequest = useCallback(async (profileId: string, path: string): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/storage${path}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Delete failed (${response.status})`);
      }
      // Refetch storage state to update UI
      await fetchStorageState(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Delete operation failed';
      setError(message);
      logger.error(message, { component: 'useStorageState', action: 'deleteRequest', path }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchStorageState]);

  const clearAllStorage = useCallback(async (profileId: string): Promise<boolean> => {
    return deleteRequest(profileId, '');
  }, [deleteRequest]);

  const clearAllCookies = useCallback(async (profileId: string): Promise<boolean> => {
    return deleteRequest(profileId, '/cookies');
  }, [deleteRequest]);

  const deleteCookiesByDomain = useCallback(async (profileId: string, domain: string): Promise<boolean> => {
    return deleteRequest(profileId, `/cookies/${encodeURIComponent(domain)}`);
  }, [deleteRequest]);

  const deleteCookie = useCallback(async (profileId: string, domain: string, name: string): Promise<boolean> => {
    return deleteRequest(profileId, `/cookies/${encodeURIComponent(domain)}/${encodeURIComponent(name)}`);
  }, [deleteRequest]);

  const clearAllLocalStorage = useCallback(async (profileId: string): Promise<boolean> => {
    return deleteRequest(profileId, '/origins');
  }, [deleteRequest]);

  const deleteLocalStorageByOrigin = useCallback(async (profileId: string, origin: string): Promise<boolean> => {
    return deleteRequest(profileId, `/origins/${encodeURIComponent(origin)}`);
  }, [deleteRequest]);

  const deleteLocalStorageItem = useCallback(async (profileId: string, origin: string, name: string): Promise<boolean> => {
    return deleteRequest(profileId, `/origins/${encodeURIComponent(origin)}/${encodeURIComponent(name)}`);
  }, [deleteRequest]);

  return useMemo(
    () => ({
      storageState,
      loading,
      error,
      deleting,
      fetchStorageState,
      clear,
      clearAllStorage,
      clearAllCookies,
      deleteCookiesByDomain,
      deleteCookie,
      clearAllLocalStorage,
      deleteLocalStorageByOrigin,
      deleteLocalStorageItem,
    }),
    [
      storageState,
      loading,
      error,
      deleting,
      fetchStorageState,
      clear,
      clearAllStorage,
      clearAllCookies,
      deleteCookiesByDomain,
      deleteCookie,
      clearAllLocalStorage,
      deleteLocalStorageByOrigin,
      deleteLocalStorageItem,
    ]
  );
}
