/**
 * useNodeActionContext — Builds ActionContext and returns ItemActions for a backlog graph node.
 *
 * Shares the "backlog-summary" react-query cache with useCommandPostBadgeCount
 * so no extra API calls are made.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBacklogStore } from "../../../stores/backlog-store";
import { useExecutionStore } from "../../../stores/execution-store";
import { backlogService } from "../../../services";
import { getItemActions, type ItemActions } from "../../../lib/backlog-queue-utils";
import type { BacklogGraphNodeData } from "../types";

export function useNodeActionContext(nodeData: BacklogGraphNodeData): ItemActions {
  const allItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  return useMemo(() => {
    const key = `${nodeData.kind}/${nodeData.name}`;

    // Find the full backlog item to get dependsOn.
    const fullItem = allItems.find((i) => i.kind === nodeData.kind && i.name === nodeData.name);
    const item = fullItem ?? { kind: nodeData.kind, name: nodeData.name, status: nodeData.status, dependsOn: [] };

    // Extract feedback/maturity for this specific item from the summary cache.
    const feedbackItem = (summaryQuery.data?.feedback?.items ?? []).find(
      (f) => `${f.kind}/${f.name}` === key,
    );
    const maturityItem = (summaryQuery.data?.maturity?.items ?? []).find(
      (m) => `${m.kind}/${m.name}` === key,
    );

    const hasExecutionHistory = executions.some(
      (e) => e.backlogKind === nodeData.kind && e.backlogName === nodeData.name,
    );

    return getItemActions({
      item,
      allItems,
      readinessReady: maturityItem ? (maturityItem.ready ?? null) : null,
      pendingSynthesis: maturityItem?.pending_synthesis ?? false,
      agentRunning: false,
      hasPendingDecisions: (feedbackItem?.pending_decisions ?? 0) > 0,
      hasExecutionHistory,
    });
  }, [nodeData.kind, nodeData.name, nodeData.status, allItems, executions, summaryQuery.data]);
}
