import { ExternalLink, AlertTriangle } from 'lucide-react';

export function MinioSetupHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">What is MinIO?</h3>
        <p className="text-slate-300">
          MinIO is an open-source, self-hosted object storage server that's fully S3-compatible.
          It runs on your own infrastructure, giving you full control over your data.
        </p>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 1: Install MinIO</h3>
        <p className="text-slate-300 mb-3">
          The easiest way to run MinIO is with Docker:
        </p>
        <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`docker run -d \\
  --name minio \\
  -p 9000:9000 \\
  -p 9001:9001 \\
  -v minio-data:/data \\
  -e MINIO_ROOT_USER=minioadmin \\
  -e MINIO_ROOT_PASSWORD=minioadmin \\
  quay.io/minio/minio server /data --console-address ":9001"`}
        </pre>
        <p className="text-xs text-slate-400 mt-2">
          This starts MinIO with the API on port 9000 and the web console on port 9001.
        </p>
      </section>

      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-4">
        <AlertTriangle className="h-5 w-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div>
          <p className="font-medium text-amber-300">Change the default credentials!</p>
          <p className="text-amber-200/80 text-xs mt-1">
            Replace <code className="bg-amber-500/20 px-1 rounded">minioadmin</code> with strong, unique credentials for production use.
          </p>
        </div>
      </div>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 2: Access the Console</h3>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Open <code className="text-blue-300 bg-blue-500/10 px-1 rounded">http://localhost:9001</code> in your browser</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Log in with your root credentials (default: <code className="text-blue-300 bg-blue-500/10 px-1 rounded">minioadmin</code> / <code className="text-blue-300 bg-blue-500/10 px-1 rounded">minioadmin</code>)</p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 3: Create a Bucket</h3>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>In the console, click <strong className="text-white">Buckets</strong> in the sidebar</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Click <strong className="text-white">Create Bucket</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Enter a bucket name (e.g., <code className="text-blue-300 bg-blue-500/10 px-1 rounded">downloads</code>)</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Click <strong className="text-white">Create Bucket</strong></p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Step 4: Create Access Keys</h3>
        <p className="text-slate-300 mb-3">
          For production, create a dedicated service account instead of using root credentials:
        </p>
        <ol className="space-y-3 text-slate-300">
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">1</span>
            <div>
              <p>Click <strong className="text-white">Access Keys</strong> in the sidebar</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">2</span>
            <div>
              <p>Click <strong className="text-white">Create access key</strong></p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">3</span>
            <div>
              <p>Optionally restrict the policy to specific buckets</p>
            </div>
          </li>
          <li className="flex gap-3">
            <span className="flex-shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-blue-500/20 text-blue-400 text-xs font-medium">4</span>
            <div>
              <p>Click <strong className="text-white">Create</strong> and save both the Access Key and Secret Key</p>
            </div>
          </li>
        </ol>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Configuration Notes</h3>
        <div className="space-y-3 text-slate-300">
          <div className="rounded-lg border border-white/10 bg-white/5 p-3">
            <p className="font-medium text-white">Endpoint URL</p>
            <p className="text-xs text-slate-400 mt-1">
              Use your server's URL with port 9000, e.g., <code className="text-blue-300 bg-blue-500/10 px-1 rounded">https://minio.example.com:9000</code>
            </p>
          </div>
          <div className="rounded-lg border border-white/10 bg-white/5 p-3">
            <p className="font-medium text-white">Path-Style URLs</p>
            <p className="text-xs text-slate-400 mt-1">
              MinIO typically requires path-style URLs. Enable "Force path-style URLs" in the configuration step.
            </p>
          </div>
          <div className="rounded-lg border border-white/10 bg-white/5 p-3">
            <p className="font-medium text-white">HTTPS</p>
            <p className="text-xs text-slate-400 mt-1">
              For production, configure MinIO with TLS certificates or put it behind a reverse proxy with HTTPS.
            </p>
          </div>
        </div>
      </section>

      <section>
        <p className="text-slate-400">
          For more details, see the{' '}
          <ExternalLinkInline href="https://min.io/docs/minio/container/index.html">
            MinIO Documentation
          </ExternalLinkInline>
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
