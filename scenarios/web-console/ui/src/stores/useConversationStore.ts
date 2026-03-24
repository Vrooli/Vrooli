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
