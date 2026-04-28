import { useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { DecisionStreamView } from "../../components/command-post/DecisionStreamView";
import { ClarificationPanel } from "../../components/backlog/clarification-panel";
import { backlogService } from "../../services";
import { useBacklogStore, buildActiveBacklogKeys } from "../../stores/backlog-store";
import { useSnoozeStore, useSnoozedKeys } from "../../stores/snooze-store";
import { aggregateCrossItemQuestions } from "../../lib/command-post-utils";
import { backlogDetailPath, commandPostPath } from "../../app/routes/route-paths";
import { useEscapeRouteBack } from "../../app/routes/useEscapeRouteBack";
import { useAppShell } from "../../app/shell/AppShellContext";
import type { BacklogKind } from "../../types";

export function DecisionStreamPage() {
  const navigate = useNavigate();
  const { openSidebar } = useAppShell();
  const snoozedKeys = useSnoozedKeys();
  const snooze = useSnoozeStore((s) => s.snooze);
  const backlogItems = useBacklogStore((s) => s.items);
  const activeItemKeys = useMemo(() => buildActiveBacklogKeys(backlogItems), [backlogItems]);

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  const questions = useMemo(() => {
    const pqi = summaryQuery.data?.pending_questions?.items ?? [];
    return aggregateCrossItemQuestions(pqi, snoozedKeys, activeItemKeys);
  }, [summaryQuery.data?.pending_questions, snoozedKeys, activeItemKeys]);

  const backToCommandPost = useCallback(() => {
    navigate(commandPostPath(), { replace: true });
  }, [navigate]);
  useEscapeRouteBack(backToCommandPost);

  return (
    <div className="h-screen bg-slate-950 text-slate-50" data-testid="decision-stream-page">
      <DecisionStreamView
        questions={questions}
        onComplete={() => {
          void summaryQuery.refetch?.();
          navigate(commandPostPath(), { replace: true });
        }}
        onBack={backToCommandPost}
        onOpenSidebar={openSidebar}
        onSnoozeItem={(key) => snooze(key, Date.now() + 3_600_000)}
        onOpenItem={(kind, name) => {
          navigate(backlogDetailPath(kind as BacklogKind, name));
        }}
      />
      <ClarificationPanel
        onAction={(action) => {
          if (action === "invalidate_round" || action === "remove_decision" || action === "update_decision") {
            void summaryQuery.refetch();
          }
        }}
      />
    </div>
  );
}
