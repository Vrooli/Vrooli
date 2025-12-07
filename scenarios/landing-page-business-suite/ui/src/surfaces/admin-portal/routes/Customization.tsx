import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Plus, Edit, Archive, Trash2, Sparkles, Eye, AlertTriangle, ArrowUpRight, ArrowDownRight, Minus, Search } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { listVariants, archiveVariant, deleteVariant, type Variant, type AnalyticsSummary, type VariantStats } from '../../../shared/api';
import { buildDateRange, fetchAnalyticsSummary } from '../controllers/analyticsController';
import { loadVariantEditorData } from '../controllers/variantEditorController';

const STALE_VARIANT_DAYS = 10;
const SNAPSHOT_DAYS = 7;
const DAY_MS = 24 * 60 * 60 * 1000;
type WeightStatus = 'empty' | 'balanced' | 'under' | 'over';

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
  const [searchParams, setSearchParams] = useSearchParams();
  const [variants, setVariants] = useState<Variant[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [analytics, setAnalytics] = useState<AnalyticsSummary | null>(null);
  const [analyticsError, setAnalyticsError] = useState<string | null>(null);
  const [analyticsLoading, setAnalyticsLoading] = useState(true);
  const [variantQuery, setVariantQuery] = useState('');
  const [attentionOnly, setAttentionOnly] = useState(false);
  const variantListRef = useRef<HTMLDivElement | null>(null);
  const [appliedFocusSlug, setAppliedFocusSlug] = useState<string | null>(null);
  const [appliedSectionFocusSlug, setAppliedSectionFocusSlug] = useState<string | null>(null);
  const focusSlug = searchParams.get('focus');
  const focusSectionIdParam = searchParams.get('focusSectionId');
  const focusSectionType = searchParams.get('focusSectionType');
  const focusSectionId = focusSectionIdParam ? Number(focusSectionIdParam) : null;

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
      const range = buildDateRange(SNAPSHOT_DAYS);
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
  const totalAssignedWeight = activeVariants.reduce((sum, variant) => sum + (variant.weight ?? 0), 0);
  const weightStatus: WeightStatus = activeVariants.length === 0
    ? 'empty'
    : totalAssignedWeight === 100
      ? 'balanced'
      : totalAssignedWeight > 100
        ? 'over'
        : 'under';
  const staleVariants = useMemo(() => {
    const now = Date.now();
    return activeVariants
      .map((variant) => {
        if (!variant.updated_at) {
          return null;
        }
        const updatedAt = new Date(variant.updated_at);
        if (Number.isNaN(updatedAt.getTime())) {
          return null;
        }
        const daysSinceUpdate = Math.floor((now - updatedAt.getTime()) / DAY_MS);
        if (daysSinceUpdate < STALE_VARIANT_DAYS) {
          return null;
        }
        return { variant, daysSinceUpdate };
      })
      .filter((entry): entry is { variant: Variant; daysSinceUpdate: number } => Boolean(entry))
      .sort((a, b) => b.daysSinceUpdate - a.daysSinceUpdate)
      .slice(0, 3);
  }, [activeVariants]);
  const neverUpdatedVariants = useMemo(
    () => activeVariants.filter((variant) => !variant.updated_at),
    [activeVariants]
  );
  const underperformingStat = useMemo(() => {
    if (!analytics?.variant_stats?.length || activeVariants.length === 0) {
      return null;
    }
    const activeSlugs = new Set(activeVariants.map((variant) => variant.slug));
    const relevant = analytics.variant_stats.filter((stat) => activeSlugs.has(stat.variant_slug));
    if (relevant.length === 0) {
      return null;
    }
    return relevant.reduce<VariantStats | null>((worst, stat) => {
      if (!worst) {
        return stat;
      }
      return stat.conversion_rate < worst.conversion_rate ? stat : worst;
    }, null);
  }, [analytics, activeVariants]);
  const underperformingVariant = underperformingStat
    ? activeVariants.find((variant) => variant.slug === underperformingStat.variant_slug)
    : undefined;
  const attentionCandidateSlugs = useMemo(() => {
    const slugs = new Set<string>();
    staleVariants.forEach(({ variant }) => slugs.add(variant.slug));
    neverUpdatedVariants.forEach((variant) => slugs.add(variant.slug));
    if (underperformingStat) {
      slugs.add(underperformingStat.variant_slug);
    }
    return slugs;
  }, [staleVariants, neverUpdatedVariants, underperformingStat]);
  const variantAttentionReasons = useMemo(() => {
    const map = new Map<string, string[]>();
    const pushReason = (slug: string, reason: string) => {
      map.set(slug, [...(map.get(slug) ?? []), reason]);
    };
    staleVariants.forEach(({ variant, daysSinceUpdate }) => {
      pushReason(variant.slug, `Stale · ${daysSinceUpdate}d`);
    });
    neverUpdatedVariants.forEach((variant) => {
      pushReason(variant.slug, 'Never customized');
    });
    if (underperformingStat?.variant_slug) {
      pushReason(underperformingStat.variant_slug, 'Lowest conversion');
    }
    return map;
  }, [staleVariants, neverUpdatedVariants, underperformingStat]);
  const filteredActiveVariants = useMemo(() => {
    const normalized = variantQuery.trim().toLowerCase();
    return activeVariants.filter((variant) => {
      const matchesQuery = !normalized
        || variant.name?.toLowerCase().includes(normalized)
        || variant.slug.toLowerCase().includes(normalized);
      const matchesAttention = !attentionOnly || attentionCandidateSlugs.has(variant.slug);
      return matchesQuery && matchesAttention;
    });
  }, [activeVariants, attentionOnly, attentionCandidateSlugs, variantQuery]);

  const highlightVariantInList = useCallback((slug?: string) => {
    if (!slug) return;
    setAttentionOnly(true);
    setVariantQuery(slug);
    requestAnimationFrame(() => {
      variantListRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  }, []);

  const clearSectionFocusParams = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    next.delete('focusSectionId');
    next.delete('focusSectionType');
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const navigateToSectionEditor = useCallback(async (slug: string, options?: { sectionId?: number; sectionType?: string }) => {
    try {
      if (options?.sectionId) {
        navigate(`/admin/customization/variants/${slug}/sections/${options.sectionId}`);
        return true;
      }

      const desiredType = options?.sectionType;
      const data = await loadVariantEditorData(slug);
      const target = desiredType
        ? data.sections.find((section) => section.section_type === desiredType)
        : data.sections[0];

      if (target?.id) {
        navigate(`/admin/customization/variants/${slug}/sections/${target.id}`);
        return true;
      }

      navigate(`/admin/customization/variants/${slug}`);
      return false;
    } catch (error) {
      console.error('Failed to resolve section editor for variant', slug, error);
      navigate(`/admin/customization/variants/${slug}`);
      return false;
    }
  }, [navigate]);

  useEffect(() => {
    if (!focusSlug || variants.length === 0 || focusSlug === appliedFocusSlug) {
      return;
    }
    highlightVariantInList(focusSlug);
    setAppliedFocusSlug(focusSlug);
    const next = new URLSearchParams(searchParams);
    next.delete('focus');
    setSearchParams(next, { replace: true });
  }, [focusSlug, appliedFocusSlug, variants.length, highlightVariantInList, searchParams, setSearchParams]);

  useEffect(() => {
    if (!focusSlug || !variants.length) {
      setAppliedFocusSlug(null);
      setAppliedSectionFocusSlug(null);
      return;
    }

    if ((!focusSectionId && !focusSectionType) || appliedSectionFocusSlug === focusSlug) {
      return;
    }

    let cancelled = false;
    (async () => {
      const success = await navigateToSectionEditor(focusSlug, {
        sectionId: focusSectionId ?? undefined,
        sectionType: focusSectionType ?? undefined,
      });
      if (!cancelled && success) {
        setAppliedSectionFocusSlug(focusSlug);
        clearSectionFocusParams();
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [
    focusSlug,
    focusSectionId,
    focusSectionType,
    appliedSectionFocusSlug,
    variants.length,
    navigateToSectionEditor,
    clearSectionFocusParams,
  ]);

  useEffect(() => {
    if (!focusSlug) {
      setAppliedFocusSlug(null);
      setAppliedSectionFocusSlug(null);
    }
  }, [focusSlug]);

  const clearVariantFilters = () => {
    setAttentionOnly(false);
    setVariantQuery('');
  };

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

        <VariantFilterBar
          query={variantQuery}
          onQueryChange={setVariantQuery}
          attentionOnly={attentionOnly}
          onAttentionToggle={() => setAttentionOnly((prev) => !prev)}
          hasFilters={Boolean(variantQuery) || attentionOnly}
          onClearFilters={clearVariantFilters}
        />

      <ExperienceOpsPanel
        totalWeight={totalAssignedWeight}
        variantCount={activeVariants.length}
        weightStatus={weightStatus}
        staleVariants={staleVariants}
        neverUpdatedVariants={neverUpdatedVariants}
        underperforming={underperformingStat ? { stats: underperformingStat, variant: underperformingVariant } : null}
        analyticsRangeDays={SNAPSHOT_DAYS}
        onEditVariant={(slug) => navigate(`/admin/customization/variants/${slug}`)}
        onViewAnalytics={(slug) => navigate(`/admin/analytics?variant=${slug}`)}
        onHighlightVariant={highlightVariantInList}
        onEditSection={(slug, options) => navigateToSectionEditor(slug, options)}
      />

      <VariantListSummary
        activeCount={activeVariants.length}
        archivedCount={archivedVariants.length}
        attentionCount={attentionCandidateSlugs.size}
        totalWeight={totalAssignedWeight}
        weightStatus={weightStatus}
      />

        {analyticsError && (
          <div className="mb-6 rounded-xl border border-amber-500/20 bg-amber-500/10 px-4 py-3 flex items-center gap-3 text-sm text-amber-100">
            <AlertTriangle className="h-4 w-4 text-amber-300" />
            <span>Metrics snapshot unavailable right now. Variant cards show configuration only.</span>
          </div>
        )}

        {/* Active Variants */}
        <div className="mb-8" ref={variantListRef}>
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-xl font-semibold">Active Variants</h2>
            {(variantQuery || attentionOnly) && (
              <p className="text-xs text-slate-500">Showing {filteredActiveVariants.length} of {activeVariants.length}</p>
            )}
          </div>
          {activeVariants.length === 0 ? (
            <Card className="bg-white/5 border-white/10">
              <CardContent className="text-center py-12">
                <p className="text-slate-400 mb-4">No active variants yet</p>
                <Button onClick={() => navigate('/admin/customization/variants/new')}>
                  Create Your First Variant
                </Button>
              </CardContent>
            </Card>
          ) : filteredActiveVariants.length === 0 ? (
            <Card className="bg-white/5 border-white/10">
              <CardContent className="text-center py-12 space-y-3">
                <p className="text-slate-400">No variants match your filters.</p>
                <Button variant="outline" size="sm" onClick={clearVariantFilters} data-testid="clear-variant-filters">
                  Reset filters
                </Button>
              </CardContent>
            </Card>
          ) : (
            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredActiveVariants.map((variant) => (
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
                    <VariantStatusBadges
                      slug={variant.slug}
                      lastUpdatedLabel={formatVariantUpdatedLabel(variant.updated_at)}
                      attentionReasons={variantAttentionReasons.get(variant.slug) ?? []}
                    />
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

interface VariantFilterBarProps {
  query: string;
  onQueryChange: (value: string) => void;
  attentionOnly: boolean;
  onAttentionToggle: () => void;
  hasFilters: boolean;
  onClearFilters: () => void;
}

function VariantFilterBar({ query, onQueryChange, attentionOnly, onAttentionToggle, hasFilters, onClearFilters }: VariantFilterBarProps) {
  return (
    <div className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-4" data-testid="variant-filter-bar">
      <div className="flex flex-col gap-4 md:flex-row md:items-center">
        <label className="relative flex-1" aria-label="Search variants">
          <Search className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            type="text"
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Search by name or slug"
            className="w-full rounded-full border border-white/10 bg-slate-950/60 py-3 pl-11 pr-4 text-sm focus:border-blue-500 focus:outline-none"
            data-testid="variant-search-input"
          />
        </label>
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant={attentionOnly ? 'default' : 'outline'}
            onClick={onAttentionToggle}
            data-testid="variant-attention-filter"
          >
            Needs attention only
          </Button>
          {hasFilters && (
            <Button size="sm" variant="ghost" onClick={onClearFilters}>
              Clear filters
            </Button>
          )}
        </div>
      </div>
      <p className="mt-3 text-xs text-slate-500">
        Filters apply to the Active Variants grid. Needs attention surfaces variants that are never customized, stale for 10+ days, or currently the weakest conversion.
      </p>
    </div>
  );
}

interface VariantListSummaryProps {
  activeCount: number;
  archivedCount: number;
  attentionCount: number;
  totalWeight: number;
  weightStatus: WeightStatus;
}

function VariantListSummary({ activeCount, archivedCount, attentionCount, totalWeight, weightStatus }: VariantListSummaryProps) {
  const weightCopy = (() => {
    if (activeCount === 0) {
      return 'No active variants are routing traffic. Create one to light up the public landing.';
    }
    if (weightStatus === 'balanced') {
      return 'Traffic is fully allocated across variants.';
    }
    if (weightStatus === 'under') {
      return `${100 - totalWeight}% of visitors are idle because weights total less than 100%.`;
    }
    if (weightStatus === 'over') {
      return `Weights exceed 100% by ${totalWeight - 100}%. Adjust them to match your intent.`;
    }
    return 'Assign weights to control where visitors land.';
  })();

  return (
    <div className="mb-6 rounded-2xl border border-white/10 bg-white/5 p-4" data-testid="variant-list-summary">
      <div className="grid gap-4 text-center sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Active</p>
          <p className="text-2xl font-semibold text-white">{activeCount}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Attention</p>
          <p className="text-2xl font-semibold text-amber-300">{attentionCount}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Archived</p>
          <p className="text-2xl font-semibold text-slate-300">{archivedCount}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Traffic assigned</p>
          <p className="text-2xl font-semibold text-white">{Math.max(0, totalWeight).toFixed(0)}%</p>
        </div>
      </div>
      <p className="mt-4 text-xs text-slate-500 text-center">{weightCopy}</p>
    </div>
  );
}

interface ExperienceOpsPanelProps {
  totalWeight: number;
  variantCount: number;
  weightStatus: WeightStatus;
  staleVariants: { variant: Variant; daysSinceUpdate: number }[];
  neverUpdatedVariants: Variant[];
  underperforming: { stats: VariantStats; variant?: Variant } | null;
  analyticsRangeDays: number;
  onEditVariant: (slug: string) => void;
  onViewAnalytics: (slug: string) => void;
  onHighlightVariant?: (slug?: string) => void;
  onEditSection?: (slug: string, options?: { sectionId?: number; sectionType?: string }) => void;
}

function ExperienceOpsPanel({
  totalWeight,
  variantCount,
  weightStatus,
  staleVariants,
  neverUpdatedVariants,
  underperforming,
  analyticsRangeDays,
  onEditVariant,
  onViewAnalytics,
  onHighlightVariant,
  onEditSection,
}: ExperienceOpsPanelProps) {
  const progressWidth = Math.max(0, Math.min(totalWeight, 100));
  const remainder = Math.max(0, Math.abs(100 - totalWeight));
  const weightMessage = (() => {
    if (variantCount === 0) {
      return 'No active variants are routing traffic. Create one to render the public landing page.';
    }
    if (weightStatus === 'balanced') {
      return 'All visitor traffic is allocated. Keep experimenting but weights already sum to 100%.';
    }
    if (weightStatus === 'under') {
      return `Only ${totalWeight}% of visitors are assigned to active variants. Allocate the remaining ${remainder}% to avoid idle traffic.`;
    }
    return `${totalWeight}% of traffic is assigned, exceeding 100% by ${remainder}%. Adjust weights so the API can honor intended splits.`;
  })();
  const neverTouchedNames = neverUpdatedVariants.map((variant) => variant.name || variant.slug);
  const underperformingSlug = underperforming?.stats.variant_slug;
  const underperformingName = underperforming?.variant?.name ?? underperformingSlug;

  return (
    <Card className="mb-8 bg-white/5 border-white/10" data-testid="experience-ops-panel">
      <CardHeader>
        <CardTitle>Experiment Ops Snapshot</CardTitle>
        <CardDescription className="text-slate-400">
          Align traffic, freshness, and optimization actions so customization and analytics flows stay connected.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="space-y-3">
            <p className="text-sm font-semibold text-slate-200">Traffic Allocation</p>
            <div className="text-3xl font-bold">{Math.max(0, totalWeight).toFixed(0)}%</div>
            <div className="h-2 rounded-full bg-white/5 overflow-hidden">
              <div
                className={`h-2 ${weightStatus === 'balanced' ? 'bg-emerald-400' : 'bg-amber-400'} transition-all`}
                style={{ width: `${progressWidth}%` }}
              />
            </div>
            <p className="text-sm text-slate-400">{weightMessage}</p>
          </div>

          <div className="space-y-3">
            <p className="text-sm font-semibold text-slate-200">Content Freshness</p>
            {staleVariants.length === 0 && neverTouchedNames.length === 0 ? (
              <p className="text-sm text-slate-400">All active variants have been edited within the last {STALE_VARIANT_DAYS} days.</p>
            ) : (
              <div className="space-y-2">
                {staleVariants.map(({ variant, daysSinceUpdate }) => (
                  <div key={variant.slug} className="flex items-center justify-between bg-slate-900/60 border border-white/10 rounded-lg px-3 py-2">
                    <div>
                      <p className="text-sm font-medium text-white">{variant.name ?? variant.slug}</p>
                      <p className="text-xs text-slate-400">Updated {daysSinceUpdate} day{daysSinceUpdate === 1 ? '' : 's'} ago</p>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() =>
                        onEditSection
                          ? onEditSection(variant.slug, { sectionType: 'hero' })
                          : onEditVariant(variant.slug)
                      }
                    >
                      Refresh copy
                    </Button>
                  </div>
                ))}
                {neverTouchedNames.length > 0 && (
                  <div className="text-xs text-amber-200 bg-amber-500/10 border border-amber-500/20 rounded-lg px-3 py-2">
                    Never customized: {neverTouchedNames.join(', ')}
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="space-y-3">
            <p className="text-sm font-semibold text-slate-200">Needs Attention</p>
            {underperforming && underperformingSlug ? (
              <div className="rounded-xl border border-white/10 bg-slate-900/60 p-4 space-y-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Lowest conversion</p>
                  <p className="text-xl font-semibold text-white">{underperformingName}</p>
                  <p className="text-2xl font-bold text-rose-300">
                    {underperforming.stats.conversion_rate.toFixed(2)}%
                    <span className="text-xs text-slate-400 ml-2">last {analyticsRangeDays} day{analyticsRangeDays === 1 ? '' : 's'}</span>
                  </p>
                </div>
                <div className="flex flex-col gap-2">
                  <Button size="sm" className="gap-2" onClick={() => onViewAnalytics(underperformingSlug)}>
                    Inspect analytics
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() =>
                      onEditSection
                        ? onEditSection(underperformingSlug, { sectionType: 'hero' })
                        : onEditVariant(underperformingSlug)
                    }
                  >
                    Tune copy
                  </Button>
                  {onHighlightVariant && (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => onHighlightVariant(underperformingSlug)}
                      data-testid="needs-attention-focus"
                    >
                      Highlight in list
                    </Button>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-sm text-slate-400">Drive traffic to gather enough data before surfacing experiment insights.</p>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
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

function formatVariantUpdatedLabel(updatedAt?: string | null) {
  if (!updatedAt) {
    return 'Never customized';
  }
  const parsed = new Date(updatedAt);
  if (Number.isNaN(parsed.getTime())) {
    return 'Last updated date unavailable';
  }
  const diffMs = Date.now() - parsed.getTime();
  const diffDays = Math.max(0, Math.floor(diffMs / DAY_MS));
  if (diffDays === 0) {
    return 'Updated today';
  }
  if (diffDays === 1) {
    return 'Updated yesterday';
  }
  return `Updated ${diffDays} days ago`;
}

interface VariantStatusBadgesProps {
  slug: string;
  lastUpdatedLabel: string;
  attentionReasons: string[];
}

function VariantStatusBadges({ slug, lastUpdatedLabel, attentionReasons }: VariantStatusBadgesProps) {
  return (
    <div className="mb-4 flex flex-wrap gap-2 text-xs" data-testid={`variant-status-${slug}`}>
      <span className="rounded-full border border-white/10 bg-slate-900/60 px-3 py-1 text-slate-300">{lastUpdatedLabel}</span>
      {attentionReasons.map((reason) => (
        <span key={`${slug}-${reason}`} className="rounded-full border border-amber-500/30 bg-amber-500/10 px-3 py-1 text-amber-200">
          {reason}
        </span>
      ))}
    </div>
  );
}
