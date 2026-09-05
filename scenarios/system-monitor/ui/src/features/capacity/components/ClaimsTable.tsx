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
      <div className="card" data-sm-style="sm-style-7b635e08e2">
        No active capacity claims.
      </div>
    );
  }

  return (
    <div className="card" data-sm-style="sm-style-e685363b82">
      <table data-sm-style="sm-style-e8222ffdff">
        <thead>
          <tr data-sm-style="sm-style-b5d228e679">
            <th data-sm-style="sm-style-d9eef182f8">Owner</th>
            <th data-sm-style="sm-style-d9eef182f8">Resource</th>
            <th data-sm-style="sm-style-d9eef182f8">Status</th>
            <th data-sm-style="sm-style-d9eef182f8">Priority</th>
            <th data-sm-style="sm-style-d9eef182f8">Activity</th>
            <th data-sm-style="sm-style-cb4435607b">Amount</th>
          </tr>
        </thead>
        <tbody>
          {claims.map((claim) => (
            <tr key={claim.claimId} data-sm-style="sm-style-b67039a0c6">
              <td data-sm-style="sm-style-d9eef182f8">
                {claim.protected && <span title="protected while active" aria-label="protected">🛡 </span>}
                {claim.ownerKind}/{claim.ownerId}
              </td>
              <td data-sm-style="sm-style-d9eef182f8">
                {claim.resourceKind}
                {claim.gpuIndex !== undefined ? ` (gpu ${String(claim.gpuIndex)})` : ''}
              </td>
              <td data-sm-style="sm-style-d9eef182f8">{claim.status}</td>
              <td data-sm-style="sm-style-d9eef182f8">{PRIORITY_LABEL[claim.priorityTier] ?? claim.priorityTier}</td>
              <td data-sm-style="sm-style-d9eef182f8">
                <span className={claim.activityState === 'active' ? 'text-success' : 'text-muted'}>
                  {claim.activityState || '—'}
                </span>
              </td>
              <td data-sm-style="sm-style-cb4435607b">{formatBytes(Number(claim.amountBytes))}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
