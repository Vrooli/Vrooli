import { useId, useMemo } from "react";
import { ArrowLeft } from "lucide-react";
import { Button } from "../../components/ui/button";
import { Breadcrumb } from "../../components/layout/Breadcrumb";
import { ScenarioDetailTabNav } from "../../components/layout/ScenarioDetailTabNav";
import { StatusPill } from "../../components/cards/StatusPill";
import { ExecutionCard } from "../../components/cards/ExecutionCard";
import { PhaseResultCard } from "../../components/cards/PhaseResultCard";
import { ExecutionForm } from "../../components/forms/ExecutionForm";
import { RequirementsPanel } from "../../components/requirements";
import { selectors } from "../../consts/selectors";
import { useScenarios } from "../../hooks/useScenarios";
import { useScenarioHistory } from "../../hooks/useExecutions";
import { useUIStore } from "../../stores/uiStore";
import { formatRelative } from "../../lib/formatters";

interface ScenarioDetailProps {
  scenarioName: string;
}

export function ScenarioDetail({ scenarioName }: ScenarioDetailProps) {
  const datalistId = useId();
  const { scenarioDirectoryEntries } = useScenarios();
  const { historyExecutions, isLoading: historyLoading } = useScenarioHistory(scenarioName);
  const {
    navigateBack,
    setRunsSubtab,
    applyFocusScenario,
    setExecutionForm,
    scenarioDetailTab,
    setScenarioDetailTab
  } = useUIStore();

  // Find the scenario summary
  const scenario = useMemo(
    () => scenarioDirectoryEntries.find((s) => s.scenarioName === scenarioName),
    [scenarioDirectoryEntries, scenarioName]
  );

  // Get scenario options for datalist
  const scenarioOptions = useMemo(
    () => scenarioDirectoryEntries.map((s) => s.scenarioName),
    [scenarioDirectoryEntries]
  );

  // Handle successful execution
  const handleExecutionSuccess = () => {
    // Could show a toast or navigate somewhere
  };

  const breadcrumbItems = [
    { label: "Runs", onClick: () => { navigateBack(); setRunsSubtab("scenarios"); } },
    { label: "Scenarios", onClick: () => { navigateBack(); setRunsSubtab("scenarios"); } },
    { label: scenarioName }
  ];

  const status = scenario?.pendingRequests && scenario.pendingRequests > 0
    ? "queued"
    : scenario?.lastExecutionSuccess === false
      ? "failed"
      : scenario?.lastExecutionSuccess === true
        ? "passed"
        : "idle";

  return (
    <div className="space-y-6" data-testid={selectors.runs.scenarioDetail}>
      {/* Navigation */}
      <div className="flex items-center gap-4">
        <Button
          variant="outline"
          size="sm"
          onClick={navigateBack}
          data-testid={selectors.runs.scenarioDetailBack}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Breadcrumb items={breadcrumbItems} />
      </div>

      {/* Scenario Header */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-3xl font-semibold">{scenarioName}</h1>
              <StatusPill status={status} />
            </div>
            {scenario?.scenarioDescription && (
              <p className="mt-2 text-sm text-slate-300">{scenario.scenarioDescription}</p>
            )}
            {scenario?.scenarioTags && scenario.scenarioTags.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-2">
                {scenario.scenarioTags.map((tag) => (
                  <span
                    key={tag}
                    className="rounded-full border border-white/10 px-3 py-1 text-xs uppercase tracking-wide text-slate-300"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            )}
          </div>

          <div className="grid gap-3 sm:grid-cols-3 text-center">
            <div className="rounded-xl border border-white/5 bg-black/30 p-3">
              <p className="text-2xl font-semibold">{scenario?.totalExecutions ?? 0}</p>
              <p className="text-xs text-slate-400">Total runs</p>
            </div>
            <div className="rounded-xl border border-white/5 bg-black/30 p-3">
              <p className="text-2xl font-semibold">{scenario?.pendingRequests ?? 0}</p>
              <p className="text-xs text-slate-400">Pending</p>
            </div>
            <div className="rounded-xl border border-white/5 bg-black/30 p-3">
              <p className="text-2xl font-semibold">
                {scenario?.lastExecutionAt ? formatRelative(scenario.lastExecutionAt) : "—"}
              </p>
              <p className="text-xs text-slate-400">Last run</p>
            </div>
          </div>
        </div>
      </section>

      {/* Tab Navigation */}
      <ScenarioDetailTabNav
        activeTab={scenarioDetailTab}
        onTabChange={setScenarioDetailTab}
      />

      {/* Tab Content */}
      {scenarioDetailTab === "overview" && (
        <OverviewTab
          scenario={scenario}
          scenarioOptions={scenarioOptions}
          datalistId={datalistId}
          onExecutionSuccess={handleExecutionSuccess}
        />
      )}

      {scenarioDetailTab === "requirements" && (
        <RequirementsPanel scenarioName={scenarioName} />
      )}

      {scenarioDetailTab === "history" && (
        <HistoryTab
          historyExecutions={historyExecutions}
          historyLoading={historyLoading}
          applyFocusScenario={applyFocusScenario}
          setExecutionForm={setExecutionForm}
        />
      )}

      {/* Hidden datalist for scenario autocomplete */}
      <datalist id={datalistId}>
        {scenarioOptions.map((name) => (
          <option key={name} value={name} />
        ))}
      </datalist>
    </div>
  );
}

// Overview Tab Component
interface OverviewTabProps {
  scenario: ReturnType<typeof useScenarios>["scenarioDirectoryEntries"][0] | undefined;
  scenarioOptions: string[];
  datalistId: string;
  onExecutionSuccess: () => void;
}

function OverviewTab({ scenario, scenarioOptions, datalistId, onExecutionSuccess }: OverviewTabProps) {
  return (
    <div className="space-y-6">
      {/* Latest Execution Summary */}
      {scenario?.lastExecutionPhases && scenario.lastExecutionPhases.length > 0 && (
        <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Last Run Details</p>
          <h2 className="mt-2 text-xl font-semibold">
            {scenario.lastExecutionPreset ? `${scenario.lastExecutionPreset} preset` : "Custom phases"}
            {" · "}
            {scenario.lastExecutionPhaseSummary?.passed}/{scenario.lastExecutionPhaseSummary?.total} phases passed
          </h2>
          <div className="mt-4 grid gap-2 md:grid-cols-2 lg:grid-cols-3">
            {scenario.lastExecutionPhases.map((phase) => (
              <PhaseResultCard key={phase.name} phase={phase} />
            ))}
          </div>
        </section>
      )}

      {/* Run Tests Form */}
      <ExecutionForm
        scenarioOptions={scenarioOptions}
        datalistId={datalistId}
        scenarioName={scenario?.scenarioName}
        onSuccess={onExecutionSuccess}
      />
    </div>
  );
}

// History Tab Component
interface HistoryTabProps {
  historyExecutions: ReturnType<typeof useScenarioHistory>["historyExecutions"];
  historyLoading: boolean;
  applyFocusScenario: (scenario: string) => void;
  setExecutionForm: (form: { scenarioName: string; preset: string; failFast: boolean; suiteRequestId: string }) => void;
}

function HistoryTab({ historyExecutions, historyLoading, applyFocusScenario, setExecutionForm }: HistoryTabProps) {
  return (
    <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
      <p className="text-xs uppercase tracking-[0.25em] text-slate-400">History</p>
      <h2 className="mt-2 text-xl font-semibold">Previous runs</h2>

      {historyLoading && (
        <p className="mt-4 text-sm text-slate-400">Loading history...</p>
      )}

      {!historyLoading && historyExecutions.length === 0 && (
        <p className="mt-4 text-sm text-slate-400">
          No previous runs for this scenario. Run tests to start tracking.
        </p>
      )}

      <div className="mt-4 space-y-4">
        {historyExecutions.slice(0, 10).map((execution) => (
          <ExecutionCard
            key={execution.executionId}
            execution={execution}
            onPrefill={() => {
              applyFocusScenario(execution.scenarioName);
              setExecutionForm({
                scenarioName: execution.scenarioName,
                preset: execution.preset ?? "quick",
                failFast: true,
                suiteRequestId: execution.suiteRequestId ?? ""
              });
            }}
          />
        ))}
      </div>
    </section>
  );
}
