import { useState, useCallback, useRef, useEffect } from 'react';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { protoFetch, toApiError } from '../../../shared/api/apiFetch';
import type {
  MetricHistory,
  ChartDataPoint,
  APIError
} from '../../../types';

export interface UseMetricHistoryReturn {
  metricHistory: MetricHistory | null;
  fetchMetricsTimeline: (windowSeconds?: number) => Promise<void>;
  appendGpuPoint: (timestamp: string, value: number) => void;
  appendDiskPoints: (timestamp: string, readRate: number, writeRate: number) => void;
  appendDiskUsagePoint: (timestamp: string, value: number) => void;
}

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
  gpu: history?.gpu ? [...history.gpu] : [],
  diskUsage: cloneSeries(history?.diskUsage),
  diskRead: cloneSeries(history?.diskRead),
  diskWrite: cloneSeries(history?.diskWrite)
});

export const useMetricHistory = (
  setError: (error: APIError | null) => void
): UseMetricHistoryReturn => {
  const [metricHistory, setMetricHistory] = useState<MetricHistory | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  const fetchMetricsTimeline = useCallback(async (windowSeconds = 120) => {
    try {
      const { parseMetricsTimelineResponse } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch(`/metrics/timeline?window=${windowSeconds}`, parseMetricsTimelineResponse);
      if (!mountedRef.current || !data || !data.samples) return;

      const toIso = (ts?: Timestamp): string =>
        ts ? timestampDate(ts).toISOString() : new Date().toISOString();

      setMetricHistory(prev => {
        const base = ensureHistoryBase(prev);
        return {
          ...base,
          windowSeconds: data.windowSeconds,
          sampleIntervalSeconds: data.sampleIntervalSeconds,
          cpu: data.samples
            .filter(sample => sample.cpu?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpu?.state.value) })),
          memory: data.samples
            .filter(sample => sample.memory?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.memory?.state.value) })),
          network: data.samples
            .filter(sample => sample.connections?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.connections?.state.value) })),
          gpu: data.samples
            .filter(sample => sample.gpu?.state.case === 'measured')
            .map(sample => ({
              timestamp: toIso(sample.timestamp),
              value: Number(sample.gpu?.state.value)
            }))
        };
      });
    } catch (err) {
      console.error('API call failed for /metrics/timeline:', err);
      if (!mountedRef.current) return;
      setError(toApiError(err));
    }
  }, [setError]);

  const appendGpuPoint = useCallback((timestamp: string, value: number) => {
    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      return {
        ...base,
        gpu: appendHistoryPoint(base.gpu, { timestamp, value })
      };
    });
  }, []);

  const appendDiskPoints = useCallback((timestamp: string, readRate: number, writeRate: number) => {
    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      return {
        ...base,
        diskRead: Number.isFinite(readRate)
          ? appendHistoryPoint(base.diskRead, { timestamp, value: readRate })
          : base.diskRead,
        diskWrite: Number.isFinite(writeRate)
          ? appendHistoryPoint(base.diskWrite, { timestamp, value: writeRate })
          : base.diskWrite
      };
    });
  }, []);

  const appendDiskUsagePoint = useCallback((timestamp: string, value: number) => {
    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      return {
        ...base,
        diskUsage: appendHistoryPoint(base.diskUsage, { timestamp, value })
      };
    });
  }, []);

  return {
    metricHistory,
    fetchMetricsTimeline,
    appendGpuPoint,
    appendDiskPoints,
    appendDiskUsagePoint
  };
};
