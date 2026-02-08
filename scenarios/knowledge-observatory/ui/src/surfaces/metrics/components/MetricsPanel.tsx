// DOC: docs/concepts/ARCHITECTURE.md#health-and-metrics-flow
import { Database, TrendingUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { cn } from "../../../shared/lib/utils";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { MetricCardView, MetricsViewModel } from "../../../shared/controllers/knowledgeController";
import type { CollectionDiagnostics, IngestHealthResponse } from "../../../shared/services/api";

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
  ingestHealth?: IngestHealthResponse | null;
  selectedCollection: string;
  diagnostics?: CollectionDiagnostics | null;
  diagnosticsError: string;
  diagnosticsLoading: boolean;
  maintenanceActionLabel: string;
  onSelectCollection: (name: string) => void;
  onRunPruneStale: (collection: string) => void;
  onRunDedupe: (collection: string) => void;
  onRetry: () => void;
};

export function MetricsPanel({
  isLoading,
  hasError,
  errorMessage,
  hasData,
  viewModel,
  ingestHealth,
  selectedCollection,
  diagnostics,
  diagnosticsError,
  diagnosticsLoading,
  maintenanceActionLabel,
  onSelectCollection,
  onRunPruneStale,
  onRunDedupe,
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
  const ingestStatus = ingestHealth?.status?.trim() || "unknown";

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

      <div className="ko-panel p-4">
        <h4 className="font-semibold ko-text-strong mb-2">Ingest Pipeline</h4>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 ko-text-xs">
          <div><span className="ko-subtle">Status:</span><span className="ko-text-strong ml-1 capitalize">{ingestStatus}</span></div>
          <div><span className="ko-subtle">Pending:</span><span className="ko-text-strong ml-1">{ingestHealth?.pending_jobs ?? 0}</span></div>
          <div><span className="ko-subtle">Running:</span><span className="ko-text-strong ml-1">{ingestHealth?.running_jobs ?? 0}</span></div>
          <div><span className="ko-subtle">Failures (24h):</span><span className="ko-text-strong ml-1">{ingestHealth?.failures_last_24h ?? 0}</span></div>
        </div>
        <p className="ko-text-xs ko-subtle mt-2">
          Runner interval: {ingestHealth?.runner_interval_ms ?? 500}ms
          {typeof ingestHealth?.oldest_pending_age_ms === "number"
            ? ` · Oldest pending age: ${Math.round(ingestHealth.oldest_pending_age_ms / 1000)}s`
            : ""}
        </p>
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
                  <div className="flex flex-wrap gap-2 mb-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => onSelectCollection(name)}
                    >
                      {selectedCollection === name ? "Refresh Diagnostics" : "Inspect"}
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => onRunPruneStale(name)}
                      disabled={maintenanceActionLabel !== ""}
                    >
                      Prune Stale
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      onClick={() => onRunDedupe(name)}
                      disabled={maintenanceActionLabel !== ""}
                    >
                      Dedupe
                    </Button>
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
                  {selectedCollection === name && (
                    <div className="mt-3 p-3 ko-surface-strong rounded ko-text-xs">
                      {diagnosticsLoading && <p className="ko-subtle">Loading diagnostics…</p>}
                      {!diagnosticsLoading && diagnosticsError && <p className="ko-text-danger-muted">{diagnosticsError}</p>}
                      {!diagnosticsLoading && !diagnosticsError && diagnostics && (
                        <div className="ko-stack-xs">
                          <div>
                            <span className="ko-subtle">Analyzed:</span>
                            <span className="ko-text-strong ml-1">
                              {diagnostics.analyzed_points}
                              {typeof diagnostics.total_points === "number" ? ` / ${diagnostics.total_points}` : ""}
                            </span>
                          </div>
                          <div>
                            <span className="ko-subtle">Stale delete candidates:</span>
                            <span className="ko-text-strong ml-1">{diagnostics.stale_chunks.candidate_delete_rows}</span>
                          </div>
                          <div>
                            <span className="ko-subtle">Duplicate ratio:</span>
                            <span className="ko-text-strong ml-1">{(diagnostics.redundancy.duplicate_ratio * 100).toFixed(1)}%</span>
                          </div>
                          <div>
                            <span className="ko-subtle">Chunk length:</span>
                            <span className="ko-text-strong ml-1">
                              {Math.round(diagnostics.chunk_length.avg_characters)} avg
                              {" · "}
                              {diagnostics.chunk_length.min_characters}-{diagnostics.chunk_length.max_characters}
                            </span>
                          </div>
                          {Array.isArray(diagnostics.recommendations) && diagnostics.recommendations.length > 0 && (
                            <div>
                              <p className="ko-subtle mb-1">Recommendations</p>
                              <ul className="list-disc list-inside ko-text-primary">
                                {diagnostics.recommendations.slice(0, 3).map((entry, recIndex) => (
                                  <li key={`${name}-rec-${recIndex}`}>{entry}</li>
                                ))}
                              </ul>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {maintenanceActionLabel && (
        <div className="ko-panel p-3 ko-text-xs ko-subtle">
          {maintenanceActionLabel}
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
