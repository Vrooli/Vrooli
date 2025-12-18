import { useEffect, useCallback } from "react";
import { Plus, WifiOff } from "lucide-react";
import { logger } from "@utils/logger";
import { getConfig } from "@/config";
import toast from "react-hot-toast";
import { selectors } from "@constants/selectors";
import type { Project } from "./store";
import { useProjectStore } from "./store";
import type { Workflow } from "@stores/workflowStore";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { ConfirmDialog } from "@shared/ui";
import ProjectModal from "./ProjectModal";

// New decomposed components
import { ProjectDetailHeader } from "./ProjectDetailHeader";
import { ProjectDetailTabs } from "./ProjectDetailTabs";
import { ExecutionPanel } from "./ExecutionPanel";
import { WorkflowCardGrid } from "./WorkflowCardGrid";
import { ProjectFileTree } from "./ProjectFileTree";
import { useProjectDetailStore } from "./hooks/useProjectDetailStore";

interface ProjectDetailProps {
  project: Project;
  onBack: () => void;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
  onStartRecording?: () => void;
}

/**
 * ProjectDetail - Orchestrator component for project detail view
 *
 * This component manages the overall layout and coordination between
 * sub-components. State is managed via the useProjectDetailStore Zustand store.
 */
function ProjectDetail({
  project,
  onBack,
  onWorkflowSelect,
  onCreateWorkflow,
  onCreateWorkflowDirect,
  onStartRecording,
}: ProjectDetailProps) {
  // Store state
  const activeTab = useProjectDetailStore((s) => s.activeTab);
  const viewMode = useProjectDetailStore((s) => s.viewMode);
  const workflows = useProjectDetailStore((s) => s.workflows);
  const error = useProjectDetailStore((s) => s.error);
  const showEditProjectModal = useProjectDetailStore((s) => s.showEditProjectModal);

  // Store actions
  const initializeForProject = useProjectDetailStore((s) => s.initializeForProject);
  const reset = useProjectDetailStore((s) => s.reset);
  const fetchWorkflows = useProjectDetailStore((s) => s.fetchWorkflows);
  const fetchProjectEntries = useProjectDetailStore((s) => s.fetchProjectEntries);
  const setError = useProjectDetailStore((s) => s.setError);
  const setShowEditProjectModal = useProjectDetailStore((s) => s.setShowEditProjectModal);
  const setIsDeletingProject = useProjectDetailStore((s) => s.setIsDeletingProject);
  const setIsImportingRecording = useProjectDetailStore((s) => s.setIsImportingRecording);

  // Project store
  const { deleteProject } = useProjectStore();

  // Dialog hook for delete project confirmation
  const {
    dialogState: confirmDialogState,
    confirm: requestConfirm,
    close: closeConfirmDialog,
  } = useConfirmDialog();

  // Initialize store when project changes
  useEffect(() => {
    initializeForProject(project.id);
    fetchWorkflows(project.id);
    fetchProjectEntries(project.id);

    return () => {
      reset();
    };
  }, [project.id, initializeForProject, fetchWorkflows, fetchProjectEntries, reset]);

  // Handle delete project
  const handleDeleteProject = useCallback(async () => {
    const confirmed = await requestConfirm({
      title: "Delete project?",
      message:
        "Delete this project and all associated workflows? This cannot be undone.",
      confirmLabel: "Delete Project",
      cancelLabel: "Cancel",
      danger: true,
    });
    if (!confirmed) return;

    setIsDeletingProject(true);
    try {
      await deleteProject(project.id);
      toast.success("Project deleted successfully");
      onBack();
    } catch (error) {
      logger.error(
        "Failed to delete project",
        {
          component: "ProjectDetail",
          action: "handleDeleteProject",
          projectId: project.id,
        },
        error,
      );
      toast.error("Failed to delete project");
    } finally {
      setIsDeletingProject(false);
    }
  }, [project.id, deleteProject, onBack, requestConfirm, setIsDeletingProject]);

  // Handle recording import
  const handleImportRecording = useCallback(
    async (file: File) => {
      setIsImportingRecording(true);

      try {
        const config = await getConfig();
        const formData = new FormData();
        formData.append("file", file);
        formData.append("project_id", project.id);
        if (project.name) {
          formData.append("project_name", project.name);
        }

        const response = await fetch(`${config.API_URL}/recordings/import`, {
          method: "POST",
          body: formData,
        });

        if (!response.ok) {
          const text = await response.text();
          try {
            const payload = JSON.parse(text);
            const message =
              payload.message || payload.error || "Failed to import recording";
            throw new Error(message);
          } catch {
            throw new Error(text || "Failed to import recording");
          }
        }

        const payload = await response.json();
        const executionId = payload.execution_id || payload.executionId;
        toast.success(
          `Recording imported${executionId ? ` (execution ${executionId})` : ""}.`,
        );
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to import recording";
        toast.error(message);
      } finally {
        setIsImportingRecording(false);
      }
    },
    [project.id, project.name, setIsImportingRecording],
  );

  // Status bar for API connection issues
  const StatusBar = () => {
    if (!error) return null;

    return (
      <div className="bg-red-900/20 border-l-4 border-red-500 p-4 mx-6 mb-4 rounded-r-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <WifiOff size={20} className="text-red-400" />
            <div>
              <div className="text-red-400 font-medium">API Connection Failed</div>
              <div className="text-red-300/80 text-sm">{error}</div>
            </div>
          </div>
          <button
            onClick={() => {
              setError(null);
              fetchWorkflows(project.id);
            }}
            className="px-3 py-1 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  };

  return (
    <>
      <div className="flex-1 flex flex-col h-full overflow-hidden min-h-0">
        {/* Header with breadcrumbs, project info, and actions */}
        <ProjectDetailHeader
          project={project}
          onBack={onBack}
          onCreateWorkflow={onCreateWorkflow}
          onStartRecording={onStartRecording}
          onDeleteProject={handleDeleteProject}
          onImportRecording={handleImportRecording}
          onWorkflowSelect={onWorkflowSelect}
        />

        {/* Tab navigation and search */}
        <div className="px-6 pb-4 border-b border-gray-800">
          <ProjectDetailTabs workflowCount={workflows.length} />
        </div>

        {/* Status Bar for API errors */}
        <StatusBar />

        {/* Content Area */}
        <div className="flex-1 overflow-hidden min-h-0">
          {activeTab === "executions" ? (
            <ExecutionPanel />
          ) : viewMode === "tree" ? (
            <ProjectFileTree
              project={project}
              onWorkflowSelect={onWorkflowSelect}
            />
          ) : (
            <WorkflowCardGrid
              projectId={project.id}
              onWorkflowSelect={onWorkflowSelect}
              onCreateWorkflow={onCreateWorkflow}
              onCreateWorkflowDirect={onCreateWorkflowDirect}
              onStartRecording={onStartRecording}
            />
          )}
        </div>
      </div>

      {/* Floating Action Button (FAB) - Mobile only */}
      <button
        data-testid={selectors.workflows.newButtonFab}
        onClick={onCreateWorkflow}
        className="md:hidden fixed bottom-6 right-6 z-40 w-14 h-14 bg-flow-accent text-white rounded-full shadow-lg hover:bg-blue-600 transition-all hover:shadow-xl flex items-center justify-center"
        aria-label="New Workflow"
      >
        <Plus size={24} />
      </button>

      {/* Edit Project Modal */}
      {showEditProjectModal && (
        <ProjectModal
          onClose={() => setShowEditProjectModal(false)}
          project={project}
          onSuccess={() => toast.success("Project updated successfully")}
        />
      )}

      {/* Delete Project Confirmation Dialog */}
      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
    </>
  );
}

export default ProjectDetail;
