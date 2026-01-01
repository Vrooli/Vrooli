/**
 * ExecutionCompletionActions - Action buttons for completed/failed/cancelled executions
 *
 * Provides Export, Re-run, and Edit Workflow actions in a consistent layout
 * that can be used across all terminal execution states.
 */

import { Download, RotateCw, Pencil, Loader2 } from 'lucide-react';

export interface ExecutionCompletionActionsProps {
  /** Callback when Export button is clicked */
  onExport?: () => void;
  /** Callback when Re-run button is clicked */
  onRerun?: () => void;
  /** Callback when Edit Workflow button is clicked */
  onEditWorkflow?: () => void;
  /** Whether an export is currently in progress */
  isExporting?: boolean;
  /** Whether export is available (timeline has frames) */
  canExport?: boolean;
  /** Whether re-run is available (workflow exists) */
  canRerun?: boolean;
  /** Whether edit workflow is available (workflow + project exist) */
  canEditWorkflow?: boolean;
}

export function ExecutionCompletionActions({
  onExport,
  onRerun,
  onEditWorkflow,
  isExporting = false,
  canExport = true,
  canRerun = true,
  canEditWorkflow = true,
}: ExecutionCompletionActionsProps) {
  // Don't render if no actions are available
  const hasAnyAction = (onExport && canExport) || (onRerun && canRerun) || (onEditWorkflow && canEditWorkflow);
  if (!hasAnyAction) return null;

  return (
    <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
      {/* Export button */}
      {onExport && canExport && (
        <button
          onClick={onExport}
          disabled={isExporting}
          className="flex items-center gap-2 px-5 py-2.5 bg-flow-accent text-white rounded-lg hover:bg-flow-accent/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          title={isExporting ? 'Export in progress...' : 'Export as video, GIF, or JSON'}
        >
          {isExporting ? (
            <Loader2 size={18} className="animate-spin" />
          ) : (
            <Download size={18} />
          )}
          {isExporting ? 'Exporting...' : 'Export'}
        </button>
      )}

      {/* Re-run button */}
      {onRerun && canRerun && (
        <button
          onClick={onRerun}
          className="flex items-center gap-2 px-5 py-2.5 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          title="Run the workflow again"
        >
          <RotateCw size={18} />
          Run Again
        </button>
      )}

      {/* Edit Workflow button */}
      {onEditWorkflow && canEditWorkflow && (
        <button
          onClick={onEditWorkflow}
          className="flex items-center gap-2 px-5 py-2.5 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          title="Open workflow in editor"
        >
          <Pencil size={18} />
          Edit Workflow
        </button>
      )}
    </div>
  );
}
