import { useState } from "react";
import { Dashboard } from "./pages/Dashboard";
import { ScenarioDetail } from "./pages/ScenarioDetail";
import { Configuration } from "./pages/Configuration";

// [REQ:SCS-UI-001] Main app with navigation between dashboard, detail, and config views
export default function App() {
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null);
  const [configOpen, setConfigOpen] = useState(false);

  // Show scenario detail if one is selected
  if (selectedScenario) {
    return (
      <ScenarioDetail
        scenario={selectedScenario}
        onBack={() => setSelectedScenario(null)}
      />
    );
  }

  // Default: show dashboard
  return (
    <>
      <Dashboard
        onSelectScenario={setSelectedScenario}
        onOpenConfig={() => setConfigOpen(true)}
      />
      {configOpen && <Configuration onClose={() => setConfigOpen(false)} />}
    </>
  );
}
