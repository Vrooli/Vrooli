import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AdminLayout } from "../components/AdminLayout";
import { Button } from "../../../shared/ui/button";
import { Activity, AlertTriangle, BarChart3, Compass, CreditCard, ExternalLink, Gauge, History, Palette, RefreshCw, ShieldCheck } from "lucide-react";
import { getAdminExperienceSnapshot, type AdminExperienceSnapshot } from "../../../shared/lib/adminExperience";
import { listVariants, type AnalyticsSummary, type Variant, type VariantStats, getStripeSettings, type StripeSettingsResponse } from "../../../shared/api";
import { buildDateRange, fetchAnalyticsSummary } from "../controllers/analyticsController";
import { useLandingVariant, type VariantResolution } from "../../../app/providers/LandingVariantProvider";

const HEALTH_SNAPSHOT_DAYS = 7;
const STALE_VARIANT_DAYS = 10;
const DAY_MS = 24 * 60 * 60 * 1000;

type WeightStatus = "empty" | "balanced" | "under" | "over";

interface VariantAttention {
  slug: string;
  name: string;
  reasons: string[];
  conversionRate?: number;
  daysSinceUpdate?: number | null;
  updatedLabel: string;
  sectionId?: number;
  sectionType?: string;
}

interface HealthSnapshot {
  activeCount: number;
  archivedCount: number;
  attentionCount: number;
  totalWeight: number;
  weightStatus: WeightStatus;
  highlightedAttention?: VariantAttention;
}

const RESOLUTION_LABELS: Record<VariantResolution, string> = {
  url_param: "URL parameter",
  local_storage: "Stored visitor assignment",
  api_select: "Weighted API",
  fallback: "Fallback payload",
  unknown: "Unknown source",
};

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
  const [experience, setExperience] = useState<AdminExperienceSnapshot | null>(null);
  const { variant: liveVariant, resolution: liveResolution, statusNote: liveStatusNote } = useLandingVariant();
  const [healthSnapshot, setHealthSnapshot] = useState<HealthSnapshot | null>(null);
  const [healthLoading, setHealthLoading] = useState(true);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [healthMetricsDegraded, setHealthMetricsDegraded] = useState(false);
  const [stripeSettings, setStripeSettings] = useState<StripeSettingsResponse | null>(null);
  const [stripeLoading, setStripeLoading] = useState(true);
  const [stripeError, setStripeError] = useState<string | null>(null);

  useEffect(() => {
    setExperience(getAdminExperienceSnapshot());
  }, []);

  const refreshHealthSnapshot = useCallback(async () => {
    setHealthLoading(true);
    setHealthError(null);
    setHealthMetricsDegraded(false);
    try {
      const range = buildDateRange(HEALTH_SNAPSHOT_DAYS);
      const [variantPayload, analyticsPayload] = await Promise.all([
        listVariants(),
        fetchAnalyticsSummary(range)
          .then((data) => ({ ok: true as const, data }))
          .catch((error) => ({ ok: false as const, error })),
      ]);

      if (!analyticsPayload.ok) {
        console.warn("Admin health analytics unavailable:", analyticsPayload.error);
        setHealthMetricsDegraded(true);
      }

      setHealthSnapshot(
        buildHealthSnapshot(
          variantPayload.variants,
          analyticsPayload.ok ? analyticsPayload.data : null
        )
      );
    } catch (error) {
      setHealthError(error instanceof Error ? error.message : "Failed to load admin health snapshot");
      setHealthSnapshot(null);
    } finally {
      setHealthLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshHealthSnapshot();
  }, [refreshHealthSnapshot]);

  const refreshStripeStatus = useCallback(async () => {
    setStripeLoading(true);
    setStripeError(null);
    try {
      const data = await getStripeSettings();
      setStripeSettings(data);
    } catch (error) {
      setStripeSettings(null);
      setStripeError(error instanceof Error ? error.message : "Failed to load monetization status");
    } finally {
      setStripeLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshStripeStatus();
  }, [refreshStripeStatus]);

  const resumeVariant = experience?.lastVariant;
  const resumeAnalytics = experience?.lastAnalytics;

  const handleResumeVariant = () => {
    if (!resumeVariant) return;
    const basePath =
      resumeVariant.surface === "section" && resumeVariant.sectionId
        ? `/admin/customization/variants/${resumeVariant.slug}/sections/${resumeVariant.sectionId}`
        : `/admin/customization/variants/${resumeVariant.slug}`;
    navigate(basePath);
  };

  const handleResumeAnalytics = () => {
    if (!resumeAnalytics) return;
    const params = new URLSearchParams();
    if (resumeAnalytics.variantSlug) {
      params.set("variant", resumeAnalytics.variantSlug);
    }
    if (resumeAnalytics.timeRangeDays && resumeAnalytics.timeRangeDays !== 7) {
      params.set("range", String(resumeAnalytics.timeRangeDays));
    }
    const query = params.toString() ? `?${params.toString()}` : "";
    navigate(`/admin/analytics${query}`);
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

  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-semibold mb-4">Landing Manager Admin</h1>

        <ExperienceGuidePanel
          onNavigateAnalytics={() => navigate('/admin/analytics')}
          onNavigateCustomization={() => navigate('/admin/customization')}
          onNavigateBilling={handleNavigateBilling}
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
  onPreviewPublicLanding: () => void;
}

function ExperienceGuidePanel({ onNavigateAnalytics, onNavigateCustomization, onNavigateBilling, onPreviewPublicLanding }: ExperienceGuidePanelProps) {
  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6" data-testid="admin-experience-guide">
      <div className="flex flex-col gap-2 mb-6">
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Purpose</p>
        <h2 className="text-2xl font-semibold text-white">Measure conversion health and ship experiments without leaving this portal.</h2>
        <p className="text-slate-400 text-sm">
          Analytics tells you what happened. Customization lets you respond. Use the quick flows below to jump to the right surface for your job.
        </p>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
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
        <div className="mt-6 grid gap-4 sm:grid-cols-3">
          {[
            { label: "Publishable key", ok: Boolean(settings?.publishable_key_set) },
            { label: "Secret key", ok: Boolean(settings?.secret_key_set) },
            { label: "Webhook secret", ok: Boolean(settings?.webhook_secret_set) },
          ].map((badge) => (
            <div
              key={badge.label}
              className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${badge.ok ? "border-emerald-500/30 bg-emerald-500/10" : "border-amber-500/30 bg-amber-500/10"}`}
            >
              <ShieldCheck className={`h-5 w-5 ${badge.ok ? "text-emerald-300" : "text-amber-300"}`} />
              <div>
                <p className="text-sm font-semibold text-white">{badge.label}</p>
                <p className="text-xs text-slate-400">{badge.ok ? "Configured" : "Missing"}</p>
              </div>
            </div>
          ))}
        </div>
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
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Needs attention</p>
          {snapshot.highlightedAttention ? (
            <>
              <div>
                <p className="text-lg font-semibold text-white">{snapshot.highlightedAttention.name}</p>
                <p className="text-xs text-slate-500">{snapshot.highlightedAttention.updatedLabel}</p>
              </div>
              <div className="flex flex-wrap gap-2">
                {snapshot.highlightedAttention.reasons.map((reason) => (
                  <span
                    key={`${snapshot.highlightedAttention.slug}-${reason}`}
                    className="rounded-full border border-amber-500/30 bg-amber-500/10 px-3 py-1 text-xs text-amber-200"
                  >
                    {reason}
                  </span>
                ))}
              </div>
              {typeof snapshot.highlightedAttention.conversionRate === "number" && (
                <p className="text-sm text-slate-300">
                  Conversion rate:&nbsp;
                  <span className="font-semibold text-rose-300">
                    {snapshot.highlightedAttention.conversionRate.toFixed(2)}%
                  </span>
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  onClick={() =>
                    onHighlightVariant(snapshot.highlightedAttention!.slug, {
                      sectionId: snapshot.highlightedAttention?.sectionId,
                      sectionType: snapshot.highlightedAttention?.sectionType,
                    })
                  }
                  data-testid="admin-health-review"
                >
                  Review in customization
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => onInspectVariantAnalytics(snapshot.highlightedAttention!.slug)}
                  data-testid="admin-health-attention-analytics"
                >
                  View analytics
                </Button>
              </div>
            </>
          ) : (
            <p className="text-sm text-slate-400">
              No variants are flagged right now. Keep routing traffic to surface the next opportunity.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

function buildHealthSnapshot(
  variants: Variant[],
  analytics: AnalyticsSummary | null,
): HealthSnapshot {
  const activeVariants = variants.filter((variant) => variant.status !== "archived");
  const archivedVariants = variants.filter((variant) => variant.status === "archived");
  const totalWeight = activeVariants.reduce((sum, variant) => sum + (variant.weight ?? 0), 0);
  let weightStatus: WeightStatus = "empty";
  if (activeVariants.length > 0) {
    if (Math.round(totalWeight) === 100) {
      weightStatus = "balanced";
    } else if (totalWeight > 100) {
      weightStatus = "over";
    } else {
      weightStatus = "under";
    }
  }

  const statsBySlug = new Map<string, VariantStats>();
  analytics?.variant_stats?.forEach((stat) => statsBySlug.set(stat.variant_slug, stat));
  const attentionEntries = new Map<string, VariantAttention>();
  const registerAttention = (variant: Variant, reason: string, extras?: Partial<VariantAttention>) => {
    const existing = attentionEntries.get(variant.slug);
    if (existing) {
      const next: VariantAttention = {
        ...existing,
        ...extras,
        reasons: existing.reasons.includes(reason) ? existing.reasons : [...existing.reasons, reason],
      };
      attentionEntries.set(variant.slug, next);
      return;
    }

    attentionEntries.set(variant.slug, {
      slug: variant.slug,
      name: variant.name ?? variant.slug,
      reasons: [reason],
      conversionRate: extras?.conversionRate,
      daysSinceUpdate: calculateDaysSinceUpdate(variant.updated_at),
      updatedLabel: formatVariantUpdatedLabel(variant.updated_at),
      sectionId: extras?.sectionId,
      sectionType: extras?.sectionType ?? "hero",
    });
  };

  activeVariants.forEach((variant) => {
    const daysSinceUpdate = calculateDaysSinceUpdate(variant.updated_at);
    if (daysSinceUpdate === null) {
      registerAttention(variant, "Never customized");
    } else if (daysSinceUpdate >= STALE_VARIANT_DAYS) {
      registerAttention(variant, `Stale · ${daysSinceUpdate}d`, { daysSinceUpdate });
    }
  });

  if (analytics?.variant_stats?.length) {
    const activeSlugs = new Set(activeVariants.map((variant) => variant.slug));
    const relevantStats = analytics.variant_stats.filter((stat) => activeSlugs.has(stat.variant_slug));
    const underperforming = relevantStats.reduce<VariantStats | null>((worst, stat) => {
      if (!worst) {
        return stat;
      }
      return stat.conversion_rate < worst.conversion_rate ? stat : worst;
    }, null);

    if (underperforming) {
      const matchingVariant = activeVariants.find((variant) => variant.slug === underperforming.variant_slug);
      if (matchingVariant) {
        registerAttention(matchingVariant, "Lowest conversion", {
          conversionRate: underperforming.conversion_rate,
        });
      }
    }
  }

  const attentionList = Array.from(attentionEntries.values());
  let highlightedAttention: VariantAttention | undefined;
  let highestPriority = -1;
  attentionList.forEach((entry) => {
    const priority = getAttentionPriority(entry);
    if (
      !highlightedAttention ||
      priority > highestPriority ||
      (priority === highestPriority &&
        (entry.daysSinceUpdate ?? -1) > (highlightedAttention.daysSinceUpdate ?? -1))
    ) {
      highestPriority = priority;
      highlightedAttention = entry;
    }
  });

  return {
    activeCount: activeVariants.length,
    archivedCount: archivedVariants.length,
    attentionCount: attentionList.length,
    totalWeight,
    weightStatus,
    highlightedAttention,
  };
}

function describeWeightStatus(status: WeightStatus, totalWeight: number, activeCount: number) {
  if (activeCount === 0) {
    return "No active variants are routing traffic. Create one to render the public landing.";
  }
  if (status === "balanced") {
    return "Traffic is fully allocated across variants.";
  }
  if (status === "under") {
    return `${Math.max(0, 100 - Math.round(totalWeight))}% of visitors are idle because weights total less than 100%.`;
  }
  if (status === "over") {
    return `Weights exceed 100% by ${Math.round(totalWeight - 100)}%. Adjust them to match your intent.`;
  }
  return "Assign weights to control where visitors land.";
}

function formatVariantUpdatedLabel(updatedAt?: string | null) {
  if (!updatedAt) {
    return "Never customized";
  }
  const parsed = new Date(updatedAt);
  if (Number.isNaN(parsed.getTime())) {
    return "Last updated date unavailable";
  }
  const diffMs = Date.now() - parsed.getTime();
  const diffDays = Math.max(0, Math.floor(diffMs / DAY_MS));
  if (diffDays === 0) {
    return "Updated today";
  }
  if (diffDays === 1) {
    return "Updated yesterday";
  }
  return `Updated ${diffDays} days ago`;
}

function calculateDaysSinceUpdate(updatedAt?: string | null) {
  if (!updatedAt) {
    return null;
  }
  const parsed = new Date(updatedAt);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  return Math.max(0, Math.floor((Date.now() - parsed.getTime()) / DAY_MS));
}

function getAttentionPriority(entry: VariantAttention) {
  if (entry.reasons.some((reason) => reason.toLowerCase().includes("lowest conversion"))) {
    return 3;
  }
  if (entry.reasons.some((reason) => reason.startsWith("Stale"))) {
    return 2;
  }
  if (entry.reasons.some((reason) => reason.startsWith("Never"))) {
    return 1;
  }
  return 0;
}
