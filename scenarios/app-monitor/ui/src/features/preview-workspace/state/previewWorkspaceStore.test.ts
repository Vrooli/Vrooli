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
