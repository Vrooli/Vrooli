import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { LandingVariantProvider } from './LandingVariantProvider';
import { useLandingVariant } from './useLandingVariant';
import { getFallbackLandingConfig } from '../../shared/lib/fallbackLandingConfig';
import type { ReactNode } from 'react';
import { createFetchMock, installFetchMock, mockResponses } from '../../shared/test-utils/api-mocks';

const setLocationSearch = (search: string) => {
  const url = new URL(window.location.href);
  url.search = search;
  window.history.replaceState({}, '', url);
};

describe('LandingVariantProvider [REQ:AB-URL,AB-API]', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
    setLocationSearch('');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const mockConfig = {
    variant: {
      id: 101,
      slug: 'test-variant',
      name: 'Test Variant',
      description: 'Test description',
      axes: {
        persona: 'ops_leader',
        jtbd: 'launch_bundle',
        conversionStyle: 'demo_led',
      },
    },
    sections: [
      { id: 1, section_type: 'hero', content: {}, order: 1, enabled: true },
    ],
    pricing: undefined,
    downloads: [],
    fallback: false,
  };

  const wrapper = ({ children }: { children: ReactNode }) => <LandingVariantProvider>{children}</LandingVariantProvider>;
  const bakedFallback = getFallbackLandingConfig();

  it('[REQ:AB-URL] should fetch variant from URL parameter', async () => {
    setLocationSearch('?variant=test-variant');

    fetchMock.mockResolvedValueOnce(mockResponses.success(mockConfig));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Initial state should be loading
    expect(result.current.loading).toBe(true);
    expect(result.current.variant).toBe(null);

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=test-variant'),
      expect.any(Object)
    );

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.config?.variant.slug).toEqual('test-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('url_param');
    expect(result.current.statusNote).toContain('URL parameter');
    expect(result.current.lastUpdated).not.toBeNull();
  });

  it('[REQ:AB-API] should select variant via API when no URL param provided', async () => {
    fetchMock.mockResolvedValueOnce(mockResponses.success(mockConfig));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(fetchMock).toHaveBeenCalledWith(expect.stringContaining('/api/v1/landing-config'), expect.any(Object));

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('api_select');
  });

  it('[REQ:AB-URL] should prioritize URL parameter over default API selection', async () => {
    setLocationSearch('?variant=url-variant');

    fetchMock.mockResolvedValueOnce(
      mockResponses.success({ ...mockConfig, variant: { ...mockConfig.variant, slug: 'url-variant' } })
    );

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=url-variant'),
      expect.any(Object)
    );

    expect(result.current.variant?.slug).toEqual('url-variant');
  });

  // [REQ:AB-FALLBACK] Baked fallback is used when landing config fetch fails.
  it('should fall back to baked config when API errors occur', async () => {
    fetchMock.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant?.slug).toEqual(bakedFallback.variant.slug);
    expect(result.current.config?.fallback).toBe(true);
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('fallback');
    expect(result.current.statusNote).toContain('API unavailable');
  });

  // [REQ:AB-FALLBACK] Invalid slugs also trigger the fallback configuration.
  it('should use fallback config for invalid variant slugs', async () => {
    setLocationSearch('?variant=invalid-slug');

    fetchMock.mockResolvedValueOnce(mockResponses.error(404, 'Not found'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant?.slug).toEqual(bakedFallback.variant.slug);
    expect(result.current.config?.fallback).toBe(true);
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('fallback');
  });

  it('should support variant_slug parameter for backwards compatibility', async () => {
    setLocationSearch('?variant_slug=test-variant');

    fetchMock.mockResolvedValueOnce(mockResponses.success(mockConfig));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=test-variant'),
      expect.any(Object)
    );
    expect(result.current.variant?.slug).toEqual('test-variant');
  });
  it('supports manual refresh to re-sync landing config', async () => {
    fetchMock.mockResolvedValue(mockResponses.success(mockConfig));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    fetchMock.mockResolvedValueOnce(
      mockResponses.success({ ...mockConfig, variant: { ...mockConfig.variant, slug: 'next-variant' } })
    );

    await act(async () => {
      await result.current.refresh();
    });

    expect(fetchMock).toHaveBeenCalledTimes(2);
    await waitFor(() => {
      expect(result.current.variant?.slug).toEqual('next-variant');
    });
  });
});
