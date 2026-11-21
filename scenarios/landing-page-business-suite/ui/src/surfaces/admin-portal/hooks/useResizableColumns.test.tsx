import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import type { MouseEvent as ReactMouseEvent } from 'react';
import { useResizableColumns } from './useResizableColumns';

describe('useResizableColumns', () => {
  afterEach(() => {
    localStorage.clear();
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  });

  it('uses a valid persisted ratio and exposes complementary column styles', () => {
    localStorage.setItem('columns', '0.6');
    const { result } = renderHook(() => useResizableColumns({ storageKey: 'columns' }));

    expect(result.current.leftRatio).toBe(0.6);
    expect(result.current.leftColumnStyle).toEqual({ width: '60%', flexShrink: 0 });
    expect(result.current.rightColumnStyle).toEqual({ width: '40%', flexShrink: 0 });
  });

  it('falls back to the requested default when persisted values are invalid or out of bounds', () => {
    localStorage.setItem('columns', 'not-a-number');
    const first = renderHook(() => useResizableColumns({ storageKey: 'columns', defaultLeftRatio: 0.45 }));
    expect(first.result.current.leftRatio).toBe(0.45);
    first.unmount();

    localStorage.setItem('columns', '0.9');
    const second = renderHook(() => useResizableColumns({ storageKey: 'columns', defaultLeftRatio: 0.45 }));
    expect(second.result.current.leftRatio).toBe(0.45);
  });

  it('does not begin a drag before a container has been attached', () => {
    const { result } = renderHook(() => useResizableColumns());

    act(() => {
      result.current.handleResizeStart({ preventDefault: () => undefined, clientX: 10 } as ReactMouseEvent);
    });

    expect(result.current.isResizing).toBe(false);
  });

  it('constrains drag movement, persists the result, and restores document interaction state on release', () => {
    const { result } = renderHook(() => useResizableColumns({ storageKey: 'columns' }));
    const container = document.createElement('div');
    Object.defineProperty(container, 'clientWidth', { value: 100 });
    Object.defineProperty(result.current.containerRef, 'current', { value: container });

    act(() => {
      result.current.handleResizeStart({ preventDefault: () => undefined, clientX: 100 } as ReactMouseEvent);
    });
    expect(result.current.isResizing).toBe(true);
    expect(document.body.style.cursor).toBe('col-resize');

    act(() => {
      window.dispatchEvent(new MouseEvent('mousemove', { clientX: 1000 }));
    });
    expect(result.current.leftRatio).toBe(0.75);

    act(() => {
      window.dispatchEvent(new MouseEvent('mousemove', { clientX: -1000 }));
    });
    expect(result.current.leftRatio).toBe(0.25);

    act(() => {
      window.dispatchEvent(new MouseEvent('mouseup'));
    });
    expect(result.current.isResizing).toBe(false);
    expect(document.body.style.cursor).toBe('');
    expect(document.body.style.userSelect).toBe('');
    expect(localStorage.getItem('columns')).toBe('0.25');
  });
});
