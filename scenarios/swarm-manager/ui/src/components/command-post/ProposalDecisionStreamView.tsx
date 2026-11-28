import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Archive, ChevronLeft, ChevronRight, GitPullRequestArrow, Loader2, Moon, SkipForward } from "lucide-react";
import { formatDisplayText } from "../../lib/format-utils";
import { daysSince, headlineFor, isNoChangeProposal, mutationSubject, parseProposalPayload, proposalMutations, type MutationBaseState } from "../../lib/mutation-archetypes";
import { backlogService } from "../../services";
import { proposalSessionService, type ProposalSessionProposal } from "../../services/proposal-session-service";
import type { BacklogItem, BacklogKind } from "../../types";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { ProposalReview } from "../session/ProposalSessionsPanel";
import { useAsyncAction } from "../../hooks/useAsyncAction";

/**
 * A read-only inspection older than this is worth flagging: the proposals in
 * this queue reason about current code, and the queue is long enough that
 * weeks pass between authoring and decision.
 */
const STALE_AFTER_DAYS = 7;

/**
 * Projects the target's current state into the shape the mutation renderers
 * diff against. Without it an `update_item` can only show its incoming value,
 * which tells the operator what the text will become but not what it replaces
 * — and the replacement is the part that decides apply-versus-archive.
 */
function baseStateFrom(item: BacklogItem | undefined): MutationBaseState | undefined {
  if (!item) return undefined;
  return {
    patch: {
      title: item.title,
      description: item.description,
      priority: item.priority,
      effort: item.effort,
      tags: item.tags,
      depends_on: item.dependsOn,
      note: item.note,
      acceptance_allow: item.acceptanceAllow,
      acceptance_deny: item.acceptanceDeny,
    },
    status: item.status,
    milestone: item.milestone,
    title: item.title,
  };
}

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
  /** Refetch the queue after an action that changes the underlying item. */
  onItemChanged?: () => void;
  /**
   * Rendered inside the decision drawer's shell, which already shows the
   * entity title and the queue position. Replaces this view's header with a
   * sub-position line, and only when this entry really holds more than one
   * proposal — a bare "1/1" next to the queue counter read as a conflict.
   */
  embedded?: boolean;
}

/** First-class proposal cards for the decision stream, not a panel below it. */
export function ProposalDecisionStreamView({ proposals, onComplete, onBack, onSnooze, onOpenItem, onItemChanged, embedded = false }: Props) {
  const [index, setIndex] = useState(0);
  const [resolved, setResolved] = useState<Set<string>>(() => new Set());
  const [snoozed, setSnoozed] = useState<Set<string>>(() => new Set());
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [note, setNote] = useState("");
  // Inline-only: the card stays on screen after a failure and shows the
  // reason next to the controls that produced it.
  const { pending, error, reset: clearError, fail, run } = useAsyncAction({ toastOnError: false, source: "ProposalDecisionStreamView" });
  const [archiveOpen, setArchiveOpen] = useState(false);
  const active = useMemo(() => proposals.filter(({ proposal }) => !resolved.has(proposal.id) && !snoozed.has(proposal.id)), [proposals, resolved, snoozed]);
  const safeIndex = Math.min(index, Math.max(active.length - 1, 0));
  const current = active[safeIndex];

  useEffect(() => {
    if (!current) { onComplete(); return; }
    setSelected(new Set(proposalMutations(current.proposal.payload_json).map((mutation) => mutation.id)));
    setNote("");
    clearError();
    setArchiveOpen(false);
  }, [current?.proposal.id, clearError]);

  const advance = useCallback(() => {
    if (safeIndex < active.length - 1) setIndex(safeIndex + 1); else onComplete();
  }, [safeIndex, active.length, onComplete]);

  const resolve = useCallback((proposalId: string) => {
    setResolved((previous) => new Set(previous).add(proposalId));
  }, []);

  const runAction = useCallback(async (action: () => Promise<unknown>, failure: string) => {
    if (!current) return;
    if (await run(action, { errorMessage: failure })) resolve(current.proposal.id);
  }, [current, resolve, run]);

  const decide = useCallback((accept: boolean) => {
    if (!current) return;
    void runAction(
      () => proposalSessionService.decide(current.sessionId, current.proposal.id, accept ? [...selected] : [], note),
      "Unable to save proposal decision.",
    );
  }, [current, selected, note, runAction]);

  const acceptKeep = useCallback(() => {
    if (!current) return;
    void runAction(
      () => proposalSessionService.acceptKeep(current.sessionId, current.proposal.id, note),
      "Unable to accept the keep recommendation.",
    );
  }, [current, note, runAction]);

  // Resolve the target's current state so edits render as before/after. The
  // proposal may reference an item that has since been archived, so a failure
  // here degrades to incoming-only rather than blocking the decision.
  const [baseItemKind = "", baseItemName = ""] = (current?.target?.ref ?? "").split("/");
  const baseItemQuery = useQuery({
    queryKey: ["backlog-item", baseItemKind, baseItemName],
    queryFn: () => backlogService.get(baseItemKind as BacklogKind, baseItemName),
    enabled: current?.target?.type === "backlog_item" && Boolean(baseItemKind && baseItemName),
    retry: false,
  });

  const archiveItem = useCallback(() => {
    if (!current?.target) return;
    const [kind, name] = current.target.ref.split("/");
    if (!kind || !name) { fail("This proposal's target is not an archivable item."); return; }
    void runAction(async () => {
      await backlogService.archiveItem(kind as BacklogKind, name);
      setArchiveOpen(false);
      onItemChanged?.();
    }, "Unable to archive this item.");
  }, [current, runAction, onItemChanged, fail]);

  if (!current) return null;

  const mutations = proposalMutations(current.proposal.payload_json);
  const noChange = isNoChangeProposal(current.proposal);
  const isFirst = safeIndex === 0;
  const isLast = safeIndex === active.length - 1;
  const payload = parseProposalPayload(current.proposal.payload_json);
  const rationale = payload.rationale;
  const targetName = current.target?.name || current.proposal.summary || "Mutation list";
  const subtitle = current.proposal.summary && current.proposal.summary !== targetName ? current.proposal.summary : current.sessionTitle !== targetName ? current.sessionTitle : "";
  const age = daysSince(current.proposal.created_at);
  const archivable = current.target?.type === "backlog_item" && current.target.ref.split("/").filter(Boolean).length === 2;

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

    <div className="flex-1 overflow-y-auto px-3 py-3">
      <div className="mx-auto flex max-w-2xl flex-col gap-3">
        <p className="text-sm text-slate-400">{noChange ? "This session concluded that no change is needed." : "Choose which proposed mutations to apply."}</p>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex rounded-full bg-slate-800 px-2 py-1 text-xs text-slate-300">{formatDisplayText(current.proposal.status)}</span>
          {/* The proposals here reason about code as it was when they were
              authored. Saying how long ago that was is the difference between
              trusting a finding and re-checking it. */}
          {age !== undefined && age >= STALE_AFTER_DAYS && (
            <span className="inline-flex rounded-full border border-amber-400/25 bg-amber-400/10 px-2 py-1 text-xs text-amber-100" data-testid="proposal-staleness">
              Inspected {age} days ago — code may have drifted since
            </span>
          )}
        </div>

        {rationale && <p className="rounded border border-violet-400/20 bg-violet-400/5 p-3 text-sm leading-6 text-slate-300">{rationale}</p>}

        {noChange
          ? <div className="rounded-lg border border-emerald-400/25 bg-emerald-400/5 p-3">
            <p className="text-sm font-medium text-emerald-100">No changes recommended</p>
            <p className="mt-1 text-xs leading-5 text-slate-300">Accepting records a freshness review without altering the item.</p>
          </div>
          : <ProposalReview mutations={mutations} proposal={current.proposal} base={baseStateFrom(baseItemQuery.data)} />}

        {!noChange && mutations.length > 1 && mutations.map((mutation) => (
          <label key={mutation.id} className="flex items-center gap-2 rounded border border-slate-800 bg-slate-900/60 p-3 text-sm text-slate-200">
            <input
              type="checkbox"
              checked={selected.has(mutation.id)}
              onChange={(event) => setSelected((previous) => {
                const next = new Set(previous);
                if (event.target.checked) next.add(mutation.id); else next.delete(mutation.id);
                return next;
              })}
            />
            {/* Same wording as the card above it: "Apply Create item", not
                "Apply Add item" for the row the operator just read. */}
            <span>Apply {headlineFor(mutation.op).toLowerCase()}{mutationSubject(mutation) ? ` · ${mutationSubject(mutation)}` : ""}</span>
          </label>
        ))}

        <input value={note} onChange={(event) => setNote(event.target.value)} placeholder="Decision note (optional)" className="w-full rounded-md border border-slate-700 bg-slate-950 px-3 py-2 text-sm" />

        <div className="flex flex-wrap gap-2">
          {noChange
            ? <button type="button" disabled={pending} onClick={acceptKeep} className="rounded bg-emerald-600 px-3 py-2 text-sm text-white disabled:opacity-50">Accept keep recommendation</button>
            : <>
              <button type="button" disabled={pending} onClick={() => decide(true)} className="rounded bg-violet-600 px-3 py-2 text-sm text-white disabled:opacity-50">{mutations.length === 1 ? "Apply" : `Apply selected (${selected.size})`}</button>
              <button type="button" disabled={pending} onClick={() => decide(false)} className="rounded border border-slate-700 px-3 py-2 text-sm text-slate-300 disabled:opacity-50">Reject proposal</button>
            </>}
          {/* Rejecting a proposal leaves the item in the queue to be proposed
              against again. When the item itself is what should go, that has
              to be reachable from here. */}
          {archivable && (
            <button type="button" disabled={pending} onClick={() => setArchiveOpen(true)} className="flex items-center gap-1.5 rounded border border-rose-400/30 px-3 py-2 text-sm text-rose-200 disabled:opacity-50" data-testid="proposal-archive-item">
              <Archive className="h-4 w-4" aria-hidden />Archive item…
            </button>
          )}
          {pending && <Loader2 className="h-5 w-5 animate-spin text-violet-300" />}
        </div>
        {error && <p className="text-sm text-red-400">{error}</p>}
      </div>
    </div>

    {/* Moving between proposals only makes sense when there is more than one;
        embedded, queue-level movement belongs to the drawer footer. */}
    {embedded && active.length <= 1 ? null : (
      <div className="border-t border-slate-700/50 px-3 py-2"><div className="mx-auto flex max-w-2xl items-center justify-between"><button type="button" onClick={() => isFirst ? onBack() : setIndex(safeIndex - 1)} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400"><ChevronLeft className="h-4 w-4" />Back</button><div className="flex items-center gap-1"><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><SkipForward className="h-4 w-4" />Skip</button><button type="button" onClick={() => { setSnoozed((previous) => new Set(previous).add(current.proposal.id)); onSnooze(current.proposal.id); }} className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500"><Moon className="h-4 w-4" />Snooze</button></div><button type="button" onClick={advance} className="flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm text-slate-400">{isLast ? "Done" : "Next"}<ChevronRight className="h-4 w-4" /></button></div></div>
    )}

    <ConfirmDialog
      isOpen={archiveOpen}
      onClose={() => setArchiveOpen(false)}
      onConfirm={archiveItem}
      title="Archive this item?"
      description={`${targetName} will be archived and leave the decision queue. Any items depending on it keep their edges until retargeted. This is reversible with the unarchive action.`}
      confirmLabel="Archive item"
      isLoading={pending}
      testIds={{ dialog: "proposal-archive-confirm" }}
    />
  </div>;
}
