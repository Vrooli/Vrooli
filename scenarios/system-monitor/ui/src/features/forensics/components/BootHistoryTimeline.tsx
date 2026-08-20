import type { BootHistoryReport, ForensicsEnvelope } from '../types';
import { classifyShutdown, shutdownLabel, type ShutdownClass } from '../utils/classifyShutdown';
import { formatBootDuration } from '../utils/formatBootDuration';
import { NotProvisionedCard } from './NotProvisionedCard';

interface BootHistoryTimelineProps {
  envelope: ForensicsEnvelope<BootHistoryReport>;
}

const shutdownClass = (c: ShutdownClass): string => {
  switch (c) {
    case 'clean':
      return 'shutdown-clean';
    case 'unclean':
      return 'shutdown-unclean';
    case 'in-progress':
      return 'shutdown-in-progress';
    case 'unknown':
      return 'shutdown-unknown';
  }
};

export const BootHistoryTimeline = ({ envelope }: BootHistoryTimelineProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Boot History" reason={envelope.reason} />;
  }
  const { boots } = envelope.data;
  return (
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <div className="font-bold" data-sm-style="sm-style-b113dc3b73">
        Boot History
      </div>
      {boots.length === 0 ? (
        <div className="text-sm text-muted">No boot records available.</div>
      ) : (
        <ul data-sm-style="sm-style-0d21d4c312">
          {boots.map((b) => {
            const c = classifyShutdown(b);
            return (
              <li
                key={b.bootId || `idx-${b.index}`}
                className={`boot-history-row ${shutdownClass(c)}`}
                style={{
                  display: 'grid',
                  gridTemplateColumns: '4ch 1fr auto auto',
                  gap: '0.5rem',
                  alignItems: 'center',
                  padding: '0.25rem 0',
                  paddingLeft: '0.5rem',
                  marginBottom: '0.25rem',
                }}
              >
                <span className="text-xs text-muted">{b.index}</span>
                <span className="text-xs" title={b.bootId} data-sm-style="sm-style-51316ccfb7">
                  {b.bootId.slice(0, 8) || '—'}
                </span>
                <span className="text-xs">{formatBootDuration(b.firstEntry, b.lastEntry)}</span>
                <span className="text-xs shutdown-label">
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
