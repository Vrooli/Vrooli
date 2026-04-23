/**
 * InitiativeReviewPanel — gates the terminal transition of an initiative.
 *
 * When an initiative reaches `in_review` (agent is gathering evidence) or
 * `review_pending` (ready for the user's verdict), this panel lets the user
 * inspect the review rounds and record a terminal verdict (accept / fail /
 * followup).
 *
 * Initiatives in other states (active / completed / failed / needs_followup)
 * still render the panel so the user can browse the decision audit log and
 * optionally trigger a manual review when the initiative is dormant.
 *
 * Wiring:
 *   - `useQuery` fetches rounds + decisions in parallel
 *   - A mutation calls `/review/trigger` for a manual kick
 *   - A second mutation calls `/review/decide` with the chosen verdict +
 *     rationale; parent refreshes the initiative on success
 */

import { useCallback, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  CheckCircle2,
  ClipboardCheck,
  Loader2,
  PlayCircle,
  XCircle,
  ArrowUpRight,
  RefreshCw,
} from "lucide-react";
import { Button } from "../ui/button";
import { StatusChip } from "../ui/status-chip";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { initiativeReviewService } from "../../services/initiative-review-service";
import { selectors } from "../../consts/selectors";
import { defaultQueryOptions, formatRelativeTime, formatDisplayText } from "../../lib";
import { cn } from "../../lib/utils";
import {
  REVIEW_CLASSIFICATION_COLORS,
  REVIEW_CLASSIFICATION_LABELS,
} from "../../types";
import type {
  InitiativeReviewDecision,
  InitiativeReviewRound,
  InitiativeReviewVerdict,
  InitiativeStatus,
} from "../../types";

export interface InitiativeReviewPanelProps {
  initiativeName: string;
  initiativeStatus: InitiativeStatus;
  /** Parent invalidator — called after a successful decision so the
   *  initiative header + status chip refresh. */
  onDecided?: () => void;
}

export function InitiativeReviewPanel({
  initiativeName,
  initiativeStatus,
  onDecided,
}: InitiativeReviewPanelProps) {
  const qc = useQueryClient();
  const [rationale, setRationale] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [pickedVerdict, setPickedVerdict] = useState<InitiativeReviewVerdict | null>(null);

  const roundsQuery = useQuery({
    queryKey: ["initiative-review-rounds", initiativeName],
    queryFn: () => initiativeReviewService.listRounds(initiativeName),
    enabled: !!initiativeName,
    refetchInterval: (query) => {
      const rounds = query.state.data;
      if (!rounds) return false;
      // Poll while a round is still gathering so the user sees completion
      // without manual refresh.
      return rounds.some((r) => r.status === "pending" || r.status === "gathering") ? 3000 : false;
    },
    ...defaultQueryOptions,
  });

  const decisionsQuery = useQuery({
    queryKey: ["initiative-review-decisions", initiativeName],
    queryFn: () => initiativeReviewService.listDecisions(initiativeName),
    enabled: !!initiativeName,
    ...defaultQueryOptions,
  });

  const triggerMutation = useMutation({
    mutationFn: () => initiativeReviewService.trigger(initiativeName),
    onSuccess: (result) => {
      setActionError(result.started ? null : result.reason ?? "Review did not start.");
      void qc.invalidateQueries({ queryKey: ["initiative-review-rounds", initiativeName] });
      void qc.invalidateQueries({ queryKey: ["initiative", initiativeName] });
    },
    onError: (err) => {
      setActionError(err instanceof Error ? err.message : "Trigger failed.");
    },
  });

  const decideMutation = useMutation({
    mutationFn: (verdict: InitiativeReviewVerdict) =>
      initiativeReviewService.decide(initiativeName, {
        verdict,
        rationale: rationale.trim() || undefined,
      }),
    onSuccess: () => {
      setRationale("");
      setPickedVerdict(null);
      setActionError(null);
      void qc.invalidateQueries({ queryKey: ["initiative-review-rounds", initiativeName] });
      void qc.invalidateQueries({ queryKey: ["initiative-review-decisions", initiativeName] });
      void qc.invalidateQueries({ queryKey: ["initiative", initiativeName] });
      onDecided?.();
    },
    onError: (err) => {
      setActionError(err instanceof Error ? err.message : "Decision failed.");
    },
  });

  const rounds = useMemo(
    () => [...(roundsQuery.data ?? [])].sort((a, b) => b.round - a.round),
    [roundsQuery.data],
  );
  const decisions = useMemo(
    () =>
      [...(decisionsQuery.data ?? [])].sort(
        (a, b) => Date.parse(b.decided_at) - Date.parse(a.decided_at),
      ),
    [decisionsQuery.data],
  );

  const canTriggerManually = initiativeStatus === "active";
  const canDecide = initiativeStatus === "review_pending";

  const handleDecide = useCallback(
    (verdict: InitiativeReviewVerdict) => {
      setPickedVerdict(verdict);
      decideMutation.mutate(verdict);
    },
    [decideMutation],
  );

  if (roundsQuery.isLoading) {
    return <PageLoadingState label="Loading review rounds…" />;
  }
  if (roundsQuery.error) {
    return (
      <ErrorState
        error={roundsQuery.error}
        title="Failed to load initiative review"
        onRetry={() => roundsQuery.refetch()}
      />
    );
  }

  return (
    <section className="space-y-4" data-testid={selectors.initiativeReview.panel}>
      <header className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-100">Initiative Review</h3>
          <p className="text-[11px] text-slate-500">
            Required before an initiative can move to a terminal state.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              void roundsQuery.refetch();
              void decisionsQuery.refetch();
            }}
            disabled={roundsQuery.isFetching || decisionsQuery.isFetching}
          >
            <RefreshCw
              className={cn(
                "mr-1.5 h-3.5 w-3.5",
                (roundsQuery.isFetching || decisionsQuery.isFetching) && "animate-spin",
              )}
            />
            Refresh
          </Button>
          <Button
            size="sm"
            onClick={() => triggerMutation.mutate()}
            disabled={triggerMutation.isPending || !canTriggerManually}
            title={
              canTriggerManually
                ? "Run the review agent now"
                : "Manual trigger only available from 'active' status"
            }
            data-testid={selectors.initiativeReview.triggerButton}
          >
            {triggerMutation.isPending ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <PlayCircle className="mr-1.5 h-3.5 w-3.5" />
            )}
            Trigger review
          </Button>
        </div>
      </header>

      {actionError && (
        <p className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {actionError}
        </p>
      )}

      {canDecide && (
        <div className="rounded-xl border border-cyan-500/40 bg-cyan-500/5 p-3">
          <div className="mb-2 flex items-center gap-2 text-sm font-medium text-cyan-200">
            <ArrowUpRight className="h-4 w-4" />
            Decision required
          </div>
          <p className="mb-2 text-xs text-cyan-200/80">
            The review agent has finished gathering evidence. Record the terminal
            verdict for this initiative.
          </p>
          <textarea
            value={rationale}
            onChange={(e) => setRationale(e.target.value)}
            rows={2}
            placeholder="Rationale (optional)…"
            disabled={decideMutation.isPending}
            className="mb-2 w-full resize-none rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 outline-none focus:border-cyan-500/50"
            data-testid={selectors.initiativeReview.rationaleInput}
          />
          <div className="flex flex-wrap justify-end gap-2">
            <VerdictButton
              verdict="followup"
              label="Needs follow-up"
              icon={AlertTriangle}
              pending={decideMutation.isPending && pickedVerdict === "followup"}
              disabled={decideMutation.isPending}
              testId={selectors.initiativeReview.verdictFollowup}
              tone="amber"
              onClick={handleDecide}
            />
            <VerdictButton
              verdict="fail"
              label="Mark as failed"
              icon={XCircle}
              pending={decideMutation.isPending && pickedVerdict === "fail"}
              disabled={decideMutation.isPending}
              testId={selectors.initiativeReview.verdictFail}
              tone="red"
              onClick={handleDecide}
            />
            <VerdictButton
              verdict="accept"
              label="Accept & complete"
              icon={CheckCircle2}
              pending={decideMutation.isPending && pickedVerdict === "accept"}
              disabled={decideMutation.isPending}
              testId={selectors.initiativeReview.verdictAccept}
              tone="emerald"
              onClick={handleDecide}
            />
          </div>
        </div>
      )}

      <section>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
          Rounds
        </h4>
        {rounds.length === 0 ? (
          <p className="rounded-lg border border-slate-800/80 bg-slate-900/40 p-3 text-xs text-slate-500">
            No review rounds yet. {canTriggerManually ? "Trigger one manually, or complete all member items to auto-trigger." : "Awaiting agent completion."}
          </p>
        ) : (
          <ul className="space-y-2">
            {rounds.map((round) => (
              <li key={round.round}>
                <ReviewRoundCard round={round} />
              </li>
            ))}
          </ul>
        )}
      </section>

      {decisions.length > 0 && (
        <section>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-500">
            Decision history
          </h4>
          <ul className="space-y-1.5">
            {decisions.map((d, i) => (
              <li
                key={i}
                data-testid={selectors.initiativeReview.decisionRecord}
                data-verdict={d.verdict}
              >
                <DecisionRow decision={d} />
              </li>
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Round card
// ---------------------------------------------------------------------------

function ReviewRoundCard({ round }: { round: InitiativeReviewRound }) {
  const chip = roundStatusChip(round.status);
  return (
    <article
      className="rounded-xl border border-slate-800/80 bg-slate-900/50 p-3"
      data-testid={selectors.initiativeReview.roundCard}
      data-round={round.round}
      data-status={round.status}
    >
      <header className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="text-xs uppercase tracking-wider text-slate-500">
            Round {round.round}
          </span>
          <StatusChip
            label={formatDisplayText(round.status)}
            colors={chip.colors}
            pulse={chip.pulse}
            leadingDot
          />
          {round.classification && (
            <span
              className={cn(
                "rounded-full px-2 py-0.5 text-[10px] font-medium",
                REVIEW_CLASSIFICATION_COLORS[round.classification] ?? "bg-slate-500/20 text-slate-200",
              )}
            >
              {REVIEW_CLASSIFICATION_LABELS[round.classification] ?? formatDisplayText(round.classification)}
            </span>
          )}
        </div>
        <span className="text-[11px] text-slate-500">{formatRelativeTime(round.generated_at)}</span>
      </header>

      {round.agent_assessment && (
        <p className="whitespace-pre-wrap break-words text-xs leading-relaxed text-slate-300">
          {round.agent_assessment}
        </p>
      )}

      {round.failure_reason && (
        <p className="mt-2 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1 text-[11px] text-red-200">
          {round.failure_reason}
        </p>
      )}

      {round.notes && round.notes.length > 0 && (
        <ul className="mt-2 list-disc space-y-0.5 pl-5 text-[11px] text-slate-400">
          {round.notes.map((n, i) => (
            <li key={i}>{n}</li>
          ))}
        </ul>
      )}

      <p className="mt-2 text-[11px] text-slate-500">
        {round.evidence.length} evidence item{round.evidence.length === 1 ? "" : "s"}
      </p>
    </article>
  );
}

// ---------------------------------------------------------------------------
// Verdict button & decision row
// ---------------------------------------------------------------------------

interface VerdictButtonProps {
  verdict: InitiativeReviewVerdict;
  label: string;
  icon: typeof CheckCircle2;
  pending: boolean;
  disabled: boolean;
  testId: string;
  tone: "emerald" | "red" | "amber";
  onClick: (v: InitiativeReviewVerdict) => void;
}

function VerdictButton({ verdict, label, icon: Icon, pending, disabled, testId, tone, onClick }: VerdictButtonProps) {
  const toneClasses = {
    emerald: "border-emerald-500/40 bg-emerald-500/15 text-emerald-200 hover:bg-emerald-500/25",
    red: "border-red-500/40 bg-red-500/10 text-red-200 hover:bg-red-500/20",
    amber: "border-amber-500/40 bg-amber-500/10 text-amber-200 hover:bg-amber-500/20",
  }[tone];
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onClick(verdict)}
      data-testid={testId}
      className={cn(
        "rounded-md border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50",
        toneClasses,
      )}
    >
      {pending ? (
        <Loader2 className="mr-1.5 inline h-3.5 w-3.5 animate-spin" />
      ) : (
        <Icon className="mr-1.5 inline h-3.5 w-3.5" />
      )}
      {label}
    </button>
  );
}

function DecisionRow({ decision }: { decision: InitiativeReviewDecision }) {
  const verdictChip = verdictChipColors(decision.verdict);
  return (
    <div className="flex items-start gap-3 rounded-md border border-slate-800/60 bg-slate-900/40 px-3 py-2 text-xs">
      <ClipboardCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <StatusChip
            label={formatDisplayText(decision.verdict)}
            colors={verdictChip}
            leadingDot
          />
          <span className="text-[11px] text-slate-400">
            {decision.prior_status} → <strong>{decision.status}</strong>
          </span>
          {decision.round ? (
            <span className="text-[11px] text-slate-500">Round {decision.round}</span>
          ) : null}
        </div>
        {decision.rationale && (
          <p className="mt-1 text-[11px] text-slate-400">{decision.rationale}</p>
        )}
        <p className="mt-0.5 text-[10px] text-slate-600">
          {formatRelativeTime(decision.decided_at)}
          {decision.decided_by ? ` · ${decision.decided_by}` : ""}
        </p>
      </div>
    </div>
  );
}

function roundStatusChip(status: InitiativeReviewRound["status"]) {
  switch (status) {
    case "pending":
      return { colors: { background: "bg-slate-500/20", text: "text-slate-300", dot: "bg-slate-400" } };
    case "gathering":
      return {
        colors: { background: "bg-cyan-500/20", text: "text-cyan-200", dot: "bg-cyan-400" },
        pulse: true,
      };
    case "complete":
      return { colors: { background: "bg-emerald-500/20", text: "text-emerald-200", dot: "bg-emerald-400" } };
    case "failed":
      return { colors: { background: "bg-red-500/20", text: "text-red-200", dot: "bg-red-400" } };
  }
}

function verdictChipColors(v: InitiativeReviewVerdict) {
  switch (v) {
    case "accept":
      return { background: "bg-emerald-500/20", text: "text-emerald-200", dot: "bg-emerald-400" };
    case "fail":
      return { background: "bg-red-500/20", text: "text-red-200", dot: "bg-red-400" };
    case "followup":
      return { background: "bg-amber-500/20", text: "text-amber-200", dot: "bg-amber-400" };
  }
}
