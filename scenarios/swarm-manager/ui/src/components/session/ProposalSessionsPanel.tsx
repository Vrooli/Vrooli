import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, ChevronDown, FileSearch, GitPullRequestArrow, ListChecks, MessageSquarePlus, RefreshCw, XCircle } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { sessionDetailPath } from "../../app/routes/route-paths";
import { selectors } from "../../consts/selectors";
import { formatDisplayText } from "../../lib/format-utils";
import { isNoChangeProposal, parseProposalPayload, proposalMutations, type MutationBaseState } from "../../lib/mutation-archetypes";
import { proposalSessionService, type ProposalSessionProposal, type ProposalSessionTargetType } from "../../services/proposal-session-service";
import type { ProposalMutation } from "../../types/proposal";
import { Button } from "../ui/button";
import { EntityAttachToSessionSheet } from "./context/EntityAttachToSessionSheet";
import { MutationView } from "./MutationView";

interface ProposalSessionsPanelProps {
  target?: { type: ProposalSessionTargetType; ref: string; name: string };
}

/*
 * Payload access goes through `lib/mutation-archetypes`, which parses into the
 * full Proposal shape. A local narrowed type used to live here and dropped
 * every payload field except `reset_scope`, so operators approved mutations
 * whose contents no surface displayed.
 */

function isMutationProposal(proposal: ProposalSessionProposal): boolean {
  return proposal.kind === "mutation_list" && !isNoChangeRecommendation(proposal);
}

function isNoChangeRecommendation(proposal: ProposalSessionProposal): boolean {
  return isNoChangeProposal(proposal);
}

function isDecidable(status: string): boolean {
  return status === "ready";
}

function isRevisable(status: string): boolean {
  return !["applied", "rejected", "superseded"].includes(status);
}

export function ProposalSessionsPanel({ target }: ProposalSessionsPanelProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const queryKey = ["proposal-sessions", target?.type ?? "all", target?.ref ?? "all"];
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedCards, setSelectedCards] = useState<Set<string>>(new Set());
  const [batchNote, setBatchNote] = useState("");
  const [batchPending, setBatchPending] = useState(false);
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [startSheetOpen, setStartSheetOpen] = useState(false);
  const { data: sessions = [], isLoading, error } = useQuery({
    queryKey,
    queryFn: () => proposalSessionService.list(target ? { type: target.type, ref: target.ref } : undefined),
  });
  const invalidate = () => void queryClient.invalidateQueries({ queryKey });
  const decide = useMutation({
    mutationFn: ({ sessionId, proposalId, ids, note }: { sessionId: string; proposalId: string; ids: string[]; note: string }) =>
      proposalSessionService.decide(sessionId, proposalId, ids, note),
    onSuccess: invalidate,
  });
  const acceptKeep = useMutation({
    mutationFn: ({ sessionId, proposalId, note }: { sessionId: string; proposalId: string; note: string }) =>
      proposalSessionService.acceptKeep(sessionId, proposalId, note),
    onSuccess: invalidate,
  });
  const revise = useMutation({
    mutationFn: ({ sessionId, proposalId, note }: { sessionId: string; proposalId: string; note: string }) =>
      proposalSessionService.revise(sessionId, proposalId, note),
    onSuccess: (session) => {
      invalidate();
      navigate(sessionDetailPath(session.id));
    },
  });
  const proposals = useMemo(
    () => sessions.flatMap((session) => (session.proposals ?? [])
      .filter((proposal) => isMutationProposal(proposal) || isNoChangeRecommendation(proposal))
      .map((proposal) => ({ session, proposal }))),
    [sessions],
  );
  const batchableProposals = useMemo(
    () => proposals.filter(({ proposal }) => isMutationProposal(proposal)),
    [proposals],
  );

  const toggleSelectionMode = () => {
    setSelectionMode((current) => !current);
    setSelectedCards(new Set());
    setBatchNote("");
  };

  const batch = async (action: "apply" | "reject" | "revise") => {
    const cards = batchableProposals.filter(({ proposal }) => selectedCards.has(proposal.id)
      && (action === "revise" ? isRevisable(proposal.status) : isDecidable(proposal.status)));
    if (cards.length === 0) return;
    setBatchPending(true);
    try {
      await Promise.all(cards.map(({ session, proposal }) => action === "revise"
        ? proposalSessionService.revise(session.id, proposal.id, batchNote)
        : proposalSessionService.decide(
          session.id,
          proposal.id,
          action === "apply" ? proposalMutations(proposal.payload_json).map((mutation) => mutation.id) : [],
          batchNote,
        )));
      setSelectedCards(new Set());
      setBatchNote("");
      invalidate();
    } finally {
      setBatchPending(false);
    }
  };

  return (
    <section className="space-y-3 py-3" data-testid="proposal-sessions-panel">
      {/* One row, not three: the standing explainer moved into the empty state,
          where it is the only thing an operator actually needs it for. */}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-100">
          Proposals
          {proposals.length > 0 && (
            <span className="rounded-full bg-slate-800 px-2 py-0.5 text-xs font-normal text-slate-400">{proposals.length}</span>
          )}
        </h2>
        <div className="flex flex-wrap gap-2">
          {batchableProposals.length > 1 && (
            <Button size="sm" variant={selectionMode ? "default" : "outline"} onClick={toggleSelectionMode}>
              <ListChecks className="mr-2 h-4 w-4" />
              {selectionMode ? "Done selecting" : "Select proposals"}
            </Button>
          )}
          {target && (
            <Button size="sm" onClick={() => setStartSheetOpen(true)} data-testid={selectors.agentSessions.proposalStart}>
              <MessageSquarePlus className="mr-2 h-4 w-4" />Start proposal
            </Button>
          )}
        </div>
      </div>

      {selectionMode && selectedCards.size > 0 && (
        <div className="rounded-xl border border-violet-400/25 bg-violet-400/5 p-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <p className="text-sm font-medium text-slate-100">{selectedCards.size} proposal{selectedCards.size === 1 ? "" : "s"} selected</p>
              <p className="text-xs text-slate-400">Apply the same decision and note to every selected proposal.</p>
            </div>
          </div>
          <input value={batchNote} onChange={(event) => setBatchNote(event.target.value)} placeholder="Shared decision note (optional)" className="mt-3 w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" />
          <div className="mt-3 flex flex-wrap gap-2">
            <Button size="sm" disabled={batchPending} onClick={() => void batch("apply")}><CheckCircle2 className="mr-2 h-4 w-4" />Apply selected</Button>
            <Button size="sm" variant="outline" disabled={batchPending} onClick={() => void batch("reject")}><XCircle className="mr-2 h-4 w-4" />Reject selected</Button>
            <Button size="sm" variant="outline" disabled={batchPending} onClick={() => void batch("revise")}><RefreshCw className="mr-2 h-4 w-4" />Request revisions</Button>
          </div>
        </div>
      )}

      {target && <EntityAttachToSessionSheet isOpen={startSheetOpen} onClose={() => setStartSheetOpen(false)} option={{ type: target.type, ref: target.ref, title: target.name }} proposalMode />}
      {error && <p className="rounded-md border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-200">{error instanceof Error ? error.message : "Unable to load proposals."}</p>}
      {isLoading ? <p className="py-8 text-sm text-slate-500">Loading proposals…</p> : proposals.length === 0 ? (
        <div className="rounded-lg border border-dashed border-slate-700 p-6 text-center">
          <p className="text-sm text-slate-400">No proposals yet.</p>
          <p className="mx-auto mt-1 max-w-sm text-xs text-slate-500">Proposals carry an agent&apos;s recommended changes, or a documented conclusion that no changes are needed, for you to accept or reject.</p>
        </div>
      ) : proposals.map(({ session, proposal }) => {
        const mutations = proposalMutations(proposal.payload_json);
        const noChangeRecommendation = isNoChangeRecommendation(proposal);
        const canAcceptKeep = noChangeRecommendation && session.proposal_target?.type === "backlog_item";
        const note = notes[proposal.id] ?? "";
        const selected = selectedCards.has(proposal.id);
        return (
          // Flat card: the old icon gutter + nested content column cost ~44px
          // of horizontal space before any text on a phone.
          <article key={proposal.id} className="rounded-xl border border-slate-800 bg-slate-900/60 p-3 sm:p-4">
            <div className="flex flex-wrap items-start gap-x-2 gap-y-1">
              {selectionMode && isMutationProposal(proposal) && (
                <input
                  type="checkbox"
                  aria-label={`Select ${proposal.summary || "proposal"}`}
                  className="mt-1.5 h-4 w-4 shrink-0 accent-violet-400"
                  checked={selected}
                  onChange={(event) => setSelectedCards((current) => {
                    const next = new Set(current);
                    if (event.target.checked) next.add(proposal.id); else next.delete(proposal.id);
                    return next;
                  })}
                />
              )}
              <GitPullRequestArrow className={`mt-1 h-4 w-4 shrink-0 ${noChangeRecommendation ? "text-emerald-300" : "text-violet-300"}`} aria-hidden />
              <h3 className="min-w-0 flex-1 text-base font-medium text-slate-100">{noChangeRecommendation ? proposal.status === "applied" ? "Keep recommendation accepted" : "No changes recommended" : proposal.summary || "Mutation list"}</h3>
              <span className="shrink-0 rounded-full bg-slate-800 px-2 py-0.5 text-xs text-slate-300">{formatDisplayText(proposal.status)}</span>
            </div>
            <button type="button" onClick={() => navigate(sessionDetailPath(session.id))} className="mt-0.5 block truncate text-xs text-slate-400 hover:text-cyan-300">From session: {session.title}</button>
            {noChangeRecommendation ? <NoChangeRecommendation proposal={proposal} /> : <ProposalReview mutations={mutations} proposal={proposal} defaultOpen={proposals.length === 1} />}
            {(proposal.parse_warnings?.length || proposal.validation_errors?.length) ? <div className="mt-3 space-y-1 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-100">{[...(proposal.parse_warnings ?? []), ...(proposal.validation_errors ?? [])].map((message) => <p key={message}>{message}</p>)}</div> : null}
            {isRevisable(proposal.status) && !selectionMode && <input value={note} onChange={(event) => setNotes((current) => ({ ...current, [proposal.id]: event.target.value }))} placeholder={proposal.status === "needs_revision" ? "Revision guidance" : "Decision note (optional)"} className="mt-3 w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" />}
            {!selectionMode && (
              <div className="mt-3 flex flex-wrap gap-2">
                {isMutationProposal(proposal) && isDecidable(proposal.status) && <Button size="sm" onClick={() => decide.mutate({ sessionId: session.id, proposalId: proposal.id, ids: mutations.map((mutation) => mutation.id), note })} disabled={decide.isPending}><CheckCircle2 className="mr-2 h-4 w-4" />Apply proposal</Button>}
                {isMutationProposal(proposal) && isDecidable(proposal.status) && <Button size="sm" variant="outline" onClick={() => decide.mutate({ sessionId: session.id, proposalId: proposal.id, ids: [], note })} disabled={decide.isPending}><XCircle className="mr-2 h-4 w-4" />Reject</Button>}
                {canAcceptKeep && isDecidable(proposal.status) && <Button size="sm" onClick={() => acceptKeep.mutate({ sessionId: session.id, proposalId: proposal.id, note })} disabled={acceptKeep.isPending}><CheckCircle2 className="mr-2 h-4 w-4" />Accept keep recommendation</Button>}
                {isRevisable(proposal.status) && <Button size="sm" variant="outline" onClick={() => revise.mutate({ sessionId: session.id, proposalId: proposal.id, note })} disabled={revise.isPending}><RefreshCw className="mr-2 h-4 w-4" />Request revision</Button>}
              </div>
            )}
          </article>
        );
      })}
    </section>
  );
}

function NoChangeRecommendation({ proposal }: { proposal: ProposalSessionProposal }) {
  const rationale = parseProposalPayload(proposal.payload_json).rationale;
  return (
    <div className="mt-4 rounded-lg border border-emerald-400/25 bg-emerald-400/5 p-3">
      <p className="text-sm font-medium text-emerald-100">{proposal.status === "applied" ? "This keep recommendation was accepted and recorded as a review." : "The review recommends keeping this target as it is."}</p>
      <p className="mt-1 text-xs leading-5 text-slate-300">{proposal.status === "applied" ? "The item remains unchanged; its review freshness was updated independently of its content timestamp." : "No changes will be applied. Accept this conclusion to record a freshness review, or request a revision if you want the session to reassess it."}</p>
      {rationale && <div className="mt-3 border-t border-emerald-400/15 pt-3"><p className="text-xs font-medium uppercase tracking-wide text-emerald-200/80">Reasoning</p><p className="mt-1 text-sm leading-6 text-slate-200">{rationale}</p></div>}
      {proposal.decisions && proposal.decisions.length > 0 && <p className="mt-3 text-xs text-slate-500">Decision history: {proposal.decisions.length} recorded.</p>}
    </div>
  );
}

/**
 * Collapsible change set. Mutations are separated by rules rather than each
 * getting its own bordered card — three nested borders pushed the actual
 * mutation text into a narrow gutter on a phone.
 */
export function ProposalReview({ mutations, proposal, base, defaultOpen = true }: {
  mutations: ProposalMutation[];
  proposal: ProposalSessionProposal;
  /** Current state of the target, so edits render as before/after. */
  base?: MutationBaseState;
  defaultOpen?: boolean;
}) {
  return (
    <details className="mt-3 rounded-lg border border-slate-800 bg-slate-950/35" open={defaultOpen}>
      <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-2 text-sm font-medium text-slate-200">
        <FileSearch className="h-4 w-4 shrink-0 text-cyan-300" />
        Review change set ({mutations.length})
        <ChevronDown className="ml-auto h-4 w-4 shrink-0 text-slate-500" />
      </summary>
      <div className="border-t border-slate-800 px-3 py-2">
        {mutations.length === 0 ? <p className="text-xs text-slate-400">This proposal has no individually reviewable mutations.</p> : (
          <ul className="divide-y divide-slate-800/70">
            {mutations.map((mutation) => (
              <li key={mutation.id} className="py-2.5 first:pt-0 last:pb-0">
                <MutationView mutation={mutation} base={base} />
              </li>
            ))}
          </ul>
        )}
        {proposal.decisions && proposal.decisions.length > 0 && <p className="mt-2 border-t border-slate-800/70 pt-2 text-xs text-slate-500">Decision history: {proposal.decisions.length} recorded.</p>}
      </div>
    </details>
  );
}
