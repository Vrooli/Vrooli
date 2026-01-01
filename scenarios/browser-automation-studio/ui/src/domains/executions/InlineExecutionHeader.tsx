/**
 * InlineExecutionHeader Component
 *
 * Simplified header for the inline execution viewer used in Project view.
 * Shows action buttons: Re-run, Export, Close.
 */

import { RotateCw, Download, X, Loader2 } from 'lucide-react';

export interface InlineExecutionHeaderProps {
  /** Workflow name to display */
  workflowName?: string;
  /** Execution status for display */
  status?: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  /** Callback when re-run is clicked */
  onRerun?: () => void;
  /** Callback when export is clicked */
  onExport?: () => void;
  /** Callback when close is clicked */
  onClose: () => void;
  /** Whether re-run is available */
  canRerun?: boolean;
  /** Whether export is available */
  canExport?: boolean;
  /** Whether an export is in progress */
  isExporting?: boolean;
  /** Whether a re-run is in progress */
  isRerunning?: boolean;
}

function getStatusBadge(status: InlineExecutionHeaderProps['status']) {
  if (!status) return null;

  const baseClasses = 'px-2 py-0.5 text-xs font-medium rounded-full';

  switch (status) {
    case 'running':
      return (
        <span className={`${baseClasses} bg-blue-500/20 text-blue-400`}>
          Running
        </span>
      );
    case 'completed':
      return (
        <span className={`${baseClasses} bg-green-500/20 text-green-400`}>
          Completed
        </span>
      );
    case 'failed':
      return (
        <span className={`${baseClasses} bg-red-500/20 text-red-400`}>
          Failed
        </span>
      );
    case 'cancelled':
      return (
        <span className={`${baseClasses} bg-yellow-500/20 text-yellow-400`}>
          Cancelled
        </span>
      );
    case 'pending':
      return (
        <span className={`${baseClasses} bg-gray-500/20 text-gray-400`}>
          Pending
        </span>
      );
    default:
      return null;
  }
}

export function InlineExecutionHeader({
  workflowName,
  status,
  onRerun,
  onExport,
  onClose,
  canRerun = true,
  canExport = true,
  isExporting = false,
  isRerunning = false,
}: InlineExecutionHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900">
      {/* Left: Title and status */}
      <div className="flex items-center gap-3 min-w-0">
        <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
          {workflowName || 'Execution'}
        </h3>
        {getStatusBadge(status)}
      </div>

      {/* Right: Action buttons */}
      <div className="flex items-center gap-2">
        {/* Re-run button */}
        {onRerun && canRerun && (
          <button
            type="button"
            onClick={onRerun}
            disabled={isRerunning || status === 'running'}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Run workflow again"
          >
            {isRerunning ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <RotateCw size={14} />
            )}
            <span className="hidden sm:inline">Re-run</span>
          </button>
        )}

        {/* Export button */}
        {onExport && canExport && (
          <button
            type="button"
            onClick={onExport}
            disabled={isExporting}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Export execution replay"
          >
            {isExporting ? (
              <Loader2 size={14} className="animate-spin" />
            ) : (
              <Download size={14} />
            )}
            <span className="hidden sm:inline">Export</span>
          </button>
        )}

        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          className="inline-flex items-center justify-center w-8 h-8 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
          title="Close viewer"
        >
          <X size={18} />
        </button>
      </div>
    </div>
  );
}
