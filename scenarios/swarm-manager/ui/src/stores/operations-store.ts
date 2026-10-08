/**
 * Operations Center store.
 *
 * The Operations Center is a single-page surface that polls the
 * `/operations` aggregate. State splits cleanly into three slices:
 *
 *   - `view` (data) — the latest `OperationsView` plus loading + error
 *     state. `refresh()` is the only mutator; it serializes concurrent
 *     polls so the 4s tick never races a manual click.
 *
 *   - `filters` — the active query parameters that get echoed onto the
 *     URL via the page-level URL-state hook. Filter mutators do not
 *     auto-refresh; the page subscribes to filter changes and triggers
 *     `refresh()` itself, keeping the store free of side effects beyond
 *     the network call.
 *
 *   - `viewMode` and `selection` — purely UI state. Selection lands here
 *     so it survives view-mode toggles in P7a; bulk-action wiring lives
 *     on top of it in P7b.
 *
 * Service injection (`setOperationsStoreService`) follows the same
 * pattern as `agent-session-store` so tests can swap a fake without
 * mocking the module graph.
 */

import { create } from "zustand";
import {
  operationsService,
  type IOperationsService,
} from "../services/operations-service";
import type {
  BulkStopFilter,
  BulkStopResponse,
  OperationsFilters,
  OperationsView,
  OperationsViewMode,
} from "../types/operations";

let service: IOperationsService = operationsService;

export function setOperationsStoreService(nextService: IOperationsService): void {
  service = nextService;
}

export function resetOperationsStoreService(): void {
  service = operationsService;
}

export const DEFAULT_OPERATIONS_WINDOW_SECONDS = 3 * 60 * 60;

export const operationsStoreInitialFilters: OperationsFilters = {
  windowSeconds: DEFAULT_OPERATIONS_WINDOW_SECONDS,
  statuses: [],
  lanes: [],
  modes: [],
  ownerTypes: [],
  q: "",
};

interface OperationsStoreState {
  view: OperationsView | null;
  isLoading: boolean;
  isRefreshing: boolean;
  error: Error | null;
  lastRefreshedAt: number | null;

  filters: OperationsFilters;
  viewMode: OperationsViewMode;
  /**
   * When false (default) the active-row checkboxes and the sticky
   * bulk-action bar are hidden — the page reads as a clean info surface.
   * When true the operator has explicitly opted into multi-select; rows
   * grow a leading checkbox and `OpsBulkActions` becomes visible.
   *
   * Turning selection mode OFF clears `selection` so toggling it back ON
   * starts from a known-empty state.
   */
  selectionMode: boolean;
  selection: ReadonlySet<string>;
  /**
   * Run IDs the operator has issued a bulk-stop call for, but for which
   * the next `refresh` has not yet returned a terminal status. The UI
   * uses this to render a disabled-row / spinner state per row without
   * waiting on the next polling tick.
   */
  stoppingRunIds: ReadonlySet<string>;
  /** Whether a bulk-stop call is currently in flight. */
  isBulkStopping: boolean;
  /**
   * Result of the most recent bulk-stop call. Surfaced in the UI as a
   * toast / outcome list. Null until the operator triggers a stop.
   */
  lastBulkStopResult: BulkStopResponse | null;

  refresh(options?: { force?: boolean }): Promise<void>;
  setFilters(next: Partial<OperationsFilters>): void;
  resetFilters(): void;
  setViewMode(mode: OperationsViewMode): void;

  /** Set selection mode explicitly. Turning OFF clears selection. */
  setSelectionMode(on: boolean): void;
  /** Flip selection mode. Same clearing semantics as `setSelectionMode(false)`. */
  toggleSelectionMode(): void;
  toggleSelection(runId: string): void;
  setSelection(ids: readonly string[]): void;
  clearSelection(): void;

  /**
   * Stop the runs whose IDs are currently in `selection`. Optimistically
   * marks each selected ID as stopping, calls the bulk endpoint, then
   * re-fetches the operations view. Selection is cleared on the
   * successful path so the operator's next interaction starts fresh.
   *
   * Resolves to the `BulkStopResponse` so the caller can render a per-run
   * outcome list (the same payload is also stored on `lastBulkStopResult`).
   */
  bulkStopSelected(): Promise<BulkStopResponse>;
  /**
   * Stop every currently-running run. The filter is sent to the backend
   * which resolves the live target set against the activity ledger, so
   * runs that started after the operator's last refresh are still
   * captured.
   */
  bulkStopAll(filter?: BulkStopFilter): Promise<BulkStopResponse>;

  reset(): void;
}

const initialState = {
  view: null as OperationsView | null,
  isLoading: false,
  isRefreshing: false,
  error: null as Error | null,
  lastRefreshedAt: null as number | null,
  filters: operationsStoreInitialFilters,
  viewMode: "by-milestone" as OperationsViewMode,
  selectionMode: false,
  selection: new Set<string>() as ReadonlySet<string>,
  stoppingRunIds: new Set<string>() as ReadonlySet<string>,
  isBulkStopping: false,
  lastBulkStopResult: null as BulkStopResponse | null,
};

export const useOperationsStore = create<OperationsStoreState>((set, get) => ({
  ...initialState,

  refresh: async (options): Promise<void> => {
    const { isRefreshing, isLoading, view } = get();
    if (isRefreshing || isLoading) return;
    if (!options?.force && view === null && get().error) {
      // First load failed and no force flag — let the operator retry
      // explicitly rather than thrashing on the polling tick.
      return;
    }

    if (view === null) {
      set({ isLoading: true, error: null });
    } else {
      set({ isRefreshing: true, error: null });
    }

    try {
      const next = await service.fetchOperations(get().filters);
      set({
        view: next,
        isLoading: false,
        isRefreshing: false,
        error: null,
        lastRefreshedAt: Date.now(),
      });
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      set({
        isLoading: false,
        isRefreshing: false,
        error,
      });
    }
  },

  setFilters: (next): void => {
    set((state) => ({
      filters: {
        ...state.filters,
        ...next,
      },
    }));
  },

  resetFilters: (): void => {
    set({ filters: operationsStoreInitialFilters });
  },

  setViewMode: (mode): void => {
    set({ viewMode: mode });
  },

  setSelectionMode: (on): void => {
    set((state) => {
      if (state.selectionMode === on) return {};
      // Turning OFF always clears any pending selection so toggling back ON
      // starts fresh; turning ON preserves whatever the operator had.
      if (!on) {
        return { selectionMode: false, selection: new Set<string>() };
      }
      return { selectionMode: true };
    });
  },

  toggleSelectionMode: (): void => {
    set((state) => {
      if (state.selectionMode) {
        return { selectionMode: false, selection: new Set<string>() };
      }
      return { selectionMode: true };
    });
  },

  toggleSelection: (runId): void => {
    if (!runId) return;
    set((state) => {
      const next = new Set(state.selection);
      if (next.has(runId)) {
        next.delete(runId);
      } else {
        next.add(runId);
      }
      return { selection: next };
    });
  },

  setSelection: (ids): void => {
    set({ selection: new Set(ids) });
  },

  clearSelection: (): void => {
    set({ selection: new Set() });
  },

  bulkStopSelected: async (): Promise<BulkStopResponse> => {
    const ids = Array.from(get().selection);
    if (ids.length === 0) {
      const empty: BulkStopResponse = { outcomes: [], total: 0, stopped: 0, failed: 0 };
      set({ lastBulkStopResult: empty });
      return empty;
    }
    return executeBulkStop(set, get, () => service.bulkStop({ runIds: ids }), ids, {
      clearSelectionOnDone: true,
    });
  },

  bulkStopAll: async (filter?: BulkStopFilter): Promise<BulkStopResponse> => {
    // Stop-all targets a server-resolved set, so the optimistic
    // `stoppingRunIds` snapshot is whatever rows the UI currently shows
    // matching the filter. The next `refresh` reconciles ledger truth.
    const visibleIds = collectActiveRunIds(get().view, filter);
    return executeBulkStop(
      set,
      get,
      () => service.bulkStop({ filter: filter ?? {} }),
      visibleIds,
      { clearSelectionOnDone: false },
    );
  },

  reset: (): void => {
    set(initialState);
  },
}));

type SetState = (
  partial:
    | Partial<OperationsStoreState>
    | ((state: OperationsStoreState) => Partial<OperationsStoreState>),
) => void;

type GetState = () => OperationsStoreState;

interface BulkStopExecuteOptions {
  clearSelectionOnDone: boolean;
}

/**
 * Run a bulk-stop call with a single optimistic snapshot. Two reasons we
 * encapsulate the recipe rather than inline it in each action:
 *
 *  1. The optimistic-`stoppingRunIds` and `isBulkStopping` flags must be
 *     cleared together on every exit path (success, partial-failure, error)
 *     so the UI never gets stuck showing a spinner.
 *  2. Both actions follow the same "post then refresh" pattern; sharing
 *     the body keeps them in lock-step if the contract changes.
 */
async function executeBulkStop(
  set: SetState,
  get: GetState,
  call: () => Promise<BulkStopResponse>,
  optimisticIds: readonly string[],
  options: BulkStopExecuteOptions,
): Promise<BulkStopResponse> {
  if (get().isBulkStopping) {
    // Re-entrant call — return the last known result so UI never sees a
    // pending promise that overlaps the previous one. The button is
    // disabled in `isBulkStopping` state, so this guard is defensive.
    return get().lastBulkStopResult ?? { outcomes: [], total: 0, stopped: 0, failed: 0 };
  }

  set((state) => ({
    isBulkStopping: true,
    stoppingRunIds: new Set([...state.stoppingRunIds, ...optimisticIds]),
  }));

  try {
    const result = await call();
    set((state) => {
      const nextStopping = new Set(state.stoppingRunIds);
      // Clear optimistic state for the runs the call addressed; per-run
      // failures still get cleared so the UI can show the error toast and
      // the next refresh tick will pick up the unchanged status.
      for (const id of optimisticIds) {
        nextStopping.delete(id);
      }
      return {
        isBulkStopping: false,
        stoppingRunIds: nextStopping,
        lastBulkStopResult: result,
        ...(options.clearSelectionOnDone ? { selection: new Set<string>() } : {}),
      };
    });
    // Fire-and-forget refresh so the operator sees ledger truth as soon
    // as the manager has cancelled. The store's refresh is itself
    // serialized so a concurrent polling tick is a free no-op.
    void get().refresh({ force: true });
    return result;
  } catch (err) {
    set((state) => {
      const nextStopping = new Set(state.stoppingRunIds);
      for (const id of optimisticIds) {
        nextStopping.delete(id);
      }
      return {
        isBulkStopping: false,
        stoppingRunIds: nextStopping,
        error: err instanceof Error ? err : new Error(String(err)),
      };
    });
    throw err;
  }
}

/**
 * Collect the run IDs from `view` that are currently active and would be
 * targeted by the given filter. Used for the optimistic `stoppingRunIds`
 * snapshot when the operator hits "Stop all". The backend re-resolves
 * against the live ledger, so this is purely a UI hint — drift between
 * snapshot and final outcome is reconciled by the post-call refresh.
 */
function collectActiveRunIds(
  view: OperationsView | null,
  filter: BulkStopFilter | undefined,
): string[] {
  if (!view) return [];
  const ids: string[] = [];
  for (const row of view.activities) {
    if (!row.runId) continue;
    if (filter?.lane && row.lane !== filter.lane) continue;
    if (filter?.status && row.status !== filter.status) continue;
    ids.push(row.runId);
  }
  return ids;
}

/**
 * Convenience selector: total count of activities (active + queued).
 * Used by the trigger button label in P8.
 */
export function selectActiveCount(state: OperationsStoreState): number {
  return state.view?.activities.length ?? 0;
}

/**
 * Convenience selector: returns `true` if any activity is currently
 * running (status active). Useful for gating polling in environments
 * where idleness should pause network traffic.
 */
export function selectHasActiveActivity(state: OperationsStoreState): boolean {
  if (!state.view) return false;
  return state.view.activities.length > 0 || state.view.queue.depth > 0;
}
