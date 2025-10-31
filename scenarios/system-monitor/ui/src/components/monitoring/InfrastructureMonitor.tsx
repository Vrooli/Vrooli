import { ChevronDown, ChevronUp, Zap } from 'lucide-react';
import type { InfrastructureMonitorData, SystemHealth } from '../../types';

interface InfrastructureMonitorProps {
  data: InfrastructureMonitorData | null;
  isExpanded: boolean;
  onToggle: () => void;
  systemHealth?: SystemHealth | null;
}

export const InfrastructureMonitor = ({ data, isExpanded, onToggle, systemHealth }: InfrastructureMonitorProps) => {
  const getUtilizationColor = (percent: number) => (
    percent >= 85 ? 'var(--color-error)' : percent >= 70 ? 'var(--color-warning)' : 'var(--color-success)'
  );
  const fdInfo = systemHealth?.file_descriptors;
  const inotifyWatchers = systemHealth?.inotify_watchers;
  const watcherPercent = inotifyWatchers && inotifyWatchers.supported ? inotifyWatchers.watches_percent : undefined;
  const watcherInstancePercent = inotifyWatchers && inotifyWatchers.supported ? inotifyWatchers.instances_percent : undefined;

  return (
    <section className="monitoring-panel collapsible card">
      <div 
        className="panel-header clickable" 
        onClick={onToggle}
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          cursor: 'pointer',
          marginBottom: isExpanded ? 'var(--spacing-md)' : 0
        }}
      >
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)', display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
          <Zap size={20} />
          INFRASTRUCTURE MONITOR
        </h2>
        <span className="expand-arrow" style={{ color: 'var(--color-accent)' }}>
          {isExpanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
        </span>
      </div>
      
      {isExpanded && (
        <div className="panel-content">
          {data ? (
            <div>
              <div className="monitor-grid" style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
                gap: 'var(--spacing-lg)',
                marginBottom: 'var(--spacing-lg)'
              }}>
                <div className="monitor-section">
                  <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                    Database Pools:
                  </h3>
                  <div className="pool-list">
                    {data.database_pools.map((pool, index) => (
                      <div key={index} style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: `1px solid ${pool.healthy ? 'var(--color-success)' : 'var(--color-error)'}`,
                        borderRadius: 'var(--border-radius-sm)',
                        marginBottom: 'var(--spacing-xs)',
                        fontSize: 'var(--font-size-sm)'
                      }}>
                        <div>
                          <div>{pool.name}</div>
                          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                            Active: {pool.active} | Idle: {pool.idle} | Max: {pool.max_size}
                          </div>
                        </div>
                        <span style={{ 
                          color: pool.healthy ? 'var(--color-success)' : 'var(--color-error)'
                        }}>
                          {pool.healthy ? '●' : '●'}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
                
                <div className="monitor-section">
                  <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                    HTTP Client Pools:
                  </h3>
                  <div className="pool-list">
                    {data.http_client_pools.map((pool, index) => (
                      <div key={index} style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(0, 0, 0, 0.3)',
                        border: `1px solid ${pool.healthy ? 'var(--color-success)' : 'var(--color-error)'}`,
                        borderRadius: 'var(--border-radius-sm)',
                        marginBottom: 'var(--spacing-xs)',
                        fontSize: 'var(--font-size-sm)'
                      }}>
                        <div>
                          <div>{pool.name}</div>
                          <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                            Active: {pool.active} | Waiting: {pool.waiting}
                          </div>
                        </div>
                        <span style={{ 
                          color: pool.leak_risk === 'high' ? 'var(--color-error)' : 
                                 pool.leak_risk === 'medium' ? 'var(--color-warning)' : 
                                 'var(--color-success)'
                        }}>
                          {pool.leak_risk}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
                
                <div className="monitor-section">
                  <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                    Message Queues:
                  </h3>
                  <div className="queue-stats">
                    <div className="stat-group" style={{ marginBottom: 'var(--spacing-md)' }}>
                      <h4 style={{ color: 'var(--color-accent)', marginBottom: 'var(--spacing-sm)' }}>
                        Redis Pub/Sub:
                      </h4>
                      <div className="stat-item" style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        marginBottom: 'var(--spacing-xs)'
                      }}>
                        <span className="stat-label">Subscribers:</span>
                        <span className="stat-value" style={{ color: 'var(--color-text-bright)' }}>
                          {data.message_queues.redis_pubsub.subscribers}
                        </span>
                      </div>
                      <div className="stat-item" style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        marginBottom: 'var(--spacing-xs)'
                      }}>
                        <span className="stat-label">Channels:</span>
                        <span className="stat-value" style={{ color: 'var(--color-text-bright)' }}>
                          {data.message_queues.redis_pubsub.channels}
                        </span>
                      </div>
                    </div>
                    
                    <div className="stat-group">
                      <h4 style={{ color: 'var(--color-accent)', marginBottom: 'var(--spacing-sm)' }}>
                        Background Jobs:
                      </h4>
                      <div className="stat-item" style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        marginBottom: 'var(--spacing-xs)'
                      }}>
                        <span className="stat-label">Pending:</span>
                        <span className="stat-value" style={{ color: 'var(--color-warning)' }}>
                          {data.message_queues.background_jobs.pending}
                        </span>
                      </div>
                      <div className="stat-item" style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        marginBottom: 'var(--spacing-xs)'
                      }}>
                        <span className="stat-label">Active:</span>
                        <span className="stat-value" style={{ color: 'var(--color-success)' }}>
                          {data.message_queues.background_jobs.active}
                        </span>
                      </div>
                      <div className="stat-item" style={{ 
                        display: 'flex', 
                        justifyContent: 'space-between',
                        marginBottom: 'var(--spacing-xs)'
                      }}>
                        <span className="stat-label">Failed:</span>
                        <span className="stat-value" style={{ 
                          color: data.message_queues.background_jobs.failed > 0 ? 'var(--color-error)' : 'var(--color-success)'
                        }}>
                          {data.message_queues.background_jobs.failed}
                        </span>
                      </div>
                  </div>
                </div>

                {(fdInfo || (inotifyWatchers && inotifyWatchers.supported)) && (
                  <div className="monitor-section">
                    <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                      Kernel Resource Limits:
                    </h3>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-sm)' }}>
                      {fdInfo && (
                        <div style={{
                          padding: 'var(--spacing-sm)',
                          background: 'rgba(0, 0, 0, 0.3)',
                          borderRadius: 'var(--border-radius-sm)',
                          border: '1px solid var(--alpha-accent-15)'
                        }}>
                          <div style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            fontSize: 'var(--font-size-sm)',
                            color: 'var(--color-text-bright)'
                          }}>
                            <span>File Descriptors</span>
                            <span style={{ color: getUtilizationColor(fdInfo.percent) }}>
                              {fdInfo.percent.toFixed(1)}%
                            </span>
                          </div>
                          <div style={{
                            color: 'var(--color-text-dim)',
                            fontSize: 'var(--font-size-xs)',
                            letterSpacing: '0.06em'
                          }}>
                            {fdInfo.used.toLocaleString()} / {fdInfo.max.toLocaleString()}
                          </div>
                        </div>
                      )}

                      {inotifyWatchers && (
                        <div style={{
                          padding: 'var(--spacing-sm)',
                          background: 'rgba(0, 0, 0, 0.3)',
                          borderRadius: 'var(--border-radius-sm)',
                          border: '1px solid var(--alpha-accent-15)'
                        }}>
                          <div style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            fontSize: 'var(--font-size-sm)',
                            color: 'var(--color-text-bright)'
                          }}>
                            <span>Inotify Watches</span>
                            <span style={{ color: watcherPercent !== undefined ? getUtilizationColor(watcherPercent) : 'var(--color-text-dim)' }}>
                              {watcherPercent !== undefined ? `${watcherPercent.toFixed(1)}%` : '—'}
                            </span>
                          </div>
                          {inotifyWatchers.supported ? (
                            <>
                              <div style={{
                                color: 'var(--color-text-dim)',
                                fontSize: 'var(--font-size-xs)',
                                letterSpacing: '0.06em'
                              }}>
                                {inotifyWatchers.watches_used.toLocaleString()} / {inotifyWatchers.watches_max.toLocaleString()} watch descriptors
                              </div>
                              <div style={{
                                color: 'var(--color-text-dim)',
                                fontSize: 'var(--font-size-xs)',
                                letterSpacing: '0.06em'
                              }}>
                                Instances: {inotifyWatchers.instances_used.toLocaleString()} / {inotifyWatchers.instances_max.toLocaleString()} ({watcherInstancePercent !== undefined ? `${watcherInstancePercent.toFixed(1)}%` : '—'})
                              </div>
                            </>
                          ) : (
                            <div style={{
                              color: 'var(--color-text-dim)',
                              fontSize: 'var(--font-size-xs)',
                              letterSpacing: '0.06em'
                            }}>
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
                  <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                    Service Dependencies & Certificates:
                  </h3>

                  <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-md)' }}>
                    <div>
                      <h4 style={{ color: 'var(--color-accent)', marginBottom: 'var(--spacing-sm)' }}>
                        Service Dependencies:
                      </h4>
                      {systemHealth?.service_dependencies && systemHealth.service_dependencies.length > 0 ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                          {systemHealth.service_dependencies.map((service, index) => (
                            <div
                              key={`${service.name}-${index}`}
                              style={{
                                display: 'flex',
                                justifyContent: 'space-between',
                                padding: 'var(--spacing-sm)',
                                background: 'rgba(0, 0, 0, 0.3)',
                                borderRadius: 'var(--border-radius-sm)',
                                border: `1px solid ${service.status === 'healthy' ? 'var(--color-success)' : 'var(--color-error)'}`,
                                fontSize: 'var(--font-size-sm)'
                              }}
                            >
                              <div>
                                <div>{service.name}</div>
                                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                                  {service.status === 'healthy' ? 'Operational' : 'Needs attention'}
                                </div>
                              </div>
                              <div style={{ textAlign: 'right' }}>
                                <div style={{
                                  color: service.status === 'healthy' ? 'var(--color-success)' : 'var(--color-error)',
                                  fontWeight: 600
                                }}>
                                  {service.status}
                                </div>
                                <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                                  {service.latency_ms.toFixed(0)} ms
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div style={{
                          padding: 'var(--spacing-sm)',
                          background: 'rgba(0, 0, 0, 0.3)',
                          borderRadius: 'var(--border-radius-sm)',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)'
                        }}>
                          Dependency telemetry unavailable.
                        </div>
                      )}
                    </div>

                    <div>
                      <h4 style={{ color: 'var(--color-accent)', marginBottom: 'var(--spacing-sm)' }}>
                        Certificates:
                      </h4>
                      {systemHealth?.certificates && systemHealth.certificates.length > 0 ? (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--spacing-xs)' }}>
                          {systemHealth.certificates.map((cert, index) => {
                            const expiryColor = cert.days_to_expiry < 15
                              ? 'var(--color-error)'
                              : cert.days_to_expiry < 45
                                ? 'var(--color-warning)'
                                : 'var(--color-success)';
                            return (
                              <div
                                key={`${cert.domain}-${index}`}
                                style={{
                                  display: 'flex',
                                  justifyContent: 'space-between',
                                  alignItems: 'center',
                                  padding: 'var(--spacing-sm)',
                                  background: 'rgba(0, 0, 0, 0.3)',
                                  borderRadius: 'var(--border-radius-sm)',
                                  border: `1px solid ${expiryColor}`,
                                  fontSize: 'var(--font-size-sm)'
                                }}
                              >
                                <div>
                                  <div>{cert.domain}</div>
                                  <div style={{ color: 'var(--color-text-dim)', fontSize: 'var(--font-size-xs)' }}>
                                    Status: {cert.status}
                                  </div>
                                </div>
                                <div style={{ textAlign: 'right', color: expiryColor, fontWeight: 600 }}>
                                  {cert.days_to_expiry} days
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      ) : (
                        <div style={{
                          padding: 'var(--spacing-sm)',
                          background: 'rgba(0, 0, 0, 0.3)',
                          borderRadius: 'var(--border-radius-sm)',
                          color: 'var(--color-text-dim)',
                          fontSize: 'var(--font-size-sm)'
                        }}>
                          No certificate data reported.
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div style={{ 
              textAlign: 'center', 
              color: 'var(--color-text-dim)', 
              padding: 'var(--spacing-xl)' 
            }}>
              LOADING INFRASTRUCTURE DATA...
            </div>
          )}
        </div>
      )}
    </section>
  );
};
