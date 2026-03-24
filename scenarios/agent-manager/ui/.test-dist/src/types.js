export { RunnerType, ModelPreset, NetworkAccess, TaskStatus, RunStatus, ApprovalState, RunMode, RunPhase, RunEventType, RecoveryAction, } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";
export { HealthStatus } from "@vrooli/proto-types/common/v1/types_pb";
/** Default context flags */
export const DEFAULT_INVESTIGATION_CONTEXT = {
    runSummaries: true,
    runEvents: true,
    runDiffs: true,
    fullLogs: false,
};
