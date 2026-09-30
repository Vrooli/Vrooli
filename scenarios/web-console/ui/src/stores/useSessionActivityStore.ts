import { create } from "zustand";
import type { SessionActivityView } from "../api/sessionActivity";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#session-activity-contract

/**
 * What each session's agent is doing between messages, as pushed by the
 * server (`session_activity` on the event stream) and seeded from the sessions
 * list. Not persisted: activity is only true now.
 */
interface SessionActivityStoreState {
  activities: Record<string, SessionActivityView>;
  /** Take a pushed activity unless its state began before the stored one. */
  apply: (sessionId: string, activity: SessionActivityView) => void;
  /** Seed from the sessions list; a newer push is never overridden. */
  hydrate: (sessions: readonly { id: string; activity?: SessionActivityView }[]) => void;
  forget: (sessionId: string) => void;
}

function startedBefore(next: SessionActivityView, stored: SessionActivityView | undefined): boolean {
  if (!stored) return false;
  const nextAt = Date.parse(next.since);
  const storedAt = Date.parse(stored.since);
  return Number.isFinite(nextAt) && Number.isFinite(storedAt) && nextAt < storedAt;
}

export const useSessionActivityStore = create<SessionActivityStoreState>()((set) => ({
  activities: {},
  apply: (sessionId, activity) => {
    set((state) => {
      if (startedBefore(activity, state.activities[sessionId])) return state;
      return { activities: { ...state.activities, [sessionId]: activity } };
    });
  },
  hydrate: (sessions) => {
    set((state) => {
      let activities = state.activities;
      for (const session of sessions) {
        if (!session.activity || startedBefore(session.activity, activities[session.id])) continue;
        if (activities === state.activities) activities = { ...activities };
        activities[session.id] = session.activity;
      }
      return activities === state.activities ? state : { activities };
    });
  },
  forget: (sessionId) => {
    set((state) => {
      if (!(sessionId in state.activities)) return state;
      const { [sessionId]: _forgotten, ...activities } = state.activities;
      return { activities };
    });
  },
}));
