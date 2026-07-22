/**
 * useCommandPostBadgeCount — Derives badge count from existing stores.
 *
 * Subscribes to backlog, execution, capture, and snooze stores.
 * Computes via groupActionItems + computeBadgeCount from command-post-utils.
 */

import { useMemo } from "react";
import { useBacklogStore } from "../stores/backlog-store";
import { useExecutionStore } from "../stores/execution-store";
import { useCaptureStore } from "../stores/capture-store";
import { useSnoozedKeys } from "../stores/snooze-store";
import { groupActionItems, computeBadgeCount } from "../lib/command-post-utils";
import type { FeedbackItem, MaturityItem } from "../lib/attention";

const EMPTY_MATURITY = new Map<string, MaturityItem>();

export function useCommandPostBadgeCount(): number {
  const backlogItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const snoozedKeys = useSnoozedKeys();

  const feedbackMap = useMemo(() => new Map<string, FeedbackItem>(), []);

  return useMemo(
    () => computeBadgeCount(groupActionItems(backlogItems, executions, captures, feedbackMap, EMPTY_MATURITY, snoozedKeys)),
    [backlogItems, executions, captures, feedbackMap, snoozedKeys],
  );
}
