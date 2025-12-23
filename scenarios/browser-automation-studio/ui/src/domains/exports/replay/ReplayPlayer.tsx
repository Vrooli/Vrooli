/**
 * ReplayPlayer component
 *
 * Visual replay player for workflow execution timeline.
 * Orchestrates playback, cursor animation, and theme rendering.
 */

import { useEffect, useMemo, useRef, useState, type ReactNode, type CSSProperties } from 'react';
import clsx from 'clsx';

// Sub-components
import { IntroCard } from './IntroCard';
import { OutroCard } from './OutroCard';
import { WatermarkOverlay } from './WatermarkOverlay';
import {
  ReplayControls,
  ReplayStoryboard,
  ReplayEmptyState,
  ReplayMetadataPanel,
  ReplayCursorEditor,
} from './components';

// Types
import type {
  ReplayRegion,
  ReplayFrame,
  ReplayPlayerController,
  ReplayPlayerProps,
  Dimensions,
  CursorPlan,
  ReplayPoint,
  PlaybackPhase,
  ReplayBoundingBox,
} from './types';

// Constants
import {
  DEFAULT_SPEED_PROFILE,
  DEFAULT_PATH_STYLE,
  FALLBACK_DIMENSIONS,
} from './constants';

// Utilities
import { isSamePoint } from './utils/cursorMath';
import { toAbsolutePoint, pickZoomAnchor } from './utils/geometry';
import { parseRgbaComponents, rgbaWithAlpha } from './utils/formatting';
import { computeReplayLayout } from '@/domains/replay-layout';
import { toOverlayPointStyle, toOverlayRectStyle, useOverlayRegistry } from '@/domains/replay-positioning';

// Theme builders
import {
  normalizeReplayStyle,
  resolveReplayStyleTokens,
  ReplayStyleFrame,
  ReplayCanvas,
  ReplayCursorOverlay,
  useResolvedReplayBackground,
} from '@/domains/replay-style';
import type { CursorDecor } from '@/domains/replay-style';

// Hooks
import { usePlayback } from './hooks/usePlayback';
import { useCursorAnimation } from './hooks/useCursorAnimation';
import { useCursorEditor } from './hooks/useCursorEditor';
import { useClickEffect } from './hooks/useClickEffect';

// Re-export types for backward compatibility
export type {
  ReplayBoundingBox,
  ReplayRegion,
  ReplayPoint,
  ReplayScreenshot,
  ReplayRetryHistoryEntry,
  ReplayFrame,
  ReplayChromeTheme,
  ReplayBackgroundSource,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
  ReplayPlayerController,
  CursorSpeedProfile,
  CursorPathStyle,
} from './types';

interface ReplayOverlayLayerProps {
  frame: ReplayFrame;
  viewportDimensions: Dimensions;
  watermark?: ReplayPlayerProps['watermark'];
  cursorDecor: CursorDecor;
  cursorPlan?: CursorPlan;
  cursorPosition: ReplayPoint | undefined;
  cursorTrailStrokeWidth: number;
  cursorScale: number;
  shouldRenderGhost: boolean;
  isCursorEnabled: boolean;
  pointerClassName: string;
  pointerEventProps?: React.HTMLAttributes<HTMLDivElement>;
  clickEffect?: ReactNode;
  introCard?: ReplayPlayerProps['introCard'];
  outroCard?: ReplayPlayerProps['outroCard'];
  playbackPhase: PlaybackPhase;
}

const hasPoint = (point: ReplayPoint | undefined): point is { x: number; y: number } =>
  typeof point?.x === 'number' && typeof point?.y === 'number';

const hasRect = (rect: ReplayBoundingBox | undefined): rect is { x: number; y: number; width: number; height: number } =>
  typeof rect?.x === 'number'
  && typeof rect?.y === 'number'
  && typeof rect?.width === 'number'
  && typeof rect?.height === 'number';

function ReplayOverlayLayer({
  frame,
  viewportDimensions,
  watermark,
  cursorDecor,
  cursorPlan,
  cursorPosition,
  cursorTrailStrokeWidth,
  cursorScale,
  shouldRenderGhost,
  isCursorEnabled,
  pointerClassName,
  pointerEventProps,
  clickEffect,
  introCard,
  outroCard,
  playbackPhase,
}: ReplayOverlayLayerProps) {
  const registry = useOverlayRegistry();
  const overlayBounds = registry?.getRect('overlay-root') ?? null;

  const resolvePointStyle = (point: ReplayPoint | undefined, dims: Dimensions) => {
    if (!registry || !hasPoint(point)) {
      return undefined;
    }
    const resolved = registry.resolvePoint({ point, source: dims });
    return toOverlayPointStyle(resolved);
  };

  const resolveRectStyle = (box: ReplayBoundingBox | undefined) => {
    if (!registry || !hasRect(box)) {
      return undefined;
    }
    const resolved = registry.resolveRect({ rect: box, source: viewportDimensions });
    return toOverlayRectStyle(resolved);
  };

  const pointerOffsetX = cursorDecor.offset?.x ?? 0;
  const pointerOffsetY = cursorDecor.offset?.y ?? 0;
  const basePointerTransform = `translate(calc(-50% + ${pointerOffsetX}px), calc(-50% + ${pointerOffsetY}px))${
    cursorScale !== 1 ? ` scale(${cursorScale})` : ''
  }`;
  const wrapperTransform = (cursorDecor.wrapperStyle as (CSSProperties & { transform?: string }) | undefined)?.transform;
  const pointerStyle = isCursorEnabled && cursorPosition
    ? resolvePointStyle(cursorPosition, cursorPlan?.dims ?? viewportDimensions)
    : undefined;
  const pointerWrapperStyle = pointerStyle && isCursorEnabled
    ? {
        ...pointerStyle,
        ...cursorDecor.wrapperStyle,
        transform: wrapperTransform ? `${basePointerTransform} ${wrapperTransform}` : basePointerTransform,
        transformOrigin: cursorDecor.transformOrigin ?? '50% 50%',
        transitionProperty: 'left, top, transform',
      }
    : undefined;

  const ghostAbsolutePoint = cursorPlan
    ? toAbsolutePoint(cursorPlan.startNormalized, cursorPlan.dims)
    : undefined;
  const ghostStyle = ghostAbsolutePoint && cursorPlan
    ? resolvePointStyle(ghostAbsolutePoint, cursorPlan.dims)
    : undefined;
  const ghostWrapperStyle = shouldRenderGhost && ghostStyle
    ? {
        ...ghostStyle,
        ...cursorDecor.wrapperStyle,
        transform: wrapperTransform ? `${basePointerTransform} ${wrapperTransform}` : basePointerTransform,
        transformOrigin: cursorDecor.transformOrigin ?? '50% 50%',
        pointerEvents: 'none' as const,
        opacity: 0.45,
        transitionProperty: 'left, top, transform',
      }
    : undefined;

  const cursorPathPoints = useMemo(() => {
    if (!isCursorEnabled || !cursorPlan || !shouldRenderGhost) {
      return [];
    }
    return [
      cursorPlan.startNormalized,
      ...cursorPlan.pathNormalized,
      cursorPlan.targetNormalized,
    ].map((point) => toAbsolutePoint(point, cursorPlan.dims));
  }, [cursorPlan, isCursorEnabled, shouldRenderGhost]);

  const resolvedTrailPoints = useMemo(() => {
    if (!registry || cursorPathPoints.length === 0 || !cursorPlan) {
      return [];
    }
    return cursorPathPoints
      .map((point) => registry.resolvePoint({ point, source: cursorPlan.dims }))
      .filter((point): point is { x: number; y: number } => Boolean(point));
  }, [cursorPathPoints, cursorPlan, registry]);

  const scaledTrailStrokeWidth = overlayBounds && overlayBounds.width > 0
    ? cursorTrailStrokeWidth * (overlayBounds.width / 100)
    : cursorTrailStrokeWidth;
  const hasTrail = resolvedTrailPoints.length >= 2 && scaledTrailStrokeWidth > 0.05;

  const overlayRegions = (regions: ReplayRegion[] | undefined, variant: 'highlight' | 'mask') => {
    if (!regions?.length) return null;
    return regions.map((region, index) => {
      const style = resolveRectStyle(region.boundingBox);
      if (!style) return null;
      const color = region.color || '#38bdf8';
      const opacity = typeof region.opacity === 'number' ? region.opacity : variant === 'mask' ? 0.45 : 0.15;

      return (
        <div
          key={`${variant}-${index}`}
          className={clsx('absolute rounded-xl pointer-events-none transition-all duration-500', {
            'border-2 shadow-[0_0_35px_rgba(56,189,248,0.45)]': variant === 'highlight',
          })}
          style={{
            ...style,
            borderColor: variant === 'highlight' ? color : undefined,
            background: variant === 'highlight' ? 'rgba(56, 189, 248, 0.14)' : `rgba(15, 23, 42, ${opacity})`,
            boxShadow: variant === 'highlight' ? '0 16px 48px rgba(56, 189, 248, 0.35)' : undefined,
          }}
        />
      );
    });
  };

  const focusedElementStyle = frame.focusedElement?.boundingBox
    ? resolveRectStyle(frame.focusedElement.boundingBox)
    : undefined;

  return (
    <>
      {watermark ? <WatermarkOverlay settings={watermark} /> : null}
      {overlayRegions(frame.maskRegions, 'mask')}
      {overlayRegions(frame.highlightRegions, 'highlight')}
      {focusedElementStyle && (
        <div
          className="absolute rounded-2xl border border-sky-400/70 bg-sky-400/10 shadow-[0_0_60px_rgba(56,189,248,0.35)]"
          style={focusedElementStyle}
        />
      )}
      <ReplayCursorOverlay
        cursorDecor={cursorDecor}
        trailPoints={resolvedTrailPoints}
        trailStrokeWidth={scaledTrailStrokeWidth}
        showTrail={hasTrail}
        overlayBounds={overlayBounds}
        ghostStyle={ghostWrapperStyle}
        pointerStyle={pointerWrapperStyle}
        pointerClassName={pointerClassName}
        pointerEventProps={pointerEventProps}
        clickEffect={clickEffect}
      />
      {playbackPhase === 'intro' && introCard && <IntroCard settings={introCard} />}
      {playbackPhase === 'outro' && outroCard && <OutroCard settings={outroCard} />}
    </>
  );
}

export function ReplayPlayer({
  frames,
  autoPlay = true,
  loop = true,
  onFrameChange,
  onFrameProgressChange,
  executionStatus,
  chromeTheme = 'aurora',
  background,
  cursorTheme = 'white',
  cursorInitialPosition = 'center',
  cursorScale = 1,
  cursorClickAnimation = 'pulse',
  browserScale = 1,
  cursorDefaultSpeedProfile,
  cursorDefaultPathStyle,
  exposeController,
  presentationMode = 'default',
  presentationFit,
  presentationBounds,
  allowPointerEditing = true,
  presentationDimensions,
  watermark,
  introCard,
  outroCard,
}: ReplayPlayerProps) {
  const screenshotRef = useRef<HTMLDivElement | null>(null);
  const playerContainerRef = useRef<HTMLDivElement | null>(null);
  const captureAreaRef = useRef<HTMLDivElement | null>(null);
  const presentationRef = useRef<HTMLDivElement | null>(null);
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const browserFrameRef = useRef<HTMLDivElement | null>(null);

  const isExternallyControlled = typeof exposeController === 'function';
  const baseSpeedProfile = cursorDefaultSpeedProfile ?? DEFAULT_SPEED_PROFILE;
  const basePathStyle = cursorDefaultPathStyle ?? DEFAULT_PATH_STYLE;

  // Normalize frames
  const normalizedFrames = useMemo(() => {
    return frames
      .filter((frame): frame is ReplayFrame => Boolean(frame))
      .map((frame, index) => ({
        ...frame,
        id: frame.id || `${index}`,
      }));
  }, [frames]);

  // Playback state
  const {
    currentIndex,
    isPlaying,
    setIsPlaying,
    frameProgress,
    playbackPhase,
    seekToFrame,
    handleNext,
    handlePrevious,
    changeFrame,
    normalizedFramesRef,
  } = usePlayback({
    frames: normalizedFrames,
    autoPlay,
    loop,
    introCard,
    outroCard,
    isExternallyControlled,
  });

  // Derived values
  const resolvedStyle = useMemo(
    () =>
      normalizeReplayStyle({
        chromeTheme,
        background,
        cursorTheme,
        cursorInitialPosition,
        cursorScale,
        cursorClickAnimation,
        browserScale,
      }),
    [
      chromeTheme,
      background,
      cursorTheme,
      cursorInitialPosition,
      cursorScale,
      cursorClickAnimation,
      browserScale,
    ],
  );
  const resolvedBackground = useResolvedReplayBackground(resolvedStyle.background);
  const resolvedStyleWithAssets = useMemo(
    () => ({ ...resolvedStyle, background: resolvedBackground }),
    [resolvedBackground, resolvedStyle],
  );

  const effectiveCursorTheme = resolvedStyle.cursorTheme;
  const isCursorEnabled = effectiveCursorTheme !== 'disabled';
  const isExportPresentation = presentationMode === 'export';
  const showInterfaceChrome = !isExportPresentation;
  const canEditCursor = allowPointerEditing && !isExportPresentation && isCursorEnabled;

  // Cursor editor state
  const {
    cursorOverrides,
    updateCursorOverride,
    pointerEventProps,
    isDraggingCursor,
  } = useCursorEditor({
    currentFrame: normalizedFrames[currentIndex] ?? null,
    screenshotRef,
    canEditCursor,
    basePathStyle,
    baseSpeedProfile,
  });

  // Cursor animation
  const { cursorPlans, cursorPosition } = useCursorAnimation({
    frames: normalizedFrames,
    currentIndex,
    frameProgress,
    isPlaying,
    isCursorEnabled,
    cursorOverrides,
    cursorInitialPosition: resolvedStyle.cursorInitialPosition,
    basePathStyle,
    baseSpeedProfile,
  });

  // Current frame
  const currentFrame = normalizedFrames.length > 0 ? normalizedFrames[currentIndex] : null;

  // Click effects
  const { activeClickEffect, isClickEffectActive } = useClickEffect({
    cursorClickAnimation: resolvedStyle.cursorClickAnimation,
    currentFrame,
    frameProgress,
    isCursorEnabled,
  });

  const headerTitle = currentFrame
    ? currentFrame.finalUrl || currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`
    : 'Replay';

  const { backgroundDecor, chromeDecor, cursorDecor } = useMemo(
    () => resolveReplayStyleTokens(resolvedStyleWithAssets, { title: headerTitle }),
    [resolvedStyleWithAssets, headerTitle],
  );

  // Cursor scale
  const pointerScale = resolvedStyle.cursorScale;
  const cursorTrailStrokeWidth = cursorDecor.trailWidth * pointerScale;
  const frameScale = resolvedStyle.browserScale;

  // Callbacks for frame changes
  useEffect(() => {
    if (!normalizedFrames[currentIndex]) return;
    onFrameChange?.(normalizedFrames[currentIndex], currentIndex);
  }, [currentIndex, normalizedFrames, onFrameChange]);

  useEffect(() => {
    onFrameProgressChange?.(currentIndex, frameProgress);
  }, [currentIndex, frameProgress, onFrameProgressChange]);

  // Metadata panel state
  const [isMetadataCollapsed, setIsMetadataCollapsed] = useState(true);

  // Computed displays
  const extractedPreview = useMemo(() => {
    if (!currentFrame || currentFrame.extractedDataPreview == null) return undefined;
    if (typeof currentFrame.extractedDataPreview === 'string') return currentFrame.extractedDataPreview;
    try {
      return JSON.stringify(currentFrame.extractedDataPreview, null, 2);
    } catch {
      return String(currentFrame.extractedDataPreview);
    }
  }, [currentFrame]);

  const domSnapshotDisplay = useMemo(() => {
    if (!currentFrame) return undefined;
    if (typeof currentFrame.domSnapshotPreview === 'string' && currentFrame.domSnapshotPreview.trim().length > 0) {
      return currentFrame.domSnapshotPreview.trim();
    }
    if (typeof currentFrame.domSnapshotHtml === 'string' && currentFrame.domSnapshotHtml.trim().length > 0) {
      const raw = currentFrame.domSnapshotHtml.trim();
      return raw.length > 1200 ? `${raw.slice(0, 1200)}...` : raw;
    }
    return undefined;
  }, [currentFrame]);

  // Early return for empty frames
  if (normalizedFrames.length === 0 || !currentFrame) {
    return <ReplayEmptyState backgroundDecor={backgroundDecor} />;
  }

  // Dimensions
  const canvasDimensions: Dimensions = {
    width: presentationDimensions?.width || currentFrame.screenshot?.width || FALLBACK_DIMENSIONS.width,
    height: presentationDimensions?.height || currentFrame.screenshot?.height || FALLBACK_DIMENSIONS.height,
  };
  const viewportDimensions: Dimensions = {
    width: currentFrame.screenshot?.width || FALLBACK_DIMENSIONS.width,
    height: currentFrame.screenshot?.height || FALLBACK_DIMENSIONS.height,
  };
  const layout = useMemo(
    () =>
      computeReplayLayout({
        canvas: canvasDimensions,
        viewport: viewportDimensions,
        browserScale: frameScale,
        chromeHeaderHeight: chromeDecor.headerHeight,
        contentInset: backgroundDecor.contentInset,
        container: presentationBounds,
        fit: presentationFit ?? (presentationBounds ? 'contain' : 'none'),
      }),
    [
      canvasDimensions,
      viewportDimensions,
      frameScale,
      presentationBounds,
      presentationFit,
      chromeDecor.headerHeight,
      backgroundDecor.contentInset,
    ],
  );
  const aspectRatio = viewportDimensions.width > 0
    ? (viewportDimensions.height / viewportDimensions.width) * 100
    : 56.25;

  // Controller
  const controller = useMemo<ReplayPlayerController | null>(() => {
    if (!exposeController) return null;
    return {
      seek: ({ frameIndex, progress }) => seekToFrame(frameIndex, progress),
      play: () => setIsPlaying(true),
      pause: () => setIsPlaying(false),
      getViewportElement: () => {
        if (isExportPresentation) {
          return viewportRef.current
            ?? browserFrameRef.current
            ?? captureAreaRef.current
            ?? presentationRef.current
            ?? screenshotRef.current
            ?? playerContainerRef.current;
        }
        return viewportRef.current ?? presentationRef.current ?? playerContainerRef.current ?? screenshotRef.current;
      },
      getPresentationElement: () => presentationRef.current ?? playerContainerRef.current,
      getLayout: () => layout,
      getFrameCount: () => normalizedFramesRef.current.length,
    };
  }, [exposeController, isExportPresentation, seekToFrame, setIsPlaying, normalizedFramesRef, layout]);

  useEffect(() => {
    if (!exposeController) return;
    exposeController(controller);
    return () => exposeController(null);
  }, [controller, exposeController]);

  // Zoom
  const zoom = currentFrame.zoomFactor && currentFrame.zoomFactor > 1 ? Math.min(currentFrame.zoomFactor, 3) : 1;
  const zoomAnchor = pickZoomAnchor(currentFrame);
  const anchorStyle = (() => {
    if (!zoomAnchor) return '50% 50%';
    if (!viewportDimensions.width || !viewportDimensions.height) return '50% 50%';
    const width = Math.max(0, Math.min(zoomAnchor.width ?? 0, viewportDimensions.width));
    const height = Math.max(0, Math.min(zoomAnchor.height ?? 0, viewportDimensions.height));
    const left = Math.max(0, Math.min(zoomAnchor.x ?? 0, viewportDimensions.width - width));
    const top = Math.max(0, Math.min(zoomAnchor.y ?? 0, viewportDimensions.height - height));
    const centerX = ((left + width / 2) / viewportDimensions.width) * 100;
    const centerY = ((top + height / 2) / viewportDimensions.height) * 100;
    return `${centerX}% ${centerY}%`;
  })();

  // Cursor rendering
  const currentCursorPlan = cursorPlans[currentIndex];
  const hasMovement = Boolean(
    currentCursorPlan && !isSamePoint(currentCursorPlan.startNormalized, currentCursorPlan.targetNormalized),
  );
  const shouldRenderGhost = Boolean(isCursorEnabled && currentCursorPlan && !isPlaying && hasMovement);

  const pointerWrapperClassName = clsx(
    'absolute transition-all duration-500 ease-out select-none relative',
    cursorDecor.wrapperClass,
    canEditCursor ? 'pointer-events-auto' : 'pointer-events-none',
    canEditCursor ? (isDraggingCursor ? 'cursor-grabbing' : 'cursor-grab') : 'cursor-default',
  );

  // Click effect
  let clickEffectElement: ReactNode = null;
  if (isClickEffectActive && activeClickEffect) {
    const rgbaComponents = parseRgbaComponents(cursorDecor.trailColor);
    const strokeColor = cursorClickAnimation === 'pulse'
      ? rgbaWithAlpha(rgbaComponents, 0.82)
      : rgbaWithAlpha(rgbaComponents, 0.68);
    const haloColor = rgbaWithAlpha(rgbaComponents, cursorClickAnimation === 'pulse' ? 0.18 : 0.12);

    if (cursorClickAnimation === 'pulse') {
      clickEffectElement = (
        <span
          key={activeClickEffect.key}
          className="pointer-events-none absolute -inset-3 rounded-full border-2 cursor-click-pulse"
          style={{
            borderColor: strokeColor,
            backgroundColor: rgbaWithAlpha(rgbaComponents, 0.22),
            boxShadow: `0 0 0 0.35rem ${haloColor}`,
          }}
        />
      );
    } else if (cursorClickAnimation === 'ripple') {
      clickEffectElement = (
        <span
          key={activeClickEffect.key}
          className="pointer-events-none absolute -inset-4 rounded-full border cursor-click-ripple"
          style={{
            borderColor: strokeColor,
            boxShadow: `0 0 0 0.45rem ${haloColor}`,
          }}
        />
      );
    }
  }

  const displayDurationMs = currentFrame.totalDurationMs ?? currentFrame.durationMs;
  const screenshotTransition = isExportPresentation ? 'none' : 'transform 600ms ease, transform-origin 600ms ease';

  // Cursor behavior panel
  const cursorBehaviorPanel = currentCursorPlan ? (
    <ReplayCursorEditor
      cursorPlan={currentCursorPlan}
      currentOverride={cursorOverrides[currentCursorPlan.frameId]}
      cursorPosition={cursorPosition}
      baseSpeedProfile={baseSpeedProfile}
      basePathStyle={basePathStyle}
      onUpdateOverride={updateCursorOverride}
      onResetAll={() => updateCursorOverride(currentCursorPlan.frameId, () => undefined)}
    />
  ) : undefined;

  const overlayTransformStyle: CSSProperties | undefined = zoom !== 1
    ? { transform: `scale(${zoom})`, transformOrigin: anchorStyle, transition: screenshotTransition }
    : undefined;

  return (
    <div ref={playerContainerRef} className="flex flex-col gap-4">
      <ReplayStyleFrame
        backgroundDecor={backgroundDecor}
        chromeDecor={chromeDecor}
        layout={layout}
        presentationRef={presentationRef}
        viewportRef={viewportRef}
        showInterfaceChrome={showInterfaceChrome}
        captureAreaRef={captureAreaRef}
        browserFrameRef={browserFrameRef}
        containerClassName={clsx(!isExportPresentation && 'transition-all duration-500')}
        overlayTransformStyle={overlayTransformStyle}
        overlayNode={
          <ReplayOverlayLayer
            frame={currentFrame}
            viewportDimensions={viewportDimensions}
            watermark={watermark}
            cursorDecor={cursorDecor}
            cursorPlan={currentCursorPlan}
            cursorPosition={cursorPosition}
            cursorTrailStrokeWidth={cursorTrailStrokeWidth}
            cursorScale={pointerScale}
            shouldRenderGhost={shouldRenderGhost}
            isCursorEnabled={isCursorEnabled}
            pointerClassName={pointerWrapperClassName}
            pointerEventProps={pointerEventProps}
            clickEffect={clickEffectElement}
            introCard={introCard}
            outroCard={outroCard}
            playbackPhase={playbackPhase}
          />
        }
        header={showInterfaceChrome ? (
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-200/80">
            <span>Replay</span>
            <span>Step {currentIndex + 1} / {normalizedFrames.length}</span>
          </div>
        ) : null}
        footer={showInterfaceChrome ? (
          <ReplayControls
            isPlaying={isPlaying}
            onPlayPause={() => setIsPlaying((prev) => !prev)}
            onPrevious={handlePrevious}
            onNext={handleNext}
            frameProgress={frameProgress}
            displayDurationMs={displayDurationMs}
          />
        ) : null}
      >
        <div className={clsx(!isExportPresentation && 'transition-all duration-300')}>
          <ReplayCanvas
            width={layout.viewportRect.width}
            height={layout.viewportRect.height}
            zoom={zoom}
            anchorStyle={anchorStyle}
            screenshotRef={screenshotRef}
            screenshotUrl={currentFrame.screenshot?.url}
            screenshotAlt={currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`}
            transition={screenshotTransition}
          >
            {null}
          </ReplayCanvas>
        </div>
      </ReplayStyleFrame>

      {showInterfaceChrome && (
        <ReplayMetadataPanel
          frame={currentFrame}
          isCollapsed={isMetadataCollapsed}
          onToggle={() => setIsMetadataCollapsed((prev) => !prev)}
          extractedPreview={extractedPreview}
          domSnapshotDisplay={domSnapshotDisplay}
          cursorBehaviorPanel={cursorBehaviorPanel}
        />
      )}

      {showInterfaceChrome && (
        <ReplayStoryboard
          frames={normalizedFrames}
          currentIndex={currentIndex}
          aspectRatio={aspectRatio}
          onSelectFrame={changeFrame}
        />
      )}

      {showInterfaceChrome && executionStatus === 'failed' && frames.length > 0 && (
        <div className="mt-6 mx-4 p-6 rounded-lg border border-dashed border-gray-600 bg-gray-900/30 text-center">
          <div className="text-sm text-gray-400 mb-2">
            Workflow execution stopped at step {frames.length}
          </div>
          <div className="text-xs text-gray-500">
            Subsequent steps were not executed due to failure
          </div>
        </div>
      )}
    </div>
  );
}

export default ReplayPlayer;
