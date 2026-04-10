/**
 * Session Store
 *
 * Centralized Zustand store for recording session state.
 * This is the single source of truth for all session-related data,
 * ensuring WebSocket subscriptions and API calls use the correct session ID.
 *
 * Key benefits:
 * - Prevents stale session ID issues after driver restart
 * - Centralizes session validation before WebSocket subscriptions
 * - Provides consistent state access across all recording components
 */

import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import { getApiBase } from '@/config';
import type { ActualViewport } from '../api/schemas';
import type { Page } from '../hooks/usePages';
import type { TimelineEntry } from '../api/schemas';
import type { RetryState } from '../services';

// ============================================================================
// Types
// ============================================================================

/** Page color for visual distinction in multi-tab UI */
export type PageColor =
  | 'bg-blue-500'
  | 'bg-green-500'
  | 'bg-purple-500'
  | 'bg-orange-500'
  | 'bg-pink-500'
  | 'bg-cyan-500'
  | 'bg-yellow-500'
  | 'bg-red-500'
  | 'bg-gray-500';

/** Frame dimensions from the stream (bitmap size) */
export interface FrameDimensions {
  width: number;
  height: number;
}

/** Display dimensions (rendered on screen) */
export interface DisplayDimensions {
  width: number;
  height: number;
}

// ============================================================================
// Store State Interface
// ============================================================================

interface SessionState {
  // Core session identity
  sessionId: string | null;
  profileId: string | null;
  actualViewport: ActualViewport | null;
  initialRestoredUrl: string | null;
  /** Session has been confirmed to exist on the server */
  isValidated: boolean;

  // Lifecycle state
  isCreating: boolean;
  isValidating: boolean;
  error: string | null;
  retryState: RetryState;

  // Pages/Tabs
  pages: Map<string, Page>;
  activePageId: string | null;
  pageColorMap: Map<string, PageColor>;

  // Timeline
  timelineEntries: TimelineEntry[];
  timelineLoading: boolean;
  timelineTotalCount: number;
  timelineHasMore: boolean;

  // Display dimensions (for coordinate mapping)
  frameDimensions: FrameDimensions | null;
  displayDimensions: DisplayDimensions | null;
}

// ============================================================================
// Store Actions Interface
// ============================================================================

interface SessionActions {
  // Session lifecycle
  setSession: (data: {
    sessionId: string;
    profileId?: string | null;
    actualViewport?: ActualViewport | null;
    initialRestoredUrl?: string | null;
  }) => void;
  /** Validate that a session exists on the server before subscribing */
  validateSession: (sessionId: string) => Promise<boolean>;
  clearSession: () => void;
  setIsCreating: (isCreating: boolean) => void;
  setIsValidating: (isValidating: boolean) => void;
  setError: (error: string | null) => void;
  setRetryState: (retryState: Partial<RetryState>) => void;
  /** Update session ID without full session data (e.g., from URL) */
  setSessionId: (sessionId: string | null) => void;
  /** Mark session as validated (when we know it's valid from another source) */
  markValidated: () => void;

  // Pages/Tabs
  setPages: (pages: Map<string, Page>) => void;
  addPage: (page: Page) => void;
  updatePage: (pageId: string, updates: Partial<Page>) => void;
  removePage: (pageId: string) => void;
  setActivePageId: (pageId: string | null) => void;
  updatePageColorMap: (pages: Page[]) => void;

  // Timeline
  setTimelineEntries: (entries: TimelineEntry[]) => void;
  addTimelineEntry: (entry: TimelineEntry) => void;
  setTimelineLoading: (loading: boolean) => void;
  setTimelineMeta: (meta: { totalCount?: number; hasMore?: boolean }) => void;
  clearTimeline: () => void;

  // Display dimensions
  setFrameDimensions: (dims: FrameDimensions | null) => void;
  setDisplayDimensions: (dims: DisplayDimensions | null) => void;
}

type SessionStore = SessionState & SessionActions;

// ============================================================================
// Constants
// ============================================================================

const PAGE_COLORS: PageColor[] = [
  'bg-blue-500',
  'bg-green-500',
  'bg-purple-500',
  'bg-orange-500',
  'bg-pink-500',
  'bg-cyan-500',
  'bg-yellow-500',
  'bg-red-500',
  'bg-gray-500',
];

const initialRetryState: RetryState = {
  inCooldown: false,
  maxRetriesExceeded: false,
  nextRetryAt: null,
  attempts: 0,
};

const initialState: SessionState = {
  sessionId: null,
  profileId: null,
  actualViewport: null,
  initialRestoredUrl: null,
  isValidated: false,
  isCreating: false,
  isValidating: false,
  error: null,
  retryState: initialRetryState,
  pages: new Map(),
  activePageId: null,
  pageColorMap: new Map(),
  timelineEntries: [],
  timelineLoading: false,
  timelineTotalCount: 0,
  timelineHasMore: false,
  frameDimensions: null,
  displayDimensions: null,
};

// ============================================================================
// Store Implementation
// ============================================================================

export const useSessionStore = create<SessionStore>()(
  subscribeWithSelector((set, get) => ({
    ...initialState,

    // ========================================================================
    // Session Lifecycle Actions
    // ========================================================================

    setSession: (data) => {
      console.log('[sessionStore] setSession called:', {
        sessionId: data.sessionId,
        profileId: data.profileId ?? null,
        isValidated: true,
      });
      return set({
        sessionId: data.sessionId,
        profileId: data.profileId ?? null,
        actualViewport: data.actualViewport ?? null,
        initialRestoredUrl: data.initialRestoredUrl ?? null,
        isValidated: true,
        isCreating: false,
        isValidating: false,
        error: null,
      });
    },

    validateSession: async (sessionId: string) => {
      const state = get();

      console.log('[sessionStore] validateSession called:', {
        sessionId,
        currentSessionId: state.sessionId,
        currentIsValidated: state.isValidated,
      });

      // Already validated this session
      if (state.sessionId === sessionId && state.isValidated) {
        console.log('[sessionStore] Session already validated, skipping');
        return true;
      }

      set({ isValidating: true, error: null });

      try {
        const apiUrl = getApiBase();
        const res = await fetch(`${apiUrl}/recordings/live/${sessionId}/pages`);
        const isValid = res.ok;

        console.log('[sessionStore] Validation result:', { sessionId, isValid, status: res.status });

        if (isValid) {
          set({
            sessionId,
            isValidated: true,
            isValidating: false,
            error: null,
          });
        } else {
          set({
            isValidated: false,
            isValidating: false,
            // Don't set error here - caller may want to create a new session
          });
        }

        return isValid;
      } catch (err) {
        console.error('[sessionStore] Session validation failed:', err);
        set({
          isValidated: false,
          isValidating: false,
        });
        return false;
      }
    },

    clearSession: () => {
      console.log('[sessionStore] clearSession called - resetting to initial state');
      return set(initialState);
    },

    setIsCreating: (isCreating) => set({ isCreating }),

    setIsValidating: (isValidating) => set({ isValidating }),

    setError: (error) => set({ error }),

    setRetryState: (retryState) =>
      set((s) => ({
        retryState: { ...s.retryState, ...retryState },
      })),

    setSessionId: (sessionId) =>
      set({
        sessionId,
        isValidated: false, // Need to validate when setting from URL
      }),

    markValidated: () => set({ isValidated: true }),

    // ========================================================================
    // Pages/Tabs Actions
    // ========================================================================

    setPages: (pages) => set({ pages }),

    addPage: (page) =>
      set((s) => {
        const next = new Map(s.pages);
        next.set(page.id, page);
        return { pages: next };
      }),

    updatePage: (pageId, updates) =>
      set((s) => {
        const existing = s.pages.get(pageId);
        if (!existing) return {};

        const next = new Map(s.pages);
        next.set(pageId, { ...existing, ...updates });
        return { pages: next };
      }),

    removePage: (pageId) =>
      set((s) => {
        const next = new Map(s.pages);
        next.delete(pageId);
        return { pages: next };
      }),

    setActivePageId: (activePageId) => set({ activePageId }),

    updatePageColorMap: (pages) => {
      const defaultColor = PAGE_COLORS[0] ?? 'bg-gray-500';
      const map = new Map<string, PageColor>();

      // Sort by creation time for consistent color assignment
      const sortedPages = [...pages].sort(
        (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      );

      sortedPages.forEach((page, index) => {
        const color = PAGE_COLORS[index % PAGE_COLORS.length];
        map.set(page.id, color ?? defaultColor);
      });

      set({ pageColorMap: map });
    },

    // ========================================================================
    // Timeline Actions
    // ========================================================================

    setTimelineEntries: (timelineEntries) => set({ timelineEntries }),

    addTimelineEntry: (entry) =>
      set((s) => {
        // Check if entry already exists
        if (s.timelineEntries.some((e) => e.id === entry.id)) {
          return {};
        }

        const updated = [...s.timelineEntries, entry];
        // Sort by timestamp
        updated.sort(
          (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
        );

        return {
          timelineEntries: updated,
          timelineTotalCount: s.timelineTotalCount + 1,
        };
      }),

    setTimelineLoading: (timelineLoading) => set({ timelineLoading }),

    setTimelineMeta: (meta) =>
      set((s) => ({
        timelineTotalCount: meta.totalCount ?? s.timelineTotalCount,
        timelineHasMore: meta.hasMore ?? s.timelineHasMore,
      })),

    clearTimeline: () =>
      set({
        timelineEntries: [],
        timelineLoading: false,
        timelineTotalCount: 0,
        timelineHasMore: false,
      }),

    // ========================================================================
    // Display Dimensions Actions
    // ========================================================================

    setFrameDimensions: (frameDimensions) => set({ frameDimensions }),

    setDisplayDimensions: (displayDimensions) => set({ displayDimensions }),
  }))
);

// ============================================================================
// Selector Hooks
// ============================================================================

/** Get current session ID */
export const useSessionId = () => useSessionStore((s) => s.sessionId);

/** Check if session is validated (confirmed to exist on server) */
export const useIsSessionValidated = () => useSessionStore((s) => s.isValidated);

/** Check if session is being created */
export const useIsCreatingSession = () => useSessionStore((s) => s.isCreating);

/** Check if session is being validated */
export const useIsValidatingSession = () => useSessionStore((s) => s.isValidating);

/** Get session error */
export const useSessionError = () => useSessionStore((s) => s.error);

/** Get retry state for UI feedback */
export const useRetryState = () => useSessionStore((s) => s.retryState);

/** Get actual viewport from session */
export const useActualViewport = () => useSessionStore((s) => s.actualViewport);

/** Get pages map */
export const usePages = () => useSessionStore((s) => s.pages);

/** Get pages as sorted array */
export const usePageList = () =>
  useSessionStore((s) => {
    const list = Array.from(s.pages.values());
    list.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
    return list;
  });

/** Get open (not closed) pages */
export const useOpenPages = () =>
  useSessionStore((s) => {
    const list = Array.from(s.pages.values()).filter((p) => p.status === 'active');
    list.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
    return list;
  });

/** Get active page ID */
export const useActivePageId = () => useSessionStore((s) => s.activePageId);

/** Get active page object */
export const useActivePage = () =>
  useSessionStore((s) => (s.activePageId ? s.pages.get(s.activePageId) ?? null : null));

/** Get page color map */
export const usePageColorMap = () => useSessionStore((s) => s.pageColorMap);

/** Get timeline entries */
export const useTimelineEntries = () => useSessionStore((s) => s.timelineEntries);

/** Check if timeline is loading */
export const useTimelineLoading = () => useSessionStore((s) => s.timelineLoading);

/** Get frame dimensions */
export const useFrameDimensions = () => useSessionStore((s) => s.frameDimensions);

/** Get display dimensions */
export const useDisplayDimensions = () => useSessionStore((s) => s.displayDimensions);

/** Get combined session/validation state for components that need both */
export const useSessionState = () =>
  useSessionStore((s) => ({
    sessionId: s.sessionId,
    isValidated: s.isValidated,
    isCreating: s.isCreating,
    isValidating: s.isValidating,
    error: s.error,
  }));
