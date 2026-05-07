import type { ForensicsEnvelope, MCEReport } from '../types';
import { NotProvisionedCard } from './NotProvisionedCard';

interface MCESummaryCardProps {
  envelope: ForensicsEnvelope<MCEReport>;
}

export const MCESummaryCard = ({ envelope }: MCESummaryCardProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Machine Check Errors" reason={envelope.reason} />;
  }
  const { window, uncorrected, corrected, rawSummary } = envelope.data;
  const errorColor = uncorrected > 0
    ? 'var(--color-error, #f87171)'
    : corrected > 0
      ? 'var(--color-warning, #facc15)'
      : 'var(--color-success, #4ade80)';
  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
        Machine Check Errors
      </div>
      <div className="text-xs text-muted" style={{ marginBottom: '0.5rem' }}>
        Window: {window}
      </div>
      <div className="flex-row-center gap-md">
        <div>
          <div className="text-xs text-muted">Uncorrected</div>
          <div style={{ fontSize: 'var(--text-xl)', color: errorColor, fontWeight: 'bold' }}>
            {uncorrected}
          </div>
        </div>
        <div>
          <div className="text-xs text-muted">Corrected</div>
          <div style={{ fontSize: 'var(--text-xl)', fontWeight: 'bold' }}>{corrected}</div>
        </div>
      </div>
      {rawSummary && (
        <details style={{ marginTop: '0.5rem' }}>
          <summary className="text-xs text-muted" style={{ cursor: 'pointer' }}>
            Raw summary
          </summary>
          <pre
            className="text-xs"
            style={{ whiteSpace: 'pre-wrap', marginTop: '0.25rem' }}
          >
            {rawSummary}
          </pre>
        </details>
      )}
    </div>
  );
};
