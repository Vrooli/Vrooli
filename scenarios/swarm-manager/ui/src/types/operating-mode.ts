import type { InitiativeOperatingMode } from "./initiative";
import type { Proposal } from "./feedback";

export type OperatingModeRoundStatus =
  | "reserved"
  | "agent_running"
  | "completed"
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

export type OperatingModeTransitionConditionKind =
  | "always"
  | "payload_bool"
  | "progress_decision";

export interface OperatingModePhaseTransition {
  from: string;
  to: string;
  conditionKind: OperatingModeTransitionConditionKind;
  label: string;
  payloadKey?: string;
  progressDecision?: string;
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
  outputContract: PhaseOutputContractSummary;
  catalogId: string;
  skillId: string;
  activityPurpose: string;
  lockPurpose: string;
  resultBindings?: PhaseResultBinding[];
  samplesReplanRate?: boolean;
  samplesAcceptanceRate?: boolean;
  autoStartAfter?: string[];
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
  scopeKind: string;
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
  scopeKind: string;
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
  error?: string;
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
  payloadKey?: string;
  progressDecision?: string;
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
