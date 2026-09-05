import { AlertCircle, ArrowRight, CheckCircle2, Download, GripVertical, ImageIcon, RefreshCw, Save, Star, Trash2, Upload } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { ImageUploader } from '../../../shared/ui/ImageUploader';
import { Textarea } from '../../../shared/ui/textarea';
import { inputBaseClassName } from './formFieldClasses';
import { DownloadStorefrontFields } from './DownloadStorefrontFields';
import { type AppFormState } from '../hooks/useDownloadsForm';
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

const isArtifactSource = (value: string): value is PlatformFormValues['artifactSource'] =>
  value === 'direct' || value === 'managed';


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

export function DownloadAppCard({
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
            onClick={() => { void onDelete(form.key); }}
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
            onChange={(event) => { onFieldChange(form.key, 'appKey', event.target.value); }}
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
              onChange={(event) => { onFieldChange(form.key, 'name', event.target.value); }}
              className={inputLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Tagline</label>
            <input
              type="text"
              value={form.values.tagline}
              onChange={(event) => { onFieldChange(form.key, 'tagline', event.target.value); }}
              className={inputLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Description</label>
            <Textarea
              value={form.values.description}
              onChange={(event) => { onFieldChange(form.key, 'description', event.target.value); }}
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
                  onChange={(url) => { onFieldChange(form.key, 'iconUrl', url ?? ''); }}
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
                  onChange={(url) => { onFieldChange(form.key, 'screenshotUrl', url ?? ''); }}
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
              onChange={(event) => { onFieldChange(form.key, 'installOverview', event.target.value); }}
              rows={3}
              className={textareaLargeClassName}
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Install steps (one per line)</label>
            <Textarea
              value={form.values.installSteps}
              onChange={(event) => { onFieldChange(form.key, 'installSteps', event.target.value); }}
              rows={4}
              className={textareaLargeClassName}
            />
          </div>
        </div>

        <div className="space-y-3">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Mobile store links</p>
          <div className="grid gap-4 md:grid-cols-2">
            <DownloadStorefrontFields
              title="Apple App Store"
              enabled={form.values.appleEnabled}
              onEnabledChange={(value) => { onFieldChange(form.key, 'appleEnabled', value); }}
              labelValue={form.values.appleLabel}
              urlValue={form.values.appleUrl}
              badgeValue={form.values.appleBadge}
              onLabelChange={(value) => { onFieldChange(form.key, 'appleLabel', value); }}
              onUrlChange={(value) => { onFieldChange(form.key, 'appleUrl', value); }}
              onBadgeChange={(value) => { onFieldChange(form.key, 'appleBadge', value); }}
            />
            <DownloadStorefrontFields
              title="Google Play"
              enabled={form.values.googleEnabled}
              onEnabledChange={(value) => { onFieldChange(form.key, 'googleEnabled', value); }}
              labelValue={form.values.googleLabel}
              urlValue={form.values.googleUrl}
              badgeValue={form.values.googleBadge}
              onLabelChange={(value) => { onFieldChange(form.key, 'googleLabel', value); }}
              onUrlChange={(value) => { onFieldChange(form.key, 'googleUrl', value); }}
              onBadgeChange={(value) => { onFieldChange(form.key, 'googleBadge', value); }}
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
                onClick={() => { onManageDownloads(form.values.appKey); }}
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
                return `${String(parseFloat((bytes / Math.pow(k, i)).toFixed(1)))} ${sizes[i] ?? 'B'}`;
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
                          { onPlatformChange(form.key, platformKey, 'enabled', event.target.checked); }
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
                          { onPlatformChange(form.key, platformKey, 'artifactUrl', event.target.value); }
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
                        { onPlatformChange(form.key, platformKey, 'releaseVersion', event.target.value); }
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
                        { onPlatformChange(form.key, platformKey, 'releaseNotes', event.target.value); }
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
                          { onPlatformChange(form.key, platformKey, 'requiresEntitlement', event.target.checked); }
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
            onClick={() => { void onSave(form.key); }}
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
            onClick={() => { onReset(form.key); }}
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
