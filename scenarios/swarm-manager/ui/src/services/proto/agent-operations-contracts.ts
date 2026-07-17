/**
 * Agent Operations proto contracts — maps the generated AgentOperationsService
 * messages onto the hand-authored domain types in types/agent-operations.ts.
 *
 * Every enum mapping is an exhaustive switch over the generated enum so a new
 * proto value fails type-checking here instead of silently leaking a numeric
 * value into the UI. Follows the settings-contracts.ts pattern.
 */

import {
  AgentOpsBindingLayer,
  AgentOpsDomainAction,
  AgentOpsWorkflowState,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_operations_pb";
import type {
  AgentOpsHumanDecision,
  AgentOpsOperationBinding,
  AgentOpsScheduledIntent,
} from "@vrooli/proto-types/swarm-manager/v1/domain/agent_operations_pb";
import { OperatingModeTargetKind } from "@vrooli/proto-types/swarm-manager/v1/domain/operating_mode_pb";
import type {
  AgentOpsBindingContribution,
  AgentOpsBindingOverrideDocument,
  AgentOpsCompatibleMode,
  AgentOpsExecutionSummary,
  AgentOpsGetMigrationStatusResponse,
  AgentOpsGetResolvedBindingsResponse,
  AgentOpsGetWorkflowProjectionResponse,
  AgentOpsListBindingOverridesResponse,
  AgentOpsListCompatibleModesResponse,
  AgentOpsListExecutionHistoryResponse,
  AgentOpsModeOperationVerdict,
  AgentOpsOperationProjection,
  AgentOpsPutBindingOverrideResponse,
  AgentOpsResolvedOperationBinding,
} from "@vrooli/proto-types/swarm-manager/v1/api/agent_operations_pb";
import type {
  AgentOpsTargetKind,
  WorkflowBindingContribution,
  WorkflowBindingLayer,
  WorkflowBindingOverrideDocument,
  WorkflowCompatibleMode,
  WorkflowDomainAction,
  WorkflowExecutionSummary,
  WorkflowHumanDecision,
  WorkflowMigrationState,
  WorkflowMigrationStatus,
  WorkflowModeOperationVerdict,
  WorkflowOperationBinding,
  WorkflowOperationProjection,
  WorkflowOperationState,
  WorkflowProjection,
  WorkflowPutBindingOverrideResult,
  WorkflowResolvedBinding,
  WorkflowState,
  WorkflowTimer,
} from "../../types/agent-operations";

// ---------------------------------------------------------------------------
// Enum mappings (proto -> kebab-case domain strings)
// ---------------------------------------------------------------------------

export function mapBindingLayer(layer: AgentOpsBindingLayer | undefined): WorkflowBindingLayer {
  switch (layer) {
    case AgentOpsBindingLayer.SYSTEM_DEFAULT:
      return "system-default";
    case AgentOpsBindingLayer.INITIATIVE_OVERRIDE:
      return "initiative-override";
    case AgentOpsBindingLayer.BACKLOG_ITEM_OVERRIDE:
      return "backlog-item-override";
    case AgentOpsBindingLayer.AUTHORIZED_INVOCATION:
      return "authorized-invocation";
    case AgentOpsBindingLayer.UNSPECIFIED:
    case undefined:
      return "unspecified";
  }
  // Runtime guard: an unknown wire value (newer server) degrades to unspecified.
  return "unspecified";
}

export function mapWorkflowState(state: AgentOpsWorkflowState | undefined): WorkflowState {
  switch (state) {
    case AgentOpsWorkflowState.OPEN:
      return "open";
    case AgentOpsWorkflowState.RUNNING:
      return "running";
    case AgentOpsWorkflowState.AWAITING_DECISION:
      return "awaiting-decision";
    case AgentOpsWorkflowState.BLOCKED:
      return "blocked";
    case AgentOpsWorkflowState.TERMINAL_COMPLETE:
      return "terminal-complete";
    case AgentOpsWorkflowState.TERMINAL_ABANDONED:
      return "terminal-abandoned";
    case AgentOpsWorkflowState.TERMINAL_FAILED:
      return "terminal-failed";
    case AgentOpsWorkflowState.UNSPECIFIED:
    case undefined:
      return "unspecified";
  }
  return "unspecified";
}

export function mapDomainAction(action: AgentOpsDomainAction | undefined): WorkflowDomainAction {
  switch (action) {
    case AgentOpsDomainAction.SAVE_DECISIONS:
      return "save-decisions";
    case AgentOpsDomainAction.COMMIT_WORKSHOP_ROUND:
      return "commit-workshop-round";
    case AgentOpsDomainAction.START_CLARIFICATION:
      return "start-clarification";
    case AgentOpsDomainAction.RESOLVE_CLARIFICATION:
      return "resolve-clarification";
    case AgentOpsDomainAction.BIND_PLAN:
      return "bind-plan";
    case AgentOpsDomainAction.QUEUE_PLAN_EXECUTION:
      return "queue-plan-execution";
    case AgentOpsDomainAction.START_EXECUTION:
      return "start-execution";
    case AgentOpsDomainAction.COMMIT_REVIEW_ROUND:
      return "commit-review-round";
    case AgentOpsDomainAction.REQUEST_REVISION:
      return "request-revision";
    case AgentOpsDomainAction.REQUEST_EVIDENCE:
      return "request-evidence";
    case AgentOpsDomainAction.COMPLETE_ITEM:
      return "complete-item";
    case AgentOpsDomainAction.FAIL_ITEM:
      return "fail-item";
    case AgentOpsDomainAction.CREATE_FOLLOWUP:
      return "create-followup";
    case AgentOpsDomainAction.OPEN_REVIEW:
      return "open-review";
    case AgentOpsDomainAction.ESCALATE_NEEDS_ATTENTION:
      return "escalate-needs-attention";
    case AgentOpsDomainAction.MARK_INITIATIVE_REVIEWED:
      return "mark-initiative-reviewed";
    case AgentOpsDomainAction.UNSPECIFIED:
    case undefined:
      return "unspecified";
  }
  return "unspecified";
}

/** Map a domain target kind to the proto enum used by target selectors. */
export function targetKindToProto(kind: AgentOpsTargetKind): OperatingModeTargetKind {
  switch (kind) {
    case "backlog-item":
      return OperatingModeTargetKind.BACKLOG_ITEM;
    case "initiative":
      return OperatingModeTargetKind.INITIATIVE;
  }
}

const OPERATION_STATES: ReadonlySet<string> = new Set([
  "running",
  "completed",
  "canceled",
  "failed",
  "needs-attention",
]);

export function mapOperationState(state: string | undefined): WorkflowOperationState {
  return state && OPERATION_STATES.has(state) ? (state as WorkflowOperationState) : "";
}

const MIGRATION_STATES: ReadonlySet<string> = new Set([
  "not-started",
  "staged",
  "promoted",
  "quarantined",
]);

export function mapMigrationState(state: string | undefined): WorkflowMigrationState {
  return state && MIGRATION_STATES.has(state)
    ? (state as WorkflowMigrationState)
    : "not-started";
}

// ---------------------------------------------------------------------------
// Message mappings
// ---------------------------------------------------------------------------

function mapDecision(d: AgentOpsHumanDecision | undefined): WorkflowHumanDecision {
  return {
    decision: d?.decision ?? "",
    actor: d?.actor ?? "",
    atVersion: d?.atVersion ?? 0,
    note: d?.note ?? "",
  };
}

function mapTimer(t: AgentOpsScheduledIntent | undefined): WorkflowTimer {
  return {
    intent: t?.intent ?? "",
    action: mapDomainAction(t?.action),
    notBefore: t?.notBefore ?? "",
  };
}

function mapOperationProjection(
  op: AgentOpsOperationProjection | undefined,
): WorkflowOperationProjection {
  return {
    operation: op?.operation ?? "",
    operationVersion: op?.operationVersion ?? "",
    executionId: op?.executionId ?? "",
    runId: op?.runId ?? "",
    state: mapOperationState(op?.state),
    outcome: op?.outcome ?? "",
    idempotencyKey: op?.idempotencyKey ?? "",
    provenanceDigest: op?.provenanceDigest ?? "",
    mode: op?.mode ?? "",
    modeRevision: op?.modeRevision ?? "",
    bindingLayer: mapBindingLayer(op?.bindingLayer),
    bindingOwnerKind: op?.bindingOwnerKind ?? "",
    bindingOwnerId: op?.bindingOwnerId ?? "",
    recordedAt: op?.recordedAt ?? "",
    snapshotFound: op?.snapshotFound ?? false,
    attempt: op?.attempt ?? 0,
    priorExecutionId: op?.priorExecutionId ?? "",
    legacyImport: op?.legacyImport ?? false,
  };
}

export function mapWorkflowProjection(
  resp: AgentOpsGetWorkflowProjectionResponse | undefined,
): WorkflowProjection {
  const workflow = resp?.workflow;
  return {
    found: resp?.found ?? false,
    instanceId: workflow?.instanceId ?? "",
    domainKind: workflow?.domainKind ?? "",
    domainId: workflow?.domainId ?? "",
    state: mapWorkflowState(workflow?.state),
    version: workflow?.version ?? 0,
    operations: (resp?.operations ?? []).map(mapOperationProjection),
    decisions: (workflow?.decisions ?? []).map(mapDecision),
    timers: (workflow?.timers ?? []).map(mapTimer),
    legalActions: (workflow?.legalActions ?? []).map(mapDomainAction),
    policyId: resp?.policyId ?? "",
    policyRevision: resp?.policyRevision ?? "",
  };
}

function mapExecutionSummary(s: AgentOpsExecutionSummary | undefined): WorkflowExecutionSummary {
  return {
    executionId: s?.executionId ?? "",
    operation: s?.operation ?? "",
    operationVersion: s?.operationVersion ?? "",
    mode: s?.mode ?? "",
    modeRevision: s?.modeRevision ?? "",
    bindingLayer: mapBindingLayer(s?.bindingLayer),
    compiledModeDigest: s?.compiledModeDigest ?? "",
    promptCatalogDigest: s?.promptCatalogDigest ?? "",
    callerInputDigest: s?.callerInputDigest ?? "",
    outcome: s?.outcome ?? "",
    reproducible: s?.reproducible ?? false,
    recordedAt: s?.recordedAt ?? "",
    legacyImport: s?.legacyImport ?? false,
  };
}

export function mapExecutionHistory(
  resp: AgentOpsListExecutionHistoryResponse | undefined,
): WorkflowExecutionSummary[] {
  return (resp?.executions ?? []).map(mapExecutionSummary);
}

function mapBinding(b: AgentOpsOperationBinding | undefined): WorkflowOperationBinding {
  return {
    operation: b?.operation ?? "",
    operationVersion: b?.operationVersion ?? "",
    layer: mapBindingLayer(b?.layer),
    ownerKind: b?.owner?.kind ?? "",
    ownerId: b?.owner?.id ?? "",
    mode: b?.mode ?? "",
    modeRevision: b?.modeRevision ?? "",
    disabled: b?.disabled ?? false,
  };
}

function mapContribution(c: AgentOpsBindingContribution | undefined): WorkflowBindingContribution {
  return {
    binding: mapBinding(c?.binding),
    winning: c?.winning ?? false,
  };
}

function mapResolvedBinding(
  r: AgentOpsResolvedOperationBinding | undefined,
): WorkflowResolvedBinding {
  return {
    operation: r?.operation ?? "",
    operationVersion: r?.operationVersion ?? "",
    resolved: r?.resolved ?? false,
    binding: r?.binding ? mapBinding(r.binding) : null,
    policyId: r?.policyId ?? "",
    policyRevision: r?.policyRevision ?? "",
    error: r?.error ?? "",
    errorMessage: r?.errorMessage ?? "",
    contributions: (r?.contributions ?? []).map(mapContribution),
  };
}

export function mapResolvedBindings(
  resp: AgentOpsGetResolvedBindingsResponse | undefined,
): WorkflowResolvedBinding[] {
  return (resp?.operations ?? []).map(mapResolvedBinding);
}

function mapOverrideDocument(
  d: AgentOpsBindingOverrideDocument | undefined,
): WorkflowBindingOverrideDocument {
  return {
    binding: mapBinding(d?.binding),
    file: d?.file ?? "",
    revision: d?.revision ?? "",
    updatedAt: d?.updatedAt ?? "",
  };
}

export function mapBindingOverrides(
  resp: AgentOpsListBindingOverridesResponse | undefined,
): WorkflowBindingOverrideDocument[] {
  return (resp?.overrides ?? []).map(mapOverrideDocument);
}

function mapVerdict(v: AgentOpsModeOperationVerdict | undefined): WorkflowModeOperationVerdict {
  return {
    operation: v?.operation ?? "",
    operationVersion: v?.operationVersion ?? "",
    compatible: v?.compatible ?? false,
    reason: v?.reason ?? "",
  };
}

function mapCompatibleMode(m: AgentOpsCompatibleMode | undefined): WorkflowCompatibleMode {
  return {
    mode: m?.mode ?? "",
    modeRevision: m?.modeRevision ?? "",
    modeDigest: m?.modeDigest ?? "",
    targetKind: targetKindLabel(m?.targetKind),
    verdicts: (m?.verdicts ?? []).map(mapVerdict),
  };
}

function targetKindLabel(kind: OperatingModeTargetKind | undefined): string {
  switch (kind) {
    case OperatingModeTargetKind.BACKLOG_ITEM:
      return "backlog-item";
    case OperatingModeTargetKind.INITIATIVE:
      return "initiative";
    case OperatingModeTargetKind.PLAN_EXECUTION:
      return "plan-execution";
    case OperatingModeTargetKind.SCENARIO:
      return "scenario";
    default:
      return "";
  }
}

export function mapCompatibleModes(
  resp: AgentOpsListCompatibleModesResponse | undefined,
): WorkflowCompatibleMode[] {
  return (resp?.modes ?? []).map(mapCompatibleMode);
}

export function mapMigrationStatus(
  resp: AgentOpsGetMigrationStatusResponse | undefined,
): WorkflowMigrationStatus {
  return {
    state: mapMigrationState(resp?.state),
    epoch: resp?.epoch ?? 0,
    stagedCount: resp?.stagedCount ?? 0,
    promotedCount: resp?.promotedCount ?? 0,
    quarantinedCount: resp?.quarantinedCount ?? 0,
    startedAt: resp?.startedAt ?? "",
    updatedAt: resp?.updatedAt ?? "",
    reportPath: resp?.reportPath ?? "",
    documentFound: resp?.documentFound ?? false,
  };
}

export function mapPutBindingOverride(
  resp: AgentOpsPutBindingOverrideResponse | undefined,
): WorkflowPutBindingOverrideResult {
  return {
    stored: resp?.stored ? mapBinding(resp.stored) : null,
    file: resp?.file ?? "",
    revision: resp?.revision ?? "",
  };
}
