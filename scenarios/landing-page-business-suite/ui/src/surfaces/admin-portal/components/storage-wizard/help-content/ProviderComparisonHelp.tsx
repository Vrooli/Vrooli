import { CheckCircle2, XCircle, Minus, ExternalLink } from 'lucide-react';

export function ProviderComparisonHelp() {
  return (
    <div className="space-y-6 text-sm">
      <section>
        <h3 className="text-base font-semibold text-white mb-2">Overview</h3>
        <p className="text-slate-300">
          S3-compatible object storage lets you store files (downloads, assets, uploads) in the
          cloud using a standard API that's supported by dozens of providers. All options below
          work with the same S3 API, so you can switch providers later without changing your code.
        </p>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-3">Quick Recommendation</h3>
        <div className="grid gap-3">
          <RecommendationCard
            label="Just starting out?"
            provider="Cloudflare R2"
            reason="Free egress, simple pricing, generous free tier"
          />
          <RecommendationCard
            label="Enterprise / AWS ecosystem?"
            provider="AWS S3"
            reason="Most features, best AWS integration, proven at scale"
          />
          <RecommendationCard
            label="Privacy / self-hosted?"
            provider="MinIO"
            reason="Full control, runs on your own server, open source"
          />
          <RecommendationCard
            label="Other cloud providers?"
            provider="Other S3-Compatible"
            reason="DigitalOcean Spaces, Backblaze B2, Wasabi, and more"
          />
        </div>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-3">Comparison</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-white/10">
                <th className="py-2 pr-4 font-medium text-slate-400">Feature</th>
                <th className="py-2 px-2 font-medium text-slate-400">AWS S3</th>
                <th className="py-2 px-2 font-medium text-slate-400">R2</th>
                <th className="py-2 px-2 font-medium text-slate-400">MinIO</th>
              </tr>
            </thead>
            <tbody className="text-slate-300">
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Storage cost</td>
                <td className="py-2 px-2">~$0.023/GB</td>
                <td className="py-2 px-2">$0.015/GB</td>
                <td className="py-2 px-2">Free*</td>
              </tr>
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Egress cost</td>
                <td className="py-2 px-2 text-amber-400">$0.09/GB</td>
                <td className="py-2 px-2 text-emerald-400">Free</td>
                <td className="py-2 px-2 text-emerald-400">Free*</td>
              </tr>
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Free tier</td>
                <td className="py-2 px-2">5GB / 12mo</td>
                <td className="py-2 px-2">10GB forever</td>
                <td className="py-2 px-2">Unlimited*</td>
              </tr>
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Setup difficulty</td>
                <td className="py-2 px-2 flex items-center gap-1">
                  <span className="text-amber-400">Medium</span>
                </td>
                <td className="py-2 px-2">
                  <span className="text-emerald-400">Easy</span>
                </td>
                <td className="py-2 px-2">
                  <span className="text-amber-400">Medium</span>
                </td>
              </tr>
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Global CDN</td>
                <td className="py-2 px-2"><FeatureCheck yes /></td>
                <td className="py-2 px-2"><FeatureCheck yes /></td>
                <td className="py-2 px-2"><FeatureCheck no /></td>
              </tr>
              <tr className="border-b border-white/5">
                <td className="py-2 pr-4 text-slate-400">Self-hosted</td>
                <td className="py-2 px-2"><FeatureCheck no /></td>
                <td className="py-2 px-2"><FeatureCheck no /></td>
                <td className="py-2 px-2"><FeatureCheck yes /></td>
              </tr>
              <tr>
                <td className="py-2 pr-4 text-slate-400">Best for</td>
                <td className="py-2 px-2 text-xs">Enterprise, AWS users</td>
                <td className="py-2 px-2 text-xs">Cost-conscious, high egress</td>
                <td className="py-2 px-2 text-xs">Privacy, full control</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p className="text-xs text-slate-500 mt-2">
          * Self-hosted means you pay for your own server/storage instead of per-GB fees
        </p>
      </section>

      <section>
        <h3 className="text-base font-semibold text-white mb-2">Other S3-Compatible Providers</h3>
        <p className="text-slate-300 mb-3">
          Many other providers offer S3-compatible APIs. Select "Other S3-Compatible" and enter
          their endpoint URL:
        </p>
        <ul className="space-y-2 text-slate-300">
          <li className="flex items-start gap-2">
            <span className="text-blue-400">DigitalOcean Spaces</span>
            <span className="text-slate-500">— Simple pricing, good for DO users</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-400">Backblaze B2</span>
            <span className="text-slate-500">— Very cheap storage ($0.005/GB)</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-400">Wasabi</span>
            <span className="text-slate-500">— No egress fees, flat $0.0059/GB</span>
          </li>
          <li className="flex items-start gap-2">
            <span className="text-blue-400">Linode Object Storage</span>
            <span className="text-slate-500">— Good for Linode/Akamai users</span>
          </li>
        </ul>
      </section>

      <section className="border-t border-white/10 pt-4">
        <h3 className="text-base font-semibold text-white mb-3">Official Documentation</h3>
        <p className="text-slate-400 text-xs mb-3">
          Pricing and features may change. Refer to official documentation for the latest information:
        </p>
        <ul className="space-y-2">
          <li>
            <ExternalLinkInline href="https://aws.amazon.com/s3/pricing/">
              AWS S3 Pricing
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://developers.cloudflare.com/r2/pricing/">
              Cloudflare R2 Pricing
            </ExternalLinkInline>
          </li>
          <li>
            <ExternalLinkInline href="https://github.com/minio/minio">
              MinIO GitHub (Open Source)
            </ExternalLinkInline>
          </li>
        </ul>
      </section>
    </div>
  );
}

function RecommendationCard({
  label,
  provider,
  reason,
}: {
  label: string;
  provider: string;
  reason: string;
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border border-white/10 bg-white/5 p-3">
      <div className="flex-1">
        <p className="text-slate-400 text-xs">{label}</p>
        <p className="text-white font-medium">{provider}</p>
        <p className="text-slate-400 text-xs mt-0.5">{reason}</p>
      </div>
    </div>
  );
}

function FeatureCheck({ yes, partial }: { yes?: boolean; partial?: boolean; no?: boolean }) {
  if (yes) {
    return <CheckCircle2 className="h-4 w-4 text-emerald-400" />;
  }
  if (partial) {
    return <Minus className="h-4 w-4 text-amber-400" />;
  }
  return <XCircle className="h-4 w-4 text-slate-500" />;
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
