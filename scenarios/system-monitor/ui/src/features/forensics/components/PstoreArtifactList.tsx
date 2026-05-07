import type { ForensicsEnvelope, PstoreReport } from '../types';
import { NotProvisionedCard } from './NotProvisionedCard';

interface PstoreArtifactListProps {
  envelope: ForensicsEnvelope<PstoreReport>;
}

export const PstoreArtifactList = ({ envelope }: PstoreArtifactListProps) => {
  if (!envelope.available || !envelope.data) {
    return <NotProvisionedCard title="Pstore Artifacts" reason={envelope.reason} />;
  }
  const { path, entries } = envelope.data;
  return (
    <div className="card" style={{ padding: 'var(--spacing-md)' }}>
      <div className="font-bold" style={{ marginBottom: '0.5rem' }}>
        Pstore Artifacts
      </div>
      <div className="text-xs text-muted" style={{ marginBottom: '0.5rem' }}>
        {path}
      </div>
      {entries.length === 0 ? (
        <div className="text-sm text-muted">
          No artifacts. (No panic captured since last boot — this is the healthy state.)
        </div>
      ) : (
        <table className="data-table" style={{ width: '100%', fontSize: 'var(--text-sm)' }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left' }}>Name</th>
              <th style={{ textAlign: 'left' }}>Kind</th>
              <th style={{ textAlign: 'right' }}>Size</th>
              <th style={{ textAlign: 'left' }}>Modified</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.name}>
                <td>{e.name}</td>
                <td>{e.kind}</td>
                <td style={{ textAlign: 'right' }}>{e.size}</td>
                <td>{e.modified ? new Date(e.modified).toLocaleString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
};
