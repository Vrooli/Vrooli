import { useCallback, useEffect, useMemo, useState } from 'react';
import { CheckCircle, HelpCircle, Sparkles, Zap } from 'lucide-react';
import {
  generateScenario,
  listTemplates,
  listGeneratedScenarios,
  customizeScenario,
  promoteScenario,
  deleteScenario,
  type GenerationResult,
  type Template,
  type GeneratedScenario,
  type CustomizeResult,
} from '../lib/api';
import { Tooltip } from '../components/Tooltip';
import { PromoteDialog } from '../components/PromoteDialog';
import { KeyboardShortcutsDialog } from '../components/KeyboardShortcutsDialog';
import { StatsOverview } from '../components/StatsOverview';
import { QuickStartGuide } from '../components/QuickStartGuide';
import { GeneratedScenariosList } from '../components/GeneratedScenariosList';
import { CreateScenarioDialog } from '../components/CreateScenarioDialog';
import { AgentCustomizationDialog } from '../components/AgentCustomizationDialog';
import { FloatingCreateButton } from '../components/FloatingCreateButton';
import { useScenarioLifecycle } from '../hooks/useScenarioLifecycle';

export default function FactoryHome() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [generated, setGenerated] = useState<GeneratedScenario[]>([]);
  const [loadingGenerated, setLoadingGenerated] = useState(true);
  const [generatedError, setGeneratedError] = useState<string | null>(null);

  // Dialog states
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [showCustomizeDialog, setShowCustomizeDialog] = useState(false);
  const [customizeTarget, setCustomizeTarget] = useState<GeneratedScenario | null>(null);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);

  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [promoteDialogScenario, setPromoteDialogScenario] = useState<string | null>(null);

  // Lifecycle management using custom hook
  const lifecycle = useScenarioLifecycle();
  const { statuses: scenarioStatuses, previewLinks, showLogs, logs: scenarioLogs } = lifecycle;

  // For StatsOverview - show first template as selected example
  const selectedTemplate = useMemo(() => templates?.[0] ?? null, [templates]);

  useEffect(() => {
    async function loadTemplates() {
      try {
        setLoadingTemplates(true);
        const tpl = await listTemplates();
        setTemplates(tpl);
        setTemplatesError(null);
      } catch (err) {
        setTemplatesError(err instanceof Error ? err.message : 'Failed to load templates');
      } finally {
        setLoadingTemplates(false);
      }
    }

    loadTemplates();
  }, []);

  useEffect(() => {
    async function loadGenerated() {
      try {
        setLoadingGenerated(true);
        const scenarios = await listGeneratedScenarios();
        setGenerated(scenarios);
        setGeneratedError(null);
      } catch (err) {
        setGeneratedError(err instanceof Error ? err.message : 'Failed to load generated scenarios');
      } finally {
        setLoadingGenerated(false);
      }
    }

    loadGenerated();
  }, []);

  const handleRefreshTemplates = useCallback((newTemplates: Template[], error: string | null, loading: boolean) => {
    setTemplates(newTemplates);
    setTemplatesError(error);
    setLoadingTemplates(loading);
  }, []);

  const handleGenerate = useCallback(async (
    templateId: string,
    name: string,
    slug: string,
    dryRun: boolean
  ): Promise<GenerationResult> => {
    const result = await generateScenario(templateId, name, slug, { dry_run: dryRun });

    // Refresh generated scenarios list if we actually wrote files
    if (!dryRun) {
      const scenarios = await listGeneratedScenarios();
      setGenerated(scenarios);
      setSuccessMessage(`Successfully generated "${name}" scenario!`);
      setTimeout(() => setSuccessMessage(null), 5000);
    }

    return result;
  }, []);

  const handleCustomize = useCallback(async (
    scenarioSlug: string,
    brief: string,
    assets: string[]
  ): Promise<CustomizeResult> => {
    const result = await customizeScenario(scenarioSlug, brief, assets, true);
    return result;
  }, []);

  const handleOpenCreateDialog = useCallback(() => {
    setShowCreateDialog(true);
  }, []);

  const handleCloseCreateDialog = useCallback(() => {
    setShowCreateDialog(false);
  }, []);

  const handleOpenCustomizeDialog = useCallback((scenario: GeneratedScenario) => {
    setCustomizeTarget(scenario);
    setShowCustomizeDialog(true);
  }, []);

  const handleCloseCustomizeDialog = useCallback(() => {
    setShowCustomizeDialog(false);
    setCustomizeTarget(null);
  }, []);

  // Lifecycle handlers
  const handleStartScenario = lifecycle.startScenario;
  const handleStopScenario = lifecycle.stopScenario;
  const handleRestartScenario = lifecycle.restartScenario;
  const handleToggleLogs = lifecycle.toggleLogs;

  const handlePromoteScenario = async (scenarioId: string) => {
    setPromoteDialogScenario(scenarioId);
  };

  const confirmPromote = async () => {
    if (!promoteDialogScenario) return;

    const scenarioId = promoteDialogScenario;
    setPromoteDialogScenario(null);

    try {
      const result = await promoteScenario(scenarioId);

      if (result.success) {
        setGenerated((prev) => prev.filter(s => s.scenario_id !== scenarioId));
        setSuccessMessage(`Successfully promoted "${scenarioId}" to production at ${result.production_path}`);
      } else {
        setGeneratedError(`Failed to promote: ${result.message}`);
      }
    } catch (err) {
      console.error('Failed to promote scenario:', err);
      setGeneratedError(`Failed to promote: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleDeleteScenario = async (scenarioId: string) => {
    try {
      const result = await deleteScenario(scenarioId);

      if (result.success) {
        setGenerated((prev) => prev.filter(s => s.scenario_id !== scenarioId));
        setSuccessMessage(`Successfully deleted "${scenarioId}" from staging`);
        setTimeout(() => setSuccessMessage(null), 5000);
      } else {
        setGeneratedError(`Failed to delete: ${result.message}`);
      }
    } catch (err) {
      console.error('Failed to delete scenario:', err);
      setGeneratedError(`Failed to delete: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  // Load scenario statuses when generated scenarios list changes
  useEffect(() => {
    if (generated.length > 0) {
      const scenarioIds = generated.map((s) => s.scenario_id);
      lifecycle.loadStatuses(scenarioIds);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generated]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl+K to open create dialog
      if ((e.metaKey || e.ctrlKey) && e.key === 'k' && !showCreateDialog) {
        e.preventDefault();
        setShowCreateDialog(true);
      }
      // Escape to close dialogs
      if (e.key === 'Escape') {
        if (showCreateDialog) setShowCreateDialog(false);
        if (showCustomizeDialog) {
          setShowCustomizeDialog(false);
          setCustomizeTarget(null);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [showCreateDialog, showCustomizeDialog]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 smooth-scroll">
      {/* Success toast notification */}
      {successMessage && (
        <div
          role="alert"
          aria-live="polite"
          className="fixed top-4 right-4 z-50 max-w-md rounded-xl border border-emerald-500/40 bg-emerald-500/10 backdrop-blur-sm px-4 py-3 text-sm text-emerald-100 shadow-2xl shadow-emerald-500/20 animate-slide-up"
        >
          <div className="flex items-start gap-3">
            <CheckCircle className="h-5 w-5 text-emerald-400 flex-shrink-0 mt-0.5" aria-hidden="true" />
            <p className="flex-1">{successMessage}</p>
            <button
              onClick={() => setSuccessMessage(null)}
              className="text-emerald-300 hover:text-emerald-200 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500 rounded"
              aria-label="Dismiss notification"
            >
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}

      <main id="main-content" className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 xl:px-10 py-8 sm:py-12 lg:py-16 xl:py-20 space-y-8 sm:space-y-10 lg:space-y-12 xl:space-y-14 safe-area-inset">
        <header className="space-y-4 sm:space-y-6 animate-fade-in">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <p className="inline-flex items-center rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300 border border-emerald-500/30">
              <Sparkles className="h-3 w-3 mr-1.5" aria-hidden="true" />
              Landing Manager · Factory
            </p>
            <div className="flex items-center gap-3">
              <Tooltip content="Press ⌘/Ctrl+K to quickly create a new landing page">
                <div className="px-2.5 py-1 text-xs text-slate-400 bg-slate-800/50 border border-slate-700 rounded-md font-mono">
                  ⌘K
                </div>
              </Tooltip>
              <Tooltip content="This is a meta-scenario that generates complete landing page applications. Each generated scenario includes a public landing page, admin portal, A/B testing, and Stripe integration.">
                <HelpCircle className="h-5 w-5 text-slate-400 hover:text-slate-300 transition-colors" aria-label="About Landing Manager" />
              </Tooltip>
            </div>
          </div>
          <h1 className="text-3xl sm:text-4xl lg:text-5xl xl:text-6xl font-bold tracking-tight">
            Landing Page Factory
          </h1>
          <p className="text-base sm:text-lg xl:text-xl text-slate-300 max-w-3xl leading-relaxed">
            Create production-ready SaaS landing pages in under 60 seconds. Each generated scenario includes a complete stack: React UI, Go API, PostgreSQL schema, Stripe integration, A/B testing, and analytics dashboard.
          </p>
          <div className="inline-flex items-center gap-3 p-3 rounded-xl border border-blue-500/30 bg-gradient-to-r from-blue-500/10 to-purple-500/10">
            <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
              <Zap className="h-5 w-5 text-blue-300" aria-hidden="true" />
            </div>
            <p className="text-sm text-blue-100 font-medium">
              New scenarios are created in <code className="px-1.5 py-0.5 rounded bg-slate-900/60 text-emerald-300 font-mono text-xs">generated/</code> staging folder for safe testing
            </p>
          </div>

          <QuickStartGuide onCreateClick={handleOpenCreateDialog} />
        </header>

        <StatsOverview
          templates={templates}
          loadingTemplates={loadingTemplates}
          selectedTemplate={selectedTemplate}
        />

        <GeneratedScenariosList
          generated={generated}
          loadingGenerated={loadingGenerated}
          generatedError={generatedError}
          scenarioStatuses={scenarioStatuses}
          previewLinks={previewLinks}
          showLogs={showLogs}
          scenarioLogs={scenarioLogs}
          onRefresh={(scenarios, error, loading) => {
            setGenerated(scenarios);
            setGeneratedError(error);
            setLoadingGenerated(loading);
          }}
          onStartScenario={handleStartScenario}
          onStopScenario={handleStopScenario}
          onRestartScenario={handleRestartScenario}
          onToggleLogs={handleToggleLogs}
          onPromoteScenario={handlePromoteScenario}
          onDeleteScenario={handleDeleteScenario}
          onSelectScenario={() => {}} // No longer needed for scrolling
          onCreateClick={handleOpenCreateDialog}
          onCustomizeClick={handleOpenCustomizeDialog}
        />

        {/* Promotion Confirmation Dialog */}
        <PromoteDialog
          scenarioId={promoteDialogScenario}
          onClose={() => setPromoteDialogScenario(null)}
          onConfirm={confirmPromote}
        />

        {/* Footer */}
        <footer className="border-t border-white/10 pt-8 mt-12">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-slate-400">
            <p>Landing Manager Factory · Vrooli</p>
            <div className="flex items-center gap-4">
              <button
                onClick={() => setShowKeyboardHelp(true)}
                className="inline-flex items-center gap-2 px-3 py-1.5 text-xs hover:text-slate-300 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500 rounded"
                aria-label="View keyboard shortcuts"
              >
                <span>Keyboard Shortcuts</span>
              </button>
              <button
                onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
                className="inline-flex items-center gap-2 px-3 py-1.5 text-xs hover:text-slate-300 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500 rounded"
                aria-label="Scroll to top"
              >
                <span>Back to top</span>
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
                </svg>
              </button>
            </div>
          </div>
        </footer>
      </main>

      {/* Floating Action Button */}
      <FloatingCreateButton onClick={handleOpenCreateDialog} />

      {/* Create Scenario Dialog */}
      <CreateScenarioDialog
        isOpen={showCreateDialog}
        onClose={handleCloseCreateDialog}
        templates={templates}
        loadingTemplates={loadingTemplates}
        templatesError={templatesError}
        onRefreshTemplates={handleRefreshTemplates}
        onGenerate={handleGenerate}
        onStartScenario={handleStartScenario}
      />

      {/* Agent Customization Dialog */}
      <AgentCustomizationDialog
        isOpen={showCustomizeDialog}
        scenario={customizeTarget}
        onClose={handleCloseCustomizeDialog}
        onCustomize={handleCustomize}
      />

      {/* Keyboard Shortcuts Help Dialog */}
      <KeyboardShortcutsDialog isOpen={showKeyboardHelp} onClose={() => setShowKeyboardHelp(false)} />
    </div>
  );
}
