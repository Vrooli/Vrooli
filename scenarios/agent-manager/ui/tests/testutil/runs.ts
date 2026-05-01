import { ApprovalState, RunMode, RunPhase, RunStatus, type Run } from "../../src/types.js";

export type RunOverrides = Partial<Run>;

export function makeRun(overrides: RunOverrides = {}): Run {
  return {
    id: overrides.id ?? "run-1",
    taskId: overrides.taskId ?? "task-1",
    tag: overrides.tag ?? "run-1",
    runMode: overrides.runMode ?? RunMode.SANDBOXED,
    status: overrides.status ?? RunStatus.COMPLETED,
    phase: overrides.phase ?? RunPhase.DONE,
    progressPercent: overrides.progressPercent ?? 100,
    idempotencyKey: overrides.idempotencyKey ?? "idem-1",
    errorMsg: overrides.errorMsg ?? "",
    approvalState: overrides.approvalState ?? ApprovalState.APPROVAL_STATE_UNSPECIFIED,
    approvedBy: overrides.approvedBy ?? "",
    diffPath: overrides.diffPath ?? "",
    logPath: overrides.logPath ?? "",
    changedFiles: overrides.changedFiles ?? 0,
    totalSizeBytes: overrides.totalSizeBytes ?? 0n,
    sessionId: overrides.sessionId ?? "",
    promptPreview: overrides.promptPreview ?? "",
    requestedModel: overrides.requestedModel ?? "",
    actualModel: overrides.actualModel ?? "",
    actions: overrides.actions ?? {
      canInvestigate: false,
      canApplyInvestigation: false,
      canDelete: false,
      canStop: false,
      canRetry: false,
      canContinue: false,
      canApprove: false,
      canReject: false,
      canReview: false,
      canExtractRecommendations: false,
      canRegenerateRecommendations: false,
      canContinueReason: "Run is complete",
      canResumeFromFailure: false,
      canResumeFromFailureReason: "",
    } as Run["actions"],
    ...overrides,
  } as Run;
}
