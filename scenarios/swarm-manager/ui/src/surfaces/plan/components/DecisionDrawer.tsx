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
import { ConfirmDialog } from "../../../components/ui/confirm-dialog";
import { Drawer } from "../../../components/ui/drawer";
import { Input } from "../../../components/ui/input";
import { Popover } from "../../../components/ui/popover";
import { aggregateCrossItemQuestions } from "../../../lib/command-post-utils";
import { backlogService, goalsService, integrationStatusService, transitionService } from "../../../services";
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
import type { GoalMilestone } from "../../../types/goal";
import { actionTargetSuffix, milestoneTargetOf } from "../../../lib/next-action-target";
import { useAsyncAction } from "../../../hooks/useAsyncAction";
import { useTransitionCatalog } from "../../../hooks/useTransitionCatalog";
import { ActionButton, ConsequenceBadge } from "../../../components/ui/action-button";

const SNOOZE_MS = 3_600_000;
/** Rows rendered in the jump list before asking the operator to filter. */
const NAVIGATOR_VISIBLE_LIMIT = 100;

/**
 * Detects the auto-generated milestone criterion that names nothing checkable
 * ("…when the milestone is independently reviewed, then its described outcome
 * is delivered with supporting evidence"). The server's `define_criteria`
 * action only fires on *zero* criteria, so a milestone carrying one of these
 * looks covered while being unreviewable in practice.
 */
function isBoilerplateCriterion(criterion: string): boolean {
  const text = criterion.toLowerCase();
  return text.includes("is independently reviewed")
    && text.includes("described outcome is delivered");
}

/** Direct actions that remove or interrupt state, so they confirm first. */
const DESTRUCTIVE_ACTION_IDS = new Set(["archive"]);

const DESTRUCTIVE_ACTION_CONSEQUENCE: Record<string, string> = {
  archive: "It leaves the backlog and the decision queue. Items depending on it keep their edges until retargeted. Reversible with the unarchive action.",
};

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

  // Keep recommendations need an explicit accept just as mutation lists do, so
  // they belong in the queue. Filtering on `kind === "mutation_list"` alone
  // made them undecidable from here — the only surface that could accept one
  // was the item's own proposals panel.
  const proposals = useMemo<ProposalDecisionStreamItem[]>(
    () => (proposalQuery.data ?? []).flatMap((session) => (session.proposals ?? [])
      .filter((proposal) => (proposal.kind === "mutation_list" || proposal.kind === "no_change_recommendation") && proposal.status === "ready")
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
    if (target.entity_kind === "capture") return;
    if (target.entity_kind === "backlog_item") {
      const [kind, name] = target.entity_ref.split("/");
      if (kind && name) navigate(`${backlogDetailPath(kind as BacklogKind, name)}${suffix}`);
      return;
    }
    // A milestone-scoped action opened the goal's overview and left the
    // operator to hunt for the milestone the card was actually about. Carry
    // the milestone through so the page can land on it.
    const milestone = milestoneTargetOf(target.action.target);
    const milestoneSuffix = milestone ? `&tab=milestones&milestone=${encodeURIComponent(milestone)}` : "";
    navigate(`/goals/${target.entity_ref}${suffix}${milestoneSuffix}`);
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
  // A milestone-review card used to name neither the milestone nor anything
  // the review would look at, even though the milestone name is already
  // encoded in action.target. The operator could not tell what was being
  // reviewed without leaving the queue.
  const milestoneName = milestoneTargetOf(entry.action.target);

  const milestoneQuery = useQuery({
    queryKey: ["goal", entry.entity_ref, "milestone-review"],
    queryFn: () => goalsService.get(entry.entity_ref),
    enabled: entry.entity_kind === "goal" && Boolean(milestoneName),
  });
  // The review card branches on `criteria`, and an empty list makes it warn
  // that the item has none. Without this query the stream passed nothing, so
  // every review decision claimed the item had no typed criteria and hid the
  // criterion/settlement/evidence table that the item detail page shows.
  const reviewItemQuery = useQuery({
    queryKey: ["backlog-item", backlogKind, backlogName],
    queryFn: () => backlogService.get(backlogKind as BacklogKind, backlogName),
    enabled: variant === "review" && backlogRef.length === 2,
  });

  if (variant === "proposal") {
    return (
      <div className="h-full" data-testid="next-action-stream-proposal">
        <ProposalDecisionStreamView
          embedded
          proposals={proposals}
          onBack={() => undefined}
          onComplete={onChanged}
          onItemChanged={onChanged}
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
        <ReviewDecisionCard
          kind={backlogKind}
          name={backlogName}
          round={reviewQuery.data?.at(-1)}
          criteria={reviewItemQuery.data?.acceptanceCriteria}
          onDecided={onChanged}
          // Recovery is authored on the item, so a send-back hands the
          // operator to the place that owns the next step.
          onSendBack={onOpen}
        />
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
      reviewMilestone={milestoneQuery.data?.goal.milestones?.find((milestone) => milestone.name === milestoneName)}
      onChanged={onChanged}
      onOpen={onOpen}
      onFeedback={onFeedback}
    />
  );
}

function ActionCard({ entry, backlogRef, closeOutMilestones, reviewMilestone, onChanged, onOpen, onFeedback }: {
  entry: NextActionFeedEntry;
  backlogRef: string[];
  closeOutMilestones?: Array<{ name: string; title: string; archivedAt?: string | null; verifiedDeliveredAt?: string | null }>;
  /** The milestone a `milestone_review` action targets, once resolved. */
  reviewMilestone?: GoalMilestone;
  onChanged: () => void;
  onOpen: () => void;
  onFeedback: () => Promise<void>;
}) {
  // Inline-only: this card stays on screen after a failure and shows the
  // reason in place, so a toast would report the same event twice.
  const { pending, error, run: runAction } = useAsyncAction({ toastOnError: false, source: "DecisionDrawer.ActionCard" });
  const [runTarget, setRunTarget] = useState<RunSheetTarget | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const isGoalPlan = entry.entity_kind === "goal" && entry.action.id === "plan_goal";
  const goalMilestone = actionTargetSuffix(entry.action.target, "milestone_review:");
  const direct = Boolean(entry.action.transition_key) || ["retry", "archive", "accept_suggestion", "accept_plan"].includes(entry.action.id);
  const transitionKey = entry.action.transition_key ?? (goalMilestone ? "milestone.review" : undefined);
  const transitionQuery = useTransitionCatalog(Boolean(transitionKey));
  const integrationQuery = useQuery({
    queryKey: ["integration-status"],
    queryFn: () => integrationStatusService.get(),
    staleTime: 15_000,
    enabled: Boolean(transitionKey),
  });
  const transition = transitionQuery.data?.find((candidate) => candidate.key === transitionKey);
  const unavailableRequirements = (transition?.requires ?? []).flatMap((required) => {
    const integration = integrationQuery.data?.integrations.find((candidate) => candidate.id === required);
    return integration && integration.availability !== "available" ? [integration] : [];
  });
  const transitionUnavailableReason = unavailableRequirements.length > 0
    ? unavailableRequirements.map((integration) => integration.diagnostic || `${integration.id} is ${integration.availability}`).join("; ")
    : undefined;

  const run = async () => {
    // Two branches below deliberately return *before* doing async work: "run"
    // hands off to the run sheet and the default falls through to Open. They
    // must not be reported as completed actions, so the flag distinguishes
    // "handed off" from "finished".
    let handedOff = false;
    const completed = await runAction(async () => {
      const transitionKey = entry.action.transition_key;
      if (transitionKey && entry.entity_kind === "goal") {
        await transitionService.start(transitionKey, entry.entity_ref);
      } else if (backlogRef.length === 2) {
        const [kind, name] = backlogRef as [BacklogKind, string];
        if (transitionKey) {
          await transitionService.start(transitionKey, `${kind}/${name}`);
        } else switch (entry.action.id) {
          case "run": setRunTarget({ kind, name, title: entry.entity_title }); handedOff = true; return;
          case "retry": await backlogService.retry(kind, name, "Retried from decision stream"); break;
          case "archive": await backlogService.archiveItem(kind, name); break;
          case "accept_suggestion": await backlogService.update(kind, name, { status: "backlog" }); break;
          case "accept_plan": await defaultApiClient.post(API_ENDPOINTS.backlogPlanAccept(kind, name), {}); break;
          default: onOpen(); handedOff = true; return;
        }
      } else {
        onOpen();
        handedOff = true;
        return;
      }
    }, { errorMessage: "Unable to complete this action." });
    if (completed && !handedOff) onChanged();
  };

  const completeGoalPlan = async () => {
    const ok = await runAction(
      () => transitionService.start(entry.action.transition_key ?? "", entry.entity_ref),
      { errorMessage: "Unable to start goal planning." },
    );
    if (ok) onChanged();
  };

  const startMilestoneReview = async () => {
    const ok = await runAction(
      () => transitionService.start("milestone.review", `${entry.entity_ref}/${goalMilestone}`),
      { errorMessage: "Unable to start milestone review." },
    );
    if (ok) onChanged();
  };

  // The queue is built for rapid sequential taps on a phone, so an action that
  // removes state must not be one tap away from the same spot as "Open".
  // The server declares this now; the local set remains only as the fallback
  // for a feed older than the field.
  const destructive = entry.action.destructive ?? DESTRUCTIVE_ACTION_IDS.has(entry.action.id);
  // ActionButton appends the ellipsis for destructive actions itself.
  const primaryLabel = entry.action.id === "run"
    ? "Choose run"
    : direct || isGoalPlan || goalMilestone
      ? entry.action.compact_label
      : "Open";
  // "Open" only navigates; anything else acts, and the consequence class tells
  // the operator which kind of acting it is before they tap.
  const primaryConsequence = primaryLabel === "Open"
    ? { effect: "none" as const }
    : { actionId: entry.action.id, effect: entry.action.effect, transitionKind: transition?.kind, destructive };

  const startPrimary = () => {
    if (destructive) { setConfirmOpen(true); return; }
    if (isGoalPlan) void completeGoalPlan();
    else if (goalMilestone) void startMilestoneReview();
    else void run();
  };

  return (
    <section className="space-y-4 p-4" data-testid="next-action-stream-card">
      {runTarget ? <RunSheet isOpen onClose={() => setRunTarget(null)} target={runTarget} onSuccess={onChanged} /> : null}
      <ConfirmDialog
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => { setConfirmOpen(false); void run(); }}
        title={`${entry.action.compact_label}?`}
        description={`${entry.entity_title || entry.entity_ref} — ${DESTRUCTIVE_ACTION_CONSEQUENCE[entry.action.id] ?? "This removes state."}`}
        confirmLabel={entry.action.compact_label}
        isLoading={pending}
        errorMessage={error ?? undefined}
        testIds={{ dialog: "next-action-destructive-confirm" }}
      />
      <p className="text-xs text-slate-500">{entry.entity_ref} · Tier {entry.tier}</p>

      <div className="rounded-xl border border-slate-800 bg-slate-900/60 p-4">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-xs font-semibold uppercase tracking-wide text-cyan-300">{entry.action.expanded_label}</p>
          {/* Names the cost in the card body, not only on the button, so the
              operator reading the reason already knows an agent is involved. */}
          <ConsequenceBadge {...primaryConsequence} />
        </div>
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

        {/* Nothing has been gathered yet for a milestone review — the action is
            derived from state. Showing the criteria the agent will grade against
            and the items it will grade is the only substance that exists, and it
            is what tells the operator whether starting a review is worthwhile. */}
        {goalMilestone ? (
          <div className="mt-3 border-t border-slate-800 pt-3">
            <p className="text-sm font-medium text-slate-100">{reviewMilestone?.title || goalMilestone}</p>
            <code className="text-xs text-cyan-200">{goalMilestone}</code>
            {reviewMilestone ? (
              <>
                <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-400">
                  Will be graded against {reviewMilestone.acceptanceCriteria.length} criteri{reviewMilestone.acceptanceCriteria.length === 1 ? "on" : "a"}
                </p>
                <ol className="mt-1 flex flex-col gap-1">
                  {reviewMilestone.acceptanceCriteria.map((criterion, index) => (
                    <li key={criterion} className="flex gap-1.5 text-xs leading-5 text-slate-300">
                      <span className="shrink-0 tabular-nums text-slate-600">{index + 1}</span>
                      <span className="min-w-0">{criterion}</span>
                    </li>
                  ))}
                </ol>
                {reviewMilestone.acceptanceCriteria.every(isBoilerplateCriterion) && reviewMilestone.acceptanceCriteria.length > 0 ? (
                  <p className="mt-2 rounded border border-amber-300/25 bg-amber-300/[0.08] p-2 text-xs text-amber-100">
                    These criteria restate the milestone rather than naming anything checkable, so a review can only
                    confirm the work is finished — not that it is correct. Sharpen them first for a review worth having.
                  </p>
                ) : null}
                <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-400">
                  Covering {reviewMilestone.items.length} completed item{reviewMilestone.items.length === 1 ? "" : "s"}
                </p>
                <ul className="mt-1 flex flex-col gap-0.5">
                  {reviewMilestone.items.map((ref) => (
                    <li key={ref}><code className="break-all text-xs text-slate-400">{ref}</code></li>
                  ))}
                </ul>
              </>
            ) : (
              <p className="mt-2 text-xs text-slate-500">Loading the milestone's criteria and members…</p>
            )}
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

      {error ? <p role="alert" className="text-sm text-rose-300">{error}</p> : null}
      {transitionUnavailableReason ? <p className="text-sm text-amber-300">Transition unavailable: {transitionUnavailableReason}</p> : null}

      <div className="flex flex-wrap justify-end gap-2">
        <button
          type="button"
          onClick={() => void runAction(onFeedback, { errorMessage: "Unable to start feedback." })}
          className="flex items-center gap-1.5 rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800"
        >
          <MessageSquarePlus className="h-4 w-4" aria-hidden />
          Feedback
        </button>
        <button type="button" onClick={onOpen} className="flex items-center gap-1.5 rounded border border-slate-700 px-3 py-1.5 text-sm text-slate-300 hover:bg-slate-800">
          <ExternalLink className="h-4 w-4" aria-hidden />
          Open
        </button>
        <ActionButton
          {...primaryConsequence}
          icon={nextActionIcon(entry.action.id)}
          label={primaryLabel}
          pending={pending}
          disabled={Boolean(transitionUnavailableReason)}
          onClick={startPrimary}
          size="sm"
          className="h-auto rounded px-3 py-1.5"
          data-testid="next-action-primary"
        />
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
  // Inline-only: this is a form, and the operator must be able to read the
  // reason without losing the steering text they just typed.
  const { pending, error, run: runAction } = useAsyncAction({ toastOnError: false, source: "DecisionDrawer.FollowUpCard" });
  const needsChild = disposition === "new_items";

  const submit = async () => {
    const followUp = { steering, disposition, ...(needsChild ? { items: [{ kind: childKind, name: childName.trim(), title: childTitle.trim() }] } : {}) };
    const ok = await runAction(
      () => defaultApiClient.post(API_ENDPOINTS.backlogAuthorFollowUp(kind, name), { follow_up: followUp }),
      { errorMessage: "Unable to save follow-up." },
    );
    if (ok) onChanged();
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
      {error ? <p role="alert" className="text-sm text-rose-300">{error}</p> : null}
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
