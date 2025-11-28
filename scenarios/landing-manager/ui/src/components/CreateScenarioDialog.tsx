import { memo, useCallback, useState, useEffect } from 'react';
import {
  AlertCircle,
  ArrowLeft,
  ArrowRight,
  CheckCircle,
  HelpCircle,
  Loader2,
  Play,
  RefreshCcw,
  Rocket,
  Sparkles,
  X,
  Zap,
} from 'lucide-react';
import { type GenerationResult, type Template, listTemplates } from '../lib/api';
import { slugify } from '../lib/utils';
import { Tooltip } from './Tooltip';
import { TemplateCard } from './TemplateCard';
import { TemplateCardSkeleton } from './TemplateCardSkeleton';
import { TemplateEmptyState } from './TemplateEmptyState';
import { TemplateDetailsPanel } from './TemplateDetailsPanel';

interface CreateScenarioDialogProps {
  isOpen: boolean;
  onClose: () => void;
  templates: Template[];
  loadingTemplates: boolean;
  templatesError: string | null;
  onRefreshTemplates: (templates: Template[], error: string | null, loading: boolean) => void;
  onGenerate: (templateId: string, name: string, slug: string, dryRun: boolean) => Promise<GenerationResult>;
  onStartScenario: (scenarioId: string) => Promise<void>;
}

type DialogStep = 'template' | 'generate' | 'success';

export const CreateScenarioDialog = memo(function CreateScenarioDialog({
  isOpen,
  onClose,
  templates,
  loadingTemplates,
  templatesError,
  onRefreshTemplates,
  onGenerate,
  onStartScenario,
}: CreateScenarioDialogProps) {
  const [step, setStep] = useState<DialogStep>('template');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [nameError, setNameError] = useState<string | null>(null);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);
  const [generateError, setGenerateError] = useState<string | null>(null);
  const [lastResult, setLastResult] = useState<{ dryRun: boolean; result: GenerationResult } | null>(null);

  const selectedTemplate = templates?.find((t) => t?.id === selectedId) ?? templates?.[0] ?? null;

  // Reset state when dialog opens
  useEffect(() => {
    if (isOpen) {
      setStep('template');
      setSelectedId(templates[0]?.id ?? null);
      setName('');
      setSlug('');
      setNameError(null);
      setSlugError(null);
      setGenerating(false);
      setGenerateError(null);
      setLastResult(null);
    }
  }, [isOpen, templates]);

  // Auto-select first template when templates load
  useEffect(() => {
    if (!selectedId && templates.length > 0) {
      setSelectedId(templates[0].id);
    }
  }, [templates, selectedId]);

  const handleNameChange = useCallback((value: string) => {
    setName(value);
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
  }, [name]);

  const handleSlugChange = useCallback((value: string) => {
    const cleaned = slugify(value);
    setSlug(cleaned);
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
  }, []);

  const handleGenerate = async (dryRun: boolean) => {
    if (!selectedTemplate || !name.trim() || !slug.trim()) return;

    try {
      setGenerating(true);
      setGenerateError(null);
      const result = await onGenerate(selectedTemplate.id, name.trim(), slug.trim(), dryRun);
      setLastResult({ dryRun, result });
      if (!dryRun) {
        setStep('success');
      }
    } catch (err) {
      setGenerateError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setGenerating(false);
    }
  };

  const handleRefreshTemplates = async () => {
    try {
      onRefreshTemplates(templates, null, true);
      const tpl = await listTemplates();
      onRefreshTemplates(tpl, null, false);
      if (!selectedId && tpl[0]?.id) {
        setSelectedId(tpl[0].id);
      }
    } catch (err) {
      onRefreshTemplates(templates, err instanceof Error ? err.message : 'Failed to refresh templates', false);
    }
  };

  const handleStartAndClose = async () => {
    if (lastResult?.result.scenario_id) {
      await onStartScenario(lastResult.result.scenario_id);
    }
    onClose();
  };

  const handleCreateAnother = () => {
    setStep('template');
    setName('');
    setSlug('');
    setLastResult(null);
    setGenerateError(null);
  };

  const canProceedToGenerate = selectedTemplate !== null;
  const canGenerate = selectedTemplate && name.trim() && slug.trim() && !nameError && !slugError;

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-scenario-dialog-title"
    >
      <div
        className="relative w-full max-w-4xl max-h-[90vh] overflow-hidden rounded-2xl border border-white/10 bg-slate-900 shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 sm:p-6 border-b border-white/10 bg-gradient-to-r from-emerald-500/10 to-blue-500/10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center">
              <Rocket className="h-5 w-5 text-emerald-300" aria-hidden="true" />
            </div>
            <div>
              <h2 id="create-scenario-dialog-title" className="text-lg sm:text-xl font-bold text-slate-100">
                Create Landing Page
              </h2>
              <p className="text-xs sm:text-sm text-slate-400">
                {step === 'template' && 'Step 1: Choose a template'}
                {step === 'generate' && 'Step 2: Name your landing page'}
                {step === 'success' && 'Success! Your landing page is ready'}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
            aria-label="Close dialog"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Progress Indicator */}
        <div className="flex-shrink-0 px-4 sm:px-6 py-3 border-b border-white/5 bg-slate-900/50">
          <div className="flex items-center gap-2">
            {['template', 'generate', 'success'].map((s, idx) => (
              <div key={s} className="flex items-center gap-2">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-colors ${
                    step === s
                      ? 'bg-emerald-500 text-white'
                      : idx < ['template', 'generate', 'success'].indexOf(step)
                      ? 'bg-emerald-500/30 text-emerald-300'
                      : 'bg-slate-700 text-slate-400'
                  }`}
                >
                  {idx < ['template', 'generate', 'success'].indexOf(step) ? (
                    <CheckCircle className="h-4 w-4" />
                  ) : (
                    idx + 1
                  )}
                </div>
                {idx < 2 && (
                  <div
                    className={`w-12 sm:w-20 h-0.5 ${
                      idx < ['template', 'generate', 'success'].indexOf(step)
                        ? 'bg-emerald-500/50'
                        : 'bg-slate-700'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-4">
          {/* Step 1: Template Selection */}
          {step === 'template' && (
            <>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <h3 className="text-base font-semibold text-slate-200">Select a Template</h3>
                  <Tooltip content="Choose a template as the starting point. Each includes pre-configured sections, metrics tracking, and Stripe integration.">
                    <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400" />
                  </Tooltip>
                </div>
                <button
                  onClick={handleRefreshTemplates}
                  disabled={loadingTemplates}
                  className="inline-flex items-center gap-2 px-3 py-1.5 text-xs text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-colors disabled:opacity-50"
                >
                  <RefreshCcw className={`h-3.5 w-3.5 ${loadingTemplates ? 'animate-spin' : ''}`} />
                  Refresh
                </button>
              </div>

              {templatesError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-red-200 text-sm flex items-start gap-2">
                  <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0" />
                  <span>{templatesError}</span>
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {loadingTemplates && (
                  <>
                    <TemplateCardSkeleton />
                    <TemplateCardSkeleton />
                  </>
                )}
                {!loadingTemplates && templates.length === 0 && !templatesError && (
                  <div className="col-span-full">
                    <TemplateEmptyState onRetry={handleRefreshTemplates} />
                  </div>
                )}
                {!loadingTemplates &&
                  templates?.filter(Boolean).map((tpl) => (
                    <TemplateCard
                      key={tpl.id}
                      template={tpl}
                      isSelected={selectedId === tpl.id}
                      onSelect={setSelectedId}
                    />
                  ))}
              </div>

              {selectedTemplate && (
                <div className="pt-2">
                  <TemplateDetailsPanel template={selectedTemplate} />
                </div>
              )}
            </>
          )}

          {/* Step 2: Generation Form */}
          {step === 'generate' && (
            <>
              {selectedTemplate && (
                <div className="flex items-center gap-3 p-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10">
                  <CheckCircle className="h-5 w-5 text-emerald-400 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-emerald-200">
                      Using template: <strong className="text-emerald-100">{selectedTemplate.name}</strong>
                    </p>
                  </div>
                  <button
                    onClick={() => setStep('template')}
                    className="text-xs text-emerald-300 hover:text-emerald-200 underline"
                  >
                    Change
                  </button>
                </div>
              )}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="dialog-name-input" className="block text-sm font-medium text-slate-300 mb-2">
                    Landing Page Name
                    <Tooltip content="Human-readable name for your landing page.">
                      <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500" />
                    </Tooltip>
                  </label>
                  <input
                    id="dialog-name-input"
                    type="text"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    className={`w-full rounded-lg border px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 bg-slate-800 focus:ring-2 outline-none transition-colors ${
                      nameError && name.trim() ? 'border-red-500/50 focus:border-red-400 focus:ring-red-500/20' : 'border-white/10 focus:border-emerald-400 focus:ring-emerald-500/20'
                    }`}
                    placeholder="My Awesome Product"
                    autoFocus
                  />
                  {nameError && name.trim().length > 0 && (
                    <p className="mt-1.5 text-xs text-red-400 flex items-center gap-1">
                      <AlertCircle className="h-3 w-3" />
                      {nameError}
                    </p>
                  )}
                  {!nameError && name && (
                    <p className="mt-1.5 text-xs text-slate-400">
                      Slug: <code className="px-1 py-0.5 rounded bg-slate-800">{slug || slugify(name)}</code>
                    </p>
                  )}
                </div>
                <div>
                  <label htmlFor="dialog-slug-input" className="block text-sm font-medium text-slate-300 mb-2">
                    URL Slug
                    <Tooltip content="URL-safe identifier for the scenario folder.">
                      <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500" />
                    </Tooltip>
                  </label>
                  <input
                    id="dialog-slug-input"
                    type="text"
                    value={slug}
                    onChange={(e) => handleSlugChange(e.target.value)}
                    className={`w-full rounded-lg border px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 font-mono bg-slate-800 focus:ring-2 outline-none transition-colors ${
                      slugError && slug.trim() ? 'border-red-500/50 focus:border-red-400 focus:ring-red-500/20' : 'border-white/10 focus:border-emerald-400 focus:ring-emerald-500/20'
                    }`}
                    placeholder="my-awesome-product"
                  />
                  {slugError && slug.trim().length > 0 && (
                    <p className="mt-1.5 text-xs text-red-400 flex items-center gap-1">
                      <AlertCircle className="h-3 w-3" />
                      {slugError}
                    </p>
                  )}
                  {!slugError && (
                    <p className="mt-1.5 text-xs text-slate-400">
                      Folder: <code className="px-1 py-0.5 rounded bg-slate-800">generated/{slug || 'your-slug'}/</code>
                    </p>
                  )}
                </div>
              </div>

              {generateError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm flex items-start gap-2">
                  <AlertCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-red-200">{generateError}</p>
                    <button
                      onClick={() => setGenerateError(null)}
                      className="mt-2 text-xs text-red-300 hover:text-red-200 underline"
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              )}

              {lastResult?.dryRun && (
                <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 space-y-2">
                  <div className="flex items-center gap-2 text-sm text-blue-200">
                    <CheckCircle className="h-4 w-4 text-blue-400" />
                    <span className="font-medium">Dry-run Preview</span>
                  </div>
                  {lastResult.result.plan?.paths && (
                    <div className="text-xs text-blue-200/80 bg-slate-900/50 rounded p-2 font-mono space-y-0.5 max-h-32 overflow-y-auto">
                      {lastResult.result.plan.paths.map((p) => (
                        <div key={p}>{p}</div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              <div className="p-3 rounded-lg border border-blue-500/20 bg-blue-500/5">
                <div className="flex items-start gap-2 text-xs text-blue-200">
                  <Zap className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-blue-400" />
                  <span>
                    Your landing page will be created in a staging folder where you can test and iterate before moving to production.
                  </span>
                </div>
              </div>
            </>
          )}

          {/* Step 3: Success */}
          {step === 'success' && lastResult && (
            <div className="text-center space-y-6 py-4">
              <div className="flex justify-center">
                <div className="relative">
                  <div className="absolute inset-0 bg-emerald-500/30 rounded-full blur-2xl" />
                  <div className="relative w-20 h-20 rounded-full bg-emerald-500/20 border-2 border-emerald-500/40 flex items-center justify-center">
                    <CheckCircle className="h-10 w-10 text-emerald-400" />
                  </div>
                  <Sparkles className="absolute -top-1 -right-1 h-8 w-8 text-yellow-400 animate-pulse" />
                </div>
              </div>

              <div className="space-y-2">
                <h3 className="text-2xl font-bold text-slate-100">Landing Page Created!</h3>
                <p className="text-slate-300">
                  <strong className="text-emerald-300">{lastResult.result.name}</strong> is ready in your staging area
                </p>
              </div>

              <div className="rounded-lg border border-white/10 bg-slate-800/50 p-4 text-left space-y-2">
                <div className="grid sm:grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Scenario ID</span>
                    <p className="font-mono text-emerald-300">{lastResult.result.scenario_id}</p>
                  </div>
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Location</span>
                    <p className="font-mono text-slate-200 text-xs truncate">{lastResult.result.path}</p>
                  </div>
                </div>
              </div>

              {lastResult.result.next_steps && (
                <div className="text-left rounded-lg border border-purple-500/20 bg-purple-500/5 p-4">
                  <p className="text-xs text-purple-300 uppercase font-medium mb-2">Next Steps</p>
                  <ul className="text-sm text-purple-200/90 space-y-1">
                    {lastResult.result.next_steps.map((step, idx) => (
                      <li key={idx} className="flex items-start gap-2">
                        <span className="text-purple-400">•</span>
                        {step}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-between gap-3 p-4 sm:p-6 border-t border-white/10 bg-slate-900/80">
          {step === 'template' && (
            <>
              <button
                onClick={onClose}
                className="px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white border border-white/10 hover:border-white/20 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => setStep('generate')}
                disabled={!canProceedToGenerate}
                className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-semibold bg-emerald-500/20 border border-emerald-500/40 text-emerald-200 rounded-lg hover:bg-emerald-500/30 hover:border-emerald-400 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Continue
                <ArrowRight className="h-4 w-4" />
              </button>
            </>
          )}

          {step === 'generate' && (
            <>
              <button
                onClick={() => setStep('template')}
                className="inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white border border-white/10 hover:border-white/20 rounded-lg transition-colors"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </button>
              <div className="flex gap-2">
                <button
                  onClick={() => handleGenerate(true)}
                  disabled={generating || !canGenerate}
                  className="inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border border-white/15 bg-white/5 text-slate-300 rounded-lg hover:bg-white/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCcw className="h-4 w-4" />}
                  Preview
                </button>
                <button
                  onClick={() => handleGenerate(false)}
                  disabled={generating || !canGenerate}
                  className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-bold bg-gradient-to-r from-emerald-500/30 to-blue-500/30 border-2 border-emerald-500/60 text-emerald-100 rounded-lg hover:from-emerald-500/40 hover:to-blue-500/40 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-emerald-500/20"
                >
                  {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
                  Generate Now
                </button>
              </div>
            </>
          )}

          {step === 'success' && (
            <>
              <button
                onClick={handleCreateAnother}
                className="inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white border border-white/10 hover:border-white/20 rounded-lg transition-colors"
              >
                <Rocket className="h-4 w-4" />
                Create Another
              </button>
              <div className="flex gap-2">
                <button
                  onClick={onClose}
                  className="px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white border border-white/10 hover:border-white/20 rounded-lg transition-colors"
                >
                  Close
                </button>
                <button
                  onClick={handleStartAndClose}
                  className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-bold bg-gradient-to-r from-emerald-500/30 to-blue-500/30 border-2 border-emerald-500/60 text-emerald-100 rounded-lg hover:from-emerald-500/40 hover:to-blue-500/40 transition-all shadow-lg shadow-emerald-500/20"
                >
                  <Play className="h-4 w-4" />
                  Start & View
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
});
