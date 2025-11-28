import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { VariantProvider, useVariant } from './VariantContext';
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

describe('VariantContext [REQ:AB-URL,AB-STORAGE,AB-API]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
    setLocationSearch('');
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const mockVariant = {
    id: 'test-id',
    slug: 'test-variant',
    name: 'Test Variant',
    description: 'Test description',
    weight: 50,
    status: 'active',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
  };

  const wrapper = ({ children }: { children: ReactNode }) => (
    <VariantProvider>{children}</VariantProvider>
  );

  it('[REQ:AB-URL] should fetch variant from URL parameter', async () => {
    setLocationSearch('?variant=test-variant');

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockVariant,
    });

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Initial state should be loading
    expect(result.current.loading).toBe(true);
    expect(result.current.variant).toBe(null);

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Should have fetched variant by slug from URL
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/variants/test-variant',
      expect.objectContaining({
        headers: { 'Content-Type': 'application/json' },
      })
    );

    expect(result.current.variant).toEqual(mockVariant);
    expect(result.current.error).toBe(null);

    // Should have stored variant in localStorage
    const stored = JSON.parse(
      localStorageMock.getItem('landing_manager_variant') || '{}'
    );
    expect(stored).toEqual(mockVariant);
  });

  it('[REQ:AB-STORAGE] should use stored variant from localStorage', async () => {
    // Store variant in localStorage
    localStorageMock.setItem('landing_manager_variant', JSON.stringify(mockVariant));

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Should NOT have called API (used localStorage)
    expect(global.fetch).not.toHaveBeenCalled();

    // Should have loaded variant from localStorage
    expect(result.current.variant).toEqual(mockVariant);
    expect(result.current.error).toBe(null);
  });

  it('[REQ:AB-API] should select variant via API when no URL or localStorage', async () => {
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockVariant,
    });

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Should have called API select endpoint
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/variants/select',
      expect.objectContaining({
        headers: { 'Content-Type': 'application/json' },
      })
    );

    expect(result.current.variant).toEqual(mockVariant);
    expect(result.current.error).toBe(null);

    // Should have stored variant in localStorage
    const stored = JSON.parse(
      localStorageMock.getItem('landing_manager_variant') || '{}'
    );
    expect(stored).toEqual(mockVariant);
  });

  it('[REQ:AB-URL] should prioritize URL parameter over localStorage', async () => {
    // Store different variant in localStorage
    const storedVariant = { ...mockVariant, slug: 'stored-variant' };
    localStorageMock.setItem('landing_manager_variant', JSON.stringify(storedVariant));

    // Set URL parameter
    setLocationSearch('?variant=url-variant');

    const urlVariant = { ...mockVariant, slug: 'url-variant' };
    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => urlVariant,
    });

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Wait for variant to load
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    // Should have fetched from URL (not used localStorage)
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/variants/url-variant',
      expect.any(Object)
    );

    expect(result.current.variant).toEqual(urlVariant);
  });

  it('should handle API errors gracefully', async () => {
    (global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant).toBe(null);
    expect(result.current.error).toBe('Network error');
  });

  it('should handle 404 errors for invalid variant slugs', async () => {
    setLocationSearch('?variant=invalid-slug');

    (global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 404,
    });

    const { result } = renderHook(() => useVariant(), { wrapper });

    // Wait for error to be set
    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.variant).toBe(null);
    expect(result.current.error).toContain('404');
  });

  it('should support variant_slug parameter for backwards compatibility', async () => {
    setLocationSearch('?variant_slug=test-variant');

    (global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: async () => mockVariant,
    });

    const { result } = renderHook(() => useVariant(), { wrapper });

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/variants/test-variant',
      expect.any(Object)
    );
  });
});
