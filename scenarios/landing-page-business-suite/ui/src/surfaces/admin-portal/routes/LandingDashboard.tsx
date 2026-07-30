import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Button } from '../../../shared/ui/button';
import {
  Palette,
  BarChart3,
  Settings2,
  Bot,
  ExternalLink,
  RefreshCw,
  AlertTriangle,
  Gauge,
  Activity,
  History,
} from 'lucide-react';
import { LAYOUT } from '../config/layout.constants';
import { useLandingVariant } from '../../../app/providers/useLandingVariant';
import { useAdminHome } from '../hooks/useAdminHome';
import { RESOLUTION_LABELS } from '../config/variant.constants';
import { describeWeightStatus } from '../services/adminHome.service';
import { StatusBadgeGrid } from '../components/StatusBadge';

/**
 * Landing Dashboard - Entry point for landing page management
 *
 * Provides quick flows, stats, and navigation for:
 * - Customization (variant management, A/B testing)
 * - Analytics (conversion metrics)
 * - Branding (site identity, SEO)
 * - Agent (AI-powered improvements)
 */
export function LandingDashboard() {
  const navigate = useNavigate();
  const {
    variant: liveVariant,
    resolution: liveResolution,
  } = useLandingVariant();

  const {
    experience,
    healthSnapshot,
    healthLoading,
    healthError,
    healthMetricsDegraded,
    refreshHealthSnapshot,
    brandingHealth,
    brandingLoading,
  } = useAdminHome();

  const resumeVariant = experience?.lastVariant;
  const resumeAnalytics = experience?.lastAnalytics;

  const handleResumeVariant = () => {
    if (!resumeVariant) return;
    const path =
      resumeVariant.surface === 'section' && (resumeVariant.sectionKey || resumeVariant.sectionId)
        ? `/admin/customization/variants/${resumeVariant.slug}/sections/${encodeURIComponent(resumeVariant.sectionKey ?? String(resumeVariant.sectionId))}`
        : `/admin/customization/variants/${resumeVariant.slug}`;
    navigate(path);
  };

  const handleResumeAnalytics = () => {
    if (!resumeAnalytics) return;
    const params = new URLSearchParams();
    if (resumeAnalytics.variantSlug) {
      params.set('variant', resumeAnalytics.variantSlug);
    }
    if (resumeAnalytics.timeRangeDays && resumeAnalytics.timeRangeDays !== 7) {
      params.set('range', String(resumeAnalytics.timeRangeDays));
    }
    const query = params.toString() ? `?${params.toString()}` : '';
    navigate(`/admin/analytics${query}`);
  };

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  const runtimeLabel =
    liveVariant?.name ?? liveVariant?.slug ?? 'Variant not resolved';
  const resolutionLabel =
    RESOLUTION_LABELS[liveResolution];

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          title="Landing Dashboard"
          icon={Palette}
          iconBgClass="bg-purple-500/10"
          iconColorClass="text-purple-400"
          testId="landing-dashboard-header"
        />

        {/* Quick Flows */}
        <div className="mb-8" data-testid="landing-quick-flows">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">
            Quick flows
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <QuickFlowCard
              title="Edit variants"
              description="Manage A/B test variants, sections, and weights"
              icon={Palette}
              iconBg="bg-purple-500/20"
              iconColor="text-purple-300"
              onClick={() => { navigate('/admin/customization'); }}
              testId="flow-customization"
            />
            <QuickFlowCard
              title="Check analytics"
              description="View conversion rates and performance metrics"
              icon={BarChart3}
              iconBg="bg-blue-500/20"
              iconColor="text-blue-300"
              onClick={() => { navigate('/admin/analytics'); }}
              testId="flow-analytics"
            />
            <QuickFlowCard
              title="Update branding"
              description="Configure logo, colors, SEO, and site identity"
              icon={Settings2}
              iconBg="bg-emerald-500/20"
              iconColor="text-emerald-300"
              onClick={() => { navigate('/admin/branding'); }}
              testId="flow-branding"
            />
            <QuickFlowCard
              title="Trigger agent"
              description="Use AI to improve landing page content"
              icon={Bot}
              iconBg="bg-amber-500/20"
              iconColor="text-amber-300"
              onClick={() => { navigate('/admin/customization/agent'); }}
              testId="flow-agent"
            />
          </div>
        </div>

        {/* Experience Health */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="landing-health"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-6">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Experience health
              </p>
              <h2 className="text-xl font-semibold text-white mt-1">
                Variant status and traffic allocation
              </h2>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={previewPublicLanding}
                className="gap-2"
              >
                <ExternalLink className="h-4 w-4" />
                Preview
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => { void refreshHealthSnapshot(); }}
                className="gap-2"
                data-testid="landing-health-refresh"
              >
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </div>
          </div>

          {healthLoading ? (
            <div className="grid gap-4 md:grid-cols-4">
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-20 rounded-xl bg-white/5 animate-pulse"
                />
              ))}
            </div>
          ) : healthError ? (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center gap-3">
              <AlertTriangle className="h-4 w-4" />
              <span>{healthError}</span>
              <Button size="sm" variant="ghost" onClick={() => { void refreshHealthSnapshot(); }}>
                Retry
              </Button>
            </div>
          ) : healthSnapshot ? (
            <>
              {healthMetricsDegraded && (
                <div className="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100 flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  Analytics data partially unavailable. Variant status is still
                  accurate.
                </div>
              )}
              <div
                className="grid gap-4 text-center md:grid-cols-4 mb-6"
                data-testid="landing-health-stats"
              >
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                    Active variants
                  </p>
                  <p className="text-3xl font-semibold text-white mt-2">
                    {healthSnapshot.activeCount}
                  </p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                    Needs attention
                  </p>
                  <p
                    className={`text-3xl font-semibold mt-2 ${
                      healthSnapshot.attentionCount > 0
                        ? 'text-amber-300'
                        : 'text-slate-300'
                    }`}
                  >
                    {healthSnapshot.attentionCount}
                  </p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                    Traffic assigned
                  </p>
                  <p className="text-3xl font-semibold text-white mt-2">
                    {Math.max(0, Math.round(healthSnapshot.totalWeight))}%
                  </p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                    Live variant
                  </p>
                  <p className="text-lg font-semibold text-white mt-2 truncate">
                    {runtimeLabel}
                  </p>
                  <p className="text-xs text-slate-400">{resolutionLabel}</p>
                </div>
              </div>

              {/* Traffic allocation bar */}
              <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                <div className="flex items-center gap-3 mb-3">
                  <Gauge className="h-5 w-5 text-blue-300" />
                  <span className="font-semibold text-white">
                    Traffic allocation
                  </span>
                </div>
                <div className="h-2 rounded-full bg-white/10 overflow-hidden mb-2">
                  <div
                    className={`h-2 ${
                      healthSnapshot.weightStatus === 'balanced'
                        ? 'bg-emerald-400'
                        : healthSnapshot.weightStatus === 'empty'
                          ? 'bg-slate-500'
                          : 'bg-amber-400'
                    }`}
                    style={{
                      width: `${String(Math.max(0, Math.min(100, healthSnapshot.totalWeight)))}%`,
                    }}
                  />
                </div>
                <p className="text-sm text-slate-400">
                  {describeWeightStatus(
                    healthSnapshot.weightStatus,
                    healthSnapshot.totalWeight,
                    healthSnapshot.activeCount
                  )}
                </p>
              </div>
            </>
          ) : null}
        </div>

        {/* Branding Health */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="landing-branding-health"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-4">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Branding status
              </p>
              <h2 className="text-xl font-semibold text-white mt-1">
                Site identity and SEO configuration
              </h2>
            </div>
            <Button size="sm" onClick={() => { navigate('/admin/branding'); }}>
              Configure branding
            </Button>
          </div>

          {brandingLoading ? (
            <div className="h-16 rounded-xl bg-white/5 animate-pulse" />
          ) : brandingHealth ? (
            <StatusBadgeGrid
              columns={4}
              badges={[
                {
                  label: 'Identity',
                  status: brandingHealth.hasIdentity ? 'success' : 'warning',
                  description: brandingHealth.hasIdentity
                    ? 'Configured'
                    : 'Missing',
                },
                {
                  label: 'Favicon',
                  status: brandingHealth.hasFavicon ? 'success' : 'warning',
                  description: brandingHealth.hasFavicon
                    ? 'Configured'
                    : 'Missing',
                },
                {
                  label: 'SEO',
                  status: brandingHealth.hasSeo ? 'success' : 'warning',
                  description: brandingHealth.hasSeo ? 'Configured' : 'Missing',
                },
                {
                  label: 'OG Image',
                  status: brandingHealth.hasOgImage ? 'success' : 'warning',
                  description: brandingHealth.hasOgImage
                    ? 'Configured'
                    : 'Missing',
                },
              ]}
            />
          ) : (
            <p className="text-sm text-slate-400">
              Unable to load branding status
            </p>
          )}
        </div>

        {/* Resume Panel */}
        {(resumeVariant || resumeAnalytics) && (
          <div className="space-y-4" data-testid="landing-resume-panel">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
              Continue where you left off
            </p>
            <div className="grid gap-4 md:grid-cols-2">
              {resumeVariant && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-purple-500/15">
                      <History className="h-5 w-5 text-purple-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">
                        Last customization
                      </p>
                      <p className="text-lg font-semibold">
                        {resumeVariant.name ?? resumeVariant.slug}
                        {resumeVariant.surface === 'section' &&
                          resumeVariant.sectionType && (
                            <span className="text-sm font-normal text-slate-400">
                              {' '}
                              · {resumeVariant.sectionType}
                            </span>
                          )}
                      </p>
                    </div>
                  </div>
                  <Button
                    onClick={handleResumeVariant}
                    className="w-full gap-2"
                    data-testid="landing-resume-customization"
                  >
                    Return to{' '}
                    {resumeVariant.surface === 'section' ? 'Section' : 'Variant'}
                  </Button>
                </div>
              )}

              {resumeAnalytics && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-blue-500/15">
                      <Activity className="h-5 w-5 text-blue-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">
                        Last analytics view
                      </p>
                      <p className="text-lg font-semibold">
                        {resumeAnalytics.variantName ??
                          resumeAnalytics.variantSlug ??
                          'All variants'}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm text-slate-400 mb-4">
                    Showing last {resumeAnalytics.timeRangeDays} day
                    {resumeAnalytics.timeRangeDays === 1 ? '' : 's'} window.
                  </p>
                  <Button
                    onClick={handleResumeAnalytics}
                    variant="outline"
                    className="w-full gap-2"
                    data-testid="landing-resume-analytics"
                  >
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

interface QuickFlowCardProps {
  title: string;
  description: string;
  icon: React.ElementType;
  iconBg: string;
  iconColor: string;
  onClick: () => void;
  testId: string;
}

function QuickFlowCard({
  title,
  description,
  icon: Icon,
  iconBg,
  iconColor,
  onClick,
  testId,
}: QuickFlowCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group rounded-xl border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 transition-all"
      data-testid={testId}
    >
      <div className="flex items-center gap-3 mb-2">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-4 w-4 ${iconColor}`} />
        </div>
        <p className="font-semibold text-white">{title}</p>
      </div>
      <p className="text-sm text-slate-400">{description}</p>
      <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-slate-300 group-hover:translate-x-1 transition-transform">
        Go →
      </span>
    </button>
  );
}
