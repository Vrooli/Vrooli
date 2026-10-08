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
        <div className="card card--excursion" role="alert">
          <p className="eyebrow card--excursion__label">Capacity request failed</p>
          <p className="capacity-blind__body">{error}</p>
        </div>
      )}

      {/*
        * Rendered only when sensing DID work. When it did not, the same
        * warnings are the reason text inside the blind panel below, and
        * showing them twice would read as two separate problems.
        */}
      {overview && sensingAvailable && overview.warnings.length > 0 && (
        <div className="card card--caution" role="status">
          <p className="eyebrow card--caution__label">Sensing warnings</p>
          <ul className="capacity-blind__reasons">
            {overview.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      )}

      {!overview && isLoading && (
        <div className="card" data-sm-style="sm-style-7b635e08e2">Loading capacity state…</div>
      )}

      {overview && (
        <>
          {sectionHeading('GPU contention')}
          {overview.gpus.length === 0 ? (
            /*
             * An empty GPU list has two completely different causes and they
             * must never share a message. If sensing was available and returned
             * nothing, the host genuinely has no GPU. If sensing was NOT
             * available, we did not look — and "No GPUs detected on this host"
             * would be an assertion about the machine that this page has no
             * evidence for. That is the exact failure this scenario exists to
             * remove, so the unavailable branch reports the blindness and the
             * reason for it, and never a count.
             */
            sensingAvailable ? (
              <div className="card capacity-empty" data-sm-style="sm-style-2dce25a9dc">
                No GPUs detected on this host.
              </div>
            ) : (
              <div className="card capacity-blind" role="status">
                <p className="eyebrow capacity-blind__label">GPU contention unreadable</p>
                <p className="capacity-blind__body">
                  No GPU probe answered on this host, so this page cannot say whether a
                  GPU is present. This is a gap in what can be measured here — not a
                  report that the machine has none.
                </p>
                {overview.warnings.length > 0 && (
                  <ul className="capacity-blind__reasons">
                    {overview.warnings.map((warning) => (
                      <li key={warning}>{warning}</li>
                    ))}
                  </ul>
                )}
              </div>
            )
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
