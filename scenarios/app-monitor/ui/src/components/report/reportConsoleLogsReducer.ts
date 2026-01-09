/**
 * Reducer for managing console logs state in the report dialog
 */

import type { ReportConsoleEntry } from './reportTypes';
import { createFetchReducer } from './createFetchReducer';

export interface ReportConsoleLogsState {
  entries: ReportConsoleEntry[];
  total: number | null;
  loading: boolean;
  error: string | null;
  expanded: boolean;
  fetchedAt: number | null;
  include: boolean;
}

export type ReportConsoleLogsAction =
  | { type: 'FETCH_START' }
  | {
      type: 'FETCH_SUCCESS';
      payload: {
        entries: ReportConsoleEntry[];
        total: number;
        fetchedAt: number;
      };
    }
  | { type: 'FETCH_ERROR'; payload: string }
  | { type: 'SET_EXPANDED'; payload: boolean }
  | { type: 'SET_INCLUDE'; payload: boolean }
  | { type: 'RESET' };

const { reducer: baseReducer, initialState: baseInitialState } = createFetchReducer<ReportConsoleEntry[]>([]);

export const initialReportConsoleLogsState: ReportConsoleLogsState = {
  entries: baseInitialState.data,
  total: baseInitialState.total,
  loading: baseInitialState.loading,
  error: baseInitialState.error,
  expanded: baseInitialState.expanded,
  fetchedAt: baseInitialState.fetchedAt,
  include: baseInitialState.include,
};

export function reportConsoleLogsReducer(
  state: ReportConsoleLogsState,
  action: ReportConsoleLogsAction,
): ReportConsoleLogsState {
  const baseState = {
    data: state.entries,
    total: state.total,
    loading: state.loading,
    error: state.error,
    expanded: state.expanded,
    fetchedAt: state.fetchedAt,
    include: state.include,
  };

  const baseAction = action.type === 'FETCH_SUCCESS'
    ? { ...action, payload: { data: action.payload.entries, total: action.payload.total, fetchedAt: action.payload.fetchedAt } }
    : action;

  const nextBaseState = baseReducer(baseState, baseAction as never);

  return {
    entries: nextBaseState.data,
    total: nextBaseState.total,
    loading: nextBaseState.loading,
    error: nextBaseState.error,
    expanded: nextBaseState.expanded,
    fetchedAt: nextBaseState.fetchedAt,
    include: nextBaseState.include,
  };
}
