/**
 * Sidebar Types
 *
 * Shared type definitions for the multi-tab sidebar.
 */

import type {
  AgentSessionKind,
  AgentSessionStatus,
  BacklogKind,
  BacklogStatus,
  CaptureStatus,
  ExecutionMode,
  ExecutionStatus,
  GoalStatus,
} from "../../../../types";

// ============================================================================
// Tab Definitions
// ============================================================================

export const SIDEBAR_TABS = ["backlog", "captures", "goals", "executions", "sessions"] as const;
export type SidebarTab = (typeof SIDEBAR_TABS)[number];

export const TAB_LABELS: Record<SidebarTab, string> = {
  backlog: "Backlog",
  captures: "Captures",
  goals: "Goals",
  executions: "Executions",
  sessions: "Sessions",
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
  backlog: { field: "priority", direction: "asc" },
  captures: { field: "recency", direction: "desc" },
  goals: { field: "priority", direction: "desc" },
  executions: { field: "recency", direction: "desc" },
  sessions: { field: "recency", direction: "desc" },
};

// ============================================================================
// Filter Definitions
// ============================================================================

export interface BacklogFilters {
  statuses: BacklogStatus[];
  kinds: BacklogKind[];
  priorityMin: number | null;
  priorityMax: number | null;
  showArchived: boolean;
}

export interface CaptureFilters {
  statuses: CaptureStatus[];
}

/** Compatibility shape for the unmounted legacy goal component. */
export interface GoalFilters {
  statuses: GoalStatus[];
  showArchived: boolean;
}

export interface ExecutionFilters {
  statuses: ExecutionStatus[];
  modes: ExecutionMode[];
}

export interface SessionFilters {
  statuses: AgentSessionStatus[];
  kinds: AgentSessionKind[];
  activeOnly: boolean;
  hasProposals: boolean;
  hasAppliedArtifacts: boolean;
}

export interface TabFilters {
  backlog: BacklogFilters;
  captures: CaptureFilters;
  goals: Record<string, never>;
  executions: ExecutionFilters;
  sessions: SessionFilters;
}

export const DEFAULT_FILTERS: TabFilters = {
  backlog: { statuses: [], kinds: [], priorityMin: null, priorityMax: null, showArchived: false },
  captures: { statuses: [] },
  goals: {},
  executions: { statuses: [], modes: [] },
  sessions: { statuses: [], kinds: [], activeOnly: false, hasProposals: false, hasAppliedArtifacts: false },
};
