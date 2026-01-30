import { inputBaseClassName } from '../formFieldClasses';
import { AWS_REGIONS, type StorageFormValues, type StorageProviderId } from '../../services/downloads.service';

interface StepConfigurationProps {
  provider: StorageProviderId;
  form: StorageFormValues;
  cloudflareAccountId: string;
  onFormChange: (form: Partial<StorageFormValues>) => void;
  onCloudflareAccountIdChange: (accountId: string) => void;
}

export function StepConfiguration({
  provider,
  form,
  cloudflareAccountId,
  onFormChange,
  onCloudflareAccountIdChange,
}: StepConfigurationProps) {
  const getProviderTitle = () => {
    switch (provider) {
      case 'aws-s3':
        return 'Configure AWS S3';
      case 'cloudflare-r2':
        return 'Configure Cloudflare R2';
      case 'minio':
        return 'Configure MinIO';
      case 'custom':
        return 'Configure S3-Compatible Storage';
    }
  };

  const getProviderDescription = () => {
    switch (provider) {
      case 'aws-s3':
        return 'Enter your S3 bucket name and select the region where your bucket is located.';
      case 'cloudflare-r2':
        return 'Enter your R2 bucket name and Cloudflare account ID (found in your dashboard URL).';
      case 'minio':
        return 'Enter your MinIO bucket name and the endpoint URL of your MinIO server.';
      case 'custom':
        return 'Enter your bucket name and endpoint URL for your S3-compatible storage provider.';
    }
  };

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-white">{getProviderTitle()}</h3>
        <p className="mt-1 text-sm text-slate-400">{getProviderDescription()}</p>
      </div>

      <div className="space-y-4">
        {/* Bucket - always shown */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300">
            Bucket name <span className="text-red-400">*</span>
          </label>
          <input
            value={form.bucket}
            onChange={(e) => onFormChange({ bucket: e.target.value })}
            className={inputBaseClassName}
            placeholder={
              provider === 'cloudflare-r2'
                ? 'my-r2-bucket'
                : provider === 'minio'
                  ? 'my-minio-bucket'
                  : 'my-download-bucket'
            }
          />
          <p className="text-xs text-slate-500">
            The name of your storage bucket where artifacts will be stored
          </p>
        </div>

        {/* AWS S3: Region dropdown */}
        {provider === 'aws-s3' && (
          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-300">Region</label>
            <select
              value={form.region}
              onChange={(e) => onFormChange({ region: e.target.value })}
              className={inputBaseClassName}
            >
              {AWS_REGIONS.map((region) => (
                <option key={region.value} value={region.value}>
                  {region.label} ({region.value})
                </option>
              ))}
            </select>
            <p className="text-xs text-slate-500">
              Select the AWS region where your bucket is located
            </p>
          </div>
        )}

        {/* Cloudflare R2: Account ID */}
        {provider === 'cloudflare-r2' && (
          <div className="space-y-2">
            <label className="text-sm font-medium text-slate-300">
              Cloudflare Account ID <span className="text-red-400">*</span>
            </label>
            <input
              value={cloudflareAccountId}
              onChange={(e) => onCloudflareAccountIdChange(e.target.value)}
              className={inputBaseClassName}
              placeholder="abc123def456"
            />
            <p className="text-xs text-slate-500">
              Found in your Cloudflare dashboard URL: dash.cloudflare.com/
              <span className="text-blue-400">[account-id]</span>
            </p>
            {form.endpoint && (
              <p className="text-xs text-emerald-400">
                Endpoint: {form.endpoint}
              </p>
            )}
          </div>
        )}

        {/* MinIO: Endpoint (required) */}
        {provider === 'minio' && (
          <>
            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-300">
                Endpoint URL <span className="text-red-400">*</span>
              </label>
              <input
                value={form.endpoint}
                onChange={(e) => onFormChange({ endpoint: e.target.value })}
                className={inputBaseClassName}
                placeholder="https://minio.example.com:9000"
              />
              <p className="text-xs text-slate-500">
                The URL of your MinIO server including port (e.g., https://minio.example.com:9000)
              </p>
            </div>

            <div className="rounded-lg border border-white/10 bg-white/5 p-3">
              <label className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={form.forcePathStyle}
                  onChange={(e) => onFormChange({ forcePathStyle: e.target.checked })}
                  className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                />
                <div>
                  <span className="text-sm font-medium text-white">Force path-style URLs</span>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Required for most MinIO installations. Uses bucket/key format instead of bucket.domain/key.
                  </p>
                </div>
              </label>
            </div>
          </>
        )}

        {/* Custom: Endpoint and Region */}
        {provider === 'custom' && (
          <>
            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-300">Endpoint URL</label>
              <input
                value={form.endpoint}
                onChange={(e) => onFormChange({ endpoint: e.target.value })}
                className={inputBaseClassName}
                placeholder="https://s3.provider.com"
              />
              <p className="text-xs text-slate-500">
                The S3-compatible endpoint URL. Leave empty for AWS S3.
              </p>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-slate-300">Region</label>
              <input
                value={form.region}
                onChange={(e) => onFormChange({ region: e.target.value })}
                className={inputBaseClassName}
                placeholder="us-east-1 or auto"
              />
              <p className="text-xs text-slate-500">
                Region identifier. Use "auto" for providers that auto-detect, or leave empty if not required.
              </p>
            </div>

            <div className="rounded-lg border border-white/10 bg-white/5 p-3">
              <label className="flex items-center gap-3">
                <input
                  type="checkbox"
                  checked={form.forcePathStyle}
                  onChange={(e) => onFormChange({ forcePathStyle: e.target.checked })}
                  className="rounded border-white/20 bg-transparent text-emerald-400 focus:ring-emerald-400"
                />
                <div>
                  <span className="text-sm font-medium text-white">Force path-style URLs</span>
                  <p className="text-xs text-slate-400 mt-0.5">
                    Enable for providers that require path-style addressing (bucket in URL path rather than subdomain).
                  </p>
                </div>
              </label>
            </div>
          </>
        )}

        {/* Default prefix - shown for all providers */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300">Default prefix</label>
          <input
            value={form.defaultPrefix}
            onChange={(e) => onFormChange({ defaultPrefix: e.target.value })}
            className={inputBaseClassName}
            placeholder="downloads/"
          />
          <p className="text-xs text-slate-500">
            Optional path prefix for all uploaded artifacts (e.g., "downloads/" or "artifacts/v1/")
          </p>
        </div>
      </div>
    </div>
  );
}
