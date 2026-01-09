import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useMetrics } from './useMetrics';

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
    expect(event.session_id).toMatch(/^session_/);
    expect(event.visitor_id).toMatch(/^visitor_/);
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

    expect(secondEvent.session_id).toBe(firstEvent.session_id);
    expect(secondEvent.visitor_id).toBe(firstEvent.visitor_id);
  });
});
