import {
  ListExecutionResponseSchema,
  ExecutionResponseSchema,
  CreateExecutionRequestSchema,
  FollowUpExecutionRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/execution_pb";
import type {
  ExecutionRecord as ProtoExecutionRecord,
  Finalization as ProtoFinalization,
  ReviewResult as ProtoReviewResult,
} from "@vrooli/proto-types/swarm-manager/v1/domain/execution_pb";
import type {
  ExecutionRecord as ExecutionRecordDomain,
  ExecutionStatus,
  ExecutionMode,
  Finalization,
  FinalizationAggregateClassification,
  FinalizationPhase,
  FinalizationScopeSource,
  FinalizationStatus,
  ReviewClassification,
} from "../../types";
import { EXECUTION_STATUSES, EXECUTION_MODES } from "../../types";
import { createProtoSchema } from "./shared";

const executionStatusSet = new Set<string>(EXECUTION_STATUSES);
const executionModeSet = new Set<string>(EXECUTION_MODES);

function isExecutionStatus(value: unknown): value is ExecutionStatus {
  return typeof value === "string" && executionStatusSet.has(value);
}

function isExecutionMode(value: unknown): value is ExecutionMode {
  return typeof value === "string" && executionModeSet.has(value);
}

export const listExecutionResponseSchema = createProtoSchema(
  ListExecutionResponseSchema,
  "execution list"
);
export const executionResponseSchema = createProtoSchema(
  ExecutionResponseSchema,
  "execution"
);

export { CreateExecutionRequestSchema, FollowUpExecutionRequestSchema };

export function mapProtoExecutionRecord(proto: ProtoExecutionRecord): ExecutionRecordDomain {
  const status = isExecutionStatus(proto.status) ? proto.status : "pending";
  const mode = isExecutionMode(proto.mode) ? proto.mode : "manual";
  const record: ExecutionRecordDomain = {
    executionId: proto.executionId ?? "",
    backlogKind: (proto.backlogKind ?? "idea") as ExecutionRecordDomain["backlogKind"],
    backlogName: proto.backlogName ?? "",
    taskId: proto.taskId,
    runId: proto.runId,
    status,
    mode,
    startedAt: proto.startedAt,
    finishedAt: proto.finishedAt,
    failureReason: proto.failureReason,
    startedBy: proto.startedBy,
    operation: proto.operation as ExecutionRecordDomain["operation"],
    parentExecutionId: proto.parentExecutionId,
    fixupAttempt: proto.fixupAttempt ?? 0,
    createdAt: proto.createdAt ?? "",
    updatedAt: proto.updatedAt ?? "",
  };
  if (proto.finalization) {
    record.finalization = mapProtoFinalization(proto.finalization);
  }
  return record;
}

function mapProtoReviewResult(
  proto: ProtoReviewResult,
): NonNullable<NonNullable<ExecutionRecordDomain["finalization"]>["scenarios"][number]["review"]["result"]> {
  return {
    jobId: proto.jobId ?? "",
    classification: (proto.classification ?? "not_assessable") as ReviewClassification,
    dimensions: (proto.dimensions ?? []).map((dim) => ({
      name: dim.name ?? "",
      status: dim.status ?? "",
      details: dim.details,
    })),
    summary: proto.summary ?? "",
    reviewedAt: proto.reviewedAt ?? "",
  };
}

function mapProtoFinalization(proto: ProtoFinalization): Finalization {
  return {
    eligible: proto.eligible ?? false,
    status: (proto.status ?? "pending") as FinalizationStatus,
    phase: (proto.phase ?? "scope_detection") as FinalizationPhase,
    scopeSource: (proto.scopeSource ?? "none") as FinalizationScopeSource,
    skipReason: proto.skipReason,
    startedAt: proto.startedAt,
    completedAt: proto.completedAt,
    warnings: (proto.warnings ?? []).map((warning) => ({
      code: warning.code ?? "",
      scenarioName: warning.scenarioName,
      message: warning.message ?? "",
      retryable: warning.retryable ?? false,
      createdAt: warning.createdAt ?? "",
    })),
    affectedScenarios: proto.affectedScenarios ?? [],
    aggregateClassification: (proto.aggregateClassification ?? "not_assessable") as FinalizationAggregateClassification,
    aggregateSummary: proto.aggregateSummary,
    scenarios: (proto.scenarios ?? []).map((scenario) => ({
      scenarioName: scenario.scenarioName ?? "",
      changedPaths: scenario.changedPaths ?? [],
      restart: {
        status: (scenario.restart?.status ?? "pending") as FinalizationStatus,
        attempts: scenario.restart?.attempts ?? 0,
        lastError: scenario.restart?.lastError,
        startedAt: scenario.restart?.startedAt,
        finishedAt: scenario.restart?.finishedAt,
      },
      health: {
        status: (scenario.health?.status ?? "pending") as FinalizationStatus,
        scenarioStatus: scenario.health?.scenarioStatus,
        healthStatus: scenario.health?.healthStatus,
        schemaValid: scenario.health?.schemaValid ?? false,
        details: scenario.health?.details,
        checkedAt: scenario.health?.checkedAt,
      },
      review: {
        status: (scenario.review?.status ?? "pending") as FinalizationStatus,
        jobId: scenario.review?.jobId,
        skipReason: scenario.review?.skipReason,
        result: scenario.review?.result ? mapProtoReviewResult(scenario.review.result) : undefined,
      },
    })),
  };
}
