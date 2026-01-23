import { useNavigate } from "react-router-dom";
import { AdminLayout } from "../components/AdminLayout";
import { PageHeader } from "../components/PageHeader";
import { StatusBadgeGrid } from "../components/StatusBadge";
import { Button } from "../../../shared/ui/button";
import { Activity, AlertTriangle, BarChart3, Compass, CreditCard, Download, ExternalLink, Gauge, History, Home, Package, Palette, RefreshCw, Settings2, Type } from "lucide-react";
import { LAYOUT } from "../config/layout.constants";
import type { Variant, StripeSettingsResponse } from "../../../shared/api";
import { useLandingVariant, type VariantResolution } from "../../../app/providers/LandingVariantProvider";
import {
  HEALTH_SNAPSHOT_DAYS,
  describeWeightStatus,
  type HealthSnapshot,
  type BrandingHealthStatus,
  type DownloadsHealthStatus,
} from "../services/adminHome.service";
import { useAdminHome } from "../hooks/useAdminHome";
import { RESOLUTION_LABELS } from "../config/variant.constants";

/**
 * Admin home page - implements ADMIN-MODES requirement (OT-P0-009)
 *
 * Shows exactly two modes:
 * 1. Analytics / Metrics
 * 2. Customization
 *
 * Navigation efficiency: ≤ 3 clicks to any customization card (OT-P0-010)
 *
 * [REQ:ADMIN-MODES] [REQ:ADMIN-NAV]
 */
export function AdminHome() {
  const navigate = useNavigate();
  const { variant: liveVariant, resolution: liveResolution, statusNote: liveStatusNote } = useLandingVariant();

  const {
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
    downloadsHealth,
    downloadsLoading,
    resettingDemoData,
    resetMessage,
    resetError,
    showResetConfirm,
    setShowResetConfirm,
    handleResetDemoData,
    buildResumeVariantPath,
    buildResumeAnalyticsPath,
  } = useAdminHome();

  const resumeVariant = experience?.lastVariant;
  const resumeAnalytics = experience?.lastAnalytics;

  const handleResumeVariant = () => {
    const path = buildResumeVariantPath();
    if (path) navigate(path);
  };

  const handleResumeAnalytics = () => {
    const path = buildResumeAnalyticsPath();
    if (path) navigate(path);
  };

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  interface HighlightOptions {
    sectionId?: number;
    sectionType?: string;
  }

  const handleReviewVariant = (slug: string, options?: HighlightOptions) => {
    const params = new URLSearchParams({ focus: slug });
    if (options?.sectionId) {
      params.set("focusSectionId", String(options.sectionId));
    }
    if (options?.sectionType) {
      params.set("focusSectionType", options.sectionType);
    }
    navigate(`/admin/customization?${params.toString()}`);
  };

  const handleInspectVariantAnalytics = (slug: string) => {
    navigate(`/admin/analytics?variant=${slug}`);
  };
  const handleNavigateBilling = () => {
    navigate("/admin/billing");
  };
  const handleNavigateDownloads = () => {
    navigate("/admin/downloads");
  };

  return (
    <AdminLayout>
      <div className={`${LAYOUT.maxWidth.wide} mx-auto ${LAYOUT.sectionSpacing}`}>
        <PageHeader
          variant="icon-title"
          title="Landing Manager Admin"
          icon={Home}
          iconBgClass="bg-slate-500/10"
          iconColorClass="text-slate-400"
          testId="admin-home-header"
        />

        <ExperienceGuidePanel
          onNavigateAnalytics={() => navigate('/admin/analytics')}
          onNavigateCustomization={() => navigate('/admin/customization')}
          onNavigateBilling={handleNavigateBilling}
          onNavigateDownloads={handleNavigateDownloads}
          onPreviewPublicLanding={previewPublicLanding}
        />

        <AdminHealthDigest
          loading={healthLoading}
          error={healthError}
          snapshot={healthSnapshot}
          analyticsDegraded={healthMetricsDegraded}
          onRetry={refreshHealthSnapshot}
          onNavigateCustomization={() => navigate('/admin/customization')}
          onNavigateAnalytics={() => navigate('/admin/analytics')}
          onInspectVariantAnalytics={handleInspectVariantAnalytics}
          onHighlightVariant={handleReviewVariant}
          liveVariant={liveVariant}
          liveResolution={liveResolution}
          statusNote={liveStatusNote}
          previewPublicLanding={previewPublicLanding}
        />

        <MonetizationStatusCard
          loading={stripeLoading}
          error={stripeError}
          settings={stripeSettings}
          onRetry={refreshStripeStatus}
          onNavigateBilling={handleNavigateBilling}
        />

        <SettingsStatusCard
          brandingHealth={brandingHealth}
          brandingLoading={brandingLoading}
          downloadsHealth={downloadsHealth}
          downloadsLoading={downloadsLoading}
          onNavigateBranding={() => navigate('/admin/branding')}
          onNavigateDownloads={handleNavigateDownloads}
        />

        <div className="mt-6 rounded-2xl border border-rose-500/30 bg-rose-500/5 p-6" data-testid="admin-reset-demo-card">
          <div className="flex items-center gap-3">
            <AlertTriangle className={`h-5 w-5 text-rose-400`} />
            <div>
              <h3 className="text-lg font-semibold text-white">Reset to template defaults</h3>
              <p className="text-sm text-rose-100/80">Re-seed all landing page content from the fallback template. Use this to apply updated template content or restore defaults.</p>
            </div>
          </div>
          {resetError && (
            <p className="mt-3 text-sm text-rose-200" data-testid="admin-reset-error">{resetError}</p>
          )}
          {resetMessage && (
            <p className="mt-3 text-sm text-emerald-200" data-testid="admin-reset-success">{resetMessage}</p>
          )}

          {!showResetConfirm ? (
            <div className="mt-4">
              <Button
                variant="outline"
                className="gap-2 border-rose-500/50 text-rose-200 hover:bg-rose-500/10"
                onClick={() => setShowResetConfirm(true)}
                disabled={resettingDemoData}
                data-testid="admin-reset-demo-btn"
              >
                <RefreshCw className={`h-4 w-4 ${resettingDemoData ? 'animate-spin' : ''}`} />
                {resettingDemoData ? 'Resetting…' : 'Reset demo data'}
              </Button>
            </div>
          ) : (
            <div className="mt-4 rounded-xl border border-rose-500/50 bg-rose-500/10 p-4 space-y-4" data-testid="admin-reset-confirm-dialog">
              <div className="flex items-start gap-3">
                <AlertTriangle className="h-5 w-5 text-rose-400 mt-0.5 flex-shrink-0" />
                <div className="space-y-2">
                  <p className="font-semibold text-rose-100">Are you sure you want to reset?</p>
                  <p className="text-sm text-rose-200/80">This action will permanently delete:</p>
                  <ul className="text-sm text-rose-200/80 list-disc list-inside space-y-1">
                    <li>All variant customizations</li>
                    <li>All section content edits</li>
                    <li>Download app configurations</li>
                    <li>Bundle pricing settings</li>
                  </ul>
                  <p className="text-sm text-rose-200/80 font-medium">
                    Data will be replaced with the current template fallback. This cannot be undone.
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap gap-3">
                <Button
                  variant="outline"
                  className="gap-2"
                  onClick={() => setShowResetConfirm(false)}
                  disabled={resettingDemoData}
                >
                  Cancel
                </Button>
                <Button
                  className="gap-2 bg-rose-600 hover:bg-rose-700 text-white"
                  onClick={handleResetDemoData}
                  disabled={resettingDemoData}
                  data-testid="admin-reset-confirm-btn"
                >
                  <RefreshCw className={`h-4 w-4 ${resettingDemoData ? 'animate-spin' : ''}`} />
                  {resettingDemoData ? 'Resetting…' : 'Yes, reset everything'}
                </Button>
              </div>
            </div>
          )}
        </div>

        <div className="grid md:grid-cols-2 gap-6">
          {/* Analytics / Metrics Mode */}
          <button
            type="button"
            onClick={() => navigate("/admin/analytics")}
            className="group relative rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur hover:bg-white/10 transition-all text-left"
            data-testid="admin-mode-analytics"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 rounded-xl bg-blue-500/20">
                <BarChart3 className="h-8 w-8 text-blue-400" />
              </div>
              <h2 className="text-2xl font-semibold">Analytics / Metrics</h2>
            </div>
            <p className="text-slate-300 mb-4">
              View conversion rates, A/B test results, visitor metrics, and performance data across all variants.
            </p>
            <span className="mt-4 inline-flex items-center gap-2 text-sm font-semibold tracking-wide text-blue-200 group-hover:translate-x-1 transition-transform">
              View Analytics →
            </span>
          </button>

          {/* Customization Mode */}
          <button
            type="button"
            onClick={() => navigate("/admin/customization")}
            className="group relative rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur hover:bg-white/10 transition-all text-left"
            data-testid="admin-mode-customization"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 rounded-xl bg-purple-500/20">
                <Palette className="h-8 w-8 text-purple-400" />
              </div>
              <h2 className="text-2xl font-semibold">Customization</h2>
            </div>
            <p className="text-slate-300 mb-4">
              Customize landing page content, trigger agent-based improvements, manage A/B test variants, and configure site settings.
            </p>
            <span className="mt-4 inline-flex items-center gap-2 text-sm font-semibold tracking-wide text-purple-200 group-hover:translate-x-1 transition-transform">
              Customize Site →
            </span>
          </button>
        </div>

        {(resumeVariant || resumeAnalytics) && (
          <div className="mt-10 space-y-4" data-testid="admin-resume-panel">
            <p className="text-sm uppercase tracking-[0.3em] text-slate-500">Continue where you left off</p>
            <div className="grid gap-4 md:grid-cols-2">
              {resumeVariant && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5" data-testid="admin-resume-card">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-purple-500/15">
                      <History className="h-5 w-5 text-purple-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Last customization</p>
                      <p className="text-lg font-semibold">
                        {resumeVariant.name ?? resumeVariant.slug}
                        {resumeVariant.surface === "section" && resumeVariant.sectionType && (
                          <span className="text-sm font-normal text-slate-400"> · {resumeVariant.sectionType}</span>
                        )}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm text-slate-400 mb-4">
                    {resumeVariant.surface === "section" ? "Resume editing the section preview you left open." : "Jump back into the variant settings and section list."}
                  </p>
                  <Button onClick={handleResumeVariant} className="w-full gap-2" data-testid="admin-resume-customization">
                    Return to {resumeVariant.surface === "section" ? "Section" : "Variant"}
                  </Button>
                </div>
              )}

              {resumeAnalytics && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5" data-testid="admin-resume-analytics-card">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-blue-500/15">
                      <Activity className="h-5 w-5 text-blue-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Last analytics view</p>
                      <p className="text-lg font-semibold">
                        {resumeAnalytics.variantName ?? (resumeAnalytics.variantSlug ?? "All variants")}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm text-slate-400 mb-4">
                    Showing last {resumeAnalytics.timeRangeDays} day{resumeAnalytics.timeRangeDays === 1 ? "" : "s"} window.
                  </p>
                  <Button onClick={handleResumeAnalytics} variant="outline" className="w-full gap-2" data-testid="admin-resume-analytics">
                    Reopen Analytics
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}

interface ExperienceGuidePanelProps {
  onNavigateAnalytics: () => void;
  onNavigateCustomization: () => void;
  onNavigateBilling: () => void;
  onNavigateDownloads: () => void;
  onPreviewPublicLanding: () => void;
}

function ExperienceGuidePanel({ onNavigateAnalytics, onNavigateCustomization, onNavigateBilling, onNavigateDownloads, onPreviewPublicLanding }: ExperienceGuidePanelProps) {
  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6" data-testid="admin-experience-guide">
      <div className="flex flex-col gap-2 mb-6">
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Purpose</p>
        <h2 className="text-2xl font-semibold text-white">Measure conversion health and ship experiments without leaving this portal.</h2>
        <p className="text-slate-400 text-sm">
          Analytics tells you what happened. Customization lets you respond. Use the quick flows below to jump to the right surface for your job.
        </p>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-5">
        <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-blue-500/20">
              <Compass className="h-4 w-4 text-blue-300" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-500">Flow 1</p>
              <p className="font-semibold text-white">Audit performance</p>
            </div>
          </div>
          <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
            <li>Open Analytics filters</li>
            <li>Inspect winning + weak variants</li>
            <li>Jump into edit or preview</li>
          </ol>
          <Button size="sm" variant="outline" className="w-full" onClick={onNavigateAnalytics} data-testid="admin-guide-analytics">
            View Analytics
          </Button>
        </div>
        <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-purple-500/20">
              <Palette className="h-4 w-4 text-purple-300" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-500">Flow 2</p>
              <p className="font-semibold text-white">Ship a variant</p>
            </div>
          </div>
          <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
            <li>Pick a variant card</li>
            <li>Edit sections + weights</li>
            <li>Preview + publish</li>
          </ol>
          <Button size="sm" variant="outline" className="w-full" onClick={onNavigateCustomization} data-testid="admin-guide-customization">
            Customize Site
          </Button>
        </div>
        <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-emerald-500/20">
              <ExternalLink className="h-4 w-4 text-emerald-300" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-500">Flow 3</p>
              <p className="font-semibold text-white">Verify public view</p>
            </div>
          </div>
          <p className="text-sm text-slate-400">
            Opens the live landing in a new tab so you can sanity-check animations, CTAs, and download buttons after edits.
          </p>
          <Button
            size="sm"
            variant="outline"
            className="w-full"
            onClick={onPreviewPublicLanding}
            data-testid="admin-guide-preview"
          >
            Preview landing
          </Button>
        </div>
        <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-amber-500/20">
              <CreditCard className="h-4 w-4 text-amber-300" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-500">Flow 4</p>
              <p className="font-semibold text-white">Protect monetization</p>
            </div>
          </div>
          <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
            <li>Check Stripe key badges</li>
            <li>Confirm visible plans</li>
            <li>Sync entitlements</li>
          </ol>
          <Button size="sm" variant="outline" className="w-full" onClick={onNavigateBilling} data-testid="admin-guide-billing">
            Open Billing
          </Button>
        </div>
        <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-emerald-500/20">
              <Download className="h-4 w-4 text-emerald-300" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-slate-500">Flow 5</p>
              <p className="font-semibold text-white">Maintain downloads</p>
            </div>
          </div>
          <ol className="text-sm text-slate-400 space-y-1 list-decimal list-inside">
            <li>Update copy + install steps</li>
            <li>Paste store links + badges</li>
            <li>Verify installers per platform</li>
          </ol>
          <Button size="sm" variant="outline" className="w-full" onClick={onNavigateDownloads} data-testid="admin-guide-downloads">
            Configure downloads
          </Button>
        </div>
      </div>
    </div>
  );
}

interface AdminHealthDigestProps {
  loading: boolean;
  error: string | null;
  snapshot: HealthSnapshot | null;
  analyticsDegraded: boolean;
  onRetry: () => void;
  onNavigateCustomization: () => void;
  onNavigateAnalytics: () => void;
  onInspectVariantAnalytics: (slug: string) => void;
  onHighlightVariant: (slug: string, options?: { sectionId?: number; sectionType?: string }) => void;
  liveVariant: Variant | null;
  liveResolution: VariantResolution;
  statusNote: string | null;
  previewPublicLanding: () => void;
}

interface MonetizationStatusCardProps {
  loading: boolean;
  error: string | null;
  settings: StripeSettingsResponse | null;
  onRetry: () => void;
  onNavigateBilling: () => void;
}

function MonetizationStatusCard({ loading, error, settings, onRetry, onNavigateBilling }: MonetizationStatusCardProps) {
  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6" data-testid="admin-monetization-card">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Monetization guardrail</p>
          <h2 className="text-2xl font-semibold text-white mt-1">Stripe keys and plan signals stay visible from here.</h2>
          <p className="text-sm text-slate-400">
            Keep billing healthy before routing campaigns or enabling gated downloads.
          </p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={onRetry}>
            Refresh
          </Button>
          <Button size="sm" onClick={onNavigateBilling}>
            Go to billing
          </Button>
        </div>
      </div>
      {loading ? (
        <div className="mt-6 grid gap-3 sm:grid-cols-3">
          {[0, 1, 2].map((entry) => (
            <div key={entry} className="h-16 rounded-xl bg-white/5 animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <div className="mt-6 rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center justify-between gap-3">
          <div>{error}</div>
          <Button size="sm" variant="ghost" onClick={onRetry}>
            Retry
          </Button>
        </div>
      ) : (
        <StatusBadgeGrid
          className="mt-6"
          columns={3}
          badges={[
            {
              label: "Publishable key",
              status: settings?.publishable_key_set ? "success" : "warning",
              description: settings?.publishable_key_set ? "Configured" : "Missing",
            },
            {
              label: "Secret key",
              status: settings?.secret_key_set ? "success" : "warning",
              description: settings?.secret_key_set ? "Configured" : "Missing",
            },
            {
              label: "Webhook secret",
              status: settings?.webhook_secret_set ? "success" : "warning",
              description: settings?.webhook_secret_set ? "Configured" : "Missing",
            },
          ]}
        />
      )}
      {!loading && !error && (
        <div className="mt-4 text-xs text-slate-500 flex flex-wrap gap-4">
          {settings?.source && <span>Source: {settings.source === "database" ? "Admin override" : "Environment variables"}</span>}
          {settings?.updated_at && (
            <span>Last updated: {new Date(settings.updated_at).toLocaleString()}</span>
          )}
        </div>
      )}
    </div>
  );
}

interface SettingsStatusCardProps {
  brandingHealth: BrandingHealthStatus | null;
  brandingLoading: boolean;
  downloadsHealth: DownloadsHealthStatus | null;
  downloadsLoading: boolean;
  onNavigateBranding: () => void;
  onNavigateDownloads: () => void;
}

function SettingsStatusCard({
  brandingHealth,
  brandingLoading,
  downloadsHealth,
  downloadsLoading,
  onNavigateBranding,
  onNavigateDownloads,
}: SettingsStatusCardProps) {
  const isLoading = brandingLoading || downloadsLoading;

  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6" data-testid="admin-settings-status-card">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Site configuration</p>
          <h2 className="text-2xl font-semibold text-white mt-1">Branding and downloads setup at a glance.</h2>
          <p className="text-sm text-slate-400">
            Configure your site identity, SEO defaults, and downloadable apps without leaving home.
          </p>
        </div>
      </div>

      {isLoading ? (
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          {[0, 1].map((i) => (
            <div key={i} className="h-24 rounded-xl bg-white/5 animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          {/* Branding Status */}
          <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-purple-500/20 p-2">
                  <Settings2 className="h-5 w-5 text-purple-300" />
                </div>
                <div>
                  <p className="font-semibold text-white">Branding</p>
                  <p className="text-xs text-slate-400">
                    {brandingHealth
                      ? `${brandingHealth.configuredCount}/${brandingHealth.totalChecks} configured`
                      : "Unable to load"}
                  </p>
                </div>
              </div>
              {brandingHealth && (
                <div className={`rounded-full px-2 py-1 text-xs ${
                  brandingHealth.configuredCount === brandingHealth.totalChecks
                    ? "bg-emerald-500/20 text-emerald-300"
                    : "bg-amber-500/20 text-amber-300"
                }`}>
                  {brandingHealth.configuredCount === brandingHealth.totalChecks ? "Complete" : "Incomplete"}
                </div>
              )}
            </div>
            {brandingHealth && (
              <div className="flex flex-wrap gap-2 text-xs">
                <span className={brandingHealth.hasIdentity ? "text-emerald-300" : "text-amber-300"}>
                  {brandingHealth.hasIdentity ? "✓" : "○"} Identity
                </span>
                <span className={brandingHealth.hasFavicon ? "text-emerald-300" : "text-amber-300"}>
                  {brandingHealth.hasFavicon ? "✓" : "○"} Favicon
                </span>
                <span className={brandingHealth.hasSeo ? "text-emerald-300" : "text-amber-300"}>
                  {brandingHealth.hasSeo ? "✓" : "○"} SEO
                </span>
                <span className={brandingHealth.hasOgImage ? "text-emerald-300" : "text-amber-300"}>
                  {brandingHealth.hasOgImage ? "✓" : "○"} OG Image
                </span>
              </div>
            )}
            <Button size="sm" variant="outline" onClick={onNavigateBranding} className="w-full gap-2">
              <Type className="h-4 w-4" />
              Configure branding
            </Button>
          </div>

          {/* Downloads Status */}
          <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-blue-500/20 p-2">
                  <Package className="h-5 w-5 text-blue-300" />
                </div>
                <div>
                  <p className="font-semibold text-white">Downloads</p>
                  <p className="text-xs text-slate-400">
                    {downloadsHealth
                      ? `${downloadsHealth.appCount} app${downloadsHealth.appCount !== 1 ? "s" : ""} configured`
                      : "Unable to load"}
                  </p>
                </div>
              </div>
              {downloadsHealth && (
                <div className={`rounded-full px-2 py-1 text-xs ${
                  downloadsHealth.hasApps
                    ? "bg-emerald-500/20 text-emerald-300"
                    : "bg-slate-500/20 text-slate-300"
                }`}>
                  {downloadsHealth.hasApps ? "Active" : "None"}
                </div>
              )}
            </div>
            {downloadsHealth && downloadsHealth.hasApps && (
              <div className="flex flex-wrap gap-2 text-xs text-slate-300">
                <span>{downloadsHealth.platformsConfigured} platform{downloadsHealth.platformsConfigured !== 1 ? "s" : ""}</span>
                <span>•</span>
                <span>{downloadsHealth.storefrontsConfigured} store link{downloadsHealth.storefrontsConfigured !== 1 ? "s" : ""}</span>
              </div>
            )}
            {downloadsHealth && !downloadsHealth.hasApps && (
              <p className="text-xs text-slate-400">Add your first app to enable downloads on the landing page.</p>
            )}
            <Button size="sm" variant="outline" onClick={onNavigateDownloads} className="w-full gap-2">
              <Download className="h-4 w-4" />
              Configure downloads
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function AdminHealthDigest({
  loading,
  error,
  snapshot,
  analyticsDegraded,
  onRetry,
  onNavigateCustomization,
  onNavigateAnalytics,
  onInspectVariantAnalytics,
  onHighlightVariant,
  liveVariant,
  liveResolution,
  statusNote,
  previewPublicLanding,
}: AdminHealthDigestProps) {
  if (loading) {
    return (
      <div className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6 animate-pulse" data-testid="admin-health-digest-loading">
        <div className="h-5 w-48 rounded-full bg-white/10" />
        <div className="mt-4 h-24 rounded-2xl bg-white/5" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="mb-8 rounded-2xl border border-red-500/30 bg-red-500/10 p-6" data-testid="admin-health-digest-error">
        <div className="flex items-center gap-3">
          <AlertTriangle className="h-5 w-5 text-red-300" />
          <div>
            <p className="text-lg font-semibold text-red-100">Health snapshot unavailable</p>
            <p className="text-sm text-red-200">{error}</p>
          </div>
        </div>
        <Button size="sm" variant="outline" className="mt-4" onClick={onRetry}>
          Retry
        </Button>
      </div>
    );
  }

  if (!snapshot) {
    return null;
  }

  const runtimeLabel = liveVariant?.name ?? liveVariant?.slug ?? "Variant not resolved";
  const resolutionLabel = RESOLUTION_LABELS[liveResolution] ?? RESOLUTION_LABELS.unknown;
  const trafficMessage = describeWeightStatus(snapshot.weightStatus, snapshot.totalWeight, snapshot.activeCount);

  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6" data-testid="admin-health-digest">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Experience health</p>
          <h2 className="text-2xl font-semibold text-white mt-2">Know if the runtime is safe before diving into a flow.</h2>
          <p className="text-slate-400 text-sm">
            Active variants, attention items, and live runtime status update every {HEALTH_SNAPSHOT_DAYS} days of analytics.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" className="gap-2" onClick={previewPublicLanding}>
            Preview landing
          </Button>
          <Button size="sm" variant="outline" className="gap-2" onClick={onRetry} data-testid="admin-health-refresh">
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
        </div>
      </div>

      <div className="grid gap-4 text-center sm:grid-cols-4" data-testid="admin-health-stats">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Active</p>
          <p className="text-3xl font-semibold text-white">{snapshot.activeCount}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Needs attention</p>
          <p className={`text-3xl font-semibold ${snapshot.attentionCount > 0 ? "text-amber-300" : "text-slate-300"}`}>
            {snapshot.attentionCount}
          </p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Archived</p>
          <p className="text-3xl font-semibold text-slate-400">{snapshot.archivedCount}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Traffic assigned</p>
          <p className="text-3xl font-semibold text-white">{Math.max(0, Math.round(snapshot.totalWeight))}%</p>
        </div>
      </div>

      {analyticsDegraded && (
        <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4" />
          Metrics snapshot degraded — analytics data is partially unavailable, but variant status is still accurate.
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-2xl border border-white/10 bg-slate-900/40 p-4 space-y-3">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Live runtime</p>
          <div className="flex items-center gap-3">
            <div className="rounded-xl bg-blue-500/15 p-2">
              <Gauge className="h-5 w-5 text-blue-300" />
            </div>
            <div>
              <p className="text-lg font-semibold text-white">{runtimeLabel}</p>
              <p className="text-xs text-slate-400">Source: {resolutionLabel}</p>
            </div>
          </div>
          {statusNote && <p className="text-xs text-slate-500">{statusNote}</p>}
          <div className="flex flex-wrap gap-2 pt-2">
            <Button size="sm" onClick={onNavigateAnalytics} data-testid="admin-health-live-analytics">
              Open analytics
            </Button>
            <Button size="sm" variant="outline" onClick={onNavigateCustomization}>
              Customize
            </Button>
          </div>
        </div>
        <div className="rounded-2xl border border-white/10 bg-slate-900/40 p-4 space-y-3">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Traffic allocation</p>
          <div className="text-3xl font-semibold text-white">{Math.max(0, Math.round(snapshot.totalWeight))}%</div>
          <div className="h-2 rounded-full bg-white/10 overflow-hidden">
            <div
              className={`h-2 ${snapshot.weightStatus === "balanced" ? "bg-emerald-400" : snapshot.weightStatus === "empty" ? "bg-slate-500" : "bg-amber-400"}`}
              style={{ width: `${Math.max(0, Math.min(100, snapshot.totalWeight))}%` }}
            />
          </div>
          <p className="text-xs text-slate-400">{trafficMessage}</p>
          <Button size="sm" variant="outline" onClick={onNavigateCustomization}>
            Adjust weights
          </Button>
        </div>
        <div className="rounded-2xl border border-white/10 bg-slate-900/40 p-4 space-y-3" data-testid="admin-health-attention-card">
          {(() => {
            const highlightedAttention = snapshot.highlightedAttention;
            if (!highlightedAttention) {
              return (
                <>
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Needs attention</p>
                  <p className="text-sm text-slate-400">
                    No variants are flagged right now. Keep routing traffic to surface the next opportunity.
                  </p>
                </>
              );
            }
            return (
              <>
                <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Needs attention</p>
                <div>
                  <p className="text-lg font-semibold text-white">{highlightedAttention.name}</p>
                  <p className="text-xs text-slate-500">{highlightedAttention.updatedLabel}</p>
                </div>
                <div className="flex flex-wrap gap-2">
                  {highlightedAttention.reasons.map((reason) => (
                    <span
                      key={`${highlightedAttention.slug}-${reason}`}
                      className="rounded-full border border-amber-500/30 bg-amber-500/10 px-3 py-1 text-xs text-amber-200"
                    >
                      {reason}
                    </span>
                  ))}
                </div>
                {typeof highlightedAttention.conversionRate === "number" && (
                  <p className="text-sm text-slate-300">
                    Conversion rate:&nbsp;
                    <span className="font-semibold text-rose-300">
                      {highlightedAttention.conversionRate.toFixed(2)}%
                    </span>
                  </p>
                )}
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    onClick={() =>
                      onHighlightVariant(highlightedAttention.slug, {
                        sectionId: highlightedAttention.sectionId,
                        sectionType: highlightedAttention.sectionType,
                      })
                    }
                    data-testid="admin-health-review"
                  >
                    Review in customization
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => onInspectVariantAnalytics(highlightedAttention.slug)}
                    data-testid="admin-health-attention-analytics"
                  >
                    View analytics
                  </Button>
                </div>
              </>
            );
          })()}
        </div>
      </div>
    </div>
  );
}

