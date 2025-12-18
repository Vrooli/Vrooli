import { useEffect, useMemo, useRef, useState, useCallback, type ReactNode, type CSSProperties, type HTMLAttributes } from 'react';
import clsx from 'clsx';
import { ChevronDown } from 'lucide-react';
import { WatermarkOverlay } from './replay/WatermarkOverlay';
import { IntroCard } from './replay/IntroCard';
import { OutroCard } from './replay/OutroCard';
import { ReplayControls, ReplayStoryboard, ReplayEmptyState } from './replay/components';

// Types - exported from centralized types file
import type {
  ReplayRegion,
  ReplayPoint,
  NormalizedPoint,
  CursorSpeedProfile,
  CursorPathStyle,
  CursorAnimationOverride,
  CursorOverrideMap,
  ReplayFrame,
  ReplayPlayerController,
  ReplayPlayerProps,
  Dimensions,
  CursorPlan,
} from './replay/types';

// Constants
import {
  DEFAULT_DURATION,
  MIN_CURSOR_SCALE,
  MAX_CURSOR_SCALE,
  DEFAULT_SPEED_PROFILE,
  DEFAULT_PATH_STYLE,
  SPEED_PROFILE_OPTIONS,
  CURSOR_PATH_STYLE_OPTIONS,
  FALLBACK_DIMENSIONS,
} from './replay/constants';

// Utilities
import {
  clamp01,
  clampNormalizedPoint,
  isSamePoint,
  applySpeedProfile,
  interpolatePath,
  generateStylizedPath,
  normalizeOverride,
} from './replay/utils/cursorMath';
import {
  toNormalizedPoint,
  toAbsolutePoint,
  toRectStyle,
  toPointStyle,
  pickZoomAnchor,
} from './replay/utils/geometry';
import {
  clampDuration,
  formatDurationSeconds,
  formatCoordinate,
  formatValue,
  parseRgbaComponents,
  rgbaWithAlpha,
} from './replay/utils/formatting';

// Theme builders
import { buildBackgroundDecor, buildCursorDecor, buildChromeDecor } from './replay/themes';

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
} from './replay/types';

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
  const normalizedFrames = useMemo(() => {
    return frames
      .filter((frame): frame is ReplayFrame => Boolean(frame))
      .map((frame, index) => ({
        ...frame,
        id: frame.id || `${index}`,
      }));
  }, [frames]);

  const isExternallyControlled = typeof exposeController === 'function';

  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(() => !isExternallyControlled && autoPlay && normalizedFrames.length > 1);
  const [frameProgress, setFrameProgress] = useState(0);
  const [isMetadataCollapsed, setIsMetadataCollapsed] = useState(true);

  // Intro/Outro card state
  type PlaybackPhase = 'intro' | 'frames' | 'outro';
  const hasIntroCard = Boolean(introCard?.enabled);
  const hasOutroCard = Boolean(outroCard?.enabled);
  const getInitialPhase = (): PlaybackPhase => {
    if (hasIntroCard) return 'intro';
    return 'frames';
  };
  const [playbackPhase, setPlaybackPhase] = useState<PlaybackPhase>(getInitialPhase);
  // Card progress tracked for potential future use (e.g., progress bar on intro/outro)
  const [_cardProgress, setCardProgress] = useState(0);
  void _cardProgress; // Suppress unused variable warning
  const rafRef = useRef<number | null>(null);
  const durationRef = useRef<number>(DEFAULT_DURATION);
  const [cursorOverrides, setCursorOverrides] = useState<CursorOverrideMap>({});
  const normalizedFramesRef = useRef(normalizedFrames);
  useEffect(() => {
    normalizedFramesRef.current = normalizedFrames;
  }, [normalizedFrames]);
  const randomSeedsRef = useRef<Record<string, NormalizedPoint>>({});
  const [cursorPosition, setCursorPosition] = useState<ReplayPoint | undefined>(undefined);
  const [activeClickEffect, setActiveClickEffect] = useState<{ frameId: string; key: number } | null>(null);
  const screenshotRef = useRef<HTMLDivElement | null>(null);
  const playerContainerRef = useRef<HTMLDivElement | null>(null);
  const captureAreaRef = useRef<HTMLDivElement | null>(null);
  const [dragState, setDragState] = useState<{ frameId: string; pointerId: number } | null>(null);
  const backgroundDecor = useMemo(
    () => buildBackgroundDecor(backgroundTheme ?? 'aurora'),
    [backgroundTheme],
  );
  const effectiveCursorTheme = cursorTheme ?? 'white';
  const isCursorEnabled = effectiveCursorTheme !== 'disabled';
  const isExportPresentation = presentationMode === 'export';
  const showInterfaceChrome = !isExportPresentation;
  const canEditCursor = allowPointerEditing && !isExportPresentation && isCursorEnabled;
  const cursorDecor = useMemo(() => buildCursorDecor(effectiveCursorTheme), [effectiveCursorTheme]);
  const pointerScale =
    typeof cursorScale === 'number' && !Number.isNaN(cursorScale)
      ? Math.min(MAX_CURSOR_SCALE, Math.max(MIN_CURSOR_SCALE, cursorScale))
      : 1;
  const cursorTrailStrokeWidth = cursorDecor.trailWidth * pointerScale;
  const baseSpeedProfile = cursorDefaultSpeedProfile ?? DEFAULT_SPEED_PROFILE;
  const basePathStyle = cursorDefaultPathStyle ?? DEFAULT_PATH_STYLE;

  const updateCursorOverride = (
    frameId: string,
    mutator: (previous: CursorAnimationOverride | undefined) => CursorAnimationOverride | null | undefined,
  ) => {
    setCursorOverrides((prev) => {
      const previous = prev[frameId];
      const next = normalizeOverride(mutator(previous), basePathStyle, baseSpeedProfile);
      if (!next) {
        if (previous === undefined) {
          return prev;
        }
        const { [frameId]: _removed, ...rest } = prev;
        return rest;
      }
      return { ...prev, [frameId]: next };
    });
  };

  const computeFallbackNormalized = (frameId: string, dims: Dimensions): NormalizedPoint => {
    const width = dims.width || FALLBACK_DIMENSIONS.width;
    const height = dims.height || FALLBACK_DIMENSIONS.height;
    const computePadRatio = (size: number) => {
      if (!size) {
        return 0.08;
      }
      return Math.min(0.48, Math.max(12 / size, 0.08));
    };
    const padX = computePadRatio(width);
    const padY = computePadRatio(height);

    switch (cursorInitialPosition) {
      case 'top-left':
        return clampNormalizedPoint({ x: padX, y: padY });
      case 'top-right':
        return clampNormalizedPoint({ x: 1 - padX, y: padY });
      case 'bottom-left':
        return clampNormalizedPoint({ x: padX, y: 1 - padY });
      case 'bottom-right':
        return clampNormalizedPoint({ x: 1 - padX, y: 1 - padY });
      case 'random': {
        const seed = randomSeedsRef.current[frameId] || { x: Math.random(), y: Math.random() };
        randomSeedsRef.current[frameId] = seed;
        const usableX = Math.max(0.02, 1 - padX * 2);
        const usableY = Math.max(0.02, 1 - padY * 2);
        return clampNormalizedPoint({ x: padX + seed.x * usableX, y: padY + seed.y * usableY });
      }
      case 'center':
      default:
        return { x: 0.5, y: 0.5 };
    }
  };

  useEffect(() => {
    if (isExternallyControlled) {
      setCurrentIndex(0);
      setFrameProgress(0);
      setCardProgress(0);
      setPlaybackPhase(hasIntroCard ? 'intro' : 'frames');
      setIsPlaying(false);
      return;
    }
    setCurrentIndex(0);
    setPlaybackPhase(hasIntroCard ? 'intro' : 'frames');
    setCardProgress(0);
    setIsPlaying(autoPlay && normalizedFrames.length > 1);
  }, [autoPlay, normalizedFrames.length, isExternallyControlled, hasIntroCard]);

  useEffect(() => {
    randomSeedsRef.current = {};
  }, [cursorInitialPosition]);

  useEffect(() => {
    if (!normalizedFrames[currentIndex]) {
      return;
    }
    onFrameChange?.(normalizedFrames[currentIndex], currentIndex);
  }, [currentIndex, normalizedFrames, onFrameChange]);

  useEffect(() => {
    onFrameProgressChange?.(currentIndex, frameProgress);
  }, [currentIndex, frameProgress, onFrameProgressChange]);

  useEffect(() => {
    if (!isPlaying || normalizedFrames.length <= 1 || isExternallyControlled) {
      setFrameProgress(0);
      setCardProgress(0);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
      return;
    }

    const playPhase = () => {
      let phaseDuration: number;

      if (playbackPhase === 'intro' && introCard) {
        phaseDuration = introCard.duration;
        setCardProgress(0);
      } else if (playbackPhase === 'outro' && outroCard) {
        phaseDuration = outroCard.duration;
        setCardProgress(0);
      } else {
        // Frames phase
        const frame = normalizedFrames[currentIndex];
        const frameDuration = frame?.totalDurationMs ?? frame?.durationMs;
        phaseDuration = clampDuration(frameDuration);
        setFrameProgress(0);
      }

      durationRef.current = phaseDuration;
      const start = performance.now();

      const step = (now: number) => {
        const elapsed = now - start;
        const progress = Math.min(1, elapsed / durationRef.current);

        if (playbackPhase === 'intro' || playbackPhase === 'outro') {
          setCardProgress(progress);
        } else {
          setFrameProgress(progress);
        }

        if (elapsed >= durationRef.current) {
          // Phase complete, determine next phase
          if (playbackPhase === 'intro') {
            setPlaybackPhase('frames');
            setCardProgress(0);
          } else if (playbackPhase === 'frames') {
            const atLastFrame = currentIndex >= normalizedFrames.length - 1;
            if (atLastFrame) {
              if (hasOutroCard) {
                setPlaybackPhase('outro');
              } else if (loop) {
                setCurrentIndex(0);
                if (hasIntroCard) {
                  setPlaybackPhase('intro');
                }
              } else {
                setIsPlaying(false);
              }
            } else {
              setCurrentIndex((prev) => Math.min(prev + 1, normalizedFrames.length - 1));
            }
          } else if (playbackPhase === 'outro') {
            if (loop) {
              setCurrentIndex(0);
              if (hasIntroCard) {
                setPlaybackPhase('intro');
              } else {
                setPlaybackPhase('frames');
              }
              setCardProgress(0);
            } else {
              setIsPlaying(false);
            }
          }
          return;
        }
        rafRef.current = requestAnimationFrame(step);
      };

      rafRef.current = requestAnimationFrame(step);
    };

    playPhase();

    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, [isPlaying, currentIndex, normalizedFrames, loop, isExternallyControlled, playbackPhase, hasIntroCard, hasOutroCard, introCard, outroCard]);

  useEffect(() => {
    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (!isCursorEnabled) {
      setCursorPosition(undefined);
    }
  }, [isCursorEnabled]);

  const cursorPlans = useMemo<CursorPlan[]>(() => {
    const plans: CursorPlan[] = [];
    let previousNormalized: NormalizedPoint | undefined;

    normalizedFrames.forEach((frame) => {
      const dims: Dimensions = {
        width: frame?.screenshot?.width || FALLBACK_DIMENSIONS.width,
        height: frame?.screenshot?.height || FALLBACK_DIMENSIONS.height,
      };

      const override = cursorOverrides[frame.id];

      const recordedTrail = Array.isArray(frame.cursorTrail)
        ? frame.cursorTrail.filter(
            (point): point is ReplayPoint => typeof point?.x === 'number' && typeof point?.y === 'number',
          )
        : [];

      const recordedTrailNormalized = recordedTrail
        .map((point) => toNormalizedPoint(point, dims))
        .filter((point): point is NormalizedPoint => Boolean(point))
        .map(clampNormalizedPoint);

      const recordedClickNormalized = toNormalizedPoint(frame.clickPosition ?? undefined, dims);

      const overrideTargetNormalized = override?.target ? clampNormalizedPoint(override.target) : undefined;
      const recordedTargetNormalized =
        recordedTrailNormalized.length > 0
          ? recordedTrailNormalized[recordedTrailNormalized.length - 1]
          : recordedClickNormalized ?? undefined;

      const fallbackNormalized = computeFallbackNormalized(frame.id, dims);
      const targetNormalized = overrideTargetNormalized ?? recordedTargetNormalized ?? fallbackNormalized;

      const recordedIntermediate =
        recordedTrailNormalized.length > 2
          ? recordedTrailNormalized.slice(1, recordedTrailNormalized.length - 1)
          : [];

      let startNormalized = previousNormalized;
      if (!startNormalized) {
        if (recordedTrailNormalized.length > 0) {
          startNormalized = recordedTrailNormalized[0];
        } else {
          startNormalized = fallbackNormalized;
        }
      }

      startNormalized = clampNormalizedPoint(startNormalized);

      const overridePathStyle = override?.pathStyle;
      const usingRecordedTrail = !overridePathStyle && recordedIntermediate.length > 0;
      const effectivePathStyle = overridePathStyle ?? basePathStyle;
      const generatedPath = usingRecordedTrail
        ? recordedIntermediate
        : generateStylizedPath(effectivePathStyle, startNormalized, targetNormalized, frame.id);

      plans.push({
        frameId: frame.id,
        dims,
        startNormalized,
        targetNormalized,
        pathNormalized: generatedPath,
        speedProfile: override?.speedProfile ?? baseSpeedProfile,
        pathStyle: effectivePathStyle,
        hasRecordedTrail: usingRecordedTrail,
        previousTargetNormalized: previousNormalized,
      });

      previousNormalized = targetNormalized;
    });

    return plans;
  }, [normalizedFrames, cursorOverrides, cursorInitialPosition, basePathStyle, baseSpeedProfile]);

  // Compute current frame safely (use index 0 as fallback for empty arrays to satisfy hooks)
  const currentFrame = normalizedFrames.length > 0 ? normalizedFrames[currentIndex] : null;

  const currentFrameId = currentFrame?.id;
  const currentStepType = currentFrame?.stepType;
  const currentClickPosition = currentFrame?.clickPosition;

  useEffect(() => {
    if (cursorClickAnimation === 'none' || !isCursorEnabled || !currentFrameId) {
      if (activeClickEffect !== null) {
        setActiveClickEffect(null);
      }
      return;
    }
    const indicatesClick = Boolean(
      (currentClickPosition &&
        typeof currentClickPosition.x === 'number' &&
        typeof currentClickPosition.y === 'number') ||
        (typeof currentStepType === 'string' && currentStepType.toLowerCase().includes('click')),
    );
    if (!indicatesClick) {
      return;
    }
    if (frameProgress === 0) {
      setActiveClickEffect({ frameId: currentFrameId, key: Date.now() });
    }
  }, [
    activeClickEffect,
    cursorClickAnimation,
    currentClickPosition?.x,
    currentClickPosition?.y,
    currentFrameId,
    currentStepType,
    frameProgress,
    isCursorEnabled,
  ]);

  useEffect(() => {
    if (!activeClickEffect) {
      return;
    }
    const timeout = window.setTimeout(() => {
      setActiveClickEffect((prev) => (prev && prev.key === activeClickEffect.key ? null : prev));
    }, cursorClickAnimation === 'ripple' ? 750 : 620);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [activeClickEffect, cursorClickAnimation]);

  const headerTitle = currentFrame
    ? currentFrame.finalUrl || currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`
    : 'Replay';
  const chromeVariant = chromeTheme ?? 'aurora';
  const chromeDecor = useMemo(
    () => buildChromeDecor(chromeVariant, headerTitle),
    [chromeVariant, headerTitle],
  );

  const seekToFrame = useCallback(
    (frameIndex: number, progress: number | undefined) => {
      const frames = normalizedFramesRef.current;
      if (frames.length === 0) {
        return;
      }
      const clampedIndex = Math.min(Math.max(frameIndex, 0), frames.length - 1);
      const clampedProgress = Number.isFinite(progress) ? Math.min(Math.max(progress ?? 0, 0), 1) : 0;
      setIsPlaying(false);
      setCurrentIndex(clampedIndex);
      setFrameProgress(clampedProgress);
    },
    [],
  );

  const changeFrame = useCallback(
    (index: number, progress?: number) => {
      seekToFrame(index, progress);
    },
    [seekToFrame],
  );

  const handleNext = useCallback(() => {
    const frames = normalizedFramesRef.current;
    if (frames.length === 0) {
      return;
    }
    if (currentIndex >= frames.length - 1) {
      if (loop) {
        changeFrame(0);
      }
      return;
    }
    changeFrame(currentIndex + 1);
  }, [changeFrame, currentIndex, loop]);

  const handlePrevious = useCallback(() => {
    const frames = normalizedFramesRef.current;
    if (frames.length === 0) {
      return;
    }
    if (currentIndex === 0) {
      if (loop) {
        changeFrame(frames.length - 1);
      }
      return;
    }
    changeFrame(currentIndex - 1);
  }, [changeFrame, currentIndex, loop]);

  const controller = useMemo<ReplayPlayerController | null>(() => {
    if (!exposeController) {
      return null;
    }
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
  }, [exposeController, isExportPresentation, seekToFrame]);

  useEffect(() => {
    if (!exposeController) {
      return;
    }
    exposeController(controller);
    return () => {
      exposeController(null);
    };
  }, [controller, exposeController]);

  // Hooks must run unconditionally - compute with safe fallbacks when no frames exist
  const extractedPreview = useMemo(() => {
    if (!currentFrame || currentFrame.extractedDataPreview == null) {
      return undefined;
    }
    if (typeof currentFrame.extractedDataPreview === 'string') {
      return currentFrame.extractedDataPreview;
    }
    try {
      return JSON.stringify(currentFrame.extractedDataPreview, null, 2);
    } catch (error) {
      return String(currentFrame.extractedDataPreview);
    }
  }, [currentFrame]);

  const domSnapshotDisplay = useMemo(() => {
    if (!currentFrame) {
      return undefined;
    }
    if (typeof currentFrame.domSnapshotPreview === 'string' && currentFrame.domSnapshotPreview.trim().length > 0) {
      return currentFrame.domSnapshotPreview.trim();
    }
    if (typeof currentFrame.domSnapshotHtml === 'string' && currentFrame.domSnapshotHtml.trim().length > 0) {
      const raw = currentFrame.domSnapshotHtml.trim();
      return raw.length > 1200 ? `${raw.slice(0, 1200)}...` : raw;
    }
    return undefined;
  }, [currentFrame]);

  const dimensions: Dimensions = {
    width:
      presentationDimensions?.width ||
      currentFrame?.screenshot?.width ||
      FALLBACK_DIMENSIONS.width,
    height:
      presentationDimensions?.height ||
      currentFrame?.screenshot?.height ||
      FALLBACK_DIMENSIONS.height,
  };

  useEffect(() => {
    if (!isCursorEnabled) {
      return;
    }
    const plan = cursorPlans[currentIndex];
    if (!plan) {
      setCursorPosition(undefined);
      return;
    }
    const normalizedPath = [plan.startNormalized, ...plan.pathNormalized, plan.targetNormalized];
    if (normalizedPath.length === 0) {
      setCursorPosition(undefined);
      return;
    }
    const absolutePath = normalizedPath.map((point) => toAbsolutePoint(point, plan.dims));
    const profiled = applySpeedProfile(isPlaying ? frameProgress : 1, plan.speedProfile);
    const evaluated = interpolatePath(absolutePath, profiled);
    setCursorPosition(evaluated);
  }, [cursorPlans, currentIndex, frameProgress, isCursorEnabled, isPlaying]);

  // Early return AFTER all hooks
  if (normalizedFrames.length === 0 || !currentFrame) {
    return <ReplayEmptyState backgroundDecor={backgroundDecor} />;
  }

  // All remaining code can safely use currentFrame (guaranteed non-null here)
  const screenshot = currentFrame.screenshot;
  const displayDurationMs = currentFrame.totalDurationMs ?? currentFrame.durationMs;
  const retryConfigured =
    typeof currentFrame.retryConfigured === 'number'
      ? currentFrame.retryConfigured > 0
      : currentFrame.retryConfigured === true;
  const hasResiliencyInfo = Boolean(
    (displayDurationMs && currentFrame.durationMs && Math.round(displayDurationMs) !== Math.round(currentFrame.durationMs)) ||
      (typeof currentFrame.retryAttempt === 'number' && currentFrame.retryAttempt > 1) ||
      (typeof currentFrame.retryMaxAttempts === 'number' && currentFrame.retryMaxAttempts > 1) ||
      retryConfigured ||
      (typeof currentFrame.retryDelayMs === 'number' && currentFrame.retryDelayMs > 0) ||
      (typeof currentFrame.retryBackoffFactor === 'number' && currentFrame.retryBackoffFactor !== 1 && currentFrame.retryBackoffFactor !== 0) ||
      (currentFrame.retryHistory && currentFrame.retryHistory.length > 0)
  );
  const aspectRatio = dimensions.width > 0 ? (dimensions.height / dimensions.width) * 100 : 56.25;

  const zoom = currentFrame.zoomFactor && currentFrame.zoomFactor > 1 ? Math.min(currentFrame.zoomFactor, 3) : 1;
  const zoomAnchor = pickZoomAnchor(currentFrame);
  const anchorStyle = (() => {
    if (!zoomAnchor) {
      return '50% 50%';
    }
    const style = toRectStyle(zoomAnchor, dimensions);
    if (!style) {
      return '50% 50%';
    }
    const left = parseFloat(style.left);
    const top = parseFloat(style.top);
    const width = parseFloat(style.width);
    const height = parseFloat(style.height);
    const cx = left + width / 2;
    const cy = top + height / 2;
    return `${cx}% ${cy}%`;
  })();

  const overlayRegions = (regions: ReplayRegion[] | undefined, variant: 'highlight' | 'mask') => {
    if (!regions || regions.length === 0) {
      return null;
    }
    return regions.map((region, index) => {
      const style = toRectStyle(region.boundingBox, dimensions);
      if (!style) {
        return null;
      }
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
            background:
              variant === 'highlight'
                ? 'rgba(56, 189, 248, 0.14)'
                : `rgba(15, 23, 42, ${opacity})`,
            boxShadow:
              variant === 'highlight'
                ? '0 16px 48px rgba(56, 189, 248, 0.35)'
                : undefined,
          }}
        />
      );
    });
  };

  const currentCursorPlan = cursorPlans[currentIndex];
  const cursorTrailPoints = isCursorEnabled && currentCursorPlan
    ? [
        currentCursorPlan.startNormalized,
        ...currentCursorPlan.pathNormalized,
        currentCursorPlan.targetNormalized,
      ].map((point) => ({
        x: clamp01(point.x) * 100,
        y: clamp01(point.y) * 100,
      }))
    : [];
  const hasMovement = Boolean(
    currentCursorPlan && !isSamePoint(currentCursorPlan.startNormalized, currentCursorPlan.targetNormalized),
  );
  const shouldRenderGhost = Boolean(isCursorEnabled && currentCursorPlan && !isPlaying && hasMovement);
  const renderTrailPoints = shouldRenderGhost ? cursorTrailPoints : [];
  const pointerStyle = isCursorEnabled && cursorPosition
    ? toPointStyle(cursorPosition, currentCursorPlan?.dims ?? dimensions)
    : undefined;
  const hasTrail = renderTrailPoints.length >= 2 && cursorTrailStrokeWidth > 0.05;
  const pointerOffsetX = cursorDecor.offset?.x ?? 0;
  const pointerOffsetY = cursorDecor.offset?.y ?? 0;
  const basePointerTransform = `translate(calc(-50% + ${pointerOffsetX}px), calc(-50% + ${pointerOffsetY}px))${
    pointerScale !== 1 ? ` scale(${pointerScale})` : ''
  }`;
  const wrapperTransform = (cursorDecor.wrapperStyle as (CSSProperties & { transform?: string }) | undefined)?.transform;
  const pointerWrapperStyle =
    pointerStyle && isCursorEnabled
      ? {
          ...pointerStyle,
          ...cursorDecor.wrapperStyle,
          transform: wrapperTransform ? `${basePointerTransform} ${wrapperTransform}` : basePointerTransform,
          transformOrigin: cursorDecor.transformOrigin ?? '50% 50%',
          transitionProperty: 'left, top, transform',
        }
      : undefined;

  const ghostAbsolutePoint = currentCursorPlan
    ? toAbsolutePoint(currentCursorPlan.startNormalized, currentCursorPlan.dims)
    : undefined;
  const ghostStyle = ghostAbsolutePoint && currentCursorPlan
    ? toPointStyle(ghostAbsolutePoint, currentCursorPlan.dims)
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

  const updateDragPosition = (clientX: number, clientY: number) => {
    if (!screenshotRef.current || !currentFrame || !canEditCursor) {
      return;
    }
    const rect = screenshotRef.current.getBoundingClientRect();
    if (!rect.width || !rect.height) {
      return;
    }
    const normalized = clampNormalizedPoint({
      x: (clientX - rect.left) / rect.width,
      y: (clientY - rect.top) / rect.height,
    });

    updateCursorOverride(currentFrame.id, (previous) => {
      const existing = previous ?? {};
      const next: CursorAnimationOverride = {
        ...existing,
        target: normalized,
      };
      if (existing.path === undefined) {
        next.path = [];
      }
      return next;
    });
  };

  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!canEditCursor || !currentFrame) {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    setDragState({ frameId: currentFrame.id, pointerId: event.pointerId });
    if (!event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.setPointerCapture(event.pointerId);
    }
    updateDragPosition(event.clientX, event.clientY);
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!canEditCursor || !dragState || dragState.pointerId !== event.pointerId || dragState.frameId !== currentFrame?.id) {
      return;
    }
    event.preventDefault();
    updateDragPosition(event.clientX, event.clientY);
  };

  const endDrag = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!canEditCursor || !dragState || dragState.pointerId !== event.pointerId) {
      return;
    }
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragState(null);
  };

  const handlePointerLeave = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!canEditCursor || !dragState?.pointerId) {
      return;
    }
    endDrag(event);
  };

  const pointerEventProps: HTMLAttributes<HTMLDivElement> = canEditCursor
    ? {
        onPointerDown: handlePointerDown,
        onPointerMove: handlePointerMove,
        onPointerUp: endDrag,
        onPointerCancel: endDrag,
        onPointerLeave: handlePointerLeave,
      }
    : {};

  const isDraggingCursor = dragState?.frameId === currentFrameId;

  const pointerWrapperClassName = clsx(
    'absolute transition-all duration-500 ease-out select-none relative',
    cursorDecor.wrapperClass,
    canEditCursor ? 'pointer-events-auto' : 'pointer-events-none',
    canEditCursor ? (isDraggingCursor ? 'cursor-grabbing' : 'cursor-grab') : 'cursor-default',
  );

  const currentOverride = currentCursorPlan ? cursorOverrides[currentCursorPlan.frameId] : undefined;
  const effectiveSpeedProfile = currentCursorPlan?.speedProfile ?? baseSpeedProfile;
  const currentTargetPoint = currentCursorPlan ? toAbsolutePoint(currentCursorPlan.targetNormalized, currentCursorPlan.dims) : undefined;
  const currentCursorNormalized = currentCursorPlan && cursorPosition
    ? toNormalizedPoint(cursorPosition, currentCursorPlan.dims)
    : undefined;
  const selectedPathStyle = currentOverride?.pathStyle ?? basePathStyle;
  const usingRecordedTrail = currentCursorPlan?.hasRecordedTrail ?? false;
  const pathStyleOption = CURSOR_PATH_STYLE_OPTIONS.find((option) => option.id === selectedPathStyle) ?? CURSOR_PATH_STYLE_OPTIONS[0];

  const isClickEffectActive = Boolean(
    cursorClickAnimation !== 'none' &&
      isCursorEnabled &&
      activeClickEffect &&
      currentFrameId &&
      activeClickEffect.frameId === currentFrameId,
  );

  let clickEffectElement: ReactNode = null;
  if (isClickEffectActive && activeClickEffect) {
    const rgbaComponents = parseRgbaComponents(cursorDecor.trailColor);
    const strokeColor =
      cursorClickAnimation === 'pulse'
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

  const handleResetSpeedProfile = () => {
    if (!currentCursorPlan) {
      return;
    }
    updateCursorOverride(currentCursorPlan.frameId, (previous) => {
      if (!previous) {
        return undefined;
      }
      const next = { ...previous };
      delete next.speedProfile;
      return next;
    });
  };

  const handleResetPathStyle = () => {
    if (!currentCursorPlan) {
      return;
    }
    updateCursorOverride(currentCursorPlan.frameId, (previous) => {
      if (!previous) {
        return undefined;
      }
      const next = { ...previous };
      delete next.pathStyle;
      return next;
    });
  };

  const handleResetAllCustomizations = () => {
    if (!currentCursorPlan) {
      return;
    }
    updateCursorOverride(currentCursorPlan.frameId, () => undefined);
  };

  const screenshotTransition = isExportPresentation
    ? 'none'
    : 'transform 600ms ease, transform-origin 600ms ease';

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
              <span>
                Step {currentIndex + 1} / {normalizedFrames.length}
              </span>
            </div>
          )}

          <div
            ref={captureAreaRef}
            className={clsx('space-y-3', { 'mt-4': showInterfaceChrome })}
          >
            <div
              className={clsx(
                'overflow-hidden rounded-2xl',
                !isExportPresentation && 'transition-all duration-300',
                chromeDecor.frameClass,
              )}
            >
            {chromeDecor.header}

            <div className={clsx('relative overflow-hidden', chromeDecor.contentClass)}>
              <div
                className="relative"
                style={{
                  paddingTop: `${aspectRatio}%`,
                }}
              >
                <div
                  ref={screenshotRef}
                  className="absolute inset-0"
                  style={{
                    transform: `scale(${zoom})`,
                    transformOrigin: anchorStyle,
                    transition: screenshotTransition,
                  }}
                >
                  {screenshot?.url ? (
                    <img
                      src={screenshot.url}
                      alt={currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`}
                      loading="lazy"
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center bg-slate-900 text-slate-500">
                      Screenshot unavailable
                    </div>
                  )}

                  {/* Watermark overlay */}
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
                      <svg
                        className="absolute inset-0 h-full w-full"
                        viewBox="0 0 100 100"
                        preserveAspectRatio="none"
                      >
                        <polyline
                          points={renderTrailPoints.map((point) => `${point.x},${point.y}`).join(' ')}
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
                        className={clsx(
                          'absolute pointer-events-none select-none transition-all duration-500 ease-out',
                          cursorDecor.wrapperClass,
                        )}
                        style={ghostWrapperStyle}
                      >
                        {cursorDecor.renderBase}
                      </div>
                    )}

                    {pointerWrapperStyle && cursorDecor.renderBase && (
                      <div
                        role="presentation"
                        className={pointerWrapperClassName}
                        style={pointerWrapperStyle}
                        {...pointerEventProps}
                      >
                        {clickEffectElement}
                        {cursorDecor.renderBase}
                      </div>
                    )}
                  </div>

                  {/* Intro/Outro card overlays */}
                  {playbackPhase === 'intro' && introCard && (
                    <IntroCard settings={introCard} />
                  )}
                  {playbackPhase === 'outro' && outroCard && (
                    <OutroCard settings={outroCard} />
                  )}
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
        <div className="rounded-3xl border border-white/10 bg-slate-950/70 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400">
          <div className="flex flex-wrap items-center gap-3">
            <span className="uppercase tracking-[0.18em] text-slate-500">Frame Metadata</span>
            {currentFrame.stepType && (
              <span className="inline-flex items-center rounded-full border border-blue-500/40 bg-blue-500/10 px-3 py-1 text-[11px] uppercase tracking-[0.2em] text-blue-200">
                {currentFrame.stepType}
              </span>
            )}
            {currentFrame.status && (
              <span
                className={clsx('inline-flex items-center rounded-full px-3 py-1 text-[11px] uppercase tracking-[0.2em]', {
                  'border border-emerald-400/40 bg-emerald-500/10 text-emerald-200': currentFrame.status === 'completed',
                  'border border-amber-400/40 bg-amber-500/10 text-amber-200': currentFrame.status === 'running',
                  'border border-rose-400/40 bg-rose-500/10 text-rose-200': currentFrame.status === 'failed',
                })}
              >
                {currentFrame.status}
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={() => setIsMetadataCollapsed((prev) => !prev)}
            className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition-colors hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950"
            aria-expanded={!isMetadataCollapsed}
            aria-label={isMetadataCollapsed ? 'Expand frame metadata' : 'Collapse frame metadata'}
          >
            <ChevronDown
              size={14}
              className={clsx('transition-transform duration-200', {
                '-rotate-180': !isMetadataCollapsed,
              })}
            />
          </button>
        </div>

        <div
          className={clsx(
            'grid gap-4 transition-all duration-300 md:grid-cols-2',
            isMetadataCollapsed
              ? 'pointer-events-none max-h-0 overflow-hidden opacity-0'
              : 'mt-3 max-h-[1200px] opacity-100',
          )}
        >
          <div className="space-y-2 text-sm text-slate-300">
            <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-500">
              <span>Node</span>
              <span>#{currentFrame.nodeId || currentFrame.stepIndex + 1}</span>
            </div>
            {currentFrame.finalUrl && (
              <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
                {currentFrame.finalUrl}
              </div>
            )}
            {currentFrame.error && (
              <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-100">
                {currentFrame.error}
              </div>
            )}
            {currentFrame.assertion && (
              <div
                className={clsx(
                  'rounded-xl border p-3 text-xs transition-colors',
                  currentFrame.assertion.success !== false
                    ? 'border-emerald-400/40 bg-emerald-500/5 text-emerald-100'
                    : 'border-rose-400/40 bg-rose-500/10 text-rose-100',
                )}
              >
                <div className="mb-1 flex items-center justify-between uppercase tracking-[0.2em]">
                  <span>Assertion</span>
                  <span>
                    {currentFrame.assertion.success === false ? 'Failed' : 'Passed'}
                  </span>
                </div>
                <div className="space-y-1 text-[11px] text-slate-200">
                  {currentFrame.assertion.selector && (
                    <div>Selector: {currentFrame.assertion.selector}</div>
                  )}
                  {currentFrame.assertion.mode && (
                    <div>Mode: {currentFrame.assertion.mode}</div>
                  )}
                  {currentFrame.assertion.expected !== undefined && currentFrame.assertion.expected !== null && (
                    <div>
                      Expected: {formatValue(currentFrame.assertion.expected)}
                    </div>
                  )}
                  {currentFrame.assertion.actual !== undefined && currentFrame.assertion.actual !== null && (
                    <div>
                      Actual: {formatValue(currentFrame.assertion.actual)}
                    </div>
                  )}
                  {currentFrame.assertion.message && (
                    <div className="italic text-slate-300">
                      {currentFrame.assertion.message}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
          <div className="space-y-2 text-sm text-slate-300">
            {extractedPreview && (
              <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/5 p-3 text-xs text-emerald-100">
                <div className="mb-1 uppercase tracking-[0.2em] text-emerald-300">Extracted Data</div>
                <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words text-[11px] text-emerald-100/90">
                  {extractedPreview}
                </pre>
              </div>
            )}
            {currentCursorPlan && (
              <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
                <div className="flex flex-wrap items-center justify-between gap-2 uppercase tracking-[0.2em] text-slate-400">
                  <span>Cursor Behavior</span>
                  {currentOverride && (
                    <button
                      type="button"
                      className="rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
                      onClick={handleResetAllCustomizations}
                    >
                      Reset
                    </button>
                  )}
                </div>
                <div className="mt-2 space-y-1 text-[11px] text-slate-300">
                  <div className="flex items-center justify-between">
                    <span>Current position</span>
                    <span className="font-mono text-slate-200">
                      {formatCoordinate(cursorPosition, currentCursorPlan.dims)}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span>Target placement</span>
                    <span className="font-mono text-slate-200">
                      {formatCoordinate(currentTargetPoint, currentCursorPlan.dims)}
                    </span>
                  </div>
                  {currentCursorNormalized && (
                    <div className="flex items-center justify-between">
                      <span>Live cursor</span>
                      <span className="font-mono text-slate-200">
                        {`${Math.round(currentCursorNormalized.x * 100)}%, ${Math.round(currentCursorNormalized.y * 100)}%`}
                      </span>
                    </div>
                  )}
                  <div className="flex items-center justify-between">
                    <span>Mode</span>
                    <span className="text-slate-400">
                      {usingRecordedTrail ? 'Recorded capture' : pathStyleOption.label}
                    </span>
                  </div>
                </div>
                <div className="mt-3 space-y-1">
                  <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.2em] text-slate-400">
                    <label htmlFor="cursor-speed-profile" className="cursor-pointer">
                      Speed profile
                    </label>
                    {currentOverride?.speedProfile && currentOverride.speedProfile !== baseSpeedProfile && (
                      <button
                        type="button"
                        className="rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
                        onClick={handleResetSpeedProfile}
                      >
                        Default
                      </button>
                    )}
                  </div>
                  <select
                    id="cursor-speed-profile"
                    className="w-full rounded-lg border border-slate-700/60 bg-slate-900/70 px-2.5 py-1.5 text-[11px] text-slate-200 focus:border-flow-accent/60 focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                    value={effectiveSpeedProfile}
                    onChange={(event) => {
                      const selected = event.target.value as CursorSpeedProfile;
                      updateCursorOverride(currentCursorPlan.frameId, (previous) => ({
                        ...(previous ?? {}),
                        speedProfile: selected,
                      }));
                    }}
                  >
                    {SPEED_PROFILE_OPTIONS.map((option) => (
                      <option key={option.id} value={option.id}>
                        {option.label} — {option.description}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="mt-3 space-y-1">
                  <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.2em] text-slate-400">
                    <label htmlFor="cursor-path-style" className="cursor-pointer">
                      Path shape
                    </label>
                    {currentOverride?.pathStyle && currentOverride.pathStyle !== basePathStyle && (
                      <button
                        type="button"
                        className="inline-flex items-center gap-1 rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
                        onClick={handleResetPathStyle}
                      >
                        Default
                      </button>
                    )}
                  </div>
                  <select
                    id="cursor-path-style"
                    className="w-full rounded-lg border border-slate-700/60 bg-slate-900/70 px-2.5 py-1.5 text-[11px] text-slate-200 focus:border-flow-accent/60 focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
                    value={selectedPathStyle}
                    onChange={(event) => {
                      const selected = event.target.value as CursorPathStyle;
                      updateCursorOverride(currentCursorPlan.frameId, (previous) => ({
                        ...(previous ?? {}),
                        pathStyle: selected,
                      }));
                    }}
                  >
                    {CURSOR_PATH_STYLE_OPTIONS.map((option) => (
                      <option key={option.id} value={option.id}>
                        {option.label} — {option.description}
                      </option>
                    ))}
                  </select>
                  <p className="text-[10px] text-slate-500">
                    {usingRecordedTrail
                      ? 'Using captured cursor trail from the execution. Choose a shape to override it.'
                      : pathStyleOption.description}
                  </p>
                </div>
                <p className="mt-3 text-[10px] text-slate-500">
                  Tip: drag the cursor directly on the replay canvas to fine-tune the target placement.
                </p>
              </div>
            )}
            {currentFrame.consoleLogCount || currentFrame.networkEventCount ? (
              <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
                <div className="uppercase tracking-[0.2em] text-slate-400">Telemetry Snapshot</div>
                <div className="mt-2 flex flex-wrap gap-3 text-[11px] text-slate-300">
                  {typeof currentFrame.consoleLogCount === 'number' && (
                    <span>{currentFrame.consoleLogCount} console events</span>
                  )}
                  {typeof currentFrame.networkEventCount === 'number' && (
                    <span>{currentFrame.networkEventCount} network events</span>
                  )}
                </div>
              </div>
            ) : null}
            {domSnapshotDisplay && (
              <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
                <div className="uppercase tracking-[0.2em] text-slate-400">DOM Snapshot</div>
                <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-[11px] text-slate-300">
                  {domSnapshotDisplay}
                </pre>
              </div>
            )}
            {hasResiliencyInfo ? (
              <div className="rounded-xl border border-amber-400/30 bg-amber-500/5 p-3 text-xs text-amber-100">
                <div className="uppercase tracking-[0.2em] text-amber-300">Resiliency</div>
                <div className="mt-2 space-y-1 text-[11px] text-amber-100/90">
                  {displayDurationMs ? (
                    <div>Total duration: {formatDurationSeconds(displayDurationMs)}</div>
                  ) : null}
                  {currentFrame.durationMs && displayDurationMs && Math.round(displayDurationMs) !== Math.round(currentFrame.durationMs) && (
                    <div>Final attempt: {formatDurationSeconds(currentFrame.durationMs)}</div>
                  )}
                  {typeof currentFrame.retryAttempt === 'number' && currentFrame.retryAttempt > 0 && (
                    <div>
                      Attempt: {currentFrame.retryAttempt}
                      {typeof currentFrame.retryMaxAttempts === 'number' && currentFrame.retryMaxAttempts > 0
                        ? ` / ${currentFrame.retryMaxAttempts}`
                        : ''}
                    </div>
                  )}
                  {retryConfigured ? (
                    <div>
                      Configured retries
                      {typeof currentFrame.retryConfigured === 'number' ? `: ${currentFrame.retryConfigured}` : ''}
                    </div>
                  ) : null}
                  {typeof currentFrame.retryDelayMs === 'number' && currentFrame.retryDelayMs > 0 && (
                    <div>Initial delay: {formatDurationSeconds(currentFrame.retryDelayMs)}</div>
                  )}
                  {typeof currentFrame.retryBackoffFactor === 'number' && currentFrame.retryBackoffFactor !== 1 && currentFrame.retryBackoffFactor !== 0 && (
                    <div>Backoff factor: ×{currentFrame.retryBackoffFactor.toFixed(2)}</div>
                  )}
                  {currentFrame.retryHistory && currentFrame.retryHistory.length > 0 && (
                    <div className="mt-2 space-y-1">
                      {currentFrame.retryHistory.map((entry, index) => (
                        <div
                          key={`${entry.attempt ?? index}`}
                          className={clsx('flex items-center justify-between rounded-lg border px-2 py-1', {
                            'border-emerald-400/20 bg-emerald-400/10 text-emerald-100': entry.success,
                            'border-rose-400/30 bg-rose-500/10 text-rose-100': entry.success === false,
                            'border-white/10 bg-white/5 text-slate-200': entry.success === undefined,
                          })}
                        >
                          <span>
                            Attempt {entry.attempt ?? index + 1}
                            {typeof entry.durationMs === 'number'
                              ? ` • ${formatDurationSeconds(entry.durationMs)}`
                              : ''}
                            {typeof entry.callDurationMs === 'number' && entry.callDurationMs !== entry.durationMs
                              ? ` (call ${formatDurationSeconds(entry.callDurationMs)})`
                              : ''}
                          </span>
                          <span className="ml-3 text-right text-[10px] uppercase tracking-[0.2em]">
                            {entry.success === false ? 'Failed' : entry.success ? 'Passed' : 'Result'}
                          </span>
                          {entry.error && (
                            <span className="ml-3 truncate text-[10px] text-amber-200/90" title={entry.error}>
                              {entry.error}
                            </span>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ) : null}
          </div>
        </div>
        </div>
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
