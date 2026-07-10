import type { InitiativeOperatingMode } from "./initiative";
import type { Proposal } from "./feedback";

export type OperatingModeRoundStatus =
  | "reserved"
  | "agent_running"
  | "completed"
  | "needs_attention"
  | "failed"
  | "canceled";

export interface OperatingModeArtifactDefinition {
  path: string;
  contentType?: string;
  required?: boolean;
}

export interface OperatingModeArtifactSnapshot extends OperatingModeArtifactDefinition {
  content?: string;
  updatedAt?: string;
  sizeBytes?: number;
}

/**
 * Phase classification axis used for Operations Center column placement,
 * lane-cap utilization bars, and per-lane metrics. Mirrors the backend
 * `operatingmode.PhaseKind` enum. New kinds require coordinated changes
 * across lane plumbing, UI columns, and the authoring contract.
 */
export type OperatingModePhaseKind =
  | "investigate"
  | "execute"
  | "review"
  | "reconcile";

export interface OperatingModeWorkspacePhase {
  phase: string;
  phaseKind: OperatingModePhaseKind | "";
  activityPurpose: string;
  profileKey: string;
  writesRepo: boolean;
  outputArtifacts?: OperatingModeArtifactDefinition[];
  requiresCriteria?: boolean;
  startable: boolean;
  reason?: string;
  next?: boolean;
  /**
   * When non-empty, names the predecessor phase whose successful completion
   * auto-starts this phase via the round-refresher hook. Length ≤ 1 in v1
   * (validator-enforced server-side).
   */
  autoStartAfter?: string[];
  /**
   * When non-empty, names the sub-mode that executes this phase (phase
   * delegation, `executed_by`). The composed sub-mode flow renders inline; the
   * backend remains the routing source of truth.
   */
  executedBy?: string;
}

/**
 * A phase's declared reads grouped by supplying provider: the generic-base
 * provider (`base`) vs the mode's target adapter (`target`). Derived from the
 * declared contract, so surfaces render reads from data, not a fixed category
 * list.
 */
export interface OperatingModePhaseReads {
  base: string[];
  target: string[];
}

export interface PhaseOutputContractSummary {
  requiresStructuredResult: boolean;
  requiresProgress: boolean;
  requiresVerdict: boolean;
  requiresHandoff: boolean;
  requiresBacklogSync: boolean;
  requiredArtifactCount: number;
}

export interface PhaseResultBinding {
  kind: "progress_artifact";
  artifact: OperatingModeArtifactDefinition;
}

/**
 * Generic field-predicate guard operator, mirroring the backend `Guard` model
 * (api/internal/operatingmode/guard.go). A transition's `conditionKind` is the
 * guard op — a leaf comparison/membership/presence op, the unconditional
 * `always`, or a composite (`all`/`any`/`not`) — not a closed, mode-specific
 * branch kind. `string` widens it so a new op never breaks the UI build.
 */
export type OperatingModeTransitionConditionKind =
  | "always"
  | "eq"
  | "ne"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "in"
  | "not_in"
  | "exists"
  | "not_exists"
  | "all"
  | "any"
  | "not"
  | (string & {});

export interface OperatingModePhaseTransition {
  from: string;
  to: string;
  conditionKind: OperatingModeTransitionConditionKind;
  label: string;
  /** Dotted field-path the leaf guard reads from the round's structured output. */
  field?: string;
  /** Server-rendered comparison value for the leaf guard (string form). */
  value?: string;
  /**
   * True when the guard field is derived by classification-on-transition (from
   * the handoff) rather than emitted directly by the round. The guard itself is
   * an ordinary eq-guard; this only records where the compared value comes from.
   */
  classified?: boolean;
}

/**
 * A phase's classification-on-transition contract, present when one of its
 * outgoing edges derives its routing field from the handoff instead of a
 * directly-emitted field. It costs no agent round — the engine derives the
 * field via the resolution ladder at the edge, abstaining to needs_attention
 * rather than fabricating a route. Rendered as a built-in step, not a phase.
 */
export interface OperatingModePhaseClassification {
  field: string;
  enum: string[];
  from?: string;
  description?: string;
}

export interface OperatingModePhaseGraph {
  startPhase: string;
  terminal: string[];
  transitions: OperatingModePhaseTransition[];
  acceptedVerdicts: string[];
}

export interface OperatingModeCatalogPhase {
  phase: string;
  phaseKind: OperatingModePhaseKind | "";
  label: string;
  title: string;
  purpose: string;
  trigger: string;
  profileKey: string;
  writesRepo: boolean;
  requiresCriteria?: boolean;
  isStart?: boolean;
  isTerminal?: boolean;
  outputArtifacts?: OperatingModeArtifactDefinition[];
  /**
   * The phase's declared input contract grouped by supplying provider — the
   * generic-base provider (`base`) vs the mode's target adapter (`target`).
   * The Reads tab renders from this, symmetric with the Emits (outputContract)
   * side. Absent for phases that declare no reads (e.g. delegated phases).
   */
  reads?: OperatingModePhaseReads;
  outputContract: PhaseOutputContractSummary;
  catalogId: string;
  skillId: string;
  activityPurpose: string;
  lockPurpose: string;
  resultBindings?: PhaseResultBinding[];
  samplesReplanRate?: boolean;
  samplesAcceptanceRate?: boolean;
  autoStartAfter?: string[];
  /**
   * When non-empty, names the sub-mode that executes this phase (phase
   * delegation, `executed_by`). The composed sub-mode flow renders inline.
   */
  executedBy?: string;
  /**
   * The phase's classification-on-transition contract, when one of its outgoing
   * edges derives its routing field at the edge instead of from an emitted
   * field. Absent when every edge routes on a directly-emitted field.
   */
  classification?: OperatingModePhaseClassification;
}

export interface OperatingModeCapabilities {
  supportsPhases: boolean;
  canStartPhases: boolean;
  canCompleteItems: boolean;
  canApplyBacklogSyncProposals: boolean;
  requiresAcceptanceCriteria: boolean;
  supportsArtifacts: boolean;
  supportsHandoffs: boolean;
  usesItemExecutionFlow: boolean;
}

export interface OperatingModeCatalogEntry {
  mode: InitiativeOperatingMode;
  label: string;
  description?: string;
  bestFor: string[];
  notFor: string[];
  tradeoffs: string[];
  whenInDoubtPickInstead?: InitiativeOperatingMode;
  usageCount: number;
  /** The mode's declared unit of work: plan-manager-plan | plan-ref | initiative. */
  targetKind: string;
  runStrategy: string;
  workspaceTabId: string;
  capabilities: OperatingModeCapabilities;
  default: boolean;
  switchable: boolean;
  supportsPhases: boolean;
  phases: OperatingModeCatalogPhase[];
  phaseGraph?: OperatingModePhaseGraph;
}

export interface OperatingModeCatalog {
  modes: OperatingModeCatalogEntry[];
}

export interface OperatingModeLinkedInitiative {
  name: string;
  title: string;
  status?: string;
  updated?: string;
}

export interface OperatingModeDetail {
  entry: OperatingModeCatalogEntry;
  linkedInitiatives: OperatingModeLinkedInitiative[];
}

export interface UpdateOperatingModeArgs {
  label?: string;
  description?: string;
}

export interface OperatingModeWorkspaceDefinition {
  mode: InitiativeOperatingMode;
  label: string;
  description?: string;
  /** The mode's declared unit of work: plan-manager-plan | plan-ref | initiative. */
  targetKind: string;
  capabilities: OperatingModeCapabilities;
  phases: OperatingModeWorkspacePhase[];
  terminal: string[];
  transitions: Record<string, string[]>;
  runStrategy: string;
}

export interface OperatingModeRoundItem {
  ref: string;
  title?: string;
  status?: string;
  priority?: number;
  effort?: string;
}

export interface OperatingModeArtifactUpdate {
  path: string;
  contentType?: string;
  required?: boolean;
  updatedAt?: string;
  source?: string;
}

export interface OperatingModeHandoff {
  summary?: string;
  completedPhases?: string[];
  changedFiles?: string[];
  tests?: string[];
  blockers?: string[];
  nextStep?: string;
  createdAt?: string;
  /** The declared "true frontier" the next round should advance (elastic slice). */
  frontier?: string;
}

export interface OperatingModePhaseResolutionRecord {
  outcome: string;
  layer?: string;
  chosenMessageIndex?: number;
  messagesScanned?: number;
  missing?: string[];
  violations?: string[];
  notes?: string[];
  /**
   * When non-empty, marks this record as a classification-on-transition
   * derivation (rather than a phase-output resolution): the routing field the
   * transition derived and the derived value the route guards matched on.
   */
  classifiedField?: string;
  classifiedValue?: string;
  selectedMessage?: OperatingModeSelectedMessageProvenance;
}

export interface OperatingModeSelectedMessageProvenance {
  eventId?: string;
  sequence?: number;
  contentDigest: string;
  selectionAlgorithmVersion: string;
  fallbackReason?: string;
}

export interface OperatingModeRound {
  round: number;
  mode: InitiativeOperatingMode;
  scopeKind: string;
  scopeId: string;
  initiativeName?: string;
  phase: string;
  runStrategy: string;
  agentProfileKey: string;
  generatedAt: string;
  runId?: string;
  status: OperatingModeRoundStatus;
  items?: OperatingModeRoundItem[];
  artifactUpdates?: OperatingModeArtifactUpdate[];
  handoffs?: OperatingModeHandoff[];
  payload?: Record<string, unknown>;
  /** Canonical validated output used for persisted guard evaluation. */
  resolvedEnvelope?: Record<string, unknown>;
  error?: string;
  resolution?: OperatingModePhaseResolutionRecord;
  /**
   * Classification-on-transition outcome: how the round's routing field was
   * derived at the edge (or why derivation abstained and the round parked in
   * needs_attention). Absent when the completed phase's transitions declare no
   * classification contract.
   */
  transitionClassification?: OperatingModePhaseResolutionRecord;
}

export interface OperatingModeBacklogSyncPlan {
  completedItems?: string[];
  createdItems?: string[];
  updatedItems?: string[];
  proposal?: Proposal;
  rationale?: string;
}

export interface OperatingModeProgressState {
  decision: string;
  completedPhases?: string[];
  currentPhase?: string;
  rationale?: string;
  updatedAt?: string;
}

export interface OperatingModePhaseResult {
  artifacts?: Array<{
    path: string;
    content: string;
    contentType?: string;
  }>;
  handoff?: OperatingModeHandoff;
  handoffs?: OperatingModeHandoff[];
  readiness?: Record<string, unknown>;
  progress?: OperatingModeProgressState;
  verdict?: string;
  replanNeeded?: boolean;
  backlogSync?: OperatingModeBacklogSyncPlan;
}

export interface OperatingModeSimulationInputs {
  initiative: {
    name: string;
    title: string;
    description?: string;
    mode: InitiativeOperatingMode;
    items: string[];
    acceptanceCriteria: string[];
  };
  items: OperatingModeRoundItem[];
  artifacts: OperatingModeArtifactSnapshot[];
  priorRounds: OperatingModeRound[];
  acceptanceCriteria: string[];
}

export interface OperatingModeSimulationTransition {
  from: string;
  to?: string;
  conditionKind: OperatingModeTransitionConditionKind;
  label: string;
  field?: string;
  value?: string;
}

export interface OperatingModeSimulationStep {
  index: number;
  phase: string;
  phaseKind: OperatingModePhaseKind | "";
  inputs: OperatingModeSimulationInputs;
  output: OperatingModePhaseResult;
  round: OperatingModeRound;
  transition?: OperatingModeSimulationTransition;
  terminal?: boolean;
  /** Agent skill this phase would spawn. */
  skillId?: string;
  /** Agent profile this phase would spawn. */
  profileKey?: string;
  /**
   * Resolved prompt template variables for this step, computed server-side
   * without a render call. Used to show what was substituted (and as the
   * Instructions fallback when the render endpoint is unavailable).
   */
  promptVariables?: Record<string, string>;
}

/**
 * Operator-facing metadata for one deterministic simulation scenario. Presets
 * seed different phase outputs to exercise real transition branches (replan,
 * continue, blocked, review, reconcile); they never bypass the registered
 * phase graph. Mirrors `operatingmode.SimulationPreset` on the backend.
 */
export interface OperatingModeSimulationPreset {
  id: string;
  label: string;
  description: string;
  /** One-line "what branch this demonstrates". */
  branch: string;
  /** Longer work-story note shown in the guide/help surface. */
  scenario: string;
}

export interface OperatingModeSimulation {
  mode: InitiativeOperatingMode;
  label: string;
  presets: OperatingModeSimulationPreset[];
  activePreset: string;
  initiative: OperatingModeSimulationInputs["initiative"];
  trace: OperatingModeSimulationStep[];
}

/**
 * A literal, data-filled agent prompt rendered without spawning — the same
 * prompt an agent would receive for a phase. Returned by the simulation and
 * live render endpoints. When `degraded` is true the prompt-manager seam was
 * unavailable, so `prompt` is empty and callers fall back to `variables`.
 * Mirrors `operatingmode.RenderPromptResponse` on the backend.
 */
export interface OperatingModeRenderedPrompt {
  mode: string;
  preset?: string;
  stepIndex?: number;
  phase: string;
  skillId: string;
  profileKey: string;
  variables: Record<string, string>;
  prompt: string;
  degraded: boolean;
  degradedReason?: string;
}

export interface OperatingModeLockHolder {
  run_id?: string;
  runId?: string;
  purpose?: string;
  round_number?: number;
  roundNumber?: number;
  acquired_by?: string;
  acquiredBy?: string;
  acquired_at?: string;
  acquiredAt?: string;
}

export interface OperatingModeWorkspace {
  initiativeName: string;
  mode: InitiativeOperatingMode;
  definition: OperatingModeWorkspaceDefinition;
  lock?: OperatingModeLockHolder;
  artifacts: OperatingModeArtifactSnapshot[];
  rounds: OperatingModeRound[];
}

export interface ActiveItemExecution {
  itemRef: string;
  executionId?: string;
  runId?: string;
  status?: string;
}

export interface SwitchOperatingModeResult {
  initiativeName: string;
  fromMode: InitiativeOperatingMode;
  toMode: InitiativeOperatingMode;
  canceledItemExecutions?: ActiveItemExecution[];
  activeItemExecutions?: ActiveItemExecution[];
  requiresCancellation?: boolean;
  operatingModeWorkspaceId?: string;
}

export interface OperatingModeCompletedItem {
  itemRef: string;
  fromStatus: string;
  toStatus: string;
}

export interface OperatingModeBacklogSyncResult {
  initiativeName: string;
  mode: InitiativeOperatingMode;
  phase: string;
  round: number;
  runId?: string;
  completedItems: OperatingModeCompletedItem[];
  proposalResult?: {
    applied: number;
    failed: number;
    skipped: number;
    created?: number;
    updated?: number;
    outcomes?: Array<{
      mutationId: string;
      op: string;
      target?: string;
      applied: boolean;
      skipped?: boolean;
      error?: string;
    }>;
  };
  noop?: boolean;
}
