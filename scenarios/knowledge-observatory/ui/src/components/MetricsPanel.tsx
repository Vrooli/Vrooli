import { useQuery } from "@tanstack/react-query";
import { Database, TrendingUp, AlertCircle, Loader2, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";
import { fetchKnowledgeHealth, type QualityMetrics } from "../lib/api";

function MetricCard({ label, value, description }: { label: string; value: number; description: string }) {
  const percentage = (value * 100).toFixed(1);
  const isGood = label === "Redundancy" ? value < 0.2 : value >= 0.6;
  const isMedium = label === "Redundancy" ? value < 0.4 : value >= 0.4;

  const color = isGood ? "text-green-400" : isMedium ? "text-yellow-400" : "text-red-400";
  const bgColor = isGood ? "bg-green-900/20" : isMedium ? "bg-yellow-900/20" : "bg-red-900/20";
  const borderColor = isGood ? "border-green-700" : isMedium ? "border-yellow-700" : "border-red-700";

  return (
    <div className={`p-4 border ${borderColor} ${bgColor} rounded`}>
      <div className="flex items-start justify-between mb-2">
        <span className="text-xs text-green-600 uppercase tracking-wider">{label}</span>
        <TrendingUp className={`h-4 w-4 ${color}`} />
      </div>
      <div className={`text-3xl font-bold ${color}`}>{percentage}%</div>
      <p className="text-xs text-green-700 mt-2">{description}</p>
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
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 border border-red-900/50 bg-red-950/20 rounded flex items-start gap-2">
        <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="text-red-400 font-semibold">Failed to load metrics</p>
          <p className="text-sm text-red-600 mt-1">{(error as Error).message}</p>
          <Button
            onClick={() => refetch()}
            className="mt-3 bg-red-900 hover:bg-red-800 text-red-100 border border-red-700"
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
    <div className="space-y-6">
      {/* Overall Status */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-green-400">Overall Health</h3>
          <p className="text-sm text-green-600 capitalize">{data.overall_health} condition</p>
        </div>
        <Button
          onClick={() => refetch()}
          variant="outline"
          size="sm"
          className="border-green-900/50 text-green-400 hover:bg-green-950/30"
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Refresh
        </Button>
      </div>

      {!metrics && (
        <div className="p-4 border border-green-900/50 bg-green-950/10 rounded text-sm text-green-600">
          Quality metrics are unavailable (not computed yet). Vector counts below are live when Qdrant is reachable.
        </div>
      )}

      {/* Metrics Grid */}
      {metrics && (
        <div className="grid grid-cols-2 gap-4">
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
        <div className="border border-green-900/50 bg-green-950/10 rounded p-4">
          <div className="flex items-center gap-2 mb-4">
            <Database className="h-5 w-5 text-green-500" />
            <h4 className="font-semibold text-green-400">Collections</h4>
          </div>
          <div className="space-y-3">
            {data.collections.map((collection) => (
              <div key={collection.name} className="border border-green-900/30 bg-black/40 rounded p-3">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-semibold text-green-300">{collection.name}</span>
                  <span className="text-xs text-green-600">
                    {typeof collection.size === "number" ? `${collection.size} vectors` : "Vectors: unknown"}
                  </span>
                </div>
                {collection.metrics && (
                  <div className="grid grid-cols-4 gap-2 text-xs">
                    {typeof collection.metrics.coherence === "number" && (
                      <div>
                        <span className="text-green-700">Coherence:</span>
                        <span className="text-green-400 ml-1">{(collection.metrics.coherence * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.freshness === "number" && (
                      <div>
                        <span className="text-green-700">Freshness:</span>
                        <span className="text-green-400 ml-1">{(collection.metrics.freshness * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.coverage === "number" && (
                      <div>
                        <span className="text-green-700">Coverage:</span>
                        <span className="text-green-400 ml-1">{(collection.metrics.coverage * 100).toFixed(0)}%</span>
                      </div>
                    )}
                    {typeof collection.metrics.redundancy === "number" && (
                      <div>
                        <span className="text-green-700">Redundancy:</span>
                        <span className="text-green-400 ml-1">{(collection.metrics.redundancy * 100).toFixed(0)}%</span>
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
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div className="border border-green-900/50 bg-black/40 p-3 rounded">
          <p className="text-xs text-green-600 uppercase tracking-wider mb-1">Total Vectors</p>
          <p className="text-xl font-bold text-green-400">
            {typeof data.total_entries === "number" ? data.total_entries.toLocaleString() : "Unknown"}
          </p>
        </div>
        <div className="border border-green-900/50 bg-black/40 p-3 rounded">
          <p className="text-xs text-green-600 uppercase tracking-wider mb-1">Last Updated</p>
          <p className="text-sm font-semibold text-green-400">
            {new Date(data.timestamp).toLocaleTimeString()}
          </p>
        </div>
      </div>
    </div>
  );
}
