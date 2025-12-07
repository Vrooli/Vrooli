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
  ArrowRight,
  Video,
  LayoutGrid,
  Wand2,
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

  const defaultPrompt = 'Open my dashboard, verify login, and capture a screenshot';

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

  const heroPrimaryAction = useCallback(() => {
    if (lastEditedWorkflow) {
      onNavigateToWorkflow(lastEditedWorkflow.projectId, lastEditedWorkflow.id);
      return;
    }
    if (aiCapability.available) {
      onAIGenerate(defaultPrompt);
      return;
    }
    onCreateManual();
  }, [aiCapability.available, lastEditedWorkflow, onAIGenerate, onCreateManual, onNavigateToWorkflow]);

  const heroPrimaryLabel = lastEditedWorkflow
    ? 'Resume last workflow'
    : aiCapability.available
      ? 'Generate with AI'
      : 'Create workflow';

  const heroSecondaryAction = useCallback(() => {
    if (aiCapability.available) {
      onUseTemplate(examplePrompts[0], 'Template jumpstart');
    } else {
      onOpenSettings();
    }
  }, [aiCapability.available, onOpenSettings, onUseTemplate]);

  const heroSecondaryLabel = aiCapability.available ? 'Start from template' : 'Setup AI keys';

  const highlightWorkflows = useMemo(() => {
    const highlights = unifiedWorkflows.filter(w => w.isLastEdited || w.isStarred);
    return highlights.slice(0, 6);
  }, [unifiedWorkflows]);

  // Render AI-enabled quick start inside the three-card row
  const renderAIQuickStart = () => (
    <div className="card-gradient-purple border border-purple-500/30 rounded-2xl p-5 space-y-3 h-full">
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-10 h-10 bg-purple-500/20 rounded-xl">
          <Sparkles size={20} className="text-purple-300" />
        </div>
        <div className="flex-1">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-white">AI Build</h2>
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-100 border border-purple-500/30">
              Step 1
            </span>
          </div>
          <p className="text-xs text-flow-text-muted">Describe what you want. We’ll draft the flow.</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="relative">
          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Log into example.com and take a screenshot..."
            className="w-full px-4 py-3 pr-28 bg-flow-node border border-flow-border rounded-lg text-white placeholder-flow-text-muted focus:outline-none focus:border-purple-500/50 focus:ring-2 focus:ring-purple-500/30 transition-colors text-sm"
            disabled={isGenerating || !aiCapability.available}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || isGenerating || !aiCapability.available}
            className="absolute right-2 top-1/2 -translate-y-1/2 px-3 py-2 bg-purple-600 hover:bg-purple-500 disabled:bg-flow-text-muted disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2 text-sm"
          >
            {isGenerating ? (
              <>
                <Loader2 size={14} className="animate-spin" />
                <span className="hidden sm:inline">Generating...</span>
              </>
            ) : (
              <>
                <Zap size={14} />
                <span className="hidden sm:inline">Generate</span>
              </>
            )}
          </button>
        </div>
        <div className="flex flex-wrap gap-2">
          {examplePrompts.map((example, index) => (
            <button
              key={index}
              onClick={() => handleExampleClick(example)}
              type="button"
              className="px-3 py-1.5 text-xs text-flow-text-secondary bg-flow-node/60 hover:bg-flow-node-hover hover:text-white border border-flow-border/50 rounded-full transition-colors"
            >
              {example.length > 40 ? example.slice(0, 40) + '...' : example}
            </button>
          ))}
          <button
            type="button"
            onClick={() => setPrompt(defaultPrompt)}
            className="px-3 py-1.5 text-[11px] text-purple-200 border border-purple-500/40 bg-purple-500/10 rounded-full hover:border-purple-400/60 transition-colors"
          >
            Use demo prompt
          </button>
        </div>
      </form>

      {!aiCapability.available && (
        <div className="flex items-center gap-3 p-3 bg-purple-900/20 border border-purple-500/30 rounded-lg">
          <Sparkles size={18} className="text-purple-300 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-purple-200">
              Enable AI to auto-build flows
            </div>
            <div className="text-[11px] text-purple-300/80 mt-0.5">
              {aiCapability.reason === 'no_credits'
                ? 'Add credits or configure your own API keys'
                : 'Configure your AI API keys in settings'}
            </div>
          </div>
          <button
            onClick={onOpenSettings}
            className="flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-medium text-purple-200 hover:text-white bg-purple-500/20 hover:bg-purple-500/30 rounded-lg transition-colors"
          >
            {aiCapability.reason === 'no_credits' ? (
              <>
                <CreditCard size={12} />
                <span>Add credits</span>
              </>
            ) : (
              <>
                <Key size={12} />
                <span>Setup</span>
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );

  // Render manual-first quick start (when AI not available) is now part of the card row
  const renderManualQuickStart = () => (
    <div className="bg-flow-surface border border-flow-border/80 rounded-2xl p-5 space-y-3 h-full">
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-10 h-10 bg-blue-500/20 rounded-xl">
          <LayoutGrid size={20} className="text-blue-300" />
        </div>
        <div className="flex-1">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-white">Visual Builder</h2>
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-blue-500/15 text-blue-100 border border-blue-500/30">
              Step 2
            </span>
          </div>
          <p className="text-xs text-flow-text-muted">Start from a blank canvas or tweak a template.</p>
        </div>
      </div>

      <div className="space-y-3">
        <button
          onClick={onCreateManual}
          className="flex w-full items-center justify-between gap-3 p-4 bg-blue-600/20 hover:bg-blue-600/30 border border-blue-500/30 hover:border-blue-500/50 rounded-lg transition-colors text-left"
        >
          <div className="flex items-start gap-3">
            <Plus size={18} className="text-blue-300 mt-0.5" />
            <div>
              <div className="font-medium text-white">New workflow</div>
              <div className="text-xs text-flow-text-muted mt-1">
                Drag-and-drop nodes to automate without code.
              </div>
            </div>
          </div>
          <ArrowRight size={16} className="text-blue-200" />
        </button>

        <button
          onClick={() => onUseTemplate(examplePrompts[0], 'Login Test')}
          className="flex items-start gap-3 p-4 bg-flow-node/50 hover:bg-flow-node-hover border border-flow-border/60 hover:border-flow-border rounded-lg transition-colors text-left"
        >
          <FolderOpen size={18} className="text-flow-text-secondary mt-0.5" />
          <div>
            <div className="font-medium text-white">Start from template</div>
            <div className="text-xs text-flow-text-muted mt-1">
              Pick a proven pattern and customize quickly.
            </div>
          </div>
        </button>
      </div>
    </div>
  );

  // Render running executions widget - prominent display of active executions
  const renderRunningExecutions = () => {
    const handleRunCta = () => {
      if (aiCapability.available) {
        onAIGenerate(defaultPrompt);
      } else {
        onCreateManual();
      }
    };

    if (runningExecutions.length === 0) {
      return (
        <div className="border border-flow-border/60 rounded-2xl p-4 flex items-center gap-3 bg-flow-node/60">
          <div className="flex items-center justify-center w-10 h-10 rounded-full bg-flow-node-hover">
            <Activity size={16} className="text-flow-text-secondary" />
          </div>
          <div className="flex-1">
            <div className="text-sm text-white font-medium">No active executions</div>
            <div className="text-xs text-flow-text-muted">Kick off a run to see live status here.</div>
          </div>
          <button
            onClick={handleRunCta}
            className="text-xs px-3 py-1.5 rounded-lg bg-flow-accent/20 text-blue-100 hover:bg-flow-accent/30 transition-colors flex items-center gap-1.5"
          >
            <Play size={12} />
            Run a workflow
          </button>
        </div>
      );
    }

    return (
      <div className="card-gradient-green border border-green-500/30 rounded-2xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-white flex items-center gap-2">
            <div className="relative">
              <Activity size={16} className="text-green-400" />
              <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-green-400 rounded-full animate-pulse" />
            </div>
            Running now
            <span className="text-xs font-normal text-green-300/70 bg-green-500/20 px-2 py-0.5 rounded-full">
              {runningExecutions.length} active
            </span>
          </h3>
          <span className="text-[11px] text-green-200/80">Live status · auto-refreshing</span>
        </div>
        <div className="space-y-2">
          {runningExecutions.slice(0, 3).map((execution) => (
            <div
              key={execution.id}
              className="flex items-center gap-3 p-3 bg-flow-node/60 border border-green-500/20 rounded-lg group"
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
      {/* Hero strip */}
      <div className="relative overflow-hidden rounded-2xl border border-flow-border/50 bg-gradient-to-r from-flow-node via-flow-node/90 to-purple-900/20 p-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-start gap-3">
            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-flow-accent/15 border border-flow-accent/30">
              <Wand2 size={22} className="text-blue-200" />
            </div>
            <div className="space-y-1">
              <div className="text-sm text-flow-text-muted">Welcome back</div>
              <div className="text-xl font-semibold text-white">
                {lastEditedWorkflow ? 'Pick up where you left off' : 'Launch your next automation'}
              </div>
              <div className="flex flex-wrap gap-2 text-xs text-flow-text-muted">
                {lastEditedWorkflow && (
                  <span className="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-blue-500/15 border border-blue-500/30 text-blue-100">
                    <Sparkles size={12} /> Last edited · {lastEditedWorkflow.name}
                  </span>
                )}
                <span className="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-green-500/15 border border-green-500/30 text-green-100">
                  <Activity size={12} />
                  {runningExecutions.length > 0 ? `${runningExecutions.length} running` : 'No active runs'}
                </span>
                <span className="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-purple-500/15 border border-purple-500/30 text-purple-100">
                  <Bot size={12} />
                  {aiCapability.available ? 'AI ready' : 'AI setup needed'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex flex-col sm:flex-row gap-2">
            <button
              onClick={heroPrimaryAction}
              className="hero-button-primary justify-center w-full sm:w-auto"
            >
              {heroPrimaryLabel}
              <div className="hero-button-glow" />
            </button>
            <button
              onClick={heroSecondaryAction}
              className="hero-button-secondary w-full sm:w-auto justify-center"
            >
              {heroSecondaryLabel}
            </button>
          </div>
        </div>
      </div>

      {/* Running Executions Widget - TOP PRIORITY when executions are active */}
      {renderRunningExecutions()}

      {/* Quick Start Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {renderAIQuickStart()}
        {renderManualQuickStart()}
        <div className="bg-flow-surface border border-flow-border/80 rounded-2xl p-5 space-y-3 h-full">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 bg-emerald-500/20 rounded-xl">
              <Video size={20} className="text-emerald-200" />
            </div>
            <div className="flex-1">
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold text-white">Template picks</h2>
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-100 border border-emerald-500/30">
                  Step 3
                </span>
              </div>
              <p className="text-xs text-flow-text-muted">Jumpstart with a curated pattern.</p>
            </div>
          </div>
          <div className="space-y-2">
            <button
              onClick={() => onUseTemplate('Log into a dashboard and capture a screenshot', 'Login capture')}
              className="flex items-center justify-between w-full p-3 bg-flow-node/60 hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-white">Login & capture</div>
                <div className="text-xs text-flow-text-muted">Auth + screenshot in one go.</div>
              </div>
              <ArrowRight size={14} className="text-emerald-200" />
            </button>
            <button
              onClick={() => onUseTemplate('Record and repeat add-to-cart flow', 'Record & replay')}
              className="flex items-center justify-between w-full p-3 bg-flow-node/60 hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-white">Record & replay</div>
                <div className="text-xs text-flow-text-muted">Capture clicks, repeat reliably.</div>
              </div>
              <ArrowRight size={14} className="text-emerald-200" />
            </button>
            <button
              onClick={() => onUseTemplate('Scrape pricing data from product grid', 'Price scraper')}
              className="flex items-center justify-between w-full p-3 bg-flow-node/60 hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-white">Price scraper</div>
                <div className="text-xs text-flow-text-muted">Extract structured data fast.</div>
              </div>
              <ArrowRight size={14} className="text-emerald-200" />
            </button>
          </div>
        </div>
      </div>

      {/* Continue / Favorites rail */}
      {highlightWorkflows.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-white">Continue & favorites</h2>
            <span className="text-xs text-flow-text-muted">Quick shortcuts to your go-tos</span>
          </div>
          <div className="flex gap-3 overflow-x-auto pb-1">
            {highlightWorkflows.map((workflow) => (
              <div
                key={workflow.id}
                className={`min-w-[240px] max-w-[260px] rounded-xl border p-4 flex flex-col gap-3 bg-flow-node/60 ${
                  workflow.isLastEdited
                    ? 'border-blue-500/40 shadow-[0_10px_30px_rgba(59,130,246,0.2)]'
                    : 'border-flow-border/60'
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-white truncate">{workflow.name}</div>
                    <div className="text-[11px] text-flow-text-muted truncate">{workflow.projectName}</div>
                  </div>
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
                </div>
                <div className="flex flex-wrap gap-2 text-[11px] text-flow-text-muted">
                  {workflow.isLastEdited && (
                    <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-blue-500/15 text-blue-100 border border-blue-500/30">
                      <Sparkles size={12} /> Last edited
                    </span>
                  )}
                  {workflow.isStarred && !workflow.isLastEdited && (
                    <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-amber-500/15 text-amber-100 border border-amber-500/30">
                      <Star size={12} /> Favorite
                    </span>
                  )}
                  <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-flow-node-hover text-flow-text-muted border border-flow-border/60">
                    Updated {formatDistanceToNow(workflow.updatedAt, { addSuffix: true })}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                    className="flex-1 px-3 py-2 rounded-lg bg-flow-node-hover text-white text-sm hover:bg-blue-500/20 transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => onRunWorkflow(workflow.id)}
                    className="p-2 rounded-lg bg-green-500/20 text-green-200 hover:bg-green-500/30 transition-colors"
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
