import { useCallback, useMemo, useState } from 'react';
import { ConnectError } from '@connectrpc/connect';
import { recordingsClient } from '@/api/recordings';
import { logger } from '@/utils/logger';

export interface TabInfo {
  url: string;
  title?: string;
  isActive: boolean;
  order: number;
}

export interface TabsResponse {
  tabs: TabInfo[];
}

export interface UseTabsResult {
  tabs: TabInfo[];
  loading: boolean;
  error: string | null;
  deleting: boolean;
  fetchTabs: (profileId: string) => Promise<void>;
  clear: () => void;
  clearAllTabs: (profileId: string) => Promise<boolean>;
  deleteTab: (profileId: string, order: number) => Promise<boolean>;
}

const COMPONENT = 'useTabs';

function describe(err: unknown): string {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return 'Tab operation failed';
}

export function useTabs(): UseTabsResult {
  const [tabs, setTabs] = useState<TabInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTabs = useCallback(async (profileId: string) => {
    setLoading(true);
    setError(null);
    try {
      const resp = await recordingsClient.getSessionTabs({ profileId });
      setTabs((resp.tabs ?? []).map((t) => ({
        url: t.url,
        title: t.title || undefined,
        isActive: t.isActive,
        order: t.order,
      })));
    } catch (err) {
      const message = describe(err);
      setError(message);
      logger.error(message, { component: COMPONENT, action: 'fetch' }, err);
    } finally {
      setLoading(false);
    }
  }, []);

  const clear = useCallback(() => {
    setTabs([]);
    setError(null);
  }, []);

  const runMutation = useCallback(
    async (profileId: string, op: () => Promise<unknown>, action: string): Promise<boolean> => {
      setDeleting(true);
      setError(null);
      try {
        await op();
        await fetchTabs(profileId);
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
    [fetchTabs]
  );

  const clearAllTabs = useCallback(
    (profileId: string) =>
      runMutation(profileId, () => recordingsClient.clearSessionTabs({ profileId }), 'clearAll'),
    [runMutation]
  );

  const deleteTab = useCallback(
    (profileId: string, order: number) =>
      runMutation(profileId, () => recordingsClient.deleteSessionTab({ profileId, order }), 'delete'),
    [runMutation]
  );

  return useMemo(
    () => ({ tabs, loading, error, deleting, fetchTabs, clear, clearAllTabs, deleteTab }),
    [tabs, loading, error, deleting, fetchTabs, clear, clearAllTabs, deleteTab]
  );
}
