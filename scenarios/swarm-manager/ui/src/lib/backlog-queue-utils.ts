import type { BacklogKind, BacklogStatus } from "../types";

export const QUEUEABLE_BACKLOG_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready"];

interface QueueableBacklogItem {
  kind: BacklogKind;
  status: BacklogStatus;
}

export const isBacklogQueueable = (item: QueueableBacklogItem): boolean =>
  (item.kind !== "research" && QUEUEABLE_BACKLOG_STATUSES.includes(item.status)) ||
  (item.kind === "idea" && item.status === "archived");

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
    case "archived":
      return "Only archived ideas can be queued directly.";
    default:
      return "This item cannot be queued from its current status.";
  }
};
