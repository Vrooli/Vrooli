import { create } from 'zustand';
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware';
import { reconcileTrackFractions, resolveWorkspaceLayout } from '../utils/layout';

export type PreviewWorkspaceLayout = 'grid' | 'split';
export type PreviewWorkspaceInteractionMode = 'browse' | 'arrange';

export interface PreviewWorkspacePane {
  id: string;
  appId: string | null;
  createdAt: number;
}

export interface PreviewWorkspacePaneViewState {
  previewUrl: string | null;
  previewUrlInput: string;
  hasCustomPreviewUrl: boolean;
  history: string[];
  historyIndex: number;
  initialPreviewUrl: string | null;
  isLogsVisible: boolean;
}

export interface PreviewWorkspaceState {
  layout: PreviewWorkspaceLayout;
  interactionMode: PreviewWorkspaceInteractionMode;
  panes: PreviewWorkspacePane[];
  paneViewState: Record<string, PreviewWorkspacePaneViewState>;
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
  setPaneViewState: (paneId: string, partial: Partial<PreviewWorkspacePaneViewState>) => void;
  resetPaneViewState: (paneId: string) => void;
  reset: () => void;
}

const MIN_PANES = 1;
const MAX_PANES = 6;
const PREVIEW_WORKSPACE_STORAGE_KEY = 'app-monitor:preview-workspace-v1';

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

const buildInitialState = (): Pick<
  PreviewWorkspaceState,
  'layout' | 'interactionMode' | 'panes' | 'paneViewState' | 'focusedPaneId' | 'columnFractions' | 'rowFractions'
> => {
  const firstPane = createPane(null);
  return {
    layout: 'grid',
    interactionMode: 'browse',
    panes: [firstPane],
    paneViewState: {},
    focusedPaneId: firstPane.id,
    columnFractions: [1],
    rowFractions: [1],
  };
};

const noopStorage: StateStorage = {
  getItem: () => null,
  setItem: () => undefined,
  removeItem: () => undefined,
};

const previewWorkspaceStorage = createJSONStorage<Pick<
  PreviewWorkspaceState,
  'layout' | 'interactionMode' | 'panes' | 'paneViewState' | 'focusedPaneId' | 'columnFractions' | 'rowFractions'
>>(() => {
  if (typeof window !== 'undefined' && window.localStorage) {
    return window.localStorage;
  }
  return noopStorage;
});

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

const createDefaultPaneViewState = (): PreviewWorkspacePaneViewState => ({
  previewUrl: null,
  previewUrlInput: '',
  hasCustomPreviewUrl: false,
  history: [],
  historyIndex: -1,
  initialPreviewUrl: null,
  isLogsVisible: false,
});

const normalizePersistedPaneViewState = (value: unknown): PreviewWorkspacePaneViewState => {
  if (!value || typeof value !== 'object') {
    return createDefaultPaneViewState();
  }

  const record = value as Partial<PreviewWorkspacePaneViewState>;
  const history = Array.isArray(record.history)
    ? record.history.filter((entry): entry is string => typeof entry === 'string' && entry.trim().length > 0)
    : [];
  const historyIndex = typeof record.historyIndex === 'number' && Number.isFinite(record.historyIndex)
    ? Math.max(-1, Math.min(history.length - 1, Math.floor(record.historyIndex)))
    : history.length - 1;

  return {
    previewUrl: typeof record.previewUrl === 'string' && record.previewUrl.trim().length > 0
      ? record.previewUrl.trim()
      : null,
    previewUrlInput: typeof record.previewUrlInput === 'string' ? record.previewUrlInput : '',
    hasCustomPreviewUrl: Boolean(record.hasCustomPreviewUrl),
    history,
    historyIndex,
    initialPreviewUrl: typeof record.initialPreviewUrl === 'string' && record.initialPreviewUrl.trim().length > 0
      ? record.initialPreviewUrl.trim()
      : null,
    isLogsVisible: Boolean(record.isLogsVisible),
  };
};
const isLayout = (value: unknown): value is PreviewWorkspaceLayout => value === 'grid' || value === 'split';
const isInteractionMode = (value: unknown): value is PreviewWorkspaceInteractionMode => (
  value === 'browse' || value === 'arrange'
);

const normalizePersistedPane = (value: unknown): PreviewWorkspacePane | null => {
  if (!value || typeof value !== 'object') {
    return null;
  }

  const record = value as Partial<PreviewWorkspacePane>;
  const id = typeof record.id === 'string' ? record.id.trim() : '';
  if (id.length === 0) {
    return null;
  }

  const createdAt = typeof record.createdAt === 'number' && Number.isFinite(record.createdAt)
    ? record.createdAt
    : Date.now();

  return {
    id,
    appId: normalizePaneApp(record.appId),
    createdAt,
  };
};

const normalizePersistedWorkspaceState = (value: unknown): Pick<
  PreviewWorkspaceState,
  'layout' | 'interactionMode' | 'panes' | 'paneViewState' | 'focusedPaneId' | 'columnFractions' | 'rowFractions'
> => {
  const defaults = buildInitialState();
  if (!value || typeof value !== 'object') {
    return defaults;
  }

  const record = value as Partial<PreviewWorkspaceState>;
  const layout = isLayout(record.layout) ? record.layout : defaults.layout;
  const interactionMode = isInteractionMode(record.interactionMode) ? record.interactionMode : defaults.interactionMode;
  const panes = Array.isArray(record.panes)
    ? record.panes
      .map(normalizePersistedPane)
      .filter((pane): pane is PreviewWorkspacePane => Boolean(pane))
      .slice(0, MAX_PANES)
    : [];
  const ensuredPanes = panes.length >= MIN_PANES ? panes : defaults.panes;
  const allowedPaneIds = new Set(ensuredPanes.map((pane) => pane.id));
  const rawPaneViewState = record.paneViewState;
  const paneViewState = (!rawPaneViewState || typeof rawPaneViewState !== 'object')
    ? {}
    : Object.entries(rawPaneViewState as Record<string, unknown>)
      .reduce<Record<string, PreviewWorkspacePaneViewState>>((accumulator, [paneId, paneState]) => {
        if (!allowedPaneIds.has(paneId)) {
          return accumulator;
        }
        accumulator[paneId] = normalizePersistedPaneViewState(paneState);
        return accumulator;
      }, {});
  const focusedPaneId = typeof record.focusedPaneId === 'string' && ensuredPanes.some((pane) => pane.id === record.focusedPaneId)
    ? record.focusedPaneId
    : ensuredPanes[ensuredPanes.length - 1]?.id ?? null;
  const columnFractions = Array.isArray(record.columnFractions)
    ? record.columnFractions.filter((fraction): fraction is number => typeof fraction === 'number' && Number.isFinite(fraction))
    : defaults.columnFractions;
  const rowFractions = Array.isArray(record.rowFractions)
    ? record.rowFractions.filter((fraction): fraction is number => typeof fraction === 'number' && Number.isFinite(fraction))
    : defaults.rowFractions;
  const reconciled = reconcileFractionsForWorkspace(layout, ensuredPanes, columnFractions, rowFractions);

  return {
    layout,
    interactionMode,
    panes: ensuredPanes,
    paneViewState,
    focusedPaneId,
    columnFractions: reconciled.columnFractions,
    rowFractions: reconciled.rowFractions,
  };
};

export const usePreviewWorkspaceStore = create<PreviewWorkspaceState>()(persist((set, get) => ({
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
        paneViewState: {
          ...state.paneViewState,
          [nextPane.id]: createDefaultPaneViewState(),
        },
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
    const nextPaneViewState = { ...state.paneViewState };
    delete nextPaneViewState[paneId];
    return {
      panes: nextPanes,
      paneViewState: nextPaneViewState,
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

  setPaneViewState: (paneId, partial) => set((state) => {
    if (!state.panes.some((pane) => pane.id === paneId)) {
      return state;
    }
    const previous = state.paneViewState[paneId] ?? createDefaultPaneViewState();
    const merged: PreviewWorkspacePaneViewState = {
      ...previous,
      ...partial,
      history: Array.isArray(partial.history)
        ? partial.history.filter((entry): entry is string => typeof entry === 'string' && entry.trim().length > 0)
        : previous.history,
    };
    if (!Number.isFinite(merged.historyIndex)) {
      merged.historyIndex = merged.history.length - 1;
    }
    merged.historyIndex = Math.max(-1, Math.min(merged.history.length - 1, Math.floor(merged.historyIndex)));

    return {
      paneViewState: {
        ...state.paneViewState,
        [paneId]: merged,
      },
    };
  }),

  resetPaneViewState: (paneId) => set((state) => {
    if (!state.panes.some((pane) => pane.id === paneId)) {
      return state;
    }
    return {
      paneViewState: {
        ...state.paneViewState,
        [paneId]: createDefaultPaneViewState(),
      },
    };
  }),

  reset: () => set(buildInitialState()),
}), {
  name: PREVIEW_WORKSPACE_STORAGE_KEY,
  storage: previewWorkspaceStorage,
  version: 1,
  partialize: (state) => ({
    layout: state.layout,
    interactionMode: state.interactionMode,
    panes: state.panes,
    paneViewState: state.paneViewState,
    focusedPaneId: state.focusedPaneId,
    columnFractions: state.columnFractions,
    rowFractions: state.rowFractions,
  }),
  migrate: (persisted) => normalizePersistedWorkspaceState(persisted),
}));

export const previewWorkspaceLimits = {
  minPanes: MIN_PANES,
  maxPanes: MAX_PANES,
};
