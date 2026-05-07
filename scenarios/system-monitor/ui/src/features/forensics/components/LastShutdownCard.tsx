import type { BootHistoryReport, ForensicsEnvelope } from '../types';
import { classifyShutdown, shutdownLabel } from '../utils/classifyShutdown';
import { NotProvisionedCard } from './NotProvisionedCard';

interface LastShutdownCardProps {
  envelope: ForensicsEnvelope<BootHistoryReport>;
}

/**
 * Surfaces the most recent prior boot's shutdown classification.
 * The "last shutdown" is the boot with index === -1 (one before current).
 */
export const LastShutdownCard = ({ envelope }: LastShutdownCardProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Last Shutdown" reason={envelope.reason} />;
  }
  const prev = envelope.data.boots.find((b) => b.index === -1);
  if (!prev) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-md)' }}>
        <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
          Last Shutdown
        </div>
        <div className="text-sm text-muted">No prior boot recorded yet.</div>
      </div>
    );
  }
  const cls = classifyShutdown(prev);
  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
        Last Shutdown
      </div>
      <div className="text-sm">
        <strong>Status:</strong> {shutdownLabel(cls)}
      </div>
      <div className="text-xs text-muted" style={{ fontFamily: 'monospace' }}>
        boot {prev.bootId.slice(0, 8)} ({prev.index})
      </div>
      {prev.reason && <div className="text-xs text-muted">Reason: {prev.reason}</div>}
    </div>
  );
};
