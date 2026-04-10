/**
 * useCommandPostBadgeCount — Derives badge count from existing stores.
 *
 * Subscribes to backlog, execution, capture, and snooze stores.
 * Computes via groupActionItems + computeBadgeCount from command-post-utils.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBacklogStore } from "../stores/backlog-store";
import { useExecutionStore } from "../stores/execution-store";
import { useCaptureStore } from "../stores/capture-store";
import { useSnoozedKeys } from "../stores/snooze-store";
import { groupActionItems, computeBadgeCount } from "../lib/command-post-utils";
import { backlogService } from "../services";
import type { FeedbackItem, MaturityItem } from "../lib/feed";

export function useCommandPostBadgeCount(): number {
  const backlogItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const snoozedKeys = useSnoozedKeys();

  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
  });

  const feedbackMap = useMemo(() => {
    const map = new Map<string, FeedbackItem>();
    for (const item of summaryQuery.data?.feedback?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        pendingDecisions: item.pending_decisions ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.feedback]);

  const maturityMap = useMemo(() => {
    const map = new Map<string, MaturityItem>();
    for (const item of summaryQuery.data?.maturity?.items ?? []) {
      map.set(`${item.kind}/${item.name}`, {
        kind: item.kind,
        name: item.name,
        ready: item.ready ?? false,
        pendingItems: item.pending_items ?? 0,
      });
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  return useMemo(
    () => computeBadgeCount(groupActionItems(backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys)),
    [backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys],
  );
}
