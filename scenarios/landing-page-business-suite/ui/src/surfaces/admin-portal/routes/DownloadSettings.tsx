import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import {
  createDownloadAppAdmin,
  deleteDownloadAppAdmin,
  listDownloadAppsAdmin,
  saveDownloadAppAdmin,
  type DownloadApp,
  type DownloadAppInput,
  type DownloadAsset,
  type DownloadStorefront,
} from '../../../shared/api';
import { AlertCircle, CheckCircle2, Download, Plus, RefreshCw, Save, ExternalLink, Package, Monitor, Smartphone, Trash2, GripVertical } from 'lucide-react';

type PlatformKey = 'windows' | 'mac' | 'linux';

interface PlatformFormValues {
  platform: PlatformKey;
  enabled: boolean;
  artifactUrl: string;
  releaseVersion: string;
  releaseNotes: string;
  requiresEntitlement: boolean;
  sizeMb: string;
}

interface AppFormValues {
  appKey: string;
  name: string;
  tagline: string;
  description: string;
  installOverview: string;
  installSteps: string;
  displayOrder: number;
  appleEnabled: boolean;
  appleLabel: string;
  appleUrl: string;
  appleBadge: string;
  googleEnabled: boolean;
  googleLabel: string;
  googleUrl: string;
  googleBadge: string;
  platforms: Record<PlatformKey, PlatformFormValues>;
}

interface AppFormState {
  key: string;
  values: AppFormValues;
  original: AppFormValues;
  saving: boolean;
  error?: string;
  isNew?: boolean;
  lastSavedAt?: number;
}

const PLATFORM_KEYS: PlatformKey[] = ['windows', 'mac', 'linux'];

function buildPlatformForm(platform: PlatformKey, asset?: DownloadAsset): PlatformFormValues {
  // A platform is considered enabled if it has content or explicitly enabled metadata
  const hasContent = Boolean(asset?.artifact_url || asset?.release_version);
  const explicitEnabled = asset?.metadata?.enabled;
  const enabled = explicitEnabled !== undefined ? Boolean(explicitEnabled) : hasContent;

  return {
    platform,
    enabled,
    artifactUrl: asset?.artifact_url ?? '',
    releaseVersion: asset?.release_version ?? '',
    releaseNotes: asset?.release_notes ?? '',
    requiresEntitlement: asset?.requires_entitlement ?? false,
    sizeMb: asset?.metadata?.size_mb ? String(asset.metadata.size_mb) : '',
  };
}

function deserializeApp(app: DownloadApp): AppFormValues {
  const appleStore = app.storefronts?.find((store) => store.store === 'app_store');
  const googleStore = app.storefronts?.find((store) => store.store === 'play_store');

  const platformMap: Record<PlatformKey, PlatformFormValues> = PLATFORM_KEYS.reduce((acc, key) => {
    const asset = app.platforms?.find((platform) => platform.platform === key);
    acc[key] = buildPlatformForm(key, asset);
    return acc;
  }, {} as Record<PlatformKey, PlatformFormValues>);

  // Storefront enabled status: true if URL exists (can be overridden by metadata)
  const appleEnabled = appleStore ? Boolean(appleStore.url) : false;
  const googleEnabled = googleStore ? Boolean(googleStore.url) : false;

  return {
    appKey: app.app_key,
    name: app.name ?? '',
    tagline: app.tagline ?? '',
    description: app.description ?? '',
    installOverview: app.install_overview ?? '',
    installSteps: (app.install_steps ?? []).join('\n'),
    displayOrder: app.display_order ?? 0,
    appleEnabled,
    appleLabel: appleStore?.label ?? 'App Store',
    appleUrl: appleStore?.url ?? '',
    appleBadge: appleStore?.badge ?? '',
    googleEnabled,
    googleLabel: googleStore?.label ?? 'Google Play',
    googleUrl: googleStore?.url ?? '',
    googleBadge: googleStore?.badge ?? '',
    platforms: platformMap,
  };
}

function buildDefaultAppValues(appKey = ''): AppFormValues {
  const platforms = PLATFORM_KEYS.reduce(
    (acc, platform) => ({
      ...acc,
      [platform]: buildPlatformForm(platform),
    }),
    {} as Record<PlatformKey, PlatformFormValues>,
  );

  return {
    appKey,
    name: '',
    tagline: '',
    description: '',
    installOverview: '',
    installSteps: '',
    displayOrder: 0,
    appleEnabled: false,
    appleLabel: 'App Store',
    appleUrl: '',
    appleBadge: '',
    googleEnabled: false,
    googleLabel: 'Google Play',
    googleUrl: '',
    googleBadge: '',
    platforms,
  };
}

function serializeApp(values: AppFormValues): DownloadAppInput {
  const storefronts: DownloadStorefront[] = [];
  // Only include storefronts that are enabled AND have a URL
  if (values.appleEnabled && values.appleUrl.trim()) {
    storefronts.push({
      store: 'app_store',
      label: values.appleLabel.trim() || 'App Store',
      url: values.appleUrl.trim(),
      badge: values.appleBadge.trim() || undefined,
    });
  }
  if (values.googleEnabled && values.googleUrl.trim()) {
    storefronts.push({
      store: 'play_store',
      label: values.googleLabel.trim() || 'Google Play',
      url: values.googleUrl.trim(),
      badge: values.googleBadge.trim() || undefined,
    });
  }

  const installSteps = values.installSteps
    .split('\n')
    .map((step) => step.trim())
    .filter(Boolean);

  // Only include platforms that are enabled AND have required fields
  const platforms = PLATFORM_KEYS.map((key) => {
    const entry = values.platforms[key];
    return {
      platform: entry.platform,
      artifact_url: entry.artifactUrl.trim(),
      release_version: entry.releaseVersion.trim(),
      release_notes: entry.releaseNotes.trim(),
      requires_entitlement: entry.requiresEntitlement,
      metadata: {
        ...(entry.sizeMb.trim() ? { size_mb: Number(entry.sizeMb) } : {}),
        enabled: entry.enabled,
      },
    };
  }).filter((platform) => platform.metadata.enabled && platform.artifact_url.length > 0 && platform.release_version.length > 0);

  return {
    app_key: values.appKey.trim(),
    name: values.name.trim(),
    tagline: values.tagline.trim(),
    description: values.description.trim(),
    install_overview: values.installOverview.trim(),
    install_steps: installSteps,
    display_order: values.displayOrder,
    storefronts,
    platforms,
  };
}

function normalizePayload(values: AppFormValues) {
  return serializeApp(values);
}

function isDirty(form: AppFormState) {
  return JSON.stringify(normalizePayload(form.values)) !== JSON.stringify(normalizePayload(form.original));
}

export function DownloadSettings() {
  const [forms, setForms] = useState<AppFormState[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [draggingKey, setDraggingKey] = useState<string | null>(null);
  const [dragOverKey, setDragOverKey] = useState<string | null>(null);

  const loadApps = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { apps } = await listDownloadAppsAdmin();
      const sorted = [...apps].sort((a, b) => (a.display_order ?? 0) - (b.display_order ?? 0));
      const nextForms = sorted.map((app) => {
        const values = deserializeApp(app);
        return {
          key: app.app_key,
          values,
          original: values,
          saving: false,
        };
      });
      setForms(nextForms);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load download apps');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadApps();
  }, [loadApps]);

  const handleFieldChange = (key: string, field: keyof AppFormValues, value: string | number | boolean) => {
    setForms((prev) =>
      prev.map((form) =>
        form.key === key
          ? {
              ...form,
              values: {
                ...form.values,
                [field]: field === 'displayOrder' ? Number(value) : value,
              },
            }
          : form,
      ),
    );
  };

  const handlePlatformChange = (
    key: string,
    platformKey: PlatformKey,
    field: keyof PlatformFormValues,
    value: string | boolean,
  ) => {
    setForms((prev) =>
      prev.map((form) =>
        form.key === key
          ? {
              ...form,
              values: {
                ...form.values,
                platforms: {
                  ...form.values.platforms,
                  [platformKey]: {
                    ...form.values.platforms[platformKey],
                    [field]: value,
                  },
                },
              },
            }
          : form,
      ),
    );
  };

  const handleAddApp = () => {
    const tempKey = `app-${Date.now()}`;
    const nextValues = {
      ...buildDefaultAppValues(tempKey),
      name: 'New Bundle App',
    };
    setForms((prev) => [
      ...prev,
      {
        key: tempKey,
        values: nextValues,
        original: nextValues,
        saving: false,
        isNew: true,
      },
    ]);
  };

  const handleReset = (key: string) => {
    setForms((prev) =>
      prev.map((form) => (form.key === key ? { ...form, values: form.original, error: undefined } : form)),
    );
  };

  const handleDelete = async (key: string) => {
    const target = forms.find((form) => form.key === key);
    if (!target) return;

    // For new unsaved apps, just remove from local state
    if (target.isNew) {
      setForms((prev) => prev.filter((form) => form.key !== key));
      return;
    }

    const confirmed = window.confirm(
      `Are you sure you want to delete "${target.values.name || target.values.appKey}"? This action cannot be undone.`
    );
    if (!confirmed) return;

    setForms((prev) =>
      prev.map((form) => (form.key === key ? { ...form, saving: true, error: undefined } : form))
    );

    try {
      await deleteDownloadAppAdmin(target.values.appKey);
      setForms((prev) => prev.filter((form) => form.key !== key));
    } catch (err) {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? { ...form, saving: false, error: err instanceof Error ? err.message : 'Failed to delete app' }
            : form
        )
      );
    }
  };

  // Drag-and-drop handlers
  const handleDragStart = (key: string) => (e: React.DragEvent) => {
    setDraggingKey(key);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', key);
  };

  const handleDragOver = (key: string) => (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (draggingKey && key !== draggingKey) {
      setDragOverKey(key);
    }
  };

  const handleDragLeave = () => {
    setDragOverKey(null);
  };

  const handleDrop = (targetKey: string) => (e: React.DragEvent) => {
    e.preventDefault();
    const sourceKey = e.dataTransfer.getData('text/plain');

    if (!sourceKey || sourceKey === targetKey) {
      setDraggingKey(null);
      setDragOverKey(null);
      return;
    }

    setForms((prev) => {
      const sourceIndex = prev.findIndex((f) => f.key === sourceKey);
      const targetIndex = prev.findIndex((f) => f.key === targetKey);

      if (sourceIndex === -1 || targetIndex === -1) return prev;

      // Create new array with reordered items
      const newForms = [...prev];
      const [removed] = newForms.splice(sourceIndex, 1);
      newForms.splice(targetIndex, 0, removed);

      // Update displayOrder for all items based on new positions
      return newForms.map((form, index) => ({
        ...form,
        values: {
          ...form.values,
          displayOrder: index,
        },
      }));
    });

    setDraggingKey(null);
    setDragOverKey(null);
  };

  const handleDragEnd = () => {
    setDraggingKey(null);
    setDragOverKey(null);
  };

  const handleSave = async (key: string) => {
    const target = forms.find((form) => form.key === key);
    if (!target) {
      return;
    }
    if (!target.values.appKey.trim()) {
      setForms((prev) =>
        prev.map((form) => (form.key === key ? { ...form, error: 'App key is required before saving.' } : form)),
      );
      return;
    }
    setForms((prev) =>
      prev.map((form) => (form.key === key ? { ...form, saving: true, error: undefined } : form)),
    );

    try {
      const payload = serializeApp(target.values);
      const response = target.isNew
        ? await createDownloadAppAdmin(payload)
        : await saveDownloadAppAdmin(target.values.appKey, payload);
      const updatedValues = deserializeApp(response);
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                key: response.app_key,
                values: updatedValues,
                original: updatedValues,
                saving: false,
                isNew: false,
                error: undefined,
                lastSavedAt: Date.now(),
              }
            : form,
        ),
      );
    } catch (err) {
      setForms((prev) =>
        prev.map((form) =>
          form.key === key
            ? {
                ...form,
                saving: false,
                error: err instanceof Error ? err.message : 'Failed to save app',
              }
            : form,
        ),
      );
    }
  };

  const handleSaveAll = async () => {
    const dirtyForms = forms.filter((form) => isDirty(form));
    if (dirtyForms.length === 0) return;

    // Mark all dirty forms as saving
    setForms((prev) =>
      prev.map((form) =>
        isDirty(form) ? { ...form, saving: true, error: undefined } : form
      )
    );

    // Save all dirty forms in parallel
    const results = await Promise.allSettled(
      dirtyForms.map(async (form) => {
        const payload = serializeApp(form.values);
        const response = form.isNew
          ? await createDownloadAppAdmin(payload)
          : await saveDownloadAppAdmin(form.values.appKey, payload);
        return { key: form.key, response };
      })
    );

    // Update forms based on results
    setForms((prev) =>
      prev.map((form) => {
        const result = results.find((r, i) => dirtyForms[i].key === form.key);
        if (!result) return form;

        if (result.status === 'fulfilled') {
          const updatedValues = deserializeApp(result.value.response);
          return {
            ...form,
            key: result.value.response.app_key,
            values: updatedValues,
            original: updatedValues,
            saving: false,
            isNew: false,
            error: undefined,
            lastSavedAt: Date.now(),
          };
        } else {
          return {
            ...form,
            saving: false,
            error: result.reason instanceof Error ? result.reason.message : 'Failed to save app',
          };
        }
      })
    );
  };

  const [savingAll, setSavingAll] = useState(false);
  const handleSaveAllWithLoading = async () => {
    setSavingAll(true);
    await handleSaveAll();
    setSavingAll(false);
  };

  const dirtyCount = useMemo(() => forms.filter((form) => isDirty(form)).length, [forms]);

  // Compute download setup health
  const downloadHealth = useMemo(() => {
    const appCount = forms.length;
    let platformsConfigured = 0;
    let platformsMissing = 0;
    let storefrontsConfigured = 0;

    forms.forEach((form) => {
      PLATFORM_KEYS.forEach((platform) => {
        const p = form.values.platforms[platform];
        if (p.artifactUrl && p.releaseVersion) {
          platformsConfigured++;
        } else {
          platformsMissing++;
        }
      });
      if (form.values.appleUrl) storefrontsConfigured++;
      if (form.values.googleUrl) storefrontsConfigured++;
    });

    return {
      appCount,
      platformsConfigured,
      platformsMissing,
      storefrontsConfigured,
      hasApps: appCount > 0,
    };
  }, [forms]);

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  return (
    <AdminLayout>
      <div className="mx-auto max-w-5xl space-y-6">
        {/* Enhanced Header with Purpose Statement */}
        <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6" data-testid="downloads-header">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="flex-1">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Downloads</p>
              <h1 className="text-2xl font-bold text-white mt-1">Configure apps and installers for your landing page</h1>
              <p className="text-sm text-slate-400 mt-2">
                Set up bundled apps with desktop installers and mobile store links. These appear in your landing page's download section for verified distribution.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={previewPublicLanding} className="gap-2" data-testid="downloads-preview">
                <ExternalLink className="h-4 w-4" />
                Preview landing
              </Button>
              <Button variant="outline" size="sm" onClick={loadApps} disabled={loading} data-testid="downloads-refresh">
                <RefreshCw className="mr-2 h-4 w-4" />
                Refresh
              </Button>
              {dirtyCount > 0 && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleSaveAllWithLoading}
                  disabled={savingAll}
                  className="gap-2 border-emerald-500/30 text-emerald-400 hover:bg-emerald-500/10"
                  data-testid="downloads-save-all"
                >
                  {savingAll ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Save All ({dirtyCount})
                </Button>
              )}
              <Button size="sm" onClick={handleAddApp} data-testid="downloads-add-app">
                <Plus className="mr-2 h-4 w-4" />
                Add App
              </Button>
            </div>
          </div>

          {/* Setup Overview Stats */}
          {!loading && (
            <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4" data-testid="downloads-health">
              <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-white/5 px-4 py-3">
                <Package className="h-5 w-5 text-blue-300" />
                <div>
                  <p className="text-sm font-semibold text-white">{downloadHealth.appCount} app{downloadHealth.appCount !== 1 ? 's' : ''}</p>
                  <p className="text-xs text-slate-400">{downloadHealth.appCount === 0 ? 'Add your first app' : 'Configured'}</p>
                </div>
              </div>
              <div className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
                downloadHealth.platformsConfigured > 0 ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-amber-500/30 bg-amber-500/10'
              }`}>
                <Monitor className="h-5 w-5 text-emerald-300" />
                <div>
                  <p className="text-sm font-semibold text-white">{downloadHealth.platformsConfigured} platform{downloadHealth.platformsConfigured !== 1 ? 's' : ''}</p>
                  <p className="text-xs text-slate-400">{downloadHealth.platformsMissing > 0 ? `${downloadHealth.platformsMissing} missing` : 'All set'}</p>
                </div>
              </div>
              <div className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
                downloadHealth.storefrontsConfigured > 0 ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-white/10 bg-white/5'
              }`}>
                <Smartphone className="h-5 w-5 text-purple-300" />
                <div>
                  <p className="text-sm font-semibold text-white">{downloadHealth.storefrontsConfigured} store link{downloadHealth.storefrontsConfigured !== 1 ? 's' : ''}</p>
                  <p className="text-xs text-slate-400">{downloadHealth.storefrontsConfigured === 0 ? 'Optional' : 'App Store / Play Store'}</p>
                </div>
              </div>
              <div className={`flex items-center gap-3 rounded-xl border px-4 py-3 ${
                dirtyCount === 0 ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-amber-500/30 bg-amber-500/10'
              }`}>
                {dirtyCount === 0 ? (
                  <CheckCircle2 className="h-5 w-5 text-emerald-300" />
                ) : (
                  <AlertCircle className="h-5 w-5 text-amber-300" />
                )}
                <div>
                  <p className="text-sm font-semibold text-white">{dirtyCount === 0 ? 'All saved' : `${dirtyCount} unsaved`}</p>
                  <p className="text-xs text-slate-400">{dirtyCount === 0 ? 'Up to date' : 'Save changes below'}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {dirtyCount > 0 && (
          <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-100" data-testid="download-settings-dirty-banner">
            {dirtyCount} app{dirtyCount === 1 ? '' : 's'} have unsaved changes. Save each card to update the runtime payload.
          </div>
        )}

        {error && (
          <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center gap-2">
            <AlertCircle className="h-4 w-4" />
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <div className="space-y-4">
            {[0, 1].map((entry) => (
              <div key={entry} className="h-48 animate-pulse rounded-3xl border border-white/10 bg-white/5" />
            ))}
          </div>
        ) : forms.length === 0 ? (
          <div className="rounded-3xl border border-white/10 bg-white/5 p-8" data-testid="downloads-empty-state">
            <div className="text-center space-y-4">
              <Download className="h-12 w-12 mx-auto text-slate-500" />
              <div>
                <h3 className="text-lg font-semibold text-white">No download apps configured yet</h3>
                <p className="text-sm text-slate-400 mt-1">
                  Add your first app to display download options on your landing page.
                </p>
              </div>
              <Button onClick={handleAddApp} className="gap-2">
                <Plus className="h-4 w-4" />
                Add Your First App
              </Button>
            </div>
            <div className="mt-8 border-t border-white/10 pt-6">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">What you'll configure</p>
              <div className="grid gap-4 sm:grid-cols-3 text-sm">
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <Package className="h-5 w-5 text-blue-300 mb-2" />
                  <p className="font-semibold text-white">App identity</p>
                  <p className="text-xs text-slate-400 mt-1">Name, tagline, description, and install instructions</p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <Monitor className="h-5 w-5 text-emerald-300 mb-2" />
                  <p className="font-semibold text-white">Desktop installers</p>
                  <p className="text-xs text-slate-400 mt-1">Windows, Mac, and Linux artifact URLs with versions</p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <Smartphone className="h-5 w-5 text-purple-300 mb-2" />
                  <p className="font-semibold text-white">Mobile store links</p>
                  <p className="text-xs text-slate-400 mt-1">App Store and Google Play URLs (optional)</p>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            {forms.length > 1 && (
              <div className="flex items-center gap-2 text-sm text-slate-500 bg-slate-900/30 rounded-lg px-4 py-2 border border-white/5">
                <GripVertical className="h-4 w-4" />
                <span>Drag apps to reorder. Use "Save All" to persist the new order.</span>
              </div>
            )}
            {forms.map((form, index) => {
              const dirty = isDirty(form);
              const isDragging = draggingKey === form.key;
              const isDragOver = dragOverKey === form.key;
              return (
                <Card
                  key={form.key}
                  className={`border-white/10 bg-white/5 transition-all duration-200 ${
                    isDragging ? 'opacity-50 scale-[0.98]' : ''
                  } ${isDragOver ? 'ring-2 ring-blue-500/50 ring-offset-2 ring-offset-slate-950' : ''}`}
                  data-testid={`download-card-${form.key}`}
                  draggable
                  onDragStart={handleDragStart(form.key)}
                  onDragOver={handleDragOver(form.key)}
                  onDragLeave={handleDragLeave}
                  onDrop={handleDrop(form.key)}
                  onDragEnd={handleDragEnd}
                >
                  <CardHeader className="space-y-1">
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-3">
                        {/* Drag Handle */}
                        <div
                          className="cursor-grab active:cursor-grabbing p-1 -ml-1 rounded hover:bg-white/5 text-slate-500 hover:text-slate-300 transition-colors"
                          title="Drag to reorder"
                        >
                          <GripVertical className="h-5 w-5" />
                        </div>
                        <div className="space-y-1">
                          <CardTitle className="text-2xl text-white flex items-center gap-2">
                            <Download className="h-5 w-5 text-blue-300" />
                            {form.values.name || 'Unnamed App'}
                            <span className="text-sm font-normal text-slate-500">#{index + 1}</span>
                          </CardTitle>
                          <CardDescription className="text-slate-400">
                            App key: {form.values.appKey || 'unset'}
                          </CardDescription>
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(form.key)}
                        disabled={form.saving}
                        className="text-rose-400 hover:text-rose-300 hover:bg-rose-500/10"
                        data-testid={`download-delete-${form.key}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div className="space-y-3">
                      <label className="text-xs uppercase tracking-[0.3em] text-slate-500">App key</label>
                      <input
                        type="text"
                        value={form.values.appKey}
                        onChange={(event) => handleFieldChange(form.key, 'appKey', event.target.value)}
                        disabled={!form.isNew}
                        className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white disabled:opacity-60"
                        placeholder="automation_suite"
                      />
                      <p className="text-xs text-slate-500">
                        {form.isNew ? 'Slug used in download APIs. Cannot be changed later.' : 'Slug locked for existing apps.'}
                      </p>
                    </div>

                    <div className="grid gap-4">
                      <div className="space-y-2">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Name</label>
                        <input
                          type="text"
                          value={form.values.name}
                          onChange={(event) => handleFieldChange(form.key, 'name', event.target.value)}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Tagline</label>
                        <input
                          type="text"
                          value={form.values.tagline}
                          onChange={(event) => handleFieldChange(form.key, 'tagline', event.target.value)}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Description</label>
                        <textarea
                          value={form.values.description}
                          onChange={(event) => handleFieldChange(form.key, 'description', event.target.value)}
                          rows={3}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Install overview</label>
                        <textarea
                          value={form.values.installOverview}
                          onChange={(event) => handleFieldChange(form.key, 'installOverview', event.target.value)}
                          rows={3}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Install steps (one per line)</label>
                        <textarea
                          value={form.values.installSteps}
                          onChange={(event) => handleFieldChange(form.key, 'installSteps', event.target.value)}
                          rows={4}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
                    </div>

                    <div className="space-y-3">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Mobile store links</p>
                      <div className="grid gap-4 md:grid-cols-2">
                        <StorefrontFields
                          title="Apple App Store"
                          enabled={form.values.appleEnabled}
                          onEnabledChange={(value) => handleFieldChange(form.key, 'appleEnabled', value)}
                          labelValue={form.values.appleLabel}
                          urlValue={form.values.appleUrl}
                          badgeValue={form.values.appleBadge}
                          onLabelChange={(value) => handleFieldChange(form.key, 'appleLabel', value)}
                          onUrlChange={(value) => handleFieldChange(form.key, 'appleUrl', value)}
                          onBadgeChange={(value) => handleFieldChange(form.key, 'appleBadge', value)}
                        />
                        <StorefrontFields
                          title="Google Play"
                          enabled={form.values.googleEnabled}
                          onEnabledChange={(value) => handleFieldChange(form.key, 'googleEnabled', value)}
                          labelValue={form.values.googleLabel}
                          urlValue={form.values.googleUrl}
                          badgeValue={form.values.googleBadge}
                          onLabelChange={(value) => handleFieldChange(form.key, 'googleLabel', value)}
                          onUrlChange={(value) => handleFieldChange(form.key, 'googleUrl', value)}
                          onBadgeChange={(value) => handleFieldChange(form.key, 'googleBadge', value)}
                        />
                      </div>
                    </div>

                    <div className="space-y-3">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Desktop installers</p>
                      <div className="grid gap-4 md:grid-cols-3">
                        {PLATFORM_KEYS.map((platformKey) => {
                          const platform = form.values.platforms[platformKey];
                          const isDisabled = !platform.enabled;
                          return (
                            <div
                              key={`${form.key}-${platformKey}`}
                              className={`space-y-3 rounded-2xl border p-4 transition-opacity ${
                                platform.enabled
                                  ? 'border-white/10 bg-[#05070E]'
                                  : 'border-white/5 bg-[#05070E]/50 opacity-60'
                              }`}
                            >
                              <div className="flex items-center justify-between">
                                <p className="text-sm font-semibold text-white">{platformKey.toUpperCase()}</p>
                                <label className="flex items-center gap-2 text-xs text-slate-400">
                                  <input
                                    type="checkbox"
                                    checked={platform.enabled}
                                    onChange={(event) =>
                                      handlePlatformChange(form.key, platformKey, 'enabled', event.target.checked)
                                    }
                                    className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                                  />
                                  Enabled
                                </label>
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Artifact URL</label>
                                <input
                                  type="text"
                                  value={platform.artifactUrl}
                                  disabled={isDisabled}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'artifactUrl', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Release version</label>
                                <input
                                  type="text"
                                  value={platform.releaseVersion}
                                  disabled={isDisabled}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'releaseVersion', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Release notes</label>
                                <textarea
                                  value={platform.releaseNotes}
                                  disabled={isDisabled}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'releaseNotes', event.target.value)
                                  }
                                  rows={2}
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="flex items-center gap-2 text-xs text-slate-500">
                                  <input
                                    type="checkbox"
                                    checked={platform.requiresEntitlement}
                                    disabled={isDisabled}
                                    onChange={(event) =>
                                      handlePlatformChange(form.key, platformKey, 'requiresEntitlement', event.target.checked)
                                    }
                                    className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400 disabled:opacity-50"
                                  />
                                  Requires entitlement
                                </label>
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Size (MB)</label>
                                <input
                                  type="number"
                                  value={platform.sizeMb}
                                  disabled={isDisabled}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'sizeMb', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                />
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-3">
                      <Button
                        onClick={() => handleSave(form.key)}
                        disabled={form.saving || !dirty}
                        className="gap-2"
                        data-testid={`download-save-${form.key}`}
                      >
                        {form.saving ? (
                          <>
                            <RefreshCw className="h-4 w-4 animate-spin" />
                            Saving…
                          </>
                        ) : (
                          <>
                            <Save className="h-4 w-4" />
                            Save
                          </>
                        )}
                      </Button>
                      <Button
                        variant="outline"
                        onClick={() => handleReset(form.key)}
                        disabled={!dirty}
                        data-testid={`download-reset-${form.key}`}
                      >
                        Reset
                      </Button>
                      {form.lastSavedAt && (
                        <div className="flex items-center gap-1 text-xs text-emerald-300">
                          <CheckCircle2 className="h-4 w-4" />
                          Saved {new Date(form.lastSavedAt).toLocaleTimeString()}
                        </div>
                      )}
                      {form.error && (
                        <div className="flex items-center gap-1 text-xs text-rose-300">
                          <AlertCircle className="h-4 w-4" />
                          {form.error}
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </AdminLayout>
  );
}

interface StorefrontFieldsProps {
  title: string;
  enabled: boolean;
  onEnabledChange: (value: boolean) => void;
  labelValue: string;
  urlValue: string;
  badgeValue: string;
  onLabelChange: (value: string) => void;
  onUrlChange: (value: string) => void;
  onBadgeChange: (value: string) => void;
}

function StorefrontFields({
  title,
  enabled,
  onEnabledChange,
  labelValue,
  urlValue,
  badgeValue,
  onLabelChange,
  onUrlChange,
  onBadgeChange,
}: StorefrontFieldsProps) {
  return (
    <div
      className={`space-y-3 rounded-2xl border p-4 transition-opacity ${
        enabled ? 'border-white/10 bg-[#05070E]' : 'border-white/5 bg-[#05070E]/50 opacity-60'
      }`}
    >
      <div className="flex items-center justify-between">
        <p className="text-sm font-semibold text-white">{title}</p>
        <label className="flex items-center gap-2 text-xs text-slate-400">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(event) => onEnabledChange(event.target.checked)}
            className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
          />
          Enabled
        </label>
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">Label</label>
        <input
          type="text"
          value={labelValue}
          disabled={!enabled}
          onChange={(event) => onLabelChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
        />
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">URL</label>
        <input
          type="text"
          value={urlValue}
          disabled={!enabled}
          onChange={(event) => onUrlChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
        />
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">Badge text (optional)</label>
        <input
          type="text"
          value={badgeValue}
          disabled={!enabled}
          onChange={(event) => onBadgeChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
        />
      </div>
    </div>
  );
}
