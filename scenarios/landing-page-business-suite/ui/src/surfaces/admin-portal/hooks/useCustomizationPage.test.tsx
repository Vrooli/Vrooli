import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { useCustomizationPage } from './useCustomizationPage';
import * as customizationController from '../controllers/customizationController';
import type { Variant, AnalyticsSummary, VariantStats } from '../../../shared/api';
import type { ReactNode } from 'react';

// Mock the controller module
vi.mock('../controllers/customizationController', async () => {
  const actual = await vi.importActual('../controllers/customizationController');
  return {
    ...actual,
    loadCustomizationData: vi.fn(),
    loadAnalyticsSnapshot: vi.fn(),
    handleArchiveVariant: vi.fn(),
    handleDeleteVariant: vi.fn(),
    handleUpdateWeight: vi.fn(),
  };
});

// Mock variantEditorController
vi.mock('../controllers/variantEditorController', () => ({
  loadVariantEditorData: vi.fn().mockResolvedValue({ sections: [] }),
}));

// Mock toast and inline alert
vi.mock('../../../shared/ui/Toast', () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
  }),
}));

vi.mock('../../../shared/ui/InlineAlert', () => ({
  useInlineAlert: () => ({
    alert: null,
    showError: vi.fn(),
    clearAlert: vi.fn(),
  }),
}));

const mockLoadCustomizationData = vi.mocked(customizationController.loadCustomizationData);
const mockLoadAnalyticsSnapshot = vi.mocked(customizationController.loadAnalyticsSnapshot);
const mockHandleArchiveVariant = vi.mocked(customizationController.handleArchiveVariant);
const mockHandleDeleteVariant = vi.mocked(customizationController.handleDeleteVariant);
const mockHandleUpdateWeight = vi.mocked(customizationController.handleUpdateWeight);

const createMockVariant = (overrides: Partial<Variant> = {}): Variant => ({
  id: 1,
  slug: 'test-variant',
  name: 'Test Variant',
  status: 'active',
  weight: 50,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-10T00:00:00Z',
  ...overrides,
});

const createMockVariantStats = (overrides: Partial<VariantStats> = {}): VariantStats => ({
  variant_id: 1,
  variant_slug: 'test-variant',
  variant_name: 'Test Variant',
  views: 1000,
  cta_clicks: 100,
  conversions: 50,
  downloads: 25,
  conversion_rate: 5.0,
  trend: 'up',
  ...overrides,
});

const createMockAnalyticsSummary = (overrides: Partial<AnalyticsSummary> = {}): AnalyticsSummary => ({
  total_visitors: 2000,
  variant_stats: [],
  ...overrides,
});

function wrapper({ children }: { children: ReactNode }) {
  return <BrowserRouter>{children}</BrowserRouter>;
}

describe('useCustomizationPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLoadCustomizationData.mockResolvedValue({ variants: [], error: null });
    mockLoadAnalyticsSnapshot.mockResolvedValue({ analytics: null, error: null });
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      expect(result.current.loading).toBe(true);
      await waitFor(() => { expect(result.current.loading).toBe(false); });
    });

    it('has empty variants initially', async () => {
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.variants).toEqual([]);
    });

    it('has default filter values', async () => {
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.variantQuery).toBe('');
      expect(result.current.attentionOnly).toBe(false);
    });
  });

  describe('loading data', () => {
    it('fetches variants and analytics on mount', async () => {
      const mockVariants = [createMockVariant({ id: 1 }), createMockVariant({ id: 2 })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.variants).toHaveLength(2);
      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(1);
      expect(mockLoadAnalyticsSnapshot).toHaveBeenCalledTimes(1);
    });

    it('handles fetch error', async () => {
      mockLoadCustomizationData.mockResolvedValue({ variants: [], error: 'Network error' });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.error).toBe('Network error');
    });

    it('can reload variants', async () => {
      mockLoadCustomizationData.mockResolvedValue({ variants: [], error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.fetchVariants();
      });

      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(2);
    });
  });

  describe('variant filtering', () => {
    it('separates active and archived variants', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'active-1', status: 'active' }),
        createMockVariant({ id: 2, slug: 'active-2', status: 'active' }),
        createMockVariant({ id: 3, slug: 'archived-1', status: 'archived' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.activeVariants).toHaveLength(2);
      expect(result.current.archivedVariants).toHaveLength(1);
    });

    it('filters by query', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'hero-test', name: 'Hero Test', status: 'active' }),
        createMockVariant({ id: 2, slug: 'footer-test', name: 'Footer Test', status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.filteredActiveVariants).toHaveLength(2);

      act(() => {
        result.current.setVariantQuery('hero');
      });

      expect(result.current.filteredActiveVariants).toHaveLength(1);
      expect(result.current.filteredActiveVariants[0]?.slug).toBe('hero-test');
    });

    it('clears filters', async () => {
      mockLoadCustomizationData.mockResolvedValue({
        variants: [createMockVariant({ status: 'active' })],
        error: null,
      });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setVariantQuery('test');
        result.current.setAttentionOnly(true);
      });

      expect(result.current.variantQuery).toBe('test');
      expect(result.current.attentionOnly).toBe(true);

      act(() => {
        result.current.clearVariantFilters();
      });

      expect(result.current.variantQuery).toBe('');
      expect(result.current.attentionOnly).toBe(false);
    });
  });

  describe('weight management', () => {
    it('calculates total weight correctly', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'v1', weight: 30, status: 'active' }),
        createMockVariant({ id: 2, slug: 'v2', weight: 40, status: 'active' }),
        createMockVariant({ id: 3, slug: 'v3', weight: 30, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.totalAssignedWeight).toBe(100);
      expect(result.current.weightStatus).toBe('balanced');
    });

    it('detects under-allocated weights', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'v1', weight: 20, status: 'active' }),
        createMockVariant({ id: 2, slug: 'v2', weight: 30, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.totalAssignedWeight).toBe(50);
      expect(result.current.weightStatus).toBe('under');
    });

    it('detects over-allocated weights', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'v1', weight: 60, status: 'active' }),
        createMockVariant({ id: 2, slug: 'v2', weight: 60, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.totalAssignedWeight).toBe(120);
      expect(result.current.weightStatus).toBe('over');
    });

    it('normalizes traffic share correctly', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'v1', weight: 50, status: 'active' }),
        createMockVariant({ id: 2, slug: 'v2', weight: 50, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.normalizeShare(50)).toBe(50);
    });

    it('uses even share mode when total weight is zero', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'v1', weight: 0, status: 'active' }),
        createMockVariant({ id: 2, slug: 'v2', weight: 0, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.trafficShareMode).toBe('even');
    });

    it('sets weight draft', async () => {
      const mockVariants = [createMockVariant({ id: 1, slug: 'v1', weight: 50, status: 'active' })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setWeightDraft('v1', 75);
      });

      expect(result.current.weightDrafts['v1']).toBe(75);
    });

    it('persists weight successfully', async () => {
      const mockVariants = [createMockVariant({ id: 1, slug: 'v1', weight: 50, status: 'active' })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });
      mockHandleUpdateWeight.mockResolvedValue(undefined);

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await result.current.persistWeight('v1', 75);
      });

      expect(mockHandleUpdateWeight).toHaveBeenCalledWith('v1', 75);
    });
  });

  describe('attention tracking', () => {
    it('finds stale variants', async () => {
      const oldDate = new Date();
      oldDate.setDate(oldDate.getDate() - 15); // 15 days ago

      const mockVariants = [
        createMockVariant({ id: 1, slug: 'stale', updated_at: oldDate.toISOString(), status: 'active' }),
        createMockVariant({ id: 2, slug: 'fresh', updated_at: new Date().toISOString(), status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.staleVariants.length).toBeGreaterThanOrEqual(1);
      expect(result.current.staleVariants.some((s) => s.variant.slug === 'stale')).toBe(true);
    });

    it('finds never updated variants', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'never-updated', updated_at: undefined, status: 'active' }),
        createMockVariant({ id: 2, slug: 'updated', updated_at: new Date().toISOString(), status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.neverUpdatedVariants).toHaveLength(1);
      expect(result.current.neverUpdatedVariants[0]?.slug).toBe('never-updated');
    });

    it('finds underperforming variant', async () => {
      const mockVariants = [
        createMockVariant({ id: 1, slug: 'good', status: 'active' }),
        createMockVariant({ id: 2, slug: 'bad', status: 'active' }),
      ];
      const mockAnalytics = createMockAnalyticsSummary({
        variant_stats: [
          createMockVariantStats({ variant_slug: 'good', conversion_rate: 10.0 }),
          createMockVariantStats({ variant_slug: 'bad', conversion_rate: 2.0 }),
        ],
      });
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });
      mockLoadAnalyticsSnapshot.mockResolvedValue({ analytics: mockAnalytics, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      await waitFor(() => { expect(result.current.analyticsLoading).toBe(false); });

      expect(result.current.underperformingInfo?.stats.variant_slug).toBe('bad');
    });

    it('builds attention candidates set', async () => {
      const oldDate = new Date();
      oldDate.setDate(oldDate.getDate() - 15);

      const mockVariants = [
        createMockVariant({ id: 1, slug: 'stale', updated_at: oldDate.toISOString(), status: 'active' }),
        createMockVariant({ id: 2, slug: 'never-updated', updated_at: undefined, status: 'active' }),
      ];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.attentionCandidateSlugs.has('stale')).toBe(true);
      expect(result.current.attentionCandidateSlugs.has('never-updated')).toBe(true);
    });
  });

  describe('variant operations', () => {
    it('archives variant successfully', async () => {
      const mockVariants = [createMockVariant({ id: 1, slug: 'to-archive', status: 'active' })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });
      mockHandleArchiveVariant.mockResolvedValue(undefined);

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await result.current.handleArchive('to-archive');
      });

      expect(mockHandleArchiveVariant).toHaveBeenCalledWith('to-archive');
      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(2); // initial + after archive
    });

    it('deletes variant successfully', async () => {
      const mockVariants = [createMockVariant({ id: 1, slug: 'to-delete', status: 'archived' })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });
      mockHandleDeleteVariant.mockResolvedValue(undefined);

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await result.current.handleDelete('to-delete');
      });

      expect(mockHandleDeleteVariant).toHaveBeenCalledWith('to-delete');
      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(2);
    });
  });

  describe('stats map', () => {
    it('builds stats map by slug', async () => {
      const mockAnalytics = createMockAnalyticsSummary({
        variant_stats: [
          createMockVariantStats({ variant_slug: 'variant-a', views: 100 }),
          createMockVariantStats({ variant_slug: 'variant-b', views: 200 }),
        ],
      });
      mockLoadAnalyticsSnapshot.mockResolvedValue({ analytics: mockAnalytics, error: null });

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.analyticsLoading).toBe(false); });

      expect(result.current.statsBySlug.get('variant-a')?.views).toBe(100);
      expect(result.current.statsBySlug.get('variant-b')?.views).toBe(200);
    });
  });
});
