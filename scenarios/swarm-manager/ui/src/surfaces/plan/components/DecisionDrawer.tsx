/** A deep-linkable decision flow for questions and mutation proposals. */
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { backlogDetailPath } from "../../../app/routes/route-paths";
import { DecisionStreamView } from "../../../components/command-post/DecisionStreamView";
import { ProposalDecisionStreamView, type ProposalDecisionStreamItem } from "../../../components/command-post/ProposalDecisionStreamView";
import { Drawer } from "../../../components/ui/drawer";
import { aggregateCrossItemQuestions } from "../../../lib/command-post-utils";
import { backlogService } from "../../../services";
import { proposalSessionService } from "../../../services/proposal-session-service";
import { buildActiveBacklogKeys, useBacklogStore } from "../../../stores/backlog-store";
import { useSnoozeStore, useSnoozedKeys } from "../../../stores/snooze-store";
import type { BacklogKind } from "../../../types";

export interface DecisionDrawerProps { isOpen: boolean; onClose: () => void; scopeItemKey: string | null; onCompleted: () => void; }

export function DecisionDrawer({ isOpen, onClose, scopeItemKey, onCompleted }: DecisionDrawerProps) {
  const navigate = useNavigate();
  const snoozedKeys = useSnoozedKeys();
  const snooze = useSnoozeStore((s) => s.snooze);
  const backlogItems = useBacklogStore((s) => s.items);
  const activeItemKeys = useMemo(() => buildActiveBacklogKeys(backlogItems), [backlogItems]);
  const [flowStage, setFlowStage] = useState<"questions" | "proposals">("questions");
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
    .map((proposal) => ({ sessionId: session.id, sessionTitle: session.title, proposal }))), [proposalQuery.data]);
  const hasProposalStage = proposalQuery.isLoading || proposals.length > 0;
  const showProposals = flowStage === "proposals" || questions.length === 0;
  const complete = () => { void summaryQuery.refetch(); void proposalQuery.refetch(); onCompleted(); onClose(); };
  return <Drawer isOpen={isOpen} onClose={onClose} title={scopeItemKey ? "Decisions — one item" : "Decision stream"} className="md:w-[560px]" testId="plan-decision-drawer">
    {showProposals && proposalQuery.isLoading ? <p className="py-8 text-center text-sm text-slate-500">Loading proposals…</p>
      : showProposals && proposals.length > 0 ? <ProposalDecisionStreamView proposals={proposals} onBack={() => setFlowStage("questions")} onComplete={complete} onSnooze={(id) => snooze(`proposal:${id}`, Date.now() + 3_600_000)} />
      : questions.length === 0 && scopeItemKey ? <p className="py-8 text-center text-sm text-slate-500" data-testid="plan-decision-drawer-empty">{summaryQuery.isLoading ? "Loading questions…" : "No pending questions."}</p>
        : questions.length > 0 ? <DecisionStreamView questions={questions} onComplete={complete} onBack={onClose} onSnoozeItem={(key) => snooze(key, Date.now() + 3_600_000)} onOpenItem={(kind, name) => navigate(backlogDetailPath(kind as BacklogKind, name))} onQueueComplete={hasProposalStage ? () => setFlowStage("proposals") : undefined} finalActionLabel={hasProposalStage ? "Next" : undefined} />
          : <p className="py-4 text-center text-sm text-slate-500">No pending decisions.</p>}
  </Drawer>;
}
