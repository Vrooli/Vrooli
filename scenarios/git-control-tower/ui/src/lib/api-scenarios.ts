// ============================================================================
// Scenario Listing, Envelope & Review — Types + API Functions
// ============================================================================

import { API_BASE, buildRepoHeaders, handleResponse, buildApiUrl } from "./api-internals";
import { create } from "@bufbuild/protobuf";
import { StartReviewRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/review/review_pb";
import { reviewClient } from "./connect";

// ── Scenario Listing ───────────────────────────────────────────────────

export interface ScenarioInfo {
  name: string;
  display_name: string;
  description: string;
  status: "running" | "stopped";
  health_status: string | null;
  tags: string[];
  runtime: string;
}

export async function fetchScenarios(): Promise<ScenarioInfo[]> {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" } });
  return handleResponse<ScenarioInfo[]>(res);
}

// ── Scenario Envelope ──────────────────────────────────────────────────

/** Enriched scenario metadata derived from service.json, used to build the agent envelope. */
export interface ScenarioEnvelopeData {
  name: string;
  displayName: string;
  description: string;
  path: string;
  tags: string[];
  dependencies: {
    scenarios: Record<string, string>;
    resources: Record<string, string>;
  };
  lifecycle: {
    testCommand?: string;
    buildCommand?: string;
  };
}

/** Fetch enriched scenario metadata for the agent envelope. */
export async function fetchScenarioEnvelope(slug: string): Promise<ScenarioEnvelopeData> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(slug)}/envelope`, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" } });
  return handleResponse<ScenarioEnvelopeData>(res);
}

// ── Review API ─────────────────────────────────────────────────────────

export type Readiness = "green" | "yellow" | "red";
export type ReviewCheckStatus = "pending" | "running" | "completed" | "failed" | "skipped";

export interface CodeQualityDimension {
  available: boolean;
  score: number;
  violations: number;
  stale: boolean;
  lastScan?: string;
}

export interface TestsDimension {
  available: boolean;
  passed: boolean;
  total: number;
  passedCount: number;
  failedCount: number;
  lastRun?: string;
}

export interface StandardsDimension {
  available: boolean;
  blockingViolations: number;
  warnings: number;
  totalViolations: number;
}

export interface VisualDimension {
  available: boolean;
  screenshotCount: number;
  stale: boolean;
}

export interface ProvenanceDimension {
  available: boolean;
  tracedFiles: number;
}

export interface ReviewDimensions {
  codeQuality?: CodeQualityDimension;
  tests?: TestsDimension;
  standards?: StandardsDimension;
  visual?: VisualDimension;
  provenance?: ProvenanceDimension;
}

export interface ReviewSummaryResponse {
  scenarioName: string;
  readiness: Readiness;
  dimensions: ReviewDimensions;
  capabilities: Record<string, boolean>;
  timestamp: string;
}

export interface ReviewJobStatus {
  jobId: string;
  status: string;
  checks: Record<string, ReviewCheckStatus>;
  summary?: ReviewSummaryResponse;
  startedAt: string;
  error?: string;
}

export async function fetchReviewSummary(scenarioName: string, repoId?: string): Promise<ReviewSummaryResponse> {
  const params = new URLSearchParams({ scenarioName });
  const url = buildApiUrl(`/review/summary?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<ReviewSummaryResponse>(res);
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
  const res = await fetch(buildApiUrl(`/review/run/${encodeURIComponent(jobId)}`, { baseUrl: API_BASE }), {
    headers: buildRepoHeaders(repoId),
    cache: "no-store",
  });
  return handleResponse<ReviewJobStatus>(res);
}
