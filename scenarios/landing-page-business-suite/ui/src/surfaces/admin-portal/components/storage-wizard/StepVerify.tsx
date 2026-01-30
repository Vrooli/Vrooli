import { AlertCircle, CheckCircle2, RefreshCw, Save, Settings } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { inputBaseClassName } from '../formFieldClasses';
import type { StorageFormValues, StorageProviderId } from '../../services/downloads.service';
import type { TestStatus } from '../../hooks/useStorageWizard';

interface StepVerifyProps {
  provider: StorageProviderId;
  form: StorageFormValues;
  cloudflareAccountId: string;
  testStatus: TestStatus;
  testError: string | null;
  saveStatus: 'idle' | 'saving' | 'success' | 'error';
  saveError: string | null;
  onFormChange: (form: Partial<StorageFormValues>) => void;
  onTestConnection: () => Promise<void>;
  onSave: () => Promise<void>;
}

export function StepVerify({
  provider,
  form,
  cloudflareAccountId,
  testStatus,
  testError,
  saveStatus,
  saveError,
  onFormChange,
  onTestConnection,
  onSave,
}: StepVerifyProps) {
  const getProviderLabel = () => {
    switch (provider) {
      case 'aws-s3':
        return 'AWS S3';
      case 'cloudflare-r2':
        return 'Cloudflare R2';
      case 'minio':
        return 'MinIO';
      case 'custom':
        return 'S3-Compatible';
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-white">Review & Save</h3>
        <p className="mt-1 text-sm text-slate-400">
          Verify your configuration and test the connection before saving
        </p>
      </div>

      {/* Configuration Summary */}
      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 space-y-3">
        <h4 className="text-sm font-semibold text-white flex items-center gap-2">
          <Settings className="h-4 w-4 text-blue-400" />
          Configuration Summary
        </h4>
        <dl className="grid gap-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-slate-400">Provider</dt>
            <dd className="text-white font-medium">{getProviderLabel()}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-slate-400">Bucket</dt>
            <dd className="text-white font-medium font-mono">{form.bucket || '(not set)'}</dd>
          </div>
          {provider === 'aws-s3' && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Region</dt>
              <dd className="text-white font-medium">{form.region}</dd>
            </div>
          )}
          {provider === 'cloudflare-r2' && cloudflareAccountId && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Account ID</dt>
              <dd className="text-white font-medium font-mono">{cloudflareAccountId}</dd>
            </div>
          )}
          {(provider === 'minio' || provider === 'custom') && form.endpoint && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Endpoint</dt>
              <dd className="text-white font-medium font-mono text-xs break-all">{form.endpoint}</dd>
            </div>
          )}
          {form.defaultPrefix && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Prefix</dt>
              <dd className="text-white font-medium font-mono">{form.defaultPrefix}</dd>
            </div>
          )}
          {form.forcePathStyle && (
            <div className="flex justify-between">
              <dt className="text-slate-400">Path Style</dt>
              <dd className="text-emerald-400">Enabled</dd>
            </div>
          )}
        </dl>
      </div>

      {/* Connection Test */}
      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 space-y-3">
        <h4 className="text-sm font-semibold text-white">Connection Test</h4>
        <p className="text-xs text-slate-400">
          Test your connection to verify credentials and bucket access before saving.
        </p>

        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            onClick={onTestConnection}
            disabled={testStatus === 'testing'}
            className="gap-2"
          >
            {testStatus === 'testing' ? (
              <>
                <RefreshCw className="h-4 w-4 animate-spin" />
                Testing...
              </>
            ) : (
              <>
                <CheckCircle2 className="h-4 w-4" />
                Test Connection
              </>
            )}
          </Button>

          {testStatus === 'success' && (
            <div className="flex items-center gap-2 text-sm text-emerald-400">
              <CheckCircle2 className="h-4 w-4" />
              Connection successful
            </div>
          )}
        </div>

        {testStatus === 'error' && testError && (
          <div className="flex items-start gap-2 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm">
            <AlertCircle className="h-4 w-4 text-red-400 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-medium text-red-300">Connection failed</p>
              <p className="text-red-400/80 text-xs mt-1">{testError}</p>
            </div>
          </div>
        )}
      </div>

      {/* Advanced Settings */}
      <details className="group rounded-2xl border border-white/10 bg-white/5">
        <summary className="cursor-pointer p-4 text-sm font-semibold text-white flex items-center gap-2">
          <span className="group-open:rotate-90 transition-transform">
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
          </span>
          Advanced Settings
        </summary>
        <div className="border-t border-white/10 p-4 space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-300">Signed URL TTL (seconds)</label>
            <input
              type="number"
              value={form.signedUrlTtlSeconds}
              onChange={(e) => onFormChange({ signedUrlTtlSeconds: Number(e.target.value) })}
              className={inputBaseClassName}
              min={60}
              max={86400}
            />
            <p className="text-xs text-slate-500">
              How long presigned download URLs remain valid. Default: 900 (15 minutes)
            </p>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-300">Public Base URL</label>
            <input
              value={form.publicBaseUrl}
              onChange={(e) => onFormChange({ publicBaseUrl: e.target.value })}
              className={inputBaseClassName}
              placeholder="https://downloads.example.com"
            />
            <p className="text-xs text-slate-500">
              Optional CDN or custom domain URL for public downloads. Leave empty to use presigned URLs.
            </p>
          </div>
        </div>
      </details>

      {/* Save Button */}
      <div className="flex items-center justify-between pt-4 border-t border-white/10">
        <div>
          {saveStatus === 'success' && (
            <div className="flex items-center gap-2 text-sm text-emerald-400">
              <CheckCircle2 className="h-4 w-4" />
              Settings saved successfully
            </div>
          )}
          {saveStatus === 'error' && saveError && (
            <div className="flex items-center gap-2 text-sm text-red-400">
              <AlertCircle className="h-4 w-4" />
              {saveError}
            </div>
          )}
        </div>

        <Button
          onClick={onSave}
          disabled={saveStatus === 'saving' || !form.bucket.trim()}
          className="gap-2"
        >
          {saveStatus === 'saving' ? (
            <>
              <RefreshCw className="h-4 w-4 animate-spin" />
              Saving...
            </>
          ) : (
            <>
              <Save className="h-4 w-4" />
              Save Configuration
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
