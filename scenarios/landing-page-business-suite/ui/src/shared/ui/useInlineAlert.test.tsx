import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/common';
import { useInlineAlert } from './useInlineAlert';

describe('useInlineAlert', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows, replaces, and clears an explicit alert', () => {
    const { result } = renderHook(() => useInlineAlert());
    const retry = vi.fn();

    act(() => {
      result.current.showAlert({ message: 'First', severity: 'warning' });
    });
    expect(result.current.alert).toMatchObject({ message: 'First', severity: 'warning' });

    act(() => {
      result.current.showAlert({ message: 'Second', severity: 'error', retryable: true, onRetry: retry });
    });
    expect(result.current.alert).toMatchObject({ message: 'Second', severity: 'error', retryable: true, onRetry: retry });

    act(() => {
      result.current.clearAlert();
    });
    expect(result.current.alert).toBeNull();
  });

  it('auto-dismisses only when configured with a positive duration', () => {
    vi.useFakeTimers();
    const { result: disabled } = renderHook(() => useInlineAlert({ autoDismissMs: 0 }));
    act(() => { disabled.current.showWarning('Stays visible'); });
    act(() => { vi.advanceTimersByTime(10_000); });
    expect(disabled.current.alert).toMatchObject({ message: 'Stays visible' });

    const { result: enabled } = renderHook(() => useInlineAlert({ autoDismissMs: 250 }));
    act(() => { enabled.current.showSuccess('Saved', 'Complete'); });
    expect(enabled.current.alert).toMatchObject({ message: 'Saved', title: 'Complete', severity: 'success' });
    act(() => { vi.advanceTimersByTime(249); });
    expect(enabled.current.alert).not.toBeNull();
    act(() => { vi.advanceTimersByTime(1); });
    expect(enabled.current.alert).toBeNull();
  });

  it('maps API, native, string, and unknown errors to safe alert state', () => {
    const retry = vi.fn();
    const { result } = renderHook(() => useInlineAlert());

    act(() => {
      result.current.showError(new ApiError('Network request failed', 'network', 503, 'Try again'), retry);
    });
    expect(result.current.alert).toMatchObject({ message: 'Try again', severity: 'error', retryable: true, onRetry: retry });

    act(() => { result.current.showError(new ApiError('Denied', 'forbidden')); });
    expect(result.current.alert).toMatchObject({ message: "You don't have permission to perform this action.", retryable: false });

    act(() => { result.current.showError(new Error('Native failure')); });
    expect(result.current.alert).toMatchObject({ message: 'Native failure', retryable: false });

    act(() => { result.current.showError('Plain failure'); });
    expect(result.current.alert).toMatchObject({ message: 'Plain failure', retryable: false });

    act(() => { result.current.showError(null); });
    expect(result.current.alert).toMatchObject({ message: 'An unexpected error occurred', retryable: false });
  });

  it('keeps warning metadata intact', () => {
    const { result } = renderHook(() => useInlineAlert());
    act(() => { result.current.showWarning('Check configuration', 'Needs attention'); });
    expect(result.current.alert).toEqual({ message: 'Check configuration', title: 'Needs attention', severity: 'warning' });
  });
});
