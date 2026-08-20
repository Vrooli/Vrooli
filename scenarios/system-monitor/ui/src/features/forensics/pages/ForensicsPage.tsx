import { useForensicsSummary } from '../hooks/useForensicsSummary';
import { AutohealChecksPanel } from '../components/AutohealChecksPanel';
import { BootHistoryTimeline } from '../components/BootHistoryTimeline';
import { LastShutdownCard } from '../components/LastShutdownCard';
import { MCESummaryCard } from '../components/MCESummaryCard';
import { PstoreArtifactList } from '../components/PstoreArtifactList';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';

export const ForensicsPage = () => {
  const { range } = useTimeRange();
  const { summary, isLoading, error, refresh } = useForensicsSummary();

  return (
    <section className="forensics-page">
      <header
        className="flex-row-center"
        data-sm-style="sm-style-740d1580c7"
      >
        <h2 data-sm-style="sm-style-2a0ca8350a">Crash Forensics</h2>
        <button
          type="button"
          className="header-button"
          onClick={() => void refresh()}
          disabled={isLoading}
        >
          {isLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      <p className="text-xs text-muted" data-sm-style="sm-style-00a48ba4a2">
        Shared time range: {range.label}. Boot, MCE, and pstore evidence is reported as host-state evidence; records outside this window are labeled by their source.
      </p>

      {error && (
        <div
          className="card"
          data-sm-style="sm-style-6a6075a1e3"
          role="alert"
        >
          {error}
        </div>
      )}

      {!summary && isLoading && (
        <div className="card" data-sm-style="sm-style-7b635e08e2">
          Loading forensics signals…
        </div>
      )}

      {summary && (
        <div
          className="forensics-grid"
          data-sm-style="sm-style-0e426d7736"
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
