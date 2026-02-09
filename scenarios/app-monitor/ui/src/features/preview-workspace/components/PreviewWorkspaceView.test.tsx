import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { useAppsStore } from '@/state/appsStore';
import { usePreviewWorkspaceStore } from '../state/previewWorkspaceStore';
import type { App } from '@/types';
import PreviewWorkspaceView from './PreviewWorkspaceView';

const WORKSPACE_STORAGE_KEY = 'app-monitor:preview-workspace-v1';
let mockPaneContentHeightPx = 0;

vi.mock('./PreviewPane', () => ({
  default: ({
    paneId,
    onRemove,
    onArrangeDragStart,
  }: {
    paneId: string;
    onRemove: (paneId: string) => void;
    onArrangeDragStart: (paneId: string, event: ReactPointerEvent<HTMLButtonElement>) => void;
  }) => (
    <div data-testid={`preview-pane-${paneId}`}>
      <div
        data-testid={`preview-pane-content-${paneId}`}
        style={mockPaneContentHeightPx > 0 ? { height: `${mockPaneContentHeightPx}px` } : undefined}
      />
      <button
        type="button"
        aria-label={`Drag pane ${paneId}`}
        onPointerDown={(event) => onArrangeDragStart(paneId, event)}
      >
        Drag
      </button>
      <button
        type="button"
        aria-label={`Remove pane ${paneId}`}
        onClick={() => onRemove(paneId)}
      >
        Remove
      </button>
    </div>
  ),
}));

vi.mock('@/hooks/useOverlayRouter', () => ({
  useOverlayRouter: () => ({
    overlay: null,
    openOverlay: vi.fn(),
    closeOverlay: vi.fn(),
  }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

const buildApp = (id: string): App => ({
  id,
  name: id,
  scenario_name: id,
  path: `/tmp/${id}`,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  status: 'running',
  port_mappings: { UI_PORT: 4310 },
  environment: {},
  config: {},
});

describe('PreviewWorkspaceView', () => {
  beforeEach(async () => {
    mockPaneContentHeightPx = 0;
    if (!HTMLElement.prototype.setPointerCapture) {
      Object.defineProperty(HTMLElement.prototype, 'setPointerCapture', {
        configurable: true,
        value: vi.fn(),
      });
    }

    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    usePreviewWorkspaceStore.getState().reset();

    useAppsStore.setState({
      apps: [buildApp('scenario-a'), buildApp('scenario-b')],
      loadingInitial: false,
      loadingDetailed: false,
      error: null,
      hasInitialized: true,
      lastLoadTimestamp: Date.now(),
    });
  });

  it('consumes add-pane intent and clears query params', async () => {
    render(
      <MemoryRouter initialEntries={['/apps/workspace?workspaceAppId=scenario-a&workspaceMode=add-pane']}>
        <Routes>
          <Route path="/apps/workspace" element={<><PreviewWorkspaceView /><LocationProbe /></>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      const panes = usePreviewWorkspaceStore.getState().panes;
      expect(panes).toHaveLength(2);
      expect(panes[1]?.appId).toBe('scenario-a');
    });

    await waitFor(() => {
      expect(screen.getByTestId('location-search').textContent).toBe('');
    });
  });

  it('loads apps when workspace mounts before app catalog initialization', async () => {
    const loadAppsSpy = vi.fn(async () => undefined);
    useAppsStore.setState({
      apps: [],
      loadingInitial: false,
      loadingDetailed: false,
      error: null,
      hasInitialized: false,
      lastLoadTimestamp: null,
      loadApps: loadAppsSpy,
    });

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(loadAppsSpy).toHaveBeenCalledTimes(1);
    });
  });

  it('replaces the focused pane app for replace-focused intent', async () => {
    const store = usePreviewWorkspaceStore.getState();
    const paneId = store.panes[0]?.id;
    expect(paneId).toBeTruthy();
    if (!paneId) {
      return;
    }
    usePreviewWorkspaceStore.getState().setPaneApp(paneId, 'scenario-b');

    render(
      <MemoryRouter initialEntries={['/apps/workspace?workspaceAppId=scenario-a&workspaceMode=replace-focused']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      const pane = usePreviewWorkspaceStore.getState().panes.find((entry) => entry.id === paneId);
      expect(pane?.appId).toBe('scenario-a');
      expect(usePreviewWorkspaceStore.getState().focusedPaneId).toBe(paneId);
    });
  });

  it('resizes workspace columns via splitter drag', async () => {
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    const setColumnFractionsSpy = vi.fn();
    usePreviewWorkspaceStore.setState({ setColumnFractions: setColumnFractionsSpy });

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const panesContainer = document.querySelector('.preview-workspace__panes');
    expect(panesContainer).not.toBeNull();
    if (!panesContainer) {
      return;
    }

    Object.defineProperty(panesContainer, 'clientWidth', {
      configurable: true,
      value: 1000,
    });

    const resizeButton = screen.getByRole('button', { name: /resize column 1/i });
    fireEvent.pointerDown(resizeButton, { pointerId: 1, clientX: 500, clientY: 100 });
    fireEvent.pointerMove(window, { pointerId: 1, clientX: 700, clientY: 100 });
    fireEvent.pointerUp(window, { pointerId: 1 });

    await waitFor(() => {
      expect(setColumnFractionsSpy).toHaveBeenCalled();
    });
  });

  it('reorders panes after arrange-mode drag', async () => {
    const firstPaneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    const secondPaneId = usePreviewWorkspaceStore.getState().addPane('scenario-a');
    expect(firstPaneId).toBeTruthy();
    expect(secondPaneId).toBeTruthy();
    usePreviewWorkspaceStore.getState().setInteractionMode('arrange');
    const movePaneToIndexSpy = vi.fn();
    usePreviewWorkspaceStore.setState({ movePaneToIndex: movePaneToIndexSpy });

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const panesContainer = document.querySelector('.preview-workspace__panes');
    expect(panesContainer).not.toBeNull();
    if (!panesContainer || !firstPaneId) {
      return;
    }

    vi.spyOn(panesContainer, 'getBoundingClientRect').mockReturnValue({
      left: 0,
      top: 0,
      width: 1000,
      height: 400,
      right: 1000,
      bottom: 400,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    });

    const dragButton = screen.getByRole('button', { name: new RegExp(`drag pane ${firstPaneId}`, 'i') });
    fireEvent.pointerDown(dragButton, { pointerId: 2, clientX: 100, clientY: 100 });
    await waitFor(() => {
      expect(panesContainer.className).toContain('preview-workspace__panes--dragging');
    });
    fireEvent.pointerMove(window, { pointerId: 2, clientX: 700, clientY: 100 });
    fireEvent.pointerUp(window, { pointerId: 2 });

    await waitFor(() => {
      expect(movePaneToIndexSpy).toHaveBeenCalled();
      expect(movePaneToIndexSpy).toHaveBeenCalledWith(firstPaneId, expect.any(Number));
    });
  });

  it('resizes pinned layout columns via splitter drag', async () => {
    const firstPaneId = usePreviewWorkspaceStore.getState().panes[0]?.id;
    const secondPaneId = usePreviewWorkspaceStore.getState().addPane('scenario-a');
    expect(firstPaneId).toBeTruthy();
    expect(secondPaneId).toBeTruthy();
    if (!firstPaneId) {
      return;
    }

    usePreviewWorkspaceStore.getState().pinPaneToColumn(firstPaneId, 'left');
    const setColumnFractionsSpy = vi.fn();
    usePreviewWorkspaceStore.setState({ setColumnFractions: setColumnFractionsSpy });

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const pinnedContainer = document.querySelector('.preview-workspace__pinned-layout');
    expect(pinnedContainer).not.toBeNull();
    if (!pinnedContainer) {
      return;
    }

    Object.defineProperty(pinnedContainer, 'clientWidth', {
      configurable: true,
      value: 1000,
    });

    const resizeButton = screen.getByRole('button', { name: /resize column 1/i });
    fireEvent.pointerDown(resizeButton, { pointerId: 3, clientX: 500, clientY: 100 });
    fireEvent.pointerMove(window, { pointerId: 3, clientX: 650, clientY: 100 });
    fireEvent.pointerUp(window, { pointerId: 3 });

    await waitFor(() => {
      expect(setColumnFractionsSpy).toHaveBeenCalled();
    });
  });

  it('resizes workspace rows via splitter drag and stores fractions', async () => {
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    usePreviewWorkspaceStore.getState().addPane('scenario-b');
    const setRowFractionsSpy = vi.fn();
    usePreviewWorkspaceStore.setState({ setRowFractions: setRowFractionsSpy });

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const panesContainer = document.querySelector('.preview-workspace__panes');
    expect(panesContainer).not.toBeNull();
    if (!panesContainer) {
      return;
    }

    Object.defineProperty(panesContainer, 'clientHeight', {
      configurable: true,
      value: 900,
    });

    const resizeButton = screen.getByRole('button', { name: /resize row 1/i });
    fireEvent.pointerDown(resizeButton, { pointerId: 4, clientX: 400, clientY: 520 });
    fireEvent.pointerMove(window, { pointerId: 4, clientX: 400, clientY: 350 });
    fireEvent.pointerUp(window, { pointerId: 4 });

    await waitFor(() => {
      expect(setRowFractionsSpy).toHaveBeenCalled();
    });
  });

  it('restores row sizing from persisted workspace fractions after rehydrate', async () => {
    window.localStorage.setItem(WORKSPACE_STORAGE_KEY, JSON.stringify({
      state: {
        interactionMode: 'browse',
        panes: [
          { id: 'pane-a', appId: 'scenario-a', createdAt: 1000 },
          { id: 'pane-b', appId: 'scenario-b', createdAt: 1001 },
          { id: 'pane-c', appId: null, createdAt: 1002 },
        ],
        paneViewState: {},
        focusedPaneId: 'pane-c',
        pinnedPaneId: null,
        pinnedColumn: null,
        columnFractions: [0.45, 0.55],
        rowFractions: [0.2, 0.8],
      },
      version: 1,
    }));
    await usePreviewWorkspaceStore.persist.rehydrate();

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      const panesContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
      expect(panesContainer).not.toBeNull();
      if (!panesContainer) {
        return;
      }
      expect(panesContainer.style.gridTemplateRows).toContain('0.2fr');
      expect(panesContainer.style.gridTemplateRows).toContain('0.8fr');
    });
  });

  it('keeps workspace grid height stable during loading-to-content handoff', async () => {
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    usePreviewWorkspaceStore.getState().addPane('scenario-b');

    Object.defineProperty(window, 'innerHeight', {
      configurable: true,
      value: 900,
    });

    const { rerender } = render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const panesContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
    expect(panesContainer).not.toBeNull();
    if (!panesContainer) {
      return;
    }
    const initialHeight = panesContainer.style.height;
    expect(initialHeight.length).toBeGreaterThan(0);

    mockPaneContentHeightPx = 2400;
    rerender(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      const updatedContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
      expect(updatedContainer).not.toBeNull();
      if (!updatedContainer) {
        return;
      }
      expect(updatedContainer.style.height).toBe(initialHeight);
      expect(updatedContainer.style.minHeight).toBe(initialHeight);
    });
  });

  it('expands remaining row to full height fraction when reduced to one row', async () => {
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    const removablePaneId = usePreviewWorkspaceStore.getState().addPane('scenario-b');

    render(
      <MemoryRouter initialEntries={['/apps/workspace']}>
        <Routes>
          <Route path="/apps/workspace" element={<PreviewWorkspaceView />} />
        </Routes>
      </MemoryRouter>,
    );

    const rowResizeButton = await screen.findByRole('button', { name: /resize row 1/i });
    fireEvent.pointerDown(rowResizeButton, { pointerId: 21, clientX: 220, clientY: 360 });
    fireEvent.pointerMove(window, { pointerId: 21, clientX: 220, clientY: -360 });
    fireEvent.pointerUp(window, { pointerId: 21 });

    await waitFor(() => {
      const panesContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
      expect(panesContainer).not.toBeNull();
      if (!panesContainer) {
        return;
      }
      expect(panesContainer.style.gridTemplateRows).not.toBe('minmax(0, 1fr)');
    });

    fireEvent.click(screen.getByRole('button', { name: new RegExp(`remove pane ${removablePaneId}`, 'i') }));

    await waitFor(() => {
      expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(2);
      const panesContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
      expect(panesContainer).not.toBeNull();
      if (!panesContainer) {
        return;
      }
      expect(panesContainer.style.gridTemplateRows).toBe('minmax(0, 1fr)');
    });
  });

});
