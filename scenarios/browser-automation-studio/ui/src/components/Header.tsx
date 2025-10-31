import {
  Play,
  Save,
  Bug,
  ArrowLeft,
  Edit2,
  Check,
  X,
  Info,
  History,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  Clock,
} from 'lucide-react';
import { useState, useRef, useEffect } from 'react';
import { formatDistanceToNow } from 'date-fns';
import { useWorkflowStore } from '../stores/workflowStore';
import { useExecutionStore } from '../stores/executionStore';
import { Project, useProjectStore } from '../stores/projectStore';
import AIEditModal from './AIEditModal';
import toast from 'react-hot-toast';
import { usePopoverPosition } from '../hooks/usePopoverPosition';
import ResponsiveDialog from './ResponsiveDialog';

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

function Header({ onNewWorkflow: _onNewWorkflow, onBackToDashboard, currentProject, currentWorkflow: selectedWorkflow, showBackToProject }: HeaderProps) {
  const currentWorkflow = useWorkflowStore((state) => state.currentWorkflow);
  const saveWorkflow = useWorkflowStore((state) => state.saveWorkflow);
  const updateWorkflow = useWorkflowStore((state) => state.updateWorkflow);
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);
  const isDirty = useWorkflowStore((state) => state.isDirty);
  const isSaving = useWorkflowStore((state) => state.isSaving);
  const lastSavedAt = useWorkflowStore((state) => state.lastSavedAt);
  const lastSaveError = useWorkflowStore((state) => state.lastSaveError);
  const hasVersionConflict = useWorkflowStore((state) => state.hasVersionConflict);
  const conflictWorkflow = useWorkflowStore((state) => state.conflictWorkflow);
  const conflictMetadata = useWorkflowStore((state) => state.conflictMetadata);
  const refreshConflictWorkflow = useWorkflowStore((state) => state.refreshConflictWorkflow);
  const resolveConflictWithReload = useWorkflowStore((state) => state.resolveConflictWithReload);
  const forceSaveWorkflow = useWorkflowStore((state) => state.forceSaveWorkflow);
  const acknowledgeSaveError = useWorkflowStore((state) => state.acknowledgeSaveError);
  const loadWorkflowVersions = useWorkflowStore((state) => state.loadWorkflowVersions);
  const versionHistory = useWorkflowStore((state) => state.versionHistory);
  const isVersionHistoryLoading = useWorkflowStore((state) => state.isVersionHistoryLoading);
  const versionHistoryError = useWorkflowStore((state) => state.versionHistoryError);
  const restoreWorkflowVersion = useWorkflowStore((state) => state.restoreWorkflowVersion);
  const restoringVersion = useWorkflowStore((state) => state.restoringVersion);
  const { startExecution } = useExecutionStore();
  const { isConnected, error } = useProjectStore();
  const [showAIEditModal, setShowAIEditModal] = useState(false);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [editTitle, setEditTitle] = useState('');
  const titleInputRef = useRef<HTMLInputElement>(null);
  const infoButtonRef = useRef<HTMLButtonElement | null>(null);
  const infoPopoverRef = useRef<HTMLDivElement | null>(null);
  const [showWorkflowInfo, setShowWorkflowInfo] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);
  const [showConflictDetails, setShowConflictDetails] = useState(false);

  const { floatingStyles: workflowInfoStyles } = usePopoverPosition(infoButtonRef, infoPopoverRef, {
    isOpen: showWorkflowInfo,
    placementPriority: ['bottom-end', 'bottom-start', 'top-end', 'top-start'],
  });

  useEffect(() => {
    if (hasVersionConflict) {
      if (!conflictMetadata && currentWorkflow) {
        refreshConflictWorkflow().catch(() => {});
      }
    } else if (showConflictDetails) {
      setShowConflictDetails(false);
    }
  }, [hasVersionConflict, conflictMetadata, currentWorkflow?.id, refreshConflictWorkflow, showConflictDetails]);
  
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

  const openVersionHistory = async () => {
    if (!currentWorkflow) {
      toast.error('No workflow loaded');
      return;
    }
    setShowVersionHistory(true);
    try {
      await loadWorkflowVersions(currentWorkflow.id);
    } catch (error) {
      toast.error('Failed to load version history');
    }
  };

  const handleHistoryButtonClick = async () => {
    if (!currentWorkflow) {
      toast.error('No workflow loaded');
      return;
    }

    if (isDirty) {
      try {
        await saveWorkflow({ source: 'manual', changeDescription: 'Manual save before viewing history' });
        toast.success('Workflow saved successfully');
      } catch (error) {
        toast.error('Failed to save workflow');
        return;
      }
    }

    await openVersionHistory();
  };

  const handleRestoreVersion = async (versionNumber: number) => {
    if (!currentWorkflow) {
      toast.error('No workflow loaded');
      return;
    }
    const confirmed = window.confirm(`Restore workflow to version ${versionNumber}? This will create a new revision.`);
    if (!confirmed) {
      return;
    }
    try {
      await restoreWorkflowVersion(currentWorkflow.id, versionNumber, `Restored from version ${versionNumber}`);
      toast.success(`Workflow restored to version ${versionNumber}`);
    } catch (error) {
      toast.error('Failed to restore workflow version');
    }
  };

  const handleReloadWorkflow = async () => {
    if (!currentWorkflow) {
      return;
    }
    try {
      if (hasVersionConflict) {
        await resolveConflictWithReload();
      } else {
        await loadWorkflow(currentWorkflow.id);
      }
      setShowConflictDetails(false);
      toast.success('Workflow reloaded');
    } catch (error) {
      toast.error('Failed to reload workflow');
    }
  };

  const handleForceSave = async () => {
    if (!currentWorkflow) {
      return;
    }
    try {
      await forceSaveWorkflow({ changeDescription: 'Force save after conflict', source: 'manual-force-save' });
      setShowConflictDetails(false);
      toast.success('Local changes saved');
    } catch (error) {
      toast.error('Failed to force save workflow');
    }
  };

  const handleRefreshConflict = async () => {
    try {
      await refreshConflictWorkflow();
      toast.success('Conflict snapshot refreshed');
    } catch (error) {
      toast.error('Failed to refresh conflict snapshot');
    }
  };

  const lastSavedLabel = lastSavedAt ? formatDistanceToNow(lastSavedAt, { addSuffix: true }) : null;
  const baseIndicatorClass =
    'flex flex-wrap items-center gap-2 rounded-lg border px-3 py-1.5 text-xs shadow-sm backdrop-blur-sm';
  const historyButtonDisabled = !currentWorkflow || isSaving || restoringVersion !== null;

  let saveStatusNode: React.ReactNode;
  if (!currentWorkflow) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-gray-800 bg-gray-900/70 text-gray-400`}>
        <Info size={14} className="text-gray-400" />
        <span>No workflow loaded</span>
      </div>
    );
  } else if (restoringVersion !== null) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-blue-500/40 bg-blue-500/10 text-blue-100`}>
        <Loader2 size={14} className="animate-spin" />
        <span>Restoring version {restoringVersion}…</span>
      </div>
    );
  } else if (hasVersionConflict) {
    const remoteVersionLabel = conflictMetadata ? `v${conflictMetadata.remoteVersion}` : 'server version';
    const remoteSavedLabel = conflictMetadata ? formatDistanceToNow(conflictMetadata.remoteUpdatedAt, { addSuffix: true }) : null;
    const remoteSourceLabel = conflictMetadata?.changeSource ?? conflictWorkflow?.lastChangeSource ?? 'unknown source';
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-amber-500/50 bg-amber-500/10 text-amber-100`}
      >
        <AlertTriangle size={14} className="text-amber-200" />
        <span className="font-medium">
          Version conflict with {remoteVersionLabel}
          {remoteSavedLabel ? ` · saved ${remoteSavedLabel}` : ''}
          {remoteSourceLabel ? ` · ${remoteSourceLabel}` : ''}
        </span>
        <div className="flex flex-wrap items-center gap-1 sm:gap-2">
          <button
            type="button"
            onClick={() => setShowConflictDetails(true)}
            className="rounded border border-amber-400/40 bg-amber-500/10 px-2 py-1 text-amber-100 hover:bg-amber-500/20"
          >
            Details
          </button>
          <button
            type="button"
            onClick={handleRefreshConflict}
            className="rounded border border-amber-400/40 bg-amber-500/10 px-2 py-1 text-amber-100 hover:bg-amber-500/20"
          >
            Refresh
          </button>
          <button
            type="button"
            onClick={handleReloadWorkflow}
            className="rounded border border-amber-400/40 bg-amber-500/10 px-2 py-1 text-amber-100 hover:bg-amber-500/20"
          >
            Reload
          </button>
          <button
            type="button"
            onClick={openVersionHistory}
            className="rounded border border-amber-400/40 bg-amber-500/10 px-2 py-1 text-amber-100 hover:bg-amber-500/20"
          >
            History
          </button>
          <button
            type="button"
            onClick={handleForceSave}
            className="rounded border border-amber-400/40 bg-amber-500/20 px-2 py-1 font-medium text-amber-50 hover:bg-amber-500/30"
          >
            Force save
          </button>
        </div>
      </div>
    );
  } else if (lastSaveError) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-rose-500/50 bg-rose-500/10 text-rose-100`}>
        <AlertTriangle size={14} className="text-rose-200" />
        <span className="font-medium">{lastSaveError.message}</span>
        <button
          type="button"
          onClick={acknowledgeSaveError}
          className="rounded border border-rose-400/50 bg-rose-500/20 px-2 py-1 text-rose-100 hover:bg-rose-500/30"
        >
          Dismiss
        </button>
        <button
          type="button"
          onClick={openVersionHistory}
          disabled={historyButtonDisabled}
          className="rounded border border-rose-400/50 bg-rose-500/20 px-2 py-1 text-rose-100 transition-colors hover:bg-rose-500/30 focus:outline-none focus:ring-2 focus:ring-rose-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
        >
          History
        </button>
      </div>
    );
  } else if (isSaving) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-blue-500/40 bg-blue-500/10 text-blue-100`}>
        <Loader2 size={14} className="animate-spin" />
        <span>Saving…</span>
      </div>
    );
  } else if (isDirty) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-yellow-500/50 bg-yellow-500/10 text-yellow-100`}>
        <span className="font-medium">Unsaved changes</span>
        <button
          type="button"
          onClick={handleHistoryButtonClick}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-yellow-400/60 bg-yellow-500/20 text-yellow-50 transition-colors hover:bg-yellow-500/30 focus:outline-none focus:ring-2 focus:ring-yellow-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Save now and open history"
          aria-label="Save now and open history"
        >
          <Save size={14} />
        </button>
      </div>
    );
  } else if (lastSavedLabel) {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-emerald-500/40 bg-emerald-500/10 text-emerald-100`}>
        <CheckCircle2 size={14} className="text-emerald-200" />
        <span>Saved {lastSavedLabel}</span>
        <button
          type="button"
          onClick={handleHistoryButtonClick}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-emerald-400/60 bg-emerald-500/20 text-emerald-50 transition-colors hover:bg-emerald-500/30 focus:outline-none focus:ring-2 focus:ring-emerald-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Open version history"
          aria-label="Open version history"
        >
          <History size={14} />
        </button>
      </div>
    );
  } else {
    saveStatusNode = (
      <div className={`${baseIndicatorClass} border-gray-700 bg-gray-900/80 text-gray-300`}>
        <Clock size={14} className="text-gray-400" />
        <span>Not saved yet</span>
        <button
          type="button"
          onClick={handleHistoryButtonClick}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-gray-600 bg-gray-800/80 text-gray-200 transition-colors hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-400/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Open version history"
          aria-label="Open version history"
        >
          <History size={14} />
        </button>
      </div>
    );
  }

  const handleExecute = async () => {
    if (!currentWorkflow) {
      toast.error('No workflow to execute');
      return;
    }
    try {
      // Auto-save workflow before execution to ensure latest changes are used
      await startExecution(currentWorkflow.id, () =>
        saveWorkflow({ source: 'execution-run', changeDescription: 'Autosave before execution' })
      );
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

  useEffect(() => {
    if (!currentWorkflow) {
      setShowVersionHistory(false);
    }
  }, [currentWorkflow?.id]);

  return (
    <>
      <header className="bg-flow-node border-b border-gray-800 px-4 py-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
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
                      style={workflowInfoStyles}
                      className="z-30 w-80 max-h-96 overflow-y-auto rounded-lg border border-gray-700 bg-flow-node p-4 shadow-lg"
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
            
          <div className="flex flex-col sm:flex-row sm:items-center sm:gap-3">
            <div className="mt-1 sm:mt-0 flex items-center gap-2">
              {saveStatusNode}
            </div>
          </div>
          </div>
          
          <div className="flex items-center gap-1 sm:gap-2">
            <button
              onClick={handleDebug}
              className="toolbar-button flex items-center gap-0 sm:gap-2"
              title="Edit with AI"
              aria-label="Edit with AI"
            >
              <Bug size={16} />
              <span className="hidden text-sm sm:inline">Debug</span>
            </button>

            <button
              onClick={handleExecute}
              className="bg-flow-accent hover:bg-blue-600 text-white px-3 sm:px-4 py-2 rounded-lg flex items-center gap-0 sm:gap-2 transition-colors"
              title="Execute Workflow"
              aria-label="Execute Workflow"
            >
              <Play size={16} />
              <span className="hidden text-sm font-medium sm:inline">Execute</span>
            </button>
          </div>
        </div>
      </header>
      
      {showAIEditModal && (
      <AIEditModal
        onClose={() => setShowAIEditModal(false)}
      />
    )}

      <ResponsiveDialog
        isOpen={showConflictDetails && hasVersionConflict}
        onDismiss={() => setShowConflictDetails(false)}
        ariaLabel="Resolve Version Conflict"
        size="wide"
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Resolve Version Conflict</h2>
          <button
            type="button"
            onClick={() => setShowConflictDetails(false)}
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
          >
            <X size={16} />
          </button>
        </div>
        <p className="text-sm text-gray-300 mb-4">
          The server copy changed before your latest save. Review the differences and choose whether to reload the
          remote version or keep your local draft.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="rounded-lg border border-gray-800 bg-gray-900/70 p-4">
            <h3 className="text-sm font-semibold text-white">Local Draft</h3>
            <ul className="mt-2 space-y-1 text-xs text-gray-400">
              <li>Version {currentWorkflow?.version ?? '—'}</li>
              <li>Nodes: {displayWorkflow?.nodes?.length ?? 0}</li>
              <li>Edges: {displayWorkflow?.edges?.length ?? 0}</li>
              {currentWorkflow?.lastChangeDescription && (
                <li>Last change: {currentWorkflow.lastChangeDescription}</li>
              )}
            </ul>
          </div>
          <div className="rounded-lg border border-amber-500/40 bg-gray-900/70 p-4">
            <h3 className="text-sm font-semibold text-amber-100">Remote {conflictMetadata ? `v${conflictMetadata.remoteVersion}` : ''}</h3>
            <ul className="mt-2 space-y-1 text-xs text-amber-200">
              <li>
                Saved {conflictMetadata ? formatDistanceToNow(conflictMetadata.remoteUpdatedAt, { addSuffix: true }) : 'recently'}
              </li>
              <li>Source: {conflictMetadata?.changeSource ?? conflictWorkflow?.lastChangeSource ?? 'unknown'}</li>
              <li>Nodes: {conflictMetadata?.nodeCount ?? conflictWorkflow?.nodes?.length ?? 0}</li>
              <li>Edges: {conflictMetadata?.edgeCount ?? conflictWorkflow?.edges?.length ?? 0}</li>
              {conflictMetadata?.changeDescription && (
                <li>Change: {conflictMetadata.changeDescription}</li>
              )}
            </ul>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => setShowConflictDetails(false)}
            className="rounded border border-gray-700 px-3 py-1.5 text-xs text-gray-200 hover:bg-gray-800"
          >
            Close
          </button>
          <button
            type="button"
            onClick={handleReloadWorkflow}
            className="rounded border border-amber-500/60 px-3 py-1.5 text-xs text-amber-100 hover:bg-amber-500/20"
          >
            Reload remote
          </button>
          <button
            type="button"
            onClick={handleForceSave}
            className="rounded bg-amber-500/20 px-3 py-1.5 text-xs text-amber-50 hover:bg-amber-500/30"
          >
            Force save local
          </button>
        </div>
      </ResponsiveDialog>

      <ResponsiveDialog
        isOpen={showVersionHistory}
        onDismiss={() => setShowVersionHistory(false)}
        ariaLabel="Workflow Versions"
        size="wide"
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Workflow Versions</h2>
          <button
            type="button"
            onClick={() => setShowVersionHistory(false)}
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        {!currentWorkflow ? (
          <div className="text-sm text-gray-400">Load a workflow to view version history.</div>
        ) : isVersionHistoryLoading ? (
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <Loader2 size={16} className="animate-spin" />
            Loading version history…
          </div>
        ) : versionHistory.length === 0 ? (
          <div className="text-sm text-gray-400">No saved versions yet. Save a workflow to create history.</div>
        ) : (
          <div className="space-y-3">
            {versionHistory.map((version) => {
              const isCurrent = currentWorkflow?.version === version.version;
              const createdLabel = formatDistanceToNow(version.createdAt, { addSuffix: true });
              return (
                <div
                  key={version.version}
                  className={`rounded-lg border p-4 bg-gray-900/70 ${
                    isCurrent ? 'border-blue-500/60 shadow-inner shadow-blue-500/20' : 'border-gray-800'
                  }`}
                >
                  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2 text-sm font-semibold text-white">
                        Version {version.version}
                        {isCurrent && <span className="text-xs text-blue-300 uppercase tracking-wide">Current</span>}
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        Saved {createdLabel} by {version.createdBy || 'Unknown'}
                      </div>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gray-400">
                      <span>Nodes: {version.nodeCount}</span>
                      <span>Edges: {version.edgeCount}</span>
                      <span className="hidden sm:inline text-gray-500">Hash</span>
                      <span className="font-mono text-gray-500 sm:text-xs break-all">
                        {version.definitionHash ? version.definitionHash.slice(0, 12) : '—'}
                      </span>
                    </div>
                  </div>
                  {version.changeDescription && (
                    <div className="mt-3 text-sm text-gray-300">{version.changeDescription}</div>
                  )}
                  <div className="mt-3 flex items-center justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => handleRestoreVersion(version.version)}
                      disabled={isCurrent || restoringVersion === version.version}
                      className="px-3 py-1.5 text-xs bg-gray-800 hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                      {restoringVersion === version.version ? (
                        <Loader2 size={14} className="animate-spin" />
                      ) : (
                        <History size={14} />
                      )}
                      Restore
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {versionHistoryError && (
          <div className="mt-4 text-xs text-rose-300">
            {versionHistoryError}
          </div>
        )}
      </ResponsiveDialog>
    </>
  );
}

export default Header;
