import { createClient } from "@connectrpc/connect";
import {
  ScoreService,
  type ActionPhase,
  type CollectorDegradation,
  type CompositeScore,
  type DimensionCount,
  type FreshnessBlock,
  type GetScoreResponse,
  type GetScoreTrendResponse,
  type ImportanceComponents,
  type ImportanceSignals,
  type ImportanceSummary,
  type ListScoresResponse,
  type MaturityHeadline,
  type MetricLine,
  type PhaseFreshness,
  type Recommendation,
  type ScoreGroup,
  ScoreSortBy,
  type ScoreRow,
  type ScoreSnapshot,
  SortOrder,
  type TrendSummary,
} from "@vrooli/proto-types/scenario-completeness-scoring/v1/scoring/scoring_pb";

import { transport } from "./client";

export const scoringClient = createClient(ScoreService, transport);

/**
 * fetchScore retrieves the full cached status payload for one scenario:
 * maturity rung "as of digest", 0-100 composite, per-phase freshness
 * verdicts, recommendations, and the action plan. Unknown scenarios reject
 * with a ConnectError carrying Code.NotFound.
 */
export async function fetchScore(scenario: string): Promise<GetScoreResponse> {
  return scoringClient.getScore({ scenario });
}

export async function fetchScoreTrend(
  scenario: string,
  limit = 12,
): Promise<GetScoreTrendResponse> {
  return scoringClient.getScoreTrend({ scenario, limit });
}

export async function fetchScores(options: {
  pageToken?: string;
  pageSize?: number;
} = {}): Promise<ListScoresResponse> {
  return scoringClient.listScores({
    sortBy: ScoreSortBy.PRIORITY,
    order: SortOrder.DESC,
    pageSize: options.pageSize ?? 10,
    pageToken: options.pageToken ?? "",
  });
}

export type {
  ActionPhase,
  CollectorDegradation,
  CompositeScore,
  DimensionCount,
  FreshnessBlock,
  GetScoreResponse,
  GetScoreTrendResponse,
  ImportanceComponents,
  ImportanceSignals,
  ImportanceSummary,
  ListScoresResponse,
  MaturityHeadline,
  MetricLine,
  PhaseFreshness,
  Recommendation,
  ScoreGroup,
  ScoreRow,
  ScoreSnapshot,
  TrendSummary,
};
