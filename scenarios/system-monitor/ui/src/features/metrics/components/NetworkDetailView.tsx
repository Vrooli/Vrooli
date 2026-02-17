import { useMemo } from 'react';
import { Network } from 'lucide-react';

import { DetailRow } from '../../../shared/components/DetailRow';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatTimeLabel } from '../../../shared/utils/formatters';
import { buildSingleSeriesData } from './metricHelpers';

export interface NetworkDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const NetworkDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: NetworkDetailViewProps) => {
  const networkData = useMemo(() => buildSingleSeriesData(metricHistory?.network), [metricHistory?.network]);
  const networkDetails = detailedMetrics?.network_details;
  const totalConnections = metrics?.tcp_connections ?? networkDetails?.tcp_states?.total ?? 0;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatTimeLabel(detailedMetrics.timestamp)}`
    : undefined;

  return (
    <MetricDetailLayout
      title="NETWORK ACTIVITY"
      icon={<Network size={22} />}
      headline={`${totalConnections.toLocaleString()} active connections`}
      subhead={subhead}
      onBack={onBack}
    >
      <MetricLineChart
        className="card"
        data={networkData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'TCP Connections', color: 'var(--color-accent)' }]}
        unit=""
        yDomain={['auto', 'auto']}
        valueFormatter={value => `${Math.round(value).toLocaleString()} connections`}
      />

      <div className="metric-grid-auto-lg">
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">TCP States</h3>
          {networkDetails ? (
            <div className="detail-grid detail-grid-sm">
              {Object.entries(networkDetails.tcp_states).filter(([key]) => key !== 'total').map(([state, value]) => (
                <DetailRow key={state} label={state.toUpperCase()} value={Number(value).toLocaleString()} />
              ))}
            </div>
          ) : (
            <div className="text-muted">
              Connection state metrics unavailable.
            </div>
          )}
        </div>

        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">Network Health</h3>
          {networkDetails ? (
            <div className="detail-grid detail-grid-md">
              <DetailRow label="Ingress Bandwidth" value={`${networkDetails.network_stats?.bandwidth_in_mbps?.toFixed(2) ?? '\u2014'} Mbps`} />
              <DetailRow label="Egress Bandwidth" value={`${networkDetails.network_stats?.bandwidth_out_mbps?.toFixed(2) ?? '\u2014'} Mbps`} />
              <DetailRow label="Packet Loss" value={`${networkDetails.network_stats?.packet_loss?.toFixed(2) ?? '\u2014'}%`} valueColor="var(--color-warning)" />
              <DetailRow label="DNS Success" value={`${networkDetails.network_stats?.dns_success_rate?.toFixed(1) ?? '\u2014'}%`} valueColor="var(--color-success)" />
              <DetailRow label="DNS Latency" value={`${networkDetails.network_stats?.dns_latency_ms?.toFixed(0) ?? '\u2014'} ms`} />
              <DetailRow label="Port Usage" value={`${networkDetails.port_usage?.used ?? '\u2014'} / ${networkDetails.port_usage?.total ?? '\u2014'}`} />
            </div>
          ) : (
            <div className="text-muted">
              Network statistics unavailable.
            </div>
          )}
        </div>
      </div>

      {networkDetails?.connection_pools && networkDetails.connection_pools.length > 0 && (
        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Connection Pools</h3>
            <div className="card-subtitle">
              Resource utilization across HTTP/database pools
            </div>
          </div>
          <div className="metric-grid-auto">
            {networkDetails.connection_pools.map(pool => (
              <div key={pool.name} className="pool-card">
                <div className="text-bright mb-sm">{pool.name}</div>
                <div className="text-dim-xs">
                  Active: <span style={{ color: 'var(--color-text)' }}>{pool.active}</span> · Idle: <span style={{ color: 'var(--color-text)' }}>{pool.idle}</span>
                </div>
                <div className="text-dim-xs">
                  Waiting: <span style={{ color: 'var(--color-text)' }}>{pool.waiting}</span> / Max {pool.max_size}
                </div>
                <div style={{
                  marginTop: 'var(--spacing-xs)',
                  color: pool.leak_risk === 'high'
                    ? 'var(--color-error)'
                    : pool.leak_risk === 'medium'
                      ? 'var(--color-warning)'
                      : 'var(--color-success)'
                }}>
                  Leak risk: {pool.leak_risk}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </MetricDetailLayout>
  );
};
