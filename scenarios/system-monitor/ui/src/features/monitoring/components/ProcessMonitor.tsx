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
        className={`panel-header clickable process-monitor-header ${expanded ? 'is-expanded' : ''} ${collapsible ? '' : 'is-static'}`}
        onClick={collapsible ? onToggle : undefined}
      >
        <h2 className="icon-text" data-sm-style="sm-style-59e966dafb">
          <Search size={20} />
          PROCESS MONITOR
        </h2>
        {collapsible && (
          <span className="expand-arrow" data-sm-style="sm-style-392c7463c7">
            {expanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
          </span>
        )}
      </div>
      
      {expanded && (
        <div className="panel-content">
          {data ? (
            <div className="monitor-grid" data-sm-style="sm-style-a894d89c52">
              <div className="monitor-section">
                <h3 data-sm-style="sm-style-68f62fb973">
                  Process Health:
                </h3>
                <div className="health-stats">
                  <div className="stat-item">
                    <span className="stat-label">Total Processes:</span>
                    <span className="stat-value" data-sm-style="sm-style-392c7463c7">
                      {data.processHealth?.totalProcesses}
                    </span>
                  </div>
                  <div className="stat-item">
                    <span className="stat-label">Zombie Processes:</span>
                    <span className={`stat-value ${(data.processHealth?.zombieProcesses?.length ?? 0) > 0 ? 'text-error' : 'text-success'}`}>
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
                <h3 data-sm-style="sm-style-68f62fb973">
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
                <h3 data-sm-style="sm-style-68f62fb973">
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
            <div data-sm-style="sm-style-b81eea4ce3">
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
            <p data-sm-style="sm-style-288cd9d43e">
              Are you sure you want to terminate the following process?
            </p>
            <div data-sm-style="sm-style-e90d3b2f2f">
              <div data-sm-style="sm-style-d47aef18a0">
                <strong>Process:</strong> {confirmDialog.processName}
              </div>
              <div data-sm-style="sm-style-d47aef18a0">
                <strong>PID:</strong> {confirmDialog.processPid}
              </div>
              <div>
                <strong>Type:</strong> {confirmDialog.processType.replace('_', ' ').toUpperCase()}
              </div>
            </div>
            <p data-sm-style="sm-style-010a0542bd">
              <strong>Warning:</strong> This action cannot be undone. The process will be forcefully terminated.
            </p>
          </>
        }
      />
    </section>
  );
};
