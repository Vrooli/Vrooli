import { useQuery } from "@tanstack/react-query";
import { Database, TrendingUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";
import { fetchKnowledgeHealth, type QualityMetrics } from "../lib/api";
import { selectors } from "../consts/selectors";

function MetricCard({ label, value, description }: { label: string; value: number; description: string }) {
  const percentage = (value * 100).toFixed(1);
  const isGood = label === "Redundancy" ? value < 0.2 : value >= 0.6;
  const isMedium = label === "Redundancy" ? value < 0.4 : value >= 0.4;

  const color = isGood ? "text-green-300" : isMedium ? "text-yellow-300" : "text-red-300";
  const bgColor = isGood ? "bg-green-900/25" : isMedium ? "bg-yellow-900/20" : "bg-red-900/20";
  const borderColor = isGood ? "border-green-600/70" : isMedium ? "border-yellow-600/70" : "border-red-600/70";

  return (
    <div className={`ko-metric-card ${borderColor} ${bgColor}`}>
      <div className="flex items-start justify-between mb-2">
        <span className="ko-meta">{label}</span>
        <TrendingUp className={`h-4 w-4 ${color}`} />
      </div>
      <div className={`text-3xl font-bold ${color}`}>{percentage}%</div>
      <p className="ko-text-xs ko-subtle mt-2">{description}</p>
    </div>
  );
}

export function MetricsPanel() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["knowledgeHealth"],
    queryFn: fetchKnowledgeHealth,
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-8 w-8 text-green-500 animate-spin" />
        <span className="ml-3 ko-text-sm ko-muted">Loading metrics...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="ko-alert ko-alert-danger">
        <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="text-red-300 ko-alert-title">Failed to load metrics</p>
          <p className="ko-text-sm text-red-600 mt-1">{(error as Error).message}</p>
          <Button
            onClick={() => refetch()}
            className="mt-3 ko-button-danger"
          >
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const metrics = data.overall_metrics;

  return (
    <div className="ko-stack">
      {/* Overall Status */}
      <div className="flex items-center justify-between" data-testid={selectors.metrics.overall}>
        <div>
          <h3 className="ko-text-lg font-semibold text-green-300">Overall Health</h3>
          <p className="ko-text-sm ko-muted capitalize">{data.overall_health} condition</p>
        </div>
        <Button
          onClick={() => refetch()}
          variant="outline"
          size="sm"
          className="ko-button-outline"
          data-testid={selectors.metrics.refresh}
        >
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

      {!metrics && (
        <div className="ko-panel p-4 ko-text-sm ko-muted">
          Quality metrics are unavailable (not computed yet). Vector counts below are live when Qdrant is reachable.
        </div>
      )}

      {/* Metrics Grid */}
      {metrics && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {typeof metrics.coherence === "number" && (
            <MetricCard label="Coherence" value={metrics.coherence} description="Topical consistency across knowledge" />
          )}
          {typeof metrics.freshness === "number" && (
            <MetricCard label="Freshness" value={metrics.freshness} description="Recency of knowledge entries" />
          )}
          {typeof metrics.coverage === "number" && (
            <MetricCard label="Coverage" value={metrics.coverage} description="Domain topic distribution" />
          )}
          {typeof metrics.redundancy === "number" && (
            <MetricCard label="Redundancy" value={metrics.redundancy} description="Duplicate detection (lower is better)" />
          )}
        </div>
      )}

      {/* Collections Breakdown */}
      {data.collections && data.collections.length > 0 && (
        <div
          className="ko-panel p-4"
          data-testid={selectors.metrics.collections}
        >
          <div className="flex items-center gap-2 mb-4">
            <Database className="h-5 w-5 text-green-500" />
            <h4 className="font-semibold text-green-300">Collections</h4>
          </div>
          <div className="ko-stack-sm">
            {data.collections.map((collection) => (
              <div key={collection.name} className="ko-card p-3 border-green-800/50">
                <div className="flex items-center justify-between mb-2">
                  <span className="ko-text-sm font-semibold text-green-200">{collection.name}</span>
                  <span className="ko-text-xs ko-subtle">
                    {typeof collection.size === "number" ? `${collection.size} vectors` : "Vectors: unknown"}
                  </span>
                </div>
                {collection.metrics && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-2 ko-text-xs">
                    {typeof collection.metrics.coherence === "number" && (
                      <div>
                        <span className="ko-subtle">Coherence:</span>
                        <span className="text-green-300 ml-1">{(collection.metrics.coherence * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.freshness === "number" && (
                      <div>
                        <span className="ko-subtle">Freshness:</span>
                        <span className="text-green-300 ml-1">{(collection.metrics.freshness * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.coverage === "number" && (
                      <div>
                        <span className="ko-subtle">Coverage:</span>
                        <span className="text-green-300 ml-1">{(collection.metrics.coverage * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.redundancy === "number" && (
                      <div>
                        <span className="ko-subtle">Redundancy:</span>
                        <span className="text-green-300 ml-1">{(collection.metrics.redundancy * 100).toFixed(0)}%</span>
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Summary Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 ko-text-sm" data-testid={selectors.metrics.summary}>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Total Vectors</p>
          <p className="text-xl font-bold text-green-300">
            {typeof data.total_entries === "number" ? data.total_entries.toLocaleString() : "Unknown"}
          </p>
        </div>
        <div className="ko-card p-3">
          <p className="ko-meta mb-1">Last Updated</p>
          <p className="ko-text-sm font-semibold text-green-300">
            {new Date(data.timestamp).toLocaleTimeString()}
          </p>
        </div>
      </div>
    </div>
  );
}
