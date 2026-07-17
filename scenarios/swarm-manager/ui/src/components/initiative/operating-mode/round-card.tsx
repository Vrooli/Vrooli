import { useMemo } from "react";
import { CheckSquare, FileText, RefreshCw, Square } from "lucide-react";
import { Button } from "../../ui/button";
import { formatRelativeTime } from "../../../lib";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCapabilities, OperatingModeRound } from "../../../types/operating-mode";
import type { OperationProvenanceData } from "../../../lib/agent-ops-utils";
import { OperationProvenancePopover } from "../../workflow/operation-provenance-popover";
import { BacklogSyncActions } from "./backlog-sync-actions";
import { buildRoundViewModel } from "./round-view-model";
import { phaseLabel, resolutionSummary, statusClasses } from "./utils";

export function RoundCard({
  round,
  capabilities,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
  onViewDetails,
  busy,
  provenance,
}: {
  round: OperatingModeRound;
  capabilities: OperatingModeCapabilities;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
  onViewDetails?: (round: OperatingModeRound) => void;
  busy: boolean;
  /** Canonical operation provenance for this round's execution, when covered. */
  provenance?: OperationProvenanceData;
}) {
  const view = useMemo(() => buildRoundViewModel(round, capabilities), [round, capabilities]);
  const resolution = resolutionSummary(round.resolution);
  const errorTone = round.status === "needs_attention" ? "text-amber-300" : "text-red-300";

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
            {provenance && <OperationProvenancePopover data={provenance} />}
          </div>
          <p className="mt-1 break-all text-[11px] text-slate-500">
            {round.agentProfileKey}
            {round.runId ? ` • ${round.runId}` : ""}
          </p>
        </div>
        <div className="flex gap-1.5">
          {onViewDetails && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onViewDetails(round)}
              title="View round details"
              data-testid={selectors.initiativeDetails.roundDetailButton}
            >
              <FileText className="h-4 w-4" />
            </Button>
          )}
          <Button variant="ghost" size="icon" onClick={() => onRefresh(round)} disabled={busy} title="Refresh round">
            <RefreshCw className="h-4 w-4" />
          </Button>
          {view.isActive && (
            <Button variant="ghost" size="icon" onClick={() => onCancel(round)} disabled={busy} title="Cancel round">
              <Square className="h-4 w-4" />
            </Button>
          )}
          {view.canCompleteItems && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onCompleteItems(round, view.pendingCompletedItems)}
              disabled={busy}
              title="Mark planned items complete"
              data-testid={selectors.initiativeDetails.completeItems}
            >
              <CheckSquare className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
      {resolution && (
        <p className="mt-2 text-xs text-amber-200">
          Resolution: <span className="font-medium">{resolution}</span>
        </p>
      )}
      {round.error && <p className={`mt-2 text-sm ${errorTone}`}>{round.error}</p>}
      {view.summary && <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-slate-300">{view.summary}</p>}
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
      {view.pendingCompletedItems.length > 0 && (
        <div className="mt-3 rounded-lg border border-slate-800 bg-slate-950/50 p-2">
          <p className="text-xs font-medium text-slate-300">Backlog sync plan</p>
          <p className="mt-1 break-words text-sm text-slate-400">{view.pendingCompletedItems.join(", ")}</p>
        </div>
      )}
      {view.proposal && view.proposal.mutations.length > 0 && (
        <BacklogSyncActions
          round={round}
          proposal={view.proposal}
          capabilities={capabilities}
          busy={busy}
          onApplyBacklogSync={onApplyBacklogSync}
        />
      )}
      {view.syncActionUnavailableReason && (view.pendingCompletedItems.length > 0 || Boolean(view.proposal?.mutations.length)) && (
        <p className="mt-2 text-xs text-amber-300">{view.syncActionUnavailableReason}</p>
      )}
      {view.appliedSync?.proposalResult && (
        <div className="mt-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-2 text-xs text-emerald-200">
          Applied backlog proposal: {view.appliedSync.proposalResult.applied} applied, {view.appliedSync.proposalResult.skipped} skipped, {view.appliedSync.proposalResult.failed} failed.
        </div>
      )}
      {round.generatedAt && <p className="mt-2 text-[11px] text-slate-500">{formatRelativeTime(round.generatedAt)}</p>}
    </div>
  );
}
