import { useMemo, useState } from 'react';
import { Button } from '../../../components/ui/button';
import { useMetrics } from '../../../hooks/useMetrics';
import { useEntitlements } from '../../../hooks/useEntitlements';
import type { DownloadAsset } from '../../../services/api';
import { requestDownload } from '../../../services/api';

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

  const handleDownload = async (download: DownloadAsset) => {
    const platformKey = download.platform;
    setDownloadStatus((prev) => ({
      ...prev,
      [platformKey]: { loading: true },
    }));

    if (download.requires_entitlement && !trimmedEmail) {
      setDownloadStatus((prev) => ({
        ...prev,
        [platformKey]: {
          loading: false,
          message: 'Enter the email tied to your subscription to unlock this download.',
        },
      }));
      return;
    }

    if (download.requires_entitlement && !hasActiveSubscription) {
      setDownloadStatus((prev) => ({
        ...prev,
        [platformKey]: {
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

      trackDownload({
        platform: download.platform,
        release_version: download.release_version,
        requires_entitlement: download.requires_entitlement,
      });

      window.open(asset.artifact_url, '_blank');

      setDownloadStatus((prev) => ({
        ...prev,
        [platformKey]: {
          loading: false,
          message: 'Download started in a new tab.',
        },
      }));
    } catch (err) {
      setDownloadStatus((prev) => ({
        ...prev,
        [platformKey]: {
          loading: false,
          message: err instanceof Error ? err.message : 'Failed to prepare download.',
        },
      }));
    }
  };

  return (
    <section className="py-20 bg-slate-950 text-white">
      <div className="container mx-auto px-6">
        <div className="max-w-4xl mx-auto text-center mb-12 space-y-3">
          <p className="text-sm uppercase tracking-[0.5em] text-slate-500">Downloads</p>
          <h2 className="text-4xl font-bold">{title}</h2>
          <p className="text-slate-400">{subtitle}</p>
        </div>

        {entitlementsRequired && (
          <div className="max-w-3xl mx-auto mb-8 rounded-2xl border border-white/10 bg-white/5 p-6 text-left space-y-4">
            <p className="text-sm text-slate-300 mb-2">
              Entitlement gated downloads require an active subscription. Provide the email tied to your plan and we'll verify your access before allowing the download.
            </p>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="you@example.com"
                className="flex-1 px-4 py-3 bg-slate-900 border border-white/10 rounded-xl focus:border-blue-500 focus:outline-none"
              />
              <Button
                variant="outline"
                size="sm"
                className="whitespace-nowrap"
                onClick={refresh}
                disabled={!trimmedEmail || entitlementsLoading}
              >
                {entitlementsLoading ? 'Checking status...' : 'Refresh status'}
              </Button>
            </div>
            <p className="text-xs text-slate-400">Stored locally on this device only.</p>
            {entitlementsError && (
              <p className="text-xs text-rose-400">{entitlementsError}</p>
            )}
            {showEntitlementSummary && (
              <div className="grid gap-4 md:grid-cols-3">
                <div className="rounded-xl border border-white/10 bg-slate-900/60 p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Subscription</p>
                  <p className="text-lg font-semibold text-white">{entitlementStatus}</p>
                  <p className="text-xs text-slate-500 mt-1">Plan tier: {planTier}</p>
                  {entitlements?.subscription?.subscription_id && (
                    <p className="text-xs text-slate-500 mt-1">ID: {entitlements.subscription.subscription_id}</p>
                  )}
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/60 p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Credits</p>
                  <p className="text-lg font-semibold text-white">{balanceDisplay}</p>
                  <p className="text-xs text-slate-500 mt-1">Bonus: {bonusDisplay}</p>
                </div>
                <div className="rounded-xl border border-white/10 bg-slate-900/60 p-4">
                  <p className="text-xs text-slate-400 uppercase tracking-[0.3em]">Pricing</p>
                  <p className="text-lg font-semibold text-white">
                    {entitlements?.price_id ?? 'Unknown price'}
                  </p>
                  <p className="text-xs text-slate-500 mt-1">
                    Updated {entitlements?.subscription?.updated_at ?? 'recently'}
                  </p>
                </div>
              </div>
            )}
          </div>
        )}

        <div className="grid md:grid-cols-3 gap-6">
          {downloads.map((download) => {
            const status = downloadStatus[download.platform];
            const buttonLabel = download.requires_entitlement ? 'Verify & Download' : 'Download';

            return (
              <div
                key={download.platform}
                className="rounded-3xl border border-white/10 bg-white/5 p-6 space-y-4"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs uppercase tracking-[0.3em] text-slate-500">{download.platform}</span>
                  <span className="text-xs text-slate-400">{download.release_version}</span>
                </div>
                <p className="text-sm text-slate-300 min-h-[3rem]">
                  {download.release_notes || 'Release notes coming soon.'}
                </p>
                <div className="text-xs text-slate-400 flex flex-wrap gap-2">
                  <span>
                    {download.requires_entitlement ? 'Entitlement required' : 'Free download'}
                  </span>
                  {download.metadata?.size_mb && (
                    <span>{download.metadata.size_mb} MB download</span>
                  )}
                </div>
                <Button
                  variant="outline"
                  size="lg"
                  onClick={() => handleDownload(download)}
                  disabled={status?.loading}
                  className="w-full"
                >
                  {status?.loading ? 'Preparing...' : buttonLabel}
                </Button>
                {status?.message && (
                  <p className="text-xs text-slate-400">{status.message}</p>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
