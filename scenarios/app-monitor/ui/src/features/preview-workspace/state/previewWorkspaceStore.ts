import { create } from 'zustand';
import { reconcileTrackFractions, resolveWorkspaceLayout } from '../utils/layout';

export type PreviewWorkspaceLayout = 'grid' | 'split';
export type PreviewWorkspaceInteractionMode = 'browse' | 'arrange';

export interface PreviewWorkspacePane {
  id: string;
  appId: string | null;
  createdAt: number;
}

export interface PreviewWorkspaceState {
  layout: PreviewWorkspaceLayout;
  interactionMode: PreviewWorkspaceInteractionMode;
  panes: PreviewWorkspacePane[];
  focusedPaneId: string | null;
  columnFractions: number[];
  rowFractions: number[];
  addPane: (appId?: string | null) => string;
  removePane: (paneId: string) => void;
  movePaneToIndex: (paneId: string, targetIndex: number) => void;
  setPaneApp: (paneId: string, appId: string | null) => void;
  focusPane: (paneId: string | null) => void;
  setLayout: (layout: PreviewWorkspaceLayout) => void;
  setInteractionMode: (mode: PreviewWorkspaceInteractionMode) => void;
  setColumnFractions: (fractions: number[]) => void;
  setRowFractions: (fractions: number[]) => void;
  reset: () => void;
}

const MIN_PANES = 1;
const MAX_PANES = 6;

const createPaneId = (): string => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `pane-${Date.now()}-${Math.random().toString(16).slice(2)}`;
};

const createPane = (appId: string | null = null): PreviewWorkspacePane => ({
  id: createPaneId(),
  appId,
  createdAt: Date.now(),
});

const buildInitialState = (): Pick<PreviewWorkspaceState, 'layout' | 'interactionMode' | 'panes' | 'focusedPaneId' | 'columnFractions' | 'rowFractions'> => {
  const firstPane = createPane(null);
  return {
    layout: 'grid',
    interactionMode: 'browse',
    panes: [firstPane],
    focusedPaneId: firstPane.id,
    columnFractions: [1],
    rowFractions: [1],
  };
};

const normalizePaneApp = (value: string | null | undefined): string | null => {
  if (!value) {
    return null;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
};

const reconcileFractionsForWorkspace = (
  layout: PreviewWorkspaceLayout,
  panes: PreviewWorkspacePane[],
  columnFractions: number[],
  rowFractions: number[],
): Pick<PreviewWorkspaceState, 'columnFractions' | 'rowFractions'> => {
  const layoutDescriptor = resolveWorkspaceLayout(layout, panes.length);
  return {
    columnFractions: reconcileTrackFractions(columnFractions, layoutDescriptor.columns),
    rowFractions: reconcileTrackFractions(rowFractions, layoutDescriptor.rows),
  };
};

export const usePreviewWorkspaceStore = create<PreviewWorkspaceState>((set, get) => ({
  ...buildInitialState(),

  addPane: (appId) => {
    const nextPane = createPane(normalizePaneApp(appId));
    set((state) => {
      if (state.panes.length >= MAX_PANES) {
        return state;
      }

      const panes = [...state.panes, nextPane];
      return {
        panes,
        focusedPaneId: nextPane.id,
        ...reconcileFractionsForWorkspace(state.layout, panes, state.columnFractions, state.rowFractions),
      };
    });
    return nextPane.id;
  },

  removePane: (paneId) => set((state) => {
    if (state.panes.length <= MIN_PANES) {
      return state;
    }

    const nextPanes = state.panes.filter((pane) => pane.id !== paneId);
    if (nextPanes.length === state.panes.length) {
      return state;
    }

    const fallbackFocusedPane = nextPanes[nextPanes.length - 1] ?? null;
    return {
      panes: nextPanes,
      focusedPaneId: state.focusedPaneId === paneId ? fallbackFocusedPane?.id ?? null : state.focusedPaneId,
      ...reconcileFractionsForWorkspace(state.layout, nextPanes, state.columnFractions, state.rowFractions),
    };
  }),

  movePaneToIndex: (paneId, targetIndex) => set((state) => {
    if (state.panes.length <= 1) {
      return state;
    }

    const fromIndex = state.panes.findIndex((pane) => pane.id === paneId);
    if (fromIndex < 0) {
      return state;
    }

    const boundedTargetIndex = Math.max(0, Math.min(state.panes.length - 1, Math.floor(targetIndex)));
    if (fromIndex === boundedTargetIndex) {
      return state;
    }

    const reordered = [...state.panes];
    const [movedPane] = reordered.splice(fromIndex, 1);
    if (!movedPane) {
      return state;
    }

    reordered.splice(boundedTargetIndex, 0, movedPane);
    return {
      panes: reordered,
      focusedPaneId: paneId,
    };
  }),

  setPaneApp: (paneId, appId) => set((state) => ({
    panes: state.panes.map((pane) => (
      pane.id === paneId
        ? { ...pane, appId: normalizePaneApp(appId) }
        : pane
    )),
  })),

  focusPane: (paneId) => set((state) => {
    if (!paneId) {
      return { focusedPaneId: null };
    }
    const exists = state.panes.some((pane) => pane.id === paneId);
    return exists ? { focusedPaneId: paneId } : state;
  }),

  setLayout: (layout) => {
    const state = get();
    const nextFractions = reconcileFractionsForWorkspace(
      layout,
      state.panes,
      state.columnFractions,
      state.rowFractions,
    );
    set({ layout, ...nextFractions });
  },

  setInteractionMode: (interactionMode) => set({ interactionMode }),

  setColumnFractions: (fractions) => set((state) => {
    const { columns } = resolveWorkspaceLayout(state.layout, state.panes.length);
    return { columnFractions: reconcileTrackFractions(fractions, columns) };
  }),

  setRowFractions: (fractions) => set((state) => {
    const { rows } = resolveWorkspaceLayout(state.layout, state.panes.length);
    return { rowFractions: reconcileTrackFractions(fractions, rows) };
  }),

  reset: () => set(buildInitialState()),
}));

export const previewWorkspaceLimits = {
  minPanes: MIN_PANES,
  maxPanes: MAX_PANES,
};
