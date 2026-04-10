import { isSameNormalizedUrl } from './previewNavigationPlanner';

export interface PreviewNavigationState {
  previewUrl: string | null;
  previewUrlInput: string;
  hasCustomPreviewUrl: boolean;
  history: string[];
  historyIndex: number;
  initialPreviewUrl: string | null;
}

export const previewNavigationActions = {
  reset: (force?: boolean) => ({
    type: 'reset' as const,
    force,
  }),
  setInput: (value: string) => ({
    type: 'set-input' as const,
    value,
  }),
  markDefaultCleared: () => ({
    type: 'mark-default-cleared' as const,
  }),
  applyDefaultUrl: (url: string) => ({
    type: 'apply-default-url' as const,
    url,
  }),
  applyLocalNavigation: (url: string) => ({
    type: 'apply-local-navigation' as const,
    url,
  }),
  travelHistory: (direction: 'back' | 'forward') => ({
    type: 'travel-history' as const,
    direction,
  }),
  syncFromBridge: (href: string) => ({
    type: 'sync-from-bridge' as const,
    href,
  }),
};

export type PreviewNavigationAction = ReturnType<
  (typeof previewNavigationActions)[keyof typeof previewNavigationActions]
>;

const clampHistoryIndex = (history: string[], historyIndex: number): number => {
  if (!Number.isFinite(historyIndex)) {
    return history.length - 1;
  }
  return Math.max(-1, Math.min(history.length - 1, Math.floor(historyIndex)));
};

const pushUniqueHistoryEntry = (state: PreviewNavigationState, url: string): PreviewNavigationState => {
  const baseHistory = state.historyIndex >= 0
    ? state.history.slice(0, state.historyIndex + 1)
    : [];
  const last = baseHistory[baseHistory.length - 1];
  const nextHistory = last === url ? baseHistory : [...baseHistory, url];
  return {
    ...state,
    history: nextHistory,
    historyIndex: nextHistory.length - 1,
  };
};

export const reducePreviewNavigationState = (
  input: PreviewNavigationState,
  action: PreviewNavigationAction,
): PreviewNavigationState => {
  const state: PreviewNavigationState = {
    ...input,
    history: [...input.history],
    historyIndex: clampHistoryIndex(input.history, input.historyIndex),
  };

  switch (action.type) {
    case 'reset': {
      if (!action.force && state.hasCustomPreviewUrl) {
        return state;
      }
      return {
        ...state,
        previewUrl: null,
        previewUrlInput: '',
        history: [],
        historyIndex: -1,
        initialPreviewUrl: null,
      };
    }

    case 'set-input': {
      return {
        ...state,
        previewUrlInput: action.value,
      };
    }

    case 'mark-default-cleared': {
      return {
        ...state,
        hasCustomPreviewUrl: false,
      };
    }

    case 'apply-default-url': {
      const withHistory = pushUniqueHistoryEntry(state, action.url);
      return {
        ...withHistory,
        previewUrl: action.url,
        previewUrlInput: action.url,
        hasCustomPreviewUrl: false,
        initialPreviewUrl: action.url,
      };
    }

    case 'apply-local-navigation': {
      const withHistory = pushUniqueHistoryEntry(state, action.url);
      return {
        ...withHistory,
        previewUrl: action.url,
        hasCustomPreviewUrl: true,
        initialPreviewUrl: action.url,
      };
    }

    case 'travel-history': {
      if (action.direction === 'back') {
        if (state.historyIndex <= 0) {
          return state;
        }
        const targetIndex = state.historyIndex - 1;
        const targetUrl = state.history[targetIndex] ?? null;
        return {
          ...state,
          historyIndex: targetIndex,
          previewUrl: targetUrl,
          previewUrlInput: targetUrl ?? '',
          hasCustomPreviewUrl: true,
        };
      }

      if (state.historyIndex === -1 || state.historyIndex >= state.history.length - 1) {
        return state;
      }
      const targetIndex = state.historyIndex + 1;
      const targetUrl = state.history[targetIndex] ?? null;
      return {
        ...state,
        historyIndex: targetIndex,
        previewUrl: targetUrl,
        previewUrlInput: targetUrl ?? '',
        hasCustomPreviewUrl: true,
      };
    }

    case 'sync-from-bridge': {
      const last = state.history[state.history.length - 1];
      if (last && isSameNormalizedUrl(last, action.href)) {
        return {
          ...state,
          previewUrlInput: action.href,
          hasCustomPreviewUrl: true,
          historyIndex: state.history.length - 1,
          initialPreviewUrl: action.href,
        };
      }

      const nextHistory = [...state.history, action.href];
      return {
        ...state,
        previewUrlInput: action.href,
        hasCustomPreviewUrl: true,
        history: nextHistory,
        historyIndex: nextHistory.length - 1,
        initialPreviewUrl: action.href,
      };
    }

    default:
      return state;
  }
};
