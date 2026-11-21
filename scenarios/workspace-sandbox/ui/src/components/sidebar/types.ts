/**
 * Type definitions for the workspace-sandbox sidebar.
 *
 * The sidebar is split into two top-level tabs:
 *   - Active:  operationally interesting (creating, active, stopped, checkpointed, error)
 *   - History: terminal-state audit (approved, rejected, deleted)
 *
 * Each tab carries its own search/filter/sort state; the reducer
 * (`useSidebarState`) keeps them isolated so switching tabs doesn't
 * stomp the other tab's user input.
 */

import type { Status } from "../../lib/api";
import { ACTIVE_STATUSES, HISTORY_STATUSES } from "../../lib/api";

export type SidebarTab = "active" | "history";

export const SIDEBAR_TABS: readonly SidebarTab[] = ["active", "history"] as const;

export const TAB_LABELS: Record<SidebarTab, string> = {
  active: "Active",
  history: "History",
};

/** Statuses the Active tab is permitted to show. */
export const ACTIVE_TAB_STATUSES = ACTIVE_STATUSES;
/** Statuses the History tab is permitted to show. */
export const HISTORY_TAB_STATUSES = HISTORY_STATUSES;

// ─── Filters ─────────────────────────────────────────────────────────

export interface ActiveFilters {
  /** Empty array = all active-tab statuses. */
  statuses: Status[];
  owner: string;
  projectRoot: string;
}

export interface HistoryFilters {
  /** Empty array = all history-tab statuses. */
  statuses: Status[];
  owner: string;
  projectRoot: string;
  search: string;
  agentManagerRunId: string;
  /** YYYY-MM-DD inclusive lower bound. */
  snapshotAtFrom: string;
  /** YYYY-MM-DD inclusive upper bound. */
  snapshotAtTo: string;
}

export interface TabFilters {
  active: ActiveFilters;
  history: HistoryFilters;
}

// ─── Sort ────────────────────────────────────────────────────────────

export type ActiveSortField = "createdAt" | "lastUsedAt" | "fileCount" | "sizeBytes";
export type HistorySortField = "snapshotAt" | "totalBlobBytes" | "fileCount";

export interface SortConfig<F extends string = string> {
  field: F;
  direction: "asc" | "desc";
}

export interface TabSorts {
  active: SortConfig<ActiveSortField>;
  history: SortConfig<HistorySortField>;
}

// ─── Defaults ────────────────────────────────────────────────────────

export const DEFAULT_FILTERS: TabFilters = {
  active: {
    statuses: [],
    owner: "",
    projectRoot: "",
  },
  history: {
    statuses: [],
    owner: "",
    projectRoot: "",
    search: "",
    agentManagerRunId: "",
    snapshotAtFrom: "",
    snapshotAtTo: "",
  },
};

export const DEFAULT_SORT: TabSorts = {
  active: { field: "createdAt", direction: "desc" },
  history: { field: "snapshotAt", direction: "desc" },
};

// ─── Toast (selection-on-transition UX) ──────────────────────────────

export type SidebarToastTone = "info" | "success";

export interface SidebarToast {
  id: number;
  message: string;
  tone: SidebarToastTone;
}
