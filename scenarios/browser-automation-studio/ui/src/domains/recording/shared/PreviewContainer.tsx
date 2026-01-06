/**
 * PreviewContainer - Unified container for preview panels
 *
 * This component provides a consistent presentation layer for all preview content
 * (recording, execution, workflow info). It handles:
 * - BrowserChrome (URL bar, replay style toggle, settings button)
 * - Container bounds measurement with ResizeObserver
 * - Replay style settings synchronization
 * - Presentation model computation
 * - StablePreviewWrapper for styled/unstyled rendering (no remounts!)
 *
 * KEY ARCHITECTURAL DECISIONS:
 *
 * 1. Children always stay mounted: Unlike the previous implementation that used
 *    conditional rendering (causing React to unmount/remount PlaywrightView),
 *    this version uses StablePreviewWrapper which keeps children in a stable
 *    DOM position regardless of replay style toggle.
 *
 * 2. Browser viewport is decoupled from display: The actual browser viewport
 *    (sent to Playwright) is based on container bounds, NOT replay settings.
 *    This prevents CDP screencast restarts when toggling replay style.
 *
 * 3. Display scaling is CSS-only: When replay style is enabled, the content
 *    is visually scaled via CSS transforms, not by changing the browser viewport.
 */

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode, type Ref } from 'react';
import clsx from 'clsx';
import { BrowserChrome, type ExecutionStatus, type NavigationStackData } from '../capture/BrowserChrome';
import type { FrameStats } from '../capture/PlaywrightView';
import type { FrameStatsAggregated } from '../hooks/usePerfStats';
import { useSettingsStore } from '@stores/settingsStore';
import { useReplaySettingsSync } from '@/domains/replay-style';
import { useReplayPresentationModel } from '@/domains/exports/replay/useReplayPresentationModel';
import { StablePreviewWrapper } from './StablePreviewWrapper';
import { WatermarkOverlay } from '@/domains/exports/replay/WatermarkOverlay';
import { useViewportOptional } from '../context';

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
  pageTitle?: string;
  placeholder?: string;

  // Navigation controls (back, forward, refresh)
  onGoBack?: () => void;
  onGoForward?: () => void;
  onRefresh?: () => void;
  canGoBack?: boolean;
  canGoForward?: boolean;

  // Navigation history popup (right-click on back/forward buttons)
  onFetchNavigationStack?: () => Promise<NavigationStackData | null>;
  onNavigateToIndex?: (delta: number) => void;
  onOpenHistorySettings?: () => void;

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

  /**
   * Callback when browser viewport changes (container-based, stable).
   * This is the viewport that should be sent to Playwright - it does NOT
   * change when toggling replay style.
   */
  onBrowserViewportChange?: (viewport: { width: number; height: number }) => void;

  /**
   * Optional: Override browser viewport (e.g., from session profile).
   * When set, this indicates the actual browser dimensions may differ from requested.
   */
  actualBrowserViewport?: { width: number; height: number } | null;

  /**
   * Whether viewport sync is in progress
   */
  isViewportSyncing?: boolean;

  // Additional class for the container
  className?: string;
}

export interface PreviewContainerContext {
  /** Current preview bounds (measured container size) */
  previewBounds: { width: number; height: number } | null;
  /**
   * Browser viewport - what Playwright uses (container-based, stable).
   * Does NOT change when toggling replay style.
   */
  browserViewport: { width: number; height: number } | null;
  /**
   * Display viewport - how content is visually rendered.
   * May differ from browserViewport when replay style is enabled (CSS scaling).
   */
  displayViewport: { width: number; height: number } | null;
  /** Whether presentation model is ready */
  isReady: boolean;
}

// Constants
const MIN_DIMENSION = 320;
const MAX_DIMENSION = 3840;

function clampDimension(value: number): number {
  return Math.min(MAX_DIMENSION, Math.max(MIN_DIMENSION, Math.round(value)));
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
  pageTitle,
  placeholder = 'Search or enter URL',
  onGoBack,
  onGoForward,
  onRefresh,
  canGoBack,
  canGoForward,
  onFetchNavigationStack,
  onNavigateToIndex,
  onOpenHistorySettings,
  frameStats,
  targetFps,
  debugStats,
  showStats = false,
  mode = 'recording',
  readOnly = false,
  children,
  footer,
  viewportRef,
  onBrowserViewportChange,
  actualBrowserViewport,
  isViewportSyncing,
  className,
}: PreviewContainerProps) {
  const previewBoundsRef = useRef<HTMLDivElement | null>(null);
  const internalViewportRef = useRef<HTMLDivElement | null>(null);
  const [previewBounds, setPreviewBounds] = useState<{ width: number; height: number } | null>(null);

  // Track if this is the first render to avoid initial viewport change callback
  const isFirstRenderRef = useRef(true);

  // Get viewport context if available (for recording mode with ViewportProvider)
  const viewportContext = useViewportOptional();

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

  /**
   * Browser viewport: ALWAYS based on container bounds, NOT replay settings.
   *
   * This is the key architectural change. The browser viewport should be stable
   * and not change when toggling replay style. This prevents:
   * - CDP screencast restarts
   * - Frame flickering
   * - WebSocket subscription resets
   */
  const browserViewport = useMemo(() => {
    if (!previewBounds) return null;
    return {
      width: clampDimension(previewBounds.width),
      height: clampDimension(previewBounds.height),
    };
  }, [previewBounds]);

  // Notify parent AND context when browser viewport changes (for syncing to Playwright)
  useEffect(() => {
    // Skip the first render to avoid initial viewport change callback
    if (isFirstRenderRef.current) {
      isFirstRenderRef.current = false;
      // Still call the callback/context on first render if we have bounds
      if (browserViewport) {
        onBrowserViewportChange?.(browserViewport);
        viewportContext?.updateFromBounds(browserViewport);
      }
      return;
    }

    if (browserViewport) {
      // Call prop callback (for session creation in RecordingSession)
      onBrowserViewportChange?.(browserViewport);
      // Update context (for viewport sync to backend)
      viewportContext?.updateFromBounds(browserViewport);
    }
  }, [browserViewport, onBrowserViewportChange, viewportContext]);

  /**
   * Display dimensions for the presentation model.
   * When replay style is enabled, use the replay settings.
   * When disabled, use the browser viewport (container-based).
   *
   * IMPORTANT: These dimensions only affect visual rendering (CSS),
   * NOT the actual browser viewport.
   */
  const displayDimensions = useMemo(() => {
    if (showReplayStyle) {
      return {
        width: clampDimension(replay.presentationWidth),
        height: clampDimension(replay.presentationHeight),
      };
    }
    // When unstyled, display matches browser viewport
    return browserViewport ?? { width: 1280, height: 720 };
  }, [showReplayStyle, replay.presentationWidth, replay.presentationHeight, browserViewport]);

  // Compute presentation model for styled rendering
  const previewTitle = pageTitle || previewUrl || 'Preview';
  // Account for p-4 padding (16px each side = 32px) when replay style is shown
  const paddingOffset = showReplayStyle ? 32 : 0;
  const adjustedBounds = previewBounds
    ? { width: previewBounds.width - paddingOffset, height: previewBounds.height - paddingOffset }
    : undefined;

  const presentationModel = useReplayPresentationModel({
    style: styleOverrides,
    title: previewTitle,
    canvasDimensions: displayDimensions,
    viewportDimensions: browserViewport ?? displayDimensions,
    presentationBounds: adjustedBounds,
    presentationFit: adjustedBounds ? 'contain' : 'none',
  });

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

  // Use context values when available, fall back to props
  const effectiveActualViewport = viewportContext?.actualViewport ?? actualBrowserViewport;
  const effectiveIsSyncing = viewportContext?.syncState.isSyncing ?? isViewportSyncing;
  // Get the viewport reason - prefer actualViewport.reason (always present), fall back to mismatchReason
  const effectiveViewportReason =
    viewportContext?.actualViewport?.reason ??
    viewportContext?.mismatchReason ??
    null;

  // Check for dimension mismatch (browser profile override)
  const hasDimensionMismatch = useMemo(() => {
    // Use context mismatch if available
    if (viewportContext) {
      return viewportContext.hasMismatch;
    }
    // Fall back to prop-based calculation
    if (!browserViewport || !actualBrowserViewport) return false;
    const widthDiff = Math.abs(browserViewport.width - actualBrowserViewport.width);
    const heightDiff = Math.abs(browserViewport.height - actualBrowserViewport.height);
    return widthDiff > 5 || heightDiff > 5;
  }, [viewportContext, browserViewport, actualBrowserViewport]);

  // Overlay for watermark
  const overlayNode = useMemo(() => {
    if (!replay.watermark?.enabled) return null;
    return <WatermarkOverlay settings={replay.watermark} />;
  }, [replay.watermark]);

  return (
    <div className={clsx('flex flex-col h-full', className)}>
      {/* Browser chrome header */}
      <BrowserChrome
        previewUrl={previewUrl}
        onPreviewUrlChange={onPreviewUrlChange}
        onNavigate={onNavigate}
        placeholder={placeholder}
        pageTitle={pageTitle}
        onGoBack={onGoBack}
        onGoForward={onGoForward}
        onRefresh={onRefresh}
        canGoBack={canGoBack}
        canGoForward={canGoForward}
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
        // Navigation history popup
        onFetchNavigationStack={onFetchNavigationStack}
        onNavigateToIndex={onNavigateToIndex}
        onOpenHistorySettings={onOpenHistorySettings}
        // New props for viewport indicator
        browserViewport={browserViewport}
        actualBrowserViewport={effectiveActualViewport}
        hasDimensionMismatch={hasDimensionMismatch}
        isViewportSyncing={effectiveIsSyncing}
        viewportReason={effectiveViewportReason ?? undefined}
      />

      {/* Main content area */}
      <div className="flex-1 overflow-hidden" ref={previewBoundsRef}>
        <div
          className={clsx(
            'h-full w-full flex transition-colors duration-200',
            showReplayStyle ? 'bg-slate-950/95' : 'bg-transparent',
            showReplayStyle ? 'items-center justify-center p-4' : 'items-stretch justify-stretch',
          )}
        >
          {/*
           * StablePreviewWrapper: Children ALWAYS stay mounted!
           *
           * This is the key to preventing flickering. Unlike the previous
           * conditional rendering approach, this wrapper keeps children
           * in a stable DOM position regardless of showReplayStyle.
           */}
          <StablePreviewWrapper
            showReplayStyle={showReplayStyle}
            layout={presentationModel.layout}
            backgroundDecor={presentationModel.backgroundDecor}
            chromeDecor={presentationModel.chromeDecor}
            deviceFrameDecor={presentationModel.deviceFrameDecor}
            viewportRef={mergedViewportRef}
            overlayNode={overlayNode}
          >
            {children}
          </StablePreviewWrapper>
        </div>
      </div>

      {/* Footer (e.g., playback controls) - outside the presentation wrapper */}
      {footer}
    </div>
  );
}
