import { useState, useMemo } from "react";
import { Button } from "./ui/button";
import { EditPricingDialog } from "./dialogs/EditPricingDialog";
import { useModelCostComparison } from "../features/stats/hooks/useModelCostComparison";
import type { CompareModelsRequest } from "../features/stats/api/types";

interface CostTotals {
  inputTokens: number;
  outputTokens: number;
  cacheCreationTokens: number;
  cacheReadTokens: number;
  totalCostUsd: number;
  webSearchRequests: number;
  serverToolUseRequests: number;
  models: string[];
  serviceTiers: string[];
  events: number;
}

interface ModelCostComparisonProps {
  costTotals: CostTotals;
  actualModel: string;
}

type ModelListType = "popular" | "recent";

export function ModelCostComparison({
  costTotals,
  actualModel,
}: ModelCostComparisonProps) {
  const [modelList, setModelList] = useState<ModelListType>("popular");
  const [editingModel, setEditingModel] = useState<string | null>(null);

  const request = useMemo<CompareModelsRequest | null>(() => {
    if (costTotals.events === 0) return null;
    return {
      inputTokens: costTotals.inputTokens,
      outputTokens: costTotals.outputTokens,
      cacheReadTokens: costTotals.cacheReadTokens,
      cacheCreationTokens: costTotals.cacheCreationTokens,
      webSearchRequests: costTotals.webSearchRequests,
      serverToolUseCount: costTotals.serverToolUseRequests,
      actualCostUsd: costTotals.totalCostUsd,
      actualModel: actualModel,
      modelList: modelList,
    };
  }, [costTotals, actualModel, modelList]);

  const { data, isLoading, error, refetch } = useModelCostComparison({ request });

  if (costTotals.events === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          Compare Models
        </h4>
        <div className="flex gap-1">
          <Button
            variant={modelList === "popular" ? "default" : "outline"}
            size="sm"
            onClick={() => setModelList("popular")}
            className="h-7 text-xs"
          >
            Popular
          </Button>
          <Button
            variant={modelList === "recent" ? "default" : "outline"}
            size="sm"
            onClick={() => setModelList("recent")}
            className="h-7 text-xs"
          >
            Recent
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="h-32 animate-pulse rounded bg-muted/20" />
      ) : error ? (
        <div className="rounded border border-red-500/30 bg-red-500/5 p-3 text-xs text-red-500">
          Failed to load comparison: {error.message}
        </div>
      ) : !data?.comparisons?.length ? (
        <div className="rounded border border-border p-3 text-xs text-muted-foreground text-center">
          No model pricing data available for comparison
        </div>
      ) : (
        <div className="rounded border border-border overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-3 py-2 text-left font-medium">Model</th>
                <th className="px-3 py-2 text-right font-medium">
                  Estimated Cost
                </th>
                <th className="px-3 py-2 text-right font-medium">Difference</th>
                <th className="px-3 py-2 text-right font-medium">% Diff</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {data.comparisons.map((comparison) => (
                <tr
                  key={comparison.model}
                  className={comparison.isActualModel ? "bg-primary/5" : ""}
                >
                  <td className="px-3 py-2">
                    <button
                      type="button"
                      onClick={() => setEditingModel(comparison.canonicalModel || comparison.model)}
                      className={`text-left hover:text-primary hover:underline transition-colors ${
                        comparison.isActualModel ? "font-medium" : ""
                      }`}
                      title="Edit pricing"
                    >
                      {formatModelName(comparison.model)}
                    </button>
                    {comparison.isActualModel && (
                      <span className="ml-2 text-[10px] text-muted-foreground">
                        (actual)
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right font-mono">
                    ${comparison.estimatedCostUsd.toFixed(4)}
                  </td>
                  <td
                    className={`px-3 py-2 text-right font-mono ${getDiffColor(comparison.differenceUsd)}`}
                  >
                    {comparison.differenceUsd >= 0 ? "+" : ""}$
                    {comparison.differenceUsd.toFixed(4)}
                  </td>
                  <td
                    className={`px-3 py-2 text-right font-mono ${getDiffColor(comparison.differencePercent)}`}
                  >
                    {comparison.differencePercent >= 0 ? "+" : ""}
                    {comparison.differencePercent.toFixed(1)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <EditPricingDialog
        model={editingModel}
        onClose={() => setEditingModel(null)}
        onPricingUpdated={() => refetch()}
      />
    </div>
  );
}

function formatModelName(model: string): string {
  if (!model || model === "unknown") return "Unknown";
  // Truncate long model names for display
  if (model.length > 30) {
    return model.slice(0, 27) + "...";
  }
  return model;
}

function getDiffColor(value: number): string {
  if (value < 0) return "text-emerald-500";
  if (value > 0) return "text-red-500";
  return "text-muted-foreground";
}
