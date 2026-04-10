import { AlertTriangle, CheckCircle2, ExternalLink } from 'lucide-react';

export function TroubleshootingHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Common Connection Issues</h3>
        <p className="text-slate-300">
          Connection tests can fail for various reasons. Here are the most common issues and their solutions.
        </p>
      </section>

      <TroubleshootingItem
        title="Access Denied / Invalid Credentials"
        symptoms={['Error: Access Denied', 'Error: SignatureDoesNotMatch', 'Error: InvalidAccessKeyId']}
        solutions={[
          'Double-check your Access Key ID and Secret Access Key for typos',
          'Ensure the credentials have the necessary permissions (s3:GetObject, s3:PutObject, s3:ListBucket)',
          'For AWS: Verify the IAM user/role has the correct policy attached',
          'For R2: Verify the API token has "Object Read & Write" permissions',
          'Make sure you\'re not using expired or rotated credentials',
        ]}
      />

      <TroubleshootingItem
        title="Bucket Not Found"
        symptoms={['Error: NoSuchBucket', 'Error: The specified bucket does not exist']}
        solutions={[
          'Verify the bucket name is spelled correctly (case-sensitive)',
          'For AWS: Confirm the bucket exists in the region you selected',
          'For R2: Bucket names don\'t include the account ID — just the bucket name',
          'Make sure the bucket hasn\'t been deleted or renamed',
        ]}
      />

      <TroubleshootingItem
        title="CORS Errors"
        symptoms={['Error: CORS policy blocks the request', 'Browser console shows CORS errors', 'Downloads work from server but not browser']}
        solutions={[
          'Configure CORS on your bucket to allow your domain',
          'For testing, you can temporarily allow all origins: ["*"]',
          'Include the correct HTTP methods: GET, HEAD (and PUT if uploading)',
          'For R2: CORS is often auto-configured for presigned URLs, but custom domains need explicit CORS',
        ]}
      />

      <TroubleshootingItem
        title="Connection Timeout / Network Error"
        symptoms={['Error: Network Error', 'Error: Request timeout', 'Connection refused']}
        solutions={[
          'Verify the endpoint URL is correct and accessible',
          'For MinIO: Ensure the server is running and the port is open',
          'Check if a firewall is blocking the connection',
          'For self-hosted: Verify your server has a valid SSL certificate if using HTTPS',
        ]}
      />

      <TroubleshootingItem
        title="Invalid Endpoint"
        symptoms={['Error: Invalid endpoint', 'Error: Hostname resolution failed', 'Error: getaddrinfo ENOTFOUND']}
        solutions={[
          'Check that the endpoint URL is correctly formatted (include https://)',
          'For AWS S3: You don\'t need to specify an endpoint — we use the region to construct it',
          'For R2: Endpoint is constructed automatically from your Account ID',
          'For custom providers: Use the full URL including protocol and port if needed',
        ]}
      />

      <TroubleshootingItem
        title="Path-Style vs Virtual-Hosted Style"
        symptoms={['Error: The bucket you are attempting to access must be addressed using the specified endpoint', 'DNS resolution errors for bucket.endpoint.com']}
        solutions={[
          'Enable "Force path-style URLs" for MinIO and most self-hosted solutions',
          'AWS S3 uses virtual-hosted style by default (leave path-style disabled)',
          'If you\'re getting DNS errors, try enabling path-style',
        ]}
      />

      <section>
        <h3 className="text-base font-semibold text-white mb-3">CORS Configuration Examples</h3>
        <p className="text-slate-300 mb-3">
          Here are working CORS configurations for each provider:
        </p>

        <div className="space-y-4">
          <div>
            <h4 className="text-white font-medium mb-2">AWS S3</h4>
            <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`[
  {
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedOrigins": ["https://yourdomain.com"],
    "ExposeHeaders": ["ETag", "Content-Length"],
    "MaxAgeSeconds": 3600
  }
]`}
            </pre>
          </div>

          <div>
            <h4 className="text-white font-medium mb-2">Cloudflare R2</h4>
            <p className="text-xs text-slate-400 mb-2">
              R2 uses a simplified CORS UI. Add your domain to the allowed origins list in bucket settings.
            </p>
          </div>

          <div>
            <h4 className="text-white font-medium mb-2">MinIO</h4>
            <p className="text-xs text-slate-400 mb-2">
              Configure CORS via the mc client:
            </p>
            <pre className="p-3 rounded-lg bg-slate-800 text-slate-300 text-xs overflow-x-auto">
{`mc admin config set myminio api cors_allow_origin="https://yourdomain.com"`}
            </pre>
          </div>
        </div>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Still Having Issues?</h3>
        <div className="space-y-2 text-slate-300">
          <p>Try these debugging steps:</p>
          <ol className="list-decimal list-inside space-y-1 text-slate-400">
            <li>Test credentials using your provider's CLI tool (aws s3, wrangler r2, mc)</li>
            <li>Check browser developer tools (Network tab) for detailed error messages</li>
            <li>Verify your server can reach the storage endpoint (no firewall blocking)</li>
            <li>Try the connection test with a minimal bucket policy first, then restrict</li>
          </ol>
        </div>
      </section>

      <section className="border-t border-white/10 pt-4">
        <h3 className="text-base font-semibold text-white mb-3">Official CORS Documentation</h3>
        <ul className="space-y-2">
          <li>
            <ExternalLinkInline href="https://docs.aws.amazon.com/AmazonS3/latest/userguide/enabling-cors-examples.html">
              AWS S3 CORS Configuration
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://developers.cloudflare.com/r2/buckets/cors/">
              Cloudflare R2 CORS Configuration
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS">
              MDN: Cross-Origin Resource Sharing (CORS)
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

function TroubleshootingItem({
  title,
  symptoms,
  solutions,
}: {
  title: string;
  symptoms: string[];
  solutions: string[];
}) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 space-y-3">
      <h4 className="font-semibold text-white">{title}</h4>

      <div>
        <p className="text-xs font-medium text-slate-400 mb-1.5">Symptoms:</p>
        <ul className="space-y-1">
          {symptoms.map((symptom, i) => (
            <li key={i} className="flex items-start gap-2 text-slate-300 text-xs">
              <AlertTriangle className="h-3 w-3 text-amber-400 flex-shrink-0 mt-0.5" />
              <code className="text-amber-200/80">{symptom}</code>
            </li>
          ))}
        </ul>
      </div>

      <div>
        <p className="text-xs font-medium text-slate-400 mb-1.5">Solutions:</p>
        <ul className="space-y-1">
          {solutions.map((solution, i) => (
            <li key={i} className="flex items-start gap-2 text-slate-300 text-xs">
              <CheckCircle2 className="h-3 w-3 text-emerald-400 flex-shrink-0 mt-0.5" />
              <span>{solution}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
