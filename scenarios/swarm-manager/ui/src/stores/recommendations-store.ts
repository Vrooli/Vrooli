import { create } from "zustand";
import { recommendationsService } from "../services";
import type { Recommendation } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

interface RecommendationsStoreState {
  recommendations: Recommendation[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchRecommendations: (options?: { force?: boolean }) => Promise<void>;
  setRecommendations: (recommendations: Recommendation[]) => void;
  upsertRecommendation: (recommendation: Recommendation) => void;
  reset: () => void;
}

export const recommendationsStoreInitialState = {
  recommendations: [],
  status: "idle" as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: null,
};

export const useRecommendationsStore = create<RecommendationsStoreState>((set, get) => ({
  ...recommendationsStoreInitialState,

  fetchRecommendations: async ({ force = false } = {}): Promise<void> => {
    const { status, recommendations, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) {
      return;
    }

    if (!shouldRefetch({ lastFetchedAt, hasData: recommendations.length > 0, force })) {
      return;
    }

    const hasData = recommendations.length > 0;

    set({
      status: hasData ? "success" : "loading",
      isRefreshing: hasData,
      error: null,
    });

    try {
      const result = await fetchWithRetry(() => recommendationsService.list());
      set({
        recommendations: result,
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load recommendations."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setRecommendations: (recommendations: Recommendation[]): void => {
    set({ recommendations, status: "success", error: null });
  },

  upsertRecommendation: (recommendation: Recommendation): void => {
    set((state) => {
      const next = state.recommendations.some((item) => item.id === recommendation.id)
        ? state.recommendations.map((item) => (item.id === recommendation.id ? recommendation : item))
        : [...state.recommendations, recommendation];

      return { recommendations: next };
    });
  },

  reset: () => {
    set({ ...recommendationsStoreInitialState });
  },
}));
