import { create } from "@bufbuild/protobuf";
import { RunSchema, RunActionsSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";
import { ApprovalState, RunFinalizationStatus, RunMode, RunPhase, RunStatus, type Run } from "../../types.js";

export type RunOverrides = Partial<Run>;

export function makeRun(overrides: RunOverrides = {}): Run {
  const { $typeName: _typeName, $unknown: _unknown, ...fields } = overrides;
  return create(RunSchema, {
    id: overrides.id ?? "run-1",
    taskId: overrides.taskId ?? "task-1",
    tag: overrides.tag ?? "run-1",
    runMode: overrides.runMode ?? RunMode.SANDBOXED,
    status: overrides.status ?? RunStatus.COMPLETE,
    phase: overrides.phase ?? RunPhase.COMPLETED,
    progressPercent: overrides.progressPercent ?? 100,
    idempotencyKey: overrides.idempotencyKey ?? "idem-1",
    errorMsg: overrides.errorMsg ?? "",
    finalizationStatus: overrides.finalizationStatus ?? RunFinalizationStatus.NONE,
    finalizationError: overrides.finalizationError ?? "",
    approvalState: overrides.approvalState ?? ApprovalState.UNSPECIFIED,
    approvedBy: overrides.approvedBy ?? "",
    diffPath: overrides.diffPath ?? "",
    logPath: overrides.logPath ?? "",
    changedFiles: overrides.changedFiles ?? 0,
    totalSizeBytes: overrides.totalSizeBytes ?? 0n,
    sessionId: overrides.sessionId ?? "",
    promptPreview: overrides.promptPreview ?? "",
    requestedModel: overrides.requestedModel ?? "",
    actualModel: overrides.actualModel ?? "",
    actions: overrides.actions ?? create(RunActionsSchema, {
      canInvestigate: false,
      canApplyInvestigation: false,
      canDelete: false,
      canStop: false,
      canRetry: false,
      canContinue: false,
      canApprove: false,
      canReject: false,
      canReview: false,
      canContinueReason: "Run is complete",
      canResumeFromFailure: false,
      canResumeFromFailureReason: "",
      finalizationWarning: "",
      canRetryFinalization: false,
    }),
    ...fields,
  });
}
