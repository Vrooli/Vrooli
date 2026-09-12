import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import type React from 'react';
import { MetricsModeContext } from './MetricsModeContext';
import { act, renderHook, waitFor } from '@testing-library/react';
import { useMetrics } from './useMetricsHook';
import type { MetricEvent } from '../api/types';
import type { useLandingVariant } from '../../app/providers/useLandingVariant';
import { getFirstCall } from '../test-utils/api-mocks';

const trackMetricMock = vi.fn<(event: MetricEvent) => Promise<{ success: boolean }>>();
const useLandingVariantMock = vi.fn<() => ReturnType<typeof useLandingVariant>>();

vi.mock('../api', () => ({
  trackMetric: (...args: Parameters<typeof trackMetricMock>) => trackMetricMock(...args),
}));

vi.mock('../../app/providers/useLandingVariant', () => ({
  useLandingVariant: () => useLandingVariantMock(),
}));

describe('useMetrics storage fallbacks [REQ:METRIC-RESILIENCE]', () => {
  const sessionDescriptor = Object.getOwnPropertyDescriptor(window, 'sessionStorage');
  const localDescriptor = Object.getOwnPropertyDescriptor(window, 'localStorage');

  beforeEach(() => {
    trackMetricMock.mockReset();
    useLandingVariantMock.mockReturnValue({
      variant: { id: 99, slug: 'control', name: 'Control' },
      config: null,
      loading: false,
      error: null,
      resolution: 'api_select',
      statusNote: null,
      lastUpdated: Date.now(),
      refresh: vi.fn(),
    });
    if (sessionDescriptor) {
      Object.defineProperty(window, 'sessionStorage', sessionDescriptor);
    }
    if (localDescriptor) {
      Object.defineProperty(window, 'localStorage', localDescriptor);
    }
  });

  afterEach(() => {
    if (sessionDescriptor) {
      Object.defineProperty(window, 'sessionStorage', sessionDescriptor);
    }
    if (localDescriptor) {
      Object.defineProperty(window, 'localStorage', localDescriptor);
    }
  });

  it('falls back to in-memory identifiers when storage is blocked', async () => {
    const throwingStorage: Storage = {
      getItem: () => {
        throw new Error('blocked');
      },
      setItem: () => {
        throw new Error('blocked');
      },
      removeItem: () => {
        throw new Error('blocked');
      },
      clear: () => {
        throw new Error('blocked');
      },
      key: () => null,
      length: 0,
    };

    Object.defineProperty(window, 'sessionStorage', { configurable: true, value: throwingStorage });
    Object.defineProperty(window, 'localStorage', { configurable: true, value: throwingStorage });

    trackMetricMock.mockResolvedValue({ success: true });
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

    const { result } = renderHook(() => useMetrics());

    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalled(); });

    trackMetricMock.mockClear();
    result.current.trackCTAClick('primary');
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalled(); });

    const [event] = getFirstCall(trackMetricMock);
    expect(event.session_id).toMatch(/^session_/);
    expect(event.visitor_id).toMatch(/^visitor_/);
    expect(event.variant_slug).toBe('control');
    expect(warnSpy).toHaveBeenCalled();

    warnSpy.mockRestore();
  });

  it('reuses persisted ids across events when storage works', async () => {
    const memory: Record<string, string> = {};
    const storage: Storage = {
      getItem: (key: string) => memory[key] ?? null,
      setItem: (key: string, value: string) => {
        memory[key] = value;
      },
      removeItem: (key: string) => {
        Reflect.deleteProperty(memory, key);
      },
      clear: () => {
        Object.keys(memory).forEach((key) => { Reflect.deleteProperty(memory, key); });
      },
      key: (index: number) => Object.keys(memory)[index] ?? null,
      length: 0,
    };

    Object.defineProperty(window, 'sessionStorage', { configurable: true, value: storage });
    Object.defineProperty(window, 'localStorage', { configurable: true, value: storage });

    trackMetricMock.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useMetrics());

    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalled(); });
    const [firstEvent] = getFirstCall(trackMetricMock);

    trackMetricMock.mockClear();
    result.current.trackDownload({ platform: 'mac' });
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalled(); });
    const [secondEvent] = getFirstCall(trackMetricMock);

    expect(secondEvent.session_id).toBe(firstEvent.session_id);
    expect(secondEvent.visitor_id).toBe(firstEvent.visitor_id);
  });

  it('continues tracking with generated identifiers when browser storage access itself throws', async () => {
    Object.defineProperty(window, 'sessionStorage', {
      configurable: true,
      get: () => { throw new Error('session storage denied'); },
    });
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      get: () => { throw new Error('local storage denied'); },
    });
    trackMetricMock.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledOnce(); });
    result.current.trackConversion({ source: 'restricted-browser' });
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(2); });
    expect(trackMetricMock.mock.calls[1]?.[0]).toMatchObject({
      event_type: 'conversion', variant_slug: 'control', event_data: { source: 'restricted-browser' },
    });
  });

  it('does not emit metrics in preview mode or when a variant is absent', async () => {
    const previewWrapper = ({ children }: { children: React.ReactNode }) => (
      <MetricsModeContext.Provider value="preview">{children}</MetricsModeContext.Provider>
    );
    renderHook(() => useMetrics(), { wrapper: previewWrapper });
    expect(trackMetricMock).not.toHaveBeenCalled();

    useLandingVariantMock.mockReturnValue({
      variant: null, config: null, loading: false, error: null, resolution: 'unknown', statusNote: null, lastUpdated: null, refresh: vi.fn(),
    });
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {
      return undefined;
    });
    const { result } = renderHook(() => useMetrics());
    result.current.trackConversion({ source: 'test' });
    await Promise.resolve();
    expect(trackMetricMock).not.toHaveBeenCalled();
    expect(warnSpy).toHaveBeenCalledWith('[useMetrics] No variant selected, skipping event tracking');
    warnSpy.mockRestore();
  });

  it('contains tracking transport failures and continues to expose interaction helpers', async () => {
    trackMetricMock.mockRejectedValue(new Error('metrics unavailable'));
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {
      return undefined;
    });
    const { result } = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalled(); });
    result.current.trackFormSubmit('waitlist', { email: 'customer@example.com' });
    result.current.trackDownload({ platform: 'linux' });
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(3); });
    expect(errorSpy).toHaveBeenCalledWith('[useMetrics] Error tracking event:', expect.any(Error));
    errorSpy.mockRestore();
  });

  it('tracks each interaction helper with its semantic event payload', async () => {
    trackMetricMock.mockResolvedValue({ success: true });
    const { result } = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(1); });
    trackMetricMock.mockClear();

    result.current.trackCTAClick('hero-cta', { placement: 'hero' });
    result.current.trackFormSubmit('waitlist', { source: 'footer' });
    result.current.trackConversion({ order_id: 'order_1' });
    result.current.trackDownload({ platform: 'linux' });

    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(4); });
    expect(trackMetricMock.mock.calls.map(([event]) => event.event_type)).toEqual(['click', 'form_submit', 'conversion', 'download']);
    expect(trackMetricMock.mock.calls[0]?.[0].event_data).toEqual({ element_id: 'hero-cta', element_type: 'cta', placement: 'hero' });
    expect(trackMetricMock.mock.calls[1]?.[0].event_data).toEqual({ form_id: 'waitlist', source: 'footer' });
  });

  it('deduplicates page views and scroll-depth bands across concurrent hook consumers, then cleans up', async () => {
    trackMetricMock.mockResolvedValue({ success: true });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 100 });
    Object.defineProperty(document.documentElement, 'scrollHeight', { configurable: true, value: 400 });
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 });

    const first = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(1); });
    const second = renderHook(() => useMetrics());
    await Promise.resolve();
    expect(trackMetricMock).toHaveBeenCalledTimes(1);

    Object.defineProperty(window, 'scrollY', { configurable: true, value: 200 });
    act(() => { window.dispatchEvent(new Event('scroll')); });
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(4); });
    expect(trackMetricMock.mock.calls.slice(1).map(([event]) => event.event_data)).toEqual([
      { depth: 25 }, { depth: 50 }, { depth: 75 },
    ]);

    act(() => { window.dispatchEvent(new Event('scroll')); });
    await Promise.resolve();
    expect(trackMetricMock).toHaveBeenCalledTimes(4);
    first.unmount();
    second.unmount();

    const remount = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(5); });
    remount.unmount();
  });

  it('records the final 100-percent depth band exactly once', async () => {
    trackMetricMock.mockResolvedValue({ success: true });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 100 });
    Object.defineProperty(document.documentElement, 'scrollHeight', { configurable: true, value: 400 });
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 300 });

    const hook = renderHook(() => useMetrics());
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledOnce(); });
    act(() => { window.dispatchEvent(new Event('scroll')); });
    await waitFor(() => { expect(trackMetricMock).toHaveBeenCalledTimes(5); });
    expect(trackMetricMock.mock.calls.slice(1).map(([event]) => event.event_data)).toEqual([
      { depth: 25 }, { depth: 50 }, { depth: 75 }, { depth: 100 },
    ]);
    act(() => { window.dispatchEvent(new Event('scroll')); });
    await Promise.resolve();
    expect(trackMetricMock).toHaveBeenCalledTimes(5);
    hook.unmount();
  });
});
