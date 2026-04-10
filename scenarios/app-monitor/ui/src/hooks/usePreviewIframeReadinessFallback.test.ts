import { renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { usePreviewIframeReadinessFallback } from './usePreviewIframeReadinessFallback';

describe('usePreviewIframeReadinessFallback', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('invokes onReady when iframe document becomes interactive/complete', () => {
    const onReady = vi.fn();
    const iframeRef = {
      current: {
        contentDocument: { readyState: 'complete' },
      } as unknown as HTMLIFrameElement,
    };

    renderHook(() => usePreviewIframeReadinessFallback({
      iframeRef,
      enabled: true,
      onReady,
      intervalMs: 100,
    }));

    vi.advanceTimersByTime(150);
    expect(onReady).toHaveBeenCalledTimes(1);
  });

  it('does not invoke onReady when disabled', () => {
    const onReady = vi.fn();
    const iframeRef = { current: null as HTMLIFrameElement | null };

    renderHook(() => usePreviewIframeReadinessFallback({
      iframeRef,
      enabled: false,
      onReady,
      intervalMs: 100,
    }));

    vi.advanceTimersByTime(500);
    expect(onReady).not.toHaveBeenCalled();
  });
});
