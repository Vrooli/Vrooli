import { useState, useEffect, useCallback } from 'react';
import type {
  MetricsResponse,
  DetailedMetrics,
  ProcessMonitorData,
  InfrastructureMonitorData,
  Investigation,
  APIError,
  MetricsTimelineResponse,
  MetricHistory,
  ChartDataPoint
} from '../types';

interface UseSystemMonitorReturn {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  processMonitorData: ProcessMonitorData | null;
  infrastructureData: InfrastructureMonitorData | null;
  investigations: Investigation[];
  metricHistory: MetricHistory | null;
  isLoading: boolean;
  error: APIError | null;
  refresh: () => void;
  refreshMetrics: () => void;
}

const API_BASE = '';  // Using Vite proxy, so same origin
const DISK_HISTORY_LIMIT = 180;

const appendHistoryPoint = (
  series: ChartDataPoint[] | undefined,
  point: ChartDataPoint,
  limit = DISK_HISTORY_LIMIT
): ChartDataPoint[] => {
  const next = series ? [...series, point] : [point];
  if (next.length > limit) {
    return next.slice(next.length - limit);
  }
  return next;
};

const cloneSeries = (series?: ChartDataPoint[]) => (series ? [...series] : undefined);

const ensureHistoryBase = (history: MetricHistory | null): MetricHistory => ({
  windowSeconds: history?.windowSeconds ?? 0,
  sampleIntervalSeconds: history?.sampleIntervalSeconds ?? 0,
  cpu: history?.cpu ? [...history.cpu] : [],
  memory: history?.memory ? [...history.memory] : [],
  network: history?.network ? [...history.network] : [],
  diskUsage: cloneSeries(history?.diskUsage),
  diskRead: cloneSeries(history?.diskRead),
  diskWrite: cloneSeries(history?.diskWrite)
});

export const useSystemMonitor = (): UseSystemMonitorReturn => {
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [detailedMetrics, setDetailedMetrics] = useState<DetailedMetrics | null>(null);
  const [processMonitorData, setProcessMonitorData] = useState<ProcessMonitorData | null>(null);
  const [infrastructureData, setInfrastructureData] = useState<InfrastructureMonitorData | null>(null);
  const [investigations, setInvestigations] = useState<Investigation[]>([]);
  const [metricHistory, setMetricHistory] = useState<MetricHistory | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<APIError | null>(null);

  const handleApiCall = useCallback(async <T,>(url: string): Promise<T | null> => {
    try {
      const response = await fetch(`${API_BASE}${url}`);
      
      if (!response.ok) {
        const errorText = await response.text();
        let errorData: APIError;
        try {
          errorData = JSON.parse(errorText);
        } catch {
          errorData = {
            error: `HTTP ${response.status}: ${response.statusText}`,
            details: errorText,
            timestamp: new Date().toISOString()
          };
        }
        throw errorData;
      }

      const data = await response.json();
      return data;
    } catch (err) {
      console.error(`API call failed for ${url}:`, err);
      
      if (err && typeof err === 'object' && 'error' in err) {
        setError(err as APIError);
      } else {
        setError({
          error: 'Network or unknown error',
          details: err instanceof Error ? err.message : String(err),
          timestamp: new Date().toISOString()
        });
      }
      return null;
    }
  }, []);

  const checkHealth = useCallback(async (): Promise<boolean> => {
    try {
      const response = await fetch(`${API_BASE}/health`);
      return response.ok;
    } catch {
      return false;
    }
  }, []);

  const fetchMetrics = useCallback(async () => {
    const data = await handleApiCall<MetricsResponse>('/api/metrics/current');
    if (data) {
      setMetrics(data);
      setError(null);
    }
  }, [handleApiCall]);

  const fetchDetailedMetrics = useCallback(async () => {
    const data = await handleApiCall<DetailedMetrics>('/api/metrics/detailed');
    if (data) {
      setDetailedMetrics(data);
      const diskPercent = data.memory_details?.disk_usage?.percent;
      if (typeof diskPercent === 'number' && Number.isFinite(diskPercent)) {
        const timestamp = data.timestamp ?? new Date().toISOString();
        setMetricHistory(prev => {
          const base = ensureHistoryBase(prev);
          return {
            ...base,
            diskUsage: appendHistoryPoint(base.diskUsage, {
              timestamp,
              value: diskPercent
            })
          };
        });
      }
    }
  }, [handleApiCall]);

  const fetchMetricsTimeline = useCallback(async (windowSeconds = 120) => {
    const data = await handleApiCall<MetricsTimelineResponse>(`/api/metrics/timeline?window=${windowSeconds}`);
    if (!data || !data.samples) {
      return;
    }

    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      return {
        ...base,
        windowSeconds: data.window_seconds,
        sampleIntervalSeconds: data.sample_interval_seconds,
        cpu: data.samples.map(sample => ({
          timestamp: sample.timestamp,
          value: sample.cpu_usage
        })),
        memory: data.samples.map(sample => ({
          timestamp: sample.timestamp,
          value: sample.memory_usage
        })),
        network: data.samples.map(sample => ({
          timestamp: sample.timestamp,
          value: sample.tcp_connections
        }))
      };
    });
  }, [handleApiCall]);

  const fetchProcessMonitorData = useCallback(async () => {
    const data = await handleApiCall<ProcessMonitorData>('/api/metrics/processes');
    if (data) {
      setProcessMonitorData(data);
    }
  }, [handleApiCall]);

  const fetchInfrastructureData = useCallback(async () => {
    const data = await handleApiCall<InfrastructureMonitorData>('/api/metrics/infrastructure');
    if (data) {
      setInfrastructureData(data);
      const { storage_io, timestamp } = data;
      if (storage_io) {
        const recordTimestamp = timestamp ?? new Date().toISOString();
        const readRate = Number(storage_io.read_mb_per_sec);
        const writeRate = Number(storage_io.write_mb_per_sec);
        if (Number.isFinite(readRate) || Number.isFinite(writeRate)) {
          setMetricHistory(prev => {
            const base = ensureHistoryBase(prev);
            return {
              ...base,
              diskRead: Number.isFinite(readRate)
                ? appendHistoryPoint(base.diskRead, {
                    timestamp: recordTimestamp,
                    value: readRate
                  })
                : base.diskRead,
              diskWrite: Number.isFinite(writeRate)
                ? appendHistoryPoint(base.diskWrite, {
                    timestamp: recordTimestamp,
                    value: writeRate
                  })
                : base.diskWrite
            };
          });
        }
      }
    }
  }, [handleApiCall]);

  const fetchInvestigations = useCallback(async () => {
    const data = await handleApiCall<Investigation>('/api/investigations/latest');
    if (data) {
      setInvestigations([data]); // For now, just latest investigation
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
    
    // Check health first
    const isHealthy = await checkHealth();
    if (!isHealthy) {
      setError({
        error: 'API server is not responding',
        details: 'Health check failed - ensure the Go backend is running',
        timestamp: new Date().toISOString()
      });
      setIsLoading(false);
      return;
    }

    // Fetch all data
    await Promise.all([
      fetchMetrics(),
      fetchDetailedMetrics(),
      fetchProcessMonitorData(),
      fetchInfrastructureData(),
      fetchInvestigations(),
      fetchMetricsTimeline(120)
    ]);
    
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
  useEffect(() => {
    const metricsInterval = setInterval(refreshMetrics, 5000);
    return () => clearInterval(metricsInterval);
  }, [refreshMetrics]);

  // Set up polling for detailed data (every 60 seconds)
  useEffect(() => {
    const detailedInterval = setInterval(() => {
      Promise.all([
        fetchProcessMonitorData(),
        fetchInfrastructureData(),
        fetchInvestigations()
      ]);
    }, 60000);
    
    return () => clearInterval(detailedInterval);
  }, [fetchProcessMonitorData, fetchInfrastructureData, fetchInvestigations]);

  return {
    metrics,
    detailedMetrics,
    processMonitorData,
    infrastructureData,
    investigations,
    metricHistory,
    isLoading,
    error,
    refresh,
    refreshMetrics
  };
};
