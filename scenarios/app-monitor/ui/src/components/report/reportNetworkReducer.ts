/**
 * Reducer for managing network events state in the report dialog
 */

import type { ReportNetworkEntry } from './reportTypes';
import { createFetchReducer } from './createFetchReducer';

export interface ReportNetworkState {
  events: ReportNetworkEntry[];
  total: number | null;
  loading: boolean;
  error: string | null;
  expanded: boolean;
  fetchedAt: number | null;
  include: boolean;
}

export type ReportNetworkAction =
  | { type: 'FETCH_START' }
  | {
      type: 'FETCH_SUCCESS';
      payload: {
        events: ReportNetworkEntry[];
        total: number;
        fetchedAt: number;
      };
    }
  | { type: 'FETCH_ERROR'; payload: string }
  | { type: 'SET_EXPANDED'; payload: boolean }
  | { type: 'SET_INCLUDE'; payload: boolean }
  | { type: 'RESET' };

const { reducer: baseReducer, initialState: baseInitialState } = createFetchReducer<ReportNetworkEntry[]>([]);

export const initialReportNetworkState: ReportNetworkState = {
  events: baseInitialState.data,
  total: baseInitialState.total,
  loading: baseInitialState.loading,
  error: baseInitialState.error,
  expanded: baseInitialState.expanded,
  fetchedAt: baseInitialState.fetchedAt,
  include: baseInitialState.include,
};

export function reportNetworkReducer(
  state: ReportNetworkState,
  action: ReportNetworkAction,
): ReportNetworkState {
  const baseState = {
    data: state.events,
    total: state.total,
    loading: state.loading,
    error: state.error,
    expanded: state.expanded,
    fetchedAt: state.fetchedAt,
    include: state.include,
  };

  const baseAction = action.type === 'FETCH_SUCCESS'
    ? { ...action, payload: { data: action.payload.events, total: action.payload.total, fetchedAt: action.payload.fetchedAt } }
    : action;

  const nextBaseState = baseReducer(baseState, baseAction as never);

  return {
    events: nextBaseState.data,
    total: nextBaseState.total,
    loading: nextBaseState.loading,
    error: nextBaseState.error,
    expanded: nextBaseState.expanded,
    fetchedAt: nextBaseState.fetchedAt,
    include: nextBaseState.include,
  };
}
