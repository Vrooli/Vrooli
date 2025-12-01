import { memo, useMemo, useState, type MouseEvent } from 'react';
import {
  AlertCircle,
  AlertTriangle,
  Calendar,
  Check,
  CheckCircle,
  Copy,
  Eye,
  FileOutput,
  FileText,
  HelpCircle,
  Loader2,
  Play,
  RefreshCcw,
  Rocket,
  RotateCw,
  Square,
  Sparkles,
  Trash2,
  X,
  Zap,
} from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { type GeneratedScenario, listGeneratedScenarios } from '../lib/api';
import { Tooltip } from './Tooltip';
import { ErrorDisplay, parseApiError, type StructuredError } from './ErrorDisplay';

interface GeneratedScenariosListProps {
  generated: GeneratedScenario[];
  loadingGenerated: boolean;
  generatedError: string | null;
  scenarioStatuses: Record<string, { running: boolean; loading: boolean }>;
  onRefresh: (scenarios: GeneratedScenario[], error: string | null, loading: boolean) => void;
  onStartScenario: (scenarioId: string) => void;
  onStopScenario: (scenarioId: string) => void;
  onRestartScenario: (scenarioId: string) => void;
  onPromoteScenario: (scenarioId: string) => void;
  onDeleteScenario: (scenarioId: string) => void;
  onSelectScenario: (slug: string) => void;
  onCreateClick: () => void;
}

export const GeneratedScenariosList = memo(function GeneratedScenariosList({
  generated,
  loadingGenerated,
  generatedError,
  scenarioStatuses,
  onRefresh,
  onStartScenario,
  onStopScenario,
  onRestartScenario,
  onPromoteScenario,
  onDeleteScenario,
  onSelectScenario,
  onCreateClick,
}: GeneratedScenariosListProps) {
  const [copiedPath, setCopiedPath] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const navigate = useNavigate();

  const interactiveSelector = useMemo(() => 'button, a, input, textarea, select, [data-card-stop]', []);

  const handleDeleteClick = (scenarioId: string) => {
    setDeleteConfirm(scenarioId);
  };

  const handleDeleteConfirm = () => {
    if (deleteConfirm) {
      onDeleteScenario(deleteConfirm);
      setDeleteConfirm(null);
    }
  };

  const handleDeleteCancel = () => {
    setDeleteConfirm(null);
  };

  const handleRefresh = async () => {
    try {
      onRefresh([], null, true);
      const scenarios = await listGeneratedScenarios();
      onRefresh(scenarios, null, false);
    } catch (err) {
      onRefresh([], err instanceof Error ? err.message : 'Failed to refresh generated scenarios', false);
    }
  };

  const handleCopyPath = (scenario: GeneratedScenario) => {
    onSelectScenario(scenario.scenario_id);
    navigator.clipboard.writeText(scenario.path);
    setCopiedPath(true);
    setTimeout(() => setCopiedPath(false), 2000);
  };

  const handleCardClick = (scenario: GeneratedScenario) => (event: MouseEvent<HTMLElement>) => {
    const element = event.target as HTMLElement | null;
    if (element && element.closest(interactiveSelector)) {
      return;
    }
    navigate(`/scenarios/${scenario.scenario_id}/preview`);
  };

  const handlePreviewClick = (scenario: GeneratedScenario) => {
    navigate(`/scenarios/${scenario.scenario_id}/preview`);
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
        <ErrorDisplay
          error={parseApiError(generatedError)}
          onRetry={handleRefresh}
          onDismiss={() => onRefresh(generated, null, false)}
          testId="generated-scenarios-error"
        />
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
            onClick={onCreateClick}
            className="inline-flex items-center gap-2 px-8 py-4 text-base font-bold bg-gradient-to-r from-emerald-500/30 to-blue-500/30 border-2 border-emerald-500/60 text-emerald-200 rounded-xl hover:from-emerald-500/40 hover:to-blue-500/40 hover:border-emerald-400/80 hover:scale-105 transition-all shadow-xl shadow-emerald-500/25 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950"
            aria-label="Create your first landing page"
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

            return (
              <article
                key={scenario.scenario_id}
                className="rounded-xl border border-white/10 bg-slate-900/40 p-4 sm:p-5 hover:border-white/20 transition-colors cursor-pointer"
                data-testid={`generated-scenario-${scenario.scenario_id}`}
                role="listitem"
                onClick={handleCardClick(scenario)}
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
                      <p className="text-xs text-slate-400 font-mono" data-testid={`generated-scenario-slug-${scenario.scenario_id}`}>
                        {scenario.scenario_id}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 mt-2 text-xs text-slate-400">
                      <span>Location:</span>
                      <code className="px-1 py-0.5 rounded bg-slate-800 font-mono text-[10px]">{scenario.path}</code>
                      <button
                        onClick={() => handleCopyPath(scenario)}
                        data-card-stop
                        className="inline-flex items-center justify-center rounded-md border border-slate-500/30 bg-slate-500/10 p-1 text-slate-400 hover:text-white hover:bg-slate-500/30 transition-colors focus:outline-none focus:ring-1 focus:ring-emerald-500"
                        aria-label={`Copy path ${scenario.path}`}
                        title="Copy scenario path"
                      >
                        {copiedPath ? <Check className="h-3 w-3" aria-hidden="true" /> : <Copy className="h-3 w-3" aria-hidden="true" />}
                      </button>
                    </div>
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
                        data-card-stop
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
                        data-card-stop
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
                        data-card-stop
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
                        onClick={() => handlePreviewClick(scenario)}
                        data-card-stop
                        className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-blue-500/40 bg-blue-500/10 text-blue-300 hover:bg-blue-500/20 hover:border-blue-400 transition-all focus:outline-none focus:ring-2 focus:ring-blue-500"
                        aria-label="Preview scenario"
                        title="Open preview for this landing page"
                        data-testid={`preview-button-${scenario.scenario_id}`}
                      >
                        <Eye className="h-3.5 w-3.5" aria-hidden="true" />
                        <span>Preview</span>
                      </button>
                      {!status.running && !status.loading && (
                        <>
                          <button
                            onClick={() => onPromoteScenario(scenario.scenario_id)}
                            data-card-stop
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-bold rounded-lg border-2 border-purple-500/60 bg-gradient-to-r from-purple-500/20 to-pink-500/20 text-purple-200 hover:from-purple-500/30 hover:to-pink-500/30 hover:border-purple-400 transition-all shadow-lg shadow-purple-500/20 focus:outline-none focus:ring-2 focus:ring-purple-500"
                            aria-label="Promote to production"
                            title="Move this scenario from staging (generated/) to production (scenarios/)"
                            data-testid={`lifecycle-promote-button-${scenario.scenario_id}`}
                            data-lifecycle-promote-button
                          >
                            <Rocket className="h-3.5 w-3.5" aria-hidden="true" />
                            <span>Promote to Production</span>
                          </button>
                          <button
                            onClick={() => handleDeleteClick(scenario.scenario_id)}
                            data-card-stop
                            className="flex-1 sm:flex-none inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium rounded-lg border border-red-500/40 bg-red-500/10 text-red-300 hover:bg-red-500/20 hover:border-red-400 transition-all focus:outline-none focus:ring-2 focus:ring-red-500"
                            aria-label="Delete scenario"
                            title="Permanently delete this scenario from staging"
                            data-testid={`lifecycle-delete-button-${scenario.scenario_id}`}
                            data-lifecycle-delete-button
                          >
                            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                            <span className="hidden sm:inline">Delete</span>
                          </button>
                        </>
                      )}
                    </div>
                  </div>

                </div>
              </article>
            );
          })}
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
          onClick={handleDeleteCancel}
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-dialog-title"
        >
          <div
            className="relative w-full max-w-md mx-4 rounded-2xl border border-red-500/30 bg-slate-900 p-6 shadow-2xl shadow-red-500/10"
            onClick={(e) => e.stopPropagation()}
          >
            <button
              onClick={handleDeleteCancel}
              className="absolute top-4 right-4 p-1 text-slate-400 hover:text-white transition-colors"
              aria-label="Close dialog"
            >
              <X className="h-5 w-5" />
            </button>

            <div className="flex items-start gap-4 mb-6">
              <div className="flex-shrink-0 w-12 h-12 rounded-full bg-red-500/20 border border-red-500/40 flex items-center justify-center">
                <AlertTriangle className="h-6 w-6 text-red-400" aria-hidden="true" />
              </div>
              <div>
                <h3 id="delete-dialog-title" className="text-lg font-bold text-white mb-1">
                  Delete Landing Page?
                </h3>
                <p className="text-sm text-slate-300">
                  Are you sure you want to delete{' '}
                  <code className="px-1.5 py-0.5 rounded bg-slate-800 font-mono text-red-300">{deleteConfirm}</code>?
                </p>
              </div>
            </div>

            <div className="rounded-lg border border-red-500/20 bg-red-500/5 p-4 mb-6">
              <p className="text-sm text-red-200 leading-relaxed">
                <strong>This action is permanent.</strong> The scenario directory and all its contents will be removed
                from the staging area. This cannot be undone.
              </p>
            </div>

            <div className="flex gap-3 justify-end">
              <button
                onClick={handleDeleteCancel}
                className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white bg-slate-800 hover:bg-slate-700 border border-slate-600 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-slate-500"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteConfirm}
                className="px-4 py-2 text-sm font-bold text-white bg-red-600 hover:bg-red-500 border border-red-500 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-red-500 shadow-lg shadow-red-500/20"
                data-testid="delete-confirm-button"
              >
                Delete Permanently
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
});
