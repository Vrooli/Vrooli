import { ExternalLink, AlertTriangle } from 'lucide-react';

export function AwsCredentialsHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Creating IAM Credentials</h3>
        <p className="text-slate-300 mb-4">
          You need an IAM user with S3 access to generate access keys. Follow these steps to create one with minimal permissions.
        </p>

        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Go to the <ExternalLinkInline href="https://console.aws.amazon.com/iam/">IAM Console</ExternalLinkInline></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Click <strong className="text-white">Users</strong> in the left sidebar, then <strong className="text-white">Create user</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Enter a name like <code className="text-blue-300 bg-blue-500/10 px-1 rounded">s3-downloads-service</code></p>
              <p className="text-xs text-slate-400 mt-1">
                Don't enable console access — this user only needs programmatic access.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Click <strong className="text-white">Next</strong>, then select <strong className="text-white">Attach policies directly</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">5</span>
            <div>
              <p>Search for and select <code className="text-blue-300 bg-blue-500/10 px-1 rounded">AmazonS3FullAccess</code></p>
              <p className="text-xs text-slate-400 mt-1">
                For tighter security, create a custom policy (see below) instead.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">6</span>
            <div>
              <p>Complete the user creation by clicking <strong className="text-white">Create user</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">7</span>
            <div>
              <p>Click on your new user, then go to <strong className="text-white">Security credentials</strong> tab</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">8</span>
            <div>
              <p>Scroll to <strong className="text-white">Access keys</strong> and click <strong className="text-white">Create access key</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">9</span>
            <div>
              <p>Select <strong className="text-white">Application running outside AWS</strong> and click Next</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">10</span>
            <div>
              <p>Click <strong className="text-white">Create access key</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="font-medium text-amber-300">Save your credentials now!</p>
          <p className="text-amber-200/80 text-xs mt-1">
            The <strong>Secret Access Key</strong> is only shown once. Copy both the Access Key ID and Secret Access Key and store them securely.
          </p>
        </div>
      </div>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Minimal Permissions Policy (Optional)</h3>
        <p className="text-slate-300 mb-3">
          For better security, create a custom policy that only allows access to your specific bucket:
        </p>
        <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::YOUR-BUCKET-NAME",
        "arn:aws:s3:::YOUR-BUCKET-NAME/*"
      ]
    }
  ]
}`}
        </pre>
        <p className="text-xs text-slate-400 mt-2">
          Replace <code className="text-blue-300 bg-blue-500/10 px-1 rounded">YOUR-BUCKET-NAME</code> with your actual bucket name.
        </p>
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
