/**
 * DecisionDrawer — a deep-linkable decision flow over the ranked next-action
 * feed, for questions, mutation proposals, reviews, and direct actions.
 *
 * The drawer owns queue position and queue navigation; the per-entry cards own
 * only their decision controls. That split exists because the two used to be
 * tangled: the proposal, question, and review branches rendered a queue counter
 * with no way to move past the entry, so the queue dead-ended on its first
 * proposal, and each card printed a second counter beside the first.
 */
import { useCallback, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ChevronLeft, ExternalLink, ListOrdered, MessageSquarePlus, Moon, SkipForward } from "lucide-react";
import { backlogDetailPath, sessionDetailPath } from "../../../app/routes/route-paths";
import { DecisionStreamView } from "../../../components/command-post/DecisionStreamView";
import { ProposalDecisionStreamView, type ProposalDecisionStreamItem } from "../../../components/command-post/ProposalDecisionStreamView";
import { ReviewDecisionCard } from "../../../components/backlog/activity-surface/review-decision-card";
import { RunSheet, type RunSheetTarget } from "../../../components/backlog/run-sheet";
import { Drawer } from "../../../components/ui/drawer";
import { Input } from "../../../components/ui/input";
import { Popover } from "../../../components/ui/popover";
import { aggregateCrossItemQuestions } from "../../../lib/command-post-utils";
import { backlogService, goalsService } from "../../../services";
import { defaultApiClient } from "../../../lib/api-client";
import { API_ENDPOINTS } from "../../../lib/api-endpoints";
import { proposalSessionService } from "../../../services/proposal-session-service";
import { nextActionService, type NextActionFeedEntry } from "../../../services/next-action-service";
import { NEXT_ACTION_FEED_QUERY_KEY } from "../../../hooks/usePendingDecisionCount";
import { reviewService } from "../../../services/review-service";
import { buildActiveBacklogKeys, useBacklogStore } from "../../../stores/backlog-store";
import { useSnoozeStore, useSnoozedKeys } from "../../../stores/snooze-store";
import { nextActionIcon } from "../../../types/constants";
import type { BacklogKind } from "../../../types";

const SNOOZE_MS = 3_600_000;
/** Rows rendered in the jump list before asking the operator to filter. */
const NAVIGATOR_VISIBLE_LIMIT = 100;

type EntryVariant = "proposal" | "question" | "review" | "followup" | "action";

export interface DecisionDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  scopeItemKey: string | null;
  currentQuestionId: string | null;
  onCurrentQuestionChange: (id: string | null) => void;
  onCompleted: () => void;
}

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

  const feedQuery = useQuery({
    queryKey: NEXT_ACTION_FEED_QUERY_KEY,
    queryFn: () => nextActionService.getFeed(),
    staleTime: 15_000,
    enabled: isOpen,
  });
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    enabled: isOpen,
  });
  const proposalQuery = useQuery({
    queryKey: ["proposal-sessions", scopeItemKey ?? "all"],
    queryFn: () => proposalSessionService.list(scopeItemKey ? { type: "backlog_item", ref: scopeItemKey } : undefined),
    enabled: isOpen,
  });

  const questions = useMemo(() => {
    const all = aggregateCrossItemQuestions(summaryQuery.data?.pending_questions?.items ?? [], snoozedKeys, activeItemKeys);
    return scopeItemKey ? all.filter((question) => `${question.parentKind}/${question.parentName}` === scopeItemKey) : all;
  }, [summaryQuery.data?.pending_questions, snoozedKeys, activeItemKeys, scopeItemKey]);

  const proposals = useMemo<ProposalDecisionStreamItem[]>(
    () => (proposalQuery.data ?? []).flatMap((session) => (session.proposals ?? [])
      .filter((proposal) => proposal.kind === "mutation_list" && proposal.status === "ready")
      .map((proposal) => ({ sessionId: session.id, sessionTitle: session.title, proposal, target: session.proposal_target }))),
    [proposalQuery.data],
  );

  const entries = useMemo(
    () => (feedQuery.data?.entries ?? []).filter((entry) => !scopeItemKey || entry.entity_ref === scopeItemKey || entry.chained_ref === scopeItemKey),
    [feedQuery.data, scopeItemKey],
  );

  // Queue position lives here, not in the cards, so every entry type moves the
  // same way and the counter has exactly one owner.
  const urlPosition = Number.parseInt(searchParams.get("decisionPosition") ?? "0", 10) || 0;
  const [position, setPosition] = useState(() => Math.max(0, urlPosition));
  const boundedPosition = entries.length > 0 ? Math.min(Math.max(position, 0), entries.length - 1) : 0;
  const entry = entries[boundedPosition];

  const goToPosition = useCallback((next: number) => {
    const clamped = Math.max(0, Math.min(next, Math.max(entries.length - 1, 0)));
    setPosition(clamped);
    const params = new URLSearchParams(searchParams);
    params.set("decisionPosition", String(clamped));
    setSearchParams(params, { replace: true });
  }, [entries.length, searchParams, setSearchParams]);

  const openEntry = useCallback((target: NextActionFeedEntry) => {
    const suffix = `?drawer=decisions&decisionPosition=${boundedPosition}`;
    if (target.entity_kind === "backlog_item") {
      const [kind, name] = target.entity_ref.split("/");
      if (kind && name) navigate(`${backlogDetailPath(kind as BacklogKind, name)}${suffix}`);
      return;
    }
    navigate(`/goals/${target.entity_ref}${suffix}`);
  }, [boundedPosition, navigate]);

  const handleChanged = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: NEXT_ACTION_FEED_QUERY_KEY });
    void summaryQuery.refetch();
    void proposalQuery.refetch();
    onCompleted();
  }, [onCompleted, proposalQuery, queryClient, summaryQuery]);

  const scopedQuestions = useMemo(
    () => (entry?.entity_kind === "backlog_item"
      ? questions.filter((question) => `${question.parentKind}/${question.parentName}` === entry.entity_ref)
      : []),
    [entry, questions],
  );
  const scopedProposals = useMemo(
    () => proposals.filter((proposal) => proposal.target?.ref === entry?.entity_ref),
    [entry, proposals],
  );

  const variant = entry ? entryVariant(entry, scopedQuestions.length, scopedProposals.length) : "action";

  const footer = entry ? (
    <QueueNavBar
      position={boundedPosition}
      total={entries.length}
      onPrevious={() => goToPosition(boundedPosition - 1)}
      onSkip={() => goToPosition(boundedPosition + 1)}
      onSnooze={() => snooze(`${entry.entity_kind}:${entry.entity_ref}`, Date.now() + SNOOZE_MS)}
    />
  ) : undefined;

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={scopeItemKey ? "Decisions — one item" : "Decision stream"}
      className="md:w-[560px]"
      testId="plan-decision-drawer"
      footer={footer}
    >
      {entry ? (
        <div className="flex h-full min-h-0 flex-col">
          <QueueHeader
            position={boundedPosition}
            entries={entries}
            entry={entry}
            variantLabel={VARIANT_LABELS[variant]}
            onJump={goToPosition}
            onOpen={() => openEntry(entry)}
          />
          <div className="min-h-0 flex-1">
            <EntryBody
              key={`${entry.entity_kind}:${entry.entity_ref}:${entry.action.id}`}
              entry={entry}
              variant={variant}
              questions={scopedQuestions}
              proposals={scopedProposals}
              onSnooze={snooze}
              onOpen={() => openEntry(entry)}
              onChanged={handleChanged}
              onFeedback={async () => {
                const session = await proposalSessionService.create({
                  title: `Feedback for ${entry.entity_title || entry.entity_ref}`,
                  target: { type: entry.entity_kind, ref: entry.entity_ref, name: entry.entity_title || entry.entity_ref },
                });
                navigate(sessionDetailPath(session.id));
              }}
            />
          </div>
        </div>
      ) : (
        <RankedActionList entries={entries} isLoading={feedQuery.isLoading} onOpen={openEntry} />
      )}
    </Drawer>
  );
}

const VARIANT_LABELS: Record<EntryVariant, string> = {
  proposal: "Proposal decision",
  question: "Question decision",
  review: "Review decision",
  followup: "Follow-up",
  action: "Operator action",
};

function entryVariant(entry: NextActionFeedEntry, questionCount: number, proposalCount: number): EntryVariant {
  const backlogRef = (entry.chained_ref || (entry.entity_kind === "backlog_item" ? entry.entity_ref : "")).split("/").filter(Boolean);
  if (entry.action.id === "decide" && proposalCount > 0) return "proposal";
  if (entry.action.id === "decide" && questionCount > 0) return "question";
  if (entry.action.id === "review" && backlogRef.length === 2) return "review";
  if (entry.action.id === "author_followup" && backlogRef.length === 2) return "followup";
  return "action";
}

/**
 * The queue's single position readout. Pressing it opens the jump list — the
 * counter was previously inert text sitting beside a second, unrelated one.
 */
function QueueHeader({ position, entries, entry, variantLabel, onJump, onOpen }: {
  position: number;
  entries: NextActionFeedEntry[];
  entry: NextActionFeedEntry;
  variantLabel: string;
  onJump: (position: number) => void;
  onOpen: () => void;
}) {
  const [navigatorOpen, setNavigatorOpen] = useState(false);
  const [anchor, setAnchor] = useState({ x: 0, y: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);

  const openNavigator = () => {
    const rect = buttonRef.current?.getBoundingClientRect();
    if (rect) setAnchor({ x: Math.max(8, rect.left - 220), y: rect.bottom + 4 });
    setNavigatorOpen(true);
  };

  return (
    <div className="shrink-0 border-b border-slate-800 px-3 py-2" data-testid="decision-queue-header">
      <div className="flex items-center justify-between gap-2">
        <button
          ref={buttonRef}
          type="button"
          onClick={openNavigator}
          className="flex items-center gap-1.5 rounded px-1.5 py-0.5 text-xs tabular-nums text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200"
          title="Jump to another decision"
          aria-label={`Decision ${position + 1} of ${entries.length}. Jump to another decision.`}
          data-testid="decision-queue-counter"
        >
          <ListOrdered className="h-3.5 w-3.5" aria-hidden />
          {position + 1} of {entries.length}
        </button>
        <span className="text-xs text-slate-500">{variantLabel}</span>
      </div>
      <button
        type="button"
        onClick={onOpen}
        className="mt-1 block w-full text-left text-sm font-medium text-cyan-300 line-clamp-2 hover:text-cyan-200 hover:underline"
        title={entry.entity_title || entry.entity_ref}
        data-testid="decision-queue-title"
      >
        {entry.entity_title || entry.entity_ref}
      </button>

      <QueueNavigatorPopover
        isOpen={navigatorOpen}
        onClose={() => setNavigatorOpen(false)}
        anchor={anchor}
        entries={entries}
        position={position}
        onJump={(next) => {
          onJump(next);
          setNavigatorOpen(false);
        }}
      />
    </div>
  );
}

function QueueNavigatorPopover({ isOpen, onClose, anchor, entries, position, onJump }: {
  isOpen: boolean;
  onClose: () => void;
  anchor: { x: number; y: number };
  entries: NextActionFeedEntry[];
  position: number;
  onJump: (position: number) => void;
}) {
  const [filter, setFilter] = useState("");

  const matches = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    const indexed = entries.map((entry, index) => ({ entry, index }));
    if (!needle) return indexed;
    return indexed.filter(({ entry }) =>
      (entry.entity_title || "").toLowerCase().includes(needle)
      || entry.entity_ref.toLowerCase().includes(needle)
      || entry.action.expanded_label.toLowerCase().includes(needle));
  }, [entries, filter]);

  const visible = matches.slice(0, NAVIGATOR_VISIBLE_LIMIT);

  return (
    <Popover
      isOpen={isOpen}
      onClose={onClose}
      x={anchor.x}
      y={anchor.y}
      className="w-[min(90vw,22rem)] p-2"
      testId="decision-queue-navigator"
    >
      <Input
        size="sm"
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
        placeholder="Filter decisions"
        aria-label="Filter decisions"
      />
      <div className="mt-2 max-h-[50vh] space-y-1 overflow-y-auto">
        {visible.map(({ entry, index }) => (
          <button
            key={`${entry.entity_kind}:${entry.entity_ref}:${entry.action.id}`}
            type="button"
            onClick={() => onJump(index)}
            className={`flex w-full items-start gap-2 rounded px-2 py-1.5 text-left text-xs hover:bg-slate-800 ${index === position ? "bg-slate-800 text-slate-100" : "text-slate-300"}`}
            data-testid="decision-queue-navigator-row"
          >
            <span className="mt-0.5 shrink-0 tabular-nums text-slate-500">{index + 1}</span>
            <span className="min-w-0 flex-1">
              <span className="block truncate">{entry.entity_title || entry.entity_ref}</span>
              <span className="block truncate text-[11px] text-slate-500">{entry.action.expanded_label}</span>
            </span>
          </button>
        ))}
        {visible.length === 0 && (
          <p className="px-2 py-3 text-xs text-slate-500">No decisions match that filter.</p>
        )}
      </div>
      {matches.length > visible.length && (
        <p className="mt-2 border-t border-slate-800 px-2 pt-2 text-[11px] text-slate-500">
          Showing {visible.length} of {matches.length} matches — refine the filter to see the rest.
        </p>
      )}
    </Popover>
  );
}

/**
 * Queue-level movement, present for every entry type. Labelled "item" so it
 * never reads as the per-decision Back/Next inside a card.
 */
function QueueNavBar({ position, total, onPrevious, onSkip, onSnooze }: {
  position: number;
  total: number;
  onPrevious: () => void;
  onSkip: () => void;
  onSnooze: () => void;
}) {
  const navButton = "flex min-h-[40px] items-center gap-1.5 rounded-lg border border-slate-700 px-3 py-1.5 text-sm text-slate-300 transition-colors hover:bg-slate-800 disabled:opacity-40";
  return (
    <div className="flex items-center justify-between gap-2" data-testid="decision-queue-nav">
      <button
        type="button"
        disabled={position === 0}
        onClick={onPrevious}
        className={navButton}
        data-testid="decision-queue-previous"
      >
        <ChevronLeft className="h-4 w-4" aria-hidden />
        Previous
      </button>
      <button type="button" onClick={onSnooze} className={navButton} data-testid="decision-queue-snooze">
        <Moon className="h-4 w-4" aria-hidden />
        Snooze item
      </button>
      <button
        type="button"
        disabled={position >= total - 1}
        onClick={onSkip}
        className={navButton}
        data-testid="decision-queue-skip"
      >
        <SkipForward className="h-4 w-4" aria-hidden />
        Skip
      </button>
    </div>
  );
}

function EntryBody({ entry, variant, questions, proposals, onSnooze, onOpen, onChanged, onFeedback }: {
  entry: NextActionFeedEntry;
  variant: EntryVariant;
  questions: ReturnType<typeof aggregateCrossItemQuestions>;
  proposals: ProposalDecisionStreamItem[];
  onSnooze: (key: string, until: number) => void;
  onOpen: () => void;
  onChanged: () => void;
  onFeedback: () => Promise<void>;
}) {
  const backlogRef = (entry.chained_ref || (entry.entity_kind === "backlog_item" ? entry.entity_ref : "")).split("/").filter(Boolean);
  const [backlogKind = "", backlogName = ""] = backlogRef;

  const reviewQuery = useQuery({
    queryKey: ["review-rounds", entry.entity_ref],
    queryFn: () => (backlogRef.length === 2 ? reviewService.listRounds(backlogKind, backlogName) : Promise.resolve([])),
    enabled: variant === "review",
  });
  const closeOutQuery = useQuery({
    queryKey: ["goal", entry.entity_ref, "close-out-evidence"],
    queryFn: () => goalsService.get(entry.entity_ref),
    enabled: entry.entity_kind === "goal" && entry.action.id === "close_out",
  });

  if (variant === "proposal") {
    return (
      <div className="h-full" data-testid="next-action-stream-proposal">
        <ProposalDecisionStreamView
          embedded
          proposals={proposals}
          onBack={() => undefined}
          onComplete={onChanged}
          onSnooze={(id) => onSnooze(`proposal:${id}`, Date.now() + SNOOZE_MS)}
          onOpenItem={onOpen}
        />
      </div>
    );
  }

  if (variant === "question") {
    return (
      <div className="h-full" data-testid="next-action-stream-question">
        <DecisionStreamView
          embedded
          questions={questions}
          onComplete={onChanged}
          onSnoozeItem={(key) => onSnooze(key, Date.now() + SNOOZE_MS)}
          onOpenItem={onOpen}
          currentQuestionId={null}
          onCurrentQuestionChange={() => undefined}
        />
      </div>
    );
  }

  if (variant === "review") {
    return (
      <section className="space-y-3 p-4" data-testid="next-action-stream-review">
        <ReviewDecisionCard kind={backlogKind} name={backlogName} round={reviewQuery.data?.[0]} onDecided={onChanged} />
        <button type="button" onClick={onOpen} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800">
          Open
        </button>
      </section>
    );
  }

  if (variant === "followup") {
    return <AuthorFollowUpCard kind={backlogKind as BacklogKind} name={backlogName} onChanged={onChanged} onOpen={onOpen} />;
  }

  return (
    <ActionCard
      entry={entry}
      backlogRef={backlogRef}
      closeOutMilestones={closeOutQuery.data?.goal.milestones}
      onChanged={onChanged}
      onOpen={onOpen}
      onFeedback={onFeedback}
    />
  );
}

function ActionCard({ entry, backlogRef, closeOutMilestones, onChanged, onOpen, onFeedback }: {
  entry: NextActionFeedEntry;
  backlogRef: string[];
  closeOutMilestones?: Array<{ name: string; title: string; archivedAt?: string | null; verifiedDeliveredAt?: string | null }>;
  onChanged: () => void;
  onOpen: () => void;
  onFeedback: () => Promise<void>;
}) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [runTarget, setRunTarget] = useState<RunSheetTarget | null>(null);

  const isGoalPlan = entry.entity_kind === "goal" && entry.action.id === "plan_goal";
  const goalMilestone = entry.action.target?.startsWith("milestone_review:") ? entry.action.target.slice("milestone_review:".length) : "";
  const direct = ["retry", "archive", "accept_suggestion", "dispatch_followup", "accept_plan", "author_plan", "repair_plan", "close_out"].includes(entry.action.id);

  const run = async () => {
    setPending(true);
    setError(null);
    try {
      if (entry.entity_kind === "goal" && entry.action.id === "close_out") {
        await goalsService.closeOut(entry.entity_ref);
      } else if (backlogRef.length === 2) {
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
          default: onOpen(); return;
        }
      } else {
        onOpen();
        return;
      }
      onChanged();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to complete this action.");
    } finally {
      setPending(false);
    }
  };

  const completeGoalPlan = async () => {
    setPending(true);
    setError(null);
    try {
      await goalsService.startPlan(entry.entity_ref);
      onChanged();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to start goal planning.");
    } finally {
      setPending(false);
    }
  };

  const startMilestoneReview = async () => {
    setPending(true);
    setError(null);
    try {
      await goalsService.startMilestoneReview(entry.entity_ref, goalMilestone);
      onChanged();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to start milestone review.");
    } finally {
      setPending(false);
    }
  };

  const PrimaryIcon = nextActionIcon(entry.action.id);
  const primaryLabel = pending
    ? "Working…"
    : entry.action.id === "run"
      ? "Choose run"
      : direct || isGoalPlan || goalMilestone
        ? entry.action.compact_label
        : "Open";

  return (
    <section className="space-y-4 p-4" data-testid="next-action-stream-card">
      {runTarget ? <RunSheet isOpen onClose={() => setRunTarget(null)} target={runTarget} onSuccess={onChanged} /> : null}
      <p className="text-xs text-slate-500">{entry.entity_ref} · Tier {entry.tier}</p>

      <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-4">
        <p className="text-xs font-semibold uppercase tracking-wide text-cyan-300">{entry.action.expanded_label}</p>
        <p className="mt-2 text-sm leading-6 text-slate-300">{entry.action.reason || "This item is ready for an operator action."}</p>

        {entry.action.id === "resolve_dependencies" && entry.action.blockers?.length ? (
          <div className="mt-3 border-t border-slate-800 pt-3">
            <p className="text-sm font-medium text-slate-100">Dependencies to resolve</p>
            <ul className="mt-2 space-y-2">
              {entry.action.blockers.map((blocker) => (
                <li key={`${blocker.code}:${blocker.message}`} className="rounded border border-amber-900/60 bg-amber-950/20 px-3 py-2 text-sm text-amber-100">
                  <span className="font-medium">{blocker.code.replaceAll("_", " ")}</span>
                  <span className="block text-xs text-amber-200/80">{blocker.message}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {entry.action.follow_up ? (
          <div className="mt-3 border-t border-slate-800 pt-3 text-sm text-slate-300">
            <p className="font-medium text-slate-100">Stored recovery direction</p>
            <p className="mt-1">{entry.action.follow_up.steering}</p>
            <p className="mt-2 text-xs uppercase tracking-wide text-cyan-300">
              {entry.action.follow_up.disposition.replaceAll("_", " ")}
              {entry.action.follow_up.items?.length ? ` · ${entry.action.follow_up.items.length} proposed item(s)` : ""}
            </p>
          </div>
        ) : null}

        {entry.action.id === "close_out" && closeOutMilestones ? (
          <div className="mt-3 border-t border-slate-800 pt-3 text-sm text-slate-300">
            <p className="font-medium text-slate-100">Milestone evidence</p>
            <ul className="mt-2 space-y-1 text-xs text-slate-400">
              {closeOutMilestones.filter((milestone) => !milestone.archivedAt).map((milestone) => (
                <li key={milestone.name}>
                  ✓ {milestone.title} — verified {milestone.verifiedDeliveredAt ? new Date(milestone.verifiedDeliveredAt).toLocaleDateString() : "delivered"}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {entry.chained_ref ? <p className="mt-3 text-xs text-slate-500">Acts on {entry.chained_ref}</p> : null}
      </div>

      {error ? <p className="text-sm text-rose-300">{error}</p> : null}

      <div className="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          onClick={() => void onFeedback().catch((cause) => setError(cause instanceof Error ? cause.message : "Unable to start feedback."))}
          className="flex items-center gap-1.5 rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800"
        >
          <MessageSquarePlus className="h-4 w-4" aria-hidden />
          Feedback
        </button>
        <button type="button" onClick={onOpen} className="flex items-center gap-1.5 rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800">
          <ExternalLink className="h-4 w-4" aria-hidden />
          Open
        </button>
        <button
          type="button"
          disabled={pending}
          onClick={() => {
            if (isGoalPlan) void completeGoalPlan();
            else if (goalMilestone) void startMilestoneReview();
            else void run();
          }}
          className="flex items-center gap-1.5 rounded bg-cyan-500 px-3 py-1.5 text-sm font-medium text-slate-950 hover:bg-cyan-400 disabled:opacity-50"
        >
          <PrimaryIcon className="h-4 w-4" aria-hidden />
          {primaryLabel}
        </button>
      </div>
    </section>
  );
}

function AuthorFollowUpCard({ kind, name, onChanged, onOpen }: { kind: BacklogKind; name: string; onChanged: () => void; onOpen: () => void }) {
  const [steering, setSteering] = useState("");
  const [disposition, setDisposition] = useState<"follow_up_run" | "replan" | "new_items">("replan");
  const [childKind, setChildKind] = useState<BacklogKind>("execute");
  const [childName, setChildName] = useState("");
  const [childTitle, setChildTitle] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const needsChild = disposition === "new_items";

  const submit = async () => {
    setPending(true);
    setError(null);
    try {
      const followUp = { steering, disposition, ...(needsChild ? { items: [{ kind: childKind, name: childName.trim(), title: childTitle.trim() }] } : {}) };
      await defaultApiClient.post(API_ENDPOINTS.backlogAuthorFollowUp(kind, name), { follow_up: followUp });
      onChanged();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Unable to save follow-up.");
    } finally {
      setPending(false);
    }
  };

  const canSubmit = steering.trim() && (!needsChild || (childName.trim() && childTitle.trim()));

  return (
    <section className="space-y-3 p-4" data-testid="next-action-stream-author-followup">
      <p className="text-xs font-semibold uppercase tracking-wide text-cyan-300">Author follow-up</p>
      <p className="text-sm text-slate-300">Record recovery direction, then dispatch it from the next card.</p>
      <textarea
        value={steering}
        onChange={(event) => setSteering(event.target.value)}
        placeholder="Describe the work needed to recover this item…"
        className="min-h-28 w-full rounded border border-slate-700 bg-slate-950 p-3 text-sm text-slate-100"
      />
      <label className="block text-sm text-slate-300">
        Recovery disposition
        <select
          value={disposition}
          onChange={(event) => setDisposition(event.target.value as typeof disposition)}
          className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100"
        >
          <option value="replan">Replan this item</option>
          <option value="follow_up_run">Run a follow-up</option>
          <option value="new_items">Create child work</option>
        </select>
      </label>
      {needsChild ? (
        <div className="grid gap-2 rounded border border-slate-800 bg-slate-950/60 p-3 sm:grid-cols-2">
          <label className="text-sm text-slate-300">
            Kind
            <select value={childKind} onChange={(event) => setChildKind(event.target.value as BacklogKind)} className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100">
              {(["idea", "research", "fix", "execute", "chore"] as BacklogKind[]).map((value) => <option key={value} value={value}>{value}</option>)}
            </select>
          </label>
          <label className="text-sm text-slate-300">
            Machine name
            <input value={childName} onChange={(event) => setChildName(event.target.value)} placeholder="recover-evidence" className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100" />
          </label>
          <label className="text-sm text-slate-300 sm:col-span-2">
            Title
            <input value={childTitle} onChange={(event) => setChildTitle(event.target.value)} placeholder="Recover missing evidence" className="mt-1 block w-full rounded border border-slate-700 bg-slate-950 p-2 text-sm text-slate-100" />
          </label>
        </div>
      ) : null}
      <div className="flex gap-2">
        <button type="button" onClick={onOpen} className="rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800">Open</button>
        <button type="button" disabled={pending || !canSubmit} onClick={() => void submit()} className="rounded bg-cyan-500 px-3 py-1.5 text-sm font-medium text-slate-950 disabled:opacity-50">
          {pending ? "Saving…" : "Save follow-up"}
        </button>
      </div>
      {error ? <p className="text-sm text-rose-300">{error}</p> : null}
    </section>
  );
}

function RankedActionList({ entries, isLoading, onOpen }: { entries: NextActionFeedEntry[]; isLoading: boolean; onOpen: (entry: NextActionFeedEntry) => void }) {
  if (isLoading) return <p className="py-8 text-center text-sm text-slate-500">Loading operator inbox…</p>;
  if (entries.length === 0) {
    return (
      <div className="py-10 text-center">
        <p className="text-sm font-medium text-slate-300">Nothing needs your decision.</p>
        <p className="mt-1 text-sm text-slate-500">In-flight work stays visible on the plan board.</p>
      </div>
    );
  }
  return (
    <div className="space-y-2 p-3" data-testid="ranked-next-action-feed">
      {entries.map((entry, index) => (
        <article key={`${entry.entity_kind}:${entry.entity_ref}:${entry.action.id}`} className="rounded-lg border border-slate-800 bg-slate-900/40 p-3">
          <div className="flex items-start gap-3">
            <span className="mt-0.5 text-xs tabular-nums text-slate-500">{index + 1}</span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-slate-100">{entry.entity_title || entry.entity_ref}</p>
              <p className="mt-1 text-xs font-medium uppercase tracking-wide text-cyan-300">{entry.action.expanded_label}</p>
              {entry.action.reason && <p className="mt-1 text-sm text-slate-400">{entry.action.reason}</p>}
            </div>
            <button type="button" onClick={() => onOpen(entry)} className="rounded border border-slate-700 px-2 py-1 text-xs text-slate-300 hover:bg-slate-800">Open</button>
          </div>
        </article>
      ))}
    </div>
  );
}
