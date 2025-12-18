/**
 * ReplayPlayer component
 *
 * Visual replay player for workflow execution timeline.
 * Orchestrates playback, cursor animation, and theme rendering.
 */

import { useEffect, useMemo, useRef, useState, type ReactNode, type CSSProperties } from 'react';
import clsx from 'clsx';

// Sub-components
import { WatermarkOverlay } from './WatermarkOverlay';
import { IntroCard } from './IntroCard';
import { OutroCard } from './OutroCard';
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
} from './types';

// Constants
import {
  MIN_CURSOR_SCALE,
  MAX_CURSOR_SCALE,
  DEFAULT_SPEED_PROFILE,
  DEFAULT_PATH_STYLE,
  FALLBACK_DIMENSIONS,
} from './constants';

// Utilities
import { clamp01, isSamePoint } from './utils/cursorMath';
import { toAbsolutePoint, toRectStyle, toPointStyle, pickZoomAnchor } from './utils/geometry';
import { parseRgbaComponents, rgbaWithAlpha } from './utils/formatting';

// Theme builders
import { buildBackgroundDecor, buildCursorDecor, buildChromeDecor } from './themes';

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
  ReplayBackgroundTheme,
  ReplayCursorTheme,
  ReplayCursorInitialPosition,
  ReplayCursorClickAnimation,
  ReplayPlayerController,
  CursorSpeedProfile,
  CursorPathStyle,
} from './types';

export function ReplayPlayer({
  frames,
  autoPlay = true,
  loop = true,
  onFrameChange,
  onFrameProgressChange,
  executionStatus,
  chromeTheme = 'aurora',
  backgroundTheme = 'aurora',
  cursorTheme = 'white',
  cursorInitialPosition = 'center',
  cursorScale = 1,
  cursorClickAnimation = 'none',
  cursorDefaultSpeedProfile,
  cursorDefaultPathStyle,
  exposeController,
  presentationMode = 'default',
  allowPointerEditing = true,
  presentationDimensions,
  watermark,
  introCard,
  outroCard,
}: ReplayPlayerProps) {
  const screenshotRef = useRef<HTMLDivElement | null>(null);
  const playerContainerRef = useRef<HTMLDivElement | null>(null);
  const captureAreaRef = useRef<HTMLDivElement | null>(null);

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

  // Cursor editor state
  const {
    cursorOverrides,
    updateCursorOverride,
    pointerEventProps,
    isDraggingCursor,
  } = useCursorEditor({
    currentFrame: normalizedFrames[currentIndex] ?? null,
    screenshotRef,
    canEditCursor: allowPointerEditing && presentationMode !== 'export' && cursorTheme !== 'disabled',
    basePathStyle,
    baseSpeedProfile,
  });

  // Derived values
  const effectiveCursorTheme = cursorTheme ?? 'white';
  const isCursorEnabled = effectiveCursorTheme !== 'disabled';
  const isExportPresentation = presentationMode === 'export';
  const showInterfaceChrome = !isExportPresentation;
  const canEditCursor = allowPointerEditing && !isExportPresentation && isCursorEnabled;

  // Cursor animation
  const { cursorPlans, cursorPosition } = useCursorAnimation({
    frames: normalizedFrames,
    currentIndex,
    frameProgress,
    isPlaying,
    isCursorEnabled,
    cursorOverrides,
    cursorInitialPosition,
    basePathStyle,
    baseSpeedProfile,
  });

  // Current frame
  const currentFrame = normalizedFrames.length > 0 ? normalizedFrames[currentIndex] : null;

  // Click effects
  const { activeClickEffect, isClickEffectActive } = useClickEffect({
    cursorClickAnimation,
    currentFrame,
    frameProgress,
    isCursorEnabled,
  });

  // Theme decors
  const backgroundDecor = useMemo(
    () => buildBackgroundDecor(backgroundTheme ?? 'aurora'),
    [backgroundTheme],
  );

  const cursorDecor = useMemo(() => buildCursorDecor(effectiveCursorTheme), [effectiveCursorTheme]);

  const headerTitle = currentFrame
    ? currentFrame.finalUrl || currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`
    : 'Replay';
  const chromeVariant = chromeTheme ?? 'aurora';
  const chromeDecor = useMemo(
    () => buildChromeDecor(chromeVariant, headerTitle),
    [chromeVariant, headerTitle],
  );

  // Cursor scale
  const pointerScale =
    typeof cursorScale === 'number' && !Number.isNaN(cursorScale)
      ? Math.min(MAX_CURSOR_SCALE, Math.max(MIN_CURSOR_SCALE, cursorScale))
      : 1;
  const cursorTrailStrokeWidth = cursorDecor.trailWidth * pointerScale;

  // Callbacks for frame changes
  useEffect(() => {
    if (!normalizedFrames[currentIndex]) return;
    onFrameChange?.(normalizedFrames[currentIndex], currentIndex);
  }, [currentIndex, normalizedFrames, onFrameChange]);

  useEffect(() => {
    onFrameProgressChange?.(currentIndex, frameProgress);
  }, [currentIndex, frameProgress, onFrameProgressChange]);

  // Controller
  const controller = useMemo<ReplayPlayerController | null>(() => {
    if (!exposeController) return null;
    return {
      seek: ({ frameIndex, progress }) => seekToFrame(frameIndex, progress),
      play: () => setIsPlaying(true),
      pause: () => setIsPlaying(false),
      getViewportElement: () => {
        if (isExportPresentation) {
          return captureAreaRef.current ?? screenshotRef.current ?? playerContainerRef.current;
        }
        return playerContainerRef.current ?? screenshotRef.current;
      },
      getPresentationElement: () => playerContainerRef.current,
      getFrameCount: () => normalizedFramesRef.current.length,
    };
  }, [exposeController, isExportPresentation, seekToFrame, setIsPlaying, normalizedFramesRef]);

  useEffect(() => {
    if (!exposeController) return;
    exposeController(controller);
    return () => exposeController(null);
  }, [controller, exposeController]);

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
  const dimensions: Dimensions = {
    width: presentationDimensions?.width || currentFrame.screenshot?.width || FALLBACK_DIMENSIONS.width,
    height: presentationDimensions?.height || currentFrame.screenshot?.height || FALLBACK_DIMENSIONS.height,
  };
  const aspectRatio = dimensions.width > 0 ? (dimensions.height / dimensions.width) * 100 : 56.25;

  // Zoom
  const zoom = currentFrame.zoomFactor && currentFrame.zoomFactor > 1 ? Math.min(currentFrame.zoomFactor, 3) : 1;
  const zoomAnchor = pickZoomAnchor(currentFrame);
  const anchorStyle = (() => {
    if (!zoomAnchor) return '50% 50%';
    const style = toRectStyle(zoomAnchor, dimensions);
    if (!style) return '50% 50%';
    const left = parseFloat(style.left);
    const top = parseFloat(style.top);
    const width = parseFloat(style.width);
    const height = parseFloat(style.height);
    return `${left + width / 2}% ${top + height / 2}%`;
  })();

  // Cursor rendering
  const currentCursorPlan = cursorPlans[currentIndex];
  const cursorTrailPoints = isCursorEnabled && currentCursorPlan
    ? [
        currentCursorPlan.startNormalized,
        ...currentCursorPlan.pathNormalized,
        currentCursorPlan.targetNormalized,
      ].map((point) => ({ x: clamp01(point.x) * 100, y: clamp01(point.y) * 100 }))
    : [];
  const hasMovement = Boolean(
    currentCursorPlan && !isSamePoint(currentCursorPlan.startNormalized, currentCursorPlan.targetNormalized),
  );
  const shouldRenderGhost = Boolean(isCursorEnabled && currentCursorPlan && !isPlaying && hasMovement);
  const renderTrailPoints = shouldRenderGhost ? cursorTrailPoints : [];
  const hasTrail = renderTrailPoints.length >= 2 && cursorTrailStrokeWidth > 0.05;

  // Pointer styles
  const pointerStyle = isCursorEnabled && cursorPosition
    ? toPointStyle(cursorPosition, currentCursorPlan?.dims ?? dimensions)
    : undefined;
  const pointerOffsetX = cursorDecor.offset?.x ?? 0;
  const pointerOffsetY = cursorDecor.offset?.y ?? 0;
  const basePointerTransform = `translate(calc(-50% + ${pointerOffsetX}px), calc(-50% + ${pointerOffsetY}px))${
    pointerScale !== 1 ? ` scale(${pointerScale})` : ''
  }`;
  const wrapperTransform = (cursorDecor.wrapperStyle as (CSSProperties & { transform?: string }) | undefined)?.transform;
  const pointerWrapperStyle = pointerStyle && isCursorEnabled
    ? {
        ...pointerStyle,
        ...cursorDecor.wrapperStyle,
        transform: wrapperTransform ? `${basePointerTransform} ${wrapperTransform}` : basePointerTransform,
        transformOrigin: cursorDecor.transformOrigin ?? '50% 50%',
        transitionProperty: 'left, top, transform',
      }
    : undefined;

  // Ghost cursor
  const ghostAbsolutePoint = currentCursorPlan ? toAbsolutePoint(currentCursorPlan.startNormalized, currentCursorPlan.dims) : undefined;
  const ghostStyle = ghostAbsolutePoint && currentCursorPlan ? toPointStyle(ghostAbsolutePoint, currentCursorPlan.dims) : undefined;
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

  // Overlay regions helper
  const overlayRegions = (regions: ReplayRegion[] | undefined, variant: 'highlight' | 'mask') => {
    if (!regions?.length) return null;
    return regions.map((region, index) => {
      const style = toRectStyle(region.boundingBox, dimensions);
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

  return (
    <div className="flex flex-col gap-4">
      <div
        ref={playerContainerRef}
        className={clsx(
          'relative overflow-hidden rounded-3xl',
          !isExportPresentation && 'transition-all duration-500',
          backgroundDecor.containerClass,
        )}
        style={backgroundDecor.containerStyle}
      >
        {backgroundDecor.baseLayer}
        {backgroundDecor.overlay}
        <div className={clsx('relative z-[1]', backgroundDecor.contentClass)}>
          {showInterfaceChrome && (
            <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-200/80">
              <span>Replay</span>
              <span>Step {currentIndex + 1} / {normalizedFrames.length}</span>
            </div>
          )}

          <div ref={captureAreaRef} className={clsx('space-y-3', { 'mt-4': showInterfaceChrome })}>
            <div className={clsx('overflow-hidden rounded-2xl', !isExportPresentation && 'transition-all duration-300', chromeDecor.frameClass)}>
              {chromeDecor.header}
              <div className={clsx('relative overflow-hidden', chromeDecor.contentClass)}>
                <div className="relative" style={{ paddingTop: `${aspectRatio}%` }}>
                  <div
                    ref={screenshotRef}
                    className="absolute inset-0"
                    style={{ transform: `scale(${zoom})`, transformOrigin: anchorStyle, transition: screenshotTransition }}
                  >
                    {currentFrame.screenshot?.url ? (
                      <img
                        src={currentFrame.screenshot.url}
                        alt={currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`}
                        loading="lazy"
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      <div className="flex h-full w-full items-center justify-center bg-slate-900 text-slate-500">
                        Screenshot unavailable
                      </div>
                    )}

                    {watermark && <WatermarkOverlay settings={watermark} />}

                    <div className="absolute inset-0">
                      {overlayRegions(currentFrame.maskRegions, 'mask')}
                      {overlayRegions(currentFrame.highlightRegions, 'highlight')}

                      {currentFrame.focusedElement?.boundingBox && (
                        <div
                          className="absolute rounded-2xl border border-sky-400/70 bg-sky-400/10 shadow-[0_0_60px_rgba(56,189,248,0.35)]"
                          style={toRectStyle(currentFrame.focusedElement.boundingBox, dimensions)}
                        />
                      )}

                      {hasTrail && (
                        <svg className="absolute inset-0 h-full w-full" viewBox="0 0 100 100" preserveAspectRatio="none">
                          <polyline
                            points={renderTrailPoints.map((p) => `${p.x},${p.y}`).join(' ')}
                            stroke={cursorDecor.trailColor}
                            strokeWidth={cursorTrailStrokeWidth}
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            fill="none"
                          />
                        </svg>
                      )}

                      {ghostWrapperStyle && cursorDecor.renderBase && (
                        <div
                          role="presentation"
                          className={clsx('absolute pointer-events-none select-none transition-all duration-500 ease-out', cursorDecor.wrapperClass)}
                          style={ghostWrapperStyle}
                        >
                          {cursorDecor.renderBase}
                        </div>
                      )}

                      {pointerWrapperStyle && cursorDecor.renderBase && (
                        <div role="presentation" className={pointerWrapperClassName} style={pointerWrapperStyle} {...pointerEventProps}>
                          {clickEffectElement}
                          {cursorDecor.renderBase}
                        </div>
                      )}
                    </div>

                    {playbackPhase === 'intro' && introCard && <IntroCard settings={introCard} />}
                    {playbackPhase === 'outro' && outroCard && <OutroCard settings={outroCard} />}
                  </div>
                </div>
              </div>
            </div>

            {showInterfaceChrome && (
              <ReplayControls
                isPlaying={isPlaying}
                onPlayPause={() => setIsPlaying((prev) => !prev)}
                onPrevious={handlePrevious}
                onNext={handleNext}
                frameProgress={frameProgress}
                displayDurationMs={displayDurationMs}
              />
            )}
          </div>
        </div>
      </div>

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
