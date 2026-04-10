/**
 * Execution domain types.
 */

import type {
  ExecutionRecord as ProtoExecutionRecord,
} from "@vrooli/proto-types/swarm-manager/v1/domain/execution_pb";
import type { BacklogKind } from "./backlog";
import type { ProtoMessage } from "./shared";

export type ExecutionBacklogKind = BacklogKind | "spec-sync";

export type ExecutionStatus = "pending" | "starting" | "running" | "needs_review" | "validating" | "needs_fixup" | "completed" | "failed" | "canceled";

export type ExecutionMode = "manual" | "yolo";

export type ExecutionOperation = "generator" | "improver" | "fixup" | "followup" | "custom";

/** Classification of a post-execution readiness review. */
export type ReviewClassification = "ready" | "ready_with_notes" | "needs_work" | "not_assessable";

/** A single review dimension result from git-control-tower. */
export interface ReviewDimension {
  name: string;
  /** green, yellow, red, skipped */
  status: string;
  details?: string;
}

/** Post-execution readiness review result. */
export interface ReviewResult {
  jobId: string;
  classification: ReviewClassification;
  dimensions: ReviewDimension[];
  summary: string;
  reviewedAt: string;
}

export type FinalizationStatus = "pending" | "running" | "completed" | "skipped" | "failed";

export type FinalizationPhase = "scope_detection" | "restarting" | "health_check" | "reviewing" | "evidence_gathering" | "completed" | "skipped" | "failed";

export type FinalizationScopeSource = "sandbox_diff" | "acceptance_allow" | "sandbox_diff_plus_acceptance_allow" | "none";

export type FinalizationAggregateClassification = ReviewClassification | "skipped";

export interface FinalizationWarning {
  code: string;
  scenarioName?: string;
  message: string;
  retryable: boolean;
  createdAt: string;
}

export interface ScenarioRestartResult {
  status: FinalizationStatus;
  attempts: number;
  lastError?: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface ScenarioHealthCheckResult {
  status: FinalizationStatus;
  scenarioStatus?: string;
  healthStatus?: string;
  schemaValid: boolean;
  details?: string;
  checkedAt?: string;
}

export interface ScenarioReviewResult {
  status: FinalizationStatus;
  jobId?: string;
  skipReason?: string;
  result?: ReviewResult;
}

export interface ScenarioFinalization {
  scenarioName: string;
  changedPaths: string[];
  restart: ScenarioRestartResult;
  health: ScenarioHealthCheckResult;
  review: ScenarioReviewResult;
}

export interface Finalization {
  eligible: boolean;
  status: FinalizationStatus;
  phase: FinalizationPhase;
  scopeSource: FinalizationScopeSource;
  skipReason?: string;
  startedAt?: string;
  completedAt?: string;
  warnings: FinalizationWarning[];
  affectedScenarios: string[];
  aggregateClassification: FinalizationAggregateClassification;
  aggregateSummary?: string;
  scenarios: ScenarioFinalization[];
}

export type ExecutionRecord = Omit<ProtoMessage<ProtoExecutionRecord>, "status" | "mode" | "operation" | "fixupAttempt" | "backlogKind" | "finalization"> & {
  status: ExecutionStatus;
  mode: ExecutionMode;
  backlogKind: ExecutionBacklogKind;
  operation?: ExecutionOperation;
  parentExecutionId?: string;
  fixupAttempt?: number;
  finalization?: Finalization;
};
