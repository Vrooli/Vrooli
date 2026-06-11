/**
 * Mock builders for `api/scoring` — the UI ↔ API scoring boundary.
 * Co-located with the scoring feature; deleting `features/scoring/`
 * takes these mocks with it.
 *
 * See `test-utils/mocks/api.ts` for the full builder/hoisting rationale;
 * the same pattern applies. Canonical usage:
 *
 *   import { makeScoringMocks } from "./mocks/scoring";
 *
 *   vi.mock("../../api/scoring", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/scoring")>();
 *     return { ...actual, ...makeScoringMocks() };
 *   });
 *
 * The `...actual` spread keeps the re-exported proto types intact — only
 * the network-touching functions are substituted.
 *
 * Default behavior: score, trend, and fleet calls resolve to representative
 * persisted snapshot data.
 */
import { vi } from "vitest";

import { makeGetScoreResponse, makeGetScoreTrendResponse, makeListScoresResponse } from "./factories";

export interface ScoringMocks {
  fetchScore: ReturnType<typeof vi.fn>;
  fetchScoreTrend: ReturnType<typeof vi.fn>;
  fetchScores: ReturnType<typeof vi.fn>;
  scoringClient: {
    getScore: ReturnType<typeof vi.fn>;
    getScoreTrend: ReturnType<typeof vi.fn>;
    listScores: ReturnType<typeof vi.fn>;
  };
}

export const makeScoringMocks = (): ScoringMocks => ({
  fetchScore: vi.fn().mockResolvedValue(makeGetScoreResponse()),
  fetchScoreTrend: vi.fn().mockResolvedValue(makeGetScoreTrendResponse()),
  fetchScores: vi.fn().mockResolvedValue(makeListScoresResponse()),
  scoringClient: {
    getScore: vi.fn().mockResolvedValue(makeGetScoreResponse()),
    getScoreTrend: vi.fn().mockResolvedValue(makeGetScoreTrendResponse()),
    listScores: vi.fn().mockResolvedValue(makeListScoresResponse()),
  },
});
