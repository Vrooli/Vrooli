import { useCallback } from "react";
import { ExecutionHistory, ExecutionViewer } from "@/domains/executions";
import { useExecutionStore } from "@/domains/executions";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import { useProjectDetailStore } from "./hooks/useProjectDetailStore";

/**
 * Panel component for execution history and viewer in ProjectDetail
 */
export function ExecutionPanel() {
  // Execution store state
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const isExecutionViewerOpen = Boolean(currentExecution);

  // Project detail store action
  const setActiveTab = useProjectDetailStore((s) => s.setActiveTab);

  const handleSelectExecution = useCallback(
    async (execution: { id: string }) => {
      try {
        setActiveTab("executions");
        await loadExecution(execution.id);
      } catch (error) {
        logger.error(
          "Failed to load execution details",
          {
            component: "ExecutionPanel",
            action: "handleSelectExecution",
            executionId: execution.id,
          },
          error,
        );
        toast.error("Failed to load execution details");
      }
    },
    [loadExecution, setActiveTab],
  );

  const handleCloseExecutionViewer = useCallback(() => {
    closeExecutionViewer();
  }, [closeExecutionViewer]);

  return (
    <div className="h-full flex flex-col md:flex-row min-h-0">
      {/* Execution History List */}
      <div
        className={`${
          isExecutionViewerOpen
            ? "hidden md:block md:w-1/2 md:border-r md:border-gray-800"
            : "block md:w-full"
        } flex-1 min-h-0`}
      >
        <ExecutionHistory onSelectExecution={handleSelectExecution} />
      </div>

      {/* Execution Viewer (side panel on desktop) */}
      {isExecutionViewerOpen && currentExecution && (
        <div className="w-full md:w-1/2 flex-1 flex flex-col min-h-0">
          <ExecutionViewer
            workflowId={currentExecution.workflowId}
            execution={currentExecution}
            onClose={handleCloseExecutionViewer}
          />
        </div>
      )}
    </div>
  );
}
