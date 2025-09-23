import { ChevronDown, ChevronUp, Search, X, AlertTriangle } from 'lucide-react';
import { useState } from 'react';
import type { ProcessMonitorData } from '../../types';

interface ProcessMonitorProps {
  data: ProcessMonitorData | null;
  isExpanded?: boolean;
  onToggle?: () => void;
  collapsible?: boolean;
}

interface ConfirmationDialog {
  isOpen: boolean;
  processName: string;
  processPid: number;
  processType: 'zombie' | 'high_thread' | 'leak_candidate';
}

export const ProcessMonitor = ({ data, isExpanded = false, onToggle, collapsible = true }: ProcessMonitorProps) => {
  const [confirmDialog, setConfirmDialog] = useState<ConfirmationDialog>({ 
    isOpen: false, 
    processName: '', 
    processPid: 0, 
    processType: 'zombie' 
  });
  
  const handleKillProcess = async (pid: number, name: string, type: 'zombie' | 'high_thread' | 'leak_candidate') => {
    setConfirmDialog({ 
      isOpen: true, 
      processName: name, 
      processPid: pid, 
      processType: type 
    });
  };
  
  const confirmKillProcess = async () => {
    try {
      console.log(`Killing process ${confirmDialog.processName} (PID: ${confirmDialog.processPid})`);
      
      const response = await fetch(`/api/processes/${confirmDialog.processPid}/kill`, { 
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      });
      
      const result = await response.json();
      
      if (!response.ok) {
        throw new Error(result.error || 'Failed to terminate process');
      }
      
      console.log('Process terminated successfully:', result.message);
      
      // Close dialog
      setConfirmDialog({ isOpen: false, processName: '', processPid: 0, processType: 'zombie' });
      
      // Refresh will happen automatically via periodic data fetch
    } catch (error) {
      console.error('Failed to kill process:', error);
      alert(`Failed to terminate process: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };
  
  const cancelKillProcess = () => {
    setConfirmDialog({ isOpen: false, processName: '', processPid: 0, processType: 'zombie' });
  };
  const expanded = collapsible ? isExpanded : true;

  return (
    <section className="monitoring-panel collapsible card">
      <div 
        className="panel-header clickable" 
        onClick={collapsible ? onToggle : undefined}
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          cursor: 'pointer',
          marginBottom: expanded ? 'var(--spacing-md)' : 0,
          ...(collapsible ? {} : {
            cursor: 'default'
          })
        }}
      >
        <h2 style={{ margin: 0, color: 'var(--color-text-bright)', display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
          <Search size={20} />
          PROCESS MONITOR
        </h2>
        {collapsible && (
          <span className="expand-arrow" style={{ color: 'var(--color-accent)' }}>
            {expanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
          </span>
        )}
      </div>
      
      {expanded && (
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
                      color: (data.process_health.zombie_processes && data.process_health.zombie_processes.length > 0) ? 'var(--color-error)' : 'var(--color-success)'
                    }}>
                      {data.process_health.zombie_processes?.length || 0}
                    </span>
                  </div>
                </div>
                
                {data.process_health.zombie_processes && data.process_health.zombie_processes.length > 0 && (
                  <div className="process-alerts">
                    {data.process_health.zombie_processes.slice(0, 5).map(process => (
                      <div key={process.pid} style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        padding: 'var(--spacing-sm)',
                        background: 'rgba(255, 0, 64, 0.1)',
                        border: '1px solid var(--color-error)',
                        borderRadius: 'var(--border-radius-sm)',
                        marginBottom: 'var(--spacing-xs)',
                        fontSize: 'var(--font-size-sm)'
                      }}>
                        <span>{process.name} (PID: {process.pid})</span>
                        <button
                          onClick={() => handleKillProcess(process.pid, process.name, 'zombie')}
                          title="Kill zombie process"
                          style={{
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--color-error)',
                            cursor: 'pointer',
                            padding: 'var(--spacing-xs)',
                            borderRadius: 'var(--border-radius-sm)',
                            transition: 'background var(--transition-fast)'
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 0, 64, 0.2)'}
                          onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                        >
                          <X size={16} />
                        </button>
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
                  {(data.process_health.high_thread_count || []).slice(0, 5).map(process => (
                    <div key={process.pid} style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(0, 0, 0, 0.3)',
                      border: '1px solid var(--color-accent)',
                      borderRadius: 'var(--border-radius-sm)',
                      marginBottom: 'var(--spacing-xs)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{process.name} ({process.pid})</span>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                        <span style={{ color: 'var(--color-warning)' }}>
                          {process.threads} threads
                        </span>
                        <button
                          onClick={() => handleKillProcess(process.pid, process.name, 'high_thread')}
                          title="Kill high-thread process"
                          style={{
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--color-warning)',
                            cursor: 'pointer',
                            padding: 'var(--spacing-xs)',
                            borderRadius: 'var(--border-radius-sm)',
                            transition: 'background var(--transition-fast)'
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 170, 0, 0.2)'}
                          onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                        >
                          <X size={16} />
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-bright)', marginBottom: 'var(--spacing-md)' }}>
                  Resource Leak Candidates:
                </h3>
                <div className="leak-list">
                  {(data.process_health.leak_candidates || []).slice(0, 5).map(process => (
                    <div key={process.pid} style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      padding: 'var(--spacing-sm)',
                      background: 'rgba(255, 170, 0, 0.1)',
                      border: '1px solid var(--color-warning)',
                      borderRadius: 'var(--border-radius-sm)',
                      marginBottom: 'var(--spacing-xs)',
                      fontSize: 'var(--font-size-sm)'
                    }}>
                      <span>{process.name} ({process.pid})</span>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 'var(--spacing-sm)' }}>
                        <span style={{ color: 'var(--color-warning)' }}>
                          {process.memory_mb.toFixed(0)} MB
                        </span>
                        <button
                          onClick={() => handleKillProcess(process.pid, process.name, 'leak_candidate')}
                          title="Kill process with resource leak"
                          style={{
                            background: 'transparent',
                            border: 'none',
                            color: 'var(--color-warning)',
                            cursor: 'pointer',
                            padding: 'var(--spacing-xs)',
                            borderRadius: 'var(--border-radius-sm)',
                            transition: 'background var(--transition-fast)'
                          }}
                          onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255, 170, 0, 0.2)'}
                          onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
                        >
                          <X size={16} />
                        </button>
                      </div>
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
      
      {/* Confirmation Dialog */}
      {confirmDialog.isOpen && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0, 0, 0, 0.8)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 9999
        }}>
          <div style={{
            background: 'var(--color-surface)',
            border: '2px solid var(--color-accent)',
            borderRadius: 'var(--border-radius-lg)',
            padding: 'var(--spacing-xl)',
            maxWidth: '500px',
            width: '90%',
            boxShadow: '0 0 50px rgba(0, 255, 0, 0.3)'
          }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 'var(--spacing-md)',
              marginBottom: 'var(--spacing-lg)',
              color: 'var(--color-error)'
            }}>
              <AlertTriangle size={32} />
              <h3 style={{
                margin: 0,
                color: 'var(--color-text-bright)',
                fontSize: 'var(--font-size-xl)'
              }}>
                CONFIRM PROCESS TERMINATION
              </h3>
            </div>
            
            <div style={{
              marginBottom: 'var(--spacing-lg)',
              color: 'var(--color-text)',
              fontSize: 'var(--font-size-md)',
              lineHeight: '1.5'
            }}>
              <p style={{ margin: '0 0 var(--spacing-md) 0' }}>
                Are you sure you want to terminate the following process?
              </p>
              
              <div style={{
                background: 'rgba(0, 0, 0, 0.5)',
                border: '1px solid var(--color-accent)',
                borderRadius: 'var(--border-radius-sm)',
                padding: 'var(--spacing-md)',
                fontFamily: 'var(--font-family-mono)',
                fontSize: 'var(--font-size-sm)'
              }}>
                <div style={{ marginBottom: 'var(--spacing-xs)' }}>
                  <strong>Process:</strong> {confirmDialog.processName}
                </div>
                <div style={{ marginBottom: 'var(--spacing-xs)' }}>
                  <strong>PID:</strong> {confirmDialog.processPid}
                </div>
                <div>
                  <strong>Type:</strong> {confirmDialog.processType.replace('_', ' ').toUpperCase()}
                </div>
              </div>
              
              <p style={{
                margin: 'var(--spacing-md) 0 0 0',
                color: 'var(--color-warning)',
                fontSize: 'var(--font-size-sm)'
              }}>
                <strong>Warning:</strong> This action cannot be undone. The process will be forcefully terminated.
              </p>
            </div>
            
            <div style={{
              display: 'flex',
              gap: 'var(--spacing-md)',
              justifyContent: 'flex-end'
            }}>
              <button
                className="btn btn-secondary"
                onClick={cancelKillProcess}
                style={{
                  padding: 'var(--spacing-sm) var(--spacing-lg)',
                  fontSize: 'var(--font-size-sm)'
                }}
              >
                CANCEL
              </button>
              <button
                onClick={confirmKillProcess}
                style={{
                  padding: 'var(--spacing-sm) var(--spacing-lg)',
                  fontSize: 'var(--font-size-sm)',
                  background: 'var(--color-error)',
                  color: 'var(--color-background)',
                  border: '1px solid var(--color-error)',
                  borderRadius: 'var(--border-radius-sm)',
                  cursor: 'pointer',
                  fontWeight: 'bold',
                  textTransform: 'uppercase',
                  transition: 'all var(--transition-fast)'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'transparent';
                  e.currentTarget.style.color = 'var(--color-error)';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'var(--color-error)';
                  e.currentTarget.style.color = 'var(--color-background)';
                }}
              >
                TERMINATE PROCESS
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
};
