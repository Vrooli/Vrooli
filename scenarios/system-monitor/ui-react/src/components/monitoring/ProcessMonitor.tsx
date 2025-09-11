import { ChevronDown, ChevronUp } from 'lucide-react';
import type { ProcessMonitorData } from '../../types';

interface ProcessMonitorProps {
  data: ProcessMonitorData | null;
  isExpanded: boolean;
  onToggle: () => void;
}

export const ProcessMonitor = ({ data, isExpanded, onToggle }: ProcessMonitorProps) => {
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
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)' }}>
          🔍 PROCESS MONITOR
        </h2>
        <span className="expand-arrow" style={{ color: 'var(--color-accent)' }}>
          {isExpanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
        </span>
      </div>
      
      {isExpanded && (
        <div className="panel-content">
          {data ? (
            <div className="monitor-grid" style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
              gap: 'var(--spacing-lg)',
              marginBottom: 'var(--spacing-lg)'
            }}>
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                  Process Health:
                </h3>
                <div className="health-stats">
                  <div className="stat-item" style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    marginBottom: 'var(--spacing-sm)'
                  }}>
                    <span className="stat-label">Total Processes:</span>
                    <span className="stat-value" style={{ color: 'var(--color-accent)' }}>
                      {data.process_health.total_processes}
                    </span>
                  </div>
                  <div className="stat-item alert" style={{ 
                    display: 'flex', 
                    justifyContent: 'space-between',
                    marginBottom: 'var(--spacing-sm)'
                  }}>
                    <span className="stat-label">Zombie Processes:</span>
                    <span className="stat-value" style={{ 
                      color: data.process_health.zombie_processes.length > 0 ? 'var(--color-error)' : 'var(--color-success)'
                    }}>
                      {data.process_health.zombie_processes.length}
                    </span>
                  </div>
                </div>
                
                {data.process_health.zombie_processes.length > 0 && (
                  <div className="process-alerts">
                    {data.process_health.zombie_processes.slice(0, 5).map(process => (
                      <div key={process.pid} style={{
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(255, 0, 64, 0.1)',
                        border: '1px solid var(--color-error)',
                        borderRadius: 'var(--border-radius-sm)',
                        marginBottom: 'var(--spacing-xs)',
                        fontSize: 'var(--font-size-sm)'
                      }}>
                        {process.name} (PID: {process.pid})
                      </div>
                    ))}
                  </div>
                )}
              </div>
              
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                  High Thread Count:
                </h3>
                <div className="thread-list">
                  {data.process_health.high_thread_count.slice(0, 5).map(process => (
                    <div key={process.pid} style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(0, 0, 0, 0.3)',
                      border: '1px solid var(--color-accent)',
                      borderRadius: 'var(--border-radius-sm)',
                      marginBottom: 'var(--spacing-xs)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{process.name} ({process.pid})</span>
                      <span style={{ color: 'var(--color-warning)' }}>
                        {process.threads} threads
                      </span>
                    </div>
                  ))}
                </div>
              </div>
              
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                  Resource Leak Candidates:
                </h3>
                <div className="leak-list">
                  {data.process_health.leak_candidates.slice(0, 5).map(process => (
                    <div key={process.pid} style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(255, 170, 0, 0.1)',
                      border: '1px solid var(--color-warning)',
                      borderRadius: 'var(--border-radius-sm)',
                      marginBottom: 'var(--spacing-xs)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{process.name} ({process.pid})</span>
                      <span style={{ color: 'var(--color-warning)' }}>
                        {process.memory_mb.toFixed(0)} MB
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div style={{ 
              textAlign: 'center', 
              color: 'var(--color-text-dim)', 
              padding: 'var(--spacing-xl)' 
            }}>
              SCANNING SYSTEM...
            </div>
          )}
        </div>
      )}
    </section>
  );
};