/**
 * Agent Operations domain types — the UI-side vocabulary for the declarative
 * agent-operations runtime (AgentOperationsService projections).
 *
 * These mirror the proto wire contract in
 * packages/proto/schemas/swarm-manager/v1/{domain,api}/agent_operations.proto,
 * projected onto kebab-case string unions the components consume. Mapping from
 * the generated proto messages lives in
 * services/proto/agent-operations-contracts.ts.
 *
 * Doctrine: scenarios/swarm-manager/docs/concepts/AGENT-OPERATIONS.md
 */

/** Target kinds the agent-operations surface addresses from the UI. */
export type AgentOpsTargetKind = "backlog-item" | "initiative";

/** A target selector: kind + domain id (backlog "kind/name" ref or initiative name). */
export interface AgentOpsTarget {
  kind: AgentOpsTargetKind;
  id: string;
}

/** Binding precedence layer (higher wins). */
export type WorkflowBindingLayer =
  | "unspecified"
  | "system-default"
  | "initiative-override"
  | "backlog-item-override"
  | "authorized-invocation";

/** Coordination-layer lifecycle state of a workflow instance. */
export type WorkflowState =
  | "unspecified"
  | "open"
  | "running"
  | "awaiting-decision"
  | "blocked"
  | "terminal-complete"
  | "terminal-abandoned"
  | "terminal-failed";

/** The CLOSED registry of legal domain actions a transition policy may select. */
export type WorkflowDomainAction =
  | "unspecified"
  | "save-decisions"
  | "commit-workshop-round"
  | "start-clarification"
  | "resolve-clarification"
  | "bind-plan"
  | "queue-plan-execution"
  | "start-execution"
  | "commit-review-round"
  | "request-revision"
  | "request-evidence"
  | "complete-item"
  | "fail-item"
  | "create-followup"
  | "open-review"
  | "escalate-needs-attention"
  | "mark-initiative-reviewed";

/** Per-operation execution state within a workflow instance. */
export type WorkflowOperationState =
  | ""
  | "running"
  | "completed"
  | "canceled"
  | "failed"
  | "needs-attention";

/** A human decision recorded against a workflow. */
export interface WorkflowHumanDecision {
  decision: string;
  actor: string;
  atVersion: number;
  note: string;
}

/** A timed intent firing a registered domain action. */
export interface WorkflowTimer {
  intent: string;
  action: WorkflowDomainAction;
  notBefore: string;
}

/**
 * One workflow operation record enriched from its immutable execution
 * snapshot: what ran, which mode at which revision, selected by which binding
 * layer, and its retry lineage.
 */
export interface WorkflowOperationProjection {
  operation: string;
  /** Contract version the execution pinned (empty when the snapshot is missing). */
  operationVersion: string;
  executionId: string;
  /** Live agent-run id (empty on the synchronous path). */
  runId: string;
  state: WorkflowOperationState;
  outcome: string;
  idempotencyKey: string;
  provenanceDigest: string;
  mode: string;
  modeRevision: string;
  bindingLayer: WorkflowBindingLayer;
  bindingOwnerKind: string;
  bindingOwnerId: string;
  /** Snapshot pin timestamp (RFC3339). */
  recordedAt: string;
  /** False when the execution snapshot is missing on disk. */
  snapshotFound: boolean;
  /**
   * True when the snapshot is a Phase-8 legacy execution import (verbatim
   * pre-cutover record). Mode/binding provenance never existed for these, so
   * those fields are honestly empty — render a "legacy import" label instead.
   */
  legacyImport: boolean;
  /** 1-based ordinal among records of the same operation (retry linkage). */
  attempt: number;
  /** Previous attempt's execution id when attempt > 1. */
  priorExecutionId: string;
}

/**
 * THE canonical workflow projection for a target: the durable instance
 * (state, decisions, timers, legal actions) plus snapshot-enriched operations.
 * `found === false` means no workflow document exists for the target
 * (pre-migration legacy item) — callers must fall back to legacy client logic.
 */
export interface WorkflowProjection {
  found: boolean;
  instanceId: string;
  domainKind: string;
  domainId: string;
  state: WorkflowState;
  version: number;
  operations: WorkflowOperationProjection[];
  decisions: WorkflowHumanDecision[];
  timers: WorkflowTimer[];
  legalActions: WorkflowDomainAction[];
  policyId: string;
  policyRevision: string;
}

/** One immutable execution provenance summary (newest first from the API). */
export interface WorkflowExecutionSummary {
  executionId: string;
  operation: string;
  operationVersion: string;
  mode: string;
  modeRevision: string;
  bindingLayer: WorkflowBindingLayer;
  compiledModeDigest: string;
  promptCatalogDigest: string;
  callerInputDigest: string;
  outcome: string;
  /** True when the snapshot's recomputed digests still equal the pinned ones. */
  reproducible: boolean;
  recordedAt: string;
  /**
   * True when this row is a Phase-8 legacy execution import (verbatim
   * pre-cutover record): mode/digest provenance never existed, outcome is the
   * legacy entry's own terminal status, and recordedAt is the deterministic
   * imported_at.
   */
  legacyImport: boolean;
}

/** A binding selecting which mode implements an operation for a scope. */
export interface WorkflowOperationBinding {
  operation: string;
  operationVersion: string;
  layer: WorkflowBindingLayer;
  ownerKind: string;
  ownerId: string;
  mode: string;
  modeRevision: string;
  disabled: boolean;
}

/** One in-scope candidate binding at some layer. */
export interface WorkflowBindingContribution {
  binding: WorkflowOperationBinding;
  winning: boolean;
}

/** Per-operation resolution result (fail-closed errors are typed, not thrown). */
export interface WorkflowResolvedBinding {
  operation: string;
  operationVersion: string;
  resolved: boolean;
  binding: WorkflowOperationBinding | null;
  policyId: string;
  policyRevision: string;
  /** no-binding | invalid-override | deleted-revision | incompatible-mode | internal */
  error: string;
  errorMessage: string;
  contributions: WorkflowBindingContribution[];
}

/** One raw override document stored at an owner's layer. */
export interface WorkflowBindingOverrideDocument {
  binding: WorkflowOperationBinding;
  file: string;
  revision: string;
  updatedAt: string;
}

/** Compatibility verdict for one mode × operation pair against a target. */
export interface WorkflowModeOperationVerdict {
  operation: string;
  operationVersion: string;
  compatible: boolean;
  reason: string;
}

/** One authored mode with its per-operation verdicts for a target. */
export interface WorkflowCompatibleMode {
  mode: string;
  modeRevision: string;
  modeDigest: string;
  targetKind: string;
  verdicts: WorkflowModeOperationVerdict[];
}

/** Persisted-state migration lifecycle (Phase-8 tooling writes the document). */
export type WorkflowMigrationState = "not-started" | "staged" | "promoted" | "quarantined";

/** Projection of the persisted-state migration status document. */
export interface WorkflowMigrationStatus {
  state: WorkflowMigrationState;
  epoch: number;
  stagedCount: number;
  promotedCount: number;
  quarantinedCount: number;
  startedAt: string;
  updatedAt: string;
  reportPath: string;
  documentFound: boolean;
}

/** Result of writing a binding override. */
export interface WorkflowPutBindingOverrideResult {
  stored: WorkflowOperationBinding | null;
  file: string;
  revision: string;
}

/** Human-readable labels for binding layers (operator-facing). */
export const WORKFLOW_BINDING_LAYER_LABELS: Record<WorkflowBindingLayer, string> = {
  unspecified: "Unknown layer",
  "system-default": "System default",
  "initiative-override": "Initiative override",
  "backlog-item-override": "Item override",
  "authorized-invocation": "Authorized invocation",
};
