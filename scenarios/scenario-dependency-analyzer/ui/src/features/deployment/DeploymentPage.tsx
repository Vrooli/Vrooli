import type { ScenarioSummary } from "../../types";
import { DeploymentDashboard } from "./DeploymentDashboard";

interface DeploymentPageProps {
  scenarios: ScenarioSummary[];
  loading: boolean;
  onRefresh: () => void;
  onScanScenario: (scenarioName: string, apply?: boolean) => void | Promise<void>;
  onSelectScenario: (scenarioName: string, options?: { openCatalog?: boolean }) => void;
}

export function DeploymentPage({
  scenarios,
  loading,
  onRefresh,
  onScanScenario,
  onSelectScenario,
}: DeploymentPageProps) {
  return (
    <DeploymentDashboard
      scenarios={scenarios}
      loading={loading}
      onRefresh={onRefresh}
      onScanScenario={onScanScenario}
      onSelectScenario={onSelectScenario}
    />
  );
}
