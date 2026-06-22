import { ChevronDown, ChevronUp, Search } from 'lucide-react';
import { useRef, useState } from 'react';
import type { ProcessMonitorData, ProcessInfo } from '../../../types';
import { apiFetch } from '../../../shared/api/apiFetch';
import { ConfirmDialog } from '../../../shared/components/ConfirmDialog';
import { ProcessAlertItem } from './ProcessAlertItem';

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
  const confirmDialogRef = useRef(confirmDialog);
  confirmDialogRef.current = confirmDialog;

  const handleKillProcess = (_pid: number, name: string, type: 'zombie' | 'high_thread' | 'leak_candidate') => {
    setConfirmDialog({
      isOpen: true,
      processName: name,
      processPid: _pid,
      processType: type
    });
  };

  const confirmKillProcess = async () => {
    const { processName, processPid } = confirmDialogRef.current;
    try {
      console.log(`Killing process ${processName} (PID: ${processPid})`);

      await apiFetch<{ message?: string }>(`/processes/${processPid}/kill`, {
        method: 'POST',
      });

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
        <h2 className="icon-text" style={{ margin: 0, color: 'var(--color-text-heading)' }}>
          <Search size={20} />
          PROCESS MONITOR
        </h2>
        {collapsible && (
          <span className="expand-arrow" style={{ color: 'var(--color-primary)' }}>
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
                <h3 style={{ color: 'var(--color-text-heading)', marginBottom: 'var(--spacing-md)' }}>
                  Process Health:
                </h3>
                <div className="health-stats">
                  <div className="stat-item">
                    <span className="stat-label">Total Processes:</span>
                    <span className="stat-value" style={{ color: 'var(--color-primary)' }}>
                      {data.processHealth?.totalProcesses}
                    </span>
                  </div>
                  <div className="stat-item">
                    <span className="stat-label">Zombie Processes:</span>
                    <span className="stat-value" style={{ 
                      color: (data.processHealth?.zombieProcesses && data.processHealth?.zombieProcesses.length > 0) ? 'var(--color-error)' : 'var(--color-success)'
                    }}>
                      {data.processHealth?.zombieProcesses?.length ?? 0}
                    </span>
                  </div>
                </div>
                
                {data.processHealth?.zombieProcesses && data.processHealth?.zombieProcesses.length > 0 && (
                  <div className="process-alerts">
                    {data.processHealth?.zombieProcesses.slice(0, 5).map((process: ProcessInfo) => (
                      <ProcessAlertItem
                        key={process.pid}
                        pid={process.pid}
                        name={process.name}
                        variant="zombie"
                        onKill={handleKillProcess}
                      />
                    ))}
                  </div>
                )}
              </div>
              
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-heading)', marginBottom: 'var(--spacing-md)' }}>
                  High Thread Count:
                </h3>
                <div className="thread-list">
                  {(data.processHealth?.highThreadCount ?? []).slice(0, 5).map((process: ProcessInfo) => (
                    <ProcessAlertItem
                      key={process.pid}
                      pid={process.pid}
                      name={process.name}
                      variant="high_thread"
                      detail={`${process.threads} threads`}
                      onKill={handleKillProcess}
                    />
                  ))}
                </div>
              </div>
              
              <div className="monitor-section">
                <h3 style={{ color: 'var(--color-text-heading)', marginBottom: 'var(--spacing-md)' }}>
                  Resource Leak Candidates:
                </h3>
                <div className="leak-list">
                  {(data.processHealth?.leakCandidates ?? []).slice(0, 5).map((process: ProcessInfo) => (
                    <ProcessAlertItem
                      key={process.pid}
                      pid={process.pid}
                      name={process.name}
                      variant="leak_candidate"
                      detail={`${process.memoryMb.toFixed(0)} MB`}
                      onKill={handleKillProcess}
                    />
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div style={{ 
              textAlign: 'center', 
              color: 'var(--color-text-secondary)', 
              padding: 'var(--spacing-xl)' 
            }}>
              SCANNING SYSTEM...
            </div>
          )}
        </div>
      )}
      
      <ConfirmDialog
        isOpen={confirmDialog.isOpen}
        title="CONFIRM PROCESS TERMINATION"
        variant="danger"
        confirmLabel="TERMINATE PROCESS"
        onConfirm={() => { void confirmKillProcess(); }}
        onCancel={cancelKillProcess}
        message={
          <>
            <p style={{ margin: '0 0 var(--spacing-md) 0' }}>
              Are you sure you want to terminate the following process?
            </p>
            <div style={{
              background: 'var(--overlay-medium)',
              border: '1px solid var(--color-primary)',
              borderRadius: 'var(--radius-sm)',
              padding: 'var(--spacing-md)',
              fontFamily: 'var(--font-mono)',
              fontSize: 'var(--text-sm)'
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
              fontSize: 'var(--text-sm)'
            }}>
              <strong>Warning:</strong> This action cannot be undone. The process will be forcefully terminated.
            </p>
          </>
        }
      />
    </section>
  );
};
