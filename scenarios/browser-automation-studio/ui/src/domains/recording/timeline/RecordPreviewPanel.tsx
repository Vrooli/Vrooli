import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { SlidersHorizontal, Palette } from 'lucide-react';
import type { RecordedAction } from '../types/types';
import { PlaywrightView, type FrameStats, type PageMetadata } from '../capture/PlaywrightView';
import { StreamSettings, useStreamSettings, type StreamSettingsValues } from '../capture/StreamSettings';
import { FrameStatsDisplay } from '../capture/FrameStatsDisplay';
import { BrowserUrlBar } from '../capture/BrowserUrlBar';
import { usePerfStats } from '../hooks/usePerfStats';
import { useSettingsStore } from '@stores/settingsStore';
import { buildBackgroundDecor, buildChromeDecor } from '@/domains/exports/replay/themes';
import { WatermarkOverlay } from '@/domains/exports/replay/WatermarkOverlay';
import { ReplaySection } from '@/views/SettingsView/sections/ReplaySection';
import ResponsiveDialog from '@shared/layout/ResponsiveDialog';
import clsx from 'clsx';

interface RecordPreviewPanelProps {
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  onViewportChange?: (size: { width: number; height: number }) => void;
  /** Callback when stream settings change (for session creation) */
  onStreamSettingsChange?: (settings: StreamSettingsValues) => void;
}

export function RecordPreviewPanel({
  previewUrl,
  onPreviewUrlChange,
  actions,
  sessionId,
  onViewportChange,
  onStreamSettingsChange,
}: RecordPreviewPanelProps) {
  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  const effectiveUrl = previewUrl || lastUrl;

  const [liveRefreshToken, setLiveRefreshToken] = useState(0);
  const previewBoundsRef = useRef<HTMLDivElement | null>(null);
  const [previewBounds, setPreviewBounds] = useState<{ width: number; height: number } | null>(null);
  const lastViewportRef = useRef<{ width: number; height: number } | null>(null);
  // Track viewport for passing to PlaywrightView (for coordinate mapping)
  const [currentViewport, setCurrentViewport] = useState<{ width: number; height: number } | null>(null);
  const [showReplayStyle, setShowReplayStyle] = useState(false);
  const [showReplaySettings, setShowReplaySettings] = useState(false);
  const [showSaveDialog, setShowSaveDialog] = useState(false);
  const [newPresetName, setNewPresetName] = useState('');

  // Stream settings management
  const { preset, settings: streamSettings, showStats, setPreset, setShowStats } = useStreamSettings();

  // Debug performance stats from server (enabled when showStats is on)
  const { stats: perfStats } = usePerfStats(sessionId ?? null, showStats);
  const {
    replay,
    randomizeSettings,
    saveAsPreset,
  } = useSettingsStore();

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

  useEffect(() => {
    const width = Math.min(3840, Math.max(320, Math.round(replay.presentationWidth)));
    const height = Math.min(3840, Math.max(320, Math.round(replay.presentationHeight)));
    const target = { width, height };
    const last = lastViewportRef.current;
    if (!last || last.width !== target.width || last.height !== target.height) {
      lastViewportRef.current = target;
      setCurrentViewport(target);
      onViewportChange?.(target);
    }
  }, [onViewportChange, replay.presentationHeight, replay.presentationWidth]);

  const backgroundDecor = useMemo(
    () => buildBackgroundDecor(replay.backgroundTheme),
    [replay.backgroundTheme],
  );

  const chromeDecor = useMemo(
    () => buildChromeDecor(replay.chromeTheme, pageTitle || effectiveUrl || 'Live Preview'),
    [replay.chromeTheme, pageTitle, effectiveUrl],
  );

  const targetWidth = currentViewport?.width ?? 1280;
  const targetHeight = currentViewport?.height ?? 720;
  const previewScale = useMemo(() => {
    if (!previewBounds) return 1;
    const scaleX = previewBounds.width / targetWidth;
    const scaleY = previewBounds.height / targetHeight;
    const nextScale = Math.min(scaleX, scaleY, 1);
    return Number.isFinite(nextScale) && nextScale > 0 ? nextScale : 1;
  }, [previewBounds, targetHeight, targetWidth]);

  const scaledWidth = Math.round(targetWidth * previewScale);
  const scaledHeight = Math.round(targetHeight * previewScale);

  const handleSavePreset = useCallback(() => {
    const trimmedName = newPresetName.trim();
    if (!trimmedName) return;
    saveAsPreset(trimmedName);
    setNewPresetName('');
    setShowSaveDialog(false);
  }, [newPresetName, saveAsPreset]);

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
            'h-full w-full flex items-center justify-center p-4',
            showReplayStyle ? 'bg-slate-950/95' : 'bg-transparent',
          )}
        >
          <div style={{ width: scaledWidth, height: scaledHeight }} className="relative">
            <div
              style={{ width: targetWidth, height: targetHeight, transform: `scale(${previewScale})`, transformOrigin: 'top left' }}
              className="relative"
            >
              {showReplayStyle ? (
                <div
                  data-theme="dark"
                  className={clsx('relative h-full w-full', backgroundDecor.containerClass)}
                  style={backgroundDecor.containerStyle}
                >
                  {backgroundDecor.baseLayer}
                  {backgroundDecor.overlay}
                  <div className={clsx('relative z-10 h-full w-full', backgroundDecor.contentClass)}>
                    <div className={clsx('flex h-full w-full flex-col overflow-hidden', chromeDecor.frameClass)}>
                      {chromeDecor.header}
                      <div className={clsx('relative flex-1 overflow-hidden', chromeDecor.contentClass)}>
                        {sessionId ? (
                          effectiveUrl ? (
                            <PlaywrightView
                              sessionId={sessionId}
                              refreshToken={liveRefreshToken}
                              viewport={currentViewport ?? undefined}
                              quality={streamSettings.quality}
                              fps={streamSettings.fps}
                              onStatsUpdate={handleStatsUpdate}
                              onPageMetadataChange={handlePageMetadataChange}
                            />
                          ) : (
                            <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the actual Playwright session." />
                          )
                        ) : (
                          <EmptyState title="Start a recording session" subtitle="Create or resume a recording session to view the live browser." />
                        )}
                        <WatermarkOverlay settings={replay.watermark} />
                      </div>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="h-full w-full">
                  {sessionId ? (
                    effectiveUrl ? (
                      <PlaywrightView
                        sessionId={sessionId}
                        refreshToken={liveRefreshToken}
                        viewport={currentViewport ?? undefined}
                        quality={streamSettings.quality}
                        fps={streamSettings.fps}
                        onStatsUpdate={handleStatsUpdate}
                        onPageMetadataChange={handlePageMetadataChange}
                      />
                    ) : (
                      <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the actual Playwright session." />
                    )
                  ) : (
                    <EmptyState title="Start a recording session" subtitle="Create or resume a recording session to view the live browser." />
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      <ResponsiveDialog
        isOpen={showReplaySettings}
        onDismiss={() => setShowReplaySettings(false)}
        ariaLabel="Replay settings"
        size="xl"
      >
        <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 max-h-[85vh] overflow-y-auto">
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

function EmptyState({ title, subtitle, actionLabel, onAction }: { title: string; subtitle?: string; actionLabel?: string; onAction?: () => void }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      <svg className="w-10 h-10 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2" />
      </svg>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{title}</p>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
      {actionLabel && onAction && (
        <button
          className="mt-3 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600"
          onClick={onAction}
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
