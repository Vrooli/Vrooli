import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { SlidersHorizontal, Palette, Globe, Loader2 } from 'lucide-react';
import { loadHistory, type HistoryEntry } from '../capture/BrowserUrlBar';
import { useLinkPreviewsBatch, type LinkPreviewData } from '../hooks/useLinkPreview';
import type { RecordedAction } from '../types/types';
import { PlaywrightView, type FrameStats, type PageMetadata, type StreamConnectionStatus } from '../capture/PlaywrightView';
import { StreamSettings, useStreamSettings, type StreamSettingsValues } from '../capture/StreamSettings';
import { FrameStatsDisplay } from '../capture/FrameStatsDisplay';
import { BrowserUrlBar } from '../capture/BrowserUrlBar';
import { usePerfStats } from '../hooks/usePerfStats';
import { useSettingsStore } from '@stores/settingsStore';
import {
  useReplaySettingsSync,
} from '@/domains/replay-style';
import type { ReplayRect } from '@/domains/replay-layout';
import { WatermarkOverlay } from '@/domains/exports/replay/WatermarkOverlay';
import ReplayPresentation from '@/domains/exports/replay/ReplayPresentation';
import { useReplayPresentationModel } from '@/domains/exports/replay/useReplayPresentationModel';
import { ReplaySection } from '@/views/SettingsView/sections/ReplaySection';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import clsx from 'clsx';

interface RecordPreviewPanelProps {
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  /** Active page ID for multi-tab sessions */
  activePageId?: string | null;
  onViewportChange?: (size: { width: number; height: number }) => void;
  /** Callback when stream settings change (for session creation) */
  onStreamSettingsChange?: (settings: StreamSettingsValues) => void;
  /** Callback when stream connection status changes */
  onConnectionStatusChange?: (status: StreamConnectionStatus) => void;
  /** Whether to hide the in-preview connection indicator (shown in header instead) */
  hideConnectionIndicator?: boolean;
}

export function RecordPreviewPanel({
  previewUrl,
  onPreviewUrlChange,
  actions,
  sessionId,
  activePageId,
  onViewportChange,
  onStreamSettingsChange,
  onConnectionStatusChange,
  hideConnectionIndicator = false,
}: RecordPreviewPanelProps) {
  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  // Check if URL is navigable (not empty, about:blank, or chrome-error://)
  const isNavigableUrl = (url: string) => {
    if (!url) return false;
    const lowerUrl = url.toLowerCase();
    return !lowerUrl.startsWith('about:') && !lowerUrl.startsWith('chrome-error://');
  };

  const rawEffectiveUrl = previewUrl || lastUrl;
  const effectiveUrl = isNavigableUrl(rawEffectiveUrl) ? rawEffectiveUrl : '';

  const [liveRefreshToken, setLiveRefreshToken] = useState(0);
  const previewBoundsRef = useRef<HTMLDivElement | null>(null);
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const playwrightContainerRef = useRef<HTMLDivElement | null>(null);
  const [previewBounds, setPreviewBounds] = useState<{ width: number; height: number } | null>(null);
  const lastViewportRef = useRef<{ width: number; height: number } | null>(null);
  // Track viewport for passing to PlaywrightView (for coordinate mapping)
  const [currentViewport, setCurrentViewport] = useState<{ width: number; height: number } | null>(null);
  const [showReplayStyle, setShowReplayStyle] = useState(false);
  const [showReplaySettings, setShowReplaySettings] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [newPresetName, setNewPresetName] = useState('');
  const [previewContentRect, setPreviewContentRect] = useState<ReplayRect | null>(null);

  // Stream settings management
  const { preset, settings: streamSettings, showStats, setPreset, setShowStats } = useStreamSettings();

  // Debug performance stats from server (enabled when showStats is on)
  const { stats: perfStats } = usePerfStats(sessionId ?? null, showStats);
  const {
    replay,
    setReplaySetting,
    randomizeSettings,
    saveAsPreset,
  } = useSettingsStore();

  const styleOverrides = useMemo(
    () => ({
      presentation: replay.presentation,
      chromeTheme: replay.chromeTheme,
      background: replay.background,
      cursorTheme: replay.cursorTheme,
      cursorInitialPosition: replay.cursorInitialPosition,
      cursorClickAnimation: replay.cursorClickAnimation,
      cursorScale: replay.cursorScale,
      browserScale: replay.browserScale,
    }),
    [
      replay.background,
      replay.browserScale,
      replay.chromeTheme,
      replay.cursorClickAnimation,
      replay.cursorInitialPosition,
      replay.cursorScale,
      replay.cursorTheme,
      replay.presentation,
    ],
  );

  const extraConfig = useMemo(
    () => ({
      cursorSpeedProfile: replay.cursorSpeedProfile,
      cursorPathStyle: replay.cursorPathStyle,
      renderSource: replay.exportRenderSource,
      watermark: replay.watermark,
      introCard: replay.introCard,
      outroCard: replay.outroCard,
    }),
    [
      replay.cursorPathStyle,
      replay.cursorSpeedProfile,
      replay.exportRenderSource,
      replay.introCard,
      replay.outroCard,
      replay.watermark,
    ],
  );

  useReplaySettingsSync({
    styleOverrides,
    extraConfig,
    onStyleHydrated: (style) => {
      setReplaySetting('presentation', style.presentation);
      setReplaySetting('chromeTheme', style.chromeTheme);
      setReplaySetting('background', style.background);
      setReplaySetting('cursorTheme', style.cursorTheme);
      setReplaySetting('cursorInitialPosition', style.cursorInitialPosition);
      setReplaySetting('cursorClickAnimation', style.cursorClickAnimation);
      setReplaySetting('cursorScale', style.cursorScale);
      setReplaySetting('browserScale', style.browserScale);
    },
  });

  // Frame statistics from PlaywrightView
  const [frameStats, setFrameStats] = useState<FrameStats | null>(null);
  const handleStatsUpdate = useCallback((stats: FrameStats) => {
    setFrameStats(stats);
  }, []);

  // Page metadata (title, url) from PlaywrightView - used for history display
  const [pageTitle, setPageTitle] = useState<string>('');
  const handlePageMetadataChange = useCallback((metadata: PageMetadata) => {
    setPageTitle(metadata.title);
  }, []);

  // Notify parent when stream settings change (for session creation)
  useEffect(() => {
    onStreamSettingsChange?.(streamSettings);
  }, [streamSettings, onStreamSettingsChange]);

  // Handle URL navigation from BrowserUrlBar
  const handleUrlNavigate = useCallback(
    (url: string) => {
      onPreviewUrlChange(url);
    },
    [onPreviewUrlChange]
  );

  // Handle refresh
  const handleRefresh = useCallback(() => {
    setLiveRefreshToken((t) => t + 1);
  }, []);

  // Observe preview container size and notify parent for viewport sizing.
  useEffect(() => {
    const node = previewBoundsRef.current;
    if (!node) return;

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      const width = entry.contentRect.width;
      const height = entry.contentRect.height;
      if (width <= 0 || height <= 0) return;
      setPreviewBounds({ width, height });
    });

    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  const activeViewport = useMemo(() => {
    if (!showReplayStyle && previewBounds) {
      const width = Math.min(3840, Math.max(320, Math.round(previewBounds.width)));
      const height = Math.min(3840, Math.max(320, Math.round(previewBounds.height)));
      return { width, height };
    }
    const width = Math.min(3840, Math.max(320, Math.round(replay.presentationWidth)));
    const height = Math.min(3840, Math.max(320, Math.round(replay.presentationHeight)));
    return { width, height };
  }, [previewBounds, replay.presentationHeight, replay.presentationWidth, showReplayStyle]);

  useEffect(() => {
    const target = activeViewport;
    const last = lastViewportRef.current;
    if (!last || last.width !== target.width || last.height !== target.height) {
      lastViewportRef.current = target;
      setCurrentViewport(target);
      onViewportChange?.(target);
    }
  }, [activeViewport, onViewportChange]);

  const targetWidth = activeViewport.width;
  const targetHeight = activeViewport.height;
  const previewTitle = pageTitle || effectiveUrl || 'Live Preview';
  const presentationModel = useReplayPresentationModel({
    style: styleOverrides,
    title: previewTitle,
    canvasDimensions: { width: targetWidth, height: targetHeight },
    viewportDimensions: currentViewport ?? { width: targetWidth, height: targetHeight },
    presentationBounds: previewBounds ?? undefined,
    presentationFit: previewBounds ? 'contain' : 'none',
  });

  const previewLayout = showReplayStyle ? presentationModel.layout : null;

  const handleSavePreset = useCallback(() => {
    const trimmedName = newPresetName.trim();
    if (!trimmedName) return;
    saveAsPreset(trimmedName);
    setNewPresetName('');
    setShowSaveDialog(false);
  }, [newPresetName, saveAsPreset]);

  const handleContentRectChange = useCallback(
    (rect: ReplayRect) => {
      const viewportNode = viewportRef.current;
      const containerNode = playwrightContainerRef.current;
      if (!viewportNode || !containerNode) {
        setPreviewContentRect(rect);
        return;
      }
      const viewportBounds = viewportNode.getBoundingClientRect();
      const containerBounds = containerNode.getBoundingClientRect();
      setPreviewContentRect({
        x: rect.x + (containerBounds.left - viewportBounds.left),
        y: rect.y + (containerBounds.top - viewportBounds.top),
        width: rect.width,
        height: rect.height,
      });
    },
    [],
  );

  return (
    <div className="flex flex-col h-full">
      {/* Header with browser-like URL bar and settings */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-2.5 flex items-center gap-2">
        {/* Browser URL bar */}
        <BrowserUrlBar
          value={previewUrl}
          onChange={onPreviewUrlChange}
          onNavigate={handleUrlNavigate}
          onRefresh={handleRefresh}
          placeholder={lastUrl || 'Search or enter URL'}
          pageTitle={pageTitle}
        />

        {/* Frame stats display (conditionally shown) */}
        {showStats && (
          <FrameStatsDisplay
            stats={frameStats}
            targetFps={streamSettings.fps}
            debugStats={perfStats}
          />
        )}

        <button
          type="button"
          role="switch"
          aria-checked={showReplayStyle}
          onClick={() => setShowReplayStyle((prev) => !prev)}
          className={clsx(
            'flex items-center gap-2 px-2 py-1.5 text-xs rounded-lg border transition-colors',
            showReplayStyle
              ? 'border-flow-accent/50 bg-flow-accent/10 text-flow-accent'
              : 'border-gray-700 bg-gray-800/70 text-gray-300 hover:text-surface',
          )}
          title="Toggle replay styling for the live preview"
        >
          <Palette size={14} />
          Replay Style
        </button>

        <button
          type="button"
          onClick={() => setShowReplaySettings(true)}
          className="flex items-center gap-2 px-2 py-1.5 text-xs text-gray-300 hover:text-surface bg-gray-800/70 border border-gray-700 rounded-lg transition-colors"
          title="Configure replay styling"
        >
          <SlidersHorizontal size={14} />
          Settings
        </button>

        {/* Stream quality settings */}
        <StreamSettings
          sessionId={sessionId}
          preset={preset}
          onPresetChange={setPreset}
          onSettingsChange={(newSettings) => onStreamSettingsChange?.(newSettings)}
          showStats={showStats}
          onShowStatsChange={setShowStats}
        />
      </div>
      <div className="flex-1 overflow-hidden" ref={previewBoundsRef}>
        <div
          className={clsx(
            'h-full w-full flex',
            showReplayStyle ? 'bg-slate-950/95' : 'bg-transparent',
            showReplayStyle ? 'items-center justify-center p-4' : 'items-stretch justify-stretch',
          )}
        >
          <div
            style={
              showReplayStyle && previewLayout
                ? {
                    width: previewLayout.display.width + previewLayout.contentInset.x * 2,
                    height: previewLayout.display.height + previewLayout.contentInset.y * 2,
                  }
                : { width: '100%', height: '100%' }
            }
            className={clsx('relative', !showReplayStyle && 'h-full w-full')}
          >
            {showReplayStyle && previewLayout ? (
              <div data-theme="dark" className="relative h-full w-full">
                <ReplayPresentation
                  model={presentationModel}
                  viewportContentRect={previewContentRect ?? undefined}
                  viewportRef={viewportRef}
                  overlayNode={replay.watermark ? <WatermarkOverlay settings={replay.watermark} /> : null}
                  containerClassName="h-full w-full"
                  contentClassName="h-full w-full"
                >
                  <div className="flex h-full w-full items-center justify-center">
                    <div ref={playwrightContainerRef} className="relative flex h-full w-full overflow-hidden">
                      {sessionId && effectiveUrl ? (
                        <PlaywrightView
                          sessionId={sessionId}
                          pageId={activePageId ?? undefined}
                          refreshToken={liveRefreshToken}
                          viewport={currentViewport ?? undefined}
                          quality={streamSettings.quality}
                          fps={streamSettings.fps}
                          onStatsUpdate={handleStatsUpdate}
                          onPageMetadataChange={handlePageMetadataChange}
                          onContentRectChange={handleContentRectChange}
                          onConnectionStatusChange={onConnectionStatusChange}
                          hideConnectionIndicator={hideConnectionIndicator}
                        />
                      ) : (
                        <StartRecordingState onNavigate={onPreviewUrlChange} />
                      )}
                    </div>
                  </div>
                </ReplayPresentation>
              </div>
            ) : (
              <div className="h-full w-full">
                {sessionId && effectiveUrl ? (
                  <PlaywrightView
                    sessionId={sessionId}
                    pageId={activePageId ?? undefined}
                    refreshToken={liveRefreshToken}
                    viewport={currentViewport ?? undefined}
                    quality={streamSettings.quality}
                    fps={streamSettings.fps}
                    onStatsUpdate={handleStatsUpdate}
                    onPageMetadataChange={handlePageMetadataChange}
                    onConnectionStatusChange={onConnectionStatusChange}
                    hideConnectionIndicator={hideConnectionIndicator}
                  />
                ) : (
                  <StartRecordingState onNavigate={onPreviewUrlChange} />
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      <ResponsiveDialog
        isOpen={showReplaySettings}
        onDismiss={() => setShowReplaySettings(false)}
        ariaLabel="Replay settings"
        size="xl"
      >
        <div className="max-h-[85vh] overflow-y-auto">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-surface">Replay Settings</h2>
              <p className="text-xs text-gray-400">Tune styling for the live replay preview.</p>
            </div>
            <button
              type="button"
              onClick={() => setShowReplaySettings(false)}
              className="px-3 py-1.5 text-xs text-gray-300 hover:text-surface bg-gray-800 rounded-lg"
            >
              Close
            </button>
          </div>
          <ReplaySection
            onRandomize={() => randomizeSettings()}
            onSavePreset={() => setShowSaveDialog(true)}
          />
        </div>
      </ResponsiveDialog>

      <ResponsiveDialog
        isOpen={showSaveDialog}
        onDismiss={() => setShowSaveDialog(false)}
        ariaLabel="Save replay preset"
      >
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 w-full max-w-md">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-gray-800 rounded-lg">
              <SlidersHorizontal size={16} className="text-flow-accent" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-surface">Save Preset</h2>
              <p className="text-xs text-gray-400">Name this replay style for reuse.</p>
            </div>
          </div>
          <input
            type="text"
            value={newPresetName}
            onChange={(e) => setNewPresetName(e.target.value)}
            placeholder="My replay preset"
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700 rounded-lg text-surface focus:outline-none focus:ring-2 focus:ring-flow-accent/50 mb-4"
          />
          <div className="flex gap-3">
            <button
              type="button"
              onClick={() => setShowSaveDialog(false)}
              className="flex-1 px-4 py-2 text-subtle hover:text-surface hover:bg-gray-800 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSavePreset}
              disabled={!newPresetName.trim()}
              className="flex-1 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Save Preset
            </button>
          </div>
        </div>
      </ResponsiveDialog>
    </div>
  );
}

/** Extract domain from URL for display */
function extractDomain(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.hostname.replace(/^www\./, '');
  } catch {
    return url;
  }
}

/** Get unique recent domains from history, grouped by domain */
function getRecentDomains(history: HistoryEntry[], maxDomains = 6): Array<{ domain: string; url: string; title?: string; favicon?: string }> {
  const domainMap = new Map<string, { url: string; title?: string; lastVisited: number }>();

  for (const entry of history) {
    const domain = extractDomain(entry.url);
    const existing = domainMap.get(domain);

    // Keep the most recent entry for each domain
    if (!existing || entry.lastVisited > existing.lastVisited) {
      domainMap.set(domain, {
        url: entry.url,
        title: entry.title,
        lastVisited: entry.lastVisited,
      });
    }
  }

  // Sort by recency and take top N
  return Array.from(domainMap.entries())
    .sort((a, b) => b[1].lastVisited - a[1].lastVisited)
    .slice(0, maxDomains)
    .map(([domain, data]) => ({
      domain,
      url: data.url,
      title: data.title,
    }));
}

/** Get first letter of domain for favicon placeholder */
function getDomainInitial(domain: string): string {
  return domain.charAt(0).toUpperCase();
}

/** Generate a consistent color gradient based on domain name */
function getDomainGradient(domain: string): string {
  const gradients = [
    'from-blue-500 to-indigo-600',
    'from-purple-500 to-pink-600',
    'from-green-500 to-teal-600',
    'from-orange-500 to-red-600',
    'from-cyan-500 to-blue-600',
    'from-rose-500 to-purple-600',
  ];
  const hash = domain.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return gradients[hash % gradients.length];
}

/** Individual site card with preview data */
function SitePreviewCard({
  domain,
  url,
  historyTitle,
  preview,
  isLoading,
  onClick,
}: {
  domain: string;
  url: string;
  historyTitle?: string;
  preview: LinkPreviewData | null | undefined;
  isLoading: boolean;
  onClick: () => void;
}) {
  const hasImage = preview?.image;
  const title = preview?.title || preview?.site_name || historyTitle || domain;
  const favicon = preview?.favicon;
  const gradient = getDomainGradient(domain);

  return (
    <button
      onClick={onClick}
      className="group relative flex flex-col overflow-hidden rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 hover:shadow-lg transition-all duration-200"
      title={preview?.description || url}
    >
      {/* Image or gradient placeholder */}
      <div className="relative h-20 w-full overflow-hidden bg-gray-100 dark:bg-gray-700">
        {isLoading ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <Loader2 className="w-5 h-5 text-gray-400 animate-spin" />
          </div>
        ) : hasImage ? (
          <img
            src={preview.image}
            alt=""
            className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
            onError={(e) => {
              // Hide image on error and show gradient fallback
              (e.target as HTMLImageElement).style.display = 'none';
              const parent = (e.target as HTMLImageElement).parentElement;
              if (parent) {
                parent.classList.add('bg-gradient-to-br', ...gradient.split(' '));
              }
            }}
          />
        ) : (
          <div className={`absolute inset-0 bg-gradient-to-br ${gradient} flex items-center justify-center`}>
            <span className="text-2xl font-bold text-white/90">{getDomainInitial(domain)}</span>
          </div>
        )}
        {/* Hover overlay */}
        <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors" />
      </div>

      {/* Content */}
      <div className="flex items-center gap-2 p-2.5">
        {/* Favicon */}
        {favicon ? (
          <img
            src={favicon}
            alt=""
            className="w-4 h-4 rounded flex-shrink-0"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
        ) : (
          <div className={`w-4 h-4 rounded bg-gradient-to-br ${gradient} flex items-center justify-center flex-shrink-0`}>
            <span className="text-[8px] font-bold text-white">{getDomainInitial(domain)}</span>
          </div>
        )}
        {/* Title/Domain */}
        <span className="text-xs font-medium text-gray-700 dark:text-gray-300 truncate group-hover:text-gray-900 dark:group-hover:text-gray-100 transition-colors">
          {title}
        </span>
      </div>
    </button>
  );
}

/** Start recording state with recently-navigated domains */
function StartRecordingState({ onNavigate }: { onNavigate?: (url: string) => void }) {
  const [recentDomains, setRecentDomains] = useState<Array<{ domain: string; url: string; title?: string }>>([]);

  useEffect(() => {
    const history = loadHistory();
    setRecentDomains(getRecentDomains(history));
  }, []);

  // Fetch link previews for all recent domains
  const urls = useMemo(() => recentDomains.map((d) => d.url), [recentDomains]);
  const { previews, isLoading } = useLinkPreviewsBatch(urls);

  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      {/* Clock icon with full circle */}
      <svg className="w-12 h-12 text-gray-300 dark:text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10" strokeWidth={1.5} />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2" />
      </svg>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Start a recording session</p>
      <p className="text-xs text-gray-500 mt-1">Enter a URL above or select a recent site to begin</p>

      {/* Recent domains grid */}
      {recentDomains.length > 0 && (
        <div className="mt-6 w-full max-w-lg">
          <p className="text-xs text-gray-400 dark:text-gray-500 mb-3 uppercase tracking-wide font-medium">Recent Sites</p>
          <div className="grid grid-cols-3 gap-3">
            {recentDomains.map((item) => (
              <SitePreviewCard
                key={item.domain}
                domain={item.domain}
                url={item.url}
                historyTitle={item.title}
                preview={previews.get(item.url)}
                isLoading={isLoading && !previews.has(item.url)}
                onClick={() => onNavigate?.(item.url)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Empty state when no history */}
      {recentDomains.length === 0 && (
        <div className="mt-6 flex items-center gap-2 text-xs text-gray-400 dark:text-gray-500">
          <Globe size={14} />
          <span>Your recently visited sites will appear here</span>
        </div>
      )}
    </div>
  );
}
