import { useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Eye, FileEdit } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { RuntimeSignalStrip } from '../components/RuntimeSignalStrip';
import { FormField, inputClassName, textareaClassName } from '../components/FormField';
import { Callout } from '../components/Callout';
import { LAYOUT } from '../config/layout.constants';
import { Button } from '../../../shared/ui/button';
import { useToast } from '../../../shared/ui/Toast';
import type { ContentSection, LandingConfigResponse } from '../../../shared/api';
import { useSectionForm } from '../hooks/useSectionForm';
import {
  VariantSectionTimeline,
  VariantContextCard,
  StylingGuardrailsCard,
  PreviewPanel,
} from '../components/SectionEditorComponents';

// Import section renderers
import { HeroSection } from '../../public-landing/sections/HeroSection';
import { FeaturesSection } from '../../public-landing/sections/FeaturesSection';
import { PricingSection } from '../../public-landing/sections/PricingSection';
import { CTASection } from '../../public-landing/sections/CTASection';
import { TestimonialsSection } from '../../public-landing/sections/TestimonialsSection';
import { FAQSection } from '../../public-landing/sections/FAQSection';
import { FooterSection } from '../../public-landing/sections/FooterSection';
import { VideoSection } from '../../public-landing/sections/VideoSection';
import { DownloadSection } from '../../public-landing/sections/DownloadSection';

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

/**
 * Section Editor - Split-screen form + live preview
 * Implements OT-P0-012 (Split customization layout: form + preview, stacked on mobile)
 * Implements OT-P0-013 (Live preview updates within 300ms without page reload)
 *
 * [REQ:CUSTOM-SPLIT] [REQ:CUSTOM-LIVE]
 */
export function SectionEditor() {
  const navigate = useNavigate();
  const { variantSlug, sectionId } = useParams<{ variantSlug: string; sectionId: string }>();
  const toast = useToast();

  const {
    // Section state
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
  } = useSectionForm({
    variantSlug,
    sectionId,
    onSuccess: (message, title) => toast.success(message, title),
    onError: (message) => toast.error(message),
  });

  const previewRenderer = useMemo(() => SECTION_PREVIEW_RENDERERS[sectionType], [sectionType]);

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
    <AdminLayout maxWidth="extraWide">
      <div className={LAYOUT.pageSpacing}>
        <RuntimeSignalStrip mode="compact" />

        <PageHeader
          variant="icon-title"
          title={isNew ? 'New Section' : `Edit ${sectionType} Section`}
          icon={FileEdit}
          iconBgClass="bg-teal-500/10"
          iconColorClass="text-teal-400"
          testId="section-editor-header"
          actions={
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => navigate(`/admin/customization/variants/${variantSlug}`)}
                className="gap-2"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </Button>
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
            </>
          }
        />

        {error && (
          <Callout type="error" message={error} className="mb-6" />
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
            <StylingGuardrailsCard variantSlug={variantContext?.variant?.slug ?? variantSlug} />

            {/* Section Settings */}
            <div className="${LAYOUT.card.base} rounded-xl p-6">
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
            <div className="${LAYOUT.card.base} rounded-xl p-6">
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
            <div className="${LAYOUT.card.base} rounded-xl p-6 space-y-4">
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
                    <p className="mt-1 text-[11px] text-slate-500">Loading variant list...</p>
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
                  Syncing landing runtime context...
                </div>
              )}
              {previewConfigError && (
                <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
                  {previewConfigError}
                </div>
              )}
              {compareLoading && (
                <div className="text-xs text-slate-500">Loading comparison variant...</div>
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
