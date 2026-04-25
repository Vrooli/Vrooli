/**
 * FeedbackRoundCard — one feedback round's collapsible detail view.
 *
 * Split out of feedback-panel.tsx so the panel stays a thin list/fetch
 * shell. This card owns the per-round mutation wiring:
 *   - decide (accept / partial_accept / reject / dismiss) via ProposalReview
 *   - continue (the "revise" follow-up that re-sends a message on the same
 *     agent-manager run)
 *
 * When the round enters awaiting_user while expanded, the revise textarea
 * auto-focuses so the user can keep typing without re-grabbing the mouse.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Loader2,
  SendHorizontal,
  Trash2,
  XCircle,
} from "lucide-react";
import { useEmbeddedServiceUrl } from "../../hooks/useEmbeddedServiceUrl";
import { Button } from "../ui/button";
import { StatusChip } from "../ui/status-chip";
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
import { formatDisplayText, formatRelativeTime } from "../../lib";
import type { FeedbackPanelProps } from "./feedback-panel";
import {
  REVISE_PLACEHOLDER,
  AGENT_WORKING_LABEL,
  PARSE_ERROR_TITLE,
  PARSE_ERROR_BODY,
} from "./feedback-strings";

export interface FeedbackRoundCardProps {
  round: FeedbackRound;
  expanded: boolean;
  onToggle: () => void;
  onChanged: () => void;
  previewItems?: FeedbackPanelProps["previewItems"];
}

export function FeedbackRoundCard({
  round,
  expanded,
  onToggle,
  onChanged,
  previewItems,
}: FeedbackRoundCardProps) {
  const qc = useQueryClient();
  const [reviseText, setReviseText] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [applyResult, setApplyResult] = useState<ApplyResult | null>(null);
  const reviseRef = useRef<HTMLTextAreaElement | null>(null);
  const chip = useMemo(() => statusChip(round.status), [round.status]);
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

  const { url: agentManagerUiUrl } = useEmbeddedServiceUrl("agent-manager");
  const runUrl = round.run_id && agentManagerUiUrl ? `${agentManagerUiUrl}/runs/${round.run_id}` : null;
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      return feedbackService.delete(round.initiative_name, round.number);
    },
    onSuccess: () => {
      setDeleteError(null);
      setConfirmingDelete(false);
      onChanged();
      void qc.invalidateQueries({ queryKey: ["initiative", round.initiative_name] });
    },
    onError: (err) => {
      setDeleteError(err instanceof Error ? err.message : "Delete failed.");
      setConfirmingDelete(false);
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async () => {
      return feedbackService.cancel(round.initiative_name, round.number, {
        rationale: "cancelled by user",
      });
    },
    onSuccess: () => {
      setActionError(null);
      onChanged();
      void qc.invalidateQueries({ queryKey: ["initiative", round.initiative_name] });
    },
    onError: (err) => {
      setActionError(err instanceof Error ? err.message : "Cancel failed.");
    },
  });

  // When the round becomes user-actionable while expanded, move focus into
  // the revise textarea so the user can keep typing. Guarded on `expanded`
  // so collapsing-then-reopening doesn't steal focus from whatever the user
  // is doing on the rest of the page.
  const needsReviseFocus = expanded && round.status === "awaiting_user";
  useEffect(() => {
    if (!needsReviseFocus) return;
    reviseRef.current?.focus({ preventScroll: true });
  }, [needsReviseFocus]);

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
          {(runUrl || isTerminal) && (
            <div className="space-y-2">
              <div className="flex flex-wrap items-center justify-end gap-2">
                {runUrl && (
                  <a
                    href={runUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 py-1 text-xs text-cyan-200 hover:bg-cyan-500/20"
                    data-testid={selectors.feedback.openRunButton}
                    title="Open run in Agent Manager"
                  >
                    <ExternalLink className="h-3 w-3" />
                    Open Run
                  </a>
                )}
                {isTerminal && !confirmingDelete && (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setDeleteError(null);
                      setConfirmingDelete(true);
                    }}
                    disabled={deleteMutation.isPending}
                    className="border-rose-500/40 bg-transparent text-rose-200 hover:bg-rose-500/10 hover:text-rose-100"
                    data-testid={selectors.feedback.deleteButton}
                  >
                    {deleteMutation.isPending ? (
                      <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Trash2 className="mr-1 h-3.5 w-3.5" />
                    )}
                    Delete
                  </Button>
                )}
                {isTerminal && confirmingDelete && (
                  <>
                    <span className="text-xs text-rose-200">
                      Delete round {round.number}? This cannot be undone.
                    </span>
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => setConfirmingDelete(false)}
                      disabled={deleteMutation.isPending}
                      className="border-slate-600 bg-transparent text-slate-300 hover:bg-slate-800"
                    >
                      Cancel
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      onClick={() => deleteMutation.mutate()}
                      disabled={deleteMutation.isPending}
                      className="bg-rose-600 text-white hover:bg-rose-500"
                      data-testid={selectors.feedback.deleteButton}
                    >
                      {deleteMutation.isPending ? (
                        <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Trash2 className="mr-1 h-3.5 w-3.5" />
                      )}
                      {deleteMutation.isPending ? "Deleting…" : "Confirm Delete"}
                    </Button>
                  </>
                )}
              </div>
              {deleteError && (
                <div
                  className="flex items-start gap-2 rounded-lg border border-rose-500/40 bg-rose-500/10 px-3 py-2 text-[11px] text-rose-200"
                  role="alert"
                >
                  <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  <div className="space-y-0.5">
                    <div className="font-medium">Delete failed</div>
                    <div className="text-rose-200/80">{deleteError}</div>
                  </div>
                </div>
              )}
            </div>
          )}
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

          {!isActive && !proposal && round.needs_revision && (
            <div
              className="space-y-1.5 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-200"
              data-testid={selectors.feedback.parseErrorNotice}
            >
              <div className="flex items-center gap-2 font-medium">
                <AlertTriangle className="h-4 w-4" aria-hidden="true" />
                {PARSE_ERROR_TITLE}
              </div>
              <p className="text-[11px] leading-relaxed text-amber-200/80">
                {PARSE_ERROR_BODY}
              </p>
              {round.last_parse_warnings && round.last_parse_warnings.length > 0 && (
                <ul className="list-disc space-y-0.5 pl-4 text-[11px] text-amber-200/70">
                  {round.last_parse_warnings.map((w, i) => (
                    <li key={i}>{w}</li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {isActive && (
            <div className="space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/5 px-3 py-2 text-xs text-cyan-200">
                <div className="flex items-center gap-2">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  {AGENT_WORKING_LABEL}
                </div>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => cancelMutation.mutate()}
                  disabled={cancelMutation.isPending}
                  className="border-rose-500/40 bg-transparent text-rose-200 hover:bg-rose-500/10 hover:text-rose-100"
                  data-testid={selectors.feedback.cancelButton}
                >
                  {cancelMutation.isPending ? (
                    <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <XCircle className="mr-1 h-3.5 w-3.5" />
                  )}
                  Cancel
                </Button>
              </div>
              {round.last_poll_error && (
                <div
                  className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-[11px] text-amber-200"
                  data-testid={selectors.feedback.pollErrorNotice}
                >
                  <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  <div className="space-y-0.5">
                    <div className="font-medium">Agent unreachable</div>
                    <div className="text-amber-200/80">{round.last_poll_error}</div>
                    {round.poll_failure_count && round.poll_failure_count > 1 ? (
                      <div className="text-amber-200/60">
                        {round.poll_failure_count} consecutive poll failures
                      </div>
                    ) : null}
                  </div>
                </div>
              )}
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
                ref={reviseRef}
                value={reviseText}
                onChange={(e) => setReviseText(e.target.value)}
                rows={2}
                disabled={continueMutation.isPending}
                placeholder={REVISE_PLACEHOLDER}
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
