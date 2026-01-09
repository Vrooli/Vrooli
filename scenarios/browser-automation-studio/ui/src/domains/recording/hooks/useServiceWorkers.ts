import { useCallback, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

/**
 * Service worker info from the API
 */
export interface ServiceWorkerInfo {
  registrationId: string;
  scopeURL: string;
  scriptURL: string;
  status: 'stopped' | 'running' | 'activating' | 'installed';
  versionId?: string;
}

/**
 * Domain override for service worker control
 */
export interface ServiceWorkerDomainOverride {
  domain: string;
  mode: 'allow' | 'block';
}

/**
 * Service worker control settings
 */
export interface ServiceWorkerControl {
  mode: 'allow' | 'block' | 'block-on-domain' | 'unregister-all';
  domainOverrides?: ServiceWorkerDomainOverride[];
  blockedDomains?: string[];
}

/**
 * Service workers response from the API
 */
export interface ServiceWorkersResponse {
  session_id: string;
  workers: ServiceWorkerInfo[];
  control: ServiceWorkerControl;
  message?: string;
}

export interface UseServiceWorkersResult {
  serviceWorkers: ServiceWorkersResponse | null;
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetchServiceWorkers: (profileId: string) => Promise<void>;
  clear: () => void;
  // Delete operations
  unregisterAll: (profileId: string) => Promise<boolean>;
  unregisterWorker: (profileId: string, scopeURL: string) => Promise<boolean>;
}

export function useServiceWorkers(): UseServiceWorkersResult {
  const [serviceWorkers, setServiceWorkers] = useState<ServiceWorkersResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const currentProfileId = useRef<string | null>(null);

  const fetchServiceWorkers = useCallback(async (profileId: string) => {
    currentProfileId.current = profileId;
    setLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/service-workers`);
      if (!response.ok) {
        throw new Error(`Failed to fetch service workers (${response.status})`);
      }
      const data = (await response.json()) as ServiceWorkersResponse;
      setServiceWorkers(data);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load service workers';
      setError(message);
      logger.error(message, { component: 'useServiceWorkers', action: 'fetchServiceWorkers' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setServiceWorkers(null);
    setError(null);
    currentProfileId.current = null;
  }, []);

  // Helper for delete operations
  const deleteRequest = useCallback(async (profileId: string, path: string): Promise<boolean> => {
    setDeleting(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/sessions/${profileId}/service-workers${path}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Delete failed (${response.status})`);
      }
      // Refetch service workers to update UI
      await fetchServiceWorkers(profileId);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Delete operation failed';
      setError(message);
      logger.error(message, { component: 'useServiceWorkers', action: 'deleteRequest', path }, err);
      return false;
    } finally {
      setDeleting(false);
    }
  }, [fetchServiceWorkers]);

  const unregisterAll = useCallback(async (profileId: string): Promise<boolean> => {
    return deleteRequest(profileId, '');
  }, [deleteRequest]);

  const unregisterWorker = useCallback(async (profileId: string, scopeURL: string): Promise<boolean> => {
    return deleteRequest(profileId, `/${encodeURIComponent(scopeURL)}`);
  }, [deleteRequest]);

  return useMemo(
    () => ({
      serviceWorkers,
      loading,
      error,
      deleting,
      fetchServiceWorkers,
      clear,
      unregisterAll,
      unregisterWorker,
    }),
    [
      serviceWorkers,
      loading,
      error,
      deleting,
      fetchServiceWorkers,
      clear,
      unregisterAll,
      unregisterWorker,
    ]
  );
}
