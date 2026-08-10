import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, GitPullRequestArrow, Loader2, Moon, SkipForward } from "lucide-react";
import { formatDisplayText } from "../../lib/format-utils";
import { proposalSessionService, type ProposalSessionProposal } from "../../services/proposal-session-service";
import { mutationPreviews, proposalPayload, ProposalReview } from "../session/ProposalSessionsPanel";

export interface ProposalDecisionStreamItem {
  sessionId: string;
  sessionTitle: string;
  proposal: ProposalSessionProposal;
  target?: { type: "goal" | "backlog_item" | "capture"; ref: string; name: string };
}

interface Props {
  proposals: ProposalDecisionStreamItem[];
  onComplete: () => void;
  onBack: () => void;
  onSnooze: (id: string) => void;
  /** Navigate to the attached backlog item / goal detail view. */
  onOpenItem?: () => void;
  /**
   * Rendered inside the decision drawer's shell, which already shows the
   * entity title and the queue position. Replaces this view's header with a
   * sub-position line, and only when this entry really holds more than one
   * proposal — a bare "1/1" next to the queue counter read as a conflict.
   */
  embedded?: boolean;
}

/** First-class proposal cards for the decision stream, not a panel below it. */
export function ProposalDecisionStreamView({ proposals, onComplete, onBack, onSnooze, onOpenItem, embedded = false }: Props) {
  const [index, setIndex] = useState(0);
  const [resolved, setResolved] = useState<Set<string>>(() => new Set());
  const [snoozed, setSnoozed] = useState<Set<string>>(() => new Set());
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [note, setNote] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const active = useMemo(() => proposals.filter(({ proposal }) => !resolved.has(proposal.id) && !snoozed.has(proposal.id)), [proposals, resolved, snoozed]);
  const safeIndex = Math.min(index, Math.max(active.length - 1, 0));
  const current = active[safeIndex];

  useEffect(() => {
    if (!current) { onComplete(); return; }
    setSelected(new Set(mutationPreviews(current.proposal).map((mutation) => mutation.id))); setNote(""); setError(null);
  }, [current?.proposal.id]);

  const advance = useCallback(() => {
    if (safeIndex < active.length - 1) setIndex(safeIndex + 1); else onComplete();
  }, [safeIndex, active.length, onComplete]);
  const decide = useCallback(async (accept: boolean) => {
    if (!current) return;
    setPending(true); setError(null);
    try {
      await proposalSessionService.decide(current.sessionId, current.proposal.id, accept ? [...selected] : [], note);
      setResolved((previous) => new Set(previous).add(current.proposal.id));
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to save proposal decision."); }
    finally { setPending(false); }
  }, [current, selected, note]);
  if (!current) return null;
  const mutations = mutationPreviews(current.proposal); const isFirst = safeIndex === 0; const isLast = safeIndex === active.length - 1;
  const rationale = proposalPayload(current.proposal).rationale;
  const targetName = current.target?.name || current.proposal.summary || "Mutation list";
  const subtitle = current.proposal.summary && current.proposal.summary !== targetName ? current.proposal.summary : current.sessionTitle !== targetName ? current.sessionTitle : "";
  return <div className="flex h-full flex-col" data-testid="proposal-decision-stream">
    {embedded ? (
      active.length > 1 ? (
        <p className="shrink-0 border-b border-slate-800 px-3 py-1.5 text-xs text-slate-500" data-testid="proposal-sub-position">
          Proposal {safeIndex + 1} of {active.length} for this item
        </p>
      ) : null
    ) : (
      <div className="flex shrink-0 items-center gap-2 border-b border-slate-800 px-3">
        <GitPullRequestArrow className="h-4 w-4 shrink-0 text-violet-300" />
        <div className="min-w-0 flex-1">
          {onOpenItem
            ? <button type="button" onClick={onOpenItem} className="block max-w-full cursor-pointer text-left text-sm font-medium text-cyan-300 line-clamp-2 hover:text-cyan-200 hover:underline" title={targetName}>{targetName}</button>
            : <p className="text-sm font-medium text-slate-200 line-clamp-2" title={targetName}>{targetName}</p>}
          {subtitle ? <p className="truncate text-xs text-slate-500">{subtitle}</p> : null}
        </div>
        {active.length > 1 ? <span className="text-xs tabular-nums text-slate-500">{safeIndex + 1}/{active.length}</span> : null}
      </div>
    )}
    <div className="flex-1 overflow-y-auto px-3 py-3"><div className="mx-auto max-w-2xl space-y-3"><p className="text-sm text-slate-400">Choose which proposed mutations to apply.</p><span className="inline-flex rounded-full bg-slate-800 px-2 py-1 text-xs text-slate-300">{formatDisplayText(current.proposal.status)}</span>{rationale && <p className="rounded border border-violet-400/20 bg-violet-400/5 p-3 text-sm leading-6 text-slate-300">{rationale}</p>}<ProposalReview mutations={mutations} proposal={current.proposal} />{mutations.length > 1 && mutations.map((mutation) => <label key={mutation.id} className="flex items-center gap-2 rounded border border-slate-800 bg-slate-900/60 p-3 text-sm text-slate-200"><input type="checkbox" checked={selected.has(mutation.id)} onChange={(event) => setSelected((previous) => { const next = new Set(previous); if (event.target.checked) next.add(mutation.id); else next.delete(mutation.id); return next; })} /><span>Apply {formatDisplayText(mutation.op)}</span></label>)}<input value={note} onChange={(event) => setNote(event.target.value)} placeholder="Decision note (optional)" className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" /><div className="flex flex-wrap gap-2"><button type="button" disabled={pending} onClick={() => void decide(true)} className="rounded bg-violet-600 px-3 py-2 text-sm text-white disabled:opacity-50">{mutations.length === 1 ? "Apply" : `Apply selected (${selected.size})`}</button><button type="button" disabled={pending} onClick={() => void decide(false)} className="rounded border border-slate-700 px-3 py-2 text-sm text-slate-300 disabled:opacity-50">Reject proposal</button>{pending && <Loader2 className="h-5 w-5 animate-spin text-violet-300" />}</div>{error && <p className="text-sm text-red-400">{error}</p>}</div></div>
    {/* Moving between proposals only makes sense when there is more than one;
        embedded, queue-level movement belongs to the drawer footer. */}
    {embedded && active.length <= 1 ? null : (
      <div className="border-t border-slate-700/50 px-3 py-2"><div className="mx-auto flex max-w-2xl items-center justify-between"><button type="button" onClick={() => isFirst ? onBack() : setIndex(safeIndex - 1)} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400"><ChevronLeft className="h-4 w-4" />Back</button><div className="flex items-center gap-1"><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><SkipForward className="h-4 w-4" />Skip</button><button type="button" onClick={() => { setSnoozed((previous) => new Set(previous).add(current.proposal.id)); onSnooze(current.proposal.id); }} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><Moon className="h-4 w-4" />Snooze</button></div><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400">{isLast ? "Done" : "Next"}<ChevronRight className="h-4 w-4" /></button></div></div>
    )}
  </div>;
}
