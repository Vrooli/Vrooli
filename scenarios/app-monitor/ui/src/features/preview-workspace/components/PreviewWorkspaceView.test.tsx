import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { useAppsStore } from '@/state/appsStore';
import { usePreviewWorkspaceStore } from '../state/previewWorkspaceStore';
import type { App } from '@/types';
import PreviewWorkspaceView from './PreviewWorkspaceView';

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

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /toggle pane arrange mode/i })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: /toggle pane arrange mode/i }));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /toggle pane arrange mode/i })).toHaveAttribute('aria-pressed', 'true');
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

  it('expands remaining row to full viewport height when reduced to one row', async () => {
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
      const firstRowMatch = panesContainer.style.gridTemplateRows.match(/minmax\(240px,\s*([0-9.]+)px\)/);
      expect(firstRowMatch).not.toBeNull();
      const firstRowHeight = firstRowMatch ? Number(firstRowMatch[1]) : 0;
      expect(firstRowHeight).toBeLessThanOrEqual(300);
    });

    fireEvent.click(screen.getByRole('button', { name: new RegExp(`remove pane ${removablePaneId}`, 'i') }));

    await waitFor(() => {
      expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(2);
      const panesContainer = document.querySelector('.preview-workspace__panes') as HTMLElement | null;
      expect(panesContainer).not.toBeNull();
      if (!panesContainer) {
        return;
      }

      const singleRowMatch = panesContainer.style.gridTemplateRows.match(/minmax\(240px,\s*([0-9.]+)px\)/);
      expect(singleRowMatch).not.toBeNull();
      const singleRowHeight = singleRowMatch ? Number(singleRowMatch[1]) : 0;
      expect(singleRowHeight).toBeGreaterThan(600);
    });
  });

});
