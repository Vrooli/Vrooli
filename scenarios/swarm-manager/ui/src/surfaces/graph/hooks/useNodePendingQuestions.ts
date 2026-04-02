/**
 * useNodePendingQuestions — Returns pending questions for a specific backlog item.
 *
 * Shares the "backlog-summary" react-query cache — no extra API call.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { backlogService } from "../../../services";
import type { BacklogKind, PendingQuestion } from "../../../types";

export function useNodePendingQuestions(kind: BacklogKind, name: string): PendingQuestion[] {
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  return useMemo(() => {
    const items = summaryQuery.data?.pending_questions?.items ?? [];
    const match = items.find((pqi) => pqi.kind === kind && pqi.name === name);
    return match?.questions ?? [];
  }, [summaryQuery.data?.pending_questions, kind, name]);
}
