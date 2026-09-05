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
  GetScoreTrendResponseSchema,
  ImportanceComponentsSchema,
  ImportanceSignalsSchema,
  ImportanceSummarySchema,
  ListScoresResponseSchema,
  MaturityHeadlineSchema,
  MetricLineSchema,
  PhaseFreshnessSchema,
  RecommendationSchema,
  ScoreGroupSchema,
  ScoreRowSchema,
  ScoreSnapshotSchema,
  TrendSummarySchema,
  type ActionPhase,
  type CollectorDegradation,
  type CompositeScore,
  type FreshnessBlock,
  type GetScoreResponse,
  type GetScoreTrendResponse,
  type ImportanceSummary,
  type ListScoresResponse,
  type MaturityHeadline,
  type PhaseFreshness,
  type Recommendation,
  type ScoreRow,
  type ScoreSnapshot,
  type TrendSummary,
} from "@vrooli/proto-types/scenario-completeness-scoring/v1/scoring/scoring_pb";

export type {
  ActionPhase,
  CollectorDegradation,
  CompositeScore,
  FreshnessBlock,
  GetScoreResponse,
  GetScoreTrendResponse,
  ImportanceSummary,
  ListScoresResponse,
  MaturityHeadline,
  PhaseFreshness,
  Recommendation,
  ScoreRow,
  ScoreSnapshot,
  TrendSummary,
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

export const makeImportanceSummary = (
  overrides: MessageInitShape<typeof ImportanceSummarySchema> = {},
): ImportanceSummary =>
  create(ImportanceSummarySchema, {
    score: 0.82,
    systemRequired: true,
    components: create(ImportanceComponentsSchema, {
      centrality: 0.8,
      coreProximity: 0.5,
      recency: 0.4,
    }),
    signals: create(ImportanceSignalsSchema, {
      directReverseDependencyCount: 2,
      transitiveReverseDependencyCount: 5,
      requiredReverseDependencyCount: 3,
      requiredEdgeWeightedScore: 8,
      distanceToCoreSeed: 1,
      nearestCoreSeed: "test-genie",
      recentActivityCount: 2,
    }),
    degraded: ["recency:not_configured"],
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

export const makeTrendSummary = (
  overrides: MessageInitShape<typeof TrendSummarySchema> = {},
): TrendSummary =>
  create(TrendSummarySchema, {
    previousScore: 75,
    previousCalculatedAt: timestampFromDate(new Date("2026-06-08T12:00:00.000Z")),
    delta: 7,
    ...overrides,
  });

export const makeScoreSnapshot = (
  overrides: MessageInitShape<typeof ScoreSnapshotSchema> = {},
): ScoreSnapshot =>
  create(ScoreSnapshotSchema, {
    scenario: "web-search",
    category: "utility",
    digest: "td:abc123",
    score: 82,
    classification: "mostly_complete",
    workingRung: "R1 Safe & standards-clean",
    source: "sweep",
    calculatedAt: timestampFromDate(new Date("2026-06-10T12:34:56.000Z")),
    ...overrides,
  });

export const makeGetScoreTrendResponse = (
  overrides: MessageInitShape<typeof GetScoreTrendResponseSchema> = {},
): GetScoreTrendResponse =>
  create(GetScoreTrendResponseSchema, {
    scenario: "web-search",
    snapshots: [
      makeScoreSnapshot(),
      makeScoreSnapshot({
        digest: "td:older",
        score: 75,
        calculatedAt: timestampFromDate(new Date("2026-06-08T12:00:00.000Z")),
      }),
    ],
    ...overrides,
  });

export const makeScoreRow = (
  overrides: MessageInitShape<typeof ScoreRowSchema> = {},
): ScoreRow =>
  create(ScoreRowSchema, {
    scenario: "web-search",
    category: "utility",
    score: 82,
    classification: "mostly_complete",
    workingRung: "R1 Safe & standards-clean",
    importance: 0.8,
    priority: 2.4,
    calculatedAt: timestampFromDate(new Date("2026-06-10T12:34:56.000Z")),
    digest: "td:abc123",
    ...overrides,
  });

export const makeListScoresResponse = (
  overrides: MessageInitShape<typeof ListScoresResponseSchema> = {},
): ListScoresResponse =>
  create(ListScoresResponseSchema, {
    scores: [
      makeScoreRow(),
      makeScoreRow({
        scenario: "cli-health",
        category: "platform",
        score: 68,
        workingRung: "R2 Validated",
        priority: 1.2,
      }),
    ],
    nextPageToken: "2",
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
    importance: makeImportanceSummary(),
    recommendations: [makeRecommendation()],
    actionPlan: [makeActionPhase()],
    degradations: [],
    calculatedAt: timestampFromDate(new Date("2026-06-10T12:34:56.000Z")),
    trend: makeTrendSummary(),
    ...overrides,
  });
