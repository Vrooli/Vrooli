import { useState, useCallback } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { inputBaseClassName } from '../components/formFieldClasses';
import { StatusBadgeGrid } from '../components/StatusBadge';
import { Callout } from '../components/Callout';
import { StorageWizard } from '../components/storage-wizard';
import { ArtifactUploader } from '../components/ArtifactUploader';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import { Textarea } from '../../../shared/ui/input';
import { LAYOUT } from '../config/layout.constants';
import { presignDownloadArtifactGetAdmin, type DownloadApp } from '../../../shared/api';
import { AlertCircle, CheckCircle2, Download, Plus, RefreshCw, Save, ExternalLink, Package, Monitor, Smartphone, Trash2, GripVertical, ImageIcon, Upload, Star, ArrowRight } from 'lucide-react';
import { useDownloadsForm, type AppFormState } from '../hooks/useDownloadsForm';
import { useDownloadHosting } from '../hooks/useDownloadHosting';
import {
  isFormDirty,
  PLATFORM_KEYS,
  type PlatformKey,
  type AppFormValues,
  type PlatformFormValues,
} from '../services/downloads.service';

const inputLargeClassName = `${inputBaseClassName} rounded-xl px-4 py-3`;
const inputLargeDisabledClassName = `${inputLargeClassName} disabled:opacity-60`;
const textareaLargeClassName = 'rounded-xl px-4 py-3';
const surfacePanelClassName = 'rounded-2xl border border-white/10 bg-surface-darker p-4';
const isPlatformKey = (value: string): value is PlatformKey => PLATFORM_KEYS.includes(value as PlatformKey);
const isArtifactSource = (value: string): value is PlatformFormValues['artifactSource'] =>
  value === 'direct' || value === 'managed';

export function DownloadSettings() {
  const {
    forms,
    loading,
    error,
    dirtyCount,
    downloadHealth,
    loadApps,
    handleFieldChange,
    handlePlatformChange,
    handleAddApp,
    handleReset,
    handleDelete,
    handleSave,
    handleSaveAll,
    savingAll,
    draggingKey,
    dragOverKey,
    handleDragStart,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    handleDragEnd,
  } = useDownloadsForm();

  const [activeTab, setActiveTab] = useState<'apps' | 'hosting'>('apps');

  const getFirstAppKey = useCallback(() => forms[0]?.values.appKey ?? '', [forms]);

  const {
    storageSettings,
    storageSuccess,
    loadStorage,
    artifactsLoading,
    artifactsError,
    artifactsQuery,
    setArtifactsQuery,
    artifactsPlatform,
    setArtifactsPlatform,
    artifactsAppKey,
    setArtifactsAppKey,
    artifacts,
    selectedArtifact,
    setSelectedArtifact,
    applyTarget,
    setApplyTarget,
    loadArtifacts,
    handleApplyArtifact,
    handleSetArtifactAsCurrent,
  } = useDownloadHosting({ activeTab, loadApps, getFirstAppKey });

  // Build apps list for ArtifactUploader (convert form values to DownloadApp shape)
  const appsForUploader: DownloadApp[] = forms.map(form => ({
    bundle_key: 'business_suite',
    app_key: form.values.appKey,
    name: form.values.name || form.values.appKey,
    platforms: [],
  }));

  const previewPublicLanding = () => {
    window.open('/', '_blank', 'noopener,noreferrer');
  };

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Configure apps and installers for your landing page"
          description="Set up bundled apps with desktop installers and mobile store links. These appear in your landing page's download section for verified distribution."
          icon={Download}
          iconBgClass="bg-green-500/10"
          iconColorClass="text-green-400"
          testId="downloads-header"
          actions={
            <>
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
                  onClick={handleSaveAll}
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
            </>
          }
        />

        {/* Setup Overview Stats */}
        {!loading && activeTab === 'apps' && (
          <StatusBadgeGrid
            testId="downloads-health"
            columns={4}
            badges={[
              {
                label: `${downloadHealth.appCount} app${downloadHealth.appCount !== 1 ? 's' : ''}`,
                status: 'info',
                description: downloadHealth.appCount === 0 ? 'Add your first app' : 'Configured',
              },
              {
                label: `${downloadHealth.platformsConfigured} platform${downloadHealth.platformsConfigured !== 1 ? 's' : ''}`,
                status: downloadHealth.platformsConfigured > 0 ? 'success' : 'warning',
                description: downloadHealth.platformsMissing > 0 ? `${downloadHealth.platformsMissing} missing` : 'All set',
              },
              {
                label: `${downloadHealth.storefrontsConfigured} store link${downloadHealth.storefrontsConfigured !== 1 ? 's' : ''}`,
                status: downloadHealth.storefrontsConfigured > 0 ? 'success' : 'info',
                description: downloadHealth.storefrontsConfigured === 0 ? 'Optional' : 'App Store / Play Store',
              },
              {
                label: dirtyCount === 0 ? 'All saved' : `${dirtyCount} unsaved`,
                status: dirtyCount === 0 ? 'success' : 'warning',
                description: dirtyCount === 0 ? 'Up to date' : 'Save changes below',
              },
            ]}
          />
        )}

        <div className="flex flex-wrap gap-2" data-testid="downloads-tabs">
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

        {activeTab === 'apps' && dirtyCount > 0 && (
          <Callout
            type="warning"
            message={`${dirtyCount} app${dirtyCount === 1 ? '' : 's'} have unsaved changes. Save each card to update the runtime payload.`}
          />
        )}

        {error && (
          <Callout type="error" message={error} />
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
              <Callout
                type="tip"
                icon={GripVertical}
                message='Drag apps to reorder. Use "Save All" to persist the new order.'
              />
            )}
            {forms.map((form, index) => (
              <AppCard
                key={form.key}
                form={form}
                index={index}
                draggingKey={draggingKey}
                dragOverKey={dragOverKey}
                onFieldChange={handleFieldChange}
                onPlatformChange={handlePlatformChange}
                onSave={handleSave}
                onReset={handleReset}
                onDelete={handleDelete}
                onDragStart={handleDragStart}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                onDragEnd={handleDragEnd}
                onManageDownloads={(appKey) => {
                  setActiveTab('hosting');
                  setArtifactsAppKey(appKey);
                }}
              />
            ))}
          </div>
        )) : (
          <div className="space-y-6" data-testid="downloads-hosting">
            <FormSection
              title="Connect download storage (S3-compatible)"
              description="Configure where installer artifacts are stored. Choose your provider and follow the guided setup."
              icon={Package}
              iconColorClass="text-blue-300"
              testId="downloads-storage-section"
            >
              <StorageWizard
                initialSettings={storageSettings}
                onComplete={() => {
                  void loadStorage();
                  void loadArtifacts();
                }}
              />
            </FormSection>

            {/* Upload New Artifact Section */}
            <FormSection
              title="Upload new artifact"
              description="Drag and drop your installer file. Platform and version are auto-detected from the filename."
              icon={Upload}
              iconColorClass="text-emerald-300"
              testId="downloads-upload-section"
            >
              <ArtifactUploader
                apps={appsForUploader}
                defaultAppKey={forms[0]?.values.appKey}
                onUploadComplete={() => void loadArtifacts()}
              />
            </FormSection>

            <FormSection
              title="Artifact history"
              description="Browse all uploaded artifacts. Set any version as the current download for an app."
              icon={Download}
              iconColorClass="text-green-300"
              testId="downloads-artifacts-section"
            >
              <div className="space-y-4">
                {/* Filters */}
                <div className="grid gap-3 md:grid-cols-4">
                  <select
                    value={artifactsAppKey}
                    onChange={(e) => {
                      setArtifactsAppKey(e.target.value);
                      void loadArtifacts();
                    }}
                    className={inputBaseClassName}
                  >
                    <option value="">All apps</option>
                    {forms.map((form) => (
                      <option key={form.values.appKey} value={form.values.appKey}>
                        {form.values.name || form.values.appKey}
                      </option>
                    ))}
                  </select>
                  <select
                    value={artifactsPlatform}
                    onChange={(e) => {
                      const nextValue = e.target.value;
                      setArtifactsPlatform(isPlatformKey(nextValue) ? nextValue : '');
                    }}
                    className={inputBaseClassName}
                  >
                    <option value="">All platforms</option>
                    <option value="windows">Windows</option>
                    <option value="mac">macOS</option>
                    <option value="linux">Linux</option>
                  </select>
                  <input
                    value={artifactsQuery}
                    onChange={(e) => setArtifactsQuery(e.target.value)}
                    className={inputBaseClassName}
                    placeholder="Search filename, version…"
                  />
                  <Button variant="outline" onClick={() => void loadArtifacts()} disabled={artifactsLoading} className="gap-2">
                    <RefreshCw className={`h-4 w-4 ${artifactsLoading ? 'animate-spin' : ''}`} />
                    Refresh
                  </Button>
                </div>

                {artifactsError && (
                  <Callout type="error" message={artifactsError} />
                )}

                {storageSuccess && (
                  <Callout type="success" message={storageSuccess} />
                )}

                {/* Artifact Table */}
                <div className="overflow-x-auto rounded-xl border border-white/10">
                  <table className="min-w-full text-sm">
                    <thead className="bg-white/5 text-slate-300">
                      <tr>
                        <th className="px-4 py-3 text-left">File</th>
                        <th className="px-4 py-3 text-left">Platform</th>
                        <th className="px-4 py-3 text-left">Version</th>
                        <th className="px-4 py-3 text-left">Size</th>
                        <th className="px-4 py-3 text-center">Status</th>
                        <th className="px-4 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {artifacts.map((artifact) => (
                        <tr key={artifact.id} className="border-t border-white/10 hover:bg-white/5">
                          <td className="px-4 py-3 text-slate-100">
                            <div className="font-medium">{artifact.original_filename || artifact.object_key}</div>
                            <div className="text-xs text-slate-500">
                              {artifact.app_key && <span className="mr-2">App: {artifact.app_key}</span>}
                              <span>{new Date(artifact.created_at).toLocaleDateString()}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3 text-slate-200">
                            {artifact.platform ? (
                              <span className="rounded-full bg-blue-500/20 px-2 py-0.5 text-xs text-blue-300">
                                {artifact.platform.toUpperCase()}
                              </span>
                            ) : '—'}
                          </td>
                          <td className="px-4 py-3 text-slate-200">
                            {artifact.release_version ? (
                              <span className="rounded-full bg-purple-500/20 px-2 py-0.5 text-xs text-purple-300">
                                v{artifact.release_version}
                              </span>
                            ) : '—'}
                          </td>
                          <td className="px-4 py-3 text-slate-400 text-xs">
                            {artifact.size_bytes ? `${(artifact.size_bytes / (1024 * 1024)).toFixed(1)} MB` : '—'}
                          </td>
                          <td className="px-4 py-3 text-center">
                            {artifact.is_current ? (
                              <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/20 px-2 py-0.5 text-xs font-medium text-emerald-300">
                                <Star className="h-3 w-3" />
                                LATEST
                              </span>
                            ) : null}
                          </td>
                          <td className="px-4 py-3 text-right">
                            <div className="flex flex-wrap justify-end gap-2">
                              {!artifact.is_current && artifact.platform && artifact.app_key && (
                                <Button
                                  size="sm"
                                  variant="outline"
                                  className="gap-1 text-emerald-400 border-emerald-500/30 hover:bg-emerald-500/10"
                                  onClick={() => {
                                    if (artifact.app_key && artifact.platform) {
                                      void handleSetArtifactAsCurrent(
                                        artifact,
                                        artifact.app_key,
                                        artifact.platform as PlatformKey
                                      );
                                    }
                                  }}
                                >
                                  <Star className="h-3 w-3" />
                                  Set as Latest
                                </Button>
                              )}
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={async () => {
                                  const { url } = await presignDownloadArtifactGetAdmin(artifact.id);
                                  window.open(url, '_blank');
                                }}
                              >
                                Download
                              </Button>
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => {
                                  setSelectedArtifact(artifact);
                                  setApplyTarget({
                                    appKey: artifact.app_key ?? (forms[0]?.values.appKey ?? ''),
                                    platform: (artifact.platform as PlatformKey) || 'windows',
                                    requiresEntitlement: false,
                                    releaseVersion: artifact.release_version ?? '',
                                    releaseNotes: '',
                                  });
                                }}
                              >
                                Apply to App…
                              </Button>
                            </div>
                          </td>
                        </tr>
                      ))}
                      {artifacts.length === 0 && (
                        <tr>
                          <td colSpan={6} className="px-4 py-8 text-center text-slate-400">
                            {artifactsLoading ? 'Loading artifacts…' : 'No artifacts yet. Upload your first installer above.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Apply Artifact Dialog */}
                {selectedArtifact && (
                  <div className="rounded-2xl border border-white/10 bg-white/5 p-4 space-y-3">
                    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                      <div>
                        <p className="text-sm font-semibold text-white">Apply artifact to app</p>
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
                          className={inputBaseClassName}
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
                          className={inputBaseClassName}
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
                          className={inputBaseClassName}
                        />
                      </div>
                      <div className="space-y-2 md:col-span-2">
                        <label className="text-xs text-slate-500">Release notes (optional)</label>
                        <input
                          value={applyTarget.releaseNotes}
                          onChange={(e) => setApplyTarget((prev) => ({ ...prev, releaseNotes: e.target.value }))}
                          className={inputBaseClassName}
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
                      onClick={handleApplyArtifact}
                      className="gap-2"
                    >
                      <Save className="h-4 w-4" />
                      Apply to app
                    </Button>
                  </div>
                )}
              </div>
            </FormSection>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}

interface AppCardProps {
  form: AppFormState;
  index: number;
  draggingKey: string | null;
  dragOverKey: string | null;
  onFieldChange: (key: string, field: keyof AppFormValues, value: string | number | boolean) => void;
  onPlatformChange: (key: string, platformKey: PlatformKey, field: keyof PlatformFormValues, value: string | boolean) => void;
  onSave: (key: string) => Promise<void>;
  onReset: (key: string) => void;
  onDelete: (key: string) => Promise<void>;
  onDragStart: (key: string) => (e: React.DragEvent) => void;
  onDragOver: (key: string) => (e: React.DragEvent) => void;
  onDragLeave: () => void;
  onDrop: (targetKey: string) => (e: React.DragEvent) => void;
  onDragEnd: () => void;
  onManageDownloads: (appKey: string) => void;
}

function AppCard({
  form,
  index,
  draggingKey,
  dragOverKey,
  onFieldChange,
  onPlatformChange,
  onSave,
  onReset,
  onDelete,
  onDragStart,
  onDragOver,
  onDragLeave,
  onDrop,
  onDragEnd,
  onManageDownloads,
}: AppCardProps) {
  const dirty = isFormDirty(form.values, form.original);
  const isDragging = draggingKey === form.key;
  const isDragOver = dragOverKey === form.key;

  return (
    <Card
      className={`border-white/10 bg-white/5 transition-all duration-200 ${
        isDragging ? 'opacity-50 scale-[0.98]' : ''
      } ${isDragOver ? 'ring-2 ring-blue-500/50 ring-offset-2 ring-offset-slate-950' : ''}`}
      data-testid={`download-card-${form.key}`}
      draggable
      onDragStart={onDragStart(form.key)}
      onDragOver={onDragOver(form.key)}
      onDragLeave={onDragLeave}
      onDrop={onDrop(form.key)}
      onDragEnd={onDragEnd}
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
            onClick={() => onDelete(form.key)}
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
            onChange={(event) => onFieldChange(form.key, 'appKey', event.target.value)}
            disabled={!form.isNew}
            className={inputLargeDisabledClassName}
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
              onChange={(event) => onFieldChange(form.key, 'name', event.target.value)}
              className={inputLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Tagline</label>
            <input
              type="text"
              value={form.values.tagline}
              onChange={(event) => onFieldChange(form.key, 'tagline', event.target.value)}
              className={inputLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Description</label>
            <Textarea
              value={form.values.description}
              onChange={(event) => onFieldChange(form.key, 'description', event.target.value)}
              rows={3}
              className={textareaLargeClassName}
            />
          </div>

          {/* App Images Section */}
          <div className={`space-y-4 ${surfacePanelClassName}`}>
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
                  onChange={(url) => onFieldChange(form.key, 'iconUrl', url ?? '')}
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
                  onChange={(url) => onFieldChange(form.key, 'screenshotUrl', url ?? '')}
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
            <Textarea
              value={form.values.installOverview}
              onChange={(event) => onFieldChange(form.key, 'installOverview', event.target.value)}
              rows={3}
              className={textareaLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Install steps (one per line)</label>
            <Textarea
              value={form.values.installSteps}
              onChange={(event) => onFieldChange(form.key, 'installSteps', event.target.value)}
              rows={4}
              className={textareaLargeClassName}
            />
          </div>
        </div>

        <div className="space-y-3">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Mobile store links</p>
          <div className="grid gap-4 md:grid-cols-2">
            <StorefrontFields
              title="Apple App Store"
              enabled={form.values.appleEnabled}
              onEnabledChange={(value) => onFieldChange(form.key, 'appleEnabled', value)}
              labelValue={form.values.appleLabel}
              urlValue={form.values.appleUrl}
              badgeValue={form.values.appleBadge}
              onLabelChange={(value) => onFieldChange(form.key, 'appleLabel', value)}
              onUrlChange={(value) => onFieldChange(form.key, 'appleUrl', value)}
              onBadgeChange={(value) => onFieldChange(form.key, 'appleBadge', value)}
            />
            <StorefrontFields
              title="Google Play"
              enabled={form.values.googleEnabled}
              onEnabledChange={(value) => onFieldChange(form.key, 'googleEnabled', value)}
              labelValue={form.values.googleLabel}
              urlValue={form.values.googleUrl}
              badgeValue={form.values.googleBadge}
              onLabelChange={(value) => onFieldChange(form.key, 'googleLabel', value)}
              onUrlChange={(value) => onFieldChange(form.key, 'googleUrl', value)}
              onBadgeChange={(value) => onFieldChange(form.key, 'googleBadge', value)}
            />
          </div>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Desktop installers</p>
            {!form.isNew && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onManageDownloads(form.values.appKey)}
                className="gap-2 text-blue-400 border-blue-500/30 hover:bg-blue-500/10"
              >
                <Upload className="h-3.5 w-3.5" />
                Manage Downloads
                <ArrowRight className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {PLATFORM_KEYS.map((platformKey) => {
              const platform = form.values.platforms[platformKey];
              const isDisabled = !platform.enabled;
              const hasArtifact = platform.artifactSource === 'managed' && platform.artifactId;
              const formatBytes = (bytes: number) => {
                if (bytes === 0) return '0 B';
                const k = 1024;
                const sizes = ['B', 'KB', 'MB', 'GB'];
                const i = Math.floor(Math.log(bytes) / Math.log(k));
                return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
              };
              return (
                <div
                  key={`${form.key}-${platformKey}`}
                  className={`space-y-3 rounded-2xl border p-4 transition-opacity ${
                    platform.enabled
                      ? 'border-white/10 bg-surface-darker'
                      : 'border-white/5 bg-surface-darker/50 opacity-60'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-semibold text-white">{platformKey.toUpperCase()}</p>
                      {platform.releaseVersion && platform.enabled && (
                        <span className="rounded-full bg-purple-500/20 px-2 py-0.5 text-xs text-purple-300">
                          v{platform.releaseVersion}
                        </span>
                      )}
                    </div>
                    <label className="flex items-center gap-2 text-xs text-slate-400">
                      <input
                        type="checkbox"
                        checked={platform.enabled}
                        onChange={(event) =>
                          onPlatformChange(form.key, platformKey, 'enabled', event.target.checked)
                        }
                        className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                      />
                      Enabled
                    </label>
                  </div>

                  {/* Artifact summary for managed source */}
                  {platform.artifactSource === 'managed' && hasArtifact && platform.artifactFilename && (
                    <div className="rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-2.5 space-y-1">
                      <div className="flex items-center gap-2">
                        <Star className="h-3.5 w-3.5 text-emerald-400" />
                        <span className="text-xs font-medium text-emerald-300">Current artifact</span>
                      </div>
                      <p className="text-xs text-slate-300 truncate" title={platform.artifactFilename}>
                        {platform.artifactFilename}
                      </p>
                      <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                        {platform.artifactSizeBytes && (
                          <span>{formatBytes(platform.artifactSizeBytes)}</span>
                        )}
                        {platform.artifactCount && platform.artifactCount > 1 && (
                          <span className="text-blue-400">
                            {platform.artifactCount} version{platform.artifactCount !== 1 ? 's' : ''} available
                          </span>
                        )}
                      </div>
                    </div>
                  )}

                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Source</label>
                    <select
                      value={platform.artifactSource}
                      disabled={isDisabled}
                      onChange={(event) => {
                        const nextValue = event.target.value;
                        if (isArtifactSource(nextValue)) {
                          onPlatformChange(form.key, platformKey, 'artifactSource', nextValue);
                        }
                      }}
                      className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                    >
                      <option value="managed">Managed artifact (recommended)</option>
                      <option value="direct">External URL (advanced)</option>
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
                          onPlatformChange(form.key, platformKey, 'artifactUrl', event.target.value)
                        }
                        className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                        placeholder="https://example.com/app.exe"
                      />
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {!hasArtifact && (
                        <p className="text-xs text-amber-400">
                          No artifact set. Use "Manage Downloads" to upload and apply one.
                        </p>
                      )}
                    </div>
                  )}
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Release version</label>
                    <input
                      type="text"
                      value={platform.releaseVersion}
                      disabled={isDisabled}
                      onChange={(event) =>
                        onPlatformChange(form.key, platformKey, 'releaseVersion', event.target.value)
                      }
                      className="w-full rounded-lg border border-white/10 bg-transparent px-3 py-2 text-sm text-white disabled:opacity-50"
                      placeholder="e.g. 2.1.0"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-xs text-slate-500">Release notes</label>
                    <Textarea
                      value={platform.releaseNotes}
                      disabled={isDisabled}
                      onChange={(event) =>
                        onPlatformChange(form.key, platformKey, 'releaseNotes', event.target.value)
                      }
                      rows={2}
                      className="w-full bg-transparent text-sm disabled:opacity-50"
                      placeholder="What's new in this version..."
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="flex items-center gap-2 text-xs text-slate-500">
                      <input
                        type="checkbox"
                        checked={platform.requiresEntitlement}
                        disabled={isDisabled}
                        onChange={(event) =>
                          onPlatformChange(form.key, platformKey, 'requiresEntitlement', event.target.checked)
                        }
                        className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400 disabled:opacity-50"
                      />
                      Requires entitlement
                    </label>
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <Button
            onClick={() => onSave(form.key)}
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
            onClick={() => onReset(form.key)}
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
        enabled ? 'border-white/10 bg-surface-darker' : 'border-white/5 bg-surface-darker/50 opacity-60'
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
