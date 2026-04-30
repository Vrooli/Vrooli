import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  OperatingModeArtifactSnapshot,
  OperatingModeRound,
  OperatingModeRoundItem,
  OperatingModeWorkspace,
  OperatingModeWorkspaceDefinition,
  OperatingModeWorkspacePhase,
  SwitchOperatingModeResult,
} from "../types/operating-mode";
import type { InitiativeOperatingMode } from "../types";

type RawRecord = Record<string, unknown>;

function stringValue(value: unknown, fallback = ""): string {
  return typeof value === "string" ? value : fallback;
}

function numberValue(value: unknown, fallback?: number): number | undefined {
  return typeof value === "number" ? value : fallback;
}

function boolValue(value: unknown, fallback?: boolean): boolean | undefined {
  return typeof value === "boolean" ? value : fallback;
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}

function recordValue(value: unknown): RawRecord {
  return value && typeof value === "object" && !Array.isArray(value) ? value as RawRecord : {};
}

function normalizeArtifact(raw: unknown): OperatingModeArtifactSnapshot {
  const item = recordValue(raw);
  return {
    path: stringValue(item.path),
    contentType: stringValue(item.content_type ?? item.contentType, undefined),
    required: boolValue(item.required, undefined),
    content: stringValue(item.content, undefined),
    updatedAt: stringValue(item.updated_at ?? item.updatedAt, undefined),
    sizeBytes: numberValue(item.size_bytes ?? item.sizeBytes, undefined),
  };
}

function normalizePhase(raw: unknown): OperatingModeWorkspacePhase {
  const phase = recordValue(raw);
  const artifacts = phase.output_artifacts ?? phase.outputArtifacts;
  return {
    phase: stringValue(phase.phase),
    activityPurpose: stringValue(phase.activity_purpose ?? phase.activityPurpose),
    profileKey: stringValue(phase.profile_key ?? phase.profileKey),
    writesRepo: boolValue(phase.writes_repo ?? phase.writesRepo) ?? false,
    outputArtifacts: Array.isArray(artifacts)
      ? artifacts.map(normalizeArtifact)
      : [],
    requiresCriteria: boolValue(phase.requires_criteria ?? phase.requiresCriteria),
  };
}

function normalizeDefinition(raw: unknown): OperatingModeWorkspaceDefinition {
  const def = recordValue(raw);
  return {
    mode: stringValue(def.mode, "item-level") as OperatingModeWorkspaceDefinition["mode"],
    label: stringValue(def.label),
    scopeKind: stringValue(def.scope_kind ?? def.scopeKind),
    phases: Array.isArray(def.phases) ? def.phases.map(normalizePhase) : [],
    terminal: stringArray(def.terminal),
    transitions: recordValue(def.transitions) as Record<string, string[]>,
    runStrategy: stringValue(def.run_strategy ?? def.runStrategy),
  };
}

function normalizeRound(raw: unknown): OperatingModeRound {
  const round = recordValue(raw);
  const artifactUpdates = round.artifact_updates ?? round.artifactUpdates;
  return {
    round: numberValue(round.round, 0) ?? 0,
    mode: stringValue(round.mode, "item-level") as OperatingModeRound["mode"],
    scopeKind: stringValue(round.scope_kind ?? round.scopeKind),
    scopeId: stringValue(round.scope_id ?? round.scopeId),
    initiativeName: stringValue(round.initiative_name ?? round.initiativeName, undefined),
    phase: stringValue(round.phase),
    runStrategy: stringValue(round.run_strategy ?? round.runStrategy),
    agentProfileKey: stringValue(round.agent_profile_key ?? round.agentProfileKey),
    generatedAt: stringValue(round.generated_at ?? round.generatedAt),
    runId: stringValue(round.run_id ?? round.runId, undefined),
    status: stringValue(round.status, "reserved") as OperatingModeRound["status"],
    items: Array.isArray(round.items) ? round.items.map((item): OperatingModeRoundItem => {
      const rawItem = recordValue(item);
      return {
        ref: stringValue(rawItem.ref),
        title: stringValue(rawItem.title, undefined),
        status: stringValue(rawItem.status, undefined),
        priority: numberValue(rawItem.priority, undefined),
        effort: stringValue(rawItem.effort, undefined),
      };
    }) : [],
    artifactUpdates: Array.isArray(artifactUpdates)
      ? artifactUpdates.map(normalizeArtifact)
      : [],
    handoffs: Array.isArray(round.handoffs) ? round.handoffs.map((item) => {
      const handoff = recordValue(item);
      return {
        summary: stringValue(handoff.summary, undefined),
        completedPhases: stringArray(handoff.completed_phases ?? handoff.completedPhases),
        changedFiles: stringArray(handoff.changed_files ?? handoff.changedFiles),
        tests: stringArray(handoff.tests),
        blockers: stringArray(handoff.blockers),
        nextStep: stringValue(handoff.next_step ?? handoff.nextStep, undefined),
        createdAt: stringValue(handoff.created_at ?? handoff.createdAt, undefined),
      };
    }) : [],
    payload: recordValue(round.payload),
    error: stringValue(round.error, undefined),
  };
}

function normalizeWorkspace(raw: unknown): OperatingModeWorkspace {
  const workspace = recordValue(raw);
  return {
    initiativeName: stringValue(workspace.initiative_name ?? workspace.initiativeName),
    mode: stringValue(workspace.mode, "item-level") as OperatingModeWorkspace["mode"],
    definition: normalizeDefinition(workspace.definition),
    lock: workspace.lock ? recordValue(workspace.lock) : undefined,
    artifacts: Array.isArray(workspace.artifacts) ? workspace.artifacts.map(normalizeArtifact) : [],
    rounds: Array.isArray(workspace.rounds) ? workspace.rounds.map(normalizeRound) : [],
  };
}

function normalizeSwitchResult(raw: unknown): SwitchOperatingModeResult {
  const result = recordValue(raw);
  const normalizeExecution = (item: unknown) => {
    const exec = recordValue(item);
    return {
      itemRef: stringValue(exec.item_ref ?? exec.itemRef),
      executionId: stringValue(exec.execution_id ?? exec.executionId, undefined),
      runId: stringValue(exec.run_id ?? exec.runId, undefined),
      status: stringValue(exec.status, undefined),
    };
  };
  const active = result.active_item_executions ?? result.activeItemExecutions;
  const canceled = result.canceled_item_executions ?? result.canceledItemExecutions;
  return {
    initiativeName: stringValue(result.initiative_name ?? result.initiativeName),
    fromMode: stringValue(result.from_mode ?? result.fromMode, "item-level") as InitiativeOperatingMode,
    toMode: stringValue(result.to_mode ?? result.toMode, "item-level") as InitiativeOperatingMode,
    activeItemExecutions: Array.isArray(active) ? active.map(normalizeExecution) : undefined,
    canceledItemExecutions: Array.isArray(canceled) ? canceled.map(normalizeExecution) : undefined,
    requiresCancellation: boolValue(result.requires_cancellation ?? result.requiresCancellation, undefined),
    operatingModeWorkspaceId: stringValue(result.operating_mode_workspace_id ?? result.operatingModeWorkspaceId, undefined),
  };
}

export interface StartOperatingModePhaseArgs {
  note?: string;
  override?: boolean;
  requestedBy?: string;
}

export interface SwitchOperatingModeArgs {
  mode: InitiativeOperatingMode;
  cancelActiveItemExecutions?: boolean;
  requestedBy?: string;
}

export interface IInitiativeModeService {
  workspace(name: string): Promise<OperatingModeWorkspace>;
  switchMode(name: string, args: SwitchOperatingModeArgs): Promise<SwitchOperatingModeResult>;
  startPhase(name: string, phase: string, args?: StartOperatingModePhaseArgs): Promise<OperatingModeRound>;
  refreshRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
  cancelRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
}

export function createInitiativeModeService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeModeService {
  return {
    async workspace(name: string): Promise<OperatingModeWorkspace> {
      const raw = await apiClient.get<unknown>(API_ENDPOINTS.initiativeOperatingModeWorkspace(name));
      return normalizeWorkspace(raw);
    },

    async switchMode(name: string, args: SwitchOperatingModeArgs): Promise<SwitchOperatingModeResult> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeSwitch(name),
        {
          mode: args.mode,
          cancel_active_item_executions: args.cancelActiveItemExecutions ?? false,
          requested_by: args.requestedBy ?? "swarm-manager-ui",
        },
      );
      return normalizeSwitchResult(raw);
    },

    async startPhase(name: string, phase: string, args: StartOperatingModePhaseArgs = {}): Promise<OperatingModeRound> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeStartPhase(name, phase),
        {
          note: args.note ?? "",
          override: args.override ?? false,
          requested_by: args.requestedBy ?? "swarm-manager-ui",
        },
      );
      return normalizeRound(raw);
    },

    async refreshRound(name: string, mode: string, round: number): Promise<OperatingModeRound> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeRefreshRound(name, round, mode),
        {},
      );
      return normalizeRound(raw);
    },

    async cancelRound(name: string, mode: string, round: number): Promise<OperatingModeRound> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeCancelRound(name, round, mode),
        {},
      );
      return normalizeRound(raw);
    },
  };
}

export const initiativeModeService = createInitiativeModeService();
