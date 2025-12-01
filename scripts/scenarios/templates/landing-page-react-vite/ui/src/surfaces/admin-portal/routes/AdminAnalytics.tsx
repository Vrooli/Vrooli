import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { BarChart3, TrendingUp, Users, MousePointerClick, DownloadCloud, ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import { AdminLayout } from "../components/AdminLayout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../shared/ui/card";
import { Button } from "../../../shared/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../shared/ui/select";
import { type AnalyticsSummary, type VariantStats } from "../../../shared/api";
import { buildDateRange, fetchAnalyticsSummary, fetchVariantAnalytics } from "../controllers/analyticsController";
import { getAdminExperienceSnapshot, rememberAnalyticsFilters } from "../../../shared/lib/adminExperience";
import { useLandingVariant, type VariantResolution } from "../../../app/providers/LandingVariantProvider";

const getTrendIcon = (trend?: 'up' | 'down' | 'stable') => {
  if (trend === 'up') return <ArrowUpRight className="h-4 w-4 text-green-400" />;
  if (trend === 'down') return <ArrowDownRight className="h-4 w-4 text-red-400" />;
  return <Minus className="h-4 w-4 text-slate-400" />;
};

const VALID_TIME_RANGES = new Set(['1', '7', '30', '90']);
const DEFAULT_TIME_RANGE = '7';
const TIME_RANGE_LABELS: Record<string, string> = {
  '1': 'Last 24 hours',
  '7': 'Last 7 days',
  '30': 'Last 30 days',
  '90': 'Last 90 days',
};
const RESOLUTION_LABELS: Record<VariantResolution, string> = {
  url_param: 'URL parameter',
  local_storage: 'Stored visitor assignment',
  api_select: 'Weighted API selection',
  fallback: 'Fallback payload',
  unknown: 'Unknown strategy',
};

/**
 * Admin Analytics Dashboard - implements OT-P0-023 and OT-P0-024
 *
 * OT-P0-023: Analytics summary shows total visitors, conversion rate per variant, top CTAs by CTR
 * OT-P0-020: Admin analytics view filters stats by variant and time range
 * OT-P0-024: Variant detail page shows views, CTA clicks, conversions, conversion rate, basic trend
 *
 * [REQ:METRIC-SUMMARY] [REQ:METRIC-DETAIL] [REQ:METRIC-FILTER]
 */
export function AdminAnalytics() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { variant: liveVariant, resolution: liveResolution, statusNote: liveStatusNote } = useLandingVariant();
  const adminExperience = useMemo(() => getAdminExperienceSnapshot(), []);
  const initialVariant = searchParams.get('variant') ?? adminExperience.lastAnalytics?.variantSlug ?? 'all';
  const initialTimeRangeFromUrl = searchParams.get('range');
  const initialRangeFromExperience = adminExperience.lastAnalytics?.timeRangeDays
    ? String(adminExperience.lastAnalytics.timeRangeDays)
    : undefined;
  const initialRange = initialTimeRangeFromUrl && VALID_TIME_RANGES.has(initialTimeRangeFromUrl)
    ? initialTimeRangeFromUrl
    : initialRangeFromExperience && VALID_TIME_RANGES.has(initialRangeFromExperience)
      ? initialRangeFromExperience
      : DEFAULT_TIME_RANGE;
  const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
  const [selectedVariant, setSelectedVariant] = useState<string>(initialVariant);
  const [timeRange, setTimeRange] = useState<string>(initialRange);
  const [variantDetails, setVariantDetails] = useState<VariantStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const searchSignature = searchParams.toString();
  const filtersChanged = selectedVariant !== 'all' || timeRange !== DEFAULT_TIME_RANGE;
  const selectedTimeRangeLabel = TIME_RANGE_LABELS[timeRange] ?? TIME_RANGE_LABELS[DEFAULT_TIME_RANGE];

  useEffect(() => {
    fetchAnalytics();
  }, [timeRange]);

  useEffect(() => {
    if (selectedVariant !== "all") {
      fetchVariantDetails(selectedVariant);
    } else {
      setVariantDetails([]);
    }
  }, [selectedVariant, timeRange]);

  useEffect(() => {
    if (!summary) return;
    const variantSlug = selectedVariant === 'all' ? null : selectedVariant;
    const variantName = variantSlug
      ? summary.variant_stats.find((stat) => stat.variant_slug === variantSlug)?.variant_name
      : undefined;
    rememberAnalyticsFilters({
      variantSlug,
      variantName,
      timeRangeDays: parseInt(timeRange, 10),
    });
  }, [summary, selectedVariant, timeRange]);

  useEffect(() => {
    const params = new URLSearchParams(searchSignature);
    const urlVariant = params.get('variant') ?? 'all';
    const urlRange = params.get('range');
    const normalizedRange = urlRange && VALID_TIME_RANGES.has(urlRange) ? urlRange : DEFAULT_TIME_RANGE;

    if (urlVariant !== selectedVariant) {
      setSelectedVariant(urlVariant);
    }
    if (normalizedRange !== timeRange) {
      setTimeRange(normalizedRange);
    }
  }, [searchSignature]);

  const fetchAnalytics = async () => {
    try {
      setLoading(true);
      const days = parseInt(timeRange, 10);
      const range = buildDateRange(days);
      const data = await fetchAnalyticsSummary(range);
      setSummary(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
      console.error('Analytics fetch error:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchVariantDetails = async (variantSlug: string) => {
    try {
      const range = buildDateRange(parseInt(timeRange, 10));
      const stats = await fetchVariantAnalytics(variantSlug, range);
      setVariantDetails(stats);
    } catch (err) {
      console.error("Failed to load variant details:", err);
      setVariantDetails([]);
    }
  };

  const syncFiltersToUrl = (nextVariant: string, nextRange: string) => {
    const params = new URLSearchParams();
    if (nextVariant !== 'all') {
      params.set('variant', nextVariant);
    }
    if (nextRange !== DEFAULT_TIME_RANGE) {
      params.set('range', nextRange);
    }

    const nextQuery = params.toString();
    if (nextQuery === searchSignature) {
      return;
    }

    setSearchParams(params, { replace: true });
  };

  const handleVariantChange = (value: string) => {
    setSelectedVariant(value);
    syncFiltersToUrl(value, timeRange);
  };

  const handleTimeRangeChange = (value: string) => {
    setTimeRange(value);
    syncFiltersToUrl(selectedVariant, value);
  };

  const handleResetFilters = () => {
    setSelectedVariant('all');
    setTimeRange(DEFAULT_TIME_RANGE);
    syncFiltersToUrl('all', DEFAULT_TIME_RANGE);
  };
  const navigateToHeroSection = (slug: string) => {
    const params = new URLSearchParams({ focus: slug, focusSectionType: 'hero' });
    navigate(`/admin/customization?${params.toString()}`);
  };

  const variantNameLookup = useMemo(() => {
    const map = new Map<string, string>();
    summary?.variant_stats.forEach((stat) => map.set(stat.variant_slug, stat.variant_name));
    return map;
  }, [summary]);

  const selectedVariantName = selectedVariant !== 'all'
    ? variantNameLookup.get(selectedVariant) ?? selectedVariant
    : null;
  const bestVariantStat = useMemo(() => {
    if (!summary?.variant_stats?.length) {
      return null;
    }
    return summary.variant_stats.reduce<VariantStats | null>((best, stat) => {
      if (!best) {
        return stat;
      }
      return stat.conversion_rate > best.conversion_rate ? stat : best;
    }, null);
  }, [summary]);
  const weakestVariantStat = useMemo(() => {
    if (!summary?.variant_stats?.length) {
      return null;
    }
    return summary.variant_stats.reduce<VariantStats | null>((worst, stat) => {
      if (!worst) {
        return stat;
      }
      return stat.conversion_rate < worst.conversion_rate ? stat : worst;
    }, null);
  }, [summary]);

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading analytics...</p>
        </div>
      </AdminLayout>
    );
  }

  if (error) {
    return (
      <AdminLayout>
        <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
          <p className="text-red-400">Error: {error}</p>
          <Button onClick={fetchAnalytics} variant="outline" className="mt-4">
            Retry
          </Button>
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout>
      <div className="max-w-7xl mx-auto">
        {/* Header with filters */}
        <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8 gap-4">
          <h1 className="text-3xl font-semibold">Analytics Dashboard</h1>

          <div className="flex gap-3" data-testid="analytics-filters">
            <Select value={timeRange} onValueChange={handleTimeRangeChange}>
              <SelectTrigger className="w-[140px] bg-white/5 border-white/10" data-testid="analytics-time-range">
                <SelectValue placeholder="Time range" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="1">Last 24 hours</SelectItem>
                <SelectItem value="7">Last 7 days</SelectItem>
                <SelectItem value="30">Last 30 days</SelectItem>
                <SelectItem value="90">Last 90 days</SelectItem>
              </SelectContent>
            </Select>

            <Select value={selectedVariant} onValueChange={handleVariantChange}>
              <SelectTrigger className="w-[160px] bg-white/5 border-white/10" data-testid="analytics-variant-filter">
                <SelectValue placeholder="All variants" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All variants</SelectItem>
                {summary?.variant_stats.map((v) => (
                  <SelectItem key={v.variant_id} value={v.variant_slug}>
                    {v.variant_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <AnalyticsFocusBanner
          selectedVariantSlug={selectedVariant === 'all' ? null : selectedVariant}
          selectedVariantName={selectedVariantName}
          timeRangeLabel={selectedTimeRangeLabel}
          filtersChanged={filtersChanged}
          onResetFilters={filtersChanged ? handleResetFilters : undefined}
          onCustomizeVariant={selectedVariant !== 'all' ? () => navigate(`/admin/customization/variants/${selectedVariant}`) : undefined}
          onPreviewVariant={selectedVariant !== 'all' ? () => window.open(`/?variant=${selectedVariant}`, '_blank') : undefined}
          liveVariant={liveVariant}
          liveResolution={liveResolution}
          liveStatusNote={liveStatusNote}
        />

        <AnalyticsShortcutsCard
          liveVariant={liveVariant}
          liveResolution={liveResolution}
          liveStatusNote={liveStatusNote}
          onFocusLiveVariant={() => {
            if (liveVariant?.slug) {
              handleVariantChange(liveVariant.slug);
            }
          }}
          bestVariant={bestVariantStat}
          weakestVariant={weakestVariantStat}
          onFocusVariant={(slug) => handleVariantChange(slug)}
          onCustomizeVariant={(slug) => navigate(`/admin/customization/variants/${slug}`)}
          timeRangeDays={parseInt(timeRange, 10)}
        />

        {/* Summary cards - OT-P0-023 */}
        <div className="grid md:grid-cols-4 gap-6 mb-8">
          <Card className="bg-white/5 border-white/10" data-testid="analytics-total-visitors">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-slate-300">Total Visitors</CardTitle>
              <Users className="h-4 w-4 text-slate-400" />
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{summary?.total_visitors.toLocaleString() ?? 0}</div>
              <p className="text-xs text-slate-400 mt-1">Unique visitors in selected period</p>
            </CardContent>
          </Card>

          <Card className="bg-white/5 border-white/10" data-testid="analytics-total-downloads">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-slate-300">Downloads</CardTitle>
              <DownloadCloud className="h-4 w-4 text-slate-400" />
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {(summary?.total_downloads ?? 0).toLocaleString()}
              </div>
              <p className="text-xs text-slate-400 mt-1">Verified download events</p>
            </CardContent>
          </Card>

          <Card className="bg-white/5 border-white/10" data-testid="analytics-conversion-rate">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-slate-300">Avg Conversion Rate</CardTitle>
              <TrendingUp className="h-4 w-4 text-slate-400" />
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">
                {summary && summary.variant_stats.length > 0
                  ? (summary.variant_stats.reduce((sum, v) => sum + v.conversion_rate, 0) / summary.variant_stats.length).toFixed(2)
                  : '0.00'}%
              </div>
              <p className="text-xs text-slate-400 mt-1">Across all variants</p>
            </CardContent>
          </Card>

          <Card className="bg-white/5 border-white/10" data-testid="analytics-top-cta">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-slate-300">Top CTA</CardTitle>
              <MousePointerClick className="h-4 w-4 text-slate-400" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold truncate">
                {summary?.top_cta ?? "N/A"}
              </div>
              <p className="text-xs text-slate-400 mt-1">
                {summary?.top_cta_ctr ? `${summary.top_cta_ctr.toFixed(1)}% CTR` : "No data yet"}
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Variant performance table - OT-P0-024 */}
        <Card className="bg-white/5 border-white/10" data-testid="analytics-variant-performance">
          <CardHeader>
            <CardTitle>Variant Performance</CardTitle>
            <CardDescription className="text-slate-400">
              Compare conversion rates and metrics across all A/B test variants
            </CardDescription>
          </CardHeader>
          <CardContent>
            {!summary || summary.variant_stats.length === 0 ? (
              <div className="text-center py-12 text-slate-400">
                <BarChart3 className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>No variant data available yet</p>
                <p className="text-sm mt-2">Create variants from the Customization section</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-white/10 text-slate-400 text-sm">
                      <th className="text-left py-3 px-4">Variant</th>
                      <th className="text-right py-3 px-4">Views</th>
                      <th className="text-right py-3 px-4">CTA Clicks</th>
                      <th className="text-right py-3 px-4">Conversions</th>
                      <th className="text-right py-3 px-4">Downloads</th>
                      <th className="text-right py-3 px-4">Conv. Rate</th>
                      <th className="text-right py-3 px-4">Trend</th>
                      <th className="text-right py-3 px-4"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {summary.variant_stats.map((variant) => (
                      <tr
                        key={variant.variant_id}
                        className="border-b border-white/5 hover:bg-white/5 transition-colors"
                        data-testid={`analytics-variant-row-${variant.variant_id}`}
                      >
                        <td className="py-4 px-4">
                          <div className="font-medium">{variant.variant_name}</div>
                          <div className="text-xs text-slate-500">{variant.variant_slug}</div>
                        </td>
                        <td className="text-right py-4 px-4">{variant.views.toLocaleString()}</td>
                        <td className="text-right py-4 px-4">{variant.cta_clicks.toLocaleString()}</td>
                        <td className="text-right py-4 px-4">{variant.conversions.toLocaleString()}</td>
                        <td
                          className="text-right py-4 px-4"
                          data-testid={`analytics-downloads-${variant.variant_id}`}
                        >
                          {variant.downloads.toLocaleString()}
                        </td>
                        <td className="text-right py-4 px-4">
                          <span className={variant.conversion_rate > 5 ? "text-green-400 font-semibold" : "text-slate-300"}>
                            {variant.conversion_rate.toFixed(2)}%
                          </span>
                        </td>
                        <td className="text-right py-4 px-4">
                          <div className="flex items-center justify-end gap-1">
                            {getTrendIcon(variant.trend)}
                            <span className="text-sm capitalize">{variant.trend ?? 'stable'}</span>
                          </div>
                        </td>
                        <td className="text-right py-4 px-4">
                          <div className="flex flex-col items-end gap-1">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => setSelectedVariant(variant.variant_slug)}
                              data-testid={`analytics-view-details-${variant.variant_id}`}
                            >
                              Details →
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => navigate(`/admin/customization/variants/${variant.variant_slug}`)}
                              data-testid={`analytics-edit-${variant.variant_id}`}
                            >
                              Customize
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => navigateToHeroSection(variant.variant_slug)}
                              data-testid={`analytics-edit-hero-${variant.variant_id}`}
                            >
                              Edit hero
                            </Button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Variant detail view - OT-P0-024 */}
        {selectedVariant !== "all" && variantDetails.length > 0 && (
          <Card className="mt-6 bg-white/5 border-white/10" data-testid="analytics-variant-detail">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Detailed Variant Stats</CardTitle>
                  <CardDescription className="text-slate-400">
                    In-depth metrics for {variantDetails[0]?.variant_name ?? selectedVariant}
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setSelectedVariant("all")}
                >
                  Back to All Variants
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="grid md:grid-cols-5 gap-6">
                <div className="p-4 rounded-lg bg-blue-500/10 border border-blue-500/20">
                  <div className="text-sm text-slate-400 mb-1">Views</div>
                  <div className="text-2xl font-bold">{variantDetails[0]?.views.toLocaleString() ?? 0}</div>
                </div>
                <div className="p-4 rounded-lg bg-purple-500/10 border border-purple-500/20">
                  <div className="text-sm text-slate-400 mb-1">CTA Clicks</div>
                  <div className="text-2xl font-bold">{variantDetails[0]?.cta_clicks.toLocaleString() ?? 0}</div>
                </div>
                <div className="p-4 rounded-lg bg-green-500/10 border border-green-500/20">
                  <div className="text-sm text-slate-400 mb-1">Conversions</div>
                  <div className="text-2xl font-bold">{variantDetails[0]?.conversions.toLocaleString() ?? 0}</div>
                </div>
                <div className="p-4 rounded-lg bg-amber-500/10 border border-amber-500/20">
                  <div className="text-sm text-slate-400 mb-1">Conversion Rate</div>
                  <div className="text-2xl font-bold">{variantDetails[0]?.conversion_rate.toFixed(2) ?? 0}%</div>
                </div>
                <div className="p-4 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
                  <div className="text-sm text-slate-400 mb-1">Downloads</div>
                  <div className="text-2xl font-bold">{variantDetails[0]?.downloads.toLocaleString() ?? 0}</div>
                </div>
              </div>

              <div className="mt-6 pt-6 border-t border-white/10">
                <div className="text-sm text-slate-400 mb-3">Performance Trend</div>
                <div className="flex items-center gap-3">
                  {getTrendIcon(variantDetails[0]?.trend)}
                  <div>
                    <span className="text-lg font-medium capitalize">{variantDetails[0]?.trend ?? "stable"}</span>
                    <p className="text-xs text-slate-500 mt-1">
                      Based on last {timeRange} {parseInt(timeRange) === 1 ? 'day' : 'days'}
                    </p>
                  </div>
                </div>
                {variantDetails[0]?.avg_scroll_depth !== undefined && (
                  <div className="mt-4 pt-4 border-t border-white/10">
                    <div className="text-sm text-slate-400 mb-2">Average Scroll Depth</div>
                    <div className="text-lg font-medium">{(variantDetails[0].avg_scroll_depth * 100).toFixed(1)}%</div>
                  </div>
                )}
                <div className="mt-6 grid gap-3 md:grid-cols-2" data-testid="analytics-variant-actions">
                  <Button
                    variant="outline"
                    className="gap-2"
                    onClick={() => navigate(`/admin/customization/variants/${selectedVariant}`)}
                  >
                    Edit {selectedVariantName ?? 'variant'}
                  </Button>
                  <Button
                    variant="outline"
                    className="gap-2"
                    onClick={() => window.open(`/?variant=${selectedVariant}`, '_blank')}
                  >
                    Preview pinned variant
                  </Button>
                </div>
                <div className="mt-3 flex justify-end">
                  <Button
                    size="sm"
                    variant="outline"
                    className="gap-2"
                    onClick={() => navigateToHeroSection(selectedVariant)}
                    data-testid="analytics-variant-edit-hero"
                  >
                    Jump to hero section
                  </Button>
                </div>
                <p className="text-xs text-slate-500 mt-2">
                  Apply changes in Customization, then refresh this detail view to confirm the next experiment run.
                </p>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </AdminLayout>
  );
}

interface AnalyticsShortcutsCardProps {
  liveVariant: { slug: string; name: string } | null;
  liveResolution: VariantResolution;
  liveStatusNote: string | null;
  onFocusLiveVariant: () => void;
  bestVariant: VariantStats | null;
  weakestVariant: VariantStats | null;
  onFocusVariant: (slug: string) => void;
  onCustomizeVariant: (slug: string) => void;
  timeRangeDays: number;
}

interface AnalyticsFocusBannerProps {
  selectedVariantSlug: string | null;
  selectedVariantName: string | null;
  timeRangeLabel: string;
  filtersChanged: boolean;
  onResetFilters?: () => void;
  onCustomizeVariant?: () => void;
  onPreviewVariant?: () => void;
  liveVariant: { slug: string; name: string } | null;
  liveResolution: VariantResolution;
  liveStatusNote: string | null;
}

function AnalyticsFocusBanner({
  selectedVariantSlug,
  selectedVariantName,
  timeRangeLabel,
  filtersChanged,
  onResetFilters,
  onCustomizeVariant,
  onPreviewVariant,
  liveVariant,
  liveResolution,
  liveStatusNote,
}: AnalyticsFocusBannerProps) {
  const focusHeading = selectedVariantSlug
    ? `Analyzing ${selectedVariantName ?? selectedVariantSlug}`
    : 'Analyzing all variants';
  const runtimeVariantLabel = liveVariant?.name ?? liveVariant?.slug ?? 'runtime variant not resolved yet';
  const resolutionLabel = RESOLUTION_LABELS[liveResolution] ?? RESOLUTION_LABELS.unknown;
  let runtimeMessage = liveVariant?.slug
    ? `Live runtime is serving ${runtimeVariantLabel} via ${resolutionLabel}.`
    : 'Live runtime has not selected a variant yet.';

  if (selectedVariantSlug && liveVariant?.slug && liveVariant.slug !== selectedVariantSlug) {
    runtimeMessage += ` You are exploring ${selectedVariantName ?? selectedVariantSlug} while visitors currently see ${runtimeVariantLabel}. Use this to compare performance.`;
  }

  if (selectedVariantSlug && liveVariant?.slug && liveVariant.slug === selectedVariantSlug) {
    runtimeMessage += ' This matches the variant currently in rotation.';
  }

  if (liveStatusNote) {
    runtimeMessage += ` ${liveStatusNote}`;
  }

  return (
    <Card className="mb-6 bg-white/5 border-white/10" data-testid="analytics-focus-banner">
      <CardContent className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Current view</p>
          <h2 className="text-xl font-semibold text-white mt-2">{focusHeading}</h2>
          <p className="text-sm text-slate-400">Time range: {timeRangeLabel}</p>
          <p className="text-xs text-slate-500 mt-2">{runtimeMessage}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {onResetFilters && filtersChanged && (
            <Button size="sm" variant="outline" onClick={onResetFilters} data-testid="analytics-reset-filters">
              Reset filters
            </Button>
          )}
          {onCustomizeVariant && selectedVariantSlug && (
            <Button size="sm" onClick={onCustomizeVariant} data-testid="analytics-focus-customize">
              Customize variant
            </Button>
          )}
          {onPreviewVariant && selectedVariantSlug && (
            <Button size="sm" variant="outline" onClick={onPreviewVariant} data-testid="analytics-focus-preview">
              Preview
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function AnalyticsShortcutsCard({
  liveVariant,
  liveResolution,
  liveStatusNote,
  onFocusLiveVariant,
  bestVariant,
  weakestVariant,
  onFocusVariant,
  onCustomizeVariant,
  timeRangeDays,
}: AnalyticsShortcutsCardProps) {
  if (!liveVariant && !bestVariant && !weakestVariant) {
    return null;
  }
  const bestIsWeakest = Boolean(bestVariant && weakestVariant && bestVariant.variant_slug === weakestVariant.variant_slug);
  const showNeedsAttention = weakestVariant && !bestIsWeakest;

  return (
    <Card className="mb-8 bg-white/5 border-white/10" data-testid="analytics-shortcuts">
      <CardHeader>
        <CardTitle>Experience Shortcuts</CardTitle>
        <CardDescription className="text-slate-400">
          Jump from observed traffic to the right optimization workflow in a single click.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-3">
          <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-2">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Live traffic</p>
            {liveVariant ? (
              <>
                <p className="text-xl font-semibold text-white">{liveVariant.name ?? liveVariant.slug}</p>
                <p className="text-xs text-slate-400">Source: {RESOLUTION_LABELS[liveResolution]}</p>
                {liveStatusNote && <p className="text-xs text-slate-500">{liveStatusNote}</p>}
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-2"
                  onClick={onFocusLiveVariant}
                  disabled={!liveVariant.slug}
                >
                  Focus analytics
                </Button>
              </>
            ) : (
              <p className="text-sm text-slate-400">Landing runtime hasn’t selected a variant yet.</p>
            )}
          </div>

          <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-2">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Top performer</p>
            {bestVariant ? (
              <>
                <p className="text-xl font-semibold text-white">{bestVariant.variant_name}</p>
                <p className="text-3xl font-bold text-emerald-300">{bestVariant.conversion_rate.toFixed(2)}%</p>
                <p className="text-xs text-slate-500">Last {timeRangeDays} day{timeRangeDays === 1 ? '' : 's'}</p>
                <div className="flex gap-2 mt-2">
                  <Button size="sm" onClick={() => onFocusVariant(bestVariant.variant_slug)}>
                    View breakdown
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => onCustomizeVariant(bestVariant.variant_slug)}>
                    Tune messaging
                  </Button>
                </div>
              </>
            ) : (
              <p className="text-sm text-slate-400">Collect events to identify a leading variant.</p>
            )}
          </div>

          <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-2">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Needs attention</p>
            {showNeedsAttention ? (
              <>
                <p className="text-xl font-semibold text-white">{weakestVariant!.variant_name}</p>
                <p className="text-3xl font-bold text-rose-300">{weakestVariant!.conversion_rate.toFixed(2)}%</p>
                <p className="text-xs text-slate-500">Slowest performer right now</p>
                <div className="flex gap-2 mt-2">
                  <Button size="sm" onClick={() => onFocusVariant(weakestVariant!.variant_slug)}>
                    Inspect metrics
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => onCustomizeVariant(weakestVariant!.variant_slug)}>
                    Edit variant
                  </Button>
                </div>
              </>
            ) : (
              <p className="text-sm text-slate-400">No variant stands out as underperforming yet.</p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
