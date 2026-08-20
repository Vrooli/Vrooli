import type { CPUMetrics, ProcessInfo } from '../../../../types';
import { formatOptionalNumber } from '../../../../shared/utils/formatters';

interface CpuExpansionProps {
  details: CPUMetrics;
}

export const CpuExpansion = ({ details }: CpuExpansionProps) => (
  <div className="metric-details" data-sm-style="sm-style-323fdcc1e0">
    {(details.topProcesses?.length ?? 0) > 0 && (
      <div className="detail-section" data-sm-style="sm-style-91394348ef">
        <h4 data-sm-style="sm-style-c8a6493830">
          Top Processes by CPU:
        </h4>
        <div className="process-list">
          {(details.topProcesses ?? []).slice(0, 5).map((process: ProcessInfo) => (
            <div key={process.pid} data-sm-style="sm-style-d820a0b3ab">
              <span>{process.name} ({process.pid})</span>
              <span data-sm-style="sm-style-392c7463c7">
                {formatOptionalNumber(process.cpuPercent)}%
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" data-sm-style="sm-style-c08663b577">
      <div className="detail-item">
        <span className="detail-label" data-sm-style="sm-style-a6b497e153">
          Load Average:
        </span>
        <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
          {details.loadAverage?.slice(0, 3).map((load: number) => load.toFixed(2)).join(', ') ?? '—'}
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" data-sm-style="sm-style-a6b497e153">
          Context Switches:
        </span>
        <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
          {details.contextSwitches?.toLocaleString() ?? '—'}
        </span>
      </div>
    </div>

    <div className="detail-item">
      <span className="detail-label" data-sm-style="sm-style-a6b497e153">
        Total Goroutines:
      </span>
      <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
        {details.totalGoroutines ?? '—'}
      </span>
    </div>
  </div>
);
