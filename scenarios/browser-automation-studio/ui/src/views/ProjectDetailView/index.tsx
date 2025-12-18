/**
 * ProjectDetailView - Route wrapper for project detail page.
 */
import { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { LoadingSpinner } from '@shared/ui';
import { selectors } from '@constants/selectors';
import { useProjectStore, type Project } from '@stores/projectStore';
import { useWorkflowStore } from '@stores/workflowStore';
import { useModals } from '@shared/modals';
import { logger } from '@utils/logger';
import toast from 'react-hot-toast';

const ProjectDetail = lazy(() => import('@/domains/projects/ProjectDetail'));
const AIPromptModal = lazy(() => import('@/domains/ai/AIPromptModal'));

export default function ProjectDetailView() {
  const navigate = useNavigate();
  const { projectId } = useParams<{ projectId: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedFolder, setSelectedFolder] = useState('/');

  const { projects, getProject, setCurrentProject } = useProjectStore();
  const { showAIModal, openAIModal, closeAIModal } = useModals();

  // Load project on mount
  useEffect(() => {
    const loadProject = async () => {
      if (!projectId) {
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

      if (foundProject) {
        setProject(foundProject);
        setCurrentProject(foundProject);
        setSelectedFolder(foundProject.folder_path ?? '/');
      } else {
        toast.error('Project not found');
        navigate('/');
      }

      setLoading(false);
    };

    loadProject();
  }, [projectId, projects, getProject, setCurrentProject, navigate]);

  const handleBack = useCallback(() => {
    navigate('/');
  }, [navigate]);

  const handleWorkflowSelect = useCallback(
    async (workflow: Record<string, unknown> & { id?: string }) => {
      if (!project || !workflow.id) return;
      navigate(`/projects/${project.id}/workflows/${workflow.id}`);
    },
    [project, navigate]
  );

  const handleCreateWorkflow = useCallback(() => {
    openAIModal();
  }, [openAIModal]);

  const handleCreateWorkflowDirect = useCallback(async () => {
    if (!project?.id) {
      toast.error('Select a project before creating a workflow.');
      return;
    }

    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || '/';

    try {
      const workflow = await useWorkflowStore
        .getState()
        .createWorkflow(workflowName, folderPath, project.id);

      if (workflow?.id) {
        navigate(`/projects/${project.id}/workflows/${workflow.id}`);
      }
    } catch (error) {
      logger.error(
        'Failed to create workflow',
        {
          component: 'ProjectDetailView',
          action: 'handleCreateWorkflowDirect',
          projectId: project?.id,
        },
        error
      );
      toast.error('Failed to create workflow. Please try again.');
    }
  }, [project, selectedFolder, navigate]);

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
          component: 'ProjectDetailView',
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
        <LoadingSpinner variant="default" size={24} message="Loading project..." />
      </div>
    );
  }

  if (!project) {
    return null;
  }

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="default" size={24} message="Loading project..." />
          </div>
        }
      >
        <ProjectDetail
          project={project}
          onBack={handleBack}
          onWorkflowSelect={handleWorkflowSelect}
          onCreateWorkflow={handleCreateWorkflow}
          onCreateWorkflowDirect={handleCreateWorkflowDirect}
          onStartRecording={handleStartRecording}
        />
      </Suspense>

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
            projectId={project.id}
            onSwitchToManual={handleSwitchToManualBuilder}
            onSuccess={handleWorkflowGenerated}
          />
        </Suspense>
      )}
    </div>
  );
}
