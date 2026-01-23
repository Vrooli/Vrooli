import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField, inputClassName, textareaClassName } from '../components/FormField';
import { StatusBadge, StatusBadgeGrid } from '../components/StatusBadge';
import { LabelWithHelp } from '../components/LabelWithHelp';
import { PasswordInput } from '../components/PasswordInput';
import { ClearableInput } from '../components/ClearableInput';
import { Callout } from '../components/Callout';
import { Button } from '../../../shared/ui/button';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import { SEOPreview } from '../../../shared/ui/SEOPreview';
import { Palette, RefreshCw, Globe, Type, Search, X, ExternalLink, MessageCircle, Mail, Clock } from 'lucide-react';
import { useBrandingForm } from '../hooks/useBrandingForm';
import { LAYOUT } from '../config/layout.constants';

export function BrandingSettings() {
  const {
    branding,
    form,
    loading,
    saving,
    error,
    successMessage,
    isDirty,
    brandingHealth,
    loadBrandingData,
    handleInput,
    handleFieldChange,
    handleImageChange,
    applyLogoDerivatives,
    applyFaviconDerivatives,
    applyOgDerivatives,
    handleClearField,
    handleSubmit,
    toggleComingSoon,
    previewPublicLanding,
  } = useBrandingForm();

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
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Configure how your landing page looks and ranks"
          description="Set your site identity, colors, and SEO defaults. These settings apply site-wide and can be overridden per-variant for specific sections."
          icon={Palette}
          iconBgClass="bg-pink-500/10"
          iconColorClass="text-pink-400"
          testId="branding-header"
          actions={
            <>
              <Button variant="outline" size="sm" onClick={previewPublicLanding} className="gap-2" data-testid="branding-preview">
                <ExternalLink className="h-4 w-4" />
                Preview landing
              </Button>
              <Button variant="ghost" size="sm" onClick={loadBrandingData} className="gap-2" data-testid="branding-refresh">
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </>
          }
        />

        {/* Setup Completeness Indicator */}
        {!loading && (
          <StatusBadgeGrid
            testId="branding-health"
            columns={4}
            badges={[
              {
                label: 'Site identity',
                status: brandingHealth.checks.identity ? 'success' : 'warning',
                description: brandingHealth.checks.identity ? 'Name and logo set' : 'Add site name and logo',
              },
              {
                label: 'Favicon',
                status: brandingHealth.checks.favicon ? 'success' : 'warning',
                description: brandingHealth.checks.favicon ? 'Browser icon set' : 'Upload a favicon',
              },
              {
                label: 'SEO defaults',
                status: brandingHealth.checks.seo ? 'success' : 'warning',
                description: brandingHealth.checks.seo ? 'Title and description set' : 'Add page title and description',
              },
              {
                label: 'Social preview',
                status: brandingHealth.checks.ogImage ? 'success' : 'warning',
                description: brandingHealth.checks.ogImage ? 'OG image uploaded' : 'Upload social share image',
              },
            ]}
          />
        )}

        {loading ? (
          <div className="text-slate-400">Loading branding settings...</div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-8">
            {/* Site Identity */}
            <FormSection
              title="Site Identity"
              description="Your site name, tagline, and brand imagery"
              icon={Type}
              iconColorClass="text-blue-300"
              testId="branding-identity-section"
            >
              <div className="space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <FormField label="Site Name">
                    <input
                      type="text"
                      value={form.site_name}
                      onChange={handleInput('site_name')}
                      placeholder="My Landing Page"
                      className={inputClassName}
                    />
                  </FormField>
                  <FormField label="Tagline">
                    <ClearableInput
                      value={form.tagline}
                      onChange={handleInput('tagline')}
                      onClear={() => handleClearField('tagline')}
                      placeholder="Your catchy tagline"
                    />
                  </FormField>
                </div>

                {/* Logo Upload */}
                <div className="grid gap-6 md:grid-cols-2">
                  <FormField label="Logo" helpText="Upload a high-quality logo (any size). We auto-generate multiple sizes and a square icon.">
                    <ImageUploader
                      value={form.logo_url}
                      onChange={handleImageChange('logo_url')}
                      onUploadComplete={applyLogoDerivatives}
                      category="logo"
                      placeholder="Upload your logo"
                      uploadLabel="Upload Logo"
                      previewSize="lg"
                      alt="Site logo"
                    />
                  </FormField>
                  <FormField label="Logo Icon (Square)">
                    <ImageUploader
                      value={form.logo_icon_url}
                      onChange={handleImageChange('logo_icon_url')}
                      onUploadComplete={applyLogoDerivatives}
                      category="logo"
                      placeholder="Upload square icon"
                      uploadLabel="Upload Icon"
                      previewSize="md"
                      alt="Site icon"
                    />
                  </FormField>
                </div>

                {/* Favicon Upload */}
                <div className="grid gap-6 md:grid-cols-2">
                  <FormField label="Favicon" helpText="Recommended: 32x32 or 16x16 pixels">
                    <ImageUploader
                      value={form.favicon_url}
                      onChange={handleImageChange('favicon_url')}
                      onUploadComplete={applyFaviconDerivatives}
                      category="favicon"
                      placeholder="Upload favicon"
                      uploadLabel="Upload Favicon"
                      previewSize="sm"
                      accept="image/png,image/x-icon,image/vnd.microsoft.icon,image/ico"
                      alt="Favicon"
                    />
                  </FormField>
                  <FormField label="Apple Touch Icon" helpText="Upload any size; we generate 180px touch icon plus small favicon sizes automatically.">
                    <ImageUploader
                      value={form.apple_touch_icon_url}
                      onChange={handleImageChange('apple_touch_icon_url')}
                      onUploadComplete={applyFaviconDerivatives}
                      category="favicon"
                      placeholder="Upload touch icon"
                      uploadLabel="Upload Icon"
                      previewSize="md"
                      alt="Apple touch icon"
                    />
                  </FormField>
                </div>
              </div>
            </FormSection>

            {/* Coming Soon Mode */}
            <FormSection
              title="Coming Soon Mode"
              description="Show a coming soon page to visitors while you prepare your landing page"
              icon={Clock}
              iconColorClass="text-amber-300"
              testId="branding-coming-soon-section"
            >
              <div className="space-y-6">
                {/* Toggle Switch */}
                <div className="flex items-center justify-between">
                  <div>
                    <label htmlFor="coming-soon-toggle" className="text-sm font-medium text-white">
                      Enable Coming Soon Mode
                    </label>
                    <p className="text-xs text-slate-400 mt-1">
                      When enabled, visitors see a "coming soon" page with email signup
                    </p>
                  </div>
                  <button
                    id="coming-soon-toggle"
                    type="button"
                    role="switch"
                    aria-checked={form.coming_soon_enabled}
                    onClick={toggleComingSoon}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      form.coming_soon_enabled ? 'bg-amber-500' : 'bg-slate-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        form.coming_soon_enabled ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>

                {/* Custom Message (only shown when enabled) */}
                {form.coming_soon_enabled && (
                  <>
                    <FormField label="Custom Message (Optional)" helpText="Leave empty to use the default message">
                      <textarea
                        value={form.coming_soon_message}
                        onChange={handleInput('coming_soon_message')}
                        placeholder="We are working hard to bring you something amazing. Stay tuned!"
                        rows={3}
                        className={textareaClassName}
                      />
                    </FormField>

                    <Callout
                      type="info"
                      title="Admin panel remains accessible"
                      message={
                        <>
                          The admin panel at <code className="bg-slate-800 px-1 rounded">/admin</code> will
                          still be accessible for you to manage your site. Only public routes are affected.
                        </>
                      }
                    />
                  </>
                )}
              </div>
            </FormSection>

            {/* Theme Colors */}
            <FormSection
              title="Theme Colors"
              description="Customize primary accent and background colors"
              icon={Palette}
              iconColorClass="text-purple-300"
              testId="branding-theme-section"
            >
              <div className="grid gap-6 md:grid-cols-2">
                <FormField label="Primary Accent Color">
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={form.theme_primary_color}
                      onChange={handleInput('theme_primary_color')}
                      placeholder="#3B82F6"
                      className={`flex-1 ${inputClassName}`}
                    />
                    <input
                      type="color"
                      value={form.theme_primary_color || '#3B82F6'}
                      onChange={(e) => handleFieldChange('theme_primary_color', e.target.value)}
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
                </FormField>
                <FormField label="Background Color">
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={form.theme_background_color}
                      onChange={handleInput('theme_background_color')}
                      placeholder="#07090F"
                      className={`flex-1 ${inputClassName}`}
                    />
                    <input
                      type="color"
                      value={form.theme_background_color || '#07090F'}
                      onChange={(e) => handleFieldChange('theme_background_color', e.target.value)}
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
                </FormField>
              </div>
            </FormSection>

            {/* Default SEO */}
            <FormSection
              title="Default SEO"
              description="Default meta tags for search engines and social sharing (can be overridden per-variant)"
              icon={Search}
              iconColorClass="text-green-300"
              testId="branding-seo-section"
            >
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
                {/* Form Fields */}
                <div className="space-y-6">
                  <FormField label="Default Page Title" charCount={{ current: form.default_title.length, max: 60 }}>
                    <input
                      type="text"
                      value={form.default_title}
                      onChange={handleInput('default_title')}
                      placeholder="My Amazing Product - Tagline Here"
                      maxLength={60}
                      className={inputClassName}
                    />
                  </FormField>

                  <FormField label="Default Description" charCount={{ current: form.default_description.length, max: 160 }}>
                    <textarea
                      value={form.default_description}
                      onChange={handleInput('default_description')}
                      placeholder="A compelling description of your product or service..."
                      rows={3}
                      maxLength={160}
                      className={textareaClassName}
                    />
                  </FormField>

                  <FormField label="Default OG Image (Social Preview)" helpText="We resize to 1200x630 for you so social shares look crisp.">
                    <ImageUploader
                      value={form.default_og_image_url}
                      onChange={handleImageChange('default_og_image_url')}
                      onUploadComplete={applyOgDerivatives}
                      category="og_image"
                      placeholder="Upload social preview image"
                      uploadLabel="Upload OG Image"
                      previewSize="xl"
                      alt="Social preview image"
                    />
                  </FormField>
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
            </FormSection>

            {/* Technical Settings */}
            <FormSection
              title="Technical Settings"
              description="Canonical URLs, verification codes, and robots.txt"
              icon={Globe}
              iconColorClass="text-cyan-300"
              testId="branding-technical-section"
            >
              <div className="space-y-6">
                <div className="grid gap-6 md:grid-cols-2">
                  <FormField label="Canonical Base URL" helpText="Used for canonical URLs and sitemap generation">
                    <input
                      type="url"
                      value={form.canonical_base_url}
                      onChange={handleInput('canonical_base_url')}
                      placeholder="https://example.com"
                      className={inputClassName}
                    />
                  </FormField>
                  <FormField label="Google Site Verification">
                    <input
                      type="text"
                      value={form.google_site_verification}
                      onChange={handleInput('google_site_verification')}
                      placeholder="verification-code"
                      className={inputClassName}
                    />
                  </FormField>
                </div>

                <FormField label="robots.txt Content">
                  <textarea
                    value={form.robots_txt}
                    onChange={handleInput('robots_txt')}
                    placeholder={'User-agent: *\nAllow: /'}
                    rows={5}
                    className={`${textareaClassName} font-mono`}
                  />
                </FormField>
              </div>
            </FormSection>

            {/* Support Settings */}
            <FormSection
              title="Support"
              description="Configure how visitors can get help (async-first, no sales calls)"
              icon={MessageCircle}
              iconColorClass="text-orange-300"
              testId="branding-support-section"
            >
              <div className="space-y-6">
                <FormField
                  label="AI Chat Agent URL"
                  helpText="Link to your AI assistant (ChatGPT GPT, Claude, or custom chat). Shown in the FAQ section as 'Ask our AI assistant'. Leave empty to hide the support CTA."
                >
                  <ClearableInput
                    type="url"
                    value={form.support_chat_url}
                    onChange={handleInput('support_chat_url')}
                    onClear={() => handleClearField('support_chat_url')}
                    placeholder="https://chat.openai.com/g/g-your-gpt-id"
                  />
                </FormField>

                <FormField label="Support Email" helpText="Feedback submissions from /feedback will be sent to this email address.">
                  <ClearableInput
                    type="email"
                    value={form.support_email}
                    onChange={handleInput('support_email')}
                    onClear={() => handleClearField('support_email')}
                    placeholder="support@yourcompany.com"
                  />
                </FormField>

                {/* SMTP Configuration - only show when support email is set */}
                {form.support_email && (
                  <div className="rounded-xl border border-white/10 bg-slate-800/50 p-4 space-y-4">
                    <div className="flex items-center gap-2">
                      <Mail className="h-4 w-4 text-slate-400" />
                      <span className="text-sm font-medium text-white">Email Server Settings</span>
                    </div>
                    <p className="text-xs text-slate-400">
                      Configure SMTP to send email notifications when feedback is submitted.
                    </p>

                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <LabelWithHelp
                          label="SMTP Host"
                          help="Your email provider's SMTP server address. Common examples: smtp.gmail.com (Gmail), smtp.office365.com (Outlook), email-smtp.us-east-1.amazonaws.com (AWS SES)"
                        />
                        <input
                          type="text"
                          value={form.smtp_host}
                          onChange={handleInput('smtp_host')}
                          placeholder="smtp.gmail.com"
                          className={inputClassName}
                        />
                      </div>
                      <div>
                        <LabelWithHelp
                          label="SMTP Port"
                          help="The port your SMTP server uses. 587 (TLS) is most common and recommended. Some providers use 465 (SSL) or 25 (unencrypted, not recommended)."
                        />
                        <input
                          type="number"
                          value={form.smtp_port}
                          onChange={handleInput('smtp_port')}
                          placeholder="587"
                          className={inputClassName}
                        />
                      </div>
                    </div>

                    <div className="grid gap-4 md:grid-cols-2">
                      <div>
                        <LabelWithHelp
                          label="SMTP Username"
                          help="Usually your full email address (e.g., you@gmail.com). For AWS SES, this is your IAM access key ID."
                        />
                        <input
                          type="text"
                          value={form.smtp_username}
                          onChange={handleInput('smtp_username')}
                          placeholder="you@gmail.com"
                          className={inputClassName}
                        />
                      </div>
                      <div>
                        <LabelWithHelp
                          label="SMTP Password / App Password"
                          help={`For Gmail: You must use an "App Password" (not your regular password). Go to Google Account → Security → 2-Step Verification → App passwords → Generate a new one for "Mail". For Outlook: Use your regular password or an app password if 2FA is enabled.`}
                        />
                        <PasswordInput
                          value={form.smtp_password}
                          onChange={handleInput('smtp_password')}
                          placeholder="Your app password"
                        />
                      </div>
                    </div>

                    <div>
                      <LabelWithHelp
                        label="From Address (optional)"
                        help="The email address that appears in the 'From' field. If left empty, uses the SMTP username. Some providers require this to match a verified sender address."
                      />
                      <input
                        type="email"
                        value={form.smtp_from}
                        onChange={handleInput('smtp_from')}
                        placeholder="noreply@yourcompany.com (optional)"
                        className={inputClassName}
                      />
                    </div>

                    {/* Gmail Quick Setup Guide */}
                    <details className="group">
                      <summary className="cursor-pointer text-xs text-blue-400 hover:text-blue-300">
                        Quick setup guide for Gmail
                      </summary>
                      <div className="mt-2 rounded-lg bg-slate-900/50 p-3 text-xs text-slate-300 space-y-2">
                        <p><strong>1.</strong> Enable 2-Step Verification on your Google Account (required for app passwords)</p>
                        <p><strong>2.</strong> Go to <a href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer" className="text-blue-400 hover:underline">myaccount.google.com/apppasswords</a></p>
                        <p><strong>3.</strong> Select "Mail" and your device, then click "Generate"</p>
                        <p><strong>4.</strong> Copy the 16-character password (ignore spaces)</p>
                        <p><strong>5.</strong> Use these settings:</p>
                        <ul className="ml-4 list-disc">
                          <li>Host: <code className="bg-slate-800 px-1 rounded">smtp.gmail.com</code></li>
                          <li>Port: <code className="bg-slate-800 px-1 rounded">587</code></li>
                          <li>Username: <code className="bg-slate-800 px-1 rounded">your-email@gmail.com</code></li>
                          <li>Password: <code className="bg-slate-800 px-1 rounded">your-16-char-app-password</code></li>
                        </ul>
                      </div>
                    </details>
                  </div>
                )}
              </div>
            </FormSection>

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
