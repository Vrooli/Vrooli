import { useNavigate, useParams } from 'react-router-dom';
import { Save, ArrowLeft, Plus, Clipboard, Edit } from 'lucide-react';
import Editor from '@monaco-editor/react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { RuntimeSignalStrip } from '../components/RuntimeSignalStrip';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Textarea } from '../../../shared/ui/input';
import { useToast } from '../../../shared/ui/useToast';
import { HeaderConfigurator } from '../components/HeaderConfigurator';
import { useVariantForm } from '../hooks/useVariantForm';
import { sanitizeSlugInput } from '../controllers/variantEditorController';
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
const editorModelPath = 'inmemory://model/landing-variant.json';

/**
 * Variant Editor - Create or edit a variant and its sections
 * Implements OT-P0-017 (Variant CRUD operations)
 *
 * [REQ:VARIANT-MGMT]
 */
export function VariantEditor() {
  const navigate = useNavigate();
  const { slug } = useParams<{ slug: string }>();
  const routeSlug = slug || '';
  const isNew = slug === 'new';
  const toast = useToast();

  const {
    // Data state
    variant,
    sections,
    loading,
    error,
    validationError,

    // Variant space
    variantSpace,
    axesSelection,
    updateAxesSelection,

    // Form state
    form,
    updateFormField,

    // Header config
    headerConfig,
    setHeaderConfig,

    // Tab state
    setActiveTab,
    isJsonTab,
    currentSaving,
    savingLabel,

    // Snapshot state
    snapshotDraft,
    setSnapshotDraft,
    snapshotError,
    snapshotLoading,
    schemaIssues,
    copyStatus,

    // Actions
    handleSave,
    handleSaveJson,
    handleEditorMount,
    handleCopyIssues,
    handleCopySchema,
  } = useVariantForm({
    slug,
    isNew,
    monacoSchemaCatalog,
    variantSchemaUri,
    editorModelPath,
    onSuccess: (message, title) => toast.success(message, title),
    onError: (message) => toast.error(message),
  });

  const handleSaveClick = async () => {
    if (isJsonTab) {
      await handleSaveJson();
    } else {
      const result = await handleSave();
      if (result.success && result.savedVariant && isNew) {
        navigate(`/admin/customization/variants/${result.savedVariant.slug}`);
      }
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
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <RuntimeSignalStrip mode="compact" />

        <PageHeader
          title={isNew ? 'New Variant' : 'Edit Variant'}
          description={isNew ? 'Create a new A/B test variant' : `Editing ${variant?.name || routeSlug || 'variant'}`}
          icon={Edit}
          iconBgClass="bg-indigo-500/10"
          iconColorClass="text-indigo-400"
          testId="variant-editor-header"
          actions={
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { navigate('/admin/customization'); }}
                className="gap-2"
              >
                <ArrowLeft className="h-4 w-4" />
                Back
              </Button>
              <Button
                onClick={() => { void handleSaveClick(); }}
                disabled={currentSaving || (isJsonTab && (snapshotLoading || isNew))}
                className="gap-2"
                data-testid="save-variant"
              >
                <Save className="h-4 w-4" />
                {savingLabel}
              </Button>
            </>
          }
        />

        {/* Tab Switcher */}
        <div className="flex items-center gap-3 mb-4">
          <Button
            variant={!isJsonTab ? 'default' : 'outline'}
            size="sm"
            onClick={() => { setActiveTab('form'); }}
          >
            Form Editor
          </Button>
          <Button
            variant={isJsonTab ? 'default' : 'outline'}
            size="sm"
            onClick={() => { setActiveTab('json'); }}
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

        {/* Error Display */}
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
          <Card className="${LAYOUT.card.base}">
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
                      onChange={(value) => { setSnapshotDraft(value ?? ''); }}
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
                  <div className="flex items-center gap-3 text-xs text-slate-400 flex-wrap">
                    <Button
                      variant="outline"
                      size="sm"
                      className="gap-2"
                      onClick={() => { void handleCopySchema(variantSchema); }}
                    >
                      <Clipboard className="h-4 w-4" />
                      Copy variant schema
                    </Button>
                    {copyStatus && <span>{copyStatus}</span>}
                  </div>
                  {schemaIssues.length > 0 && (
                    <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100 space-y-2" data-testid="variant-json-schema-issues">
                      <div className="flex items-center justify-between">
                        <div className="font-semibold text-amber-200">Schema validation issues</div>
                        <div className="flex items-center gap-2">
                          <Button size="sm" variant="outline" onClick={() => { void handleCopyIssues(); }}>
                            Copy issues
                          </Button>
                          {copyStatus && <span className="text-[11px] text-slate-200">{copyStatus}</span>}
                        </div>
                      </div>
                      <ul className="list-disc list-inside space-y-1">
                        {schemaIssues.map((issue, idx) => (
                          <li key={idx}>{issue}</li>
                        ))}
                      </ul>
                    </div>
                  )}
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
            <Card className="${LAYOUT.card.base} mb-6">
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
                    onChange={(e) => { updateFormField('name', e.target.value); }}
                    className="w-full px-4 py-2 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none focus:ring-0"
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
                    onChange={(e) => { updateFormField('slug', sanitizeSlugInput(e.target.value)); }}
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
                  <Textarea
                    id="description"
                    value={form.description}
                    onChange={(e) => { updateFormField('description', e.target.value); }}
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
                      <span className="text-xs text-slate-500">Persona · Jobs-to-be-done · Conversion style</span>
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
                              onChange={(e) => { updateAxesSelection(axisId, e.target.value); }}
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
                    onChange={(e) => { updateFormField('weight', parseInt(e.target.value, 10)); }}
                    className="w-full"
                    data-testid="variant-weight-input"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    Higher weight = more traffic. Weights are proportional; 0% disables a variant. If all active variants are 0, traffic splits evenly.
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
              <Card className={LAYOUT.card.base}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle>Content Sections</CardTitle>
                      <CardDescription className="text-slate-400">
                        Customize landing page sections with live preview
                      </CardDescription>
                    </div>
                    <Button
                      onClick={() => { navigate(`/admin/customization/variants/${routeSlug}/sections/new`); }}
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
                        onClick={() => { navigate(`/admin/customization/variants/${routeSlug}/sections/new`); }}
                        variant="outline"
                      >
                        Add Your First Section
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {[...sections]
                        .sort((a, b) => a.order - b.order)
                        .map((section) => (
                          <div
                            key={section.id}
                            className="flex items-center justify-between p-4 bg-slate-900/50 rounded-lg border border-white/10 hover:border-white/20 transition-colors"
                            data-testid={`section-${String(section.id)}`}
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
                              onClick={() => { navigate(`/admin/customization/variants/${routeSlug}/sections/${String(section.id)}`); }}
                              data-testid={`edit-section-${String(section.id)}`}
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
