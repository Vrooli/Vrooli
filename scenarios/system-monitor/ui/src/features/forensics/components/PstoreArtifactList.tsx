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
    <div className="card" data-sm-style="sm-style-7b635e08e2">
      <div className="font-bold" data-sm-style="sm-style-b113dc3b73">
        Pstore Artifacts
      </div>
      <div className="text-xs text-muted" data-sm-style="sm-style-b113dc3b73">
        {path}
      </div>
      {entries.length === 0 ? (
        <div className="text-sm text-muted">
          No artifacts. (No panic captured since last boot — this is the healthy state.)
        </div>
      ) : (
        <table className="data-table" data-sm-style="sm-style-edba66b80f">
          <thead>
            <tr>
              <th data-sm-style="sm-style-90a773519e">Name</th>
              <th data-sm-style="sm-style-90a773519e">Kind</th>
              <th data-sm-style="sm-style-905bfede49">Size</th>
              <th data-sm-style="sm-style-90a773519e">Modified</th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.name}>
                <td>{e.name}</td>
                <td>{e.kind}</td>
                <td data-sm-style="sm-style-905bfede49">{e.size}</td>
                <td>{e.modified ? new Date(e.modified).toLocaleString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
};
