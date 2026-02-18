import type { MemoryMetrics, ProcessInfo, MemoryGrowth } from '../../../../types';
import { formatOptionalNumber } from '../../../../shared/utils/formatters';
import { getRiskLevelColor } from '../../../../shared/utils/colors';

interface MemoryExpansionProps {
  details: MemoryMetrics;
}

export const MemoryExpansion = ({ details }: MemoryExpansionProps) => (
  <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
    {(details.topProcesses?.length ?? 0) > 0 && (
      <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-heading)' }}>
          Top Processes by Memory:
        </h4>
        <div className="process-list">
          {(details.topProcesses ?? []).slice(0, 5).map((process: ProcessInfo) => (
            <div key={process.pid} style={{
              display: 'flex',
              justifyContent: 'space-between',
              margin: 'var(--spacing-xs) 0',
              fontSize: 'var(--text-sm)'
            }}>
              <span>{process.name} ({process.pid})</span>
              <span style={{ color: 'var(--color-primary)' }}>
                {formatOptionalNumber(process.memoryMb)} MB
              </span>
            </div>
          ))}
        </div>
      </div>
    )}

    <div className="metric-grid-2col" style={{ marginBottom: 'var(--spacing-sm)' }}>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>
          Swap Usage:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
          {formatOptionalNumber(details.swapUsage?.percent)}%
        </span>
      </div>
      <div className="detail-item">
        <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>
          Disk Usage:
        </span>
        <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
          {formatOptionalNumber(details.diskUsage?.percent)}%
        </span>
      </div>
    </div>

    {(details.growthPatterns?.length ?? 0) > 0 && (
      <div className="detail-section">
        <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-heading)' }}>
          Memory Growth Patterns:
        </h4>
        <div className="growth-patterns">
          {(details.growthPatterns ?? []).slice(0, 3).map((pattern: MemoryGrowth, index: number) => (
            <div key={index} style={{
              margin: 'var(--spacing-xs) 0',
              fontSize: 'var(--text-sm)'
            }}>
              <span>{pattern.process}: </span>
              <span style={{ color: getRiskLevelColor(pattern.riskLevel) }}>
                {formatOptionalNumber(pattern.growthMbPerHour)} MB/hr ({pattern.riskLevel})
              </span>
            </div>
          ))}
        </div>
      </div>
    )}
  </div>
);
