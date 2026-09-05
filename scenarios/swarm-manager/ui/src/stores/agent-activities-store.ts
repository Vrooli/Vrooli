import { create } from "zustand";
import { agentActivityService } from "../services/agent-activity-service";
import type { AgentActivity, AgentActivityStatus, BacklogKind } from "../types";

export interface AgentActivityRecord extends AgentActivity {
  isStopping: boolean;
}

interface AgentActivitiesStoreState {
  activities: AgentActivityRecord[];
  isRefreshing: boolean;
  refreshActivities: (activeOnly?: boolean) => Promise<void>;
  stopRun: (runId: string) => Promise<void>;
}

/**
 * Statuses that keep an activity relevant to coordination/UI. This includes
 * `needs_review`, which is no longer executing but is still awaiting a user
 * decision and should remain discoverable on the item.
 */
const TRACKED_STATUSES: ReadonlySet<AgentActivityStatus> = new Set<AgentActivityStatus>([
  "pending",
  "starting",
  "running",
  "needs_review",
]);

const sortActivities = (activities: AgentActivityRecord[]): AgentActivityRecord[] => {
  return [...activities].sort((a, b) => {
    const first = new Date(a.requestedAt).getTime();
    const second = new Date(b.requestedAt).getTime();
    return second - first;
  });
};

export const agentActivitiesStoreInitialState = {
  activities: [],
  isRefreshing: false,
};

export const useAgentActivitiesStore = create<AgentActivitiesStoreState>((set, get) => ({
  ...agentActivitiesStoreInitialState,

  refreshActivities: async (activeOnly = true): Promise<void> => {
    if (get().isRefreshing) return;
    set({ isRefreshing: true });
    try {
      const activities = await agentActivityService.list({ active: activeOnly });
      set({
        activities: sortActivities(
          activities.map((activity) => ({
            ...activity,
            isStopping: get().activities.find((entry) => entry.runId === activity.runId)?.isStopping ?? false,
          }))
        ),
      });
    } catch {
      // Activity polling is supplemental UI state. Preserve the last known
      // activities instead of surfacing a secondary failure over primary page data.
    } finally {
      set({ isRefreshing: false });
    }
  },

  stopRun: async (runId): Promise<void> => {
    if (!runId) return;
    const current = get().activities.find((entry) => entry.runId === runId);
    if (!current || current.isStopping) return;

    set((state) => ({
      activities: state.activities.map((activity) =>
        activity.runId === runId ? { ...activity, isStopping: true } : activity
      ),
    }));

    try {
      await agentActivityService.stopRun(runId);
      await get().refreshActivities(true);
    } catch {
      set((state) => ({
        activities: state.activities.map((activity) =>
          activity.runId === runId ? { ...activity, isStopping: false } : activity
        ),
      }));
    }
  },
}));

export const selectActiveAgentActivities = (state: AgentActivitiesStoreState): AgentActivityRecord[] => {
  return state.activities.filter((activity) => TRACKED_STATUSES.has(activity.status));
};

export const selectLatestActivityForBacklog = (
  state: AgentActivitiesStoreState,
  backlogKind: BacklogKind,
  backlogName: string
): AgentActivityRecord | null => {
  const match = state.activities.find(
    (activity) =>
      activity.ownerType === "backlog" &&
      activity.ownerKind === backlogKind &&
      activity.ownerName === backlogName &&
      TRACKED_STATUSES.has(activity.status)
  );
  return match ?? null;
};
