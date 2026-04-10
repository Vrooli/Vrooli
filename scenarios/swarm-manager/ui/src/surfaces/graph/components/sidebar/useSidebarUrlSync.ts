/**
 * useSidebarUrlSync - Bidirectional URL ↔ sidebar state synchronization.
 *
 * Only the active tab's filters are written to the URL. On page load,
 * the active tab and its filters are restored from URL params.
 */

import { useCallback, useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import type { SidebarAction } from "./useSidebarState";
import type { SidebarState } from "./useSidebarState";
import {
  DEFAULT_FILTERS,
  DEFAULT_SORT,
  SIDEBAR_TABS,
  URL_PARAMS,
  type SidebarTab,
  type SortDirection,
  type SortField,
} from "./types";

const SORT_FIELDS = new Set<SortField>(["priority", "recency", "status", "alphabetical"]);
const SORT_DIRS = new Set<SortDirection>(["asc", "desc"]);

function isSidebarTab(value: string | null): value is SidebarTab {
  return value !== null && (SIDEBAR_TABS as readonly string[]).includes(value);
}

/**
 * Restore sidebar state from URL on mount.
 */
export function useRestoreFromUrl(dispatch: React.Dispatch<SidebarAction>): void {
  const [searchParams] = useSearchParams();
  const restored = useRef(false);

  useEffect(() => {
    if (restored.current) return;
    restored.current = true;

    const tabParam = searchParams.get(URL_PARAMS.tab);
    if (!isSidebarTab(tabParam)) return;

    const tab = tabParam;
    const sortField = searchParams.get(URL_PARAMS.sortField);
    const sortDir = searchParams.get(URL_PARAMS.sortDirection);

    const sort: Record<string, unknown> = {};
    if (sortField && SORT_FIELDS.has(sortField as SortField)) sort.field = sortField;
    if (sortDir && SORT_DIRS.has(sortDir as SortDirection)) sort.direction = sortDir;

    const statusStr = searchParams.get(URL_PARAMS.statuses);
    const statuses = statusStr ? statusStr.split(",").filter(Boolean) : [];

    const kindStr = searchParams.get(URL_PARAMS.kinds);
    const kinds = kindStr ? kindStr.split(",").filter(Boolean) : [];

    const modeStr = searchParams.get(URL_PARAMS.modes);
    const modes = modeStr ? modeStr.split(",").filter(Boolean) : [];

    const filters: Record<string, unknown> = {};
    if (tab === "backlog") {
      if (statuses.length > 0) filters.statuses = statuses;
      if (kinds.length > 0) filters.kinds = kinds;
    } else if (tab === "captures") {
      if (statuses.length > 0) filters.statuses = statuses;
    } else if (tab === "initiatives") {
      if (statuses.length > 0) filters.statuses = statuses;
    } else if (tab === "executions") {
      if (statuses.length > 0) filters.statuses = statuses;
      if (modes.length > 0) filters.modes = modes;
    }

    dispatch({
      type: "RESTORE_FROM_URL",
      tab,
      filters,
      sort,
    });
  }, [dispatch, searchParams]);
}

/**
 * Write active tab's state to URL whenever it changes.
 */
export function useSyncToUrl(state: SidebarState): void {
  const [, setSearchParams] = useSearchParams();

  const syncToUrl = useCallback(() => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);

      // Clear all sidebar params first
      for (const key of Object.values(URL_PARAMS)) {
        next.delete(key);
      }

      const { activeTab } = state;

      // Always write the active tab if not default
      if (activeTab !== "activity") {
        next.set(URL_PARAMS.tab, activeTab);
      }

      // Write sort if non-default
      const sort = state.sorts[activeTab];
      const defaultSort = DEFAULT_SORT[activeTab];
      if (sort.field !== defaultSort.field) {
        next.set(URL_PARAMS.sortField, sort.field);
      }
      if (sort.direction !== defaultSort.direction) {
        next.set(URL_PARAMS.sortDirection, sort.direction);
      }

      // Write filters for active tab
      if (activeTab === "backlog") {
        const f = state.filters.backlog;
        const d = DEFAULT_FILTERS.backlog;
        if (f.statuses.length > 0 && f.statuses.length !== d.statuses.length) next.set(URL_PARAMS.statuses, f.statuses.join(","));
        if (f.kinds.length > 0) next.set(URL_PARAMS.kinds, f.kinds.join(","));
      } else if (activeTab === "captures") {
        const f = state.filters.captures;
        if (f.statuses.length > 0) next.set(URL_PARAMS.statuses, f.statuses.join(","));
      } else if (activeTab === "initiatives") {
        const f = state.filters.initiatives;
        if (f.statuses.length > 0) next.set(URL_PARAMS.statuses, f.statuses.join(","));
      } else if (activeTab === "executions") {
        const f = state.filters.executions;
        if (f.statuses.length > 0) next.set(URL_PARAMS.statuses, f.statuses.join(","));
        if (f.modes.length > 0) next.set(URL_PARAMS.modes, f.modes.join(","));
      }

      return next;
    }, { replace: true });
  }, [setSearchParams, state]);

  const prevState = useRef(state);
  useEffect(() => {
    if (prevState.current === state) return;
    prevState.current = state;
    syncToUrl();
  }, [state, syncToUrl]);
}
