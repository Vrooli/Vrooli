import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, GitPullRequestArrow, MessageSquarePlus, RefreshCw } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { sessionDetailPath } from "../../app/routes/route-paths";
import { selectors } from "../../consts/selectors";
import { formatDisplayText } from "../../lib/format-utils";
import { proposalSessionService, type ProposalSessionProposal, type ProposalSessionTargetType } from "../../services/proposal-session-service";
import { Button } from "../ui/button";
import { EntityAttachToSessionSheet } from "./context/EntityAttachToSessionSheet";

interface ProposalSessionsPanelProps {
  target: { type: ProposalSessionTargetType; ref: string; name: string };
}

function mutationIDs(proposal: ProposalSessionProposal): string[] {
  try {
    const payload = JSON.parse(proposal.payload_json) as { mutations?: Array<{ id?: string }> };
    return (payload.mutations ?? []).map((mutation) => mutation.id).filter((id): id is string => Boolean(id));
  } catch {
    return [];
  }
}

export function ProposalSessionsPanel({ target }: ProposalSessionsPanelProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const queryKey = ["proposal-sessions", target.type, target.ref];
  const [selected, setSelected] = useState<Record<string, Set<string>>>({});
  const [note, setNote] = useState("");
  const [startSheetOpen, setStartSheetOpen] = useState(false);
  const { data: sessions = [], isLoading, error } = useQuery({ queryKey, queryFn: () => proposalSessionService.list(target) });
  const invalidate = () => void queryClient.invalidateQueries({ queryKey });
  const decide = useMutation({ mutationFn: ({ sessionId, proposalId, ids }: { sessionId: string; proposalId: string; ids: string[] }) => proposalSessionService.decide(sessionId, proposalId, ids, note), onSuccess: invalidate });
  const revise = useMutation({ mutationFn: ({ sessionId, proposalId }: { sessionId: string; proposalId: string }) => proposalSessionService.revise(sessionId, proposalId, note), onSuccess: (session) => { invalidate(); navigate(sessionDetailPath(session.id)); } });
  const proposals = useMemo(() => sessions.flatMap((session) => (session.proposals ?? []).filter((proposal) => proposal.kind === "mutation_list").map((proposal) => ({ session, proposal }))), [sessions]);

  return <section className="space-y-3 py-4" data-testid="proposal-sessions-panel">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <div><h2 className="text-base font-semibold text-slate-100">Proposals</h2><p className="text-sm text-slate-400">Review agent mutation lists before they change this work.</p></div>
      <Button size="sm" onClick={() => setStartSheetOpen(true)} data-testid={selectors.agentSessions.proposalStart}><MessageSquarePlus className="mr-2 h-4 w-4" />Start proposal</Button>
    </div>
    <EntityAttachToSessionSheet
      isOpen={startSheetOpen}
      onClose={() => setStartSheetOpen(false)}
      option={{ type: target.type, ref: target.ref, title: target.name }}
      proposalMode
    />
    {error && <p className="rounded-md border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error instanceof Error ? error.message : "Unable to load proposals."}</p>}
    {isLoading ? <p className="py-8 text-sm text-slate-500">Loading proposals…</p> : proposals.length === 0 ? <p className="rounded-lg border border-dashed border-slate-700 p-8 text-center text-sm text-slate-500">No proposal sessions yet.</p> : proposals.map(({ session, proposal }) => {
      const ids = mutationIDs(proposal); const checked = selected[proposal.id] ?? new Set(ids);
      return <article key={proposal.id} className="rounded-lg border border-slate-800 bg-slate-900/60 p-4">
        <div className="flex items-start justify-between gap-3"><div><div className="flex items-center gap-2 text-sm font-medium text-slate-100"><GitPullRequestArrow className="h-4 w-4 text-violet-300" />{proposal.summary || "Mutation list"}</div><p className="mt-1 text-xs text-slate-500">Session {session.title}</p></div><span className="rounded-full bg-slate-800 px-2 py-1 text-xs text-slate-300">{formatDisplayText(proposal.status)}</span></div>
        {(proposal.parse_warnings?.length || proposal.validation_errors?.length) ? <div className="mt-3 space-y-1 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-100">{[...(proposal.parse_warnings ?? []), ...(proposal.validation_errors ?? [])].map((message) => <p key={message}>{message}</p>)}</div> : null}
        {proposal.status === "ready" && <div className="mt-3 space-y-2">{ids.map((id) => <label key={id} className="flex items-center gap-2 text-sm text-slate-200"><input type="checkbox" checked={checked.has(id)} onChange={(event) => setSelected((current) => { const next = new Set(current[proposal.id] ?? ids); if (event.target.checked) next.add(id); else next.delete(id); return { ...current, [proposal.id]: next }; })} />{id}</label>)}<input value={note} onChange={(event) => setNote(event.target.value)} placeholder="Decision note (optional)" className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" /><Button size="sm" onClick={() => decide.mutate({ sessionId: session.id, proposalId: proposal.id, ids: [...checked] })} disabled={decide.isPending}><CheckCircle2 className="mr-2 h-4 w-4" />Apply selected ({checked.size})</Button></div>}
        {proposal.status === "needs_revision" && <div className="mt-3"><input value={note} onChange={(event) => setNote(event.target.value)} placeholder="Revision guidance" className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" /><Button className="mt-2" size="sm" variant="outline" onClick={() => revise.mutate({ sessionId: session.id, proposalId: proposal.id })} disabled={revise.isPending}><RefreshCw className="mr-2 h-4 w-4" />Request revision</Button></div>}
      </article>;
    })}
  </section>;
}
