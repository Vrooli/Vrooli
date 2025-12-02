import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import { SEOPreview } from '../../../shared/ui/SEOPreview';
import { getBranding, updateBranding, clearBrandingField, type SiteBranding } from '../../../shared/api';
import { Palette, RefreshCw, Globe, Type, Search, X, ExternalLink, CheckCircle2, AlertCircle } from 'lucide-react';

interface BrandingFormState {
  site_name: string;
  tagline: string;
  logo_url: string;
  logo_icon_url: string;
  favicon_url: string;
  apple_touch_icon_url: string;
  default_title: string;
  default_description: string;
  default_og_image_url: string;
  theme_primary_color: string;
  theme_background_color: string;
  canonical_base_url: string;
  google_site_verification: string;
  robots_txt: string;
}

const defaultForm: BrandingFormState = {
  site_name: '',
  tagline: '',
  logo_url: '',
  logo_icon_url: '',
  favicon_url: '',
  apple_touch_icon_url: '',
  default_title: '',
  default_description: '',
  default_og_image_url: '',
  theme_primary_color: '',
  theme_background_color: '',
  canonical_base_url: '',
  google_site_verification: '',
  robots_txt: '',
};

function brandingToForm(branding: SiteBranding): BrandingFormState {
  return {
    site_name: branding.site_name ?? '',
    tagline: branding.tagline ?? '',
    logo_url: branding.logo_url ?? '',
    logo_icon_url: branding.logo_icon_url ?? '',
    favicon_url: branding.favicon_url ?? '',
    apple_touch_icon_url: branding.apple_touch_icon_url ?? '',
    default_title: branding.default_title ?? '',
    default_description: branding.default_description ?? '',
    default_og_image_url: branding.default_og_image_url ?? '',
    theme_primary_color: branding.theme_primary_color ?? '',
    theme_background_color: branding.theme_background_color ?? '',
    canonical_base_url: branding.canonical_base_url ?? '',
    google_site_verification: branding.google_site_verification ?? '',
    robots_txt: branding.robots_txt ?? '',
  };
}

export function BrandingSettings() {
  const [branding, setBranding] = useState<SiteBranding | null>(null);
  const [form, setForm] = useState<BrandingFormState>(defaultForm);
  const [originalForm, setOriginalForm] = useState<BrandingFormState>(defaultForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const loadBranding = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getBranding();
      setBranding(data);
      const formData = brandingToForm(data);
      setForm(formData);
      setOriginalForm(formData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load branding');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBranding();
  }, [loadBranding]);

  const handleInput = (field: keyof BrandingFormState) => (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    setForm((prev) => ({ ...prev, [field]: event.target.value }));
    setSuccessMessage(null);
  };

  const handleImageChange = (field: keyof BrandingFormState) => (url: string | null) => {
    setForm((prev) => ({ ...prev, [field]: url ?? '' }));
    setSuccessMessage(null);
  };

  const handleClearField = async (field: keyof BrandingFormState) => {
    try {
      const data = await clearBrandingField(field);
      setBranding(data);
      const formData = brandingToForm(data);
      setForm(formData);
      setOriginalForm(formData);
      setSuccessMessage(`Cleared ${field.replace(/_/g, ' ')}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear field');
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const payload: Record<string, string> = {};
      (Object.keys(form) as (keyof BrandingFormState)[]).forEach((key) => {
        const current = form[key].trim();
        const original = originalForm[key].trim();
        if (current !== original && current.length > 0) {
          payload[key] = current;
        }
      });

      if (Object.keys(payload).length === 0) {
        setError('No changes to save');
        setSaving(false);
        return;
      }

      const updated = await updateBranding(payload);
      setBranding(updated);
      const formData = brandingToForm(updated);
      setForm(formData);
      setOriginalForm(formData);
      setSuccessMessage('Branding updated successfully');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update branding');
    } finally {
      setSaving(false);
    }
  };

  const isDirty = JSON.stringify(form) !== JSON.stringify(originalForm);

  // Compute branding setup health
  const brandingHealth = useMemo(() => {
    const checks = {
      identity: Boolean(form.site_name && form.logo_url),
      favicon: Boolean(form.favicon_url),
      seo: Boolean(form.default_title && form.default_description),
      ogImage: Boolean(form.default_og_image_url),
    };
    const configured = Object.values(checks).filter(Boolean).length;
    const total = Object.keys(checks).length;
    return { checks, configured, total, percentage: Math.round((configured / total) * 100) };
  }, [form]);

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  const renderColorPreview = (color: string) => {
    if (!color) return null;
    return (
      <div
        className="mt-2 h-8 w-16 rounded-lg border border-white/10"
        style={{ backgroundColor: color }}
        title={color}
      />
    );
  };

  return (
    <AdminLayout>
      <div className="space-y-8">
        {/* Enhanced Header with Purpose Statement */}
        <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6" data-testid="branding-header">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="flex-1">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Branding</p>
              <h1 className="text-2xl font-bold text-white mt-1">Configure how your landing page looks and ranks</h1>
              <p className="text-slate-400 text-sm mt-2">
                Set your site identity, colors, and SEO defaults. These settings apply site-wide and can be overridden per-variant for specific sections.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={previewPublicLanding} className="gap-2" data-testid="branding-preview">
                <ExternalLink className="h-4 w-4" />
                Preview landing
              </Button>
              <Button variant="ghost" size="sm" onClick={loadBranding} className="gap-2" data-testid="branding-refresh">
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </div>
          </div>

          {/* Setup Completeness Indicator */}
          {!loading && (
            <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4" data-testid="branding-health">
              <BrandingHealthBadge
                label="Site identity"
                configured={brandingHealth.checks.identity}
                description={brandingHealth.checks.identity ? 'Name and logo set' : 'Add site name and logo'}
              />
              <BrandingHealthBadge
                label="Favicon"
                configured={brandingHealth.checks.favicon}
                description={brandingHealth.checks.favicon ? 'Browser icon set' : 'Upload a favicon'}
              />
              <BrandingHealthBadge
                label="SEO defaults"
                configured={brandingHealth.checks.seo}
                description={brandingHealth.checks.seo ? 'Title and description set' : 'Add page title and description'}
              />
              <BrandingHealthBadge
                label="Social preview"
                configured={brandingHealth.checks.ogImage}
                description={brandingHealth.checks.ogImage ? 'OG image uploaded' : 'Upload social share image'}
              />
            </div>
          )}
        </div>

        {loading ? (
          <div className="text-slate-400">Loading branding settings...</div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-8">
            {/* Site Identity */}
            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Type className="h-5 w-5 text-blue-300" /> Site Identity
                </CardTitle>
                <CardDescription>
                  Your site name, tagline, and brand imagery
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Site Name
                    </label>
                    <input
                      type="text"
                      value={form.site_name}
                      onChange={handleInput('site_name')}
                      placeholder="My Landing Page"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Tagline
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={form.tagline}
                        onChange={handleInput('tagline')}
                        placeholder="Your catchy tagline"
                        className="mt-1 flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      {form.tagline && (
                        <button
                          type="button"
                          onClick={() => handleClearField('tagline')}
                          className="mt-1 p-2 text-slate-400 hover:text-rose-400"
                          title="Clear tagline"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>

                {/* Logo Upload */}
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">
                      Logo
                    </label>
                    <ImageUploader
                      value={form.logo_url}
                      onChange={handleImageChange('logo_url')}
                      category="logo"
                      placeholder="Upload your logo"
                      uploadLabel="Upload Logo"
                      previewSize="lg"
                      alt="Site logo"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">
                      Logo Icon (Square)
                    </label>
                    <ImageUploader
                      value={form.logo_icon_url}
                      onChange={handleImageChange('logo_icon_url')}
                      category="logo"
                      placeholder="Upload square icon"
                      uploadLabel="Upload Icon"
                      previewSize="md"
                      alt="Site icon"
                    />
                  </div>
                </div>

                {/* Favicon Upload */}
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">
                      Favicon
                    </label>
                    <ImageUploader
                      value={form.favicon_url}
                      onChange={handleImageChange('favicon_url')}
                      category="favicon"
                      placeholder="Upload favicon"
                      uploadLabel="Upload Favicon"
                      previewSize="sm"
                      accept="image/png,image/x-icon,image/vnd.microsoft.icon,image/ico"
                      alt="Favicon"
                    />
                    <p className="mt-1 text-xs text-slate-500">
                      Recommended: 32x32 or 16x16 pixels
                    </p>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">
                      Apple Touch Icon
                    </label>
                    <ImageUploader
                      value={form.apple_touch_icon_url}
                      onChange={handleImageChange('apple_touch_icon_url')}
                      category="favicon"
                      placeholder="Upload touch icon"
                      uploadLabel="Upload Icon"
                      previewSize="md"
                      alt="Apple touch icon"
                    />
                    <p className="mt-1 text-xs text-slate-500">
                      Recommended: 180x180 pixels for iOS
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Theme Colors */}
            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Palette className="h-5 w-5 text-purple-300" /> Theme Colors
                </CardTitle>
                <CardDescription>
                  Customize primary accent and background colors
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Primary Accent Color
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={form.theme_primary_color}
                        onChange={handleInput('theme_primary_color')}
                        placeholder="#3B82F6"
                        className="mt-1 flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      <input
                        type="color"
                        value={form.theme_primary_color || '#3B82F6'}
                        onChange={(e) => setForm((prev) => ({ ...prev, theme_primary_color: e.target.value }))}
                        className="mt-1 h-10 w-10 cursor-pointer rounded-lg border border-white/10 bg-slate-900/70"
                      />
                      {form.theme_primary_color && (
                        <button
                          type="button"
                          onClick={() => handleClearField('theme_primary_color')}
                          className="mt-1 p-2 text-slate-400 hover:text-rose-400"
                          title="Clear color"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                    {renderColorPreview(form.theme_primary_color)}
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Background Color
                    </label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={form.theme_background_color}
                        onChange={handleInput('theme_background_color')}
                        placeholder="#07090F"
                        className="mt-1 flex-1 rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      <input
                        type="color"
                        value={form.theme_background_color || '#07090F'}
                        onChange={(e) => setForm((prev) => ({ ...prev, theme_background_color: e.target.value }))}
                        className="mt-1 h-10 w-10 cursor-pointer rounded-lg border border-white/10 bg-slate-900/70"
                      />
                      {form.theme_background_color && (
                        <button
                          type="button"
                          onClick={() => handleClearField('theme_background_color')}
                          className="mt-1 p-2 text-slate-400 hover:text-rose-400"
                          title="Clear color"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      )}
                    </div>
                    {renderColorPreview(form.theme_background_color)}
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Default SEO */}
            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Search className="h-5 w-5 text-green-300" /> Default SEO
                </CardTitle>
                <CardDescription>
                  Default meta tags for search engines and social sharing (can be overridden per-variant)
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                  {/* Form Fields */}
                  <div className="space-y-6">
                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Default Page Title
                      </label>
                      <input
                        type="text"
                        value={form.default_title}
                        onChange={handleInput('default_title')}
                        placeholder="My Amazing Product - Tagline Here"
                        maxLength={60}
                        className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      <p className="mt-1 text-xs text-slate-500">
                        {form.default_title.length}/60 characters (recommended)
                      </p>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Default Description
                      </label>
                      <textarea
                        value={form.default_description}
                        onChange={handleInput('default_description')}
                        placeholder="A compelling description of your product or service..."
                        rows={3}
                        maxLength={160}
                        className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                      />
                      <p className="mt-1 text-xs text-slate-500">
                        {form.default_description.length}/160 characters (recommended)
                      </p>
                    </div>

                    <div>
                      <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">
                        Default OG Image (Social Preview)
                      </label>
                      <ImageUploader
                        value={form.default_og_image_url}
                        onChange={handleImageChange('default_og_image_url')}
                        category="og_image"
                        placeholder="Upload social preview image"
                        uploadLabel="Upload OG Image"
                        previewSize="xl"
                        alt="Social preview image"
                      />
                      <p className="mt-1 text-xs text-slate-500">
                        Recommended: 1200x630 pixels for optimal social media display
                      </p>
                    </div>
                  </div>

                  {/* Live Preview */}
                  <div className="space-y-4">
                    <div>
                      <h4 className="text-sm font-medium text-slate-300 mb-4">Live Preview</h4>
                      <p className="text-xs text-slate-500 mb-4">
                        See how your site appears in search results and social shares. Updates as you type.
                      </p>
                    </div>
                    <SEOPreview
                      title={form.default_title || form.site_name || 'Page Title'}
                      description={form.default_description || 'Your page description will appear here...'}
                      url={form.canonical_base_url || 'https://example.com'}
                      ogImage={form.default_og_image_url || undefined}
                      siteName={form.site_name || undefined}
                      favicon={form.favicon_url || undefined}
                      twitterCard="summary_large_image"
                    />
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Technical Settings */}
            <Card className="border-white/10 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Globe className="h-5 w-5 text-cyan-300" /> Technical Settings
                </CardTitle>
                <CardDescription>
                  Canonical URLs, verification codes, and robots.txt
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Canonical Base URL
                    </label>
                    <input
                      type="url"
                      value={form.canonical_base_url}
                      onChange={handleInput('canonical_base_url')}
                      placeholder="https://example.com"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                    <p className="mt-1 text-xs text-slate-500">
                      Used for canonical URLs and sitemap generation
                    </p>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                      Google Site Verification
                    </label>
                    <input
                      type="text"
                      value={form.google_site_verification}
                      onChange={handleInput('google_site_verification')}
                      placeholder="verification-code"
                      className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 text-sm text-white"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wide text-slate-400">
                    robots.txt Content
                  </label>
                  <textarea
                    value={form.robots_txt}
                    onChange={handleInput('robots_txt')}
                    placeholder={'User-agent: *\nAllow: /'}
                    rows={5}
                    className="mt-1 w-full rounded-lg border border-white/10 bg-slate-900/70 px-3 py-2 font-mono text-sm text-white"
                  />
                </div>
              </CardContent>
            </Card>

            {/* Save Button */}
            <div className="flex items-center gap-4">
              <Button type="submit" disabled={!isDirty || saving} className="gap-2">
                {saving && <RefreshCw className="h-4 w-4 animate-spin" />}
                {isDirty ? 'Save Changes' : 'No Changes'}
              </Button>
              {error && <p className="text-sm text-rose-300">{error}</p>}
              {successMessage && <p className="text-sm text-emerald-300">{successMessage}</p>}
              {branding?.updated_at && (
                <p className="text-xs text-slate-500">
                  Last updated: {new Date(branding.updated_at).toLocaleString()}
                </p>
              )}
            </div>
          </form>
        )}
      </div>
    </AdminLayout>
  );
}

interface BrandingHealthBadgeProps {
  label: string;
  configured: boolean;
  description: string;
}

function BrandingHealthBadge({ label, configured, description }: BrandingHealthBadgeProps) {
  return (
    <div
      className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
        configured
          ? 'border-emerald-500/30 bg-emerald-500/10'
          : 'border-amber-500/30 bg-amber-500/10'
      }`}
    >
      {configured ? (
        <CheckCircle2 className="h-5 w-5 text-emerald-300" />
      ) : (
        <AlertCircle className="h-5 w-5 text-amber-300" />
      )}
      <div>
        <p className="text-sm font-semibold text-white">{label}</p>
        <p className="text-xs text-slate-400">{description}</p>
      </div>
    </div>
  );
}
