import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Plus } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { type Variant, type ContentSection, type VariantSpace, type VariantAxes } from '../../../shared/api';
import {
  buildAxesSelection,
  hydrateFormFromVariant,
  loadVariantEditorData,
  loadVariantSpaceDefinition,
  persistVariant,
  sanitizeSlugInput,
  validateVariantForm,
  type VariantFormState,
} from '../controllers/variantEditorController';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';

/**
 * Variant Editor - Create or edit a variant and its sections
 * Implements OT-P0-017 (Variant CRUD operations)
 *
 * [REQ:VARIANT-MGMT]
 */
export function VariantEditor() {
  const navigate = useNavigate();
  const { slug } = useParams<{ slug: string }>();
  const isNew = slug === 'new';

  const [variant, setVariant] = useState<Variant | null>(null);
  const [sections, setSections] = useState<ContentSection[]>([]);
  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [variantSpace, setVariantSpace] = useState<VariantSpace | null>(null);
  const [axesSelection, setAxesSelection] = useState<VariantAxes>({});
  const [axesSeeded, setAxesSeeded] = useState(false);
  const [form, setForm] = useState<VariantFormState>({
    name: '',
    slug: '',
    description: '',
    weight: 50,
  });

  const applyAxesSelection = (space: VariantSpace, existing?: VariantAxes) => {
    setAxesSelection(buildAxesSelection(space, existing));
  };

  const updateFormField = <K extends keyof VariantFormState>(field: K, value: VariantFormState[K]) => {
    setForm((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  useEffect(() => {
    if (!isNew && slug) {
      fetchVariant();
    }
  }, [slug]);

  useEffect(() => {
    const fetchVariantSpaceData = async () => {
      try {
        const space = await loadVariantSpaceDefinition();
        setVariantSpace(space);
      } catch (err) {
        console.error('Variant space fetch error:', err);
        setError(err instanceof Error ? err.message : 'Failed to load variant axes');
      }
    };
    fetchVariantSpaceData();
  }, []);

  useEffect(() => {
    if (!variantSpace || axesSeeded) {
      return;
    }

    if (isNew) {
      applyAxesSelection(variantSpace);
      setAxesSeeded(true);
      return;
    }

    if (variant?.axes) {
      applyAxesSelection(variantSpace, variant.axes);
      setAxesSeeded(true);
    }
  }, [variantSpace, variant, isNew, axesSeeded]);

  const fetchVariant = async () => {
    if (!slug) return;

    try {
      setLoading(true);
      const data = await loadVariantEditorData(slug);
      setAxesSeeded(false);
      setVariant(data.variant);
      setForm(hydrateFormFromVariant(data.variant));
      setAxesSelection(data.variant.axes || {});
      setSections(data.sections);
      rememberVariantSession({
        slug: data.variant.slug,
        name: data.variant.name,
        surface: 'variant',
      });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load variant');
      console.error('Variant fetch error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    const validationMessage = validateVariantForm({
      form,
      variantSpace,
      axesSelection,
      requireSlug: isNew,
    });

    if (validationMessage) {
      alert(validationMessage);
      return;
    }

    try {
      setSaving(true);

      const saved = await persistVariant({
        isNew,
        slugFromRoute: slug,
        form,
        axesSelection,
      });

      if (saved && isNew) {
        rememberVariantSession({
          slug: saved.slug,
          name: saved.name,
          surface: 'variant',
        });
        navigate(`/admin/customization/variants/${saved.slug}`);
      } else if (slug) {
        await fetchVariant();
      }

      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save variant');
      console.error('Variant save error:', err);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <AdminLayout>
        <div className="flex items-center justify-center min-h-[400px]">
          <p className="text-slate-400">Loading variant...</p>
        </div>
      </AdminLayout>
    );
  }

  return (
    <AdminLayout>
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4 mb-8">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/admin/customization')}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-semibold">{isNew ? 'New Variant' : 'Edit Variant'}</h1>
            <p className="text-slate-400 mt-1">
              {isNew ? 'Create a new A/B test variant' : `Editing ${variant?.name ?? slug}`}
            </p>
          </div>
          <Button
            onClick={handleSave}
            disabled={saving}
            className="gap-2"
            data-testid="save-variant"
          >
            <Save className="h-4 w-4" />
            {saving ? 'Saving...' : 'Save'}
          </Button>
        </div>

        {error && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 mb-6">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {/* Variant Settings */}
        <Card className="bg-white/5 border-white/10 mb-6">
          <CardHeader>
            <CardTitle>Variant Settings</CardTitle>
            <CardDescription className="text-slate-400">
              Basic information and A/B testing weight
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label htmlFor="name" className="block text-sm font-medium text-slate-300 mb-2">
                Name
              </label>
              <input
                id="name"
                type="text"
                value={form.name}
                onChange={(e) => updateFormField('name', e.target.value)}
                className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                placeholder="e.g., Variant A"
                data-testid="variant-name-input"
              />
            </div>

            <div>
              <label htmlFor="slug" className="block text-sm font-medium text-slate-300 mb-2">
                Slug {!isNew && <span className="text-slate-500">(cannot be changed)</span>}
              </label>
              <input
                id="slug"
                type="text"
                value={form.slug}
                onChange={(e) => updateFormField('slug', sanitizeSlugInput(e.target.value))}
                disabled={!isNew}
                className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed"
                placeholder="e.g., variant-a"
                data-testid="variant-slug-input"
              />
              <p className="text-xs text-slate-500 mt-1">
                Used in URLs: /?variant={form.slug || 'slug'}
              </p>
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-slate-300 mb-2">
                Description (optional)
              </label>
              <textarea
                id="description"
                value={form.description}
                onChange={(e) => updateFormField('description', e.target.value)}
                className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                rows={3}
                placeholder="Brief description of this variant's purpose"
                data-testid="variant-description-input"
              />
            </div>

            {variantSpace && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="text-sm font-medium text-slate-300">Variant Axes</label>
                  <span className="text-xs text-slate-500">Persona • Jobs-to-be-done • Conversion style</span>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  {Object.entries(variantSpace.axes).map(([axisId, axisDef]) => {
                    const selectedValue = axesSelection[axisId] || '';
                    const selectedVariant = axisDef.variants.find((v) => v.id === selectedValue);
                    return (
                      <div key={axisId} className="bg-slate-900/60 border border-white/10 rounded-lg p-4">
                        <label className="block text-sm font-medium text-slate-200 mb-2 capitalize">
                          {axisId}
                        </label>
                        <select
                          className="w-full px-3 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                          value={selectedValue}
                          onChange={(e) =>
                            setAxesSelection((prev) => ({
                              ...prev,
                              [axisId]: e.target.value,
                            }))
                          }
                        >
                          {axisDef.variants.map((axisVariant) => (
                            <option key={axisVariant.id} value={axisVariant.id}>
                              {axisVariant.label}
                            </option>
                          ))}
                        </select>
                        {(selectedVariant?.description || axisDef._note) && (
                          <p className="text-xs text-slate-400 mt-2">
                            {selectedVariant?.description || axisDef._note}
                          </p>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            <div>
              <label htmlFor="weight" className="block text-sm font-medium text-slate-300 mb-2">
                A/B Testing Weight: {form.weight}%
              </label>
              <input
                id="weight"
                type="range"
                min="0"
                max="100"
                value={form.weight}
                onChange={(e) => updateFormField('weight', parseInt(e.target.value, 10))}
                className="w-full"
                data-testid="variant-weight-input"
              />
              <p className="text-xs text-slate-500 mt-1">
                Higher weight = more traffic. Total weight across all active variants should equal 100%.
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Content Sections */}
        {!isNew && variant && (
          <Card className="bg-white/5 border-white/10">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Content Sections</CardTitle>
                  <CardDescription className="text-slate-400">
                    Customize landing page sections with live preview
                  </CardDescription>
                </div>
                <Button
                  onClick={() => navigate(`/admin/customization/variants/${slug}/sections/new`)}
                  variant="outline"
                  size="sm"
                  className="gap-2"
                  data-testid="add-section"
                >
                  <Plus className="h-4 w-4" />
                  Add Section
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {sections.length === 0 ? (
                <div className="text-center py-12 text-slate-400">
                  <p className="mb-4">No sections yet</p>
                  <Button
                    onClick={() => navigate(`/admin/customization/variants/${slug}/sections/new`)}
                    variant="outline"
                  >
                    Add Your First Section
                  </Button>
                </div>
              ) : (
                <div className="space-y-3">
                  {sections
                    .sort((a, b) => a.order - b.order)
                    .map((section) => (
                      <div
                        key={section.id}
                        className="flex items-center justify-between p-4 bg-slate-900/50 rounded-lg border border-white/10 hover:border-white/20 transition-colors"
                        data-testid={`section-${section.id}`}
                      >
                        <div className="flex items-center gap-4">
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">#{section.order}</span>
                            <span className={`px-2 py-1 rounded text-xs ${
                              section.enabled ? 'bg-green-500/20 text-green-400' : 'bg-slate-500/20 text-slate-400'
                            }`}>
                              {section.enabled ? 'Enabled' : 'Disabled'}
                            </span>
                          </div>
                          <div>
                            <div className="font-medium capitalize">{section.section_type}</div>
                            <div className="text-xs text-slate-500">
                              Last updated {new Date(section.updated_at).toLocaleDateString()}
                            </div>
                          </div>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => navigate(`/admin/customization/variants/${slug}/sections/${section.id}`)}
                          data-testid={`edit-section-${section.id}`}
                        >
                          Edit
                        </Button>
                      </div>
                    ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </AdminLayout>
  );
}
