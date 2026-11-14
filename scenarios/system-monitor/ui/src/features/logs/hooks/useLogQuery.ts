import { useCallback, useEffect, useReducer, useRef } from 'react';
import { extractErrorMessage } from '../../../shared/api/apiFetch';
import { fetchLogs } from '../api';
import type { LogQueryFilters } from '../types';
import type { TimeRange } from '../../../shared/time/TimeRangeContext';
import { isAbortError } from '../../../shared/utils/errors';
import {
  initialState,
  logQueryReducer,
  type LogQueryState,
} from './logQueryReducer';

/**
 * useLogQuery — manages filter state plus a cursor stack for paginated
 * journald queries. The pure reducer lives in logQueryReducer.ts and is
 * tested independently.
 *
 * Cursor stack semantics:
 *   - `next-page` pushes the current page's nextCursor onto the stack and
 *     refetches with that cursor.
 *   - `prev-page` pops; an empty stack means "head" (no cursor).
 *   - Filter changes always reset the stack.
 */
export interface UseLogQueryResult extends LogQueryState {
  setFilter: (patch: Partial<LogQueryFilters>) => void;
  resetFilters: () => void;
  nextPage: () => void;
  prevPage: () => void;
  refresh: () => Promise<void>;
  refreshBackground: () => Promise<void>;
  cancel: () => void;
  cancelBackground: () => void;
  returnToLatest: () => Promise<boolean>;
  restoreLoaded: () => void;
  retryPreviousQuery: () => void;
  retryFailed: () => void;
}

export function useLogQuery(range?: Pick<TimeRange, 'since' | 'until'>, restored?: LogQueryState): UseLogQueryResult {
  const initialQueryState = restored ? { ...restored, isLoading: false } : range
    ? {
        ...initialState,
        filters: { ...initialState.filters, since: range.since, until: range.until },
      }
    : initialState;
  const [state, dispatch] = useReducer(logQueryReducer, initialQueryState);
  const abortRef = useRef<AbortController | null>(null);
  const requestKind = useRef<'foreground' | 'background'>('foreground');
  const requestActive = useRef(false);
  const recoveryRequired = useRef(false);
  recoveryRequired.current = !!state.error || !state.available;
  const previousRange = useRef({ since: range?.since, until: range?.until });
  const skipCursorRefresh = useRef(false);
  const skipRestoredRefresh = useRef(!!restored && !restored.isLoading);
  const rangeSince = range?.since;
  const rangeUntil = range?.until;

  const fetchCurrent = useCallback(async (kind: 'foreground' | 'background') => {
    if (kind === 'background' && recoveryRequired.current) return;
    if (kind === 'background' && requestActive.current && requestKind.current === 'foreground') return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    requestKind.current = kind;
    requestActive.current = true;
    dispatch({ type: 'fetch-start' });
    try {
      const resp = await fetchLogs({
        filters: state.filters,
        cursor: state.currentCursor,
        direction: 'reverse',
        signal: controller.signal,
      });
      if (controller.signal.aborted) return;
      if (resp.error) {
        recoveryRequired.current = true;
        dispatch({ type: 'fetch-error', error: resp.error });
        return;
      }
      if (!resp.available) recoveryRequired.current = true;
      dispatch({
        type: 'fetch-success',
        entries: resp.entries ?? [],
        nextCursor: resp.nextCursor,
        available: resp.available,
        reason: resp.reason,
      });
    } catch (err) {
      if (controller.signal.aborted) return;
      if (isAbortError(err)) return;
      recoveryRequired.current = true;
      dispatch({ type: 'fetch-error', error: extractErrorMessage(err, 'Failed to load logs') });
    } finally {
      if (abortRef.current === controller) requestActive.current = false;
    }
  }, [state.filters, state.currentCursor]);
  const refresh = useCallback(() => fetchCurrent('foreground'), [fetchCurrent]);
  const refreshBackground = useCallback(() => fetchCurrent('background'), [fetchCurrent]);

  useEffect(() => {
    if (skipRestoredRefresh.current) {
      skipRestoredRefresh.current = false;
      return () => abortRef.current?.abort();
    }
    if (skipCursorRefresh.current) {
      skipCursorRefresh.current = false;
      return;
    }
    void refresh();
    return () => abortRef.current?.abort();
  }, [refresh]);

  useEffect(() => {
    if (rangeSince === undefined || rangeUntil === undefined || (previousRange.current.since === rangeSince && previousRange.current.until === rangeUntil)) return;
    previousRange.current = { since: rangeSince, until: rangeUntil };
    dispatch({ type: 'set-filter', patch: { since: rangeSince, until: rangeUntil } });
  }, [rangeSince, rangeUntil]);

  const setFilter = useCallback((patch: Partial<LogQueryFilters>) => {
    abortRef.current?.abort();
    dispatch({ type: 'set-filter', patch });
  }, []);
  const resetFilters = useCallback(() => {
    abortRef.current?.abort();
    dispatch({ type: 'reset-filters' });
    if (range) {
      dispatch({ type: 'set-filter', patch: { since: range.since, until: range.until } });
    }
  }, [range]);
  const loadPage = useCallback(async (cursorStack: string[]) => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    requestKind.current = 'foreground';
    requestActive.current = true;
    const currentCursor = cursorStack[cursorStack.length - 1];
    dispatch({ type: 'fetch-start' });
    try {
      const resp = await fetchLogs({ filters: state.filters, cursor: currentCursor, direction: 'reverse', signal: controller.signal });
      if (controller.signal.aborted) return false;
      if (resp.error) {
        recoveryRequired.current = true;
        dispatch({ type: 'fetch-error', error: resp.error, retryCursorStack: cursorStack });
        return false;
      }
      if (!resp.available) {
        recoveryRequired.current = true;
        dispatch({ type: 'fetch-error', error: resp.reason || 'Log source unavailable', retryCursorStack: cursorStack });
        return false;
      }
      skipCursorRefresh.current = currentCursor !== state.currentCursor;
      dispatch({ type: 'page-success', entries: resp.entries ?? [], nextCursor: resp.nextCursor, cursorStack, currentCursor });
      return true;
    } catch (err) {
      if (controller.signal.aborted) return false;
      recoveryRequired.current = true;
      dispatch({ type: 'fetch-error', error: extractErrorMessage(err, 'Failed to load logs'), retryCursorStack: cursorStack });
      return false;
    } finally {
      if (abortRef.current === controller) requestActive.current = false;
    }
  }, [state.filters, state.currentCursor]);
  const nextPage = useCallback(() => {
    if (state.nextCursor && !state.isLoading) void loadPage([...state.cursorStack, state.nextCursor]);
  }, [loadPage, state.cursorStack, state.nextCursor, state.isLoading]);
  const prevPage = useCallback(() => {
    if (state.cursorStack.length && !state.isLoading) void loadPage(state.cursorStack.slice(0, -1));
  }, [loadPage, state.cursorStack, state.isLoading]);
  const retryFailed = useCallback(() => {
    if (state.retryCursorStack !== undefined) void loadPage(state.retryCursorStack);
    else void refresh();
  }, [loadPage, refresh, state.retryCursorStack]);

  const cancel = useCallback(() => {
    abortRef.current?.abort();
    requestActive.current = false;
    dispatch({ type: 'fetch-cancel' });
  }, []);
  const cancelBackground = useCallback(() => {
    if (requestActive.current && requestKind.current === 'background' && abortRef.current && !abortRef.current.signal.aborted) {
      abortRef.current.abort();
      requestActive.current = false;
      dispatch({ type: 'fetch-cancel' });
    }
  }, []);

  const returnToLatest = useCallback(() => loadPage([]), [loadPage]);
  const restoreLoaded = useCallback(() => {
    if (!state.loadedPage) return;
    abortRef.current?.abort();
    skipCursorRefresh.current = state.filters !== state.loadedPage.filters || state.currentCursor !== state.loadedPage.currentCursor;
    dispatch({ type: 'restore-loaded' });
  }, [state.loadedPage, state.filters, state.currentCursor]);
  const retryPreviousQuery = useCallback(() => {
    if (state.retryFilters) {
      abortRef.current?.abort();
      dispatch({ type: 'set-filter', patch: state.retryFilters });
    }
  }, [state.retryFilters]);
  return { ...state, setFilter, resetFilters, nextPage, prevPage, refresh, refreshBackground, cancel, cancelBackground, returnToLatest, restoreLoaded, retryPreviousQuery, retryFailed };
}

// Re-export reducer surface so existing consumers (and tests) can import
// from either location.
export { initialState, logQueryReducer } from './logQueryReducer';
export type { LogQueryAction, LogQueryState } from './logQueryReducer';
