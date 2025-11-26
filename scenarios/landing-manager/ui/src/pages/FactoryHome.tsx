import { useEffect, useMemo, useState } from 'react';
import { CheckCircle, Loader2, RefreshCcw, Rocket, AlertCircle, ExternalLink, HelpCircle, Sparkles, FileText, Zap, Play, Square, RotateCw, FileOutput, Monitor, Copy, Check, Globe, Settings, Calendar, X, Keyboard } from 'lucide-react';
import {
  generateScenario,
  getTemplate,
  listTemplates,
  listGeneratedScenarios,
  customizeScenario,
  startScenario,
  stopScenario,
  restartScenario,
  getScenarioStatus,
  getScenarioLogs,
  getPreviewLinks,
  promoteScenario,
  type GenerationResult,
  type Template,
  type GeneratedScenario,
  type CustomizeResult,
  type PreviewLinks,
} from '../lib/api';

function slugify(input: string) {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
    .slice(0, 60);
}

// Tooltip component for contextual help
function Tooltip({ children, content }: { children: React.ReactNode; content: string }) {
  const [show, setShow] = useState(false);
  return (
    <div className="relative inline-block">
      <button
        type="button"
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onFocus={() => setShow(true)}
        onBlur={() => setShow(false)}
        className="inline-flex items-center"
        aria-label={content}
      >
        {children}
      </button>
      {show && (
        <div
          role="tooltip"
          className="absolute z-10 left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-2 text-xs text-slate-100 bg-slate-800 border border-slate-700 rounded-lg shadow-lg w-64 pointer-events-none"
        >
          {content}
          <div className="absolute left-1/2 -translate-x-1/2 top-full -mt-1 border-4 border-transparent border-t-slate-800"></div>
        </div>
      )}
    </div>
  );
}

// Skeleton loader for template cards
function TemplateCardSkeleton() {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-5 animate-pulse" aria-hidden="true">
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex-1 space-y-2">
          <div className="h-3 w-16 bg-slate-700/50 rounded"></div>
          <div className="h-5 w-40 bg-slate-700/50 rounded"></div>
        </div>
        <div className="h-5 w-10 bg-slate-700/50 rounded-full"></div>
      </div>
      <div className="space-y-2 mt-2">
        <div className="h-3 w-full bg-slate-700/50 rounded"></div>
        <div className="h-3 w-3/4 bg-slate-700/50 rounded"></div>
      </div>
    </div>
  );
}

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

  // Lifecycle management state
  const [scenarioStatuses, setScenarioStatuses] = useState<Record<string, { running: boolean; loading: boolean }>>({});
  const [showLogs, setShowLogs] = useState<Record<string, boolean>>({});
  const [scenarioLogs, setScenarioLogs] = useState<Record<string, string>>({});
  const [previewLinks, setPreviewLinks] = useState<Record<string, PreviewLinks>>({});
  const [copiedSlug, setCopiedSlug] = useState(false);
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
  const handleStartScenario = async (scenarioId: string) => {
    try {
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true } }));
      await startScenario(scenarioId);
      const status = await getScenarioStatus(scenarioId);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: status.running, loading: false } }));

      // Fetch preview links after starting
      if (status.running) {
        try {
          const links = await getPreviewLinks(scenarioId);
          setPreviewLinks((prev) => ({ ...prev, [scenarioId]: links }));
        } catch {
          // Preview links may not be available immediately, ignore error
        }
      }
    } catch (err) {
      console.error('Failed to start scenario:', err);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: false, loading: false } }));
    }
  };

  const handleStopScenario = async (scenarioId: string) => {
    try {
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true } }));
      await stopScenario(scenarioId);
      const status = await getScenarioStatus(scenarioId);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: status.running, loading: false } }));
    } catch (err) {
      console.error('Failed to stop scenario:', err);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: true, loading: false } }));
    }
  };

  const handleRestartScenario = async (scenarioId: string) => {
    try {
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true } }));
      await restartScenario(scenarioId);
      const status = await getScenarioStatus(scenarioId);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: status.running, loading: false } }));

      // Refresh preview links after restarting
      if (status.running) {
        try {
          const links = await getPreviewLinks(scenarioId);
          setPreviewLinks((prev) => ({ ...prev, [scenarioId]: links }));
        } catch {
          // Ignore
        }
      }
    } catch (err) {
      console.error('Failed to restart scenario:', err);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: false, loading: false } }));
    }
  };

  const handlePromoteScenario = async (scenarioId: string) => {
    // Show confirmation dialog
    setPromoteDialogScenario(scenarioId);
  };

  const confirmPromote = async () => {
    if (!promoteDialogScenario) return;

    const scenarioId = promoteDialogScenario;
    setPromoteDialogScenario(null);

    try {
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: prev[scenarioId]?.running ?? false, loading: true } }));
      const result = await promoteScenario(scenarioId);

      if (result.success) {
        // Remove from generated list since it's now in production
        setGenerated((prev) => prev.filter(s => s.scenario_id !== scenarioId));
        // Clear status and links
        setScenarioStatuses((prev) => {
          const newStatuses = { ...prev };
          delete newStatuses[scenarioId];
          return newStatuses;
        });
        setPreviewLinks((prev) => {
          const newLinks = { ...prev };
          delete newLinks[scenarioId];
          return newLinks;
        });
        setSuccessMessage(`Successfully promoted "${scenarioId}" to production at ${result.production_path}`);
      } else {
        setGenerateError(`Failed to promote: ${result.message}`);
        setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: false, loading: false } }));
      }
    } catch (err) {
      console.error('Failed to promote scenario:', err);
      setGenerateError(`Failed to promote: ${err instanceof Error ? err.message : 'Unknown error'}`);
      setScenarioStatuses((prev) => ({ ...prev, [scenarioId]: { running: false, loading: false } }));
    }
  };

  const handleToggleLogs = async (scenarioId: string) => {
    const show = !showLogs[scenarioId];
    setShowLogs((prev) => ({ ...prev, [scenarioId]: show }));

    if (show && !scenarioLogs[scenarioId]) {
      try {
        const logs = await getScenarioLogs(scenarioId, 100);
        setScenarioLogs((prev) => ({ ...prev, [scenarioId]: logs.logs }));
      } catch (err) {
        console.error('Failed to load logs:', err);
        setScenarioLogs((prev) => ({ ...prev, [scenarioId]: 'Failed to load logs' }));
      }
    }
  };

  // Load scenario statuses on mount and when generated list changes
  useEffect(() => {
    async function loadStatuses() {
      for (const scenario of generated) {
        try {
          const status = await getScenarioStatus(scenario.scenario_id);
          setScenarioStatuses((prev) => ({ ...prev, [scenario.scenario_id]: { running: status.running, loading: false } }));

          // Load preview links if running
          if (status.running) {
            try {
              const links = await getPreviewLinks(scenario.scenario_id);
              setPreviewLinks((prev) => ({ ...prev, [scenario.scenario_id]: links }));
            } catch {
              // Ignore
            }
          }
        } catch {
          // Set default status on error
          setScenarioStatuses((prev) => ({ ...prev, [scenario.scenario_id]: { running: false, loading: false } }));
        }
      }
    }

    if (generated.length > 0) {
      loadStatuses();
    }
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

          {/* Quick start guide for first-time users */}
          <div className="rounded-xl border border-emerald-500/30 bg-gradient-to-br from-emerald-500/10 to-blue-500/10 p-5 sm:p-6" role="region" aria-label="Quick start guide">
            <div className="flex items-start gap-4">
              <div className="flex-shrink-0 w-10 h-10 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center">
                <Rocket className="h-5 w-5 text-emerald-300" aria-hidden="true" />
              </div>
              <div className="flex-1 space-y-4">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                  <h2 className="text-lg font-bold text-emerald-100">Get Started in 3 Steps</h2>
                  <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-bold bg-emerald-500/30 border border-emerald-500/50 text-emerald-200 rounded-full">
                    <Monitor className="h-3 w-3" aria-hidden="true" />
                    <span>100% UI · No Terminal Required</span>
                  </span>
                </div>
                <div className="grid gap-3">
                  <div className="flex items-start gap-3">
                    <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">1</div>
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-emerald-100 mb-1">Select a Template</p>
                      <p className="text-xs text-emerald-200/80">Browse templates below and click to select one (already done if you see a green checkmark)</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">2</div>
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-emerald-100 mb-1">Generate Your Landing Page</p>
                      <p className="text-xs text-emerald-200/80">Enter a name, click "Generate Now" — your scenario appears instantly in the staging area</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-3">
                    <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">3</div>
                    <div className="flex-1">
                      <p className="text-sm font-semibold text-emerald-100 mb-1">Launch & Test</p>
                      <p className="text-xs text-emerald-200/80">Click "Start" on your scenario → get instant access links to live landing page + admin dashboard</p>
                    </div>
                  </div>
                </div>
                <div className="flex items-start gap-2 pt-3 border-t border-emerald-500/20">
                  <Sparkles className="h-4 w-4 text-blue-300 mt-0.5 flex-shrink-0" aria-hidden="true" />
                  <p className="text-xs text-blue-200 font-medium">
                    <strong>Iterate risk-free:</strong> All new scenarios start in a staging folder. Test, refine, and customize before moving to production.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Stats overview cards */}
        <div className="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3" role="region" aria-label="Overview statistics">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 xl:p-7 space-y-3" role="article">
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium text-slate-400">Templates Available</div>
              <Tooltip content="Pre-built templates with different landing page configurations. Each includes a full stack: React UI, Go API, PostgreSQL schema, and Stripe integration.">
                <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
              </Tooltip>
            </div>
            <div className="flex items-center gap-2 text-2xl xl:text-3xl font-bold" aria-live="polite">
              {loadingTemplates ? (
                <Loader2 className="h-5 w-5 xl:h-6 xl:w-6 animate-spin text-emerald-300" aria-label="Loading templates" />
              ) : (
                <CheckCircle className="h-5 w-5 xl:h-6 xl:w-6 text-emerald-300" aria-label="Templates loaded successfully" />
              )}
              <span>{templates.length}</span>
            </div>
            <p className="text-xs sm:text-sm xl:text-base text-slate-400 leading-relaxed">
              Browse templates below to see features, sections, and customization options
            </p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 xl:p-7 space-y-3" role="article">
            <div className="flex items-center justify-between">
              <div className="text-sm font-medium text-slate-400">Generation Methods</div>
              <Tooltip content="Create scenarios via the UI form below or using CLI commands. Both support dry-run mode for planning before generation.">
                <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
              </Tooltip>
            </div>
            <div className="flex items-center gap-2 text-2xl xl:text-3xl font-bold">
              <FileText className="h-5 w-5 xl:h-6 xl:w-6 text-blue-400" aria-hidden="true" />
              <span>CLI + UI</span>
            </div>
            <div className="space-y-2">
              <p className="text-xs sm:text-sm xl:text-base text-slate-400">
                Use buttons below or CLI:
              </p>
              <code className="hidden sm:block text-[10px] lg:text-xs xl:text-sm rounded-lg bg-slate-900 px-2 py-1.5 xl:px-3 xl:py-2 border border-white/10 overflow-x-auto whitespace-nowrap">
                landing-manager generate {selectedTemplate?.id || 'template-id'} --name "..." --slug "..."
              </code>
            </div>
          </div>

          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4 sm:p-6 xl:p-7 space-y-3 sm:col-span-2 lg:col-span-1" role="alert" aria-live="polite">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4 xl:h-5 xl:w-5 text-amber-400 flex-shrink-0" aria-hidden="true" />
              <div className="text-sm xl:text-base font-medium text-amber-200">Important Note</div>
            </div>
            <p className="text-sm sm:text-base xl:text-lg font-semibold text-amber-100">Runtime lives in generated scenarios</p>
            <p className="text-xs sm:text-sm xl:text-base text-amber-100/90 leading-relaxed">
              This factory creates scenarios. The admin portal, analytics, A/B testing, and Stripe integration run inside each generated scenario, not here.
            </p>
          </div>
        </div>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden" data-testid="template-catalog" aria-labelledby="template-catalog-heading">
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent"></div>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="flex items-center gap-2">
              <h2 id="template-catalog-heading" className="text-xl sm:text-2xl font-semibold">Template Catalog</h2>
              <Tooltip content="Choose a template as the starting point for your landing page. Each template includes pre-configured sections, metrics tracking, and Stripe integration.">
                <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
              </Tooltip>
            </div>
            <button
              data-testid="refresh-templates-button"
              className="group touch-target inline-flex items-center gap-2 px-3 py-2 text-sm text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-all duration-250 hover:scale-105 active:scale-95 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950 disabled:opacity-50 disabled:cursor-not-allowed"
              onClick={async () => {
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
              }}
              disabled={loadingTemplates}
              aria-label="Refresh template list"
            >
              <RefreshCcw className={`h-4 w-4 transition-transform duration-300 ${loadingTemplates ? 'animate-spin' : 'group-hover:rotate-180'}`} aria-hidden="true" />
              <span className="hidden sm:inline">Refresh</span>
            </button>
          </div>

          {templatesError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{templatesError}</div>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4" data-testid="template-grid" role="list" aria-label="Available templates">
            {loadingTemplates && (
              <>
                <TemplateCardSkeleton />
                <TemplateCardSkeleton />
              </>
            )}
            {!loadingTemplates && templates.length === 0 && !templatesError && (
              <div className="col-span-full rounded-xl border border-white/10 bg-slate-900/40 p-8 sm:p-12 text-center space-y-4" role="status">
                <div className="flex justify-center">
                  <FileText className="h-12 w-12 text-slate-600" aria-hidden="true" />
                </div>
                <div className="space-y-2">
                  <h3 className="text-lg font-semibold text-slate-300">No templates available</h3>
                  <p className="text-sm text-slate-400 max-w-md mx-auto">
                    Templates will appear here once they're loaded. Try refreshing or check your API connection.
                  </p>
                </div>
                <button
                  onClick={async () => {
                    try {
                      setLoadingTemplates(true);
                      const tpl = await listTemplates();
                      setTemplates(tpl);
                      setSelectedId((id) => id ?? tpl[0]?.id ?? null);
                    } catch (err) {
                      setTemplatesError(err instanceof Error ? err.message : 'Failed to load templates');
                    } finally {
                      setLoadingTemplates(false);
                    }
                  }}
                  className="inline-flex items-center gap-2 px-4 py-2 text-sm bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 rounded-lg hover:bg-emerald-500/20 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <RefreshCcw className="h-4 w-4" aria-hidden="true" />
                  Try again
                </button>
              </div>
            )}
            {!loadingTemplates &&
              templates?.filter(Boolean).map((tpl) => (
                <button
                  key={tpl.id}
                  role="listitem"
                  data-testid={`template-card-${tpl.id}`}
                  onClick={() => setSelectedId(tpl.id)}
                  className={`rounded-xl border p-4 sm:p-5 text-left transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950 ${
                    selectedTemplate?.id === tpl.id
                      ? 'border-emerald-400/60 bg-emerald-500/5 shadow-lg shadow-emerald-500/10 scale-[1.02]'
                      : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10 hover:scale-[1.01]'
                  }`}
                  aria-current={selectedTemplate?.id === tpl.id ? 'true' : 'false'}
                  aria-label={`Select ${tpl.name} template, version ${tpl.version}${selectedTemplate?.id === tpl.id ? ' (currently selected)' : ''}`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="text-xs font-medium text-slate-400 uppercase tracking-wide mb-1">Template</div>
                      <div className="text-base sm:text-lg font-semibold truncate" data-testid={`template-name-${tpl.id}`}>{tpl.name}</div>
                    </div>
                    <span className="flex-shrink-0 text-xs text-slate-400 border border-white/10 rounded-full px-2 py-1" data-testid={`template-version-${tpl.id}`}>
                      v{tpl.version}
                    </span>
                  </div>
                  <p className="text-xs sm:text-sm text-slate-300 mt-2 line-clamp-2 leading-relaxed">{tpl.description}</p>
                  {selectedTemplate?.id === tpl.id && (
                    <div className="mt-3 pt-3 border-t border-emerald-500/20">
                      <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-300">
                        <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                        Selected
                      </span>
                    </div>
                  )}
                </button>
              ))}
          </div>

          {selectedTemplate && (
            <div className="rounded-xl border border-white/10 bg-slate-900/40 p-5 space-y-3" data-testid="selected-template-details">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm text-slate-400">Selected Template</div>
                  <div className="text-xl font-semibold" data-testid="selected-template-name">{selectedTemplate.name}</div>
                </div>
                <div className="text-xs text-slate-400" data-testid="selected-template-id">ID: {selectedTemplate.id}</div>
              </div>
              <p className="text-sm text-slate-300">{selectedTemplate.description}</p>

              <div className="grid md:grid-cols-3 gap-4">
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Sections</div>
                  <div className="text-sm text-slate-200 space-y-1">
                    {selectedTemplate.sections
                      ? Object.keys(selectedTemplate.sections).map((key) => <div key={key}>• {key}</div>)
                      : 'Defined in template payload'}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Metrics Hooks</div>
                  <div className="text-sm text-slate-200 space-y-1">
                    {selectedTemplate.metrics_hooks?.length
                      ? selectedTemplate.metrics_hooks.map((hook, idx) => <div key={idx}>• {(hook as { name?: string; id?: string }).name || (hook as { name?: string; id?: string }).id || 'hook'}</div>)
                      : 'page_view, scroll_depth, click, form_submit, conversion'}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Customization Schema</div>
                  <div className="text-sm text-slate-200 space-y-1">
                    {(selectedTemplate.customization_schema && Object.keys(selectedTemplate.customization_schema).length > 0)
                      ? Object.keys(selectedTemplate.customization_schema).map((key) => <div key={key}>• {key}</div>)
                      : 'Branding, SEO, Stripe, sections'}
                  </div>
                </div>
              </div>
            </div>
          )}
        </section>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden" data-testid="generation-form" aria-labelledby="generation-form-heading">
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-blue-500/20 to-transparent"></div>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="flex items-center gap-2">
              <h2 id="generation-form-heading" className="text-xl sm:text-2xl font-semibold">Generate Scenario</h2>
              <Tooltip content="Create a new landing page scenario from the selected template. Dry-run shows what will be generated without creating files. Generate creates the actual scenario folder.">
                <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
              </Tooltip>
            </div>
            {generating && (
              <div className="inline-flex items-center gap-2 text-sm text-emerald-300 bg-emerald-500/10 border border-emerald-500/30 rounded-lg px-3 py-1.5" aria-live="polite">
                <Loader2 className="h-4 w-4 animate-spin" data-testid="generation-loading" aria-label="Generating" />
                <span className="hidden sm:inline font-medium">Generating scenario...</span>
              </div>
            )}
          </div>

          {!selectedTemplate && (
            <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-100 flex items-start gap-3" role="alert">
              <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0 text-amber-400" aria-hidden="true" />
              <p>Please select a template from the catalog above to enable generation.</p>
            </div>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="generation-name-input" className="block text-sm font-medium text-slate-300 mb-2">
                Scenario Name
                <Tooltip content="Human-readable name for your landing page. This will be used in the UI and documentation.">
                  <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
                </Tooltip>
              </label>
              <input
                id="generation-name-input"
                data-testid="generation-name-input"
                type="text"
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                className={`w-full rounded-lg border px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 bg-slate-900 focus:ring-2 outline-none transition-colors ${
                  nameError ? 'border-red-500/50 focus:border-red-400 focus:ring-red-500/20' : 'border-white/10 focus:border-emerald-400 focus:ring-emerald-500/20'
                }`}
                placeholder="My Awesome Product"
                required
                aria-required="true"
                aria-invalid={nameError ? 'true' : 'false'}
                aria-describedby={nameError ? 'name-error' : name ? 'name-helper' : undefined}
              />
              {nameError && name.trim().length > 0 && (
                <p id="name-error" className="mt-1.5 text-xs text-red-400 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" aria-hidden="true" />
                  {nameError}
                </p>
              )}
              {!nameError && name && (
                <p id="name-helper" className="mt-1.5 text-xs text-slate-400">
                  Slug will be: <code className="px-1 py-0.5 rounded bg-slate-800">{slug || slugify(name)}</code>
                </p>
              )}
              <p className="mt-1.5 text-xs text-slate-500 text-right">
                {name.length}/100 characters
              </p>
            </div>
            <div>
              <label htmlFor="generation-slug-input" className="block text-sm font-medium text-slate-300 mb-2">
                Scenario Slug
                <Tooltip content="URL-safe identifier for the scenario folder. Auto-generated from name, but you can customize it. Only lowercase letters, numbers, and hyphens allowed.">
                  <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
                </Tooltip>
              </label>
              <input
                id="generation-slug-input"
                data-testid="generation-slug-input"
                type="text"
                value={slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                className={`w-full rounded-lg border px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 font-mono bg-slate-900 focus:ring-2 outline-none transition-colors ${
                  slugError ? 'border-red-500/50 focus:border-red-400 focus:ring-red-500/20' : 'border-white/10 focus:border-emerald-400 focus:ring-emerald-500/20'
                }`}
                placeholder="my-awesome-product"
                pattern="[a-z0-9-]+"
                required
                aria-required="true"
                aria-invalid={slugError ? 'true' : 'false'}
                aria-describedby={slugError ? 'slug-error' : 'slug-helper'}
              />
              {slugError && slug.trim().length > 0 && (
                <p id="slug-error" className="mt-1.5 text-xs text-red-400 flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" aria-hidden="true" />
                  {slugError}
                </p>
              )}
              {!slugError && (
                <p id="slug-helper" className="mt-1.5 text-xs text-slate-400">
                  Folder: <code className="px-1 py-0.5 rounded bg-slate-800">generated/{slug || 'your-slug'}/</code>
                </p>
              )}
              <p className="mt-1.5 text-xs text-slate-500 text-right">
                {slug.length}/60 characters
              </p>
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex flex-col sm:flex-row gap-3">
              <button
                data-testid="dry-run-button"
                onClick={() => handleGenerate(true)}
                className="group touch-target inline-flex items-center justify-center gap-2 rounded-lg border border-white/15 bg-white/5 px-4 py-2.5 text-sm font-medium hover:border-emerald-300 hover:bg-white/10 transition-all duration-250 hover:scale-105 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
                disabled={generating || !selectedTemplate || !name.trim() || !slug.trim() || !!nameError || !!slugError}
                aria-label="Preview generation plan without creating files. Keyboard shortcut: Cmd+Shift+Enter"
                title={nameError || slugError || (!name.trim() ? 'Please enter a name' : !slug.trim() ? 'Please enter a slug' : !selectedTemplate ? 'Please select a template' : 'Cmd/Ctrl+Shift+Enter')}
              >
                {generating ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <RefreshCcw className="h-4 w-4 group-hover:rotate-180 transition-transform duration-300" aria-hidden="true" />}
                <span>Dry-run (preview only)</span>
              </button>
              <button
                data-testid="generate-button"
                onClick={() => handleGenerate(false)}
                className="group touch-target inline-flex items-center justify-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2.5 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-all duration-250 hover:scale-105 active:scale-95 hover:shadow-lg hover:shadow-emerald-500/20 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:scale-100 disabled:hover:shadow-none focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
                disabled={generating || !selectedTemplate || !name.trim() || !slug.trim() || !!nameError || !!slugError}
                aria-label="Generate scenario and create files. Keyboard shortcut: Cmd+Enter"
                title={nameError || slugError || (!name.trim() ? 'Please enter a name' : !slug.trim() ? 'Please enter a slug' : !selectedTemplate ? 'Please select a template' : 'Cmd/Ctrl+Enter')}
              >
                {generating ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Rocket className="h-4 w-4 group-hover:translate-y-[-2px] transition-transform" aria-hidden="true" />}
                <span>Generate Now</span>
              </button>
            </div>
            <div className="flex items-center justify-between">
              <p className="text-xs text-slate-500 flex items-center gap-1.5">
                <Zap className="h-3 w-3" aria-hidden="true" />
                <span className="hidden sm:inline">Shortcuts: <kbd className="px-1.5 py-0.5 rounded bg-slate-800 border border-slate-700 font-mono text-[10px]">⌘/Ctrl+Enter</kbd> to generate, <kbd className="px-1.5 py-0.5 rounded bg-slate-800 border border-slate-700 font-mono text-[10px]">⌘/Ctrl+Shift+Enter</kbd> for dry-run</span>
                <span className="sm:hidden">Keyboard shortcuts available</span>
              </p>
              <button
                onClick={() => setShowKeyboardHelp(true)}
                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-slate-400 hover:text-slate-300 bg-slate-800/50 hover:bg-slate-800 border border-slate-700 rounded-md transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
                aria-label="View all keyboard shortcuts"
              >
                <HelpCircle className="h-3.5 w-3.5" aria-hidden="true" />
                <span className="hidden sm:inline">Keyboard Help</span>
                <span className="sm:hidden">?</span>
              </button>
            </div>
          </div>

          {generateError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm flex items-start gap-3" role="alert">
              <AlertCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" aria-hidden="true" />
              <div className="flex-1 space-y-2">
                <p className="text-red-200 font-medium">{generateError}</p>
                <div className="flex flex-wrap gap-2">
                  <button
                    onClick={() => setGenerateError(null)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-red-500/20 hover:bg-red-500/30 border border-red-500/40 rounded-md text-red-200 transition-colors"
                  >
                    Dismiss
                  </button>
                  {!selectedTemplate && (
                    <button
                      onClick={() => {
                        const catalogElement = document.querySelector('[data-testid="template-catalog"]');
                        catalogElement?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                      }}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-emerald-500/20 hover:bg-emerald-500/30 border border-emerald-500/40 rounded-md text-emerald-200 transition-colors"
                    >
                      Select Template
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}

          {lastResult && (
            <div className="rounded-xl border border-white/10 bg-slate-900/50 p-5 space-y-3" data-testid="generation-result">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-5 w-5 text-emerald-300" />
                  <div className="text-sm text-slate-300" data-testid="generation-result-message">
                    {lastResult.dryRun ? 'Dry-run plan generated' : 'Scenario generated'}
                  </div>
                </div>
                <span className="text-xs text-slate-400" data-testid="generation-result-status">Status: {lastResult.result.status}</span>
              </div>

              <div className="grid md:grid-cols-2 gap-4 text-sm text-slate-200">
                <div className="space-y-1">
                  <div className="text-slate-400 text-xs uppercase">Scenario</div>
                  <div className="font-semibold">{lastResult.result.name}</div>
                  <div className="text-slate-400 text-xs">Slug: {lastResult.result.scenario_id}</div>
                  <div className="text-slate-400 text-xs">Template: {lastResult.result.template}</div>
                </div>
                <div className="space-y-1">
                  <div className="text-slate-400 text-xs uppercase">Output path</div>
                  <div className="font-mono text-emerald-200 break-all">{lastResult.result.path}</div>
                </div>
              </div>

              {lastResult.result.plan?.paths && (
                <div className="space-y-2">
                  <div className="text-xs text-slate-400 uppercase">Planned paths</div>
                  <div className="text-xs text-slate-200 bg-slate-900 rounded-lg border border-white/10 p-3 grid gap-1">
                    {lastResult.result.plan.paths.map((p) => (
                      <div key={p} className="font-mono">{p}</div>
                    ))}
                  </div>
                </div>
              )}

              {lastResult.result.next_steps && (
                <div className="space-y-1">
                  <div className="text-xs text-slate-400 uppercase">Next steps</div>
                  <ul className="list-disc list-inside text-sm text-slate-200 space-y-1">
                    {lastResult.result.next_steps.map((step, idx) => (
                      <li key={idx}>{step}</li>
                    ))}
                  </ul>
                </div>
              )}

              {!lastResult.dryRun && (
                <div className="space-y-3">
                  <div className="flex flex-col sm:flex-row gap-2">
                    <button
                      onClick={async () => {
                        await handleStartScenario(lastResult.result.scenario_id);
                        // Scroll to generated scenarios
                        const generatedSection = document.querySelector('[data-testid="generated-scenarios"]');
                        generatedSection?.scrollIntoView({ behavior: 'smooth', block: 'start' });
                      }}
                      className="inline-flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium bg-emerald-500/10 border border-emerald-500/40 text-emerald-300 rounded-lg hover:bg-emerald-500/20 hover:border-emerald-400 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
                    >
                      <Play className="h-4 w-4" aria-hidden="true" />
                      Start Now & View
                    </button>
                    <button
                      onClick={() => {
                        setName('');
                        setSlug('');
                        setLastResult(null);
                        setGenerateError(null);
                      }}
                      className="inline-flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium border border-white/15 bg-white/5 text-slate-300 rounded-lg hover:bg-white/10 transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500"
                    >
                      <Rocket className="h-4 w-4" aria-hidden="true" />
                      Generate Another
                    </button>
                  </div>
                  <div className="text-xs text-slate-400 space-y-1">
                    <p className="flex items-start gap-2">
                      <AlertCircle className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" aria-hidden="true" />
                      <span>Scenario created in staging area. Click <strong className="text-slate-300">Start Now</strong> to launch immediately, or find it in Generated Scenarios below.</span>
                    </p>
                  </div>
                </div>
              )}
            </div>
          )}
        </section>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden" data-testid="agent-customization-form" aria-labelledby="agent-customization-heading">
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-purple-500/20 to-transparent"></div>
          <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2">
                <h2 id="agent-customization-heading" className="text-xl sm:text-2xl font-semibold">Agent Customization</h2>
                <Tooltip content="Request AI-powered customization for a generated landing page. The agent will modify content, styling, and structure based on your brief.">
                  <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
                </Tooltip>
              </div>
              <p className="text-xs sm:text-sm text-slate-400 max-w-2xl leading-relaxed">
                Files an issue in app-issue-tracker and triggers an AI agent to customize your landing page based on your brief and assets.
              </p>
            </div>
            {customizing && (
              <div className="flex items-center gap-2 text-sm text-emerald-300" aria-live="polite">
                <Loader2 className="h-4 w-4 animate-spin" data-testid="customization-loading" aria-label="Customizing" />
                <span className="hidden sm:inline">Customizing...</span>
              </div>
            )}
          </div>

          {generated.length === 0 && !loadingGenerated && (
            <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-100 flex items-start gap-3" role="alert">
              <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0 text-amber-400" aria-hidden="true" />
              <div>
                <p className="font-medium mb-1">No scenarios available to customize</p>
                <p className="text-xs text-amber-200/80">Generate a landing page scenario first, then come back here to customize it with AI assistance.</p>
              </div>
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="customize-slug-input" className="block text-sm font-medium text-slate-300 mb-2">
                Scenario Slug
                <Tooltip content="The slug of the scenario you want to customize. Must be a generated scenario from the list above.">
                  <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
                </Tooltip>
              </label>
              {generated.length > 0 ? (
                <select
                  id="customize-slug-input"
                  data-testid="customize-slug-input"
                  value={customizeSlug}
                  onChange={(e) => setCustomizeSlug(e.target.value)}
                  className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 font-mono focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors"
                  aria-describedby="customize-slug-helper"
                >
                  <option value="">Select a scenario...</option>
                  {generated.map((scenario) => (
                    <option key={scenario.scenario_id} value={scenario.scenario_id}>
                      {scenario.name} ({scenario.scenario_id})
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  id="customize-slug-input"
                  data-testid="customize-slug-input"
                  value={customizeSlug}
                  onChange={(e) => setCustomizeSlug(slugify(e.target.value))}
                  className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 font-mono focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors"
                  placeholder="my-landing-page"
                  aria-describedby="customize-slug-helper"
                  disabled
                />
              )}
              <p id="customize-slug-helper" className="mt-1.5 text-xs text-slate-400">
                {generated.length > 0 ? 'Choose from your generated scenarios' : 'Generate a scenario first to customize it'}
              </p>
            </div>
            <div>
              <label htmlFor="customize-assets-input" className="block text-sm font-medium text-slate-300 mb-2">
                Assets (Optional)
                <Tooltip content="Comma-separated list of file paths or URLs to assets like logos, images, or brand guidelines for the agent to use.">
                  <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
                </Tooltip>
              </label>
              <input
                id="customize-assets-input"
                data-testid="customize-assets-input"
                value={customizeAssets}
                onChange={(e) => setCustomizeAssets(e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors"
                placeholder="logo.svg, hero-image.png"
                aria-describedby="customize-assets-helper"
              />
              <p id="customize-assets-helper" className="mt-1.5 text-xs text-slate-400">
                Leave empty if not using custom assets
              </p>
            </div>
          </div>

          <div>
            <label htmlFor="customize-brief-input" className="block text-sm font-medium text-slate-300 mb-2">
              Customization Brief
              <Tooltip content="Describe your goals, target audience, value proposition, desired tone, call-to-action, and success metrics. The more specific, the better the results.">
                <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500 hover:text-slate-400" />
              </Tooltip>
            </label>
            <textarea
              id="customize-brief-input"
              data-testid="customize-brief-input"
              value={customizeBrief}
              onChange={(e) => setCustomizeBrief(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 outline-none transition-colors min-h-[120px] resize-y"
              placeholder="Example: Target SaaS founders looking for rapid launch. Professional but friendly tone. Emphasize time savings and built-in features. CTA: Start Free Trial. Success metric: 10% conversion rate."
              required
              aria-required="true"
              aria-invalid={!customizeBrief.trim() && customizeError ? 'true' : 'false'}
              aria-describedby="customize-brief-helper"
            />
            <div className="flex items-center justify-between mt-1.5">
              <p id="customize-brief-helper" className="text-xs text-slate-400">
                Be specific about goals, audience, tone, and desired outcomes
              </p>
              <p className="text-xs text-slate-500">
                {customizeBrief.length} characters
              </p>
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              data-testid="customize-button"
              onClick={handleCustomize}
              className="inline-flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              disabled={customizing || !customizeSlug.trim() || !customizeBrief.trim()}
              title={!customizeSlug.trim() ? 'Select a scenario first' : !customizeBrief.trim() ? 'Add a customization brief' : 'File issue and trigger AI agent'}
            >
              {customizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
              File issue & trigger agent
            </button>
            <button
              onClick={() => {
                setCustomizeSlug('');
                setCustomizeBrief('');
                setCustomizeAssets('');
                setCustomizeError(null);
                setCustomizeResult(null);
              }}
              className="inline-flex items-center gap-2 rounded-lg border border-white/15 bg-white/5 px-4 py-2 text-sm hover:border-white/25 transition-colors"
            >
              <RefreshCcw className="h-4 w-4" />
              Reset form
            </button>
          </div>

          {customizeError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-sm flex items-start gap-3" role="alert">
              <AlertCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" aria-hidden="true" />
              <div className="flex-1 space-y-2">
                <p className="text-red-200 font-medium">{customizeError}</p>
                <button
                  onClick={() => setCustomizeError(null)}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-red-500/20 hover:bg-red-500/30 border border-red-500/40 rounded-md text-red-200 transition-colors"
                >
                  Dismiss
                </button>
              </div>
            </div>
          )}

          {customizeResult && (
            <div className="rounded-xl border border-white/10 bg-slate-900/50 p-4 space-y-2 text-sm text-slate-200" data-testid="customization-result">
              <div className="flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-emerald-300" />
                <div data-testid="customization-result-message">Issue filed and agent queued</div>
              </div>
              <div className="grid md:grid-cols-2 gap-2 text-xs text-slate-300">
                <div data-testid="customization-issue-id">Issue ID: {customizeResult.issue_id || 'unknown'}</div>
                <div data-testid="customization-agent">Agent: {customizeResult.agent || 'auto'}</div>
                <div data-testid="customization-run-id">Run ID: {customizeResult.run_id || 'pending'}</div>
                <div data-testid="customization-status">Status: {customizeResult.status}</div>
              </div>
              {customizeResult.tracker_url && (
                <div className="text-xs text-slate-300">
                  Tracker API: <code className="px-1 py-0.5 rounded bg-slate-900">{customizeResult.tracker_url}</code>
                </div>
              )}
              {customizeResult.message && <div className="text-xs text-slate-400">{customizeResult.message}</div>}
            </div>
          )}
        </div>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden" data-testid="generated-scenarios" aria-labelledby="generated-scenarios-heading">
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent"></div>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  <h2 id="generated-scenarios-heading" className="text-xl sm:text-2xl font-semibold">Generated Landing Pages</h2>
                  <Tooltip content="Your staging area for testing landing pages before deploying to production. All scenarios start here in the generated/ folder where you can iterate, test, and refine before moving to scenarios/.">
                    <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
                  </Tooltip>
                </div>
                <p className="text-xs sm:text-sm text-slate-400">
                  Staging workspace for testing and iteration
                </p>
              </div>
            <button
              data-testid="refresh-generated-button"
              className="inline-flex items-center gap-2 px-3 py-2 text-sm text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
              onClick={async () => {
                try {
                  setLoadingGenerated(true);
                  setGeneratedError(null);
                  const scenarios = await listGeneratedScenarios();
                  setGenerated(scenarios);
                } catch (err) {
                  setGeneratedError(err instanceof Error ? err.message : 'Failed to refresh generated scenarios');
                } finally {
                  setLoadingGenerated(false);
                }
              }}
              disabled={loadingGenerated}
              aria-label="Refresh generated scenarios list"
            >
              <RefreshCcw className={`h-4 w-4 ${loadingGenerated ? 'animate-spin' : ''}`} aria-hidden="true" />
              <span className="hidden sm:inline">Refresh</span>
            </button>
            </div>

            {/* Staging Area Workflow Explanation */}
            <div className="rounded-xl border border-purple-500/30 bg-gradient-to-br from-purple-500/10 to-blue-500/10 p-4 sm:p-5" role="region" aria-label="Staging workflow explanation">
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-9 h-9 rounded-xl bg-purple-500/20 border border-purple-500/40 flex items-center justify-center">
                  <FileOutput className="h-5 w-5 text-purple-300" aria-hidden="true" />
                </div>
                <div className="flex-1 space-y-3">
                  <div>
                    <h3 className="text-base font-bold text-purple-100 mb-1">Staging Workflow</h3>
                    <p className="text-xs text-purple-200/80">
                      Your safe testing zone. All scenarios below live in <code className="px-1.5 py-0.5 rounded bg-slate-900/60 text-emerald-300 font-mono">generated/</code> where you can experiment without affecting production.
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium bg-purple-500/20 border border-purple-500/40 text-purple-200 rounded-lg">
                      <Zap className="h-3 w-3" aria-hidden="true" />
                      Test safely
                    </span>
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium bg-purple-500/20 border border-purple-500/40 text-purple-200 rounded-lg">
                      <RotateCw className="h-3 w-3" aria-hidden="true" />
                      Iterate freely
                    </span>
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium bg-purple-500/20 border border-purple-500/40 text-purple-200 rounded-lg">
                      <CheckCircle className="h-3 w-3" aria-hidden="true" />
                      Move when ready
                    </span>
                  </div>
                  <p className="text-xs text-purple-200/70 leading-relaxed">
                    When satisfied with your landing page, move it from <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">generated/&lt;slug&gt;/</code> to <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">scenarios/&lt;slug&gt;/</code> for permanent deployment.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {generatedError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{generatedError}</div>
            </div>
          )}

          {loadingGenerated && (
            <div className="flex flex-col items-center justify-center py-12 text-slate-300 space-y-2">
              <Loader2 className="h-6 w-6 animate-spin text-emerald-300" aria-label="Loading generated scenarios" />
              <p className="text-sm">Loading scenarios...</p>
            </div>
          )}

          {!loadingGenerated && generated.length === 0 && !generatedError && (
            <div className="rounded-2xl border-2 border-dashed border-slate-700 bg-gradient-to-br from-slate-900/60 to-slate-800/40 p-8 sm:p-12 text-center space-y-6" role="status">
              <div className="flex justify-center">
                <div className="relative">
                  <div className="absolute inset-0 bg-emerald-500/20 rounded-full blur-2xl"></div>
                  <Rocket className="relative h-20 w-20 text-emerald-400" aria-hidden="true" />
                  <Sparkles className="absolute -top-1 -right-1 h-8 w-8 text-yellow-400 animate-pulse" aria-hidden="true" />
                </div>
              </div>
              <div className="space-y-3">
                <h3 className="text-2xl font-bold text-slate-100">Ready to Launch Your First Landing Page?</h3>
                <p className="text-base text-slate-300 max-w-md mx-auto leading-relaxed">
                  Create a complete, production-ready landing page in under 60 seconds. Just pick a name and click Generate.
                </p>
                <div className="flex flex-wrap items-center justify-center gap-3 pt-2">
                  <span className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 rounded-full">
                    <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                    React + TypeScript + Vite
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-blue-500/20 border border-blue-500/40 text-blue-300 rounded-full">
                    <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                    Go API + PostgreSQL
                  </span>
                  <span className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-purple-500/20 border border-purple-500/40 text-purple-300 rounded-full">
                    <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                    Stripe + A/B Testing
                  </span>
                </div>
              </div>
              <button
                onClick={() => {
                  const formElement = document.querySelector('[data-testid="generation-form"]');
                  formElement?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                  const nameInput = document.getElementById('generation-name-input') as HTMLInputElement;
                  nameInput?.focus();
                }}
                className="inline-flex items-center gap-2 px-8 py-4 text-base font-bold bg-gradient-to-r from-emerald-500/30 to-blue-500/30 border-2 border-emerald-500/60 text-emerald-200 rounded-xl hover:from-emerald-500/40 hover:to-blue-500/40 hover:border-emerald-400/80 hover:scale-105 transition-all shadow-xl shadow-emerald-500/25 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
                aria-label="Scroll to generation form and start creating"
              >
                <Rocket className="h-5 w-5" aria-hidden="true" />
                Create My First Landing Page
              </button>
            </div>
          )}

          {!loadingGenerated && generated.length > 0 && (
            <div className="grid gap-4" data-testid="generated-scenarios-list" role="list" aria-label="Generated scenarios">
              {generated.map((scenario) => {
                const status = scenarioStatuses[scenario.scenario_id] || { running: false, loading: false };
                const links = previewLinks[scenario.scenario_id];

                return (
                  <article
                    key={scenario.scenario_id}
                    className="rounded-xl border border-white/10 bg-slate-900/40 p-4 sm:p-5 hover:border-white/20 transition-colors"
                    data-testid={`generated-scenario-${scenario.scenario_id}`}
                    role="listitem"
                  >
                    <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-3">
                      <div className="flex-1 min-w-0">
                        <div className="text-xs font-medium text-slate-400 uppercase tracking-wide mb-1">Scenario</div>
                        <h3 className="text-base sm:text-lg font-semibold text-slate-100 truncate" data-testid={`generated-scenario-name-${scenario.scenario_id}`}>
                          {scenario.name}
                        </h3>
                        <div className="flex items-center gap-2 mt-1">
                          <p className="text-xs text-slate-400" data-testid={`generated-scenario-slug-${scenario.scenario_id}`}>
                            Slug: <code className="px-1 py-0.5 rounded bg-slate-800 font-mono">{scenario.scenario_id}</code>
                          </p>
                          <button
                            onClick={() => {
                              setCustomizeSlug(scenario.scenario_id);
                              navigator.clipboard.writeText(scenario.scenario_id);
                              setCopiedSlug(true);
                              setTimeout(() => setCopiedSlug(false), 2000);
                              // Scroll to agent customization form
                              const agentForm = document.querySelector('[data-testid="agent-customization-form"]');
                              agentForm?.scrollIntoView({ behavior: 'smooth', block: 'center' });
                            }}
                            className="inline-flex items-center gap-1 px-2 py-1 text-[10px] rounded border border-slate-500/30 bg-slate-500/10 text-slate-400 hover:text-slate-300 hover:bg-slate-500/20 transition-colors focus:outline-none focus:ring-1 focus:ring-emerald-500"
                            title="Copy slug and select for agent customization"
                            aria-label={`Copy slug and select ${scenario.scenario_id} for agent customization`}
                          >
                            {copiedSlug ? <Check className="h-2.5 w-2.5" aria-hidden="true" /> : <Copy className="h-2.5 w-2.5" aria-hidden="true" />}
                            <span className="hidden sm:inline">{copiedSlug ? 'Copied!' : 'Copy'}</span>
                          </button>
                        </div>
                        <p className="text-xs text-slate-400 mt-1">
                          Location: <code className="px-1 py-0.5 rounded bg-slate-800 font-mono text-[10px]">{scenario.path}</code>
                        </p>
                      </div>
                      <div className="flex flex-col gap-2 flex-shrink-0">
                        <span
                          className={`inline-flex items-center gap-1.5 text-xs font-medium rounded-full px-2.5 py-1 transition-all ${
                            status.loading
                              ? 'text-blue-300 border border-blue-500/30 bg-blue-500/10 animate-pulse'
                              : status.running
                              ? 'text-emerald-300 border border-emerald-500/30 bg-emerald-500/10'
                              : 'text-slate-400 border border-slate-500/30 bg-slate-500/10'
                          }`}
                          data-testid={`generated-scenario-status-${scenario.scenario_id}`}
                          role="status"
                          aria-live="polite"
                        >
                          {status.loading ? (
                            <>
                              <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
                              <span className="sr-only">Loading</span>
                            </>
                          ) : status.running ? (
                            <>
                              <CheckCircle className="h-3 w-3" aria-hidden="true" />
                              <span className="sr-only">Running</span>
                            </>
                          ) : (
                            <>
                              <Square className="h-3 w-3" aria-hidden="true" />
                              <span className="sr-only">Stopped</span>
                            </>
                          )}
                          {status.loading ? 'Starting...' : status.running ? 'Running' : 'Stopped'}
                        </span>
                      </div>
                    </div>

                    <div className="space-y-3">
                      <div className="flex flex-wrap gap-x-4 gap-y-2 text-xs text-slate-400">
                        {scenario.template_id && (
                          <span className="flex items-center gap-1">
                            <FileText className="h-3.5 w-3.5" aria-hidden="true" />
                            {scenario.template_id} {scenario.template_version && `v${scenario.template_version}`}
                          </span>
                        )}
                        {scenario.generated_at && (
                          <span className="flex items-center gap-1">
                            <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
                            <span className="sr-only">Generated on</span>
                            {new Date(scenario.generated_at).toLocaleString()}
                          </span>
                        )}
                      </div>

                      {/* Lifecycle Controls */}
                      <div className="flex flex-col sm:flex-row gap-2">
                        <div className="flex gap-2" role="group" aria-label="Lifecycle controls">
                          <button
                            onClick={() => handleStartScenario(scenario.scenario_id)}
                            disabled={status.loading || status.running}
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 hover:border-emerald-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-emerald-500"
                            aria-label="Start scenario"
                            title="Start this landing page scenario"
                          >
                            <Play className="h-3.5 w-3.5" aria-hidden="true" />
                            <span className="hidden sm:inline">Start</span>
                          </button>
                          <button
                            onClick={() => handleStopScenario(scenario.scenario_id)}
                            disabled={status.loading || !status.running}
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-red-500/40 bg-red-500/10 text-red-300 hover:bg-red-500/20 hover:border-red-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-red-500"
                            aria-label="Stop scenario"
                            title="Stop this landing page scenario"
                          >
                            <Square className="h-3.5 w-3.5" aria-hidden="true" />
                            <span className="hidden sm:inline">Stop</span>
                          </button>
                          <button
                            onClick={() => handleRestartScenario(scenario.scenario_id)}
                            disabled={status.loading || !status.running}
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-blue-500/40 bg-blue-500/10 text-blue-300 hover:bg-blue-500/20 hover:border-blue-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
                            aria-label="Restart scenario"
                            title="Restart this landing page scenario"
                          >
                            <RotateCw className="h-3.5 w-3.5" aria-hidden="true" />
                            <span className="hidden sm:inline">Restart</span>
                          </button>
                        </div>
                        <div className="flex gap-2 flex-1 sm:flex-none">
                          <button
                            onClick={() => handleToggleLogs(scenario.scenario_id)}
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-slate-500/30 bg-slate-500/10 text-slate-300 hover:bg-slate-500/20 hover:border-slate-400 transition-all focus:outline-none focus:ring-2 focus:ring-slate-500"
                            aria-label={showLogs[scenario.scenario_id] ? 'Hide logs' : 'Show logs'}
                            title={showLogs[scenario.scenario_id] ? 'Hide scenario logs' : 'View scenario logs'}
                          >
                            <FileOutput className="h-3.5 w-3.5" aria-hidden="true" />
                            <span>{showLogs[scenario.scenario_id] ? 'Hide' : 'Show'} Logs</span>
                          </button>
                          {!status.running && !status.loading && (
                            <button
                              onClick={() => handlePromoteScenario(scenario.scenario_id)}
                              className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-bold rounded-lg border-2 border-purple-500/60 bg-gradient-to-r from-purple-500/20 to-pink-500/20 text-purple-200 hover:from-purple-500/30 hover:to-pink-500/30 hover:border-purple-400 transition-all shadow-lg shadow-purple-500/20 focus:outline-none focus:ring-2 focus:ring-purple-500"
                              aria-label="Promote to production"
                              title="Move this scenario from staging (generated/) to production (scenarios/)"
                            >
                              <Rocket className="h-3.5 w-3.5" aria-hidden="true" />
                              <span>Promote to Production</span>
                            </button>
                          )}
                        </div>
                      </div>

                      {/* Logs Display */}
                      {showLogs[scenario.scenario_id] && (
                        <div className="rounded-lg border border-white/10 bg-slate-900/60 p-3">
                          <div className="text-xs font-medium text-slate-400 mb-2">Recent Logs</div>
                          <pre className="text-[10px] text-slate-300 bg-slate-950 border border-white/10 rounded p-2 overflow-x-auto whitespace-pre-wrap max-h-64 overflow-y-auto">
                            {scenarioLogs[scenario.scenario_id] || 'Loading logs...'}
                          </pre>
                        </div>
                      )}

                      {/* Access Links - shown when running */}
                      {status.running && links && (
                        <div className="rounded-xl border-2 border-emerald-500/40 bg-gradient-to-br from-emerald-500/15 to-blue-500/15 p-4 space-y-3 shadow-lg shadow-emerald-500/10">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <div className="h-2.5 w-2.5 bg-emerald-400 rounded-full animate-pulse" aria-hidden="true" />
                              <div className="text-sm font-bold text-emerald-200 uppercase tracking-wide flex items-center gap-1.5">
                                <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
                                Live & Ready
                              </div>
                            </div>
                            <span className="text-xs text-emerald-300/70 font-medium">Click to open</span>
                          </div>
                          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                            {links.links.public && (
                              <a
                                href={links.links.public}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="group flex items-center justify-between gap-2 text-sm font-semibold text-emerald-100 bg-emerald-900/40 hover:bg-emerald-900/60 border-2 border-emerald-500/40 hover:border-emerald-400/60 rounded-lg px-4 py-3 transition-all shadow-md hover:shadow-lg hover:shadow-emerald-500/20 hover:scale-102"
                              >
                                <span className="truncate flex items-center gap-2">
                                  <Globe className="h-4 w-4" aria-hidden="true" />
                                  Public Landing
                                </span>
                                <ExternalLink className="h-4 w-4 flex-shrink-0 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform" aria-hidden="true" />
                              </a>
                            )}
                            {links.links.admin && (
                              <a
                                href={links.links.admin}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="group flex items-center justify-between gap-2 text-sm font-semibold text-blue-100 bg-blue-900/40 hover:bg-blue-900/60 border-2 border-blue-500/40 hover:border-blue-400/60 rounded-lg px-4 py-3 transition-all shadow-md hover:shadow-lg hover:shadow-blue-500/20 hover:scale-102"
                              >
                                <span className="truncate flex items-center gap-2">
                                  <Settings className="h-4 w-4" aria-hidden="true" />
                                  Admin Dashboard
                                </span>
                                <ExternalLink className="h-4 w-4 flex-shrink-0 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform" aria-hidden="true" />
                              </a>
                            )}
                          </div>
                          <p className="text-xs text-center text-emerald-200/70 pt-1">
                            Your landing page is fully operational. Make changes, then click Restart to see updates.
                          </p>
                        </div>
                      )}

                      {/* Staging Info - shown when not running */}
                      {!status.running && (
                        <div className="rounded-xl border border-blue-500/30 bg-gradient-to-br from-blue-500/10 to-slate-800/20 p-4 space-y-3">
                          <div className="flex items-start gap-3">
                            <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
                              <Zap className="h-4 w-4 text-blue-300" aria-hidden="true" />
                            </div>
                            <div className="flex-1 space-y-2">
                              <p className="text-sm text-blue-100 font-semibold">Ready to Launch</p>
                              <p className="text-xs text-blue-200/80 leading-relaxed">
                                Click <strong className="text-blue-100 bg-blue-500/20 px-1.5 py-0.5 rounded">Start</strong> above to launch. You'll instantly see access links to your live landing page and admin dashboard.
                              </p>
                              <div className="pt-2 border-t border-blue-500/20">
                                <p className="text-xs text-blue-300/70 leading-relaxed flex items-start gap-1.5">
                                  <Zap className="h-3.5 w-3.5 flex-shrink-0 mt-0.5 text-blue-300" aria-hidden="true" />
                                  <span><strong className="text-blue-200">Testing zone:</strong> Lives in <code className="px-1 py-0.5 rounded bg-slate-900/60 text-emerald-300 font-mono text-[10px]">{scenario.path}</code></span>
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>

        {/* Promotion Confirmation Dialog */}
        {promoteDialogScenario && (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in"
            onClick={() => setPromoteDialogScenario(null)}
            role="dialog"
            aria-modal="true"
            aria-labelledby="promote-dialog-title"
          >
            <div
              className="relative w-full max-w-md rounded-2xl border-2 border-purple-500/40 bg-gradient-to-br from-slate-900 via-slate-900 to-purple-900/30 p-6 shadow-2xl shadow-purple-500/20"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => setPromoteDialogScenario(null)}
                className="absolute top-4 right-4 text-slate-400 hover:text-slate-200 transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500 rounded"
                aria-label="Close dialog"
              >
                <X className="h-5 w-5" aria-hidden="true" />
              </button>

              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  <div className="flex-shrink-0 w-12 h-12 rounded-xl bg-purple-500/20 border-2 border-purple-500/40 flex items-center justify-center">
                    <Rocket className="h-6 w-6 text-purple-300" aria-hidden="true" />
                  </div>
                  <div className="flex-1">
                    <h3 id="promote-dialog-title" className="text-xl font-bold text-purple-100 mb-1">
                      Promote to Production?
                    </h3>
                    <p className="text-sm text-purple-200/80">
                      Move <code className="px-1.5 py-0.5 rounded bg-slate-800 text-emerald-300 font-mono text-xs">{promoteDialogScenario}</code> to production
                    </p>
                  </div>
                </div>

                <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 space-y-2">
                  <div className="flex items-start gap-2">
                    <AlertCircle className="h-4 w-4 flex-shrink-0 mt-0.5 text-amber-400" aria-hidden="true" />
                    <div className="text-xs text-amber-100 space-y-1">
                      <p className="font-semibold">This action will:</p>
                      <ul className="list-disc list-inside space-y-0.5 text-amber-200/90">
                        <li>Stop the scenario if running</li>
                        <li>Move from <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">generated/</code> to <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">scenarios/</code></li>
                        <li>Remove from staging list</li>
                      </ul>
                    </div>
                  </div>
                </div>

                <div className="flex gap-3 pt-2">
                  <button
                    onClick={() => setPromoteDialogScenario(null)}
                    className="flex-1 px-4 py-2.5 text-sm font-medium rounded-lg border border-slate-500/30 bg-slate-500/10 text-slate-300 hover:bg-slate-500/20 hover:border-slate-400 transition-all focus:outline-none focus:ring-2 focus:ring-slate-500"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={confirmPromote}
                    className="flex-1 px-4 py-2.5 text-sm font-bold rounded-lg border-2 border-purple-500/60 bg-gradient-to-r from-purple-500/30 to-pink-500/30 text-purple-100 hover:from-purple-500/40 hover:to-pink-500/40 hover:border-purple-400 transition-all shadow-lg shadow-purple-500/20 focus:outline-none focus:ring-2 focus:ring-purple-500"
                  >
                    Promote to Production
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

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
      {showKeyboardHelp && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 backdrop-blur-sm px-4"
          onClick={() => setShowKeyboardHelp(false)}
          role="dialog"
          aria-modal="true"
          aria-labelledby="keyboard-shortcuts-title"
        >
          <div
            className="bg-slate-900 border border-white/10 rounded-2xl shadow-2xl max-w-md w-full p-6 space-y-4"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <h2 id="keyboard-shortcuts-title" className="text-xl font-bold text-slate-100">Keyboard Shortcuts</h2>
              <button
                onClick={() => setShowKeyboardHelp(false)}
                className="text-slate-400 hover:text-slate-200 transition-colors p-1 rounded-lg hover:bg-white/5 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                aria-label="Close dialog"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3 text-sm">
              <div className="flex items-center justify-between py-2 border-b border-white/10">
                <span className="text-slate-300">Generate scenario</span>
                <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + Enter</kbd>
              </div>
              <div className="flex items-center justify-between py-2 border-b border-white/10">
                <span className="text-slate-300">Dry-run preview</span>
                <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + Shift + Enter</kbd>
              </div>
              <div className="flex items-center justify-between py-2 border-b border-white/10">
                <span className="text-slate-300">Refresh templates</span>
                <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">⌘/Ctrl + R</kbd>
              </div>
              <div className="flex items-center justify-between py-2">
                <span className="text-slate-300">Skip to main content</span>
                <kbd className="px-2.5 py-1.5 rounded bg-slate-800 border border-slate-700 font-mono text-xs text-slate-200">Tab</kbd>
              </div>
            </div>

            <div className="pt-4 border-t border-white/10">
              <p className="text-xs text-slate-400 flex items-start gap-2">
                <Zap className="h-4 w-4 flex-shrink-0 mt-0.5 text-emerald-400" aria-hidden="true" />
                <span>All keyboard shortcuts work when the generation form is visible and not disabled.</span>
              </p>
            </div>

            <button
              onClick={() => setShowKeyboardHelp(false)}
              className="w-full px-4 py-2.5 text-sm font-medium rounded-lg border border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 hover:border-emerald-400 transition-all focus:outline-none focus:ring-2 focus:ring-emerald-500"
            >
              Got it!
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
