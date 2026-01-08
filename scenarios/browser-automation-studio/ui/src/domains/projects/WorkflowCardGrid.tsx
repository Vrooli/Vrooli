import { useCallback } from "react";
import { Search } from "lucide-react";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useStartWorkflow } from "@/domains/executions";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { selectors } from "@constants/selectors";
import { ConfirmDialog, PromptDialog } from "@shared/ui";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import {
  useProjectDetailStore,
  useFilteredWorkflows,
  type WorkflowWithStats,
} from "./hooks/useProjectDetailStore";
import { WorkflowCard } from "./WorkflowCard";
import { EmptyWorkflowState } from "./EmptyWorkflowState";

interface WorkflowCardGridProps {
  projectId: string;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
  onStartRecording?: () => void;
}

/**
 * Grid view of workflow cards with selection, execution, and delete actions
 */
export function WorkflowCardGrid({
  projectId,
  onWorkflowSelect,
  onCreateWorkflow,
  onCreateWorkflowDirect,
  onStartRecording,
}: WorkflowCardGridProps) {
  // Store state
  const isLoading = useProjectDetailStore((s) => s.isLoading);
  const error = useProjectDetailStore((s) => s.error);
  const searchTerm = useProjectDetailStore((s) => s.searchTerm);
  const selectionMode = useProjectDetailStore((s) => s.selectionMode);
  const selectedWorkflows = useProjectDetailStore((s) => s.selectedWorkflows);
  const executionInProgress = useProjectDetailStore((s) => s.executionInProgress);
  const deletingWorkflowId = useProjectDetailStore((s) => s.deletingWorkflowId);
  const showWorkflowActionsFor = useProjectDetailStore((s) => s.showWorkflowActionsFor);

  // Store actions
  const setSearchTerm = useProjectDetailStore((s) => s.setSearchTerm);
  const toggleWorkflowSelection = useProjectDetailStore((s) => s.toggleWorkflowSelection);
  const setExecutionInProgress = useProjectDetailStore((s) => s.setExecutionInProgress);
  const setDeletingWorkflowId = useProjectDetailStore((s) => s.setDeletingWorkflowId);
  const setShowWorkflowActionsFor = useProjectDetailStore((s) => s.setShowWorkflowActionsFor);
  const removeWorkflow = useProjectDetailStore((s) => s.removeWorkflow);
  const setActiveTab = useProjectDetailStore((s) => s.setActiveTab);

  // Filtered workflows
  const filteredWorkflows = useFilteredWorkflows();

  // External stores
  const { bulkDeleteWorkflows } = useWorkflowStore();

  // Execution hook with start URL prompt
  const { startWorkflow, promptDialogProps } = useStartWorkflow({
    onSuccess: () => {
      setActiveTab("executions");
    },
  });

  // Dialog hooks
  const {
    dialogState: confirmDialogState,
    confirm: requestConfirm,
    close: closeConfirmDialog,
  } = useConfirmDialog();

  // Handlers
  const handleExecuteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string) => {
      e.stopPropagation();
      setExecutionInProgress(workflowId, true);

      try {
        const executionId = await startWorkflow({ workflowId });
        if (executionId) {
          logger.info("Workflow execution started", {
            component: "WorkflowCardGrid",
            action: "handleExecuteWorkflow",
            workflowId,
            executionId,
          });
        }
      } catch (error) {
        logger.error(
          "Failed to execute workflow",
          {
            component: "WorkflowCardGrid",
            action: "handleExecuteWorkflow",
            workflowId,
          },
          error,
        );
        toast.error("Failed to execute workflow. Please try again.");
      } finally {
        setExecutionInProgress(workflowId, false);
      }
    },
    [startWorkflow, setExecutionInProgress],
  );

  const handleDeleteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string, workflowName: string) => {
      e.stopPropagation();
      setShowWorkflowActionsFor(null);

      const confirmed = await requestConfirm({
        title: "Delete workflow?",
        message: `Delete "${workflowName}"? This cannot be undone.`,
        confirmLabel: "Delete",
        cancelLabel: "Cancel",
        danger: true,
      });
      if (!confirmed) return;

      setDeletingWorkflowId(workflowId);
      try {
        const deletedIds = await bulkDeleteWorkflows(projectId, [workflowId]);
        if (deletedIds.includes(workflowId)) {
          removeWorkflow(workflowId);
          toast.success(`Workflow "${workflowName}" deleted`);
        } else {
          throw new Error("Workflow was not deleted");
        }
      } catch (error) {
        logger.error(
          "Failed to delete workflow",
          {
            component: "WorkflowCardGrid",
            action: "handleDeleteWorkflow",
            workflowId,
          },
          error,
        );
        toast.error("Failed to delete workflow");
      } finally {
        setDeletingWorkflowId(null);
      }
    },
    [projectId, bulkDeleteWorkflows, requestConfirm, setDeletingWorkflowId, setShowWorkflowActionsFor, removeWorkflow],
  );

  const handleCardClick = useCallback(
    async (workflow: WorkflowWithStats) => {
      if (deletingWorkflowId === workflow.id) return;
      if (selectionMode) {
        toggleWorkflowSelection(workflow.id);
      } else {
        await onWorkflowSelect(workflow);
      }
    },
    [deletingWorkflowId, selectionMode, toggleWorkflowSelection, onWorkflowSelect],
  );

  // Render loading state
  if (isLoading) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-400">Loading workflows...</div>
        </div>
      </div>
    );
  }

  // Render empty state (no workflows and no search)
  if (filteredWorkflows.length === 0 && searchTerm === "") {
    return (
      <>
        <EmptyWorkflowState
          error={error}
          onCreateWorkflow={onCreateWorkflow}
          onCreateWorkflowDirect={onCreateWorkflowDirect}
          onStartRecording={onStartRecording}
        />
        <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
        <PromptDialog {...promptDialogProps} />
      </>
    );
  }

  // Render no results state (search active but no matches)
  if (filteredWorkflows.length === 0) {
    return (
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-center h-64 animate-fade-in">
          <div className="text-center max-w-sm">
            <div className="mb-4 flex items-center justify-center">
              <div className="w-16 h-16 rounded-full bg-gray-800 flex items-center justify-center">
                <Search size={28} className="text-gray-500" />
              </div>
            </div>
            <h3 className="text-lg font-semibold text-surface mb-2">
              No Workflows Found
            </h3>
            <p className="text-gray-400 mb-4">
              No workflows match &ldquo;<span className="text-gray-300">{searchTerm}</span>&rdquo;
            </p>
            <button
              onClick={() => setSearchTerm("")}
              className="text-flow-accent hover:text-blue-400 text-sm font-medium transition-colors"
            >
              Clear search
            </button>
          </div>
        </div>
        <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
        <PromptDialog {...promptDialogProps} />
      </div>
    );
  }

  // Render workflow grid
  return (
    <div className="flex-1 overflow-auto p-6">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filteredWorkflows.map((workflow: WorkflowWithStats) => (
          <WorkflowCard
            key={workflow.id}
            workflow={workflow}
            onClick={handleCardClick}
            testId={selectors.workflows.card}
            selectionMode={selectionMode}
            isSelected={selectedWorkflows.has(workflow.id)}
            onToggleSelection={toggleWorkflowSelection}
            isDeleting={deletingWorkflowId === workflow.id}
            isExecuting={executionInProgress[workflow.id]}
            onRun={handleExecuteWorkflow}
            onDelete={handleDeleteWorkflow}
            isActionsOpen={showWorkflowActionsFor === workflow.id}
            onToggleActions={setShowWorkflowActionsFor}
          />
        ))}
      </div>
      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
      <PromptDialog {...promptDialogProps} />
    </div>
  );
}
