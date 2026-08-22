import type { MemoryMetrics, ProcessInfo } from '../../../../types';
import { formatOptionalNumber } from '../../../../shared/utils/formatters';

interface MemoryExpansionProps {
  details: MemoryMetrics;
}

export const MemoryExpansion = ({ details }: MemoryExpansionProps) => (
  <div className="metric-details" data-sm-style="sm-style-323fdcc1e0">
    {(details.topProcesses?.length ?? 0) > 0 && (
      <div className="detail-section" data-sm-style="sm-style-91394348ef">
        <h4 data-sm-style="sm-style-c8a6493830">
          Top Processes by Memory:
        </h4>
        <div className="process-list">
          {(details.topProcesses ?? []).slice(0, 5).map((process: ProcessInfo) => (
            <div key={process.pid} data-sm-style="sm-style-d820a0b3ab">
              <span>{process.name} ({process.pid})</span>
              <span data-sm-style="sm-style-392c7463c7">
                {formatOptionalNumber(process.memoryMb)} MB
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" data-sm-style="sm-style-c08663b577">
      <div className="detail-item">
        <span className="detail-label" data-sm-style="sm-style-a6b497e153">
          Swap Usage:
        </span>
        <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
          {formatOptionalNumber(details.swapUsage?.percent)}%
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" data-sm-style="sm-style-a6b497e153">
          Disk Usage:
        </span>
        <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
          {formatOptionalNumber(details.diskUsage?.percent)}%
        </span>
      </div>
    </div>

  </div>
);
