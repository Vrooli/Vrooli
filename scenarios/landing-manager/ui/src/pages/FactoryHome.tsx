import { useEffect, useMemo, useState } from 'react';
import { CheckCircle, Loader2, HelpCircle, Sparkles, Zap } from 'lucide-react';
import {
  generateScenario,
  getTemplate,
  listTemplates,
  listGeneratedScenarios,
  customizeScenario,
  promoteScenario,
  type GenerationResult,
  type Template,
  type GeneratedScenario,
  type CustomizeResult,
} from '../lib/api';
import { slugify } from '../lib/utils';
import { Tooltip } from '../components/Tooltip';
import { PromoteDialog } from '../components/PromoteDialog';
import { KeyboardShortcutsDialog } from '../components/KeyboardShortcutsDialog';
import { StatsOverview } from '../components/StatsOverview';
import { QuickStartGuide } from '../components/QuickStartGuide';
import { TemplateCatalog } from '../components/TemplateCatalog';
import { GenerationForm } from '../components/GenerationForm';
import { AgentCustomizationForm } from '../components/AgentCustomizationForm';
import { GeneratedScenariosList } from '../components/GeneratedScenariosList';
import { useScenarioLifecycle } from '../hooks/useScenarioLifecycle';

export default function FactoryHome() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [generated, setGenerated] = useState<GeneratedScenario[]>([]);
  const [loadingGenerated, setLoadingGenerated] = useState(true);
  const [generatedError, setGeneratedError] = useState<string | null>(null);

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [generating, setGenerating] = useState(false);
  const [lastResult, setLastResult] = useState<{ dryRun: boolean; result: GenerationResult } | null>(null);
  const [generateError, setGenerateError] = useState<string | null>(null);

  const [customizeSlug, setCustomizeSlug] = useState('');
  const [customizeBrief, setCustomizeBrief] = useState('');
  const [customizeAssets, setCustomizeAssets] = useState('');
  const [customizing, setCustomizing] = useState(false);
  const [customizeError, setCustomizeError] = useState<string | null>(null);
  const [customizeResult, setCustomizeResult] = useState<CustomizeResult | null>(null);

  // Lifecycle management using custom hook
  const lifecycle = useScenarioLifecycle();
  const { statuses: scenarioStatuses, previewLinks, showLogs, logs: scenarioLogs } = lifecycle;

  const [nameError, setNameError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [promoteDialogScenario, setPromoteDialogScenario] = useState<string | null>(null);
  const [showKeyboardHelp, setShowKeyboardHelp] = useState(false);

  const selectedTemplate = useMemo(
    () => templates?.find((t) => t?.id === selectedId) ?? templates?.[0] ?? null,
    [selectedId, templates],
  );

  useEffect(() => {
    async function loadTemplates() {
      try {
        setLoadingTemplates(true);
        const tpl = await listTemplates();
        setTemplates(tpl);
        setSelectedId((id) => id ?? tpl[0]?.id ?? null);
        setTemplatesError(null);

        // Hydrate the selected template with full metadata if needed
        if (tpl[0]?.id) {
          const full = await getTemplate(tpl[0].id);
          setTemplates([full, ...tpl.slice(1)]);
        }
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

  const handleNameChange = (value: string) => {
    setName(value);
    // Validate name
    if (value.trim().length === 0) {
      setNameError('Name is required');
    } else if (value.trim().length < 3) {
      setNameError('Name must be at least 3 characters');
    } else if (value.trim().length > 100) {
      setNameError('Name must be less than 100 characters');
    } else {
      setNameError(null);
    }
    const nextSlug = slugify(value);
    setSlug((prev) => {
      const prevAuto = slugify(name);
      if (!prev || prev === prevAuto) {
        return nextSlug;
      }
      return prev;
    });
  };

  const handleSlugChange = (value: string) => {
    const cleaned = slugify(value);
    setSlug(cleaned);
    // Validate slug
    if (cleaned.length === 0) {
      setSlugError('Slug is required');
    } else if (cleaned.length < 3) {
      setSlugError('Slug must be at least 3 characters');
    } else if (cleaned.length > 60) {
      setSlugError('Slug must be less than 60 characters');
    } else if (!/^[a-z][a-z0-9-]*[a-z0-9]$/.test(cleaned)) {
      setSlugError('Slug must start with a letter and end with a letter or number');
    } else {
      setSlugError(null);
    }
  };

  const handleGenerate = async (dryRun: boolean) => {
    if (!selectedTemplate) return;
    if (!name.trim() || !slug.trim()) {
      setGenerateError('Name and slug are required');
      return;
    }

    if (!selectedTemplate) {
      setGenerateError('No template selected');
      return;
    }

    try {
      setGenerating(true);
      setGenerateError(null);
      const result = await generateScenario(selectedTemplate.id, name.trim(), slug.trim(), { dry_run: dryRun });
      setLastResult({ dryRun, result });
      // Refresh generated scenarios list if we actually wrote files
      if (!dryRun) {
        const scenarios = await listGeneratedScenarios();
        setGenerated(scenarios);
        setSuccessMessage(`✅ Successfully generated "${name}" scenario!`);
        setTimeout(() => setSuccessMessage(null), 5000);
      } else {
        setSuccessMessage(`📋 Dry-run plan generated for "${name}"`);
        setTimeout(() => setSuccessMessage(null), 3000);
      }
    } catch (err) {
      setGenerateError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setGenerating(false);
    }
  };

  const handleCustomize = async () => {
    if (!customizeSlug.trim() || !customizeBrief.trim()) {
      setCustomizeError('Scenario slug and a brief are required');
      return;
    }

    try {
      setCustomizeError(null);
      setCustomizeResult(null);
      setCustomizing(true);
      const assets = customizeAssets
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const result = await customizeScenario(customizeSlug.trim(), customizeBrief.trim(), assets, true);
      setCustomizeResult(result);
    } catch (err) {
      setCustomizeError(err instanceof Error ? err.message : 'Failed to file customization request');
    } finally {
      setCustomizing(false);
    }
  };

  // Lifecycle handlers
  // Delegate lifecycle operations to the custom hook
  const handleStartScenario = lifecycle.startScenario;
  const handleStopScenario = lifecycle.stopScenario;
  const handleRestartScenario = lifecycle.restartScenario;

  const handlePromoteScenario = async (scenarioId: string) => {
    // Show confirmation dialog
    setPromoteDialogScenario(scenarioId);
  };

  const confirmPromote = async () => {
    if (!promoteDialogScenario) return;

    const scenarioId = promoteDialogScenario;
    setPromoteDialogScenario(null);

    try {
      const result = await promoteScenario(scenarioId);

      if (result.success) {
        // Remove from generated list since it's now in production
        // Lifecycle state will be cleaned up automatically via useEffect
        setGenerated((prev) => prev.filter(s => s.scenario_id !== scenarioId));
        setSuccessMessage(`Successfully promoted "${scenarioId}" to production at ${result.production_path}`);
      } else {
        setGenerateError(`Failed to promote: ${result.message}`);
      }
    } catch (err) {
      console.error('Failed to promote scenario:', err);
      setGenerateError(`Failed to promote: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleToggleLogs = lifecycle.toggleLogs;

  // Load scenario statuses when generated scenarios list changes
  useEffect(() => {
    if (generated.length > 0) {
      const scenarioIds = generated.map((s) => s.scenario_id);
      lifecycle.loadStatuses(scenarioIds);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generated]);

  // Keyboard shortcuts for power users
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl+Enter to generate (when form is valid)
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && !generating && selectedTemplate && name.trim() && slug.trim() && !nameError && !slugError) {
        e.preventDefault();
        handleGenerate(false);
      }
      // Cmd/Ctrl+Shift+Enter for dry-run
      if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'Enter' && !generating && selectedTemplate && name.trim() && slug.trim() && !nameError && !slugError) {
        e.preventDefault();
        handleGenerate(true);
      }
      // Cmd/Ctrl+R to refresh templates
      if ((e.metaKey || e.ctrlKey) && e.key === 'r' && !loadingTemplates) {
        e.preventDefault();
        (async () => {
          try {
            setLoadingTemplates(true);
            setTemplatesError(null);
            const tpl = await listTemplates();
            setTemplates(tpl);
            setSelectedId((id) => id ?? tpl[0]?.id ?? null);
          } catch (err) {
            setTemplatesError(err instanceof Error ? err.message : 'Failed to refresh templates');
          } finally {
            setLoadingTemplates(false);
          }
        })();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [generating, selectedTemplate, name, slug, nameError, slugError, loadingTemplates]);

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
            <Tooltip content="This is a meta-scenario that generates complete landing page applications. Each generated scenario includes a public landing page, admin portal, A/B testing, and Stripe integration.">
              <HelpCircle className="h-5 w-5 text-slate-400 hover:text-slate-300 transition-colors" aria-label="About Landing Manager" />
            </Tooltip>
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

          <QuickStartGuide />
        </header>

        <StatsOverview
          templates={templates}
          loadingTemplates={loadingTemplates}
          selectedTemplate={selectedTemplate}
        />

        <TemplateCatalog
          templates={templates}
          selectedTemplate={selectedTemplate}
          loadingTemplates={loadingTemplates}
          templatesError={templatesError}
          selectedId={selectedId}
          onSelectTemplate={setSelectedId}
          onRefreshTemplates={(newTemplates, error, loading) => {
            setTemplates(newTemplates);
            setTemplatesError(error);
            setLoadingTemplates(loading);
          }}
        />

        <GenerationForm
          selectedTemplate={selectedTemplate}
          name={name}
          slug={slug}
          nameError={nameError}
          slugError={slugError}
          generating={generating}
          generateError={generateError}
          lastResult={lastResult}
          onNameChange={handleNameChange}
          onSlugChange={handleSlugChange}
          onGenerate={handleGenerate}
          onStartScenario={handleStartScenario}
          onClearResult={() => {
            setName('');
            setSlug('');
            setLastResult(null);
            setGenerateError(null);
          }}
          onShowKeyboardHelp={() => setShowKeyboardHelp(true)}
          onClearError={() => setGenerateError(null)}
        />

        <AgentCustomizationForm
          generated={generated}
          loadingGenerated={loadingGenerated}
          customizeSlug={customizeSlug}
          customizeBrief={customizeBrief}
          customizeAssets={customizeAssets}
          customizing={customizing}
          customizeError={customizeError}
          customizeResult={customizeResult}
          onCustomizeSlugChange={setCustomizeSlug}
          onCustomizeBriefChange={setCustomizeBrief}
          onCustomizeAssetsChange={setCustomizeAssets}
          onCustomize={handleCustomize}
          onResetForm={() => {
            setCustomizeSlug('');
            setCustomizeBrief('');
            setCustomizeAssets('');
            setCustomizeError(null);
            setCustomizeResult(null);
          }}
          onClearError={() => setCustomizeError(null)}
          onSelectScenario={setCustomizeSlug}
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
          onSelectScenario={setCustomizeSlug}
        />

        {/* Promotion Confirmation Dialog */}
        <PromoteDialog
          scenarioId={promoteDialogScenario}
          onClose={() => setPromoteDialogScenario(null)}
          onConfirm={confirmPromote}
        />

        {/* [REQ:A11Y-FOOTER] Footer with back-to-top link */}
        <footer className="border-t border-white/10 pt-8 mt-12">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4 text-sm text-slate-400">
            <p>Landing Manager Factory · Vrooli</p>
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
        </footer>
      </main>

      {/* Keyboard Shortcuts Help Dialog */}
      <KeyboardShortcutsDialog isOpen={showKeyboardHelp} onClose={() => setShowKeyboardHelp(false)} />
    </div>
  );
}
