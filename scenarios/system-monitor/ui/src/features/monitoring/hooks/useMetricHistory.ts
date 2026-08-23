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
	cpuContextSwitches: cloneSeries(history?.cpuContextSwitches),
	cpuInterrupts: cloneSeries(history?.cpuInterrupts),
	cpuNormalizedLoad1: cloneSeries(history?.cpuNormalizedLoad1),
	cpuNormalizedLoad5: cloneSeries(history?.cpuNormalizedLoad5),
	cpuRunQueue: cloneSeries(history?.cpuRunQueue),
	cpuStallSome: cloneSeries(history?.cpuStallSome),
	cpuStallFull: cloneSeries(history?.cpuStallFull),
	cpuCoreImbalance: cloneSeries(history?.cpuCoreImbalance),
	cpuModeIowait: cloneSeries(history?.cpuModeIowait),
	cpuModeSteal: cloneSeries(history?.cpuModeSteal),
  memory: history?.memory ? [...history.memory] : [],
  swap: history?.swap ? [...history.swap] : [],
  swapTraffic: history?.swapTraffic ? [...history.swapTraffic] : [],
  majorFaults: history?.majorFaults ? [...history.majorFaults] : [],
  fragmentation: history?.fragmentation ? [...history.fragmentation] : [],
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
			cpuContextSwitches: data.samples.filter(sample => sample.cpuContextSwitchesPerSecond?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuContextSwitchesPerSecond?.state.value) })),
			cpuInterrupts: data.samples.filter(sample => sample.cpuInterruptsPerSecond?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuInterruptsPerSecond?.state.value) })),
			cpuNormalizedLoad1: data.samples.filter(sample => sample.cpuNormalizedLoad1?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuNormalizedLoad1?.state.value) })),
			cpuNormalizedLoad5: data.samples.filter(sample => sample.cpuNormalizedLoad5?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuNormalizedLoad5?.state.value) })),
			cpuRunQueue: data.samples.filter(sample => sample.cpuRunQueueDepth?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuRunQueueDepth?.state.value) })),
			cpuStallSome: data.samples.filter(sample => sample.cpuStallSomeAvg10?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuStallSomeAvg10?.state.value) })),
			cpuStallFull: data.samples.filter(sample => sample.cpuStallFullAvg10?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuStallFullAvg10?.state.value) })),
			cpuCoreImbalance: data.samples.filter(sample => sample.cpuCoreImbalanceIndex?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuCoreImbalanceIndex?.state.value) })),
			cpuModeIowait: data.samples.filter(sample => sample.cpuModeIowait?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuModeIowait?.state.value) })),
			cpuModeSteal: data.samples.filter(sample => sample.cpuModeSteal?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuModeSteal?.state.value) })),
          memory: data.samples
            .filter(sample => sample.memory?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.memory?.state.value) })),
          swap: data.samples
            .filter(sample => sample.swap?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.swap?.state.value) })),
          swapTraffic: data.samples
            .filter(sample => sample.swapTraffic?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.swapTraffic?.state.value) })),
          majorFaults: data.samples
            .filter(sample => sample.majorFaults?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.majorFaults?.state.value) })),
          fragmentation: data.samples
            .filter(sample => sample.fragmentationIndex?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.fragmentationIndex?.state.value) })),
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
