import { Plus, Play, Save, FolderOpen, Brain, Bug, ArrowLeft, Wifi, WifiOff, Edit2, Check, X } from 'lucide-react';
import { useState, useRef, useEffect } from 'react';
import { useWorkflowStore } from '../stores/workflowStore';
import { useExecutionStore } from '../stores/executionStore';
import { Project, useProjectStore } from '../stores/projectStore';
import AIEditModal from './AIEditModal';
import toast from 'react-hot-toast';

interface Workflow {
  id: string;
  name: string;
  description?: string;
  folderPath: string;
  createdAt: Date;
  updatedAt: Date;
  projectId?: string;
}

interface HeaderProps {
  onNewWorkflow: () => void;
  onBackToDashboard?: () => void;
  currentProject?: Project | null;
  currentWorkflow?: Workflow | null;
  showBackToProject?: boolean;
}

function Header({ onNewWorkflow, onBackToDashboard, currentProject, currentWorkflow: selectedWorkflow, showBackToProject }: HeaderProps) {
  const { currentWorkflow, saveWorkflow, updateWorkflow } = useWorkflowStore();
  const { startExecution } = useExecutionStore();
  const { isConnected, error } = useProjectStore();
  const [showAIEditModal, setShowAIEditModal] = useState(false);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [editTitle, setEditTitle] = useState('');
  const titleInputRef = useRef<HTMLInputElement>(null);
  
  // Use the workflow from props if provided, otherwise use the one from store
  const displayWorkflow = selectedWorkflow || currentWorkflow;

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
    if (!currentWorkflow) {
      toast.error('No workflow loaded to edit');
      return;
    }
    setShowAIEditModal(true);
  };

  const handleStartEditTitle = () => {
    if (!displayWorkflow) return;
    setEditTitle(displayWorkflow.name);
    setIsEditingTitle(true);
  };

  const handleSaveTitle = async () => {
    if (!displayWorkflow || !editTitle.trim()) {
      setIsEditingTitle(false);
      return;
    }
    
    try {
      await updateWorkflow({ ...displayWorkflow, name: editTitle.trim() });
      setIsEditingTitle(false);
      toast.success('Workflow title updated');
    } catch (error) {
      toast.error('Failed to update workflow title');
    }
  };

  const handleCancelEditTitle = () => {
    setIsEditingTitle(false);
    setEditTitle('');
  };

  const handleTitleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveTitle();
    } else if (e.key === 'Escape') {
      handleCancelEditTitle();
    }
  };

  // Focus input when editing starts
  useEffect(() => {
    if (isEditingTitle && titleInputRef.current) {
      titleInputRef.current.focus();
      titleInputRef.current.select();
    }
  }, [isEditingTitle]);

  return (
    <>
      <header className="bg-flow-node border-b border-gray-800 px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              {onBackToDashboard && (
                <button
                  onClick={onBackToDashboard}
                  className="toolbar-button flex items-center gap-2"
                  title={showBackToProject ? "Back to Project" : "Back to Dashboard"}
                >
                  <ArrowLeft size={16} />
                </button>
              )}
              
              <h1 className="text-xl font-bold text-white flex items-center gap-2">
                {displayWorkflow ? (
                  <div className="flex items-center gap-2">
                    {isEditingTitle ? (
                      <div className="flex items-center gap-2">
                        <input
                          ref={titleInputRef}
                          type="text"
                          value={editTitle}
                          onChange={(e) => setEditTitle(e.target.value)}
                          onKeyDown={handleTitleKeyDown}
                          onBlur={handleSaveTitle}
                          className="bg-gray-800 text-white px-2 py-1 rounded border border-gray-600 focus:border-flow-accent focus:outline-none"
                          placeholder="Workflow name..."
                        />
                        <button
                          onClick={handleSaveTitle}
                          className="text-green-400 hover:text-green-300 p-1"
                          title="Save title"
                        >
                          <Check size={14} />
                        </button>
                        <button
                          onClick={handleCancelEditTitle}
                          className="text-red-400 hover:text-red-300 p-1"
                          title="Cancel editing"
                        >
                          <X size={14} />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 group">
                        <span className="cursor-pointer" onClick={handleStartEditTitle}>
                          {displayWorkflow.name}
                        </span>
                        <button
                          onClick={handleStartEditTitle}
                          className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-white transition-opacity p-1"
                          title="Edit workflow name"
                        >
                          <Edit2 size={14} />
                        </button>
                      </div>
                    )}
                  </div>
                ) : (currentProject ? currentProject.name : 'Browser Automation Studio')}
              </h1>
              
              {(currentProject || displayWorkflow) && (
                <div className="text-sm text-gray-400">
                  {displayWorkflow ? (
                    <>
                      <span className="mr-2">• {currentProject?.name}</span>
                      <span className="text-xs">{displayWorkflow.folderPath}</span>
                    </>
                  ) : currentProject ? (
                    <>
                      {currentProject.description && (
                        <span className="mr-2">• {currentProject.description}</span>
                      )}
                      <span className="text-xs">{currentProject.folder_path}</span>
                    </>
                  ) : null}
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
              title="Edit with AI"
            >
              <Bug size={16} />
              <span className="text-sm">Debug</span>
            </button>
          </div>
        </div>
      </header>
      
      {showAIEditModal && (
        <AIEditModal
          onClose={() => setShowAIEditModal(false)}
        />
      )}
    </>
  );
}

export default Header;