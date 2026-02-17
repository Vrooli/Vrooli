import { CheckCircle, XCircle, Clock, Terminal } from 'lucide-react';
import { Modal, ModalHeader } from '../../../shared/components/Modal';
import type { ScriptExecution } from '../../../types';

interface ScriptResultsModalProps {
  isOpen: boolean;
  execution?: ScriptExecution;
  onClose: () => void;
}

export const ScriptResultsModal = ({ isOpen, execution, onClose }: ScriptResultsModalProps) => {
  if (!execution) return null;

  const stdoutContent = execution.stdout ?? execution.output ?? '';
  const stderrContent = execution.stderr ?? execution.error ?? '';

  const getStatusIcon = () => {
    switch (execution.status) {
      case 'completed':
        return execution.exit_code === 0 && !execution.timed_out ?
          <CheckCircle size={20} style={{ color: 'var(--color-success)' }} /> :
          <XCircle size={20} style={{ color: 'var(--color-error)' }} />;
      case 'failed':
        return <XCircle size={20} style={{ color: 'var(--color-error)' }} />;
      case 'running':
        return <Clock size={20} style={{ color: 'var(--color-warning)' }} />;
      default:
        return <Terminal size={20} style={{ color: 'var(--color-info)' }} />;
    }
  };

  const getStatusColor = () => {
    switch (execution.status) {
      case 'completed':
        return execution.exit_code === 0 && !execution.timed_out ? 'var(--color-success)' : 'var(--color-error)';
      case 'failed':
        return 'var(--color-error)';
      case 'running':
        return 'var(--color-warning)';
      default:
        return 'var(--color-info)';
    }
  };

  const formatDuration = () => {
    if (typeof execution.duration_seconds === 'number') {
      const duration = Math.round(execution.duration_seconds);
      if (duration < 1) return '< 1s';
      if (duration < 60) return `${duration}s`;
      if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
      return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
    }

    if (!execution.started_at) return 'Unknown';

    const start = new Date(execution.started_at);
    const end = execution.completed_at ? new Date(execution.completed_at) : new Date();
    const duration = Math.round((end.getTime() - start.getTime()) / 1000);

    if (duration < 1) return '< 1s';
    if (duration < 60) return `${duration}s`;
    if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
    return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="modal-md" ariaLabel="Script execution results">
      <ModalHeader onClose={onClose}>
        <div className="icon-text" style={{ gap: 'var(--spacing-md)' }}>
          {getStatusIcon()}
          <div>
            <h3 style={{
              margin: '0 0 var(--spacing-xs) 0',
              color: 'var(--color-text-bright)',
              fontSize: 'var(--font-size-xl)'
            }}>
              Script Execution Results
            </h3>
            <p style={{
              margin: 0,
              color: 'var(--color-text-dim)',
              fontSize: 'var(--font-size-sm)'
            }}>
              Script: {execution.script_id} | Execution ID: {execution.execution_id}
            </p>
          </div>
        </div>
      </ModalHeader>

      {/* Execution Summary */}
      <div className="execution-summary" style={{
        padding: 'var(--spacing-lg)',
        borderBottom: '1px solid var(--alpha-accent-20)',
        background: 'var(--overlay-medium)'
      }}>
        <div className="detail-grid detail-grid-md" style={{ gap: 'var(--spacing-lg)' }}>
          <div className="summary-stat">
            <span className="summary-stat-label">Status:</span>
            <span className="summary-stat-value" style={{
              color: getStatusColor(),
              textTransform: 'uppercase'
            }}>
              {execution.status}
            </span>
          </div>

          {execution.exit_code !== undefined && (
            <div className="summary-stat">
              <span className="summary-stat-label">Exit Code:</span>
              <span className="summary-stat-value" style={{
                color: execution.exit_code === 0 ? 'var(--color-success)' : 'var(--color-error)',
                fontFamily: 'var(--font-family-mono)'
              }}>
                {execution.exit_code}
              </span>
            </div>
          )}

          <div className="summary-stat">
            <span className="summary-stat-label">Duration:</span>
            <span className="summary-stat-value" style={{
              color: 'var(--color-text-bright)',
              fontFamily: 'var(--font-family-mono)'
            }}>
              {formatDuration()}
            </span>
          </div>

          {execution.timed_out !== undefined && (
            <div className="summary-stat">
              <span className="summary-stat-label">Timed Out:</span>
              <span className="summary-stat-value" style={{
                color: execution.timed_out ? 'var(--color-error)' : 'var(--color-text-bright)'
              }}>
                {execution.timed_out ? 'Yes' : 'No'}
              </span>
            </div>
          )}

          <div className="summary-stat">
            <span className="summary-stat-label">Started:</span>
            <span style={{
              color: 'var(--color-text)',
              fontSize: 'var(--font-size-sm)'
            }}>
              {execution.started_at ? new Date(execution.started_at).toLocaleString() : 'Unknown'}
            </span>
          </div>
        </div>
      </div>

      {/* Modal Body - Output */}
      <div className="modal-body" style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        padding: 0
      }}>

        {/* Output Section */}
        {stdoutContent && (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
            <div className="output-header icon-text" style={{
              padding: 'var(--spacing-sm) var(--spacing-md)',
              background: 'var(--alpha-accent-05)',
              borderBottom: '1px solid var(--alpha-accent-20)',
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-text-bright)'
            }}>
              <Terminal size={16} />
              <span>Script Output</span>
              <span style={{
                marginLeft: 'auto',
                color: 'var(--color-text-dim)',
                fontSize: 'var(--font-size-xs)'
              }}>
                {stdoutContent.split('\n').length} lines
              </span>
            </div>

            <div className="output-content" style={{
              flex: 1,
              padding: 'var(--spacing-md)',
              background: 'var(--overlay-darker)',
              overflowY: 'auto',
              maxHeight: '60vh',
              fontFamily: 'var(--font-family-mono)',
              fontSize: 'var(--font-size-sm)',
              lineHeight: '1.4',
              whiteSpace: 'pre-wrap',
              color: 'var(--color-text)'
            }}>
              {stdoutContent}
            </div>
          </div>
        )}

        {/* Error Section */}
        {stderrContent && (
          <div style={{
            borderTop: stdoutContent ? `1px solid var(--color-error-border)` : 'none',
            background: 'var(--color-error-bg)'
          }}>
            <div className="error-header icon-text" style={{
              padding: 'var(--spacing-sm) var(--spacing-md)',
              borderBottom: '1px solid var(--color-error-border)',
              fontSize: 'var(--font-size-sm)',
              color: 'var(--color-error)'
            }}>
              <XCircle size={16} />
              <span>Error Output</span>
            </div>

            <div className="error-content" style={{
              padding: 'var(--spacing-md)',
              background: 'var(--color-error-bg)',
              fontFamily: 'var(--font-family-mono)',
              fontSize: 'var(--font-size-sm)',
              lineHeight: '1.4',
              whiteSpace: 'pre-wrap',
              color: 'var(--color-error)',
              maxHeight: '200px',
              overflow: 'auto'
            }}>
              {stderrContent}
            </div>
          </div>
        )}

        {/* No Output Message */}
        {!stdoutContent && !stderrContent && execution.status === 'running' && (
          <div style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--color-text-dim)',
            fontSize: 'var(--font-size-lg)'
          }}>
            <div style={{ textAlign: 'center' }}>
              <Clock size={48} style={{ marginBottom: 'var(--spacing-md)' }} />
              <div>Script is still running...</div>
              <div style={{ fontSize: 'var(--font-size-sm)', marginTop: 'var(--spacing-sm)' }}>
                Started {formatDuration()} ago
              </div>
            </div>
          </div>
        )}

        {!stdoutContent && !stderrContent && execution.status !== 'running' && (
          <div style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--color-text-dim)',
            fontSize: 'var(--font-size-lg)'
          }}>
            No output available
          </div>
        )}
      </div>
    </Modal>
  );
};
