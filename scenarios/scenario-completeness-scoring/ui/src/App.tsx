import { useState } from "react";
import { Dashboard } from "./pages/Dashboard";
import { ScenarioDetail } from "./pages/ScenarioDetail";
import { Configuration } from "./pages/Configuration";
import { ErrorBoundary } from "./components/ErrorBoundary";

// [REQ:SCS-UI-001] Main app with navigation between dashboard, detail, and config views
// [REQ:SCS-CORE-003] Graceful degradation with error boundaries
export default function App() {
  const [selectedScenario, setSelectedScenario] = useState<string | null>(null);
  const [configOpen, setConfigOpen] = useState(false);

  const handleOpenConfig = () => setConfigOpen(true);

  // Show scenario detail if one is selected
  if (selectedScenario) {
    return (
      <>
        <ErrorBoundary context="Scenario Detail">
          <ScenarioDetail
            scenario={selectedScenario}
            onBack={() => setSelectedScenario(null)}
            onOpenConfig={handleOpenConfig}
          />
        </ErrorBoundary>
        {configOpen && (
          <ErrorBoundary context="Configuration">
            <Configuration onClose={() => setConfigOpen(false)} />
          </ErrorBoundary>
        )}
      </>
    );
  }

  // Default: show dashboard
  return (
    <>
      <ErrorBoundary context="Dashboard">
        <Dashboard
          onSelectScenario={setSelectedScenario}
          onOpenConfig={handleOpenConfig}
        />
      </ErrorBoundary>
      {configOpen && (
        <ErrorBoundary context="Configuration">
          <Configuration onClose={() => setConfigOpen(false)} />
        </ErrorBoundary>
      )}
    </>
  );
}
