import type { NetworkMetrics } from '../../../../types';

interface NetworkExpansionProps {
  details: NetworkMetrics;
}

export const NetworkExpansion = ({ details }: NetworkExpansionProps) => {
  const tcpStates = details?.tcpStates;
  const portUsage = details?.portUsage;
  const networkStats = details?.networkStats;

  return (
    <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
      {tcpStates && (
        <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
          <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-heading)' }}>
            Connection States:
          </h4>
          <div className="connection-states" style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, 1fr)',
            gap: 'var(--spacing-sm)',
            fontSize: 'var(--text-sm)'
          }}>
            <div>Established: <span style={{ color: 'var(--color-primary)' }}>
              {tcpStates.established ?? '—'}
            </span></div>
            <div>Time Wait: <span style={{ color: 'var(--color-primary)' }}>
              {tcpStates.timeWait ?? '—'}
            </span></div>
            <div>Listen: <span style={{ color: 'var(--color-primary)' }}>
              {tcpStates.listen ?? '—'}
            </span></div>
            <div>Total: <span style={{ color: 'var(--color-primary)' }}>
              {tcpStates.total ?? '—'}
            </span></div>
          </div>
        </div>
      )}

      <div className="metric-grid-2col" style={{ marginBottom: 'var(--spacing-sm)' }}>
        {portUsage && (
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>
              Port Usage:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
              {portUsage.used} / {portUsage.total}
            </span>
          </div>
        )}
        {networkStats && (
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-secondary)' }}>
              DNS Health:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-heading)' }}>
              {networkStats.dnsSuccessRate?.toFixed(1) ?? '—'}%
            </span>
          </div>
        )}
      </div>
    </div>
  );
};
