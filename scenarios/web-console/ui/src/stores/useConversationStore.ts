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

  hydrateSession: (sessionId, events, cursor) => set((state) => {
    // Merge with anything appendEvent already added while the GET was in
    // flight — naively replacing would drop live WS events that arrived
    // after the request but before the response, leaving a permanent
    // sequence gap (refresh uses max seq as since_sequence and never
    // backfills lower events). See useConversationSession mount effect.
    const existing = state.sessions[sessionId];
    if (!existing || existing.events.length === 0) {
      return {
        sessions: {
          ...state.sessions,
          [sessionId]: { events, cursor, hydrated: true },
        },
      };
    }
    const seen = new Set(events.map((e) => e.id));
    const extras = existing.events.filter((e) => !seen.has(e.id));
    const merged = extras.length > 0
      ? [...events, ...extras].sort((a, b) => a.sequence - b.sequence)
      : events;
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: { events: merged, cursor, hydrated: true },
      },
    };
  }),

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

/**
 * Returns the `since_sequence` value to use for a refetch that will both pull
 * any new tail events AND backfill any gap in the local sequence. Strategy:
 *   - empty store → 0 (full hydrate)
 *   - first event's sequence > 1 → 0 (we may be missing the prefix)
 *   - first internal gap at events[i] → events[i-1].sequence (refetch from before the gap)
 *   - no gap → max sequence (current tail-only behaviour)
 */
export function getSessionRefetchSinceSequence(state: ConversationStoreState, sessionId: string): number {
  const events = state.sessions[sessionId]?.events;
  if (!events || events.length === 0) return 0;
  const sorted = [...events].sort((a, b) => a.sequence - b.sequence);
  const first = sorted[0];
  if (!first || first.sequence > 1) return 0;
  for (let i = 1; i < sorted.length; i++) {
    const prev = sorted[i - 1];
    const cur = sorted[i];
    if (!prev || !cur) continue;
    if (cur.sequence !== prev.sequence + 1) {
      return prev.sequence;
    }
  }
  const last = sorted[sorted.length - 1];
  return last ? last.sequence : 0;
}
