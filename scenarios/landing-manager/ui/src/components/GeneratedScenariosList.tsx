import { memo, useState } from 'react';
import {
  AlertCircle,
  Calendar,
  Check,
  CheckCircle,
  Copy,
  ExternalLink,
  FileOutput,
  FileText,
  Globe,
  HelpCircle,
  Loader2,
  Play,
  RefreshCcw,
  Rocket,
  RotateCw,
  Settings,
  Sparkles,
  Square,
  Zap,
} from 'lucide-react';
import { type GeneratedScenario, listGeneratedScenarios } from '../lib/api';
import { Tooltip } from './Tooltip';

interface GeneratedScenariosListProps {
  generated: GeneratedScenario[];
  loadingGenerated: boolean;
  generatedError: string | null;
  scenarioStatuses: Record<string, { running: boolean; loading: boolean }>;
  previewLinks: Record<string, { links: { public?: string; admin?: string } }>;
  showLogs: Record<string, boolean>;
  scenarioLogs: Record<string, string>;
  onRefresh: (scenarios: GeneratedScenario[], error: string | null, loading: boolean) => void;
  onStartScenario: (scenarioId: string) => void;
  onStopScenario: (scenarioId: string) => void;
  onRestartScenario: (scenarioId: string) => void;
  onToggleLogs: (scenarioId: string) => void;
  onPromoteScenario: (scenarioId: string) => void;
  onSelectScenario: (slug: string) => void;
}

export const GeneratedScenariosList = memo(function GeneratedScenariosList({
  generated,
  loadingGenerated,
  generatedError,
  scenarioStatuses,
  previewLinks,
  showLogs,
  scenarioLogs,
  onRefresh,
  onStartScenario,
  onStopScenario,
  onRestartScenario,
  onToggleLogs,
  onPromoteScenario,
  onSelectScenario,
}: GeneratedScenariosListProps) {
  const [copiedSlug, setCopiedSlug] = useState(false);

  const handleRefresh = async () => {
    try {
      onRefresh([], null, true);
      const scenarios = await listGeneratedScenarios();
      onRefresh(scenarios, null, false);
    } catch (err) {
      onRefresh([], err instanceof Error ? err.message : 'Failed to refresh generated scenarios', false);
    }
  };

  const handleCopySlug = (slug: string) => {
    onSelectScenario(slug);
    navigator.clipboard.writeText(slug);
    setCopiedSlug(true);
    setTimeout(() => setCopiedSlug(false), 2000);
    const agentForm = document.querySelector('[data-testid="agent-customization-form"]');
    agentForm?.scrollIntoView({ behavior: 'smooth', block: 'center' });
  };

  const handleScrollToForm = () => {
    const formElement = document.querySelector('[data-testid="generation-form"]');
    formElement?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    const nameInput = document.getElementById('generation-name-input') as HTMLInputElement;
    nameInput?.focus();
  };

  return (
    <section
      className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden"
      data-testid="generated-scenarios"
      aria-labelledby="generated-scenarios-heading"
    >
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent"></div>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
          <div className="flex-1">
            <div className="flex items-center gap-2 mb-2">
              <h2 id="generated-scenarios-heading" className="text-xl sm:text-2xl font-semibold">
                Generated Landing Pages
              </h2>
              <Tooltip content="Your staging area for testing landing pages before deploying to production. All scenarios start here in the generated/ folder where you can iterate, test, and refine before moving to scenarios/.">
                <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
              </Tooltip>
            </div>
            <p className="text-xs sm:text-sm text-slate-400">Staging workspace for testing and iteration</p>
          </div>
          <button
            data-testid="refresh-generated-button"
            className="inline-flex items-center gap-2 px-3 py-2 text-sm text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
            onClick={handleRefresh}
            disabled={loadingGenerated}
            aria-label="Refresh generated scenarios list"
          >
            <RefreshCcw className={`h-4 w-4 ${loadingGenerated ? 'animate-spin' : ''}`} aria-hidden="true" />
            <span className="hidden sm:inline">Refresh</span>
          </button>
        </div>

        {/* Staging Area Workflow Explanation */}
        <div
          className="rounded-xl border border-purple-500/30 bg-gradient-to-br from-purple-500/10 to-blue-500/10 p-4 sm:p-5"
          role="region"
          aria-label="Staging workflow explanation"
        >
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 w-9 h-9 rounded-xl bg-purple-500/20 border border-purple-500/40 flex items-center justify-center">
              <FileOutput className="h-5 w-5 text-purple-300" aria-hidden="true" />
            </div>
            <div className="flex-1 space-y-3">
              <div>
                <h3 className="text-base font-bold text-purple-100 mb-1">Staging Workflow</h3>
                <p className="text-xs text-purple-200/80">
                  Your safe testing zone. All scenarios below live in{' '}
                  <code className="px-1.5 py-0.5 rounded bg-slate-900/60 text-emerald-300 font-mono">generated/</code> where
                  you can experiment without affecting production.
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
                When satisfied with your landing page, move it from{' '}
                <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">generated/&lt;slug&gt;/</code> to{' '}
                <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">scenarios/&lt;slug&gt;/</code>{' '}
                for permanent deployment.
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
        <div
          className="rounded-2xl border-2 border-dashed border-slate-700 bg-gradient-to-br from-slate-900/60 to-slate-800/40 p-8 sm:p-12 text-center space-y-6"
          role="status"
        >
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
            onClick={handleScrollToForm}
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
                    <h3
                      className="text-base sm:text-lg font-semibold text-slate-100 truncate"
                      data-testid={`generated-scenario-name-${scenario.scenario_id}`}
                    >
                      {scenario.name}
                    </h3>
                    <div className="flex items-center gap-2 mt-1">
                      <p className="text-xs text-slate-400" data-testid={`generated-scenario-slug-${scenario.scenario_id}`}>
                        Slug: <code className="px-1 py-0.5 rounded bg-slate-800 font-mono">{scenario.scenario_id}</code>
                      </p>
                      <button
                        onClick={() => handleCopySlug(scenario.scenario_id)}
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
                      data-testid={`scenario-status-badge-${scenario.scenario_id}`}
                      data-scenario-status-badge
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
                        onClick={() => onStartScenario(scenario.scenario_id)}
                        disabled={status.loading || status.running}
                        className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 hover:border-emerald-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-emerald-500"
                        aria-label="Start scenario"
                        title="Start this landing page scenario"
                        data-testid={`lifecycle-start-button-${scenario.scenario_id}`}
                        data-lifecycle-start-button
                      >
                        <Play className="h-3.5 w-3.5" aria-hidden="true" />
                        <span className="hidden sm:inline">Start</span>
                      </button>
                      <button
                        onClick={() => onStopScenario(scenario.scenario_id)}
                        disabled={status.loading || !status.running}
                        className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-red-500/40 bg-red-500/10 text-red-300 hover:bg-red-500/20 hover:border-red-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-red-500"
                        aria-label="Stop scenario"
                        title="Stop this landing page scenario"
                        data-testid={`lifecycle-stop-button-${scenario.scenario_id}`}
                        data-lifecycle-stop-button
                      >
                        <Square className="h-3.5 w-3.5" aria-hidden="true" />
                        <span className="hidden sm:inline">Stop</span>
                      </button>
                      <button
                        onClick={() => onRestartScenario(scenario.scenario_id)}
                        disabled={status.loading || !status.running}
                        className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-blue-500/40 bg-blue-500/10 text-blue-300 hover:bg-blue-500/20 hover:border-blue-400 disabled:opacity-40 disabled:cursor-not-allowed transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
                        aria-label="Restart scenario"
                        title="Restart this landing page scenario"
                        data-testid={`lifecycle-restart-button-${scenario.scenario_id}`}
                        data-lifecycle-restart-button
                      >
                        <RotateCw className="h-3.5 w-3.5" aria-hidden="true" />
                        <span className="hidden sm:inline">Restart</span>
                      </button>
                    </div>
                    <div className="flex gap-2 flex-1 sm:flex-none">
                      <button
                        onClick={() => onToggleLogs(scenario.scenario_id)}
                        className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-slate-500/30 bg-slate-500/10 text-slate-300 hover:bg-slate-500/20 hover:border-slate-400 transition-all focus:outline-none focus:ring-2 focus:ring-slate-500"
                        aria-label={showLogs[scenario.scenario_id] ? 'Hide logs' : 'Show logs'}
                        title={showLogs[scenario.scenario_id] ? 'Hide scenario logs' : 'View scenario logs'}
                        data-testid={`lifecycle-logs-button-${scenario.scenario_id}`}
                        data-lifecycle-logs-button
                      >
                        <FileOutput className="h-3.5 w-3.5" aria-hidden="true" />
                        <span>{showLogs[scenario.scenario_id] ? 'Hide' : 'Show'} Logs</span>
                      </button>
                      {!status.running && !status.loading && (
                        <button
                          onClick={() => onPromoteScenario(scenario.scenario_id)}
                          className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-bold rounded-lg border-2 border-purple-500/60 bg-gradient-to-r from-purple-500/20 to-pink-500/20 text-purple-200 hover:from-purple-500/30 hover:to-pink-500/30 hover:border-purple-400 transition-all shadow-lg shadow-purple-500/20 focus:outline-none focus:ring-2 focus:ring-purple-500"
                          aria-label="Promote to production"
                          title="Move this scenario from staging (generated/) to production (scenarios/)"
                          data-testid={`lifecycle-promote-button-${scenario.scenario_id}`}
                          data-lifecycle-promote-button
                        >
                          <Rocket className="h-3.5 w-3.5" aria-hidden="true" />
                          <span>Promote to Production</span>
                        </button>
                      )}
                    </div>
                  </div>

                  {/* Logs Display */}
                  {showLogs[scenario.scenario_id] && (
                    <div className="rounded-lg border border-white/10 bg-slate-900/60 p-3" data-testid={`scenario-logs-display-${scenario.scenario_id}`} data-scenario-logs-display>
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
                            data-testid="scenario-public-link"
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
                            data-testid="scenario-admin-link"
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
                            Click{' '}
                            <strong className="text-blue-100 bg-blue-500/20 px-1.5 py-0.5 rounded">Start</strong> above to
                            launch. You'll instantly see access links to your live landing page and admin dashboard.
                          </p>
                          <div className="pt-2 border-t border-blue-500/20">
                            <p className="text-xs text-blue-300/70 leading-relaxed flex items-start gap-1.5">
                              <Zap className="h-3.5 w-3.5 flex-shrink-0 mt-0.5 text-blue-300" aria-hidden="true" />
                              <span>
                                <strong className="text-blue-200">Testing zone:</strong> Lives in{' '}
                                <code className="px-1 py-0.5 rounded bg-slate-900/60 text-emerald-300 font-mono text-[10px]">
                                  {scenario.path}
                                </code>
                              </span>
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
  );
});
