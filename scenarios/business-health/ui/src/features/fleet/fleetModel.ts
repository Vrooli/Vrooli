/**
 * Pure sort/filter model for the worst-first fleet table. Kept free of React
 * and proto-schema construction so the ranking and filtering rules are unit
 * testable in isolation from the component and the connect client.
 */
import type { FleetScenarioEntry } from "@vrooli/proto-types/business-health/v1/fleet/fleet_pb";

/** Sortable table columns, keyed to the `strings.fleet.column.*` headers. */
export type SortColumn =
  | "scenario"
  | "status"
  | "errors"
  | "warnings"
  | "autofix"
  | "orphans"
  | "unproven"
  | "template"
  | "debt";

export type SortDirection = "asc" | "desc";

export interface SortState {
  readonly column: SortColumn;
  readonly direction: SortDirection;
}

export interface FleetFilters {
  /** Case-insensitive substring match against the scenario slug. */
  readonly text: string;
  /** Keep only entries still carrying a starter-template registry. */
  readonly starter: boolean;
  /** Keep only entries whose template version lags the current contract. */
  readonly laggard: boolean;
  /** Keep only entries with at least one unproven claim. */
  readonly unproven: boolean;
}

/** Worst-first: highest debt score at the top on first render. */
export const DEFAULT_SORT: SortState = { column: "debt", direction: "desc" };

export const EMPTY_FILTERS: FleetFilters = {
  text: "",
  starter: false,
  laggard: false,
  unproven: false,
};

/**
 * Alphabetic columns default to ascending; every numeric/severity column
 * defaults to descending so the first click surfaces the worst offenders.
 */
export const defaultDirectionFor = (column: SortColumn): SortDirection =>
  column === "scenario" || column === "template" ? "asc" : "desc";

/**
 * Compute the next sort state when a header is clicked: flip direction when the
 * same column is re-selected, otherwise switch columns at that column's natural
 * default direction.
 */
export const toggleSort = (current: SortState, column: SortColumn): SortState =>
  current.column === column
    ? { column, direction: current.direction === "asc" ? "desc" : "asc" }
    : { column, direction: defaultDirectionFor(column) };

const sortValue = (entry: FleetScenarioEntry, column: SortColumn): string | number => {
  switch (column) {
    case "scenario":
      return entry.scenario;
    case "template":
      return entry.templateVersion;
    case "status":
      // Passing sorts high so "asc" puts failing scenarios first.
      return entry.passed ? 1 : 0;
    case "errors":
      return entry.errorCount;
    case "warnings":
      return entry.warningCount;
    case "autofix":
      return entry.autofixableCount;
    case "orphans":
      return entry.orphanedTargets;
    case "unproven":
      return entry.unprovenClaims;
    case "debt":
      return entry.debtScore;
  }
};

/** Keep only the entries that satisfy every active filter (AND semantics). */
export const filterEntries = (
  entries: readonly FleetScenarioEntry[],
  filters: FleetFilters,
): FleetScenarioEntry[] => {
  const needle = filters.text.trim().toLowerCase();
  return entries.filter((entry) => {
    if (needle && !entry.scenario.toLowerCase().includes(needle)) return false;
    if (filters.starter && !entry.starterRegistry) return false;
    if (filters.laggard && !entry.templateLaggard) return false;
    if (filters.unproven && entry.unprovenClaims <= 0) return false;
    return true;
  });
};

/**
 * Return a sorted copy of the entries. Ties break on the scenario slug so the
 * order is deterministic regardless of the source ordering.
 */
export const sortEntries = (
  entries: readonly FleetScenarioEntry[],
  sort: SortState,
): FleetScenarioEntry[] => {
  const factor = sort.direction === "asc" ? 1 : -1;
  return [...entries].sort((a, b) => {
    const av = sortValue(a, sort.column);
    const bv = sortValue(b, sort.column);
    let cmp: number;
    if (typeof av === "string" && typeof bv === "string") {
      cmp = av.localeCompare(bv);
    } else {
      cmp = Number(av) - Number(bv);
    }
    if (cmp !== 0) return cmp * factor;
    return a.scenario.localeCompare(b.scenario);
  });
};
