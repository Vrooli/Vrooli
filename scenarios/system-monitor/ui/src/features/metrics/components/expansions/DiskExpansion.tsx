import type { DiskCardDetails } from '../../../../types';
import { formatBytes, formatTime } from '../../../../shared/utils/formatters';

interface DiskExpansionProps {
  details: DiskCardDetails;
}

export const DiskExpansion = ({ details }: DiskExpansionProps) => {
  const diskUsage = details.diskUsage;
  const total = Number(diskUsage?.total ?? 0);
  const used = Number(diskUsage?.used ?? 0);
  const freeBytes = Number.isFinite(total - used) ? total - used : undefined;

  return (
    <div className="metric-details" data-sm-style="sm-style-323fdcc1e0">
      <div className="metric-grid-auto" data-sm-style="sm-style-91394348ef">
        <div className="detail-item">
          <span className="detail-label" data-sm-style="sm-style-a6b497e153">
            Total Capacity:
          </span>
          <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
            {formatBytes(total)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" data-sm-style="sm-style-a6b497e153">
            Used:
          </span>
          <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
            {formatBytes(used)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" data-sm-style="sm-style-a6b497e153">
            Free:
          </span>
          <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
            {formatBytes(freeBytes)}
          </span>
        </div>
        <div className="detail-item">
          <span className="detail-label" data-sm-style="sm-style-a6b497e153">
            Utilization:
          </span>
          <span className="detail-value" data-sm-style="sm-style-38c5f4e767">
            {diskUsage?.percent?.toFixed(1) ?? '—'}%
          </span>
        </div>
      </div>

      {details.storageIO && (
        <div className="metric-grid-auto" data-sm-style="sm-style-c08663b577">
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              Disk Queue Depth:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {details.storageIO.diskQueueDepth?.toFixed(2) ?? '—'}
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              I/O Wait:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {details.storageIO.ioWaitPercent?.toFixed(1) ?? '—'}%
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              Read Throughput:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {details.storageIO.readMbPerSec?.toFixed(2) ?? '—'} MB/s
            </span>
          </div>
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              Write Throughput:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {details.storageIO.writeMbPerSec?.toFixed(2) ?? '—'} MB/s
            </span>
          </div>
        </div>
      )}

      {details.lastUpdated && (
        <div data-sm-style="sm-style-988eeae870">
          Updated {formatTime(details.lastUpdated)}
        </div>
      )}
    </div>
  );
};
