import { memo } from 'react';
import { AlertCircle, HelpCircle, Loader2, RefreshCcw, Rocket } from 'lucide-react';
import { type GeneratedScenario, type CustomizeResult } from '../lib/api';
import { Tooltip } from './Tooltip';
import { NoScenariosWarning } from './agent-form/NoScenariosWarning';
import { SlugSelector } from './agent-form/SlugSelector';
import { BriefTextarea } from './agent-form/BriefTextarea';
import { CustomizationResult } from './agent-form/CustomizationResult';

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

export const AgentCustomizationForm = memo(function AgentCustomizationForm({
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
  const isFormValid = customizeSlug.trim() && customizeBrief.trim();
  const hasError = !customizeBrief.trim() && !!customizeError;

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

      {generated.length === 0 && !loadingGenerated && <NoScenariosWarning />}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <SlugSelector
          value={customizeSlug}
          generated={generated}
          onChange={onCustomizeSlugChange}
          onSelectScenario={onSelectScenario}
        />
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

      <BriefTextarea
        value={customizeBrief}
        onChange={onCustomizeBriefChange}
        hasError={hasError}
      />

      <div className="flex flex-wrap gap-3">
        <button
          data-testid="customize-button"
          onClick={onCustomize}
          className="inline-flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          disabled={customizing || !isFormValid}
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

      {customizeResult && <CustomizationResult result={customizeResult} />}
    </div>
  );
});
