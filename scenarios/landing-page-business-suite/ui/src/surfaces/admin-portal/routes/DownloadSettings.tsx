import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import {
  applyDownloadArtifactAdmin,
  commitDownloadArtifactAdmin,
  createDownloadAppAdmin,
  deleteDownloadAppAdmin,
  getDownloadStorageAdmin,
  listDownloadAppsAdmin,
  listDownloadArtifactsAdmin,
  presignDownloadArtifactGetAdmin,
  presignDownloadArtifactUploadAdmin,
  saveDownloadAppAdmin,
  testDownloadStorageAdmin,
  updateDownloadStorageAdmin,
  type DownloadArtifact,
  type DownloadApp,
  type DownloadAppInput,
  type DownloadAsset,
  type DownloadStorageSettingsSnapshot,
  type DownloadStorefront,
} from '../../../shared/api';
import { AlertCircle, CheckCircle2, Download, Plus, RefreshCw, Save, ExternalLink, Package, Monitor, Smartphone, Trash2, GripVertical, ImageIcon } from 'lucide-react';

type PlatformKey = 'windows' | 'mac' | 'linux';

interface PlatformFormValues {
  platform: PlatformKey;
  enabled: boolean;
  artifactSource: 'direct' | 'managed';
  artifactUrl: string;
  artifactId: string;
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
  iconUrl: string;
  screenshotUrl: string;
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
  const hasContent = Boolean(asset?.artifact_url || asset?.artifact_id || asset?.release_version);
  const explicitEnabled = asset?.metadata?.enabled;
  const enabled = explicitEnabled !== undefined ? Boolean(explicitEnabled) : hasContent;
  const artifactSource = asset?.artifact_source ?? (asset?.artifact_id ? 'managed' : 'direct');

  return {
    platform,
    enabled,
    artifactSource,
    artifactUrl: asset?.artifact_url ?? '',
    artifactId: asset?.artifact_id ? String(asset.artifact_id) : '',
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
    iconUrl: app.icon_url ?? '',
    screenshotUrl: app.screenshot_url ?? '',
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
    iconUrl: '',
    screenshotUrl: '',
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
    const artifactSource = entry.artifactSource;
    return {
      platform: entry.platform,
      artifact_source: artifactSource,
      artifact_id: artifactSource === 'managed' ? Number(entry.artifactId) || undefined : undefined,
      artifact_url: artifactSource === 'direct' ? entry.artifactUrl.trim() : '',
      release_version: entry.releaseVersion.trim(),
      release_notes: entry.releaseNotes.trim(),
      requires_entitlement: entry.requiresEntitlement,
      metadata: {
        ...(entry.sizeMb.trim() ? { size_mb: Number(entry.sizeMb) } : {}),
        enabled: entry.enabled,
      },
    };
  }).filter((platform) => {
    if (!platform.metadata.enabled) return false;
    if (!platform.release_version?.length) return false;
    if (platform.artifact_source === 'managed') return Boolean(platform.artifact_id);
    return platform.artifact_url.length > 0;
  });

  return {
    app_key: values.appKey.trim(),
    name: values.name.trim(),
    tagline: values.tagline.trim(),
    description: values.description.trim(),
    icon_url: values.iconUrl.trim() || undefined,
    screenshot_url: values.screenshotUrl.trim() || undefined,
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
  const [activeTab, setActiveTab] = useState<'apps' | 'hosting'>('apps');

  const [storageSettings, setStorageSettings] = useState<DownloadStorageSettingsSnapshot | null>(null);
  const [storageLoading, setStorageLoading] = useState(false);
  const [storageSaving, setStorageSaving] = useState(false);
  const [storageError, setStorageError] = useState<string | null>(null);
  const [storageSuccess, setStorageSuccess] = useState<string | null>(null);
  const [storageForm, setStorageForm] = useState({
    bucket: '',
    region: '',
    endpoint: '',
    forcePathStyle: false,
    defaultPrefix: '',
    signedUrlTtlSeconds: 900,
    publicBaseUrl: '',
  });
  const [credentialsForm, setCredentialsForm] = useState({
    accessKeyId: '',
    secretAccessKey: '',
    sessionToken: '',
    clearAccessKeyId: false,
    clearSecretAccessKey: false,
    clearSessionToken: false,
  });

  const [artifactsLoading, setArtifactsLoading] = useState(false);
  const [artifactsError, setArtifactsError] = useState<string | null>(null);
  const [artifactsQuery, setArtifactsQuery] = useState('');
  const [artifactsPlatform, setArtifactsPlatform] = useState<PlatformKey | ''>('');
  const [artifacts, setArtifacts] = useState<DownloadArtifact[]>([]);
  const [selectedArtifact, setSelectedArtifact] = useState<DownloadArtifact | null>(null);
  const [applyTarget, setApplyTarget] = useState({
    appKey: '',
    platform: 'windows' as PlatformKey,
    requiresEntitlement: false,
    releaseVersion: '',
    releaseNotes: '',
  });

  const [uploadState, setUploadState] = useState({
    file: null as File | null,
    platform: '' as PlatformKey | '',
    releaseVersion: '',
    appKey: '',
    busy: false,
    message: '' as string,
    error: '' as string,
  });

  const loadApps = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { apps } = await listDownloadAppsAdmin();
      const sorted = [...(apps ?? [])].sort((a, b) => (a.display_order ?? 0) - (b.display_order ?? 0));
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

  const loadStorage = useCallback(async () => {
    setStorageLoading(true);
    setStorageError(null);
    setStorageSuccess(null);
    try {
      const { settings } = await getDownloadStorageAdmin();
      setStorageSettings(settings);
      setStorageForm({
        bucket: settings.bucket ?? '',
        region: settings.region ?? '',
        endpoint: settings.endpoint ?? '',
        forcePathStyle: settings.force_path_style ?? false,
        defaultPrefix: settings.default_prefix ?? '',
        signedUrlTtlSeconds: settings.signed_url_ttl_seconds ?? 900,
        publicBaseUrl: settings.public_base_url ?? '',
      });
      setCredentialsForm((prev) => ({
        ...prev,
        accessKeyId: '',
        secretAccessKey: '',
        sessionToken: '',
        clearAccessKeyId: false,
        clearSecretAccessKey: false,
        clearSessionToken: false,
      }));
    } catch (err) {
      setStorageError(err instanceof Error ? err.message : 'Failed to load storage settings');
    } finally {
      setStorageLoading(false);
    }
  }, []);

  const loadArtifacts = useCallback(async () => {
    setArtifactsLoading(true);
    setArtifactsError(null);
    try {
      const response = await listDownloadArtifactsAdmin({
        query: artifactsQuery.trim() || undefined,
        platform: artifactsPlatform || undefined,
        page_size: 50,
      });
      setArtifacts(response.artifacts ?? []);
    } catch (err) {
      setArtifactsError(err instanceof Error ? err.message : 'Failed to load artifacts');
    } finally {
      setArtifactsLoading(false);
    }
  }, [artifactsPlatform, artifactsQuery]);

  useEffect(() => {
    if (activeTab !== 'hosting') return;
    void loadStorage();
    void loadArtifacts();
  }, [activeTab, loadArtifacts, loadStorage]);

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
        const configured =
          Boolean(p.releaseVersion) &&
          (p.artifactSource === 'managed' ? Boolean(p.artifactId) : Boolean(p.artifactUrl));
        if (configured) {
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
              {activeTab === 'apps' && dirtyCount > 0 && (
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
              {activeTab === 'apps' && (
                <Button size="sm" onClick={handleAddApp} data-testid="downloads-add-app">
                  <Plus className="mr-2 h-4 w-4" />
                  Add App
                </Button>
              )}
            </div>
          </div>

          {/* Setup Overview Stats */}
          {!loading && activeTab === 'apps' && (
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

          <div className="mt-6 flex flex-wrap gap-2" data-testid="downloads-tabs">
            <Button
              size="sm"
              variant={activeTab === 'apps' ? 'default' : 'outline'}
              onClick={() => setActiveTab('apps')}
              className="gap-2"
            >
              <Download className="h-4 w-4" />
              Apps
            </Button>
            <Button
              size="sm"
              variant={activeTab === 'hosting' ? 'default' : 'outline'}
              onClick={() => setActiveTab('hosting')}
              className="gap-2"
            >
              <Package className="h-4 w-4" />
              Hosting
            </Button>
          </div>
        </div>

        {activeTab === 'apps' && dirtyCount > 0 && (
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

        {activeTab === 'apps' ? (loading ? (
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

                      {/* App Images Section */}
                      <div className="space-y-4 rounded-2xl border border-white/10 bg-[#05070E] p-4">
                        <div className="flex items-center gap-2 text-sm font-semibold text-white">
                          <ImageIcon className="h-4 w-4 text-blue-300" />
                          App Images
                        </div>
                        <div className="grid gap-4 md:grid-cols-2">
                          <div className="space-y-2">
                            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Icon</label>
                            <p className="text-xs text-slate-500 mb-2">Small icon shown in the download section (recommended: 128x128px)</p>
                            <ImageUploader
                              value={form.values.iconUrl}
                              onChange={(url) => handleFieldChange(form.key, 'iconUrl', url ?? '')}
                              category="general"
                              placeholder="No icon set"
                              uploadLabel="Upload icon"
                              previewSize="md"
                              allowUrlInput
                              clearable
                            />
                          </div>
                          <div className="space-y-2">
                            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Screenshot</label>
                            <p className="text-xs text-slate-500 mb-2">Preview image shown in expanded view (recommended: 1280x720px)</p>
                            <ImageUploader
                              value={form.values.screenshotUrl}
                              onChange={(url) => handleFieldChange(form.key, 'screenshotUrl', url ?? '')}
                              category="general"
                              placeholder="No screenshot set"
                              uploadLabel="Upload screenshot"
                              previewSize="lg"
                              allowUrlInput
                              clearable
                            />
                          </div>
                        </div>
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
                                <label className="text-xs text-slate-500">Source</label>
                                <select
                                  value={platform.artifactSource}
                                  disabled={isDisabled}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'artifactSource', event.target.value as any)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                >
                                  <option value="direct">Paste URL</option>
                                  <option value="managed">Managed artifact (hosting)</option>
                                </select>
                              </div>
                              {platform.artifactSource === 'direct' ? (
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
                              ) : (
                                <div className="space-y-2">
                                  <label className="text-xs text-slate-500">Artifact ID</label>
                                  <input
                                    type="number"
                                    value={platform.artifactId}
                                    disabled={isDisabled}
                                    onChange={(event) =>
                                      handlePlatformChange(form.key, platformKey, 'artifactId', event.target.value)
                                    }
                                    className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                                  />
                                  <p className="text-xs text-slate-500">
                                    Use the Hosting tab to upload and browse artifacts, then apply one here.
                                  </p>
                                </div>
                              )}
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
        )) : (
          <div className="space-y-6" data-testid="downloads-hosting">
            <Card className="border-white/10 bg-white/5">
              <CardHeader>
                <CardTitle className="text-white">Connect download storage (S3-compatible)</CardTitle>
                <CardDescription className="text-slate-400">
                  Configure where installer artifacts are stored. Credentials can be provided here or inherited from the runtime environment.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {storageError && (
                  <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                    {storageError}
                  </div>
                )}
                {storageSuccess && (
                  <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-100">
                    {storageSuccess}
                  </div>
                )}
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Bucket</label>
                    <input
                      value={storageForm.bucket}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, bucket: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="my-download-bucket"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Region (optional for S3-compatible)</label>
                    <input
                      value={storageForm.region}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, region: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="us-east-1"
                    />
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <label className="text-xs text-slate-500">Endpoint (optional for R2/MinIO)</label>
                    <input
                      value={storageForm.endpoint}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, endpoint: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="https://<accountid>.r2.cloudflarestorage.com"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Default prefix</label>
                    <input
                      value={storageForm.defaultPrefix}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, defaultPrefix: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="business-suite/"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Signed URL TTL (seconds)</label>
                    <input
                      type="number"
                      value={storageForm.signedUrlTtlSeconds}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, signedUrlTtlSeconds: Number(e.target.value) }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      min={60}
                      max={86400}
                    />
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <label className="text-xs text-slate-500">Public base URL (optional)</label>
                    <input
                      value={storageForm.publicBaseUrl}
                      onChange={(e) => setStorageForm((prev) => ({ ...prev, publicBaseUrl: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="https://downloads.example.com"
                    />
                  </div>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">
                      Access key ID {storageSettings?.access_key_id_set ? <span className="text-emerald-300">(set)</span> : <span className="text-slate-500">(not set)</span>}
                    </label>
                    <input
                      value={credentialsForm.accessKeyId}
                      onChange={(e) => setCredentialsForm((prev) => ({ ...prev, accessKeyId: e.target.value, clearAccessKeyId: false }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="AKIA..."
                    />
                    <label className="flex items-center gap-2 text-xs text-slate-400">
                      <input
                        type="checkbox"
                        checked={credentialsForm.clearAccessKeyId}
                        onChange={(e) => setCredentialsForm((prev) => ({ ...prev, clearAccessKeyId: e.target.checked, accessKeyId: e.target.checked ? '' : prev.accessKeyId }))}
                        className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
                      />
                      Clear saved access key ID
                    </label>
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">
                      Secret access key {storageSettings?.secret_access_key_set ? <span className="text-emerald-300">(set)</span> : <span className="text-slate-500">(not set)</span>}
                    </label>
                    <input
                      type="password"
                      value={credentialsForm.secretAccessKey}
                      onChange={(e) => setCredentialsForm((prev) => ({ ...prev, secretAccessKey: e.target.value, clearSecretAccessKey: false }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="********"
                    />
                    <label className="flex items-center gap-2 text-xs text-slate-400">
                      <input
                        type="checkbox"
                        checked={credentialsForm.clearSecretAccessKey}
                        onChange={(e) => setCredentialsForm((prev) => ({ ...prev, clearSecretAccessKey: e.target.checked, secretAccessKey: e.target.checked ? '' : prev.secretAccessKey }))}
                        className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
                      />
                      Clear saved secret access key
                    </label>
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <label className="text-xs text-slate-500">
                      Session token {storageSettings?.session_token_set ? <span className="text-emerald-300">(set)</span> : <span className="text-slate-500">(not set)</span>}
                    </label>
                    <input
                      type="password"
                      value={credentialsForm.sessionToken}
                      onChange={(e) => setCredentialsForm((prev) => ({ ...prev, sessionToken: e.target.value, clearSessionToken: false }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                      placeholder="Optional (STS session token)"
                    />
                    <label className="flex items-center gap-2 text-xs text-slate-400">
                      <input
                        type="checkbox"
                        checked={credentialsForm.clearSessionToken}
                        onChange={(e) => setCredentialsForm((prev) => ({ ...prev, clearSessionToken: e.target.checked, sessionToken: e.target.checked ? '' : prev.sessionToken }))}
                        className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
                      />
                      Clear saved session token
                    </label>
                  </div>
                  <div className="md:col-span-2">
                    <label className="flex items-center gap-2 text-xs text-slate-400">
                      <input
                        type="checkbox"
                        checked={storageForm.forcePathStyle}
                        onChange={(e) => setStorageForm((prev) => ({ ...prev, forcePathStyle: e.target.checked }))}
                        className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                      />
                      Force path-style (often required for MinIO)
                    </label>
                  </div>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" disabled={storageLoading || storageSaving} onClick={() => void loadStorage()} className="gap-2">
                    <RefreshCw className={`h-4 w-4 ${storageLoading ? 'animate-spin' : ''}`} />
                    Reload
                  </Button>
                  <Button
                    variant="outline"
                    disabled={storageLoading || storageSaving}
                    onClick={async () => {
                      setStorageSaving(true);
                      setStorageError(null);
                      setStorageSuccess(null);
                      try {
                        const payload: Record<string, unknown> = {
                          provider: 's3',
                          bucket: storageForm.bucket,
                          region: storageForm.region,
                          endpoint: storageForm.endpoint,
                          force_path_style: storageForm.forcePathStyle,
                          default_prefix: storageForm.defaultPrefix,
                          signed_url_ttl_seconds: storageForm.signedUrlTtlSeconds,
                          public_base_url: storageForm.publicBaseUrl,
                        };
                        if (credentialsForm.clearAccessKeyId) payload.access_key_id = '';
                        else if (credentialsForm.accessKeyId.trim()) payload.access_key_id = credentialsForm.accessKeyId.trim();
                        if (credentialsForm.clearSecretAccessKey) payload.secret_access_key = '';
                        else if (credentialsForm.secretAccessKey.trim()) payload.secret_access_key = credentialsForm.secretAccessKey.trim();
                        if (credentialsForm.clearSessionToken) payload.session_token = '';
                        else if (credentialsForm.sessionToken.trim()) payload.session_token = credentialsForm.sessionToken.trim();

                        const { settings } = await updateDownloadStorageAdmin(payload as any);
                        setStorageSettings(settings);
                        setStorageSuccess('Saved storage settings.');
                      } catch (err) {
                        setStorageError(err instanceof Error ? err.message : 'Failed to save settings');
                      } finally {
                        setStorageSaving(false);
                      }
                    }}
                    className="gap-2"
                  >
                    {storageSaving ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    Save settings
                  </Button>
                  <Button
                    disabled={storageLoading || storageSaving}
                    onClick={async () => {
                      setStorageError(null);
                      setStorageSuccess(null);
                      try {
                        await testDownloadStorageAdmin();
                        setStorageSuccess('Connection test succeeded.');
                      } catch (err) {
                        setStorageError(err instanceof Error ? err.message : 'Connection test failed');
                      }
                    }}
                    className="gap-2"
                  >
                    <CheckCircle2 className="h-4 w-4" />
                    Test connection
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card className="border-white/10 bg-white/5">
              <CardHeader>
                <CardTitle className="text-white">Artifacts</CardTitle>
                <CardDescription className="text-slate-400">Upload, browse, and apply managed download artifacts.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 md:grid-cols-3">
                  <input
                    value={artifactsQuery}
                    onChange={(e) => setArtifactsQuery(e.target.value)}
                    className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white md:col-span-2"
                    placeholder="Search filename, key, version…"
                  />
                  <select
                    value={artifactsPlatform}
                    onChange={(e) => setArtifactsPlatform(e.target.value as any)}
                    className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                  >
                    <option value="">All platforms</option>
                    <option value="windows">Windows</option>
                    <option value="mac">macOS</option>
                    <option value="linux">Linux</option>
                  </select>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  <Button variant="outline" onClick={() => void loadArtifacts()} disabled={artifactsLoading} className="gap-2">
                    <RefreshCw className={`h-4 w-4 ${artifactsLoading ? 'animate-spin' : ''}`} />
                    Refresh list
                  </Button>
                  <Button
                    disabled={uploadState.busy}
                    onClick={async () => {
                      if (!uploadState.file) {
                        setUploadState((prev) => ({ ...prev, error: 'Choose a file first.' }));
                        return;
                      }
                      setUploadState((prev) => ({ ...prev, busy: true, error: '', message: '' }));
                      try {
                        const presign = await presignDownloadArtifactUploadAdmin({
                          filename: uploadState.file.name,
                          content_type: uploadState.file.type || 'application/octet-stream',
                          app_key: uploadState.appKey.trim() || undefined,
                          platform: uploadState.platform || undefined,
                          release_version: uploadState.releaseVersion.trim() || undefined,
                        });
                        const headers = new Headers();
                        Object.entries(presign.required_headers ?? {}).forEach(([key, value]) => {
                          if (key.toLowerCase() === 'host') return;
                          headers.set(key, value);
                        });
                        if (!headers.has('Content-Type')) {
                          headers.set('Content-Type', uploadState.file.type || 'application/octet-stream');
                        }
                        const uploadResp = await fetch(presign.upload_url, { method: 'PUT', headers, body: uploadState.file });
                        if (!uploadResp.ok) throw new Error(`Upload failed (${uploadResp.status})`);
                        await commitDownloadArtifactAdmin({
                          bucket: presign.bucket,
                          object_key: presign.object_key,
                          original_filename: uploadState.file.name,
                          content_type: uploadState.file.type || undefined,
                          platform: uploadState.platform || undefined,
                          release_version: uploadState.releaseVersion.trim() || undefined,
                        });
                        setUploadState((prev) => ({ ...prev, busy: false, file: null, message: 'Upload committed.', error: '' }));
                        await loadArtifacts();
                      } catch (err) {
                        setUploadState((prev) => ({ ...prev, busy: false, error: err instanceof Error ? err.message : 'Upload failed' }));
                      }
                    }}
                    className="gap-2"
                  >
                    {uploadState.busy ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />}
                    Upload
                  </Button>
                </div>

                {(uploadState.error || uploadState.message) && (
                  <div className={`rounded-xl border px-4 py-3 text-sm ${uploadState.error ? 'border-rose-500/30 bg-rose-500/10 text-rose-100' : 'border-emerald-500/30 bg-emerald-500/10 text-emerald-100'}`}>
                    {uploadState.error || uploadState.message}
                  </div>
                )}

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="space-y-2 md:col-span-2">
                    <label className="text-xs text-slate-500">File</label>
                    <input
                      type="file"
                      onChange={(e) => {
                        const file = e.target.files?.[0] ?? null;
                        setUploadState((prev) => ({ ...prev, file, error: '', message: '' }));
                      }}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">App key (optional)</label>
                    <input
                      value={uploadState.appKey}
                      onChange={(e) => setUploadState((prev) => ({ ...prev, appKey: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Platform (optional)</label>
                    <select
                      value={uploadState.platform}
                      onChange={(e) => setUploadState((prev) => ({ ...prev, platform: e.target.value as any }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                    >
                      <option value="">Unspecified</option>
                      <option value="windows">Windows</option>
                      <option value="mac">macOS</option>
                      <option value="linux">Linux</option>
                    </select>
                  </div>
                  <div className="space-y-2 md:col-span-2">
                    <label className="text-xs text-slate-500">Release version (optional)</label>
                    <input
                      value={uploadState.releaseVersion}
                      onChange={(e) => setUploadState((prev) => ({ ...prev, releaseVersion: e.target.value }))}
                      className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                    />
                  </div>
                </div>

                {artifactsError && (
                  <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
                    {artifactsError}
                  </div>
                )}

                <div className="overflow-x-auto rounded-xl border border-white/10">
                  <table className="min-w-full text-sm">
                    <thead className="bg-white/5 text-slate-300">
                      <tr>
                        <th className="px-4 py-3 text-left">Name</th>
                        <th className="px-4 py-3 text-left">Platform</th>
                        <th className="px-4 py-3 text-left">Version</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {artifacts.map((artifact) => (
                        <tr key={artifact.id} className="border-t border-white/10">
                          <td className="px-4 py-3 text-slate-100">
                            <div className="font-medium">{artifact.original_filename || artifact.object_key}</div>
                            <div className="text-xs text-slate-500">{artifact.stable_object_uri}</div>
                          </td>
                          <td className="px-4 py-3 text-slate-200">{artifact.platform || '—'}</td>
                          <td className="px-4 py-3 text-slate-200">{artifact.release_version || '—'}</td>
                          <td className="px-4 py-3 text-right">
                            <div className="flex flex-wrap justify-end gap-2">
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={async () => {
                                  if (artifact.stable_object_uri) await navigator.clipboard.writeText(artifact.stable_object_uri);
                                }}
                              >
                                Copy URI
                              </Button>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={async () => {
                                  const { url } = await presignDownloadArtifactGetAdmin(artifact.id);
                                  await navigator.clipboard.writeText(url);
                                }}
                              >
                                Copy signed URL
                              </Button>
                              <Button
                                size="sm"
                                onClick={() => {
                                  setSelectedArtifact(artifact);
                                  setApplyTarget({
                                    appKey: forms[0]?.values.appKey ?? '',
                                    platform: (artifact.platform as PlatformKey) || 'windows',
                                    requiresEntitlement: false,
                                    releaseVersion: artifact.release_version ?? '',
                                    releaseNotes: '',
                                  });
                                }}
                              >
                                Apply…
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                      {artifacts.length === 0 && (
                        <tr>
                          <td colSpan={4} className="px-4 py-8 text-center text-slate-400">
                            {artifactsLoading ? 'Loading artifacts…' : 'No artifacts yet.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {selectedArtifact && (
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4 space-y-3">
                    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-semibold text-white">Apply artifact #{selectedArtifact.id}</p>
                        <p className="text-xs text-slate-400">{selectedArtifact.original_filename || selectedArtifact.object_key}</p>
                      </div>
                      <Button variant="outline" size="sm" onClick={() => setSelectedArtifact(null)}>
                        Cancel
                      </Button>
                    </div>
                    <div className="grid gap-4 md:grid-cols-3">
                      <div className="space-y-2 md:col-span-2">
                        <label className="text-xs text-slate-500">Target app</label>
                        <select
                          value={applyTarget.appKey}
                          onChange={(e) => setApplyTarget((prev) => ({ ...prev, appKey: e.target.value }))}
                          className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                        >
                          {forms.map((form) => (
                            <option key={form.values.appKey} value={form.values.appKey}>
                              {form.values.name} ({form.values.appKey})
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs text-slate-500">Platform</label>
                        <select
                          value={applyTarget.platform}
                          onChange={(e) => setApplyTarget((prev) => ({ ...prev, platform: e.target.value as PlatformKey }))}
                          className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                        >
                          <option value="windows">Windows</option>
                          <option value="mac">macOS</option>
                          <option value="linux">Linux</option>
                        </select>
                      </div>
                      <div className="space-y-2">
                        <label className="text-xs text-slate-500">Release version</label>
                        <input
                          value={applyTarget.releaseVersion}
                          onChange={(e) => setApplyTarget((prev) => ({ ...prev, releaseVersion: e.target.value }))}
                          className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2 md:col-span-2">
                        <label className="text-xs text-slate-500">Release notes (optional)</label>
                        <input
                          value={applyTarget.releaseNotes}
                          onChange={(e) => setApplyTarget((prev) => ({ ...prev, releaseNotes: e.target.value }))}
                          className="w-full rounded-lg border border-white/10 bg-[#05070E] px-3 py-2 text-sm text-white"
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="flex items-center gap-2 text-xs text-slate-500">
                          <input
                            type="checkbox"
                            checked={applyTarget.requiresEntitlement}
                            onChange={(e) => setApplyTarget((prev) => ({ ...prev, requiresEntitlement: e.target.checked }))}
                            className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                          />
                          Requires entitlement
                        </label>
                      </div>
                    </div>
                    <Button
                      onClick={async () => {
                        if (!applyTarget.appKey.trim()) {
                          setArtifactsError('Select an app to apply to.');
                          return;
                        }
                        try {
                          await applyDownloadArtifactAdmin({
                            app_key: applyTarget.appKey,
                            platform: applyTarget.platform,
                            artifact_id: selectedArtifact.id,
                            release_version: applyTarget.releaseVersion.trim() || undefined,
                            release_notes: applyTarget.releaseNotes.trim() || undefined,
                            requires_entitlement: applyTarget.requiresEntitlement,
                          });
                          setSelectedArtifact(null);
                          await loadApps();
                          setStorageSuccess('Applied artifact to download asset.');
                        } catch (err) {
                          setArtifactsError(err instanceof Error ? err.message : 'Failed to apply artifact');
                        }
                      }}
                      className="gap-2"
                    >
                      <Save className="h-4 w-4" />
                      Apply to app
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
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
