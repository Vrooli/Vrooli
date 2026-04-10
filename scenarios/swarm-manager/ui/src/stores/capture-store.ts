import { create } from "zustand";
import { captureService } from "../services/capture-service";
import type { Capture } from "../types";
import { clearStorage, fetchWithRetry, loadFromStorage, saveToStorage, shouldRefetch, type LoadStatus, type StorePersistConfig } from "./store-utils";

const PERSIST_CONFIG: StorePersistConfig = {
  key: "swarm-manager.captures.v1",
  version: 1,
  maxItems: 100,
};

interface CaptureStoreState {
  captures: Capture[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchCaptures: (options?: { force?: boolean }) => Promise<void>;
  addCapture: (capture: Capture) => void;
  updateCapture: (id: string, updates: Partial<Capture>) => void;
  removeCapture: (id: string) => void;
  reset: () => void;
}

const hydrated = loadFromStorage<Capture[]>(PERSIST_CONFIG, []);

export const captureStoreInitialState = {
  captures: hydrated.data,
  status: (hydrated.data.length > 0 ? "success" : "idle") as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: hydrated.lastFetchedAt,
};

export const useCaptureStore = create<CaptureStoreState>((set, get) => ({
  ...captureStoreInitialState,

  fetchCaptures: async ({ force = false } = {}): Promise<void> => {
    const { status, captures, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) return;

    if (!shouldRefetch({ lastFetchedAt, hasData: captures.length > 0, force })) return;

    const hasData = captures.length > 0;
    set({ status: hasData ? "success" : "loading", isRefreshing: hasData, error: null });

    try {
      const result = await fetchWithRetry(() => captureService.list());
      const now = Date.now();
      set({
        captures: result,
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: now,
      });
      saveToStorage(PERSIST_CONFIG, result, now);
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load captures."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  addCapture: (capture: Capture): void => {
    set((state) => {
      const next = [capture, ...state.captures];
      saveToStorage(PERSIST_CONFIG, next, state.lastFetchedAt ?? Date.now());
      return { captures: next };
    });
  },

  updateCapture: (id: string, updates: Partial<Capture>): void => {
    set((state) => {
      const next = state.captures.map((c) => (c.id === id ? { ...c, ...updates } : c));
      saveToStorage(PERSIST_CONFIG, next, state.lastFetchedAt ?? Date.now());
      return { captures: next };
    });
  },

  removeCapture: (id: string): void => {
    set((state) => {
      const next = state.captures.filter((c) => c.id !== id);
      saveToStorage(PERSIST_CONFIG, next, state.lastFetchedAt ?? Date.now());
      return { captures: next };
    });
  },

  reset: () => {
    clearStorage(PERSIST_CONFIG.key);
    set({
      captures: [],
      status: "idle" as LoadStatus,
      error: null,
      isRefreshing: false,
      lastFetchedAt: null,
    });
  },
}));
