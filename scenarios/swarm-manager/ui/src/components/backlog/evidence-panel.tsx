/**
 * EvidencePanel
 *
 * Displays review evidence rounds in the Output tab. Each round shows
 * typed evidence cards (screenshots, API tests, CLI output, etc.) with
 * verification checkboxes. Users can request additional evidence via
 * the Request More button.
 */

import { useState } from "react";
import {
  ChevronDown,
  ChevronRight,
  Eye,
  Loader2,
  MessageSquarePlus,
} from "lucide-react";
import type { ReviewRound } from "../../services/review-service";
import { EvidenceItemCard } from "./evidence-item-card";

export interface EvidencePanelProps {
  rounds: ReviewRound[];
  backlogKind: string;
  backlogName: string;
  isGathering: boolean;
  onVerify: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMore: (round: number, evidenceId?: string) => void;
}

export function EvidencePanel({
  rounds,
  backlogKind,
  backlogName,
  isGathering,
  onVerify,
  onRequestMore,
}: EvidencePanelProps) {
  const [expandedRounds, setExpandedRounds] = useState<Set<number>>(() => {
    // Expand the latest round by default.
    const last = rounds[rounds.length - 1];
    if (last) {
      return new Set([last.round]);
    }
    return new Set();
  });

  const toggleRound = (roundNum: number) => {
    setExpandedRounds((prev) => {
      const next = new Set(prev);
      if (next.has(roundNum)) {
        next.delete(roundNum);
      } else {
        next.add(roundNum);
      }
      return next;
    });
  };

  // Compute verification progress across all rounds.
  const allEvidence = rounds.flatMap((r) => r.evidence);
  const verifiedCount = allEvidence.filter((e) => e.verified).length;
  const totalCount = allEvidence.length;

  return (
    <div className="border-t border-slate-200 dark:border-slate-700">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3">
        <div className="flex items-center gap-2">
          <Eye className="h-4 w-4 text-violet-500" />
          <span className="text-sm font-medium text-slate-700 dark:text-slate-200">
            Evidence
          </span>
          {totalCount > 0 && (
            <span className="text-xs text-slate-500 dark:text-slate-400">
              {verifiedCount} of {totalCount} verified
            </span>
          )}
          {isGathering && (
            <span className="flex items-center gap-1 text-xs text-cyan-600 dark:text-cyan-400">
              <Loader2 className="h-3 w-3 animate-spin" />
              Gathering evidence...
            </span>
          )}
        </div>
        {rounds.length > 0 && (
          <button
            onClick={() => { const last = rounds[rounds.length - 1]; if (last) onRequestMore(last.round); }}
            className="flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-violet-600 hover:bg-violet-50 dark:text-violet-400 dark:hover:bg-violet-900/20"
          >
            <MessageSquarePlus className="h-3.5 w-3.5" />
            Request More
          </button>
        )}
      </div>

      {/* Rounds (reverse order, latest first) */}
      <div className="space-y-0">
        {[...rounds].reverse().map((round) => (
          <RoundSection
            key={round.round}
            round={round}
            backlogKind={backlogKind}
            backlogName={backlogName}
            expanded={expandedRounds.has(round.round)}
            onToggle={() => toggleRound(round.round)}
            onVerify={onVerify}
            onRequestMore={onRequestMore}
          />
        ))}
      </div>

      {/* Empty state */}
      {rounds.length === 0 && !isGathering && (
        <div className="px-4 pb-3 text-xs text-slate-400 dark:text-slate-500">
          Review agent will gather evidence after post-run checks complete.
        </div>
      )}
    </div>
  );
}

// --- Round Section ---

interface RoundSectionProps {
  round: ReviewRound;
  backlogKind: string;
  backlogName: string;
  expanded: boolean;
  onToggle: () => void;
  onVerify: (round: number, evidenceId: string, verified: boolean) => void;
  onRequestMore: (round: number, evidenceId?: string) => void;
}

function RoundSection({
  round,
  backlogKind,
  backlogName,
  expanded,
  onToggle,
  onVerify,
  onRequestMore,
}: RoundSectionProps) {
  const verifiedCount = round.evidence.filter((e) => e.verified).length;
  const Icon = expanded ? ChevronDown : ChevronRight;

  return (
    <div className="border-t border-slate-100 dark:border-slate-800">
      {/* Round header */}
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-2 px-4 py-2 text-left hover:bg-slate-50 dark:hover:bg-slate-800/50"
      >
        <Icon className="h-4 w-4 text-slate-400" />
        <span className="text-xs font-medium text-slate-600 dark:text-slate-300">
          Round {round.round}
        </span>
        <RoundStatusBadge status={round.status} classification={round.classification} />
        <span className="text-xs text-slate-400 dark:text-slate-500">
          {round.evidence.length} items, {verifiedCount} verified
        </span>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="space-y-2 px-4 pb-3">
          {/* Evidence cards */}
          {round.evidence.map((item) => (
            <EvidenceItemCard
              key={item.id}
              item={item}
              backlogKind={backlogKind}
              backlogName={backlogName}
              onVerify={(evidenceId, verified) => onVerify(round.round, evidenceId, verified)}
              onRequestMore={(evidenceId) => onRequestMore(round.round, evidenceId)}
            />
          ))}

          {/* Agent assessment */}
          {round.agent_assessment && (
            <div className="mt-2 rounded-md bg-slate-50 p-3 text-xs text-slate-600 dark:bg-slate-800/50 dark:text-slate-400">
              <span className="font-medium">Agent assessment: </span>
              {round.agent_assessment}
            </div>
          )}

          {/* Notes */}
          {round.notes && round.notes.length > 0 && (
            <div className="mt-1 space-y-1">
              {round.notes.map((note, i) => (
                <div key={i} className="text-xs text-slate-500 dark:text-slate-400">
                  - {note}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// --- Status Badge ---

function RoundStatusBadge({
  status,
  classification,
}: {
  status: string;
  classification?: string;
}) {
  if (status === "gathering") {
    return (
      <span className="flex items-center gap-1 rounded-full bg-cyan-100 px-2 py-0.5 text-xs text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400">
        <Loader2 className="h-3 w-3 animate-spin" />
        Gathering
      </span>
    );
  }
  if (status === "failed") {
    return (
      <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-400">
        Failed
      </span>
    );
  }
  if (status === "complete" && classification) {
    const colors: Record<string, string> = {
      ready: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
      ready_with_notes: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
      needs_work: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
      not_assessable: "bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400",
    };
    const labels: Record<string, string> = {
      ready: "Ready",
      ready_with_notes: "Ready with notes",
      needs_work: "Needs work",
      not_assessable: "Inconclusive",
    };
    return (
      <span className={`rounded-full px-2 py-0.5 text-xs ${colors[classification] ?? colors.not_assessable}`}>
        {labels[classification] ?? classification}
      </span>
    );
  }
  return null;
}
