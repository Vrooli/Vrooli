import { useMemo } from 'react';
import { Network } from 'lucide-react';

import type {
  MetricsResponse,
  DetailedMetrics,
  MetricHistory
} from '../../../types';
import { MetricDetailLayout, MetricLineChart } from './MetricDetailViews';
import { formatTimeLabel, buildSingleSeriesData } from './metricHelpers';

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
        style={{ padding: 'var(--spacing-lg)' }}
        data={networkData.map(point => ({ timestamp: point.timestamp, value: point.value }))}
        lines={[{ dataKey: 'value', name: 'TCP Connections', color: 'var(--color-accent)' }]}
        unit=""
        yDomain={['auto', 'auto']}
        valueFormatter={value => `${Math.round(value).toLocaleString()} connections`}
      />

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))', gap: 'var(--spacing-lg)' }}>
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>TCP States</h3>
          {networkDetails ? (
            <div className="detail-grid detail-grid-sm">
              {Object.entries(networkDetails.tcp_states).filter(([key]) => key !== 'total').map(([state, value]) => (
                <div key={state} className="detail-row">
                  <span className="detail-row-label">{state.toUpperCase()}</span>
                  <span style={{ color: 'var(--color-text-bright)', fontSize: 'var(--font-size-md)' }}>{Number(value).toLocaleString()}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-muted">
              Connection state metrics unavailable.
            </div>
          )}
        </div>

        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
          <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Network Health</h3>
          {networkDetails ? (
            <div className="detail-grid detail-grid-md">
              <div className="detail-row">
                <span className="detail-row-label">Ingress Bandwidth</span>
                <span className="detail-row-value">{networkDetails.network_stats?.bandwidth_in_mbps?.toFixed(2) ?? '—'} Mbps</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">Egress Bandwidth</span>
                <span className="detail-row-value">{networkDetails.network_stats?.bandwidth_out_mbps?.toFixed(2) ?? '—'} Mbps</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">Packet Loss</span>
                <span style={{ color: 'var(--color-warning)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats?.packet_loss?.toFixed(2) ?? '—'}%</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">DNS Success</span>
                <span style={{ color: 'var(--color-success)', fontSize: 'var(--font-size-lg)' }}>{networkDetails.network_stats?.dns_success_rate?.toFixed(1) ?? '—'}%</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">DNS Latency</span>
                <span className="detail-row-value">{networkDetails.network_stats?.dns_latency_ms?.toFixed(0) ?? '—'} ms</span>
              </div>
              <div className="detail-row">
                <span className="detail-row-label">Port Usage</span>
                <span className="detail-row-value">{networkDetails.port_usage?.used ?? '—'} / {networkDetails.port_usage?.total ?? '—'}</span>
              </div>
            </div>
          ) : (
            <div className="text-muted">
              Network statistics unavailable.
            </div>
          )}
        </div>
      </div>

      {networkDetails?.connection_pools && networkDetails.connection_pools.length > 0 && (
        <div className="card" style={{ padding: 'var(--spacing-lg)', display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
          <div>
            <h3 style={{ margin: 0, color: 'var(--color-text-bright)' }}>Connection Pools</h3>
            <div className="card-subtitle">
              Resource utilization across HTTP/database pools
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 'var(--spacing-md)' }}>
            {networkDetails.connection_pools.map(pool => (
              <div key={pool.name} style={{
                border: '1px solid var(--alpha-accent-15)',
                borderRadius: 'var(--border-radius-md)',
                padding: 'var(--spacing-md)',
                background: 'rgba(0, 0, 0, 0.4)'
              }}>
                <div style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-sm)' }}>{pool.name}</div>
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
