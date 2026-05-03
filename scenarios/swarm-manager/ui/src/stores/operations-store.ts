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
  selection: ReadonlySet<string>;

  refresh(options?: { force?: boolean }): Promise<void>;
  setFilters(next: Partial<OperationsFilters>): void;
  resetFilters(): void;
  setViewMode(mode: OperationsViewMode): void;

  toggleSelection(runId: string): void;
  setSelection(ids: readonly string[]): void;
  clearSelection(): void;

  reset(): void;
}

const initialState = {
  view: null as OperationsView | null,
  isLoading: false,
  isRefreshing: false,
  error: null as Error | null,
  lastRefreshedAt: null as number | null,
  filters: operationsStoreInitialFilters,
  viewMode: "by-initiative" as OperationsViewMode,
  selection: new Set<string>() as ReadonlySet<string>,
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

  reset: (): void => {
    set(initialState);
  },
}));

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
