// The model price comparison the run Usage tab shows: one REST read and its query hook.

import { useQuery } from "@tanstack/react-query";
import { fetchWithDeadline } from "../../lib/api";

export interface ModelCostComparison {
  model: string;
  canonicalModel?: string;
  estimatedCostUsd: number;
  differenceUsd: number;
  differencePercent: number;
  isActualModel: boolean;
  priceSource?: string;
}

export interface CompareModelsRequest {
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheCreationTokens: number;
  webSearchRequests: number;
  serverToolUseCount: number;
  actualCostUsd: number;
  actualModel: string;
  modelList: "popular" | "recent";
}

export interface CompareModelsResponse {
  comparisons: ModelCostComparison[];
}

async function fetchModelCostComparison(request: CompareModelsRequest): Promise<CompareModelsResponse> {
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

export function useModelCostComparison({ request, enabled = true }: { request: CompareModelsRequest | null; enabled?: boolean }) {
  return useQuery<CompareModelsResponse, Error>({
    queryKey: request
      ? ["pricing", "compare", request.modelList, request.actualModel, request.inputTokens, request.outputTokens, request.cacheReadTokens, request.cacheCreationTokens]
      : ["pricing", "compare", "disabled"],
    queryFn: () => {
      if (!request) throw new Error("request is required");
      return fetchModelCostComparison(request);
    },
    enabled: enabled && request !== null && request.inputTokens + request.outputTokens > 0,
    staleTime: 60_000,
  });
}
