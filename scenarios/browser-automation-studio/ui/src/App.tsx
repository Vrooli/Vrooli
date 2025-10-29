import { useCallback, useEffect, useState } from 'react';
import { ReactFlowProvider } from 'reactflow';
import WorkflowBuilder from './components/WorkflowBuilder';
import ExecutionViewer from './components/ExecutionViewer';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import AIPromptModal from './components/AIPromptModal';
import Dashboard from './components/Dashboard';
import ProjectDetail from './components/ProjectDetail';
import ProjectModal from './components/ProjectModal';
import { useExecutionStore } from './stores/executionStore';
import { useProjectStore, Project } from './stores/projectStore';
import { useWorkflowStore } from './stores/workflowStore';
import { logger } from './utils/logger';
import 'reactflow/dist/style.css';


type AppView = 'dashboard' | 'project-detail' | 'project-workflow';

function App() {
  const [currentView, setCurrentView] = useState<AppView>('dashboard');
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string>('/');
  const [selectedWorkflow, setSelectedWorkflow] = useState<any>(null);
  
  const { currentProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow } = useWorkflowStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);

  const transformWorkflow = useCallback((workflow: any) => {
    if (!workflow) return null;
    return {
      ...workflow,
      folderPath: workflow.folder_path ?? workflow.folderPath ?? '/',
      createdAt: workflow.created_at ? new Date(workflow.created_at) : workflow.createdAt ? new Date(workflow.createdAt) : new Date(),
      updatedAt: workflow.updated_at ? new Date(workflow.updated_at) : workflow.updatedAt ? new Date(workflow.updatedAt) : new Date(),
      projectId: workflow.project_id ?? workflow.projectId,
    };
  }, []);

  const safeNavigate = useCallback((state: Record<string, unknown>, url: string, replace = false) => {
    try {
      if (replace) {
        window.history.replaceState(state, '', url);
      } else {
        window.history.pushState(state, '', url);
      }
    } catch (error) {
      // Some embedded hosts sandbox history APIs; log but continue rendering.
      logger.warn('Failed to update history state', { component: 'App', action: 'safeNavigate' }, error);
    }
  }, []);

  const navigateToDashboard = useCallback((replace = false) => {
    const url = '/';
    const state = { view: 'dashboard' };
    safeNavigate(state, url, replace);
    setShowAIModal(false);
    setShowProjectModal(false);
    setSelectedWorkflow(null);
    setSelectedFolder('/');
    setCurrentProject(null);
    setCurrentView('dashboard');
  }, [safeNavigate, setCurrentProject]);

  const openProject = useCallback((project: Project, options?: { replace?: boolean }) => {
    if (!project) {
      navigateToDashboard(options?.replace ?? false);
      return;
    }

    const url = `/projects/${project.id}`;
    const state = { view: 'project-detail', projectId: project.id };
    safeNavigate(state, url, options?.replace ?? false);

    setShowAIModal(false);
    setShowProjectModal(false);
    setCurrentProject(project);
    setSelectedFolder(project.folder_path ?? '/');
    setSelectedWorkflow(null);
    setCurrentView('project-detail');
  }, [navigateToDashboard, safeNavigate, setCurrentProject]);

  const openWorkflow = useCallback(async (
    project: Project,
    workflowId: string,
    options?: { replace?: boolean; workflowData?: any }
  ) => {
    if (!project || !workflowId) {
      navigateToDashboard(options?.replace ?? false);
      return;
    }

    const url = `/projects/${project.id}/workflows/${workflowId}`;
    const state = { view: 'project-workflow', projectId: project.id, workflowId };
    safeNavigate(state, url, options?.replace ?? false);

    setShowAIModal(false);
    setShowProjectModal(false);
    setCurrentProject(project);

    const initialWorkflow = options?.workflowData ? transformWorkflow(options.workflowData) : null;
    if (initialWorkflow) {
      setSelectedWorkflow(initialWorkflow);
      setSelectedFolder(initialWorkflow.folderPath || project.folder_path || '/');
    } else {
      setSelectedWorkflow(null);
      setSelectedFolder(project.folder_path || '/');
    }

    await loadWorkflow(workflowId);
    const loadedWorkflow = useWorkflowStore.getState().currentWorkflow;
    if (loadedWorkflow) {
      const normalized = transformWorkflow(loadedWorkflow);
      if (normalized) {
        setSelectedWorkflow(normalized);
        setSelectedFolder(normalized.folderPath || project.folder_path || '/');
      }
    }
    setCurrentView('project-workflow');
  }, [loadWorkflow, navigateToDashboard, setCurrentProject, transformWorkflow]);

  const handleProjectSelect = (project: Project) => {
    openProject(project);
  };

  const handleWorkflowSelect = async (workflow: any) => {
    if (!currentProject) return;
    await openWorkflow(currentProject, workflow.id, { workflowData: workflow });
  };

  const handleBackToDashboard = () => {
    navigateToDashboard();
  };

  const handleBackToProjectDetail = () => {
    if (currentProject) {
      openProject(currentProject);
    } else {
      navigateToDashboard();
    }
  };

  const handleCreateProject = () => {
    setShowProjectModal(true);
  };

  const handleProjectCreated = (project: Project) => {
    // Automatically select the newly created project
    openProject(project);
  };

  const handleCreateWorkflow = () => {
    setShowAIModal(true);
  };

  const handleSwitchToManualBuilder = async () => {
    setShowAIModal(false);
    // Create a new empty workflow and switch to the manual builder
    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || '/';
    try {
      const workflow = await useWorkflowStore.getState().createWorkflow(
        workflowName,
        folderPath,
        currentProject?.id
      );
      if (currentProject && workflow?.id) {
        await openWorkflow(currentProject, workflow.id, { workflowData: workflow });
      }
    } catch (error) {
      logger.error('Failed to create workflow', { component: 'App', action: 'handleCreateWorkflow', projectId: currentProject?.id }, error);
    }
  };

  const handleWorkflowGenerated = async (workflow: any) => {
    if (!workflow?.id || !currentProject) {
      return;
    }

    try {
      await openWorkflow(currentProject, workflow.id, { workflowData: workflow });
    } catch (error) {
      logger.error('Failed to open generated workflow', { component: 'App', action: 'handleWorkflowGenerated', workflowId: workflow.id, projectId: currentProject.id }, error);
    }
  };

  useEffect(() => {
    const resolvePath = async (path: string, replace = false) => {
      const normalized = path.replace(/\/+/g, '/').replace(/\/$/, '') || '/';

      if (normalized === '/') {
        navigateToDashboard(replace);
        return;
      }

      const segments = normalized.split('/').filter(Boolean);
      if (segments[0] !== 'projects' || !segments[1]) {
        navigateToDashboard(replace);
        return;
      }

      const projectId = segments[1];
      const projectState = useProjectStore.getState();
      let project: Project | null | undefined = projectState.projects.find((p) => p.id === projectId);
      if (!project) {
        project = await projectState.getProject(projectId) as Project | null;
        if (project) {
          useProjectStore.setState((state) => ({
            projects: state.projects.some((p) => p.id === project!.id)
              ? state.projects
              : [project!, ...state.projects],
          }));
        }
      }

      if (!project) {
        navigateToDashboard(replace);
        return;
      }

      if (segments.length >= 4 && segments[2] === 'workflows') {
        const workflowId = segments[3];
        await openWorkflow(project, workflowId, { replace });
        return;
      }

      openProject(project, { replace });
    };

    resolvePath(window.location.pathname, true).catch((error) => {
      logger.warn('Failed to resolve initial route', { component: 'App', action: 'resolvePath' }, error);
    });

    const popHandler = () => {
      resolvePath(window.location.pathname, true).catch((error) => {
        logger.warn('Failed to resolve popstate route', { component: 'App', action: 'handlePopState' }, error);
      });
    };

    window.addEventListener('popstate', popHandler);
    return () => window.removeEventListener('popstate', popHandler);
  }, [navigateToDashboard, openProject, openWorkflow]);

  // Dashboard View
  if (currentView === 'dashboard') {
    return (
      <div className="h-screen flex flex-col bg-flow-bg">
        <Dashboard 
          onProjectSelect={handleProjectSelect}
          onCreateProject={handleCreateProject}
        />
        
        {showProjectModal && (
          <ProjectModal
            onClose={() => setShowProjectModal(false)}
            onSuccess={handleProjectCreated}
          />
        )}
      </div>
    );
  }

  // Project Detail View
  if (currentView === 'project-detail' && currentProject) {
    return (
      <div className="h-screen flex flex-col bg-flow-bg">
        <ProjectDetail
          project={currentProject}
          onBack={handleBackToDashboard}
          onWorkflowSelect={handleWorkflowSelect}
          onCreateWorkflow={handleCreateWorkflow}
        />
        
        {showAIModal && (
          <AIPromptModal
            onClose={() => setShowAIModal(false)}
            folder={selectedFolder}
            projectId={currentProject.id}
            onSwitchToManual={handleSwitchToManualBuilder}
            onSuccess={handleWorkflowGenerated}
          />
        )}
      </div>
    );
  }

  // Project Workflow View (fallback)
  return (
    <ReactFlowProvider>
      <div className="h-screen flex flex-col bg-flow-bg">
        <Header 
          onNewWorkflow={() => setShowAIModal(true)}
          onBackToDashboard={currentView === 'project-workflow' ? handleBackToProjectDetail : handleBackToDashboard}
          currentProject={currentProject}
          currentWorkflow={selectedWorkflow}
          showBackToProject={currentView === 'project-workflow'}
        />
        
        <div className="flex-1 flex overflow-hidden">
          <Sidebar
            selectedFolder={selectedFolder}
            onFolderSelect={setSelectedFolder}
            projectId={currentProject?.id}
          />
          
          <div className="flex-1 flex">
            <div className="flex-1 flex flex-col">
              <WorkflowBuilder projectId={currentProject?.id} />
            </div>
            
            {currentExecution && (
              <div className="w-1/3 min-w-[400px] border-l border-gray-800">
                <ExecutionViewer execution={currentExecution} />
              </div>
            )}
          </div>
        </div>
        
        {showAIModal && (
          <AIPromptModal
            onClose={() => setShowAIModal(false)}
            folder={selectedFolder}
            projectId={currentProject?.id}
            onSwitchToManual={handleSwitchToManualBuilder}
            onSuccess={handleWorkflowGenerated}
          />
        )}
      </div>
    </ReactFlowProvider>
  );
}

export default App;
