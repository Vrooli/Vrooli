import React, { useState, useCallback } from 'react';
import {
  Sparkles,
  Plus,
  Play,
  Star,
  StarOff,
  Clock,
  ChevronRight,
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

  // Render AI-enabled quick start (prominent)
  const renderAIQuickStart = () => (
    <div className="bg-gradient-to-br from-purple-900/30 via-blue-900/20 to-gray-900 border border-purple-500/30 rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-12 h-12 bg-purple-500/20 rounded-xl">
          <Sparkles size={24} className="text-purple-400" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Create New Automation</h2>
          <p className="text-sm text-gray-400">Describe what you want to automate in plain English</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="mb-4">
        <div className="relative">
          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Log into example.com and take a screenshot of the dashboard..."
            className="w-full px-4 py-4 pr-28 bg-gray-800/80 border border-gray-600/50 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50 focus:ring-2 focus:ring-purple-500/30 transition-colors text-base"
            disabled={isGenerating}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || isGenerating}
            className="absolute right-2 top-1/2 -translate-y-1/2 px-4 py-2 bg-purple-600 hover:bg-purple-500 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
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
            className="px-3 py-1.5 text-xs text-gray-400 bg-gray-800/50 hover:bg-gray-700/50 hover:text-gray-300 border border-gray-700/50 rounded-full transition-colors"
          >
            {example.length > 40 ? example.slice(0, 40) + '...' : example}
          </button>
        ))}
      </div>

      <div className="flex items-center gap-4 pt-4 border-t border-gray-700/50">
        <span className="text-xs text-gray-500">or</span>
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
    <div className="bg-gray-800/50 border border-gray-700 rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-12 h-12 bg-blue-500/20 rounded-xl">
          <Bot size={24} className="text-blue-400" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-white">Create New Automation</h2>
          <p className="text-sm text-gray-400">Build visual workflows to automate browser tasks</p>
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
            <div className="text-xs text-gray-400 mt-1">
              Use the visual drag-and-drop workflow builder
            </div>
          </div>
        </button>

        <button
          onClick={() => onUseTemplate(examplePrompts[0], 'Login Test')}
          className="flex items-start gap-3 p-4 bg-gray-700/30 hover:bg-gray-700/50 border border-gray-600/50 hover:border-gray-600 rounded-lg transition-colors text-left"
        >
          <FolderOpen size={20} className="text-gray-400 mt-0.5" />
          <div>
            <div className="font-medium text-white">Start from Template</div>
            <div className="text-xs text-gray-400 mt-1">
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

  // Render continue editing banner
  const renderContinueEditing = () => {
    if (!lastEditedWorkflow) return null;

    return (
      <button
        onClick={() => onNavigateToWorkflow(lastEditedWorkflow.projectId, lastEditedWorkflow.id)}
        className="w-full flex items-center gap-4 p-4 bg-blue-900/20 hover:bg-blue-900/30 border border-blue-500/30 hover:border-blue-500/50 rounded-lg transition-colors text-left group"
      >
        <div className="flex items-center justify-center w-10 h-10 bg-blue-500/20 rounded-lg">
          <Pencil size={18} className="text-blue-400" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-sm text-blue-300">Continue editing</div>
          <div className="font-medium text-white truncate">{lastEditedWorkflow.name}</div>
          <div className="text-xs text-gray-500 mt-0.5">
            {lastEditedWorkflow.projectName} · Edited {formatDistanceToNow(lastEditedWorkflow.updatedAt, { addSuffix: true })}
          </div>
        </div>
        <ChevronRight size={20} className="text-gray-500 group-hover:text-gray-400 transition-colors" />
      </button>
    );
  };

  // Render favorites section
  const renderFavorites = () => {
    if (favoriteWorkflows.length === 0) return null;

    return (
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-300 flex items-center gap-2">
            <Star size={14} className="text-amber-400" />
            Favorites
          </h3>
        </div>
        <div className="space-y-2">
          {favoriteWorkflows.slice(0, 5).map((workflow) => (
            <div
              key={workflow.id}
              className="flex items-center gap-3 p-3 bg-gray-800/50 hover:bg-gray-800 border border-gray-700/50 rounded-lg group transition-colors"
            >
              <button
                onClick={() => removeFavorite(workflow.id)}
                className="text-amber-400 hover:text-amber-300 transition-colors"
                title="Remove from favorites"
              >
                <Star size={16} fill="currentColor" />
              </button>
              <button
                onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                className="flex-1 min-w-0 text-left"
              >
                <div className="font-medium text-white truncate group-hover:text-blue-300 transition-colors">
                  {workflow.name}
                </div>
                <div className="text-xs text-gray-500">{workflow.projectName}</div>
              </button>
              <button
                onClick={() => onRunWorkflow(workflow.id)}
                className="p-2 text-gray-500 hover:text-green-400 hover:bg-green-500/10 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
                title="Run workflow"
              >
                <Play size={14} />
              </button>
            </div>
          ))}
        </div>
      </div>
    );
  };

  // Render running executions widget - prominent display of active executions
  const renderRunningExecutions = () => {
    if (runningExecutions.length === 0) return null;

    return (
      <div className="bg-gradient-to-r from-green-900/20 via-emerald-900/15 to-gray-900 border border-green-500/30 rounded-xl p-4">
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
              className="flex items-center gap-3 p-3 bg-gray-800/50 border border-green-500/20 rounded-lg group"
            >
              <div className="flex-shrink-0">
                <Loader2 size={16} className="text-green-400 animate-spin" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium text-white text-sm truncate">
                  {execution.workflowName}
                </div>
                <div className="text-xs text-gray-500">
                  {execution.projectName && `${execution.projectName} · `}
                  Started {formatDistanceToNow(execution.startedAt, { addSuffix: true })}
                </div>
              </div>
              <div className="flex items-center gap-1">
                <button
                  onClick={() => onViewExecution(execution.id, execution.workflowId)}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title="View execution"
                >
                  <Eye size={14} />
                </button>
                <button
                  onClick={() => handleStopExecution(execution.id)}
                  className="p-2 text-gray-400 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors"
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

      {/* Continue Editing Banner */}
      {renderContinueEditing()}

      {/* Recent Workflows - Elevated as a primary section */}
      {recentWorkflows.length > 0 && (
        <div className="bg-gray-800/40 border border-gray-700/60 rounded-xl p-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-white flex items-center gap-2">
              <Clock size={18} className="text-blue-400" />
              Recent Workflows
            </h2>
            <span className="text-xs text-gray-500">
              Quick access to your latest work
            </span>
          </div>
          <div className="space-y-2">
            {recentWorkflows.slice(0, 5).map((workflow) => {
              const starred = isFavorite(workflow.id);
              return (
                <div
                  key={workflow.id}
                  className="flex items-center gap-3 p-3 bg-gray-800/50 hover:bg-gray-700/50 border border-gray-700/40 hover:border-gray-600/60 rounded-lg group transition-all"
                >
                  <button
                    onClick={() => handleToggleFavorite(workflow)}
                    className={`flex-shrink-0 transition-colors ${
                      starred
                        ? 'text-amber-400 hover:text-amber-300'
                        : 'text-gray-600 hover:text-amber-400'
                    }`}
                    title={starred ? 'Remove from favorites' : 'Add to favorites'}
                  >
                    {starred ? <Star size={16} fill="currentColor" /> : <StarOff size={16} />}
                  </button>
                  <button
                    onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                    className="flex-1 min-w-0 text-left"
                  >
                    <div className="font-medium text-white truncate group-hover:text-blue-300 transition-colors">
                      {workflow.name}
                    </div>
                    <div className="text-xs text-gray-500">
                      {workflow.projectName} · {formatDistanceToNow(workflow.updatedAt, { addSuffix: true })}
                    </div>
                  </button>
                  {/* Always visible action buttons */}
                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button
                      onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                      className="p-2 text-gray-500 hover:text-blue-400 hover:bg-blue-500/10 rounded-lg transition-colors"
                      title="Edit workflow"
                    >
                      <Pencil size={14} />
                    </button>
                    <button
                      onClick={() => onRunWorkflow(workflow.id)}
                      className="p-2 text-gray-500 hover:text-green-400 hover:bg-green-500/10 rounded-lg transition-colors"
                      title="Run workflow"
                    >
                      <Play size={14} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Favorites Section - Secondary but visible */}
      {favoriteWorkflows.length > 0 && (
        <div className="bg-gray-800/30 border border-gray-700/50 rounded-lg p-4">
          {renderFavorites()}
        </div>
      )}

      {/* Templates Gallery */}
      {aiCapability.available && (
        <TemplatesGallery onUseTemplate={onUseTemplate} />
      )}
    </div>
  );
};

export default HomeTab;
