import { useState } from 'react';
import { AlertCircle, CheckCircle2, Key } from 'lucide-react';
import { inputBaseClassName } from '../formFieldClasses';
import type { CredentialsFormValues, StorageProviderId } from '../../services/downloads.service';
import type { DownloadStorageSettingsSnapshot } from '../../../../shared/api';
import { Callout } from '../Callout';
import { HelpModal } from './HelpModal';
import { AwsCredentialsHelp, CloudflareR2SetupHelp, MinioSetupHelp } from './help-content';

interface StepCredentialsProps {
  provider: StorageProviderId;
  credentials: CredentialsFormValues;
  existingSettings: DownloadStorageSettingsSnapshot | null;
  onCredentialsChange: (credentials: Partial<CredentialsFormValues>) => void;
}

export function StepCredentials({
  provider,
  credentials,
  existingSettings,
  onCredentialsChange,
}: StepCredentialsProps) {
  const [showCredentialsHelp, setShowCredentialsHelp] = useState(false);

  const getProviderHelp = () => {
    switch (provider) {
      case 'aws-s3':
        return {
          title: 'AWS IAM Credentials',
          accessKeyHelp: 'Create an IAM user with S3 access, then generate access keys in the AWS console.',
          secretKeyHelp: 'The secret access key is shown only once when you create it. Keep it secure.',
          sessionTokenHelp: 'Only needed for temporary credentials from AWS STS.',
          docsUrl: 'https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html',
        };
      case 'cloudflare-r2':
        return {
          title: 'R2 API Token',
          accessKeyHelp: 'Create an API token in R2 > Manage R2 API Tokens with read/write permissions.',
          secretKeyHelp: 'The secret key is shown only once when you create the token.',
          sessionTokenHelp: 'Not typically used with R2.',
          docsUrl: 'https://developers.cloudflare.com/r2/api/s3/tokens/',
        };
      case 'minio':
        return {
          title: 'MinIO Credentials',
          accessKeyHelp: 'The access key configured for your MinIO server or service account.',
          secretKeyHelp: 'The secret key associated with your MinIO access key.',
          sessionTokenHelp: 'Only needed if using MinIO STS.',
          docsUrl: 'https://min.io/docs/minio/linux/administration/identity-access-management.html',
        };
      case 'custom':
      default:
        return {
          title: 'S3 API Credentials',
          accessKeyHelp: 'The access key ID provided by your storage provider.',
          secretKeyHelp: 'The secret access key provided by your storage provider.',
          sessionTokenHelp: 'Only needed for temporary/session-based credentials.',
          docsUrl: null,
        };
    }
  };

  const help = getProviderHelp();
  const hasEnvCredentials = existingSettings?.credentials_from_env ?? false;

  const getCredentialsHelpContent = () => {
    switch (provider) {
      case 'aws-s3':
        return <AwsCredentialsHelp />;
      case 'cloudflare-r2':
        return <CloudflareR2SetupHelp />;
      case 'minio':
        return <MinioSetupHelp />;
      default:
        return null;
    }
  };

  const getCredentialsHelpTitle = () => {
    switch (provider) {
      case 'aws-s3':
        return 'Creating AWS IAM Credentials';
      case 'cloudflare-r2':
        return 'Creating R2 API Tokens';
      case 'minio':
        return 'Creating MinIO Access Keys';
      default:
        return 'Creating Credentials';
    }
  };

  const getCredentialsCalloutMessage = () => {
    switch (provider) {
      case 'aws-s3':
        return "Need to generate IAM access keys? We'll show you how to create them in the AWS Console.";
      case 'cloudflare-r2':
        return "Need an R2 API token? We'll walk you through creating one with the right permissions.";
      case 'minio':
        return "Need MinIO access keys? We'll show you how to generate them in the MinIO Console.";
      default:
        return null;
    }
  };

  const credentialsCalloutMessage = getCredentialsCalloutMessage();

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-lg font-semibold text-white">{help.title}</h3>
        <p className="mt-1 text-sm text-slate-400">
          Enter your credentials to authenticate with {provider === 'aws-s3' ? 'AWS S3' : provider === 'cloudflare-r2' ? 'Cloudflare R2' : provider === 'minio' ? 'MinIO' : 'your storage provider'}
        </p>
      </div>

      {credentialsCalloutMessage && (
        <Callout
          type="info"
          message={credentialsCalloutMessage}
          actions={[{ label: 'Credentials guide', onClick: () => setShowCredentialsHelp(true) }]}
        />
      )}

      {hasEnvCredentials && (
        <div className="flex items-start gap-3 rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4">
          <CheckCircle2 className="h-5 w-5 text-emerald-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-emerald-300">Environment credentials detected</p>
            <p className="mt-1 text-xs text-emerald-400/80">
              Your server is configured to use credentials from environment variables. You can leave these fields empty to use them, or enter values here to override.
            </p>
          </div>
        </div>
      )}

      <div className="space-y-4">
        {/* Access Key ID */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300 flex items-center gap-2">
            <Key className="h-4 w-4" />
            Access Key ID
            {existingSettings?.access_key_id_set && (
              <span className="text-xs text-emerald-400">(configured)</span>
            )}
          </label>
          <input
            value={credentials.accessKeyId}
            onChange={(e) =>
              onCredentialsChange({
                accessKeyId: e.target.value,
                clearAccessKeyId: false,
              })
            }
            className={inputBaseClassName}
            placeholder={existingSettings?.access_key_id_set ? '••••••••••••' : 'AKIA...'}
            autoComplete="off"
          />
          <p className="text-xs text-slate-500">{help.accessKeyHelp}</p>
          {existingSettings?.access_key_id_set && (
            <label className="flex items-center gap-2 text-xs text-slate-400 mt-1">
              <input
                type="checkbox"
                checked={credentials.clearAccessKeyId}
                onChange={(e) =>
                  onCredentialsChange({
                    clearAccessKeyId: e.target.checked,
                    accessKeyId: e.target.checked ? '' : credentials.accessKeyId,
                  })
                }
                className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
              />
              Clear saved access key ID
            </label>
          )}
        </div>

        {/* Secret Access Key */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-slate-300 flex items-center gap-2">
            <Key className="h-4 w-4" />
            Secret Access Key
            {existingSettings?.secret_access_key_set && (
              <span className="text-xs text-emerald-400">(configured)</span>
            )}
          </label>
          <input
            type="password"
            value={credentials.secretAccessKey}
            onChange={(e) =>
              onCredentialsChange({
                secretAccessKey: e.target.value,
                clearSecretAccessKey: false,
              })
            }
            className={inputBaseClassName}
            placeholder={existingSettings?.secret_access_key_set ? '••••••••••••' : 'Enter secret key'}
            autoComplete="off"
          />
          <p className="text-xs text-slate-500">{help.secretKeyHelp}</p>
          {existingSettings?.secret_access_key_set && (
            <label className="flex items-center gap-2 text-xs text-slate-400 mt-1">
              <input
                type="checkbox"
                checked={credentials.clearSecretAccessKey}
                onChange={(e) =>
                  onCredentialsChange({
                    clearSecretAccessKey: e.target.checked,
                    secretAccessKey: e.target.checked ? '' : credentials.secretAccessKey,
                  })
                }
                className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
              />
              Clear saved secret access key
            </label>
          )}
        </div>

        {/* Session Token (collapsible) */}
        <details className="group">
          <summary className="cursor-pointer text-sm font-medium text-slate-400 hover:text-slate-300 flex items-center gap-2">
            <span className="group-open:rotate-90 transition-transform">
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </span>
            Session Token (optional)
            {existingSettings?.session_token_set && (
              <span className="text-xs text-emerald-400">(configured)</span>
            )}
          </summary>
          <div className="mt-3 space-y-2 pl-6">
            <input
              type="password"
              value={credentials.sessionToken}
              onChange={(e) =>
                onCredentialsChange({
                  sessionToken: e.target.value,
                  clearSessionToken: false,
                })
              }
              className={inputBaseClassName}
              placeholder="Optional session token"
              autoComplete="off"
            />
            <p className="text-xs text-slate-500">{help.sessionTokenHelp}</p>
            {existingSettings?.session_token_set && (
              <label className="flex items-center gap-2 text-xs text-slate-400 mt-1">
                <input
                  type="checkbox"
                  checked={credentials.clearSessionToken}
                  onChange={(e) =>
                    onCredentialsChange({
                      clearSessionToken: e.target.checked,
                      sessionToken: e.target.checked ? '' : credentials.sessionToken,
                    })
                  }
                  className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
                />
                Clear saved session token
              </label>
            )}
          </div>
        </details>
      </div>

      {/* Security notice */}
      <div className="flex items-start gap-3 rounded-xl border border-blue-500/30 bg-blue-500/10 p-4">
        <AlertCircle className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-blue-300">Security note</p>
          <p className="mt-1 text-xs text-blue-400/80">
            Credentials are encrypted at rest and never logged. For production environments, consider using environment variables or IAM roles instead of storing credentials.
          </p>
          {help.docsUrl && (
            <a
              href={help.docsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="mt-2 inline-flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300"
            >
              View documentation
              <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
            </a>
          )}
        </div>
      </div>

      {provider !== 'custom' && (
        <HelpModal
          open={showCredentialsHelp}
          onClose={() => setShowCredentialsHelp(false)}
          title={getCredentialsHelpTitle()}
        >
          {getCredentialsHelpContent()}
        </HelpModal>
      )}
    </div>
  );
}
