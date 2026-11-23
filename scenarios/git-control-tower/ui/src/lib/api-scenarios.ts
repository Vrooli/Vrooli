// ============================================================================
// Scenario Listing, Envelope & Review — Types + API Functions
// ============================================================================

import { create } from "@bufbuild/protobuf";
import { GetReviewJobStatusRequestSchema, GetReviewSummaryRequestSchema, StartReviewRequestSchema, type CodeQualityDimension, type ProvenanceDimension, type ReviewDimensions, type ReviewJobStatus as ReviewJobStatusMessage, type ReviewSummary as ReviewSummaryMessage, type StandardsDimension, type TestsDimension, type VisualDimension } from "@vrooli/proto-types/git-control-tower/v1/review/review_pb";
import { GetScenarioEnvelopeRequestSchema, ListScenariosRequestSchema, type ScenarioEnvelope as ScenarioEnvelopeMessage, type ScenarioInfo as ScenarioInfoMessage } from "@vrooli/proto-types/git-control-tower/v1/scenarios/scenarios_pb";
import { reviewClient, scenarioClient } from "./connect";

// ── Scenario Listing ───────────────────────────────────────────────────

export type ScenarioInfo = Pick<ScenarioInfoMessage, "name" | "displayName" | "description" | "status" | "healthStatus" | "tags" | "runtime">;

export async function fetchScenarios(signal?: AbortSignal): Promise<ScenarioInfo[]> {
  const response = await scenarioClient.listScenarios(create(ListScenariosRequestSchema), { signal });
  return response.scenarios;
}

// ── Scenario Envelope ──────────────────────────────────────────────────

/** Enriched scenario metadata derived from service.json, used to build the agent envelope. */
export type ScenarioEnvelopeData = Pick<ScenarioEnvelopeMessage, "name" | "displayName" | "description" | "path" | "tags"> & {
  dependencies: { scenarios: Record<string, string>; resources: Record<string, string> };
  lifecycle: { testCommand?: string; buildCommand?: string };
};

/** Fetch enriched scenario metadata for the agent envelope. */
export async function fetchScenarioEnvelope(slug: string, signal?: AbortSignal): Promise<ScenarioEnvelopeData> {
  const response = await scenarioClient.getScenarioEnvelope(create(GetScenarioEnvelopeRequestSchema, { slug }), { signal });
  return { ...response, dependencies: response.dependencies ?? { scenarios: {}, resources: {} }, lifecycle: response.lifecycle ?? {} };
}

// ── Review API ─────────────────────────────────────────────────────────

export type Readiness = "green" | "yellow" | "red";
export type ReviewCheckStatus = "pending" | "running" | "completed" | "failed" | "skipped";
export type { CodeQualityDimension, ProvenanceDimension, ReviewDimensions, StandardsDimension, TestsDimension, VisualDimension };
export type ReviewSummaryResponse = Omit<ReviewSummaryMessage, "readiness"> & { readiness: Readiness };
export type ReviewJobStatus = ReviewJobStatusMessage;

export async function fetchReviewSummary(scenarioName: string, repoId?: string): Promise<ReviewSummaryResponse> {
  return await reviewClient.getSummary(create(GetReviewSummaryRequestSchema, {
    repositoryId: BigInt(repoId ?? "0"), scenarioName,
  })) as ReviewSummaryResponse;
}

export async function triggerReviewRun(req: { scenarioName: string; checks?: string[] }, repoId?: string): Promise<{ jobId: string }> {
  const response = await reviewClient.start(create(StartReviewRequestSchema, {
    repositoryId: BigInt(repoId ?? "0"),
    scenarioName: req.scenarioName,
    checks: req.checks ?? [],
  }));
  return { jobId: response.jobId };
}

export async function fetchReviewJobStatus(jobId: string, repoId?: string): Promise<ReviewJobStatus> {
  return reviewClient.getJobStatus(create(GetReviewJobStatusRequestSchema, {
    repositoryId: BigInt(repoId ?? "0"), jobId,
  }));
}
