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
  Circle,
} from 'lucide-react';
import { useDashboardStore, type RecentWorkflow, type FavoriteWorkflow } from '@stores/dashboardStore';
import { useExecutionStore } from '@stores/executionStore';
import { useAICapability } from '@stores/aiCapabilityStore';
import { formatDistanceToNow } from 'date-fns';
import { TemplatesGallery } from './widgets/TemplatesGallery';

interface HomeTabProps {
  onAIGenerate: (prompt: string) => void;
  onCreateManual: () => void;
  onNavigateToWorkflow: (projectId: string, workflowId: string) => void;
  onRunWorkflow: (workflowId: string) => void;
  onViewExecution: (executionId: string, workflowId: string) => void;
  onOpenSettings: () => void;
  onUseTemplate: (prompt: string, templateName: string) => void;
  onStartRecording?: () => void;
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
  onStartRecording,
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
    <div className="bg-flow-surface border border-flow-border/70 rounded-2xl p-5 space-y-4 h-full shadow-[0_12px_30px_rgba(0,0,0,0.2)]">
      <div className="flex items-start gap-3">
        <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-[rgb(var(--flow-surface))] border border-[rgb(var(--flow-border))]">
          <Sparkles size={20} className="text-purple-700" />
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-flow-text">AI Build</h2>
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-[rgb(var(--flow-surface))] text-flow-text-secondary border border-[rgb(var(--flow-border))]">
              Step 1
            </span>
          </div>
          <p className="text-sm text-flow-text-secondary">Describe what you want. We’ll draft the flow.</p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div className="relative">
          <input
            type="text"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Log into example.com and take a screenshot..."
            className="w-full px-4 py-3 pr-28 bg-[rgb(var(--flow-surface))] border border-flow-border/70 rounded-lg text-flow-text placeholder-flow-text-muted focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/25 transition-colors text-sm"
            disabled={isGenerating || !aiCapability.available}
          />
          <button
            type="submit"
            disabled={!prompt.trim() || isGenerating || !aiCapability.available}
            aria-label={isGenerating ? "Generating workflow" : "Generate workflow"}
            className="absolute right-2 top-1/2 -translate-y-1/2 px-3 py-2 bg-flow-accent hover:bg-flow-accent/90 disabled:bg-flow-text-muted disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2 text-sm"
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
            className="px-3 py-1.5 text-xs text-flow-text-secondary bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover hover:text-flow-text border border-flow-border/50 rounded-full transition-colors"
          >
              {example.length > 40 ? example.slice(0, 40) + '...' : example}
            </button>
          ))}
          <button
            type="button"
            onClick={() => setPrompt(defaultPrompt)}
            className="px-3 py-1.5 text-[11px] text-purple-100 border border-purple-500/30 bg-purple-500/10 rounded-full hover:border-purple-400/60 transition-colors"
          >
            Use demo prompt
          </button>
        </div>
      </form>

      {!aiCapability.available && (
        <div className="flex items-center gap-3 p-3 bg-[rgb(var(--flow-surface))] border border-[rgb(var(--flow-border))] rounded-lg">
          <Sparkles size={18} className="text-purple-600 flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-xs text-flow-text-secondary">
              Enable AI to auto-build flows
            </div>
            <div className="text-[11px] text-flow-text-muted mt-0.5">
              {aiCapability.reason === 'no_credits'
                ? 'Add credits or configure your own API keys'
                : 'Configure your AI API keys in settings'}
            </div>
          </div>
          <button
            onClick={onOpenSettings}
            className="flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-medium text-flow-text bg-white hover:bg-flow-node-hover rounded-lg transition-colors border border-flow-border"
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
    <div className="bg-flow-surface border border-flow-border/70 rounded-2xl p-5 space-y-4 h-full shadow-[0_12px_30px_rgba(0,0,0,0.2)]">
      <div className="flex items-start gap-3">
        <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-[rgb(var(--flow-surface))] border border-[rgb(var(--flow-border))]">
          <LayoutGrid size={20} className="text-blue-700" />
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold text-flow-text">Visual Builder</h2>
            <span className="text-[11px] px-2 py-0.5 rounded-full bg-[rgb(var(--flow-surface))] text-flow-text-muted border border-[rgb(var(--flow-border))]">
              Step 2
            </span>
          </div>
          <p className="text-sm text-flow-text-muted">Start from a blank canvas or tweak a template.</p>
        </div>
      </div>

      <div className="space-y-3">
        <button
          onClick={onCreateManual}
          className="flex w-full items-center justify-between gap-3 p-4 bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
        >
          <div className="flex items-start gap-3">
            <Plus size={18} className="text-blue-700 mt-0.5" />
            <div>
              <div className="font-medium text-flow-text">New workflow</div>
              <div className="text-xs text-flow-text-muted mt-1">
                Drag-and-drop nodes to automate without code.
              </div>
            </div>
          </div>
          <ArrowRight size={16} className="text-flow-text-secondary" />
        </button>

        <button
          onClick={() => onUseTemplate(examplePrompts[0], 'Login Test')}
          className="flex items-start gap-3 p-4 bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
        >
          <FolderOpen size={18} className="text-flow-text-secondary mt-0.5" />
          <div>
            <div className="font-medium text-flow-text">Start from template</div>
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
        <div className="border border-flow-border/70 rounded-2xl p-4 flex items-center gap-3 bg-flow-surface/70 shadow-[0_8px_24px_rgba(0,0,0,0.18)]">
          <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-flow-node">
            <Activity size={16} className="text-flow-text-secondary" />
          </div>
          <div className="flex-1">
            <div className="text-sm text-flow-text font-medium">No active executions</div>
            <div className="text-xs text-flow-text-muted">Kick off a run to see live status here.</div>
          </div>
          <button
            onClick={handleRunCta}
            className="text-xs px-3 py-1.5 rounded-lg bg-flow-accent/20 text-surface hover:bg-flow-accent/30 transition-colors flex items-center gap-1.5"
          >
            <Play size={12} />
            Run a workflow
          </button>
        </div>
      );
    }

    return (
      <div className="bg-flow-surface border border-flow-border/70 rounded-2xl p-4 shadow-[0_12px_30px_rgba(0,0,0,0.2)]">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-flow-text flex items-center gap-2">
            <div className="relative">
              <Activity size={16} className="text-emerald-600" />
              <span className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-emerald-500 rounded-full animate-pulse" />
            </div>
            Running now
            <span className="text-xs font-normal text-flow-text-muted bg-flow-node px-2 py-0.5 rounded-full border border-flow-border/60">
              {runningExecutions.length} active
            </span>
          </h3>
          <span className="text-[11px] text-flow-text-muted">Live status · auto-refreshing</span>
        </div>
        <div className="space-y-2">
          {runningExecutions.slice(0, 3).map((execution) => (
            <div
              key={execution.id}
              className="flex items-center gap-3 p-3 bg-flow-node border border-flow-border/60 rounded-lg group"
            >
              <div className="flex-shrink-0">
                <Loader2 size={16} className="text-emerald-600 animate-spin" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="font-medium text-flow-text text-sm truncate">
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
                  className="p-2 text-flow-text-secondary hover:text-surface hover:bg-flow-node-hover rounded-lg transition-colors data-[theme=light]:hover:bg-flow-node"
                  title="View execution"
                >
                  <Eye size={14} />
                </button>
                <button
                  onClick={() => handleStopExecution(execution.id)}
                  className="p-2 text-flow-text-secondary hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors data-[theme=light]:hover:bg-red-100"
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
              className="w-full text-center text-xs text-flow-text-secondary hover:text-surface py-2 transition-colors"
            >
              +{runningExecutions.length - 3} more running...
            </button>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-8">
      {/* Hero strip */}
      <div className="relative overflow-hidden rounded-2xl border border-flow-border/60 bg-flow-surface shadow-[0_16px_50px_rgba(0,0,0,0.28)]">
        <div className="absolute inset-0 hero-background opacity-50" />
        <div className="absolute inset-0 bg-gradient-to-r from-flow-node/60 via-flow-node/30 to-transparent" />
        <div className="relative p-6 sm:p-8 space-y-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex items-start gap-4">
              <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-[rgb(var(--flow-surface))] border border-[rgb(var(--flow-border))] shadow-inner shadow-blue-500/10">
                <Wand2 size={22} className="text-flow-text-secondary" />
              </div>
              <div className="space-y-2">
                <div className="text-xs uppercase tracking-[0.08em] text-flow-text-muted">Welcome back</div>
                <div className="text-2xl sm:text-3xl font-bold hero-gradient-text leading-tight">
                  {lastEditedWorkflow ? 'Pick up where you left off' : 'Launch your next automation'}
                </div>
              <div className="text-sm text-flow-text-secondary max-w-2xl">
                Keep momentum with your latest workflow, jump into recording, or start a new build with AI.
              </div>
                <div className="flex flex-wrap items-center gap-3 text-xs text-flow-text-muted">
                  {lastEditedWorkflow && (
                    <div className="inline-flex items-center gap-1.5 text-flow-text-secondary">
                      <Sparkles size={12} className="text-flow-text-secondary" />
                      <span className="text-flow-text">{lastEditedWorkflow.name}</span>
                      <span className="text-flow-text-muted">· last edited</span>
                    </div>
                  )}
                  <span className="h-4 w-px bg-flow-border/70" />
                <div className="inline-flex items-center gap-1.5 text-flow-text-secondary">
                  <Activity size={12} className="text-emerald-600" />
                  {runningExecutions.length > 0 ? `${runningExecutions.length} active run${runningExecutions.length > 1 ? 's' : ''}` : 'No active runs'}
                </div>
                <span className="h-4 w-px bg-flow-border/70" />
                <div className="inline-flex items-center gap-1.5 text-flow-text-secondary">
                  <Bot size={12} className="text-purple-600" />
                  {aiCapability.available ? 'AI ready' : 'Connect AI to enable drafting'}
                </div>
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
                onClick={onStartRecording ?? heroSecondaryAction}
                className="hero-button-secondary w-full sm:w-auto justify-center"
              >
                {onStartRecording ? 'Start recording' : heroSecondaryLabel}
              </button>
            </div>
          </div>
          {onStartRecording && (
            <div className="flex flex-wrap items-center gap-3 text-xs text-flow-text-secondary">
              <button
                onClick={heroSecondaryAction}
                className="inline-flex items-center gap-1.5 text-flow-text-muted hover:text-flow-text transition-colors"
              >
                <LayoutGrid size={12} />
                <span>Browse templates</span>
              </button>
              <span className="h-3 w-px bg-flow-border/70" />
              <button
                onClick={onOpenSettings}
                className="inline-flex items-center gap-1.5 text-flow-text-muted hover:text-flow-text transition-colors"
              >
                <Key size={12} />
                <span>AI settings</span>
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Running Executions Widget - TOP PRIORITY when executions are active */}
      {renderRunningExecutions()}

      {/* Quick Start Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {renderAIQuickStart()}
            {renderManualQuickStart()}
            <div className="bg-flow-surface border border-flow-border/70 rounded-2xl p-5 space-y-4 h-full shadow-[0_12px_30px_rgba(0,0,0,0.2)]">
              <div className="flex items-start gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-[rgb(var(--flow-surface))] border border-[rgb(var(--flow-border))]">
              <Video size={20} className="text-emerald-700" />
            </div>
            <div className="flex-1 space-y-1">
              <div className="flex items-center gap-2">
                <h2 className="text-base font-semibold text-flow-text">Template picks</h2>
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-[rgb(var(--flow-surface))] text-flow-text-muted border border-[rgb(var(--flow-border))]">
                  Step 3
                </span>
              </div>
              <p className="text-sm text-flow-text-muted">Jumpstart with a curated pattern.</p>
            </div>
          </div>
          {onStartRecording && (
            <div className="p-3 rounded-xl border border-red-500/25 bg-red-500/10 space-y-2">
              <div className="flex items-center justify-between">
                <div className="inline-flex items-center gap-2 text-red-700">
                  <Circle size={12} className="text-red-500 fill-red-500" />
                  <span className="font-semibold">Record Mode</span>
                </div>
                <span className="text-[11px] text-red-700">Live capture</span>
              </div>
              <p className="text-xs text-red-700/80">
                Capture clicks and inputs, then turn them into a workflow with one click.
              </p>
              <button
                onClick={onStartRecording}
                className="inline-flex items-center gap-2 text-sm font-medium text-white px-3 py-2 rounded-lg bg-red-500 hover:bg-red-600 transition-colors w-full justify-center"
              >
                <Circle size={12} className="text-white fill-white" />
                Start recording
              </button>
            </div>
          )}
          <div className="space-y-2">
            <button
              onClick={() => onUseTemplate('Log into a dashboard and capture a screenshot', 'Login capture')}
              className="flex items-center justify-between w-full p-3 bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-flow-text">Login & capture</div>
                <div className="text-xs text-flow-text-muted">Auth + screenshot in one go.</div>
              </div>
              <ArrowRight size={14} className="text-emerald-200" />
            </button>
            <button
              onClick={() => onUseTemplate('Record and repeat add-to-cart flow', 'Record & replay')}
              className="flex items-center justify-between w-full p-3 bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-flow-text">Record & replay</div>
                <div className="text-xs text-flow-text-muted">Capture clicks, repeat reliably.</div>
              </div>
              <ArrowRight size={14} className="text-emerald-200" />
            </button>
            <button
              onClick={() => onUseTemplate('Scrape pricing data from product grid', 'Price scraper')}
              className="flex items-center justify-between w-full p-3 bg-[rgb(var(--flow-surface))] hover:bg-flow-node-hover border border-flow-border/60 rounded-lg transition-colors text-left"
            >
              <div>
                <div className="text-sm font-medium text-flow-text">Price scraper</div>
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
            <h2 className="text-sm font-semibold text-flow-text">Continue & favorites</h2>
            <span className="text-xs text-flow-text-muted">Quick shortcuts to your go-tos</span>
          </div>
          <div className="flex gap-3 overflow-x-auto pb-1">
            {highlightWorkflows.map((workflow) => (
              <div
                key={workflow.id}
                className="min-w-[240px] max-w-[260px] rounded-xl border border-flow-border/70 p-4 flex flex-col gap-3 bg-flow-surface shadow-[0_8px_24px_rgba(0,0,0,0.18)]"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-flow-text truncate">{workflow.name}</div>
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
                  <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-white text-flow-text-secondary border border-flow-border/60">
                    <Sparkles size={12} /> Last edited
                  </span>
                )}
                {workflow.isStarred && !workflow.isLastEdited && (
                    <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-white text-flow-text-secondary border border-flow-border/60">
                      <Star size={12} /> Favorite
                    </span>
                  )}
                  <span className="inline-flex items-center gap-1 px-2 py-1 rounded-full bg-white text-flow-text-muted border border-flow-border/60">
                    Updated {formatDistanceToNow(workflow.updatedAt, { addSuffix: true })}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
                    className="flex-1 px-3 py-2 rounded-lg bg-flow-node-hover text-flow-text text-sm hover:bg-flow-node transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => onRunWorkflow(workflow.id)}
                    className="p-2 rounded-lg bg-flow-node text-flow-text-secondary hover:text-green-300 hover:bg-flow-node-hover transition-colors"
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
        <div className="bg-flow-surface border border-flow-border/70 rounded-xl p-5 shadow-[0_12px_30px_rgba(0,0,0,0.18)]">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-flow-text">
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
                className="flex items-center gap-3 p-3 rounded-lg group transition-all bg-flow-node hover:bg-flow-node-hover border border-flow-border/60"
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
                      workflow.isLastEdited ? 'text-flow-text group-hover:text-blue-500' : 'text-flow-text group-hover:text-blue-500'
                    }`}>
                      {workflow.name}
                    </span>
                    {workflow.isLastEdited && (
                      <span className="flex-shrink-0 px-1.5 py-0.5 text-[10px] font-medium text-flow-text-secondary bg-white rounded border border-flow-border/60">
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
                    className="p-2 rounded-lg transition-colors text-flow-text-muted hover:text-surface hover:bg-flow-node"
                    title="Edit workflow"
                  >
                    <Pencil size={14} />
                  </button>
                  <button
                    onClick={() => onRunWorkflow(workflow.id)}
                    className="p-2 rounded-lg transition-colors text-flow-text-muted hover:text-green-300 hover:bg-flow-node"
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
