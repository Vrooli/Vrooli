import type { CPUMetrics, ProcessInfo } from '../../../../types';
import { formatOptionalNumber } from '../../../../shared/utils/formatters';

interface CpuExpansionProps {
  details: CPUMetrics;
}

export const CpuExpansion = ({ details }: CpuExpansionProps) => (
  <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
    {(details.topProcesses?.length ?? 0) > 0 && (
      <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
          Top Processes by CPU:
        </h4>
        <div className="process-list">
          {(details.topProcesses ?? []).slice(0, 5).map((process: ProcessInfo) => (
            <div key={process.pid} style={{
              display: 'flex',
              justifyContent: 'space-between',
              margin: 'var(--spacing-xs) 0',
              fontSize: 'var(--font-size-sm)'
            }}>
              <span>{process.name} ({process.pid})</span>
              <span style={{ color: 'var(--color-accent)' }}>
                {formatOptionalNumber(process.cpuPercent)}%
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" style={{ marginBottom: 'var(--spacing-sm)' }}>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
          Load Average:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
          {details.loadAverage?.slice(0, 3).map((load: number) => load.toFixed(2)).join(', ') ?? '—'}
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
          Context Switches:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
          {details.contextSwitches?.toLocaleString() ?? '—'}
        </span>
      </div>
    </div>

    <div className="detail-item">
      <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
        Total Goroutines:
      </span>
      <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
        {details.totalGoroutines ?? '—'}
      </span>
    </div>
  </div>
);
