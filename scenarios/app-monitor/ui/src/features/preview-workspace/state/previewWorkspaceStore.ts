import { create } from 'zustand';
import { createJSONStorage, persist, type StateStorage } from 'zustand/middleware';
import { isAppMonitorScenarioId } from '@/utils/appPreview';
import { fromPortablePreviewUrl, toPortablePreviewUrl } from '@/utils/previewUrl';
import { reconcileTrackFractions, resolveWorkspaceLayout } from '../utils/layout';

// AI_CHECK: APP_MONITOR_RENDER_PERF=2 | LAST: 2026-02-13
export type PreviewWorkspaceInteractionMode = 'browse' | 'arrange';
export type PreviewWorkspacePinnedColumn = 'left' | 'right';
export const PREVIEW_WORKSPACE_ZOOM_LEVELS = [0.5, 0.67, 0.75, 0.9, 1, 1.1, 1.25, 1.5] as const;
export type PreviewWorkspaceZoomLevel = typeof PREVIEW_WORKSPACE_ZOOM_LEVELS[number];
const DEFAULT_WORKSPACE_ZOOM: PreviewWorkspaceZoomLevel = 1;

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
  isFullView: boolean;
}

export interface PreviewWorkspaceState {
  interactionMode: PreviewWorkspaceInteractionMode;
  workspaceZoom: PreviewWorkspaceZoomLevel;
  isWorkspaceMinimapVisible: boolean;
  panes: PreviewWorkspacePane[];
  paneViewState: Record<string, PreviewWorkspacePaneViewState>;
  focusedPaneId: string | null;
  pinnedPaneId: string | null;
  pinnedColumn: PreviewWorkspacePinnedColumn | null;
  columnFractions: number[];
  rowFractions: number[];
  addPane: (appId?: string | null) => string;
  removePane: (paneId: string) => void;
  movePaneToIndex: (paneId: string, targetIndex: number) => void;
  setPaneApp: (paneId: string, appId: string | null) => void;
  focusPane: (paneId: string | null) => void;
  pinPaneToColumn: (paneId: string, column: PreviewWorkspacePinnedColumn) => void;
  clearPinnedPane: () => void;
  setInteractionMode: (mode: PreviewWorkspaceInteractionMode) => void;
  setWorkspaceZoom: (zoom: PreviewWorkspaceZoomLevel) => void;
  resetWorkspaceZoom: () => void;
  setWorkspaceMinimapVisible: (isVisible: boolean) => void;
  setColumnFractions: (fractions: number[]) => void;
  setRowFractions: (fractions: number[]) => void;
  resetLayout: () => void;
  clearAllPanes: () => void;
  setPaneViewState: (paneId: string, partial: Partial<PreviewWorkspacePaneViewState>) => void;
  resetPaneViewState: (paneId: string) => void;
  reset: () => void;
  exportPresetData: () => {
    interactionMode: PreviewWorkspaceInteractionMode;
    workspaceZoom: PreviewWorkspaceZoomLevel;
    paneApps: (string | null)[];
    panePreviewURLs: (string | null)[];
    columnFractions: number[];
    rowFractions: number[];
    pinnedPaneIndex: number | null;
    pinnedColumn: PreviewWorkspacePinnedColumn | null;
  };
  applyPreset: (data: {
    interaction_mode: string;
    workspace_zoom: number;
    pane_apps: (string | null)[];
    pane_preview_urls?: (string | null)[];
    column_fractions: number[];
    row_fractions: number[];
    pinned_pane_index: number | null;
    pinned_column: string | null;
  }) => void;
}

const MIN_PANES = 1;
const MAX_PANES = 50;
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
  | 'interactionMode'
  | 'workspaceZoom'
  | 'isWorkspaceMinimapVisible'
  | 'panes'
  | 'paneViewState'
  | 'focusedPaneId'
  | 'pinnedPaneId'
  | 'pinnedColumn'
  | 'columnFractions'
  | 'rowFractions'
> => {
  const firstPane = createPane(null);
  return {
    interactionMode: 'browse',
    workspaceZoom: DEFAULT_WORKSPACE_ZOOM,
    isWorkspaceMinimapVisible: true,
    panes: [firstPane],
    paneViewState: {},
    focusedPaneId: firstPane.id,
    pinnedPaneId: null,
    pinnedColumn: null,
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
  | 'interactionMode'
  | 'workspaceZoom'
  | 'isWorkspaceMinimapVisible'
  | 'panes'
  | 'paneViewState'
  | 'focusedPaneId'
  | 'pinnedPaneId'
  | 'pinnedColumn'
  | 'columnFractions'
  | 'rowFractions'
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
  if (trimmed.length === 0 || isAppMonitorScenarioId(trimmed)) {
    return null;
  }
  return trimmed;
};

const reconcileFractionsForWorkspace = (
  panes: PreviewWorkspacePane[],
  columnFractions: number[],
  rowFractions: number[],
): Pick<PreviewWorkspaceState, 'columnFractions' | 'rowFractions'> => {
  const layoutDescriptor = resolveWorkspaceLayout(panes.length);
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
  isFullView: false,
});

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
    isFullView: Boolean(record.isFullView),
  };
};
const isInteractionMode = (value: unknown): value is PreviewWorkspaceInteractionMode => (
  value === 'browse' || value === 'arrange'
);

const normalizeWorkspaceZoom = (value: unknown): PreviewWorkspaceZoomLevel => {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return DEFAULT_WORKSPACE_ZOOM;
  }
  if (PREVIEW_WORKSPACE_ZOOM_LEVELS.includes(value as PreviewWorkspaceZoomLevel)) {
    return value as PreviewWorkspaceZoomLevel;
  }
  let nearest: PreviewWorkspaceZoomLevel = PREVIEW_WORKSPACE_ZOOM_LEVELS[0] ?? DEFAULT_WORKSPACE_ZOOM;
  let nearestDelta = Math.abs(value - nearest);
  PREVIEW_WORKSPACE_ZOOM_LEVELS.forEach((candidate) => {
    const delta = Math.abs(value - candidate);
    if (delta < nearestDelta) {
      nearest = candidate;
      nearestDelta = delta;
    }
  });
  return nearest;
};

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
  | 'interactionMode'
  | 'workspaceZoom'
  | 'isWorkspaceMinimapVisible'
  | 'panes'
  | 'paneViewState'
  | 'focusedPaneId'
  | 'pinnedPaneId'
  | 'pinnedColumn'
  | 'columnFractions'
  | 'rowFractions'
> => {
  const defaults = buildInitialState();
  if (!value || typeof value !== 'object') {
    return defaults;
  }

  const record = value as Partial<PreviewWorkspaceState>;
  const interactionMode = isInteractionMode(record.interactionMode) ? record.interactionMode : defaults.interactionMode;
  const workspaceZoom = normalizeWorkspaceZoom(record.workspaceZoom);
  const isWorkspaceMinimapVisible = typeof record.isWorkspaceMinimapVisible === 'boolean'
    ? record.isWorkspaceMinimapVisible
    : defaults.isWorkspaceMinimapVisible;
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
  const pinnedPaneId = typeof record.pinnedPaneId === 'string' && ensuredPanes.some((pane) => pane.id === record.pinnedPaneId)
    ? record.pinnedPaneId
    : null;
  const pinnedColumn = record.pinnedColumn === 'left' || record.pinnedColumn === 'right'
    ? record.pinnedColumn
    : null;
  const columnFractions = Array.isArray(record.columnFractions)
    ? record.columnFractions.filter((fraction): fraction is number => typeof fraction === 'number' && Number.isFinite(fraction))
    : defaults.columnFractions;
  const rowFractions = Array.isArray(record.rowFractions)
    ? record.rowFractions.filter((fraction): fraction is number => typeof fraction === 'number' && Number.isFinite(fraction))
    : defaults.rowFractions;
  const reconciled = reconcileFractionsForWorkspace(ensuredPanes, columnFractions, rowFractions);

  return {
    interactionMode,
    workspaceZoom,
    isWorkspaceMinimapVisible,
    panes: ensuredPanes,
    paneViewState,
    focusedPaneId,
    pinnedPaneId,
    pinnedColumn,
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
        ...reconcileFractionsForWorkspace(panes, state.columnFractions, state.rowFractions),
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
      pinnedPaneId: state.pinnedPaneId === paneId ? null : state.pinnedPaneId,
      pinnedColumn: state.pinnedPaneId === paneId ? null : state.pinnedColumn,
      ...reconcileFractionsForWorkspace(nextPanes, state.columnFractions, state.rowFractions),
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

  setPaneApp: (paneId, appId) => set((state) => {
    const nextAppId = normalizePaneApp(appId);
    let didChangeApp = false;

    const nextPanes = state.panes.map((pane) => {
      if (pane.id !== paneId) {
        return pane;
      }

      if (pane.appId === nextAppId) {
        return pane;
      }

      didChangeApp = true;
      return { ...pane, appId: nextAppId };
    });

    if (!didChangeApp) {
      return state;
    }

    return {
      panes: nextPanes,
      paneViewState: {
        ...state.paneViewState,
        [paneId]: createDefaultPaneViewState(),
      },
    };
  }),

  focusPane: (paneId) => set((state) => {
    if (!paneId) {
      return { focusedPaneId: null };
    }
    const exists = state.panes.some((pane) => pane.id === paneId);
    return exists ? { focusedPaneId: paneId } : state;
  }),

  pinPaneToColumn: (paneId, column) => set((state) => {
    if (!state.panes.some((pane) => pane.id === paneId)) {
      return state;
    }
    return {
      pinnedPaneId: paneId,
      pinnedColumn: column,
      focusedPaneId: paneId,
    };
  }),

  clearPinnedPane: () => set((state) => {
    if (!state.pinnedPaneId && !state.pinnedColumn) {
      return state;
    }
    return {
      pinnedPaneId: null,
      pinnedColumn: null,
    };
  }),

  setInteractionMode: (interactionMode) => set({ interactionMode }),

  setWorkspaceZoom: (workspaceZoom) => set({ workspaceZoom: normalizeWorkspaceZoom(workspaceZoom) }),

  resetWorkspaceZoom: () => set({ workspaceZoom: DEFAULT_WORKSPACE_ZOOM }),

  setWorkspaceMinimapVisible: (isWorkspaceMinimapVisible) => set({ isWorkspaceMinimapVisible }),

  setColumnFractions: (fractions) => set((state) => {
    const { columns } = resolveWorkspaceLayout(state.panes.length);
    return { columnFractions: reconcileTrackFractions(fractions, columns) };
  }),

  setRowFractions: (fractions) => set((state) => {
    const { rows } = resolveWorkspaceLayout(state.panes.length);
    return { rowFractions: reconcileTrackFractions(fractions, rows) };
  }),

  resetLayout: () => set((state) => {
    const reconciled = reconcileFractionsForWorkspace(state.panes, [1], [1]);
    return {
      interactionMode: 'browse',
      pinnedPaneId: null,
      pinnedColumn: null,
      columnFractions: reconciled.columnFractions,
      rowFractions: reconciled.rowFractions,
    };
  }),

  clearAllPanes: () => set((state) => ({
    ...buildInitialState(),
    isWorkspaceMinimapVisible: state.isWorkspaceMinimapVisible,
  })),

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

    if (isPaneViewStateEqual(previous, merged)) {
      return state;
    }

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

  exportPresetData: () => {
    const state = get();
    const pinnedPaneIndex = state.pinnedPaneId
      ? state.panes.findIndex((pane) => pane.id === state.pinnedPaneId)
      : null;
    return {
      interactionMode: state.interactionMode,
      workspaceZoom: state.workspaceZoom,
      paneApps: state.panes.map((pane) => pane.appId),
      panePreviewURLs: state.panes.map((pane) => {
        const viewState = state.paneViewState[pane.id];
        return toPortablePreviewUrl(viewState?.previewUrl ?? null);
      }),
      columnFractions: state.columnFractions,
      rowFractions: state.rowFractions,
      pinnedPaneIndex: pinnedPaneIndex !== null && pinnedPaneIndex >= 0 ? pinnedPaneIndex : null,
      pinnedColumn: state.pinnedColumn,
    };
  },

  applyPreset: (data) => set(() => {
    const paneApps = Array.isArray(data.pane_apps) && data.pane_apps.length > 0
      ? data.pane_apps
      : [null];
    const previewURLs = Array.isArray(data.pane_preview_urls) ? data.pane_preview_urls : [];
    const newPanes = paneApps.map((appId) => createPane(normalizePaneApp(appId)));
    const interactionMode = isInteractionMode(data.interaction_mode) ? data.interaction_mode : 'browse';
    const workspaceZoom = normalizeWorkspaceZoom(data.workspace_zoom);
    const columnFractions = Array.isArray(data.column_fractions) ? data.column_fractions : [1];
    const rowFractions = Array.isArray(data.row_fractions) ? data.row_fractions : [1];
    const reconciled = reconcileFractionsForWorkspace(newPanes, columnFractions, rowFractions);
    const pinnedPaneId = typeof data.pinned_pane_index === 'number' && data.pinned_pane_index >= 0 && data.pinned_pane_index < newPanes.length
      ? newPanes[data.pinned_pane_index]?.id ?? null
      : null;
    const pinnedColumn: PreviewWorkspacePinnedColumn | null =
      pinnedPaneId && (data.pinned_column === 'left' || data.pinned_column === 'right')
        ? data.pinned_column
        : null;
    const paneViewState: Record<string, PreviewWorkspacePaneViewState> = {};
    for (let i = 0; i < newPanes.length; i++) {
      const url = fromPortablePreviewUrl(previewURLs[i] ?? null);
      if (typeof url === 'string' && url.trim().length > 0) {
        const pane = newPanes[i];
        if (!pane) continue;
        paneViewState[pane.id] = {
          ...createDefaultPaneViewState(),
          previewUrl: url,
          previewUrlInput: url,
          initialPreviewUrl: url,
        };
      }
    }
    return {
      interactionMode,
      workspaceZoom,
      panes: newPanes,
      paneViewState,
      focusedPaneId: newPanes[0]?.id ?? null,
      pinnedPaneId,
      pinnedColumn,
      columnFractions: reconciled.columnFractions,
      rowFractions: reconciled.rowFractions,
    };
  }),
}), {
  name: PREVIEW_WORKSPACE_STORAGE_KEY,
  storage: previewWorkspaceStorage,
  version: 1,
  partialize: (state) => ({
    interactionMode: state.interactionMode,
    workspaceZoom: state.workspaceZoom,
    isWorkspaceMinimapVisible: state.isWorkspaceMinimapVisible,
    panes: state.panes,
    paneViewState: state.paneViewState,
    focusedPaneId: state.focusedPaneId,
    pinnedPaneId: state.pinnedPaneId,
    pinnedColumn: state.pinnedColumn,
    columnFractions: state.columnFractions,
    rowFractions: state.rowFractions,
  }),
  migrate: (persisted) => normalizePersistedWorkspaceState(persisted),
}));

export const previewWorkspaceLimits = {
  minPanes: MIN_PANES,
  maxPanes: MAX_PANES,
};
