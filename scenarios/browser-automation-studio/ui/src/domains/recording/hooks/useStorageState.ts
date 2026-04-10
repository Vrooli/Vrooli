import { useCallback, useMemo } from 'react';
import { createProfileResourceHook } from './useProfileResource';
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

// Create base hook using factory
const useStorageStateBase = createProfileResourceHook<StorageStateResponse>({
  endpoint: 'storage',
  componentName: 'useStorageState',
  fetchErrorMessage: 'Failed to load storage state',
  clearAllErrorMessage: 'Failed to clear storage',
});

export function useStorageState(): UseStorageStateResult {
  const base = useStorageStateBase();
  const {
    data,
    loading,
    error,
    deleting,
    fetch,
    clear,
    clearAll,
    deleteRequest,
  } = base;

  // Cookie operations
  const clearAllCookies = useCallback(
    async (profileId: string): Promise<boolean> => {
      return deleteRequest(profileId, 'cookies');
    },
    [deleteRequest]
  );

  const deleteCookiesByDomain = useCallback(
    async (profileId: string, domain: string): Promise<boolean> => {
      return deleteRequest(profileId, `cookies/${encodeURIComponent(domain)}`);
    },
    [deleteRequest]
  );

  const deleteCookie = useCallback(
    async (profileId: string, domain: string, name: string): Promise<boolean> => {
      return deleteRequest(profileId, `cookies/${encodeURIComponent(domain)}/${encodeURIComponent(name)}`);
    },
    [deleteRequest]
  );

  // LocalStorage operations
  const clearAllLocalStorage = useCallback(
    async (profileId: string): Promise<boolean> => {
      return deleteRequest(profileId, 'origins');
    },
    [deleteRequest]
  );

  const deleteLocalStorageByOrigin = useCallback(
    async (profileId: string, origin: string): Promise<boolean> => {
      return deleteRequest(profileId, `origins/${encodeURIComponent(origin)}`);
    },
    [deleteRequest]
  );

  const deleteLocalStorageItem = useCallback(
    async (profileId: string, origin: string, name: string): Promise<boolean> => {
      return deleteRequest(profileId, `origins/${encodeURIComponent(origin)}/${encodeURIComponent(name)}`);
    },
    [deleteRequest]
  );

  return useMemo(
    () => ({
      storageState: data,
      loading,
      error,
      deleting,
      fetchStorageState: fetch,
      clear,
      clearAllStorage: clearAll,
      clearAllCookies,
      deleteCookiesByDomain,
      deleteCookie,
      clearAllLocalStorage,
      deleteLocalStorageByOrigin,
      deleteLocalStorageItem,
    }),
    [
      data,
      loading,
      error,
      deleting,
      fetch,
      clear,
      clearAll,
      clearAllCookies,
      deleteCookiesByDomain,
      deleteCookie,
      clearAllLocalStorage,
      deleteLocalStorageByOrigin,
      deleteLocalStorageItem,
    ]
  );
}
