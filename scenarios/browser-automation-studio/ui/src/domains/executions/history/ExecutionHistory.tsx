import { useState, useEffect, useMemo, useCallback } from "react";
import { Clock, Loader, AlertCircle } from "lucide-react";
import { useExecutionStore } from "../store";
import { logger } from "@utils/logger";
import { selectors } from "@constants/selectors";
import { ExecutionCard, type ExecutionCardData, type ExecutionStatus } from "./ExecutionCard";
import { ExecutionFilters, type StatusFilter } from "./ExecutionFilters";

interface ExecutionHistoryProps {
  workflowId?: string;
  /** Filter executions by project ID */
  projectId?: string;
  onSelectExecution?: (execution: ExecutionSummary) => void;
  /** The currently selected execution ID (for highlighting) */
  selectedExecutionId?: string;
}

interface ExecutionSummary {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  started_at: string;
  completed_at?: string;
  progress: number;
  current_step?: string;
  error?: string;
}

/** Convert ExecutionSummary to ExecutionCardData */
const toCardData = (execution: ExecutionSummary): ExecutionCardData => ({
  id: execution.id,
  workflowId: execution.workflow_id,
  workflowName: execution.workflow_name ?? 'Unknown Workflow',
  status: execution.status as ExecutionStatus,
  startedAt: new Date(execution.started_at),
  completedAt: execution.completed_at ? new Date(execution.completed_at) : undefined,
  error: execution.error,
  progress: execution.progress,
  currentStep: execution.current_step,
});

function ExecutionHistory({
  workflowId,
  projectId,
  onSelectExecution,
  selectedExecutionId,
}: ExecutionHistoryProps) {
  // Get the current execution from the store to determine which card is selected
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const effectiveSelectedId = selectedExecutionId ?? currentExecution?.id;
  const [executions, setExecutions] = useState<ExecutionSummary[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [isRefreshing, setIsRefreshing] = useState(false);

  const loadExecutions = useExecutionStore((state) => state.loadExecutions);
  const storeExecutions = useExecutionStore((state) => state.executions);

  const fetchExecutions = useCallback(
    async (options?: { skipSpinner?: boolean }) => {
      if (!options?.skipSpinner) {
        setIsLoading(true);
      }
      setError(null);
      try {
        // Use the store's loadExecutions to fetch and populate the executions array
        await loadExecutions(workflowId, projectId);

        // Get the executions from the store
        const storeExecutions = useExecutionStore.getState().executions;
        setExecutions(storeExecutions as unknown as ExecutionSummary[]);
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load executions";
        setError(message);
        logger.error(
          "Failed to load execution history",
          {
            component: "ExecutionHistory",
            action: "fetchExecutions",
            workflowId,
            projectId,
          },
          err,
        );
      } finally {
        setIsLoading(false);
      }
    },
    [loadExecutions, workflowId, projectId],
  );

  const handleRefresh = async () => {
    setIsRefreshing(true);
    await fetchExecutions();
    setIsRefreshing(false);
  };

  useEffect(() => {
    const existing = useExecutionStore.getState()
      .executions as unknown as ExecutionSummary[];
    if (existing.length > 0) {
      setExecutions(existing);
    }
    fetchExecutions({ skipSpinner: existing.length > 0 });
  }, [fetchExecutions]);

  useEffect(() => {
    setExecutions(storeExecutions as unknown as ExecutionSummary[]);
  }, [storeExecutions]);

  const filteredExecutions = useMemo(() => {
    if (statusFilter === "all") {
      return executions;
    }
    return executions.filter((exec) => exec.status === statusFilter);
  }, [executions, statusFilter]);

  const handleSelectExecution = useCallback(
    (executionId: string, _workflowId: string) => {
      const execution = executions.find((e) => e.id === executionId);
      if (execution && onSelectExecution) {
        onSelectExecution(execution);
      }
    },
    [executions, onSelectExecution],
  );

  const statusCounts = useMemo(() => {
    return {
      all: executions.length,
      completed: executions.filter((e) => e.status === "completed").length,
      failed: executions.filter((e) => e.status === "failed").length,
      running: executions.filter((e) => e.status === "running").length,
      cancelled: executions.filter((e) => e.status === "cancelled").length,
    };
  }, [executions]);

  if (isLoading && executions.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader
            size={32}
            className="animate-spin text-flow-accent mx-auto mb-4"
          />
          <div className="text-gray-400">Loading execution history...</div>
        </div>
      </div>
    );
  }

  if (error && executions.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertCircle size={48} className="text-red-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-surface mb-2">
            Failed to Load Executions
          </h3>
          <p className="text-gray-400 mb-4">{error}</p>
          <button
            onClick={() => fetchExecutions()}
            className="px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      className="absolute inset-0 flex flex-col"
      data-testid={selectors.executions.list.root}
    >
      {/* Header with filters */}
      <div className="flex-shrink-0 p-4 border-b border-gray-800">
        <ExecutionFilters
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          statusCounts={statusCounts}
          onRefresh={handleRefresh}
          isRefreshing={isRefreshing}
          filters={['all', 'running', 'completed', 'failed', 'cancelled']}
          testIdPrefix="execution-filter"
        />
      </div>

      {/* Execution list */}
      <div
        className="flex-1 min-h-0 overflow-y-auto"
        data-testid={selectors.executions.list.list}
      >
        {filteredExecutions.length === 0 ? (
          <div className="flex items-center justify-center h-full p-8">
            <div className="text-center">
              <Clock size={48} className="text-gray-600 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-surface mb-2">
                {statusFilter === "all"
                  ? "No Executions Yet"
                  : `No ${statusFilter} executions`}
              </h3>
              <p className="text-gray-400">
                {statusFilter === "all"
                  ? "Execute workflows to see their history here"
                  : `Change the filter to see other executions`}
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-2 p-4">
            {filteredExecutions.map((execution) => (
              <ExecutionCard
                key={execution.id}
                execution={toCardData(execution)}
                isRunning={execution.status === 'running' || execution.status === 'pending'}
                isSelected={effectiveSelectedId === execution.id}
                onClick={onSelectExecution ? handleSelectExecution : undefined}
                showProjectName={false}
                showProgressBar
                testId={selectors.executions.list.item}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default ExecutionHistory;
