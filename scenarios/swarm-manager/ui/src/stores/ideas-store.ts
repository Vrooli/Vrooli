import { create } from "zustand";
import { ideasService } from "../services";
import type { Idea } from "../types";
import { fetchWithRetry, shouldRefetch, type LoadStatus } from "./store-utils";

interface EnsureIdeaOrderOptions {
  existing?: Idea[];
}

const sortIdeas = (items: Idea[], _options?: EnsureIdeaOrderOptions): Idea[] => {
  return [...items].sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    return new Date(b.updated).getTime() - new Date(a.updated).getTime();
  });
};

interface IdeasStoreState {
  ideas: Idea[];
  status: LoadStatus;
  error: Error | null;
  isRefreshing: boolean;
  lastFetchedAt: number | null;
  fetchIdeas: (options?: { force?: boolean }) => Promise<void>;
  setIdeas: (ideas: Idea[]) => void;
  upsertIdea: (idea: Idea) => void;
  removeIdea: (name: string) => void;
  reset: () => void;
}

export const ideasStoreInitialState = {
  ideas: [],
  status: "idle" as LoadStatus,
  error: null,
  isRefreshing: false,
  lastFetchedAt: null,
};

export const useIdeasStore = create<IdeasStoreState>((set, get) => ({
  ...ideasStoreInitialState,

  fetchIdeas: async ({ force = false } = {}): Promise<void> => {
    const { status, ideas, lastFetchedAt, isRefreshing } = get();

    if (status === "loading" || isRefreshing) {
      return;
    }

    if (!shouldRefetch({ lastFetchedAt, hasData: ideas.length > 0, force })) {
      return;
    }

    const hasData = ideas.length > 0;

    set({
      status: hasData ? "success" : "loading",
      isRefreshing: hasData,
      error: null,
    });

    try {
      const result = await fetchWithRetry(() => ideasService.list());
      set({
        ideas: sortIdeas(result),
        status: "success",
        error: null,
        isRefreshing: false,
        lastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        error: error instanceof Error ? error : new Error("Unable to load ideas."),
        status: hasData ? "success" : "error",
        isRefreshing: false,
      });
    }
  },

  setIdeas: (ideas: Idea[]): void => {
    set({ ideas: sortIdeas(ideas), status: "success", error: null });
  },

  upsertIdea: (idea: Idea): void => {
    set((state) => {
      const next = state.ideas.some((item) => item.name === idea.name)
        ? state.ideas.map((item) => (item.name === idea.name ? idea : item))
        : [...state.ideas, idea];

      return { ideas: sortIdeas(next) };
    });
  },

  removeIdea: (name: string): void => {
    set((state) => ({
      ideas: state.ideas.filter((idea) => idea.name !== name),
    }));
  },

  reset: () => {
    set({ ...ideasStoreInitialState });
  },
}));
