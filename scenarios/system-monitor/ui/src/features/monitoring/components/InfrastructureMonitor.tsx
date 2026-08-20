import { ChevronDown, ChevronUp, Zap } from 'lucide-react';
import type { InfrastructureMonitorData, SystemHealth, ConnectionPool, ServiceHealth } from '../../../types';
import { getUtilizationColor } from '../../../shared/utils/formatters';
import { getStatusColor, getHealthColor } from '../../../shared/utils/colors';

interface InfrastructureMonitorProps {
  data: InfrastructureMonitorData | null;
  isExpanded: boolean;
  onToggle: () => void;
  systemHealth?: SystemHealth | null;
}

export const InfrastructureMonitor = ({ data, isExpanded, onToggle, systemHealth }: InfrastructureMonitorProps) => {
  const fdInfo = systemHealth?.fileDescriptors;
  const inotifyWatchers = systemHealth?.inotifyWatchers;
  const watcherPercent = inotifyWatchers && inotifyWatchers.supported ? inotifyWatchers.watchesPercent : undefined;
  const watcherInstancePercent = inotifyWatchers && inotifyWatchers.supported ? inotifyWatchers.instancesPercent : undefined;

  return (
    <section className="monitoring-panel collapsible card">
      <div
        className={`panel-header clickable monitor-panel-header ${isExpanded ? 'monitor-panel-header-expanded' : ''}`}
        onClick={onToggle}
      >
        <h2 className="icon-text monitor-heading">
          <Zap size={20} />
          INFRASTRUCTURE MONITOR
        </h2>
        <span className="expand-arrow text-accent">
          {isExpanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
        </span>
      </div>

      {isExpanded && (
        <div className="panel-content">
          {data ? (
            <div>
              <div className="monitor-grid">
                <div className="monitor-section">
                  <h3 className="monitor-section-heading">
                    Database Pools:
                  </h3>
                  <div className="pool-list">
                    {(data.databasePools ?? []).map((pool: ConnectionPool, index: number) => (
                      <div key={index} className="pool-item" style={{
                        border: `1px solid ${getHealthColor(pool.healthy)}`
                      }}>
                        <div>
                          <div>{pool.name}</div>
                          <div className="text-dim-xs">
                            Active: {pool.active} | Idle: {pool.idle} | Max: {pool.maxSize}
                          </div>
                        </div>
                        <span style={{
                          color: getHealthColor(pool.healthy)
                        }}>
                          {pool.healthy ? '●' : '●'}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="monitor-section">
                  <h3 className="monitor-section-heading">
                    HTTP Client Pools:
                  </h3>
                  <div className="pool-list">
                    {(data.httpClientPools ?? []).map((pool: ConnectionPool, index: number) => (
                      <div key={index} className="pool-item" style={{
                        border: `1px solid ${pool.healthy ? 'var(--color-success)' : 'var(--color-error)'}`
                      }}>
                        <div>
                          <div>{pool.name}</div>
                          <div className="text-dim-xs">
                            Active: {pool.active} | Waiting: {pool.waiting}
                          </div>
                        </div>
                        <span style={{
                          color: getStatusColor(pool.leakRisk ?? '')
                        }}>
                          {pool.leakRisk}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="monitor-section">
                  <h3 className="monitor-section-heading">
                    Message Queues:
                  </h3>
                  <div className="queue-stats">
                    <div className="stat-group mb-md">
                      <h4 className="monitor-sub-heading">
                        Redis Pub/Sub:
                      </h4>
                      <div className="stat-item">
                        <span className="stat-label">Subscribers:</span>
                        <span className="stat-value text-bright">
                          {data.messageQueues?.redisPubsub?.subscribers ?? '—'}
                        </span>
                      </div>
                      <div className="stat-item">
                        <span className="stat-label">Channels:</span>
                        <span className="stat-value text-bright">
                          {data.messageQueues?.redisPubsub?.channels ?? '—'}
                        </span>
                      </div>
                    </div>

                    <div className="stat-group">
                      <h4 className="monitor-sub-heading">
                        Background Jobs:
                      </h4>
                      <div className="stat-item">
                        <span className="stat-label">Pending:</span>
                        <span className="stat-value text-warning">
                          {data.messageQueues?.backgroundJobs?.pending ?? '—'}
                        </span>
                      </div>
                      <div className="stat-item">
                        <span className="stat-label">Active:</span>
                        <span className="stat-value text-success">
                          {data.messageQueues?.backgroundJobs?.active ?? '—'}
                        </span>
                      </div>
                      <div className="stat-item">
                        <span className="stat-label">Failed:</span>
                        <span className="stat-value" style={{
                          color: (data.messageQueues?.backgroundJobs?.failed ?? 0) > 0 ? 'var(--color-error)' : 'var(--color-success)'
                        }}>
                          {data.messageQueues?.backgroundJobs?.failed ?? '—'}
                        </span>
                      </div>
                  </div>
                </div>

                {(fdInfo || (inotifyWatchers && inotifyWatchers.supported)) && (
                  <div className="monitor-section">
                    <h3 className="monitor-section-heading">
                      Kernel Resource Limits:
                    </h3>
                    <div className="flex-col-gap-sm">
                      {fdInfo && (
                        <div className="kernel-resource-card">
                          <div className="kernel-resource-header">
                            <span>File Descriptors</span>
                            <span style={{ color: getUtilizationColor(fdInfo.percent) }}>
                              {fdInfo.percent.toFixed(1)}%
                            </span>
                          </div>
                          <div className="text-dim-xs">
                            {fdInfo.used.toLocaleString()} / {fdInfo.max.toLocaleString()}
                          </div>
                        </div>
                      )}

                      {inotifyWatchers && (
                        <div className="kernel-resource-card">
                          <div className="kernel-resource-header">
                            <span>Inotify Watches</span>
                            <span style={{ color: watcherPercent !== undefined ? getUtilizationColor(watcherPercent) : 'var(--color-text-secondary)' }}>
                              {watcherPercent !== undefined ? `${watcherPercent.toFixed(1)}%` : '—'}
                            </span>
                          </div>
                          {inotifyWatchers.supported ? (
                            <>
                              <div className="text-dim-xs">
                                {inotifyWatchers.watchesUsed.toLocaleString()} / {inotifyWatchers.watchesMax.toLocaleString()} watch descriptors
                              </div>
                              <div className="text-dim-xs">
                                Instances: {inotifyWatchers.instancesUsed.toLocaleString()} / {inotifyWatchers.instancesMax.toLocaleString()} ({watcherInstancePercent !== undefined ? `${watcherInstancePercent.toFixed(1)}%` : '—'})
                              </div>
                            </>
                          ) : (
                            <div className="text-dim-xs">
                              Inotify metrics unavailable on this host.
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>

                <div className="monitor-section">
                  <h3 className="monitor-section-heading">
                    Service Dependencies & Certificates:
                  </h3>

                  <div className="flex-col-gap-md">
                    <div>
                      <h4 className="monitor-sub-heading">
                        Service Dependencies:
                      </h4>
                      {systemHealth?.serviceDependencies && systemHealth.serviceDependencies.length > 0 ? (
                        <div className="flex-col-gap-sm">
                          {systemHealth.serviceDependencies.map((service: ServiceHealth, index: number) => (
                            <div
                              key={`${service.name}-${index}`}
                              className="pool-item"
                              style={{
                                border: `1px solid ${service.status === 'healthy' ? 'var(--color-success)' : 'var(--color-error)'}`
                              }}
                            >
                              <div>
                                <div>{service.name}</div>
                                <div className="text-dim-xs">
                                  {service.status === 'healthy' ? 'Operational' : 'Needs attention'}
                                </div>
                              </div>
                              <div data-sm-style="sm-style-905bfede49">
                                <div style={{
                                  color: service.status === 'healthy' ? 'var(--color-success)' : 'var(--color-error)',
                                  fontWeight: 600
                                }}>
                                  {service.status}
                                </div>
                                <div className="text-dim-xs">
                                  {service.latencyMs.toFixed(0)} ms
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="unavailable-notice">
                          Dependency telemetry unavailable.
                        </div>
                      )}
                    </div>

                    <div>
                      <h4 className="monitor-sub-heading">
                        Certificates:
                      </h4>
                      {systemHealth?.certificates && systemHealth.certificates.length > 0 ? (
                        <div className="flex-col-gap-sm">
                          {systemHealth.certificates.map((cert, index) => {
                            const expiryColor = cert.daysToExpiry < 15
                              ? 'var(--color-error)'
                              : cert.daysToExpiry < 45
                                ? 'var(--color-warning)'
                                : 'var(--color-success)';
                            return (
                              <div
                                key={`${cert.domain}-${index}`}
                                className="pool-item"
                                style={{
                                  border: `1px solid ${expiryColor}`
                                }}
                              >
                                <div>
                                  <div>{cert.domain}</div>
                                  <div className="text-dim-xs">
                                    Status: {cert.status}
                                  </div>
                                </div>
                                <div style={{ textAlign: 'right', color: expiryColor, fontWeight: 600 }}>
                                  {cert.daysToExpiry} days
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="unavailable-notice">
                          No certificate data reported.
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="monitor-loading">
              LOADING INFRASTRUCTURE DATA...
            </div>
          )}
        </div>
      )}
    </section>
  );
};
