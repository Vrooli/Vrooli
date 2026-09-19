import { useState } from 'react';
import { AlertCircle, CheckCircle2, Key } from 'lucide-react';
import { inputBaseClassName } from '../formFieldClasses';
import type { CredentialsFormValues, StorageProviderId } from '../../services/downloads.service';
import type { DownloadStorageSettingsSnapshot } from '../../../../shared/api';
import { Callout } from '../Callout';
import { HelpModal } from './HelpModal';
import { AwsCredentialsHelp, CloudflareR2SetupHelp, MinioSetupHelp } from './help-content';
import { SetupTask, type SetupTaskStatus } from '@vrooli/react-component-library/SetupTask/0';

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
  const hasStoredCredentials = Boolean(
    existingSettings?.access_key_id_set && existingSettings.secret_access_key_set,
  );
  const credentialsAvailable = hasStoredCredentials;
  const taskStatus: SetupTaskStatus = credentialsAvailable ? 'unknown' : 'needs_attention';

  const stateFor = (state: string | undefined, fallback: boolean | undefined) => {
    if (state === 'unavailable') return { label: 'authority unavailable', tone: 'text-amber-400' };
    if (state === 'authority_error') return { label: 'authority error', tone: 'text-red-400' };
    if (state === 'configured' || (state === undefined && fallback)) {
      return { label: 'configured', tone: 'text-emerald-400' };
    }
    return { label: 'missing', tone: 'text-slate-400' };
  };
  const accessKeyState = stateFor(existingSettings?.access_key_id_state, existingSettings?.access_key_id_set);
  const secretKeyState = stateFor(existingSettings?.secret_access_key_state, existingSettings?.secret_access_key_set);
  const sessionTokenState = stateFor(existingSettings?.session_token_state, existingSettings?.session_token_set);

  const provisionCommands = [
    'vrooli credentials provision --identity vrooli/landing-page-business-suite --field delivery-s3-access-key-id',
    'vrooli credentials provision --identity vrooli/landing-page-business-suite --field delivery-s3-secret-access-key',
    'vrooli credentials doctor --format json',
  ];
  const providerLabel = provider === 'aws-s3'
    ? 'AWS S3'
    : provider === 'cloudflare-r2'
      ? 'Cloudflare R2'
      : provider === 'minio'
        ? 'MinIO'
        : 'S3-compatible storage';

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
    <SetupTask
      title={help.title}
      purpose={`Authenticate the ${providerLabel} storage destination used by the download service.`}
      target="local admin settings"
      account={providerLabel}
      status={taskStatus}
      statusLabel={credentialsAvailable ? 'Stored · verify next' : 'Credentials needed'}
      guidance={
        credentialsAvailable
          ? 'Values are available to the server, but storage access is not considered verified until the connection test succeeds on the Verify step.'
          : 'Provide the access identity and secret, then save and test the connection on the Verify step.'
      }
      testId="storage-credentials-task"
    >
      <div className="space-y-6">

      {credentialsCalloutMessage && (
        <Callout
          type="info"
          message={credentialsCalloutMessage}
          actions={[{ label: 'Credentials guide', onClick: () => { setShowCredentialsHelp(true); } }]}
        />
      )}

      {hasStoredCredentials && (
        <div className="flex items-start gap-3 rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4">
          <CheckCircle2 className="h-5 w-5 text-emerald-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-sm font-medium text-emerald-300">Credentials stored</p>
            <p className="mt-1 text-xs text-emerald-400/80">
              The access key and secret are held in this host&apos;s credential authority and are never
              returned to the browser. Enter new values to rotate them, or clear them below.
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
            {existingSettings && (
              <span className={`text-xs ${accessKeyState.tone}`}>({accessKeyState.label})</span>
            )}
          </label>
          <input
            value={credentials.accessKeyId}
            onChange={(e) =>
              { onCredentialsChange({
                accessKeyId: e.target.value,
                clearAccessKeyId: false,
              }); }
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
                  { onCredentialsChange({
                    clearAccessKeyId: e.target.checked,
                    accessKeyId: e.target.checked ? '' : credentials.accessKeyId,
                  }); }
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
            {existingSettings && (
              <span className={`text-xs ${secretKeyState.tone}`}>({secretKeyState.label})</span>
            )}
          </label>
          <input
            type="password"
            value={credentials.secretAccessKey}
            onChange={(e) =>
              { onCredentialsChange({
                secretAccessKey: e.target.value,
                clearSecretAccessKey: false,
              }); }
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
                  { onCredentialsChange({
                    clearSecretAccessKey: e.target.checked,
                    secretAccessKey: e.target.checked ? '' : credentials.secretAccessKey,
                  }); }
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
            {existingSettings && (
              <span className={`text-xs ${sessionTokenState.tone}`}>({sessionTokenState.label})</span>
            )}
          </summary>
          <div className="mt-3 space-y-2 pl-6">
            <input
              type="password"
              value={credentials.sessionToken}
              onChange={(e) =>
                { onCredentialsChange({
                  sessionToken: e.target.value,
                  clearSessionToken: false,
                }); }
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
                    { onCredentialsChange({
                      clearSessionToken: e.target.checked,
                      sessionToken: e.target.checked ? '' : credentials.sessionToken,
                    }); }
                  }
                  className="rounded border-white/20 bg-transparent text-amber-400 focus:ring-amber-400"
                />
                Clear saved session token
              </label>
            )}
          </div>
        </details>
      </div>

      {/* Provisioning reference */}
      <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-3">
        <p className="text-sm font-medium text-white">Provision with the credential authority</p>
        <p className="text-xs text-slate-400">
          Use a separate credential pair for this host and for production
          (<code className="text-blue-300">vrooli-lpbs-local</code> /{' '}
          <code className="text-blue-300">vrooli-lpbs-prod</code>). They should not be identical. Input uses a
          secure prompt or stdin; never pass a value as a command-line argument.
        </p>
        <pre className="overflow-x-auto rounded-lg bg-slate-800 p-3 text-xs text-slate-300">
          {provisionCommands.join('\n')}
        </pre>
        <p className="text-xs text-slate-500">
          The bucket stays private. Free and paid downloads both use short-lived presigned URLs, so a free
          download never needs a public S3 object or AWS credentials on the client.
        </p>
        <p className="text-xs text-slate-500">
          The secret access key is shown by AWS only once. If it is lost, create a replacement key and
          deactivate the old one. AWS allows at most two access keys per IAM user.
        </p>
      </div>

      {/* Security notice */}
      <div className="flex items-start gap-3 rounded-xl border border-blue-500/30 bg-blue-500/10 p-4">
        <AlertCircle className="h-5 w-5 text-blue-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="text-sm font-medium text-blue-300">Security note</p>
          <p className="mt-1 text-xs text-blue-400/80">
            Credentials are stored in this host&apos;s credential authority, encrypted at rest, and never
            logged or returned to the browser. Leave the fields empty to keep any existing values, or
            use an IAM role on the host instead of storing long-lived keys.
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
          onClose={() => { setShowCredentialsHelp(false); }}
          title={getCredentialsHelpTitle()}
        >
          {getCredentialsHelpContent()}
        </HelpModal>
      )}
      </div>
    </SetupTask>
  );
}
