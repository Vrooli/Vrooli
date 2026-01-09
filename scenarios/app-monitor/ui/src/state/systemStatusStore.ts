import { useEffect } from 'react';
import { create } from 'zustand';
import { healthService } from '@/services/api';

interface SystemStatusState {
  status: string | null;
  uptimeSeconds: number | null;
  appCount: number;
  resourceCount: number;
  loading: boolean;
  error: string | null;
  lastChecked: Date | null;
  refresh: () => Promise<void>;
}

export const useSystemStatusStore = create<SystemStatusState>((set, get) => ({
  status: null,
  uptimeSeconds: null,
  appCount: 0,
  resourceCount: 0,
  loading: false,
  error: null,
  lastChecked: null,

  refresh: async (): Promise<void> => {
    if (get().loading) {
      return;
    }

    // Set loading flag but preserve previous data so UI stays visible
    set({ loading: true });

    try {
      // Use new lightweight endpoint that returns just counts + status
      const result = await healthService.getSystemStatus();

      if (result) {
        set({
          status: result.status || 'unknown',
          uptimeSeconds: result.uptime_seconds ?? null,
          appCount: result.app_count || 0,
          resourceCount: result.resource_count || 0,
          lastChecked: new Date(),
          error: null, // Clear error only on success
        });
      } else {
        set({
          status: 'unhealthy',
          uptimeSeconds: null,
          lastChecked: new Date(),
        });
      }
    } catch (error) {
      set({
        error: 'Unable to refresh system status.',
        lastChecked: new Date(),
      });
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
