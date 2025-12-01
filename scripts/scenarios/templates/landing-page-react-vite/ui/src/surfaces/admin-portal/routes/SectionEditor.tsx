import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Eye } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../../../shared/ui/button';
import type { ContentSection } from '../../../shared/api';
import { useDebounce } from '../../../shared/hooks/useDebounce';
import {
  loadSectionEditor,
  persistExistingSectionContent,
  loadVariantContext,
  type SectionEditorState,
  type VariantContext,
} from '../controllers/sectionEditorController';
import { getStylingConfig, getVariantStylingGuidance } from '../../../shared/lib/stylingConfig';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';

/**
 * Section Editor - Split-screen form + live preview
 * Implements OT-P0-012 (Split customization layout: form + preview, stacked on mobile)
 * Implements OT-P0-013 (Live preview updates within 300ms without page reload)
 *
 * [REQ:CUSTOM-SPLIT] [REQ:CUSTOM-LIVE]
 */
const STYLING_CONFIG = getStylingConfig();

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
                    <option value="testimonials">Testimonials</option>
                    <option value="faq">FAQ</option>
                    <option value="footer">Footer</option>
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
            <div className="bg-white/5 border border-white/10 rounded-xl p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <Eye className="h-5 w-5" />
                  Live Preview
                </h2>
                <span className="text-xs text-slate-500">Updates in 300ms</span>
              </div>

              {/* Preview Panel */}
              <div className="bg-slate-900 rounded-lg border border-white/10 p-8" data-testid="section-preview">
                {!enabled && (
                  <div className="mb-4 px-3 py-2 bg-amber-500/20 border border-amber-500/30 rounded text-amber-400 text-sm">
                    Section is currently disabled
                  </div>
                )}

                {/* Hero Section Preview */}
                {sectionType === 'hero' && (
                  <div className="text-center space-y-6">
                    {debouncedContent.image_url && (
                      <div className="mb-6">
                        <img
                          src={debouncedContent.image_url as string}
                          alt="Hero"
                          className="max-w-full h-auto rounded-lg"
                          onError={(e) => {
                            (e.target as HTMLImageElement).style.display = 'none';
                          }}
                        />
                      </div>
                    )}
                    <h1 className="text-4xl font-bold">{debouncedContent.title || 'Your Title Here'}</h1>
                    <p className="text-xl text-slate-300">{debouncedContent.subtitle || 'Your subtitle here'}</p>
                    {debouncedContent.cta_text && (
                      <button className="px-6 py-3 bg-blue-600 hover:bg-blue-700 rounded-lg font-semibold transition-colors">
                        {debouncedContent.cta_text}
                      </button>
                    )}
                  </div>
                )}

                {/* CTA Section Preview */}
                {sectionType === 'cta' && (
                  <div className="text-center space-y-4 py-8 px-6 bg-blue-500/10 rounded-lg border border-blue-500/20">
                    <h2 className="text-3xl font-bold">{debouncedContent.title || 'Call to Action'}</h2>
                    <p className="text-lg text-slate-300">{debouncedContent.subtitle || 'Subtitle goes here'}</p>
                    {debouncedContent.cta_text && (
                      <button className="px-8 py-4 bg-blue-600 hover:bg-blue-700 rounded-lg font-semibold text-lg transition-colors">
                        {debouncedContent.cta_text}
                      </button>
                    )}
                  </div>
                )}

                {/* Generic Preview for other types */}
                {!['hero', 'cta'].includes(sectionType) && (
                  <div className="space-y-4">
                    <h2 className="text-2xl font-bold capitalize">{sectionType}</h2>
                    <h3 className="text-xl">{debouncedContent.title || 'Section Title'}</h3>
                    <p className="text-slate-300">{debouncedContent.subtitle || 'Section content will appear here'}</p>
                    {debouncedContent.cta_text && (
                      <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded transition-colors">
                        {debouncedContent.cta_text}
                      </button>
                    )}
                  </div>
                )}
              </div>

              <div className="mt-4 text-xs text-slate-500">
                Preview automatically updates as you type (debounced 300ms)
              </div>
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
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
