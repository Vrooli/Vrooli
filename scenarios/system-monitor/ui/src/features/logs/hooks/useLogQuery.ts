import { useCallback, useEffect, useReducer, useRef } from 'react';
import { extractErrorMessage } from '../../../shared/api/apiFetch';
import { fetchLogs } from '../api';
import type { LogQueryFilters } from '../types';
import type { TimeRange } from '../../../shared/time/TimeRangeContext';
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
}

export function useLogQuery(range?: Pick<TimeRange, 'since' | 'until'>): UseLogQueryResult {
  const initialQueryState = range
    ? {
        ...initialState,
        filters: { ...initialState.filters, since: range.since, until: range.until },
      }
    : initialState;
  const [state, dispatch] = useReducer(logQueryReducer, initialQueryState);
  const abortRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    dispatch({ type: 'fetch-start' });
    try {
      const resp = await fetchLogs({
        filters: state.filters,
        cursor: state.currentCursor,
        direction: state.currentCursor ? 'forward' : 'reverse',
        signal: controller.signal,
      });
      if (controller.signal.aborted) return;
      dispatch({
        type: 'fetch-success',
        entries: resp.entries ?? [],
        nextCursor: resp.nextCursor,
        available: resp.available,
        reason: resp.reason,
      });
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      dispatch({ type: 'fetch-error', error: extractErrorMessage(err, 'Failed to load logs') });
    }
  }, [state.filters, state.currentCursor]);

  useEffect(() => {
    void refresh();
    return () => abortRef.current?.abort();
  }, [refresh]);

  useEffect(() => {
    if (!range || (state.filters.since === range.since && state.filters.until === range.until)) return;
    dispatch({ type: 'set-filter', patch: { since: range.since, until: range.until } });
  }, [range?.since, range?.until, state.filters.since, state.filters.until]);

  const setFilter = useCallback((patch: Partial<LogQueryFilters>) => {
    dispatch({ type: 'set-filter', patch });
  }, []);
  const resetFilters = useCallback(() => {
    dispatch({ type: 'reset-filters' });
    if (range) {
      dispatch({ type: 'set-filter', patch: { since: range.since, until: range.until } });
    }
  }, [range]);
  const nextPage = useCallback(() => { dispatch({ type: 'next-page' }); }, []);
  const prevPage = useCallback(() => { dispatch({ type: 'prev-page' }); }, []);

  return { ...state, setFilter, resetFilters, nextPage, prevPage, refresh };
}

// Re-export reducer surface so existing consumers (and tests) can import
// from either location.
export { initialState, logQueryReducer } from './logQueryReducer';
export type { LogQueryAction, LogQueryState } from './logQueryReducer';
