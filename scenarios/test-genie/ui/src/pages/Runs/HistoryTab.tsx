import { HistoryTable } from "../../components/tables/HistoryTable";
import { useExecutions } from "../../hooks/useExecutions";
import { useUIStore } from "../../stores/uiStore";
import type { SuiteExecutionResult } from "../../lib/api";

export function HistoryTab() {
  const { executions, isLoading } = useExecutions(20);
  const {
    navigateToScenarioDetail,
    applyFocusScenario,
    setExecutionForm
  } = useUIStore();

  const handleRerunClick = (execution: SuiteExecutionResult) => {
    applyFocusScenario(execution.scenarioName);
    setExecutionForm({
      scenarioName: execution.scenarioName,
      preset: execution.preset ?? "quick",
      failFast: true,
      suiteRequestId: execution.suiteRequestId ?? ""
    });
    navigateToScenarioDetail(execution.scenarioName);
  };

  const handleScenarioClick = (scenarioName: string) => {
    navigateToScenarioDetail(scenarioName);
  };

  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-6">
      <div className="mb-6">
        <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test Runs</p>
        <h2 className="mt-2 text-2xl font-semibold">Run history</h2>
        <p className="mt-2 text-sm text-slate-300">
          View all test runs across scenarios. Filter by status or search by name.
        </p>
      </div>

      <HistoryTable
        executions={executions}
        onRerunClick={handleRerunClick}
        onScenarioClick={handleScenarioClick}
        isLoading={isLoading}
      />
    </div>
  );
}
