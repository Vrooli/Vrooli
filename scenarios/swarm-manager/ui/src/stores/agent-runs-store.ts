import { create } from "zustand";
import type { AgentRunState, AgentRunStatus, BacklogKind } from "../types";
import { agentManagerService } from "../services";

const AGENT_RUNS_STORAGE_KEY = "swarm-manager.agent-runs.v1";

export interface AgentRunRecord extends AgentRunState {
  taskId: string;
  createdAt: string;
  baseUrl: string;
  backlogKind?: BacklogKind;
  backlogName?: string;
  backlogTitle?: string;
  mode?: string;
  isStopping: boolean;
  lastCheckedAt: number | null;
}

interface UpsertSpawnedRunInput {
  runId: string;
  taskId: string;
  baseUrl: string;
  createdAt: string;
  backlogKind?: BacklogKind;
  backlogName?: string;
  backlogTitle?: string;
  mode?: string;
}

interface AgentRunsStoreState {
  runs: AgentRunRecord[];
  isRefreshing: boolean;
  upsertSpawnedRun: (input: UpsertSpawnedRunInput) => void;
  refreshRun: (runId: string) => Promise<void>;
  refreshActiveRuns: () => Promise<void>;
  stopRun: (runId: string) => Promise<void>;
  removeRun: (runId: string) => void;
}

const activeStatuses: AgentRunStatus[] = ["pending", "starting", "running", "needs_review"];

const isActiveStatus = (status: AgentRunStatus): boolean => activeStatuses.includes(status);

const sortRuns = (runs: AgentRunRecord[]): AgentRunRecord[] => {
  return [...runs].sort((a, b) => {
    const first = new Date(a.createdAt).getTime();
    const second = new Date(b.createdAt).getTime();
    return second - first;
  });
};

const mergeRun = (runs: AgentRunRecord[], runId: string, updater: (run: AgentRunRecord) => AgentRunRecord): AgentRunRecord[] => {
  return runs.map((run) => (run.runId === runId ? updater(run) : run));
};

const applyRunState = (run: AgentRunRecord, state: AgentRunState): AgentRunRecord => ({
  ...run,
  taskId: state.taskId ?? run.taskId,
  status: state.status,
  startedAt: state.startedAt,
  finishedAt: state.finishedAt,
  errorMessage: state.errorMessage,
  durationSeconds: state.durationSeconds,
  active: state.active,
  lastCheckedAt: Date.now(),
});

export const agentRunsStoreInitialState = {
  runs: [],
  isRefreshing: false,
};

const loadPersistedRuns = (): AgentRunRecord[] => {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(AGENT_RUNS_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as { runs?: AgentRunRecord[] };
    if (!parsed.runs || !Array.isArray(parsed.runs)) return [];
    return parsed.runs;
  } catch {
    return [];
  }
};

const persistRuns = (runs: AgentRunRecord[]): void => {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(AGENT_RUNS_STORAGE_KEY, JSON.stringify({ runs }));
  } catch {
    // Ignore storage failures and continue with in-memory tracking.
  }
};

const trimRuns = (runs: AgentRunRecord[]): AgentRunRecord[] => {
  const sorted = sortRuns(runs);
  return sorted.slice(0, 100);
};

export const useAgentRunsStore = create<AgentRunsStoreState>((set, get) => ({
  ...agentRunsStoreInitialState,
  runs: loadPersistedRuns(),

  upsertSpawnedRun: (input): void => {
    set((state) => {
      const existing = state.runs.find((run) => run.runId === input.runId);
      if (existing) {
        const nextRuns = trimRuns(
          mergeRun(state.runs, input.runId, (run) => ({
            ...run,
            taskId: input.taskId,
            baseUrl: input.baseUrl,
            createdAt: input.createdAt,
            backlogKind: input.backlogKind,
            backlogName: input.backlogName,
            backlogTitle: input.backlogTitle,
            mode: input.mode,
          }))
        );
        persistRuns(nextRuns);
        return {
          runs: nextRuns,
        };
      }
      const nextRuns = trimRuns([
        {
          runId: input.runId,
          taskId: input.taskId,
          baseUrl: input.baseUrl,
          createdAt: input.createdAt,
          backlogKind: input.backlogKind,
          backlogName: input.backlogName,
          backlogTitle: input.backlogTitle,
          mode: input.mode,
          status: "pending",
          startedAt: undefined,
          finishedAt: undefined,
          errorMessage: undefined,
          durationSeconds: undefined,
          active: true,
          isStopping: false,
          lastCheckedAt: null,
        },
        ...state.runs,
      ]);
      persistRuns(nextRuns);
      return {
        runs: nextRuns,
      };
    });
  },

  refreshRun: async (runId): Promise<void> => {
    const run = get().runs.find((entry) => entry.runId === runId);
    if (!run) return;

    try {
      const state = await agentManagerService.getRunState(runId);
      set((prev) => ({
        runs: (() => {
          const nextRuns = trimRuns(mergeRun(prev.runs, runId, (entry) => applyRunState(entry, state)));
          persistRuns(nextRuns);
          return nextRuns;
        })(),
      }));
    } catch {
      // Keep stale status if fetch fails; next poll will retry.
    }
  },

  refreshActiveRuns: async (): Promise<void> => {
    if (get().isRefreshing) return;

    const activeRuns = get().runs.filter((run) => isActiveStatus(run.status));
    if (activeRuns.length === 0) return;

    set({ isRefreshing: true });
    try {
      const states = await Promise.all(
        activeRuns.map(async (run) => {
          try {
            return await agentManagerService.getRunState(run.runId);
          } catch {
            return null;
          }
        })
      );
      const byRunId = new Map(states.filter((state): state is AgentRunState => state !== null).map((state) => [state.runId, state]));
      set((prev) => ({
        runs: (() => {
          const nextRuns = trimRuns(
            prev.runs.map((run) => {
              const state = byRunId.get(run.runId);
              return state ? applyRunState(run, state) : run;
            })
          );
          persistRuns(nextRuns);
          return nextRuns;
        })(),
      }));
    } finally {
      set({ isRefreshing: false });
    }
  },

  stopRun: async (runId): Promise<void> => {
    const run = get().runs.find((entry) => entry.runId === runId);
    if (!run || run.isStopping) return;

    set((prev) => ({
      runs: mergeRun(prev.runs, runId, (entry) => ({ ...entry, isStopping: true })),
    }));

    try {
      await agentManagerService.stopRun(runId);
      await get().refreshRun(runId);
    } catch {
      // ignore network errors; preserve existing state
    } finally {
      set((prev) => ({
        runs: (() => {
          const nextRuns = trimRuns(mergeRun(prev.runs, runId, (entry) => ({ ...entry, isStopping: false })));
          persistRuns(nextRuns);
          return nextRuns;
        })(),
      }));
    }
  },

  removeRun: (runId): void => {
    set((state) => {
      const nextRuns = state.runs.filter((run) => run.runId !== runId);
      persistRuns(nextRuns);
      return { runs: nextRuns };
    });
  },
}));

export const selectActiveAgentRuns = (state: AgentRunsStoreState): AgentRunRecord[] => {
  return state.runs.filter((run) => isActiveStatus(run.status));
};

export const selectLatestRunForBacklog = (
  state: AgentRunsStoreState,
  backlogKind: BacklogKind,
  backlogName: string
): AgentRunRecord | null => {
  const match = state.runs.find((run) => run.backlogKind === backlogKind && run.backlogName === backlogName);
  return match ?? null;
};
