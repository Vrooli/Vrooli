import { useState } from 'react';
import { ReactFlowProvider } from 'reactflow';
import WorkflowBuilder from './components/WorkflowBuilder';
import ExecutionViewer from './components/ExecutionViewer';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import AIPromptModal from './components/AIPromptModal';
import { useWorkflowStore } from './stores/workflowStore';
import { useExecutionStore } from './stores/executionStore';
import 'reactflow/dist/style.css';

function App() {
  const [showAIModal, setShowAIModal] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string>('/');
  const currentWorkflow = useWorkflowStore((state) => state.currentWorkflow);
  const currentExecution = useExecutionStore((state) => state.currentExecution);

  return (
    <ReactFlowProvider>
      <div className="h-screen flex flex-col bg-flow-bg">
        <Header onNewWorkflow={() => setShowAIModal(true)} />
        
        <div className="flex-1 flex overflow-hidden">
          <Sidebar
            selectedFolder={selectedFolder}
            onFolderSelect={setSelectedFolder}
          />
          
          <div className="flex-1 flex">
            <div className="flex-1 flex flex-col">
              <WorkflowBuilder />
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
          />
        )}
      </div>
    </ReactFlowProvider>
  );
}

export default App;