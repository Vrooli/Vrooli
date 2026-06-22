import { formatBytes } from '../../../shared/utils/formatters';
import type { CapacityClaim } from '../types';

interface ClaimsTableProps {
  claims: CapacityClaim[];
}

const PRIORITY_LABEL: Record<string, string> = {
  interactive: 'Interactive',
  service: 'Service',
  batch: 'Batch',
};

/** The active claim ledger as a table. */
export const ClaimsTable = ({ claims }: ClaimsTableProps) => {
  if (claims.length === 0) {
    return (
      <div className="card" style={{ padding: 'var(--spacing-md)' }}>
        No active capacity claims.
      </div>
    );
  }

  return (
    <div className="card" style={{ padding: 'var(--spacing-md)', overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
        <thead>
          <tr style={{ textAlign: 'left', borderBottom: '1px solid var(--color-border, #444)' }}>
            <th style={{ padding: '6px 8px' }}>Owner</th>
            <th style={{ padding: '6px 8px' }}>Resource</th>
            <th style={{ padding: '6px 8px' }}>Status</th>
            <th style={{ padding: '6px 8px' }}>Priority</th>
            <th style={{ padding: '6px 8px' }}>Activity</th>
            <th style={{ padding: '6px 8px', textAlign: 'right' }}>Amount</th>
          </tr>
        </thead>
        <tbody>
          {claims.map((claim) => (
            <tr key={claim.claimId} style={{ borderBottom: '1px solid var(--color-border-subtle, #333)' }}>
              <td style={{ padding: '6px 8px' }}>
                {claim.protected && <span title="protected while active" aria-label="protected">🛡 </span>}
                {claim.ownerKind}/{claim.ownerId}
              </td>
              <td style={{ padding: '6px 8px' }}>
                {claim.resourceKind}
                {claim.gpuIndex !== undefined ? ` (gpu ${String(claim.gpuIndex)})` : ''}
              </td>
              <td style={{ padding: '6px 8px' }}>{claim.status}</td>
              <td style={{ padding: '6px 8px' }}>{PRIORITY_LABEL[claim.priorityTier] ?? claim.priorityTier}</td>
              <td style={{ padding: '6px 8px' }}>
                <span style={{ color: claim.activityState === 'active' ? 'var(--color-success, #4caf50)' : 'var(--color-text-muted, #999)' }}>
                  {claim.activityState || '—'}
                </span>
              </td>
              <td style={{ padding: '6px 8px', textAlign: 'right' }}>{formatBytes(Number(claim.amountBytes))}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
