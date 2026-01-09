import { ScenarioTable } from "../../components/tables/ScenarioTable";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import type { ScenarioDirectoryEntry } from "../../hooks/useScenarios";

export function ScenariosTab() {
  const { scenarioDirectoryEntries, isLoading } = useScenarios();
  const {
    navigateToScenarioDetail,
    applyFocusScenario,
    setQueueForm,
    setExecutionForm,
    setActiveTab
  } = useUIStore();

  const handleScenarioClick = (scenarioName: string) => {
    navigateToScenarioDetail(scenarioName);
  };

  const handleQueueClick = (scenario: ScenarioDirectoryEntry) => {
    applyFocusScenario(scenario.scenarioName);
    setQueueForm({
      scenarioName: scenario.scenarioName,
      requestedTypes: scenario.lastRequestTypes?.length
        ? scenario.lastRequestTypes
        : ["unit", "integration"],
      coverageTarget: scenario.lastRequestCoverageTarget ?? 95,
      priority: scenario.lastRequestPriority ?? "normal",
      notes: scenario.lastRequestNotes ?? ""
    });
    setActiveTab("generate");
  };

  const handleRunClick = (scenario: ScenarioDirectoryEntry) => {
    applyFocusScenario(scenario.scenarioName);
    setExecutionForm({
      scenarioName: scenario.scenarioName,
      preset: scenario.lastExecutionPreset ?? "quick",
      failFast: true,
      suiteRequestId: ""
    });
    // Navigate to scenario detail to show run form
    navigateToScenarioDetail(scenario.scenarioName);
  };

  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
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
        onQueueClick={handleQueueClick}
        onRunClick={handleRunClick}
        isLoading={isLoading}
      />
    </div>
  );
}
