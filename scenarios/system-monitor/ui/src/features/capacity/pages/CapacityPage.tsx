import { useCapacity } from '../hooks/useCapacity';
import { GpuContentionCard } from '../components/GpuContentionCard';
import { ClaimsTable } from '../components/ClaimsTable';
import { FindingsPanel } from '../components/FindingsPanel';
import { PolicyPanel } from '../components/PolicyPanel';
import { useTimeRange } from '../../../shared/time/TimeRangeContext';

const sectionHeading = (text: string) => (
  <h3 data-sm-style="sm-style-91a0904255">{text}</h3>
);

/**
 * The Capacity governance dashboard: live per-GPU contention, the active claim
 * ledger, unclaimed-consumer warnings, and the tunable policy levers. Reads the
 * platform `internal/capacity` ledger via system-monitor's capacity API.
 */
export const CapacityPage = () => {
  const { range } = useTimeRange();
  const {
    overview,
    reconciliation,
    policy,
    isLoading,
    error,
    policyError,
    isSavingPolicy,
    refresh,
    savePolicy,
  } = useCapacity();

  const sensingAvailable = overview?.sensingAvailable ?? false;

  return (
    <section className="capacity-page">
      <header className="flex-row-center" data-sm-style="sm-style-740d1580c7">
        <h2 data-sm-style="sm-style-2a0ca8350a">Capacity</h2>
        <button type="button" className="header-button" onClick={() => void refresh()} disabled={isLoading}>
          {isLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      <p className="text-xs text-muted" data-sm-style="sm-style-00a48ba4a2">
        Shared time range: {range.label}. Capacity is a current-state ledger; historical utilization is shown only where metric history is available.
      </p>

      {error && (
        <div className="card" role="alert" data-sm-style="sm-style-0fdae7acd4">
          {error}
        </div>
      )}

      {overview && overview.warnings.length > 0 && (
        <div className="card" data-sm-style="sm-style-71cbba1723">
          {overview.warnings.map((warning) => (
            <div key={warning}>⚠ {warning}</div>
          ))}
        </div>
      )}

      {!overview && isLoading && (
        <div className="card" data-sm-style="sm-style-7b635e08e2">Loading capacity state…</div>
      )}

      {overview && (
        <>
          {sectionHeading('GPU contention')}
          {overview.gpus.length === 0 ? (
            <div className="card" data-sm-style="sm-style-2dce25a9dc">
              No GPUs detected on this host.
            </div>
          ) : (
            <div data-sm-style="sm-style-20558dad9f">
              {overview.gpus.map((gpu) => (
                <GpuContentionCard key={gpu.index} gpu={gpu} />
              ))}
            </div>
          )}

          {sectionHeading('Active claims')}
          <ClaimsTable claims={overview.claims} />

          {sectionHeading('Unclaimed consumers')}
          <FindingsPanel findings={reconciliation?.findings ?? []} available={sensingAvailable} />
        </>
      )}

      {policy && (
        <>
          {sectionHeading('Policy levers')}
          <PolicyPanel levers={policy} isSaving={isSavingPolicy} error={policyError} onSave={(key, value) => { void savePolicy(key, value); }} />
        </>
      )}
    </section>
  );
};
