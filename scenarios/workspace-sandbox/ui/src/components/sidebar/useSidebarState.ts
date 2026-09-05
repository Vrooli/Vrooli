/**
 * Reducer + hook for sidebar UI state.
 *
 * Owns the active tab pointer, search query, per-tab filters, and
 * per-tab sort. State is persisted to `localStorage` so the sidebar
 * behaves consistently across reloads (matches the swarm-manager
 * `useSidebarState` pattern).
 */

import { useEffect, useReducer } from "react";

import type { Status } from "../../lib/api";
import {
  ACTIVE_STATUSES,
  HISTORY_STATUSES,
} from "../../lib/api";
import {
  DEFAULT_FILTERS,
  DEFAULT_SORT,
  SIDEBAR_TABS,
  type ActiveFilters,
  type ActiveSortField,
  type HistoryFilters,
  type HistorySortField,
  type SidebarTab,
  type SortConfig,
  type TabFilters,
  type TabSorts,
} from "./types";

const STORAGE_KEY = "workspace-sandbox.sidebar.state.v1";

export interface SidebarState {
  activeTab: SidebarTab;
  /** Per-tab search input. History tab also uses its filter `search`
   *  string; active-tab search is purely client-side. */
  searchQuery: { active: string; history: string };
  filters: TabFilters;
  sorts: TabSorts;
}

export type SidebarAction =
  | { type: "SET_TAB"; tab: SidebarTab }
  | { type: "SET_SEARCH"; tab: SidebarTab; query: string }
  | { type: "SET_ACTIVE_FILTERS"; filters: Partial<ActiveFilters> }
  | { type: "SET_HISTORY_FILTERS"; filters: Partial<HistoryFilters> }
  | { type: "SET_ACTIVE_SORT"; sort: Partial<SortConfig<ActiveSortField>> }
  | { type: "SET_HISTORY_SORT"; sort: Partial<SortConfig<HistorySortField>> }
  | { type: "CLEAR_FILTERS"; tab: SidebarTab };

export function createInitialState(): SidebarState {
  return {
    activeTab: "active",
    searchQuery: { active: "", history: "" },
    filters: { ...DEFAULT_FILTERS },
    sorts: { ...DEFAULT_SORT },
  };
}

// ─── Persistence ──────────────────────────────────────────────────────

function isSidebarTab(value: unknown): value is SidebarTab {
  return typeof value === "string" && (SIDEBAR_TABS as readonly string[]).includes(value);
}

function isStatus(value: unknown, allowed: readonly Status[]): value is Status {
  return typeof value === "string" && (allowed as readonly string[]).includes(value);
}

function restoreStatusArray(value: unknown, allowed: readonly Status[]): Status[] {
  if (!Array.isArray(value)) return [];
  return value.filter((entry): entry is Status => isStatus(entry, allowed));
}

function restoreString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function restoreActiveSort(value: unknown): SortConfig<ActiveSortField> {
  const fallback = DEFAULT_SORT.active;
  if (!value || typeof value !== "object") return fallback;
  const record = value as Record<string, unknown>;
  const field: ActiveSortField =
    record.field === "createdAt" ||
    record.field === "lastUsedAt" ||
    record.field === "fileCount" ||
    record.field === "sizeBytes"
      ? record.field
      : fallback.field;
  const direction = record.direction === "asc" || record.direction === "desc" ? record.direction : fallback.direction;
  return { field, direction };
}

function restoreHistorySort(value: unknown): SortConfig<HistorySortField> {
  const fallback = DEFAULT_SORT.history;
  if (!value || typeof value !== "object") return fallback;
  const record = value as Record<string, unknown>;
  const field: HistorySortField =
    record.field === "snapshotAt" || record.field === "totalBlobBytes" || record.field === "fileCount"
      ? record.field
      : fallback.field;
  const direction = record.direction === "asc" || record.direction === "desc" ? record.direction : fallback.direction;
  return { field, direction };
}

function loadPersisted(fallback: SidebarState): SidebarState {
  if (typeof window === "undefined") return fallback;
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const filters = (parsed.filters && typeof parsed.filters === "object"
      ? parsed.filters
      : {}) as Record<string, Record<string, unknown>>;
    const sorts = (parsed.sorts && typeof parsed.sorts === "object"
      ? parsed.sorts
      : {}) as Record<string, unknown>;
    const search = (parsed.searchQuery && typeof parsed.searchQuery === "object"
      ? parsed.searchQuery
      : {}) as Record<string, unknown>;

    return {
      activeTab: isSidebarTab(parsed.activeTab) ? parsed.activeTab : fallback.activeTab,
      searchQuery: {
        active: restoreString(search.active),
        history: restoreString(search.history),
      },
      filters: {
        active: {
          statuses: restoreStatusArray(filters.active?.statuses, ACTIVE_STATUSES),
          owner: restoreString(filters.active?.owner),
          projectRoot: restoreString(filters.active?.projectRoot),
        },
        history: {
          statuses: restoreStatusArray(filters.history?.statuses, HISTORY_STATUSES),
          owner: restoreString(filters.history?.owner),
          projectRoot: restoreString(filters.history?.projectRoot),
          search: restoreString(filters.history?.search),
          agentManagerRunId: restoreString(filters.history?.agentManagerRunId),
          snapshotAtFrom: restoreString(filters.history?.snapshotAtFrom),
          snapshotAtTo: restoreString(filters.history?.snapshotAtTo),
        },
      },
      sorts: {
        active: restoreActiveSort(sorts.active),
        history: restoreHistorySort(sorts.history),
      },
    };
  } catch {
    return fallback;
  }
}

function savePersisted(state: SidebarState): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Quota exceeded or storage disabled — silently degrade.
  }
}

// ─── Reducer ──────────────────────────────────────────────────────────

export function sidebarReducer(state: SidebarState, action: SidebarAction): SidebarState {
  switch (action.type) {
    case "SET_TAB":
      return state.activeTab === action.tab ? state : { ...state, activeTab: action.tab };

    case "SET_SEARCH":
      return {
        ...state,
        searchQuery: { ...state.searchQuery, [action.tab]: action.query },
      };

    case "SET_ACTIVE_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          active: { ...state.filters.active, ...action.filters },
        },
      };

    case "SET_HISTORY_FILTERS":
      return {
        ...state,
        filters: {
          ...state.filters,
          history: { ...state.filters.history, ...action.filters },
        },
      };

    case "SET_ACTIVE_SORT":
      return {
        ...state,
        sorts: {
          ...state.sorts,
          active: { ...state.sorts.active, ...action.sort },
        },
      };

    case "SET_HISTORY_SORT":
      return {
        ...state,
        sorts: {
          ...state.sorts,
          history: { ...state.sorts.history, ...action.sort },
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
        searchQuery: { ...state.searchQuery, [action.tab]: "" },
      };
  }
}

// ─── Hook ─────────────────────────────────────────────────────────────

export function useSidebarState() {
  const [state, dispatch] = useReducer(sidebarReducer, undefined, () =>
    loadPersisted(createInitialState()),
  );

  useEffect(() => {
    savePersisted(state);
  }, [state]);

  return [state, dispatch] as const;
}
