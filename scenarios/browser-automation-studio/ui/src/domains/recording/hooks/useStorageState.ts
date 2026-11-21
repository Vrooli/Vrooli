import { useCallback, useMemo, useState } from 'react';
import { ConnectError } from '@connectrpc/connect';
import { recordingsClient } from '@/api/recordings';
import { logger } from '@/utils/logger';
import type { StorageStateResponse } from '../types/types';

export interface UseStorageStateResult {
  storageState: StorageStateResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetchStorageState: (profileId: string) => Promise<void>;
  clear: () => void;
  clearAllStorage: (profileId: string) => Promise<boolean>;
  clearAllCookies: (profileId: string) => Promise<boolean>;
  deleteCookiesByDomain: (profileId: string, domain: string) => Promise<boolean>;
  deleteCookie: (profileId: string, domain: string, name: string) => Promise<boolean>;
  clearAllLocalStorage: (profileId: string) => Promise<boolean>;
  deleteLocalStorageByOrigin: (profileId: string, origin: string) => Promise<boolean>;
  deleteLocalStorageItem: (profileId: string, origin: string, name: string) => Promise<boolean>;
}

function describe(err: unknown): string {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return 'Storage operation failed';
}

const COMPONENT = 'useStorageState';

export function useStorageState(): UseStorageStateResult {
  const [storageState, setStorageState] = useState<StorageStateResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStorageState = useCallback(async (profileId: string) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await recordingsClient.getStorageState({ profileId });
      const cookies = (resp.cookies ?? []).map((c) => ({
        name: c.name,
        value: c.value,
        valueMasked: c.valueMasked,
        domain: c.domain,
        path: c.path,
        expires: c.expires,
        httpOnly: c.httpOnly,
        secure: c.secure,
        sameSite: (c.sameSite || 'None') as 'Strict' | 'Lax' | 'None',
      }));
      const origins = (resp.origins ?? []).map((o) => ({
        origin: o.origin,
        localStorage: (o.localStorage ?? []).map((i) => ({ name: i.name, value: i.value })),
      }));
      setStorageState({
        cookies,
        origins,
        stats: {
          cookieCount: resp.stats?.cookieCount ?? 0,
          localStorageCount: resp.stats?.localStorageCount ?? 0,
          originCount: resp.stats?.originCount ?? 0,
        },
      });
    } catch (err) {
      const message = describe(err);
      setError(message);
      logger.error(message, { component: COMPONENT, action: 'fetch' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setStorageState(null);
    setError(null);
  }, []);

  const runMutation = useCallback(
    async (
      profileId: string,
      op: () => Promise<unknown>,
      action: string
    ): Promise<boolean> => {
      setDeleting(true);
      setError(null);
      try {
        await op();
        await fetchStorageState(profileId);
        return true;
      } catch (err) {
        const message = describe(err);
        setError(message);
        logger.error(message, { component: COMPONENT, action }, err);
        return false;
      } finally {
        setDeleting(false);
      }
    },
    [fetchStorageState]
  );

  const clearAllStorage = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearAllStorage({ profileId }), 'clearAllStorage'),
    [runMutation]
  );

  const clearAllCookies = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearAllCookies({ profileId }), 'clearAllCookies'),
    [runMutation]
  );

  const deleteCookiesByDomain = useCallback(
    (profileId: string, domain: string) =>
      runMutation(profileId, () => recordingsClient.deleteCookiesByDomain({ profileId, domain }), 'deleteCookiesByDomain'),
    [runMutation]
  );

  const deleteCookie = useCallback(
    (profileId: string, domain: string, name: string) =>
      runMutation(profileId, () => recordingsClient.deleteCookie({ profileId, domain, name }), 'deleteCookie'),
    [runMutation]
  );

  const clearAllLocalStorage = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearAllLocalStorage({ profileId }), 'clearAllLocalStorage'),
    [runMutation]
  );

  const deleteLocalStorageByOrigin = useCallback(
    (profileId: string, origin: string) =>
      runMutation(profileId, () => recordingsClient.deleteLocalStorageByOrigin({ profileId, origin }), 'deleteLocalStorageByOrigin'),
    [runMutation]
  );

  const deleteLocalStorageItem = useCallback(
    (profileId: string, origin: string, name: string) =>
      runMutation(profileId, () => recordingsClient.deleteLocalStorageItem({ profileId, origin, name }), 'deleteLocalStorageItem'),
    [runMutation]
  );

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
