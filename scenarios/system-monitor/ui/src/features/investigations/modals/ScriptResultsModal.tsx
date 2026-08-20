import { CheckCircle, XCircle, Clock, Terminal } from 'lucide-react';
import { Modal, ModalHeader } from '../../../shared/components/Modal';
import type { ScriptExecution } from '../../../types';
import { ScriptExecutionStatus } from '../../../types';
import { timestampDate } from '@bufbuild/protobuf/wkt';
import { formatDurationSeconds } from '../../../shared/utils/formatters';

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
      case ScriptExecutionStatus.COMPLETED:
        return execution.exitCode === 0 && !execution.timedOut ?
          <CheckCircle size={20} data-sm-style="sm-style-eab9fc4afc" /> :
          <XCircle size={20} data-sm-style="sm-style-6d06f948c5" />;
      case ScriptExecutionStatus.FAILED:
        return <XCircle size={20} data-sm-style="sm-style-6d06f948c5" />;
      case ScriptExecutionStatus.RUNNING:
        return <Clock size={20} data-sm-style="sm-style-38c5f4e767" />;
      default:
        return <Terminal size={20} data-sm-style="sm-style-60c4dfc517" />;
    }
  };

  const getStatusColor = () => {
    switch (execution.status) {
      case ScriptExecutionStatus.COMPLETED:
        return execution.exitCode === 0 && !execution.timedOut ? 'var(--color-success)' : 'var(--color-error)';
      case ScriptExecutionStatus.FAILED:
        return 'var(--color-error)';
      case ScriptExecutionStatus.RUNNING:
        return 'var(--color-warning)';
      default:
        return 'var(--color-info)';
    }
  };

  const formatDuration = () => {
    if (typeof execution.durationSeconds === 'number') {
      return formatDurationSeconds(execution.durationSeconds);
    }

    if (!execution.startedAt) return 'Unknown';

    const start = timestampDate(execution.startedAt);
    const end = execution.completedAt ? timestampDate(execution.completedAt) : new Date();
    const duration = Math.round((end.getTime() - start.getTime()) / 1000);
    return formatDurationSeconds(duration);
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="modal-md" ariaLabel="Script execution results">
      <ModalHeader onClose={onClose}>
        <div className="icon-text" data-sm-style="sm-style-8322eb66c1">
          {getStatusIcon()}
          <div>
            <h3 data-sm-style="sm-style-bd3930e88e">
              Script Execution Results
            </h3>
            <p data-sm-style="sm-style-5c239e09c9">
              Script: {execution.scriptId} | Execution ID: {execution.executionId}
            </p>
            {execution.executionMode && (
              <p className="text-dim-xs">Execution path: {execution.executionMode}{execution.skipReason ? ` — ${execution.skipReason}` : ''}</p>
            )}
          </div>
        </div>
      </ModalHeader>

      {/* Execution Summary */}
      <div className="execution-summary" data-sm-style="sm-style-e35c711cad">
        <div className="detail-grid detail-grid-md" data-sm-style="sm-style-f383142193">
          <div className="summary-stat">
            <span className="summary-stat-label">Status:</span>
            <span className="summary-stat-value" style={{
              color: getStatusColor(),
              textTransform: 'uppercase'
            }}>
              {execution.status}
            </span>
          </div>

          {execution.exitCode !== undefined && (
            <div className="summary-stat">
              <span className="summary-stat-label">Exit Code:</span>
              <span className="summary-stat-value" style={{
                color: execution.exitCode === 0 ? 'var(--color-success)' : 'var(--color-error)',
                fontFamily: 'var(--font-mono)'
              }}>
                {execution.exitCode}
              </span>
            </div>
          )}

          <div className="summary-stat">
            <span className="summary-stat-label">Duration:</span>
            <span className="summary-stat-value" data-sm-style="sm-style-2cc55f187c">
              {formatDuration()}
            </span>
          </div>

          {execution.timedOut !== undefined && (
            <div className="summary-stat">
              <span className="summary-stat-label">Timed Out:</span>
              <span className="summary-stat-value" style={{
                color: execution.timedOut ? 'var(--color-error)' : 'var(--color-text-heading)'
              }}>
                {execution.timedOut ? 'Yes' : 'No'}
              </span>
            </div>
          )}

          <div className="summary-stat">
            <span className="summary-stat-label">Started:</span>
            <span data-sm-style="sm-style-d7b793f3b3">
              {execution.startedAt ? timestampDate(execution.startedAt).toLocaleString() : 'Unknown'}
            </span>
          </div>
        </div>
      </div>

      {/* Modal Body - Output */}
      <div className="modal-body" data-sm-style="sm-style-500d25478f">

        {/* Output Section */}
        {stdoutContent && (
          <div data-sm-style="sm-style-e19d496461">
            <div className="output-header icon-text" data-sm-style="sm-style-20b2bbf6af">
              <Terminal size={16} />
              <span>Script Output</span>
              <span data-sm-style="sm-style-2dd53c91dc">
                {stdoutContent.split('\n').length} lines
              </span>
            </div>

            <div className="output-content" data-sm-style="sm-style-1b2da6b47f">
              {stdoutContent}
            </div>
          </div>
        )}

        {/* Error Section */}
        {stderrContent && (
          <div style={{
            borderTop: stdoutContent ? `1px solid var(--color-error)` : 'none',
            background: 'var(--color-error-muted)'
          }}>
            <div className="error-header icon-text" data-sm-style="sm-style-46f4f9ea6e">
              <XCircle size={16} />
              <span>Error Output</span>
            </div>

            <div className="error-content" data-sm-style="sm-style-3b31792f5d">
              {stderrContent}
            </div>
          </div>
        )}

        {/* No Output Message */}
        {!stdoutContent && !stderrContent && execution.status === ScriptExecutionStatus.RUNNING && (
          <div data-sm-style="sm-style-be12747b9c">
            <div data-sm-style="sm-style-980597f335">
              <Clock size={48} data-sm-style="sm-style-91394348ef" />
              <div>Script is still running...</div>
              <div data-sm-style="sm-style-d523b883ef">
                Started {formatDuration()} ago
              </div>
            </div>
          </div>
        )}

        {!stdoutContent && !stderrContent && execution.status !== ScriptExecutionStatus.RUNNING && (
          <div data-sm-style="sm-style-be12747b9c">
            No output available
          </div>
        )}
      </div>
    </Modal>
  );
};
