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
import { nextActionService } from "../services/next-action-service";
import type { FeedbackItem, MaturityItem } from "../lib/attention";

const EMPTY_MATURITY = new Map<string, MaturityItem>();

export function useCommandPostBadgeCount(): number {
  const backlogItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const snoozedKeys = useSnoozedKeys();
  const { data: nextActionFeed } = useQuery({
    queryKey: ["next-actions-feed"],
    queryFn: () => nextActionService.getFeed(),
    staleTime: 15_000,
  });

  const feedbackMap = useMemo(() => new Map<string, FeedbackItem>(), []);

  return useMemo(
    () => computeBadgeCount(groupActionItems(backlogItems, executions, captures, feedbackMap, EMPTY_MATURITY, snoozedKeys))
      + (nextActionFeed?.entries.filter((entry) => entry.entity_kind === "backlog_item").length ?? 0),
    [backlogItems, executions, captures, feedbackMap, snoozedKeys, nextActionFeed],
  );
}
