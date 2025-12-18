import { useCallback } from "react";
import {
  FileCode,
  Play,
  Clock,
  PlayCircle,
  Loader,
  WifiOff,
  FolderTree,
  Trash2,
  CheckSquare,
  Square,
  MoreVertical,
  Search,
  Plus,
} from "lucide-react";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useExecutionStore } from "@stores/executionStore";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { selectors } from "@constants/selectors";
import { ConfirmDialog } from "@shared/ui";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import {
  useProjectDetailStore,
  useFilteredWorkflows,
  type WorkflowWithStats,
} from "./hooks/useProjectDetailStore";

interface WorkflowCardGridProps {
  projectId: string;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
}

/**
 * Grid view of workflow cards with selection, execution, and delete actions
 */
export function WorkflowCardGrid({
  projectId,
  onWorkflowSelect,
  onCreateWorkflow,
  onCreateWorkflowDirect,
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
  const startExecution = useExecutionStore((state) => state.startExecution);

  // Dialog hooks
  const {
    dialogState: confirmDialogState,
    confirm: requestConfirm,
    close: closeConfirmDialog,
  } = useConfirmDialog();

  // Format date helper
  const formatDate = useCallback((dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }, []);

  // Handlers
  const handleExecuteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string) => {
      e.stopPropagation();
      setExecutionInProgress(workflowId, true);

      try {
        await startExecution(workflowId);
        setActiveTab("executions");
        logger.info("Workflow execution started", {
          component: "WorkflowCardGrid",
          action: "handleExecuteWorkflow",
          workflowId,
        });
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
        alert("Failed to execute workflow. Please try again.");
      } finally {
        setExecutionInProgress(workflowId, false);
      }
    },
    [startExecution, setExecutionInProgress, setActiveTab],
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
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-2xl mx-auto pt-8">
          <div className="text-center mb-8">
            <div className="mb-5 flex items-center justify-center">
              <div className="p-4 bg-green-500/20 rounded-2xl">
                {error ? (
                  <WifiOff size={40} className="text-red-400" />
                ) : (
                  <FileCode size={40} className="text-green-400" />
                )}
              </div>
            </div>
            <h3 className="text-xl font-semibold text-surface mb-2">
              {error ? "Unable to Load Workflows" : "Ready to Automate"}
            </h3>
            <p className="text-gray-400 max-w-md mx-auto">
              {error
                ? "There was an issue connecting to the API. You can still use the interface when the connection is restored."
                : "Create your first workflow to automate browser tasks. Use AI to describe what you want, or build visually with the drag-and-drop builder."}
            </p>
          </div>

          {!error && (
            <>
              {/* Quick Actions */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-8">
                <button
                  onClick={onCreateWorkflow}
                  className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/10 transition-all group"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 bg-amber-500/20 rounded-lg group-hover:bg-amber-500/30 transition-colors">
                      <Plus size={20} className="text-amber-400" />
                    </div>
                    <h4 className="font-medium text-surface">AI-Assisted</h4>
                  </div>
                  <p className="text-sm text-gray-400">
                    Describe your automation in plain language and let AI generate the workflow.
                  </p>
                </button>

                <button
                  onClick={onCreateWorkflowDirect ?? onCreateWorkflow}
                  className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/10 transition-all group"
                >
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                      <FolderTree size={20} className="text-blue-400" />
                    </div>
                    <h4 className="font-medium text-surface">Visual Builder</h4>
                  </div>
                  <p className="text-sm text-gray-400">
                    Use the drag-and-drop interface to build workflows step by step.
                  </p>
                </button>
              </div>

              {/* Workflow ideas */}
              <div className="text-center text-sm text-gray-500">
                <p className="mb-2">Try automating:</p>
                <div className="flex flex-wrap justify-center gap-2">
                  <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Form submissions</span>
                  <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">UI testing</span>
                  <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Data extraction</span>
                  <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Screenshots</span>
                </div>
              </div>
            </>
          )}
        </div>
        <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
      </div>
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
      </div>
    );
  }

  // Render workflow grid
  return (
    <div className="flex-1 overflow-auto p-6">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {filteredWorkflows.map((workflow: WorkflowWithStats) => {
          const isSelected = selectedWorkflows.has(workflow.id);
          const isDeleting = deletingWorkflowId === workflow.id;
          const isActionsOpen = showWorkflowActionsFor === workflow.id;
          const executionCount = workflow.stats?.execution_count || 0;
          const successRate = workflow.stats?.success_rate;

          return (
            <div
              key={workflow.id}
              data-testid={selectors.workflows.card}
              data-workflow-id={workflow.id}
              data-workflow-name={workflow.name}
              onClick={() => handleCardClick(workflow)}
              className={`group relative bg-flow-node border rounded-xl p-5 cursor-pointer transition-all ${
                isDeleting
                  ? "opacity-50 pointer-events-none"
                  : selectionMode
                    ? isSelected
                      ? "border-flow-accent shadow-lg shadow-blue-500/20"
                      : "border-gray-700 hover:border-flow-accent/60"
                    : "border-gray-700 hover:border-flow-accent/60 hover:shadow-lg hover:shadow-blue-500/10"
              }`}
            >
              {/* Deleting Overlay */}
              {isDeleting && (
                <div className="absolute inset-0 bg-flow-node/80 rounded-xl flex items-center justify-center z-10">
                  <Loader size={24} className="animate-spin text-red-400" />
                </div>
              )}

              {/* Workflow Header */}
              <div className="flex items-start justify-between gap-3 mb-3" data-workflow-header>
                <div className="flex items-center gap-3 min-w-0">
                  <div
                    className={`flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center border ${
                      selectionMode && isSelected
                        ? "bg-flow-accent/20 border-flow-accent/30"
                        : "bg-gradient-to-br from-green-500/20 to-emerald-500/10 border-green-500/20"
                    }`}
                  >
                    <FileCode size={18} className="text-green-400" />
                  </div>
                  <div className="min-w-0">
                    <h3
                      className="font-semibold text-surface truncate"
                      title={String(workflow.name)}
                    >
                      {String(workflow.name)}
                    </h3>
                    <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
                      <span>{executionCount} run{executionCount !== 1 ? "s" : ""}</span>
                      {successRate != null && (
                        <>
                          <span>•</span>
                          <span
                            className={
                              successRate >= 80
                                ? "text-green-500"
                                : successRate >= 50
                                  ? "text-amber-500"
                                  : "text-red-400"
                            }
                          >
                            {successRate}% success
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                </div>

                {selectionMode ? (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleWorkflowSelection(workflow.id);
                    }}
                    className="flex-shrink-0 p-2 text-gray-300 hover:text-surface transition-colors"
                    title={isSelected ? "Deselect workflow" : "Select workflow"}
                  >
                    {isSelected ? <CheckSquare size={16} /> : <Square size={16} />}
                  </button>
                ) : (
                  <div className="relative flex-shrink-0">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        setShowWorkflowActionsFor(isActionsOpen ? null : workflow.id);
                      }}
                      className="p-1.5 text-gray-500 hover:text-surface hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
                      aria-label="Workflow actions"
                      aria-expanded={isActionsOpen}
                    >
                      <MoreVertical size={16} />
                    </button>

                    {isActionsOpen && (
                      <>
                        <div
                          className="fixed inset-0 z-20"
                          onClick={(e) => {
                            e.stopPropagation();
                            setShowWorkflowActionsFor(null);
                          }}
                        />
                        <div className="absolute right-0 top-full mt-1 z-30 w-44 bg-flow-node border border-gray-700 rounded-lg shadow-xl overflow-hidden animate-fade-in">
                          <button
                            data-testid={selectors.workflowBuilder.executeButton}
                            onClick={(e) => {
                              setShowWorkflowActionsFor(null);
                              handleExecuteWorkflow(e, workflow.id);
                            }}
                            disabled={executionInProgress[workflow.id]}
                            className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-surface transition-colors disabled:opacity-50"
                          >
                            {executionInProgress[workflow.id] ? (
                              <Loader size={14} className="animate-spin" />
                            ) : (
                              <PlayCircle size={14} />
                            )}
                            Run Workflow
                          </button>
                          <div className="border-t border-gray-700" />
                          <button
                            onClick={(e) => handleDeleteWorkflow(e, workflow.id, workflow.name)}
                            className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                          >
                            <Trash2 size={14} />
                            Delete
                          </button>
                        </div>
                      </>
                    )}
                  </div>
                )}
              </div>

              {/* Workflow Description */}
              {workflow.description ? (
                <p
                  className={`text-sm mb-4 line-clamp-2 ${
                    selectionMode && isSelected ? "text-gray-200" : "text-gray-400"
                  }`}
                >
                  {(workflow.description as string | undefined) || ""}
                </p>
              ) : (
                <p className="text-gray-600 text-sm mb-4 italic">No description</p>
              )}

              {/* Workflow Stats */}
              {(executionCount > 0 || successRate != null) && (
                <div className="grid grid-cols-2 gap-3 mb-4">
                  <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                    <div className="text-lg font-semibold text-surface">{executionCount}</div>
                    <div className="text-xs text-gray-500">Executions</div>
                  </div>
                  <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                    <div
                      className={`text-lg font-semibold ${
                        successRate != null
                          ? successRate >= 80
                            ? "text-green-400"
                            : successRate >= 50
                              ? "text-amber-400"
                              : "text-red-400"
                          : "text-gray-500"
                      }`}
                    >
                      {successRate != null ? `${successRate}%` : "—"}
                    </div>
                    <div className="text-xs text-gray-500">Success</div>
                  </div>
                </div>
              )}

              {/* Footer */}
              <div
                className={`flex items-center justify-between text-xs pt-3 border-t border-gray-700/50 ${
                  selectionMode && isSelected ? "text-gray-200" : "text-gray-500"
                }`}
              >
                <div className="flex items-center gap-1.5">
                  <Clock size={12} />
                  <span>Updated {formatDate(workflow.updated_at || "")}</span>
                </div>
                {workflow.stats?.last_execution && (
                  <div className="flex items-center gap-1.5 text-green-500/80">
                    <Play size={10} />
                    <span>{formatDate(workflow.stats.last_execution)}</span>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
    </div>
  );
}
