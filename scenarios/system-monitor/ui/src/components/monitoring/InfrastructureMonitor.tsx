import { ChevronDown, ChevronUp, Zap } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { InfrastructureMonitorData } from '../../types';

interface InfrastructureMonitorProps {
  data: InfrastructureMonitorData | null;
  isExpanded: boolean;
  onToggle: () => void;
}

export const InfrastructureMonitor = ({ data, isExpanded, onToggle }: InfrastructureMonitorProps) => {
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
                </div>
              </div>
              
              <div style={{
                marginTop: 'var(--spacing-lg)',
                textAlign: 'center',
                display: 'flex',
                flexDirection: 'column',
                gap: 'var(--spacing-sm)',
                alignItems: 'center'
              }}>
                <div style={{ color: 'var(--color-text-dim)', letterSpacing: '0.08em' }}>
                  Detailed storage metrics now live on the dedicated Disk performance view.
                </div>
                <Link
                  to="/metrics/disk"
                  className="btn btn-action"
                  style={{ textTransform: 'uppercase', letterSpacing: '0.1em', fontSize: 'var(--font-size-xs)' }}
                >
                  View Disk Performance
                </Link>
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
