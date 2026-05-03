/**
 * Operations Center page (`/operations`).
 *
 * Greenfield replacement for the legacy `<AgentsDropdown>` popover.
 * Composes the stats header, filter bar, and body view; wires polling
 * via `useOperationsPolling`; mirrors filter and view state onto the
 * URL query string for deep-link / reload survival.
 *
 * Phase 6 ships the by-initiative body view and the read-only filter
 * bar. P7a layers in the by-phase board view + view toggle, and P7b
 * adds bulk actions (selection, "Stop selected" / "Stop all running").
 */

import { useCallback, useEffect, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { Bot } from "lucide-react";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { OpsHeader } from "../components/operations/OpsHeader";
import { OpsFilterBar } from "../components/operations/OpsFilterBar";
import { OpsBody } from "../components/operations/OpsBody";
import { selectors } from "../consts/selectors";
import { useOperationsPolling } from "../hooks/useOperationsPolling";
import {
  DEFAULT_OPERATIONS_WINDOW_SECONDS,
  operationsStoreInitialFilters,
  useOperationsStore,
} from "../stores/operations-store";
import {
  OPERATIONS_LANES,
  OPERATIONS_VIEW_MODES,
  type OperationsFilters,
  type OperationsViewMode,
} from "../types/operations";

const ALLOWED_STATUSES = new Set([
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
]);

const ALLOWED_OWNER_TYPES = new Set([
  "initiative",
  "backlog",
  "scenario",
  "capture",
  "session",
]);

const ALLOWED_LANES = new Set(OPERATIONS_LANES as readonly string[]);
const ALLOWED_WINDOWS = new Set([
  60 * 60,
  3 * 60 * 60,
  6 * 60 * 60,
  12 * 60 * 60,
  24 * 60 * 60,
]);

function readFiltersFromUrl(searchParams: URLSearchParams): OperationsFilters {
  const status = (searchParams.get("status") ?? "").trim();
  const lane = (searchParams.get("lane") ?? "").trim();
  const ownerType = (searchParams.get("owner_type") ?? "").trim();
  const q = (searchParams.get("q") ?? "").trim();
  const windowRaw = Number(searchParams.get("window_seconds"));
  const windowSeconds = ALLOWED_WINDOWS.has(windowRaw)
    ? windowRaw
    : DEFAULT_OPERATIONS_WINDOW_SECONDS;

  return {
    windowSeconds,
    statuses: status && ALLOWED_STATUSES.has(status) ? [status] : [],
    lanes: lane && ALLOWED_LANES.has(lane) ? [lane] : [],
    modes: [],
    ownerTypes:
      ownerType && ALLOWED_OWNER_TYPES.has(ownerType) ? [ownerType] : [],
    q,
  };
}

function writeFiltersToParams(
  current: URLSearchParams,
  filters: OperationsFilters,
): URLSearchParams {
  const next = new URLSearchParams(current);
  const setOrDelete = (key: string, value: string | undefined) => {
    if (!value) next.delete(key);
    else next.set(key, value);
  };
  setOrDelete("status", filters.statuses?.[0]);
  setOrDelete("lane", filters.lanes?.[0]);
  setOrDelete("owner_type", filters.ownerTypes?.[0]);
  setOrDelete("q", filters.q && filters.q.length > 0 ? filters.q : undefined);
  if (
    filters.windowSeconds &&
    filters.windowSeconds !== DEFAULT_OPERATIONS_WINDOW_SECONDS
  ) {
    next.set("window_seconds", String(filters.windowSeconds));
  } else {
    next.delete("window_seconds");
  }
  return next;
}

function readViewFromUrl(searchParams: URLSearchParams): OperationsViewMode {
  const raw = searchParams.get("view");
  if (raw && (OPERATIONS_VIEW_MODES as readonly string[]).includes(raw)) {
    return raw as OperationsViewMode;
  }
  return "by-initiative";
}

export function OperationsCenterPage() {
  const view = useOperationsStore((state) => state.view);
  const isLoading = useOperationsStore((state) => state.isLoading);
  const isRefreshing = useOperationsStore((state) => state.isRefreshing);
  const error = useOperationsStore((state) => state.error);
  const filters = useOperationsStore((state) => state.filters);
  const viewMode = useOperationsStore((state) => state.viewMode);
  const setFilters = useOperationsStore((state) => state.setFilters);
  const resetFilters = useOperationsStore((state) => state.resetFilters);
  const setViewMode = useOperationsStore((state) => state.setViewMode);
  const refresh = useOperationsStore((state) => state.refresh);

  const [searchParams, setSearchParams] = useSearchParams();

  // URL → store on mount. Subsequent URL changes (back/forward) also
  // re-sync the store; the writer below skips when filters are deeply
  // equal so we don't loop.
  useEffect(() => {
    const next = readFiltersFromUrl(searchParams);
    const sameFilters =
      filters.windowSeconds === next.windowSeconds &&
      (filters.statuses?.[0] ?? "") === (next.statuses?.[0] ?? "") &&
      (filters.lanes?.[0] ?? "") === (next.lanes?.[0] ?? "") &&
      (filters.ownerTypes?.[0] ?? "") === (next.ownerTypes?.[0] ?? "") &&
      (filters.q ?? "") === (next.q ?? "");
    if (!sameFilters) setFilters(next);
    const nextView = readViewFromUrl(searchParams);
    if (nextView !== viewMode) setViewMode(nextView);
    // Intentional: only on URL changes, not store changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  // Store → URL when filters / view change. We compute the next params
  // from the current ones to preserve unrelated query keys.
  useEffect(() => {
    setSearchParams(
      (prev) => {
        const updated = writeFiltersToParams(prev, filters);
        if (viewMode === "by-initiative") updated.delete("view");
        else updated.set("view", viewMode);
        return updated;
      },
      { replace: true },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters, viewMode]);

  useOperationsPolling();

  const handleManualRefresh = useCallback(() => {
    void refresh({ force: true });
  }, [refresh]);

  const handleResetFilters = useCallback(() => {
    resetFilters();
    setSearchParams(
      (prev) => {
        const updated = writeFiltersToParams(prev, operationsStoreInitialFilters);
        return updated;
      },
      { replace: true },
    );
  }, [resetFilters, setSearchParams]);

  const isEmpty = useMemo(() => {
    if (!view) return false;
    if (view.activities.length > 0) return false;
    if (view.recentlyFinished.length > 0) return false;
    if (view.queue.depth > 0) return false;
    return true;
  }, [view]);

  const showFullPageLoading = isLoading && view === null;

  return (
    <div
      className="flex min-h-screen flex-col gap-4 bg-slate-950 px-4 py-4 text-slate-50 md:px-6 md:py-6"
      data-testid={selectors.operationsCenter.page}
    >
      <OpsHeader
        view={view}
        isRefreshing={isRefreshing || isLoading}
        onRefresh={handleManualRefresh}
        windowSeconds={filters.windowSeconds ?? DEFAULT_OPERATIONS_WINDOW_SECONDS}
      />
      <OpsFilterBar
        filters={filters}
        onFiltersChange={setFilters}
        onReset={handleResetFilters}
      />

      {error && view === null && (
        <div data-testid={selectors.operationsCenter.errorState}>
          <ErrorState error={error} onRetry={handleManualRefresh} />
        </div>
      )}

      {showFullPageLoading && <PageLoadingState label="Loading operations…" />}

      {view !== null && !isEmpty && (
        <OpsBody
          view={viewMode}
          onViewChange={setViewMode}
          activities={view.activities}
          recentlyFinished={view.recentlyFinished}
          enableByPhaseView={true}
        />
      )}

      {view !== null && isEmpty && (
        <div
          className="flex flex-col items-center justify-center gap-3 rounded-xl border border-white/5 bg-slate-900/40 px-6 py-12 text-center"
          data-testid={selectors.operationsCenter.emptyState}
        >
          <Bot className="h-10 w-10 text-slate-600" aria-hidden />
          <h2 className="text-base font-medium text-slate-200">
            No agentic activity in window
          </h2>
          <p className="max-w-sm text-sm text-slate-400">
            Nothing is running, queued, or finished within the selected time
            window. Spawn an agent or widen the window to see more.
          </p>
        </div>
      )}
    </div>
  );
}
