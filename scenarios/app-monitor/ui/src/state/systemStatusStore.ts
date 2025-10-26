import { useEffect } from 'react';
import { create } from 'zustand';
import { appService, resourceService, healthService } from '@/services/api';

interface SystemStatusState {
  status: string | null;
  uptimeSeconds: number | null;
  appCount: number;
  resourceCount: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

export const useSystemStatusStore = create<SystemStatusState>((set, get) => ({
  status: null,
  uptimeSeconds: null,
  appCount: 0,
  resourceCount: 0,
  loading: false,
  error: null,

  refresh: async (): Promise<void> => {
    if (get().loading) {
      return;
    }

    set({ loading: true, error: null });

    try {
      const [appsResult, resourcesResult, healthResult] = await Promise.allSettled([
        appService.getApps(),
        resourceService.getResources(),
        healthService.checkHealth(),
      ]);

      if (appsResult.status === 'fulfilled') {
        set({ appCount: appsResult.value.length });
      }

      if (resourcesResult.status === 'fulfilled') {
        set({ resourceCount: resourcesResult.value.length });
      }

      if (healthResult.status === 'fulfilled' && healthResult.value) {
        const nextStatus = typeof healthResult.value.status === 'string'
          ? healthResult.value.status.toLowerCase()
          : null;
        const uptime = typeof healthResult.value.metrics?.uptime_seconds === 'number'
          ? healthResult.value.metrics.uptime_seconds
          : null;
        set({ status: nextStatus, uptimeSeconds: nextStatus === 'unhealthy' ? null : uptime });
      } else if (healthResult.status === 'rejected') {
        set({ status: 'unhealthy', uptimeSeconds: null });
      }
    } catch (error) {
      set({ error: 'Unable to refresh system status.', status: 'unhealthy', uptimeSeconds: null });
    } finally {
      set({ loading: false });
    }
  },
}));

let subscriberCount = 0;
let pollTimer: number | null = null;

export const useSystemStatus = () => {
  const state = useSystemStatusStore();
  const refresh = useSystemStatusStore(current => current.refresh);

  useEffect(() => {
    subscriberCount += 1;
    void refresh();

    if (typeof window !== 'undefined' && pollTimer === null) {
      pollTimer = window.setInterval(() => {
        void refresh();
      }, 60000);
    }

    return () => {
      subscriberCount -= 1;
      if (subscriberCount <= 0 && pollTimer !== null && typeof window !== 'undefined') {
        window.clearInterval(pollTimer);
        pollTimer = null;
      }
    };
  }, [refresh]);

  return state;
};
