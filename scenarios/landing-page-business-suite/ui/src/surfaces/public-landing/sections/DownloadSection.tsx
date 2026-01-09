import { useCallback, useMemo, useState } from 'react';
import { Button } from '../../../shared/ui/button';
import { useMetrics } from '../../../shared/hooks/useMetrics';
import type { DownloadApp, DownloadAsset } from '../../../shared/api';
import { createBillingPortalSession, requestDownload, getAssetUrl } from '../../../shared/api'
import { useEntitlements } from '../../../shared/hooks/useEntitlements';

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
  success?: boolean;
  message?: string;
};

type DetectedPlatform = 'windows' | 'mac' | 'linux' | 'unknown';

interface PlatformGroup {
  platform: DetectedPlatform | string;
  installers: DownloadAsset[];
}

const PLATFORM_DISPLAY: Record<string, { label: string; icon: JSX.Element }> = {
  windows: {
    label: 'Windows',
    icon: (
      <svg className="h-8 w-8" viewBox="0 0 24 24" fill="currentColor">
        <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
      </svg>
    ),
  },
  mac: {
    label: 'macOS',
    icon: (
      <svg className="h-8 w-8" viewBox="0 0 24 24" fill="currentColor">
        <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
      </svg>
    ),
  },
  linux: {
    label: 'Linux',
    icon: (
      <svg className="h-8 w-8" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139zm.529 3.405h.013c.213 0 .396.062.584.198.19.135.33.332.438.533.105.259.158.459.166.724 0-.02.006-.04.006-.06v.105a.086.086 0 01-.004-.021l-.004-.024a1.807 1.807 0 01-.15.706.953.953 0 01-.213.335.71.71 0 00-.088-.042c-.104-.045-.198-.064-.284-.133a1.312 1.312 0 00-.22-.066c.05-.06.146-.133.183-.198.053-.128.082-.264.088-.402v-.02a1.21 1.21 0 00-.061-.4c-.045-.134-.101-.2-.183-.333-.084-.066-.167-.132-.267-.132h-.016c-.093 0-.176.03-.262.132a.8.8 0 00-.205.334 1.18 1.18 0 00-.09.4v.019c.002.089.008.179.02.267-.193-.067-.438-.135-.607-.202a1.635 1.635 0 01-.018-.2v-.02a1.772 1.772 0 01.15-.768c.082-.22.232-.406.43-.533a.985.985 0 01.594-.2zm-2.962.059h.036c.142 0 .27.048.399.135.146.129.264.288.344.465.09.199.14.4.153.667v.004c.007.134.006.2-.002.266v.08c-.03.007-.056.018-.083.024-.152.055-.274.135-.393.2.012-.09.013-.18.003-.267v-.015c-.012-.133-.04-.2-.082-.333a.613.613 0 00-.166-.267.248.248 0 00-.183-.064h-.021c-.071.006-.13.04-.186.132a.552.552 0 00-.12.27.944.944 0 00-.023.33v.015c.012.135.037.2.08.334.046.134.098.2.166.268.01.009.02.018.034.024-.07.057-.117.07-.176.136a.304.304 0 01-.131.068 2.62 2.62 0 01-.275-.402 1.772 1.772 0 01-.155-.667 1.759 1.759 0 01.08-.668 1.43 1.43 0 01.283-.535c.128-.133.26-.2.418-.2zm1.37 1.706c.332 0 .733.065 1.216.399.293.2.523.269 1.052.468h.003c.255.136.405.266.478.399v-.131a.571.571 0 01.016.47c-.123.31-.516.643-1.063.842v.002c-.268.135-.501.333-.775.465-.276.135-.588.292-1.012.267a1.139 1.139 0 01-.448-.067 3.566 3.566 0 01-.322-.198c-.195-.135-.363-.332-.612-.465v-.005h-.005c-.4-.246-.616-.512-.686-.71-.07-.268-.005-.47.193-.6.224-.135.38-.271.483-.336.104-.074.143-.102.176-.131h.002v-.003c.169-.202.436-.47.839-.601.139-.036.294-.065.466-.065zm2.8 2.142c.358 1.417 1.196 3.475 1.735 4.473.286.534.855 1.659 1.102 3.024.156-.005.33.018.513.064.646-1.671-.546-3.467-1.089-3.966-.22-.2-.232-.335-.123-.335.59.534 1.365 1.572 1.646 2.757.13.535.16 1.104.021 1.67.067.028.135.06.205.067 1.032.534 1.413.938 1.23 1.537v-.002c-.06-.135-.12-.2-.283-.334-.152-.135-.34-.2-.545-.266a.652.652 0 00.03-.2v-.002a.704.704 0 00-.03-.133 3.567 3.567 0 00-.27-.936 3.253 3.253 0 00-.143-.334c.013.003.024.006.034.003.104-.026.193-.064.24-.202.043-.134.016-.334-.105-.467-.118-.135-.074-.2.058-.267.095-.054.225-.075.37-.107.126-.026.257-.053.346-.146.085-.066.102-.2.044-.333-.058-.135-.135-.133-.244-.2-.21-.135-.48-.202-.725-.2a1.862 1.862 0 00-.373.032c.009-.018.012-.035.016-.055a.658.658 0 00-.014-.268c-.034-.135-.113-.267-.218-.333a.476.476 0 00-.31-.135c-.104 0-.18.033-.285.135-.104.134-.143.267-.15.4-.004.134.04.27.08.335-.069.07-.148.127-.232.182-.166.112-.357.212-.545.378-.09-.2-.146-.334-.236-.667-.09-.332-.135-.666-.04-1.067v-.005c.06-.333.163-.667.32-.999.16-.335.38-.666.7-.934.312-.266.654-.466 1.046-.533h.003c.169-.032.378-.038.592-.003.213.038.442.107.655.232.215.135.4.332.541.598zm-7.67.403c.009.06.015.122.015.184v.02c-.004.09-.01.131-.022.2a2.15 2.15 0 01-.104.4c.05.055.118.135.18.2.14.135.28.27.48.333.2.067.418.067.6 0 .202-.066.374-.2.496-.333l.054-.067c-.073.066-.173.135-.323.2-.147.066-.317.131-.49.131-.075 0-.15-.01-.223-.032a.697.697 0 01-.186-.097c-.068-.042-.138-.106-.177-.164a.566.566 0 01-.086-.201.405.405 0 01-.016-.199c.01-.07.03-.131.055-.2.042-.134.123-.266.193-.399h-.002c.068-.133.17-.266.22-.332.048-.067.078-.134.078-.2a.37.37 0 00-.044-.167v-.002c-.02-.037-.046-.068-.078-.098a.303.303 0 00-.112-.066.217.217 0 00-.115-.005c-.095.012-.215.066-.32.2a1.65 1.65 0 00-.257.534c-.06.2-.085.4-.077.598v.003a.833.833 0 00.026.2zm2.66.03c.05-.06.09-.135.115-.2.025-.066.044-.132.054-.199.008-.067.009-.133.001-.2a.548.548 0 00-.045-.198.51.51 0 00-.098-.15.361.361 0 00-.138-.106.27.27 0 00-.16-.013c-.06.013-.118.043-.17.086-.052.046-.095.1-.13.164a.7.7 0 00-.08.201.618.618 0 00-.018.2c.005.067.024.133.054.2.03.066.068.13.115.2-.017.067-.05.135-.11.2-.06.066-.14.132-.246.199a.693.693 0 01-.323.098c-.092.003-.194-.01-.287-.032a1.47 1.47 0 01-.256-.097c-.086-.038-.17-.085-.245-.138a1.155 1.155 0 01-.186-.167c-.055-.058-.1-.12-.136-.188a.72.72 0 01-.077-.206.63.63 0 01-.013-.21c.013-.074.035-.14.068-.206.041-.088.098-.171.162-.248.065-.079.137-.152.213-.22a2.96 2.96 0 01.468-.334c.17-.103.348-.193.531-.265a3.428 3.428 0 011.2-.26c.145-.006.29.007.43.032.142.025.279.063.408.117.13.052.25.12.36.2.111.08.21.175.296.282.085.104.157.222.212.348a1.4 1.4 0 01.11.397c.016.135.016.27 0 .398a1.312 1.312 0 01-.11.393 1.414 1.414 0 01-.508.563c-.11.08-.23.148-.358.2-.127.05-.262.087-.4.1a1.58 1.58 0 01-.407-.012 1.574 1.574 0 01-.386-.116 1.78 1.78 0 01-.33-.199z" />
      </svg>
    ),
  },
};

function detectPlatform(): DetectedPlatform {
  if (typeof navigator === 'undefined') return 'unknown';

  const ua = navigator.userAgent.toLowerCase();
  const platform = (navigator.platform || '').toLowerCase();

  if (platform.includes('win') || ua.includes('windows')) return 'windows';
  if (platform.includes('mac') || ua.includes('macintosh') || ua.includes('mac os')) return 'mac';
  if (platform.includes('linux') || ua.includes('linux')) return 'linux';

  return 'unknown';
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
  if (typeof artifactUrl !== 'string') return '';
  const trimmed = artifactUrl.trim();
  if (!trimmed) return '';
  const lower = trimmed.toLowerCase();
  if (lower.startsWith('javascript:') || lower.startsWith('data:') || lower.startsWith('vbscript:')) return '';
  if (/^https?:\/\//i.test(trimmed)) return trimmed;
  if (/^[./]/.test(trimmed) || !trimmed.includes(':')) return trimmed;
  return '';
}

function openDownloadWindow(url: string) {
  if (typeof window === 'undefined' || typeof window.open !== 'function') return false;
  const target = window.open(url, '_blank', 'noopener,noreferrer');
  return target !== null;
}

function hasInstallTargets(app: DownloadApp) {
  return (app.platforms?.length ?? 0) > 0 || (app.storefronts?.length ?? 0) > 0;
}

/** Extract a human-readable variant label from artifact URL or release notes */
function getVariantLabel(installer: DownloadAsset): string {
  const url = installer.artifact_url?.toLowerCase() ?? '';
  const notes = installer.release_notes?.toLowerCase() ?? '';

  // Check for specific formats
  if (url.includes('.appimage') || notes.includes('appimage')) return 'AppImage';
  if (url.includes('.deb') || notes.includes('.deb')) return '.deb';
  if (url.includes('.rpm') || notes.includes('.rpm')) return '.rpm';
  if (url.includes('.tar.gz') || url.includes('.tgz')) return '.tar.gz';
  if (url.includes('.dmg') || notes.includes('.dmg')) return '.dmg';
  if (url.includes('.pkg') || notes.includes('.pkg')) return '.pkg';
  if (url.includes('.exe') || notes.includes('.exe')) return 'Installer';
  if (url.includes('.msi') || notes.includes('.msi')) return '.msi';
  if (url.includes('.zip')) return '.zip';

  // Check for architecture
  if (url.includes('arm64') || url.includes('aarch64') || notes.includes('arm')) return 'ARM64';
  if (url.includes('x64') || url.includes('x86_64') || url.includes('amd64')) return '64-bit';
  if (url.includes('x86') || url.includes('i386') || url.includes('i686')) return '32-bit';

  // Fallback to version
  return installer.release_version ? `v${installer.release_version}` : 'Download';
}

export function DownloadSection({ content, downloads }: DownloadSectionProps) {
  const filteredApps = (downloads ?? []).filter(hasInstallTargets);
  if (filteredApps.length === 0) return null;

  const title = content?.title || 'Download Vrooli Ascension';
  const subtitle = content?.subtitle || 'Install now and start automating today.';

  const { trackDownload } = useMetrics();
  const [downloadStatus, setDownloadStatus] = useState<Record<string, DownloadStatus>>({});
  const { email, setEmail, entitlements, loading: entitlementsLoading, error: entitlementsError, refresh } = useEntitlements();
  const [portalMessage, setPortalMessage] = useState<string | null>(null);
  const [portalLoading, setPortalLoading] = useState(false);
  const [showOtherPlatforms, setShowOtherPlatforms] = useState(false);
  const [showSubscriptionInput, setShowSubscriptionInput] = useState(false);

  const detectedPlatform = useMemo(() => detectPlatform(), []);
  const activeApp = filteredApps[0];

  // Group installers by platform
  const platformGroups = useMemo<PlatformGroup[]>(() => {
    const platforms = activeApp?.platforms ?? [];
    const groups: Record<string, DownloadAsset[]> = {};

    for (const installer of platforms) {
      const key = installer.platform.toLowerCase();
      if (!groups[key]) groups[key] = [];
      groups[key].push(installer);
    }

    // Sort: detected platform first, then alphabetically
    const sortedKeys = Object.keys(groups).sort((a, b) => {
      if (a === detectedPlatform) return -1;
      if (b === detectedPlatform) return 1;
      return a.localeCompare(b);
    });

    return sortedKeys.map((key) => ({
      platform: key,
      installers: groups[key],
    }));
  }, [activeApp?.platforms, detectedPlatform]);

  const recommendedGroup = platformGroups.find((g) => g.platform === detectedPlatform) ?? platformGroups[0];
  const otherGroups = platformGroups.filter((g) => g !== recommendedGroup);

  const handleDownload = useCallback(
    async (download: DownloadAsset, assetKey: string) => {
      setDownloadStatus((prev) => ({ ...prev, [assetKey]: { loading: true } }));

      try {
        const asset = await requestDownload(download.app_key || activeApp.app_key, download.platform, email.trim() || undefined);
        const artifactUrl = sanitizeArtifactUrl(asset?.artifact_url);

        if (!artifactUrl) {
          setDownloadStatus((prev) => ({
            ...prev,
            [assetKey]: { loading: false, message: 'Download not available yet. Check back soon.' },
          }));
          return;
        }

        const opened = openDownloadWindow(artifactUrl);
        if (!opened) {
          setDownloadStatus((prev) => ({
            ...prev,
            [assetKey]: { loading: false, message: 'Pop-up blocked. Allow pop-ups and try again.' },
          }));
          return;
        }

        trackDownload({
          app_key: activeApp.app_key,
          platform: download.platform,
          release_version: download.release_version,
          requires_entitlement: download.requires_entitlement,
        });

        setDownloadStatus((prev) => ({
          ...prev,
          [assetKey]: { loading: false, success: true, message: 'Download started!' },
        }));
      } catch (err) {
        setDownloadStatus((prev) => ({
          ...prev,
          [assetKey]: { loading: false, message: err instanceof Error ? err.message : 'Download failed.' },
        }));
      }
    },
    [activeApp, email, trackDownload]
  );

  const handlePortal = async () => {
    if (!email.trim()) {
      setPortalMessage('Enter your subscription email first.');
      return;
    }
    setPortalLoading(true);
    setPortalMessage(null);
    try {
      const resp = await createBillingPortalSession(undefined, email.trim());
      if (!openDownloadWindow(resp.url)) {
        setPortalMessage('Pop-up blocked. Allow pop-ups to open the billing portal.');
      }
    } catch (err) {
      setPortalMessage(err instanceof Error ? err.message : 'Unable to open billing portal.');
    } finally {
      setPortalLoading(false);
    }
  };

  const entitlementsStatus = useMemo(() => {
    if (entitlementsLoading) return { text: 'Checking...', type: 'loading' as const };
    if (entitlementsError) return { text: entitlementsError, type: 'error' as const };
    if (!email.trim()) return { text: 'Enter email to verify', type: 'neutral' as const };
    if (!entitlements) return { text: 'Not verified', type: 'neutral' as const };
    if (entitlements.status === 'active') return { text: 'Active subscription', type: 'success' as const };
    if (entitlements.status === 'trialing') return { text: 'Trial active', type: 'success' as const };
    return { text: 'No active subscription', type: 'warning' as const };
  }, [email, entitlements, entitlementsError, entitlementsLoading]);

  const platformInfo = PLATFORM_DISPLAY[recommendedGroup?.platform] ?? PLATFORM_DISPLAY.windows;
  const isRecommended = recommendedGroup?.platform === detectedPlatform;

  const renderDownloadButton = (installer: DownloadAsset, isPrimary: boolean = false) => {
    const assetKey = getDownloadAssetKey(installer);
    const status = downloadStatus[assetKey];
    const variantLabel = getVariantLabel(installer);

    return (
      <div key={assetKey} className="flex flex-col items-center gap-1">
        <Button
          size={isPrimary ? 'default' : 'sm'}
          variant={isPrimary ? 'default' : 'outline'}
          onClick={() => handleDownload(installer, assetKey)}
          disabled={status?.loading}
          className={isPrimary ? 'min-w-[140px] gap-2' : 'min-w-[100px]'}
          data-testid={isPrimary ? 'download-btn-primary' : `download-btn-${assetKey}`}
        >
          {status?.loading ? (
            <>
              <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              {isPrimary && 'Preparing...'}
            </>
          ) : status?.success ? (
            <>
              <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M20 6L9 17l-5-5" />
              </svg>
              {isPrimary && 'Done'}
            </>
          ) : (
            <>
              {isPrimary && (
                <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4M7 10l5 5 5-5M12 15V3" />
                </svg>
              )}
              {variantLabel}
            </>
          )}
        </Button>
        {status?.message && !status.success && (
          <p className="text-center text-xs text-amber-300">{status.message}</p>
        )}
      </div>
    );
  };

  return (
    <section
      className="relative overflow-hidden border-t border-white/5 bg-[#0B1020] py-20 text-white"
      data-testid="downloads-section"
    >
      {/* Subtle gradient background */}
      <div className="pointer-events-none absolute inset-0 opacity-40">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_20%,rgba(249,115,22,0.15),transparent_40%)]" />
      </div>

      <div className="container relative mx-auto max-w-4xl px-6">
        {/* Header */}
        <div className="mb-10 text-center">
          <h2 className="text-3xl font-semibold md:text-4xl">{title}</h2>
          <p className="mt-3 text-lg text-slate-300">{subtitle}</p>
        </div>

        {/* Primary Download Card */}
        {recommendedGroup && (
          <div
            className="relative overflow-hidden rounded-2xl border border-white/10 bg-gradient-to-b from-white/[0.08] to-white/[0.02] p-6 md:p-8"
            data-testid="download-card-primary"
          >
            <div className="flex flex-col gap-6 md:flex-row md:items-center">
              {/* App Icon or Platform Icon */}
              <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-2xl bg-white/10 text-white md:h-20 md:w-20 overflow-hidden">
                {activeApp?.icon_url ? (
                  <img
                    src={getAssetUrl(activeApp.icon_url)}
                    alt={`${activeApp.name} icon`}
                    className="h-full w-full object-contain"
                    onError={(e) => {
                      // Fallback to platform icon on error
                      e.currentTarget.style.display = 'none';
                      e.currentTarget.nextElementSibling?.classList.remove('hidden');
                    }}
                  />
                ) : null}
                <div className={activeApp?.icon_url ? 'hidden' : ''}>
                  {platformInfo.icon}
                </div>
              </div>

              {/* Details */}
              <div className="flex-1">
                <div className="flex flex-wrap items-center gap-3">
                  <h3 className="text-2xl font-semibold">{activeApp?.name || platformInfo.label}</h3>
                  {isRecommended && (
                    <span className="rounded-full bg-emerald-500/20 px-3 py-1 text-xs font-medium text-emerald-300">
                      Recommended for {platformInfo.label}
                    </span>
                  )}
                </div>
                {activeApp?.tagline && (
                  <p className="mt-1 text-sm text-slate-300">{activeApp.tagline}</p>
                )}
                <p className="mt-2 text-sm text-slate-400">
                  Start for free
                </p>
              </div>

              {/* Download Buttons */}
              <div className="flex flex-wrap items-center justify-center gap-3 md:justify-end">
                {recommendedGroup.installers.map((installer, idx) =>
                  renderDownloadButton(installer, idx === 0)
                )}
              </div>
            </div>

            {/* Screenshot Preview (if available) */}
            {activeApp?.screenshot_url && (
              <div className="mt-6 overflow-hidden rounded-xl border border-white/10">
                <img
                  src={getAssetUrl(activeApp.screenshot_url)}
                  alt={`${activeApp.name} screenshot`}
                  className="w-full object-cover"
                  style={{ maxHeight: '300px' }}
                />
              </div>
            )}
          </div>
        )}

        {/* Other Platforms Toggle */}
        {otherGroups.length > 0 && (
          <div className="mt-6">
            <button
              type="button"
              onClick={() => setShowOtherPlatforms(!showOtherPlatforms)}
              className="mx-auto flex items-center gap-2 rounded-full px-4 py-2 text-sm text-slate-400 transition hover:bg-white/5 hover:text-slate-200"
              data-testid="toggle-other-platforms"
            >
              <span>Other platforms</span>
              <svg
                className={`h-4 w-4 transition-transform ${showOtherPlatforms ? 'rotate-180' : ''}`}
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
              >
                <path d="M6 9l6 6 6-6" />
              </svg>
            </button>

            {showOtherPlatforms && (
              <div className="mt-4 space-y-3" data-testid="other-platforms-list">
                {otherGroups.map((group) => {
                  const info = PLATFORM_DISPLAY[group.platform] ?? PLATFORM_DISPLAY.windows;
                  return (
                    <div
                      key={group.platform}
                      className="flex flex-col items-center gap-4 rounded-xl border border-white/10 bg-white/[0.03] p-4 sm:flex-row"
                      data-testid={`download-card-${group.platform}`}
                    >
                      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-white/10 text-white">
                        {info.icon}
                      </div>
                      <div className="flex-1 text-center sm:text-left">
                        <span className="font-medium">{info.label}</span>
                      </div>
                      <div className="flex flex-wrap justify-center gap-2">
                        {group.installers.map((installer) => renderDownloadButton(installer, false))}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* Subscription Input (Collapsible) */}
        <div className="mt-8">
          <button
            type="button"
            onClick={() => setShowSubscriptionInput(!showSubscriptionInput)}
            className="mx-auto flex items-center gap-2 text-sm text-slate-400 transition hover:text-slate-200"
            data-testid="toggle-subscription-input"
          >
            <svg className="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 16v-4M12 8h.01" />
            </svg>
            <span>Already have a subscription?</span>
          </button>

          {showSubscriptionInput && (
            <div
              className="mx-auto mt-4 max-w-md rounded-xl border border-white/10 bg-white/[0.03] p-4"
              data-testid="subscription-input-panel"
            >
              <label className="block text-sm text-slate-300">
                Subscription email
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  onBlur={() => refresh()}
                  placeholder="you@example.com"
                  className="mt-2 w-full rounded-lg border border-white/10 bg-[#0B1020] px-3 py-2 text-white placeholder:text-slate-500 focus:border-[#F97316] focus:outline-none"
                />
              </label>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <span
                  className={`rounded-full px-3 py-1 text-xs ${
                    entitlementsStatus.type === 'success'
                      ? 'bg-emerald-500/20 text-emerald-300'
                      : entitlementsStatus.type === 'error'
                        ? 'bg-red-500/20 text-red-300'
                        : entitlementsStatus.type === 'warning'
                          ? 'bg-amber-500/20 text-amber-300'
                          : 'bg-white/10 text-slate-300'
                  }`}
                >
                  {entitlementsStatus.text}
                </span>
                <Button size="sm" variant="ghost" onClick={refresh} disabled={entitlementsLoading}>
                  Refresh
                </Button>
                <Button size="sm" variant="ghost" onClick={handlePortal} disabled={portalLoading}>
                  Billing portal
                </Button>
              </div>
              {portalMessage && <p className="mt-2 text-xs text-amber-300">{portalMessage}</p>}
            </div>
          )}
        </div>

        {/* Post-download instructions (shown after successful download) */}
        {Object.values(downloadStatus).some((s) => s.success) && (
          <div
            className="mx-auto mt-8 max-w-lg rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-4 text-center"
            data-testid="post-download-instructions"
          >
            <h4 className="font-medium text-emerald-300">Download started</h4>
            <ol className="mt-2 space-y-1 text-sm text-slate-300">
              <li>1. Run the installer once the download completes</li>
              <li>2. Follow the setup wizard</li>
              <li>3. Sign in with your subscription email to unlock features</li>
            </ol>
          </div>
        )}

        {/* Footer note */}
        <p className="mt-8 text-center text-xs text-slate-500">
          Downloads are direct. The app verifies your subscription after install.
          <br />
          Need help? <a href="mailto:support@vrooli.com" className="text-slate-400 underline hover:text-white">Contact support</a>
        </p>
      </div>
    </section>
  );
}
