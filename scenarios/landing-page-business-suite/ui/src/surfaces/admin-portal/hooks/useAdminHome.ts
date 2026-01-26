import { useCallback, useEffect, useState } from 'react';
import {
  listVariants,
  getStripeSettings,
  getApiErrorMessage,
  resetDemoData,
  getBranding,
  listDownloadAppsAdmin,
  type StripeSettingsResponse,
} from '../../../shared/api';
import { getAdminExperienceSnapshot, type AdminExperienceSnapshot } from '../../../shared/lib/adminExperience';
import { buildDateRange, fetchAnalyticsSummary } from '../controllers/analyticsController';
import {
  HEALTH_SNAPSHOT_DAYS,
  buildHealthSnapshot,
  computeBrandingHealth,
  computeDownloadsHealth,
  type HealthSnapshot,
  type BrandingHealthStatus,
  type DownloadsHealthStatus,
} from '../services/adminHome.service';

/**
 * Return type for the useAdminHome hook
 */
export interface UseAdminHomeReturn {
  /** Admin experience snapshot for resume functionality */
  experience: AdminExperienceSnapshot | null;

  /** Health snapshot data */
  healthSnapshot: HealthSnapshot | null;
  /** Whether health snapshot is loading */
  healthLoading: boolean;
  /** Health snapshot error message */
  healthError: string | null;
  /** Whether analytics data is degraded */
  healthMetricsDegraded: boolean;
  /** Refresh health snapshot */
  refreshHealthSnapshot: () => Promise<void>;

  /** Stripe settings response */
  stripeSettings: StripeSettingsResponse | null;
  /** Whether stripe settings are loading */
  stripeLoading: boolean;
  /** Stripe settings error message */
  stripeError: string | null;
  /** Refresh stripe status */
  refreshStripeStatus: () => Promise<void>;

  /** Branding health status */
  brandingHealth: BrandingHealthStatus | null;
  /** Whether branding health is loading */
  brandingLoading: boolean;
  /** Refresh branding health */
  refreshBrandingHealth: () => Promise<void>;

  /** Downloads health status */
  downloadsHealth: DownloadsHealthStatus | null;
  /** Whether downloads health is loading */
  downloadsLoading: boolean;
  /** Refresh downloads health */
  refreshDownloadsHealth: () => Promise<void>;

  /** Whether demo data is being reset */
  resettingDemoData: boolean;
  /** Success message after reset */
  resetMessage: string | null;
  /** Error message after reset failure */
  resetError: string | null;
  /** Whether reset confirmation dialog is showing */
  showResetConfirm: boolean;
  /** Show the reset confirmation dialog */
  setShowResetConfirm: (show: boolean) => void;
  /** Handle demo data reset */
  handleResetDemoData: () => Promise<void>;

  /** Build path for resuming variant editing */
  buildResumeVariantPath: () => string | null;
  /** Build path for resuming analytics view */
  buildResumeAnalyticsPath: () => string | null;
}

/**
 * Custom hook for managing admin home page state
 *
 * Encapsulates all state management for the admin home page,
 * including health snapshot loading, Stripe status, branding/downloads
 * health, and demo data reset functionality.
 *
 * @returns Object containing state and handlers
 */
export function useAdminHome(): UseAdminHomeReturn {
  // Experience snapshot for resume functionality
  const [experience, setExperience] = useState<AdminExperienceSnapshot | null>(null);

  // Health snapshot state
  const [healthSnapshot, setHealthSnapshot] = useState<HealthSnapshot | null>(null);
  const [healthLoading, setHealthLoading] = useState(true);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [healthMetricsDegraded, setHealthMetricsDegraded] = useState(false);

  // Stripe settings state
  const [stripeSettings, setStripeSettings] = useState<StripeSettingsResponse | null>(null);
  const [stripeLoading, setStripeLoading] = useState(true);
  const [stripeError, setStripeError] = useState<string | null>(null);

  // Branding health state
  const [brandingHealth, setBrandingHealth] = useState<BrandingHealthStatus | null>(null);
  const [brandingLoading, setBrandingLoading] = useState(true);

  // Downloads health state
  const [downloadsHealth, setDownloadsHealth] = useState<DownloadsHealthStatus | null>(null);
  const [downloadsLoading, setDownloadsLoading] = useState(true);

  // Demo data reset state
  const [resettingDemoData, setResettingDemoData] = useState(false);
  const [resetMessage, setResetMessage] = useState<string | null>(null);
  const [resetError, setResetError] = useState<string | null>(null);
  const [showResetConfirm, setShowResetConfirm] = useState(false);

  // Load experience snapshot on mount
  useEffect(() => {
    setExperience(getAdminExperienceSnapshot());
  }, []);

  /**
   * Refresh health snapshot data
   */
  const refreshHealthSnapshot = useCallback(async () => {
    setHealthLoading(true);
    setHealthError(null);
    setHealthMetricsDegraded(false);
    try {
      const range = buildDateRange(HEALTH_SNAPSHOT_DAYS);
      const [variantPayload, analyticsPayload] = await Promise.all([
        listVariants(),
        fetchAnalyticsSummary(range)
          .then((data) => ({ ok: true as const, data }))
          .catch((error) => ({ ok: false as const, error })),
      ]);

      if (!analyticsPayload.ok) {
        console.warn('Admin health analytics unavailable:', analyticsPayload.error);
        setHealthMetricsDegraded(true);
      }

      setHealthSnapshot(
        buildHealthSnapshot(
          variantPayload.variants,
          analyticsPayload.ok ? analyticsPayload.data : null
        )
      );
    } catch (error) {
      setHealthError(error instanceof Error ? error.message : 'Failed to load admin health snapshot');
      setHealthSnapshot(null);
    } finally {
      setHealthLoading(false);
    }
  }, []);

  // Load health snapshot on mount
  useEffect(() => {
    refreshHealthSnapshot();
  }, [refreshHealthSnapshot]);

  /**
   * Refresh Stripe status
   */
  const refreshStripeStatus = useCallback(async () => {
    setStripeLoading(true);
    setStripeError(null);
    try {
      const data = await getStripeSettings();
      setStripeSettings(data);
    } catch (error) {
      setStripeSettings(null);
      setStripeError(getApiErrorMessage(error, 'Failed to load monetization status'));
    } finally {
      setStripeLoading(false);
    }
  }, []);

  // Load Stripe status on mount
  useEffect(() => {
    refreshStripeStatus();
  }, [refreshStripeStatus]);

  /**
   * Refresh branding health
   */
  const refreshBrandingHealth = useCallback(async () => {
    setBrandingLoading(true);
    try {
      const branding = await getBranding();
      setBrandingHealth(computeBrandingHealth(branding));
    } catch {
      setBrandingHealth(null);
    } finally {
      setBrandingLoading(false);
    }
  }, []);

  // Load branding health on mount
  useEffect(() => {
    refreshBrandingHealth();
  }, [refreshBrandingHealth]);

  /**
   * Refresh downloads health
   */
  const refreshDownloadsHealth = useCallback(async () => {
    setDownloadsLoading(true);
    try {
      const { apps } = await listDownloadAppsAdmin();
      setDownloadsHealth(computeDownloadsHealth(apps));
    } catch {
      setDownloadsHealth(null);
    } finally {
      setDownloadsLoading(false);
    }
  }, []);

  // Load downloads health on mount
  useEffect(() => {
    refreshDownloadsHealth();
  }, [refreshDownloadsHealth]);

  /**
   * Handle demo data reset
   */
  const handleResetDemoData = useCallback(async () => {
    setResettingDemoData(true);
    setResetError(null);
    setResetMessage(null);
    setShowResetConfirm(false);
    try {
      await resetDemoData();
      setResetMessage('Demo data restored to template defaults.');
      await Promise.all([refreshHealthSnapshot(), refreshStripeStatus()]);
    } catch (error) {
      setResetError(error instanceof Error ? error.message : 'Failed to reset demo data');
    } finally {
      setResettingDemoData(false);
    }
  }, [refreshHealthSnapshot, refreshStripeStatus]);

  /**
   * Build path for resuming variant editing
   */
  const buildResumeVariantPath = useCallback((): string | null => {
    const resumeVariant = experience?.lastVariant;
    if (!resumeVariant) return null;

    return resumeVariant.surface === 'section' && resumeVariant.sectionId
      ? `/admin/customization/variants/${resumeVariant.slug}/sections/${resumeVariant.sectionId}`
      : `/admin/customization/variants/${resumeVariant.slug}`;
  }, [experience]);

  /**
   * Build path for resuming analytics view
   */
  const buildResumeAnalyticsPath = useCallback((): string | null => {
    const resumeAnalytics = experience?.lastAnalytics;
    if (!resumeAnalytics) return null;

    const params = new URLSearchParams();
    if (resumeAnalytics.variantSlug) {
      params.set('variant', resumeAnalytics.variantSlug);
    }
    if (resumeAnalytics.timeRangeDays && resumeAnalytics.timeRangeDays !== 7) {
      params.set('range', String(resumeAnalytics.timeRangeDays));
    }
    const query = params.toString() ? `?${params.toString()}` : '';
    return `/admin/analytics${query}`;
  }, [experience]);

  return {
    experience,
    healthSnapshot,
    healthLoading,
    healthError,
    healthMetricsDegraded,
    refreshHealthSnapshot,
    stripeSettings,
    stripeLoading,
    stripeError,
    refreshStripeStatus,
    brandingHealth,
    brandingLoading,
    refreshBrandingHealth,
    downloadsHealth,
    downloadsLoading,
    refreshDownloadsHealth,
    resettingDemoData,
    resetMessage,
    resetError,
    showResetConfirm,
    setShowResetConfirm,
    handleResetDemoData,
    buildResumeVariantPath,
    buildResumeAnalyticsPath,
  };
}
