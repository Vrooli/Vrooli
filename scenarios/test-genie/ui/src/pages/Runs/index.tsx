import { SubtabNav } from "../../components/layout/SubtabNav";
import { ScenariosTab } from "./ScenariosTab";
import { HistoryTab } from "./HistoryTab";
import { ScenarioDetail } from "./ScenarioDetail";
import { useUIStore } from "../../stores/uiStore";

export function RunsPage() {
  const { runsSubtab, setRunsSubtab, selectedScenario } = useUIStore();

  // If a scenario is selected, show the detail view
  if (selectedScenario) {
    return <ScenarioDetail scenarioName={selectedScenario} />;
  }

  return (
    <div className="space-y-6">
      {/* Subtab Navigation */}
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Test Runs</p>
          <h1 className="mt-2 text-2xl font-semibold">
            {runsSubtab === "scenarios" ? "Scenarios" : "History"}
          </h1>
        </div>
        <SubtabNav activeSubtab={runsSubtab} onSubtabChange={setRunsSubtab} />
      </div>

      {/* Content */}
      {runsSubtab === "scenarios" && <ScenariosTab />}
      {runsSubtab === "history" && <HistoryTab />}
    </div>
  );
}

export { ScenariosTab, HistoryTab, ScenarioDetail };
