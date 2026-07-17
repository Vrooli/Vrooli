/**
 * Presentation helpers over the canonical workflow projection.
 *
 * These only reshape server-provided data for display (indexing, matching a
 * round/run back to its operation record) — they never make precedence or
 * transition decisions.
 */

import type {
  WorkflowCompatibleMode,
  WorkflowExecutionSummary,
  WorkflowOperationProjection,
  WorkflowProjection,
} from "../types/agent-operations";
import { WORKFLOW_BINDING_LAYER_LABELS } from "../types/agent-operations";
import type { WorkflowBindingLayer } from "../types/agent-operations";

/**
 * Normalized provenance view rendered by OperationProvenancePopover. Built
 * from projection operations or execution-history summaries — never computed
 * client-side.
 */
export interface OperationProvenanceData {
  /** Which store the record came from — label sources honestly in the UI. */
  source: "canonical";
  operation: string;
  operationVersion: string;
  executionId: string;
  runId: string;
  mode: string;
  modeRevision: string;
  bindingLayer: WorkflowBindingLayer;
  bindingOwnerKind: string;
  bindingOwnerId: string;
  recordedAt: string;
  attempt?: number;
  priorExecutionId?: string;
  state?: string;
  outcome?: string;
  snapshotFound?: boolean;
  provenanceDigest?: string;
  compiledModeDigest?: string;
  promptCatalogDigest?: string;
  callerInputDigest?: string;
  /** Verified evidence: snapshot digests still reproduce the pinned provenance. */
  reproducible?: boolean;
}

export function provenanceFromOperation(op: WorkflowOperationProjection): OperationProvenanceData {
  return {
    source: "canonical",
    operation: op.operation,
    operationVersion: op.operationVersion,
    executionId: op.executionId,
    runId: op.runId,
    mode: op.mode,
    modeRevision: op.modeRevision,
    bindingLayer: op.bindingLayer,
    bindingOwnerKind: op.bindingOwnerKind,
    bindingOwnerId: op.bindingOwnerId,
    recordedAt: op.recordedAt,
    attempt: op.attempt,
    priorExecutionId: op.priorExecutionId || undefined,
    state: op.state || undefined,
    outcome: op.outcome || undefined,
    snapshotFound: op.snapshotFound,
    provenanceDigest: op.provenanceDigest || undefined,
  };
}

export function provenanceFromExecutionSummary(
  summary: WorkflowExecutionSummary,
): OperationProvenanceData {
  return {
    source: "canonical",
    operation: summary.operation,
    operationVersion: summary.operationVersion,
    executionId: summary.executionId,
    runId: "",
    mode: summary.mode,
    modeRevision: summary.modeRevision,
    bindingLayer: summary.bindingLayer,
    bindingOwnerKind: "",
    bindingOwnerId: "",
    recordedAt: summary.recordedAt,
    outcome: summary.outcome || undefined,
    compiledModeDigest: summary.compiledModeDigest || undefined,
    promptCatalogDigest: summary.promptCatalogDigest || undefined,
    callerInputDigest: summary.callerInputDigest || undefined,
    reproducible: summary.reproducible,
  };
}

/** Terminal operation states — anything else counts as in-flight. */
const TERMINAL_OPERATION_STATES: ReadonlySet<string> = new Set([
  "completed",
  "canceled",
  "failed",
]);

/**
 * Whether the projection has work in flight — drives the poll-while-active
 * idiom (any projected operation in a non-terminal state).
 */
export function isWorkflowProjectionActive(
  projection: WorkflowProjection | undefined | null,
): boolean {
  if (!projection?.found) return false;
  return projection.operations.some(
    (op) => op.state !== "" && !TERMINAL_OPERATION_STATES.has(op.state),
  );
}

/** Index canonical operations by execution id (initiative rounds carry one). */
export function provenanceByExecutionId(
  projection: WorkflowProjection | undefined | null,
): ReadonlyMap<string, OperationProvenanceData> {
  const map = new Map<string, OperationProvenanceData>();
  for (const op of projection?.operations ?? []) {
    if (op.executionId) map.set(op.executionId, provenanceFromOperation(op));
  }
  return map;
}

/** Find the canonical operation dispatched under a live agent run, if any. */
export function provenanceForRun(
  projection: WorkflowProjection | undefined | null,
  runId: string | undefined | null,
): OperationProvenanceData | null {
  if (!runId || !projection?.found) return null;
  const op = projection.operations.find((o) => o.runId === runId);
  return op ? provenanceFromOperation(op) : null;
}

/**
 * Index canonical operations by attempt ordinal for operations whose id
 * matches `predicate` — correlates workshop/review round N with the Nth
 * attempt of the corresponding operation (a retry is always a new record).
 */
export function provenanceByAttempt(
  projection: WorkflowProjection | undefined | null,
  predicate: (operationId: string) => boolean,
): ReadonlyMap<number, OperationProvenanceData> {
  const map = new Map<number, OperationProvenanceData>();
  for (const op of projection?.operations ?? []) {
    if (op.attempt > 0 && predicate(op.operation)) {
      map.set(op.attempt, provenanceFromOperation(op));
    }
  }
  return map;
}

export const isWorkshopOperation = (operationId: string): boolean =>
  operationId.includes("workshop");

export const isReviewOperation = (operationId: string): boolean =>
  operationId.includes("review");

/** Short display form of a sha256 digest ("sha256:ab12cd34…"). */
export function shortDigest(digest: string | undefined, chars = 12): string {
  if (!digest) return "";
  const [prefix, hex] = digest.includes(":")
    ? [digest.slice(0, digest.indexOf(":") + 1), digest.slice(digest.indexOf(":") + 1)]
    : ["", digest];
  return hex.length > chars ? `${prefix}${hex.slice(0, chars)}…` : digest;
}

/** Operator label for a binding source: layer + owning scope. */
export function bindingSourceLabel(data: OperationProvenanceData): string {
  const layer = WORKFLOW_BINDING_LAYER_LABELS[data.bindingLayer];
  if (data.bindingOwnerKind === "system" || !data.bindingOwnerId) return layer;
  return `${layer} (${data.bindingOwnerKind} ${data.bindingOwnerId})`;
}

/**
 * Display filter over the server's ListCompatibleModes verdicts: a mode is
 * offered for an operation when the server marked that operation compatible.
 * Version-pinned verdicts must match the pinned version; version-agnostic
 * verdicts (empty) apply to any. No client-side compatibility judgement —
 * this only selects which server verdicts to show.
 */
export function compatibleModesForOperation(
  modes: WorkflowCompatibleMode[],
  operation: string,
  operationVersion: string,
): WorkflowCompatibleMode[] {
  return modes.filter((mode) =>
    mode.verdicts.some(
      (verdict) =>
        verdict.operation === operation &&
        verdict.compatible &&
        (verdict.operationVersion === "" ||
          operationVersion === "" ||
          verdict.operationVersion === operationVersion),
    ),
  );
}
