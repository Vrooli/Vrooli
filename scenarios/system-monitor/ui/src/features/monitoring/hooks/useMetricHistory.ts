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
  clearHistory: () => void;
  invalidatePending: () => void;
  appendGpuPoint: (timestamp: string, value: number, gapBefore?: boolean) => void;
  appendDiskPoints: (timestamp: string, readRate: number, writeRate: number, gapBefore?: boolean) => void;
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
  missingTimestampCount: history?.missingTimestampCount,
  windowStart: history?.windowStart,
  windowEnd: history?.windowEnd,
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
  kernelSlab: cloneSeries(history?.kernelSlab),
  kernelReclaimable: cloneSeries(history?.kernelReclaimable),
  kernelUnreclaimable: cloneSeries(history?.kernelUnreclaimable),
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
  const requestRef = useRef(0);

  useEffect(() => {
    return () => { mountedRef.current = false; };
  }, []);

  const fetchMetricsTimeline = useCallback(async (windowSeconds = 120) => {
    const request = ++requestRef.current;
    try {
      const { parseMetricsTimelineResponse } = await import('../../../shared/api/proto-contracts');
      const data = await protoFetch(`/metrics/timeline?window=${windowSeconds}`, parseMetricsTimelineResponse);
      if (!mountedRef.current || request !== requestRef.current || !data || !data.samples) return;
      setError(null);

      const samples = data.samples.filter(sample => sample.timestamp);
      const toIso = (ts?: Timestamp): string =>
        ts ? timestampDate(ts).toISOString() : ''; 
      // The endpoint echoes a requested sampling interval; it is not the
      // scheduler cadence. Infer typical cycle spacing from the cycle timestamps
      // and preserve explicit non-measured cycles as breaks in each series.
      const ordered = [...samples].sort((a, b) => timestampDate(a.timestamp!).getTime() - timestampDate(b.timestamp!).getTime());
      const deltas = ordered.slice(1).map((sample, index) => timestampDate(sample.timestamp!).getTime() - timestampDate(ordered[index]!.timestamp!).getTime()).filter(delta => delta > 0).sort((a, b) => a - b);
      const cadenceMs = deltas.length ? deltas[Math.floor(deltas.length / 2)]! : 0;
      const series = (pick: (sample: typeof ordered[number]) => { state: { case: string | undefined; value?: unknown } } | undefined): ChartDataPoint[] => {
        let missed = false;
        let previousTime: number | undefined;
        const result: ChartDataPoint[] = [];
        for (const sample of ordered) {
          const time = timestampDate(sample.timestamp!).getTime();
          const value = pick(sample)?.state;
          if (value?.case !== 'measured' || typeof value.value !== 'number' || !Number.isFinite(value.value)) { missed = true; continue; }
          const gapBefore = result.length > 0 && (missed || Boolean(cadenceMs && previousTime !== undefined && time - previousTime > cadenceMs * 1.5));
          result.push({ timestamp: toIso(sample.timestamp), value: value.value, ...(gapBefore ? { gapBefore: true } : {}) });
          missed = false;
          previousTime = time;
        }
        return result;
      };

      // GPU observations have their own, slower cadence. A cycle timestamp
      // cannot establish when an absent or repeated GPU reading was measured.
      const gpuSeries = (): ChartDataPoint[] => {
        const observedTimes = [...new Set(ordered.flatMap(sample => {
          const gpu = sample.gpu;
          if (gpu?.state.case !== 'measured' || !Number.isFinite(gpu.state.value) || !gpu.observedAt) return [];
          const time = timestampDate(gpu.observedAt).getTime();
          return Number.isFinite(time) ? [time] : [];
        }))].sort((a, b) => a - b);
        const intervals = observedTimes.slice(1).map((time, index) => time - observedTimes[index]!).filter(delta => delta > 0).sort((a, b) => a - b);
        const gpuCadence = intervals.length >= 2 ? intervals[Math.floor((intervals.length - 1) / 2)]! : 0;
        const result: ChartDataPoint[] = [];
        let gap = false;
        let lastTime = -Infinity;
        for (const sample of ordered) {
          const gpu = sample.gpu;
          // No collector row in a cycle is not a failed observation; GPU
          // collection can be slower than the scheduler's other collectors.
          if (!gpu) continue;
          if (gpu?.state.case !== 'measured' || !Number.isFinite(gpu.state.value) || !gpu.observedAt) {
            gap = true;
            continue;
          }
          const time = timestampDate(gpu.observedAt).getTime();
          if (!Number.isFinite(time)) { gap = true; continue; }
          if (time <= lastTime) continue;
          const gapBefore = result.length > 0 && (gap || Boolean(gpuCadence && time - lastTime > gpuCadence * 1.5));
          result.push({ timestamp: toIso(gpu.observedAt), value: gpu.state.value, ...(gapBefore ? { gapBefore: true } : {}) });
          lastTime = time;
          gap = false;
        }
        return result;
      };


      setMetricHistory(prev => {
        const base = ensureHistoryBase(prev);
        return {
          ...base,
          missingTimestampCount: data.samples.length - samples.length,
          windowStart: data.windowStart ? timestampDate(data.windowStart).toISOString() : undefined,
          windowEnd: data.windowEnd ? timestampDate(data.windowEnd).toISOString() : undefined,
          windowSeconds: data.windowSeconds,
          sampleIntervalSeconds: cadenceMs / 1000,
          cpu: series(sample => sample.cpu),
			cpuContextSwitches: samples.filter(sample => sample.cpuContextSwitchesPerSecond?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuContextSwitchesPerSecond?.state.value) })),
			cpuInterrupts: samples.filter(sample => sample.cpuInterruptsPerSecond?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuInterruptsPerSecond?.state.value) })),
			cpuNormalizedLoad1: samples.filter(sample => sample.cpuNormalizedLoad1?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuNormalizedLoad1?.state.value) })),
			cpuNormalizedLoad5: samples.filter(sample => sample.cpuNormalizedLoad5?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuNormalizedLoad5?.state.value) })),
			cpuRunQueue: samples.filter(sample => sample.cpuRunQueueDepth?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuRunQueueDepth?.state.value) })),
			cpuStallSome: samples.filter(sample => sample.cpuStallSomeAvg10?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuStallSomeAvg10?.state.value) })),
			cpuStallFull: samples.filter(sample => sample.cpuStallFullAvg10?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuStallFullAvg10?.state.value) })),
			cpuCoreImbalance: samples.filter(sample => sample.cpuCoreImbalanceIndex?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuCoreImbalanceIndex?.state.value) })),
			cpuModeIowait: samples.filter(sample => sample.cpuModeIowait?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuModeIowait?.state.value) })),
			cpuModeSteal: samples.filter(sample => sample.cpuModeSteal?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.cpuModeSteal?.state.value) })),
          memory: series(sample => sample.memory),
          swap: samples
            .filter(sample => sample.swap?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.swap?.state.value) })),
          swapTraffic: samples
            .filter(sample => sample.swapTraffic?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.swapTraffic?.state.value) })),
          majorFaults: samples
            .filter(sample => sample.majorFaults?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.majorFaults?.state.value) })),
          fragmentation: samples
            .filter(sample => sample.fragmentationIndex?.state.case === 'measured')
            .map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.fragmentationIndex?.state.value) })),
          kernelSlab: samples.filter(sample => sample.kernelMemory?.slabBytes?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.kernelMemory?.slabBytes?.state.value) })),
          kernelReclaimable: samples.filter(sample => sample.kernelMemory?.reclaimableBytes?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.kernelMemory?.reclaimableBytes?.state.value) })),
          kernelUnreclaimable: samples.filter(sample => sample.kernelMemory?.unreclaimableBytes?.state.case === 'measured').map(sample => ({ timestamp: toIso(sample.timestamp), value: Number(sample.kernelMemory?.unreclaimableBytes?.state.value) })),
          network: series(sample => sample.connections),
          gpu: gpuSeries()
        };
      });
    } catch (err) {
      console.error('API call failed for /metrics/timeline:', err);
      if (!mountedRef.current || request !== requestRef.current) return;
      setError(toApiError(err));
    }
  }, [setError]);

  const appendGpuPoint = useCallback((timestamp: string, value: number, gapBefore = false) => {
    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      const last = base.gpu?.at(-1)?.timestamp;
      if (last === timestamp || (last && Number.isFinite(Date.parse(last)) && Number.isFinite(Date.parse(timestamp)) && Date.parse(timestamp) < Date.parse(last))) return base;
      return {
        ...base,
        gpu: appendHistoryPoint(base.gpu, { timestamp, value, ...(gapBefore ? { gapBefore: true } : {}) })
      };
    });
  }, []);

  const appendDiskPoints = useCallback((timestamp: string, readRate: number, writeRate: number, gapBefore = false) => {
    setMetricHistory(prev => {
      const base = ensureHistoryBase(prev);
      if (base.diskRead?.at(-1)?.timestamp === timestamp && base.diskWrite?.at(-1)?.timestamp === timestamp) return base;
      return {
        ...base,
        diskRead: Number.isFinite(readRate)
          ? appendHistoryPoint(base.diskRead, { timestamp, value: readRate, ...(gapBefore ? { gapBefore: true } : {}) })
          : base.diskRead,
        diskWrite: Number.isFinite(writeRate)
          ? appendHistoryPoint(base.diskWrite, { timestamp, value: writeRate, ...(gapBefore ? { gapBefore: true } : {}) })
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

  const invalidatePending = useCallback(() => { requestRef.current += 1; }, []);
  const clearHistory = useCallback(() => { invalidatePending(); setMetricHistory(null); setError(null); }, [invalidatePending, setError]);

  return {
    metricHistory,
    fetchMetricsTimeline,
    clearHistory,
    invalidatePending,
    appendGpuPoint,
    appendDiskPoints,
    appendDiskUsagePoint
  };
};
