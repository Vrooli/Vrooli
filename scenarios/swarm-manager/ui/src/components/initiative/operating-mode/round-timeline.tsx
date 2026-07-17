import { useState } from "react";
import { ChevronRight } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCapabilities, OperatingModeEvidenceRecord, OperatingModeRound } from "../../../types/operating-mode";
import type { OperationProvenanceData } from "../../../lib/agent-ops-utils";
import { RoundCard } from "./round-card";
import { RoundDetailDialog } from "./round-detail-dialog";
import { phaseLabel, statusClasses } from "./utils";

interface PhaseBucket {
  phase: string;
  rounds: OperatingModeRound[];
}

// Bucket the reverse-chronological rounds by phase. Bucket order = order of
// first appearance in the reversed list (so the most-recent phase floats to
// the top, ties broken by recency). Inside each bucket, rounds remain in the
// reverse-chronological order they came in.
function bucketByPhase(rounds: OperatingModeRound[]): PhaseBucket[] {
  const reversed = [...rounds].reverse();
  const order: string[] = [];
  const map = new Map<string, OperatingModeRound[]>();
  for (const round of reversed) {
    let bucket = map.get(round.phase);
    if (!bucket) {
      bucket = [];
      map.set(round.phase, bucket);
      order.push(round.phase);
    }
    bucket.push(round);
  }
  return order.map((phase) => ({ phase, rounds: map.get(phase) ?? [] }));
}

export function RoundTimeline({
  rounds,
  capabilities,
  busy,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
  evidenceByRun,
  provenanceByExecutionId,
}: {
  rounds: OperatingModeRound[];
  capabilities: OperatingModeCapabilities;
  busy: boolean;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
  evidenceByRun?: Record<string, OperatingModeEvidenceRecord[]>;
  /** Canonical operation provenance keyed by execution id (workflow projection). */
  provenanceByExecutionId?: ReadonlyMap<string, OperationProvenanceData>;
}) {
  const [detailRound, setDetailRound] = useState<OperatingModeRound | null>(null);

  if (rounds.length === 0) {
    const message = capabilities.supportsPhases
      ? "No rounds yet. Start a phase from the composer above to begin."
      : "This mode does not run rounds — execution happens through item runs.";
    return (
      <p
        className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500"
        data-testid={selectors.initiativeDetails.roundTimelineEmpty}
        data-supports-phases={String(capabilities.supportsPhases)}
      >
        {message}
      </p>
    );
  }

  const buckets = bucketByPhase(rounds);
  const COLLAPSE_THRESHOLD = 10;

  return (
    <div className="space-y-3">
      {buckets.map((bucket) => {
        // bucket.rounds is guaranteed non-empty by bucketByPhase: a bucket is
        // only created when at least one round goes into it.
        const [last, ...rest] = bucket.rounds;
        if (!last) return null;
        const all = [last, ...rest];
        return (
          <details
            key={bucket.phase}
            open
            className="rounded-lg border border-slate-800/80 bg-slate-900/40"
            data-testid={selectors.initiativeDetails.roundsPhaseGroup}
            data-phase={bucket.phase}
          >
            <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm text-slate-200 hover:bg-slate-800/40">
              <ChevronRight
                className="h-3.5 w-3.5 text-slate-500 transition-transform [details[open]_>_summary_&]:rotate-90"
                aria-hidden="true"
              />
              <span className="font-medium">{phaseLabel(bucket.phase)}</span>
              <span className="text-xs text-slate-500">
                — {bucket.rounds.length} round{bucket.rounds.length === 1 ? "" : "s"}
              </span>
              <span
                className={`ml-auto rounded-full border px-2 py-0.5 text-[11px] ${statusClasses(last.status)}`}
              >
                last: {phaseLabel(last.status)}
              </span>
            </summary>
            <PhaseBucketBody
              all={all}
              capabilities={capabilities}
              busy={busy}
              onRefresh={onRefresh}
              onCancel={onCancel}
              onCompleteItems={onCompleteItems}
              onApplyBacklogSync={onApplyBacklogSync}
              onViewDetails={setDetailRound}
              collapseThreshold={COLLAPSE_THRESHOLD}
              provenanceByExecutionId={provenanceByExecutionId}
            />
          </details>
        );
      })}
      {detailRound && (
        <RoundDetailDialog
          round={detailRound}
          isOpen={Boolean(detailRound)}
          onClose={() => setDetailRound(null)}
          evidence={detailRound.runId ? evidenceByRun?.[detailRound.runId] : undefined}
        />
      )}
    </div>
  );
}

function PhaseBucketBody({
  all,
  capabilities,
  busy,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
  onViewDetails,
  collapseThreshold,
  provenanceByExecutionId,
}: {
  all: OperatingModeRound[];
  capabilities: OperatingModeCapabilities;
  busy: boolean;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
  onViewDetails: (round: OperatingModeRound) => void;
  collapseThreshold: number;
  provenanceByExecutionId?: ReadonlyMap<string, OperationProvenanceData>;
}) {
  const [expanded, setExpanded] = useState(false);
  const showAll = expanded || all.length <= collapseThreshold;
  const visibleRounds = showAll ? all : all.slice(0, collapseThreshold);
  const hiddenCount = all.length - visibleRounds.length;

  return (
    <div className="space-y-3 border-t border-slate-800 p-3">
      {visibleRounds.map((round) => (
        <RoundCard
          key={`${round.mode}-${round.round}`}
          round={round}
          capabilities={capabilities}
          busy={busy}
          onRefresh={onRefresh}
          onCancel={onCancel}
          onCompleteItems={onCompleteItems}
          onApplyBacklogSync={onApplyBacklogSync}
          onViewDetails={onViewDetails}
          provenance={round.executionId ? provenanceByExecutionId?.get(round.executionId) : undefined}
        />
      ))}
      {hiddenCount > 0 && (
        <button
          type="button"
          onClick={() => setExpanded(true)}
          data-testid={selectors.initiativeDetails.roundTimelineShowMore}
          className="w-full rounded-md border border-dashed border-slate-700 bg-slate-900/40 px-3 py-2 text-xs font-medium text-slate-400 transition-colors hover:border-cyan-500/50 hover:text-cyan-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
        >
          Show {hiddenCount} more round{hiddenCount === 1 ? "" : "s"}
        </button>
      )}
    </div>
  );
}
