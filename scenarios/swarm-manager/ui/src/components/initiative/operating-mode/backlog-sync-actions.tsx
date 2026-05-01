import { useEffect, useMemo, useState } from "react";
import { CheckCircle2, GitBranch } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCapabilities, OperatingModeRound } from "../../../types/operating-mode";
import { Button } from "../../ui/button";
import { phaseLabel } from "./utils";
import {
  canApplyBacklogProposal,
  mutationSummary,
  selectedMutationDefaults,
  type OperatingModeBacklogProposal,
} from "./round-view-model";

export function BacklogSyncActions({
  round,
  proposal,
  capabilities,
  busy,
  onApplyBacklogSync,
}: {
  round: OperatingModeRound;
  proposal: OperatingModeBacklogProposal;
  capabilities: OperatingModeCapabilities;
  busy: boolean;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
}) {
  const proposalMutationIds = useMemo(() => selectedMutationDefaults(proposal), [proposal]);
  const [selectedMutationIds, setSelectedMutationIds] = useState<Set<string>>(
    () => new Set(proposalMutationIds),
  );
  useEffect(() => {
    setSelectedMutationIds(new Set(proposalMutationIds));
  }, [proposalMutationIds]);
  const canApplyProposal = canApplyBacklogProposal(round, selectedMutationIds, capabilities);
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
  );
}
