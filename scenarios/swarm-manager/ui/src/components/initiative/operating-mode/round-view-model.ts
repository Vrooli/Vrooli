import type { ProposalMutation } from "../../../types/proposal";
import type {
  OperatingModeBacklogSyncResult,
  OperatingModeCapabilities,
  OperatingModeRound,
} from "../../../types/operating-mode";

export interface OperatingModeBacklogProposal {
  form: "mutation_list";
  rationale?: string;
  mutations: ProposalMutation[];
}

export interface RoundViewModel {
  isActive: boolean;
  summary: string;
  pendingCompletedItems: string[];
  proposal?: OperatingModeBacklogProposal;
  appliedSync?: OperatingModeBacklogSyncResult;
  defaultSelectedMutationIds: string[];
  canCompleteItems: boolean;
  syncActionUnavailableReason?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function stringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
}

function backlogSyncPlan(round: OperatingModeRound): Record<string, unknown> | undefined {
  const plan = round.payload?.backlog_sync_plan;
  return isRecord(plan) ? plan : undefined;
}

export function hasAppliedBacklogSync(round: OperatingModeRound): boolean {
  return isOperatingModeBacklogSyncResult(round.payload?.backlog_sync);
}

function isOperatingModeBacklogSyncResult(value: unknown): value is OperatingModeBacklogSyncResult {
  if (!isRecord(value)) {
    return false;
  }
  return typeof value.initiativeName === "string" &&
    typeof value.mode === "string" &&
    typeof value.phase === "string" &&
    typeof value.round === "number" &&
    Array.isArray(value.completedItems);
}

export function appliedBacklogSync(round: OperatingModeRound): OperatingModeBacklogSyncResult | undefined {
  const raw = round.payload?.backlog_sync;
  return isOperatingModeBacklogSyncResult(raw) ? raw : undefined;
}

export function pendingCompletedItems(round: OperatingModeRound): string[] {
  if (hasAppliedBacklogSync(round)) {
    return [];
  }
  const plan = backlogSyncPlan(round);
  if (!plan) {
    return [];
  }
  return stringArray(plan.completed_items ?? plan.completedItems);
}

function isProposalMutation(value: unknown): value is ProposalMutation {
  return isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.op === "string";
}

export function pendingBacklogProposal(round: OperatingModeRound): OperatingModeBacklogProposal | undefined {
  if (hasAppliedBacklogSync(round)) {
    return undefined;
  }
  const proposal = backlogSyncPlan(round)?.proposal;
  if (!isRecord(proposal) || proposal.form !== "mutation_list") {
    return undefined;
  }
  const mutations = Array.isArray(proposal.mutations)
    ? proposal.mutations.filter(isProposalMutation)
    : [];
  return {
    form: "mutation_list",
    rationale: typeof proposal.rationale === "string" ? proposal.rationale : undefined,
    mutations,
  };
}

export function selectedMutationDefaults(proposal: OperatingModeBacklogProposal | undefined): string[] {
  return proposal?.mutations.map((mutation) => mutation.id) ?? [];
}

export function canApplyBacklogProposal(
  round: OperatingModeRound,
  selectedMutationIds: ReadonlySet<string>,
  capabilities: OperatingModeCapabilities,
): boolean {
  return capabilities.canApplyBacklogSyncProposals &&
    round.status === "completed" &&
    Boolean(round.runId) &&
    selectedMutationIds.size > 0 &&
    Boolean(pendingBacklogProposal(round)?.mutations.length);
}

export function backlogSyncActionUnavailableReason(
  round: OperatingModeRound,
  capabilities: OperatingModeCapabilities,
): string | undefined {
  if (!capabilities.canCompleteItems && !capabilities.canApplyBacklogSyncProposals) {
    return "This mode does not support backlog sync actions.";
  }
  if (round.status !== "completed") {
    return "Round must be completed before backlog sync actions are available.";
  }
  if (!round.runId) {
    return "Round is missing an AgentManager run ID, so backlog sync actions are disabled.";
  }
  return undefined;
}

export function mutationSummary(mutation: ProposalMutation): string {
  const bits: string[] = [];
  if (mutation.target) {
    bits.push(mutation.target);
  }
  if (mutation.item) {
    bits.push(`${mutation.item.kind}/${mutation.item.name}`);
    if (mutation.item.title) {
      bits.push(mutation.item.title);
    }
  }
  if (mutation.patch) {
    const fields = Object.keys(mutation.patch);
    if (fields.length > 0) {
      bits.push(`patch: ${fields.join(", ")}`);
    }
  }
  if (mutation.status) {
    bits.push(`status: ${mutation.status}`);
  }
  if (mutation.priority != null) {
    bits.push(`priority: ${mutation.priority}`);
  }
  if (mutation.from && mutation.to) {
    bits.push(`${mutation.from} -> ${mutation.to}`);
  }
  return bits.join(" | ");
}

export function buildRoundViewModel(round: OperatingModeRound, capabilities: OperatingModeCapabilities): RoundViewModel {
  const proposal = pendingBacklogProposal(round);
  const completedItems = pendingCompletedItems(round);
  const unavailableReason = backlogSyncActionUnavailableReason(round, capabilities);
  return {
    isActive: round.status === "reserved" || round.status === "agent_running",
    summary: typeof round.payload?.agent_summary === "string" ? round.payload.agent_summary : "",
    pendingCompletedItems: completedItems,
    proposal,
    appliedSync: appliedBacklogSync(round),
    defaultSelectedMutationIds: selectedMutationDefaults(proposal),
    canCompleteItems: capabilities.canCompleteItems && round.status === "completed" && Boolean(round.runId) && completedItems.length > 0,
    syncActionUnavailableReason: unavailableReason,
  };
}
