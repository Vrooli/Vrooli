// Typed reads shared outside the Usage feature: the run status distribution and the model price comparison.

import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import type { TimeWindow } from "@vrooli/proto-types/measures/v1/measures_pb";
import { fetchWithDeadline } from "../../../lib/api";
import { measuresClient } from "./measuresClient";
import type { CompareModelsRequest, CompareModelsResponse, RunStatusCounts } from "./types";

export async function fetchModelCostComparison(request: CompareModelsRequest): Promise<CompareModelsResponse> {
  const response = await fetchWithDeadline("/api/v1/pricing/compare", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API error ${response.status}: ${errorText}`);
  }
  return response.json() as Promise<CompareModelsResponse>;
}

export const statsQueryKeys = {
  modelCostComparison: (request: CompareModelsRequest) =>
    ["pricing", "compare", request.modelList, request.actualModel, request.inputTokens, request.outputTokens, request.cacheReadTokens, request.cacheCreationTokens] as const,
} as const;

export interface DurableRunStatusDistribution {
  rows: Array<{ status: string; count: number }>;
}

const PRESET_HOURS = { "6h": 6, "12h": 12, "24h": 24, "7d": 168, "30d": 720 } as const;

/** Run counts per status over the trailing preset window. */
export async function fetchDurableRunStatusDistribution(filter: { preset: keyof typeof PRESET_HOURS }): Promise<DurableRunStatusDistribution> {
  const now = new Date();
  const window = { window: { case: "custom", value: { from: timestampFromDate(new Date(now.getTime() - PRESET_HOURS[filter.preset] * 3600_000)), to: timestampFromDate(now) } } } as TimeWindow;
  const response = await measuresClient.runStatusDistribution({ window });
  return { rows: response.rows.map((row) => ({ status: row.status, count: Number(row.count) })) };
}

/** Folds durable status rows into the fixed per-status counts the dashboard and summary render. */
export function runStatusCountsFromRows(rows: DurableRunStatusDistribution["rows"]): RunStatusCounts {
  const counts = Object.fromEntries(rows.map((row) => [row.status, row.count]));
  return {
    pending: counts.pending ?? 0,
    running: counts.running ?? 0,
    complete: counts.complete ?? 0,
    failed: counts.failed ?? 0,
    cancelled: counts.cancelled ?? 0,
    needsReview: counts.needs_review ?? 0,
    total: rows.reduce((total, row) => total + row.count, 0),
  };
}
