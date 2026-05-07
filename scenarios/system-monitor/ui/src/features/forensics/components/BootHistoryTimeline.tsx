import type { BootHistoryReport, ForensicsEnvelope } from '../types';
import { classifyShutdown, shutdownLabel, type ShutdownClass } from '../utils/classifyShutdown';
import { formatBootDuration } from '../utils/formatBootDuration';
import { NotProvisionedCard } from './NotProvisionedCard';

interface BootHistoryTimelineProps {
  envelope: ForensicsEnvelope<BootHistoryReport>;
}

const colorFor = (c: ShutdownClass): string => {
  switch (c) {
    case 'clean':
      return 'var(--color-success, #4ade80)';
    case 'unclean':
      return 'var(--color-error, #f87171)';
    case 'in-progress':
      return 'var(--color-info, #60a5fa)';
    case 'unknown':
      return 'var(--color-warning, #facc15)';
  }
};

export const BootHistoryTimeline = ({ envelope }: BootHistoryTimelineProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Boot History" reason={envelope.reason} />;
  }
  const { boots } = envelope.data;
  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
        Boot History
      </div>
      {boots.length === 0 ? (
        <div className="text-sm text-muted">No boot records available.</div>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {boots.map((b) => {
            const c = classifyShutdown(b);
            return (
              <li
                key={b.bootId || `idx-${b.index}`}
                className="boot-history-row"
                style={{
                  display: 'grid',
                  gridTemplateColumns: '4ch 1fr auto auto',
                  gap: '0.5rem',
                  alignItems: 'center',
                  padding: '0.25rem 0',
                  borderLeft: `3px solid ${colorFor(c)}`,
                  paddingLeft: '0.5rem',
                  marginBottom: '0.25rem',
                }}
              >
                <span className="text-xs text-muted">{b.index}</span>
                <span className="text-xs" title={b.bootId} style={{ fontFamily: 'monospace' }}>
                  {b.bootId.slice(0, 8) || '—'}
                </span>
                <span className="text-xs">{formatBootDuration(b.firstEntry, b.lastEntry)}</span>
                <span className="text-xs" style={{ color: colorFor(c) }}>
                  {shutdownLabel(c)}
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};
