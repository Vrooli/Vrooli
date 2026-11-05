// DOC: docs/internal/SEAMS.md#ui-lifecycle-and-shared-utility-seams
import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { timestampMs } from '@bufbuild/protobuf/wkt';
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

export type Subsystem = 'metrics' | 'history' | 'detailedMetrics' | 'deviceGraph' | 'processes' | 'infrastructure' | 'investigations';

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
  /** When the polling loop last tried, so a surface can count down honestly. */
  lastAttemptAt: Date | null;
  /** The real polling period for the current subject, in seconds. */
  retryIntervalSeconds: number;
  healthStatus: SystemHealthStatus | null;
  healthError: string | null;
  toggleMonitoring: () => Promise<void>;
  refreshHealth: () => Promise<void>;
  refresh: () => void;
  refreshMetrics: () => void;
}

interface UseSystemMonitorOptions {
  enabled?: boolean;
  node?: string;
}

export const useSystemMonitor = (historyWindowSeconds = 120, options: UseSystemMonitorOptions = {}): UseSystemMonitorReturn => {
  const pollingEnabled = options.enabled ?? true;
  const viewEpochRef = useRef({ enabled: pollingEnabled, generation: 0 });
  const [stateGeneration, setStateGeneration] = useState(0);
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [deviceGraph, setDeviceGraph] = useState<DeviceGraph | null>(null);
  const [detailedMetrics, setDetailedMetrics] = useState<DetailedMetrics | null>(null);
  const [processMonitorData, setProcessMonitorData] = useState<ProcessMonitorData | null>(null);
  const [infrastructureData, setInfrastructureData] = useState<InfrastructureMonitorData | null>(null);
  const [investigations, setInvestigations] = useState<Investigation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [subsystemErrors, setSubsystemErrors] = useState<Partial<Record<Subsystem, APIError>>>({});
  const [lastSuccessfulFetch, setLastSuccessfulFetch] = useState<Date | null>(null);
  const [lastAttemptAt, setLastAttemptAt] = useState<Date | null>(null);
  const [consecutiveFailures, setConsecutiveFailures] = useState(0);
  const isStale = consecutiveFailures >= 3;
  const mountedRef = useRef(true);
  const abortRef = useRef<AbortController | null>(null);
  const { showApiError } = useToast();
  const lastMetricsErrorRef = useRef<string | null>(null);
  const subjectRef = useRef({ node: options.node, generation: 0 });
  const diskGapRef = useRef(false);
  const lastDiskObservationRef = useRef<string | null>(null);
  const gpuGapRef = useRef(false);
  const lastGpuObservationRef = useRef<string | null>(null);
  if (subjectRef.current.node !== options.node) {
    subjectRef.current = { node: options.node, generation: subjectRef.current.generation + 1 };
  }
  const remoteNodeSelectedRef = useRef(false);
  remoteNodeSelectedRef.current = Boolean(options.node);

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

  // Only show toast for metrics errors (primary data source).
  //
  // Not while a remote machine is the subject: the identity strip states that
  // condition persistently, names the machine, and offers the retry. A toast
  // beside it repeats the same fact in a form that disappears, and carries an
  // internal reference code the reader cannot act on.
  useEffect(() => {
    if (remoteNodeSelectedRef.current) return;
    const metricsError = subsystemErrors.metrics;
    if (metricsError && metricsError.error !== lastMetricsErrorRef.current) {
      lastMetricsErrorRef.current = metricsError.error;
      showApiError(metricsError);
    } else if (!metricsError) {
      lastMetricsErrorRef.current = null;
    }
  }, [subsystemErrors, showApiError]);

  const { healthStatus, healthError, checkHealth, refreshHealth, toggleMonitoring } = useHealthCheck();
  const setHistoryError = useCallback((err: APIError | null) => { setSubsystemError('history', err); }, [setSubsystemError]);
  const { metricHistory, fetchMetricsTimeline, clearHistory, invalidatePending, appendGpuPoint, appendDiskPoints, appendDiskUsagePoint } = useMetricHistory(setHistoryError);
  // A response started under the previous view state must not update the new one.
  if (viewEpochRef.current.enabled !== pollingEnabled) {
    viewEpochRef.current = { enabled: pollingEnabled, generation: viewEpochRef.current.generation + 1 };
    invalidatePending();
  }
  useEffect(() => { if (!pollingEnabled) setIsLoading(false); }, [pollingEnabled]);
  const remoteNodeSelected = Boolean(options.node);
  // Remote readings cross a relay, so they are polled less often than local
  // ones. The value is named because surfaces quote it back to the reader.
  const metricsPollIntervalMs = remoteNodeSelected ? 15000 : 5000;

  const fetchMetrics = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    const params = new URLSearchParams();
    if (options.node) params.set('node', options.node);
    const query = params.toString();
    const url = `/metrics/current${query ? `?${query}` : ''}`;
    try {
      const data = await protoFetch(url, parseMetricsResponse);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setMetrics(data);
      setSubsystemError('metrics', null);
      setLastSuccessfulFetch(new Date());
      setConsecutiveFailures(0);
      const gpuAt = data.gpu?.observedAt;
      if (gpuAt && data.gpu?.state.case === 'measured' && Number.isFinite(data.gpu.state.value)) {
        const observedAt = toIsoString(gpuAt);
        if (!lastGpuObservationRef.current || observedAt > lastGpuObservationRef.current) {
          appendGpuPoint(observedAt, data.gpu.state.value, gpuGapRef.current);
          lastGpuObservationRef.current = observedAt;
          gpuGapRef.current = false;
        }
      } else {
        gpuGapRef.current = true;
      }
    } catch (err) {
      console.debug(`API call failed for ${url}:`, err);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      gpuGapRef.current = true;
      setSubsystemError('metrics', toApiError(err));
      setConsecutiveFailures(prev => prev + 1);
    }
  }, [options.node, appendGpuPoint, setSubsystemError]);

  const fetchDeviceGraph = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    try {
      const { parseDeviceGraph } = await import('../../../shared/api/current-metrics-contract');
      const query = options.node ? `?node=${encodeURIComponent(options.node)}` : '';
      const data = await protoFetch(`/metrics/devices${query}`, parseDeviceGraph);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setDeviceGraph(data);
      setSubsystemError('deviceGraph', null);
    } catch (err) {
      console.debug('API call failed for /metrics/devices:', err);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setSubsystemError('deviceGraph', toApiError(err));
    }
  }, [options.node, setSubsystemError]);

  const fetchDetailedMetrics = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    if (remoteNodeSelected) return;
    try {
      const { parseDetailedMetrics } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/detailed', parseDetailedMetrics);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setDetailedMetrics(data);
      setSubsystemError('detailedMetrics', null);
      const diskPercent = data.memoryDetails?.diskUsage?.percent;
      if (typeof diskPercent === 'number' && Number.isFinite(diskPercent)) {
        appendDiskUsagePoint(toIsoString(data.timestamp), diskPercent);
      }
    } catch (err) {
      console.debug('API call failed for /metrics/detailed:', err);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setSubsystemError('detailedMetrics', toApiError(err));
    }
  }, [appendDiskUsagePoint, remoteNodeSelected, setSubsystemError]);

  const fetchProcessMonitorData = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    if (remoteNodeSelected) return;
    try {
      const { parseProcessMonitorData } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/processes', parseProcessMonitorData);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setProcessMonitorData(data);
      setSubsystemError('processes', null);
    } catch (err) {
      console.debug('API call failed for /metrics/processes:', err);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setSubsystemError('processes', toApiError(err));
    }
  }, [remoteNodeSelected, setSubsystemError]);

  const fetchInfrastructureData = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    if (remoteNodeSelected) return;
    try {
      const { parseInfrastructureMonitorData } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/metrics/infrastructure', parseInfrastructureMonitorData);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setInfrastructureData(data);
      setSubsystemError('infrastructure', null);
      const read = data.storageIo?.readRate?.state;
      const write = data.storageIo?.writeRate?.state;
      const readAt = data.storageIo?.readRate?.observedAt;
      const writeAt = data.storageIo?.writeRate?.observedAt;
      if (readAt && writeAt && readAt.seconds === writeAt.seconds && readAt.nanos === writeAt.nanos && read?.case === 'measured' && write?.case === 'measured' && Number.isFinite(read.value) && Number.isFinite(write.value)) {
        const observedAt = toIsoString(readAt);
        if (!lastDiskObservationRef.current || observedAt > lastDiskObservationRef.current) {
          appendDiskPoints(observedAt, read.value, write.value, diskGapRef.current);
          lastDiskObservationRef.current = observedAt;
          diskGapRef.current = false;
        }
      } else {
        diskGapRef.current = true;
      }
    } catch (err) {
      console.debug('API call failed for /metrics/infrastructure:', err);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      diskGapRef.current = true;
      setSubsystemError('infrastructure', toApiError(err));
    }
  }, [appendDiskPoints, remoteNodeSelected, setSubsystemError]);

  const fetchInvestigations = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    if (remoteNodeSelected) return;
    try {
      const { parseInvestigations } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch('/investigations?limit=10', parseInvestigations);
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
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
      if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
      setSubsystemError('investigations', toApiError(err));
    }
  }, [remoteNodeSelected, setSubsystemError]);

  const refreshMetrics = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    // Stamped before the request rather than after it: the countdown a reader
    // sees must be the schedule the loop is actually keeping, and the next tick
    // is scheduled relative to this attempt whether or not it succeeds.
    setLastAttemptAt(new Date());
    await Promise.all([fetchMetrics(), fetchDeviceGraph()]);
    if (generation === subjectRef.current.generation && viewEpoch === viewEpochRef.current.generation && !remoteNodeSelected) {
      await Promise.all([
        fetchDetailedMetrics(),
        fetchInfrastructureData(),
        fetchMetricsTimeline(historyWindowSeconds)
      ]);
    }
  }, [fetchDetailedMetrics, fetchDeviceGraph, fetchMetrics, fetchInfrastructureData, fetchMetricsTimeline, historyWindowSeconds, remoteNodeSelected]);

  const fetchDeferredData = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    if (remoteNodeSelected || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
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
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    setIsLoading(true);

    const isHealthy = await checkHealth();
    if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
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

    if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
    setIsLoading(false);

    await fetchDeferredData();

    if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
  }, [
    checkHealth,
    setSubsystemError,
    fetchDeviceGraph,
    fetchMetrics,
    fetchDeferredData
  ]);

  const refreshInitial = useCallback(async () => {
    const generation = subjectRef.current.generation;
    const viewEpoch = viewEpochRef.current.generation;
    setIsLoading(true);

    const isHealthy = await checkHealth();
    if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
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

    if (!mountedRef.current || (generation !== subjectRef.current.generation || viewEpoch !== viewEpochRef.current.generation)) return;
    setIsLoading(false);
  }, [checkHealth, fetchDeviceGraph, fetchMetrics, setSubsystemError]);

  // A machine switch is a new observation context. Do not retain local
  // histories or detailed panels while the selected target is remote; those
  // values cannot be silently presented as if they belonged to that machine.
  useEffect(() => {
    diskGapRef.current = false;
    lastDiskObservationRef.current = null;
    gpuGapRef.current = false;
    lastGpuObservationRef.current = null;
    setMetrics(null);
    setDeviceGraph(null);
    setIsLoading(true);
    setConsecutiveFailures(0);
    setLastSuccessfulFetch(null);
    setLastAttemptAt(null);
    clearHistory();
    setSubsystemErrors({});
    setStateGeneration(subjectRef.current.generation);
    if (!remoteNodeSelected) return;
    setDetailedMetrics(null);
    setProcessMonitorData(null);
    setInfrastructureData(null);
    setInvestigations([]);
    // Keyed on the node id, not on `remoteNodeSelected`: moving between two
    // remote machines is also a change of subject, and leaving the previous
    // machine's readings on screen under the new machine's name is the exact
    // failure the identity strip exists to prevent.
  }, [clearHistory, options.node, remoteNodeSelected]);

  // Initial load
  useEffect(() => {
    let cancelled = false;
    const viewEpoch = viewEpochRef.current.generation;
    void refreshInitial();
    const deferredTimer = window.setTimeout(() => {
      if (!cancelled && viewEpoch === viewEpochRef.current.generation) {
        void fetchDeferredData();
      }
    }, 2200);
    return () => {
      cancelled = true;
      window.clearTimeout(deferredTimer);
    };
  }, [fetchDeferredData, refreshInitial]);

  // Set up polling for metrics (every 5 seconds for responsive graphs)
  usePolling(refreshMetrics, metricsPollIntervalMs, pollingEnabled, { enabled: true, maxIntervalMs: 60000 });

  // Set up polling for detailed data + health (every 60 seconds)
  const fetchDetailedAll = useCallback(() => {
    if (remoteNodeSelected) return;
    Promise.all([
      fetchProcessMonitorData(),
      fetchInvestigations(),
      checkHealth()
    ]).catch((err: unknown) => { console.error('Failed to fetch detailed data:', err); });
  }, [fetchProcessMonitorData, fetchInvestigations, checkHealth, remoteNodeSelected]);
  usePolling(fetchDetailedAll, 60000, pollingEnabled, { enabled: true, maxIntervalMs: 300000 });

  // A new subject renders before the reset effect runs. Never expose the old
  // subject's evidence during that render, even for a single commit.
  const subjectReady = stateGeneration === subjectRef.current.generation;
  return {
    metrics: subjectReady ? metrics : null,
    deviceGraph: subjectReady ? deviceGraph : null,
    detailedMetrics: subjectReady ? detailedMetrics : null,
    processMonitorData: subjectReady ? processMonitorData : null,
    infrastructureData: subjectReady ? infrastructureData : null,
    investigations: subjectReady ? investigations : [],
    metricHistory: subjectReady ? metricHistory : null,
    isLoading: subjectReady ? isLoading : true,
    error: subjectReady ? error : null,
    subsystemErrors: subjectReady ? subsystemErrors : {},
    isStale: subjectReady && isStale,
    retryAttempt: subjectReady ? consecutiveFailures : 0,
    lastSuccessfulFetch: subjectReady ? lastSuccessfulFetch : null,
    lastAttemptAt: subjectReady ? lastAttemptAt : null,
    retryIntervalSeconds: metricsPollIntervalMs / 1000,
    healthStatus,
    healthError,
    toggleMonitoring,
    refreshHealth,
    refresh: () => { void refresh(); },
    refreshMetrics: () => { void refreshMetrics(); }
  };
};
