// React Query hook for fetching model cost comparisons

import { useQuery } from "@tanstack/react-query";
import { fetchModelCostComparison, statsQueryKeys } from "../api/statsClient";
import type { CompareModelsRequest, CompareModelsResponse } from "../api/types";

export interface UseModelCostComparisonOptions {
  request: CompareModelsRequest | null;
  enabled?: boolean;
}

export function useModelCostComparison(options: UseModelCostComparisonOptions) {
  const { request, enabled = true } = options;

  return useQuery<CompareModelsResponse, Error>({
    queryKey: request
      ? statsQueryKeys.modelCostComparison(request)
      : ["pricing", "compare", "disabled"],
    queryFn: () => fetchModelCostComparison(request!),
    enabled:
      enabled &&
      request !== null &&
      request.inputTokens + request.outputTokens > 0,
    staleTime: 60_000, // Cache for 1 minute
  });
}
