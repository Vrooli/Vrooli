import type { IApiClient } from "../lib/api-client";
import { defaultApiClient, isApiError } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  ActiveItemExecution,
  OperatingModeArtifactDefinition,
  OperatingModeArtifactSnapshot,
  OperatingModeBacklogSyncResult,
  OperatingModeCatalog,
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModeCapabilities,
  OperatingModeDetail,
  OperatingModeLinkedInitiative,
  OperatingModePhaseGraph,
  OperatingModePhaseTransition,
  OperatingModeRound,
  OperatingModeRoundItem,
  OperatingModeTransitionConditionKind,
  OperatingModeWorkspace,
  OperatingModeWorkspaceDefinition,
  OperatingModeWorkspacePhase,
  PhaseOutputContractSummary,
  PhaseResultBinding,
  SwitchOperatingModeResult,
  UpdateOperatingModeArgs,
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
    startable: boolValue(phase.startable) ?? false,
    reason: stringValue(phase.reason, undefined),
    next: boolValue(phase.next),
  };
}

function normalizeArtifactDefinition(raw: unknown): OperatingModeArtifactDefinition {
  const item = recordValue(raw);
  return {
    path: stringValue(item.path),
    contentType: stringValue(item.content_type ?? item.contentType, undefined),
    required: boolValue(item.required, undefined),
  };
}

function normalizeContractSummary(raw: unknown): PhaseOutputContractSummary {
  const contract = recordValue(raw);
  return {
    requiresStructuredResult:
      boolValue(contract.requires_structured_result ?? contract.requiresStructuredResult) ?? false,
    requiresProgress: boolValue(contract.requires_progress ?? contract.requiresProgress) ?? false,
    requiresVerdict: boolValue(contract.requires_verdict ?? contract.requiresVerdict) ?? false,
    requiresHandoff: boolValue(contract.requires_handoff ?? contract.requiresHandoff) ?? false,
    requiredArtifactCount:
      numberValue(contract.required_artifact_count ?? contract.requiredArtifactCount, 0) ?? 0,
  };
}

function normalizeResultBinding(raw: unknown): PhaseResultBinding {
  const binding = recordValue(raw);
  return {
    kind: "progress_artifact",
    artifact: normalizeArtifactDefinition(binding.artifact),
  };
}

const TRANSITION_KINDS: ReadonlySet<OperatingModeTransitionConditionKind> = new Set([
  "always",
  "payload_bool",
  "progress_decision",
]);

function normalizeTransitionKind(raw: unknown): OperatingModeTransitionConditionKind {
  const value = stringValue(raw);
  return TRANSITION_KINDS.has(value as OperatingModeTransitionConditionKind)
    ? (value as OperatingModeTransitionConditionKind)
    : "always";
}

function normalizeTransition(raw: unknown): OperatingModePhaseTransition {
  const edge = recordValue(raw);
  return {
    from: stringValue(edge.from),
    to: stringValue(edge.to),
    conditionKind: normalizeTransitionKind(edge.condition_kind ?? edge.conditionKind),
    label: stringValue(edge.label),
    payloadKey: stringValue(edge.payload_key ?? edge.payloadKey, undefined),
    progressDecision: stringValue(edge.progress_decision ?? edge.progressDecision, undefined),
  };
}

function normalizePhaseGraph(raw: unknown): OperatingModePhaseGraph | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  const graph = recordValue(raw);
  const startPhase = stringValue(graph.start_phase ?? graph.startPhase);
  if (!startPhase) return undefined;
  const transitions = graph.transitions;
  return {
    startPhase,
    terminal: stringArray(graph.terminal),
    transitions: Array.isArray(transitions) ? transitions.map(normalizeTransition) : [],
    acceptedVerdicts: stringArray(graph.accepted_verdicts ?? graph.acceptedVerdicts),
  };
}

function normalizeCatalogPhase(raw: unknown): OperatingModeCatalogPhase {
  const phase = recordValue(raw);
  const artifacts = phase.output_artifacts ?? phase.outputArtifacts;
  const bindings = phase.result_bindings ?? phase.resultBindings;
  return {
    phase: stringValue(phase.phase),
    label: stringValue(phase.label),
    title: stringValue(phase.title),
    purpose: stringValue(phase.purpose),
    trigger: stringValue(phase.trigger),
    profileKey: stringValue(phase.profile_key ?? phase.profileKey),
    writesRepo: boolValue(phase.writes_repo ?? phase.writesRepo) ?? false,
    requiresCriteria: boolValue(phase.requires_criteria ?? phase.requiresCriteria),
    isStart: boolValue(phase.is_start ?? phase.isStart),
    isTerminal: boolValue(phase.is_terminal ?? phase.isTerminal),
    outputArtifacts: Array.isArray(artifacts)
      ? artifacts.map(normalizeArtifactDefinition)
      : undefined,
    outputContract: normalizeContractSummary(phase.output_contract ?? phase.outputContract),
    catalogId: stringValue(phase.catalog_id ?? phase.catalogId),
    skillId: stringValue(phase.skill_id ?? phase.skillId),
    activityPurpose: stringValue(phase.activity_purpose ?? phase.activityPurpose),
    lockPurpose: stringValue(phase.lock_purpose ?? phase.lockPurpose),
    resultBindings: Array.isArray(bindings) ? bindings.map(normalizeResultBinding) : undefined,
    samplesReplanRate: boolValue(phase.samples_replan_rate ?? phase.samplesReplanRate),
    samplesAcceptanceRate: boolValue(phase.samples_acceptance_rate ?? phase.samplesAcceptanceRate),
  };
}

function normalizeCapabilities(raw: unknown, fallbackSupportsPhases = false): OperatingModeCapabilities {
  const capabilities = recordValue(raw);
  return {
    supportsPhases: boolValue(capabilities.supports_phases ?? capabilities.supportsPhases) ?? fallbackSupportsPhases,
    canStartPhases: boolValue(capabilities.can_start_phases ?? capabilities.canStartPhases) ?? false,
    canCompleteItems: boolValue(capabilities.can_complete_items ?? capabilities.canCompleteItems) ?? false,
    canApplyBacklogSyncProposals: boolValue(capabilities.can_apply_backlog_sync_proposals ?? capabilities.canApplyBacklogSyncProposals) ?? false,
    requiresAcceptanceCriteria: boolValue(capabilities.requires_acceptance_criteria ?? capabilities.requiresAcceptanceCriteria) ?? false,
    supportsArtifacts: boolValue(capabilities.supports_artifacts ?? capabilities.supportsArtifacts) ?? false,
    supportsHandoffs: boolValue(capabilities.supports_handoffs ?? capabilities.supportsHandoffs) ?? false,
    usesItemExecutionFlow: boolValue(capabilities.uses_item_execution_flow ?? capabilities.usesItemExecutionFlow) ?? false,
  };
}

function normalizeCatalogEntry(raw: unknown): OperatingModeCatalogEntry {
  const mode = recordValue(raw);
  const phases = mode.phases;
  const supportsPhases = boolValue(mode.supports_phases ?? mode.supportsPhases) ?? false;
  return {
    mode: stringValue(mode.mode, "item-level"),
    label: stringValue(mode.label),
    description: stringValue(mode.description, undefined),
    usageCount: numberValue(mode.usage_count ?? mode.usageCount, 0) ?? 0,
    scopeKind: stringValue(mode.scope_kind ?? mode.scopeKind),
    runStrategy: stringValue(mode.run_strategy ?? mode.runStrategy),
    workspaceTabId: stringValue(mode.workspace_tab_id ?? mode.workspaceTabId),
    capabilities: normalizeCapabilities(mode.capabilities, supportsPhases),
    default: boolValue(mode.default) ?? false,
    switchable: boolValue(mode.switchable) ?? false,
    supportsPhases,
    phases: Array.isArray(phases) ? phases.map(normalizeCatalogPhase) : [],
    phaseGraph: normalizePhaseGraph(mode.phase_graph ?? mode.phaseGraph),
  };
}

function normalizeLinkedInitiative(raw: unknown): OperatingModeLinkedInitiative {
  const item = recordValue(raw);
  return {
    name: stringValue(item.name),
    title: stringValue(item.title),
    status: stringValue(item.status, undefined),
    updated: stringValue(item.updated, undefined),
  };
}

function normalizeModeDetail(raw: unknown): OperatingModeDetail {
  const detail = recordValue(raw);
  const linked = detail.linked_initiatives ?? detail.linkedInitiatives;
  return {
    entry: normalizeCatalogEntry(detail.entry),
    linkedInitiatives: Array.isArray(linked) ? linked.map(normalizeLinkedInitiative) : [],
  };
}

function normalizeCatalog(raw: unknown): OperatingModeCatalog {
  const catalog = recordValue(raw);
  const modes = catalog.modes;
  return {
    modes: Array.isArray(modes) ? modes.map(normalizeCatalogEntry) : [],
  };
}

function normalizeDefinition(raw: unknown): OperatingModeWorkspaceDefinition {
  const def = recordValue(raw);
  return {
    mode: stringValue(def.mode, "item-level"),
    label: stringValue(def.label),
    description: stringValue(def.description, undefined),
    scopeKind: stringValue(def.scope_kind ?? def.scopeKind),
    capabilities: normalizeCapabilities(def.capabilities, Array.isArray(def.phases) && def.phases.length > 0),
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
    mode: stringValue(round.mode, "item-level"),
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
    mode: stringValue(workspace.mode, "item-level"),
    definition: normalizeDefinition(workspace.definition),
    lock: workspace.lock ? recordValue(workspace.lock) : undefined,
    artifacts: Array.isArray(workspace.artifacts) ? workspace.artifacts.map(normalizeArtifact) : [],
    rounds: Array.isArray(workspace.rounds) ? workspace.rounds.map(normalizeRound) : [],
  };
}

function normalizeExecution(item: unknown): ActiveItemExecution {
  const exec = recordValue(item);
  return {
    itemRef: stringValue(exec.item_ref ?? exec.itemRef),
    executionId: stringValue(exec.execution_id ?? exec.executionId, undefined),
    runId: stringValue(exec.run_id ?? exec.runId, undefined),
    status: stringValue(exec.status, undefined),
  };
}

function normalizeSwitchResult(raw: unknown): SwitchOperatingModeResult {
  const result = recordValue(raw);
  const active = result.active_item_executions ?? result.activeItemExecutions;
  const canceled = result.canceled_item_executions ?? result.canceledItemExecutions;
  return {
    initiativeName: stringValue(result.initiative_name ?? result.initiativeName),
    fromMode: stringValue(result.from_mode ?? result.fromMode, "item-level"),
    toMode: stringValue(result.to_mode ?? result.toMode, "item-level"),
    activeItemExecutions: Array.isArray(active) ? active.map(normalizeExecution) : undefined,
    canceledItemExecutions: Array.isArray(canceled) ? canceled.map(normalizeExecution) : undefined,
    requiresCancellation: boolValue(result.requires_cancellation ?? result.requiresCancellation, undefined),
    operatingModeWorkspaceId: stringValue(result.operating_mode_workspace_id ?? result.operatingModeWorkspaceId, undefined),
  };
}

/**
 * Server-side 409 conflict shape returned when SwitchMode is called against
 * an initiative with active item executions and `cancel_active_item_executions`
 * is false. Mirrors `ActiveItemExecutionsConflict` in
 * `scenarios/swarm-manager/api/internal/operatingmode/service.go`.
 */
export interface ActiveItemExecutionsConflict {
  initiativeName: string;
  fromMode: string;
  toMode: string;
  executions: ActiveItemExecution[];
}

/**
 * Detect whether an error came from the server's 409 active-item-executions
 * conflict response. The mode-picker dialog uses this to render the affected
 * items list before re-submitting with cancel_active_item_executions=true.
 *
 * Returns the parsed conflict payload, or `null` for any other error shape.
 */
export function parseActiveItemExecutionsConflict(error: unknown): ActiveItemExecutionsConflict | null {
  if (!isApiError(error)) return null;
  if (error.status !== 409) return null;
  if (error.code !== "active_item_executions") return null;
  const details = recordValue(error.details);
  const executions = details.active_item_executions ?? details.activeItemExecutions;
  if (!Array.isArray(executions)) return null;
  return {
    initiativeName: stringValue(details.initiative_name ?? details.initiativeName),
    fromMode: stringValue(details.from_mode ?? details.fromMode, "item-level"),
    toMode: stringValue(details.to_mode ?? details.toMode, "item-level"),
    executions: executions.map(normalizeExecution),
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

export interface CompleteOperatingModeItemsArgs {
  mode: InitiativeOperatingMode;
  round: number;
  runId: string;
  itemRefs: string[];
  requestedBy?: string;
}

export interface ApplyOperatingModeBacklogSyncArgs {
  mode: InitiativeOperatingMode;
  round: number;
  runId: string;
  acceptedMutationIds?: string[];
  requestedBy?: string;
}

export interface IInitiativeModeService {
  catalog(): Promise<OperatingModeCatalog>;
  getMode(mode: string): Promise<OperatingModeDetail>;
  updateMode(mode: string, args: UpdateOperatingModeArgs): Promise<OperatingModeDetail>;
  workspace(name: string): Promise<OperatingModeWorkspace>;
  switchMode(name: string, args: SwitchOperatingModeArgs): Promise<SwitchOperatingModeResult>;
  startPhase(name: string, phase: string, args?: StartOperatingModePhaseArgs): Promise<OperatingModeRound>;
  refreshRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
  cancelRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
  completeItems(name: string, args: CompleteOperatingModeItemsArgs): Promise<OperatingModeBacklogSyncResult>;
  applyBacklogSync(name: string, args: ApplyOperatingModeBacklogSyncArgs): Promise<OperatingModeBacklogSyncResult>;
}

function normalizeBacklogSyncResult(raw: unknown): OperatingModeBacklogSyncResult {
  const result = recordValue(raw);
  const completed = result.completed_items ?? result.completedItems;
  const proposalResult = recordValue(result.proposal_result ?? result.proposalResult);
  const outcomes = proposalResult.outcomes;
  return {
    initiativeName: stringValue(result.initiative_name ?? result.initiativeName),
    mode: stringValue(result.mode, "item-level"),
    phase: stringValue(result.phase),
    round: numberValue(result.round, 0) ?? 0,
    runId: stringValue(result.run_id ?? result.runId, undefined),
    completedItems: Array.isArray(completed) ? completed.map((item) => {
      const completedItem = recordValue(item);
      return {
        itemRef: stringValue(completedItem.item_ref ?? completedItem.itemRef),
        fromStatus: stringValue(completedItem.from_status ?? completedItem.fromStatus),
        toStatus: stringValue(completedItem.to_status ?? completedItem.toStatus),
      };
    }) : [],
    proposalResult: Object.keys(proposalResult).length > 0 ? {
      applied: numberValue(proposalResult.applied, 0) ?? 0,
      failed: numberValue(proposalResult.failed, 0) ?? 0,
      skipped: numberValue(proposalResult.skipped, 0) ?? 0,
      created: numberValue(proposalResult.created, undefined),
      updated: numberValue(proposalResult.updated, undefined),
      outcomes: Array.isArray(outcomes) ? outcomes.map((item) => {
        const outcome = recordValue(item);
        return {
          mutationId: stringValue(outcome.mutation_id ?? outcome.mutationId),
          op: stringValue(outcome.op),
          target: stringValue(outcome.target, undefined),
          applied: boolValue(outcome.applied) ?? false,
          skipped: boolValue(outcome.skipped, undefined),
          error: stringValue(outcome.error, undefined),
        };
      }) : [],
    } : undefined,
    noop: boolValue(result.noop, undefined),
  };
}

export function createInitiativeModeService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeModeService {
  return {
    async catalog(): Promise<OperatingModeCatalog> {
      const raw = await apiClient.get<unknown>(API_ENDPOINTS.operatingModes);
      return normalizeCatalog(raw);
    },

    async getMode(mode: string): Promise<OperatingModeDetail> {
      const raw = await apiClient.get<unknown>(API_ENDPOINTS.operatingMode(mode));
      return normalizeModeDetail(raw);
    },

    async updateMode(mode: string, args: UpdateOperatingModeArgs): Promise<OperatingModeDetail> {
      const body: Record<string, unknown> = {};
      if (args.label !== undefined) body.label = args.label;
      if (args.description !== undefined) body.description = args.description;
      const raw = await apiClient.patch<unknown>(API_ENDPOINTS.operatingMode(mode), body);
      return normalizeModeDetail(raw);
    },

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

    async completeItems(name: string, args: CompleteOperatingModeItemsArgs): Promise<OperatingModeBacklogSyncResult> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeCompleteItems(name, args.round, args.mode),
        {
          mode: args.mode,
          run_id: args.runId,
          item_refs: args.itemRefs,
          requested_by: args.requestedBy ?? "swarm-manager-ui",
        },
      );
      return normalizeBacklogSyncResult(raw);
    },

    async applyBacklogSync(name: string, args: ApplyOperatingModeBacklogSyncArgs): Promise<OperatingModeBacklogSyncResult> {
      const raw = await apiClient.post<unknown>(
        API_ENDPOINTS.initiativeOperatingModeApplyBacklogSync(name, args.round, args.mode),
        {
          mode: args.mode,
          run_id: args.runId,
          accepted_mutation_ids: args.acceptedMutationIds ?? [],
          requested_by: args.requestedBy ?? "swarm-manager-ui",
        },
      );
      return normalizeBacklogSyncResult(raw);
    },
  };
}

export const initiativeModeService = createInitiativeModeService();
