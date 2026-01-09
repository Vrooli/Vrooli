/**
 * HistoryTab Component
 *
 * Displays execution history for the current workflow, allowing users
 * to switch between different execution runs. Only visible in execution mode.
 */

import ExecutionHistory from '@/domains/executions/history/ExecutionHistory';

export interface HistoryTabProps {
  /** Workflow ID to fetch execution history for */
  workflowId: string | null;
  /** Currently selected execution ID (for highlighting) */
  currentExecutionId?: string;
  /** Callback when user selects a different execution */
  onSelectExecution: (executionId: string) => void;
  /** Optional className for styling */
  className?: string;
}

export function HistoryTab({
  workflowId,
  currentExecutionId,
  onSelectExecution,
  className = '',
}: HistoryTabProps) {
  if (!workflowId) {
    return (
      <div className={`flex items-center justify-center h-full text-gray-400 text-sm ${className}`}>
        <p>Select a workflow to view execution history</p>
      </div>
    );
  }

  return (
    <div className={`h-full overflow-hidden ${className}`}>
      <ExecutionHistory
        workflowId={workflowId}
        onSelectExecution={(execution) => {
          // Only trigger if selecting a different execution
          if (execution.id !== currentExecutionId) {
            onSelectExecution(execution.id);
          }
        }}
      />
    </div>
  );
}
