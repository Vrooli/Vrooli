import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Edit, Archive, Trash2, Sparkles, Eye } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/card';
import { Button } from '../components/ui/button';
import { listVariants, archiveVariant, deleteVariant, type Variant } from '../lib/api';

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

  useEffect(() => {
    fetchVariants();
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

  const activeVariants = variants.filter(v => v.status === 'active');
  const archivedVariants = variants.filter(v => v.status === 'archived');

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
