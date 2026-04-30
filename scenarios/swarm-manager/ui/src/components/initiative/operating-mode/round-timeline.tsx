import type { OperatingModeRound } from "../../../types/operating-mode";
import { RoundCard } from "./round-card";

export function RoundTimeline({
  rounds,
  busy,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
}: {
  rounds: OperatingModeRound[];
  busy: boolean;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
}) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-100">Rounds</h3>
      {rounds.length === 0 ? (
        <p className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500">No operating-mode rounds have run yet.</p>
      ) : (
        [...rounds].reverse().map((round) => (
          <RoundCard
            key={`${round.mode}-${round.round}`}
            round={round}
            busy={busy}
            onRefresh={onRefresh}
            onCancel={onCancel}
            onCompleteItems={onCompleteItems}
            onApplyBacklogSync={onApplyBacklogSync}
          />
        ))
      )}
    </div>
  );
}
