/**
 * DashboardViewWrapper - Route wrapper for the dashboard that provides navigation callbacks.
 *
 * This component wraps DashboardView and provides navigation callbacks using React Router.
 */
import { useCallback, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import Dashboard from './DashboardView';
import { type DashboardTab } from './DashboardTabs';
import { type Project, useProjectStore, buildProjectFolderPath } from '@stores/projectStore';
import { useWorkflowStore } from '@stores/workflowStore';
import { useExecutionStore } from '@stores/executionStore';
import { useModals } from '@shared/modals';
import { useGuidedTour } from '@shared/onboarding';
import { selectors } from '@constants/selectors';
import { logger } from '@utils/logger';
import toast from 'react-hot-toast';

interface DashboardViewWrapperProps {
  initialTab?: DashboardTab;
}

export default function DashboardViewWrapper({ initialTab }: DashboardViewWrapperProps) {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [isGeneratingWorkflow, setIsGeneratingWorkflow] = useState(false);

  const { projects, setCurrentProject } = useProjectStore();
  const { openProjectModal, openDocs } = useModals();
  const { openTour } = useGuidedTour();
  const startExecution = useExecutionStore((state) => state.startExecution);
  const loadExecution = useExecutionStore((state) => state.loadExecution);

  // Get tab from URL params or use initialTab or default to 'home'
  const tabParam = searchParams.get('tab') as DashboardTab | null;
  const activeTab = tabParam ?? initialTab ?? 'home';

  const handleProjectSelect = useCallback(
    (project: Project) => {
      navigate(`/projects/${project.id}`);
    },
    [navigate]
  );

  const handleCreateProject = useCallback(() => {
    openProjectModal();
  }, [openProjectModal]);

  const handleCreateFirstWorkflow = useCallback(async () => {
    try {
      // Create a default project
      const project = await useProjectStore.getState().createProject({
        name: 'My Automations',
        description: 'Automated browser workflows',
        folder_path: buildProjectFolderPath('my-automations'),
      });

      // Create an empty workflow in the project
      const workflowName = `my-first-workflow`;
      const workflow = await useWorkflowStore
        .getState()
        .createWorkflow(workflowName, '/', project.id);

      if (workflow?.id) {
        // Navigate directly to the workflow builder
        navigate(`/projects/${project.id}/workflows/${workflow.id}`);
        toast.success('Welcome! Start building your first automation.');
      } else {
        throw new Error('Failed to create workflow');
      }
    } catch (error) {
      logger.error(
        'Failed to create first workflow',
        { component: 'DashboardViewWrapper', action: 'handleCreateFirstWorkflow' },
        error
      );
      toast.error('Failed to create workflow. Please try again.');
    }
  }, [navigate]);

  const handleStartRecording = useCallback(() => {
    navigate('/record/new');
  }, [navigate]);

  const handleOpenSettings = useCallback(() => {
    navigate('/settings');
  }, [navigate]);

  const handleOpenHelp = useCallback(() => {
    openDocs('getting-started');
  }, [openDocs]);

  const handleOpenTutorial = useCallback(() => {
    openTour();
  }, [openTour]);

  const handleNavigateToWorkflow = useCallback(
    async (projectId: string, workflowId: string) => {
      navigate(`/projects/${projectId}/workflows/${workflowId}`);
    },
    [navigate]
  );

  const handleViewExecution = useCallback(
    async (executionId: string, workflowId: string) => {
      // Load the execution details
      await loadExecution(executionId);

      // Find the workflow's project and navigate to the workflow view
      const workflowsResponse = await fetch(`/api/v1/workflows/${workflowId}`);
      if (workflowsResponse.ok) {
        const workflowData = await workflowsResponse.json();
        const projectId = workflowData.project_id ?? workflowData.projectId;
        if (projectId) {
          navigate(`/projects/${projectId}/workflows/${workflowId}`);
        }
      }
    },
    [loadExecution, navigate]
  );

  const handleAIGenerateWorkflow = useCallback(
    async (prompt: string) => {
      setIsGeneratingWorkflow(true);

      try {
        // If no project exists, create one first
        if (projects.length === 0) {
          const project = await useProjectStore.getState().createProject({
            name: 'My Automations',
            description: 'Automated workflows',
            folder_path: buildProjectFolderPath('my-automations'),
          });
          setCurrentProject(project);

          // Generate workflow in new project
          const workflow = await useWorkflowStore
            .getState()
            .generateWorkflow(prompt, 'ai-generated-workflow', '/', project.id);

          if (workflow?.id) {
            navigate(`/projects/${project.id}/workflows/${workflow.id}`);
            toast.success('Workflow generated successfully!');
          }
        } else {
          // Use first project
          const targetProject = projects[0];
          setCurrentProject(targetProject);

          // Generate workflow
          const workflow = await useWorkflowStore
            .getState()
            .generateWorkflow(prompt, 'ai-generated-workflow', '/', targetProject.id);

          if (workflow?.id) {
            navigate(`/projects/${targetProject.id}/workflows/${workflow.id}`);
            toast.success('Workflow generated successfully!');
          }
        }
      } catch (error) {
        logger.error(
          'Failed to generate workflow',
          { component: 'DashboardViewWrapper', action: 'handleAIGenerateWorkflow' },
          error
        );
        toast.error('Failed to generate workflow. Please try again.');
      } finally {
        setIsGeneratingWorkflow(false);
      }
    },
    [projects, setCurrentProject, navigate]
  );

  const handleRunWorkflow = useCallback(
    async (workflowId: string) => {
      try {
        await startExecution(workflowId);
        toast.success('Workflow execution started');
      } catch (error) {
        logger.error(
          'Failed to start workflow execution',
          { component: 'DashboardViewWrapper', action: 'handleRunWorkflow', workflowId },
          error
        );
        toast.error('Failed to start workflow');
      }
    },
    [startExecution]
  );

  const handleViewAllWorkflows = useCallback(() => {
    navigate('/workflows');
  }, [navigate]);

  const handleViewAllExecutions = useCallback(() => {
    navigate('/executions');
  }, [navigate]);

  const handleTabChange = useCallback(
    (tab: DashboardTab) => {
      if (tab === 'home') {
        setSearchParams({});
      } else {
        setSearchParams({ tab });
      }
    },
    [setSearchParams]
  );

  return (
    <div data-testid={selectors.app.shell.ready}>
      <Dashboard
        activeTab={activeTab}
        onTabChange={handleTabChange}
        onProjectSelect={handleProjectSelect}
        onCreateProject={handleCreateProject}
        onCreateFirstWorkflow={handleCreateFirstWorkflow}
        onStartRecording={handleStartRecording}
        onOpenSettings={handleOpenSettings}
        onOpenHelp={handleOpenHelp}
        onOpenTutorial={handleOpenTutorial}
        onNavigateToWorkflow={handleNavigateToWorkflow}
        onViewExecution={handleViewExecution}
        onAIGenerateWorkflow={handleAIGenerateWorkflow}
        onRunWorkflow={handleRunWorkflow}
        onViewAllWorkflows={handleViewAllWorkflows}
        onViewAllExecutions={handleViewAllExecutions}
        isGeneratingWorkflow={isGeneratingWorkflow}
      />
    </div>
  );
}
