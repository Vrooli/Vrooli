import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { useCustomizationPage } from './useCustomizationPage';
import * as customizationController from '../controllers/customizationController';
import { loadVariantEditorData } from '../controllers/variantEditorController';
import type { Variant, AnalyticsSummary, VariantStats } from '../../../shared/api';
import type { ReactNode } from 'react';

const toastSuccessMock = vi.hoisted(() => vi.fn());
const showOperationErrorMock = vi.hoisted(() => vi.fn());
const clearOperationAlertMock = vi.hoisted(() => vi.fn());

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
vi.mock('../../../shared/ui/useToast', () => ({
  useToast: () => ({
    success: toastSuccessMock,
    error: vi.fn(),
  }),
}));

vi.mock('../../../shared/ui/useInlineAlert', () => ({
  useInlineAlert: () => ({
    alert: null,
    showError: showOperationErrorMock,
    clearAlert: clearOperationAlertMock,
  }),
}));

const mockLoadCustomizationData = vi.mocked(customizationController.loadCustomizationData);
const mockLoadAnalyticsSnapshot = vi.mocked(customizationController.loadAnalyticsSnapshot);
const mockHandleArchiveVariant = vi.mocked(customizationController.handleArchiveVariant);
const mockHandleDeleteVariant = vi.mocked(customizationController.handleDeleteVariant);
const mockHandleUpdateWeight = vi.mocked(customizationController.handleUpdateWeight);
const mockLoadVariantEditorData = vi.mocked(loadVariantEditorData);

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
    window.history.replaceState({}, '', '/');
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

  describe('variant highlighting', () => {
    it('focuses the matching variant action without cross-frame scrolling', async () => {
      vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
        callback(0);
        return 1;
      });
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const list = document.createElement('div');
      const editButton = document.createElement('button');
      editButton.dataset.testid = 'edit-variant-target';
      list.append(editButton);
      document.body.append(list);
      Object.defineProperty(result.current.variantListRef, 'current', { value: list });

      act(() => { result.current.highlightVariantInList('target'); });

      expect(editButton).toHaveFocus();
      expect(result.current.variantQuery).toBe('target');
      expect(result.current.attentionOnly).toBe(true);
      list.remove();
      vi.unstubAllGlobals();
    });

    it('applies a URL-requested focus once variants load and consumes the focus parameter', async () => {
      window.history.replaceState({}, '', '/admin/customization?focus=target');
      mockLoadCustomizationData.mockResolvedValue({
        variants: [createMockVariant({ slug: 'target', status: 'active' })],
        error: null,
      });
      vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
        callback(0);
        return 1;
      });
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(result.current.variantQuery).toBe('target');
      });

      expect(result.current.attentionOnly).toBe(true);
      expect(window.location.search).toBe('');
      vi.unstubAllGlobals();
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

    it('surfaces analytics errors and allows a fresh analytics snapshot to replace them', async () => {
      mockLoadAnalyticsSnapshot.mockResolvedValue({ analytics: null, error: 'Metrics temporarily unavailable' });
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });

      await waitFor(() => {
        expect(result.current.analyticsLoading).toBe(false);
      });
      expect(result.current.analyticsError).toBe('Metrics temporarily unavailable');

      const analytics = createMockAnalyticsSummary({ total_visitors: 44 });
      mockLoadAnalyticsSnapshot.mockResolvedValue({ analytics, error: null });
      await act(async () => {
        await result.current.fetchAnalyticsSnapshot();
      });

      expect(result.current.analytics).toEqual(analytics);
      expect(result.current.analyticsError).toBeNull();
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

    it('does not send no-op or unknown weights and restores the draft after a failed update', async () => {
      const mockVariants = [createMockVariant({ id: 1, slug: 'v1', weight: 50, status: 'active' })];
      mockLoadCustomizationData.mockResolvedValue({ variants: mockVariants, error: null });
      mockHandleUpdateWeight.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => { await result.current.persistWeight('missing', 25); });
      await act(async () => { await result.current.persistWeight('v1', 50); });
      expect(mockHandleUpdateWeight).not.toHaveBeenCalled();

      act(() => { result.current.setWeightDraft('v1', 75); });
      await act(async () => { await result.current.persistWeight('v1', 75); });

      expect(mockHandleUpdateWeight).toHaveBeenCalledWith('v1', 75);
      expect(result.current.weightDrafts['v1']).toBe(50);
      expect(result.current.savingWeights['v1']).toBe(false);
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

    it('keeps operators on the page and exposes retryable alerts when archive or delete fails', async () => {
      mockHandleArchiveVariant.mockRejectedValue(new Error('Archive unavailable'));
      mockHandleDeleteVariant.mockRejectedValue('Delete unavailable');
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      await act(async () => {
        await result.current.handleArchive('control');
        await result.current.handleDelete('control');
      });

      expect(showOperationErrorMock).toHaveBeenCalledTimes(2);
      expect(mockLoadCustomizationData).toHaveBeenCalledTimes(1);
    });
  });

  describe('navigation helpers', () => {
    it('navigates to each admin destination and opens the selected public preview', async () => {
      const open = vi.spyOn(window, 'open').mockImplementation(() => null);
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => { result.current.navigateToVariantEditor('control'); });
      expect(window.location.pathname).toBe('/admin/customization/variants/control');
      act(() => { result.current.navigateToAgentCustomization(); });
      expect(window.location.pathname).toBe('/admin/customization/agent');
      act(() => { result.current.navigateToNewVariant(); });
      expect(window.location.pathname).toBe('/admin/customization/variants/new');
      act(() => { result.current.navigateToAnalytics('control'); });
      expect(`${window.location.pathname}${window.location.search}`).toBe('/admin/analytics?variant=control');
      act(() => { result.current.openVariantPreview('control'); });
      expect(open).toHaveBeenCalledWith('/?variant=control', '_blank');
      open.mockRestore();
    });
  });

  describe('section navigation', () => {
    it('uses a direct section ID without loading the variant snapshot', async () => {
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      await act(async () => {
        await expect(result.current.navigateToSectionEditor('control', { sectionId: 42 })).resolves.toBe(true);
      });

      expect(mockLoadVariantEditorData).not.toHaveBeenCalled();
      expect(window.location.pathname).toBe('/admin/customization/variants/control/sections/42');
    });

    it('honors a URL-requested direct section focus and consumes the request after navigation', async () => {
      window.history.replaceState({}, '', '/admin/customization?focus=control&focusSectionId=42&focusSectionType=hero');
      mockLoadCustomizationData.mockResolvedValue({
        variants: [createMockVariant({ slug: 'control', status: 'active' })],
        error: null,
      });
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(window.location.pathname).toBe('/admin/customization/variants/control/sections/42');
      });

      expect(window.location.search).toBe('');
      expect(mockLoadVariantEditorData).not.toHaveBeenCalled();
    });

    it('resolves a URL-requested section type before navigating and clears the completed request', async () => {
      window.history.replaceState({}, '', '/admin/customization?focus=control&focusSectionType=faq');
      mockLoadCustomizationData.mockResolvedValue({
        variants: [createMockVariant({ slug: 'control', status: 'active' })],
        error: null,
      });
      mockLoadVariantEditorData.mockResolvedValue({ variant: createMockVariant({ slug: 'control' }), sections: [{
        id: 8, variant_id: 1, key: 'faq', section_type: 'faq', content: {}, order: 0, enabled: true, created_at: '', updated_at: '',
      }] });
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(window.location.pathname).toBe('/admin/customization/variants/control/sections/8');
      });

      expect(mockLoadVariantEditorData).toHaveBeenCalledWith('control');
      expect(window.location.search).toBe('');
    });

    it('resolves requested section types and falls back safely when a target cannot be found', async () => {
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      mockLoadVariantEditorData.mockResolvedValue({ variant: createMockVariant({ slug: 'control' }), sections: [{
        id: 7, variant_id: 1, key: 'hero', section_type: 'hero', content: {}, order: 0, enabled: true, created_at: '', updated_at: '',
      }] });

      await act(async () => {
        await expect(result.current.navigateToSectionEditor('control', { sectionType: 'hero' })).resolves.toBe(true);
      });
      expect(window.location.pathname).toBe('/admin/customization/variants/control/sections/7');

      mockLoadVariantEditorData.mockResolvedValue({ variant: createMockVariant({ slug: 'control' }), sections: [{
        id: 0, variant_id: 1, key: 'hero', section_type: 'hero', content: {}, order: 0, enabled: true, created_at: '', updated_at: '',
      }] });
      await act(async () => {
        await expect(result.current.navigateToSectionEditor('control', { sectionType: 'missing' })).resolves.toBe(false);
      });
      expect(window.location.pathname).toBe('/admin/customization/variants/control');

      mockLoadVariantEditorData.mockRejectedValue(new Error('Unavailable'));
      await act(async () => {
        await expect(result.current.navigateToSectionEditor('control')).resolves.toBe(false);
      });
      expect(window.location.pathname).toBe('/admin/customization/variants/control');
      expect(consoleError).toHaveBeenCalledWith('Failed to resolve section editor for variant', 'control', expect.any(Error));
      consoleError.mockRestore();
    });

    it('uses the first available section when no section type was requested', async () => {
      const { result } = renderHook(() => useCustomizationPage(), { wrapper });
      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
      mockLoadVariantEditorData.mockResolvedValue({ variant: createMockVariant({ slug: 'control' }), sections: [{
        id: 11, variant_id: 1, key: 'faq', section_type: 'faq', content: {}, order: 0, enabled: true, created_at: '', updated_at: '',
      }] });

      await act(async () => {
        await expect(result.current.navigateToSectionEditor('control')).resolves.toBe(true);
      });

      expect(window.location.pathname).toBe('/admin/customization/variants/control/sections/11');
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
