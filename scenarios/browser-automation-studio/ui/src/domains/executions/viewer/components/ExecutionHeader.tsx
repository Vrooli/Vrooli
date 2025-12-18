import {
  Activity,
  Pause,
  RotateCw,
  X,
  Square,
  Clock,
  CheckCircle,
  XCircle,
  Loader,
  AlertTriangle,
  Download,
  Pencil,
} from 'lucide-react';
import type { Execution } from '../../store';
import { selectors } from '@constants/selectors';
import type { HeartbeatDescriptor } from '../useExecutionHeartbeat';

interface ExecutionHeaderProps {
  execution: Execution;
  statusMessage: string;
  heartbeatDescriptor: HeartbeatDescriptor | null;
  inStepLabel: string | null;
  isRunning: boolean;
  canRestart: boolean;
  isStopping: boolean;
  isRestarting: boolean;
  replayFramesCount: number;
  onStop: () => void;
  onRestart: () => void;
  onExport: () => void;
  onClose?: () => void;
}

function getStatusIcon(status: Execution['status']) {
  const statusTestId = `execution-status-${status}`;
  const testIds = `${selectors.executions.viewer.status} ${statusTestId}`;
  switch (status) {
    case 'running':
      return <Loader size={16} className="animate-spin text-blue-400" data-testid={testIds} />;
    case 'completed':
      return <CheckCircle size={16} className="text-green-400" data-testid={testIds} />;
    case 'failed':
      return <XCircle size={16} className="text-red-400" data-testid={testIds} />;
    case 'cancelled':
      return <AlertTriangle size={16} className="text-yellow-400" data-testid={testIds} />;
    default:
      return <Clock size={16} className="text-gray-400" data-testid={testIds} />;
  }
}

export function ExecutionHeader({
  execution,
  statusMessage,
  heartbeatDescriptor,
  inStepLabel,
  isRunning,
  canRestart,
  isStopping,
  isRestarting,
  replayFramesCount,
  onStop,
  onRestart,
  onExport,
  onClose,
}: ExecutionHeaderProps) {
  return (
    <div className="flex items-center justify-between p-3 border-b border-gray-800">
      <div className="flex items-center gap-3">
        {getStatusIcon(execution.status)}
        <div>
          <div className="text-sm font-medium text-surface">
            Execution #{execution.id.slice(0, 8)}
          </div>
          <div
            className="text-xs text-gray-500"
            data-testid={selectors.executions.viewer.status}
          >
            {statusMessage}
          </div>
          {heartbeatDescriptor && (
            <div
              className="mt-1 flex items-center gap-2 text-[11px]"
              data-testid={selectors.heartbeat.indicator}
            >
              {heartbeatDescriptor.tone === 'stalled' ? (
                <AlertTriangle
                  size={12}
                  className={heartbeatDescriptor.iconClass}
                  data-testid={
                    heartbeatDescriptor.tone === 'stalled'
                      ? selectors.heartbeat.lagWarning
                      : undefined
                  }
                />
              ) : (
                <Activity size={12} className={heartbeatDescriptor.iconClass} />
              )}
              <span
                className={heartbeatDescriptor.textClass}
                data-testid={selectors.heartbeat.status}
              >
                {heartbeatDescriptor.label}
              </span>
              {inStepLabel && execution.lastHeartbeat && (
                <span className={`${heartbeatDescriptor.textClass} opacity-80`}>
                  • {inStepLabel} in step
                </span>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          className="toolbar-button p-1.5 text-gray-500 opacity-50 cursor-not-allowed"
          title="Pause (coming soon)"
          disabled
          aria-disabled="true"
        >
          <Pause size={14} />
        </button>
        <button
          className="toolbar-button p-1.5"
          title={canRestart ? 'Re-run workflow' : 'Stop execution before re-running'}
          onClick={onRestart}
          disabled={!canRestart || isRestarting || isStopping}
          data-testid={selectors.executions.viewer.rerunButton}
        >
          {isRestarting ? <Loader size={14} className="animate-spin" /> : <RotateCw size={14} />}
        </button>
        <button
          className="toolbar-button p-1.5 disabled:text-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
          title={replayFramesCount === 0 ? 'Replay not ready to export' : 'Export replay'}
          onClick={onExport}
          disabled={replayFramesCount === 0}
          data-testid={selectors.executions.actions.exportReplayButton}
        >
          <Download size={14} />
        </button>
        <button
          className="toolbar-button p-1.5 text-red-400 disabled:text-red-400/50 disabled:cursor-not-allowed"
          title={isRunning ? 'Stop execution' : 'Execution not running'}
          onClick={onStop}
          disabled={!isRunning || isStopping}
          data-testid={selectors.executions.viewer.stopButton}
        >
          {isStopping ? <Loader size={14} className="animate-spin" /> : <Square size={14} />}
        </button>
        {onClose && (
          <>
            <button
              className="toolbar-button p-1.5 ml-2 border-l border-gray-700 pl-3 text-blue-400 hover:text-blue-300"
              title="Edit workflow"
              onClick={onClose}
              data-testid={selectors.executions.viewer.editWorkflowButton}
            >
              <Pencil size={14} />
            </button>
            <button
              className="toolbar-button p-1.5"
              title="Close"
              onClick={onClose}
            >
              <X size={14} />
            </button>
          </>
        )}
      </div>
    </div>
  );
}
