// Types shared outside the Usage feature: run status counts and the model price comparison.

export interface RunStatusCounts {
  pending: number;
  running: number;
  complete: number;
  failed: number;
  cancelled: number;
  needsReview: number;
  total: number;
}

// =============================================================================
// Model Cost Comparison Types
// =============================================================================

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
