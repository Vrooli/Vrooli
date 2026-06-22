import { useCapacity } from '../hooks/useCapacity';
import { GpuContentionCard } from '../components/GpuContentionCard';
import { ClaimsTable } from '../components/ClaimsTable';
import { FindingsPanel } from '../components/FindingsPanel';
import { PolicyPanel } from '../components/PolicyPanel';

const sectionHeading = (text: string) => (
  <h3 style={{ margin: 'var(--spacing-lg) 0 var(--spacing-sm)' }}>{text}</h3>
);

/**
 * The Capacity governance dashboard: live per-GPU contention, the active claim
 * ledger, unclaimed-consumer warnings, and the tunable policy levers. Reads the
 * platform `internal/capacity` ledger via system-monitor's capacity API.
 */
export const CapacityPage = () => {
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
      <header className="flex-row-center" style={{ justifyContent: 'space-between', marginBottom: 'var(--spacing-md)' }}>
        <h2 style={{ margin: 0 }}>Capacity</h2>
        <button type="button" className="header-button" onClick={() => void refresh()} disabled={isLoading}>
          {isLoading ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      {error && (
        <div className="card" role="alert" style={{ padding: 'var(--spacing-md)', color: 'var(--color-error)', marginBottom: 'var(--spacing-md)' }}>
          {error}
        </div>
      )}

      {overview && overview.warnings.length > 0 && (
        <div className="card" style={{ padding: 'var(--spacing-md)', marginBottom: 'var(--spacing-md)', color: 'var(--color-warning, #e0a020)' }}>
          {overview.warnings.map((warning) => (
            <div key={warning}>⚠ {warning}</div>
          ))}
        </div>
      )}

      {!overview && isLoading && (
        <div className="card" style={{ padding: 'var(--spacing-md)' }}>Loading capacity state…</div>
      )}

      {overview && (
        <>
          {sectionHeading('GPU contention')}
          {overview.gpus.length === 0 ? (
            <div className="card" style={{ padding: 'var(--spacing-md)', color: 'var(--color-text-muted, #999)' }}>
              No GPUs detected on this host.
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(340px, 1fr))', gap: 'var(--spacing-md)' }}>
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
