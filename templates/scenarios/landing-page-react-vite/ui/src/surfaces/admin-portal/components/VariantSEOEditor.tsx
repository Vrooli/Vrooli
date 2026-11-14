import { useState, useEffect, useCallback } from 'react';
import { Save, AlertCircle, CheckCircle, RefreshCw } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import { SEOPreview } from '../../../shared/ui/SEOPreview';
import { getVariantSEO, updateVariantSEO } from '../../../shared/api';
import type { VariantSEOConfigInput, SiteBranding } from '../../../shared/api';

interface VariantSEOEditorProps {
  variantSlug: string;
  variantName: string;
  siteBranding?: SiteBranding | null;
  onSave?: () => void;
}

export function VariantSEOEditor({
  variantSlug,
  variantName,
  siteBranding,
  onSave,
}: VariantSEOEditorProps) {
  const [seoConfig, setSeoConfig] = useState<VariantSEOConfigInput>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const fetchSEO = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getVariantSEO(variantSlug);
      setSeoConfig({
        title: data.title !== siteBranding?.defaultTitle ? data.title : undefined,
        description: data.description !== siteBranding?.defaultDescription ? data.description : undefined,
        ogTitle: data.ogTitle,
        ogDescription: data.ogDescription,
        ogImageUrl: data.ogImageUrl !== siteBranding?.defaultOgImageUrl ? data.ogImageUrl : undefined,
        twitterCard: data.twitterCard,
        noindex: data.noindex || false,
      });
    } catch (err) {
      setError('Failed to load SEO settings');
    } finally {
      setLoading(false);
    }
  }, [variantSlug, siteBranding]);

  useEffect(() => {
    fetchSEO();
  }, [fetchSEO]);

  const handleSave = async () => {
    setSaving(true);
    setError(null);
    setSuccess(false);

    try {
      await updateVariantSEO(variantSlug, seoConfig);

      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
      onSave?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  const updateField = <K extends keyof VariantSEOConfigInput>(
    field: K,
    value: VariantSEOConfigInput[K]
  ) => {
    setSeoConfig((prev) => ({ ...prev, [field]: value }));
    setSuccess(false);
  };

  // Compute preview values (variant overrides site defaults)
  const previewTitle = seoConfig.title || siteBranding?.defaultTitle || siteBranding?.siteName || 'Page Title';
  const previewDescription = seoConfig.description || siteBranding?.defaultDescription || '';
  const previewOgImage = seoConfig.ogImageUrl || siteBranding?.defaultOgImageUrl || undefined;
  const previewTwitterCard: 'summary' | 'summary_large_image' =
    seoConfig.twitterCard === 'summary' ? 'summary' : 'summary_large_image';

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <RefreshCw className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-white">SEO Settings</h3>
          <p className="text-sm text-slate-400">
            Customize meta tags for the "{variantName}" variant. Leave blank to use site defaults.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {success && (
            <div className="flex items-center gap-2 text-sm text-green-400">
              <CheckCircle className="h-4 w-4" />
              Saved
            </div>
          )}
          <Button onClick={handleSave} disabled={saving} className="gap-2">
            <Save className="h-4 w-4" />
            {saving ? 'Saving...' : 'Save SEO'}
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-400 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" />
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Form */}
        <div className="space-y-6">
          {/* Page Title */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Page Title
            </label>
            <input
              type="text"
              value={seoConfig.title || ''}
              onChange={(e) => updateField('title', e.target.value || undefined)}
              placeholder={siteBranding?.defaultTitle || 'Use site default'}
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
            <p className="text-xs text-slate-500">
              Recommended: 50-60 characters. Current: {(seoConfig.title || '').length}
            </p>
          </div>

          {/* Meta Description */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Meta Description
            </label>
            <textarea
              value={seoConfig.description || ''}
              onChange={(e) => updateField('description', e.target.value || undefined)}
              placeholder={siteBranding?.defaultDescription || 'Use site default'}
              rows={3}
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
            <p className="text-xs text-slate-500">
              Recommended: 150-160 characters. Current: {(seoConfig.description || '').length}
            </p>
          </div>

          {/* OG Title Override */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Social Share Title (Open Graph)
            </label>
            <input
              type="text"
              value={seoConfig.ogTitle || ''}
              onChange={(e) => updateField('ogTitle', e.target.value || undefined)}
              placeholder="Same as page title"
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {/* OG Description Override */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Social Share Description
            </label>
            <textarea
              value={seoConfig.ogDescription || ''}
              onChange={(e) => updateField('ogDescription', e.target.value || undefined)}
              placeholder="Same as meta description"
              rows={2}
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
          </div>

          {/* OG Image */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Social Share Image
            </label>
            <ImageUploader
              value={seoConfig.ogImageUrl}
              onChange={(url) => updateField('ogImageUrl', url || undefined)}
              category="og_image"
              placeholder={siteBranding?.defaultOgImageUrl || 'No image set'}
              uploadLabel="Upload OG Image"
              previewSize="lg"
            />
            <p className="text-xs text-slate-500">
              Recommended: 1200x630px for best display on social platforms
            </p>
          </div>

          {/* Twitter Card Type */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Twitter Card Type
            </label>
            <select
              value={seoConfig.twitterCard || 'summary_large_image'}
              onChange={(e) => updateField('twitterCard', e.target.value || undefined)}
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            >
              <option value="summary_large_image">Large Image Card</option>
              <option value="summary">Summary Card (Small Image)</option>
            </select>
          </div>

          {/* Canonical Path */}
          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-300">
              Canonical Path
            </label>
            <input
              type="text"
              value={seoConfig.canonicalPath || ''}
              onChange={(e) => updateField('canonicalPath', e.target.value || undefined)}
              placeholder="/"
              className="w-full rounded-lg border border-white/10 bg-slate-900/70 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
            <p className="text-xs text-slate-500">
              Path relative to your domain (e.g., /landing, /promo)
            </p>
          </div>

          {/* NoIndex Toggle */}
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="noindex"
              checked={seoConfig.noindex || false}
              onChange={(e) => updateField('noindex', e.target.checked)}
              className="h-4 w-4 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-blue-500"
            />
            <label htmlFor="noindex" className="text-sm text-slate-300">
              Hide from search engines (noindex)
            </label>
          </div>
        </div>

        {/* Preview */}
        <div className="space-y-4">
          <h4 className="text-sm font-medium text-slate-300">Preview</h4>
          <SEOPreview
            title={previewTitle}
            description={previewDescription}
            url={siteBranding?.canonicalBaseUrl || 'https://example.com'}
            ogImage={previewOgImage || undefined}
            siteName={siteBranding?.siteName}
            favicon={siteBranding?.faviconUrl || undefined}
            twitterCard={previewTwitterCard}
          />
        </div>
      </div>
    </div>
  );
}
