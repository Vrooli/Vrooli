import { useEffect, useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Eye } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../components/ui/button';
import { getSection, updateSection, createSection, type ContentSection } from '../lib/api';

// Debounce hook for live preview updates (OT-P0-013: 300ms)
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

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
  const isNew = sectionId === 'new';

  const [section, setSection] = useState<ContentSection | null>(null);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

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
    if (!isNew && sectionId) {
      fetchSection();
    }
  }, [sectionId]);

  const fetchSection = async () => {
    if (!sectionId || isNew) return;

    try {
      setLoading(true);
      const sectionData = await getSection(parseInt(sectionId));
      setSection(sectionData);
      setSectionType(sectionData.section_type);
      setEnabled(sectionData.enabled);
      setOrder(sectionData.order);
      setContent(sectionData.content);
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
      } else if (sectionId) {
        await updateSection(parseInt(sectionId), content);
        await fetchSection();
      }

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
