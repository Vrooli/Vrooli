/**
 * useSidebarState - Reducer hook for sidebar UI state.
 *
 * Manages active tab, search query, per-tab filters, and per-tab sort.
 * URL sync is handled at the Sidebar component level.
 */

import { useReducer } from "react";
import {
  DEFAULT_FILTERS,
  DEFAULT_SORT,
  type BacklogFilters,
  type CaptureFilters,
  type ExecutionFilters,
  type InitiativeFilters,
  type SidebarTab,
  type SortConfig,
  type TabFilters,
} from "./types";

// ============================================================================
// State
// ============================================================================

export type SearchMode = "plain" | "ai";

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
  return useReducer(sidebarReducer, initialTab, (tab) => createInitialState(tab));
}
