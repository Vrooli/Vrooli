import { MetricsPanel } from "./components/MetricsPanel";
import { useKnowledgeMetrics } from "../../shared/hooks/knowledgeHooks";

export function MetricsPanelContainer() {
  const { viewModel, isLoading, errorMessage, hasError, hasData, refetch } = useKnowledgeMetrics();

  return (
    <MetricsPanel
      isLoading={isLoading}
      hasError={hasError}
      errorMessage={errorMessage}
      hasData={hasData}
      viewModel={viewModel}
      onRetry={() => refetch()}
    />
  );
}
