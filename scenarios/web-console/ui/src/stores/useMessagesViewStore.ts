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

/**
 * The reader open in a session's pane: the reply it shows, and the newest
 * sequence it has accounted for, so replies arriving later read as new.
 */
export interface OpenReader {
  eventId: string;
  seenThrough: number;
}

export const MESSAGES_FONT_MIN = 12;
export const MESSAGES_FONT_MAX = 32;
export const MESSAGES_FONT_DEFAULT = 16;

interface MessagesViewState {
  viewModes: Record<string, PaneViewMode>;
  positions: Record<string, MessagesPosition>;
  /** One persisted text size shared by the list and reader. */
  messagesFontSize: number;
  /** Open readers by session. Kept while the page lives (a tab switch
   *  remounts the pane), never across a reload. */
  readers: Record<string, OpenReader>;
}

interface MessagesViewActions {
  setViewMode: (sessionId: string, mode: PaneViewMode) => void;
  savePosition: (sessionId: string, position: Omit<MessagesPosition, "savedAt">) => void;
  setMessagesFontSize: (size: number) => void;
  /** Opens (or moves) a session's reader; null closes it. */
  setReader: (sessionId: string, reader: OpenReader | null) => void;
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
      messagesFontSize: MESSAGES_FONT_DEFAULT,
      readers: {},
      setViewMode: (sessionId, mode) => { set((state) => ({ viewModes: { ...state.viewModes, [sessionId]: mode } })); },
      savePosition: (sessionId, position) => {
        set((state) => ({ positions: { ...state.positions, [sessionId]: { ...position, savedAt: Date.now() } } }));
      },
      setMessagesFontSize: (size) => { set({ messagesFontSize: size }); },
      setReader: (sessionId, reader) => {
        set((state) => {
          const others = Object.fromEntries(Object.entries(state.readers).filter(([id]) => id !== sessionId));
          return { readers: reader ? { ...others, [sessionId]: reader } : others };
        });
      },
      forget: (sessionId) => {
        set((state) => ({
          viewModes: Object.fromEntries(Object.entries(state.viewModes).filter(([id]) => id !== sessionId)),
          positions: Object.fromEntries(Object.entries(state.positions).filter(([id]) => id !== sessionId)),
          readers: Object.fromEntries(Object.entries(state.readers).filter(([id]) => id !== sessionId)),
        }));
      },
    }),
    {
      name: "wc-messages-view",
      version: 2,
      migrate: (persisted, version) => {
        const state = (persisted ?? {}) as Record<string, unknown>;
        if (version < 1) {
          // Version 0 predates positions: keep its view modes, remember no place.
          state.viewModes ??= {};
          state.positions ??= {};
        }
        if (version < 2) {
          const legacy = state.readerFontSize;
          state.messagesFontSize = typeof legacy === "number" ? legacy : MESSAGES_FONT_DEFAULT;
          delete state.readerFontSize;
        }
        return state as unknown as MessagesViewState & MessagesViewActions;
      },
      partialize: (state) => ({ viewModes: state.viewModes, positions: state.positions, messagesFontSize: state.messagesFontSize }),
      merge: (persisted, current) => {
        const state = (persisted ?? {}) as Partial<MessagesViewState>;
        return {
          ...current,
          viewModes: state.viewModes ?? current.viewModes,
          positions: pruneStalePositions(state.positions ?? {}, Date.now()),
          messagesFontSize: typeof state.messagesFontSize === "number" ? state.messagesFontSize : current.messagesFontSize,
        };
      },
    },
  ),
);
