import { beforeEach, describe, expect, it } from 'vitest';
import {
  previewWorkspaceLimits,
  usePreviewWorkspaceStore,
} from './previewWorkspaceStore';

const STORAGE_KEY = 'app-monitor:preview-workspace-v1';

const resetStore = () => {
  usePreviewWorkspaceStore.getState().reset();
};

describe('previewWorkspaceStore', () => {
  beforeEach(async () => {
    window.localStorage.removeItem(STORAGE_KEY);
    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    resetStore();
  });

  it('starts with one pane and a focused pane', () => {
    const state = usePreviewWorkspaceStore.getState();
    expect(state.panes).toHaveLength(1);
    expect(state.focusedPaneId).toBe(state.panes[0]?.id ?? null);
    expect(state.pinnedPaneId).toBeNull();
    expect(state.pinnedColumn).toBeNull();
    expect(state.interactionMode).toBe('browse');
    expect(state.columnFractions).toEqual([1]);
    expect(state.rowFractions).toEqual([1]);
  });

  it('adds panes and focuses the newest pane', () => {
    const firstState = usePreviewWorkspaceStore.getState();
    const firstPaneId = firstState.panes[0]?.id;
    expect(firstPaneId).toBeTruthy();

    const newPaneId = usePreviewWorkspaceStore.getState().addPane('scenario-a');
    const nextState = usePreviewWorkspaceStore.getState();

    expect(nextState.panes).toHaveLength(2);
    const newPane = nextState.panes.find((pane) => pane.id === newPaneId);
    expect(newPane?.appId).toBe('scenario-a');
    expect(nextState.focusedPaneId).toBe(newPaneId);
    expect(nextState.columnFractions).toHaveLength(2);
  });

  it('enforces max pane limit', () => {
    const { maxPanes } = previewWorkspaceLimits;
    for (let index = 0; index < maxPanes + 2; index += 1) {
      usePreviewWorkspaceStore.getState().addPane(`app-${index}`);
    }

    const state = usePreviewWorkspaceStore.getState();
    expect(state.panes.length).toBe(maxPanes);
  });

  it('does not remove the last remaining pane', () => {
    const state = usePreviewWorkspaceStore.getState();
    const paneId = state.panes[0]?.id;
    expect(paneId).toBeTruthy();

    if (paneId) {
      usePreviewWorkspaceStore.getState().removePane(paneId);
    }

    const nextState = usePreviewWorkspaceStore.getState();
    expect(nextState.panes).toHaveLength(1);
  });

  it('moves panes to a target index', () => {
    const paneA = usePreviewWorkspaceStore.getState().panes[0]?.id;
    const paneB = usePreviewWorkspaceStore.getState().addPane('scenario-b');
    const paneC = usePreviewWorkspaceStore.getState().addPane('scenario-c');

    expect(paneA).toBeTruthy();
    expect(paneB).toBeTruthy();
    expect(paneC).toBeTruthy();

    if (!paneA) {
      return;
    }

    usePreviewWorkspaceStore.getState().movePaneToIndex(paneA, 2);
    const nextState = usePreviewWorkspaceStore.getState();

    expect(nextState.panes[2]?.id).toBe(paneA);
    expect(nextState.focusedPaneId).toBe(paneA);
  });

  it('updates pane app and interaction mode', () => {
    const state = usePreviewWorkspaceStore.getState();
    const paneId = state.panes[0]?.id;
    expect(paneId).toBeTruthy();

    if (!paneId) {
      return;
    }

    usePreviewWorkspaceStore.getState().setPaneApp(paneId, 'scenario-b');
    usePreviewWorkspaceStore.getState().setInteractionMode('arrange');

    const nextState = usePreviewWorkspaceStore.getState();
    expect(nextState.interactionMode).toBe('arrange');
    expect(nextState.panes[0]?.appId).toBe('scenario-b');
  });

  it('disallows assigning app-monitor to a pane', () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }

    usePreviewWorkspaceStore.getState().setPaneApp(paneId, 'app-monitor');
    expect(usePreviewWorkspaceStore.getState().panes[0]?.appId).toBeNull();

    const secondPaneId = usePreviewWorkspaceStore.getState().addPane('app-monitor');
    const secondPane = usePreviewWorkspaceStore.getState().panes.find((pane) => pane.id === secondPaneId);
    expect(secondPane?.appId).toBeNull();
  });

  it('pins and unpins panes by column', () => {
    const paneA = usePreviewWorkspaceStore.getState().panes[0]?.id;
    const paneB = usePreviewWorkspaceStore.getState().addPane('scenario-b');
    expect(paneA).toBeTruthy();
    expect(paneB).toBeTruthy();
    if (!paneB) {
      return;
    }

    usePreviewWorkspaceStore.getState().pinPaneToColumn(paneB, 'right');
    let state = usePreviewWorkspaceStore.getState();
    expect(state.pinnedPaneId).toBe(paneB);
    expect(state.pinnedColumn).toBe('right');
    expect(state.focusedPaneId).toBe(paneB);

    usePreviewWorkspaceStore.getState().clearPinnedPane();
    state = usePreviewWorkspaceStore.getState();
    expect(state.pinnedPaneId).toBeNull();
    expect(state.pinnedColumn).toBeNull();
  });

  it('resets layout while preserving panes and assignments', () => {
    const paneA = usePreviewWorkspaceStore.getState().panes[0]?.id;
    const paneB = usePreviewWorkspaceStore.getState().addPane('scenario-b');
    expect(paneA).toBeTruthy();
    expect(paneB).toBeTruthy();
    if (!paneA || !paneB) {
      return;
    }

    usePreviewWorkspaceStore.getState().setInteractionMode('arrange');
    usePreviewWorkspaceStore.getState().setColumnFractions([0.2, 0.8]);
    usePreviewWorkspaceStore.getState().pinPaneToColumn(paneB, 'left');

    usePreviewWorkspaceStore.getState().resetLayout();
    const state = usePreviewWorkspaceStore.getState();

    expect(state.panes).toHaveLength(2);
    expect(state.panes.find((pane) => pane.id === paneB)?.appId).toBe('scenario-b');
    expect(state.interactionMode).toBe('browse');
    expect(state.pinnedPaneId).toBeNull();
    expect(state.pinnedColumn).toBeNull();
    expect(state.columnFractions).toEqual([0.5, 0.5]);
    expect(state.rowFractions).toEqual([1]);
  });

  it('clears all panes back to one empty pane', () => {
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    usePreviewWorkspaceStore.getState().addPane('scenario-b');
    usePreviewWorkspaceStore.getState().setInteractionMode('arrange');

    usePreviewWorkspaceStore.getState().clearAllPanes();
    const state = usePreviewWorkspaceStore.getState();

    expect(state.panes).toHaveLength(1);
    expect(state.panes[0]?.appId).toBeNull();
    expect(state.focusedPaneId).toBe(state.panes[0]?.id ?? null);
    expect(state.interactionMode).toBe('browse');
    expect(state.pinnedPaneId).toBeNull();
    expect(state.pinnedColumn).toBeNull();
    expect(state.columnFractions).toEqual([1]);
    expect(state.rowFractions).toEqual([1]);
  });

  it('resets pane-local view state when pane app changes', () => {
    const paneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }

    usePreviewWorkspaceStore.getState().setPaneViewState(paneId, {
      previewUrl: 'http://localhost:5000',
      previewUrlInput: 'http://localhost:5000',
      hasCustomPreviewUrl: true,
      history: ['http://localhost:5000'],
      historyIndex: 0,
      initialPreviewUrl: 'http://localhost:5000',
      isLogsVisible: true,
    });

    usePreviewWorkspaceStore.getState().setPaneApp(paneId, 'scenario-b');
    const nextState = usePreviewWorkspaceStore.getState();
    expect(nextState.panes[0]?.appId).toBe('scenario-b');
    expect(nextState.paneViewState[paneId]).toEqual({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
      initialPreviewUrl: null,
      isLogsVisible: false,
    });
  });

  it('exportPresetData captures current state correctly including preview URLs', () => {
    const paneA = usePreviewWorkspaceStore.getState().panes[0]?.id;
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    usePreviewWorkspaceStore.getState().setWorkspaceZoom(0.75);

    // Set a same-origin proxy URL on the second pane
    const paneB = usePreviewWorkspaceStore.getState().panes[1]?.id;
    if (paneB) {
      usePreviewWorkspaceStore.getState().setPaneViewState(paneB, {
        previewUrl: 'http://localhost:3000/apps/scenario-a/proxy/',
        previewUrlInput: 'http://localhost:3000/apps/scenario-a/proxy/',
      });
    }

    if (paneA) {
      usePreviewWorkspaceStore.getState().pinPaneToColumn(paneA, 'left');
    }

    const data = usePreviewWorkspaceStore.getState().exportPresetData();
    expect(data.interactionMode).toBe('browse');
    expect(data.workspaceZoom).toBe(0.75);
    expect(data.paneApps).toHaveLength(2);
    expect(data.paneApps[0]).toBeNull();
    expect(data.paneApps[1]).toBe('scenario-a');
    expect(data.panePreviewURLs).toHaveLength(2);
    expect(data.panePreviewURLs[0]).toBeNull();
    // Same-origin URL is stored as a portable relative path
    expect(data.panePreviewURLs[1]).toBe('/apps/scenario-a/proxy/');
    expect(data.pinnedPaneIndex).toBe(0);
    expect(data.pinnedColumn).toBe('left');
  });

  it('applyPreset replaces state with fresh pane IDs and restores URLs', () => {
    usePreviewWorkspaceStore.getState().addPane('old-app');
    const oldPaneIds = usePreviewWorkspaceStore.getState().panes.map((p) => p.id);

    usePreviewWorkspaceStore.getState().applyPreset({
      interaction_mode: 'arrange',
      workspace_zoom: 0.67,
      pane_apps: ['scenario-x', null, 'scenario-y'],
      pane_preview_urls: ['http://localhost:4000', null, 'http://localhost:5000'],
      column_fractions: [0.3, 0.7],
      row_fractions: [0.5, 0.5],
      pinned_pane_index: 2,
      pinned_column: 'right',
    });

    const state = usePreviewWorkspaceStore.getState();
    expect(state.interactionMode).toBe('arrange');
    expect(state.workspaceZoom).toBe(0.67);
    expect(state.panes).toHaveLength(3);
    expect(state.panes[0]?.appId).toBe('scenario-x');
    expect(state.panes[1]?.appId).toBeNull();
    expect(state.panes[2]?.appId).toBe('scenario-y');
    // Fresh pane IDs
    const newPaneIds = state.panes.map((p) => p.id);
    expect(newPaneIds).not.toEqual(oldPaneIds);
    // Pinned pane
    expect(state.pinnedPaneId).toBe(state.panes[2]?.id);
    expect(state.pinnedColumn).toBe('right');
    // Focused first pane
    expect(state.focusedPaneId).toBe(state.panes[0]?.id);
    // Preview URLs restored for panes that had them
    const pane0Id = state.panes[0]?.id ?? '';
    const pane1Id = state.panes[1]?.id ?? '';
    const pane2Id = state.panes[2]?.id ?? '';
    expect(state.paneViewState[pane0Id]?.previewUrl).toBe('http://localhost:4000');
    expect(state.paneViewState[pane0Id]?.previewUrlInput).toBe('http://localhost:4000');
    expect(state.paneViewState[pane1Id]).toBeUndefined();
    expect(state.paneViewState[pane2Id]?.previewUrl).toBe('http://localhost:5000');
  });

  it('preset round-trip makes same-origin URLs portable and restores them', () => {
    // Set up workspace: pane A with same-origin proxy URL, pane B with external URL
    const paneAId = usePreviewWorkspaceStore.getState().panes[0]?.id ?? '';
    usePreviewWorkspaceStore.getState().setPaneApp(paneAId, 'scenario-a');
    usePreviewWorkspaceStore.getState().setPaneViewState(paneAId, {
      previewUrl: 'http://localhost:3000/apps/scenario-a/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/scenario-a/proxy/',
      hasCustomPreviewUrl: true,
    });
    usePreviewWorkspaceStore.getState().addPane('scenario-b');
    const paneBId = usePreviewWorkspaceStore.getState().panes[1]?.id ?? '';
    usePreviewWorkspaceStore.getState().setPaneViewState(paneBId, {
      previewUrl: 'http://localhost:8080',
      previewUrlInput: 'http://localhost:8080',
    });

    // Export: same-origin URL becomes relative, external stays absolute
    const exported = usePreviewWorkspaceStore.getState().exportPresetData();
    expect(exported.panePreviewURLs).toEqual([
      '/apps/scenario-a/proxy/',
      'http://localhost:8080',
    ]);

    // Clear workspace and re-apply
    usePreviewWorkspaceStore.getState().clearAllPanes();
    usePreviewWorkspaceStore.getState().applyPreset({
      interaction_mode: exported.interactionMode,
      workspace_zoom: exported.workspaceZoom,
      pane_apps: exported.paneApps,
      pane_preview_urls: exported.panePreviewURLs,
      column_fractions: exported.columnFractions,
      row_fractions: exported.rowFractions,
      pinned_pane_index: exported.pinnedPaneIndex,
      pinned_column: exported.pinnedColumn,
    });

    const state = usePreviewWorkspaceStore.getState();
    expect(state.panes).toHaveLength(2);
    // Same-origin URL restored with current origin
    const newPaneAView = state.paneViewState[state.panes[0]?.id ?? ''];
    expect(newPaneAView?.previewUrl).toBe('http://localhost:3000/apps/scenario-a/proxy/');
    expect(newPaneAView?.previewUrlInput).toBe('http://localhost:3000/apps/scenario-a/proxy/');
    // External URL preserved as-is
    const newPaneBView = state.paneViewState[state.panes[1]?.id ?? ''];
    expect(newPaneBView?.previewUrl).toBe('http://localhost:8080');
    expect(newPaneBView?.previewUrlInput).toBe('http://localhost:8080');
  });

  it('applyPreset with empty paneApps creates one empty pane', () => {
    usePreviewWorkspaceStore.getState().applyPreset({
      interaction_mode: 'browse',
      workspace_zoom: 1,
      pane_apps: [],
      column_fractions: [],
      row_fractions: [],
      pinned_pane_index: null,
      pinned_column: null,
    });

    const state = usePreviewWorkspaceStore.getState();
    expect(state.panes).toHaveLength(1);
    expect(state.panes[0]?.appId).toBeNull();
  });

  it('rehydrates panes, focused pane, and fractions from localStorage', async () => {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({
      state: {
        interactionMode: 'arrange',
        panes: [
          { id: 'pane-a', appId: 'scenario-a', createdAt: 1000 },
          { id: 'pane-b', appId: 'scenario-b', createdAt: 2000 },
        ],
        paneViewState: {
          'pane-a': {
            previewUrl: 'http://localhost:3000',
            previewUrlInput: 'http://localhost:3000',
            hasCustomPreviewUrl: true,
            history: ['http://localhost:3000'],
            historyIndex: 0,
            initialPreviewUrl: 'http://localhost:3000',
            isLogsVisible: true,
          },
        },
        focusedPaneId: 'pane-b',
        columnFractions: [0.35, 0.65],
        rowFractions: [1],
      },
      version: 1,
    }));

    await usePreviewWorkspaceStore.persist.rehydrate();

    const state = usePreviewWorkspaceStore.getState();
    expect(state.interactionMode).toBe('arrange');
    expect(state.panes.map((pane) => pane.id)).toEqual(['pane-a', 'pane-b']);
    expect(state.focusedPaneId).toBe('pane-b');
    expect(state.columnFractions).toEqual([0.35, 0.65]);
    expect(state.rowFractions).toEqual([1]);
    expect(state.paneViewState['pane-a']?.history).toEqual(['http://localhost:3000']);
    expect(state.paneViewState['pane-a']?.isLogsVisible).toBe(true);
  });
});
