import type { NetworkMetrics } from '../../../../types';

interface NetworkExpansionProps {
  details: NetworkMetrics;
}

export const NetworkExpansion = ({ details }: NetworkExpansionProps) => {
  const tcpStates = details?.tcp_states;
  const portUsage = details?.port_usage;
  const networkStats = details?.network_stats;

  return (
    <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
      {tcpStates && (
        <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
          <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
            Connection States:
          </h4>
          <div className="connection-states" style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, 1fr)',
            gap: 'var(--spacing-sm)',
            fontSize: 'var(--font-size-sm)'
          }}>
            <div>Established: <span style={{ color: 'var(--color-accent)' }}>
              {tcpStates.established ?? '\u2014'}
            </span></div>
            <div>Time Wait: <span style={{ color: 'var(--color-accent)' }}>
              {tcpStates.time_wait ?? '\u2014'}
            </span></div>
            <div>Listen: <span style={{ color: 'var(--color-accent)' }}>
              {tcpStates.listen ?? '\u2014'}
            </span></div>
            <div>Total: <span style={{ color: 'var(--color-accent)' }}>
              {tcpStates.total ?? '\u2014'}
            </span></div>
          </div>
        </div>
      )}

      <div className="metric-grid-2col" style={{ marginBottom: 'var(--spacing-sm)' }}>
        {portUsage && (
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              Port Usage:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {portUsage.used} / {portUsage.total}
            </span>
          </div>
        )}
        {networkStats && (
          <div className="detail-item">
            <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
              DNS Health:
            </span>
            <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
              {networkStats.dns_success_rate?.toFixed(1) ?? '\u2014'}%
            </span>
          </div>
        )}
      </div>
    </div>
  );
};
