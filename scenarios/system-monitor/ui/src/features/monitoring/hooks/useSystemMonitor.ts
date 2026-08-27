// DOC: docs/internal/COHERENCE-NOTES.md#state-architecture
import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { timestampMs } from '@bufbuild/protobuf/wkt';
import { buildUrl as buildApiUrl } from '../../../lib/api-client';
import { protoFetch, toApiError } from '../../../shared/api/apiFetch';
import { parseMetricsResponse } from '../../../shared/api/current-metrics-contract';
import { useToast } from '../../../shared/components/ToastProvider';
import { toIsoString } from '../../../shared/utils/timestamps';
import { usePolling } from '../../../shared/hooks/usePolling';
import { useHealthCheck } from './useHealthCheck';
import { useMetricHistory } from './useMetricHistory';
import type {
  MetricsResponse,
  DeviceGraph,
  DetailedMetrics,
  ProcessMonitorData,
  InfrastructureMonitorData,
  Investigation,
  APIError
} from '../../../types';

// TODO: proto-back when health/maintenance protos exist
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

export type Subsystem = 'metrics' | 'detailedMetrics' | 'deviceGraph' | 'processes' | 'infrastructure' | 'investigations';

interface UseSystemMonitorReturn {
  metrics: MetricsResponse | null;
  deviceGraph: DeviceGraph | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  infrastructureData: InfrastructureMonitorData | null;
  investigations: Investigation[];
  metricHistory: import('../../../types').MetricHistory | null;
  isLoading: boolean;
  error: APIError | null;
  subsystemErrors: Partial<Record<Subsystem, APIError>>;
  isStale: boolean;
  retryAttempt: number;
  lastSuccessfulFetch: Date | null;
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  toggleMonitoring: () => Promise<void>;
  refreshHealth: () => Promise<void>;
  refresh: () => void;
  refreshMetrics: () => void;
}

type MaintenanceState = 'active' | 'inactive' | string;

interface UseSystemMonitorOptions {
  enabled?: boolean;
  node?: string;
}

export const useSystemMonitor = (historyWindowSeconds = 120, options: UseSystemMonitorOptions = {}): UseSystemMonitorReturn => {
  const pollingEnabled = options.enabled ?? true;
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [deviceGraph, setDeviceGraph] = useState<DeviceGraph | null>(null);
  const [detailedMetrics, setDetailedMetrics] = useState<DetailedMetrics | null>(null);
  const [processMonitorData, setProcessMonitorData] = useState<ProcessMonitorData | null>(null);
  const [infrastructureData, setInfrastructureData] = useState<InfrastructureMonitorData | null>(null);
  const [investigations, setInvestigations] = useState<Investigation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [subsystemErrors, setSubsystemErrors] = useState<Partial<Record<Subsystem, APIError>>>({});
  const [lastSuccessfulFetch, setLastSuccessfulFetch] = useState<Date | null>(null);
  const [consecutiveFailures, setConsecutiveFailures] = useState(0);
  const isStale = consecutiveFailures >= 3;
  const [uiBoostActive, setUiBoostActive] = useState(false);
  const mountedRef = useRef(true);
  const abortRef = useRef<AbortController | null>(null);
  const maintenanceStateRef = useRef<{ previous: MaintenanceState | null; activated: boolean }>({
    previous: null,
    activated: false
  });
  const { showApiError } = useToast();
  const lastMetricsErrorRef = useRef<string | null>(null);

  const setSubsystemError = useCallback((subsystem: Subsystem, err: APIError | null) => {
    setSubsystemErrors(prev => {
      if (err === null) {
        if (!(subsystem in prev)) return prev;
        const { [subsystem]: _omit, ...rest } = prev;
        return rest;
      }
      return { ...prev, [subsystem]: err };
    });
  }, []);

  const error = useMemo(() => {
    const values = Object.values(subsystemErrors);
    return values.length > 0 ? values[0] ?? null : null;
  }, [subsystemErrors]);

  useEffect(() => {
    const abort = abortRef.current;
    return () => {
      mountedRef.current = false;
      abort?.abort();
    };
  }, []);

  // Only show toast for metrics errors (primary data source)
  useEffect(() => {
    const metricsError = subsystemErrors.metrics;
    if (metricsError && metricsError.error !== lastMetricsErrorRef.current) {
      lastMetricsErrorRef.current = metricsError.error;
      showApiError(metricsError);
    } else if (!metricsError) {
      lastMetricsErrorRef.current = null;
    }
  }, [subsystemErrors, showApiError]);

  const { healthStatus, healthError, checkHealth, refreshHealth, toggleMonitoring } = useHealthCheck();
  const setMetricsError = useCallback((err: APIError | null) => { setSubsystemError('metrics', err); }, [setSubsystemError]);
  const { metricHistory, fetchMetricsTimeline, clearHistory, appendGpuPoint, appendDiskPoints, appendDiskUsagePoint } = useMetricHistory(setMetricsError);
  const remoteNodeSelected = Boolean(options.node);

  const fetchMetrics = useCallback(async () => {
    const params = new URLSearchParams();
    if (uiBoostActive) params.set('fresh', '1');
    if (options.node) params.set('node', options.node);
    const query = params.toString();
    const url = `/metrics/current${query ? `?${query}` : ''}`;
    try {
      const data = await protoFetch(url, parseMetricsResponse);
      if (!mountedRef.current) return;
      setMetrics(data);
      setSubsystemError('metrics', null);
      setLastSuccessfulFetch(new Date());
      setConsecutiveFailures(0);
      if (typeof data.gpuUsage === 'number' && Number.isFinite(data.gpuUsage)) {
        appendGpuPoint(toIsoString(data.timestamp), data.gpuUsage);
      }
    } catch (err) {
      console.debug(`API call failed for ${url}:`, err);
      if (!mountedRef.current) return;
      setSubsystemError('metrics', toApiError(err));
      setConsecutiveFailures(prev => prev + 1);
    }
  }, [options.node, uiBoostActive, appendGpuPoint, setSubsystemError]);

  const fetchDeviceGraph = useCallback(async () => {
    try {
      const { parseDeviceGraph } = await import('../../../shared/api/current-metrics-contract');
      const query = options.node ? `?node=${encodeURIComponent(options.node)}` : '';
      const data = await protoFetch(`/metrics/devices${query}`, parseDeviceGraph);
      if (!mountedRef.current) return;
      setDeviceGraph(data);
      setSubsystemError('deviceGraph', null);
    } catch (err) {
      console.debug('API call failed for /metrics/devices:', err);
      if (!mountedRef.current) return;
      setSubsystemError('deviceGraph', toApiError(err));
    }
  }, [options.node, setSubsystemError]);

  useEffect(() => {
    let cancelled = false;

    const activateMonitoring = async () => {
      const {
        parseGetMaintenanceStateResponse,
        parseSetMaintenanceStateResponse,
      } = await import('../../../shared/api/proto-contracts');
      let currentState: MaintenanceState = 'inactive';
      try {
        const state = await protoFetch('/maintenance/state', parseGetMaintenanceStateResponse);
        if (cancelled) return;
        currentState = state.maintenanceState || 'inactive';
      } catch (err) {
        console.debug('API call failed for /maintenance/state:', err);
        if (cancelled) return;
      }

      maintenanceStateRef.current.previous = currentState;

      if (currentState !== 'active') {
        try {
          const resp = await protoFetch('/maintenance/state', parseSetMaintenanceStateResponse, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ maintenanceState: 'active' }),
          });
          if (resp.success) {
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

    const activationTimer = window.setTimeout(() => {
      activateMonitoring().catch((err: unknown) => { console.error('Failed to activate monitoring:', err); });
    }, 2200);

    const stateRef = maintenanceStateRef.current;

    return () => {
      cancelled = true;
      window.clearTimeout(activationTimer);
      const previous = stateRef.previous;
      if (stateRef.activated && previous && previous !== 'active') {
        fetch(buildApiUrl('/maintenance/state'), {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({ maintenanceState: previous })
        }).catch((unmountError: unknown) => {
          console.warn('Failed to restore monitoring state:', unmountError);
        });
      }
    };
  }, []);

  const fetchDetailedMetrics = useCallback(async () => {
    if (remoteNodeSelected) return;
    try {
      const { parseDetailedMetrics } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/detailed', parseDetailedMetrics);
      if (!mountedRef.current) return;
      setDetailedMetrics(data);
      setSubsystemError('detailedMetrics', null);
      const diskPercent = data.memoryDetails?.diskUsage?.percent;
      if (typeof diskPercent === 'number' && Number.isFinite(diskPercent)) {
        appendDiskUsagePoint(toIsoString(data.timestamp), diskPercent);
      }
    } catch (err) {
      console.debug('API call failed for /metrics/detailed:', err);
      if (!mountedRef.current) return;
      setSubsystemError('detailedMetrics', toApiError(err));
    }
  }, [appendDiskUsagePoint, remoteNodeSelected, setSubsystemError]);

  const fetchProcessMonitorData = useCallback(async () => {
    if (remoteNodeSelected) return;
    try {
      const { parseProcessMonitorData } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/processes', parseProcessMonitorData);
      if (!mountedRef.current) return;
      setProcessMonitorData(data);
      setSubsystemError('processes', null);
    } catch (err) {
      console.debug('API call failed for /metrics/processes:', err);
      if (!mountedRef.current) return;
      setSubsystemError('processes', toApiError(err));
    }
  }, [remoteNodeSelected, setSubsystemError]);

  const fetchInfrastructureData = useCallback(async () => {
    if (remoteNodeSelected) return;
    try {
      const { parseInfrastructureMonitorData } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/infrastructure', parseInfrastructureMonitorData);
      if (!mountedRef.current) return;
      setInfrastructureData(data);
      setSubsystemError('infrastructure', null);
      const { storageIo, timestamp } = data;
      if (storageIo) {
        const readRate = Number(storageIo.readMbPerSec);
        const writeRate = Number(storageIo.writeMbPerSec);
        if (Number.isFinite(readRate) || Number.isFinite(writeRate)) {
          appendDiskPoints(toIsoString(timestamp), readRate, writeRate);
        }
      }
    } catch (err) {
      console.debug('API call failed for /metrics/infrastructure:', err);
      if (!mountedRef.current) return;
      setSubsystemError('infrastructure', toApiError(err));
    }
  }, [appendDiskPoints, remoteNodeSelected, setSubsystemError]);

  const fetchInvestigations = useCallback(async () => {
    if (remoteNodeSelected) return;
    try {
      const { parseInvestigations } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/investigations?limit=10', parseInvestigations);
      if (!mountedRef.current) return;
      const sorted = [...data].sort((a, b) => {
        const aTime = a.startTime ? timestampMs(a.startTime) : NaN;
        const bTime = b.startTime ? timestampMs(b.startTime) : NaN;
        return isNaN(bTime) && isNaN(aTime)
          ? 0
          : isNaN(bTime)
          ? -1
          : isNaN(aTime)
          ? 1
          : bTime - aTime;
      });
      setInvestigations(sorted);
      setSubsystemError('investigations', null);
    } catch (err) {
      console.debug('API call failed for /investigations:', err);
      if (!mountedRef.current) return;
      setSubsystemError('investigations', toApiError(err));
    }
  }, [remoteNodeSelected, setSubsystemError]);

  const refreshMetrics = useCallback(async () => {
    await Promise.all([fetchMetrics(), fetchDeviceGraph()]);
    if (!remoteNodeSelected) {
      await Promise.all([
        fetchDetailedMetrics(),
        fetchMetricsTimeline(historyWindowSeconds)
      ]);
    }
  }, [fetchDetailedMetrics, fetchDeviceGraph, fetchMetrics, fetchMetricsTimeline, historyWindowSeconds, remoteNodeSelected]);

  const fetchDeferredData = useCallback(async () => {
    if (remoteNodeSelected) return;
    await Promise.all([
      fetchDetailedMetrics(),
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations(),
      fetchMetricsTimeline(historyWindowSeconds)
    ]);
  }, [
    fetchMetricsTimeline,
    fetchDetailedMetrics,
    fetchProcessMonitorData,
    fetchInfrastructureData,
	  fetchInvestigations,
	  historyWindowSeconds,
	  remoteNodeSelected
  ]);

  const refresh = useCallback(async () => {
    setIsLoading(true);

    const isHealthy = await checkHealth();
    if (!mountedRef.current) return;
    if (!isHealthy) {
      setSubsystemError('metrics', {
        error: 'API server is not responding',
        detail: { code: 'unavailable', message: 'Health check failed - ensure the Go backend is running', retryable: true, recovery: 'wait' },
        timestamp: new Date().toISOString(),
      });
      setIsLoading(false);
      return;
    }

    await Promise.all([fetchMetrics(), fetchDeviceGraph()]);

    if (!mountedRef.current) return;
    setIsLoading(false);

    await fetchDeferredData();

    if (!mountedRef.current) return;
  }, [
    checkHealth,
    setSubsystemError,
    fetchDeviceGraph,
    fetchMetrics,
    fetchDeferredData
  ]);

  const refreshInitial = useCallback(async () => {
    setIsLoading(true);

    const isHealthy = await checkHealth();
    if (!mountedRef.current) return;
    if (!isHealthy) {
      setSubsystemError('metrics', {
        error: 'API server is not responding',
        detail: { code: 'unavailable', message: 'Health check failed - ensure the Go backend is running', retryable: true, recovery: 'wait' },
        timestamp: new Date().toISOString(),
      });
      setIsLoading(false);
      return;
    }

    await Promise.all([fetchMetrics(), fetchDeviceGraph()]);

    if (!mountedRef.current) return;
    setIsLoading(false);
  }, [checkHealth, fetchDeviceGraph, fetchMetrics, setSubsystemError]);

  // A machine switch is a new observation context. Do not retain local
  // histories or detailed panels while the selected target is remote; those
  // values cannot be silently presented as if they belonged to that machine.
  useEffect(() => {
    setMetrics(null);
    setDeviceGraph(null);
    setIsLoading(true);
    setConsecutiveFailures(0);
    setLastSuccessfulFetch(null);
    if (!remoteNodeSelected) return;
    setDetailedMetrics(null);
    setProcessMonitorData(null);
    setInfrastructureData(null);
    setInvestigations([]);
    clearHistory();
    setSubsystemErrors({});
  }, [clearHistory, remoteNodeSelected]);

  // Initial load
  useEffect(() => {
    let cancelled = false;
    void refreshInitial();
    const deferredTimer = window.setTimeout(() => {
      if (!cancelled) {
        void fetchDeferredData();
      }
    }, 2200);
    return () => {
      cancelled = true;
      window.clearTimeout(deferredTimer);
    };
  }, [fetchDeferredData, refreshInitial]);

  // Set up polling for metrics (every 5 seconds for responsive graphs)
  usePolling(refreshMetrics, remoteNodeSelected ? 15000 : 5000, pollingEnabled, { enabled: true, maxIntervalMs: 60000 });

  // Set up polling for detailed data + health (every 60 seconds)
  const fetchDetailedAll = useCallback(() => {
    if (remoteNodeSelected) return;
    Promise.all([
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations(),
      checkHealth()
    ]).catch((err: unknown) => { console.error('Failed to fetch detailed data:', err); });
  }, [fetchProcessMonitorData, fetchInfrastructureData, fetchInvestigations, checkHealth, remoteNodeSelected]);
  usePolling(fetchDetailedAll, 60000, pollingEnabled, { enabled: true, maxIntervalMs: 300000 });

  return {
    metrics,
    deviceGraph,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    metricHistory,
    isLoading,
    error,
    subsystemErrors,
    isStale,
    retryAttempt: consecutiveFailures,
    lastSuccessfulFetch,
    healthStatus,
    healthError,
    toggleMonitoring,
    refreshHealth,
    refresh: () => { void refresh(); },
    refreshMetrics: () => { void refreshMetrics(); }
  };
};
