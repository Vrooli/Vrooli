import { useMemo, useState, useCallback, useEffect, useRef } from 'react';
import type { ChangeEvent } from 'react';
import { HardDrive } from 'lucide-react';

import { apiFetch } from '../../../shared/api/apiFetch';
import { DetailRow } from '../../../shared/components/DetailRow';
import { formatMbPerSecond, formatPercentage, formatTimeLabel, getUtilizationColor } from '../../../shared/utils/formatters';
import type {
  DetailedMetrics,
  MetricHistory,
  StorageIOInfo,
  DiskInfo,
  DiskDetailResponse,
  DiskPartitionInfo,
  DiskUsageEntry
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import {
  buildSingleSeriesData,
  combineDiskSeries,
  buildDiskUsageCard
} from './metricHelpers';

export interface DiskDetailViewProps {
  detailedMetrics: DetailedMetrics | null;
  storageIO?: StorageIOInfo | null;
  metricHistory: MetricHistory | null;
  diskLastUpdated?: string;
  onBack: () => void;
}

export const DiskDetailView = ({ detailedMetrics, storageIO, metricHistory, diskLastUpdated, onBack }: DiskDetailViewProps) => {
  const DEFAULT_DEPTH = 2;
  const [diskDetails, setDiskDetails] = useState<DiskDetailResponse | null>(null);
  const [selectedMount, setSelectedMount] = useState<string>('/');
  const [depth, setDepth] = useState<number>(DEFAULT_DEPTH);
  const [includeFiles, setIncludeFiles] = useState<boolean>(false);
  const [detailsLoading, setDetailsLoading] = useState<boolean>(false);
  const [detailsError, setDetailsError] = useState<string | null>(null);
  const activeRequestRef = useRef<AbortController | null>(null);

  const diskUsage = detailedMetrics?.memory_details?.disk_usage;
  const diskIoHistory = useMemo(
    () => combineDiskSeries(metricHistory?.diskRead, metricHistory?.diskWrite),
    [metricHistory?.diskRead, metricHistory?.diskWrite]
  );
  const diskUsageHistory = useMemo(() => buildSingleSeriesData(metricHistory?.diskUsage), [metricHistory?.diskUsage]);
  const fileDescriptors = detailedMetrics?.system_details?.file_descriptors;
  const inotifyWatchers = detailedMetrics?.system_details?.inotify_watchers;
  const watchersSupported = inotifyWatchers?.supported ?? true;
  const watcherPercent = inotifyWatchers && Number.isFinite(inotifyWatchers.watches_percent)
    ? inotifyWatchers.watches_percent
    : undefined;
  const watcherInstancePercent = inotifyWatchers && Number.isFinite(inotifyWatchers.instances_percent)
    ? inotifyWatchers.instances_percent
    : undefined;
  const fetchDiskDetails = useCallback(
    async (mount: string, nextDepth: number, includeFilesValue: boolean) => {
      if (activeRequestRef.current) {
        activeRequestRef.current.abort();
      }
      setDetailsLoading(true);
      setDetailsError(null);
      const controller = new AbortController();
      activeRequestRef.current = controller;
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

        const data = await apiFetch<DiskDetailResponse>(`/metrics/disk/details?${params.toString()}`, {
          signal: controller.signal
        });
        setDiskDetails(data);
        setSelectedMount(data.active_mount || mount);
        setDepth(data.depth);
        setIncludeFiles(includeFilesValue);
        setDetailsError(null);
        activeRequestRef.current = null;
      } catch (error) {
        activeRequestRef.current = null;
        if ((error as { name?: string })?.name === 'AbortError') {
          setDetailsError('Scan cancelled');
        } else {
          setDetailsError(error instanceof Error ? error.message : String(error));
        }
      } finally {
        setDetailsLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    fetchDiskDetails('/', DEFAULT_DEPTH, false);
    return () => {
      if (activeRequestRef.current) {
        activeRequestRef.current.abort();
      }
    };
  }, [fetchDiskDetails]);

  const partitions = useMemo(() => diskDetails?.partitions ?? [], [diskDetails?.partitions]);
  const activePartition = useMemo<DiskPartitionInfo | null>(() => {
    if (partitions.length === 0) {
      return null;
    }
    const exact = partitions.find(partition => partition.mount_point === selectedMount);
    return exact ?? partitions[0] ?? null;
  }, [partitions, selectedMount]);

  const summaryDiskInfo: DiskInfo | undefined = activePartition
    ? {
        used: activePartition.used_bytes,
        total: activePartition.size_bytes,
        percent: activePartition.use_percent
      }
    : diskUsage;

  const selectedMountLabel = activePartition?.mount_point ?? selectedMount;
  const deviceLabel = activePartition?.device ? `Device ${activePartition.device}` : undefined;

  const lastUpdated = diskDetails?.timestamp
    ? diskDetails.timestamp
    : diskLastUpdated ?? detailedMetrics?.timestamp;

  const subheadParts: string[] = [];
  if (deviceLabel) {
    subheadParts.push(deviceLabel);
  }
  if (lastUpdated) {
    subheadParts.push(`Last scan ${formatTimeLabel(lastUpdated)}`);
  }
  if (detailsLoading) {
    subheadParts.push('Analyzing...');
  }

  const topDirectories = diskDetails?.top_directories ?? [];
  const largestFiles = diskDetails?.largest_files ?? [];

  const handleMountSelect = (mountPoint: string) => {
    setSelectedMount(mountPoint);
    fetchDiskDetails(mountPoint, depth, includeFiles);
  };

  const handleDepthChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextDepth = Number(event.target.value);
    setDepth(nextDepth);
    fetchDiskDetails(selectedMount, nextDepth, includeFiles);
  };

  const handleRefresh = () => {
    fetchDiskDetails(selectedMount, depth, includeFiles);
  };

  const handleScanLargestFiles = () => {
    setIncludeFiles(true);
    fetchDiskDetails(selectedMount, depth, true);
  };

  const handleStopScan = () => {
    if (activeRequestRef.current) {
      activeRequestRef.current.abort();
      activeRequestRef.current = null;
      setDetailsLoading(false);
    }
  };

  return (
    <MetricDetailLayout
      title="DISK PERFORMANCE"
      icon={<HardDrive size={22} />}
      headline={summaryDiskInfo ? `${summaryDiskInfo.percent.toFixed(1)}% utilized on ${selectedMountLabel}` : 'Awaiting disk telemetry'}
      subhead={subheadParts.length > 0 ? subheadParts.join(' \u2022 ') : undefined}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        data={diskIoHistory.map(point => ({ timestamp: point.timestamp, read: point.read, write: point.write }))}
        lines={[
          { dataKey: 'read', name: 'Read Throughput', color: 'var(--color-accent)' },
          { dataKey: 'write', name: 'Write Throughput', color: 'var(--color-warning)' }
        ]}
        unit=" MB/s"
        valueFormatter={formatMbPerSecond}
        yDomain={['auto', 'auto']}
      />

      <div className="metric-grid-auto-lg">
        {buildDiskUsageCard(summaryDiskInfo, {
          title: `Usage for ${selectedMountLabel}`,
          subtitle: deviceLabel ?? 'Current usage across monitored volumes'
        })}

        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Storage I/O Snapshot</h3>
            <div className="card-subtitle">
              Real-time disk queue and wait metrics
            </div>
          </div>
          {storageIO ? (
            <div className="detail-grid detail-grid-md">
              <DetailRow label="Disk Queue Depth" value={storageIO.disk_queue_depth?.toFixed(2) ?? '\u2014'} />
              <DetailRow label="I/O Wait" value={`${storageIO.io_wait_percent?.toFixed(1) ?? '\u2014'}%`} valueColor="var(--color-warning)" />
              <DetailRow label="Read Throughput" value={`${storageIO.read_mb_per_sec?.toFixed(2) ?? '\u2014'} MB/s`} />
              <DetailRow label="Write Throughput" value={`${storageIO.write_mb_per_sec?.toFixed(2) ?? '\u2014'} MB/s`} />
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
                    : 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))'
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
                  <div className="utilization-value-lg" style={{ fontSize: 'var(--font-size-lg)' }}>
                    {inotifyWatchers.watches_used.toLocaleString()} / {inotifyWatchers.watches_max.toLocaleString()} watches
                  </div>
                  <div className="utilization-percent" style={{
                    color: watcherPercent !== undefined ? getUtilizationColor(watcherPercent) : 'var(--color-text-dim)',
                    fontSize: 'var(--font-size-md)'
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
                        : 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))'
                    }}
                  />
                </div>
                <div className="utilization-header text-dim-xs">
                  <span>
                    Instances: {inotifyWatchers.instances_used.toLocaleString()} / {inotifyWatchers.instances_max.toLocaleString()}
                  </span>
                  <span style={{ color: watcherInstancePercent !== undefined ? getUtilizationColor(watcherInstancePercent) : 'var(--color-text-dim)' }}>
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
      </div>

      <div className="metric-grid-auto-lg">
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
                const isActive = partition.mount_point === selectedMount;
                const percent = Math.min(Math.max(partition.use_percent, 0), 100);
                return (
                  <button
                    key={`${partition.device}-${partition.mount_point}`}
                    type="button"
                    className={`partition-btn ${isActive ? 'active' : ''}`}
                    onClick={() => handleMountSelect(partition.mount_point)}
                    disabled={detailsLoading && isActive}
                  >
                    <div className="flex-row-baseline">
                      <span className="text-bright font-semibold">{partition.mount_point}</span>
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
                      <span>Used {partition.used_human}</span>
                      <span>Free {partition.available_human}</span>
                    </div>
                  </button>
                );
              })}
            </div>
          ) : (
            <div className="text-muted">
              Partition information is unavailable on this platform.
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
        </div>
      </div>

      {detailsError && (
        <div className="card" style={{ color: 'var(--color-error)', letterSpacing: '0.08em' }}>
          Failed to analyze disk usage: {detailsError}
        </div>
      )}

      {diskDetails?.notes && diskDetails.notes.length > 0 && (
        <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)', color: 'var(--color-warning)' }}>
          {diskDetails.notes.map((note, index) => (
            <div key={`${note}-${index}`} style={{ letterSpacing: '0.08em' }}>
              {'\u2022'} {note}
            </div>
          ))}
        </div>
      )}

      <div className="card flex-col-gap-md">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', flexWrap: 'wrap', gap: 'var(--spacing-sm)' }}>
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
          <div style={{ overflowX: 'auto' }}>
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
                    <td className="text-accent" style={{ whiteSpace: 'nowrap' }}>{entry.size_human}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-muted">
            No directories exceeded the scan threshold at this depth.
          </div>
        )}
      </div>

      <div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Largest Files</h3>
          <div className="card-subtitle">
            Files larger than 50 MB within {selectedMountLabel}
          </div>
        </div>
        {includeFiles ? (
          largestFiles.length > 0 ? (
            <div style={{ overflowX: 'auto' }}>
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
                      <td className="text-accent" style={{ whiteSpace: 'nowrap' }}>{entry.size_human}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : detailsLoading ? (
            <div className="text-muted">Scanning for large files...</div>
          ) : (
            <div className="text-muted">
              No files above 50 MB were detected in this mount.
            </div>
          )
        ) : (
          <div className="text-muted">
            Run the "Find Largest Files" scan to surface oversized artifacts.
          </div>
        )}
      </div>

      <div className="card flex-col-gap-md">
        <div>
          <h3 className="section-heading">Disk Usage History</h3>
          <div className="card-subtitle">
            Utilization trend across the observation window
          </div>
        </div>
        <MetricLineChart
          data={diskUsageHistory.map(point => ({ timestamp: point.timestamp, value: point.value }))}
          lines={[{ dataKey: 'value', name: 'Disk Usage', color: 'var(--color-info)' }]}
          unit="%"
          valueFormatter={formatPercentage}
          yDomain={[0, 100]}
          height={260}
        />
      </div>
    </MetricDetailLayout>
  );
};
