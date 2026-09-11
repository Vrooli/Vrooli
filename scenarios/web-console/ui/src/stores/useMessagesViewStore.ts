import { create } from "zustand";
import { persist } from "zustand/middleware";

/** Which view a pane shows for its session. */
export type PaneViewMode = "terminal" | "messages";

/**
 * Where the reader left a session's Messages list: the top visible message,
 * how far that message's top sits above the viewport top, and whether the
 * list was following new content. `topSequence` lets restore page the message
 * in when it is outside the loaded window.
 */
export interface MessagesPosition {
  topEventId: string;
  topSequence: number;
  offsetPx: number;
  follow: boolean;
  savedAt: number;
}

/** Positions older than this are dropped when the store hydrates. */
export const MESSAGES_POSITION_TTL_MS = 30 * 24 * 60 * 60 * 1000;

interface MessagesViewState {
  viewModes: Record<string, PaneViewMode>;
  positions: Record<string, MessagesPosition>;
}

interface MessagesViewActions {
  setViewMode: (sessionId: string, mode: PaneViewMode) => void;
  savePosition: (sessionId: string, position: Omit<MessagesPosition, "savedAt">) => void;
  /** Drops everything remembered for a session; called when it is deleted. */
  forget: (sessionId: string) => void;
}

export function pruneStalePositions(
  positions: Record<string, MessagesPosition>,
  now: number,
): Record<string, MessagesPosition> {
  return Object.fromEntries(
    Object.entries(positions).filter(([, position]) => now - position.savedAt <= MESSAGES_POSITION_TTL_MS),
  );
}

/**
 * Per-session Messages view memory that survives reloads: the pane's view
 * mode and the reader's place in the list. Kept apart from the workspace
 * store so it has its own small migration ladder and can be forgotten per
 * session.
 */
export const useMessagesViewStore = create<MessagesViewState & MessagesViewActions>()(
  persist(
    (set) => ({
      viewModes: {},
      positions: {},
      setViewMode: (sessionId, mode) => { set((state) => ({ viewModes: { ...state.viewModes, [sessionId]: mode } })); },
      savePosition: (sessionId, position) => {
        set((state) => ({ positions: { ...state.positions, [sessionId]: { ...position, savedAt: Date.now() } } }));
      },
      forget: (sessionId) => {
        set((state) => ({
          viewModes: Object.fromEntries(Object.entries(state.viewModes).filter(([id]) => id !== sessionId)),
          positions: Object.fromEntries(Object.entries(state.positions).filter(([id]) => id !== sessionId)),
        }));
      },
    }),
    {
      name: "wc-messages-view",
      version: 1,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Record<string, unknown>;
        if (version < 1) {
          // Version 0 predates positions: keep its view modes, remember no place.
          state.viewModes ??= {};
          state.positions ??= {};
        }
        return state as unknown as MessagesViewState & MessagesViewActions;
      },
      partialize: (state) => ({ viewModes: state.viewModes, positions: state.positions }),
      merge: (persisted, current) => {
        const state = (persisted ?? {}) as Partial<MessagesViewState>;
        return {
          ...current,
          viewModes: state.viewModes ?? current.viewModes,
          positions: pruneStalePositions(state.positions ?? {}, Date.now()),
        };
      },
    },
  ),
);
