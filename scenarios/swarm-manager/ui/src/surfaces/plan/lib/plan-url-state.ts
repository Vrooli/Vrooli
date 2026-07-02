/**
 * URL query-param persistence for the Plan board's filters — the same
 * shareable-link contract the Operations Center used: `status`, `lane`,
 * `owner_type`, `q`, `window_seconds`, `view`, plus the board-only
 * `show_snoozed`. Values are validated against allow-sets; invalid values
 * fall back to defaults rather than erroring.
 */

import { DEFAULT_PLAN_WINDOW_SECONDS } from "../stores/plan-data-store";
import {
  OPERATIONS_LANES,
  OPERATIONS_VIEW_MODES,
  type OperationsFilters,
  type OperationsViewMode,
} from "../../../types/operations";

export const ALLOWED_STATUSES = new Set([
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
]);

export const ALLOWED_OWNER_TYPES = new Set([
  "initiative",
  "backlog",
  "scenario",
  "capture",
  "session",
]);

export const ALLOWED_LANES = new Set(OPERATIONS_LANES as readonly string[]);

export const ALLOWED_WINDOWS = new Set([
  60 * 60,
  3 * 60 * 60,
  6 * 60 * 60,
  12 * 60 * 60,
  24 * 60 * 60,
]);

export interface PlanBoardUrlState {
  filters: OperationsFilters;
  viewMode: OperationsViewMode;
  showSnoozed: boolean;
}

export function readPlanStateFromUrl(searchParams: URLSearchParams): PlanBoardUrlState {
  const status = (searchParams.get("status") ?? "").trim();
  const lane = (searchParams.get("lane") ?? "").trim();
  const ownerType = (searchParams.get("owner_type") ?? "").trim();
  const q = (searchParams.get("q") ?? "").trim();
  const windowRaw = Number(searchParams.get("window_seconds"));
  const windowSeconds = ALLOWED_WINDOWS.has(windowRaw)
    ? windowRaw
    : DEFAULT_PLAN_WINDOW_SECONDS;

  const viewRaw = searchParams.get("view");
  const viewMode: OperationsViewMode =
    viewRaw && (OPERATIONS_VIEW_MODES as readonly string[]).includes(viewRaw)
      ? (viewRaw as OperationsViewMode)
      : "by-initiative";

  return {
    filters: {
      windowSeconds,
      statuses: status && ALLOWED_STATUSES.has(status) ? [status] : [],
      lanes: lane && ALLOWED_LANES.has(lane) ? [lane] : [],
      modes: [],
      ownerTypes: ownerType && ALLOWED_OWNER_TYPES.has(ownerType) ? [ownerType] : [],
      q,
    },
    viewMode,
    showSnoozed: searchParams.get("show_snoozed") === "1",
  };
}

export function writePlanStateToParams(
  current: URLSearchParams,
  state: PlanBoardUrlState,
): URLSearchParams {
  const next = new URLSearchParams(current);
  const setOrDelete = (key: string, value: string | undefined) => {
    if (!value) next.delete(key);
    else next.set(key, value);
  };
  setOrDelete("status", state.filters.statuses?.[0]);
  setOrDelete("lane", state.filters.lanes?.[0]);
  setOrDelete("owner_type", state.filters.ownerTypes?.[0]);
  setOrDelete("q", state.filters.q && state.filters.q.length > 0 ? state.filters.q : undefined);
  if (
    state.filters.windowSeconds &&
    state.filters.windowSeconds !== DEFAULT_PLAN_WINDOW_SECONDS
  ) {
    next.set("window_seconds", String(state.filters.windowSeconds));
  } else {
    next.delete("window_seconds");
  }
  if (state.viewMode !== "by-initiative") {
    next.set("view", state.viewMode);
  } else {
    next.delete("view");
  }
  if (state.showSnoozed) {
    next.set("show_snoozed", "1");
  } else {
    next.delete("show_snoozed");
  }
  return next;
}

/** Whether any user-facing filter differs from the defaults. */
export function hasActiveFilters(state: PlanBoardUrlState): boolean {
  const f = state.filters;
  return (
    (f.statuses?.length ?? 0) > 0 ||
    (f.lanes?.length ?? 0) > 0 ||
    (f.ownerTypes?.length ?? 0) > 0 ||
    (f.q ?? "").length > 0 ||
    state.showSnoozed
  );
}
