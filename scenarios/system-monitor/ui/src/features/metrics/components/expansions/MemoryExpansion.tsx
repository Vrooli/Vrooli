import type { MemoryMetrics } from '../../../../types';

interface MemoryExpansionProps {
  details: MemoryMetrics;
}

export const MemoryExpansion = ({ details }: MemoryExpansionProps) => (
  <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
    {(details.top_processes?.length ?? 0) > 0 && (
      <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
          Top Processes by Memory:
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
                {process.memory_mb?.toFixed(1) ?? '\u2014'} MB
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" style={{ marginBottom: 'var(--spacing-sm)' }}>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
          Swap Usage:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
          {details.swap_usage?.percent?.toFixed(1) ?? '\u2014'}%
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
          Disk Usage:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
          {details.disk_usage?.percent?.toFixed(1) ?? '\u2014'}%
        </span>
      </div>
    </div>

    {(details.growth_patterns?.length ?? 0) > 0 && (
      <div className="detail-section">
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
          Memory Growth Patterns:
        </h4>
        <div className="growth-patterns">
          {(details.growth_patterns ?? []).slice(0, 3).map((pattern, index) => (
            <div key={index} style={{
              margin: 'var(--spacing-xs) 0',
              fontSize: 'var(--font-size-sm)'
            }}>
              <span>{pattern.process}: </span>
              <span style={{
                color: pattern.risk_level === 'high' ? 'var(--color-error)' :
                       pattern.risk_level === 'medium' ? 'var(--color-warning)' :
                       'var(--color-accent)'
              }}>
                {pattern.growth_mb_per_hour?.toFixed(1) ?? '\u2014'} MB/hr ({pattern.risk_level})
              </span>
            </div>
          ))}
        </div>
      </div>
    )}
  </div>
);
