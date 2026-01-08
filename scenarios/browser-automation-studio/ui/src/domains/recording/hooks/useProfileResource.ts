/**
 * Generic hook factory for profile-scoped API resources.
 *
 * This factory creates hooks that manage loading, error, and deleting states
 * for resources accessed via /recordings/sessions/{profileId}/{resource} endpoints.
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

export interface ProfileResourceState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
}

export interface ProfileResourceActions<T> {
  /** Fetch the resource data for a profile */
  fetch: (profileId: string) => Promise<void>;
  /** Clear data and reset state */
  clear: () => void;
  /** Delete all items (DELETE to base endpoint) */
  clearAll: (profileId: string) => Promise<boolean>;
  /** Helper to make delete requests, returns refetched data on success */
  deleteRequest: (profileId: string, subPath: string) => Promise<boolean>;
  /** Set the data directly (for custom transformations) */
  setData: React.Dispatch<React.SetStateAction<T | null>>;
  /** Set deleting state */
  setDeleting: React.Dispatch<React.SetStateAction<boolean>>;
  /** Set loading state */
  setLoading: React.Dispatch<React.SetStateAction<boolean>>;
  /** Set error state */
  setError: React.Dispatch<React.SetStateAction<string | null>>;
}

export interface ProfileResourceConfig<T, TRaw = T> {
  /** The resource endpoint path (e.g., 'storage', 'history', 'tabs') */
  endpoint: string;
  /** Component name for logging */
  componentName: string;
  /** Transform raw API response to data type (default: identity) */
  transform?: (raw: TRaw) => T;
  /** Initial data value (default: null) */
  initialData?: T | null;
  /** Error message for fetch failures */
  fetchErrorMessage?: string;
  /** Error message for clear all failures */
  clearAllErrorMessage?: string;
}

/**
 * Creates a hook for managing a profile-scoped API resource.
 *
 * @example
 * ```ts
 * const useMyResource = createProfileResourceHook<MyData>({
 *   endpoint: 'my-resource',
 *   componentName: 'useMyResource',
 * });
 *
 * // In component:
 * const { data, loading, error, fetch, clear, clearAll } = useMyResource();
 * ```
 */
export function createProfileResourceHook<T, TRaw = T>(
  config: ProfileResourceConfig<T, TRaw>
): () => ProfileResourceState<T> & ProfileResourceActions<T> {
  const {
    endpoint,
    componentName,
    transform = (raw: TRaw) => raw as unknown as T,
    initialData = null,
    fetchErrorMessage = `Failed to load ${endpoint}`,
    clearAllErrorMessage = `Failed to clear ${endpoint}`,
  } = config;

  return function useProfileResource(): ProfileResourceState<T> & ProfileResourceActions<T> {
    const [data, setData] = useState<T | null>(initialData);
    const [loading, setLoading] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const currentProfileId = useRef<string | null>(null);

    const fetchData = useCallback(async (profileId: string) => {
      currentProfileId.current = profileId;
      setLoading(true);
      setError(null);
      try {
        const cfg = await getConfig();
        const response = await fetch(`${cfg.API_URL}/recordings/sessions/${profileId}/${endpoint}`);
        if (!response.ok) {
          throw new Error(`${fetchErrorMessage} (${response.status})`);
        }
        const raw = (await response.json()) as TRaw;
        setData(transform(raw));
      } catch (err) {
        const message = err instanceof Error ? err.message : fetchErrorMessage;
        setError(message);
        logger.error(message, { component: componentName, action: 'fetch' }, err);
      } finally {
        setLoading(false);
      }
    }, []);

    const clear = useCallback(() => {
      setData(initialData);
      setError(null);
      currentProfileId.current = null;
    }, []);

    const deleteRequest = useCallback(
      async (profileId: string, subPath: string): Promise<boolean> => {
        setDeleting(true);
        setError(null);
        try {
          const cfg = await getConfig();
          const url = subPath
            ? `${cfg.API_URL}/recordings/sessions/${profileId}/${endpoint}/${subPath}`
            : `${cfg.API_URL}/recordings/sessions/${profileId}/${endpoint}`;
          const response = await fetch(url, { method: 'DELETE' });
          if (!response.ok) {
            throw new Error(`Delete failed (${response.status})`);
          }
          // Refetch to update UI
          await fetchData(profileId);
          return true;
        } catch (err) {
          const message = err instanceof Error ? err.message : 'Delete operation failed';
          setError(message);
          logger.error(message, { component: componentName, action: 'deleteRequest', subPath }, err);
          return false;
        } finally {
          setDeleting(false);
        }
      },
      [fetchData]
    );

    const clearAll = useCallback(
      async (profileId: string): Promise<boolean> => {
        setDeleting(true);
        setError(null);
        try {
          const cfg = await getConfig();
          const response = await fetch(`${cfg.API_URL}/recordings/sessions/${profileId}/${endpoint}`, {
            method: 'DELETE',
          });
          if (!response.ok) {
            throw new Error(`${clearAllErrorMessage} (${response.status})`);
          }
          // Refetch to update UI
          await fetchData(profileId);
          return true;
        } catch (err) {
          const message = err instanceof Error ? err.message : clearAllErrorMessage;
          setError(message);
          logger.error(message, { component: componentName, action: 'clearAll' }, err);
          return false;
        } finally {
          setDeleting(false);
        }
      },
      [fetchData]
    );

    return useMemo(
      () => ({
        data,
        loading,
        error,
        deleting,
        fetch: fetchData,
        clear,
        clearAll,
        deleteRequest,
        setData,
        setDeleting,
        setLoading,
        setError,
      }),
      [data, loading, error, deleting, fetchData, clear, clearAll, deleteRequest]
    );
  };
}
