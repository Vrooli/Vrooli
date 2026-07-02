/**
 * usePlanUrlState — bidirectional URL ⇄ store sync for the board's
 * filters, group-by mode, show-snoozed flag, and Done window, mirroring
 * the Operations Center's shareable-link behavior on /graph/plan.
 */

import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useOperationsStore } from "../../../stores/operations-store";
import type { OperationsFilters, OperationsViewMode } from "../../../types/operations";
import { usePlanDataStore } from "../stores/plan-data-store";
import {
  hasActiveFilters,
  readPlanStateFromUrl,
  writePlanStateToParams,
  type PlanBoardUrlState,
} from "../lib/plan-url-state";

function filtersEqual(a: OperationsFilters, b: OperationsFilters): boolean {
  return (
    a.windowSeconds === b.windowSeconds &&
    (a.q ?? "") === (b.q ?? "") &&
    (a.statuses?.[0] ?? "") === (b.statuses?.[0] ?? "") &&
    (a.lanes?.[0] ?? "") === (b.lanes?.[0] ?? "") &&
    (a.ownerTypes?.[0] ?? "") === (b.ownerTypes?.[0] ?? "")
  );
}

export interface PlanUrlStateResult {
  filters: OperationsFilters;
  viewMode: OperationsViewMode;
  showSnoozed: boolean;
  hasFilters: boolean;
  setFilters: (next: Partial<OperationsFilters>) => void;
  setViewMode: (mode: OperationsViewMode) => void;
  setShowSnoozed: (show: boolean) => void;
  resetFilters: () => void;
}

export function usePlanUrlState(): PlanUrlStateResult {
  const [searchParams, setSearchParams] = useSearchParams();

  const filters = useOperationsStore((s) => s.filters);
  const setFilters = useOperationsStore((s) => s.setFilters);
  const viewMode = useOperationsStore((s) => s.viewMode);
  const setViewMode = useOperationsStore((s) => s.setViewMode);

  const showSnoozed = usePlanDataStore((s) => s.showSnoozed);
  const setShowSnoozed = usePlanDataStore((s) => s.setShowSnoozed);
  const windowSeconds = usePlanDataStore((s) => s.windowSeconds);
  const setWindowSeconds = usePlanDataStore((s) => s.setWindowSeconds);

  // URL → stores. Deep-equal guard prevents ping-pong with the write effect.
  useEffect(() => {
    const urlState = readPlanStateFromUrl(searchParams);
    if (!filtersEqual(urlState.filters, {
      ...filters,
      windowSeconds,
    })) {
      setFilters(urlState.filters);
      setWindowSeconds(urlState.filters.windowSeconds ?? windowSeconds);
    }
    if (urlState.viewMode !== viewMode) {
      setViewMode(urlState.viewMode);
    }
    if (urlState.showSnoozed !== showSnoozed) {
      setShowSnoozed(urlState.showSnoozed);
    }
    // Intentionally only re-runs on URL changes — store-driven changes
    // flow through the write effect below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  // Stores → URL (replace, so filter tweaks don't pollute history).
  const state: PlanBoardUrlState = {
    filters: { ...filters, windowSeconds },
    viewMode,
    showSnoozed,
  };
  const stateRef = useRef(state);
  stateRef.current = state;
  useEffect(() => {
    setSearchParams((prev) => writePlanStateToParams(prev, stateRef.current), { replace: true });
  }, [filters, viewMode, showSnoozed, windowSeconds, setSearchParams]);

  return {
    filters,
    viewMode,
    showSnoozed,
    hasFilters: hasActiveFilters(state),
    setFilters,
    setViewMode,
    setShowSnoozed,
    resetFilters: () => {
      setFilters({ statuses: [], lanes: [], ownerTypes: [], modes: [], q: "" });
      setViewMode("by-initiative");
      setShowSnoozed(false);
    },
  };
}
