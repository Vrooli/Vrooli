import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Plus } from 'lucide-react';
import Editor, { type OnMount } from '@monaco-editor/react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import {
  type Variant,
  type ContentSection,
  type VariantSpace,
  type VariantAxes,
  type LandingHeaderConfig,
  type LandingHeaderNavLink,
  type LandingSection,
} from '../../../shared/api';
import {
  buildAxesSelection,
  hydrateFormFromVariant,
  loadVariantEditorData,
  loadVariantSpaceDefinition,
  loadVariantSnapshot,
  persistVariant,
  persistVariantSnapshot,
  sanitizeSlugInput,
  validateVariantForm,
  type VariantFormState,
  type VariantSnapshotPayload,
} from '../controllers/variantEditorController';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';
import { buildDefaultHeaderConfig, cloneHeaderConfig, normalizeHeaderConfig } from '../../../shared/lib/headerConfig';
import { DOWNLOAD_ANCHOR_ID, getSectionAnchorId } from '../../../shared/lib/sections';
import variantSchema from '../../../../../.vrooli/schemas/variant.schema.json';
import heroSchema from '../../../../../.vrooli/schemas/sections/hero.schema.json';
import featuresSchema from '../../../../../.vrooli/schemas/sections/features.schema.json';
import pricingSchema from '../../../../../.vrooli/schemas/sections/pricing.schema.json';
import ctaSchema from '../../../../../.vrooli/schemas/sections/cta.schema.json';
import testimonialsSchema from '../../../../../.vrooli/schemas/sections/testimonials.schema.json';
import faqSchema from '../../../../../.vrooli/schemas/sections/faq.schema.json';
import footerSchema from '../../../../../.vrooli/schemas/sections/footer.schema.json';
import videoSchema from '../../../../../.vrooli/schemas/sections/video.schema.json';

const variantSchemaUri = (variantSchema as { $id?: string }).$id ?? 'https://vrooli.dev/schemas/variant.json';
const sectionSchemaBase = 'https://vrooli.dev/schemas/sections/';
const monacoSchemaCatalog = [
  { uri: variantSchemaUri, schema: variantSchema },
  { uri: `${sectionSchemaBase}hero.schema.json`, schema: heroSchema },
  { uri: `${sectionSchemaBase}features.schema.json`, schema: featuresSchema },
  { uri: `${sectionSchemaBase}pricing.schema.json`, schema: pricingSchema },
  { uri: `${sectionSchemaBase}cta.schema.json`, schema: ctaSchema },
  { uri: `${sectionSchemaBase}testimonials.schema.json`, schema: testimonialsSchema },
  { uri: `${sectionSchemaBase}faq.schema.json`, schema: faqSchema },
  { uri: `${sectionSchemaBase}footer.schema.json`, schema: footerSchema },
  { uri: `${sectionSchemaBase}video.schema.json`, schema: videoSchema },
];

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
  const [validationError, setValidationError] = useState<string | null>(null);
  const [variantSpace, setVariantSpace] = useState<VariantSpace | null>(null);
  const [axesSelection, setAxesSelection] = useState<VariantAxes>({});
  const [axesSeeded, setAxesSeeded] = useState(false);
  const [form, setForm] = useState<VariantFormState>({
    name: '',
    slug: '',
    description: '',
    weight: 50,
  });
  const [headerConfig, setHeaderConfig] = useState<LandingHeaderConfig>(() => buildDefaultHeaderConfig(''));
  const [activeTab, setActiveTab] = useState<'form' | 'json'>('form');
  const [snapshotDraft, setSnapshotDraft] = useState('');
  const [snapshotError, setSnapshotError] = useState<string | null>(null);
  const [snapshotLoading, setSnapshotLoading] = useState(!isNew);
  const [snapshotSaving, setSnapshotSaving] = useState(false);
  const isJsonTab = activeTab === 'json';
  const currentSaving = isJsonTab ? snapshotSaving : saving;
  const savingLabel = isJsonTab
    ? snapshotSaving
      ? 'Saving JSON...'
      : 'Save JSON'
    : saving
    ? 'Saving...'
    : 'Save';
  const editorModelPath = 'inmemory://model/landing-variant.json';

  const handleEditorMount: OnMount = (_editor, monaco) => {
    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      allowComments: false,
      schemas: monacoSchemaCatalog.map(({ uri, schema }) => ({
        uri,
        fileMatch: uri === variantSchemaUri ? ['*landing-variant*'] : [uri],
        schema,
      })),
    });
  };

  const applyAxesSelection = (space: VariantSpace, existing?: VariantAxes) => {
    setAxesSelection(buildAxesSelection(space, existing));
  };

  const updateFormField = <K extends keyof VariantFormState>(field: K, value: VariantFormState[K]) => {
    if (validationError) {
      setValidationError(null);
    }
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

  const fetchSnapshot = async () => {
    if (!slug || isNew) return;
    try {
      setSnapshotLoading(true);
      const snapshot = await loadVariantSnapshot(slug);
      setSnapshotDraft(JSON.stringify(snapshot, null, 2));
      setSnapshotError(null);
    } catch (err) {
      console.error('Variant snapshot fetch error:', err);
      setSnapshotError(err instanceof Error ? err.message : 'Failed to load variant JSON');
    } finally {
      setSnapshotLoading(false);
    }
  };

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
      setHeaderConfig(normalizeHeaderConfig(data.variant.header_config, data.variant.name));
      rememberVariantSession({
        slug: data.variant.slug,
        name: data.variant.name,
        surface: 'variant',
      });
      await fetchSnapshot();
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load variant');
      console.error('Variant fetch error:', err);
      setSnapshotLoading(false);
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
      setValidationError(validationMessage);
      return;
    }

    try {
      setSaving(true);
      setValidationError(null);

      const saved = await persistVariant({
        isNew,
        slugFromRoute: slug,
        form,
        axesSelection,
        headerConfig,
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

  const handleSaveJson = async () => {
    if (!slug) {
      setSnapshotError('Variant slug missing');
      return;
    }
    try {
      setSnapshotSaving(true);
      setSnapshotError(null);

      const parsed = JSON.parse(snapshotDraft) as VariantSnapshotPayload;
      const saved = await persistVariantSnapshot(slug, parsed);
      setSnapshotDraft(JSON.stringify(saved, null, 2));
      await fetchVariant();
    } catch (err) {
      if (err instanceof SyntaxError) {
        setSnapshotError(`Invalid JSON: ${err.message}`);
      } else {
        setSnapshotError(err instanceof Error ? err.message : 'Failed to save variant JSON');
      }
    } finally {
      setSnapshotSaving(false);
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
            onClick={isJsonTab ? handleSaveJson : handleSave}
            disabled={currentSaving || (isJsonTab && (snapshotLoading || isNew))}
            className="gap-2"
            data-testid="save-variant"
          >
            <Save className="h-4 w-4" />
            {savingLabel}
          </Button>
        </div>

        <div className="flex items-center gap-3 mb-4">
          <Button
            variant={!isJsonTab ? 'default' : 'outline'}
            size="sm"
            onClick={() => setActiveTab('form')}
          >
            Form Editor
          </Button>
          <Button
            variant={isJsonTab ? 'default' : 'outline'}
            size="sm"
            onClick={() => setActiveTab('json')}
            disabled={isNew}
            title={isNew ? 'Save the variant before using JSON editor' : 'Edit the raw variant JSON'}
          >
            JSON Editor
          </Button>
          {isNew && (
            <span className="text-xs text-slate-500">
              Save the variant once before switching to JSON
            </span>
          )}
        </div>

        {error && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 mb-6">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {validationError && (
          <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 mb-6" data-testid="variant-validation-error">
            <p className="text-amber-300 text-sm">{validationError}</p>
          </div>
        )}

        {isJsonTab ? (
          <Card className="bg-white/5 border-white/10">
            <CardHeader>
              <CardTitle>Variant JSON</CardTitle>
              <CardDescription className="text-slate-400">
                Edit the entire variant (metadata + sections) as a single JSON payload. Applies in one transaction.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {snapshotLoading ? (
                <p className="text-slate-400">Loading variant JSON...</p>
              ) : (
                <>
                  <div className="border border-white/10 rounded-lg overflow-hidden" data-testid="variant-json-editor">
                    <Editor
                      language="json"
                      theme="vs-dark"
                      height="520px"
                      path={editorModelPath}
                      value={snapshotDraft}
                      onChange={(value) => setSnapshotDraft(value ?? '')}
                      onMount={handleEditorMount}
                      options={{
                        minimap: { enabled: false },
                        fontSize: 13,
                        wordWrap: 'on',
                        lineNumbers: 'on',
                        scrollBeyondLastLine: false,
                        automaticLayout: true,
                      }}
                    />
                  </div>
                  <p className="text-xs text-slate-500">
                    Must include <code>variant</code> and <code>sections</code>. The <code>variant.slug</code> must match this page.
                  </p>
                  {snapshotError && (
                    <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
                      {snapshotError}
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        ) : (
          <>
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
                              onChange={(e) => {
                                if (validationError) {
                                  setValidationError(null);
                                }
                                setAxesSelection((prev) => ({
                                  ...prev,
                                  [axisId]: e.target.value,
                                }));
                              }}
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

            <HeaderConfigurator
              config={headerConfig}
              sections={sections}
              onChange={setHeaderConfig}
              variantName={form.name || variant?.name || ''}
            />

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
          </>
        )}
      </div>
    </AdminLayout>
  );
}

interface HeaderConfiguratorProps {
  config: LandingHeaderConfig;
  sections: ContentSection[];
  onChange: React.Dispatch<React.SetStateAction<LandingHeaderConfig>>;
  variantName: string;
}

function HeaderConfigurator({ config, sections, onChange, variantName }: HeaderConfiguratorProps) {
  const [navTarget, setNavTarget] = useState('');
  const downloadsSection = sections.some((section) => section.section_type === 'downloads');

  const updateConfig = (updater: (draft: LandingHeaderConfig) => void) => {
    onChange((prev) => {
      const next = cloneHeaderConfig(prev);
      updater(next);
      return next;
    });
  };

  const handleAddLink = () => {
    if (!navTarget) {
      return;
    }
    try {
      const parsed = JSON.parse(navTarget) as { type: string; id?: number; section_type?: string; order?: number };
      if (parsed.type === 'downloads') {
        updateConfig((draft) => {
          draft.nav.links.push({
            id: generateNavLinkId('downloads'),
            type: 'downloads',
            label: 'Downloads',
            anchor: DOWNLOAD_ANCHOR_ID,
            visible_on: { desktop: true, mobile: true },
          });
        });
      } else if (parsed.type === 'section') {
        const targetSection = sections.find((section) => {
          if (typeof parsed.id === 'number' && section.id) {
            return section.id === parsed.id;
          }
          return section.section_type === parsed.section_type && section.order === parsed.order;
        });
        if (targetSection) {
          updateConfig((draft) => {
            draft.nav.links.push(createNavLinkFromSection(targetSection));
          });
        }
      }
      setNavTarget('');
    } catch (err) {
      console.warn('Invalid nav target', err);
    }
  };

  const handleNavLabelChange = (index: number, value: string) => {
    updateConfig((draft) => {
      draft.nav.links[index].label = value;
    });
  };

  const handleMenuChildChange = (
    linkIndex: number,
    childIndex: number,
    field: 'label' | 'href',
    value: string,
  ) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu') return;
      if (!Array.isArray(link.children)) {
        link.children = [];
      }
      if (!link.children[childIndex]) {
        link.children[childIndex] = {
          id: generateNavLinkId('child'),
          type: 'custom',
          label: '',
          href: '',
          visible_on: { desktop: true, mobile: true },
        };
      }
      if (field === 'label') {
        link.children[childIndex].label = value;
      } else {
        link.children[childIndex].href = value;
      }
    });
  };

  const handleAddMenuChild = (linkIndex: number) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu') return;
      if (!Array.isArray(link.children)) {
        link.children = [];
      }
      link.children.push({
        id: generateNavLinkId('child'),
        type: 'custom',
        label: 'Menu item',
        href: '#',
        visible_on: { desktop: true, mobile: true },
      });
    });
  };

  const handleRemoveMenuChild = (linkIndex: number, childIndex: number) => {
    updateConfig((draft) => {
      const link = draft.nav.links[linkIndex];
      if (!link || link.type !== 'menu' || !Array.isArray(link.children)) return;
      link.children.splice(childIndex, 1);
    });
  };

  const handleVisibilityToggle = (index: number, key: 'desktop' | 'mobile', value: boolean) => {
    updateConfig((draft) => {
      draft.nav.links[index].visible_on = {
        desktop: key === 'desktop' ? value : draft.nav.links[index].visible_on?.desktop ?? true,
        mobile: key === 'mobile' ? value : draft.nav.links[index].visible_on?.mobile ?? true,
      };
    });
  };

  const handleRemoveLink = (index: number) => {
    updateConfig((draft) => {
      draft.nav.links.splice(index, 1);
    });
  };

  const handleAddMenu = () => {
    updateConfig((draft) => {
      draft.nav.links.push({
        id: generateNavLinkId('menu'),
        type: 'menu',
        label: 'Menu',
        visible_on: { desktop: true, mobile: true },
        children: [
          {
            id: generateNavLinkId('menu-item'),
            type: 'custom',
            label: 'First link',
            href: '#',
            visible_on: { desktop: true, mobile: true },
          },
          {
            id: generateNavLinkId('menu-item'),
            type: 'custom',
            label: 'Second link',
            href: '#',
            visible_on: { desktop: true, mobile: true },
          },
        ],
      });
    });
  };

  const handleMoveLink = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= config.nav.links.length) {
      return;
    }
    updateConfig((draft) => {
      const [link] = draft.nav.links.splice(index, 1);
      draft.nav.links.splice(nextIndex, 0, link);
    });
  };

  const handleCTAModeChange = (
    target: 'primary' | 'secondary',
    updates: { mode?: string; label?: string; href?: string; variant?: 'solid' | 'ghost' },
  ) => {
    updateConfig((draft) => {
      draft.ctas[target] = {
        ...draft.ctas[target],
        ...updates,
      };
    });
  };

  return (
    <Card className="bg-white/5 border-white/10 mb-6">
      <CardHeader>
        <CardTitle>Header Presentation</CardTitle>
        <CardDescription className="text-slate-400">Branding, navigation, and CTA controls</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-2">
          <h3 className="text-slate-200 font-medium">Branding</h3>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Display mode</label>
              <select
                value={config.branding?.mode ?? 'logo_and_name'}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.mode = e.target.value as LandingHeaderConfig['branding']['mode'];
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="logo_and_name">Logo + Name</option>
                <option value="logo">Logo only</option>
                <option value="name">Name only</option>
                <option value="none">Minimal</option>
              </select>
            </div>
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Mobile emphasis</label>
              <select
                value={config.branding?.mobile_preference ?? 'auto'}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.mobile_preference = e.target.value as LandingHeaderConfig['branding']['mobile_preference'];
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="auto">Show both</option>
                <option value="logo">Logo on mobile</option>
                <option value="name">Name on mobile</option>
                <option value="stacked">Stacked</option>
              </select>
            </div>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Brand label</label>
              <input
                type="text"
                value={config.branding?.label ?? variantName}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.label = e.target.value;
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
                placeholder="Header title"
              />
            </div>
            <div>
              <label className="text-sm text-slate-300 mb-1 block">Subtitle</label>
              <input
                type="text"
                value={config.branding?.subtitle ?? ''}
                onChange={(e) =>
                  updateConfig((draft) => {
                    draft.branding.subtitle = e.target.value;
                  })
                }
                className="w-full bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
                placeholder="Optional tagline"
              />
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-slate-200 font-medium">Navigation Links</h3>
            <div className="flex gap-2">
              <select
                value={navTarget}
                onChange={(e) => setNavTarget(e.target.value)}
                className="bg-slate-900/50 border border-slate-800 rounded-lg px-3 py-2 text-white"
              >
                <option value="">Select target</option>
                {sections.map((section, index) => (
                  <option
                    key={`${section.section_type}-${section.id ?? index}`}
                    value={JSON.stringify({
                      type: 'section',
                      id: section.id ?? null,
                      section_type: section.section_type,
                      order: section.order,
                    })}
                  >
                    Section · {section.section_type} #{section.order}
                  </option>
                ))}
                <option
                  value={JSON.stringify({ type: 'downloads' })}
                  disabled={!downloadsSection}
                >
                  Downloads anchor
                </option>
              </select>
              <Button variant="secondary" size="sm" onClick={handleAddLink}>
                Add link
              </Button>
              <Button variant="outline" size="sm" onClick={handleAddMenu}>
                Add menu
              </Button>
            </div>
          </div>
          {config.nav.links.length === 0 ? (
            <p className="text-sm text-slate-400">
              No manual links added. The header will mirror section order automatically.
            </p>
          ) : (
            <div className="space-y-3">
              {config.nav.links.map((link, index) => (
                <div key={link.id} className="rounded-lg border border-white/5 bg-slate-900/40 p-3 space-y-2">
                  <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                    <div className="flex-1">
                      <label className="text-xs text-slate-400 block mb-1">Label</label>
                      <input
                        type="text"
                        value={link.label}
                        onChange={(e) => handleNavLabelChange(index, e.target.value)}
                        className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-2 text-white"
                      />
                    </div>
                    <div className="flex items-center gap-3 text-xs text-slate-400">
                      <label className="flex items-center gap-1">
                        <input
                          type="checkbox"
                          checked={link.visible_on?.desktop ?? true}
                          onChange={(e) => handleVisibilityToggle(index, 'desktop', e.target.checked)}
                        />
                        Desktop
                      </label>
                      <label className="flex items-center gap-1">
                        <input
                          type="checkbox"
                          checked={link.visible_on?.mobile ?? true}
                          onChange={(e) => handleVisibilityToggle(index, 'mobile', e.target.checked)}
                        />
                        Mobile
                      </label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button variant="ghost" size="icon" onClick={() => handleMoveLink(index, -1)} disabled={index === 0}>
                        ↑
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleMoveLink(index, 1)}
                        disabled={index === config.nav.links.length - 1}
                      >
                        ↓
                      </Button>
                      <Button variant="destructive" size="icon" onClick={() => handleRemoveLink(index)}>
                        ×
                      </Button>
                    </div>
                  </div>
                  <p className="text-xs text-slate-500">
                    {link.type === 'downloads'
                      ? 'Downloads anchor'
                      : link.type === 'menu'
                        ? 'Dropdown menu'
                        : `Link to ${link.section_type ?? 'custom target'}`}
                  </p>
                  {link.type === 'menu' && (
                    <div className="space-y-2 rounded-md border border-white/10 bg-slate-900/60 p-3">
                      <div className="flex items-center justify-between text-xs text-slate-400">
                        <span>Menu items</span>
                        <Button size="sm" variant="secondary" onClick={() => handleAddMenuChild(index)}>
                          Add item
                        </Button>
                      </div>
                      {(!link.children || link.children.length === 0) && (
                        <p className="text-xs text-slate-500">No items yet.</p>
                      )}
                      {link.children?.map((child, childIndex) => (
                        <div key={child.id} className="flex flex-col gap-2 rounded border border-white/5 bg-slate-900/50 p-2 md:flex-row md:items-center md:gap-3">
                          <div className="flex-1">
                            <label className="text-[11px] text-slate-400 block mb-1">Item label</label>
                            <input
                              type="text"
                              value={child.label}
                              onChange={(e) => handleMenuChildChange(index, childIndex, 'label', e.target.value)}
                              className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-1.5 text-white"
                            />
                          </div>
                          <div className="flex-1">
                            <label className="text-[11px] text-slate-400 block mb-1">URL or anchor</label>
                            <input
                              type="text"
                              value={child.href ?? ''}
                              onChange={(e) => handleMenuChildChange(index, childIndex, 'href', e.target.value)}
                              className="w-full bg-slate-900/60 border border-slate-800 rounded px-3 py-1.5 text-white"
                            />
                          </div>
                          <Button variant="destructive" size="sm" onClick={() => handleRemoveMenuChild(index, childIndex)}>
                            Remove
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <h3 className="text-slate-200 font-medium">Primary CTA</h3>
            <select
              value={config.ctas.primary.mode ?? 'inherit_hero'}
              onChange={(e) => handleCTAModeChange('primary', { mode: e.target.value })}
              className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
            >
              <option value="inherit_hero">Use hero CTA</option>
              <option value="downloads">Downloads anchor</option>
              <option value="custom">Custom link</option>
              <option value="hidden">Hidden</option>
            </select>
            {config.ctas.primary.mode === 'custom' && (
              <div className="space-y-2">
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="Button label"
                  value={config.ctas.primary.label ?? ''}
                  onChange={(e) => handleCTAModeChange('primary', { label: e.target.value })}
                />
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="https://example.com"
                  value={config.ctas.primary.href ?? ''}
                  onChange={(e) => handleCTAModeChange('primary', { href: e.target.value })}
                />
              </div>
            )}
          </div>
          <div className="space-y-2">
            <h3 className="text-slate-200 font-medium">Secondary CTA</h3>
            <select
              value={config.ctas.secondary.mode ?? 'downloads'}
              onChange={(e) => handleCTAModeChange('secondary', { mode: e.target.value })}
              className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
            >
              <option value="downloads">Downloads anchor</option>
              <option value="inherit_hero">Use hero CTA</option>
              <option value="custom">Custom link</option>
              <option value="hidden">Hidden</option>
            </select>
            {(config.ctas.secondary.mode === 'custom' || config.ctas.secondary.mode === 'downloads') && (
              <div className="space-y-2">
                <input
                  type="text"
                  className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                  placeholder="Button label"
                  value={config.ctas.secondary.label ?? ''}
                  onChange={(e) => handleCTAModeChange('secondary', { label: e.target.value })}
                />
                {config.ctas.secondary.mode === 'custom' && (
                  <input
                    type="text"
                    className="w-full bg-slate-900/50 border border-slate-800 rounded px-3 py-2 text-white"
                    placeholder="https://example.com"
                    value={config.ctas.secondary.href ?? ''}
                    onChange={(e) => handleCTAModeChange('secondary', { href: e.target.value })}
                  />
                )}
              </div>
            )}
          </div>
        </div>

        <div className="space-y-2">
          <h3 className="text-slate-200 font-medium">Behavior</h3>
          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={config.behavior.sticky}
              onChange={(e) =>
                updateConfig((draft) => {
                  draft.behavior.sticky = e.target.checked;
                  if (!e.target.checked) {
                    draft.behavior.hide_on_scroll = false;
                  }
                })
              }
            />
            Sticky header
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-300">
            <input
              type="checkbox"
              checked={config.behavior.hide_on_scroll}
              disabled={!config.behavior.sticky}
              onChange={(e) =>
                updateConfig((draft) => {
                  draft.behavior.hide_on_scroll = e.target.checked;
                })
              }
            />
            Hide on downward scroll
          </label>
        </div>
      </CardContent>
    </Card>
  );
}

function createNavLinkFromSection(section: ContentSection): LandingHeaderNavLink {
  const anchorSection = {
    id: section.id,
    section_type: section.section_type,
    order: section.order,
    content: section.content,
  } as LandingSection;

  return {
    id: generateNavLinkId(section.section_type),
    type: 'section',
    label: section.section_type.replace(/_/g, ' '),
    section_type: section.section_type,
    section_id: section.id ?? undefined,
    anchor: getSectionAnchorId(anchorSection),
    visible_on: { desktop: true, mobile: true },
  };
}

function generateNavLinkId(prefix: string) {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    try {
      return crypto.randomUUID();
    } catch {
      // ignore
    }
  }
  return `${prefix}-${Date.now()}-${Math.round(Math.random() * 1000)}`;
}
