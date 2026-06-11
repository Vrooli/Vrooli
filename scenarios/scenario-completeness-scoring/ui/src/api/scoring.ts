import { createClient } from "@connectrpc/connect";
import {
  ScoreService,
  type ActionPhase,
  type CollectorDegradation,
  type CompositeScore,
  type DimensionCount,
  type FreshnessBlock,
  type GetScoreResponse,
  type ImportanceComponents,
  type ImportanceSignals,
  type ImportanceSummary,
  type MaturityHeadline,
  type MetricLine,
  type PhaseFreshness,
  type Recommendation,
  type ScoreGroup,
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

export type {
  ActionPhase,
  CollectorDegradation,
  CompositeScore,
  DimensionCount,
  FreshnessBlock,
  GetScoreResponse,
  ImportanceComponents,
  ImportanceSignals,
  ImportanceSummary,
  MaturityHeadline,
  MetricLine,
  PhaseFreshness,
  Recommendation,
  ScoreGroup,
};
