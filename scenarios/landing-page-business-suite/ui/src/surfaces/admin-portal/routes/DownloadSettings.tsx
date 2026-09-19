import { useState, useCallback } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { DownloadAppCard } from '../components/DownloadAppCard';
import { DownloadAppsEmptyState } from '../components/DownloadAppsEmptyState';
import { DownloadSettingsControls } from '../components/DownloadSettingsControls';
import { inputBaseClassName } from '../components/formFieldClasses';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { StatusBadgeGrid } from '../components/StatusBadge';
import { Callout } from '../components/Callout';
import { StorageWizard } from '../components/storage-wizard';
import { ArtifactUploader } from '../components/ArtifactUploader';
import { Button } from '../../../shared/ui/button';
import { LAYOUT } from '../config/layout.constants';
import { presignDownloadArtifactGetAdmin, type DownloadApp } from '../../../shared/api';
import { Download, Plus, RefreshCw, Save, ExternalLink, Package, GripVertical, Upload, Star } from 'lucide-react';
import { useDownloadsForm } from '../hooks/useDownloadsForm';
import { useDownloadHosting } from '../hooks/useDownloadHosting';
import {
  PLATFORM_KEYS,
  type PlatformKey,
} from '../services/downloads.service';

const isPlatformKey = (value: string): value is PlatformKey =>
  PLATFORM_KEYS.includes(value as PlatformKey);

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
              <Button variant="outline" size="sm" onClick={() => { void loadApps(); }} disabled={loading} data-testid="downloads-refresh">
                <RefreshCw className="mr-2 h-4 w-4" />
                Refresh
              </Button>
              {activeTab === 'apps' && dirtyCount > 0 && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => { void handleSaveAll(); }}
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
                label: `${String(downloadHealth.appCount)} app${downloadHealth.appCount !== 1 ? 's' : ''}`,
                status: 'info',
                description: downloadHealth.appCount === 0 ? 'Add your first app' : 'Configured',
              },
              {
                label: `${String(downloadHealth.platformsConfigured)} platform${downloadHealth.platformsConfigured !== 1 ? 's' : ''}`,
                status: downloadHealth.platformsConfigured > 0 ? 'success' : 'warning',
                description: downloadHealth.platformsMissing > 0 ? `${String(downloadHealth.platformsMissing)} missing` : 'All set',
              },
              {
                label: `${String(downloadHealth.storefrontsConfigured)} store link${downloadHealth.storefrontsConfigured !== 1 ? 's' : ''}`,
                status: downloadHealth.storefrontsConfigured > 0 ? 'success' : 'info',
                description: downloadHealth.storefrontsConfigured === 0 ? 'Optional' : 'App Store / Play Store',
              },
              {
                label: dirtyCount === 0 ? 'All saved' : `${String(dirtyCount)} unsaved`,
                status: dirtyCount === 0 ? 'success' : 'warning',
                description: dirtyCount === 0 ? 'Up to date' : 'Save changes below',
              },
            ]}
          />
        )}

        <DownloadSettingsControls
          activeTab={activeTab}
          onTabChange={setActiveTab}
          dirtyCount={dirtyCount}
          error={error}
        />

        {activeTab === 'apps' ? (loading ? (
          <div className="space-y-4">
            {[0, 1].map((entry) => (
              <div key={entry} className="h-48 animate-pulse rounded-3xl border border-white/10 bg-white/5" />
            ))}
          </div>
        ) : forms.length === 0 ? (
          <DownloadAppsEmptyState onAddApp={handleAddApp} />
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
              <DownloadAppCard
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
                    onChange={(e) => { setArtifactsQuery(e.target.value); }}
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
                                onClick={() => {
                                  void presignDownloadArtifactGetAdmin(artifact.id).then(({ url }) => {
                                    window.open(url, '_blank');
                                  });
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
                                    platform: artifact.platform as PlatformKey,
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
                      <Button variant="outline" size="sm" onClick={() => { setSelectedArtifact(null); }}>
                        Cancel
                      </Button>
                    </div>
                    <div className="grid gap-4 md:grid-cols-3">
                      <div className="space-y-2 md:col-span-2">
                        <label className="text-xs text-slate-500">Target app</label>
                        <select
                          value={applyTarget.appKey}
                          onChange={(e) => { setApplyTarget((prev) => ({ ...prev, appKey: e.target.value })); }}
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
                          onChange={(e) => { setApplyTarget((prev) => ({ ...prev, platform: e.target.value as PlatformKey })); }}
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
                          onChange={(e) => { setApplyTarget((prev) => ({ ...prev, releaseVersion: e.target.value })); }}
                          className={inputBaseClassName}
                        />
                      </div>
                      <div className="space-y-2 md:col-span-2">
                        <label className="text-xs text-slate-500">Release notes (optional)</label>
                        <input
                          value={applyTarget.releaseNotes}
                          onChange={(e) => { setApplyTarget((prev) => ({ ...prev, releaseNotes: e.target.value })); }}
                          className={inputBaseClassName}
                        />
                      </div>
                      <div className="space-y-2">
                        <label className="flex items-center gap-2 text-xs text-slate-500">
                          <input
                            type="checkbox"
                            checked={applyTarget.requiresEntitlement}
                            onChange={(e) => { setApplyTarget((prev) => ({ ...prev, requiresEntitlement: e.target.checked })); }}
                            className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                          />
                          Requires entitlement
                        </label>
                      </div>
                    </div>
                    <Button
                      onClick={() => { void handleApplyArtifact(); }}
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
