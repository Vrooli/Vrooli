import { create } from "zustand";
import type { ConversationCursor, ConversationEvent } from "../lib/api";

export type PaneViewMode = "terminal" | "messages";

type SessionConversationState = {
  events: ConversationEvent[];
  cursor: ConversationCursor;
  hydrated: boolean;
};

interface ConversationStoreState {
  sessions: Record<string, SessionConversationState>;
  viewModes: Record<string, PaneViewMode>;
}

interface ConversationStoreActions {
  hydrateSession: (sessionId: string, events: ConversationEvent[], cursor: ConversationCursor) => void;
  appendEvent: (event: ConversationEvent) => void;
  /**
   * Merge a batch of events into a session, skipping any whose id already
   * exists. Used by reconnect / view-open / out-of-sync refresh paths to
   * fill gaps without disturbing the tail the live WS already delivered.
   */
  mergeEvents: (sessionId: string, events: ConversationEvent[], cursor?: ConversationCursor) => void;
  /** Merge updated fields (e.g. summarization) into an existing event by ID. */
  updateEvent: (sessionId: string, eventId: string, patch: { speechParagraphs?: string[]; originalSpeechParagraphs?: string[]; summarized?: boolean }) => void;
  updateCursor: (sessionId: string, cursor: Partial<ConversationCursor>) => void;
  setViewMode: (sessionId: string, mode: PaneViewMode) => void;
  clearSession: (sessionId: string) => void;
}

const defaultCursor = (): ConversationCursor => ({
  lastSeenSequence: 0,
  lastListenedSequence: 0,
});

export const useConversationStore = create<ConversationStoreState & ConversationStoreActions>((set) => ({
  sessions: {},
  viewModes: {},

  hydrateSession: (sessionId, events, cursor) => set((state) => ({
    sessions: {
      ...state.sessions,
      [sessionId]: {
        events,
        cursor,
        hydrated: true,
      },
    },
  })),

  appendEvent: (event) => set((state) => {
    const existing = state.sessions[event.sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
    if (existing.events.some((candidate) => candidate.id === event.id)) {
      return state;
    }
    return {
      sessions: {
        ...state.sessions,
        [event.sessionId]: {
          ...existing,
          events: [...existing.events, event],
        },
      },
    };
  }),

  mergeEvents: (sessionId, incoming, cursor) => set((state) => {
    const existing = state.sessions[sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
    if (incoming.length === 0 && !cursor) {
      // Still mark hydrated so UI stops showing "not loaded" states.
      if (existing.hydrated) return state;
      return {
        sessions: {
          ...state.sessions,
          [sessionId]: { ...existing, hydrated: true },
        },
      };
    }
    const seen = new Set(existing.events.map((e) => e.id));
    const added = incoming.filter((e) => !seen.has(e.id));
    const merged = added.length > 0
      ? [...existing.events, ...added].sort((a, b) => a.sequence - b.sequence)
      : existing.events;
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: {
          events: merged,
          cursor: cursor ?? existing.cursor,
          hydrated: true,
        },
      },
    };
  }),

  updateEvent: (sessionId, eventId, patch) => set((state) => {
    const session = state.sessions[sessionId];
    if (!session) return state;
    const idx = session.events.findIndex((e) => e.id === eventId);
    if (idx === -1) return state;
    const updatedEvents = [...session.events];
    const existing = updatedEvents[idx] as ConversationEvent;
    const ev: ConversationEvent = { ...existing };
    if (patch.speechParagraphs != null) ev.speechParagraphs = patch.speechParagraphs;
    if (patch.originalSpeechParagraphs != null) ev.originalSpeechParagraphs = patch.originalSpeechParagraphs;
    if (patch.summarized != null) ev.summarized = patch.summarized;
    updatedEvents[idx] = ev;
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: { ...session, events: updatedEvents },
      },
    };
  }),

  updateCursor: (sessionId, cursor) => set((state) => {
    const existing = state.sessions[sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: {
          ...existing,
          cursor: {
            lastSeenSequence: cursor.lastSeenSequence ?? existing.cursor.lastSeenSequence,
            lastListenedSequence: cursor.lastListenedSequence ?? existing.cursor.lastListenedSequence,
          },
        },
      },
    };
  }),

  setViewMode: (sessionId, mode) => set((state) => ({
    viewModes: {
      ...state.viewModes,
      [sessionId]: mode,
    },
  })),

  clearSession: (sessionId) => set((state) => {
    const sessions = { ...state.sessions };
    const viewModes = { ...state.viewModes };
    delete sessions[sessionId];
    delete viewModes[sessionId];
    return { sessions, viewModes };
  }),
}));

export function getSessionConversationEvents(state: ConversationStoreState, sessionId: string): ConversationEvent[] {
  return state.sessions[sessionId]?.events ?? [];
}

export function getSessionConversationCursor(state: ConversationStoreState, sessionId: string): ConversationCursor {
  return state.sessions[sessionId]?.cursor ?? defaultCursor();
}

export function getSessionUnreadCount(state: ConversationStoreState, sessionId: string): number {
  const session = state.sessions[sessionId];
  if (!session) return 0;
  return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
}

export function getSessionUnlistenedEvents(state: ConversationStoreState, sessionId: string): ConversationEvent[] {
  const session = state.sessions[sessionId];
  if (!session) return [];
  return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastListenedSequence);
}

export function getSessionViewMode(state: ConversationStoreState, sessionId: string): PaneViewMode {
  return state.viewModes[sessionId] ?? "terminal";
}
