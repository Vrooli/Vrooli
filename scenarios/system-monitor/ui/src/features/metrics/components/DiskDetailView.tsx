import { useMemo, useState, useCallback, useEffect, useRef } from 'react';
import type { ChangeEvent } from 'react';
import { HardDrive } from 'lucide-react';

import { protoFetch } from '../../../shared/api/apiFetch';
import { parseDiskDetailResponse } from '../../../shared/api/proto-contracts';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { create } from '@bufbuild/protobuf';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { DiskInfoSchema } from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';
import { DetailRow } from '../../../shared/components/DetailRow';
import { formatMbPerSecond, formatPercentage, formatTimeLabel, getUtilizationColor } from '../../../shared/utils/formatters';
import type {
  DetailedMetrics,
  MetricHistory,
  StorageIOInfo,
  DiskInfo,
  DiskDetailResponse,
  DiskPartitionInfo,
  DiskUsageEntry,
  MetricValue,
  APIError
} from '../../../types';
import { DetailSection, MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { buildSingleSeriesData, combineDiskSeries } from '../../../shared/utils/chartData';
import { buildDiskUsageCard } from './MetricRenderHelpers';
import { fetchDiskReadActivity } from '../diskReadActivity';
import type { DiskReadActivity } from '../diskReadActivity';

export interface DiskDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  storageIO?: StorageIOInfo | null;
  metricHistory: MetricHistory | null;
  historyError?: APIError | null;
  onRetryHistory?: () => void;
  diskLastUpdated?: string;
  onBack: () => void;
}

const storageMetricLabel = (metric: MetricValue | undefined, unit: string): string => {
  if (!metric?.state?.case) return 'Not yet sampled';
  if (metric.state.case === 'measured') return `${metric.state.value.toFixed(2)}${unit ? ` ${unit}` : ''}`;
  if (metric.state.case === 'notYetSampledReason') return `Waiting: ${metric.state.value}`;
  if (metric.state.case === 'unsupportedReason') return `Not collected: ${metric.state.value}`;
  if (metric.state.case === 'failedError') return `Unavailable: ${metric.state.value}`;
  return `${metric.state.case}: ${String(metric.state.value || 'unavailable')}`;
};

export const DiskDetailView = ({ detailedMetrics, storageIO, metricHistory, historyError, onRetryHistory, diskLastUpdated, onBack }: DiskDetailViewProps) => {
  const DEFAULT_DEPTH = 2;
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const searchParamsRef = useRef(searchParams);
  const locationPathRef = useRef(location.pathname);
  const navigateRef = useRef(navigate);
  searchParamsRef.current = searchParams;
  locationPathRef.current = location.pathname;
  navigateRef.current = navigate;
  const [initialDiskRequest] = useState(() => {
    const requestedDepth = Number(searchParams.get('disk_depth'));
    return {
      mount: searchParams.get('disk_mount') || '/',
      depth: Number.isInteger(requestedDepth) && requestedDepth >= 1 && requestedDepth <= 4 ? requestedDepth : DEFAULT_DEPTH,
      includeFiles: searchParams.get('disk_files') === 'true',
    };
  });
  const [diskDetails, setDiskDetails] = useState<DiskDetailResponse | null>(null);
  const [selectedMount, setSelectedMount] = useState<string>(initialDiskRequest.mount);
  const [depth, setDepth] = useState<number>(initialDiskRequest.depth);
  const [includeFiles, setIncludeFiles] = useState<boolean>(initialDiskRequest.includeFiles);
  const [detailsLoading, setDetailsLoading] = useState<boolean>(false);
  const [detailsError, setDetailsError] = useState<string | null>(null);
  const activeRequestRef = useRef<AbortController | null>(null);
  const failedScanRef = useRef<{ mount: string; depth: number; includeFiles: boolean } | null>(null);
  const [readActivity, setReadActivity] = useState<DiskReadActivity | null>(null);
  const [readActivityError, setReadActivityError] = useState<string | null>(null);
  const [readActivityLoading, setReadActivityLoading] = useState<boolean>(false);
  const readActivityRequestRef = useRef<AbortController | null>(null);

  const diskUsage = detailedMetrics?.memoryDetails?.diskUsage;
  const diskIoHistory = useMemo(
    () => combineDiskSeries(metricHistory?.diskRead, metricHistory?.diskWrite),
    [metricHistory?.diskRead, metricHistory?.diskWrite]
  );
  const diskUsageHistory = useMemo(() => buildSingleSeriesData(metricHistory?.diskUsage), [metricHistory?.diskUsage]);
  const historyStatus = historyError && !metricHistory?.windowStart ? 'error' : metricHistory === null ? 'loading' : 'ready';
  const fileDescriptors = detailedMetrics?.systemDetails?.fileDescriptors;
  const inotifyWatchers = detailedMetrics?.systemDetails?.inotifyWatchers;
  const watchersSupported = inotifyWatchers?.supported ?? true;
  const watcherPercent = inotifyWatchers && Number.isFinite(inotifyWatchers.watchesPercent)
    ? inotifyWatchers.watchesPercent
    : undefined;
  const watcherInstancePercent = inotifyWatchers && Number.isFinite(inotifyWatchers.instancesPercent)
    ? inotifyWatchers.instancesPercent
    : undefined;
  const fetchDiskDetails = useCallback(
    async (mount: string, nextDepth: number, includeFilesValue: boolean) => {
      const nextSearch = new URLSearchParams(searchParamsRef.current);
      nextSearch.set('disk_mount', mount);
      nextSearch.set('disk_depth', String(nextDepth));
      nextSearch.set('disk_files', String(includeFilesValue));
      searchParamsRef.current = nextSearch;
      const query = nextSearch.toString();
      void navigateRef.current({ pathname: locationPathRef.current, search: query ? `?${query}` : '' }, { replace: true });
      if (activeRequestRef.current) {
        activeRequestRef.current.abort();
      }
      const controller = new AbortController();
      activeRequestRef.current = controller;
      failedScanRef.current = { mount, depth: nextDepth, includeFiles: includeFilesValue };
      setDetailsLoading(true);
      setDetailsError(null);
      try {
        const params = new URLSearchParams();
        if (mount) {
          params.set('mount', mount);
        }
        params.set('depth', String(nextDepth));
        params.set('limit', '8');
        if (includeFilesValue) {
          params.set('include_files', 'true');
        }

        const data = await protoFetch(`/metrics/disk/details?${params.toString()}`, parseDiskDetailResponse, {
          signal: controller.signal
        });
        if (activeRequestRef.current !== controller) return;
        setDiskDetails(data);
        setSelectedMount(data.activeMount || mount);
        setDepth(data.depth);
        setIncludeFiles(includeFilesValue);
        setDetailsError(null);
        failedScanRef.current = null;
        activeRequestRef.current = null;
        setDetailsLoading(false);
      } catch (error) {
        if (activeRequestRef.current !== controller) return;
        activeRequestRef.current = null;
        setDetailsError(error instanceof Error ? error.message : String(error));
        setDetailsLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    void fetchDiskDetails(initialDiskRequest.mount, initialDiskRequest.depth, initialDiskRequest.includeFiles);
    return () => {
      const controller = activeRequestRef.current;
      activeRequestRef.current = null;
      controller?.abort();
    };
  }, [fetchDiskDetails, initialDiskRequest]);

  useEffect(() => () => {
    const controller = readActivityRequestRef.current;
    readActivityRequestRef.current = null;
    controller?.abort();
  }, []);

  const partitions = useMemo(() => diskDetails?.partitions ?? [], [diskDetails?.partitions]);
  const activePartition = useMemo<DiskPartitionInfo | null>(() => {
    if (partitions.length === 0) {
      return null;
    }
    return partitions.find(partition => partition.mountPoint === selectedMount) ?? null;
  }, [partitions, selectedMount]);

  const summaryDiskInfo: DiskInfo | undefined = activePartition
    ? create(DiskInfoSchema, {
        used: activePartition.usedBytes,
        total: activePartition.sizeBytes,
        percent: activePartition.usePercent
      })
    : selectedMount === '/'
      ? diskUsage
      : undefined;

  const selectedMountLabel = activePartition?.mountPoint ?? selectedMount;
  const deviceLabel = activePartition?.device ? `Device ${activePartition.device}` : undefined;

  const lastUpdated = diskDetails?.timestamp
    ? timestampDate(diskDetails.timestamp).toISOString()
    : diskLastUpdated ?? (detailedMetrics?.timestamp ? timestampDate(detailedMetrics.timestamp).toISOString() : undefined);

  const subheadParts: string[] = [];
  if (deviceLabel) {
    subheadParts.push(deviceLabel);
  }
  if (lastUpdated) {
    subheadParts.push(`Last scan ${formatTimeLabel(lastUpdated)}`);
  }
  if (detailsLoading && failedScanRef.current) {
    const request = failedScanRef.current;
    subheadParts.push(`Scanning ${request.mount} at depth ${request.depth}${request.includeFiles ? ' with large-file discovery' : ''}`);
  }

  const topDirectories = diskDetails?.topDirectories ?? [];
  const largestFiles = diskDetails?.largestFiles ?? [];
  const directoryAttributionUnavailable = diskDetails?.notes.some(note => note.startsWith('Directory attribution is unavailable')) ?? false;
  const fileAttributionUnavailable = diskDetails?.notes.some(note => note.startsWith('Largest-file attribution is unavailable')) ?? false;

  const handleMountSelect = (mountPoint: string) => {
    void fetchDiskDetails(mountPoint, depth, includeFiles);
  };

  const handleDepthChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextDepth = Number(event.target.value);
    void fetchDiskDetails(selectedMount, nextDepth, includeFiles);
  };

  const handleRefresh = () => {
    void fetchDiskDetails(selectedMount, depth, includeFiles);
  };

  const handleScanLargestFiles = () => {
    void fetchDiskDetails(selectedMount, depth, true);
  };

  const handleRetryScan = () => {
    const failedScan = failedScanRef.current;
    if (failedScan) void fetchDiskDetails(failedScan.mount, failedScan.depth, failedScan.includeFiles);
  };

  const handleReadActivity = useCallback(async () => {
    readActivityRequestRef.current?.abort();
    const controller = new AbortController();
    readActivityRequestRef.current = controller;
    setReadActivityLoading(true);
    setReadActivityError(null);
    try {
      const result = await fetchDiskReadActivity(controller.signal);
      if (readActivityRequestRef.current !== controller) return;
      setReadActivity(result);
    } catch (error) {
      if (readActivityRequestRef.current !== controller || controller.signal.aborted) return;
      setReadActivityError(error instanceof Error ? error.message : String(error));
    } finally {
      if (readActivityRequestRef.current === controller) {
        readActivityRequestRef.current = null;
        setReadActivityLoading(false);
      }
    }
  }, []);

  const handleStopScan = () => {
    if (activeRequestRef.current) {
      activeRequestRef.current.abort();
      activeRequestRef.current = null;
      setDetailsLoading(false);
        setDetailsError('Scan stopped before it completed. The last successful results remain available.');
    }
  };

  return (
    <MetricDetailLayout
      layoutId="disk"
      title="DISK PERFORMANCE"
      icon={<HardDrive size={22} />}
      headline={summaryDiskInfo
        ? `${summaryDiskInfo.percent.toFixed(1)}% utilized on ${selectedMountLabel}`
        : selectedMount !== '/' && !activePartition
          ? `Usage for ${selectedMountLabel} is unavailable`
          : 'Awaiting disk telemetry'}
      subhead={subheadParts.length > 0 ? subheadParts.join(' \u2022 ') : undefined}
      onBack={onBack}
    >
      {historyError && <DetailSection id="disk-history-refresh-error" title="History refresh failed"><div className="card mb-lg" role="alert">
        <p>History could not be refreshed: {historyError.error}. {metricHistory?.windowStart ? 'The previous dated history remains visible.' : 'No history has been received for this time range.'}</p>
        {onRetryHistory && <button type="button" className="btn btn-action" onClick={onRetryHistory}>Retry history</button>}
      </div></DetailSection>}
      <DetailSection id="disk-usage-history" title="Disk usage history"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">System disk usage history</h3>
          <div className="card-subtitle">
            Utilization over the selected time range for the system monitored path. The mount scan below is a current snapshot; it does not rewrite this history.
          </div>
        </div>
        <MetricLineChart
          status={historyStatus}
          errorMessage={historyError?.error}
          seriesLabel="disk"
          data={diskUsageHistory.map(point => ({ timestamp: point.timestamp, value: point.value, gapBefore: point.gapBefore }))}
          lines={[{ dataKey: 'value', name: 'Disk Usage', color: 'var(--color-info)' }]}
          unit="%"
          valueFormatter={formatPercentage}
          yDomain={[0, 100]}
          height={260}
        />
      </div></DetailSection>
      <DetailSection id="disk-io-history" title="Disk I/O history"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Disk I/O history</h3>
          <div className="card-subtitle">
            Read and write rates are summed across physical devices. They are not attributed to the selected mount or a process.
          </div>
        </div>
        <MetricLineChart
          status={historyStatus}
          errorMessage={historyError?.error}
          seriesLabel="disk"
          data={diskIoHistory.map(point => ({ timestamp: point.timestamp, read: point.read, write: point.write, gapBefore: point.gapBefore }))}
          lines={[
            { dataKey: 'read', name: 'Read Throughput', color: 'var(--color-primary)' },
            { dataKey: 'write', name: 'Write Throughput', color: 'var(--color-warning)' }
          ]}
          unit=" MB/s"
          valueFormatter={formatMbPerSecond}
          yDomain={['auto', 'auto']}
        />
        <div className="card-subtitle">Breaks in either trace mark an interval with no trustworthy measurement.</div>
      </div></DetailSection>

      <DetailSection id="disk-hot-writers" title="Hot writers"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Process read activity</h3>
          <div className="card-subtitle">
            On demand, System Monitor compares two bounded host-wide process samples about two seconds apart. It is not attributed to {selectedMountLabel}.
          </div>
        </div>
        <div className="flex-row-baseline">
          <button type="button" className="btn btn-action" onClick={() => { void handleReadActivity(); }} disabled={readActivityLoading}>
            {readActivityLoading ? 'Sampling process reads…' : readActivityError ? 'Retry hot-writer sample' : 'Find hot writers (2-second sample)'}
          </button>
          {readActivityLoading && <span role="status" aria-live="polite" className="text-dim-xs">Sampling up to 4,096 processes. This does not change system state.</span>}
        </div>
        {readActivityError && <div role="alert" className="card-subtitle">
          Read activity could not be refreshed: {readActivityError}. {readActivity ? 'The previous dated sample remains visible.' : 'No process sample is available yet.'}
        </div>}
        {readActivity && (
          <div className="flex-col-gap-sm" aria-label="Process read activity results">
            <div className="card-subtitle">
              {readActivity.status.replaceAll('_', ' ')} · {readActivity.windowSeconds?.toFixed(1) ?? '—'} second sample · {readActivity.matched ?? 0} matched · {readActivity.permissionDenied ?? 0} permission-limited · {readActivity.unreadable ?? 0} unreadable · {readActivity.disappeared ?? 0} disappeared · {readActivity.unmatched ?? 0} unmatched
              {readActivity.lastObservedAt ? ` · last observed ${formatTimeLabel(readActivity.lastObservedAt)}` : ''}
            </div>
            {(readActivity.reason || readActivity.truncated || readActivity.rowsTruncated) && (
              <div className="card-subtitle" role="status">
                {readActivity.reason ?? 'The process list or result rows reached their observation bound.'}
                {readActivity.truncated ? ' Process inventory was truncated.' : ''}
                {readActivity.rowsTruncated ? ' Only the highest-rate rows are shown.' : ''}
              </div>
            )}
            {readActivity.storageRows.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="data-table">
                  <thead><tr><th>Process</th><th>Owner</th><th>Storage read MB/s</th><th>Read return MB/s</th></tr></thead>
                  <tbody>{readActivity.storageRows.map(row => (
                    <tr key={`${row.pid}-${row.intervalSeconds}`}>
                      <td className="text-bright">{row.name} <span className="text-dim-xs">PID {row.pid}</span></td>
                      <td>{row.owner || 'Unattributed'}</td>
                      <td>{(row.storageBytesPerSecond / 1_000_000).toFixed(2)}</td>
                      <td>{(row.returnedBytesPerSecond / 1_000_000).toFixed(2)}</td>
                    </tr>
                  ))}</tbody>
                </table>
              </div>
            ) : (
              <div className="text-muted">{readActivity.status === 'measured' ? 'No matched process read rates were detected in this sample.' : 'No process rows were available; the sample is not a confirmed zero.'}</div>
            )}
            <details>
              <summary>How this sample is measured</summary>
              <p className="text-dim-xs">{readActivity.semantics}</p>
            </details>
          </div>
        )}
      </div></DetailSection>

      <DetailSection id="storage-overview" title="Storage overview"><div className="metric-grid-auto-lg">
        {buildDiskUsageCard(summaryDiskInfo, {
          title: `Usage for ${selectedMountLabel}`,
          subtitle: deviceLabel ?? 'Current usage across monitored volumes'
        })}

        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Storage I/O Snapshot</h3>
            <div className="card-subtitle">
              Queue depth sums in-flight requests across physical devices; CPU I/O wait is not collected here.
            </div>
          </div>
          {storageIO ? (
            <div className="detail-grid detail-grid-md">
              <DetailRow label="Disk Queue Depth" value={storageMetricLabel(storageIO.diskQueueDepthState, 'requests')} />
              <DetailRow label="I/O Wait" value={storageMetricLabel(storageIO.ioWaitState, '%')} valueColor="var(--color-warning)" />
              <DetailRow label="Read Throughput" value={storageMetricLabel(storageIO.readRate, 'MB/s')} />
              <DetailRow label="Write Throughput" value={storageMetricLabel(storageIO.writeRate, 'MB/s')} />
            </div>
          ) : (
            <div className="text-muted">
              Storage I/O metrics unavailable.
            </div>
          )}
        </div>

        {fileDescriptors && (
          <div className="card flex-col-gap-md">
            <div>
              <h3 className="section-heading">File Descriptor Utilization</h3>
              <div className="card-subtitle">
                Tracks open file handles across all services
              </div>
            </div>
          <div className="utilization-header">
            <div className="utilization-value-lg">
              {fileDescriptors.used.toLocaleString()} / {fileDescriptors.max.toLocaleString()}
            </div>
            <div className="utilization-percent" style={{ color: getUtilizationColor(fileDescriptors.percent) }}>
              {fileDescriptors.percent.toFixed(1)}%
            </div>
          </div>
            <div className="progress-bar">
              <div
                className="progress-fill"
                style={{
                  width: `${Math.min(Math.max(fileDescriptors.percent, 0), 100)}%`,
                  background: fileDescriptors.percent >= 70
                    ? getUtilizationColor(fileDescriptors.percent)
                    : 'linear-gradient(90deg, var(--color-primary), var(--color-text-heading))'
                }}
              />
            </div>
            <div className="text-dim-xs">
              Sustained values above 80% risk "too many open files" errors during heavy disk activity.
            </div>
          </div>
        )}

        {inotifyWatchers && (
          <div className="card flex-col-gap-md">
            <div>
              <h3 className="section-heading">Inotify Watcher Utilization</h3>
              <div className="card-subtitle">
                Kernel file watcher instances and watch descriptors in use
              </div>
            </div>
            {watchersSupported ? (
              <>
                <div className="utilization-header">
                  <div className="utilization-value-lg" data-sm-style="sm-style-374ebe9a52">
                    {inotifyWatchers.watchesUsed.toLocaleString()} / {inotifyWatchers.watchesMax.toLocaleString()} watches
                  </div>
                  <div className="utilization-percent" style={{
                    color: watcherPercent !== undefined ? getUtilizationColor(watcherPercent) : 'var(--color-text-secondary)',
                    fontSize: 'var(--text-base)'
                  }}>
                    {watcherPercent !== undefined ? `${watcherPercent.toFixed(1)}%` : '—'}
                  </div>
                </div>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${Math.min(Math.max(watcherPercent ?? 0, 0), 100)}%`,
                      background: watcherPercent !== undefined && watcherPercent >= 70
                        ? getUtilizationColor(watcherPercent)
                        : 'linear-gradient(90deg, var(--color-primary), var(--color-text-heading))'
                    }}
                  />
                </div>
                <div className="utilization-header text-dim-xs">
                  <span>
                    Instances: {inotifyWatchers.instancesUsed.toLocaleString()} / {inotifyWatchers.instancesMax.toLocaleString()}
                  </span>
                  <span style={{ color: watcherInstancePercent !== undefined ? getUtilizationColor(watcherInstancePercent) : 'var(--color-text-secondary)' }}>
                    {watcherInstancePercent !== undefined ? `${watcherInstancePercent.toFixed(1)}%` : '—'}
                  </span>
                </div>
                <div className="text-dim-xs">
                  Keep watcher usage below 80% to avoid hitting Linux inotify limits that break file watchers and dev servers.
                </div>
              </>
            ) : (
              <div className="card-subtitle">
                Inotify watcher metrics are not available on this platform.
              </div>
            )}
          </div>
        )}
      </div></DetailSection>

      {diskDetails?.notes && diskDetails.notes.length > 0 && (
        <DetailSection id="analysis-notes" title="Analysis notes"><div className="card" data-sm-style="sm-style-d370fc4399">
          {diskDetails.notes.map((note, index) => (
            <div key={`${note}-${index}`} data-sm-style="sm-style-84a0f8980c">
              {'\u2022'} {note}
            </div>
          ))}
        </div></DetailSection>
      )}

      <DetailSection id="volume-analysis" title="Volumes and analysis"><div className="metric-grid-auto-lg">
        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Mounted Volumes</h3>
            <div className="card-subtitle">
              Select a mount to drill into its usage profile
            </div>
          </div>
          {partitions.length > 0 ? (
            <div className="flex-col-gap-sm">
              {partitions.map(partition => {
                const isActive = partition.mountPoint === selectedMount;
                const percent = Math.min(Math.max(partition.usePercent, 0), 100);
                return (
                  <button
                    key={`${partition.device}-${partition.mountPoint}`}
                    type="button"
                    className={`partition-btn ${isActive ? 'active' : ''}`}
                    aria-pressed={isActive}
                    onClick={() => { handleMountSelect(partition.mountPoint); }}
                    disabled={detailsLoading && isActive}
                  >
                    <div className="flex-row-baseline">
                      <span className="text-bright font-semibold">{partition.mountPoint}</span>
                      <span className="text-warning text-sm">{percent.toFixed(1)}%</span>
                    </div>
                    <div className="text-dim-xs">{partition.device}</div>
                    <div className="mini-progress-track">
                      <div
                        className="mini-progress-fill"
                        style={{
                          width: `${percent}%`,
                          background: 'linear-gradient(90deg, var(--color-warning), var(--color-error))'
                        }}
                      />
                    </div>
                    <div className="flex-row-baseline text-dim-xs">
                      <span>Used {partition.usedHuman}</span>
                      <span>Free {partition.availableHuman}</span>
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="text-muted">
              {!diskDetails && detailsLoading
                ? 'Loading mounted-volume data…'
                : !diskDetails && detailsError
                  ? 'No successful mount inventory is available for this request.'
                  : diskDetails
                    ? 'No mounted-volume rows were returned; see Analysis notes for any warnings.'
                    : 'Mounted-volume data has not been received.'}
            </div>
          )}
        </div>

        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Analysis Controls</h3>
            <div className="card-subtitle">
              Customize the depth and scope of disk analysis
            </div>
          </div>
          <label className="analysis-select-label">
            Depth
            <select
              value={depth}
              onChange={handleDepthChange}
              disabled={detailsLoading}
              className="analysis-select"
            >
              <option value={1}>Top-level directories</option>
              <option value={2}>Include first sub-level</option>
              <option value={3}>Include two sub-levels</option>
              <option value={4}>Deep scan (slower)</option>
            </select>
          </label>
          <div className="analysis-btn-row">
            <button
              type="button"
              className="btn btn-action"
              onClick={handleRefresh}
              disabled={detailsLoading}
            >
              {detailsLoading ? 'Scanning...' : 'Refresh Scan'}
            </button>
            <button
              type="button"
              className="btn btn-action"
              onClick={handleScanLargestFiles}
              disabled={detailsLoading}
            >
              {includeFiles ? 'Rescan Largest Files' : 'Find Largest Files (>50MB)'}
            </button>
            <button
              type="button"
              className="btn btn-action"
              onClick={handleStopScan}
              disabled={!detailsLoading}
            >
              Stop Scan
            </button>
          </div>
          <div className="text-dim-xs">
            Deeper scans and file discovery may take longer on large volumes.
          </div>
          {detailsLoading && failedScanRef.current && (
            <div role="status" aria-live="polite" className="text-dim-xs">
              Scanning {failedScanRef.current.mount} at depth {failedScanRef.current.depth}{failedScanRef.current.includeFiles ? ', including largest files' : ', directories only'}.
            </div>
          )}
        </div>
      </div></DetailSection>

      {detailsError && (
        <DetailSection id="analysis-error" title="Analysis could not be completed"><div className="card" role="alert" data-sm-style="sm-style-f4d03882e0">
          <p>Disk analysis was not completed for {failedScanRef.current?.mount || selectedMount} at depth {failedScanRef.current?.depth ?? depth}{failedScanRef.current?.includeFiles ? ' with large-file discovery' : ''}: {detailsError}</p>
          {diskDetails ? <p>Still showing the last successful results for {diskDetails.activeMount || selectedMount} at depth {diskDetails.depth}{diskDetails.largestFiles.length > 0 ? ', including large files' : ''}. This failed request did not replace them.</p> : <p>No successful scan is available yet.</p>}
          <button type="button" className="btn btn-action" onClick={handleRetryScan} disabled={detailsLoading || !failedScanRef.current}>Retry this scan</button>
          {!diskDetails && failedScanRef.current && failedScanRef.current.mount !== '/' && (
            <button type="button" className="btn btn-secondary" onClick={() => { setSelectedMount('/'); void fetchDiskDetails('/', depth, false); }} disabled={detailsLoading}>
              Inspect the root mount instead
            </button>
          )}
        </div></DetailSection>
      )}

      <DetailSection id="top-directories" title="Top directories"><div className="card flex-col-gap-md">
        <div data-sm-style="sm-style-a2c4df2a66">
          <div>
            <h3 className="section-heading">Top Directories</h3>
            <div className="card-subtitle">
              Heaviest paths within {selectedMountLabel} (depth {depth})
            </div>
          </div>
        </div>
        {detailsLoading && topDirectories.length === 0 ? (
          <div className="text-muted">Analyzing directory usage...</div>
        ) : topDirectories.length > 0 ? (
          <div data-sm-style="sm-style-d383f0755e">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Path</th>
                  <th>Size</th>
                </tr>
              </thead>
              <tbody>
                {topDirectories.map((entry: DiskUsageEntry) => (
                  <tr key={entry.path}>
                    <td className="text-bright">{entry.path}</td>
                    <td className="text-accent" data-sm-style="sm-style-f8d335ec48">{entry.sizeHuman}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-muted">
            {directoryAttributionUnavailable
              ? 'Directory results are unavailable; see the analysis notes for the read failure.'
              : !diskDetails
                ? 'No successful directory scan is available for this request.'
                : 'No directories exceeded the scan threshold at this depth.'}
          </div>
        )}
      </div></DetailSection>

      <DetailSection id="largest-files" title="Largest files"><div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Largest Files</h3>
          <div className="card-subtitle">
            Files larger than 50 MB within {selectedMountLabel}
          </div>
        </div>
        {includeFiles ? (
          largestFiles.length > 0 ? (
            <div data-sm-style="sm-style-d383f0755e">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>File</th>
                    <th>Size</th>
                  </tr>
                </thead>
                <tbody>
                  {largestFiles.map(entry => (
                    <tr key={entry.path}>
                      <td className="text-bright">{entry.path}</td>
                      <td className="text-accent" data-sm-style="sm-style-f8d335ec48">{entry.sizeHuman}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : detailsLoading ? (
            <div className="text-muted">Scanning for large files...</div>
          ) : (
            <div className="text-muted">
              {fileAttributionUnavailable
                ? 'Largest-file results are unavailable; see the analysis notes for the read failure.'
                : !diskDetails
                  ? 'No successful largest-file scan is available for this request.'
                  : 'No files above 50 MB were detected in this mount.'}
            </div>
          )
        ) : (
          <div className="text-muted">
            Run the "Find Largest Files" scan to surface oversized artifacts.
          </div>
        )}
      </div></DetailSection>

    </MetricDetailLayout>
  );
};
