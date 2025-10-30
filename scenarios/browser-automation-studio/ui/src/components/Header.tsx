import { Plus, Play, Save, FolderOpen, Bug, ArrowLeft, Edit2, Check, X, Info } from 'lucide-react';
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
  const infoButtonRef = useRef<HTMLButtonElement | null>(null);
  const infoPopoverRef = useRef<HTMLDivElement | null>(null);
  const [showWorkflowInfo, setShowWorkflowInfo] = useState(false);
  
  // Use the workflow from props if provided, otherwise use the one from store
  const displayWorkflow = selectedWorkflow || currentWorkflow;

  const formatDetailDate = (value?: Date | string) => {
    if (!value) {
      return 'Not available';
    }
    const date = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(date.getTime())) {
      return 'Not available';
    }
    return date.toLocaleString();
  };

  const workflowDescription = displayWorkflow?.description?.trim();

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

  useEffect(() => {
    if (!showWorkflowInfo) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        infoPopoverRef.current &&
        !infoPopoverRef.current.contains(target) &&
        infoButtonRef.current &&
        !infoButtonRef.current.contains(target)
      ) {
        setShowWorkflowInfo(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setShowWorkflowInfo(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [showWorkflowInfo]);

  useEffect(() => {
    setShowWorkflowInfo(false);
  }, [displayWorkflow?.id, currentProject?.id]);

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
                <div className="relative">
                  <button
                    ref={infoButtonRef}
                    type="button"
                    onClick={() => setShowWorkflowInfo((prev) => !prev)}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
                    title="Workflow details"
                    aria-label="Workflow details"
                    aria-expanded={showWorkflowInfo}
                  >
                    <Info size={16} />
                  </button>
                  {showWorkflowInfo && (
                    <div
                      ref={infoPopoverRef}
                      className="absolute left-0 sm:right-0 sm:left-auto z-30 mt-2 w-80 max-h-96 overflow-y-auto rounded-lg border border-gray-700 bg-flow-node p-4 shadow-lg"
                    >
                      <div className="flex items-center justify-between mb-3">
                        <h3 className="text-sm font-semibold text-white">Workflow Details</h3>
                        <button
                          type="button"
                          onClick={() => setShowWorkflowInfo(false)}
                          className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
                          aria-label="Close workflow details"
                        >
                          <X size={14} />
                        </button>
                      </div>

                      {displayWorkflow && (
                        <div className="mb-4 pb-4 border-b border-gray-700">
                          <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-2">Workflow</h4>
                          <dl className="space-y-2 text-sm text-gray-300">
                            <div>
                              <dt className="text-xs text-gray-500">Name</dt>
                              <dd className="text-sm font-medium text-white">{displayWorkflow.name}</dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">Description</dt>
                              <dd className="text-sm text-gray-300">
                                {workflowDescription || 'No description provided'}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">Folder</dt>
                              <dd className="text-sm text-gray-300 font-mono break-all">{displayWorkflow.folderPath || '/'}</dd>
                            </div>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                              <div>
                                <dt className="text-xs text-gray-500">Created</dt>
                                <dd className="text-xs text-gray-300">{formatDetailDate(displayWorkflow.createdAt)}</dd>
                              </div>
                              <div>
                                <dt className="text-xs text-gray-500">Last Updated</dt>
                                <dd className="text-xs text-gray-300">{formatDetailDate(displayWorkflow.updatedAt)}</dd>
                              </div>
                            </div>
                          </dl>
                        </div>
                      )}

                      {currentProject && (
                        <div className="space-y-2 text-sm text-gray-300">
                          <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Project</h4>
                          <dl className="space-y-2">
                            <div>
                              <dt className="text-xs text-gray-500">Name</dt>
                              <dd className="text-sm font-medium text-white">{currentProject.name}</dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">Description</dt>
                              <dd className="text-sm text-gray-300">
                                {currentProject.description?.trim() || 'No description provided'}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">Save Path</dt>
                              <dd className="text-sm text-gray-300 font-mono break-all">{currentProject.folder_path}</dd>
                            </div>
                          </dl>
                        </div>
                      )}

                      <div className="mt-4 pt-4 border-t border-gray-800 space-y-2 text-sm text-gray-300">
                        <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500">Connection</h4>
                        <p className={isConnected ? 'text-xs text-green-300' : 'text-xs text-rose-300'}>
                          {isConnected ? 'Connected to project services' : error ? `Disconnected: ${error}` : 'Disconnected from project services'}
                        </p>
                      </div>
                    </div>
                  )}
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
