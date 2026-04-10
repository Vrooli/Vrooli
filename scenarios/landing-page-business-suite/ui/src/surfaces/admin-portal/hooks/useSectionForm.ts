import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type {
  ContentSection,
  LandingConfigResponse,
  LandingSection,
  Variant,
} from '../../../shared/api';
import { getLandingConfig, listVariants } from '../../../shared/api';
import {
  loadSectionEditor,
  persistExistingSectionContent,
  loadVariantContext,
  updateSectionOrder,
  type SectionEditorState,
  type VariantContext,
} from '../controllers/sectionEditorController';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';
import { useDebounce } from '../../../shared/hooks/useDebounce';
import {
  loadComparePreference,
  saveComparePreference,
  sortSectionsByOrder,
  findSectionByType,
  buildDefaultSectionContent,
} from '../services/section.service';

/**
 * Props for the useSectionForm hook
 */
export interface UseSectionFormProps {
  variantSlug: string | undefined;
  sectionId: string | undefined;
  onSuccess?: (message: string, title?: string) => void;
  onError?: (message: string) => void;
}

/**
 * Custom hook for managing section editor form state
 */
export function useSectionForm({
  variantSlug,
  sectionId,
  onSuccess,
  onError,
}: UseSectionFormProps) {
  const navigate = useNavigate();
  const isNew = sectionId === 'new';
  const parsedSectionId = !isNew && sectionId ? Number(sectionId) : NaN;
  const numericSectionId = Number.isNaN(parsedSectionId) ? null : parsedSectionId;

  // Section data state
  const [section, setSection] = useState<ContentSection | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Variant context state
  const [variantContext, setVariantContext] = useState<VariantContext | null>(null);
  const [variantContextError, setVariantContextError] = useState<string | null>(null);
  const [variantContextLoading, setVariantContextLoading] = useState(Boolean(variantSlug));

  // Preview config state
  const [previewConfig, setPreviewConfig] = useState<LandingConfigResponse | null>(null);
  const [previewConfigLoading, setPreviewConfigLoading] = useState(false);
  const [previewConfigError, setPreviewConfigError] = useState<string | null>(null);

  // Variant options for comparison
  const [variantOptions, setVariantOptions] = useState<Variant[]>([]);
  const [variantOptionsLoading, setVariantOptionsLoading] = useState(false);
  const [variantOptionsError, setVariantOptionsError] = useState<string | null>(null);

  // Comparison state
  const [compareVariantSlug, setCompareVariantSlug] = useState('');
  const [compareConfig, setCompareConfig] = useState<LandingConfigResponse | null>(null);
  const [compareLoading, setCompareLoading] = useState(false);
  const [compareError, setCompareError] = useState<string | null>(null);
  const compareConfigCache = useRef<Map<string, LandingConfigResponse>>(new Map());

  // Reorder state
  const [reorderingSectionId, setReorderingSectionId] = useState<number | null>(null);
  const [reorderError, setReorderError] = useState<string | null>(null);

  // Form state
  const [sectionType, setSectionType] = useState<ContentSection['section_type']>('hero');
  const [enabled, setEnabled] = useState(true);
  const [order, setOrder] = useState(0);
  const [content, setContent] = useState<Record<string, unknown>>(buildDefaultSectionContent());

  // Debounced content for live preview
  const debouncedContent = useDebounce(content, 300);

  /**
   * Apply section state from controller response
   */
  const applySectionState = useCallback((state: SectionEditorState) => {
    setSection(state.section);
    setSectionType(state.form.sectionType);
    setEnabled(state.form.enabled);
    setOrder(state.form.order);
    setContent(state.form.content);
  }, []);

  /**
   * Fetch section data
   */
  const fetchSection = useCallback(async () => {
    if (isNew || numericSectionId === null) {
      return;
    }

    try {
      setLoading(true);
      const state = await loadSectionEditor(numericSectionId);
      applySectionState(state);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load section');
      console.error('Section fetch error:', err);
    } finally {
      setLoading(false);
    }
  }, [isNew, numericSectionId, applySectionState]);

  /**
   * Load preview config for variant
   */
  const loadPreviewConfig = useCallback(async (slug?: string | null) => {
    if (!slug) {
      setPreviewConfig(null);
      setPreviewConfigError('Variant slug missing for preview');
      setPreviewConfigLoading(false);
      return;
    }

    try {
      setPreviewConfigLoading(true);
      const config = await getLandingConfig(slug);
      setPreviewConfig(config);
      setPreviewConfigError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load landing preview context';
      setPreviewConfig(null);
      setPreviewConfigError(message);
    } finally {
      setPreviewConfigLoading(false);
    }
  }, []);

  /**
   * Refresh preview config
   */
  const refreshPreviewConfig = useCallback(async () => {
    await loadPreviewConfig(variantSlug);
  }, [variantSlug, loadPreviewConfig]);

  /**
   * Handle comparison variant change
   */
  const handleCompareVariantChange = useCallback(async (slug: string) => {
    setCompareVariantSlug(slug);
    if (!slug) {
      setCompareConfig(null);
      setCompareError(null);
      return;
    }

    if (compareConfigCache.current.has(slug)) {
      setCompareConfig(compareConfigCache.current.get(slug) ?? null);
      setCompareError(null);
      return;
    }

    try {
      setCompareLoading(true);
      const config = await getLandingConfig(slug);
      compareConfigCache.current.set(slug, config);
      setCompareConfig(config);
      setCompareError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load comparison variant';
      setCompareError(message);
      setCompareConfig(null);
    } finally {
      setCompareLoading(false);
    }
  }, []);

  /**
   * Save section content
   */
  const handleSave = useCallback(async () => {
    if (!variantSlug) {
      onError?.('Variant slug is required to save section');
      return;
    }

    try {
      setSaving(true);

      if (isNew) {
        onError?.('Creating new sections requires variant ID. This is a placeholder.');
        return;
      }

      if (numericSectionId === null) {
        setError('Section ID is missing.');
        onError?.('Cannot save: section ID is missing');
        return;
      }

      const state = await persistExistingSectionContent(numericSectionId, content);
      applySectionState(state);
      setError(null);
      onSuccess?.(`${sectionType} section saved`, 'Section updated');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save section');
      onError?.('Failed to save section changes');
      console.error('Section save error:', err);
    } finally {
      setSaving(false);
    }
  }, [variantSlug, isNew, numericSectionId, content, sectionType, applySectionState, onSuccess, onError]);

  /**
   * Update a content field
   */
  const updateContentField = useCallback((key: string, value: unknown) => {
    setContent((prev) => ({
      ...prev,
      [key]: value,
    }));
  }, []);

  /**
   * Navigate to a different section
   */
  const handleNavigateSection = useCallback((target: LandingSection) => {
    if (!variantSlug) {
      return;
    }
    const targetPath = target.id
      ? `/admin/customization/variants/${variantSlug}/sections/${target.id}`
      : `/admin/customization/variants/${variantSlug}/sections/new`;
    navigate(targetPath);
  }, [variantSlug, navigate]);

  /**
   * Navigate to add new section
   */
  const handleAddSection = useCallback(() => {
    if (!variantSlug) return;
    navigate(`/admin/customization/variants/${variantSlug}/sections/new`);
  }, [variantSlug, navigate]);

  /**
   * Reorder a section
   */
  const handleReorderSection = useCallback(
    async (target: LandingSection, direction: 'up' | 'down') => {
      const timelineSections = sortSectionsByOrder(previewConfig?.sections ?? []);

      if (!target.id || !timelineSections.length) {
        return;
      }
      const currentIndex = timelineSections.findIndex((s) => s.id === target.id);
      if (currentIndex === -1) {
        return;
      }
      const neighborIndex = currentIndex + (direction === 'up' ? -1 : 1);
      const neighbor = timelineSections[neighborIndex];
      if (!neighbor || !neighbor.id || typeof neighbor.order !== 'number' || typeof target.order !== 'number') {
        setReorderError('Unable to move section. Missing neighbor information.');
        return;
      }

      try {
        setReorderingSectionId(target.id);
        setReorderError(null);
        await Promise.all([
          updateSectionOrder(target.id, neighbor.order),
          updateSectionOrder(neighbor.id, target.order),
        ]);
        await refreshPreviewConfig();
        onSuccess?.(`Section moved ${direction}`, 'Order updated');
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to reorder sections';
        setReorderError(message);
        onError?.('Failed to reorder sections');
      } finally {
        setReorderingSectionId(null);
      }
    },
    [previewConfig, refreshPreviewConfig, onSuccess, onError]
  );

  // Computed values
  const timelineSections = useMemo(() => {
    return sortSectionsByOrder(previewConfig?.sections ?? []);
  }, [previewConfig]);

  const previewVariantLabel = useMemo(() => {
    return previewConfig?.variant?.name || variantContext?.variant?.name || variantSlug || 'Active variant';
  }, [previewConfig, variantContext, variantSlug]);

  const comparisonVariantLabel = useMemo(() => {
    if (!compareVariantSlug) {
      return null;
    }
    const matched = variantOptions.find((variant) => variant.slug === compareVariantSlug);
    return matched?.name || compareVariantSlug;
  }, [compareVariantSlug, variantOptions]);

  const comparisonSection = useMemo(() => {
    if (!compareConfig) {
      return null;
    }
    return findSectionByType(compareConfig.sections ?? [], sectionType);
  }, [compareConfig, sectionType]);

  const comparisonContent = comparisonSection?.content ?? {};
  const comparisonEnabled = comparisonSection?.enabled !== false;

  // Load section on mount
  useEffect(() => {
    if (!isNew && numericSectionId !== null) {
      fetchSection();
    }
  }, [isNew, numericSectionId, fetchSection]);

  // Load variant context
  useEffect(() => {
    if (!variantSlug) {
      setVariantContext(null);
      setVariantContextError(null);
      setVariantContextLoading(false);
      return;
    }

    let cancelled = false;
    const fetchContext = async () => {
      try {
        setVariantContextLoading(true);
        const context = await loadVariantContext(variantSlug);
        if (!cancelled) {
          setVariantContext(context);
          setVariantContextError(null);
        }
      } catch (err) {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : 'Failed to load variant guidance';
          setVariantContextError(message);
        }
      } finally {
        if (!cancelled) {
          setVariantContextLoading(false);
        }
      }
    };
    fetchContext();

    return () => {
      cancelled = true;
    };
  }, [variantSlug]);

  // Load preview config
  useEffect(() => {
    loadPreviewConfig(variantSlug);
  }, [variantSlug, loadPreviewConfig]);

  // Load variant options
  useEffect(() => {
    let cancelled = false;
    const fetchVariantOptions = async () => {
      try {
        setVariantOptionsLoading(true);
        const data = await listVariants();
        if (!cancelled) {
          setVariantOptions(data.variants);
          setVariantOptionsError(null);
        }
      } catch (err) {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : 'Failed to load variants';
          setVariantOptionsError(message);
        }
      } finally {
        if (!cancelled) {
          setVariantOptionsLoading(false);
        }
      }
    };
    fetchVariantOptions();
    return () => {
      cancelled = true;
    };
  }, []);

  // Remember variant session
  useEffect(() => {
    if (!variantSlug || !variantContext?.variant) {
      return;
    }

    if (isNew || numericSectionId === null) {
      rememberVariantSession({
        slug: variantSlug,
        name: variantContext.variant.name,
        surface: 'variant',
      });
      return;
    }

    rememberVariantSession({
      slug: variantSlug,
      name: variantContext.variant.name,
      surface: 'section',
      sectionId: numericSectionId,
      sectionType,
    });
  }, [variantSlug, variantContext, numericSectionId, isNew, sectionType]);

  // Load saved comparison preference
  useEffect(() => {
    if (!variantSlug) return;
    const saved = loadComparePreference(variantSlug);
    if (saved && saved !== compareVariantSlug) {
      handleCompareVariantChange(saved);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variantSlug]);

  // Save comparison preference
  useEffect(() => {
    if (!variantSlug) return;
    saveComparePreference(variantSlug, compareVariantSlug || null);
  }, [variantSlug, compareVariantSlug]);

  return {
    // Section state
    section,
    loading,
    saving,
    error,
    isNew,
    numericSectionId,

    // Form state
    sectionType,
    setSectionType,
    enabled,
    setEnabled,
    order,
    setOrder,
    content,
    debouncedContent,
    updateContentField,

    // Variant context
    variantContext,
    variantContextError,
    variantContextLoading,

    // Preview state
    previewConfig,
    previewConfigLoading,
    previewConfigError,
    previewVariantLabel,
    timelineSections,

    // Variant options
    variantOptions,
    variantOptionsLoading,
    variantOptionsError,

    // Comparison state
    compareVariantSlug,
    compareConfig,
    compareLoading,
    compareError,
    comparisonVariantLabel,
    comparisonSection,
    comparisonContent,
    comparisonEnabled,
    handleCompareVariantChange,

    // Reorder state
    reorderingSectionId,
    reorderError,
    handleReorderSection,

    // Actions
    handleSave,
    handleNavigateSection,
    handleAddSection,
  };
}
