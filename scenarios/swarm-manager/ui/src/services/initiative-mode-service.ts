// Operating-mode service — the UI's typed client for the swarm-manager
// operating-mode subsystem. Talks Proto + Connect-RPC end-to-end via the
// generated `OperatingModeService` client (mirroring `../api/discovery.ts`),
// so the browser consumes the same wire contract the CLI and backend do — no
// hand-rolled JSON endpoints or snake_case normalization. The mappers below
// project the generated proto messages onto the hand-authored domain types the
// pages/components already consume (kept stable so their tests, which mock at
// the domain level, are untouched).

import { createClient, ConnectError, Code, type Client } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import {
  OperatingModeService,
  OperatingModeActiveItemExecutionsConflictSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/operating_mode_pb";
import type * as ompb from "@vrooli/proto-types/swarm-manager/v1/api/operating_mode_pb";
import { API_BASE } from "../lib/api-client";
import type {
  ActiveItemExecution,
  OperatingModeArtifactDefinition,
  OperatingModeArtifactSnapshot,
  OperatingModeArtifactUpdate,
  OperatingModeBacklogSyncPlan,
  OperatingModeBacklogSyncResult,
  OperatingModeCapabilities,
  OperatingModeCatalog,
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModeDetail,
  OperatingModeHandoff,
  OperatingModeLinkedInitiative,
  OperatingModePhaseClassification,
  OperatingModePhaseGraph,
  OperatingModePhaseKind,
  OperatingModePhaseReads,
  OperatingModePhaseResolutionRecord,
  OperatingModePhaseResult,
  OperatingModePhaseTransition,
  OperatingModeProgressState,
  OperatingModeRenderedPrompt,
  OperatingModeRound,
  OperatingModeRoundItem,
  OperatingModeRoundStatus,
  OperatingModeSimulation,
  OperatingModeSimulationInputs,
  OperatingModeSimulationPreset,
  OperatingModeSimulationStep,
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

export type OperatingModeClient = Client<typeof OperatingModeService>;

// ---------------------------------------------------------------------------
// Scalar coercion helpers
// ---------------------------------------------------------------------------

// Proto3 scalar fields are always present (zero value when unset); the domain
// types treat "" as absent for a number of optional string fields. orUndef
// re-establishes that distinction where the domain type is optional.
function orUndef(value: string | undefined): string | undefined {
  return value ? value : undefined;
}

function asMode(value: string | undefined): InitiativeOperatingMode {
  return value || "item-level";
}

const PHASE_KINDS: ReadonlySet<OperatingModePhaseKind> = new Set([
  "investigate",
  "execute",
  "review",
  "reconcile",
]);

function mapPhaseKind(raw: string | undefined): OperatingModePhaseKind | "" {
  return raw && PHASE_KINDS.has(raw as OperatingModePhaseKind)
    ? (raw as OperatingModePhaseKind)
    : "";
}

function mapConditionKind(raw: string | undefined): OperatingModeTransitionConditionKind {
  // The wire carries the generic guard op (eq/ne/exists/all/… or `always`);
  // an empty condition is the unconditional default edge.
  return (raw || "always") as OperatingModeTransitionConditionKind;
}

// ---------------------------------------------------------------------------
// Projection mappers (proto message -> domain type)
//
// Every mapper accepts `T | undefined` and defaults so a partially-populated
// message (or a nested optional the server omitted) never throws — the same
// defensive posture the previous JSON normalizers took, which also keeps the
// unit tests able to pass sparse fixtures.
// ---------------------------------------------------------------------------

function mapArtifactDef(a: ompb.OperatingModeArtifactDefinition | undefined): OperatingModeArtifactDefinition {
  return {
    path: a?.path ?? "",
    contentType: orUndef(a?.contentType),
    required: a?.required,
  };
}

function mapArtifactSnapshot(a: ompb.OperatingModeArtifactSnapshot | undefined): OperatingModeArtifactSnapshot {
  return {
    path: a?.path ?? "",
    contentType: orUndef(a?.contentType),
    required: a?.required,
    content: orUndef(a?.content),
    updatedAt: orUndef(a?.updatedAt),
    sizeBytes: a?.sizeBytes ? Number(a.sizeBytes) : undefined,
  };
}

function mapArtifactUpdate(u: ompb.OperatingModeArtifactUpdate | undefined): OperatingModeArtifactUpdate {
  return {
    path: u?.path ?? "",
    contentType: orUndef(u?.contentType),
    required: u?.required,
    updatedAt: orUndef(u?.updatedAt),
    source: orUndef(u?.source),
  };
}

function mapCapabilities(
  c: ompb.OperatingModeCapabilities | undefined,
  fallbackSupportsPhases: boolean,
): OperatingModeCapabilities {
  return {
    supportsPhases: c?.supportsPhases ?? fallbackSupportsPhases,
    canStartPhases: c?.canStartPhases ?? false,
    canCompleteItems: c?.canCompleteItems ?? false,
    canApplyBacklogSyncProposals: c?.canApplyBacklogSyncProposals ?? false,
    requiresAcceptanceCriteria: c?.requiresAcceptanceCriteria ?? false,
    supportsArtifacts: c?.supportsArtifacts ?? false,
    supportsHandoffs: c?.supportsHandoffs ?? false,
    usesItemExecutionFlow: c?.usesItemExecutionFlow ?? false,
  };
}

function mapContractSummary(
  c: ompb.OperatingModePhaseOutputContractSummary | undefined,
): PhaseOutputContractSummary {
  return {
    requiresStructuredResult: c?.requiresStructuredResult ?? false,
    requiresProgress: c?.requiresProgress ?? false,
    requiresVerdict: c?.requiresVerdict ?? false,
    requiresHandoff: c?.requiresHandoff ?? false,
    requiresBacklogSync: c?.requiresBacklogSync ?? false,
    requiredArtifactCount: c?.requiredArtifactCount ?? 0,
  };
}

function mapResultBinding(b: ompb.OperatingModeResultBindingSummary | undefined): PhaseResultBinding {
  return {
    kind: "progress_artifact",
    artifact: mapArtifactDef(b?.artifact),
  };
}

function mapTransition(
  t: ompb.OperatingModeCatalogTransition | undefined,
): OperatingModePhaseTransition {
  return {
    from: t?.from ?? "",
    to: t?.to ?? "",
    conditionKind: mapConditionKind(t?.conditionKind),
    label: t?.label ?? "",
    field: orUndef(t?.field),
    value: orUndef(t?.value),
    classified: t?.classified ? true : undefined,
  };
}

function mapPhaseGraph(
  g: ompb.OperatingModeCatalogPhaseGraph | undefined,
): OperatingModePhaseGraph | undefined {
  if (!g || !g.startPhase) return undefined;
  return {
    startPhase: g.startPhase,
    terminal: g.terminal ?? [],
    transitions: (g.transitions ?? []).map(mapTransition),
    acceptedVerdicts: g.acceptedVerdicts ?? [],
  };
}

function mapCatalogPhase(p: ompb.OperatingModeCatalogPhase | undefined): OperatingModeCatalogPhase {
  return {
    phase: p?.phase ?? "",
    phaseKind: mapPhaseKind(p?.phaseKind),
    label: p?.label ?? "",
    title: p?.title ?? "",
    purpose: p?.purpose ?? "",
    trigger: p?.trigger ?? "",
    profileKey: p?.profileKey ?? "",
    writesRepo: p?.writesRepo ?? false,
    requiresCriteria: p?.requiresCriteria,
    isStart: p?.isStart,
    isTerminal: p?.isTerminal,
    outputArtifacts: (p?.outputArtifacts ?? []).map(mapArtifactDef),
    reads: mapPhaseReads(p?.reads),
    outputContract: mapContractSummary(p?.outputContract),
    catalogId: p?.catalogId ?? "",
    skillId: p?.skillId ?? "",
    activityPurpose: p?.activityPurpose ?? "",
    lockPurpose: p?.lockPurpose ?? "",
    resultBindings: (p?.resultBindings ?? []).map(mapResultBinding),
    samplesReplanRate: p?.samplesReplanRate,
    samplesAcceptanceRate: p?.samplesAcceptanceRate,
    autoStartAfter: p?.autoStartAfter ?? [],
    executedBy: orUndef(p?.executedBy),
    classification: mapPhaseClassification(p?.classification),
  };
}

function mapPhaseReads(
  r: ompb.OperatingModePhaseReads | undefined,
): OperatingModePhaseReads | undefined {
  if (!r || ((r.base ?? []).length === 0 && (r.target ?? []).length === 0)) return undefined;
  return { base: r.base ?? [], target: r.target ?? [] };
}

function mapPhaseClassification(
  c: ompb.OperatingModeTransitionClassification | undefined,
): OperatingModePhaseClassification | undefined {
  if (!c || !c.field) return undefined;
  return {
    field: c.field,
    enum: c.enum ?? [],
    from: orUndef(c.from),
    description: orUndef(c.description),
  };
}

function mapCatalogEntry(e: ompb.OperatingModeCatalogEntry | undefined): OperatingModeCatalogEntry {
  const supportsPhases = e?.supportsPhases ?? false;
  return {
    mode: asMode(e?.mode),
    label: e?.label ?? "",
    description: orUndef(e?.description),
    bestFor: e?.bestFor ?? [],
    notFor: e?.notFor ?? [],
    tradeoffs: e?.tradeoffs ?? [],
    whenInDoubtPickInstead: e?.whenInDoubtPickInstead ? asMode(e.whenInDoubtPickInstead) : undefined,
    usageCount: e?.usageCount ?? 0,
    targetKind: e?.targetKind ?? "",
    runStrategy: e?.runStrategy ?? "",
    workspaceTabId: e?.workspaceTabId ?? "",
    capabilities: mapCapabilities(e?.capabilities, supportsPhases),
    default: e?.default ?? false,
    switchable: e?.switchable ?? false,
    supportsPhases,
    phases: (e?.phases ?? []).map(mapCatalogPhase),
    phaseGraph: mapPhaseGraph(e?.phaseGraph),
  };
}

function mapCatalog(resp: ompb.OperatingModeCatalogResponse | undefined): OperatingModeCatalog {
  return { modes: (resp?.modes ?? []).map(mapCatalogEntry) };
}

function mapLinkedInitiative(i: ompb.OperatingModeInitiativeRef | undefined): OperatingModeLinkedInitiative {
  return {
    name: i?.name ?? "",
    title: i?.title ?? "",
    status: orUndef(i?.status),
    updated: orUndef(i?.updated),
  };
}

function mapModeDetail(d: ompb.OperatingModeDetailResponse | undefined): OperatingModeDetail {
  return {
    entry: mapCatalogEntry(d?.entry),
    linkedInitiatives: (d?.linkedInitiatives ?? []).map(mapLinkedInitiative),
  };
}

function mapHandoff(h: ompb.OperatingModeHandoff | undefined): OperatingModeHandoff {
  return {
    summary: orUndef(h?.summary),
    completedPhases: h?.completedPhases ?? [],
    changedFiles: h?.changedFiles ?? [],
    tests: h?.tests ?? [],
    blockers: h?.blockers ?? [],
    nextStep: orUndef(h?.nextStep),
    createdAt: orUndef(h?.createdAt),
    frontier: orUndef(h?.frontier),
  };
}

function mapRoundItem(i: ompb.OperatingModeRoundItem | undefined): OperatingModeRoundItem {
  return {
    ref: i?.ref ?? "",
    title: orUndef(i?.title),
    status: orUndef(i?.status),
    priority: i?.priority ? i.priority : undefined,
    effort: orUndef(i?.effort),
  };
}

function mapRound(r: ompb.OperatingModeRoundEnvelope | undefined): OperatingModeRound {
  return {
    round: r?.round ?? 0,
    mode: asMode(r?.mode),
    scopeKind: r?.scopeKind ?? "",
    scopeId: r?.scopeId ?? "",
    initiativeName: orUndef(r?.initiativeName),
    phase: r?.phase ?? "",
    runStrategy: r?.runStrategy ?? "",
    agentProfileKey: r?.agentProfileKey ?? "",
    generatedAt: r?.generatedAt ?? "",
    runId: orUndef(r?.runId),
    status: (r?.status || "reserved") as OperatingModeRoundStatus,
    items: (r?.items ?? []).map(mapRoundItem),
    artifactUpdates: (r?.artifactUpdates ?? []).map(mapArtifactUpdate),
    handoffs: (r?.handoffs ?? []).map(mapHandoff),
    payload: (r?.payload ?? {}) as Record<string, unknown>,
    error: orUndef(r?.error),
    resolution: mapResolution(r?.resolution),
    transitionClassification: mapResolution(r?.transitionClassification),
  };
}

function mapResolution(
  rec: ompb.OperatingModePhaseResolutionRecord | undefined,
): OperatingModePhaseResolutionRecord | undefined {
  if (!rec || !rec.outcome) return undefined;
  return {
    outcome: rec.outcome,
    layer: orUndef(rec.layer),
    chosenMessageIndex: rec.chosenMessageIndex,
    messagesScanned: rec.messagesScanned,
    missing: rec.missing ?? [],
    violations: rec.violations ?? [],
    notes: rec.notes ?? [],
    classifiedField: orUndef(rec.classifiedField),
    classifiedValue: orUndef(rec.classifiedValue),
  };
}

function mapProgress(p: ompb.OperatingModeProgressState | undefined): OperatingModeProgressState | undefined {
  if (!p) return undefined;
  return {
    decision: p.decision ?? "",
    completedPhases: p.completedPhases ?? [],
    currentPhase: orUndef(p.currentPhase),
    rationale: orUndef(p.rationale),
    updatedAt: orUndef(p.updatedAt),
  };
}

function mapBacklogSyncPlan(p: ompb.OperatingModeBacklogSyncPlan | undefined): OperatingModeBacklogSyncPlan | undefined {
  if (!p) return undefined;
  return {
    completedItems: p.completedItems ?? [],
    createdItems: p.createdItems ?? [],
    updatedItems: p.updatedItems ?? [],
    proposal: p.proposal ? (p.proposal as never) : undefined,
    rationale: orUndef(p.rationale),
  };
}

function mapReadiness(r: ompb.OperatingModeReadinessReport | undefined): Record<string, unknown> | undefined {
  if (!r) return undefined;
  const dimensions = r.dimensions ?? [];
  if (dimensions.length === 0 && !r.ready && !r.overallScore) return undefined;
  return {
    dimensions: dimensions.map((d) => ({
      key: d.key,
      label: d.label,
      score: d.score,
      rationale: d.rationale,
    })),
    overallScore: r.overallScore,
    ready: r.ready,
  };
}

function mapPhaseResult(r: ompb.OperatingModePhaseResult | undefined): OperatingModePhaseResult {
  const artifacts = r?.artifacts ?? [];
  return {
    artifacts: artifacts.length
      ? artifacts.map((a) => ({
          path: a.path,
          content: a.content,
          contentType: orUndef(a.contentType),
        }))
      : undefined,
    handoff: r?.handoff ? mapHandoff(r.handoff) : undefined,
    handoffs: r?.handoffs?.length ? r.handoffs.map(mapHandoff) : undefined,
    readiness: mapReadiness(r?.readiness),
    progress: mapProgress(r?.progress),
    verdict: orUndef(r?.verdict),
    replanNeeded: r?.replanNeeded,
    backlogSync: mapBacklogSyncPlan(r?.backlogSync),
  };
}

function mapInitiativeSnapshot(
  s: ompb.OperatingModeInitiativeSnapshot | undefined,
): OperatingModeSimulationInputs["initiative"] {
  return {
    name: s?.name ?? "",
    title: s?.title ?? "",
    description: orUndef(s?.description),
    mode: asMode(s?.mode),
    items: s?.items ?? [],
    acceptanceCriteria: s?.acceptanceCriteria ?? [],
  };
}

function mapSimulationInputs(in_: ompb.OperatingModeSimulationInputs | undefined): OperatingModeSimulationInputs {
  return {
    initiative: mapInitiativeSnapshot(in_?.initiative),
    items: (in_?.items ?? []).map(mapRoundItem),
    artifacts: (in_?.artifacts ?? []).map(mapArtifactSnapshot),
    priorRounds: (in_?.priorRounds ?? []).map(mapRound),
    acceptanceCriteria: in_?.acceptanceCriteria ?? [],
  };
}

function mapSimulationStep(s: ompb.OperatingModeSimulationStep | undefined): OperatingModeSimulationStep {
  const transition = s?.transition;
  const promptVariables = s?.promptVariables ?? {};
  return {
    index: s?.index ?? 0,
    phase: s?.phase ?? "",
    phaseKind: mapPhaseKind(s?.phaseKind),
    inputs: mapSimulationInputs(s?.inputs),
    output: mapPhaseResult(s?.output),
    round: mapRound(s?.round),
    transition: transition
      ? {
          from: transition.from ?? "",
          to: orUndef(transition.to),
          conditionKind: mapConditionKind(transition.conditionKind),
          label: transition.label ?? "",
          field: orUndef(transition.field),
          value: orUndef(transition.value),
        }
      : undefined,
    terminal: s?.terminal,
    skillId: orUndef(s?.skillId),
    profileKey: orUndef(s?.profileKey),
    promptVariables: Object.keys(promptVariables).length ? promptVariables : undefined,
  };
}

function mapSimulationPreset(p: ompb.OperatingModeSimulationPreset | undefined): OperatingModeSimulationPreset {
  return {
    id: p?.id ?? "",
    label: p?.label ?? "",
    description: p?.description ?? "",
    branch: p?.branch ?? "",
    scenario: p?.scenario ?? "",
  };
}

function mapSimulation(s: ompb.OperatingModeSimulationResponse | undefined): OperatingModeSimulation {
  return {
    mode: asMode(s?.mode),
    label: s?.label ?? "",
    presets: (s?.presets ?? []).map(mapSimulationPreset),
    activePreset: s?.activePreset ?? "",
    initiative: mapInitiativeSnapshot(s?.initiative),
    trace: (s?.trace ?? []).map(mapSimulationStep),
  };
}

function mapRenderedPrompt(r: ompb.OperatingModeRenderPromptResponse | undefined): OperatingModeRenderedPrompt {
  return {
    mode: r?.mode ?? "",
    preset: orUndef(r?.preset),
    stepIndex: r?.stepIndex,
    phase: r?.phase ?? "",
    skillId: r?.skillId ?? "",
    profileKey: r?.profileKey ?? "",
    variables: r?.variables ?? {},
    prompt: r?.prompt ?? "",
    degraded: r?.degraded ?? false,
    degradedReason: orUndef(r?.degradedReason),
  };
}

function mapExecution(e: ompb.OperatingModeActiveItemExecution | undefined): ActiveItemExecution {
  return {
    itemRef: e?.itemRef ?? "",
    executionId: orUndef(e?.executionId),
    runId: orUndef(e?.runId),
    status: orUndef(e?.status),
  };
}

function mapSwitchResult(r: ompb.OperatingModeSwitchResult | undefined): SwitchOperatingModeResult {
  return {
    initiativeName: r?.initiativeName ?? "",
    fromMode: asMode(r?.fromMode),
    toMode: asMode(r?.toMode),
    canceledItemExecutions: (r?.canceledItemExecutions ?? []).map(mapExecution),
    activeItemExecutions: (r?.activeItemExecutions ?? []).map(mapExecution),
    requiresCancellation: r?.requiresCancellation,
    operatingModeWorkspaceId: orUndef(r?.operatingModeWorkspaceId),
  };
}

function mapWorkspacePhase(p: ompb.OperatingModeWorkspacePhase | undefined): OperatingModeWorkspacePhase {
  return {
    phase: p?.phase ?? "",
    phaseKind: mapPhaseKind(p?.phaseKind),
    activityPurpose: p?.activityPurpose ?? "",
    profileKey: p?.profileKey ?? "",
    writesRepo: p?.writesRepo ?? false,
    outputArtifacts: (p?.outputArtifacts ?? []).map(mapArtifactDef),
    requiresCriteria: p?.requiresCriteria,
    startable: p?.startable ?? false,
    reason: orUndef(p?.reason),
    next: p?.next,
    autoStartAfter: p?.autoStartAfter ?? [],
    executedBy: orUndef(p?.executedBy),
  };
}

function mapStringListMap(
  m: Record<string, ompb.OperatingModeStringList> | undefined,
): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const [key, value] of Object.entries(m ?? {})) {
    out[key] = value?.values ?? [];
  }
  return out;
}

function mapWorkspaceDefinition(m: ompb.OperatingModeWorkspaceMode | undefined): OperatingModeWorkspaceDefinition {
  return {
    mode: asMode(m?.mode),
    label: m?.label ?? "",
    description: orUndef(m?.description),
    targetKind: m?.targetKind ?? "",
    capabilities: mapCapabilities(m?.capabilities, (m?.phases?.length ?? 0) > 0),
    phases: (m?.phases ?? []).map(mapWorkspacePhase),
    terminal: m?.terminal ?? [],
    transitions: mapStringListMap(m?.transitions),
    runStrategy: m?.runStrategy ?? "",
  };
}

function mapWorkspace(w: ompb.OperatingModeWorkspace | undefined): OperatingModeWorkspace {
  const lock = w?.lock;
  return {
    initiativeName: w?.initiativeName ?? "",
    mode: asMode(w?.mode),
    definition: mapWorkspaceDefinition(w?.definition),
    lock: lock
      ? {
          runId: orUndef(lock.runId),
          purpose: orUndef(lock.purpose),
          roundNumber: lock.roundNumber ? lock.roundNumber : undefined,
          acquiredBy: orUndef(lock.acquiredBy),
          acquiredAt: orUndef(lock.acquiredAt),
        }
      : undefined,
    artifacts: (w?.artifacts ?? []).map(mapArtifactSnapshot),
    rounds: (w?.rounds ?? []).map(mapRound),
  };
}

function mapBacklogSyncResult(r: ompb.OperatingModeBacklogSyncResult | undefined): OperatingModeBacklogSyncResult {
  const proposal = r?.proposalResult;
  return {
    initiativeName: r?.initiativeName ?? "",
    mode: asMode(r?.mode),
    phase: r?.phase ?? "",
    round: r?.round ?? 0,
    runId: orUndef(r?.runId),
    completedItems: (r?.completedItems ?? []).map((c) => ({
      itemRef: c.itemRef,
      fromStatus: c.fromStatus,
      toStatus: c.toStatus,
    })),
    proposalResult: proposal
      ? {
          applied: proposal.applied,
          failed: proposal.failed,
          skipped: proposal.skipped,
          created: proposal.created ? proposal.created : undefined,
          updated: proposal.updated ? proposal.updated : undefined,
          outcomes: (proposal.outcomes ?? []).map((o) => ({
            mutationId: o.mutationId,
            op: o.op,
            target: orUndef(o.target),
            applied: o.applied,
            skipped: o.skipped,
            error: orUndef(o.error),
          })),
        }
      : undefined,
    noop: r?.noop,
  };
}

// ---------------------------------------------------------------------------
// Switch-conflict error detail
// ---------------------------------------------------------------------------

/**
 * Server-side conflict shape returned when SwitchMode is called against an
 * initiative with active item executions and `cancelActiveItemExecutions` is
 * false. Carried as a Connect error detail (FailedPrecondition) — see
 * `OperatingModeActiveItemExecutionsConflict` in
 * `scenarios/swarm-manager/api/internal/operatingmode/connect_service.go`.
 */
export interface ActiveItemExecutionsConflict {
  initiativeName: string;
  fromMode: string;
  toMode: string;
  executions: ActiveItemExecution[];
}

/**
 * Detect whether an error is the server's active-item-executions switch
 * conflict and decode its structured detail. The mode-picker dialog uses this
 * to render the affected items list before re-submitting with
 * `cancelActiveItemExecutions=true`.
 *
 * Returns the parsed conflict, or `null` for any other error shape.
 */
export function parseActiveItemExecutionsConflict(error: unknown): ActiveItemExecutionsConflict | null {
  if (!(error instanceof ConnectError)) return null;
  if (error.code !== Code.FailedPrecondition) return null;
  const [detail] = error.findDetails(OperatingModeActiveItemExecutionsConflictSchema);
  if (!detail) return null;
  return {
    initiativeName: detail.initiativeName,
    fromMode: detail.fromMode || "item-level",
    toMode: detail.toMode || "item-level",
    executions: (detail.executions ?? []).map(mapExecution),
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
  simulateMode(mode: string, preset?: string): Promise<OperatingModeSimulation>;
  renderSimulationPrompt(mode: string, preset: string, stepIndex: number): Promise<OperatingModeRenderedPrompt>;
  renderLivePrompt(name: string, phase: string, round?: number, note?: string): Promise<OperatingModeRenderedPrompt>;
  workspace(name: string): Promise<OperatingModeWorkspace>;
  switchMode(name: string, args: SwitchOperatingModeArgs): Promise<SwitchOperatingModeResult>;
  startPhase(name: string, phase: string, args?: StartOperatingModePhaseArgs): Promise<OperatingModeRound>;
  refreshRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
  cancelRound(name: string, mode: string, round: number): Promise<OperatingModeRound>;
  completeItems(name: string, args: CompleteOperatingModeItemsArgs): Promise<OperatingModeBacklogSyncResult>;
  applyBacklogSync(name: string, args: ApplyOperatingModeBacklogSyncArgs): Promise<OperatingModeBacklogSyncResult>;
}

const REQUESTED_BY = "swarm-manager-ui";

function defaultOperatingModeClient(): OperatingModeClient {
  return createClient(OperatingModeService, createConnectTransport({ baseUrl: API_BASE }));
}

export function createInitiativeModeService(
  client: OperatingModeClient = defaultOperatingModeClient(),
): IInitiativeModeService {
  return {
    async catalog(): Promise<OperatingModeCatalog> {
      return mapCatalog(await client.catalog({}));
    },

    async getMode(mode: string): Promise<OperatingModeDetail> {
      return mapModeDetail(await client.getMode({ mode }));
    },

    async updateMode(mode: string, args: UpdateOperatingModeArgs): Promise<OperatingModeDetail> {
      return mapModeDetail(
        await client.updateMode({ mode, label: args.label, description: args.description }),
      );
    },

    async simulateMode(mode: string, preset?: string): Promise<OperatingModeSimulation> {
      return mapSimulation(await client.simulateMode({ mode, preset: preset ?? "" }));
    },

    async renderSimulationPrompt(
      mode: string,
      preset: string,
      stepIndex: number,
    ): Promise<OperatingModeRenderedPrompt> {
      return mapRenderedPrompt(await client.renderSimulationPrompt({ mode, preset, stepIndex }));
    },

    async renderLivePrompt(
      name: string,
      phase: string,
      round?: number,
      note?: string,
    ): Promise<OperatingModeRenderedPrompt> {
      return mapRenderedPrompt(
        await client.renderLivePrompt({
          initiativeName: name,
          phase,
          round: typeof round === "number" && round > 0 ? round : 0,
          note: note?.trim() ?? "",
        }),
      );
    },

    async workspace(name: string): Promise<OperatingModeWorkspace> {
      return mapWorkspace(await client.getWorkspace({ initiativeName: name }));
    },

    async switchMode(name: string, args: SwitchOperatingModeArgs): Promise<SwitchOperatingModeResult> {
      return mapSwitchResult(
        await client.switchMode({
          initiativeName: name,
          mode: args.mode,
          cancelActiveItemExecutions: args.cancelActiveItemExecutions ?? false,
          requestedBy: args.requestedBy ?? REQUESTED_BY,
        }),
      );
    },

    async startPhase(
      name: string,
      phase: string,
      args: StartOperatingModePhaseArgs = {},
    ): Promise<OperatingModeRound> {
      return mapRound(
        await client.startPhase({
          initiativeName: name,
          phase,
          note: args.note ?? "",
          override: args.override ?? false,
          requestedBy: args.requestedBy ?? REQUESTED_BY,
        }),
      );
    },

    async refreshRound(name: string, mode: string, round: number): Promise<OperatingModeRound> {
      return mapRound(await client.refreshRound({ initiativeName: name, mode, round }));
    },

    async cancelRound(name: string, mode: string, round: number): Promise<OperatingModeRound> {
      return mapRound(await client.cancelRound({ initiativeName: name, mode, round }));
    },

    async completeItems(
      name: string,
      args: CompleteOperatingModeItemsArgs,
    ): Promise<OperatingModeBacklogSyncResult> {
      return mapBacklogSyncResult(
        await client.completeItems({
          initiativeName: name,
          mode: args.mode,
          round: args.round,
          runId: args.runId,
          itemRefs: args.itemRefs,
          requestedBy: args.requestedBy ?? REQUESTED_BY,
        }),
      );
    },

    async applyBacklogSync(
      name: string,
      args: ApplyOperatingModeBacklogSyncArgs,
    ): Promise<OperatingModeBacklogSyncResult> {
      return mapBacklogSyncResult(
        await client.applyBacklogSync({
          initiativeName: name,
          mode: args.mode,
          round: args.round,
          runId: args.runId,
          acceptedMutationIds: args.acceptedMutationIds ?? [],
          requestedBy: args.requestedBy ?? REQUESTED_BY,
        }),
      );
    },
  };
}

export const initiativeModeService = createInitiativeModeService();
