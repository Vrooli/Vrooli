import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAdminHome } from './useAdminHome';
import type { Variant, StripeSettingsResponse, SiteBranding, DownloadApp } from '../../../shared/api';
import type { AdminExperienceSnapshot } from '../../../shared/lib/adminExperience';

// Mock API calls
const listVariantsMock = vi.fn();
const getStripeSettingsMock = vi.fn();
const resetDemoDataMock = vi.fn();
const getBrandingMock = vi.fn();
const listDownloadAppsAdminMock = vi.fn();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    listVariants: (...args: unknown[]) => listVariantsMock(...args),
    getStripeSettings: (...args: unknown[]) => getStripeSettingsMock(...args),
    resetDemoData: (...args: unknown[]) => resetDemoDataMock(...args),
    getBranding: (...args: unknown[]) => getBrandingMock(...args),
    listDownloadAppsAdmin: (...args: unknown[]) => listDownloadAppsAdminMock(...args),
  };
});

// Mock analytics controller
const fetchAnalyticsSummaryMock = vi.fn();

vi.mock('../controllers/analyticsController', async () => {
  const actual = await vi.importActual<typeof import('../controllers/analyticsController')>(
    '../controllers/analyticsController'
  );
  return {
    ...actual,
    fetchAnalyticsSummary: (...args: unknown[]) => fetchAnalyticsSummaryMock(...args),
  };
});

// Mock admin experience
const getAdminExperienceSnapshotMock = vi.fn();

vi.mock('../../../shared/lib/adminExperience', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/lib/adminExperience')>(
    '../../../shared/lib/adminExperience'
  );
  return {
    ...actual,
    getAdminExperienceSnapshot: () => getAdminExperienceSnapshotMock(),
  };
});

const mockVariant: Variant = {
  id: 1,
  slug: 'test-variant',
  name: 'Test Variant',
  weight: 50,
  status: 'active',
  updated_at: new Date().toISOString(),
  created_at: new Date().toISOString(),
};

const mockStripeSettings: StripeSettingsResponse = {
  publishable_key_set: true,
  secret_key_set: true,
  webhook_secret_set: true,
  source: 'database',
  updated_at: new Date().toISOString(),
};

const mockBranding: SiteBranding = {
  id: 1,
  site_name: 'Test Site',
  logo_url: 'https://example.com/logo.png',
  favicon_url: 'https://example.com/favicon.ico',
  default_title: 'Test Title',
  default_description: 'Test description',
  default_og_image_url: 'https://example.com/og.png',
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

const mockDownloadApp: DownloadApp = {
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  name: 'Test App',
  description: 'A test app',
  platforms: [
    {
      bundle_key: 'test-bundle',
      app_key: 'test-app',
      platform: 'windows',
      artifact_url: 'https://example.com/app.exe',
      release_version: '1.0.0',
      requires_entitlement: false,
    },
  ],
  storefronts: [{ store: 'github', label: 'GitHub', url: 'https://github.com/test/app' }],
  display_order: 0,
};

const mockAnalytics = {
  total_visits: 1000,
  total_conversions: 50,
  conversion_rate: 5.0,
  variant_stats: [
    {
      variant_slug: 'test-variant',
      visits: 500,
      conversions: 25,
      conversion_rate: 5.0,
    },
  ],
};

const mockExperience: AdminExperienceSnapshot = {
  version: 1,
  lastVariant: {
    slug: 'test-variant',
    name: 'Test Variant',
    surface: 'variant',
    lastVisitedAt: new Date().toISOString(),
  },
  lastAnalytics: {
    variantSlug: 'test-variant',
    variantName: 'Test Variant',
    timeRangeDays: 7,
    savedAt: new Date().toISOString(),
  },
};

describe('useAdminHome', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listVariantsMock.mockResolvedValue({ variants: [mockVariant] });
    fetchAnalyticsSummaryMock.mockResolvedValue(mockAnalytics);
    getStripeSettingsMock.mockResolvedValue(mockStripeSettings);
    getBrandingMock.mockResolvedValue(mockBranding);
    listDownloadAppsAdminMock.mockResolvedValue({ apps: [mockDownloadApp] });
    getAdminExperienceSnapshotMock.mockReturnValue(mockExperience);
  });

  describe('initial state', () => {
    it('starts with loading states', () => {
      const { result } = renderHook(() => useAdminHome());

      expect(result.current.healthLoading).toBe(true);
      expect(result.current.stripeLoading).toBe(true);
      expect(result.current.brandingLoading).toBe(true);
      expect(result.current.downloadsLoading).toBe(true);
    });

    it('loads experience snapshot on mount', async () => {
      const { result } = renderHook(() => useAdminHome());

      expect(result.current.experience).toEqual(mockExperience);
      expect(getAdminExperienceSnapshotMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('health snapshot loading', () => {
    it('loads health snapshot on mount', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      expect(listVariantsMock).toHaveBeenCalledTimes(1);
      expect(fetchAnalyticsSummaryMock).toHaveBeenCalledTimes(1);
      expect(result.current.healthSnapshot).not.toBeNull();
      expect(result.current.healthError).toBeNull();
    });

    it('handles health snapshot load error', async () => {
      listVariantsMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      expect(result.current.healthError).toBe('API failure');
      expect(result.current.healthSnapshot).toBeNull();
    });

    it('sets degraded state when analytics fail', async () => {
      fetchAnalyticsSummaryMock.mockRejectedValue(new Error('Analytics unavailable'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      expect(result.current.healthMetricsDegraded).toBe(true);
      expect(result.current.healthSnapshot).not.toBeNull();
    });

    it('refreshes health snapshot on demand', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      listVariantsMock.mockClear();
      fetchAnalyticsSummaryMock.mockClear();

      await act(async () => {
        await result.current.refreshHealthSnapshot();
      });

      expect(listVariantsMock).toHaveBeenCalledTimes(1);
      expect(fetchAnalyticsSummaryMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('stripe settings loading', () => {
    it('loads stripe settings on mount', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.stripeLoading).toBe(false);
      });

      expect(getStripeSettingsMock).toHaveBeenCalledTimes(1);
      expect(result.current.stripeSettings).toEqual(mockStripeSettings);
      expect(result.current.stripeError).toBeNull();
    });

    it('handles stripe settings load error', async () => {
      getStripeSettingsMock.mockRejectedValue(new Error('Stripe API error'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.stripeLoading).toBe(false);
      });

      expect(result.current.stripeError).toBe('Stripe API error');
      expect(result.current.stripeSettings).toBeNull();
    });

    it('refreshes stripe status on demand', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.stripeLoading).toBe(false);
      });

      getStripeSettingsMock.mockClear();

      await act(async () => {
        await result.current.refreshStripeStatus();
      });

      expect(getStripeSettingsMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('branding health loading', () => {
    it('loads branding health on mount', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.brandingLoading).toBe(false);
      });

      expect(getBrandingMock).toHaveBeenCalledTimes(1);
      expect(result.current.brandingHealth).not.toBeNull();
      expect(result.current.brandingHealth?.hasIdentity).toBe(true);
    });

    it('handles branding load error gracefully', async () => {
      getBrandingMock.mockRejectedValue(new Error('Branding API error'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.brandingLoading).toBe(false);
      });

      expect(result.current.brandingHealth).toBeNull();
    });

    it('refreshes branding health on demand', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.brandingLoading).toBe(false);
      });

      getBrandingMock.mockClear();

      await act(async () => {
        await result.current.refreshBrandingHealth();
      });

      expect(getBrandingMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('downloads health loading', () => {
    it('loads downloads health on mount', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.downloadsLoading).toBe(false);
      });

      expect(listDownloadAppsAdminMock).toHaveBeenCalledTimes(1);
      expect(result.current.downloadsHealth).not.toBeNull();
      expect(result.current.downloadsHealth?.hasApps).toBe(true);
    });

    it('handles downloads load error gracefully', async () => {
      listDownloadAppsAdminMock.mockRejectedValue(new Error('Downloads API error'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.downloadsLoading).toBe(false);
      });

      expect(result.current.downloadsHealth).toBeNull();
    });

    it('refreshes downloads health on demand', async () => {
      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.downloadsLoading).toBe(false);
      });

      listDownloadAppsAdminMock.mockClear();

      await act(async () => {
        await result.current.refreshDownloadsHealth();
      });

      expect(listDownloadAppsAdminMock).toHaveBeenCalledTimes(1);
    });
  });

  describe('demo data reset', () => {
    it('handles successful demo data reset', async () => {
      resetDemoDataMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      // Show confirmation dialog
      act(() => {
        result.current.setShowResetConfirm(true);
      });
      expect(result.current.showResetConfirm).toBe(true);

      // Execute reset
      await act(async () => {
        await result.current.handleResetDemoData();
      });

      expect(resetDemoDataMock).toHaveBeenCalledTimes(1);
      expect(result.current.resetMessage).toBe('Demo data restored to template defaults.');
      expect(result.current.resetError).toBeNull();
      expect(result.current.showResetConfirm).toBe(false);
      expect(result.current.resettingDemoData).toBe(false);
    });

    it('handles demo data reset error', async () => {
      resetDemoDataMock.mockRejectedValue(new Error('Reset failed'));

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleResetDemoData();
      });

      expect(result.current.resetError).toBe('Reset failed');
      expect(result.current.resetMessage).toBeNull();
      expect(result.current.resettingDemoData).toBe(false);
    });

    it('refreshes health and stripe after successful reset', async () => {
      resetDemoDataMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      listVariantsMock.mockClear();
      getStripeSettingsMock.mockClear();

      await act(async () => {
        await result.current.handleResetDemoData();
      });

      expect(listVariantsMock).toHaveBeenCalled();
      expect(getStripeSettingsMock).toHaveBeenCalled();
    });

    it('sets resetting state during reset', async () => {
      let resolveReset: () => void;
      resetDemoDataMock.mockReturnValue(
        new Promise<void>((resolve) => {
          resolveReset = resolve;
        })
      );

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      act(() => {
        result.current.handleResetDemoData();
      });

      expect(result.current.resettingDemoData).toBe(true);

      await act(async () => {
        resolveReset!();
      });

      await waitFor(() => {
        expect(result.current.resettingDemoData).toBe(false);
      });
    });
  });

  describe('resume path builders', () => {
    it('builds variant resume path for variant surface', () => {
      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeVariantPath();
      expect(path).toBe('/admin/customization/variants/test-variant');
    });

    it('builds variant resume path for section surface', () => {
      getAdminExperienceSnapshotMock.mockReturnValue({
        version: 1,
        lastVariant: {
          slug: 'test-variant',
          name: 'Test Variant',
          surface: 'section',
          sectionId: 123,
          lastVisitedAt: new Date().toISOString(),
        },
        lastAnalytics: undefined,
      });

      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeVariantPath();
      expect(path).toBe('/admin/customization/variants/test-variant/sections/123');
    });

    it('returns null when no variant to resume', () => {
      getAdminExperienceSnapshotMock.mockReturnValue({
        version: 1,
        lastVariant: undefined,
        lastAnalytics: undefined,
      });

      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeVariantPath();
      expect(path).toBeNull();
    });

    it('builds analytics resume path', () => {
      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeAnalyticsPath();
      expect(path).toBe('/admin/analytics?variant=test-variant');
    });

    it('builds analytics resume path with custom range', () => {
      getAdminExperienceSnapshotMock.mockReturnValue({
        version: 1,
        lastVariant: undefined,
        lastAnalytics: {
          variantSlug: 'test-variant',
          variantName: 'Test Variant',
          timeRangeDays: 30,
          savedAt: new Date().toISOString(),
        },
      });

      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeAnalyticsPath();
      expect(path).toBe('/admin/analytics?variant=test-variant&range=30');
    });

    it('builds analytics resume path without variant', () => {
      getAdminExperienceSnapshotMock.mockReturnValue({
        version: 1,
        lastVariant: undefined,
        lastAnalytics: {
          variantSlug: null,
          variantName: undefined,
          timeRangeDays: 7,
          savedAt: new Date().toISOString(),
        },
      });

      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeAnalyticsPath();
      expect(path).toBe('/admin/analytics');
    });

    it('returns null when no analytics to resume', () => {
      getAdminExperienceSnapshotMock.mockReturnValue({
        version: 1,
        lastVariant: undefined,
        lastAnalytics: undefined,
      });

      const { result } = renderHook(() => useAdminHome());

      const path = result.current.buildResumeAnalyticsPath();
      expect(path).toBeNull();
    });
  });

  describe('confirmation dialog', () => {
    it('toggles reset confirmation dialog', () => {
      const { result } = renderHook(() => useAdminHome());

      expect(result.current.showResetConfirm).toBe(false);

      act(() => {
        result.current.setShowResetConfirm(true);
      });

      expect(result.current.showResetConfirm).toBe(true);

      act(() => {
        result.current.setShowResetConfirm(false);
      });

      expect(result.current.showResetConfirm).toBe(false);
    });
  });

  describe('error handling', () => {
    it('handles non-Error objects in health snapshot', async () => {
      listVariantsMock.mockRejectedValue('String error');

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      expect(result.current.healthError).toBe('Failed to load admin health snapshot');
    });

    it('handles non-Error objects in stripe settings', async () => {
      getStripeSettingsMock.mockRejectedValue('String error');

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.stripeLoading).toBe(false);
      });

      expect(result.current.stripeError).toBe('String error');
    });

    it('handles non-Error objects in demo reset', async () => {
      resetDemoDataMock.mockRejectedValue('String error');

      const { result } = renderHook(() => useAdminHome());

      await waitFor(() => {
        expect(result.current.healthLoading).toBe(false);
      });

      await act(async () => {
        await result.current.handleResetDemoData();
      });

      expect(result.current.resetError).toBe('Failed to reset demo data');
    });
  });
});
