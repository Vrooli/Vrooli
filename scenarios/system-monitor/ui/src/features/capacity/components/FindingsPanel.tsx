import { formatBytes } from '../../../shared/utils/formatters';
import type { CapacityFinding } from '../types';

interface FindingsPanelProps {
  findings: CapacityFinding[];
  available: boolean;
}

const CLASS_LABEL: Record<string, string> = {
  unclaimed: 'Unclaimed',
  over_claim: 'Over-claim',
  claimed: 'Claimed',
};

/**
 * The reconciliation panel: observed GPU consumers classified against the
 * ledger. Unclaimed/over-claim findings are warnings (the adoption-tracker
 * signal); claimed findings are informational. Surfaces sensing unavailability
 * honestly rather than implying "no consumers".
 */
export const FindingsPanel = ({ findings, available }: FindingsPanelProps) => {
  if (!available) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-md)', color: 'var(--color-text-muted, #999)' }}>
        Capacity sensing is unavailable on this host (no GPU probe), so unclaimed
        consumers cannot be reconciled.
      </div>
    );
  }

  if (findings.length === 0) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-md)' }}>
        No GPU consumers above the tracking threshold.
      </div>
    );
  }

  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {findings.map((finding) => {
          const isWarn = finding.severity === 'warn';
          return (
            <li
              key={`${finding.ownerId}-${String(finding.pid)}-${finding.class}`}
              style={{
                display: 'flex',
                gap: '8px',
                alignItems: 'baseline',
                borderLeft: `3px solid ${isWarn ? 'var(--color-warning, #e0a020)' : 'var(--color-border, #444)'}`,
                paddingLeft: '8px',
              }}
            >
              <span
                style={{
                  fontSize: '0.7rem',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  color: isWarn ? 'var(--color-warning, #e0a020)' : 'var(--color-text-muted, #999)',
                  minWidth: '88px',
                }}
              >
                {CLASS_LABEL[finding.class] ?? finding.class}
              </span>
              <span style={{ fontSize: '0.85rem' }}>
                {finding.message || `${finding.ownerId} — ${formatBytes(Number(finding.observedBytes))}`}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
