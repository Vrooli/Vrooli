import { ExternalLink, AlertTriangle } from 'lucide-react';

export function CloudflareR2SetupHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 1: Find Your Account ID</h3>
        <p className="text-slate-300 mb-3">
          Your Cloudflare Account ID is needed to construct the R2 endpoint URL.
        </p>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Sign in to the <ExternalLinkInline href="https://dash.cloudflare.com/">Cloudflare Dashboard</ExternalLinkInline></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Look at the URL in your browser:</p>
              <code className="block mt-1 text-xs text-blue-300 bg-blue-500/10 px-2 py-1 rounded">
                dash.cloudflare.com/<strong className="text-white">abc123def456</strong>/...
              </code>
              <p className="text-xs text-slate-400 mt-1">
                The string after the first slash is your Account ID.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Alternatively: Click any domain, go to <strong className="text-white">Overview</strong>, scroll to the <strong className="text-white">API</strong> section on the right</p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 2: Create an R2 Bucket</h3>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>In the Cloudflare Dashboard, click <strong className="text-white">R2</strong> in the left sidebar</p>
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
              <p>Enter a bucket name (e.g., <code className="text-blue-300 bg-blue-500/10 px-1 rounded">my-downloads</code>)</p>
              <p className="text-xs text-slate-400 mt-1">
                Bucket names must be lowercase and can contain hyphens.
              </p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Optionally select a location hint (helps with latency, but R2 is global)</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">5</span>
            <div>
              <p>Click <strong className="text-white">Create bucket</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 3: Generate API Token</h3>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>In R2, click <strong className="text-white">Manage R2 API Tokens</strong> (top right)</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Click <strong className="text-white">Create API token</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Name your token (e.g., <code className="text-blue-300 bg-blue-500/10 px-1 rounded">Downloads API</code>)</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Set permissions to <strong className="text-white">Object Read & Write</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">5</span>
            <div>
              <p>Under "Specify bucket(s)", select your bucket for better security (or leave blank for all buckets)</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">6</span>
            <div>
              <p>Click <strong className="text-white">Create API Token</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="font-medium text-amber-300">Save your credentials now!</p>
          <p className="text-amber-200/80 text-xs mt-1">
            Copy both the <strong>Access Key ID</strong> and <strong>Secret Access Key</strong>. The secret key is only shown once!
          </p>
        </div>
      </div>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 4: Configure CORS (Optional)</h3>
        <p className="text-slate-300 mb-3">
          R2 handles CORS automatically for presigned URLs, but you can customize it if needed:
        </p>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Click on your bucket, then go to <strong className="text-white">Settings</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Scroll to <strong className="text-white">CORS Policy</strong> and click <strong className="text-white">Add CORS policy</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Add your domain(s) to the allowed origins</p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Your R2 Endpoint</h3>
        <p className="text-slate-300 mb-2">
          R2 uses this endpoint format (we'll construct it automatically from your Account ID):
        </p>
        <code className="block text-xs text-blue-300 bg-blue-500/10 px-3 py-2 rounded">
          https://<span className="text-white">[account-id]</span>.r2.cloudflarestorage.com
        </code>
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
