import { Plus, Play, Save, FolderOpen, Brain, Bug, ArrowLeft, Wifi, WifiOff } from 'lucide-react';
import { useWorkflowStore } from '../stores/workflowStore';
import { useExecutionStore } from '../stores/executionStore';
import { Project, useProjectStore } from '../stores/projectStore';
import toast from 'react-hot-toast';

interface HeaderProps {
  onNewWorkflow: () => void;
  onBackToDashboard?: () => void;
  currentProject?: Project | null;
}

function Header({ onNewWorkflow, onBackToDashboard, currentProject }: HeaderProps) {
  const { currentWorkflow, saveWorkflow } = useWorkflowStore();
  const { startExecution } = useExecutionStore();
  const { isConnected, error } = useProjectStore();

  const handleSave = async () => {
    if (!currentWorkflow) {
      toast.error('No workflow to save');
      return;
    }
    try {
      await saveWorkflow();
      toast.success('Workflow saved successfully');
    } catch (error) {
      toast.error('Failed to save workflow');
    }
  };

  const handleExecute = async () => {
    if (!currentWorkflow) {
      toast.error('No workflow to execute');
      return;
    }
    try {
      await startExecution(currentWorkflow.id);
      toast.success('Workflow execution started');
    } catch (error) {
      toast.error('Failed to start execution');
    }
  };

  const handleDebug = () => {
    toast('Claude Code debugging coming soon!', { icon: '🤖' });
  };

  return (
    <header className="bg-flow-node border-b border-gray-800 px-4 py-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            {onBackToDashboard && (
              <button
                onClick={onBackToDashboard}
                className="toolbar-button flex items-center gap-2"
                title="Back to Dashboard"
              >
                <ArrowLeft size={16} />
              </button>
            )}
            
            <h1 className="text-xl font-bold text-white flex items-center gap-2">
              <div className="w-8 h-8 bg-flow-accent rounded-lg flex items-center justify-center">
                <Brain size={20} />
              </div>
              {currentProject ? currentProject.name : 'Browser Automation Studio'}
            </h1>
            
            {currentProject && (
              <div className="text-sm text-gray-400">
                {currentProject.description && (
                  <span className="mr-2">• {currentProject.description}</span>
                )}
                <span className="text-xs">{currentProject.folder_path}</span>
              </div>
            )}
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={onNewWorkflow}
              className="toolbar-button flex items-center gap-2"
              title="New Workflow"
            >
              <Plus size={16} />
              <span className="text-sm">New</span>
            </button>
            
            <button
              onClick={handleSave}
              className="toolbar-button flex items-center gap-2"
              title="Save Workflow"
            >
              <Save size={16} />
              <span className="text-sm">Save</span>
            </button>
            
            <button
              className="toolbar-button flex items-center gap-2"
              title="Open Workflow"
            >
              <FolderOpen size={16} />
              <span className="text-sm">Open</span>
            </button>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          {/* API Status Indicator */}
          <div className="flex items-center gap-2 px-3 py-1 rounded-lg bg-flow-node border border-gray-700">
            {isConnected ? (
              <>
                <Wifi size={14} className="text-green-400" />
                <span className="text-xs text-green-400">Connected</span>
              </>
            ) : (
              <>
                <WifiOff size={14} className="text-red-400" />
                <span className="text-xs text-red-400" title={error || 'Connection failed'}>
                  Disconnected
                </span>
              </>
            )}
          </div>
          
          <button
            onClick={handleExecute}
            className="bg-flow-accent hover:bg-blue-600 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
            title="Execute Workflow"
          >
            <Play size={16} />
            <span className="text-sm font-medium">Execute</span>
          </button>
          
          <button
            onClick={handleDebug}
            className="toolbar-button flex items-center gap-2"
            title="Debug with Claude"
          >
            <Bug size={16} />
            <span className="text-sm">Debug</span>
          </button>
        </div>
      </div>
    </header>
  );
}

export default Header;