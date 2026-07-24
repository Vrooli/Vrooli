/** A deep-linkable decision flow for questions and mutation proposals. */
import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearchParams } from "react-router-dom";
import { backlogDetailPath, sessionDetailPath } from "../../../app/routes/route-paths";
import { DecisionStreamView } from "../../../components/command-post/DecisionStreamView";
import { ProposalDecisionStreamView, type ProposalDecisionStreamItem } from "../../../components/command-post/ProposalDecisionStreamView";
import { ReviewDecisionCard } from "../../../components/backlog/activity-surface/review-decision-card";
import { RunSheet, type RunSheetTarget } from "../../../components/backlog/run-sheet";
import { Drawer } from "../../../components/ui/drawer";
import { aggregateCrossItemQuestions } from "../../../lib/command-post-utils";
import { backlogService, goalsService } from "../../../services";
import { defaultApiClient } from "../../../lib/api-client";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { proposalSessionService } from "../../../services/proposal-session-service";
import { nextActionService, type NextActionFeedEntry } from "../../../services/next-action-service";
import { reviewService } from "../../../services/review-service";
import { buildActiveBacklogKeys, useBacklogStore } from "../../../stores/backlog-store";
import { useSnoozeStore, useSnoozedKeys } from "../../../stores/snooze-store";
import type { BacklogKind } from "../../../types";

export interface DecisionDrawerProps { isOpen: boolean; onClose: () => void; scopeItemKey: string | null; currentQuestionId: string | null; onCurrentQuestionChange: (id: string | null) => void; onCompleted: () => void; }

export function DecisionDrawer({ isOpen, onClose, scopeItemKey, currentQuestionId, onCurrentQuestionChange, onCompleted }: DecisionDrawerProps) {
  // Retained for the URL-level question deep-link contract. Question cards now
  // resolve through the ranked feed rather than a separate drawer stage.
  void currentQuestionId;
  void onCurrentQuestionChange;
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const snoozedKeys = useSnoozedKeys();
  const snooze = useSnoozeStore((s) => s.snooze);
  const backlogItems = useBacklogStore((s) => s.items);
  const activeItemKeys = useMemo(() => buildActiveBacklogKeys(backlogItems), [backlogItems]);
  const queryClient = useQueryClient();
  const feedQuery = useQuery({ queryKey: ["next-actions-feed", scopeItemKey ?? "all"], queryFn: () => nextActionService.getFeed(), staleTime: 15_000, enabled: isOpen });
  const summaryQuery = useQuery({ queryKey: ["backlog-summary"], queryFn: () => backlogService.getBacklogSummary(), staleTime: 60_000, enabled: isOpen });
  const questions = useMemo(() => {
    const all = aggregateCrossItemQuestions(summaryQuery.data?.pending_questions?.items ?? [], snoozedKeys, activeItemKeys);
    return scopeItemKey ? all.filter((question) => `${question.parentKind}/${question.parentName}` === scopeItemKey) : all;
  }, [summaryQuery.data?.pending_questions, snoozedKeys, activeItemKeys, scopeItemKey]);
  const proposalQuery = useQuery({
    queryKey: ["proposal-sessions", scopeItemKey ?? "all"],
    queryFn: () => proposalSessionService.list(scopeItemKey ? { type: "backlog_item", ref: scopeItemKey } : undefined),
    enabled: isOpen,
  });
  const proposals = useMemo<ProposalDecisionStreamItem[]>(() => (proposalQuery.data ?? []).flatMap((session) => (session.proposals ?? [])
    .filter((proposal) => proposal.kind === "mutation_list" && proposal.status === "ready")
    .map((proposal) => ({ sessionId: session.id, sessionTitle: session.title, proposal, target: session.proposal_target }))), [proposalQuery.data]);
  const feedEntries = useMemo(() => (feedQuery.data?.entries ?? []).filter((entry) => !scopeItemKey || entry.entity_ref === scopeItemKey || entry.chained_ref === scopeItemKey), [feedQuery.data, scopeItemKey]);
  return <Drawer isOpen={isOpen} onClose={onClose} title={scopeItemKey ? "Decisions — one item" : "Decision stream"} className="md:w-[560px]" testId="plan-decision-drawer">
    {feedEntries.length > 0 ? <FeedDecisionStream entries={feedEntries} questions={questions} proposals={proposals} onSnooze={snooze} initialPosition={Number.parseInt(searchParams.get("decisionPosition") ?? "0", 10) || 0} onPositionChange={(position) => { const next = new URLSearchParams(searchParams); next.set("decisionPosition", String(position)); setSearchParams(next, { replace: true }); }} onOpen={(entry) => { if (entry.entity_kind === "backlog_item") { const [kind, name] = entry.entity_ref.split("/"); if (kind && name) navigate(`${backlogDetailPath(kind as BacklogKind, name)}?drawer=decisions&decisionPosition=${searchParams.get("decisionPosition") ?? "0"}`); } else { navigate(`/goals/${entry.entity_ref}?drawer=decisions&decisionPosition=${searchParams.get("decisionPosition") ?? "0"}`); } }} onFeedback={async (entry) => { const session = await proposalSessionService.create({ title: `Feedback for ${entry.entity_title || entry.entity_ref}`, target: { type: entry.entity_kind, ref: entry.entity_ref, name: entry.entity_title || entry.entity_ref } }); navigate(sessionDetailPath(session.id)); }} onChanged={() => { void queryClient.invalidateQueries({ queryKey: ["next-actions-feed"] }); void summaryQuery.refetch(); void proposalQuery.refetch(); onCompleted(); }} />
      : <RankedActionList entries={feedEntries} isLoading={feedQuery.isLoading} onOpen={(entry) => { if (entry.entity_kind === "backlog_item") { const [kind, name] = entry.entity_ref.split("/"); if (kind && name) navigate(backlogDetailPath(kind as BacklogKind, name)); } else { navigate(`/goals/${entry.entity_ref}`); } }} />}
  </Drawer>;
}

function FeedDecisionStream({ entries, questions, proposals, onSnooze, initialPosition, onPositionChange, onOpen, onFeedback, onChanged }: { entries: NextActionFeedEntry[]; questions: ReturnType<typeof aggregateCrossItemQuestions>; proposals: ProposalDecisionStreamItem[]; onSnooze: (key: string, until: number) => void; initialPosition: number; onPositionChange: (position: number) => void; onOpen: (entry: NextActionFeedEntry) => void; onFeedback: (entry: NextActionFeedEntry) => Promise<void>; onChanged: () => void }) {
  const [position, setPosition] = useState(() => Math.max(0, initialPosition));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [runTarget, setRunTarget] = useState<RunSheetTarget | null>(null);
  const boundedPosition = Math.min(position, entries.length - 1);
  const entry = entries[boundedPosition];
  const entityRef = entry?.entity_ref ?? "";
  const actionRef = entry?.chained_ref || (entry?.entity_kind === "backlog_item" ? entityRef : "");
  const backlogRef = actionRef.split("/").filter(Boolean);
  const [backlogKind = "", backlogName = ""] = backlogRef;
  const reviewQuery = useQuery({ queryKey: ["review-rounds", entityRef], queryFn: () => backlogRef.length === 2 ? reviewService.listRounds(backlogKind, backlogName) : Promise.resolve([]), enabled: entry?.action.id === "review" && backlogRef.length === 2 });
  const closeOutQuery = useQuery({ queryKey: ["goal", entityRef, "close-out-evidence"], queryFn: () => goalsService.get(entityRef), enabled: entry?.entity_kind === "goal" && entry.action.id === "close_out" });
  if (!entry) return null;
  const scopedQuestions = entry.entity_kind === "backlog_item" ? questions.filter((question) => `${question.parentKind}/${question.parentName}` === entry.entity_ref) : [];
  const scopedProposals = proposals.filter((proposal) => proposal.target?.ref === entry.entity_ref);
  if (entry.action.id === "decide" && scopedProposals.length > 0) return <section className="p-3" data-testid="next-action-stream-proposal"><div className="mb-3 flex items-center justify-between text-xs text-slate-500"><span>{boundedPosition + 1} of {entries.length}</span><span>Proposal decision</span></div><ProposalDecisionStreamView proposals={scopedProposals} onBack={() => undefined} onComplete={onChanged} onSnooze={(id) => onSnooze(`proposal:${id}`, Date.now() + 3_600_000)} /></section>;
  if (entry.action.id === "decide" && scopedQuestions.length > 0) return <section className="p-3" data-testid="next-action-stream-question"><div className="mb-3 flex items-center justify-between text-xs text-slate-500"><span>{boundedPosition + 1} of {entries.length}</span><span>Question decision</span></div><DecisionStreamView questions={scopedQuestions} onComplete={onChanged} onSnoozeItem={(key) => onSnooze(key, Date.now() + 3_600_000)} onOpenItem={() => onOpen(entry)} currentQuestionId={null} onCurrentQuestionChange={() => undefined} /></section>;
  if (entry.action.id === "review" && backlogRef.length === 2) return <section className="space-y-3 p-4" data-testid="next-action-stream-review"><div className="flex items-center justify-between text-xs text-slate-500"><span>{boundedPosition + 1} of {entries.length}</span><span>Review decision</span></div><ReviewDecisionCard kind={backlogKind} name={backlogName} round={reviewQuery.data?.[0]} onDecided={onChanged} /><button type="button" onClick={() => onOpen(entry)} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300">Open</button></section>;
  if (entry.action.id === "author_followup" && backlogRef.length === 2) return <AuthorFollowUpCard kind={backlogKind as BacklogKind} name={backlogName} onChanged={onChanged} onOpen={() => onOpen(entry)} />;
  const run = async () => {
    setPending(true); setError(null);
    try {
      if (entry.entity_kind === "goal" && entry.action.id === "close_out") await goalsService.closeOut(entry.entity_ref);
      else if (backlogRef.length === 2) {
        const [kind, name] = backlogRef as [BacklogKind, string];
        switch (entry.action.id) {
          case "run": setRunTarget({ kind, name, title: entry.entity_title }); return;
          case "retry": await backlogService.retry(kind, name, "Retried from decision stream"); break;
          case "archive": await backlogService.archiveItem(kind, name); break;
          case "accept_suggestion": await backlogService.update(kind, name, { status: "backlog" }); break;
          case "dispatch_followup": await backlogService.dispatchFollowUp(kind, name); break;
          case "accept_plan": await defaultApiClient.post(API_ENDPOINTS.backlogPlanAccept(kind, name), {}); break;
          case "author_plan": await defaultApiClient.post(API_ENDPOINTS.backlogPlanAuthor(kind, name), {}); break;
          case "repair_plan": await defaultApiClient.post(API_ENDPOINTS.backlogPlanRepair(kind, name), {}); break;
          default: onOpen(entry); return;
        }
      } else { onOpen(entry); return; }
      onChanged();
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to complete this action."); }
    finally { setPending(false); }
  };
  const direct = ["retry", "archive", "accept_suggestion", "dispatch_followup", "accept_plan", "author_plan", "repair_plan", "close_out"].includes(entry.action.id);
  const isGoalPlan = entry.entity_kind === "goal" && entry.action.id === "plan_goal";
  const goalMilestone = entry.action.target?.startsWith("milestone_review:") ? entry.action.target.slice("milestone_review:".length) : "";
  const completeGoalPlan = async () => { setPending(true); setError(null); try { await goalsService.startPlan(entry.entity_ref); onChanged(); } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to start goal planning."); } finally { setPending(false); } };
  const startMilestoneReview = async () => { setPending(true); setError(null); try { await goalsService.startMilestoneReview(entry.entity_ref, goalMilestone); onChanged(); } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to start milestone review."); } finally { setPending(false); } };
  return <section className="space-y-4 p-4" data-testid="next-action-stream-card">
    {runTarget ? <RunSheet isOpen onClose={() => setRunTarget(null)} target={runTarget} onSuccess={onChanged} /> : null}
    <div className="flex items-center justify-between text-xs text-slate-500"><span>{boundedPosition + 1} of {entries.length}</span><span>Tier {entry.tier}</span></div>
    <button type="button" onClick={() => onOpen(entry)} className="block w-full text-left"><h2 className="text-base font-semibold text-slate-100 hover:text-cyan-200">{entry.entity_title || entry.entity_ref}</h2><p className="mt-1 text-xs text-slate-500">{entry.entity_ref}</p></button>
    <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-4"><p className="text-xs font-semibold uppercase tracking-wide text-cyan-300">{entry.action.expanded_label}</p><p className="mt-2 text-sm leading-6 text-slate-300">{entry.action.reason || "This item is ready for an operator action."}</p>{entry.action.id === "resolve_dependencies" && entry.action.blockers?.length ? <div className="mt-3 border-t border-slate-800 pt-3"><p className="text-sm font-medium text-slate-100">Dependencies to resolve</p><ul className="mt-2 space-y-2">{entry.action.blockers.map((blocker) => <li key={`${blocker.code}:${blocker.message}`} className="rounded border border-amber-900/60 bg-amber-950/20 px-3 py-2 text-sm text-amber-100"><span className="font-medium">{blocker.code.replaceAll("_", " ")}</span><span className="block text-xs text-amber-200/80">{blocker.message}</span></li>)}</ul></div> : null}{entry.action.follow_up ? <div className="mt-3 border-t border-slate-800 pt-3 text-sm text-slate-300"><p className="font-medium text-slate-100">Stored recovery direction</p><p className="mt-1">{entry.action.follow_up.steering}</p><p className="mt-2 text-xs uppercase tracking-wide text-cyan-300">{entry.action.follow_up.disposition.replaceAll("_", " ")}{entry.action.follow_up.items?.length ? ` · ${entry.action.follow_up.items.length} proposed item(s)` : ""}</p></div> : null}{entry.action.id === "close_out" && closeOutQuery.data ? <div className="mt-3 border-t border-slate-800 pt-3 text-sm text-slate-300"><p className="font-medium text-slate-100">Milestone evidence</p><ul className="mt-2 space-y-1 text-xs text-slate-400">{closeOutQuery.data.goal.milestones.filter((milestone) => !milestone.archivedAt).map((milestone) => <li key={milestone.name}>✓ {milestone.title} — verified {milestone.verifiedDeliveredAt ? new Date(milestone.verifiedDeliveredAt).toLocaleDateString() : "delivered"}</li>)}</ul></div> : null}{entry.chained_ref ? <p className="mt-3 text-xs text-slate-500">Acts on {entry.chained_ref}</p> : null}</div>
    {error ? <p className="text-sm text-rose-300">{error}</p> : null}
    <div className="flex flex-wrap justify-between gap-2"><div className="flex gap-2"><button type="button" disabled={boundedPosition === 0} onClick={() => { const next = Math.max(0, boundedPosition - 1); setPosition(next); onPositionChange(next); }} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 disabled:opacity-40">Back</button><button type="button" disabled={boundedPosition >= entries.length - 1} onClick={() => { const next = Math.min(entries.length - 1, boundedPosition + 1); setPosition(next); onPositionChange(next); }} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 disabled:opacity-40">Skip</button><button type="button" onClick={() => onSnooze(`${entry.entity_kind}:${entry.entity_ref}`, Date.now() + 3_600_000)} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300">Snooze</button></div><div className="flex gap-2"><button type="button" onClick={() => void onFeedback(entry).catch((cause) => setError(cause instanceof Error ? cause.message : "Unable to start feedback."))} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300">Feedback</button><button type="button" onClick={() => onOpen(entry)} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300">Open</button><button type="button" disabled={pending} onClick={() => { if (isGoalPlan) void completeGoalPlan(); else if (goalMilestone) void startMilestoneReview(); else void run(); }} className="rounded bg-cyan-500 px-3 py-1.5 text-sm font-medium text-slate-950 disabled:opacity-50">{pending ? "Working…" : entry.action.id === "run" ? "Choose run" : direct || isGoalPlan || goalMilestone ? entry.action.compact_label : "Open"}</button></div></div>
  </section>;
}

function AuthorFollowUpCard({ kind, name, onChanged, onOpen }: { kind: BacklogKind; name: string; onChanged: () => void; onOpen: () => void }) {
  const [steering, setSteering] = useState(""); const [disposition, setDisposition] = useState<"follow_up_run" | "replan" | "new_items">("replan"); const [childKind, setChildKind] = useState<BacklogKind>("execute"); const [childName, setChildName] = useState(""); const [childTitle, setChildTitle] = useState(""); const [pending, setPending] = useState(false); const [error, setError] = useState<string | null>(null);
  const needsChild = disposition === "new_items";
  const submit = async () => { setPending(true); setError(null); try { const followUp = { steering, disposition, ...(needsChild ? { items: [{ kind: childKind, name: childName.trim(), title: childTitle.trim() }] } : {}) }; await defaultApiClient.post(API_ENDPOINTS.backlogAuthorFollowUp(kind, name), { follow_up: followUp }); onChanged(); } catch (cause) { setError(cause instanceof Error ? cause.message : "Unable to save follow-up."); } finally { setPending(false); } };
  const canSubmit = steering.trim() && (!needsChild || (childName.trim() && childTitle.trim()));
  return <section className="space-y-3 p-4" data-testid="next-action-stream-author-followup"><p className="text-xs font-semibold uppercase tracking-wide text-cyan-300">Author follow-up</p><p className="text-sm text-slate-300">Record recovery direction, then dispatch it from the next card.</p><textarea value={steering} onChange={(event) => setSteering(event.target.value)} placeholder="Describe the work needed to recover this item…" className="min-h-28 w-full rounded border border-slate-700 bg-slate-950 p-3 text-sm text-slate-100" /><label className="block text-sm text-slate-300">Recovery disposition<select value={disposition} onChange={(event) => setDisposition(event.target.value as typeof disposition)} className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100"><option value="replan">Replan this item</option><option value="follow_up_run">Run a follow-up</option><option value="new_items">Create child work</option></select></label>{needsChild ? <div className="grid gap-2 rounded border border-slate-800 bg-slate-950/60 p-3 sm:grid-cols-2"><label className="text-sm text-slate-300">Kind<select value={childKind} onChange={(event) => setChildKind(event.target.value as BacklogKind)} className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100">{(["idea", "research", "fix", "execute", "chore"] as BacklogKind[]).map((value) => <option key={value} value={value}>{value}</option>)}</select></label><label className="text-sm text-slate-300">Machine name<input value={childName} onChange={(event) => setChildName(event.target.value)} placeholder="recover-evidence" className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100" /></label><label className="text-sm text-slate-300 sm:col-span-2">Title<input value={childTitle} onChange={(event) => setChildTitle(event.target.value)} placeholder="Recover missing evidence" className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100" /></label></div> : null}<div className="flex gap-2"><button type="button" onClick={onOpen} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300">Open</button><button type="button" disabled={pending || !canSubmit} onClick={() => void submit()} className="rounded bg-cyan-500 px-3 py-1.5 text-sm font-medium text-slate-950 disabled:opacity-50">{pending ? "Saving…" : "Save follow-up"}</button></div>{error ? <p className="text-sm text-rose-300">{error}</p> : null}</section>;
}

function RankedActionList({ entries, isLoading, onOpen }: { entries: NextActionFeedEntry[]; isLoading: boolean; onOpen: (entry: NextActionFeedEntry) => void }) {
  if (isLoading) return <p className="py-8 text-center text-sm text-slate-500">Loading operator inbox…</p>;
  if (entries.length === 0) return <div className="py-10 text-center"><p className="text-sm font-medium text-slate-300">Nothing needs your decision.</p><p className="mt-1 text-sm text-slate-500">In-flight work stays visible on the plan board.</p></div>;
  return <div className="space-y-2 p-3" data-testid="ranked-next-action-feed">{entries.map((entry, index) => <article key={`${entry.entity_kind}:${entry.entity_ref}:${entry.action.id}`} className="rounded-lg border border-slate-800 bg-slate-900/40 p-3"><div className="flex items-start gap-3"><span className="mt-0.5 text-xs tabular-nums text-slate-500">{index + 1}</span><div className="min-w-0 flex-1"><p className="truncate text-sm font-medium text-slate-100">{entry.entity_title || entry.entity_ref}</p><p className="mt-1 text-xs font-medium uppercase tracking-wide text-cyan-300">{entry.action.expanded_label}</p>{entry.action.reason && <p className="mt-1 text-sm text-slate-400">{entry.action.reason}</p>}</div><button type="button" onClick={() => onOpen(entry)} className="rounded border border-slate-700 px-2 py-1 text-xs text-slate-300 hover:bg-slate-800">Open</button></div></article>)}</div>;
}
