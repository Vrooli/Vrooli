import { create } from "zustand";
import { captureService } from "../services/capture-service";
import type { Capture } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

interface CaptureStoreState {
  captures: Capture[];
  status: LoadStatus;
  error: Error | null;
  lastFetchedAt: number | null;
  fetchCaptures: (options?: { force?: boolean }) => Promise<void>;
  addCapture: (capture: Capture) => void;
  updateCapture: (id: string, updates: Partial<Capture>) => void;
  removeCapture: (id: string) => void;
  reset: () => void;
}

export const captureStoreInitialState = {
  captures: [],
  status: "idle" as LoadStatus,
  error: null,
  lastFetchedAt: null,
};

export const useCaptureStore = create<CaptureStoreState>((set, get) => ({
  ...captureStoreInitialState,

  fetchCaptures: async ({ force = false } = {}): Promise<void> => {
    const { status, captures, lastFetchedAt } = get();

    if (status === "loading") return;

    if (!shouldRefetch({ lastFetchedAt, hasData: captures.length > 0, force })) return;

    const hasData = captures.length > 0;
    set({ status: hasData ? "success" : "loading", error: null });

    try {
      const result = await fetchWithRetry(() => captureService.list());
      set({
        captures: result,
        status: "success",
        error: null,
        lastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load captures."),
        status: hasData ? "success" : "error",
      });
    }
  },

  addCapture: (capture: Capture): void => {
    set((state) => ({
      captures: [capture, ...state.captures],
    }));
  },

  updateCapture: (id: string, updates: Partial<Capture>): void => {
    set((state) => ({
      captures: state.captures.map((c) => (c.id === id ? { ...c, ...updates } : c)),
    }));
  },

  removeCapture: (id: string): void => {
    set((state) => ({
      captures: state.captures.filter((c) => c.id !== id),
    }));
  },

  reset: () => {
    set({ ...captureStoreInitialState });
  },
}));
