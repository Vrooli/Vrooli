import { create } from "zustand";
import type { ConversationCursor, ConversationEvent } from "../api/conversation";
import type { MessageCaptureStatus } from "../api/messageCapture";
import { UNKNOWN_CAPTURE } from "../api/messageCapture";

export type PaneViewMode = "terminal" | "messages";

/**
 * How the last attempt to load this session's history ended. The Messages view
 * renders from this, never from `events.length`: an empty array is produced by
 * a load in flight, a failed request, an unreadable transcript, and a genuinely
 * new session, and those need four different answers.
 */
export type ConversationLoadStatus = "unloaded" | "loading" | "loaded" | "failed";

export interface ConversationLoadError {
  /** Message safe to show a user. */
  message: string;
  /** Connect code or "network" — drives copy selection, not display. */
  code: string;
  /** False for faults retrying cannot fix, e.g. an unknown session. */
  retryable: boolean;
}

export type SessionConversationState = {
  events: ConversationEvent[];
  cursor: ConversationCursor;
  hydrated: boolean;
  knownEventIds?: ReadonlySet<string>;
  unreadCount?: number;
  maxSequence?: number;
  hasSequenceGap?: boolean;
  refetchSinceSequence?: number;
  windowOldestSequence?: number;
  hasOlder?: boolean;
  totalCount?: number;
  /** The server's explanation for the event list. */
  capture: MessageCaptureStatus;
  status: ConversationLoadStatus;
  error?: ConversationLoadError;
};

interface ConversationStoreState {
  sessions: Record<string, SessionConversationState>;
  viewModes: Record<string, PaneViewMode>;
}

/** Paging metadata that accompanies a windowed fetch. */
export interface ConversationPage {
  oldestSequence: number;
  hasOlder: boolean;
  totalCount: number;
  capture?: MessageCaptureStatus;
}

interface ConversationStoreActions {
  hydrateSession: (sessionId: string, events: ConversationEvent[], cursor: ConversationCursor, page?: ConversationPage) => void;
  /** Marks a load in flight so the view can show progress instead of emptiness. */
  beginLoad: (sessionId: string) => void;
  /** Records a failed load. The previously loaded events, if any, are kept. */
  failLoad: (sessionId: string, error: ConversationLoadError) => void;
  /** Stores a fresh capture diagnosis without touching events. */
  setCapture: (sessionId: string, capture: MessageCaptureStatus) => void;
  setSessionWindow: (sessionId: string, events: ConversationEvent[], cursor: ConversationCursor, page: ConversationPage) => void;
  prependEvents: (sessionId: string, events: ConversationEvent[], page: ConversationPage) => void;
  appendEvent: (event: ConversationEvent) => void;
  /**
   * Merge a batch of events into a session, skipping any whose id already
   * exists. Used by reconnect / view-open / out-of-sync refresh paths to
   * fill gaps without disturbing the tail the live WS already delivered.
   */
  mergeEvents: (sessionId: string, events: ConversationEvent[], cursor?: ConversationCursor, capture?: MessageCaptureStatus) => void;
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

// Every site that lazily creates a session must produce the same shape;
// spreading this keeps a partially-initialized session from reaching the view
// with an undefined status and rendering as "loaded and empty". Exported so
// tests seeding the store cannot drift from the real shape either.
export const createConversationSessionState = (
  overrides: Partial<SessionConversationState> = {},
): SessionConversationState => ({ ...blankSession(), ...overrides });

const blankSession = (): SessionConversationState => ({
  events: [],
  cursor: defaultCursor(),
  hydrated: true,
  capture: UNKNOWN_CAPTURE,
  status: "loaded",
});

// Shared frozen empty array so selectors for sessions with no events return a
// referentially-stable value. Returning a fresh `[]` each call would defeat
// Zustand's Object.is short-circuit and re-render the subscriber on every
// unrelated store update.
const EMPTY_EVENTS: readonly ConversationEvent[] = Object.freeze([]);

function sessionMetadata(events: ConversationEvent[], cursor: ConversationCursor) {
  const knownEventIds = new Set(events.map((event) => event.id));
  let maxSequence = 0;
  let previous = 0;
  let hasSequenceGap = false;
  let refetchSinceSequence = 0;
  let unreadCount = 0;
  for (const event of events) {
    maxSequence = Math.max(maxSequence, event.sequence);
    if (previous > 0 && event.sequence !== previous + 1) {
      hasSequenceGap = true;
      if (refetchSinceSequence === 0) refetchSinceSequence = previous;
    }
    previous = event.sequence;
    if (event.role === "assistant" && event.sequence > cursor.lastSeenSequence) unreadCount += 1;
  }
  if ((events[0]?.sequence ?? 1) > 1) refetchSinceSequence = 0;
  return { knownEventIds, unreadCount, maxSequence, hasSequenceGap: hasSequenceGap || (events[0]?.sequence ?? 1) > 1, refetchSinceSequence };
}

export const useConversationStore = create<ConversationStoreState & ConversationStoreActions>((set) => ({
  sessions: {},
  viewModes: {},

  hydrateSession: (sessionId, events, cursor, page) => set((state) => {
    // Merge with anything appendEvent already added while the GET was in
    // flight — naively replacing would drop live WS events that arrived
    // after the request but before the response, leaving a permanent
    // sequence gap (refresh uses max offset as since_sequence and never
    // backfills lower events). See useConversationSession mount effect.
    const existing = state.sessions[sessionId];
    const capture = page?.capture ?? existing?.capture ?? UNKNOWN_CAPTURE;
    if (!existing || existing.events.length === 0) {
      return {
        sessions: {
          ...state.sessions,
          [sessionId]: { events, cursor, hydrated: true, ...sessionMetadata(events, cursor), windowOldestSequence: page?.oldestSequence, hasOlder: page?.hasOlder, totalCount: page?.totalCount, capture, status: "loaded", error: undefined },
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
        [sessionId]: { events: merged, cursor, hydrated: true, ...sessionMetadata(merged, cursor), windowOldestSequence: page?.oldestSequence, hasOlder: page?.hasOlder, totalCount: page?.totalCount, capture, status: "loaded", error: undefined },
      },
    };
  }),

  beginLoad: (sessionId) => set((state) => {
    const existing = state.sessions[sessionId];
    // A reload of an already-loaded session keeps rendering its events; only a
    // session with nothing on screen shows a loading state.
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: existing
          ? { ...existing, status: "loading", error: undefined }
          : { ...blankSession(), hydrated: false, status: "loading" },
      },
    };
  }),

  failLoad: (sessionId, error) => set((state) => {
    const existing = state.sessions[sessionId] ?? { ...blankSession(), hydrated: false };
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: { ...existing, status: "failed", error },
      },
    };
  }),

  setCapture: (sessionId, capture) => set((state) => {
    const existing = state.sessions[sessionId];
    if (!existing) {
      return { sessions: { ...state.sessions, [sessionId]: { ...blankSession(), hydrated: false, status: "unloaded", capture } } };
    }
    if (existing.capture === capture) return state;
    return { sessions: { ...state.sessions, [sessionId]: { ...existing, capture } } };
  }),

  setSessionWindow: (sessionId, events, cursor, page) => set((state) => ({
    sessions: {
      ...state.sessions,
      [sessionId]: {
        events,
        cursor,
        hydrated: true,
        ...sessionMetadata(events, cursor),
        windowOldestSequence: page.oldestSequence,
        hasOlder: page.hasOlder,
        totalCount: page.totalCount,
        capture: page.capture ?? state.sessions[sessionId]?.capture ?? UNKNOWN_CAPTURE,
        status: "loaded",
        error: undefined,
      },
    },
  })),

  prependEvents: (sessionId, incoming, page) => set((state) => {
    const existing = state.sessions[sessionId] ?? blankSession();
    const known = existing.knownEventIds ?? new Set(existing.events.map((event) => event.id));
    const added = incoming.filter((event) => !known.has(event.id));
    const events = added.length > 0 ? [...added, ...existing.events].sort((a, b) => a.sequence - b.sequence) : existing.events;
    return { sessions: { ...state.sessions, [sessionId]: { ...existing, events, ...sessionMetadata(events, existing.cursor), windowOldestSequence: page.oldestSequence, hasOlder: page.hasOlder, totalCount: page.totalCount } } };
  }),

  appendEvent: (event) => set((state) => {
    const existing = state.sessions[event.sessionId] ?? blankSession();
    const knownEventIds = existing.knownEventIds ?? new Set(existing.events.map((candidate) => candidate.id));
    if (knownEventIds.has(event.id)) {
      return state;
    }
    const nextKnownEventIds = new Set(knownEventIds).add(event.id);
    const hasSequenceGap = (existing.hasSequenceGap ?? false) || ((existing.maxSequence ?? 0) > 0 && event.sequence !== (existing.maxSequence ?? 0) + 1);
    const unreadCount = (existing.unreadCount ?? getSessionUnreadCount(state, event.sessionId)) + (event.role === "assistant" && event.sequence > existing.cursor.lastSeenSequence ? 1 : 0);
    return {
      sessions: {
        ...state.sessions,
        [event.sessionId]: {
          ...existing,
          events: [...existing.events, event],
          knownEventIds: nextKnownEventIds,
          unreadCount,
          maxSequence: Math.max(existing.maxSequence ?? 0, event.sequence),
          hasSequenceGap,
        },
      },
    };
  }),

  mergeEvents: (sessionId, incoming, cursor, capture) => set((state) => {
    const existing = state.sessions[sessionId] ?? blankSession();
    if (incoming.length === 0 && !cursor && !capture) {
      // Still mark hydrated so UI stops showing "not loaded" states.
      if (existing.hydrated && existing.status === "loaded") return state;
      return {
        sessions: {
          ...state.sessions,
          [sessionId]: { ...existing, hydrated: true, status: "loaded", error: undefined },
        },
      };
    }
    const seen = existing.knownEventIds ?? new Set(existing.events.map((e) => e.id));
    const added = incoming.filter((e) => !seen.has(e.id));
    const merged = added.length > 0
      ? [...existing.events, ...added].sort((a, b) => a.sequence - b.sequence)
      : existing.events;
    return {
      sessions: {
        ...state.sessions,
        // Spreading `existing` is load-bearing, not tidiness. Rebuilding the
        // session from scratch here dropped windowOldestSequence/hasOlder/
        // totalCount on every merge, and because the pane refreshes on mount
        // and on every window focus, "load older messages" stopped working
        // almost immediately and stayed broken.
        [sessionId]: {
          ...existing,
          events: merged,
          cursor: cursor ?? existing.cursor,
          hydrated: true,
          status: "loaded",
          error: undefined,
          capture: capture ?? existing.capture,
          ...sessionMetadata(merged, cursor ?? existing.cursor),
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
    const existing = state.sessions[sessionId] ?? blankSession();
    return {
      sessions: {
        ...state.sessions,
        [sessionId]: {
          ...existing,
          cursor: {
            lastSeenSequence: cursor.lastSeenSequence ?? existing.cursor.lastSeenSequence,
            lastListenedSequence: cursor.lastListenedSequence ?? existing.cursor.lastListenedSequence,
          },
          ...sessionMetadata(existing.events, {
            lastSeenSequence: cursor.lastSeenSequence ?? existing.cursor.lastSeenSequence,
            lastListenedSequence: cursor.lastListenedSequence ?? existing.cursor.lastListenedSequence,
          }),
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
  return state.sessions[sessionId]?.events ?? (EMPTY_EVENTS as ConversationEvent[]);
}

/**
 * What the Messages view should render, resolved in one place.
 *
 * "messages"     — there is history to show.
 * "loading"      — a first load is in flight; show progress, never emptiness.
 * "failed"       — the request did not complete; retryable.
 * "unavailable"  — the server can't capture this session's messages; has a cause.
 * "not-applicable" — no agent runs here, so there is nothing to capture.
 * "empty"        — capture works and nobody has said anything yet.
 *
 * The ordering below is the whole point: every condition that explains an empty
 * list is checked before falling through to "empty". Previously the view tested
 * `events.length === 0` first, so all six of these rendered the same sentence.
 */
export type ConversationViewState =
  | { kind: "messages" }
  | { kind: "loading" }
  | { kind: "failed"; error: ConversationLoadError }
  | { kind: "unavailable"; capture: MessageCaptureStatus }
  | { kind: "not-applicable"; capture: MessageCaptureStatus }
  | { kind: "empty"; capture: MessageCaptureStatus };

const LOADING_VIEW: ConversationViewState = { kind: "loading" };
const MESSAGES_VIEW: ConversationViewState = { kind: "messages" };

/**
 * resolveConversationView is a pure function of one session slice rather than a
 * store selector, because three of its results allocate. Subscribing to a
 * selector that returns a fresh object on every call defeats Zustand's Object.is
 * short-circuit and re-renders the pane on every unrelated store write; callers
 * subscribe to the slice (which is referentially stable) and memoize this.
 */
export function resolveConversationView(session: SessionConversationState | undefined): ConversationViewState {
  if (!session) return LOADING_VIEW;
  if (session.events.length > 0) return MESSAGES_VIEW;
  if (session.status === "loading" || session.status === "unloaded") return LOADING_VIEW;
  if (session.status === "failed" && session.error) return { kind: "failed", error: session.error };
  const capture = session.capture ?? UNKNOWN_CAPTURE;
  if (capture.state === "unavailable") return { kind: "unavailable", capture };
  if (capture.state === "not_applicable") return { kind: "not-applicable", capture };
  return { kind: "empty", capture };
}

/** Referentially-stable slice selector; pair with resolveConversationView. */
export function getSessionSlice(state: ConversationStoreState, sessionId: string): SessionConversationState | undefined {
  return state.sessions[sessionId];
}

/** Convenience for tests and non-render callers. */
export function getSessionConversationView(state: ConversationStoreState, sessionId: string): ConversationViewState {
  return resolveConversationView(state.sessions[sessionId]);
}

export function getSessionCapture(state: ConversationStoreState, sessionId: string): MessageCaptureStatus {
  return state.sessions[sessionId]?.capture ?? UNKNOWN_CAPTURE;
}

export function getSessionConversationCursor(state: ConversationStoreState, sessionId: string): ConversationCursor {
  return state.sessions[sessionId]?.cursor ?? defaultCursor();
}

export function getSessionUnreadCount(state: ConversationStoreState, sessionId: string): number {
  const session = state.sessions[sessionId];
  if (!session) return 0;
  return session.unreadCount ?? session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
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
  const session = state.sessions[sessionId];
  const events = session?.events;
  if (!events || events.length === 0) return 0;
  if (session?.maxSequence != null && session.hasSequenceGap != null) {
    return session.hasSequenceGap ? (session.refetchSinceSequence ?? 0) : session.maxSequence;
  }
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
