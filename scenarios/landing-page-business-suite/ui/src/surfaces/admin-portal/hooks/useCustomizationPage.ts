import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { Variant, VariantStats, AnalyticsSummary } from '../../../shared/api';
import { useInlineAlert } from '../../../shared/ui/useInlineAlert';
import { useToast } from '../../../shared/ui/useToast';
import { loadVariantEditorData } from '../controllers/variantEditorController';
import {
  type WeightStatus,
  type StaleVariantEntry,
  type TrafficShareMode,
  type UnderperformingInfo,
  SNAPSHOT_DAYS,
  loadCustomizationData,
  loadAnalyticsSnapshot,
  handleArchiveVariant,
  handleDeleteVariant,
  handleUpdateWeight,
  filterActiveVariants,
  filterArchivedVariants,
  getVariantWeight,
  calculateTotalWeight,
  determineWeightStatus,
  determineTrafficShareMode,
  normalizeTrafficShare,
  findStaleVariants,
  findNeverUpdatedVariants,
  findUnderperformingVariant,
  getAttentionCandidateSlugs,
  buildAttentionReasonsMap,
  filterVariantsByQuery,
  buildStatsMap,
} from '../controllers/customizationController';

export interface UseCustomizationPageReturn {
  // Core data
  variants: Variant[];
  activeVariants: Variant[];
  archivedVariants: Variant[];
  filteredActiveVariants: Variant[];
  analytics: AnalyticsSummary | null;

  // Weight management
  weightDrafts: Record<string, number>;
  savingWeights: Record<string, boolean>;
  totalAssignedWeight: number;
  weightStatus: WeightStatus;
  trafficShareMode: TrafficShareMode;

  // Attention tracking
  staleVariants: StaleVariantEntry[];
  neverUpdatedVariants: Variant[];
  underperformingInfo: UnderperformingInfo | null;
  attentionCandidateSlugs: Set<string>;
  variantAttentionReasons: Map<string, string[]>;

  // Stats
  statsBySlug: Map<string, VariantStats>;

  // Filter state
  variantQuery: string;
  attentionOnly: boolean;

  // Loading/error state
  loading: boolean;
  error: string | null;
  analyticsLoading: boolean;
  analyticsError: string | null;

  // Inline alert
  operationAlert: ReturnType<typeof useInlineAlert>['alert'];
  clearOperationAlert: () => void;

  // Ref for scroll
  variantListRef: React.RefObject<HTMLDivElement>;

  // Constants
  snapshotDays: number;

  // Actions
  fetchVariants: () => Promise<void>;
  fetchAnalyticsSnapshot: () => Promise<void>;
  handleArchive: (slug: string) => Promise<void>;
  handleDelete: (slug: string) => Promise<void>;
  persistWeight: (slug: string, nextWeight: number) => Promise<void>;
  setWeightDraft: (slug: string, weight: number) => void;

  // Filter actions
  setVariantQuery: (query: string) => void;
  setAttentionOnly: (value: boolean) => void;
  clearVariantFilters: () => void;

  // Computed helpers
  getWeight: (variant: Variant) => number;
  normalizeShare: (weight: number) => number;

  // Navigation helpers
  highlightVariantInList: (slug?: string) => void;
  navigateToVariantEditor: (slug: string) => void;
  navigateToSectionEditor: (slug: string, options?: { sectionId?: number; sectionType?: string }) => Promise<boolean>;
  navigateToAgentCustomization: () => void;
  navigateToNewVariant: () => void;
  navigateToAnalytics: (slug: string) => void;
  openVariantPreview: (slug: string) => void;
}

/**
 * Hook for customization page state management
 *
 * Wraps customizationController functions with React state
 * and provides actions for variant management operations.
 */
export function useCustomizationPage(): UseCustomizationPageReturn {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const toast = useToast();
  const { alert: operationAlert, showError: showOperationError, clearAlert: clearOperationAlert } = useInlineAlert({ autoDismissMs: 8000 });

  // Core state
  const [variants, setVariants] = useState<Variant[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Analytics state
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null);
  const [analyticsLoading, setAnalyticsLoading] = useState(true);
  const [analyticsError, setAnalyticsError] = useState<string | null>(null);

  // Filter state
  const [variantQuery, setVariantQuery] = useState('');
  const [attentionOnly, setAttentionOnly] = useState(false);

  // Weight state
  const [weightDrafts, setWeightDrafts] = useState<Record<string, number>>({});
  const [savingWeights, setSavingWeights] = useState<Record<string, boolean>>({});

  // Focus tracking
  const [appliedFocusSlug, setAppliedFocusSlug] = useState<string | null>(null);
  const [appliedSectionFocusSlug, setAppliedSectionFocusSlug] = useState<string | null>(null);
  const variantListRef = useRef<HTMLDivElement>(null);

  // URL params
  const focusSlug = searchParams.get('focus');
  const focusSectionIdParam = searchParams.get('focusSectionId');
  const focusSectionType = searchParams.get('focusSectionType');
  const focusSectionId = focusSectionIdParam ? Number(focusSectionIdParam) : null;

  // Fetch variants
  const fetchVariants = useCallback(async () => {
    setLoading(true);
    const result = await loadCustomizationData();
    setVariants(result.variants);
    setError(result.error);
    setLoading(false);
  }, []);

  // Fetch analytics snapshot
  const fetchAnalyticsSnapshot = useCallback(async () => {
    setAnalyticsLoading(true);
    const result = await loadAnalyticsSnapshot();
    setAnalytics(result.analytics);
    setAnalyticsError(result.error);
    setAnalyticsLoading(false);
  }, []);

  // Initial load
  useEffect(() => {
    fetchVariants();
    fetchAnalyticsSnapshot();
  }, [fetchVariants, fetchAnalyticsSnapshot]);

  // Derived: active/archived variants
  const activeVariants = useMemo(() => filterActiveVariants(variants), [variants]);
  const archivedVariants = useMemo(() => filterArchivedVariants(variants), [variants]);

  // Sync weight drafts when variants change
  useEffect(() => {
    const drafts: Record<string, number> = {};
    activeVariants.forEach((v) => {
      drafts[v.slug] = v.weight ?? 0;
    });
    setWeightDrafts(drafts);
  }, [activeVariants]);

  // Weight helpers
  const getWeight = useCallback(
    (variant: Variant) => getVariantWeight(variant, weightDrafts),
    [weightDrafts]
  );

  const totalAssignedWeight = useMemo(
    () => calculateTotalWeight(activeVariants, weightDrafts),
    [activeVariants, weightDrafts]
  );

  const weightStatus = useMemo(
    () => determineWeightStatus(activeVariants.length, totalAssignedWeight),
    [activeVariants.length, totalAssignedWeight]
  );

  const trafficShareMode = useMemo(
    () => determineTrafficShareMode(totalAssignedWeight),
    [totalAssignedWeight]
  );

  const normalizeShare = useCallback(
    (weight: number) => normalizeTrafficShare(weight, totalAssignedWeight, activeVariants.length, trafficShareMode),
    [totalAssignedWeight, activeVariants.length, trafficShareMode]
  );

  // Stats map
  const statsBySlug = useMemo(
    () => buildStatsMap(analytics?.variant_stats),
    [analytics]
  );

  // Attention tracking
  const staleVariants = useMemo(
    () => findStaleVariants(activeVariants),
    [activeVariants]
  );

  const neverUpdatedVariants = useMemo(
    () => findNeverUpdatedVariants(activeVariants),
    [activeVariants]
  );

  const underperformingInfo = useMemo(
    () => findUnderperformingVariant(analytics?.variant_stats, activeVariants),
    [analytics, activeVariants]
  );

  const attentionCandidateSlugs = useMemo(
    () => getAttentionCandidateSlugs(staleVariants, neverUpdatedVariants, underperformingInfo?.stats.variant_slug),
    [staleVariants, neverUpdatedVariants, underperformingInfo]
  );

  const variantAttentionReasons = useMemo(
    () => buildAttentionReasonsMap(staleVariants, neverUpdatedVariants, underperformingInfo?.stats.variant_slug),
    [staleVariants, neverUpdatedVariants, underperformingInfo]
  );

  // Filtered variants
  const filteredActiveVariants = useMemo(
    () => filterVariantsByQuery(activeVariants, variantQuery, attentionOnly, attentionCandidateSlugs),
    [activeVariants, variantQuery, attentionOnly, attentionCandidateSlugs]
  );

  // Archive handler
  const handleArchive = useCallback(async (slug: string) => {
    try {
      await handleArchiveVariant(slug);
      await fetchVariants();
      toast.success(`Variant "${slug}" archived`, 'Variant archived');
    } catch (err) {
      showOperationError(err, () => handleArchive(slug));
    }
  }, [fetchVariants, toast, showOperationError]);

  // Delete handler
  const handleDelete = useCallback(async (slug: string) => {
    try {
      await handleDeleteVariant(slug);
      await fetchVariants();
      toast.success(`Variant "${slug}" permanently deleted`, 'Variant deleted');
    } catch (err) {
      showOperationError(err, () => handleDelete(slug));
    }
  }, [fetchVariants, toast, showOperationError]);

  // Weight persistence
  const persistWeight = useCallback(
    async (slug: string, nextWeight: number) => {
      if (savingWeights[slug]) return;
      const currentVariant = variants.find((v) => v.slug === slug);
      if (!currentVariant) return;
      if (currentVariant.weight === nextWeight) return;

      setSavingWeights((prev) => ({ ...prev, [slug]: true }));
      try {
        await handleUpdateWeight(slug, nextWeight);
        setVariants((prev) =>
          prev.map((v) => (v.slug === slug ? { ...v, weight: nextWeight } : v))
        );
        toast.success(`Traffic weight updated to ${nextWeight}%`, 'Weight saved');
      } catch (err) {
        showOperationError(err, () => persistWeight(slug, nextWeight));
        setWeightDrafts((prev) => ({
          ...prev,
          [slug]: currentVariant.weight ?? 0,
        }));
      } finally {
        setSavingWeights((prev) => ({ ...prev, [slug]: false }));
      }
    },
    [savingWeights, variants, showOperationError, toast]
  );

  // Set weight draft
  const setWeightDraft = useCallback((slug: string, weight: number) => {
    setWeightDrafts((prev) => ({ ...prev, [slug]: weight }));
  }, []);

  // Clear filters
  const clearVariantFilters = useCallback(() => {
    setAttentionOnly(false);
    setVariantQuery('');
  }, []);

  // Highlight variant in list
  const highlightVariantInList = useCallback((slug?: string) => {
    if (!slug) return;
    setAttentionOnly(true);
    setVariantQuery(slug);
    requestAnimationFrame(() => {
      variantListRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  }, []);

  // Navigation helpers
  const navigateToVariantEditor = useCallback((slug: string) => {
    navigate(`/admin/customization/variants/${slug}`);
  }, [navigate]);

  const navigateToAgentCustomization = useCallback(() => {
    navigate('/admin/customization/agent');
  }, [navigate]);

  const navigateToNewVariant = useCallback(() => {
    navigate('/admin/customization/variants/new');
  }, [navigate]);

  const navigateToAnalytics = useCallback((slug: string) => {
    navigate(`/admin/analytics?variant=${slug}`);
  }, [navigate]);

  const openVariantPreview = useCallback((slug: string) => {
    window.open(`/?variant=${slug}`, '_blank');
  }, []);

  const clearSectionFocusParams = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    next.delete('focusSectionId');
    next.delete('focusSectionType');
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const navigateToSectionEditor = useCallback(
    async (slug: string, options?: { sectionId?: number; sectionType?: string }) => {
      try {
        if (options?.sectionId) {
          navigate(`/admin/customization/variants/${slug}/sections/${options.sectionId}`);
          return true;
        }

        const desiredType = options?.sectionType;
        const data = await loadVariantEditorData(slug);
        const target = desiredType
          ? data.sections.find((section) => section.section_type === desiredType)
          : data.sections[0];

        if (target?.id) {
          navigate(`/admin/customization/variants/${slug}/sections/${target.id}`);
          return true;
        }

        navigate(`/admin/customization/variants/${slug}`);
        return false;
      } catch (navError) {
        console.error('Failed to resolve section editor for variant', slug, navError);
        navigate(`/admin/customization/variants/${slug}`);
        return false;
      }
    },
    [navigate]
  );

  // Focus handling effects
  useEffect(() => {
    if (!focusSlug || variants.length === 0 || focusSlug === appliedFocusSlug) {
      return;
    }
    highlightVariantInList(focusSlug);
    setAppliedFocusSlug(focusSlug);
    const next = new URLSearchParams(searchParams);
    next.delete('focus');
    setSearchParams(next, { replace: true });
  }, [focusSlug, appliedFocusSlug, variants.length, highlightVariantInList, searchParams, setSearchParams]);

  useEffect(() => {
    if (!focusSlug || !variants.length) {
      setAppliedFocusSlug(null);
      setAppliedSectionFocusSlug(null);
      return;
    }

    if ((!focusSectionId && !focusSectionType) || appliedSectionFocusSlug === focusSlug) {
      return;
    }

    let cancelled = false;
    (async () => {
      const success = await navigateToSectionEditor(focusSlug, {
        sectionId: focusSectionId ?? undefined,
        sectionType: focusSectionType ?? undefined,
      });
      if (!cancelled && success) {
        setAppliedSectionFocusSlug(focusSlug);
        clearSectionFocusParams();
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [
    focusSlug,
    focusSectionId,
    focusSectionType,
    appliedSectionFocusSlug,
    variants.length,
    navigateToSectionEditor,
    clearSectionFocusParams,
  ]);

  useEffect(() => {
    if (!focusSlug) {
      setAppliedFocusSlug(null);
      setAppliedSectionFocusSlug(null);
    }
  }, [focusSlug]);

  return {
    // Core data
    variants,
    activeVariants,
    archivedVariants,
    filteredActiveVariants,
    analytics,

    // Weight management
    weightDrafts,
    savingWeights,
    totalAssignedWeight,
    weightStatus,
    trafficShareMode,

    // Attention tracking
    staleVariants,
    neverUpdatedVariants,
    underperformingInfo,
    attentionCandidateSlugs,
    variantAttentionReasons,

    // Stats
    statsBySlug,

    // Filter state
    variantQuery,
    attentionOnly,

    // Loading/error state
    loading,
    error,
    analyticsLoading,
    analyticsError,

    // Inline alert
    operationAlert,
    clearOperationAlert,

    // Ref
    variantListRef,

    // Constants
    snapshotDays: SNAPSHOT_DAYS,

    // Actions
    fetchVariants,
    fetchAnalyticsSnapshot,
    handleArchive,
    handleDelete,
    persistWeight,
    setWeightDraft,

    // Filter actions
    setVariantQuery,
    setAttentionOnly,
    clearVariantFilters,

    // Computed helpers
    getWeight,
    normalizeShare,

    // Navigation helpers
    highlightVariantInList,
    navigateToVariantEditor,
    navigateToSectionEditor,
    navigateToAgentCustomization,
    navigateToNewVariant,
    navigateToAnalytics,
    openVariantPreview,
  };
}
