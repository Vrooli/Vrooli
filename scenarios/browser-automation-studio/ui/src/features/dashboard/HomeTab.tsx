import React, { useState, useCallback, useMemo } from 'react';
import {
  Sparkles,
  Plus,
  Play,
  Star,
  StarOff,
  Pencil,
  Key,
  CreditCard,
  Loader2,
  FolderOpen,
  Zap,
  Bot,
  Eye,
  Square,
  Activity,
} from 'lucide-react';
import { useDashboardStore, type RecentWorkflow, type FavoriteWorkflow } from '@stores/dashboardStore';
import { useExecutionStore } from '@stores/executionStore';
import { useAICapability } from '@stores/aiCapabilityStore';
import { formatDistanceToNow } from 'date-fns';
import { TemplatesGallery } from './TemplatesGallery';

interface HomeTabProps {
  onAIGenerate: (prompt: string) => void;
  onCreateManual: () => void;
  onNavigateToWorkflow: (projectId: string, workflowId: string) => void;
  onRunWorkflow: (workflowId: string) => void;
  onViewExecution: (executionId: string, workflowId: string) => void;
  onOpenSettings: () => void;
  onUseTemplate: (prompt: string, templateName: string) => void;
  isGenerating?: boolean;
}

export const HomeTab: React.FC<HomeTabProps> = ({
  onAIGenerate,
  onCreateManual,
  onNavigateToWorkflow,
  onRunWorkflow,
  onViewExecution,
  onOpenSettings,
  onUseTemplate,
  isGenerating = false,
}) => {
  const [prompt, setPrompt] = useState('');
  const {
    recentWorkflows,
    favoriteWorkflows,
    lastEditedWorkflow,
    runningExecutions,
    addFavorite,
    removeFavorite,
    isFavorite,
    fetchRunningExecutions,
  } = useDashboardStore();
  const { stopExecution } = useExecutionStore();
  const aiCapability = useAICapability();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (prompt.trim() && !isGenerating && aiCapability.available) {
      onAIGenerate(prompt.trim());
    }
  };

  const handleToggleFavorite = useCallback((workflow: RecentWorkflow) => {
    if (isFavorite(workflow.id)) {
      removeFavorite(workflow.id);
    } else {
      const favorite: FavoriteWorkflow = {
        id: workflow.id,
        name: workflow.name,
        projectId: workflow.projectId,
        projectName: workflow.projectName,
        addedAt: new Date(),
      };
      addFavorite(favorite);
    }
  }, [addFavorite, removeFavorite, isFavorite]);

  const examplePrompts = [
    'Log into my dashboard and take a screenshot',
    'Fill out a contact form and submit it',
    'Scrape product prices from a page',
    'Check if a website is loading correctly',
  ];

  const handleExampleClick = (example: string) => {
    setPrompt(example);
  };

  const handleStopExecution = useCallback(async (executionId: string) => {
    try {
      await stopExecution(executionId);
      await fetchRunningExecutions();
    } catch (error) {
      console.error('Failed to stop execution:', error);
    }
  }, [stopExecution, fetchRunningExecutions]);

  // Build unified workflow list: last edited → starred → other recent
  const unifiedWorkflows = useMemo(() => {
    const seen = new Set<string>();
    const result: Array<RecentWorkflow & { isLastEdited?: boolean; isStarred?: boolean }> = [];

    // 1. Last edited workflow first (if exists)
    if (lastEditedWorkflow) {
      seen.add(lastEditedWorkflow.id);
      result.push({ ...lastEditedWorkflow, isLastEdited: true, isStarred: isFavorite(lastEditedWorkflow.id) });
    }

    // 2. Starred workflows (not already added)
    for (const fav of favoriteWorkflows) {
      if (!seen.has(fav.id)) {
        seen.add(fav.id);
        // Find the full workflow data from recent if available
        const recentData = recentWorkflows.find(w => w.id === fav.id);
        result.push({
          id: fav.id,
          name: fav.name,
          projectId: fav.projectId,
          projectName: fav.projectName,
          updatedAt: recentData?.updatedAt ?? fav.addedAt,
          folderPath: recentData?.folderPath ?? '/',
          isStarred: true,
        });
      }
    }

    // 3. Other recent workflows (not already added)
    for (const workflow of recentWorkflows) {
      if (!seen.has(workflow.id)) {
        seen.add(workflow.id);
        result.push({ ...workflow, isStarred: false });
      }
    }

    return result.slice(0, 8); // Show up to 8 items
  }, [lastEditedWorkflow, favoriteWorkflows, recentWorkflows, isFavorite]);

  // Render AI-enabled quick start (prominent)
  const renderAIQuickStart = () => (
    <div className="card-gradient-purple border border-purple-500/30 rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-12 h-12 bg-purple-500/20 rounded-xl">
          <Sparkles size={24} className="text-purple-400" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Create New Automation</h2>
          <p className="text-sm text-flow-text-muted">Describe what you want to automate in plain English</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="mb-4">
        <div className="relative">
          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Log into example.com and take a screenshot of the dashboard..."
            className="w-full px-4 py-4 pr-28 bg-flow-node border border-flow-border rounded-lg text-white placeholder-flow-text-muted focus:outline-none focus:border-purple-500/50 focus:ring-2 focus:ring-purple-500/30 transition-colors text-base"
            disabled={isGenerating}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || isGenerating}
            className="absolute right-2 top-1/2 -translate-y-1/2 px-4 py-2 bg-purple-600 hover:bg-purple-500 disabled:bg-flow-text-muted disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
          >
            {isGenerating ? (
              <>
                <Loader2 size={16} className="animate-spin" />
                <span className="hidden sm:inline">Generating...</span>
              </>
            ) : (
              <>
                <Zap size={16} />
                <span className="hidden sm:inline">Generate</span>
              </>
            )}
          </button>
        </div>
      </form>

      <div className="flex flex-wrap gap-2 mb-4">
        {examplePrompts.map((example, index) => (
          <button
            key={index}
            onClick={() => handleExampleClick(example)}
            className="px-3 py-1.5 text-xs text-flow-text-secondary bg-flow-node/60 hover:bg-flow-node-hover hover:text-white border border-flow-border/50 rounded-full transition-colors"
          >
            {example.length > 40 ? example.slice(0, 40) + '...' : example}
          </button>
        ))}
      </div>

      <div className="flex items-center gap-4 pt-4 border-t border-flow-border/50">
        <span className="text-xs text-flow-text-muted">or</span>
        <button
          onClick={onCreateManual}
          className="text-sm text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1.5"
        >
          <Plus size={16} />
          Create workflow manually
        </button>
      </div>
    </div>
  );

  // Render manual-first quick start (when AI not available)
  const renderManualQuickStart = () => (
    <div className="bg-flow-surface border border-flow-border rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-12 h-12 bg-blue-500/20 rounded-xl">
          <Bot size={24} className="text-blue-400" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Create New Automation</h2>
          <p className="text-sm text-flow-text-muted">Build visual workflows to automate browser tasks</p>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
        <button
          onClick={onCreateManual}
          className="flex items-start gap-3 p-4 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 hover:border-blue-500/50 rounded-lg transition-colors text-left"
        >
          <Plus size={20} className="text-blue-400 mt-0.5" />
          <div>
            <div className="font-medium text-white">Create Manually</div>
            <div className="text-xs text-flow-text-muted mt-1">
              Use the visual drag-and-drop workflow builder
            </div>
          </div>
        </button>

        <button
          onClick={() => onUseTemplate(examplePrompts[0], 'Login Test')}
          className="flex items-start gap-3 p-4 bg-flow-node/50 hover:bg-flow-node-hover border border-flow-border/50 hover:border-flow-border rounded-lg transition-colors text-left"
        >
          <FolderOpen size={20} className="text-flow-text-secondary mt-0.5" />
          <div>
            <div className="font-medium text-white">Start from Template</div>
            <div className="text-xs text-flow-text-muted mt-1">
              Choose from pre-built automation patterns
            </div>
          </div>
        </button>
      </div>

      {/* AI Upsell */}
      <div className="flex items-center gap-3 p-4 bg-purple-900/20 border border-purple-500/30 rounded-lg">
        <Sparkles size={20} className="text-purple-400 flex-shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="text-sm text-purple-200">
            Want AI to build workflows for you?
          </div>
          <div className="text-xs text-purple-300/70 mt-0.5">
            {aiCapability.reason === 'no_credits'
              ? 'Add credits to your account or configure your own API keys'
              : 'Configure your AI API keys in settings'}
          </div>
        </div>
        <button
          onClick={onOpenSettings}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-purple-300 hover:text-purple-200 bg-purple-500/20 hover:bg-purple-500/30 rounded-lg transition-colors"
        >
          {aiCapability.reason === 'no_credits' ? (
            <>
              <CreditCard size={14} />
              <span>Add Credits</span>
            </>
          ) : (
            <>
              <Key size={14} />
              <span>Setup</span>
            </>
          )}
        </button>
      </div>
    </div>
  );

  // Render running executions widget - prominent display of active executions
  const renderRunningExecutions = () => {
    if (runningExecutions.length === 0) return null;

    return (
      <div className="card-gradient-green border border-green-500/30 rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-white flex items-center gap-2">
            <div className="relative">
              <Activity size={16} className="text-green-400" />
              <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-green-400 rounded-full animate-pulse" />
            </div>
            Running Now
            <span className="text-xs font-normal text-green-300/70 bg-green-500/20 px-2 py-0.5 rounded-full">
              {runningExecutions.length} active
            </span>
          </h3>
        </div>
        <div className="space-y-2">
          {runningExecutions.slice(0, 3).map((execution) => (
            <div
              key={execution.id}
              className="flex items-center gap-3 p-3 bg-flow-node/50 border border-green-500/20 rounded-lg group"
            >
              <div className="flex-shrink-0">
                <Loader2 size={16} className="text-green-400 animate-spin" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium text-white text-sm truncate">
                  {execution.workflowName}
                </div>
                <div className="text-xs text-flow-text-muted">
                  {execution.projectName && `${execution.projectName} · `}
                  Started {formatDistanceToNow(execution.startedAt, { addSuffix: true })}
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => onViewExecution(execution.id, execution.workflowId)}
                  className="p-2 text-flow-text-secondary hover:text-white hover:bg-flow-node-hover rounded-lg transition-colors"
                  title="View execution"
                >
                  <Eye size={14} />
                </button>
                <button
                  onClick={() => handleStopExecution(execution.id)}
                  className="p-2 text-flow-text-secondary hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
                  title="Stop execution"
                >
                  <Square size={14} />
                </button>
              </div>
            </div>
          ))}
          {runningExecutions.length > 3 && (
            <button
              onClick={() => onViewExecution(runningExecutions[0].id, runningExecutions[0].workflowId)}
              className="w-full text-center text-xs text-green-400 hover:text-green-300 py-2 transition-colors"
            >
              +{runningExecutions.length - 3} more running...
            </button>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* Running Executions Widget - TOP PRIORITY when executions are active */}
      {renderRunningExecutions()}

      {/* Quick Start Section - conditional based on AI availability */}
      {aiCapability.available ? renderAIQuickStart() : renderManualQuickStart()}

      {/* Unified Workflows List - Last edited → Starred → Recent */}
      {unifiedWorkflows.length > 0 && (
        <div className="bg-flow-surface border border-flow-border rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-white">
              Your Workflows
            </h2>
            <span className="text-xs text-flow-text-muted">
              Quick access to your work
            </span>
          </div>
          <div className="space-y-2">
            {unifiedWorkflows.map((workflow) => (
              <div
                key={workflow.id}
                className={`flex items-center gap-3 p-3 rounded-lg group transition-all ${
                  workflow.isLastEdited
                    ? 'bg-blue-900/20 hover:bg-blue-900/30 border border-blue-500/30 hover:border-blue-500/50'
                    : 'bg-flow-node/50 hover:bg-flow-node-hover border border-flow-border/40 hover:border-flow-border'
                }`}
              >
                <button
                  onClick={() => handleToggleFavorite(workflow)}
                  className={`flex-shrink-0 transition-colors ${
                    workflow.isStarred
                      ? 'text-amber-400 hover:text-amber-300'
                      : 'text-flow-text-muted hover:text-amber-400'
                  }`}
                  title={workflow.isStarred ? 'Remove from favorites' : 'Add to favorites'}
                >
                  {workflow.isStarred ? <Star size={16} fill="currentColor" /> : <StarOff size={16} />}
                </button>
                <button
                  onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                  className="flex-1 min-w-0 text-left"
                >
                  <div className="flex items-center gap-2">
                    <span className={`font-medium truncate transition-colors ${
                      workflow.isLastEdited ? 'text-blue-100 group-hover:text-blue-50' : 'text-white group-hover:text-blue-300'
                    }`}>
                      {workflow.name}
                    </span>
                    {workflow.isLastEdited && (
                      <span className="flex-shrink-0 px-1.5 py-0.5 text-[10px] font-medium text-blue-300 bg-blue-500/20 rounded">
                        Last edited
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-flow-text-muted">
                    {workflow.projectName} · {formatDistanceToNow(workflow.updatedAt, { addSuffix: true })}
                  </div>
                </button>
                {/* Always visible action buttons */}
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                    className={`p-2 rounded-lg transition-colors ${
                      workflow.isLastEdited
                        ? 'text-blue-300 hover:text-blue-200 hover:bg-blue-500/20'
                        : 'text-flow-text-muted hover:text-blue-400 hover:bg-blue-500/10'
                    }`}
                    title="Edit workflow"
                  >
                    <Pencil size={14} />
                  </button>
                  <button
                    onClick={() => onRunWorkflow(workflow.id)}
                    className={`p-2 rounded-lg transition-colors ${
                      workflow.isLastEdited
                        ? 'text-blue-300 hover:text-green-400 hover:bg-green-500/20'
                        : 'text-flow-text-muted hover:text-green-400 hover:bg-green-500/10'
                    }`}
                    title="Run workflow"
                  >
                    <Play size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Templates Gallery - Available regardless of AI capability */}
      <TemplatesGallery onUseTemplate={onUseTemplate} />
    </div>
  );
};

export default HomeTab;
