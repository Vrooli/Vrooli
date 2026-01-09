import { ScenarioCatalogPanel } from "../components/ScenarioCatalogPanel";
import { ScenarioDetailPanel } from "../components/ScenarioDetailPanel";
import type { ScenarioSummary, ScenarioDetailResponse } from "../types";

interface CatalogPageProps {
  scenarios: ScenarioSummary[];
  selectedScenario: string | null;
  detail: ScenarioDetailResponse | null;
  loadingSummaries: boolean;
  detailLoading: boolean;
  scanLoading: boolean;
  optimizeLoading: boolean;
  onSelectScenario: (scenarioName: string) => void;
  onRefresh: () => void;
  onScan: (options?: { apply?: boolean }) => void;
  onOptimize: (options?: { apply?: boolean }) => void;
}

export function CatalogPage({
  scenarios,
  selectedScenario,
  detail,
  loadingSummaries,
  detailLoading,
  scanLoading,
  optimizeLoading,
  onSelectScenario,
  onRefresh,
  onScan,
  onOptimize,
}: CatalogPageProps) {
  return (
    <section className="grid gap-6 lg:grid-cols-[320px_minmax(0,1fr)]">
      <ScenarioCatalogPanel
        scenarios={scenarios}
        selected={selectedScenario}
        loading={loadingSummaries}
        onSelect={onSelectScenario}
        onRefresh={onRefresh}
      />
      <ScenarioDetailPanel
        detail={detail}
        loading={detailLoading}
        scanning={scanLoading}
        optimizing={optimizeLoading}
        onScan={onScan}
        onOptimize={onOptimize}
      />
    </section>
  );
}
