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

export interface OperatingModeWorkspacePhase {
  phase: string;
  activityPurpose: string;
  profileKey: string;
  writesRepo: boolean;
  outputArtifacts?: OperatingModeArtifactDefinition[];
  requiresCriteria?: boolean;
  startable: boolean;
  reason?: string;
  next?: boolean;
}

export interface OperatingModeWorkspaceDefinition {
  mode: InitiativeOperatingMode;
  label: string;
  scopeKind: string;
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
