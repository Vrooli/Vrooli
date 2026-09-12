import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import {
  LandingConfigResponseSchema,
  LandingVariantSummarySchema,
  LandingSectionSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/config_pb';
import { LandingVariantProvider, useLandingVariant } from './LandingVariantProvider';
import { getFallbackLandingConfig } from '../../shared/lib/fallbackLandingConfig';
import type { ReactNode } from 'react';

// The provider resolves the landing payload through the proto/Connect api layer
// (getLandingConfig), not raw fetch. Mock that boundary and resolve real
// LandingConfigResponse messages.
const { mockGetLandingConfig } = vi.hoisted(() => ({ mockGetLandingConfig: vi.fn() }));

vi.mock('../../shared/api', () => ({
  getLandingConfig: mockGetLandingConfig,
}));

const makeConfig = (slug: string, fallback = false) =>
  create(LandingConfigResponseSchema, {
    variant: create(LandingVariantSummarySchema, {
      id: 101n,
      slug,
      name: 'Test Variant',
      description: 'Test description',
      axes: { persona: 'ops_leader', jtbd: 'launch_bundle', conversionStyle: 'demo_led' },
    }),
    sections: [create(LandingSectionSchema, { sectionType: 'hero', order: 1, enabled: true, content: {} })],
    downloads: [],
    fallback,
  });

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
  vi.stubGlobal('location', { search });
};

describe('LandingVariantProvider [REQ:AB-URL,AB-STORAGE,AB-API]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    setLocationSearch('');
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  const wrapper = ({ children }: { children: ReactNode }) => <LandingVariantProvider>{children}</LandingVariantProvider>;
  const bakedFallback = getFallbackLandingConfig();

  it('[REQ:AB-URL] should fetch variant from URL parameter', async () => {
    setLocationSearch('?variant=test-variant');

    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('test-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Initial state should be loading
    expect(result.current.loading).toBe(true);
    expect(result.current.variant).toBe(null);

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockGetLandingConfig).toHaveBeenCalledWith('test-variant');

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.config?.variant?.slug).toEqual('test-variant');
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

    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('stored-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockGetLandingConfig).toHaveBeenCalledWith('stored-variant');

    expect(result.current.variant?.slug).toEqual('stored-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('local_storage');
  });

  it('[REQ:AB-API] should select variant via API when no URL or localStorage', async () => {
    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('test-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockGetLandingConfig).toHaveBeenCalledWith(undefined);

    expect(result.current.variant?.slug).toEqual('test-variant');
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('api_select');
    expect(localStorageMock.getItem('landing_manager_variant_slug')).toEqual('test-variant');
  });

  it('[REQ:AB-URL] should prioritize URL parameter over localStorage', async () => {
    localStorageMock.setItem('landing_manager_variant_slug', 'stored-variant');

    setLocationSearch('?variant=url-variant');

    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('url-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockGetLandingConfig).toHaveBeenCalledWith('url-variant');

    expect(result.current.variant?.slug).toEqual('url-variant');
  });

  // [REQ:AB-FALLBACK] Baked fallback is used when landing config fetch fails.
  it('should fall back to baked config when API errors occur', async () => {
    mockGetLandingConfig.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant?.slug).toEqual(bakedFallback.variant?.slug);
    expect(result.current.config?.fallback).toBe(true);
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('fallback');
    expect(result.current.statusNote).toContain('API unavailable');
  });

  // [REQ:AB-FALLBACK] A payload missing its variant also triggers the fallback.
  it('should use fallback config when the payload has no resolved variant', async () => {
    setLocationSearch('?variant=invalid-slug');

    mockGetLandingConfig.mockResolvedValueOnce(
      create(LandingConfigResponseSchema, { sections: [], downloads: [], fallback: false })
    );

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant?.slug).toEqual(bakedFallback.variant?.slug);
    expect(result.current.config?.fallback).toBe(true);
    expect(result.current.error).toBe(null);
    expect(result.current.resolution).toEqual('fallback');
  });

  it('should support variant_slug parameter for backwards compatibility', async () => {
    setLocationSearch('?variant_slug=test-variant');

    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('test-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(mockGetLandingConfig).toHaveBeenCalledWith('test-variant');
    expect(result.current.variant?.slug).toEqual('test-variant');
  });

  it('supports manual refresh to re-sync landing config', async () => {
    mockGetLandingConfig.mockResolvedValue(makeConfig('test-variant'));

    const { result } = renderHook(() => useLandingVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    mockGetLandingConfig.mockResolvedValueOnce(makeConfig('next-variant'));

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockGetLandingConfig).toHaveBeenCalledTimes(2);
    await waitFor(() => {
      expect(result.current.variant?.slug).toEqual('next-variant');
    });
  });
});
