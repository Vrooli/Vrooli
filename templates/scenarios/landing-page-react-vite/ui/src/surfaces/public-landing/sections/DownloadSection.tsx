import { useMemo, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import { useEntitlements } from '../../../shared/hooks/useEntitlements';
import type { DownloadApp, DownloadAsset } from '../../../shared/api';
import { requestDownload } from '../../../shared/api';

interface DownloadSectionProps {
  content?: {
    title?: string;
    subtitle?: string;
    description?: string;
  };
  downloads?: DownloadApp[];
}

type DownloadStatus = {
  loading: boolean;
  message?: string;
};

const PLATFORM_LABELS: Record<string, string> = {
  windows: 'Windows',
  mac: 'macOS',
  linux: 'Linux',
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
  return `app-${download.app_key}-${download.platform}-${version}-${artifact}`;
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

function formatPlatformLabel(platform: string) {
  return PLATFORM_LABELS[platform.toLowerCase()] ?? platform;
}

function hasInstallTargets(app: DownloadApp) {
  return (app.platforms?.length ?? 0) > 0 || (app.storefronts?.length ?? 0) > 0;
}

export function DownloadSection({ content, downloads }: DownloadSectionProps) {
  const filteredApps = (downloads ?? []).filter(hasInstallTargets);
  if (filteredApps.length === 0) {
    return null;
  }

  const title = content?.title || 'Download & install the bundle';
  const subtitle =
    content?.subtitle ||
    'Entitlement-aware installers, app store links, and install steps keep every runtime consistent.';
  const { trackDownload } = useMetrics();
  const { email, setEmail, entitlements, loading: entitlementsLoading, error: entitlementsError, refresh } =
    useEntitlements();
  const [downloadStatus, setDownloadStatus] = useState<Record<string, DownloadStatus>>({});

  const trimmedEmail = email.trim();
  const allPlatforms = useMemo(
    () => filteredApps.flatMap((app) => app.platforms ?? []),
    [filteredApps],
  );
  const entitlementsRequired = allPlatforms.some((asset) => asset.requires_entitlement);
  const hasInstallers = allPlatforms.length > 0;
  const anyStorefronts = filteredApps.some((app) => (app.storefronts?.length ?? 0) > 0);

  const hasActiveSubscription = entitlements?.status === 'active' || entitlements?.status === 'trialing';
  const entitlementStatus = entitlements?.status ?? 'unknown';
  const planTier = entitlements?.plan_tier ?? entitlements?.subscription?.plan_tier ?? 'unknown';
  const creditsLabel = entitlements?.credits?.display_credits_label;
  const creditsMultiplier = entitlements?.credits?.display_credits_multiplier;
  const balanceDisplay = formatCreditDisplay(entitlements?.credits?.balance_credits, creditsMultiplier, creditsLabel);
  const bonusDisplay = formatCreditDisplay(entitlements?.credits?.bonus_credits, creditsMultiplier, creditsLabel);

  const showEntitlementSummary = entitlementsRequired && trimmedEmail.length > 0;

  const handleDownload = async (app: DownloadApp, download: DownloadAsset, assetKey: string) => {
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
        download.app_key || app.app_key,
        download.platform,
        download.requires_entitlement ? trimmedEmail : undefined,
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
        app_key: app.app_key,
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
    <section className="border-t border-white/5 bg-[#0F172A] py-24 text-white" data-testid="downloads-section">
      <div className="container mx-auto px-6">
        <div className="mb-12 max-w-3xl space-y-3">
          <p className="text-xs uppercase tracking-[0.35em] text-slate-500">Downloads & installs</p>
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

        <div className="space-y-8">
          {filteredApps.map((app) => {
            const installers = app.platforms ?? [];
            const storefronts = app.storefronts ?? [];
            return (
              <div
                key={app.app_key}
                className="rounded-3xl border border-white/10 bg-[#07090F] p-8"
                data-testid={`download-app-${app.app_key}`}
              >
                <div className="flex flex-col gap-8 lg:flex-row lg:items-start">
                  <div className="flex-1 space-y-4">
                    <div className="space-y-1">
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Bundle app</p>
                      <h3 className="text-2xl font-semibold text-white">{app.name}</h3>
                      <p className="text-sm text-slate-400">
                        {app.tagline ||
                          'Offline-safe companion that inherits entitlement checks and telemetry instrumentation.'}
                      </p>
                    </div>
                    {app.description && <p className="text-sm text-slate-300">{app.description}</p>}
                    {app.install_overview && (
                      <div className="rounded-2xl border border-white/5 bg-white/5 p-4 text-sm text-slate-200">
                        {app.install_overview}
                      </div>
                    )}
                    {app.install_steps && app.install_steps.length > 0 && (
                      <ol className="list-decimal list-inside space-y-1 text-sm text-slate-400" data-testid={`install-steps-${app.app_key}`}>
                        {app.install_steps.map((step, idx) => (
                          <li key={`${app.app_key}-step-${idx}`}>{step}</li>
                        ))}
                      </ol>
                    )}
                    {storefronts.length > 0 && (
                      <div className="flex flex-wrap gap-3 pt-2">
                        {storefronts.map((store) => (
                          <a
                            key={`${app.app_key}-${store.store}-${store.url}`}
                            href={store.url}
                            target="_blank"
                            rel="noreferrer"
                            className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white transition hover:border-white/20 hover:bg-white/10"
                            data-testid={`storefront-${app.app_key}-${store.store}`}
                          >
                            {store.badge || store.label}
                          </a>
                        ))}
                      </div>
                    )}
                  </div>

                  {installers.length > 0 && (
                    <div className="flex-1 space-y-4" data-testid={`installer-list-${app.app_key}`}>
                      <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Desktop installers</p>
                      <div className="grid gap-4 md:grid-cols-2">
                        {installers.map((installer) => {
                          const assetKey = getDownloadAssetKey(installer);
                          const status = downloadStatus[assetKey];
                          const buttonLabel = installer.requires_entitlement ? 'Verify & Download' : 'Download';
                          return (
                            <div
                              key={assetKey}
                              className="space-y-3 rounded-2xl border border-white/10 bg-[#05070E] p-4"
                              data-testid={`download-card-${assetKey}`}
                            >
                              <div className="flex items-center justify-between gap-2">
                                <span className="text-xs uppercase tracking-[0.3em] text-slate-500">
                                  {formatPlatformLabel(installer.platform)}
                                </span>
                                <span className="text-xs text-slate-400">{installer.release_version}</span>
                              </div>
                              <p className="text-sm text-slate-300">
                                {installer.release_notes || 'Release notes coming soon.'}
                              </p>
                              <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                                <span>{installer.requires_entitlement ? 'Entitlement required' : 'Free download'}</span>
                                {installer.metadata?.size_mb && <span>{installer.metadata.size_mb} MB</span>}
                              </div>
                              <Button
                                variant="default"
                                size="sm"
                                onClick={() => handleDownload(app, installer, assetKey)}
                                disabled={status?.loading}
                                className={`w-full ${installer.requires_entitlement ? '' : 'bg-[#F97316]'}`}
                              >
                                {status?.loading ? 'Preparing…' : buttonLabel}
                              </Button>
                              {status?.message && <p className="text-xs text-slate-400">{status.message}</p>}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        {(hasInstallers || anyStorefronts) && (
          <div className="mx-auto mt-12 max-w-4xl rounded-3xl border border-white/10 bg-[#07090F] p-6 text-center text-sm text-slate-300">
            Download access inherits your Vrooli Ascension entitlements. Need additional seats or bundle access?
            Contact support before sharing installers so telemetry and gating stay accurate.
          </div>
        )}
      </div>
    </section>
  );
}
