import { ExternalLink, AlertTriangle } from 'lucide-react';

const POLICY_DOCUMENT = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "InspectVrooliDeliveryBucket",
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListBucket",
        "s3:ListBucketMultipartUploads"
      ],
      "Resource": "arn:aws:s3:::YOUR-BUCKET-NAME"
    },
    {
      "Sid": "ManageVrooliDeliveryObjects",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:AbortMultipartUpload",
        "s3:ListMultipartUploadParts"
      ],
      "Resource": "arn:aws:s3:::YOUR-BUCKET-NAME/*"
    }
  ]
}`;

export function AwsCredentialsHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Creating IAM Credentials</h3>
        <p className="text-slate-300 mb-4">
          LPBS needs one IAM access-key pair to reach its private delivery bucket. The access key ID and
          secret access key are the two halves of one pair, not two independent keys. Create separate IAM
          users — <code className="text-blue-300 bg-blue-500/10 px-1 rounded">vrooli-lpbs-local</code> and{' '}
          <code className="text-blue-300 bg-blue-500/10 px-1 rounded">vrooli-lpbs-prod</code> — so credentials
          can be audited, revoked, and rotated independently.
        </p>

        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Sign in to <ExternalLinkInline href="https://console.aws.amazon.com/iam/">AWS</ExternalLinkInline>. Personal account owners generally select Root user and enter the account email. Root access may administer IAM, but never create a root access key.</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Open <strong className="text-white">IAM → Policies</strong> and create <code className="text-blue-300 bg-blue-500/10 px-1 rounded">VrooliDeliveryBucketAccess</code> using the policy below.</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Open <strong className="text-white">IAM → Users → Create user</strong> and create <code className="text-blue-300 bg-blue-500/10 px-1 rounded">vrooli-lpbs-local</code> or <code className="text-blue-300 bg-blue-500/10 px-1 rounded">vrooli-lpbs-prod</code>.</p>
              <p className="text-xs text-slate-400 mt-1">Do not enable AWS Console access — these identities need programmatic access only.</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Attach <strong className="text-white">VrooliDeliveryBucketAccess</strong> directly. Do <strong className="text-white">not</strong> attach <code className="text-blue-300 bg-blue-500/10 px-1 rounded">AdministratorAccess</code> or <code className="text-blue-300 bg-blue-500/10 px-1 rounded">AmazonS3FullAccess</code>.</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">5</span>
            <div>
              <p>Open the user&apos;s <strong className="text-white">Security credentials</strong> tab, then under <strong className="text-white">Access keys</strong> select <strong className="text-white">Create access key</strong>.</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">6</span>
            <div>
              <p>Select <strong className="text-white">Application running outside AWS</strong>, enter the matching description (<em>Vrooli LPBS local deployment credential</em> or <em>Vrooli LPBS production deployment credential</em>), and create the key.</p>
            </div>
          </li>
        </ol>
      </section>

      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="font-medium text-amber-300">Save the secret only once</p>
          <p className="text-amber-200/80 text-xs mt-1">
            The <strong>secret access key</strong> is displayed only once. Never paste either credential into
            chat, logs, screenshots, or issue reports. Never commit credentials to Git or place them in
            service.json, application configuration, source files, or shell history.
          </p>
        </div>
      </div>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Required policy: VrooliDeliveryBucketAccess</h3>
        <p className="text-slate-300 mb-3">
          Both ARNs must name the configured bucket. Replace{' '}
          <code className="text-blue-300 bg-blue-500/10 px-1 rounded">YOUR-BUCKET-NAME</code> with your actual
          bucket name.
        </p>
        <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">{POLICY_DOCUMENT}</pre>
      </section>

      <section className="border-t border-white/10 pt-4">
        <h3 className="text-base font-semibold text-white mb-2">Provision in Vrooli</h3>
        <p className="text-slate-300 mb-3">
          Run these on the matching host and paste each value at the secure prompt — never as a command-line
          argument. Leave <code className="text-blue-300 bg-blue-500/10 px-1 rounded">delivery-s3-session-token</code>{' '}
          unset unless you deliberately use temporary AWS STS credentials.
        </p>
        <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`vrooli credentials provision --identity vrooli/landing-page-business-suite --field delivery-s3-access-key-id
vrooli credentials provision --identity vrooli/landing-page-business-suite --field delivery-s3-secret-access-key
vrooli credentials doctor --format json`}
        </pre>
      </section>

      <section className="border-t border-white/10 pt-4">
        <h3 className="text-base font-semibold text-white mb-2">Rotation and revocation</h3>
        <ol className="list-decimal space-y-1 pl-5 text-slate-300 text-xs">
          <li>Create a second access key for the IAM user.</li>
          <li>Provision the new pair into Vrooli.</li>
          <li>Restart or reload the affected service if necessary.</li>
          <li>Run presence and operational validation.</li>
          <li>Complete a real presign/upload/download check.</li>
          <li>Deactivate the old access key, verify deployment still works, then delete it after a short observation period.</li>
        </ol>
      </section>

      <section className="border-t border-white/10 pt-4">
        <h3 className="text-base font-semibold text-white mb-3">Official Documentation</h3>
        <ul className="space-y-2">
          <li>
            <ExternalLinkInline href="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html">
              Managing access keys - AWS IAM User Guide
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html">
              IAM policies - AWS IAM User Guide
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html">
              Security best practices - AWS IAM User Guide
            </ExternalLinkInline>
          </li>
        </ul>
      </section>
    </div>
  );
}

function ExternalLinkInline({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 text-blue-400 hover:text-blue-300"
    >
      {children}
      <ExternalLink className="h-3 w-3" />
    </a>
  );
}
