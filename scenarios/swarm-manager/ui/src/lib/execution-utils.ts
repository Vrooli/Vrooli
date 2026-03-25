import type { ExecutionMode, ExecutionRecord, ExecutionStatus } from "../types";

export type ExecutionTabId = "all" | "pending" | "running" | "review" | "completed" | "failed";

interface ExecutionTabConfig {
  id: ExecutionTabId;
  label: string;
  statuses: ExecutionStatus[];
  emptyTitle: string;
  emptyDescription: string;
}

export const EXECUTION_TAB_CONFIG: ExecutionTabConfig[] = [
  {
    id: "all",
    label: "All Runs",
    statuses: ["pending", "scheduled", "starting", "running", "needs_review", "validating", "needs_fixup", "completed", "failed", "canceled"],
    emptyTitle: "No executions yet",
    emptyDescription: "Queue a backlog item to create your first execution run.",
  },
  {
    id: "pending",
    label: "Pending",
    statuses: ["pending", "scheduled"],
    emptyTitle: "No pending runs",
    emptyDescription: "Newly queued and scheduled runs will appear here.",
  },
  {
    id: "running",
    label: "Running",
    statuses: ["starting", "running", "validating"],
    emptyTitle: "No running runs",
    emptyDescription: "When a run is active, it appears here with live status updates.",
  },
  {
    id: "review",
    label: "Needs Review",
    statuses: ["needs_review", "needs_fixup"],
    emptyTitle: "No runs awaiting review",
    emptyDescription: "Runs that finish and need human approval will appear here.",
  },
  {
    id: "completed",
    label: "Completed",
    statuses: ["completed"],
    emptyTitle: "No completed runs",
    emptyDescription: "Finished runs will appear here when work succeeds.",
  },
  {
    id: "failed",
    label: "Failed",
    statuses: ["failed", "canceled"],
    emptyTitle: "No failed runs",
    emptyDescription: "Failed or canceled runs appear here so you can retry or inspect them.",
  },
];

export interface ExecutionFilters {
  searchTerm: string;
  statusFilter: ExecutionStatus | "";
  modeFilter: ExecutionMode | "";
  startedByFilter: string;
  operationFilter: "generator" | "improver" | "";
  backlogFilter: string;
  fromFilter: string;
  toFilter: string;
}

export const isExecutionInTab = (item: ExecutionRecord, tab: ExecutionTabId): boolean => {
  const config = EXECUTION_TAB_CONFIG.find((entry) => entry.id === tab);
  if (!config) {
    return true;
  }
  return config.statuses.includes(item.status);
};

export const isExecutionActive = (item: ExecutionRecord): boolean =>
  item.status === "pending" || item.status === "scheduled" || item.status === "starting" || item.status === "running" || item.status === "needs_review" || item.status === "validating";

const parseDateFilter = (value: string): number | null => {
  if (!value) {
    return null;
  }
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? null : parsed;
};

const matchesSearch = (item: ExecutionRecord, searchTerm: string): boolean => {
  if (!searchTerm.trim()) {
    return true;
  }
  const term = searchTerm.trim().toLowerCase();
  const haystack = [
    item.executionId,
    item.backlogKind,
    item.backlogName,
    item.status,
    item.mode,
    item.startedBy ?? "",
    item.runId ?? "",
    item.taskId ?? "",
    item.failureReason ?? "",
  ]
    .join(" ")
    .toLowerCase();

  return haystack.includes(term);
};

export const matchesExecutionFilters = (item: ExecutionRecord, filters: ExecutionFilters): boolean => {
  if (!matchesSearch(item, filters.searchTerm)) {
    return false;
  }

  if (filters.statusFilter && item.status !== filters.statusFilter) {
    return false;
  }

  if (filters.modeFilter && item.mode !== filters.modeFilter) {
    return false;
  }

  if (filters.startedByFilter) {
    const startedBy = (item.startedBy ?? "").toLowerCase();
    if (!startedBy.includes(filters.startedByFilter.toLowerCase())) {
      return false;
    }
  }

  if (filters.operationFilter && item.operation !== filters.operationFilter) {
    return false;
  }

  if (filters.backlogFilter) {
    const backlogValue = `${item.backlogKind}/${item.backlogName}`.toLowerCase();
    if (!backlogValue.includes(filters.backlogFilter.toLowerCase())) {
      return false;
    }
  }

  const createdAt = new Date(item.createdAt).getTime();
  if (Number.isNaN(createdAt)) {
    return false;
  }

  const fromDate = parseDateFilter(filters.fromFilter);
  if (fromDate !== null && createdAt < fromDate) {
    return false;
  }

  const toDate = parseDateFilter(filters.toFilter);
  if (toDate !== null && createdAt > toDate) {
    return false;
  }

  return true;
};

export const canStartExecution = (status: ExecutionStatus): boolean =>
  status === "pending" || status === "scheduled";

export const canCancelExecution = (status: ExecutionStatus): boolean =>
  status === "pending" || status === "scheduled" || status === "starting" || status === "running" || status === "needs_review" || status === "validating";

export const canFollowUpExecution = (status: ExecutionStatus): boolean =>
  status === "completed" || status === "failed" || status === "needs_fixup";

export const canRetryExecution = (status: ExecutionStatus): boolean => status === "failed";
