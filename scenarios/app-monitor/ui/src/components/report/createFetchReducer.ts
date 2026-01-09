/**
 * Generic reducer factory for managing fetch state with expansion and inclusion toggles
 */

export interface BaseFetchState<TData = unknown> {
  data: TData;
  total: number | null;
  loading: boolean;
  error: string | null;
  expanded: boolean;
  fetchedAt: number | null;
  include: boolean;
}

export type FetchAction<TData = unknown> =
  | { type: 'FETCH_START' }
  | {
      type: 'FETCH_SUCCESS';
      payload: {
        data: TData;
        total: number;
        fetchedAt: number;
      };
    }
  | { type: 'FETCH_ERROR'; payload: string }
  | { type: 'SET_EXPANDED'; payload: boolean }
  | { type: 'SET_INCLUDE'; payload: boolean }
  | { type: 'RESET' };

/**
 * Creates a typed reducer for managing fetch operations with consistent patterns
 */
export function createFetchReducer<TData>(
  initialData: TData,
) {
  const initialState: BaseFetchState<TData> = {
    data: initialData,
    total: null,
    loading: false,
    error: null,
    expanded: false,
    fetchedAt: null,
    include: true,
  };

  function reducer(
    state: BaseFetchState<TData>,
    action: FetchAction<TData>,
  ): BaseFetchState<TData> {
    switch (action.type) {
      case 'FETCH_START':
        return {
          ...state,
          loading: true,
          error: null,
        };

      case 'FETCH_SUCCESS':
        return {
          ...state,
          loading: false,
          data: action.payload.data,
          total: action.payload.total,
          fetchedAt: action.payload.fetchedAt,
          error: null,
        };

      case 'FETCH_ERROR':
        return {
          ...state,
          loading: false,
          data: initialData,
          total: null,
          error: action.payload,
          fetchedAt: null,
        };

      case 'SET_EXPANDED':
        return {
          ...state,
          expanded: action.payload,
        };

      case 'SET_INCLUDE':
        return {
          ...state,
          include: action.payload,
        };

      case 'RESET':
        return initialState;

      default:
        return state;
    }
  }

  return {
    reducer,
    initialState,
  };
}
