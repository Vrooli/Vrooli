import { useState } from 'react';
import { ReactFlowProvider } from 'reactflow';
import WorkflowBuilder from './components/WorkflowBuilder';
import ExecutionViewer from './components/ExecutionViewer';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import AIPromptModal from './components/AIPromptModal';
import Dashboard from './components/Dashboard';
import ProjectModal from './components/ProjectModal';
import { useExecutionStore } from './stores/executionStore';
import { useProjectStore, Project } from './stores/projectStore';
import 'reactflow/dist/style.css';

type AppView = 'dashboard' | 'project';

function App() {
  const [currentView, setCurrentView] = useState<AppView>('dashboard');
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string>('/');
  
  const { currentProject, setCurrentProject } = useProjectStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);

  const handleProjectSelect = (project: Project) => {
    setCurrentProject(project);
    setSelectedFolder(project.folder_path);
    setCurrentView('project');
  };

  const handleBackToDashboard = () => {
    setCurrentProject(null);
    setCurrentView('dashboard');
  };

  const handleCreateProject = () => {
    setShowProjectModal(true);
  };

  const handleProjectCreated = (project: Project) => {
    // Automatically select the newly created project
    handleProjectSelect(project);
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

  // Project Workspace View
  return (
    <ReactFlowProvider>
      <div className="h-screen flex flex-col bg-flow-bg">
        <Header 
          onNewWorkflow={() => setShowAIModal(true)}
          onBackToDashboard={handleBackToDashboard}
          currentProject={currentProject}
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
          />
        )}
      </div>
    </ReactFlowProvider>
  );
}

export default App;