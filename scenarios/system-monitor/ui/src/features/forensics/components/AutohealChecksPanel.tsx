import type { AutohealEnvelope } from '../types';
import { NotProvisionedCard } from './NotProvisionedCard';

interface AutohealChecksPanelProps {
  envelope: AutohealEnvelope;
}

const statusColor = (status: string): string => {
  const s = status.toLowerCase();
  if (s === 'ok' || s === 'pass' || s === 'passed') return 'var(--color-success, #4ade80)';
  if (s === 'critical' || s === 'fail' || s === 'failed') return 'var(--color-error, #f87171)';
  if (s === 'warning' || s === 'warn') return 'var(--color-warning, #facc15)';
  return 'var(--color-muted, #94a3b8)';
};

export const AutohealChecksPanel = ({ envelope }: AutohealChecksPanelProps) => {
  if (!envelope.available) {
    return <NotProvisionedCard title="Autoheal Checks" reason={envelope.reason || 'autoheal offline'} />;
  }
  const checks = envelope.checks ?? [];
  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
        Autoheal Checks (forensics)
      </div>
      {checks.length === 0 ? (
        <div className="text-sm text-muted">No forensics-relevant checks reported.</div>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {checks.map((c) => (
            <li
              key={c.checkId}
              style={{
                display: 'grid',
                gridTemplateColumns: '1fr auto',
                alignItems: 'center',
                padding: '0.25rem 0',
                gap: '0.5rem',
              }}
            >
              <div>
                <div className="text-sm" style={{ fontFamily: 'monospace' }}>
                  {c.checkId}
                </div>
                {c.message && <div className="text-xs text-muted">{c.message}</div>}
              </div>
              <span
                className="text-xs"
                style={{
                  color: statusColor(c.status),
                  fontWeight: 'bold',
                  textTransform: 'uppercase',
                }}
              >
                {c.status}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};
