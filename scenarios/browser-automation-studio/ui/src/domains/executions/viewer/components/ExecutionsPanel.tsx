import ExecutionHistory from '../../history/ExecutionHistory';

interface ExecutionsPanelProps {
  workflowId: string | null;
  onSelectExecution: (execution: { id: string }) => void;
}

export function ExecutionsPanel({ workflowId, onSelectExecution }: ExecutionsPanelProps) {
  if (!workflowId) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-gray-400">
        Workflow identifier unavailable for this execution.
      </div>
    );
  }

  return (
    <div className="h-full overflow-hidden rounded-xl border border-gray-800 bg-flow-node/40">
      <ExecutionHistory
        workflowId={workflowId}
        onSelectExecution={onSelectExecution}
      />
    </div>
  );
}
