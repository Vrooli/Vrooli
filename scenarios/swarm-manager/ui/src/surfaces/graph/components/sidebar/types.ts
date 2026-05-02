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
  InitiativeStatus,
} from "../../../../types";

// ============================================================================
// Tab Definitions
// ============================================================================

export const SIDEBAR_TABS = ["activity", "backlog", "captures", "initiatives", "operatingModes", "executions", "sessions"] as const;
export type SidebarTab = (typeof SIDEBAR_TABS)[number];

export const TAB_LABELS: Record<SidebarTab, string> = {
  activity: "Activity",
  backlog: "Backlog",
  captures: "Captures",
  initiatives: "Initiatives",
  operatingModes: "Operating Modes",
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
  activity: { field: "priority", direction: "asc" },
  backlog: { field: "priority", direction: "asc" },
  captures: { field: "recency", direction: "desc" },
  initiatives: { field: "alphabetical", direction: "asc" },
  operatingModes: { field: "alphabetical", direction: "asc" },
  executions: { field: "recency", direction: "desc" },
  sessions: { field: "recency", direction: "desc" },
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

export interface SessionFilters {
  statuses: AgentSessionStatus[];
  kinds: AgentSessionKind[];
  activeOnly: boolean;
  hasProposals: boolean;
  hasAppliedArtifacts: boolean;
}

export interface TabFilters {
  activity: Record<string, never>;
  backlog: BacklogFilters;
  captures: CaptureFilters;
  initiatives: InitiativeFilters;
  operatingModes: Record<string, never>;
  executions: ExecutionFilters;
  sessions: SessionFilters;
}

export const DEFAULT_FILTERS: TabFilters = {
  activity: {},
  backlog: { statuses: [], kinds: [], priorityMin: null, priorityMax: null, showArchived: false, validationStatus: "" },
  captures: { statuses: [] },
  initiatives: { statuses: [], showArchived: false },
  operatingModes: {},
  executions: { statuses: [], modes: [] },
  sessions: { statuses: [], kinds: [], activeOnly: false, hasProposals: false, hasAppliedArtifacts: false },
};
