import { memo, useState, useEffect } from 'react';
import {
  AlertCircle,
  CheckCircle,
  HelpCircle,
  Loader2,
  Rocket,
  Sparkles,
  Wand2,
  X,
} from 'lucide-react';
import { type CustomizeResult, type GeneratedScenario } from '../lib/api';
import { Tooltip } from './Tooltip';

interface AgentCustomizationDialogProps {
  isOpen: boolean;
  scenario: GeneratedScenario | null;
  onClose: () => void;
  onCustomize: (slug: string, brief: string, assets: string[]) => Promise<CustomizeResult>;
}

export const AgentCustomizationDialog = memo(function AgentCustomizationDialog({
  isOpen,
  scenario,
  onClose,
  onCustomize,
}: AgentCustomizationDialogProps) {
  const [brief, setBrief] = useState('');
  const [assets, setAssets] = useState('');
  const [customizing, setCustomizing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<CustomizeResult | null>(null);

  // Reset state when dialog opens with a new scenario
  useEffect(() => {
    if (isOpen && scenario) {
      setBrief('');
      setAssets('');
      setCustomizing(false);
      setError(null);
      setResult(null);
    }
  }, [isOpen, scenario?.scenario_id]);

  const handleCustomize = async () => {
    if (!scenario || !brief.trim()) return;

    try {
      setCustomizing(true);
      setError(null);
      const assetList = assets
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const customizeResult = await onCustomize(scenario.scenario_id, brief.trim(), assetList);
      setResult(customizeResult);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Customization failed');
    } finally {
      setCustomizing(false);
    }
  };

  const handleClose = () => {
    onClose();
  };

  if (!isOpen || !scenario) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fade-in"
      onClick={handleClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="customize-dialog-title"
    >
      <div
        className="relative w-full max-w-2xl max-h-[90vh] overflow-hidden rounded-2xl border border-purple-500/30 bg-slate-900 shadow-2xl shadow-purple-500/10 flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 sm:p-6 border-b border-white/10 bg-gradient-to-r from-purple-500/10 to-pink-500/10">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-purple-500/20 border border-purple-500/40 flex items-center justify-center">
              <Wand2 className="h-5 w-5 text-purple-300" aria-hidden="true" />
            </div>
            <div>
              <h2 id="customize-dialog-title" className="text-lg sm:text-xl font-bold text-slate-100">
                AI Customization
              </h2>
              <p className="text-xs sm:text-sm text-slate-400">
                Let AI refine your landing page
              </p>
            </div>
          </div>
          <button
            onClick={handleClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500"
            aria-label="Close dialog"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-4">
          {/* Scenario Info */}
          <div className="flex items-center gap-3 p-3 rounded-lg border border-purple-500/30 bg-purple-500/10">
            <Sparkles className="h-5 w-5 text-purple-400 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <p className="text-sm text-purple-100 font-medium truncate">{scenario.name}</p>
              <p className="text-xs text-purple-300/70">
                <code className="px-1 py-0.5 rounded bg-slate-900/50">{scenario.scenario_id}</code>
              </p>
            </div>
          </div>

          {!result ? (
            <>
              {/* Brief Input */}
              <div>
                <label htmlFor="dialog-brief-input" className="block text-sm font-medium text-slate-300 mb-2">
                  Customization Brief
                  <Tooltip content="Describe your goals, target audience, value proposition, desired tone, and success metrics. Be specific for better results.">
                    <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500" />
                  </Tooltip>
                </label>
                <textarea
                  id="dialog-brief-input"
                  value={brief}
                  onChange={(e) => setBrief(e.target.value)}
                  className="w-full rounded-lg border border-white/10 bg-slate-800 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-purple-400 focus:ring-2 focus:ring-purple-500/20 outline-none transition-colors min-h-[140px] resize-y"
                  placeholder="Example: Target SaaS founders looking for rapid launch. Professional but friendly tone. Emphasize time savings and built-in features. CTA: Start Free Trial. Success metric: 10% conversion rate."
                  autoFocus
                />
                <div className="flex items-center justify-between mt-1.5">
                  <p className="text-xs text-slate-400">
                    Be specific about goals, audience, tone, and desired outcomes
                  </p>
                  <p className="text-xs text-slate-500">{brief.length} characters</p>
                </div>
              </div>

              {/* Assets Input */}
              <div>
                <label htmlFor="dialog-assets-input" className="block text-sm font-medium text-slate-300 mb-2">
                  Assets (Optional)
                  <Tooltip content="Comma-separated list of file paths or URLs to assets like logos, images, or brand guidelines.">
                    <HelpCircle className="inline h-3.5 w-3.5 ml-1 text-slate-500" />
                  </Tooltip>
                </label>
                <input
                  id="dialog-assets-input"
                  value={assets}
                  onChange={(e) => setAssets(e.target.value)}
                  className="w-full rounded-lg border border-white/10 bg-slate-800 px-3 py-2.5 text-sm text-slate-100 placeholder-slate-500 focus:border-purple-400 focus:ring-2 focus:ring-purple-500/20 outline-none transition-colors"
                  placeholder="logo.svg, hero-image.png, brand-colors.json"
                />
                <p className="mt-1.5 text-xs text-slate-400">
                  Leave empty if not using custom assets
                </p>
              </div>

              {/* Info Box */}
              <div className="p-3 rounded-lg border border-blue-500/20 bg-blue-500/5">
                <div className="flex items-start gap-2 text-xs text-blue-200">
                  <Rocket className="h-3.5 w-3.5 mt-0.5 flex-shrink-0 text-blue-400" />
                  <span>
                    This will file an issue and trigger an AI agent to customize your landing page. The agent will modify content, styling, and structure based on your brief.
                  </span>
                </div>
              </div>

              {/* Error */}
              {error && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm flex items-start gap-2">
                  <AlertCircle className="h-4 w-4 mt-0.5 text-red-400 flex-shrink-0" />
                  <div className="flex-1">
                    <p className="text-red-200">{error}</p>
                    <button
                      onClick={() => setError(null)}
                      className="mt-2 text-xs text-red-300 hover:text-red-200 underline"
                    >
                      Dismiss
                    </button>
                  </div>
                </div>
              )}
            </>
          ) : (
            /* Success Result */
            <div className="text-center space-y-6 py-4">
              <div className="flex justify-center">
                <div className="relative">
                  <div className="absolute inset-0 bg-purple-500/30 rounded-full blur-2xl" />
                  <div className="relative w-20 h-20 rounded-full bg-purple-500/20 border-2 border-purple-500/40 flex items-center justify-center">
                    <CheckCircle className="h-10 w-10 text-purple-400" />
                  </div>
                  <Sparkles className="absolute -top-1 -right-1 h-8 w-8 text-yellow-400 animate-pulse" />
                </div>
              </div>

              <div className="space-y-2">
                <h3 className="text-xl font-bold text-slate-100">Customization Queued!</h3>
                <p className="text-slate-300">
                  An AI agent is now working on your landing page
                </p>
              </div>

              <div className="rounded-lg border border-white/10 bg-slate-800/50 p-4 text-left space-y-3">
                <div className="grid sm:grid-cols-2 gap-3 text-sm">
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Issue ID</span>
                    <p className="font-mono text-purple-300">{result.issue_id || 'unknown'}</p>
                  </div>
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Status</span>
                    <p className="text-emerald-300">{result.status}</p>
                  </div>
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Agent</span>
                    <p className="text-slate-200">{result.agent || 'auto'}</p>
                  </div>
                  <div>
                    <span className="text-slate-400 text-xs uppercase">Run ID</span>
                    <p className="font-mono text-slate-300 text-xs">{result.run_id || 'pending'}</p>
                  </div>
                </div>
                {result.tracker_url && (
                  <div className="pt-2 border-t border-white/10">
                    <span className="text-slate-400 text-xs uppercase">Tracker API</span>
                    <p className="font-mono text-xs text-slate-300 break-all">{result.tracker_url}</p>
                  </div>
                )}
                {result.message && (
                  <p className="text-xs text-slate-400 pt-2 border-t border-white/10">{result.message}</p>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 sm:p-6 border-t border-white/10 bg-slate-900/80">
          {!result ? (
            <>
              <button
                onClick={handleClose}
                className="px-4 py-2.5 text-sm font-medium text-slate-300 hover:text-white border border-white/10 hover:border-white/20 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCustomize}
                disabled={customizing || !brief.trim()}
                className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-bold bg-gradient-to-r from-purple-500/30 to-pink-500/30 border-2 border-purple-500/60 text-purple-100 rounded-lg hover:from-purple-500/40 hover:to-pink-500/40 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-purple-500/20"
              >
                {customizing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Wand2 className="h-4 w-4" />
                )}
                Start AI Customization
              </button>
            </>
          ) : (
            <button
              onClick={handleClose}
              className="inline-flex items-center gap-2 px-6 py-2.5 text-sm font-semibold bg-purple-500/20 border border-purple-500/40 text-purple-200 rounded-lg hover:bg-purple-500/30 transition-colors"
            >
              Done
            </button>
          )}
        </div>
      </div>
    </div>
  );
});
