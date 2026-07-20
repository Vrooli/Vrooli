import { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, GitPullRequestArrow, Loader2, Moon, SkipForward } from "lucide-react";
import { formatDisplayText } from "../../lib/format-utils";
import { proposalSessionService, type ProposalSessionProposal } from "../../services/proposal-session-service";

export interface ProposalDecisionStreamItem {
  sessionId: string;
  sessionTitle: string;
  proposal: ProposalSessionProposal;
}

function mutationIDs(proposal: ProposalSessionProposal): string[] {
  try {
    const payload = JSON.parse(proposal.payload_json) as { mutations?: Array<{ id?: string }> };
    return (payload.mutations ?? []).map((mutation) => mutation.id).filter((id): id is string => Boolean(id));
  } catch { return []; }
}

interface Props { proposals: ProposalDecisionStreamItem[]; onComplete: () => void; onBack: () => void; onSnooze: (id: string) => void; }

/** First-class proposal cards for the decision stream, not a panel below it. */
export function ProposalDecisionStreamView({ proposals, onComplete, onBack, onSnooze }: Props) {
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
    setSelected(new Set(mutationIDs(current.proposal))); setNote(""); setError(null);
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
  const ids = mutationIDs(current.proposal); const isFirst = safeIndex === 0; const isLast = safeIndex === active.length - 1;
  return <div className="flex h-full flex-col" data-testid="proposal-decision-stream">
    <div className="flex shrink-0 items-center gap-2 border-b border-slate-700/50 bg-slate-950 px-3"><GitPullRequestArrow className="h-4 w-4 shrink-0 text-violet-300" /><div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-slate-200">{current.proposal.summary || "Mutation list"}</p><p className="truncate text-xs text-slate-500">{current.sessionTitle}</p></div><span className="text-xs tabular-nums text-slate-500">{safeIndex + 1}/{active.length}</span></div>
    <div className="flex-1 overflow-y-auto px-3 py-3"><div className="mx-auto max-w-2xl space-y-3"><p className="text-sm text-slate-400">Choose which proposed mutations to apply.</p><span className="inline-flex rounded-full bg-slate-800 px-2 py-1 text-xs text-slate-300">{formatDisplayText(current.proposal.status)}</span>{ids.map((id) => <label key={id} className="flex items-center gap-2 rounded border border-slate-800 bg-slate-900/60 p-3 text-sm text-slate-200"><input type="checkbox" checked={selected.has(id)} onChange={(event) => setSelected((previous) => { const next = new Set(previous); event.target.checked ? next.add(id) : next.delete(id); return next; })} />{id}</label>)}<input value={note} onChange={(event) => setNote(event.target.value)} placeholder="Decision note (optional)" className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" /><div className="flex flex-wrap gap-2"><button type="button" disabled={pending} onClick={() => void decide(true)} className="rounded bg-violet-600 px-3 py-2 text-sm text-white disabled:opacity-50">Apply selected ({selected.size})</button><button type="button" disabled={pending} onClick={() => void decide(false)} className="rounded border border-slate-700 px-3 py-2 text-sm text-slate-300 disabled:opacity-50">Reject proposal</button>{pending && <Loader2 className="h-5 w-5 animate-spin text-violet-300" />}</div>{error && <p className="text-sm text-red-400">{error}</p>}</div></div>
    <div className="border-t border-slate-700/50 px-3 py-2"><div className="mx-auto flex max-w-2xl items-center justify-between"><button type="button" onClick={() => isFirst ? onBack() : setIndex(safeIndex - 1)} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400"><ChevronLeft className="h-4 w-4" />Back</button><div className="flex items-center gap-1"><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><SkipForward className="h-4 w-4" />Skip</button><button type="button" onClick={() => { setSnoozed((previous) => new Set(previous).add(current.proposal.id)); onSnooze(current.proposal.id); }} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><Moon className="h-4 w-4" />Snooze</button></div><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400">{isLast ? "Done" : "Next"}<ChevronRight className="h-4 w-4" /></button></div></div>
  </div>;
}
