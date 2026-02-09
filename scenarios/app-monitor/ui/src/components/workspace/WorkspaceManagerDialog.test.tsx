import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import WorkspaceManagerDialog from './WorkspaceManagerDialog';
import { usePreviewWorkspaceStore } from '@/features/preview-workspace/state/previewWorkspaceStore';

describe('WorkspaceManagerDialog', () => {
  let fullscreenElement: Element | null = null;

  beforeEach(async () => {
    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    usePreviewWorkspaceStore.getState().reset();
    fullscreenElement = null;
    Object.defineProperty(document, 'fullscreenElement', {
      configurable: true,
      get: () => fullscreenElement,
    });
  });

  it('adds a pane and closes the dialog', () => {
    const onClose = vi.fn();
    render(<WorkspaceManagerDialog onClose={onClose} />);

    fireEvent.click(screen.getByRole('button', { name: /add pane/i }));

    expect(usePreviewWorkspaceStore.getState().panes).toHaveLength(2);
    expect(onClose).toHaveBeenCalledTimes(1);
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
});
