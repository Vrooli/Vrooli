import { useForensicsSummary } from '../hooks/useForensicsSummary';
import { AutohealChecksPanel } from '../components/AutohealChecksPanel';
import { BootHistoryTimeline } from '../components/BootHistoryTimeline';
import { LastShutdownCard } from '../components/LastShutdownCard';
import { MCESummaryCard } from '../components/MCESummaryCard';
import { PstoreArtifactList } from '../components/PstoreArtifactList';

export const ForensicsPage = () => {
  const { summary, isLoading, error, refresh } = useForensicsSummary();

  return (
    <section className="forensics-page">
      <header
        className="flex-row-center"
        style={{ justifyContent: 'space-between', marginBottom: 'var(--spacing-md)' }}
      >
        <h2 style={{ margin: 0 }}>Crash Forensics</h2>
        <button
          type="button"
          className="header-button"
          onClick={() => void refresh()}
          disabled={isLoading}
        >
          {isLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      {error && (
        <div
          className="card"
          style={{
            padding: 'var(--spacing-md)',
            color: 'var(--color-error)',
            marginBottom: 'var(--spacing-md)',
          }}
          role="alert"
        >
          {error}
        </div>
      )}

      {!summary && isLoading && (
        <div className="card" style={{ padding: 'var(--spacing-md)' }}>
          Loading forensics signals…
        </div>
      )}

      {summary && (
        <div
          className="forensics-grid"
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
            gap: 'var(--spacing-md)',
          }}
        >
          <LastShutdownCard envelope={summary.bootHistory} />
          <MCESummaryCard envelope={summary.mce} />
          <PstoreArtifactList envelope={summary.pstore} />
          <BootHistoryTimeline envelope={summary.bootHistory} />
          <AutohealChecksPanel envelope={summary.autoheal} />
        </div>
      )}
    </section>
  );
};
