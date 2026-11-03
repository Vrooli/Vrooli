import { useState, useCallback, useRef, useEffect } from 'react';
// Note: health endpoint returns free-form JSON, no proto schema
import { apiFetch, protoFetch } from '../../../shared/api/apiFetch';
import type { SystemHealthStatus } from './useSystemMonitor';
import { isRecord } from '../../../shared/utils/typeGuards';

export function parseSystemHealthStatus(value: unknown): SystemHealthStatus {
  if (!isRecord(value)) throw new TypeError('health response must be an object');
  const status = value['status'];
  const service = value['service'];
  const timestamp = value['timestamp'];
  const uptime = value['uptime'];
  const processorActive = value['processor_active'];
  const maintenanceState = value['maintenance_state'];
  if (status !== undefined && typeof status !== 'string') throw new TypeError('health.status must be a string');
  if (service !== undefined && typeof service !== 'string') throw new TypeError('health.service must be a string');
  if (timestamp !== undefined && typeof timestamp !== 'string' && typeof timestamp !== 'number') throw new TypeError('health.timestamp must be text or a number');
  if (uptime !== undefined && (typeof uptime !== 'number' || !Number.isFinite(uptime))) throw new TypeError('health.uptime must be finite');
  if (processorActive !== undefined && typeof processorActive !== 'boolean') throw new TypeError('health.processor_active must be boolean');
  if (maintenanceState !== undefined && typeof maintenanceState !== 'string') throw new TypeError('health.maintenance_state must be a string');

  let apiConnectivity: SystemHealthStatus['api_connectivity'];
  const connectivity = value['api_connectivity'];
  if (connectivity !== undefined) {
    if (!isRecord(connectivity)) throw new TypeError('health.api_connectivity must be an object');
    const connected = connectivity['connected'];
    const latency = connectivity['latency_ms'];
    const error = connectivity['error'];
    if (connected !== undefined && typeof connected !== 'boolean') throw new TypeError('health.api_connectivity.connected must be boolean');
    if (latency !== undefined && (typeof latency !== 'number' || !Number.isFinite(latency))) throw new TypeError('health.api_connectivity.latency_ms must be finite');
    if (error !== undefined && error !== null && !isRecord(error)) throw new TypeError('health.api_connectivity.error must be an object');
    if (isRecord(error)) {
      if (error['code'] !== undefined && typeof error['code'] !== 'string') throw new TypeError('health API error code must be a string');
      if (error['message'] !== undefined && typeof error['message'] !== 'string') throw new TypeError('health API error message must be a string');
    }
    apiConnectivity = {
      connected: typeof connected === 'boolean' ? connected : undefined,
      latency_ms: typeof latency === 'number' ? latency : undefined,
      error: isRecord(error) ? {
        code: typeof error['code'] === 'string' ? error['code'] : undefined,
        message: typeof error['message'] === 'string' ? error['message'] : undefined,
      } : error === null ? null : undefined,
    };
  }

  const checks = value['checks'];
  if (checks !== undefined && !isRecord(checks)) throw new TypeError('health.checks must be an object');
  return {
    status: typeof status === 'string' ? status : undefined,
    service: typeof service === 'string' ? service : undefined,
    timestamp: typeof timestamp === 'string' || typeof timestamp === 'number' ? timestamp : undefined,
    uptime: typeof uptime === 'number' ? uptime : undefined,
    processor_active: typeof processorActive === 'boolean' ? processorActive : undefined,
    maintenance_state: typeof maintenanceState === 'string' ? maintenanceState : undefined,
    api_connectivity: apiConnectivity,
    checks: isRecord(checks) ? checks : undefined,
  };
}

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
      const data = await apiFetch('/health', parseSystemHealthStatus, {
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
      const { parseSetMaintenanceStateResponse } = await import('../../../shared/api/proto-contracts');
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
