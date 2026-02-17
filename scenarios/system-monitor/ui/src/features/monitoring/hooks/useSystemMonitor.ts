import { useState, useEffect, useCallback, useRef } from 'react';
import { buildApiUrl } from '../../../shared/api/apiBase';
import { apiFetch, toApiError } from '../../../shared/api/apiFetch';
import { usePolling } from '../../../shared/hooks/usePolling';
import { useHealthCheck } from './useHealthCheck';
import { useMetricHistory } from './useMetricHistory';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  InfrastructureMonitorData,
  Investigation,
  APIError
} from '../../../types';

export interface SystemHealthStatus {
  status?: string;
  service?: string;
  timestamp?: number | string;
  uptime?: number;
  processor_active?: boolean;
  maintenance_state?: string;
  api_connectivity?: {
    connected?: boolean;
    latency_ms?: number;
    error?: { code?: string; message?: string } | null;
  };
  checks?: Record<string, unknown>;
}

interface UseSystemMonitorReturn {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  infrastructureData: InfrastructureMonitorData | null;
  investigations: Investigation[];
  metricHistory: import('../../../types').MetricHistory | null;
  isLoading: boolean;
  error: APIError | null;
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  toggleMonitoring: () => Promise<void>;
  refreshHealth: () => Promise<void>;
  refresh: () => void;
  refreshMetrics: () => void;
}

type MaintenanceState = 'active' | 'inactive' | string;

interface MaintenanceStateResponse {
  success?: boolean;
  maintenanceState?: MaintenanceState;
  maintenance_state?: MaintenanceState;
  error?: string;
}

export const useSystemMonitor = (): UseSystemMonitorReturn => {
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [detailedMetrics, setDetailedMetrics] = useState<DetailedMetrics | null>(null);
  const [processMonitorData, setProcessMonitorData] = useState<ProcessMonitorData | null>(null);
  const [infrastructureData, setInfrastructureData] = useState<InfrastructureMonitorData | null>(null);
  const [investigations, setInvestigations] = useState<Investigation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<APIError | null>(null);
  const [uiBoostActive, setUiBoostActive] = useState(false);
  const mountedRef = useRef(true);
  const maintenanceStateRef = useRef<{ previous: MaintenanceState | null; activated: boolean }>({
    previous: null,
    activated: false
  });

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  const { healthStatus, healthError, checkHealth, refreshHealth, toggleMonitoring } = useHealthCheck();
  const { metricHistory, fetchMetricsTimeline, appendGpuPoint, appendDiskPoints, appendDiskUsagePoint } = useMetricHistory(setError);

  const handleApiCall = useCallback(async <T,>(url: string): Promise<T | null> => {
    try {
      return await apiFetch<T>(url);
    } catch (err) {
      console.error(`API call failed for ${url}:`, err);
      if (!mountedRef.current) return null;
      setError(toApiError(err));
      return null;
    }
  }, []);

  const fetchMetrics = useCallback(async () => {
    const url = uiBoostActive ? '/metrics/current?fresh=1' : '/metrics/current';
    const data = await handleApiCall<MetricsResponse>(url);
    if (data) {
      setMetrics(data);
      setError(null);
      if (typeof data.gpu_usage === 'number' && Number.isFinite(data.gpu_usage)) {
        const timestamp = data.timestamp ?? new Date().toISOString();
        appendGpuPoint(timestamp, data.gpu_usage as number);
      }
    }
  }, [handleApiCall, uiBoostActive, appendGpuPoint]);

  useEffect(() => {
    let cancelled = false;

    const activateMonitoring = async () => {
      const state = await handleApiCall<MaintenanceStateResponse>('/maintenance/state');
      if (cancelled || !state) {
        return;
      }

      const currentState = state.maintenanceState ?? state.maintenance_state ?? 'inactive';
      maintenanceStateRef.current.previous = currentState;

      if (currentState !== 'active') {
        try {
          const response = await fetch(buildApiUrl('/maintenance/state'), {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json'
            },
            body: JSON.stringify({ maintenanceState: 'active' })
          });
          if (response.ok) {
            maintenanceStateRef.current.activated = true;
          }
        } catch (postError) {
          console.warn('Failed to activate monitoring:', postError);
        }
      }

      if (!cancelled) {
        setUiBoostActive(true);
      }
    };

    void activateMonitoring();

    const stateRef = maintenanceStateRef.current;

    return () => {
      cancelled = true;
      const previous = stateRef.previous;
      if (stateRef.activated && previous && previous !== 'active') {
        fetch(buildApiUrl('/maintenance/state'), {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ maintenanceState: previous })
        }).catch(unmountError => {
          console.warn('Failed to restore monitoring state:', unmountError);
        });
      }
    };
  }, [handleApiCall]);

  const fetchDetailedMetrics = useCallback(async () => {
    const data = await handleApiCall<DetailedMetrics>('/metrics/detailed');
    if (data) {
      setDetailedMetrics(data);
      const diskPercent = data.memory_details?.disk_usage?.percent;
      if (typeof diskPercent === 'number' && Number.isFinite(diskPercent)) {
        const timestamp = data.timestamp ?? new Date().toISOString();
        appendDiskUsagePoint(timestamp, diskPercent);
      }
    }
  }, [handleApiCall, appendDiskUsagePoint]);

  const fetchProcessMonitorData = useCallback(async () => {
    const data = await handleApiCall<ProcessMonitorData>('/metrics/processes');
    if (data) {
      setProcessMonitorData(data);
    }
  }, [handleApiCall]);

  const fetchInfrastructureData = useCallback(async () => {
    const data = await handleApiCall<InfrastructureMonitorData>('/metrics/infrastructure');
    if (data) {
      setInfrastructureData(data);
      const { storage_io, timestamp } = data;
      if (storage_io) {
        const recordTimestamp = timestamp ?? new Date().toISOString();
        const readRate = Number(storage_io.read_mb_per_sec);
        const writeRate = Number(storage_io.write_mb_per_sec);
        if (Number.isFinite(readRate) || Number.isFinite(writeRate)) {
          appendDiskPoints(recordTimestamp, readRate, writeRate);
        }
      }
    }
  }, [handleApiCall, appendDiskPoints]);

  const fetchInvestigations = useCallback(async () => {
    const data = await handleApiCall<Investigation[]>('/investigations?limit=10');
    if (Array.isArray(data)) {
      const sorted = [...data].sort((a, b) => {
        const aTime = Date.parse(a.start_time ?? a.timestamp ?? '');
        const bTime = Date.parse(b.start_time ?? b.timestamp ?? '');
        return isNaN(bTime) && isNaN(aTime)
          ? 0
          : isNaN(bTime)
          ? -1
          : isNaN(aTime)
          ? 1
          : bTime - aTime;
      });
      setInvestigations(sorted);
    }
  }, [handleApiCall]);

  const refreshMetrics = useCallback(async () => {
    await Promise.all([
      fetchMetrics(),
      fetchDetailedMetrics(),
      fetchMetricsTimeline()
    ]);
  }, [fetchDetailedMetrics, fetchMetrics, fetchMetricsTimeline]);

  const refresh = useCallback(async () => {
    setIsLoading(true);

    const isHealthy = await checkHealth();
    if (!mountedRef.current) return;
    if (!isHealthy) {
      setError({
        error: 'API server is not responding',
        details: 'Health check failed - ensure the Go backend is running',
        timestamp: new Date().toISOString()
      });
      setIsLoading(false);
      return;
    }

    await Promise.all([
      fetchMetrics(),
      fetchDetailedMetrics(),
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations(),
      fetchMetricsTimeline(120)
    ]);

    if (!mountedRef.current) return;
    setIsLoading(false);
  }, [
    checkHealth,
    fetchMetricsTimeline,
    fetchMetrics,
    fetchDetailedMetrics,
    fetchProcessMonitorData,
    fetchInfrastructureData,
    fetchInvestigations
  ]);

  // Initial load
  useEffect(() => {
    refresh();
  }, [refresh]);

  // Set up polling for metrics (every 5 seconds for responsive graphs)
  usePolling(refreshMetrics, 5000);

  // Set up polling for detailed data + health (every 60 seconds)
  const fetchDetailedAll = useCallback(() => {
    void Promise.all([
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations(),
      checkHealth()
    ]);
  }, [fetchProcessMonitorData, fetchInfrastructureData, fetchInvestigations, checkHealth]);
  usePolling(fetchDetailedAll, 60000);

  return {
    metrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    metricHistory,
    isLoading,
    error,
    healthStatus,
    healthError,
    toggleMonitoring,
    refreshHealth,
    refresh,
    refreshMetrics
  };
};
