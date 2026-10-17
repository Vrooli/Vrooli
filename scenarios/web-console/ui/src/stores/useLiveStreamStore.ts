import { create } from "zustand";

/**
 * Health of the server-sent event stream that delivers live conversation
 * updates.
 *
 * The stream used to fail silently: `onerror` called `console.warn` and nothing
 * else. A dropped connection — routine when a phone backgrounds the tab — left
 * the Messages view showing stale content with no indication that new messages
 * had stopped arriving, which is a large part of why refreshing by hand felt
 * necessary and unreliable.
 */
export type LiveStreamStatus = "connecting" | "open" | "reconnecting" | "closed";

interface LiveStreamState {
  status: LiveStreamStatus;
  /** Incremented on each successful (re)connect; useful for resync triggers. */
  connectedGeneration: number;
  setStatus: (status: LiveStreamStatus) => void;
}

export const useLiveStreamStore = create<LiveStreamState>((set) => ({
  status: "connecting",
  connectedGeneration: 0,
  setStatus: (status) => { set((state) => {
    if (state.status === status) return state;
    return {
      status,
      connectedGeneration: status === "open" ? state.connectedGeneration + 1 : state.connectedGeneration,
    };
  }); },
}));

/** True when live updates are not currently arriving. */
export function isLiveStreamInterrupted(status: LiveStreamStatus): boolean {
  return status === "reconnecting" || status === "closed";
}

/**
 * How long an interruption must persist before it is worth telling the user
 * about.
 *
 * Most drops are sub-second — a server restart, a phone switching networks —
 * and EventSource recovers before anyone could act on the news. Announcing
 * them instantly produced a banner that flickered in and out during normal
 * operation and, worse, sat on screen contradicting a "Up to date" message
 * that had just been shown.
 */
export const LIVE_INTERRUPTION_GRACE_MS = 4000;
