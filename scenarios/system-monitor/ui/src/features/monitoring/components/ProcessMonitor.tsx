import { ChevronDown, ChevronUp, Search } from 'lucide-react';
import type { ProcessMonitorData, ProcessInfo } from '../../../types';
import { ProcessAlertItem } from './ProcessAlertItem';

interface ProcessMonitorProps {
  data: ProcessMonitorData | null;
  isExpanded?: boolean;
  onToggle?: () => void;
  collapsible?: boolean;
}

export const ProcessMonitor = ({ data, isExpanded = false, onToggle, collapsible = true }: ProcessMonitorProps) => {
  const expanded = collapsible ? isExpanded : true;
  const zombieMetric = data?.processHealth?.zombieCount;
  const zombieState = zombieMetric?.state;
  const zombieCount = zombieState?.case === 'measured' ? Math.max(0, Math.round(zombieState.value)) : undefined;
  const zombieTone = zombieCount === undefined ? 'text-muted' : zombieCount > 0 ? 'text-error' : 'text-success';
  const zombieStateLabel = zombieState?.case === 'failedError'
    ? `Count unavailable: ${zombieState.value}`
    : zombieState?.case === 'unsupportedReason'
      ? `Unavailable on this host: ${zombieState.value}`
      : zombieState?.case === 'notYetSampledReason'
        ? 'Count has not been sampled'
        : zombieState?.case === 'staleReason'
          ? `Last count is stale: ${zombieState.value}`
          : zombieCount !== undefined
            ? zombieCount > 0 && !data?.processHealth?.zombieProcesses?.length
              ? 'Zombie details were not returned for this count.'
              : zombieCount > 0
                ? `Showing up to ${data?.processHealth?.zombieProcesses.length} process names.`
                : 'No zombie processes in the latest scan.'
            : 'The latest collection did not report a zombie count.';

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
            <>
              <p className="process-termination-notice" role="note">
                Process termination is unavailable because this service does not provide a supported termination endpoint.
              </p>
              <div className="monitor-grid" data-sm-style="sm-style-a894d89c52">
              <div className="monitor-section">
                <h3 data-sm-style="sm-style-68f62fb973">
                  Process Health:
                </h3>
                <div className="health-stats">
                  <div className="stat-item">
                    <span className="stat-label">Total Processes:</span>
                    <span className="stat-value" data-sm-style="sm-style-392c7463c7">
                      {data.processHealth ? data.processHealth.totalProcesses.toLocaleString() : '—'}
                    </span>
                  </div>
                  <div className="stat-item process-health-stat">
                    <span className="stat-label">Zombie Processes:</span>
                    <span className={`stat-value ${zombieTone}`} aria-label="Zombie process count">
                      {zombieCount === undefined ? '—' : zombieCount.toLocaleString()}
                    </span>
                    <span className="stat-help" role={zombieCount === undefined ? 'status' : undefined}>{zombieStateLabel}</span>
                  </div>
                </div>

                {!data.processHealth && <p className="text-muted" role="status">Process health is unavailable; the latest collection did not return a health summary.</p>}
                
                {data.processHealth?.zombieProcesses && data.processHealth.zombieProcesses.length > 0 && (
                  <div className="process-alerts">
                    {data.processHealth?.zombieProcesses.slice(0, 5).map((process: ProcessInfo) => (
                      <ProcessAlertItem
                        key={process.pid}
                        pid={process.pid}
                        name={process.name}
                        variant="zombie"
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
                    />
                  ))}
                </div>
              </div>
              </div>
            </>
          ) : (
            <div data-sm-style="sm-style-b81eea4ce3">
              SCANNING SYSTEM...
            </div>
          )}
        </div>
      )}
      
    </section>
  );
};
