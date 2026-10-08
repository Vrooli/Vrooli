import { create } from 'zustand';
import {
  createDefaultPaneViewState,
  normalizePreviewPaneViewState,
  type PreviewWorkspacePaneViewState,
} from './previewWorkspaceStore';

interface PreviewPaneRuntimeState {
  paneViewState: Record<string, PreviewWorkspacePaneViewState>;
  hydratePaneViewState: (paneId: string, state: PreviewWorkspacePaneViewState | undefined) => void;
  setPaneViewState: (paneId: string, partial: Partial<PreviewWorkspacePaneViewState>) => void;
  resetPaneViewState: (paneId: string) => void;
  removePaneViewState: (paneId: string) => void;
  replacePaneViewState: (states: Record<string, PreviewWorkspacePaneViewState>) => void;
  reset: () => void;
}

const areStringArraysEqual = (a: string[], b: string[]): boolean => {
  if (a === b) {
    return true;
  }
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
};

const isPaneViewStateEqual = (
  previous: PreviewWorkspacePaneViewState,
  next: PreviewWorkspacePaneViewState,
): boolean => (
  previous.previewUrl === next.previewUrl
  && previous.previewUrlInput === next.previewUrlInput
  && previous.hasCustomPreviewUrl === next.hasCustomPreviewUrl
  && previous.historyIndex === next.historyIndex
  && previous.initialPreviewUrl === next.initialPreviewUrl
  && previous.isLogsVisible === next.isLogsVisible
  && previous.isFullView === next.isFullView
  && areStringArraysEqual(previous.history, next.history)
);

export const usePreviewPaneRuntimeStore = create<PreviewPaneRuntimeState>()((set) => ({
  paneViewState: {},

  hydratePaneViewState: (paneId, state) => set((current) => {
    if (!state || current.paneViewState[paneId]) {
      return current;
    }
    return {
      paneViewState: {
        ...current.paneViewState,
        [paneId]: normalizePreviewPaneViewState(state),
      },
    };
  }),

  setPaneViewState: (paneId, partial) => set((current) => {
    const previous = current.paneViewState[paneId] ?? createDefaultPaneViewState();
    const merged = normalizePreviewPaneViewState({
      ...previous,
      ...partial,
      history: Array.isArray(partial.history)
        ? partial.history
        : previous.history,
    });

    if (isPaneViewStateEqual(previous, merged)) {
      return current;
    }

    return {
      paneViewState: {
        ...current.paneViewState,
        [paneId]: merged,
      },
    };
  }),

  resetPaneViewState: (paneId) => set((current) => ({
    paneViewState: {
      ...current.paneViewState,
      [paneId]: createDefaultPaneViewState(),
    },
  })),

  removePaneViewState: (paneId) => set((current) => {
    if (!current.paneViewState[paneId]) {
      return current;
    }
    const paneViewState = { ...current.paneViewState };
    delete paneViewState[paneId];
    return { paneViewState };
  }),

  replacePaneViewState: (states) => set({
    paneViewState: Object.fromEntries(
      Object.entries(states).map(([paneId, state]) => [
        paneId,
        normalizePreviewPaneViewState(state),
      ]),
    ),
  }),

  reset: () => set({ paneViewState: {} }),
}));
