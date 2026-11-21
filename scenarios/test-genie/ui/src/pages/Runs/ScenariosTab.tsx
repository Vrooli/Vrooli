import { ScenarioTable } from "../../components/tables/ScenarioTable";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import type { ScenarioDirectoryEntry } from "../../hooks/useScenarios";
import { useTargets } from "../../hooks/useTargets";

export function ScenariosTab() {
  const { scenarioDirectoryEntries, isLoading } = useScenarios();
  const { targets, isLoading: targetsLoading } = useTargets();
  const {
    navigateToScenarioDetail,
    applyFocusScenario,
    setExecutionForm,
  } = useUIStore();

  const handleScenarioClick = (scenarioName: string) => {
    navigateToScenarioDetail(scenarioName);
  };

  const handleRunClick = (scenario: ScenarioDirectoryEntry) => {
    applyFocusScenario(scenario.scenarioName);
    setExecutionForm({
      scenarioName: scenario.scenarioName,
      preset: scenario.lastExecutionPreset ?? "quick",
      failFast: true
    });
    // Navigate to scenario detail to show run form
    navigateToScenarioDetail(scenario.scenarioName);
  };

  return (
    <div className="space-y-6">
      <section className="rounded-2xl border border-cyan-400/20 bg-cyan-400/[0.04] p-6">
        <p className="text-xs uppercase tracking-[0.25em] text-cyan-200">Repository targets</p>
        <h2 className="mt-2 text-2xl font-semibold">Target browser</h2>
        <p className="mt-2 text-sm text-slate-300">
          Every executable and static validation run is addressed by this typed target inventory. Scenarios remain available below as the legacy detail view.
        </p>
        <div className="mt-4 flex flex-wrap gap-2" data-testid="validation-target-browser">
          {targetsLoading && <span className="text-sm text-slate-400">Loading target inventory…</span>}
          {!targetsLoading && targets.slice(0, 80).map((target) => (
            <span key={`${target.kind}:${target.id}`} className="rounded-full border border-white/10 bg-black/20 px-3 py-1 text-xs text-slate-200" title={target.root}>
              {target.kind}:{target.id}
            </span>
          ))}
          {!targetsLoading && targets.length === 0 && <span className="text-sm text-slate-400">Target inventory unavailable.</span>}
        </div>
        {targets.length > 80 && <p className="mt-3 text-xs text-slate-500">Showing 80 of {targets.length} targets; use the CLI/API for filtered inventory queries.</p>}
      </section>
      <section className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
      <div className="mb-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">All Scenarios</p>
        <h2 className="mt-2 text-2xl font-semibold">Scenario directory</h2>
        <p className="mt-2 text-sm text-slate-300">
          Browse all tracked scenarios. Click a row to view details and run tests.
        </p>
      </div>

      <ScenarioTable
        scenarios={scenarioDirectoryEntries}
        onScenarioClick={handleScenarioClick}
        onRunClick={handleRunClick}
        isLoading={isLoading}
      />
      </section>
    </div>
  );
}
