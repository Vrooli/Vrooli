import React from 'react';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, fireEvent, waitFor } from '@testing-library/react';
import { useMetrics, MetricsModeProvider } from './useMetrics';

const trackMetricMock = vi.fn();
const useLandingVariantMock = vi.fn();

vi.mock('../api', () => ({
  trackMetric: (...args: unknown[]) => trackMetricMock(...args),
}));

vi.mock('../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => useLandingVariantMock(),
}));

describe('useMetrics storage fallbacks [REQ:METRIC-RESILIENCE]', () => {
  const sessionDescriptor = Object.getOwnPropertyDescriptor(window, 'sessionStorage');
  const localDescriptor = Object.getOwnPropertyDescriptor(window, 'localStorage');

  beforeEach(() => {
    trackMetricMock.mockReset();
    useLandingVariantMock.mockReturnValue({
      variant: { id: 99 },
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

    await waitFor(() => expect(trackMetricMock).toHaveBeenCalled());

    trackMetricMock.mockClear();
    result.current.trackCTAClick('primary');
    await waitFor(() => expect(trackMetricMock).toHaveBeenCalled());

    const event = trackMetricMock.mock.calls[0][0];
    expect(event.sessionId).toMatch(/^session_/);
    expect(event.visitorId).toMatch(/^visitor_/);
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
        delete memory[key];
      },
      clear: () => {
        Object.keys(memory).forEach((key) => delete memory[key]);
      },
      key: (index: number) => Object.keys(memory)[index] ?? null,
      length: 0,
    };

    Object.defineProperty(window, 'sessionStorage', { configurable: true, value: storage });
    Object.defineProperty(window, 'localStorage', { configurable: true, value: storage });

    trackMetricMock.mockResolvedValue({ success: true });

    const { result } = renderHook(() => useMetrics());

    await waitFor(() => expect(trackMetricMock).toHaveBeenCalled());
    const firstEvent = trackMetricMock.mock.calls[0][0];

    trackMetricMock.mockClear();
    result.current.trackDownload({ platform: 'mac' });
    await waitFor(() => expect(trackMetricMock).toHaveBeenCalled());
    const secondEvent = trackMetricMock.mock.calls[0][0];

    expect(secondEvent.sessionId).toBe(firstEvent.sessionId);
    expect(secondEvent.visitorId).toBe(firstEvent.visitorId);
  });
});

describe('useMetrics gating and helpers', () => {
  beforeEach(() => {
    trackMetricMock.mockReset();
    trackMetricMock.mockResolvedValue({ success: true });
    useLandingVariantMock.mockReturnValue({ variant: { id: 7 } });
  });

  const previewWrapper = ({ children }: { children: React.ReactNode }) => (
    <MetricsModeProvider mode="preview">{children}</MetricsModeProvider>
  );

  it('skips every tracking helper in preview mode', async () => {
    const { result } = renderHook(() => useMetrics(), { wrapper: previewWrapper });
    act(() => result.current.trackCTAClick('cta'));
    act(() => result.current.trackFormSubmit('form'));
    act(() => result.current.trackConversion({ v: 1 }));
    act(() => result.current.trackDownload({ platform: 'mac' }));
    await act(async () => {
      await result.current.trackEvent('click');
    });
    expect(trackMetricMock).not.toHaveBeenCalled();
  });

  it('skips tracking when no variant is resolved', async () => {
    useLandingVariantMock.mockReturnValue({ variant: null });
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { result } = renderHook(() => useMetrics());
    await act(async () => {
      await result.current.trackEvent('click');
    });
    expect(trackMetricMock).not.toHaveBeenCalled();
    warn.mockRestore();
  });

  it('logs but does not throw when the tracking API rejects', async () => {
    trackMetricMock.mockRejectedValue(new Error('network'));
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { result } = renderHook(() => useMetrics());
    await act(async () => {
      await result.current.trackEvent('conversion', { a: 1 });
    });
    expect(trackMetricMock).toHaveBeenCalled();
    err.mockRestore();
  });

  it('forwards form-submit and conversion helpers as typed events', async () => {
    const { result } = renderHook(() => useMetrics());
    await waitFor(() => expect(trackMetricMock).toHaveBeenCalled());
    trackMetricMock.mockClear();
    act(() => result.current.trackFormSubmit('signup', { plan: 'pro' }));
    act(() => result.current.trackConversion({ amount: 99 }));
    await waitFor(() =>
      expect(trackMetricMock.mock.calls.map((c) => (c[0] as { eventType: string }).eventType)).toEqual(
        expect.arrayContaining(['form_submit', 'conversion']),
      ),
    );
  });

  it('tracks scroll-depth bands on scroll', async () => {
    Object.defineProperty(document.documentElement, 'scrollHeight', { configurable: true, value: 1000 });
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 900 });
    renderHook(() => useMetrics());
    window.scrollY = 200;
    fireEvent.scroll(window);
    await waitFor(() =>
      expect(trackMetricMock).toHaveBeenCalledWith(expect.objectContaining({ eventType: 'scroll_depth' })),
    );
  });
});
