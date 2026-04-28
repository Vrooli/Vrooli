/**
 * FeedbackPanel — the initiative's feedback surface.
 *
 * Lists every feedback round in reverse chronological order and owns:
 *   - fetching rounds via useQuery (with a short poll while any round is in
 *     the agent_thinking state)
 *   - the expand/collapse state across rounds
 *   - the Add Feedback dialog handoff
 *
 * Per-round behavior (decide / revise / dismiss + proposal rendering) lives
 * in feedback-round-card.tsx so this file stays a thin list shell.
 */

import { useCallback, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { MessageCirclePlus, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { FeedbackDialog } from "./feedback-dialog";
import { FeedbackRoundCard } from "./feedback-round-card";
import { feedbackService } from "../../services/feedback-service";
import type { BacklogStatus } from "../../types";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import { defaultQueryOptions } from "../../lib";

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
    status: BacklogStatus;
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
        items={(previewItems ?? [])
          .filter((it) => !it.archivedAt && !it.missing)
          .map((it) => ({ ref: `${it.kind}/${it.name}`, title: it.title }))}
        onSubmitted={(r) => {
          invalidate();
          setExpanded((prev) => new Set(prev).add(r.number));
        }}
      />
    </section>
  );
}
