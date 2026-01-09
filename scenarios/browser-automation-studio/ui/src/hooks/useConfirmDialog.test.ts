import { describe, it, expect, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useConfirmDialog } from './useConfirmDialog';

describe('useConfirmDialog', () => {
  it('initializes with null dialog state', () => {
    const { result } = renderHook(() => useConfirmDialog());
    expect(result.current.dialogState).toBeNull();
  });

  it('opens dialog with confirm() and resolves true on close(true)', async () => {
    const { result } = renderHook(() => useConfirmDialog());

    let confirmPromise: Promise<boolean>;
    act(() => {
      confirmPromise = result.current.confirm({
        title: 'Delete item?',
        message: 'This cannot be undone.',
        confirmLabel: 'Delete',
        danger: true,
      });
    });

    // Dialog should now be open
    expect(result.current.dialogState).not.toBeNull();
    expect(result.current.dialogState?.title).toBe('Delete item?');
    expect(result.current.dialogState?.message).toBe('This cannot be undone.');
    expect(result.current.dialogState?.confirmLabel).toBe('Delete');
    expect(result.current.dialogState?.danger).toBe(true);
    expect(result.current.dialogState?.isOpen).toBe(true);

    // Close with true
    act(() => {
      result.current.close(true);
    });

    // Dialog should be closed
    expect(result.current.dialogState).toBeNull();

    // Promise should resolve to true
    const confirmed = await confirmPromise!;
    expect(confirmed).toBe(true);
  });

  it('resolves false on close(false)', async () => {
    const { result } = renderHook(() => useConfirmDialog());

    let confirmPromise: Promise<boolean>;
    act(() => {
      confirmPromise = result.current.confirm({
        title: 'Confirm?',
      });
    });

    expect(result.current.dialogState?.isOpen).toBe(true);

    act(() => {
      result.current.close(false);
    });

    expect(result.current.dialogState).toBeNull();
    const confirmed = await confirmPromise!;
    expect(confirmed).toBe(false);
  });

  it('cancels previous dialog when opening a new one', async () => {
    const { result } = renderHook(() => useConfirmDialog());

    let firstPromise: Promise<boolean>;
    let secondPromise: Promise<boolean>;

    act(() => {
      firstPromise = result.current.confirm({ title: 'First' });
    });

    expect(result.current.dialogState?.title).toBe('First');

    act(() => {
      secondPromise = result.current.confirm({ title: 'Second' });
    });

    expect(result.current.dialogState?.title).toBe('Second');

    // First promise should resolve to false (cancelled)
    const firstResult = await firstPromise!;
    expect(firstResult).toBe(false);

    // Close second dialog
    act(() => {
      result.current.close(true);
    });

    const secondResult = await secondPromise!;
    expect(secondResult).toBe(true);
  });

  it('uses default labels when not provided', async () => {
    const { result } = renderHook(() => useConfirmDialog());

    act(() => {
      result.current.confirm({ title: 'Minimal' });
    });

    expect(result.current.dialogState?.title).toBe('Minimal');
    expect(result.current.dialogState?.message).toBeUndefined();
    expect(result.current.dialogState?.confirmLabel).toBeUndefined();
    expect(result.current.dialogState?.cancelLabel).toBeUndefined();
    expect(result.current.dialogState?.danger).toBeUndefined();
  });
});
