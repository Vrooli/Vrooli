/**
 * Sidebar Types
 *
 * Shared type definitions for the multi-tab sidebar.
 */

import type { BacklogKind, BacklogStatus, CaptureStatus, ExecutionMode, ExecutionStatus, InitiativeStatus } from "../../../../types";

// ============================================================================
// Tab Definitions
// ============================================================================

export const SIDEBAR_TABS = ["activity", "backlog", "captures", "initiatives", "executions"] as const;
export type SidebarTab = (typeof SIDEBAR_TABS)[number];

export const TAB_LABELS: Record<SidebarTab, string> = {
  activity: "Activity",
  backlog: "Backlog",
  captures: "Captures",
  initiatives: "Initiatives",
  executions: "Executions",
};

// ============================================================================
// Sort Definitions
// ============================================================================

export type SortField = "priority" | "recency" | "status" | "alphabetical";
export type SortDirection = "asc" | "desc";

export interface SortConfig {
  field: SortField;
  direction: SortDirection;
}

export const DEFAULT_SORT: Record<SidebarTab, SortConfig> = {
  activity: { field: "priority", direction: "asc" },
  backlog: { field: "priority", direction: "asc" },
  captures: { field: "recency", direction: "desc" },
  initiatives: { field: "alphabetical", direction: "asc" },
  executions: { field: "recency", direction: "desc" },
};

// ============================================================================
// Filter Definitions
// ============================================================================

export type ValidationStatusFilter = "passed" | "failed" | "none" | "";

export interface BacklogFilters {
  statuses: BacklogStatus[];
  kinds: BacklogKind[];
  priorityMin: number | null;
  priorityMax: number | null;
  showArchived: boolean;
  validationStatus: ValidationStatusFilter;
}

export interface CaptureFilters {
  statuses: CaptureStatus[];
}

export interface InitiativeFilters {
  statuses: InitiativeStatus[];
  showArchived: boolean;
}

export interface ExecutionFilters {
  statuses: ExecutionStatus[];
  modes: ExecutionMode[];
}

export interface TabFilters {
  activity: Record<string, never>;
  backlog: BacklogFilters;
  captures: CaptureFilters;
  initiatives: InitiativeFilters;
  executions: ExecutionFilters;
}

export const DEFAULT_FILTERS: TabFilters = {
  activity: {},
  backlog: { statuses: [], kinds: [], priorityMin: null, priorityMax: null, showArchived: false, validationStatus: "" },
  captures: { statuses: [] },
  initiatives: { statuses: [], showArchived: false },
  executions: { statuses: [], modes: [] },
};
