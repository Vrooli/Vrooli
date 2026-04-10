import { useState, useCallback, useRef } from 'react';
import { Upload, RefreshCw, CheckCircle2, X } from 'lucide-react';
import { Button } from '../../../shared/ui/button';
import { Callout } from './Callout';
import { inputBaseClassName } from './formFieldClasses';
import {
  presignDownloadArtifactUploadAdmin,
  commitDownloadArtifactAdmin,
} from '../../../shared/api';
import type { DownloadApp } from '../../../shared/api';

type PlatformKey = 'windows' | 'mac' | 'linux';

interface DetectedInfo {
  platform?: PlatformKey;
  version?: string;
  size: number;
}

/**
 * Parse filename to auto-detect platform and version
 */
function parseFilename(filename: string): { platform?: PlatformKey; version?: string } {
  const lower = filename.toLowerCase();

  // Platform detection
  let platform: PlatformKey | undefined;
  if (/win(dows)?|\.exe|\.msi/.test(lower)) platform = 'windows';
  else if (/mac|darwin|osx|\.dmg|\.pkg/.test(lower)) platform = 'mac';
  else if (/linux|\.deb|\.rpm|\.appimage/.test(lower)) platform = 'linux';

  // Version detection (patterns: v1.2.3, 1.2.3, -1.2.3-)
  const versionMatch = filename.match(/v?(\d+\.\d+\.\d+)/);
  const version = versionMatch?.[1];

  return { platform, version };
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

interface ArtifactUploaderProps {
  apps: DownloadApp[];
  defaultAppKey?: string;
  onUploadComplete?: (artifactId: number) => void;
  onCancel?: () => void;
}

export function ArtifactUploader({
  apps,
  defaultAppKey,
  onUploadComplete,
  onCancel,
}: ArtifactUploaderProps) {
  const [file, setFile] = useState<File | null>(null);
  const [detected, setDetected] = useState<DetectedInfo | null>(null);
  const [isDragOver, setIsDragOver] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);

  // Form fields (editable after auto-detection)
  const [appKey, setAppKey] = useState(defaultAppKey ?? apps[0]?.app_key ?? '');
  const [platform, setPlatform] = useState<PlatformKey | ''>('');
  const [version, setVersion] = useState('');
  const [setAsCurrent, setSetAsCurrent] = useState(true);

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileDrop = useCallback((droppedFile: File) => {
    setFile(droppedFile);
    setError(null);
    setSuccess(null);

    const { platform: detectedPlatform, version: detectedVersion } = parseFilename(droppedFile.name);
    setDetected({
      platform: detectedPlatform,
      version: detectedVersion,
      size: droppedFile.size,
    });

    // Pre-fill form fields with detected values (if not already set)
    if (detectedPlatform && !platform) {
      setPlatform(detectedPlatform);
    }
    if (detectedVersion && !version) {
      setVersion(detectedVersion);
    }
  }, [platform, version]);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    const droppedFile = e.dataTransfer.files[0];
    if (droppedFile) {
      handleFileDrop(droppedFile);
    }
  }, [handleFileDrop]);

  const handleFileInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      handleFileDrop(selectedFile);
    }
  }, [handleFileDrop]);

  const handleClearFile = useCallback(() => {
    setFile(null);
    setDetected(null);
    setError(null);
    setSuccess(null);
    setProgress(0);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, []);

  const handleUpload = useCallback(async () => {
    if (!file) {
      setError('Please select a file first');
      return;
    }
    if (!appKey) {
      setError('Please select an app');
      return;
    }
    if (!platform) {
      setError('Please select a platform');
      return;
    }
    if (!version) {
      setError('Please enter a version');
      return;
    }

    setUploading(true);
    setError(null);
    setSuccess(null);
    setProgress(10);

    try {
      // Step 1: Get presigned URL
      const presign = await presignDownloadArtifactUploadAdmin({
        filename: file.name,
        content_type: file.type || 'application/octet-stream',
        app_key: appKey,
        platform,
        release_version: version,
      });
      setProgress(20);

      // Step 2: Upload to S3
      const headers = new Headers();
      Object.entries(presign.required_headers ?? {}).forEach(([key, value]) => {
        if (key.toLowerCase() === 'host') return;
        headers.set(key, value);
      });
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', file.type || 'application/octet-stream');
      }

      const uploadResp = await fetch(presign.upload_url, {
        method: 'PUT',
        headers,
        body: file,
      });

      if (!uploadResp.ok) {
        throw new Error(`Upload failed (${uploadResp.status})`);
      }
      setProgress(70);

      // Step 3: Commit the artifact
      const artifact = await commitDownloadArtifactAdmin({
        bucket: presign.bucket,
        object_key: presign.object_key,
        original_filename: file.name,
        content_type: file.type || undefined,
        app_key: appKey,
        platform,
        release_version: version,
        set_as_current: setAsCurrent,
      });
      setProgress(100);

      setSuccess(`Uploaded ${file.name}${setAsCurrent ? ' and set as latest version' : ''}`);

      if (onUploadComplete) {
        onUploadComplete(artifact.id);
      }

      // Reset form for next upload
      handleClearFile();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  }, [file, appKey, platform, version, setAsCurrent, onUploadComplete, handleClearFile]);

  return (
    <div className="space-y-4">
      {/* Drag and drop zone */}
      <div
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
        className={`
          relative cursor-pointer rounded-2xl border-2 border-dashed p-8 text-center transition-all
          ${isDragOver
            ? 'border-blue-400 bg-blue-500/10'
            : file
              ? 'border-emerald-500/50 bg-emerald-500/5'
              : 'border-white/20 bg-white/5 hover:border-white/30 hover:bg-white/10'
          }
        `}
      >
        <input
          ref={fileInputRef}
          type="file"
          onChange={handleFileInputChange}
          className="hidden"
          accept=".exe,.msi,.dmg,.pkg,.deb,.rpm,.AppImage,.tar.gz,.zip"
        />

        {file ? (
          <div className="space-y-2">
            <CheckCircle2 className="mx-auto h-10 w-10 text-emerald-400" />
            <p className="text-lg font-medium text-white">{file.name}</p>
            {detected && (
              <div className="flex flex-wrap items-center justify-center gap-2 text-sm text-slate-300">
                {detected.platform && (
                  <span className="rounded-full bg-blue-500/20 px-2 py-0.5 text-blue-300">
                    {detected.platform.toUpperCase()}
                  </span>
                )}
                {detected.version && (
                  <span className="rounded-full bg-purple-500/20 px-2 py-0.5 text-purple-300">
                    v{detected.version}
                  </span>
                )}
                <span className="rounded-full bg-slate-500/20 px-2 py-0.5 text-slate-300">
                  {formatBytes(detected.size)}
                </span>
              </div>
            )}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                handleClearFile();
              }}
              className="mt-2 text-xs text-slate-400 hover:text-white"
            >
              <X className="mr-1 inline h-3 w-3" />
              Remove
            </button>
          </div>
        ) : (
          <div className="space-y-2">
            <Upload className="mx-auto h-10 w-10 text-slate-400" />
            <p className="text-lg font-medium text-white">
              Drag & drop your installer
            </p>
            <p className="text-sm text-slate-400">
              or click to browse
            </p>
            <p className="text-xs text-slate-500">
              Supports .exe, .msi, .dmg, .pkg, .deb, .rpm, .AppImage
            </p>
          </div>
        )}
      </div>

      {/* Form fields */}
      {file && (
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">App</label>
            <select
              value={appKey}
              onChange={(e) => setAppKey(e.target.value)}
              className={inputBaseClassName}
              disabled={uploading}
            >
              {apps.map((app) => (
                <option key={app.app_key} value={app.app_key}>
                  {app.name} ({app.app_key})
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Platform</label>
            <select
              value={platform}
              onChange={(e) => setPlatform(e.target.value as PlatformKey | '')}
              className={inputBaseClassName}
              disabled={uploading}
            >
              <option value="">Select platform...</option>
              <option value="windows">Windows</option>
              <option value="mac">macOS</option>
              <option value="linux">Linux</option>
            </select>
            {detected?.platform && platform !== detected.platform && (
              <p className="text-xs text-amber-400">
                Detected: {detected.platform.toUpperCase()} (changed)
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-xs uppercase tracking-[0.3em] text-slate-500">Version</label>
            <input
              type="text"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="e.g. 2.1.0"
              className={inputBaseClassName}
              disabled={uploading}
            />
            {detected?.version && version !== detected.version && (
              <p className="text-xs text-amber-400">
                Detected: v{detected.version} (changed)
              </p>
            )}
          </div>

          <div className="flex items-end space-y-2">
            <label className="flex items-center gap-2 text-sm text-slate-300">
              <input
                type="checkbox"
                checked={setAsCurrent}
                onChange={(e) => setSetAsCurrent(e.target.checked)}
                className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                disabled={uploading}
              />
              Set as latest version for {platform || 'platform'}
            </label>
          </div>
        </div>
      )}

      {/* Progress bar */}
      {uploading && (
        <div className="space-y-2">
          <div className="h-2 overflow-hidden rounded-full bg-white/10">
            <div
              className="h-full bg-blue-500 transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
          <p className="text-center text-sm text-slate-400">
            {progress < 20 && 'Preparing upload...'}
            {progress >= 20 && progress < 70 && 'Uploading to storage...'}
            {progress >= 70 && progress < 100 && 'Committing artifact...'}
            {progress === 100 && 'Complete!'}
          </p>
        </div>
      )}

      {/* Error/success messages */}
      {error && <Callout type="error" message={error} />}
      {success && <Callout type="success" message={success} />}

      {/* Action buttons */}
      <div className="flex flex-wrap items-center gap-3">
        <Button
          onClick={handleUpload}
          disabled={!file || !platform || !version || !appKey || uploading}
          className="gap-2"
        >
          {uploading ? (
            <>
              <RefreshCw className="h-4 w-4 animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <Upload className="h-4 w-4" />
              Upload
            </>
          )}
        </Button>
        {onCancel && (
          <Button variant="outline" onClick={onCancel} disabled={uploading}>
            Cancel
          </Button>
        )}
      </div>
    </div>
  );
}
