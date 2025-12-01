import { useMemo, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import { useEntitlements } from '../../../shared/hooks/useEntitlements';
import type { DownloadAsset } from '../../../shared/api';
import { requestDownload } from '../../../shared/api';

interface DownloadSectionProps {
  content?: {
    title?: string;
    subtitle?: string;
    description?: string;
  };
  downloads?: DownloadAsset[];
}

type DownloadStatus = {
  loading: boolean;
  message?: string;
};

function formatCreditDisplay(amount?: number, multiplier?: number, label?: string) {
  const scaled = Math.round((amount ?? 0) * (multiplier ?? 1));
  const displayLabel = label || 'credits';

  if (scaled >= 1_000_000) {
    return `${(scaled / 1_000_000).toFixed(1).replace(/\.0$/, '')}M ${displayLabel}`;
  }
  if (scaled >= 1_000) {
    return `${(scaled / 1_000).toFixed(1).replace(/\.0$/, '')}k ${displayLabel}`;
  }

  return `${scaled.toLocaleString()} ${displayLabel}`;
}

export function getDownloadAssetKey(download: DownloadAsset): string {
  if (typeof download.id === 'number') {
    return `asset-${download.id}`;
  }
  const version = download.release_version || 'unknown';
  const artifact = download.artifact_url || 'na';
  return `platform-${download.platform}-${version}-${artifact}`;
}

function sanitizeArtifactUrl(artifactUrl?: string) {
  if (typeof artifactUrl !== 'string') {
    return '';
  }

  const trimmed = artifactUrl.trim();
  if (!trimmed) {
    return '';
  }

  const lower = trimmed.toLowerCase();
  if (lower.startsWith('javascript:') || lower.startsWith('data:') || lower.startsWith('vbscript:')) {
    return '';
  }

  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed;
  }

  if (/^[./]/.test(trimmed) || !trimmed.includes(':')) {
    return trimmed;
  }

  return '';
}

function openDownloadWindow(url: string) {
  if (typeof window === 'undefined' || typeof window.open !== 'function') {
    return false;
  }
  const target = window.open(url, '_blank', 'noopener,noreferrer');
  return target !== null;
}

export function DownloadSection({ content, downloads }: DownloadSectionProps) {
  if (!downloads || downloads.length === 0) {
    return null;
  }

  const title = content?.title || 'Download the bundle';
  const subtitle = content?.subtitle || 'Choose your platform and unlock the Browser Automation Studio experience.';
  const { trackDownload } = useMetrics();
  const { email, setEmail, entitlements, loading: entitlementsLoading, error: entitlementsError, refresh } =
    useEntitlements();
  const [downloadStatus, setDownloadStatus] = useState<Record<string, DownloadStatus>>({});

  const trimmedEmail = email.trim();
  const entitlementsRequired = useMemo(
    () => downloads.some((download) => download.requires_entitlement),
    [downloads]
  );
  const hasActiveSubscription = entitlements?.status === 'active' || entitlements?.status === 'trialing';
  const entitlementStatus = entitlements?.status ?? 'unknown';
  const planTier = entitlements?.plan_tier ?? entitlements?.subscription?.plan_tier ?? 'unknown';
  const creditsLabel = entitlements?.credits?.display_credits_label;
  const creditsMultiplier = entitlements?.credits?.display_credits_multiplier;
  const balanceDisplay = formatCreditDisplay(entitlements?.credits?.balance_credits, creditsMultiplier, creditsLabel);
  const bonusDisplay = formatCreditDisplay(entitlements?.credits?.bonus_credits, creditsMultiplier, creditsLabel);

  const showEntitlementSummary = entitlementsRequired && trimmedEmail.length > 0;

  const handleDownload = async (download: DownloadAsset, assetKey: string) => {
    setDownloadStatus((prev) => ({
      ...prev,
      [assetKey]: { loading: true },
    }));

    if (download.requires_entitlement && !trimmedEmail) {
      setDownloadStatus((prev) => ({
        ...prev,
        [assetKey]: {
          loading: false,
          message: 'Enter the email tied to your subscription to unlock this download.',
        },
      }));
      return;
    }

    if (download.requires_entitlement && !hasActiveSubscription) {
      setDownloadStatus((prev) => ({
        ...prev,
        [assetKey]: {
          loading: false,
          message: `Subscription status is ${entitlementStatus}. Refresh to verify access before downloading.`,
        },
      }));
      return;
    }

    try {
      const asset = await requestDownload(
        download.platform,
        download.requires_entitlement ? trimmedEmail : undefined
      );

      const artifactUrl = sanitizeArtifactUrl(asset?.artifact_url);
      if (!artifactUrl) {
        setDownloadStatus((prev) => ({
          ...prev,
          [assetKey]: {
            loading: false,
            message: 'Download artifact is not available yet. Please try again later.',
          },
        }));
        return;
      }

      const opened = openDownloadWindow(artifactUrl);
      if (!opened) {
        setDownloadStatus((prev) => ({
          ...prev,
          [assetKey]: {
            loading: false,
            message: 'Unable to open download. Allow pop-ups and try again.',
          },
        }));
        return;
      }

      trackDownload({
        platform: download.platform,
        release_version: download.release_version,
        requires_entitlement: download.requires_entitlement,
      });

      setDownloadStatus((prev) => ({
        ...prev,
        [assetKey]: {
          loading: false,
          message: 'Download started in a new tab.',
        },
      }));
    } catch (err) {
      setDownloadStatus((prev) => ({
        ...prev,
        [assetKey]: {
          loading: false,
          message: err instanceof Error ? err.message : 'Failed to prepare download.',
        },
      }));
    }
  };

  return (
    <section className="border-t border-white/5 bg-[#0F172A] py-24 text-white">
      <div className="container mx-auto px-6">
        <div className="mb-12 max-w-3xl space-y-3">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Downloads</p>
          <h2 className="text-4xl font-semibold">{title}</h2>
          <p className="text-slate-300">{subtitle}</p>
        </div>

        {entitlementsRequired && (
          <div className="mx-auto mb-8 max-w-3xl space-y-4 rounded-3xl border border-white/10 bg-[#07090F] p-6 text-left">
            <p className="text-sm text-slate-300">
              Entitlement gated downloads require an active subscription. Provide the email tied to your plan and we'll verify your access before allowing the download.
            </p>
            <div className="flex flex-col gap-4 md:flex-row md:items-center">
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="you@company.com"
                className="flex-1 rounded-xl border border-white/10 bg-[#07090F] px-4 py-3 text-white placeholder-slate-500 focus:border-[#F97316] focus:outline-none focus:ring-1 focus:ring-[#F97316]"
              />
              <Button
                variant="ghost"
                size="sm"
                className="whitespace-nowrap bg-white/5 text-white hover:bg-white/10"
                onClick={refresh}
                disabled={!trimmedEmail || entitlementsLoading}
              >
                {entitlementsLoading ? 'Checking status…' : 'Refresh status'}
              </Button>
            </div>
            <p className="text-xs text-slate-500">Stored locally on this device only.</p>
            {entitlementsError && <p className="text-xs text-rose-400">{entitlementsError}</p>}
            {showEntitlementSummary && (
              <div className="grid gap-4 md:grid-cols-3">
                <div className="rounded-xl border border-white/10 bg-[#0F172A] p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Subscription</p>
                  <p className="text-lg font-semibold text-white">{entitlementStatus}</p>
                  <p className="mt-1 text-xs text-slate-500">Plan tier: {planTier}</p>
                  {entitlements?.subscription?.subscription_id && (
                    <p className="mt-1 text-xs text-slate-500">ID: {entitlements.subscription.subscription_id}</p>
                  )}
                </div>
                <div className="rounded-xl border border-white/10 bg-[#0F172A] p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Credits</p>
                  <p className="text-lg font-semibold text-white">{balanceDisplay}</p>
                  <p className="mt-1 text-xs text-slate-500">Bonus: {bonusDisplay}</p>
                </div>
                <div className="rounded-xl border border-white/10 bg-[#0F172A] p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Pricing</p>
                  <p className="text-lg font-semibold text-white">{entitlements?.price_id ?? 'Unknown price'}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    Updated {entitlements?.subscription?.updated_at ?? 'recently'}
                  </p>
                </div>
              </div>
            )}
          </div>
        )}

        <div className="grid gap-6 md:grid-cols-3">
          {downloads.map((download) => {
            const assetKey = getDownloadAssetKey(download);
            const status = downloadStatus[assetKey];
            const buttonLabel = download.requires_entitlement ? 'Verify & Download' : 'Download';

            return (
              <div
                key={assetKey}
                className="space-y-4 rounded-3xl border border-white/10 bg-[#07090F] p-6"
                data-testid={`download-card-${assetKey}`}
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs uppercase tracking-[0.3em] text-slate-500">{download.platform}</span>
                  <span className="text-xs text-slate-400">{download.release_version}</span>
                </div>
                <p className="min-h-[3rem] text-sm text-slate-300">
                  {download.release_notes || 'Release notes coming soon.'}
                </p>
                <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                  <span>{download.requires_entitlement ? 'Entitlement required' : 'Free download'}</span>
                  {download.metadata?.size_mb && <span>{download.metadata.size_mb} MB download</span>}
                </div>
                <Button
                  variant="default"
                  size="lg"
                  onClick={() => handleDownload(download, assetKey)}
                  disabled={status?.loading}
                  className={`w-full ${download.requires_entitlement ? '' : 'bg-[#F97316]'}`}
                >
                  {status?.loading ? 'Preparing…' : buttonLabel}
                </Button>
                {status?.message && <p className="text-xs text-slate-400">{status.message}</p>}
              </div>
            );
          })}
        </div>

        <div className="mx-auto mt-12 max-w-4xl rounded-3xl border border-white/10 bg-[#07090F] p-6 text-center text-sm text-slate-300">
          Download access is tied to your Browser Automation Studio entitlements. Need additional access? Contact support to add more seats or credits.
        </div>
      </div>
    </section>
  );
}
