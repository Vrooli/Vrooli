import { useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ExecutionHistory, InlineExecutionViewer } from "@/domains/executions";
import { useExecutionStore } from "@/domains/executions";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import { useProjectDetailStore } from "./hooks/useProjectDetailStore";

/**
 * Panel component for execution history and viewer in ProjectDetail
 */
export function ExecutionPanel() {
  const navigate = useNavigate();

  // Execution store state
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const isExecutionViewerOpen = Boolean(currentExecution);

  // Project detail store
  const projectId = useProjectDetailStore((s) => s.projectId);
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

  const handleRerun = useCallback(() => {
    if (!currentExecution || !projectId) return;
    // Navigate to Record page in execution mode with the workflow
    navigate(`/record/new?mode=execution&workflow_id=${currentExecution.workflowId}&project_id=${projectId}`);
  }, [currentExecution, projectId, navigate]);

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
        <ExecutionHistory projectId={projectId ?? undefined} onSelectExecution={handleSelectExecution} />
      </div>

      {/* Inline Execution Viewer (side panel on desktop) */}
      {isExecutionViewerOpen && currentExecution && projectId && (
        <div className="w-full md:w-1/2 flex-1 flex flex-col min-h-0">
          <InlineExecutionViewer
            executionId={currentExecution.id}
            workflowId={currentExecution.workflowId}
            projectId={projectId}
            onClose={handleCloseExecutionViewer}
            onRerun={handleRerun}
          />
        </div>
      )}
    </div>
  );
}
