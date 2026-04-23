/**
 * FeedbackPanel — the initiative's feedback surface.
 *
 * Lists every feedback round in reverse chronological order. Expanding a
 * round reveals its thread, the current proposal (if any), revise input
 * (for awaiting_user rounds), and the decision chip (for terminal rounds).
 *
 * Owns:
 *   - fetching rounds via useQuery
 *   - the expand/collapse state
 *   - the "revise" continuation mutation (re-sends a user message on the
 *     same agent-manager run)
 *   - the decide/reject/dismiss mutations routed through ProposalReview
 *
 * The "Add Feedback" button lives at the top of the panel AND is mirrored
 * from the page header — either entry opens the same dialog.
 */

import { useCallback, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronDown, ChevronRight, Loader2, MessageCirclePlus, RefreshCw, SendHorizontal } from "lucide-react";
import { Button } from "../ui/button";
import { StatusChip } from "../ui/status-chip";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { FeedbackDialog } from "./feedback-dialog";
import { FeedbackThread } from "./feedback-thread";
import { ProposalReview } from "./proposal-review";
import { feedbackService } from "../../services/feedback-service";
import {
  currentProposal,
  type ApplyResult,
  type FeedbackRound,
  type FeedbackRoundStatus,
  isFeedbackRoundTerminal,
} from "../../types";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { defaultQueryOptions, formatRelativeTime, formatDisplayText } from "../../lib";

export interface FeedbackPanelProps {
  initiativeName: string;
  /**
   * Current member items of the initiative. When provided, expanded rounds
   * render an overlay graph preview of the accepted mutations so the user
   * sees the topological diff before applying.
   */
  previewItems?: Array<{
    kind: string;
    name: string;
    title: string;
    status: import("../../types").BacklogStatus;
    dependsOn: string[];
    priority?: number;
    archivedAt?: string;
    missing?: boolean;
  }>;
}

export function FeedbackPanel({ initiativeName, previewItems }: FeedbackPanelProps) {
  const qc = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["initiative-feedback", initiativeName],
    queryFn: () => feedbackService.list(initiativeName),
    enabled: !!initiativeName,
    refetchInterval: (query) => {
      const rounds = query.state.data;
      if (!rounds) return false;
      // Poll while any round is in an "agent thinking" state — the agent
      // turn hook lands asynchronously via the agent-manager callback, so
      // we catch the state flip on our next tick.
      return rounds.some((r) => r.status === "agent_thinking") ? 3000 : false;
    },
    ...defaultQueryOptions,
  });

  const rounds = useMemo(() => {
    const list = [...(data ?? [])];
    list.sort((a, b) => b.number - a.number);
    return list;
  }, [data]);

  const toggle = useCallback((n: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(n)) next.delete(n);
      else next.add(n);
      return next;
    });
  }, []);

  const invalidate = useCallback(() => {
    void qc.invalidateQueries({ queryKey: ["initiative-feedback", initiativeName] });
  }, [qc, initiativeName]);

  if (isLoading) {
    return <PageLoadingState label="Loading feedback rounds…" />;
  }
  if (error) {
    return (
      <ErrorState
        error={error}
        title="Failed to load feedback"
        onRetry={() => refetch()}
      />
    );
  }

  return (
    <section className="space-y-3" data-testid={selectors.feedback.panel}>
      <header className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-100">Feedback</h3>
          <p className="text-[11px] text-slate-500">
            Capture observations; agent turns them into reviewable proposals.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => refetch()}
            disabled={isFetching}
            title="Refresh"
          >
            <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", isFetching && "animate-spin")} />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setDialogOpen(true)}>
            <MessageCirclePlus className="mr-1.5 h-3.5 w-3.5" />
            Add Feedback
          </Button>
        </div>
      </header>

      {rounds.length === 0 ? (
        <p
          className="rounded-lg border border-slate-800/80 bg-slate-900/40 p-4 text-xs text-slate-500"
          data-testid={selectors.feedback.panelEmpty}
        >
          No feedback yet. Use <strong>Add Feedback</strong> to log a note, share
          an observation, or ask the agent to propose changes.
        </p>
      ) : (
        <ul className="space-y-2">
          {rounds.map((round) => (
            <li key={round.number}>
              <FeedbackRoundCard
                round={round}
                expanded={expanded.has(round.number)}
                onToggle={() => toggle(round.number)}
                onChanged={invalidate}
                previewItems={previewItems}
              />
            </li>
          ))}
        </ul>
      )}

      <FeedbackDialog
        initiativeName={initiativeName}
        isOpen={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onSubmitted={(r) => {
          invalidate();
          setExpanded((prev) => new Set(prev).add(r.number));
        }}
      />
    </section>
  );
}

// ---------------------------------------------------------------------------
// Round card
// ---------------------------------------------------------------------------

interface FeedbackRoundCardProps {
  round: FeedbackRound;
  expanded: boolean;
  onToggle: () => void;
  onChanged: () => void;
  previewItems?: FeedbackPanelProps["previewItems"];
}

function FeedbackRoundCard({ round, expanded, onToggle, onChanged, previewItems }: FeedbackRoundCardProps) {
  const qc = useQueryClient();
  const [reviseText, setReviseText] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [applyResult, setApplyResult] = useState<ApplyResult | null>(null);
  const chip = statusChip(round.status);
  const proposal = currentProposal(round);

  const decideMutation = useMutation({
    mutationFn: async (input: {
      kind: "accept" | "partial_accept" | "reject" | "dismiss";
      rationale?: string;
      acceptedMutationIds?: string[];
    }) => {
      return feedbackService.decide(round.initiative_name, round.number, {
        kind: input.kind,
        acceptedMutationIds: input.acceptedMutationIds,
        rationale: input.rationale,
      });
    },
    onSuccess: (resp) => {
      setActionError(null);
      setApplyResult(resp.apply_result ?? null);
      onChanged();
      void qc.invalidateQueries({ queryKey: ["initiative", round.initiative_name] });
    },
    onError: (err) => {
      setActionError(err instanceof Error ? err.message : "Decision failed.");
    },
  });

  const continueMutation = useMutation({
    mutationFn: async () => {
      return feedbackService.continue_(round.initiative_name, round.number, {
        text: reviseText.trim(),
      });
    },
    onSuccess: () => {
      setReviseText("");
      setActionError(null);
      onChanged();
    },
    onError: (err) => {
      setActionError(err instanceof Error ? err.message : "Continue failed.");
    },
  });

  const handleAccept = (ids: string[], rationale: string) => {
    const totalMutations = proposal?.proposal.mutations?.length ?? 0;
    const kind = ids.length === totalMutations ? "accept" : "partial_accept";
    decideMutation.mutate({ kind, acceptedMutationIds: ids, rationale });
  };
  const handleReject = (rationale: string) => {
    decideMutation.mutate({ kind: "reject", rationale });
  };
  const handleDismiss = (rationale: string) => {
    decideMutation.mutate({ kind: "dismiss", rationale });
  };

  const isTerminal = isFeedbackRoundTerminal(round.status);
  const isActive = round.status === "agent_thinking";

  return (
    <article
      className="overflow-hidden rounded-xl border border-slate-800/80 bg-slate-900/50"
      data-testid={selectors.feedback.panelRoundCard}
      data-round-number={round.number}
      data-round-status={round.status}
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left transition-colors hover:bg-slate-800/40"
        data-testid={selectors.feedback.panelRoundExpand}
      >
        <div className="flex min-w-0 items-center gap-2">
          {expanded ? (
            <ChevronDown className="h-4 w-4 shrink-0 text-slate-400" />
          ) : (
            <ChevronRight className="h-4 w-4 shrink-0 text-slate-400" />
          )}
          <span className="text-xs uppercase tracking-wider text-slate-500">
            Round {round.number}
          </span>
          <span className="truncate text-sm font-medium text-slate-100">
            {summarizeSubmission(round)}
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <span className="text-[10px] uppercase tracking-wider text-slate-500">
            {formatDisplayText(round.type)}
          </span>
          <StatusChip
            label={formatDisplayText(round.status)}
            colors={chip.colors}
            pulse={chip.pulse}
            leadingDot
          />
          <span className="text-[10px] text-slate-600">
            {formatRelativeTime(round.updated_at)}
          </span>
        </div>
      </button>

      {expanded && (
        <div className="space-y-3 border-t border-slate-800/60 bg-slate-950/40 p-3">
          <FeedbackThread round={round} />

          {proposal && !isActive && (
            <ProposalReview
              revision={proposal}
              onAccept={handleAccept}
              onReject={handleReject}
              onDismiss={handleDismiss}
              isPending={decideMutation.isPending}
              error={actionError}
              applyResult={applyResult}
              readOnly={isTerminal}
              previewItems={previewItems}
            />
          )}

          {isActive && (
            <div className="flex items-center gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/5 px-3 py-2 text-xs text-cyan-200">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Agent is working on a response…
            </div>
          )}

          {round.status === "awaiting_user" && (
            <form
              className="flex items-end gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                if (!reviseText.trim() || continueMutation.isPending) return;
                continueMutation.mutate();
              }}
            >
              <textarea
                value={reviseText}
                onChange={(e) => setReviseText(e.target.value)}
                rows={2}
                disabled={continueMutation.isPending}
                placeholder="Ask the agent to revise the proposal… (Ctrl+Enter to send)"
                className="flex-1 resize-none rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 outline-none focus:border-cyan-500/50"
                data-testid={selectors.feedback.threadReviseInput}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                    e.preventDefault();
                    if (reviseText.trim()) continueMutation.mutate();
                  }
                }}
              />
              <Button
                type="submit"
                size="sm"
                disabled={!reviseText.trim() || continueMutation.isPending}
                data-testid={selectors.feedback.threadReviseSubmit}
              >
                {continueMutation.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <SendHorizontal className="h-3.5 w-3.5" />
                )}
              </Button>
            </form>
          )}

          {round.decision && (
            <div className="rounded-md border border-slate-700 bg-slate-900/60 px-3 py-2 text-xs text-slate-300">
              <div className="font-medium text-slate-200">
                Decision · {formatDisplayText(round.decision.kind)}
              </div>
              {round.decision.rationale && (
                <p className="mt-0.5 text-slate-400">{round.decision.rationale}</p>
              )}
              <p className="mt-0.5 text-[11px] text-slate-500">
                {formatRelativeTime(round.decision.decided_at)}
                {round.decision.decided_by ? ` · ${round.decision.decided_by}` : ""}
              </p>
            </div>
          )}
        </div>
      )}
    </article>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function summarizeSubmission(round: FeedbackRound): string {
  const text = round.submission?.text ?? "";
  if (!text) return `(round ${round.number})`;
  const firstLine = text.split(/\r?\n/)[0] ?? "";
  return firstLine.length > 80 ? firstLine.slice(0, 77) + "…" : firstLine;
}

interface StatusChipSpec {
  colors: {
    background: string;
    text: string;
    border?: string;
    dot?: string;
  };
  pulse?: boolean;
}

function statusChip(status: FeedbackRoundStatus): StatusChipSpec {
  switch (status) {
    case "submitting":
      return { colors: { background: "bg-slate-500/20", text: "text-slate-300", dot: "bg-slate-400" } };
    case "agent_thinking":
      return {
        colors: { background: "bg-cyan-500/20", text: "text-cyan-200", dot: "bg-cyan-400" },
        pulse: true,
      };
    case "awaiting_user":
      return { colors: { background: "bg-amber-500/20", text: "text-amber-200", dot: "bg-amber-400" } };
    case "applied":
      return { colors: { background: "bg-emerald-500/20", text: "text-emerald-200", dot: "bg-emerald-400" } };
    case "rejected":
      return { colors: { background: "bg-red-500/20", text: "text-red-200", dot: "bg-red-400" } };
    case "dismissed":
      return { colors: { background: "bg-slate-500/15", text: "text-slate-400", dot: "bg-slate-500" } };
  }
}
