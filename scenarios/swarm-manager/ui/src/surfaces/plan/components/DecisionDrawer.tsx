/**
 * DecisionDrawer — right-side drawer hosting the Command Post's
 * DecisionStreamView wholesale (D9: it was built callback-shaped; we
 * re-host, not rewrite). Deep-linkable via ?drawer=decisions; optionally
 * scoped to a single item's questions when opened from a decide gate card.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { backlogDetailPath } from "../../../app/routes/route-paths";
import { DecisionStreamView } from "../../../components/command-post/DecisionStreamView";
import { Drawer } from "../../../components/ui/drawer";
import { aggregateCrossItemQuestions } from "../../../lib/command-post-utils";
import { backlogService } from "../../../services";
import { buildActiveBacklogKeys, useBacklogStore } from "../../../stores/backlog-store";
import { useSnoozeStore, useSnoozedKeys } from "../../../stores/snooze-store";
import type { BacklogKind } from "../../../types";

export interface DecisionDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /** Restrict the stream to one item's questions ("kind/name"), or null for all. */
  scopeItemKey: string | null;
  /** Called after the stream completes so the board refetches. */
  onCompleted: () => void;
}

export function DecisionDrawer({ isOpen, onClose, scopeItemKey, onCompleted }: DecisionDrawerProps) {
  const navigate = useNavigate();
  const snoozedKeys = useSnoozedKeys();
  const snooze = useSnoozeStore((s) => s.snooze);
  const backlogItems = useBacklogStore((s) => s.items);
  const activeItemKeys = useMemo(() => buildActiveBacklogKeys(backlogItems), [backlogItems]);

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    enabled: isOpen,
  });

  const questions = useMemo(() => {
    const pqi = summaryQuery.data?.pending_questions?.items ?? [];
    const all = aggregateCrossItemQuestions(pqi, snoozedKeys, activeItemKeys);
    if (!scopeItemKey) return all;
    return all.filter((q) => `${q.parentKind}/${q.parentName}` === scopeItemKey);
  }, [summaryQuery.data?.pending_questions, snoozedKeys, activeItemKeys, scopeItemKey]);

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={scopeItemKey ? "Decisions — one item" : "Decision stream"}
      className="md:w-[560px]"
      testId="plan-decision-drawer"
    >
      {questions.length === 0 ? (
        <p className="py-8 text-center text-sm text-slate-500" data-testid="plan-decision-drawer-empty">
          {summaryQuery.isLoading ? "Loading questions…" : "No pending questions."}
        </p>
      ) : (
        <DecisionStreamView
          questions={questions}
          onComplete={() => {
            void summaryQuery.refetch();
            onCompleted();
            onClose();
          }}
          onBack={onClose}
          onSnoozeItem={(key) => snooze(key, Date.now() + 3_600_000)}
          onOpenItem={(kind, name) => {
            navigate(backlogDetailPath(kind as BacklogKind, name));
          }}
        />
      )}
    </Drawer>
  );
}
