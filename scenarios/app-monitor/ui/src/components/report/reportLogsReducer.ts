/**
 * Reducer for managing app logs state in the report dialog
 */

import type { ReportAppLogStream } from './reportTypes';
import { createFetchReducer } from './createFetchReducer';

export interface ReportLogsState {
  logs: string[];
  logsTotal: number | null;
  loading: boolean;
  error: string | null;
  expanded: boolean;
  fetchedAt: number | null;
  streams: ReportAppLogStream[];
  selections: Record<string, boolean>;
  include: boolean;
}

export type ReportLogsAction =
  | { type: 'FETCH_START' }
  | {
      type: 'FETCH_SUCCESS';
      payload: {
        logs: string[];
        logsTotal: number;
        streams: ReportAppLogStream[];
        selections: Record<string, boolean>;
        fetchedAt: number;
      };
    }
  | { type: 'FETCH_ERROR'; payload: string }
  | { type: 'SET_EXPANDED'; payload: boolean }
  | { type: 'TOGGLE_STREAM'; payload: { key: string; checked: boolean } }
  | { type: 'SET_INCLUDE'; payload: boolean }
  | { type: 'RESET' };

interface LogsData {
  logs: string[];
  streams: ReportAppLogStream[];
  selections: Record<string, boolean>;
}

const { reducer: baseReducer, initialState: baseInitialState } = createFetchReducer<LogsData>({
  logs: [],
  streams: [],
  selections: {},
});

export const initialReportLogsState: ReportLogsState = {
  logs: baseInitialState.data.logs,
  logsTotal: baseInitialState.total,
  loading: baseInitialState.loading,
  error: baseInitialState.error,
  expanded: baseInitialState.expanded,
  fetchedAt: baseInitialState.fetchedAt,
  streams: baseInitialState.data.streams,
  selections: baseInitialState.data.selections,
  include: baseInitialState.include,
};

export function reportLogsReducer(
  state: ReportLogsState,
  action: ReportLogsAction,
): ReportLogsState {
  // Handle TOGGLE_STREAM action separately since it's specific to logs
  if (action.type === 'TOGGLE_STREAM') {
    return {
      ...state,
      selections: {
        ...state.selections,
        [action.payload.key]: action.payload.checked,
      },
    };
  }

  const baseState = {
    data: {
      logs: state.logs,
      streams: state.streams,
      selections: state.selections,
    },
    total: state.logsTotal,
    loading: state.loading,
    error: state.error,
    expanded: state.expanded,
    fetchedAt: state.fetchedAt,
    include: state.include,
  };

  const baseAction = action.type === 'FETCH_SUCCESS'
    ? {
        ...action,
        payload: {
          data: {
            logs: action.payload.logs,
            streams: action.payload.streams,
            selections: action.payload.selections,
          },
          total: action.payload.logsTotal,
          fetchedAt: action.payload.fetchedAt,
        },
      }
    : action;

  const nextBaseState = baseReducer(baseState, baseAction as never);

  return {
    logs: nextBaseState.data.logs,
    logsTotal: nextBaseState.total,
    loading: nextBaseState.loading,
    error: nextBaseState.error,
    expanded: nextBaseState.expanded,
    fetchedAt: nextBaseState.fetchedAt,
    streams: nextBaseState.data.streams,
    selections: nextBaseState.data.selections,
    include: nextBaseState.include,
  };
}
