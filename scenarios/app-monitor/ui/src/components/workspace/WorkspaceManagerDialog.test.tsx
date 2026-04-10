import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import WorkspaceManagerDialog from './WorkspaceManagerDialog';
import { usePreviewWorkspaceStore } from '@/features/preview-workspace/state/previewWorkspaceStore';
import { useAppsStore } from '@/state/appsStore';
import { presetService } from '@/services/api';
import type { App } from '@/types';

vi.mock('@/services/api', () => ({
  presetService: {
    listPresets: vi.fn().mockResolvedValue([]),
    createPreset: vi.fn().mockResolvedValue(null),
    deletePreset: vi.fn().mockResolvedValue(true),
  },
}));

const makeApp = (overrides: Partial<App>): App => ({
  id: 'test-app',
  name: 'Test App',
  scenario_name: 'test-scenario',
  path: '/test',
  created_at: '2024-01-01',
  updated_at: '2024-01-01',
  status: 'running',
  port_mappings: {},
  environment: {},
  config: {},
  ...overrides,
});

describe('WorkspaceManagerDialog', () => {
  let fullscreenElement: Element | null = null;

  beforeEach(async () => {
    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    usePreviewWorkspaceStore.getState().reset();
    useAppsStore.setState({ apps: [] });
    fullscreenElement = null;
    Object.defineProperty(document, 'fullscreenElement', {
      configurable: true,
      get: () => fullscreenElement,
    });
  });

  it('adds a pane, scrolls to it, focuses URL input, and closes the dialog', () => {
    const onClose = vi.fn();
    const animationFrames: FrameRequestCallback[] = [];
    const requestAnimationFrameSpy = vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      animationFrames.push(callback);
      return animationFrames.length;
    });
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    const scrollIntoViewSpy = vi.fn();
    Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
      configurable: true,
      value: scrollIntoViewSpy,
    });
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /add pane/i }));

    expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(2);
    const newPaneId = usePreviewWorkspaceStore.getState().panes[1]?.id ?? null;
    expect(newPaneId).toBeTruthy();
    if (newPaneId) {
      const paneElement = document.createElement('div');
      paneElement.setAttribute('data-preview-pane-id', newPaneId);
      const previewUrlInput = document.createElement('input');
      previewUrlInput.setAttribute('aria-label', 'Preview URL');
      const selectSpy = vi.spyOn(previewUrlInput, 'select');
      paneElement.appendChild(previewUrlInput);
      document.body.appendChild(paneElement);
      animationFrames.shift()?.(0);
      animationFrames.shift()?.(0);
      expect(scrollIntoViewSpy).toHaveBeenCalledWith({
        behavior: 'smooth',
        block: 'nearest',
        inline: 'nearest',
      });
      expect(document.activeElement).toBe(previewUrlInput);
      expect(selectSpy).toHaveBeenCalledTimes(1);
      selectSpy.mockRestore();
      paneElement.remove();
    }
    expect(onClose).toHaveBeenCalledTimes(1);
    requestAnimationFrameSpy.mockRestore();
    if (originalScrollIntoView) {
      Object.defineProperty(HTMLElement.prototype, 'scrollIntoView', {
        configurable: true,
        value: originalScrollIntoView,
      });
    } else {
      delete (HTMLElement.prototype as Partial<HTMLElement>).scrollIntoView;
    }
  });

  it('toggles arrange mode and closes the dialog', () => {
    const onClose = vi.fn();
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /turn arrange on/i }));

    expect(usePreviewWorkspaceStore.getState().interactionMode).toBe('arrange');
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('clears all panes after confirmation', () => {
    const onClose = vi.fn();
    usePreviewWorkspaceStore.getState().addPane('scenario-a');
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /clear all panes/i }));

    expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(1);
    expect(usePreviewWorkspaceStore.getState().panes[0]?.appId).toBeNull();
    expect(onClose).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });

  it('updates workspace zoom without closing the dialog', () => {
    const onClose = vi.fn();
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.change(screen.getByLabelText(/all panes/i), { target: { value: '0.75' } });

    expect(usePreviewWorkspaceStore.getState().workspaceZoom).toBe(0.75);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('toggles workspace minimap visibility without closing the dialog', () => {
    const onClose = vi.fn();
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /hide workspace minimap/i }));

    expect(usePreviewWorkspaceStore.getState().isWorkspaceMinimapVisible).toBe(false);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('enters workspace fullscreen and closes the dialog', async () => {
    const onClose = vi.fn();
    const exitFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: exitFullscreen,
    });

    const workspaceRoot = document.createElement('div');
    workspaceRoot.className = 'preview-workspace';
    const requestFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(workspaceRoot, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen,
    });
    document.body.appendChild(workspaceRoot);

    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /enter workspace fullscreen/i }));

    await waitFor(() => {
      expect(requestFullscreen).toHaveBeenCalledTimes(1);
    });
    expect(onClose).toHaveBeenCalledTimes(1);

    workspaceRoot.remove();
  });

  it('exits workspace fullscreen and closes the dialog', async () => {
    const onClose = vi.fn();
    const exitFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: exitFullscreen,
    });

    const workspaceRoot = document.createElement('div');
    workspaceRoot.className = 'preview-workspace';
    const requestFullscreen = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(workspaceRoot, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen,
    });
    document.body.appendChild(workspaceRoot);
    fullscreenElement = workspaceRoot;

    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /exit workspace fullscreen/i }));

    await waitFor(() => {
      expect(exitFullscreen).toHaveBeenCalledTimes(1);
    });
    expect(requestFullscreen).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);

    workspaceRoot.remove();
  });

  describe('pane list', () => {
    it('renders a row for each pane with the app display name', () => {
      useAppsStore.setState({
        apps: [makeApp({ id: 'scenario-a', scenario_name: 'My Dashboard', name: 'dashboard' })],
      });
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      const list = screen.getByRole('list');
      const items = list.querySelectorAll('li');
      expect(items).toHaveLength(2);
      expect(items[0]?.textContent).toContain('Empty pane');
      expect(items[1]?.textContent).toContain('My Dashboard');
    });

    it('shows appId as fallback when app is not in the store', () => {
      usePreviewWorkspaceStore.getState().addPane('unknown-app');
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      const items = screen.getByRole('list').querySelectorAll('li');
      expect(items[1]?.textContent).toContain('unknown-app');
    });

    it('moves a pane up when the up button is clicked', () => {
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      usePreviewWorkspaceStore.getState().addPane('scenario-b');
      const paneIds = usePreviewWorkspaceStore.getState().panes.map(p => p.id);
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      // Move last pane (index 2) up
      fireEvent.click(screen.getByRole('button', { name: /move pane 3 up/i }));

      const reordered = usePreviewWorkspaceStore.getState().panes;
      expect(reordered[1]?.id).toBe(paneIds[2]);
      expect(reordered[2]?.id).toBe(paneIds[1]);
    });

    it('moves a pane down when the down button is clicked', () => {
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      const paneIds = usePreviewWorkspaceStore.getState().panes.map(p => p.id);
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      // Move first pane (index 0) down
      fireEvent.click(screen.getByRole('button', { name: /move pane 1 down/i }));

      const reordered = usePreviewWorkspaceStore.getState().panes;
      expect(reordered[0]?.id).toBe(paneIds[1]);
      expect(reordered[1]?.id).toBe(paneIds[0]);
    });

    it('disables up on first pane and down on last pane', () => {
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      expect(screen.getByRole('button', { name: /move pane 1 up/i })).toBeDisabled();
      expect(screen.getByRole('button', { name: /move pane 2 down/i })).toBeDisabled();
    });

    it('scroll-to button focuses the pane and closes the dialog', () => {
      const onClose = vi.fn();
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      const paneId = usePreviewWorkspaceStore.getState().panes[1]?.id ?? '';
      render(<WorkspaceManagerDialog onClose={onClose} />);

      fireEvent.click(screen.getByRole('button', { name: /scroll to pane 2/i }));

      expect(usePreviewWorkspaceStore.getState().focusedPaneId).toBe(paneId);
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('removes a pane when the remove button is clicked', () => {
      usePreviewWorkspaceStore.getState().addPane('scenario-a');
      expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(2);
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      fireEvent.click(screen.getByRole('button', { name: /remove pane 2/i }));

      expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(1);
    });

    it('disables remove when only one pane exists', () => {
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      expect(screen.getByRole('button', { name: /remove pane 1/i })).toBeDisabled();
    });
  });

  describe('presets', () => {
    it('shows empty state message when no presets exist', async () => {
      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('No saved presets yet.')).toBeInTheDocument();
      });
    });

    it('saves a preset and adds it to the list', async () => {
      const mockPreset = {
        id: 'preset-1',
        name: 'My Layout',
        color: '#7aa0ff',
        interaction_mode: 'browse' as const,
        workspace_zoom: 1,
        pane_apps: [null],
        pane_preview_urls: [null],
        column_fractions: [1],
        row_fractions: [1],
        pinned_pane_index: null,
        pinned_column: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };
      vi.mocked(presetService.createPreset).mockResolvedValueOnce(mockPreset);

      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('No saved presets yet.')).toBeInTheDocument();
      });

      const nameInput = screen.getByLabelText('Preset name');
      fireEvent.change(nameInput, { target: { value: 'My Layout' } });
      fireEvent.click(screen.getByRole('button', { name: /save current workspace/i }));

      await waitFor(() => {
        expect(presetService.createPreset).toHaveBeenCalled();
      });

      await waitFor(() => {
        expect(screen.getByText('My Layout')).toBeInTheDocument();
      });
    });

    it('loads a preset with preview URLs and closes the dialog', async () => {
      const onClose = vi.fn();
      const mockPreset = {
        id: 'preset-1',
        name: 'Debug Layout',
        color: '#ff7a7a',
        interaction_mode: 'arrange' as const,
        workspace_zoom: 0.75,
        pane_apps: ['scenario-a', 'scenario-b'],
        // First URL is a portable relative path (saved from another origin),
        // second is an external absolute URL
        pane_preview_urls: ['/apps/scenario-a/proxy/', 'http://localhost:5000'],
        column_fractions: [0.5, 0.5],
        row_fractions: [1],
        pinned_pane_index: null,
        pinned_column: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };
      vi.mocked(presetService.listPresets).mockResolvedValueOnce([mockPreset]);

      render(<WorkspaceManagerDialog onClose={onClose} />);

      await waitFor(() => {
        expect(screen.getByText('Debug Layout')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: /load preset debug layout/i }));

      const state = usePreviewWorkspaceStore.getState();
      expect(state.panes).toHaveLength(2);
      // Portable relative URL resolved against current origin
      const pane0View = state.paneViewState[state.panes[0]?.id ?? ''];
      const pane1View = state.paneViewState[state.panes[1]?.id ?? ''];
      expect(pane0View?.previewUrl).toBe('http://localhost:3000/apps/scenario-a/proxy/');
      // External URL preserved as-is
      expect(pane1View?.previewUrl).toBe('http://localhost:5000');
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('deletes a preset after confirmation', async () => {
      const mockPreset = {
        id: 'preset-del',
        name: 'To Delete',
        color: '#7aff9e',
        interaction_mode: 'browse' as const,
        workspace_zoom: 1,
        pane_apps: [null],
        pane_preview_urls: [null],
        column_fractions: [1],
        row_fractions: [1],
        pinned_pane_index: null,
        pinned_column: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };
      vi.mocked(presetService.listPresets).mockResolvedValueOnce([mockPreset]);
      vi.mocked(presetService.deletePreset).mockResolvedValueOnce(true);
      const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);

      render(<WorkspaceManagerDialog onClose={vi.fn()} />);

      await waitFor(() => {
        expect(screen.getByText('To Delete')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByRole('button', { name: /delete preset to delete/i }));

      await waitFor(() => {
        expect(presetService.deletePreset).toHaveBeenCalledWith('preset-del');
      });

      await waitFor(() => {
        expect(screen.queryByText('To Delete')).not.toBeInTheDocument();
      });

      confirmSpy.mockRestore();
    });
  });
});
