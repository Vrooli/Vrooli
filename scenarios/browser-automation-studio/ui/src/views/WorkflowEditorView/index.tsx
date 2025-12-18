/**
 * WorkflowEditorView - Route wrapper for the workflow builder/editor.
 */
import { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { useProjectStore, type Project } from '@/domains/projects';
import { useWorkflowStore } from '@stores/workflowStore';
import { useExecutionStore } from '@/domains/executions';
import { useDashboardStore, type RecentWorkflow } from '@stores/dashboardStore';
import { useModals } from '@shared/modals';
import { useMediaQuery } from '@hooks/useMediaQuery';
import { ExecutionPaneLayout } from '@/domains/executions';
import { logger } from '@utils/logger';
import toast from 'react-hot-toast';

const Header = lazy(() => import('@shared/layout/Header'));
const Sidebar = lazy(() => import('@shared/layout/Sidebar'));
const WorkflowBuilder = lazy(() => import('@/domains/workflows/builder/WorkflowBuilder'));
const AIPromptModal = lazy(() => import('@/domains/ai/AIPromptModal'));

export default function WorkflowEditorView() {
  const navigate = useNavigate();
  const { projectId, workflowId } = useParams<{ projectId: string; workflowId: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedFolder, setSelectedFolder] = useState('/');

  const { projects, getProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow, currentWorkflow } = useWorkflowStore();
  const { setLastEditedWorkflow } = useDashboardStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const viewerWorkflowId = useExecutionStore((state) => state.viewerWorkflowId);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);

  const { showAIModal, openAIModal, closeAIModal, openDocs } = useModals();
  const isLargeScreen = useMediaQuery('(min-width: 1024px)');

  const isExecutionViewerOpen = Boolean(viewerWorkflowId);
  const activeExecutionWorkflowId = viewerWorkflowId ?? currentExecution?.workflowId ?? workflowId ?? null;

  // Load project and workflow on mount
  useEffect(() => {
    const loadData = async () => {
      if (!projectId || !workflowId) {
        navigate('/');
        return;
      }

      setLoading(true);

      // First check if project is already in store
      let foundProject = projects.find((p) => p.id === projectId);

      if (!foundProject) {
        // Fetch from API
        foundProject = (await getProject(projectId)) ?? undefined;
      }

      if (!foundProject) {
        toast.error('Project not found');
        navigate('/');
        setLoading(false);
        return;
      }

      setProject(foundProject);
      setCurrentProject(foundProject);
      setSelectedFolder(foundProject.folder_path ?? '/');

      // Load the workflow
      try {
        await loadWorkflow(workflowId);
        const loaded = useWorkflowStore.getState().currentWorkflow;
        if (loaded) {
          setSelectedFolder(loaded.folderPath ?? foundProject.folder_path ?? '/');

          // Update last edited workflow for "Continue Editing" feature
          const lastEdited: RecentWorkflow = {
            id: loaded.id,
            name: loaded.name ?? 'Untitled',
            projectId: foundProject.id,
            projectName: foundProject.name,
            updatedAt: new Date(),
            folderPath: loaded.folderPath ?? '/',
          };
          setLastEditedWorkflow(lastEdited);
        }
      } catch (error) {
        logger.error(
          'Failed to load workflow',
          {
            component: 'WorkflowEditorView',
            action: 'loadData',
            workflowId,
            projectId,
          },
          error
        );
        toast.error('Failed to load workflow');
        navigate(`/projects/${projectId}`);
      }

      setLoading(false);
    };

    loadData();
  }, [projectId, workflowId, projects, getProject, setCurrentProject, loadWorkflow, setLastEditedWorkflow, navigate]);

  const handleBackToDashboard = useCallback(() => {
    navigate('/');
  }, [navigate]);

  const handleBackToProject = useCallback(() => {
    if (projectId) {
      navigate(`/projects/${projectId}`);
    } else {
      navigate('/');
    }
  }, [projectId, navigate]);

  const handleStartRecording = useCallback(() => {
    navigate('/record/new');
  }, [navigate]);

  const handleSwitchToManualBuilder = useCallback(async () => {
    if (!project?.id) {
      toast.error('Select a project before using the manual builder.');
      return;
    }

    closeAIModal();

    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || '/';

    try {
      const workflow = await useWorkflowStore
        .getState()
        .createWorkflow(workflowName, folderPath, project.id);

      if (workflow?.id) {
        navigate(`/projects/${project.id}/workflows/${workflow.id}`);
      } else {
        toast.error('Failed to create workflow.');
      }
    } catch (error) {
      logger.error(
        'Failed to create workflow',
        {
          component: 'WorkflowEditorView',
          action: 'handleSwitchToManualBuilder',
          projectId: project?.id,
        },
        error
      );
      toast.error('Failed to open the manual builder. Please try again.');
    }
  }, [project, selectedFolder, closeAIModal, navigate]);

  const handleWorkflowGenerated = useCallback(
    async (workflow: Record<string, unknown> & { id?: string }) => {
      if (!workflow?.id || !project) return;
      navigate(`/projects/${project.id}/workflows/${workflow.id}`);
    },
    [project, navigate]
  );

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center bg-flow-bg">
        <LoadingSpinner variant="default" size={24} message="Loading workflow..." />
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col bg-flow-bg" data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-[72px] border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90" />
        }
      >
        <Header
          onNewWorkflow={openAIModal}
          onBackToDashboard={handleBackToDashboard}
          onBackToProject={handleBackToProject}
          currentProject={project}
          currentWorkflow={currentWorkflow}
          showBackToProject={true}
          onOpenHelp={() => openDocs('shortcuts')}
        />
      </Suspense>

      <div className="flex-1 flex overflow-hidden min-h-0">
        <Suspense
          fallback={
            <div className="hidden lg:block w-64 border-r border-gray-800 bg-flow-bg" />
          }
        >
          <Sidebar />
        </Suspense>

        <ExecutionPaneLayout
          isOpen={isExecutionViewerOpen}
          workflowId={activeExecutionWorkflowId}
          execution={currentExecution}
          isLargeScreen={isLargeScreen}
          onClose={closeExecutionViewer}
        >
          <Suspense
            fallback={
              <div className="h-full flex items-center justify-center">
                <LoadingSpinner variant="default" size={24} message="Loading workflow builder..." />
              </div>
            }
          >
            <WorkflowBuilder projectId={project?.id} onStartRecording={handleStartRecording} />
          </Suspense>
        </ExecutionPaneLayout>
      </div>

      {showAIModal && (
        <Suspense
          fallback={
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <LoadingSpinner variant="branded" size={24} message="Loading AI assistant..." />
            </div>
          }
        >
          <AIPromptModal
            onClose={closeAIModal}
            folder={selectedFolder}
            projectId={project?.id}
            onSwitchToManual={handleSwitchToManualBuilder}
            onSuccess={handleWorkflowGenerated}
          />
        </Suspense>
      )}
    </div>
  );
}
