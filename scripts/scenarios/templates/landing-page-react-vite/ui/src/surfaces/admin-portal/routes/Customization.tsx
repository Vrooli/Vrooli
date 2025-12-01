import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Edit, Archive, Trash2, Sparkles, Eye, AlertTriangle, ArrowUpRight, ArrowDownRight, Minus } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { listVariants, archiveVariant, deleteVariant, type Variant, type AnalyticsSummary, type VariantStats } from '../../../shared/api';
import { buildDateRange, fetchAnalyticsSummary } from '../controllers/analyticsController';

const getTrendGlyph = (trend?: VariantStats['trend']) => {
  if (trend === 'up') return <ArrowUpRight className="h-3 w-3 text-emerald-300" />;
  if (trend === 'down') return <ArrowDownRight className="h-3 w-3 text-rose-300" />;
  return <Minus className="h-3 w-3 text-slate-400" />;
};

/**
 * Customization home page - implements OT-P0-010 (≤ 3 clicks to any customization card)
 * Shows variant list and agent customization trigger
 *
 * [REQ:ADMIN-NAV] [REQ:VARIANT-MGMT]
 */
export function Customization() {
  const navigate = useNavigate();
  const [variants, setVariants] = useState<Variant[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null);
  const [analyticsError, setAnalyticsError] = useState<string | null>(null);
  const [analyticsLoading, setAnalyticsLoading] = useState(true);

  useEffect(() => {
    fetchVariants();
    fetchAnalyticsSnapshot();
  }, []);

  const fetchVariants = async () => {
    try {
      setLoading(true);
      const data = await listVariants();
      setVariants(data.variants);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load variants');
      console.error('Variants fetch error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleArchive = async (slug: string) => {
    if (!confirm('Archive this variant? It will no longer be shown to visitors but analytics will be preserved.')) {
      return;
    }

    try {
      await archiveVariant(slug);
      await fetchVariants();
    } catch (err) {
      alert(`Failed to archive variant: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleDelete = async (slug: string) => {
    if (!confirm('Permanently delete this variant? This action cannot be undone and all analytics data will be lost.')) {
      return;
    }

    try {
      await deleteVariant(slug);
      await fetchVariants();
    } catch (err) {
      alert(`Failed to delete variant: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const fetchAnalyticsSnapshot = async () => {
    try {
      setAnalyticsLoading(true);
      const range = buildDateRange(7);
      const data = await fetchAnalyticsSummary(range);
      setAnalytics(data);
      setAnalyticsError(null);
    } catch (err) {
      setAnalyticsError(err instanceof Error ? err.message : 'Metrics not available');
      console.error('Customization analytics error:', err);
    } finally {
      setAnalyticsLoading(false);
    }
  };

  const activeVariants = variants.filter(v => v.status === 'active');
  const archivedVariants = variants.filter(v => v.status === 'archived');
  const statsBySlug = useMemo(() => {
    const map = new Map<string, VariantStats>();
    analytics?.variant_stats.forEach((stat) => map.set(stat.variant_slug, stat));
    return map;
  }, [analytics]);

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading variants...</p>
        </div>
      </AdminLayout>
    );
  }

  if (error) {
    return (
      <AdminLayout>
        <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4">
          <p className="text-red-400">Error: {error}</p>
          <Button onClick={fetchVariants} variant="outline" className="mt-4">
            Retry
          </Button>
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout>
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-semibold mb-2">Customization</h1>
            <p className="text-slate-400">Manage A/B test variants and customize landing page content</p>
          </div>
          <div className="flex gap-3">
            <Button
              onClick={() => navigate('/admin/customization/agent')}
              variant="outline"
              className="gap-2"
              data-testid="trigger-agent-customization"
            >
              <Sparkles className="h-4 w-4" />
              Agent Customization
            </Button>
            <Button
              onClick={() => navigate('/admin/customization/variants/new')}
              className="gap-2"
              data-testid="create-variant"
            >
              <Plus className="h-4 w-4" />
              New Variant
            </Button>
          </div>
        </div>

        {analyticsError && (
          <div className="mb-6 rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 flex items-center gap-3 text-sm text-amber-100">
            <AlertTriangle className="h-4 w-4 text-amber-300" />
            <span>Metrics snapshot unavailable right now. Variant cards show configuration only.</span>
          </div>
        )}

        {/* Active Variants */}
        <div className="mb-8">
          <h2 className="text-xl font-semibold mb-4">Active Variants</h2>
          {activeVariants.length === 0 ? (
            <Card className="bg-white/5 border-white/10">
              <CardContent className="text-center py-12">
                <p className="text-slate-400 mb-4">No active variants yet</p>
                <Button onClick={() => navigate('/admin/customization/variants/new')}>
                  Create Your First Variant
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {activeVariants.map((variant) => (
                <Card
                  key={variant.id}
                  className="bg-white/5 border-white/10 hover:bg-white/10 transition-colors"
                  data-testid={`variant-card-${variant.slug}`}
                >
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <CardTitle className="text-lg">{variant.name}</CardTitle>
                        <CardDescription className="text-slate-400 text-sm mt-1">
                          {variant.slug}
                        </CardDescription>
                      </div>
                      <div className="flex items-center gap-1 px-2 py-1 rounded text-xs bg-blue-500/20 text-blue-400">
                        Weight: {variant.weight}%
                      </div>
                    </div>
                    {variant.description && (
                      <p className="text-sm text-slate-300 mt-2">{variant.description}</p>
                    )}
                  </CardHeader>
                  <CardContent>
                    <VariantPerformanceSummary
                      slug={variant.slug}
                      stats={statsBySlug.get(variant.slug)}
                      loading={analyticsLoading}
                    />
                    {variant.axes && (
                      <div className="flex flex-wrap gap-2 mb-4">
                        {Object.entries(variant.axes).map(([axisId, axisValue]) => (
                          <span key={axisId} className="text-xs px-2 py-1 rounded-full bg-blue-500/15 text-blue-200">
                            {axisId}: {axisValue}
                          </span>
                        ))}
                      </div>
                    )}
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-1 gap-2"
                        onClick={() => navigate(`/admin/customization/variants/${variant.slug}`)}
                        data-testid={`edit-variant-${variant.slug}`}
                      >
                        <Edit className="h-4 w-4" />
                        Edit
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => window.open(`/?variant=${variant.slug}`, '_blank')}
                        data-testid={`preview-variant-${variant.slug}`}
                      >
                        <Eye className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleArchive(variant.slug)}
                        data-testid={`archive-variant-${variant.slug}`}
                      >
                        <Archive className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => navigate(`/admin/analytics?variant=${variant.slug}`)}
                        data-testid={`variant-analytics-${variant.slug}`}
                      >
                        View Analytics
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>

        {/* Archived Variants */}
        {archivedVariants.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold mb-4 text-slate-400">Archived Variants</h2>
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {archivedVariants.map((variant) => (
                <Card
                  key={variant.id}
                  className="bg-white/5 border-white/10 opacity-60"
                  data-testid={`variant-card-archived-${variant.slug}`}
                >
                  <CardHeader>
                    <CardTitle className="text-lg">{variant.name}</CardTitle>
                    <CardDescription className="text-slate-400 text-sm">
                      {variant.slug} • Archived {variant.archived_at ? new Date(variant.archived_at).toLocaleDateString() : ''}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {variant.axes && (
                      <div className="flex flex-wrap gap-2 mb-4">
                        {Object.entries(variant.axes).map(([axisId, axisValue]) => (
                          <span key={axisId} className="text-xs px-2 py-1 rounded-full bg-slate-700/60 text-slate-200">
                            {axisId}: {axisValue}
                          </span>
                        ))}
                      </div>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDelete(variant.slug)}
                      className="w-full gap-2 text-red-400 hover:text-red-300"
                      data-testid={`delete-variant-${variant.slug}`}
                    >
                      <Trash2 className="h-4 w-4" />
                      Delete Permanently
                    </Button>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}

interface VariantPerformanceSummaryProps {
  slug: string;
  stats?: VariantStats;
  loading: boolean;
}

function VariantPerformanceSummary({ slug, stats, loading }: VariantPerformanceSummaryProps) {
  if (loading && !stats) {
    return (
      <div className="mb-4 rounded-lg border border-white/5 bg-white/5 p-3 text-xs text-slate-400" data-testid={`variant-performance-${slug}`}>
        Syncing analytics snapshot...
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="mb-4 rounded-lg border border-white/5 bg-white/5 p-3 text-xs text-slate-400" data-testid={`variant-performance-${slug}`}>
        No analytics events captured yet. Drive traffic to see performance indicators here.
      </div>
    );
  }

  return (
    <div className="mb-4 rounded-lg border border-white/10 bg-slate-900/40 p-3" data-testid={`variant-performance-${slug}`}>
      <div className="flex items-center justify-between text-xs text-slate-400 mb-2">
        <span>Last 7 days</span>
        <div className="flex items-center gap-1 capitalize text-slate-300">
          {getTrendGlyph(stats.trend)}
          <span>{stats.trend ?? 'stable'}</span>
        </div>
      </div>
      <div className="flex flex-wrap gap-4 text-sm text-slate-200">
        <div>
          <span className="font-semibold text-white">{stats.views.toLocaleString()}</span> views
        </div>
        <div>
          <span className="font-semibold text-white">{stats.conversions.toLocaleString()}</span> conversions
        </div>
        <div>
          <span className="font-semibold text-white">{stats.conversion_rate.toFixed(2)}%</span> CVR
        </div>
      </div>
    </div>
  );
}
