import { useQuery } from "@tanstack/react-query";
import { defaultQueryOptions } from "../../lib";
import { defaultApiClient } from "../../lib/api-client";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";

interface VariantStats {
  variantId: string;
  totalRuns: number;
  readyCount: number;
  needsWorkCount: number;
  fixupRate: number;
  avgDurationSecs?: number;
}

interface ExperimentResultsData {
  experimentId: string;
  skillId?: string;
  status?: string;
  variants: VariantStats[];
  totalOutcomes: number;
  analyzedAt: string;
}

async function fetchExperimentResults(experimentId: string): Promise<ExperimentResultsData> {
  return defaultApiClient.get<ExperimentResultsData>(
    `/prompts/experiments/${encodeURIComponent(experimentId)}/results`
  );
}

function formatRate(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`;
}

function formatDuration(secs?: number): string {
  if (!secs || secs <= 0) return "-";
  if (secs < 60) return `${secs.toFixed(1)}s`;
  return `${(secs / 60).toFixed(1)}m`;
}

export function ExperimentResults({ experimentId }: { experimentId: string }) {
  const query = useQuery({
    queryKey: ["prompts", "experiment-results", experimentId],
    queryFn: () => fetchExperimentResults(experimentId),
    enabled: experimentId.length > 0,
    ...defaultQueryOptions,
  });

  if (query.isLoading) {
    return <PageLoadingState variant="settings" label="Loading experiment results..." />;
  }

  if (query.error) {
    return (
      <ErrorState
        title="Unable to load experiment results"
        message={query.error instanceof Error ? query.error.message : "Failed to load results."}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const data = query.data;
  if (!data) return null;

  const bestVariant =
    data.variants.length > 0
      ? data.variants.reduce((best, v) =>
          v.readyCount / Math.max(v.totalRuns, 1) > best.readyCount / Math.max(best.totalRuns, 1)
            ? v
            : best
        )
      : null;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Experiment Results</h3>
        {data.status && (
          <span
            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
              data.status === "running"
                ? "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
                : data.status === "concluded"
                  ? "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                  : "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200"
            }`}
          >
            {data.status}
          </span>
        )}
      </div>

      <div className="text-sm text-muted-foreground">
        <span>Experiment: {data.experimentId}</span>
        {data.skillId && <span className="ml-4">Skill: {data.skillId}</span>}
        <span className="ml-4">Total outcomes: {data.totalOutcomes}</span>
      </div>

      {data.variants.length === 0 ? (
        <p className="text-sm text-muted-foreground italic">No outcomes recorded yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th className="pb-2 pr-4 font-medium">Variant</th>
                <th className="pb-2 pr-4 font-medium text-right">Runs</th>
                <th className="pb-2 pr-4 font-medium text-right">Ready</th>
                <th className="pb-2 pr-4 font-medium text-right">Needs Work</th>
                <th className="pb-2 pr-4 font-medium text-right">Fixup Rate</th>
                <th className="pb-2 font-medium text-right">Avg Duration</th>
              </tr>
            </thead>
            <tbody>
              {data.variants.map((v) => {
                const isWinner = bestVariant?.variantId === v.variantId && data.variants.length > 1;
                return (
                  <tr
                    key={v.variantId}
                    className={`border-b last:border-0 ${isWinner ? "bg-green-50 dark:bg-green-900/20" : ""}`}
                  >
                    <td className="py-2 pr-4 font-medium">
                      {v.variantId || "(control)"}
                      {isWinner && (
                        <span className="ml-2 text-xs text-green-600 dark:text-green-400">
                          best
                        </span>
                      )}
                    </td>
                    <td className="py-2 pr-4 text-right">{v.totalRuns}</td>
                    <td className="py-2 pr-4 text-right">{v.readyCount}</td>
                    <td className="py-2 pr-4 text-right">{v.needsWorkCount}</td>
                    <td className="py-2 pr-4 text-right">{formatRate(v.fixupRate)}</td>
                    <td className="py-2 text-right">{formatDuration(v.avgDurationSecs)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <p className="text-xs text-muted-foreground">
        Analyzed at {new Date(data.analyzedAt).toLocaleString()}
      </p>
    </div>
  );
}
