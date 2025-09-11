import { ChevronDown, ChevronUp } from 'lucide-react';
import type { CardType, CPUMetrics, MemoryMetrics, NetworkMetrics, SystemHealth } from '../../types';

interface MetricCardProps {
  type: CardType;
  label: string;
  unit: string;
  value: number | string;
  isExpanded: boolean;
  onToggle: () => void;
  details?: CPUMetrics | MemoryMetrics | NetworkMetrics | SystemHealth;
  alertCount: number;
  isStatusCard?: boolean;
}

export const MetricCard = ({ 
  type, 
  label, 
  unit, 
  value, 
  isExpanded, 
  onToggle, 
  details, 
  alertCount,
  isStatusCard = false 
}: MetricCardProps) => {
  
  const getProgressValue = (): number => {
    if (typeof value === 'number') {
      return isStatusCard ? 100 : Math.min(value, 100);
    }
    return 0;
  };

  const getValueColor = (): string => {
    if (isStatusCard) {
      return 'var(--color-success)';
    }
    
    const numValue = typeof value === 'number' ? value : 0;
    if (numValue >= 90) return 'var(--color-error)';
    if (numValue >= 70) return 'var(--color-warning)';
    return 'var(--color-text-bright)';
  };

  const renderExpandedContent = () => {
    if (!isExpanded || !details) return null;

    switch (type) {
      case 'cpu':
        const cpuDetails = details as CPUMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Top Processes by CPU:
              </h4>
              <div className="process-list">
                {cpuDetails.top_processes.slice(0, 5).map((process) => (
                  <div key={process.pid} style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    margin: 'var(--spacing-xs) 0',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    <span>{process.name} ({process.pid})</span>
                    <span style={{ color: 'var(--color-accent)' }}>
                      {process.cpu_percent.toFixed(1)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Load Average:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {cpuDetails.load_average.slice(0, 3).map(load => load.toFixed(2)).join(', ')}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Context Switches:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {cpuDetails.context_switches.toLocaleString()}
                </span>
              </div>
            </div>
            
            <div className="detail-item">
              <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                Total Goroutines:
              </span>
              <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                {cpuDetails.total_goroutines}
              </span>
            </div>
          </div>
        );

      case 'memory':
        const memoryDetails = details as MemoryMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Top Processes by Memory:
              </h4>
              <div className="process-list">
                {memoryDetails.top_processes.slice(0, 5).map((process) => (
                  <div key={process.pid} style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    margin: 'var(--spacing-xs) 0',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    <span>{process.name} ({process.pid})</span>
                    <span style={{ color: 'var(--color-accent)' }}>
                      {process.memory_mb.toFixed(1)} MB
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Swap Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {memoryDetails.swap_usage.percent.toFixed(1)}%
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Disk Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {memoryDetails.disk_usage.percent.toFixed(1)}%
                </span>
              </div>
            </div>
            
            {memoryDetails.growth_patterns.length > 0 && (
              <div className="detail-section">
                <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                  Memory Growth Patterns:
                </h4>
                <div className="growth-patterns">
                  {memoryDetails.growth_patterns.slice(0, 3).map((pattern, index) => (
                    <div key={index} style={{ 
                      margin: 'var(--spacing-xs) 0',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{pattern.process}: </span>
                      <span style={{ 
                        color: pattern.risk_level === 'high' ? 'var(--color-error)' : 
                               pattern.risk_level === 'medium' ? 'var(--color-warning)' : 
                               'var(--color-accent)'
                      }}>
                        {pattern.growth_mb_per_hour.toFixed(1)} MB/hr ({pattern.risk_level})
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        );

      case 'network':
        const networkDetails = details as NetworkMetrics;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
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
                  {networkDetails.tcp_states.established}
                </span></div>
                <div>Time Wait: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.time_wait}
                </span></div>
                <div>Listen: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.listen}
                </span></div>
                <div>Total: <span style={{ color: 'var(--color-accent)' }}>
                  {networkDetails.tcp_states.total}
                </span></div>
              </div>
            </div>
            
            <div className="detail-row" style={{ 
              display: 'grid', 
              gridTemplateColumns: '1fr 1fr', 
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-sm)'
            }}>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  Port Usage:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {networkDetails.port_usage.used} / {networkDetails.port_usage.total}
                </span>
              </div>
              <div className="detail-item">
                <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                  DNS Health:
                </span>
                <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                  {networkDetails.network_stats.dns_success_rate.toFixed(1)}%
                </span>
              </div>
            </div>
          </div>
        );

      case 'system':
        const systemDetails = details as SystemHealth;
        return (
          <div className="metric-details" style={{ marginTop: 'var(--spacing-md)' }}>
            <div className="detail-item" style={{ marginBottom: 'var(--spacing-md)' }}>
              <span className="detail-label" style={{ color: 'var(--color-text-dim)' }}>
                File Descriptors:
              </span>
              <span className="detail-value" style={{ color: 'var(--color-text-bright)' }}>
                {systemDetails.file_descriptors.used} / {systemDetails.file_descriptors.max} 
                ({systemDetails.file_descriptors.percent.toFixed(1)}%)
              </span>
            </div>
            
            <div className="detail-section" style={{ marginBottom: 'var(--spacing-md)' }}>
              <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                Service Dependencies:
              </h4>
              <div className="service-list">
                {systemDetails.service_dependencies.slice(0, 5).map((service, index) => (
                  <div key={index} style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    margin: 'var(--spacing-xs) 0',
                    fontSize: 'var(--font-size-sm)'
                  }}>
                    <span>{service.name}</span>
                    <span style={{ 
                      color: service.status === 'healthy' ? 'var(--color-success)' : 'var(--color-error)'
                    }}>
                      {service.status}
                    </span>
                  </div>
                ))}
              </div>
            </div>
            
            {systemDetails.certificates.length > 0 && (
              <div className="detail-section">
                <h4 style={{ margin: '0 0 var(--spacing-sm) 0', color: 'var(--color-text-bright)' }}>
                  Certificates:
                </h4>
                <div className="certificate-list">
                  {systemDetails.certificates.slice(0, 3).map((cert, index) => (
                    <div key={index} style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between',
                      margin: 'var(--spacing-xs) 0',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{cert.domain}</span>
                      <span style={{ 
                        color: cert.days_to_expiry < 30 ? 'var(--color-error)' : 
                               cert.days_to_expiry < 60 ? 'var(--color-warning)' : 
                               'var(--color-success)'
                      }}>
                        {cert.days_to_expiry} days
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <div 
      className={`metric-card expandable ${isExpanded ? 'expanded' : ''}`}
      onClick={onToggle}
      style={{
        background: 'rgba(0, 0, 0, 0.9)',
        border: '1px solid var(--color-accent)',
        borderRadius: 'var(--border-radius-lg)',
        padding: 'var(--spacing-lg)',
        cursor: 'pointer',
        transition: 'all var(--transition-normal)',
        ...(isExpanded && {
          borderColor: 'var(--color-text-bright)',
          boxShadow: '0 0 20px var(--color-matrix-glow)'
        })
      }}
    >
      <div className="metric-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 'var(--spacing-md)'
      }}>
        <span className="metric-label" style={{
          fontSize: 'var(--font-size-sm)',
          color: 'var(--color-text-bright)',
          letterSpacing: '0.1em'
        }}>
          {label}
        </span>
        
        <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
          {alertCount > 0 && (
            <span className="alert-badge" style={{
              background: 'var(--color-error)',
              color: 'var(--color-background)',
              padding: 'var(--spacing-xs) var(--spacing-sm)',
              borderRadius: '50px',
              fontSize: 'var(--font-size-xs)',
              fontWeight: 'bold',
              minWidth: '20px',
              textAlign: 'center'
            }}>
              {alertCount}
            </span>
          )}
          
          <span className="metric-unit" style={{
            fontSize: 'var(--font-size-sm)',
            color: 'var(--color-text-dim)'
          }}>
            {unit}
          </span>
          
          <span className="expand-arrow" style={{ color: 'var(--color-accent)' }}>
            {isExpanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
          </span>
        </div>
      </div>

      <div className="metric-value" style={{
        fontSize: 'var(--font-size-xxl)',
        fontWeight: 'bold',
        color: getValueColor(),
        textShadow: 'var(--text-shadow-glow)',
        marginBottom: 'var(--spacing-sm)'
      }}>
        {typeof value === 'number' ? value.toFixed(1) : value}
      </div>

      {!isStatusCard && (
        <div className="metric-bar" style={{
          width: '100%',
          height: '4px',
          background: 'rgba(0, 255, 0, 0.2)',
          borderRadius: '2px',
          overflow: 'hidden'
        }}>
          <div 
            className="metric-fill" 
            style={{
              height: '100%',
              width: `${getProgressValue()}%`,
              background: 'linear-gradient(90deg, var(--color-accent), var(--color-text-bright))',
              transition: 'width var(--transition-normal)',
              boxShadow: '0 0 10px var(--color-matrix-green)'
            }}
          />
        </div>
      )}

      {renderExpandedContent()}
    </div>
  );
};