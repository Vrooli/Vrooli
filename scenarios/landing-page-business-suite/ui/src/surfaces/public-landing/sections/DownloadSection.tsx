import { useEffect, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
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

  const title = content?.title || 'Download Vrooli Ascension';
  const subtitle =
    content?.subtitle ||
    'Get the first Silent Founder OS app. Automate your browser work and export studio-quality recordings.';
  const { trackDownload } = useMetrics();
  const [downloadStatus, setDownloadStatus] = useState<Record<string, DownloadStatus>>({});

  const hasAnyInstallers = filteredApps.some((app) => (app.platforms?.length ?? 0) > 0);
  const anyStorefronts = filteredApps.some((app) => (app.storefronts?.length ?? 0) > 0);
  const [activeAppKey, setActiveAppKey] = useState(filteredApps[0]?.app_key ?? '');

  useEffect(() => {
    if (!filteredApps.find((app) => app.app_key === activeAppKey)) {
      setActiveAppKey(filteredApps[0]?.app_key ?? '');
    }
  }, [activeAppKey, filteredApps]);

  const activeApp = filteredApps.find((app) => app.app_key === activeAppKey) ?? filteredApps[0];

  const handleDownload = async (app: DownloadApp, download: DownloadAsset, assetKey: string) => {
    setDownloadStatus((prev) => ({
      ...prev,
      [assetKey]: { loading: true },
    }));

    try {
      const asset = await requestDownload(download.app_key || app.app_key, download.platform);

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
    <section
      className="relative overflow-hidden border-t border-white/5 bg-[#0B1020] py-24 text-white"
      data-testid="downloads-section"
    >
      <div className="pointer-events-none absolute inset-0 opacity-60">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(124,58,237,0.18),transparent_30%),radial-gradient(circle_at_80%_0%,rgba(14,165,233,0.15),transparent_28%),radial-gradient(circle_at_50%_80%,rgba(249,115,22,0.12),transparent_30%)]" />
      </div>
      <div className="container relative mx-auto px-6">
        <div className="mb-12 max-w-4xl space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-2">
              <h2 className="text-4xl font-semibold leading-tight">{title}</h2>
              <p className="text-lg text-slate-200">{subtitle}</p>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <div className="flex flex-wrap gap-2 rounded-2xl border border-white/10 bg-white/5 p-3">
            {filteredApps.map((app) => (
              <button
                key={app.app_key}
                type="button"
                onClick={() => setActiveAppKey(app.app_key)}
                className={`rounded-xl px-4 py-2 text-sm font-semibold transition focus:outline-none focus:ring-2 focus:ring-[#F97316] ${
                  activeApp?.app_key === app.app_key
                    ? 'bg-[#F97316] text-white shadow-lg'
                    : 'bg-white/5 text-slate-200 hover:bg-white/10'
                }`}
              >
                {app.name}
              </button>
            ))}
          </div>

          {activeApp && (
            <div
              className="relative overflow-hidden rounded-3xl border border-white/10 bg-gradient-to-br from-[#0C1228] via-[#0A0F20] to-[#0B132E] p-8 shadow-[0_40px_120px_-40px_rgba(0,0,0,0.6)]"
              data-testid={`download-app-${activeApp.app_key}`}
            >
              <div className="absolute inset-x-0 top-0 h-1 bg-gradient-to-r from-[#F97316] via-[#7C3AED] to-[#0EA5E9]" />
              <div className="relative grid gap-8 lg:grid-cols-[1.15fr_1fr]">
                <div className="space-y-5">
                  <div className="flex flex-wrap items-center gap-3">
                    <span className="rounded-full border border-white/10 bg-white/10 px-3 py-1 text-[11px] uppercase tracking-[0.3em] text-slate-300">
                      Bundle app
                    </span>
                    <span className="rounded-full bg-white/5 px-3 py-1 text-xs font-semibold text-white">
                      {(activeApp.platforms?.length ?? 0)} installer{(activeApp.platforms?.length ?? 0) === 1 ? '' : 's'}
                    </span>
                    {(activeApp.storefronts?.length ?? 0) > 0 && (
                      <span className="rounded-full bg-emerald-400/15 px-3 py-1 text-xs font-semibold text-emerald-200">
                        {(activeApp.storefronts?.length ?? 0)} storefront link
                        {(activeApp.storefronts?.length ?? 0) === 1 ? '' : 's'}
                      </span>
                    )}
                  </div>
                  <div className="space-y-2">
                    <h3 className="text-3xl font-semibold text-white">{activeApp.name}</h3>
                    <p className="text-sm text-slate-300">
                      {activeApp.tagline || 'Offline-safe companion; the app verifies auth and telemetry after install.'}
                    </p>
                  </div>
                  {activeApp.description && <p className="text-sm text-slate-200">{activeApp.description}</p>}
                  {activeApp.install_overview && (
                    <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-100">
                      {activeApp.install_overview}
                    </div>
                  )}
                  {activeApp.install_steps && activeApp.install_steps.length > 0 && (
                    <ol
                      className="list-decimal list-inside space-y-1 text-sm text-slate-300"
                      data-testid={`install-steps-${activeApp.app_key}`}
                    >
                      {activeApp.install_steps.map((step, idx) => (
                        <li key={`${activeApp.app_key}-step-${idx}`}>{step}</li>
                      ))}
                    </ol>
                  )}
                  {(activeApp.storefronts?.length ?? 0) > 0 && (
                    <div className="flex flex-wrap gap-3 pt-2">
                      {activeApp.storefronts?.map((store) => (
                        <a
                          key={`${activeApp.app_key}-${store.store}-${store.url}`}
                          href={store.url}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white transition hover:border-white/20 hover:bg-white/10"
                          data-testid={`storefront-${activeApp.app_key}-${store.store}`}
                        >
                          {store.badge || store.label}
                        </a>
                      ))}
                    </div>
                  )}
                </div>

                {(activeApp.platforms?.length ?? 0) > 0 && (
                  <div className="space-y-4 rounded-2xl border border-white/10 bg-white/5 p-5" data-testid={`installer-list-${activeApp.app_key}`}>
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                      <p className="text-[11px] uppercase tracking-[0.3em] text-slate-300">Desktop installers</p>
                      <span className="rounded-full bg-black/20 px-3 py-1 text-xs text-slate-200">
                        Downloads are direct; subscription is handled in-app.
                      </span>
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                      {activeApp.platforms?.map((installer) => {
                        const assetKey = getDownloadAssetKey(installer);
                        const status = downloadStatus[assetKey];
                        return (
                          <div
                            key={assetKey}
                            className="space-y-3 rounded-2xl border border-white/10 bg-[#05070E] p-4 shadow-[0_20px_40px_-24px_rgba(0,0,0,0.7)]"
                            data-testid={`download-card-${assetKey}`}
                          >
                            <div className="flex items-center justify-between gap-2">
                              <div className="flex items-center gap-2">
                                <span className="rounded-full bg-white/10 px-3 py-1 text-[11px] uppercase tracking-[0.3em] text-slate-200">
                                  {formatPlatformLabel(installer.platform)}
                                </span>
                              </div>
                              <span className="text-xs text-slate-400">{installer.release_version}</span>
                            </div>
                            <p className="text-sm text-slate-300">
                              {installer.release_notes || 'Release notes coming soon.'}
                            </p>
                            <div className="flex flex-wrap gap-2 text-[11px] uppercase tracking-[0.2em] text-slate-400">
                              <span>{installer.requires_entitlement ? 'Sign-in in app' : 'Free download'}</span>
                              {installer.metadata?.size_mb && <span>{installer.metadata.size_mb} MB</span>}
                            </div>
                            <Button
                              variant="default"
                              size="sm"
                              onClick={() => handleDownload(activeApp, installer, assetKey)}
                              disabled={status?.loading}
                              className={`w-full ${installer.requires_entitlement ? '' : 'bg-[#F97316]'}`}
                            >
                              {status?.loading ? 'Preparing…' : 'Download'}
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
          )}
        </div>

        {(hasAnyInstallers || anyStorefronts) && (
          <div className="mx-auto mt-12 max-w-4xl rounded-3xl border border-white/10 bg-white/5 p-6 text-center text-sm text-slate-200">
            Downloads are direct and the apps will prompt for sign-in or subscription once launched. Need additional
            seats or bundle access? Contact support before sharing installers so telemetry stays accurate.
          </div>
        )}
      </div>
    </section>
  );
}
