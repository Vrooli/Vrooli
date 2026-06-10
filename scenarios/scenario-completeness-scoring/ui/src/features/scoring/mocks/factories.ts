/**
 * Test data factories for the scoring domain. Co-located with the feature
 * so the factories travel with `features/scoring/` (no central residue).
 *
 * Each `make<Shape>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`. Instances are
 * built with `create(<Schema>, …)` so proto3 defaults and reflection state
 * match what the generated Connect client returns at runtime.
 *
 * `makeGetScoreResponse()` defaults to a "healthy but improvable" payload:
 * working rung R1, score 82, one stale phase with a refresh command, one
 * recommendation, one action phase, and no degradations — enough surface
 * that a single default render exercises every dashboard section.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  ActionPhaseSchema,
  CollectorDegradationSchema,
  CompositeScoreSchema,
  DimensionCountSchema,
  FreshnessBlockSchema,
  GetScoreResponseSchema,
  MaturityHeadlineSchema,
  MetricLineSchema,
  PhaseFreshnessSchema,
  RecommendationSchema,
  ScoreGroupSchema,
  type ActionPhase,
  type CollectorDegradation,
  type CompositeScore,
  type FreshnessBlock,
  type GetScoreResponse,
  type MaturityHeadline,
  type PhaseFreshness,
  type Recommendation,
} from "@vrooli/proto-types/scenario-completeness-scoring/v1/scoring/scoring_pb";

export type {
  ActionPhase,
  CollectorDegradation,
  CompositeScore,
  FreshnessBlock,
  GetScoreResponse,
  MaturityHeadline,
  PhaseFreshness,
  Recommendation,
};

export const makeMaturityHeadline = (
  overrides: MessageInitShape<typeof MaturityHeadlineSchema> = {},
): MaturityHeadline =>
  create(MaturityHeadlineSchema, {
    workingRung: "R1 Safe & standards-clean",
    ladderClean: false,
    satisfiedThrough: "R0 Runnable & green",
    buildPassing: true,
    dimensions: [
      create(DimensionCountSchema, { dimension: "standards", errorPlus: 2, total: 5 }),
      create(DimensionCountSchema, { dimension: "tests", errorPlus: 0, total: 1, approximate: true }),
    ],
    ...overrides,
  });

export const makeCompositeScore = (
  overrides: MessageInitShape<typeof CompositeScoreSchema> = {},
): CompositeScore =>
  create(CompositeScoreSchema, {
    score: 82,
    classification: "mostly_complete",
    classificationLabel: "Mostly complete — a few gaps left before launch readiness.",
    groups: [
      create(ScoreGroupSchema, {
        id: "quality",
        label: "Quality",
        score: 41,
        max: 50,
        metrics: [
          create(MetricLineSchema, {
            id: "requirement_pass_rate",
            label: "Requirement pass rate",
            observed: "34 total, 30 passing (88%)",
            points: 26,
            maxPoints: 30,
            threshold: "good",
          }),
        ],
      }),
    ],
    ...overrides,
  });

export const makePhaseFreshness = (
  overrides: MessageInitShape<typeof PhaseFreshnessSchema> = {},
): PhaseFreshness =>
  create(PhaseFreshnessSchema, {
    phase: "unit",
    verdict: "fresh",
    lastRunId: "run-20260610-1200",
    lastRunAt: timestampFromDate(new Date("2026-06-10T12:00:00.000Z")),
    lastDigest: "td:abc123",
    lastStatus: "passed",
    ...overrides,
  });

export const makeFreshnessBlock = (
  overrides: MessageInitShape<typeof FreshnessBlockSchema> = {},
): FreshnessBlock =>
  create(FreshnessBlockSchema, {
    currentDigest: "td:abc123",
    phases: [
      makePhaseFreshness(),
      makePhaseFreshness({ phase: "smoke", verdict: "stale", lastDigest: "td:older" }),
    ],
    suggestedCommand: "vrooli scenario test web-search --phases smoke",
    ...overrides,
  });

export const makeRecommendation = (
  overrides: MessageInitShape<typeof RecommendationSchema> = {},
): Recommendation =>
  create(RecommendationSchema, {
    priority: "high",
    description: "Fix the 2 standards errors blocking R1.",
    impactPoints: 6,
    ...overrides,
  });

export const makeActionPhase = (
  overrides: MessageInitShape<typeof ActionPhaseSchema> = {},
): ActionPhase =>
  create(ActionPhaseSchema, {
    title: "Stabilize standards",
    actions: ["Run the standards phase", "Fix reported violations"],
    estimatedPoints: 6,
    ...overrides,
  });

export const makeCollectorDegradation = (
  overrides: MessageInitShape<typeof CollectorDegradationSchema> = {},
): CollectorDegradation =>
  create(CollectorDegradationSchema, {
    collector: "ui",
    state: "failed",
    reason: "ui sources unreadable",
    ...overrides,
  });

export const makeGetScoreResponse = (
  overrides: MessageInitShape<typeof GetScoreResponseSchema> = {},
): GetScoreResponse =>
  create(GetScoreResponseSchema, {
    scenario: "web-search",
    category: "utility",
    maturity: makeMaturityHeadline(),
    composite: makeCompositeScore(),
    freshness: makeFreshnessBlock(),
    recommendations: [makeRecommendation()],
    actionPlan: [makeActionPhase()],
    degradations: [],
    calculatedAt: timestampFromDate(new Date("2026-06-10T12:34:56.000Z")),
    ...overrides,
  });
