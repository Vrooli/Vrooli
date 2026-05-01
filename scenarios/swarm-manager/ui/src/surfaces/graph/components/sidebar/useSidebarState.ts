/**
 * useSidebarState - Reducer hook for sidebar UI state.
 *
 * Manages active tab, search query, per-tab filters, and per-tab sort.
 * State is persisted so the global app sidebar behaves consistently across
 * routes and reloads.
 */

import { useEffect, useReducer } from "react";
import {
  DEFAULT_FILTERS,
  DEFAULT_SORT,
  SIDEBAR_TABS,
  type BacklogFilters,
  type CaptureFilters,
  type ExecutionFilters,
  type InitiativeFilters,
  type SessionFilters,
  type SidebarTab,
  type SortConfig,
  type TabFilters,
} from "./types";

// ============================================================================
// State
// ============================================================================

export type SearchMode = "plain" | "ai";

const SIDEBAR_STATE_STORAGE_KEY = "swarm-manager.sidebar.state.v1";

export interface SidebarState {
  activeTab: SidebarTab;
  searchQuery: string;
  searchMode: SearchMode;
  filters: TabFilters;
  sorts: Record<SidebarTab, SortConfig>;
}

export function createInitialState(tab: SidebarTab = "activity"): SidebarState {
  return {
    activeTab: tab,
    searchQuery: "",
    searchMode: "plain",
    filters: { ...DEFAULT_FILTERS },
    sorts: { ...DEFAULT_SORT },
  };
}

function isSidebarTab(value: unknown): value is SidebarTab {
  return typeof value === "string" && (SIDEBAR_TABS as readonly string[]).includes(value);
}

function isSearchMode(value: unknown): value is SearchMode {
  return value === "plain" || value === "ai";
}

function restoreArray<T extends string>(value: unknown): T[] {
  return Array.isArray(value) ? value.filter((entry): entry is T => typeof entry === "string") : [];
}

function restoreSort(value: unknown, fallback: SortConfig): SortConfig {
  if (!value || typeof value !== "object") return fallback;
  const record = value as Record<string, unknown>;
  const field = record.field;
  const direction = record.direction;
  return {
    field: field === "priority" || field === "recency" || field === "status" || field === "alphabetical" ? field : fallback.field,
    direction: direction === "asc" || direction === "desc" ? direction : fallback.direction,
  };
}

function loadPersistedState(fallback: SidebarState): SidebarState {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(SIDEBAR_STATE_STORAGE_KEY);
    if (!raw) return fallback;
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const filters = parsed.filters && typeof parsed.filters === "object"
      ? parsed.filters as Record<string, Record<string, unknown>>
      : {};
    const sorts = parsed.sorts && typeof parsed.sorts === "object"
      ? parsed.sorts as Record<string, unknown>
      : {};

    return {
      activeTab: isSidebarTab(parsed.activeTab) ? parsed.activeTab : fallback.activeTab,
      searchQuery: typeof parsed.searchQuery === "string" ? parsed.searchQuery : fallback.searchQuery,
      searchMode: isSearchMode(parsed.searchMode) ? parsed.searchMode : fallback.searchMode,
      filters: {
        activity: {},
        backlog: {
          statuses: restoreArray(filters.backlog?.statuses),
          kinds: restoreArray(filters.backlog?.kinds),
          priorityMin: typeof filters.backlog?.priorityMin === "number" ? filters.backlog.priorityMin : null,
          priorityMax: typeof filters.backlog?.priorityMax === "number" ? filters.backlog.priorityMax : null,
          showArchived: filters.backlog?.showArchived === true,
          validationStatus: filters.backlog?.validationStatus === "passed" || filters.backlog?.validationStatus === "failed" || filters.backlog?.validationStatus === "none"
            ? filters.backlog.validationStatus
            : "",
        },
        captures: {
          statuses: restoreArray(filters.captures?.statuses),
        },
        initiatives: {
          statuses: restoreArray(filters.initiatives?.statuses),
          showArchived: filters.initiatives?.showArchived === true,
        },
        executions: {
          statuses: restoreArray(filters.executions?.statuses),
          modes: restoreArray(filters.executions?.modes),
        },
        sessions: {
          statuses: restoreArray(filters.sessions?.statuses),
          kinds: restoreArray(filters.sessions?.kinds),
          activeOnly: filters.sessions?.activeOnly === true,
          hasProposals: filters.sessions?.hasProposals === true,
          hasAppliedArtifacts: filters.sessions?.hasAppliedArtifacts === true,
        },
      },
      sorts: {
        activity: restoreSort(sorts.activity, DEFAULT_SORT.activity),
        backlog: restoreSort(sorts.backlog, DEFAULT_SORT.backlog),
        captures: restoreSort(sorts.captures, DEFAULT_SORT.captures),
        initiatives: restoreSort(sorts.initiatives, DEFAULT_SORT.initiatives),
        executions: restoreSort(sorts.executions, DEFAULT_SORT.executions),
        sessions: restoreSort(sorts.sessions, DEFAULT_SORT.sessions),
      },
    };
  } catch {
    return fallback;
  }
}

function savePersistedState(state: SidebarState): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(SIDEBAR_STATE_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore persistence failures.
  }
}

// ============================================================================
// Actions
// ============================================================================

type SidebarAction =
  | { type: "SET_TAB"; tab: SidebarTab }
  | { type: "SET_SEARCH"; query: string }
  | { type: "SET_SEARCH_MODE"; mode: SearchMode }
  | { type: "SET_BACKLOG_FILTERS"; filters: Partial<BacklogFilters> }
  | { type: "SET_CAPTURE_FILTERS"; filters: Partial<CaptureFilters> }
  | { type: "SET_INITIATIVE_FILTERS"; filters: Partial<InitiativeFilters> }
  | { type: "SET_EXECUTION_FILTERS"; filters: Partial<ExecutionFilters> }
  | { type: "SET_SESSION_FILTERS"; filters: Partial<SessionFilters> }
  | { type: "SET_SORT"; tab: SidebarTab; sort: Partial<SortConfig> }
  | { type: "CLEAR_FILTERS"; tab: SidebarTab }
  | { type: "RESTORE_FROM_URL"; tab: SidebarTab; filters: Record<string, unknown>; sort: Record<string, unknown> };

export type { SidebarAction };

// ============================================================================
// Reducer
// ============================================================================

export function sidebarReducer(state: SidebarState, action: SidebarAction): SidebarState {
  switch (action.type) {
    case "SET_TAB":
      return { ...state, activeTab: action.tab };

    case "SET_SEARCH":
      return { ...state, searchQuery: action.query };

    case "SET_SEARCH_MODE":
      return { ...state, searchMode: action.mode };

    case "SET_BACKLOG_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          backlog: { ...state.filters.backlog, ...action.filters },
        },
      };

    case "SET_CAPTURE_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          captures: { ...state.filters.captures, ...action.filters },
        },
      };

    case "SET_INITIATIVE_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          initiatives: { ...state.filters.initiatives, ...action.filters },
        },
      };

    case "SET_EXECUTION_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          executions: { ...state.filters.executions, ...action.filters },
        },
      };

    case "SET_SESSION_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          sessions: { ...state.filters.sessions, ...action.filters },
        },
      };

    case "SET_SORT":
      return {
        ...state,
        sorts: {
          ...state.sorts,
          [action.tab]: { ...state.sorts[action.tab], ...action.sort },
        },
      };

    case "CLEAR_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          [action.tab]: DEFAULT_FILTERS[action.tab],
        },
        sorts: {
          ...state.sorts,
          [action.tab]: DEFAULT_SORT[action.tab],
        },
      };

    case "RESTORE_FROM_URL":
      return {
        ...state,
        activeTab: action.tab,
        filters: {
          ...state.filters,
          [action.tab]: { ...DEFAULT_FILTERS[action.tab], ...action.filters },
        },
        sorts: {
          ...state.sorts,
          [action.tab]: { ...DEFAULT_SORT[action.tab], ...action.sort },
        },
      };
  }
}

// ============================================================================
// Hook
// ============================================================================

export function useSidebarState(initialTab?: SidebarTab) {
  const [state, dispatch] = useReducer(sidebarReducer, initialTab, (tab) => loadPersistedState(createInitialState(tab)));

  useEffect(() => {
    savePersistedState(state);
  }, [state]);

  return [state, dispatch] as const;
}
