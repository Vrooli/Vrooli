import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import {
  createDownloadAppAdmin,
  listDownloadAppsAdmin,
  saveDownloadAppAdmin,
  type DownloadApp,
  type DownloadAppInput,
  type DownloadAsset,
  type DownloadStorefront,
} from '../../../shared/api';
import { AlertCircle, CheckCircle2, Download, Plus, RefreshCw, Save } from 'lucide-react';

type PlatformKey = 'windows' | 'mac' | 'linux';

interface PlatformFormValues {
  platform: PlatformKey;
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
  appleLabel: string;
  appleUrl: string;
  appleBadge: string;
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
  return {
    platform,
    artifactUrl: asset?.artifact_url ?? '',
    releaseVersion: asset?.release_version ?? '',
    releaseNotes: asset?.release_notes ?? '',
    requiresEntitlement: asset?.requires_entitlement ?? true,
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

  return {
    appKey: app.app_key,
    name: app.name ?? '',
    tagline: app.tagline ?? '',
    description: app.description ?? '',
    installOverview: app.install_overview ?? '',
    installSteps: (app.install_steps ?? []).join('\n'),
    displayOrder: app.display_order ?? 0,
    appleLabel: appleStore?.label ?? 'App Store',
    appleUrl: appleStore?.url ?? '',
    appleBadge: appleStore?.badge ?? '',
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
    appleLabel: 'App Store',
    appleUrl: '',
    appleBadge: '',
    googleLabel: 'Google Play',
    googleUrl: '',
    googleBadge: '',
    platforms,
  };
}

function serializeApp(values: AppFormValues): DownloadAppInput {
  const storefronts: DownloadStorefront[] = [];
  if (values.appleUrl.trim()) {
    storefronts.push({
      store: 'app_store',
      label: values.appleLabel.trim() || 'App Store',
      url: values.appleUrl.trim(),
      badge: values.appleBadge.trim() || undefined,
    });
  }
  if (values.googleUrl.trim()) {
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

  const platforms = PLATFORM_KEYS.map((key) => {
    const entry = values.platforms[key];
    return {
      platform: entry.platform,
      artifact_url: entry.artifactUrl.trim(),
      release_version: entry.releaseVersion.trim(),
      release_notes: entry.releaseNotes.trim(),
      requires_entitlement: entry.requiresEntitlement,
      metadata: entry.sizeMb.trim() ? { size_mb: Number(entry.sizeMb) } : undefined,
    };
  }).filter((platform) => platform.artifact_url.length > 0 && platform.release_version.length > 0);

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

  const handleFieldChange = (key: string, field: keyof AppFormValues, value: string | number) => {
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

  const dirtyCount = useMemo(() => forms.filter((form) => isDirty(form)).length, [forms]);

  return (
    <AdminLayout>
      <div className="mx-auto max-w-5xl space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Downloads</p>
            <h1 className="text-3xl font-semibold text-white">Download & install settings</h1>
            <p className="text-sm text-slate-400">
              Configure bundled apps, store links, and installer metadata so the public landing can surface verified
              downloads.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={loadApps} disabled={loading}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Refresh
            </Button>
            <Button size="sm" onClick={handleAddApp}>
              <Plus className="mr-2 h-4 w-4" />
              Add App
            </Button>
          </div>
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
          <div className="rounded-3xl border border-white/10 bg-white/5 p-8 text-center text-sm text-slate-400">
            No download apps configured yet. Use “Add App” to create the first bundle entry.
          </div>
        ) : (
          <div className="space-y-6">
            {forms.map((form) => {
              const dirty = isDirty(form);
              return (
                <Card key={form.key} className="border-white/10 bg-white/5" data-testid={`download-card-${form.key}`}>
                  <CardHeader className="space-y-1">
                    <CardTitle className="text-2xl text-white flex items-center gap-2">
                      <Download className="h-5 w-5 text-blue-300" />
                      {form.values.name || 'Unnamed App'}
                    </CardTitle>
                    <CardDescription className="text-slate-400">
                      App key: {form.values.appKey || 'unset'}
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div className="grid gap-4 md:grid-cols-2">
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
                      <div className="space-y-3">
                        <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Display order</label>
                        <input
                          type="number"
                          value={form.values.displayOrder}
                          onChange={(event) => handleFieldChange(form.key, 'displayOrder', Number(event.target.value))}
                          className="w-full rounded-xl border border-white/10 bg-[#05070E] px-4 py-3 text-sm text-white"
                        />
                      </div>
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

                    <div className="grid gap-4 md:grid-cols-2">
                      <StorefrontFields
                        title="Apple App Store"
                        labelValue={form.values.appleLabel}
                        urlValue={form.values.appleUrl}
                        badgeValue={form.values.appleBadge}
                        onLabelChange={(value) => handleFieldChange(form.key, 'appleLabel', value)}
                        onUrlChange={(value) => handleFieldChange(form.key, 'appleUrl', value)}
                        onBadgeChange={(value) => handleFieldChange(form.key, 'appleBadge', value)}
                      />
                      <StorefrontFields
                        title="Google Play"
                        labelValue={form.values.googleLabel}
                        urlValue={form.values.googleUrl}
                        badgeValue={form.values.googleBadge}
                        onLabelChange={(value) => handleFieldChange(form.key, 'googleLabel', value)}
                        onUrlChange={(value) => handleFieldChange(form.key, 'googleUrl', value)}
                        onBadgeChange={(value) => handleFieldChange(form.key, 'googleBadge', value)}
                      />
                    </div>

                    <div className="space-y-3">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Desktop installers</p>
                      <div className="grid gap-4 md:grid-cols-3">
                        {PLATFORM_KEYS.map((platformKey) => {
                          const platform = form.values.platforms[platformKey];
                          return (
                            <div key={`${form.key}-${platformKey}`} className="space-y-3 rounded-2xl border border-white/10 bg-[#05070E] p-4">
                              <p className="text-sm font-semibold text-white">{platformKey.toUpperCase()}</p>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Artifact URL</label>
                                <input
                                  type="text"
                                  value={platform.artifactUrl}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'artifactUrl', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Release version</label>
                                <input
                                  type="text"
                                  value={platform.releaseVersion}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'releaseVersion', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Release notes</label>
                                <textarea
                                  value={platform.releaseNotes}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'releaseNotes', event.target.value)
                                  }
                                  rows={2}
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
                                />
                              </div>
                              <div className="space-y-2">
                                <label className="flex items-center gap-2 text-xs text-slate-500">
                                  <input
                                    type="checkbox"
                                    checked={platform.requiresEntitlement}
                                    onChange={(event) =>
                                      handlePlatformChange(form.key, platformKey, 'requiresEntitlement', event.target.checked)
                                    }
                                    className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                                  />
                                  Requires entitlement
                                </label>
                              </div>
                              <div className="space-y-2">
                                <label className="text-xs text-slate-500">Size (MB)</label>
                                <input
                                  type="number"
                                  value={platform.sizeMb}
                                  onChange={(event) =>
                                    handlePlatformChange(form.key, platformKey, 'sizeMb', event.target.value)
                                  }
                                  className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
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
  labelValue: string;
  urlValue: string;
  badgeValue: string;
  onLabelChange: (value: string) => void;
  onUrlChange: (value: string) => void;
  onBadgeChange: (value: string) => void;
}

function StorefrontFields({ title, labelValue, urlValue, badgeValue, onLabelChange, onUrlChange, onBadgeChange }: StorefrontFieldsProps) {
  return (
    <div className="space-y-3 rounded-2xl border border-white/10 bg-[#05070E] p-4">
      <p className="text-sm font-semibold text-white">{title}</p>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">Label</label>
        <input
          type="text"
          value={labelValue}
          onChange={(event) => onLabelChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
        />
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">URL</label>
        <input
          type="text"
          value={urlValue}
          onChange={(event) => onUrlChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
        />
      </div>
      <div className="space-y-2">
        <label className="text-xs text-slate-500">Badge text (optional)</label>
        <input
          type="text"
          value={badgeValue}
          onChange={(event) => onBadgeChange(event.target.value)}
          className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white"
        />
      </div>
    </div>
  );
}
