import { useCallback, useMemo, useState } from 'react';
import { ConnectError } from '@connectrpc/connect';
import { recordingsClient } from '@/api/recordings';
import { logger } from '@/utils/logger';

/**
 * Service worker info from the API.
 */
export interface ServiceWorkerInfo {
  registrationId: string;
  scopeURL: string;
  scriptURL: string;
  status: 'stopped' | 'running' | 'activating' | 'installed';
  versionId?: string;
}

/**
 * Domain override for service worker control.
 */
export interface ServiceWorkerDomainOverride {
  domain: string;
  mode: 'allow' | 'block';
}

/**
 * Service worker control settings.
 */
export interface ServiceWorkerControl {
  mode: 'allow' | 'block' | 'block-on-domain' | 'unregister-all';
  domainOverrides?: ServiceWorkerDomainOverride[];
  blockedDomains?: string[];
}

/**
 * Service workers response from the API.
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
  unregisterAll: (profileId: string) => Promise<boolean>;
  unregisterWorker: (profileId: string, scopeURL: string) => Promise<boolean>;
}

const COMPONENT = 'useServiceWorkers';

function describe(err: unknown): string {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return 'Service worker operation failed';
}

export function useServiceWorkers(): UseServiceWorkersResult {
  const [serviceWorkers, setServiceWorkers] = useState<ServiceWorkersResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchServiceWorkers = useCallback(async (profileId: string) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await recordingsClient.getServiceWorkers({ profileId });
      const workers = (resp.workers ?? []).map((w) => ({
        registrationId: w.registrationId,
        scopeURL: w.scopeUrl,
        scriptURL: w.scriptUrl,
        status: (w.status || 'stopped') as ServiceWorkerInfo['status'],
        versionId: w.versionId || undefined,
      }));
      const control: ServiceWorkerControl = {
        mode: (resp.control?.mode || 'allow') as ServiceWorkerControl['mode'],
        domainOverrides: (resp.control?.domainOverrides ?? []).map((d) => ({
          domain: d.domain,
          mode: (d.mode || 'allow') as 'allow' | 'block',
        })),
        blockedDomains: resp.control?.blockedDomains ?? [],
      };
      setServiceWorkers({
        session_id: resp.sessionId,
        workers,
        control,
        message: resp.message || undefined,
      });
    } catch (err) {
      const message = describe(err);
      setError(message);
      logger.error(message, { component: COMPONENT, action: 'fetchServiceWorkers' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setServiceWorkers(null);
    setError(null);
  }, []);

  const runMutation = useCallback(
    async (profileId: string, op: () => Promise<unknown>, action: string): Promise<boolean> => {
      setDeleting(true);
      setError(null);
      try {
        await op();
        await fetchServiceWorkers(profileId);
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
    [fetchServiceWorkers]
  );

  const unregisterAll = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearAllServiceWorkers({ profileId }), 'unregisterAll'),
    [runMutation]
  );

  const unregisterWorker = useCallback(
    (profileId: string, scopeURL: string) =>
      runMutation(profileId, () => recordingsClient.deleteServiceWorker({ profileId, scopeUrl: scopeURL }), 'unregisterWorker'),
    [runMutation]
  );

  return useMemo(
    () => ({ serviceWorkers, loading, error, deleting, fetchServiceWorkers, clear, unregisterAll, unregisterWorker }),
    [serviceWorkers, loading, error, deleting, fetchServiceWorkers, clear, unregisterAll, unregisterWorker]
  );
}
