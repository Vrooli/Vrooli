import type { BacklogItem, BacklogKind, BacklogStatus } from "../types";

export const QUEUEABLE_BACKLOG_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready"];

interface QueueableBacklogItem {
  kind: BacklogKind;
  status: BacklogStatus;
}

export const isBacklogQueueable = (item: QueueableBacklogItem): boolean =>
  (item.kind !== "research" && QUEUEABLE_BACKLOG_STATUSES.includes(item.status)) ||
  (item.kind === "idea" && item.status === "archived");

/** Statuses that indicate a dependency is not yet planned/ready — blocking downstream items. */
const BLOCKING_DEP_STATUSES = new Set<BacklogStatus>(["backlog", "researching"]);

/**
 * Check whether any of an item's dependencies are still in an unplanned state,
 * meaning this item should not be run yet.
 */
export function hasBlockingDeps(item: Pick<BacklogItem, "dependsOn">, allItems: BacklogItem[]): boolean {
  if (!item.dependsOn || item.dependsOn.length === 0) return false;
  const itemsByKey = new Map(allItems.map((i) => [`${i.kind}/${i.name}`, i]));
  return item.dependsOn.some((dep) => {
    const depItem = itemsByKey.get(dep);
    return depItem && BLOCKING_DEP_STATUSES.has(depItem.status);
  });
}

export const getBacklogNotQueueableReason = (item: QueueableBacklogItem): string | null => {
  if (isBacklogQueueable(item)) {
    return null;
  }
  if (item.kind === "research") {
    return "Research items must be converted before queueing.";
  }
  switch (item.status) {
    case "queued":
      return "Already queued. Check Execution for run progress.";
    case "in_progress":
      return "Already in progress. Wait for it to finish before re-queueing.";
    case "completed":
      return "Completed items cannot be queued again.";
    case "failed":
      return "Reset status to retry. Check Execution History for failure details.";
    case "archived":
      return "Only archived ideas can be queued directly.";
    default:
      return "This item cannot be queued from its current status.";
  }
};
