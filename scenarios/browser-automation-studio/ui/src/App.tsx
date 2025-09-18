import { useState } from 'react';
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

  const handleProjectSelect = (project: Project) => {
    setCurrentProject(project);
    setSelectedFolder(project.folder_path);
    setSelectedWorkflow(null);
    setCurrentView('project-detail');
  };

  const handleWorkflowSelect = async (workflow: any) => {
    // Convert to our format
    const convertedWorkflow = {
      ...workflow,
      folderPath: workflow.folder_path,
      createdAt: new Date(workflow.created_at),
      updatedAt: new Date(workflow.updated_at),
      projectId: workflow.project_id,
    };
    setSelectedWorkflow(convertedWorkflow);
    setSelectedFolder(workflow.folder_path);
    // Load the workflow into the workflow store
    await loadWorkflow(workflow.id);
    setCurrentView('project-workflow');
  };

  const handleBackToDashboard = () => {
    setCurrentProject(null);
    setSelectedWorkflow(null);
    setCurrentView('dashboard');
  };

  const handleBackToProjectDetail = () => {
    setSelectedWorkflow(null);
    setCurrentView('project-detail');
  };

  const handleCreateProject = () => {
    setShowProjectModal(true);
  };

  const handleProjectCreated = (project: Project) => {
    // Automatically select the newly created project
    handleProjectSelect(project);
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
      await useWorkflowStore.getState().createWorkflow(
        workflowName, 
        folderPath,
        currentProject?.id
      );
      setCurrentView('project-workflow');
    } catch (error) {
      console.error('Failed to create workflow:', error);
    }
  };

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
          />
        )}
      </div>
    </ReactFlowProvider>
  );
}

export default App;