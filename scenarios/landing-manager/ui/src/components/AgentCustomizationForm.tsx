import { AlertCircle, HelpCircle, Loader2, RefreshCcw, Rocket, CheckCircle } from 'lucide-react';
import { type GeneratedScenario, type CustomizeResult } from '../lib/api';
import { slugify } from '../lib/utils';
import { Tooltip } from './Tooltip';

interface AgentCustomizationFormProps {
  generated: GeneratedScenario[];
  loadingGenerated: boolean;
  customizeSlug: string;
  customizeBrief: string;
  customizeAssets: string;
  customizing: boolean;
  customizeError: string | null;
  customizeResult: CustomizeResult | null;
  onCustomizeSlugChange: (value: string) => void;
  onCustomizeBriefChange: (value: string) => void;
  onCustomizeAssetsChange: (value: string) => void;
  onCustomize: () => void;
  onResetForm: () => void;
  onClearError: () => void;
  onSelectScenario: (slug: string) => void;
}

export function AgentCustomizationForm({
  generated,
  loadingGenerated,
  customizeSlug,
  customizeBrief,
  customizeAssets,
  customizing,
  customizeError,
  customizeResult,
  onCustomizeSlugChange,
  onCustomizeBriefChange,
  onCustomizeAssetsChange,
  onCustomize,
  onResetForm,
  onClearError,
  onSelectScenario,
}: AgentCustomizationFormProps) {
  return (
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
              onChange={(e) => {
                onCustomizeSlugChange(e.target.value);
                onSelectScenario(e.target.value);
              }}
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
              onChange={(e) => onCustomizeSlugChange(slugify(e.target.value))}
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
            onChange={(e) => onCustomizeAssetsChange(e.target.value)}
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
          onChange={(e) => onCustomizeBriefChange(e.target.value)}
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
          onClick={onCustomize}
          className="inline-flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          disabled={customizing || !customizeSlug.trim() || !customizeBrief.trim()}
          title={!customizeSlug.trim() ? 'Select a scenario first' : !customizeBrief.trim() ? 'Add a customization brief' : 'File issue and trigger AI agent'}
        >
          {customizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
          File issue & trigger agent
        </button>
        <button
          onClick={onResetForm}
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
              onClick={onClearError}
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
  );
}
