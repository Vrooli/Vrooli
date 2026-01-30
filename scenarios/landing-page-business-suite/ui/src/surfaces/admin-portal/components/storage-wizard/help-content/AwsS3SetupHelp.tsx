import { ExternalLink } from 'lucide-react';

export function AwsS3SetupHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 1: Create an S3 Bucket</h3>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Sign in to the <ExternalLinkInline href="https://console.aws.amazon.com/s3/">AWS S3 Console</ExternalLinkInline></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Click <strong className="text-white">Create bucket</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Enter a <strong className="text-white">bucket name</strong></p>
              <p className="text-xs text-slate-400 mt-1">
                Must be globally unique across all AWS accounts. Use lowercase letters, numbers, and hyphens only.
                Example: <code className="text-blue-300 bg-blue-500/10 px-1 rounded">mycompany-downloads</code>
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Select your preferred <strong className="text-white">AWS Region</strong></p>
              <p className="text-xs text-slate-400 mt-1">
                Choose a region close to your users for faster downloads. You'll need to remember this for the configuration step.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">5</span>
            <div>
              <p>Keep <strong className="text-white">Block all public access</strong> enabled</p>
              <p className="text-xs text-slate-400 mt-1">
                We use presigned URLs for secure downloads, so the bucket should remain private.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">6</span>
            <div>
              <p>Click <strong className="text-white">Create bucket</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 2: Configure CORS</h3>
        <p className="text-slate-300 mb-3">
          CORS (Cross-Origin Resource Sharing) allows browsers to download files from your bucket.
        </p>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Click on your bucket, then go to the <strong className="text-white">Permissions</strong> tab</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Scroll to <strong className="text-white">Cross-origin resource sharing (CORS)</strong> and click Edit</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Paste this configuration:</p>
              <pre className="mt-2 p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`[
  {
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedOrigins": ["*"],
    "ExposeHeaders": ["ETag", "Content-Length"],
    "MaxAgeSeconds": 3600
  }
]`}
              </pre>
              <p className="text-xs text-slate-400 mt-2">
                For production, replace <code className="text-blue-300 bg-blue-500/10 px-1 rounded">"*"</code> in AllowedOrigins with your domain: <code className="text-blue-300 bg-blue-500/10 px-1 rounded">["https://yourdomain.com"]</code>
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Click <strong className="text-white">Save changes</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Next: Create Credentials</h3>
        <p className="text-slate-300">
          You'll need to create IAM credentials to authenticate. Continue to the Credentials step and click the help button there for detailed instructions.
        </p>
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
