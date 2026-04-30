import { useEffect, useMemo, useState } from "react";
import { CheckCircle2, CheckSquare, GitBranch, RefreshCw, Square } from "lucide-react";
import { Button } from "../../ui/button";
import { formatRelativeTime } from "../../../lib";
import { selectors } from "../../../consts/selectors";
import type { ProposalMutation } from "../../../types/feedback";
import type { OperatingModeBacklogSyncResult, OperatingModeRound } from "../../../types/operating-mode";
import { phaseLabel, statusClasses } from "./utils";

function backlogSyncCompletedItems(round: OperatingModeRound): string[] {
  if (round.payload?.backlog_sync) {
    return [];
  }
  const plan = round.payload?.backlog_sync_plan;
  if (!plan || typeof plan !== "object" || Array.isArray(plan)) {
    return [];
  }
  const completedItems = (plan as { completed_items?: unknown; completedItems?: unknown }).completed_items ??
    (plan as { completed_items?: unknown; completedItems?: unknown }).completedItems;
  return Array.isArray(completedItems)
    ? completedItems.filter((item): item is string => typeof item === "string")
    : [];
}

function backlogSyncProposal(round: OperatingModeRound) {
  if (round.payload?.backlog_sync) {
    return undefined;
  }
  const plan = round.payload?.backlog_sync_plan;
  if (!plan || typeof plan !== "object" || Array.isArray(plan)) {
    return undefined;
  }
  const proposal = (plan as { proposal?: unknown }).proposal;
  if (!proposal || typeof proposal !== "object" || Array.isArray(proposal)) {
    return undefined;
  }
  const typedProposal = proposal as { form?: unknown; mutations?: unknown; rationale?: unknown };
  if (typedProposal.form !== "mutation_list" || !Array.isArray(typedProposal.mutations)) {
    return undefined;
  }
  return {
    form: "mutation_list" as const,
    rationale: typeof typedProposal.rationale === "string" ? typedProposal.rationale : undefined,
    mutations: typedProposal.mutations.filter((mutation): mutation is ProposalMutation => {
      return Boolean(mutation) &&
        typeof mutation === "object" &&
        !Array.isArray(mutation) &&
        typeof (mutation as { id?: unknown }).id === "string" &&
        typeof (mutation as { op?: unknown }).op === "string";
    }),
  };
}

function appliedBacklogSync(round: OperatingModeRound): OperatingModeBacklogSyncResult | undefined {
  const raw = round.payload?.backlog_sync;
  return raw && typeof raw === "object" && !Array.isArray(raw)
    ? raw as OperatingModeBacklogSyncResult
    : undefined;
}

function mutationSummary(mutation: ProposalMutation): string {
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

export function RoundCard({
  round,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
  busy,
}: {
  round: OperatingModeRound;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
  busy: boolean;
}) {
  const isActive = round.status === "reserved" || round.status === "agent_running";
  const summary = typeof round.payload?.agent_summary === "string" ? round.payload.agent_summary : "";
  const pendingCompletedItems = backlogSyncCompletedItems(round);
  const proposal = useMemo(() => backlogSyncProposal(round), [round]);
  const appliedSync = useMemo(() => appliedBacklogSync(round), [round]);
  const proposalMutationIds = useMemo(() => (proposal?.mutations ?? []).map((mutation) => mutation.id), [proposal]);
  const [selectedMutationIds, setSelectedMutationIds] = useState<Set<string>>(
    () => new Set(proposalMutationIds),
  );
  useEffect(() => {
    setSelectedMutationIds(new Set(proposalMutationIds));
  }, [proposalMutationIds]);
  const canCompleteItems = round.status === "completed" && Boolean(round.runId) && pendingCompletedItems.length > 0;
  const canApplyProposal = round.status === "completed" && Boolean(round.runId) && Boolean(proposal?.mutations.length) && selectedMutationIds.size > 0;
  const toggleMutation = (id: string) => {
    setSelectedMutationIds((previous) => {
      const next = new Set(previous);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  return (
    <div
      className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-3"
      data-testid={selectors.initiativeDetails.roundCard}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-slate-100">Round {round.round}</span>
            <span className={`rounded-full border px-2 py-0.5 text-[11px] ${statusClasses(round.status)}`}>
              {phaseLabel(round.status)}
            </span>
            <span className="rounded-full border border-slate-700/80 bg-slate-900/80 px-2 py-0.5 text-[11px] text-slate-400">
              {phaseLabel(round.phase)}
            </span>
          </div>
          <p className="mt-1 break-all text-[11px] text-slate-500">
            {round.agentProfileKey}
            {round.runId ? ` • ${round.runId}` : ""}
          </p>
        </div>
        <div className="flex gap-1.5">
          <Button variant="ghost" size="icon" onClick={() => onRefresh(round)} disabled={busy} title="Refresh round">
            <RefreshCw className="h-4 w-4" />
          </Button>
          {isActive && (
            <Button variant="ghost" size="icon" onClick={() => onCancel(round)} disabled={busy} title="Cancel round">
              <Square className="h-4 w-4" />
            </Button>
          )}
          {canCompleteItems && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onCompleteItems(round, pendingCompletedItems)}
              disabled={busy}
              title="Mark planned items complete"
              data-testid={selectors.initiativeDetails.completeItems}
            >
              <CheckSquare className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
      {round.error && <p className="mt-2 text-sm text-red-300">{round.error}</p>}
      {summary && <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-slate-300">{summary}</p>}
      {round.handoffs && round.handoffs.length > 0 && (
        <div className="mt-3 space-y-2">
          {round.handoffs.map((handoff, index) => (
            <div key={`${round.round}-handoff-${index}`} className="rounded-lg border border-slate-800 bg-slate-950/50 p-2">
              <p className="text-xs font-medium text-slate-300">Handoff {index + 1}</p>
              {handoff.summary && <p className="mt-1 text-sm text-slate-400">{handoff.summary}</p>}
              {handoff.nextStep && <p className="mt-1 text-xs text-cyan-300">Next: {handoff.nextStep}</p>}
            </div>
          ))}
        </div>
      )}
      {pendingCompletedItems.length > 0 && (
        <div className="mt-3 rounded-lg border border-slate-800 bg-slate-950/50 p-2">
          <p className="text-xs font-medium text-slate-300">Backlog sync plan</p>
          <p className="mt-1 break-words text-sm text-slate-400">{pendingCompletedItems.join(", ")}</p>
        </div>
      )}
      {proposal && proposal.mutations.length > 0 && (
        <div
          className="mt-3 space-y-2 rounded-lg border border-slate-800 bg-slate-950/50 p-2"
          data-testid={selectors.initiativeDetails.backlogProposal}
        >
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-xs font-medium text-slate-300">Backlog proposal</p>
              {proposal.rationale && <p className="mt-1 text-xs text-slate-500">{proposal.rationale}</p>}
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onApplyBacklogSync(round, Array.from(selectedMutationIds))}
              disabled={busy || !canApplyProposal}
              data-testid={selectors.initiativeDetails.applyBacklogSync}
            >
              <CheckCircle2 className="mr-1.5 h-4 w-4" />
              Apply {selectedMutationIds.size || 0} of {proposal.mutations.length}
            </Button>
          </div>
          <ul className="space-y-1.5">
            {proposal.mutations.map((mutation) => (
              <li
                key={mutation.id}
                className="rounded-md border border-slate-800 bg-slate-900/70 p-2"
                data-testid={selectors.initiativeDetails.backlogProposalMutation}
              >
                <label className="flex items-start gap-2">
                  <input
                    type="checkbox"
                    checked={selectedMutationIds.has(mutation.id)}
                    onChange={() => toggleMutation(mutation.id)}
                    disabled={busy}
                    className="mt-1 h-3.5 w-3.5 shrink-0 accent-cyan-500"
                    data-testid={selectors.initiativeDetails.backlogProposalMutationToggle}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
                      <GitBranch className="h-3.5 w-3.5 text-cyan-400" />
                      <span className="font-semibold">{phaseLabel(mutation.op)}</span>
                      <code className="rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-400">{mutation.id}</code>
                    </span>
                    {mutation.rationale && <span className="mt-1 block text-xs text-slate-500">{mutation.rationale}</span>}
                    {mutationSummary(mutation) && <span className="mt-1 block text-[11px] font-mono text-slate-500">{mutationSummary(mutation)}</span>}
                  </span>
                </label>
              </li>
            ))}
          </ul>
        </div>
      )}
      {appliedSync?.proposalResult && (
        <div className="mt-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-2 text-xs text-emerald-200">
          Applied backlog proposal: {appliedSync.proposalResult.applied} applied, {appliedSync.proposalResult.skipped} skipped, {appliedSync.proposalResult.failed} failed.
        </div>
      )}
      {round.generatedAt && <p className="mt-2 text-[11px] text-slate-500">{formatRelativeTime(round.generatedAt)}</p>}
    </div>
  );
}
