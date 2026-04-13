import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Eye, Plus } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../../../shared/ui/button';
import { getLandingConfig, listVariants, type ContentSection, type LandingConfigResponse, type LandingSection, type Variant } from '../../../shared/api';
import { useDebounce } from '../../../shared/hooks/useDebounce';
import {
  loadSectionEditor,
  persistExistingSectionContent,
  loadVariantContext,
  updateSectionOrder,
  type SectionEditorState,
  type VariantContext,
} from '../controllers/sectionEditorController';
import { getStylingConfig, getVariantStylingGuidance } from '../../../shared/lib/stylingConfig';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';
import { MetricsModeProvider } from '../../../shared/hooks/useMetrics';
import { HeroSection } from '../../public-landing/sections/HeroSection';
import { FeaturesSection } from '../../public-landing/sections/FeaturesSection';
import { PricingSection } from '../../public-landing/sections/PricingSection';
import { CTASection } from '../../public-landing/sections/CTASection';
import { TestimonialsSection } from '../../public-landing/sections/TestimonialsSection';
import { FAQSection } from '../../public-landing/sections/FAQSection';
import { FooterSection } from '../../public-landing/sections/FooterSection';
import { VideoSection } from '../../public-landing/sections/VideoSection';
import { DownloadSection } from '../../public-landing/sections/DownloadSection';

/**
 * Section Editor - Split-screen form + live preview
 * Implements OT-P0-012 (Split customization layout: form + preview, stacked on mobile)
 * Implements OT-P0-013 (Live preview updates within 300ms without page reload)
 *
 * [REQ:CUSTOM-SPLIT] [REQ:CUSTOM-LIVE]
 */
const STYLING_CONFIG = getStylingConfig();
const COMPARE_STORAGE_KEY = 'landing-manager-section-editor-compare';

type PreviewRenderer = (params: {
  content: Record<string, unknown>;
  config: LandingConfigResponse | null;
}) => JSX.Element | null;

const SECTION_PREVIEW_RENDERERS: Record<ContentSection['section_type'], PreviewRenderer> = {
  hero: ({ content }) => <HeroSection content={content as any} />,
  features: ({ content }) => <FeaturesSection content={content as any} />,
  pricing: ({ content, config }) => <PricingSection content={content as any} pricingOverview={config?.pricing} />,
  cta: ({ content }) => <CTASection content={content as any} />,
  testimonials: ({ content }) => <TestimonialsSection content={content as any} />,
  faq: ({ content }) => <FAQSection content={content as any} />,
  footer: ({ content }) => <FooterSection content={content as any} />,
  video: ({ content }) => <VideoSection content={content as any} />,
  downloads: ({ content, config }) => (
    <DownloadSection content={content as any} downloads={config?.downloads} />
  ),
};

export function SectionEditor() {
  const navigate = useNavigate();
  const { variantSlug, sectionId } = useParams<{ variantSlug: string; sectionId: string }>();
  const isNew = sectionId === 'new';
  const parsedSectionId = !isNew && sectionId ? Number(sectionId) : NaN;
  const numericSectionId = Number.isNaN(parsedSectionId) ? null : parsedSectionId;

  const [section, setSection] = useState<ContentSection | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [variantContext, setVariantContext] = useState<VariantContext | null>(null);
  const [variantContextError, setVariantContextError] = useState<string | null>(null);
  const [variantContextLoading, setVariantContextLoading] = useState(Boolean(variantSlug));
  const [previewConfig, setPreviewConfig] = useState<LandingConfigResponse | null>(null);
  const [previewConfigLoading, setPreviewConfigLoading] = useState(false);
  const [previewConfigError, setPreviewConfigError] = useState<string | null>(null);
  const [variantOptions, setVariantOptions] = useState<Variant[]>([]);
  const [variantOptionsLoading, setVariantOptionsLoading] = useState(false);
  const [variantOptionsError, setVariantOptionsError] = useState<string | null>(null);
  const [compareVariantSlug, setCompareVariantSlug] = useState('');
  const [compareConfig, setCompareConfig] = useState<LandingConfigResponse | null>(null);
  const [compareLoading, setCompareLoading] = useState(false);
  const [compareError, setCompareError] = useState<string | null>(null);
  const compareConfigCache = useRef<Map<string, LandingConfigResponse>>(new Map());
  const [reorderingSectionId, setReorderingSectionId] = useState<number | null>(null);
  const [reorderError, setReorderError] = useState<string | null>(null);

  // Form state
  const [sectionType, setSectionType] = useState<ContentSection['section_type']>('hero');
  const [enabled, setEnabled] = useState(true);
  const [order, setOrder] = useState(0);
  const [content, setContent] = useState<Record<string, unknown>>({
    title: '',
    subtitle: '',
    cta_text: '',
    cta_url: '',
  });

  // Debounced content for live preview (300ms delay)
  const debouncedContent = useDebounce(content, 300);

  useEffect(() => {
    if (!isNew && numericSectionId !== null) {
      fetchSection();
    }
  }, [isNew, numericSectionId]);

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

  useEffect(() => {
    loadPreviewConfig(variantSlug);
  }, [variantSlug, loadPreviewConfig]);

  const refreshPreviewConfig = useCallback(async () => {
    await loadPreviewConfig(variantSlug);
  }, [variantSlug, loadPreviewConfig]);

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

  const applySectionState = (state: SectionEditorState) => {
    setSection(state.section);
    setSectionType(state.form.sectionType);
    setEnabled(state.form.enabled);
    setOrder(state.form.order);
    setContent(state.form.content);
  };

  const fetchSection = async () => {
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
  };

  const handleSave = async () => {
    if (!variantSlug) {
      alert('Variant slug is required');
      return;
    }

    try {
      setSaving(true);

      if (isNew) {
        // Note: This is a simplified version. In a real app, you'd fetch the variant ID
        // For now, we'll assume variant ID is stored or fetched separately
        alert('Creating new sections requires variant ID. This is a placeholder.');
        return;
      }

      if (numericSectionId === null) {
        setError('Section ID is missing.');
        return;
      }

      const state = await persistExistingSectionContent(numericSectionId, content);
      applySectionState(state);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save section');
      console.error('Section save error:', err);
    } finally {
      setSaving(false);
    }
  };

  const updateContentField = (key: string, value: unknown) => {
    setContent((prev) => ({
      ...prev,
      [key]: value,
    }));
  };

  const handleNavigateSection = (target: LandingSection) => {
    if (!variantSlug) {
      return;
    }
    const targetPath = target.id
      ? `/admin/customization/variants/${variantSlug}/sections/${target.id}`
      : `/admin/customization/variants/${variantSlug}/sections/new`;
    navigate(targetPath);
  };

  const handleAddSection = () => {
    if (!variantSlug) return;
    navigate(`/admin/customization/variants/${variantSlug}/sections/new`);
  };

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

  useEffect(() => {
    if (!variantSlug) {
      return;
    }
    if (typeof window === 'undefined') {
      return;
    }

    try {
      const raw = window.localStorage.getItem(COMPARE_STORAGE_KEY);
      if (!raw) {
        return;
      }
      const stored = JSON.parse(raw);
      const savedSlug = stored?.[variantSlug];
      if (typeof savedSlug === 'string' && savedSlug !== compareVariantSlug) {
        handleCompareVariantChange(savedSlug);
      }
    } catch (err) {
      console.warn('Failed to load comparison preference', err);
    }
    // intentionally omit handleCompareVariantChange from deps to avoid duplicate fetch loops
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variantSlug]);

  useEffect(() => {
    if (!variantSlug) {
      return;
    }
    if (typeof window === 'undefined') {
      return;
    }
    try {
      const raw = window.localStorage.getItem(COMPARE_STORAGE_KEY);
      const stored = raw ? JSON.parse(raw) : {};
      if (compareVariantSlug) {
        stored[variantSlug] = compareVariantSlug;
      } else {
        delete stored[variantSlug];
      }
      window.localStorage.setItem(COMPARE_STORAGE_KEY, JSON.stringify(stored));
    } catch (err) {
      console.warn('Failed to persist comparison preference', err);
    }
  }, [variantSlug, compareVariantSlug]);

  const previewRenderer = useMemo(() => SECTION_PREVIEW_RENDERERS[sectionType], [sectionType]);
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
  const timelineSections = useMemo(() => {
    const sections = previewConfig?.sections ?? [];
    return [...sections].sort((a, b) => a.order - b.order);
  }, [previewConfig]);
  const comparisonSection = useMemo(() => {
    if (!compareConfig) {
      return null;
    }
    return (
      compareConfig.sections
        ?.filter((section) => section.section_type === sectionType)
        .sort((a, b) => a.order - b.order)[0] ?? null
    );
  }, [compareConfig, sectionType]);
  const comparisonContent = comparisonSection?.content ?? {};
  const comparisonEnabled = comparisonSection?.enabled !== false;
  const handleReorderSection = useCallback(
    async (target: LandingSection, direction: 'up' | 'down') => {
      if (!target.id || !timelineSections.length) {
        return;
      }
      const currentIndex = timelineSections.findIndex((section) => section.id === target.id);
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
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to reorder sections';
        setReorderError(message);
      } finally {
        setReorderingSectionId(null);
      }
    },
    [timelineSections, refreshPreviewConfig],
  );

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading section...</p>
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout>
      <div className="max-w-[2000px] mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate(`/admin/customization/variants/${variantSlug}`)}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <div className="flex-1">
            <h1 className="text-2xl font-semibold">
              {isNew ? 'New Section' : `Edit ${sectionType} Section`}
            </h1>
          </div>
          {variantSlug && (
            <Button
              asChild
              variant="outline"
              size="sm"
              className="gap-2 hidden sm:inline-flex"
            >
              <a href={`/?variant=${variantSlug}`} target="_blank" rel="noopener noreferrer">
                <Eye className="h-4 w-4" />
                View Variant
              </a>
            </Button>
          )}
          <Button
            onClick={handleSave}
            disabled={saving}
            className="gap-2"
            data-testid="save-section"
          >
            <Save className="h-4 w-4" />
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </div>

        {error && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 mb-6">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {/* Split Layout: Form + Live Preview (OT-P0-012) */}
        <div className="grid lg:grid-cols-2 gap-6">
          {/* Left Column: Form */}
          <div className="space-y-6" data-testid="section-form">
            {variantSlug && (
              <VariantSectionTimeline
                variantName={previewVariantLabel}
                sections={timelineSections}
                loading={previewConfigLoading}
                error={previewConfigError}
                currentSectionId={numericSectionId}
                currentSectionType={sectionType}
                onNavigateSection={handleNavigateSection}
                onAddSection={handleAddSection}
                onReorderSection={handleReorderSection}
                reorderingSectionId={reorderingSectionId}
                reorderError={reorderError}
              />
            )}
            <VariantContextCard
              context={variantContext}
              error={variantContextError}
              loading={variantContextLoading}
            />
            <StylingGuardrailsCard variantSlug={variantContext?.variant.slug ?? variantSlug} />
            {/* Section Settings */}
            <div className="bg-white/5 border border-white/10 rounded-xl p-6">
              <h2 className="text-lg font-semibold mb-4">Section Settings</h2>

              <div className="space-y-4">
                <div>
                  <label htmlFor="section-type" className="block text-sm font-medium text-slate-300 mb-2">
                    Section Type
                  </label>
                  <select
                    id="section-type"
                    value={sectionType}
                    onChange={(e) => setSectionType(e.target.value as ContentSection['section_type'])}
                    disabled={!isNew}
                    className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none disabled:opacity-50"
                    data-testid="section-type-input"
                  >
                    <option value="hero">Hero</option>
                    <option value="features">Features</option>
                    <option value="pricing">Pricing</option>
                    <option value="cta">Call to Action</option>
                    <option value="video">Video</option>
                    <option value="testimonials">Testimonials</option>
                    <option value="faq">FAQ</option>
                    <option value="footer">Footer</option>
                    <option value="downloads">Downloads</option>
                  </select>
                </div>

                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={enabled}
                      onChange={(e) => setEnabled(e.target.checked)}
                      className="w-4 h-4"
                      data-testid="section-enabled-input"
                    />
                    <span className="text-sm text-slate-300">Enabled</span>
                  </label>

                  <div className="flex-1">
                    <label htmlFor="order" className="block text-sm text-slate-300 mb-1">
                      Display Order
                    </label>
                    <input
                      id="order"
                      type="number"
                      value={order}
                      onChange={(e) => setOrder(parseInt(e.target.value) || 0)}
                      className="w-full px-3 py-1 bg-slate-900 border border-white/10 rounded"
                      data-testid="section-order-input"
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* Content Fields */}
              <div className="bg-white/5 border border-white/10 rounded-xl p-6">
                <h2 className="text-lg font-semibold mb-4">Content</h2>

              <div className="space-y-4">
                <div>
                  <label htmlFor="title" className="block text-sm font-medium text-slate-300 mb-2">
                    Title
                  </label>
                  <input
                    id="title"
                    type="text"
                    value={(content.title as string) || ''}
                    onChange={(e) => updateContentField('title', e.target.value)}
                    className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                    placeholder="Enter title"
                    data-testid="content-title-input"
                  />
                </div>

                <div>
                  <label htmlFor="subtitle" className="block text-sm font-medium text-slate-300 mb-2">
                    Subtitle
                  </label>
                  <textarea
                    id="subtitle"
                    value={(content.subtitle as string) || ''}
                    onChange={(e) => updateContentField('subtitle', e.target.value)}
                    className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                    rows={3}
                    placeholder="Enter subtitle"
                    data-testid="content-subtitle-input"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label htmlFor="cta-text" className="block text-sm font-medium text-slate-300 mb-2">
                      CTA Text
                    </label>
                    <input
                      id="cta-text"
                      type="text"
                      value={(content.cta_text as string) || ''}
                      onChange={(e) => updateContentField('cta_text', e.target.value)}
                      className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                      placeholder="Get Started"
                      data-testid="content-cta-text-input"
                    />
                  </div>

                  <div>
                    <label htmlFor="cta-url" className="block text-sm font-medium text-slate-300 mb-2">
                      CTA URL
                    </label>
                    <input
                      id="cta-url"
                      type="text"
                      value={(content.cta_url as string) || ''}
                      onChange={(e) => updateContentField('cta_url', e.target.value)}
                      className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                      placeholder="/signup"
                      data-testid="content-cta-url-input"
                    />
                  </div>
                </div>

                {/* Additional fields based on section type */}
                {sectionType === 'hero' && (
                  <div>
                    <label htmlFor="image-url" className="block text-sm font-medium text-slate-300 mb-2">
                      Hero Image URL
                    </label>
                    <input
                      id="image-url"
                      type="text"
                      value={(content.image_url as string) || ''}
                      onChange={(e) => updateContentField('image_url', e.target.value)}
                      className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                      placeholder="https://example.com/hero.jpg"
                      data-testid="content-image-url-input"
                    />
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Right Column: Live Preview (OT-P0-013: updates within 300ms) */}
          <div className="lg:sticky lg:top-6 lg:self-start">
            <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h2 className="text-lg font-semibold flex items-center gap-2">
                    <Eye className="h-5 w-5" />
                    Live Preview
                  </h2>
                  <p className="text-xs text-slate-500">
                    Mirrors the actual landing component for this section type.
                  </p>
                </div>
                <div className="w-full sm:w-auto">
                  <label htmlFor="compare-variant" className="block text-xs uppercase tracking-[0.3em] text-slate-500 mb-1">
                    Compare Variant
                  </label>
                  <select
                    id="compare-variant"
                    value={compareVariantSlug}
                    onChange={(e) => handleCompareVariantChange(e.target.value)}
                    className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none"
                  >
                    <option value="">Single preview</option>
                    {variantOptions
                      .filter((variant) => variant.slug !== variantSlug)
                      .map((variant) => (
                        <option key={variant.slug} value={variant.slug}>
                          {variant.name || variant.slug}
                        </option>
                      ))}
                  </select>
                  {variantOptionsLoading && (
                    <p className="mt-1 text-[11px] text-slate-500">Loading variant list…</p>
                  )}
                  {variantOptionsError && (
                    <p className="mt-1 text-[11px] text-amber-300">{variantOptionsError}</p>
                  )}
                </div>
              </div>

              <div className="flex items-center justify-between text-xs text-slate-500">
                <div>Updates in 300ms</div>
                <div className="text-[11px] text-slate-400">Editing: {previewVariantLabel}</div>
              </div>

              <div
                className={`grid gap-4 ${compareVariantSlug && compareConfig ? 'lg:grid-cols-2' : ''}`}
                data-testid="section-preview"
              >
                <PreviewPanel
                  title="Editing variant"
                  variantLabel={previewVariantLabel}
                  renderer={previewRenderer}
                  content={debouncedContent}
                  config={previewConfig}
                  sectionEnabled={enabled}
                  missingSectionMessage={`No ${sectionType} preview available yet.`}
                />
                {compareVariantSlug && (
                  <PreviewPanel
                    title="Comparison variant"
                    variantLabel={comparisonVariantLabel || compareVariantSlug}
                    renderer={comparisonSection ? previewRenderer : undefined}
                    content={comparisonContent}
                    config={compareConfig}
                    sectionEnabled={comparisonSection ? comparisonEnabled : true}
                    missingSectionMessage={
                      comparisonSection
                        ? `Unable to render ${sectionType} for ${comparisonVariantLabel || compareVariantSlug}.`
                        : `${comparisonVariantLabel || compareVariantSlug} does not include a ${sectionType} section yet.`
                    }
                  />
                )}
              </div>

              {previewConfigLoading && (
                <div className="text-xs text-slate-500">
                  Syncing landing runtime context…
                </div>
              )}
              {previewConfigError && (
                <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                  {previewConfigError}
                </div>
              )}
              {compareLoading && (
                <div className="text-xs text-slate-500">Loading comparison variant…</div>
              )}
              {compareError && (
                <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                  {compareError}
                </div>
              )}

              <div className="text-xs text-slate-500">
                Preview automatically updates as you type (debounced 300ms)
              </div>

              {variantSlug && (
                <Button
                  asChild
                  variant="ghost"
                  size="sm"
                  className="gap-2 sm:hidden"
                >
                  <a href={`/?variant=${variantSlug}`} target="_blank" rel="noopener noreferrer">
                    <Eye className="h-4 w-4" />
                    View Variant
                  </a>
                </Button>
              )}
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  );
}

function PreviewPanel({
  title,
  variantLabel,
  renderer,
  content,
  config,
  sectionEnabled,
  missingSectionMessage,
}: {
  title: string;
  variantLabel: string;
  renderer?: PreviewRenderer;
  content: Record<string, unknown>;
  config: LandingConfigResponse | null;
  sectionEnabled: boolean;
  missingSectionMessage: string;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-xs text-slate-500">
        <span className="font-semibold text-white">{title}</span>
        <span className="text-[11px] text-slate-400">{variantLabel}</span>
      </div>
      <div className="rounded-2xl border border-white/10 bg-slate-950/90 p-4">
        {!sectionEnabled && (
          <div className="mb-4 px-3 py-2 bg-amber-500/20 border border-amber-500/30 rounded text-amber-300 text-sm">
            Section is currently disabled
          </div>
        )}
        <MetricsModeProvider mode="preview">
          <div className="relative rounded-[28px] border border-white/10 bg-[#07090F] shadow-[0_10px_50px_rgba(7,9,15,0.8)]">
            <div className="max-h-[720px] overflow-y-auto rounded-[28px]">
              {renderer ? (
                renderer({
                  content,
                  config,
                })
              ) : (
                <div className="p-8 text-center text-sm text-slate-400">{missingSectionMessage}</div>
              )}
            </div>
          </div>
        </MetricsModeProvider>
      </div>
    </div>
  );
}

function VariantSectionTimeline({
  variantName,
  sections,
  loading,
  error,
  currentSectionId,
  currentSectionType,
  onNavigateSection,
  onAddSection,
  onReorderSection,
  reorderingSectionId,
  reorderError,
}: {
  variantName: string;
  sections: LandingSection[];
  loading: boolean;
  error: string | null;
  currentSectionId: number | null;
  currentSectionType: ContentSection['section_type'];
  onNavigateSection: (section: LandingSection) => void;
  onAddSection: () => void;
  onReorderSection: (section: LandingSection, direction: 'up' | 'down') => void;
  reorderingSectionId: number | null;
  reorderError: string | null;
}) {
  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="variant-section-timeline">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Variant Timeline</p>
          <h2 className="text-lg font-semibold text-white">Sections for {variantName}</h2>
          <p className="text-xs text-slate-500">Jump directly to any section without leaving the editor.</p>
        </div>
        <Button variant="outline" size="sm" className="gap-2" onClick={onAddSection}>
          <Plus className="h-4 w-4" />
          New Section
        </Button>
      </div>
      {loading && <p className="text-sm text-slate-400">Loading sections…</p>}
      {error && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {error}
        </div>
      )}
      {!loading && !error && sections.length === 0 && (
        <p className="text-sm text-slate-400">
          This variant has no sections yet. Use the button above to create the first one.
        </p>
      )}
      {!loading && !error && sections.length > 0 && (
        <div className="space-y-2">
          {sections.map((section) => {
            const isActive = currentSectionId
              ? section.id === currentSectionId
              : section.section_type === currentSectionType;
            const badge = section.enabled === false ? 'Disabled' : 'Enabled';
            const isFirst = sections[0]?.id === section.id;
            const isLast = sections[sections.length - 1]?.id === section.id;
            return (
              <div
                key={`${section.section_type}-${section.id ?? section.order}`}
                role="button"
                tabIndex={0}
                onClick={() => onNavigateSection(section)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    onNavigateSection(section);
                  }
                }}
                className={`rounded-xl border px-4 py-3 text-left transition-colors ${
                  isActive ? 'border-white/40 bg-white/10' : 'border-white/10 hover:border-white/30'
                }`}
              >
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <div className="text-xs text-slate-500">#{section.order ?? '-'}</div>
                    <div className="text-sm font-medium capitalize text-white">{section.section_type}</div>
                    <div className="text-[11px] uppercase tracking-wide text-slate-500">{badge}</div>
                  </div>
                  <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                    {section.id && (
                      <>
                        <button
                          type="button"
                          className="rounded-full border border-white/20 px-3 py-1 hover:border-white/40 disabled:opacity-50"
                          onClick={(event) => {
                            event.stopPropagation();
                            onReorderSection(section, 'up');
                          }}
                          disabled={reorderingSectionId !== null || isFirst}
                        >
                          Move up
                        </button>
                        <button
                          type="button"
                          className="rounded-full border border-white/20 px-3 py-1 hover:border-white/40 disabled:opacity-50"
                          onClick={(event) => {
                            event.stopPropagation();
                            onReorderSection(section, 'down');
                          }}
                          disabled={reorderingSectionId !== null || isLast}
                        >
                          Move down
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
      {reorderError && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {reorderError}
        </div>
      )}
    </div>
  );
}

function VariantContextCard({
  context,
  error,
  loading,
}: {
  context: VariantContext | null;
  error: string | null;
  loading: boolean;
}) {
  if (!context && !error && !loading) {
    return null;
  }

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="variant-context-card">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">Variant Context</h2>
          <p className="text-sm text-slate-400">
            Align copy with the selected persona, JTBD, and conversion style pulled from variant_space.json.
          </p>
        </div>
        {context?.variant && (
          <span className="text-xs uppercase tracking-wide text-slate-500 bg-slate-900/60 px-3 py-1 rounded-full">
            {context.variant.name}
          </span>
        )}
      </div>

      {loading && (
        <div className="text-slate-400 text-sm">Loading variant guidance…</div>
      )}

      {error && (
        <div className="text-sm text-red-400">
          {error}
        </div>
      )}

      {context && (
        <div className="space-y-4">
          {context.axes.map((axis) => (
            <div key={axis.axisId} className="border-l-2 border-purple-500/40 pl-4">
              <div className="flex items-center justify-between text-xs uppercase text-slate-500 mb-1">
                <span>{axis.axisLabel}</span>
                {axis.axisNote && <span className="text-slate-600">{axis.axisNote}</span>}
              </div>
              <div className="text-lg font-semibold text-white">
                {axis.selectionLabel || 'Not selected'}
              </div>
              {axis.selectionDescription && (
                <p className="text-sm text-slate-400 mt-1">{axis.selectionDescription}</p>
              )}
              {axis.agentHints && axis.agentHints.length > 0 && (
                <ul className="mt-2 text-sm text-slate-400 space-y-1 list-disc list-inside">
                  {axis.agentHints.map((hint, index) => (
                    <li key={index}>{hint}</li>
                  ))}
                </ul>
              )}
            </div>
          ))}

          {(context.variantSpace.agentGuidelines && context.variantSpace.agentGuidelines.length > 0) && (
            <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 text-sm text-slate-300 space-y-2">
              <div className="font-medium text-slate-200">Agent Guidelines</div>
              <ul className="list-disc list-inside space-y-1">
                {context.variantSpace.agentGuidelines.map((guideline, index) => (
                  <li key={index}>{guideline}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function StylingGuardrailsCard({ variantSlug }: { variantSlug?: string }) {
  const variantGuidance = getVariantStylingGuidance(variantSlug);

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="styling-guardrails-card">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">Styling & Tone Guardrails</h2>
          <p className="text-sm text-slate-400">
            Pulled from styling.json so CTA, palette, and messaging stay aligned.
          </p>
        </div>
        <span className="text-xs uppercase tracking-wide text-slate-500 bg-slate-900/60 px-3 py-1 rounded-full">
          {STYLING_CONFIG.brand?.product_name ?? 'Brand'}
        </span>
      </div>

      {STYLING_CONFIG.tone?.voice && (
        <p className="text-sm text-slate-300">
          Voice: <span className="text-white font-medium">{STYLING_CONFIG.tone.voice}</span>
        </p>
      )}

      {STYLING_CONFIG.tone?.keywords && (
        <div className="flex flex-wrap gap-2">
          {STYLING_CONFIG.tone.keywords.map((keyword) => (
            <span key={keyword} className="text-xs uppercase tracking-wide bg-purple-500/20 text-purple-200 px-2 py-1 rounded-full">
              {keyword}
            </span>
          ))}
        </div>
      )}

      {STYLING_CONFIG.usage_notes && STYLING_CONFIG.usage_notes.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs uppercase text-slate-500">Usage Notes</div>
          <ul className="list-disc list-inside text-sm text-slate-300 space-y-1">
            {STYLING_CONFIG.usage_notes.map((note, index) => (
              <li key={index}>{note}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-2">
        <div className="text-xs uppercase text-slate-500">Variant CTA Guidance</div>
        {variantGuidance.promise && (
          <p className="text-base text-white font-semibold">{variantGuidance.promise}</p>
        )}
        <div className="text-sm text-slate-300 space-y-1">
          {variantGuidance.primary_cta && <div>Primary CTA: {variantGuidance.primary_cta}</div>}
          {variantGuidance.secondary_cta && <div>Secondary CTA: {variantGuidance.secondary_cta}</div>}
        </div>
        {variantGuidance.notes && variantGuidance.notes.length > 0 && (
          <ul className="list-disc list-inside text-sm text-slate-400 space-y-1">
            {variantGuidance.notes.map((note, index) => (
              <li key={index}>{note}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
