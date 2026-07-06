/**
 * Plan data store — board projection state for the Plan lens.
 *
 * Patterned on graph-data-store but single-projection: one board, one
 * in-flight request, staleness gating, and a service seam for tests.
 * WebSocket invalidation reaches this store via graph-data-store's
 * fetchGraph("plan") delegation (see that file), so the board refreshes
 * on any "plan" lens invalidation without its own socket.
 */

import { create } from "zustand";
import type { IPlanService } from "../../../services/plan-service";
import { planService as defaultPlanService } from "../../../services/plan-service";
import type { PlanBoardData } from "../types";

const PLAN_SNAPSHOT_STALE_MS = 30_000;

/** Default Done-column window (24h), mirroring the server default. */
export const DEFAULT_PLAN_WINDOW_SECONDS = 24 * 60 * 60;

// ---------------------------------------------------------------------------
// Service + request seams (module-level, test-resettable)
// ---------------------------------------------------------------------------

let activePlanService: IPlanService = defaultPlanService;

export function setPlanStoreService(service: IPlanService): void {
  activePlanService = service;
}

export function resetPlanStoreService(): void {
  activePlanService = defaultPlanService;
}

let requestSequence = 0;
let abortController: AbortController | null = null;
let inFlight: Promise<void> | null = null;

export function resetPlanRequestState(): void {
  abortController?.abort();
  abortController = null;
  inFlight = null;
  requestSequence = 0;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export interface FetchBoardOptions {
  /** Bypass the staleness gate. */
  force?: boolean;
  /** Refresh without toggling the loading flag (background refetch). */
  silent?: boolean;
}

export interface PlanDataState {
  board: PlanBoardData | null;
  loading: boolean;
  error: string | null;
  fetchedAtMs: number | null;
  windowSeconds: number;
  /** Presentation flag: render snoozed cards dimmed instead of hiding them. */
  showSnoozed: boolean;
  /** When set, the board is scoped to this goal's closure (server-side). */
  goal: string;
  fetchBoard: (options?: FetchBoardOptions) => Promise<void>;
  setWindowSeconds: (seconds: number) => void;
  setShowSnoozed: (show: boolean) => void;
  setGoal: (goal: string) => void;
}

export function createPlanDataInitialState() {
  return {
    board: null as PlanBoardData | null,
    loading: false,
    error: null as string | null,
    fetchedAtMs: null as number | null,
    windowSeconds: DEFAULT_PLAN_WINDOW_SECONDS,
    showSnoozed: false,
    goal: "",
  };
}

export const usePlanDataStore = create<PlanDataState>((set, get) => ({
  ...createPlanDataInitialState(),

  setWindowSeconds: (seconds) => {
    if (seconds === get().windowSeconds) return;
    set({ windowSeconds: seconds });
    void get().fetchBoard({ force: true });
  },

  setShowSnoozed: (show) => set({ showSnoozed: show }),

  setGoal: (goal) => {
    if (goal === get().goal) return;
    set({ goal });
    void get().fetchBoard({ force: true });
  },

  fetchBoard: async (options) => {
    const { fetchedAtMs, board } = get();
    const fresh =
      board !== null &&
      fetchedAtMs !== null &&
      Date.now() - fetchedAtMs < PLAN_SNAPSHOT_STALE_MS;
    if (fresh && !options?.force) {
      return;
    }
    // Dedupe concurrent fetches; a forced fetch aborts and supersedes the
    // in-flight one instead.
    if (inFlight && !options?.force) {
      return inFlight;
    }

    abortController?.abort();
    const controller = new AbortController();
    abortController = controller;
    const sequence = ++requestSequence;

    if (!options?.silent || get().board === null) {
      set({ loading: true, error: null });
    }

    const request = (async () => {
      try {
        const data = await activePlanService.getBoard({
          signal: controller.signal,
          windowSeconds: get().windowSeconds,
          goal: get().goal || undefined,
        });
        if (sequence !== requestSequence) return;
        set({ board: data, loading: false, error: null, fetchedAtMs: Date.now() });
      } catch (error) {
        if (sequence !== requestSequence) return;
        if (error instanceof DOMException && error.name === "AbortError") return;
        set({
          loading: false,
          error: error instanceof Error ? error.message : "Failed to load plan board",
        });
      }
    })().finally(() => {
      // Only clear if no newer request has replaced this one.
      if (sequence === requestSequence) {
        inFlight = null;
      }
    });
    inFlight = request;
    return request;
  },
}));
