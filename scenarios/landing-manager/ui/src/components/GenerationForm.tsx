import { memo } from 'react';
import { AlertCircle, CheckCircle, HelpCircle, Loader2, Play, RefreshCcw, Rocket, Zap } from 'lucide-react';
import { type GenerationResult, type Template } from '../lib/api';
import { slugify } from '../lib/utils';
import { Tooltip } from './Tooltip';

interface GenerationFormProps {
  selectedTemplate: Template | null;
  name: string;
  slug: string;
  nameError: string | null;
  slugError: string | null;
  generating: boolean;
  generateError: string | null;
  lastResult: { dryRun: boolean; result: GenerationResult } | null;
  onNameChange: (value: string) => void;
  onSlugChange: (value: string) => void;
  onGenerate: (dryRun: boolean) => void;
  onStartScenario: (scenarioId: string) => Promise<void>;
  onClearResult: () => void;
  onShowKeyboardHelp: () => void;
  onClearError: () => void;
}

export const GenerationForm = memo(function GenerationForm({
  selectedTemplate,
  name,
  slug,
  nameError,
  slugError,
  generating,
  generateError,
  lastResult,
  onNameChange,
  onSlugChange,
  onGenerate,
  onStartScenario,
  onClearResult,
  onShowKeyboardHelp,
  onClearError,
}: GenerationFormProps) {
  return (
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
            onChange={(e) => onNameChange(e.target.value)}
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
            onChange={(e) => onSlugChange(e.target.value)}
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
            onClick={() => onGenerate(true)}
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
            onClick={() => onGenerate(false)}
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
            onClick={onShowKeyboardHelp}
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
                onClick={onClearError}
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
                    await onStartScenario(lastResult.result.scenario_id);
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
                  onClick={onClearResult}
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
  );
});
