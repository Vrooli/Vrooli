/**
 * Pure reducer + types for the logs query state machine.
 *
 * Extracted from useLogQuery so the reducer can be unit-tested without
 * pulling in the runtime import graph (apiFetch → @vrooli/api-base, etc.).
 */
import {
  DEFAULT_LIMIT,
  type LogEntry,
  type LogQueryFilters,
  MAX_LIMIT,
  emptyFilters,
} from '../types';

export interface LogQueryState {
  filters: LogQueryFilters;
  cursorStack: string[];
  currentCursor?: string;
  nextCursor?: string;
  entries: LogEntry[];
  isLoading: boolean;
  error: string | null;
  available: boolean;
  reason?: string;
  loadedPage?: { filters: LogQueryFilters; cursorStack: string[]; currentCursor?: string; nextCursor?: string };
  retryFilters?: LogQueryFilters;
  retryCursorStack?: string[];
}

export const initialState: LogQueryState = {
  filters: emptyFilters,
  cursorStack: [],
  entries: [],
  isLoading: false,
  error: null,
  available: true,
};

export type LogQueryAction =
  | { type: 'set-filter'; patch: Partial<LogQueryFilters> }
  | { type: 'reset-filters' }
  | { type: 'fetch-start' }
  | { type: 'fetch-cancel' }
  | {
      type: 'fetch-success';
      entries: LogEntry[];
      nextCursor?: string;
      available: boolean;
      reason?: string;
    }
  | { type: 'fetch-error'; error: string; retryCursorStack?: string[] }
  | { type: 'restore-loaded' }
  | { type: 'page-success'; entries: LogEntry[]; nextCursor?: string; cursorStack: string[]; currentCursor?: string }
  | { type: 'next-page' }
  | { type: 'prev-page' };

const clampLimit = (n: number): number => {
  if (!Number.isFinite(n) || n <= 0) return DEFAULT_LIMIT;
  return Math.min(Math.max(1, Math.floor(n)), MAX_LIMIT);
};

export function logQueryReducer(state: LogQueryState, action: LogQueryAction): LogQueryState {
  switch (action.type) {
    case 'set-filter': {
      const merged: LogQueryFilters = { ...state.filters, ...action.patch };
      if ('limit' in action.patch) merged.limit = clampLimit(merged.limit);
      return {
        ...state,
        filters: merged,
        cursorStack: [],
        currentCursor: undefined,
        nextCursor: undefined,
        retryCursorStack: undefined,
      };
    }
    case 'reset-filters':
      return {
        ...state,
        filters: emptyFilters,
        cursorStack: [],
        currentCursor: undefined,
        nextCursor: undefined,
        retryCursorStack: undefined,
      };
    case 'fetch-start':
      return { ...state, isLoading: true, error: null };
    case 'fetch-cancel':
      return { ...state, isLoading: false };
    case 'fetch-success':
      if (!action.available) {
        return { ...state, isLoading: false, available: false, reason: action.reason };
      }
      return {
        ...state,
        isLoading: false,
        error: null,
        entries: action.entries,
        nextCursor: action.nextCursor,
        available: action.available,
        reason: action.reason,
        loadedPage: { filters: state.filters, cursorStack: state.cursorStack, currentCursor: state.currentCursor, nextCursor: action.nextCursor },
        retryFilters: undefined,
        retryCursorStack: undefined,
      };
    case 'fetch-error':
      return { ...state, isLoading: false, error: action.error, available: true, reason: undefined, retryCursorStack: action.retryCursorStack };
    case 'restore-loaded':
      if (!state.loadedPage) return state;
      return {
        ...state, filters: state.loadedPage.filters, cursorStack: state.loadedPage.cursorStack,
        currentCursor: state.loadedPage.currentCursor, nextCursor: state.loadedPage.nextCursor,
        retryFilters: state.filters !== state.loadedPage.filters ? state.filters : state.retryFilters,
        error: null, available: true, reason: undefined, isLoading: false, retryCursorStack: undefined,
      };
    case 'page-success':
      return {
        ...state, isLoading: false, error: null, available: true, reason: undefined,
        entries: action.entries, nextCursor: action.nextCursor,
        cursorStack: action.cursorStack, currentCursor: action.currentCursor,
        loadedPage: { filters: state.filters, cursorStack: action.cursorStack, currentCursor: action.currentCursor, nextCursor: action.nextCursor },
        retryFilters: undefined,
        retryCursorStack: undefined,
      };
    case 'next-page': {
      if (!state.nextCursor) return state;
      return {
        ...state,
        cursorStack: [...state.cursorStack, state.nextCursor],
        currentCursor: state.nextCursor,
      };
    }
    case 'prev-page': {
      if (state.cursorStack.length === 0) return state;
      const popped = state.cursorStack.slice(0, -1);
      return {
        ...state,
        cursorStack: popped,
        currentCursor: popped[popped.length - 1],
      };
    }
    default:
      return state;
  }
}
