/**
 * PreviewContainer - Unified container for preview panels
 *
 * This component provides a consistent presentation layer for all preview content
 * (recording, execution, workflow info). It handles:
 * - BrowserChrome (URL bar, replay style toggle, settings button)
 * - Container bounds measurement with ResizeObserver
 * - Replay style settings synchronization
 * - Presentation model computation
 * - PresentationWrapper for styled/unstyled rendering
 *
 * Content components passed as children are agnostic to whether replay styling
 * is enabled - they simply render their content and this container handles
 * the presentation wrapper.
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode, type Ref } from 'react';
import clsx from 'clsx';
import { BrowserChrome, type ExecutionStatus } from '../capture/BrowserChrome';
import type { FrameStats } from '../capture/PlaywrightView';
import type { FrameStatsAggregated } from '../hooks/usePerfStats';
import { useSettingsStore } from '@stores/settingsStore';
import { useReplaySettingsSync } from '@/domains/replay-style';
import { useReplayPresentationModel } from '@/domains/exports/replay/useReplayPresentationModel';
import ReplayPresentation from '@/domains/exports/replay/ReplayPresentation';
import { WatermarkOverlay } from '@/domains/exports/replay/WatermarkOverlay';

export interface PreviewContainerProps {
  // Replay style state (controlled from parent)
  showReplayStyle: boolean;
  onReplayStyleToggle: () => void;
  onSettingsClick: () => void;
  /** Whether the settings panel is open (hides settings button) */
  isSettingsPanelOpen?: boolean;

  // Left sidebar toggle (passed through to BrowserChrome)
  isSidebarOpen?: boolean;
  onToggleSidebar?: () => void;
  actionCount?: number;

  // BrowserChrome props
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  onNavigate?: (url: string) => void;
  onRefresh?: () => void;
  pageTitle?: string;
  placeholder?: string;

  // Frame stats (optional - shown during live streaming)
  frameStats?: FrameStats | null;
  targetFps?: number;
  debugStats?: FrameStatsAggregated | null;
  showStats?: boolean;

  // Mode context
  mode?: 'recording' | 'execution';
  executionStatus?: ExecutionStatus;
  readOnly?: boolean;

  // Content
  children: ReactNode;

  // Optional footer content (e.g., playback controls) - rendered outside the presentation wrapper
  footer?: ReactNode;

  // Optional ref for viewport element
  viewportRef?: Ref<HTMLDivElement>;

  // Additional class for the container
  className?: string;
}

export interface PreviewContainerContext {
  /** Current preview bounds (measured container size) */
  previewBounds: { width: number; height: number } | null;
  /** Computed viewport dimensions for the browser */
  activeViewport: { width: number; height: number };
  /** Whether presentation model is ready */
  isReady: boolean;
}

export function PreviewContainer({
  showReplayStyle,
  onReplayStyleToggle,
  onSettingsClick,
  isSettingsPanelOpen,
  isSidebarOpen,
  onToggleSidebar,
  actionCount,
  previewUrl,
  onPreviewUrlChange,
  onNavigate,
  onRefresh,
  pageTitle,
  placeholder = 'Search or enter URL',
  frameStats,
  targetFps,
  debugStats,
  showStats = false,
  mode = 'recording',
  readOnly = false,
  children,
  footer,
  viewportRef,
  className,
}: PreviewContainerProps) {
  const previewBoundsRef = useRef<HTMLDivElement | null>(null);
  const internalViewportRef = useRef<HTMLDivElement | null>(null);
  const [previewBounds, setPreviewBounds] = useState<{ width: number; height: number } | null>(null);

  // Get replay settings from store
  const { replay, setReplaySetting } = useSettingsStore();

  const styleOverrides = useMemo(
    () => ({
      presentation: replay.presentation,
      chromeTheme: replay.chromeTheme,
      deviceFrameTheme: replay.deviceFrameTheme,
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
      replay.deviceFrameTheme,
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

  // Sync replay settings with server
  useReplaySettingsSync({
    styleOverrides,
    extraConfig,
    onStyleHydrated: (style) => {
      setReplaySetting('presentation', style.presentation);
      setReplaySetting('chromeTheme', style.chromeTheme);
      setReplaySetting('deviceFrameTheme', style.deviceFrameTheme);
      setReplaySetting('background', style.background);
      setReplaySetting('cursorTheme', style.cursorTheme);
      setReplaySetting('cursorInitialPosition', style.cursorInitialPosition);
      setReplaySetting('cursorClickAnimation', style.cursorClickAnimation);
      setReplaySetting('cursorScale', style.cursorScale);
      setReplaySetting('browserScale', style.browserScale);
    },
  });

  // Observe preview container size
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

  // Calculate viewport dimensions
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

  // Compute presentation model
  const previewTitle = pageTitle || previewUrl || 'Preview';
  // Account for p-4 padding (16px each side = 32px) when replay style is shown
  const paddingOffset = showReplayStyle ? 32 : 0;
  const adjustedBounds = previewBounds
    ? { width: previewBounds.width - paddingOffset, height: previewBounds.height - paddingOffset }
    : undefined;
  const presentationModel = useReplayPresentationModel({
    style: styleOverrides,
    title: previewTitle,
    canvasDimensions: { width: activeViewport.width, height: activeViewport.height },
    viewportDimensions: activeViewport,
    presentationBounds: adjustedBounds,
    presentationFit: adjustedBounds ? 'contain' : 'none',
  });

  const previewLayout = showReplayStyle ? presentationModel.layout : null;

  // Merge viewport refs
  const mergedViewportRef = useCallback(
    (node: HTMLDivElement | null) => {
      internalViewportRef.current = node;
      if (typeof viewportRef === 'function') {
        viewportRef(node);
      } else if (viewportRef && typeof viewportRef === 'object') {
        (viewportRef as React.MutableRefObject<HTMLDivElement | null>).current = node;
      }
    },
    [viewportRef],
  );

  return (
    <div className={clsx('flex flex-col h-full', className)}>
      {/* Browser chrome header */}
      <BrowserChrome
        previewUrl={previewUrl}
        onPreviewUrlChange={onPreviewUrlChange}
        onNavigate={onNavigate}
        onRefresh={onRefresh}
        placeholder={placeholder}
        pageTitle={pageTitle}
        isSidebarOpen={isSidebarOpen}
        onToggleSidebar={onToggleSidebar}
        actionCount={actionCount}
        frameStats={frameStats ?? undefined}
        targetFps={targetFps}
        debugStats={debugStats ?? undefined}
        showStats={showStats}
        showReplayStyleToggle={true}
        showReplayStyle={showReplayStyle}
        onReplayStyleToggle={onReplayStyleToggle}
        onSettingsClick={onSettingsClick}
        isSettingsPanelOpen={isSettingsPanelOpen}
        mode={mode}
        readOnly={readOnly}
      />

      {/* Main content area */}
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
                  viewportRef={mergedViewportRef}
                  overlayNode={replay.watermark ? <WatermarkOverlay settings={replay.watermark} /> : null}
                  containerClassName="h-full w-full"
                  contentClassName="h-full w-full"
                >
                  <div className="h-full w-full">
                    {children}
                  </div>
                </ReplayPresentation>
              </div>
            ) : (
              <div className="h-full w-full">
                {children}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Footer (e.g., playback controls) - outside the presentation wrapper */}
      {footer}
    </div>
  );
}
