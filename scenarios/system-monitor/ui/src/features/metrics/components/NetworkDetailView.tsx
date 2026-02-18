import { useMemo } from 'react';
import { Network } from 'lucide-react';

import { DetailRow } from '../../../shared/components/DetailRow';
import { getStatusColor } from '../../../shared/utils/colors';
import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory,
  ConnectionPool,
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatProtoTimestamp } from '../../../shared/utils/formatters';
import { buildSingleSeriesData } from '../../../shared/utils/chartData';

export interface NetworkDetailViewProps {
  metrics: MetricsResponse | null;
  detailedMetrics: DetailedMetrics | null;
  metricHistory: MetricHistory | null;
  onBack: () => void;
}

export const NetworkDetailView = ({ metrics, detailedMetrics, metricHistory, onBack }: NetworkDetailViewProps) => {
  const networkData = useMemo(() => buildSingleSeriesData(metricHistory?.network), [metricHistory?.network]);
  const networkDetails = detailedMetrics?.networkDetails;
  const totalConnections = metrics?.tcpConnections ?? networkDetails?.tcpStates?.total ?? 0;

  const subhead = detailedMetrics?.timestamp
    ? `Updated ${formatProtoTimestamp(detailedMetrics.timestamp)}`
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
        lines={[{ dataKey: 'value', name: 'TCP Connections', color: 'var(--color-primary)' }]}
        unit=""
        yDomain={['auto', 'auto']}
        valueFormatter={value => `${Math.round(value).toLocaleString()} connections`}
      />

      <div className="metric-grid-auto-lg">
        <div className="card flex-col-gap-sm">
          <h3 className="section-heading">TCP States</h3>
          {networkDetails?.tcpStates ? (
            <div className="detail-grid detail-grid-sm">
              {Object.entries(networkDetails.tcpStates).filter(([key]) => key !== 'total' && key !== '$typeName' && key !== '$unknown').map(([state, value]) => (
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
              <DetailRow label="Ingress Bandwidth" value={`${networkDetails.networkStats?.bandwidthInMbps?.toFixed(2) ?? '—'} Mbps`} />
              <DetailRow label="Egress Bandwidth" value={`${networkDetails.networkStats?.bandwidthOutMbps?.toFixed(2) ?? '—'} Mbps`} />
              <DetailRow label="Packet Loss" value={`${networkDetails.networkStats?.packetLoss?.toFixed(2) ?? '—'}%`} valueColor="var(--color-warning)" />
              <DetailRow label="DNS Success" value={`${networkDetails.networkStats?.dnsSuccessRate?.toFixed(1) ?? '—'}%`} valueColor="var(--color-success)" />
              <DetailRow label="DNS Latency" value={`${networkDetails.networkStats?.dnsLatencyMs?.toFixed(0) ?? '—'} ms`} />
              <DetailRow label="Port Usage" value={`${networkDetails.portUsage?.used ?? '—'} / ${networkDetails.portUsage?.total ?? '—'}`} />
            </div>
          ) : (
            <div className="text-muted">
              Network statistics unavailable.
            </div>
          )}
        </div>
      </div>

      {networkDetails?.connectionPools && networkDetails.connectionPools.length > 0 && (
        <div className="card flex-col-gap-md">
          <div>
            <h3 className="section-heading">Connection Pools</h3>
            <div className="card-subtitle">
              Resource utilization across HTTP/database pools
            </div>
          </div>
          <div className="metric-grid-auto">
            {networkDetails.connectionPools.map((pool: ConnectionPool) => (
              <div key={pool.name} className="pool-card">
                <div className="text-bright mb-sm">{pool.name}</div>
                <div className="text-dim-xs">
                  Active: <span style={{ color: 'var(--color-text)' }}>{pool.active}</span> · Idle: <span style={{ color: 'var(--color-text)' }}>{pool.idle}</span>
                </div>
                <div className="text-dim-xs">
                  Waiting: <span style={{ color: 'var(--color-text)' }}>{pool.waiting}</span> / Max {pool.maxSize}
                </div>
                <div style={{
                  marginTop: 'var(--spacing-xs)',
                  color: getStatusColor(pool.leakRisk ?? '')
                }}>
                  Leak risk: {pool.leakRisk}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </MetricDetailLayout>
  );
};
