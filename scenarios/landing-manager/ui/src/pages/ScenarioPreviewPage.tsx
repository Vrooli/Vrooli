import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { AlertCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { LandingPreviewView } from '../components/LandingPreviewView';
import { ErrorDisplay, parseApiError } from '../components/ErrorDisplay';
import { PromoteDialog } from '../components/PromoteDialog';
import { type GeneratedScenario, listGeneratedScenarios, promoteScenario } from '../lib/api';
import { useScenarioLifecycle } from '../hooks/useScenarioLifecycle';

export function ScenarioPreviewPage() {
  const { scenarioId } = useParams<{ scenarioId: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const {
    statuses,
    previewLinks,
    logs: lifecycleLogs,
    logsLoading,
    lifecycleError,
    clearError,
    startScenario,
    stopScenario,
    restartScenario,
    loadStatuses,
    loadLogs,
  } = useScenarioLifecycle();

  const [scenario, setScenario] = useState<GeneratedScenario | null>(null);
  const [loadingScenario, setLoadingScenario] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [promoteDialogScenarioId, setPromoteDialogScenarioId] = useState<string | null>(null);
  const [promotionSuccess, setPromotionSuccess] = useState<string | null>(null);
  const [promotionError, setPromotionError] = useState<string | null>(null);
  const [promoteInFlight, setPromoteInFlight] = useState(false);

  const initialView = searchParams.get('view') === 'admin' ? 'admin' : 'public';

  const isRunning = scenario ? statuses[scenario.scenario_id]?.running ?? false : false;
  const isBusy = scenario ? statuses[scenario.scenario_id]?.loading ?? false : false;

  useEffect(() => {
    if (!scenarioId) {
      setScenario(null);
      setLoadingScenario(false);
      setLoadError('Missing scenario id');
      return;
    }

    let isMounted = true;
    const fetchScenario = async () => {
      try {
        setLoadingScenario(true);
        const scenarios = await listGeneratedScenarios();
        if (!isMounted) return;

        const match = scenarios.find((s) => s.scenario_id === scenarioId);
        if (!match) {
          setLoadError(`Scenario "${scenarioId}" was not found in generated/`);
          setScenario(null);
        } else {
          setScenario(match);
          setLoadError(null);
        }
      } catch (error) {
        if (!isMounted) return;
        setLoadError(error instanceof Error ? error.message : 'Failed to load scenario');
        setScenario(null);
      } finally {
        if (isMounted) {
          setLoadingScenario(false);
        }
      }
    };

    void fetchScenario();
    return () => {
      isMounted = false;
    };
  }, [scenarioId]);

  useEffect(() => {
    if (scenario) {
      void loadStatuses([scenario.scenario_id]);
    }
  }, [scenario, loadStatuses]);

  const handleBack = () => navigate('/');

  const handleReloadStatus = useCallback(() => {
    if (scenario) {
      void loadStatuses([scenario.scenario_id]);
    }
  }, [scenario, loadStatuses]);

  const handleRequestPromote = useCallback((target: GeneratedScenario) => {
    setPromotionError(null);
    setPromotionSuccess(null);
    setPromoteDialogScenarioId(target.scenario_id);
  }, []);

  const confirmPromote = useCallback(async () => {
    if (!promoteDialogScenarioId || promoteInFlight) {
      return;
    }
    try {
      setPromoteInFlight(true);
      setPromotionError(null);
      setPromotionSuccess(null);
      const result = await promoteScenario(promoteDialogScenarioId);
      setPromotionSuccess(
        result.production_path
          ? `Promoted ${promoteDialogScenarioId} to ${result.production_path}`
          : `Promoted ${promoteDialogScenarioId}`,
      );
      setPromoteDialogScenarioId(null);
    } catch (error) {
      setPromotionError(error instanceof Error ? error.message : 'Failed to promote scenario');
    } finally {
      setPromoteInFlight(false);
    }
  }, [promoteDialogScenarioId, promoteInFlight]);

  return (
    <div className="h-screen bg-slate-950 text-slate-50 flex flex-col">
      <main className="flex-1 relative flex flex-col min-h-0">
        {(lifecycleError || promotionError || promotionSuccess) && (
          <div className="w-full flex justify-center px-4 pt-6 sm:px-8 lg:px-16">
            <div className="w-full max-w-5xl flex flex-col gap-4">
              {lifecycleError && (
                <ErrorDisplay error={lifecycleError} onDismiss={clearError} size="sm" />
              )}
              {promotionError && (
                <ErrorDisplay error={promotionError} onDismiss={() => setPromotionError(null)} size="sm" />
              )}
              {promotionSuccess && (
                <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200 flex items-start gap-2">
                  <CheckCircle2 className="h-4 w-4 mt-0.5 opacity-70" />
                  <p>{promotionSuccess}</p>
                </div>
              )}
            </div>
          </div>
        )}

        {loadingScenario && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3">
            <Loader2 className="h-8 w-8 animate-spin text-emerald-400" />
            <p className="text-sm text-slate-400">Loading scenario details...</p>
          </div>
        )}

        {loadError && !loadingScenario && (
          <div className="max-w-xl mx-auto mt-12 px-4">
            <ErrorDisplay
              error={parseApiError(loadError)}
              onRetry={handleReloadStatus}
              onDismiss={() => setLoadError(null)}
            />
          </div>
        )}

        {scenario && !loadingScenario && (
          <div className="flex-1 min-h-0 flex">
            <LandingPreviewView
              scenario={scenario}
              isRunning={isRunning}
              onClose={handleBack}
              onCustomize={() => undefined}
              onStartScenario={startScenario}
              onStopScenario={stopScenario}
              onRestartScenario={restartScenario}
              previewLinks={previewLinks[scenario.scenario_id]}
              initialView={initialView}
              lifecycleLoading={isBusy}
              enableInfoPanel
              scenarioLogs={lifecycleLogs[scenario.scenario_id]}
              logsLoading={logsLoading[scenario.scenario_id]}
              onLoadLogs={loadLogs}
              onPromoteScenario={handleRequestPromote}
            />
          </div>
        )}

        {!loadingScenario && !scenario && !loadError && (
          <div className="max-w-2xl mx-auto mt-16 px-4">
            <div className="rounded-2xl border border-white/10 bg-slate-900/60 p-6 flex items-start gap-3">
              <AlertCircle className="h-6 w-6 text-amber-400 flex-shrink-0" />
              <div>
                <p className="text-lg font-semibold">Scenario not found</p>
                <p className="text-sm text-slate-300 mt-2">
                  The requested scenario could not be located inside the generated workspace. Verify the slug and try again.
                </p>
              </div>
            </div>
          </div>
        )}
      </main>
      {promoteDialogScenarioId && (
        <PromoteDialog
          scenarioId={promoteDialogScenarioId}
          onClose={() => setPromoteDialogScenarioId(null)}
          onConfirm={confirmPromote}
        />
      )}
    </div>
  );
}

export default ScenarioPreviewPage;
