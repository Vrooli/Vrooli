import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { BarChart3, TrendingUp, Users, MousePointerClick, DownloadCloud, ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import { AdminLayout } from "../components/AdminLayout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../../shared/ui/card";
import { Button } from "../../../shared/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../../../shared/ui/select";
import { getMetricsSummary, getVariantMetrics, type AnalyticsSummary, type VariantStats } from "../../../shared/api";

const getTrendIcon = (trend?: 'up' | 'down' | 'stable') => {
  if (trend === 'up') return <ArrowUpRight className="h-4 w-4 text-green-400" />;
  if (trend === 'down') return <ArrowDownRight className="h-4 w-4 text-red-400" />;
  return <Minus className="h-4 w-4 text-slate-400" />;
};

const formatDate = (daysAgo: number): string => {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  return date.toISOString().split('T')[0];
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
  const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
  const [selectedVariant, setSelectedVariant] = useState<string>("all");
  const [timeRange, setTimeRange] = useState<string>("7");
  const [variantDetails, setVariantDetails] = useState<VariantStats[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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

  const fetchAnalytics = async () => {
    try {
      setLoading(true);
      const days = parseInt(timeRange);
      const endDate = formatDate(0);
      const startDate = formatDate(days);
      const data = await getMetricsSummary(startDate, endDate);
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
      const days = parseInt(timeRange);
      const endDate = formatDate(0);
      const startDate = formatDate(days);
      const data = await getVariantMetrics(variantSlug, startDate, endDate);
      setVariantDetails(data.stats);
    } catch (err) {
      console.error("Failed to load variant details:", err);
      setVariantDetails([]);
    }
  };

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
            <Select value={timeRange} onValueChange={setTimeRange}>
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

            <Select value={selectedVariant} onValueChange={setSelectedVariant}>
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
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setSelectedVariant(variant.variant_slug)}
                            data-testid={`analytics-view-details-${variant.variant_id}`}
                          >
                            Details →
                          </Button>
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
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </AdminLayout>
  );
}
