import { useState, useCallback, useRef, useEffect } from 'react';
// Note: health endpoint returns free-form JSON, no proto schema
import { apiFetch, protoFetch } from '../../../shared/api/apiFetch';
import { parseSetMaintenanceStateResponse } from '../../../shared/api/proto-contracts';
import type { SystemHealthStatus } from './useSystemMonitor';

export interface UseHealthCheckReturn {
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  checkHealth: () => Promise<boolean>;
  refreshHealth: () => Promise<void>;
  toggleMonitoring: () => Promise<void>;
}

export const useHealthCheck = (): UseHealthCheckReturn => {
  const [healthStatus, setHealthStatus] = useState<SystemHealthStatus | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  const checkHealth = useCallback(async (): Promise<boolean> => {
    try {
      setHealthError(null);
      const data = await apiFetch<SystemHealthStatus>('/health', {
        headers: { 'Accept': 'application/json' }
      });
      if (!mountedRef.current) return false;
      setHealthStatus(data);
      return true;
    } catch (err) {
      if (!mountedRef.current) return false;
      const msg = err instanceof Error ? err.message : 'Unknown error';
      setHealthError(msg);
      return false;
    }
  }, []);

  const refreshHealth = useCallback(async () => {
    await checkHealth();
  }, [checkHealth]);

  const toggleMonitoring = useCallback(async () => {
    if (!healthStatus) {
      await checkHealth();
      return;
    }

    const isCurrentlyActive = healthStatus.processor_active ?? (healthStatus.maintenance_state === 'active');
    const nextState = isCurrentlyActive ? 'inactive' : 'active';

    try {
      const data = await protoFetch('/maintenance/state', parseSetMaintenanceStateResponse, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ maintenanceState: nextState })
      });

      if (!mountedRef.current) return;
      if (!data.success) {
        throw new Error(data.error ?? 'Failed to update status');
      }

      setHealthStatus(prev => prev ? {
        ...prev,
        processor_active: nextState === 'active',
        maintenance_state: nextState
      } : {
        processor_active: nextState === 'active',
        maintenance_state: nextState
      });

      // Refresh from server to confirm
      setTimeout(() => { void checkHealth(); }, 500);
    } catch (err) {
      if (!mountedRef.current) return;
      console.error('Failed to toggle system status:', err);
      setHealthError(err instanceof Error ? err.message : 'Failed to update status');
      void checkHealth();
    }
  }, [checkHealth, healthStatus]);

  return {
    healthStatus,
    healthError,
    checkHealth,
    refreshHealth,
    toggleMonitoring
  };
};
