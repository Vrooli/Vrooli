import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import WorkspaceManagerDialog from './WorkspaceManagerDialog';
import { usePreviewWorkspaceStore } from '@/features/preview-workspace/state/previewWorkspaceStore';

describe('WorkspaceManagerDialog', () => {
  beforeEach(async () => {
    await usePreviewWorkspaceStore.persist.clearStorage();
    await usePreviewWorkspaceStore.persist.rehydrate();
    usePreviewWorkspaceStore.getState().reset();
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
});
