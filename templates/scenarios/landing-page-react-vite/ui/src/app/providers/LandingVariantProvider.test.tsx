import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { LandingVariantProvider, useLandingVariant } from './LandingVariantProvider';
import { getFallbackLandingConfig } from '../../shared/lib/fallbackLandingConfig';
import type { ReactNode } from 'react';

// Mock fetch globally
global.fetch = vi.fn();

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value;
    },
    clear: () => {
      store = {};
    },
    removeItem: (key: string) => {
      delete store[key];
    },
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock location search
const setLocationSearch = (search: string) => {
  delete (window as any).location;
  (window as any).location = { search };
};

describe('LandingVariantProvider [REQ:AB-URL,AB-STORAGE,AB-API]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
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

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Initial state should be loading
    expect(result.current.loading).toBe(true);
    expect(result.current.variant).toBe(null);

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=test-variant'),
      expect.any(Object)
    );

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.config?.variant.slug).toEqual('test-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('url_param');
    expect(result.current.statusNote).toContain('URL parameter');
    expect(result.current.lastUpdated).not.toBeNull();

    // Should have stored slug
    const stored = localStorageMock.getItem('landing_manager_variant_slug');
    expect(stored).toEqual('test-variant');
  });

  it('[REQ:AB-STORAGE] should use stored variant from localStorage', async () => {
    localStorageMock.setItem('landing_manager_variant_slug', 'stored-variant');

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ...mockConfig, variant: { ...mockConfig.variant, slug: 'stored-variant' } }),
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=stored-variant'),
      expect.any(Object)
    );

    expect(result.current.variant?.slug).toEqual('stored-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('local_storage');
  });

  it('[REQ:AB-API] should select variant via API when no URL or localStorage', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/landing-config'), expect.any(Object));

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('api_select');
    expect(localStorageMock.getItem('landing_manager_variant_slug')).toEqual('test-variant');
  });

  it('[REQ:AB-URL] should prioritize URL parameter over localStorage', async () => {
    localStorageMock.setItem('landing_manager_variant_slug', 'stored-variant');

    setLocationSearch('?variant=url-variant');

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ...mockConfig, variant: { ...mockConfig.variant, slug: 'url-variant' } }),
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=url-variant'),
      expect.any(Object)
    );

    expect(result.current.variant?.slug).toEqual('url-variant');
  });

  // [REQ:AB-FALLBACK] Baked fallback is used when landing config fetch fails.
  it('should fall back to baked config when API errors occur', async () => {
    (global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

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

    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 404,
      text: async () => 'Not found',
    });

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

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockConfig,
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/landing-config?variant=test-variant'),
      expect.any(Object)
    );
    expect(result.current.variant?.slug).toEqual('test-variant');
  });
  it('supports manual refresh to re-sync landing config', async () => {
    (global.fetch as any).mockResolvedValue({
      ok: true,
      json: async () => mockConfig,
    });

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ...mockConfig, variant: { ...mockConfig.variant, slug: 'next-variant' } }),
    });

    await act(async () => {
      await result.current.refresh();
    });

    expect(global.fetch).toHaveBeenCalledTimes(2);
    await waitFor(() => {
      expect(result.current.variant?.slug).toEqual('next-variant');
    });
  });
});
