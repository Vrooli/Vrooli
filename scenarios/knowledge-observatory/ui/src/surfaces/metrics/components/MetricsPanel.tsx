// DOC: docs/concepts/ARCHITECTURE.md#health-and-metrics-flow
import { Database, TrendingUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { cn } from "../../../shared/lib/utils";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { MetricCardView, MetricsViewModel } from "../../../shared/controllers/knowledgeController";

// AI_CHECK: REACT_STABILITY=1 | LAST: 2026-01-25

const toneStyles: Record<MetricCardView["tone"], string> = {
  good: "ko-tone-good",
  medium: "ko-tone-medium",
  poor: "ko-tone-poor",
};

const EMPTY_METRIC_CARDS: MetricCardView[] = [];
const EMPTY_COLLECTIONS: MetricsViewModel["collections"] = [];
const DEFAULT_VIEW_MODEL: MetricsViewModel = {
  metricCards: EMPTY_METRIC_CARDS,
  collections: EMPTY_COLLECTIONS,
  overallHealth: "unknown",
  lastUpdated: "Unknown",
  totalEntriesLabel: "Unknown",
  hasMetrics: false,
};

function MetricCard({ label, percentageLabel, description, tone }: MetricCardView) {
  const styles = toneStyles[tone] ?? toneStyles.medium;

  return (
    <div className={cn("ko-metric-card", styles)}>
      <div className="flex items-start justify-between mb-2">
        <span className="ko-meta">{label}</span>
        <TrendingUp className="h-4 w-4" />
      </div>
      <div className="text-3xl font-bold">{percentageLabel}</div>
      <p className="ko-text-xs ko-subtle mt-2">{description}</p>
    </div>
  );
}

export type MetricsPanelProps = {
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  hasData: boolean;
  viewModel: MetricsViewModel;
  onRetry: () => void;
};

export function MetricsPanel({
  isLoading,
  hasError,
  errorMessage,
  hasData,
  viewModel,
  onRetry,
}: MetricsPanelProps) {
  const handleRetry = () => {
    if (typeof onRetry === "function") {
      onRetry();
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 ko-icon animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading metrics...</span>
      </div>
    );
  }

  if (hasError) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="ko-alert-title ko-text-danger-strong">Failed to load metrics</p>
          <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          <Button onClick={handleRetry} variant="danger" className="mt-3">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!hasData) {
    return (
      <div className="ko-panel p-6 text-center">
        <p className="ko-text-sm ko-muted">Metrics data is not available yet.</p>
        <Button onClick={handleRetry} variant="primary" className="mt-3">
          Retry
        </Button>
      </div>
    );
  }

  const safeViewModel = viewModel ?? DEFAULT_VIEW_MODEL;
  const metricCards = Array.isArray(safeViewModel.metricCards) ? safeViewModel.metricCards : EMPTY_METRIC_CARDS;
  const collections = Array.isArray(safeViewModel.collections) ? safeViewModel.collections : EMPTY_COLLECTIONS;
  const overallHealth =
    typeof safeViewModel.overallHealth === "string" && safeViewModel.overallHealth.trim().length > 0
      ? safeViewModel.overallHealth
      : "unknown";
  const lastUpdated =
    typeof safeViewModel.lastUpdated === "string" && safeViewModel.lastUpdated.trim().length > 0
      ? safeViewModel.lastUpdated
      : "Unknown";
  const totalEntriesLabel =
    typeof safeViewModel.totalEntriesLabel === "string" && safeViewModel.totalEntriesLabel.trim().length > 0
      ? safeViewModel.totalEntriesLabel
      : "Unknown";
  const hasMetrics = safeViewModel.hasMetrics || metricCards.length > 0;

  return (
    <div className="ko-stack">
      {/* Overall Status */}
      <div className="flex items-center justify-between" data-testid={selectors.metrics.overall}>
        <div>
          <h3 className="ko-text-lg font-semibold ko-text-strong">Overall Health</h3>
          <p className="ko-text-sm ko-muted capitalize">{overallHealth} condition</p>
        </div>
        <Button onClick={handleRetry} variant="outline" size="sm" data-testid={selectors.metrics.refresh}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      <div
        className="ko-card p-3 ko-text-xs ko-subtle"
        data-testid={selectors.metrics.legend}
      >
        Green signals healthy quality, yellow means watch closely, and red indicates degradation. Redundancy scores are
        better when lower.
      </div>

      {!hasMetrics && (
        <div className="ko-panel p-4 ko-text-sm ko-muted">
          Quality metrics are unavailable (not computed yet). Vector counts below are live when Qdrant is reachable.
        </div>
      )}

      {/* Metrics Grid */}
      {hasMetrics && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {metricCards.map((card, index) => (
            <MetricCard key={card.label || `metric-${index + 1}`} {...card} />
          ))}
        </div>
      )}

      {/* Collections Breakdown */}
      {collections.length > 0 && (
        <div className="ko-panel p-4" data-testid={selectors.metrics.collections}>
          <div className="flex items-center gap-2 mb-4">
            <Database className="h-5 w-5 ko-icon" />
            <h4 className="font-semibold ko-text-strong">Collections</h4>
          </div>
          <div className="ko-stack-sm">
            {collections.map((collection, index) => {
              const metrics = Array.isArray(collection.metrics) ? collection.metrics : [];
              const name =
                typeof collection.name === "string" && collection.name.trim().length > 0
                  ? collection.name
                  : `Collection ${index + 1}`;
              const sizeLabel =
                typeof collection.sizeLabel === "string" && collection.sizeLabel.trim().length > 0
                  ? collection.sizeLabel
                  : "Vectors: unknown";

              return (
                <div key={name} className="ko-card p-3">
                  <div className="flex items-center justify-between mb-2">
                    <span className="ko-text-sm font-semibold ko-text-primary">{name}</span>
                    <span className="ko-text-xs ko-subtle">{sizeLabel}</span>
                  </div>
                  {metrics.length > 0 && (
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 ko-text-xs">
                      {metrics.map((metric, metricIndex) => (
                        <div key={`${name}-${metric.label || metricIndex}`}>
                          <span className="ko-subtle">{metric.label ?? "Metric"}:</span>
                          <span className="ko-text-strong ml-1">
                            {metric.percentageLabel ?? "N/A"}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Summary Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 ko-text-sm" data-testid={selectors.metrics.summary}>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Total Vectors</p>
          <p className="text-xl font-bold ko-text-strong">
            {totalEntriesLabel}
          </p>
        </div>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Last Updated</p>
          <p className="ko-text-sm font-semibold ko-text-strong">
            {lastUpdated}
          </p>
        </div>
      </div>
    </div>
  );
}
