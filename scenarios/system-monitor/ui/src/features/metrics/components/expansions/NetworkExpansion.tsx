import type { NetworkMetrics } from '../../../../types';

interface NetworkExpansionProps {
  details: NetworkMetrics;
}

export const NetworkExpansion = ({ details }: NetworkExpansionProps) => {
  const tcpStates = details?.tcpStates;
  const portUsage = details?.portUsage;
  const networkStats = details?.networkStats;

  return (
    <div className="metric-details" data-sm-style="sm-style-323fdcc1e0">
      {tcpStates && (
        <div className="detail-section" data-sm-style="sm-style-91394348ef">
          <h4 data-sm-style="sm-style-c8a6493830">
            Connection States:
          </h4>
          <div className="connection-states" data-sm-style="sm-style-952c530927">
            <div>Established: <span data-sm-style="sm-style-392c7463c7">
              {tcpStates.established ?? '—'}
            </span></div>
            <div>Time Wait: <span data-sm-style="sm-style-392c7463c7">
              {tcpStates.timeWait ?? '—'}
            </span></div>
            <div>Listen: <span data-sm-style="sm-style-392c7463c7">
              {tcpStates.listen ?? '—'}
            </span></div>
            <div>Total: <span data-sm-style="sm-style-392c7463c7">
              {tcpStates.total ?? '—'}
            </span></div>
          </div>
        </div>
      )}

      <div className="metric-grid-2col" data-sm-style="sm-style-c08663b577">
        {portUsage && (
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              Port Usage:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {portUsage.used} / {portUsage.total}
            </span>
          </div>
        )}
        {networkStats && (
          <div className="detail-item">
            <span className="detail-label" data-sm-style="sm-style-a6b497e153">
              DNS Health:
            </span>
            <span className="detail-value" data-sm-style="sm-style-dbed1e5364">
              {networkStats.dnsSuccessRate?.toFixed(1) ?? '—'}%
            </span>
          </div>
        )}
      </div>
    </div>
  );
};
