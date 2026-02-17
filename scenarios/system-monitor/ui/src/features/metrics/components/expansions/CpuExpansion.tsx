import type { CPUMetrics } from '../../../../types';

interface CpuExpansionProps {
  details: CPUMetrics;
}

export const CpuExpansion = ({ details }: CpuExpansionProps) => (
  <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
    {(details.top_processes?.length ?? 0) > 0 && (
      <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
          Top Processes by CPU:
        </h4>
        <div className="process-list">
          {(details.top_processes ?? []).slice(0, 5).map((process) => (
            <div key={process.pid} style={{
              display: 'flex',
              justifyContent: 'space-between',
              margin: 'var(--spacing-xs) 0',
              fontSize: 'var(--font-size-sm)'
            }}>
              <span>{process.name} ({process.pid})</span>
              <span style={{ color: 'var(--color-accent)' }}>
                {process.cpu_percent?.toFixed(1) ?? '\u2014'}%
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
          {details.load_average?.slice(0, 3).map(load => load.toFixed(2)).join(', ') ?? '\u2014'}
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
          Context Switches:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
          {details.context_switches?.toLocaleString() ?? '\u2014'}
        </span>
      </div>
    </div>

    <div className="detail-item">
      <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
        Total Goroutines:
      </span>
      <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
        {details.total_goroutines ?? '\u2014'}
      </span>
    </div>
  </div>
);
