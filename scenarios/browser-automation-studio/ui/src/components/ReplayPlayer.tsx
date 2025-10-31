import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import clsx from 'clsx';
import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Pause, Play } from 'lucide-react';

export interface ReplayBoundingBox {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface ReplayRegion {
  selector?: string;
  boundingBox?: ReplayBoundingBox;
  padding?: number;
  color?: string;
  opacity?: number;
}

export interface ReplayPoint {
  x?: number;
  y?: number;
}

export interface ReplayScreenshot {
  artifactId: string;
  url?: string;
  thumbnailUrl?: string;
  width?: number;
  height?: number;
  contentType?: string;
  sizeBytes?: number;
}

export interface ReplayRetryHistoryEntry {
  attempt?: number;
  success?: boolean;
  durationMs?: number;
  callDurationMs?: number;
  error?: string;
}

export interface ReplayFrame {
  id: string;
  stepIndex: number;
  nodeId?: string;
  stepType?: string;
  status?: string;
  success: boolean;
  durationMs?: number;
  totalDurationMs?: number;
  progress?: number;
  finalUrl?: string;
  error?: string;
  extractedDataPreview?: unknown;
  consoleLogCount?: number;
  networkEventCount?: number;
  screenshot?: ReplayScreenshot | null;
  highlightRegions?: ReplayRegion[];
  maskRegions?: ReplayRegion[];
  focusedElement?: {
    selector?: string;
    boundingBox?: ReplayBoundingBox;
  } | null;
  elementBoundingBox?: ReplayBoundingBox | null;
  clickPosition?: ReplayPoint | null;
  cursorTrail?: ReplayPoint[];
  zoomFactor?: number;
  assertion?: {
    mode?: string;
    selector?: string;
    expected?: unknown;
    actual?: unknown;
    success?: boolean;
    message?: string;
    negated?: boolean;
    caseSensitive?: boolean;
  };
  retryAttempt?: number;
  retryMaxAttempts?: number;
  retryConfigured?: number;
  retryDelayMs?: number;
  retryBackoffFactor?: number;
  retryHistory?: ReplayRetryHistoryEntry[];
  domSnapshotHtml?: string;
  domSnapshotPreview?: string;
  domSnapshotArtifactId?: string;
}

export type ReplayChromeTheme = 'aurora' | 'chromium' | 'midnight' | 'minimal';
export type ReplayBackgroundTheme =
  | 'aurora'
  | 'sunset'
  | 'ocean'
  | 'nebula'
  | 'grid'
  | 'charcoal'
  | 'steel'
  | 'emerald'
  | 'none';

interface ReplayPlayerProps {
  frames: ReplayFrame[];
  autoPlay?: boolean;
  loop?: boolean;
  onFrameChange?: (frame: ReplayFrame, index: number) => void;
  executionStatus?: 'pending' | 'running' | 'completed' | 'failed';
  chromeTheme?: ReplayChromeTheme;
  backgroundTheme?: ReplayBackgroundTheme;
}

const DEFAULT_DURATION = 1600;
const MIN_DURATION = 800;
const MAX_DURATION = 6000;

const clampDuration = (value?: number) => {
  if (!value || Number.isNaN(value)) {
    return DEFAULT_DURATION;
  }
  return Math.min(MAX_DURATION, Math.max(MIN_DURATION, value));
};

const formatDurationSeconds = (value?: number | null) => {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '0.0s';
  }
  const seconds = Math.max(0, value) / 1000;
  return `${seconds.toFixed(1)}s`;
};

type Dimensions = { width: number; height: number };

const FALLBACK_DIMENSIONS: Dimensions = { width: 1920, height: 1080 };

const formatValue = (value: unknown) => {
  if (value == null) {
    return 'null';
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value);
    } catch (error) {
      return String(value);
    }
  }
  return String(value);
};

const toRectStyle = (box: ReplayBoundingBox | null | undefined, dims: Dimensions) => {
  if (!box || !dims.width || !dims.height) {
    return undefined;
  }
  const clamp = (value: number | undefined, max: number) => {
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return 0;
    }
    return Math.max(0, Math.min(value, max));
  };

  const width = clamp(box.width, dims.width);
  const height = clamp(box.height, dims.height);
  const left = clamp(box.x, dims.width - width);
  const top = clamp(box.y, dims.height - height);

  return {
    left: `${(left / dims.width) * 100}%`,
    top: `${(top / dims.height) * 100}%`,
    width: `${(width / dims.width) * 100}%`,
    height: `${(height / dims.height) * 100}%`,
  } as const;
};

const toPointStyle = (point: ReplayPoint | null | undefined, dims: Dimensions) => {
  if (!point || !dims.width || !dims.height) {
    return undefined;
  }
  if (typeof point.x !== 'number' || typeof point.y !== 'number') {
    return undefined;
  }
  return {
    left: `${(point.x / dims.width) * 100}%`,
    top: `${(point.y / dims.height) * 100}%`,
  } as const;
};

const toTrailPoints = (trail: ReplayPoint[] | undefined, dims: Dimensions) => {
  if (!trail || trail.length === 0 || !dims.width || !dims.height) {
    return [] as Array<{ x: number; y: number }>;
  }

  return trail
    .filter((point): point is Required<ReplayPoint> => typeof point?.x === 'number' && typeof point?.y === 'number')
    .map((point) => ({
      x: (point.x / dims.width) * 100,
      y: (point.y / dims.height) * 100,
    }));
};

const pickZoomAnchor = (frame: ReplayFrame): ReplayBoundingBox | undefined => {
  if (frame.focusedElement?.boundingBox) {
    return frame.focusedElement.boundingBox;
  }
  if (frame.highlightRegions && frame.highlightRegions.length > 0) {
    const candidate = frame.highlightRegions.find((region) => region.boundingBox);
    if (candidate?.boundingBox) {
      return candidate.boundingBox;
    }
  }
  return undefined;
};

type BackgroundDecor = {
  containerClass: string;
  contentClass: string;
  overlay?: ReactNode;
};

const buildBackgroundDecor = (theme: ReplayBackgroundTheme): BackgroundDecor => {
  switch (theme) {
    case 'sunset':
      return {
        containerClass:
          'border border-rose-200/40 bg-gradient-to-br from-fuchsia-700/95 via-rose-600/85 to-amber-400/80 shadow-[0_26px_70px_rgba(236,72,153,0.38)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.5px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-55 mix-blend-screen"
            style={{
              background:
                'radial-gradient(140% 120% at 0% 0%, rgba(254,226,226,0.28), transparent 65%), radial-gradient(120% 120% at 100% 100%, rgba(254,249,195,0.28), transparent 60%)',
            }}
          />
        ),
      };
    case 'ocean':
      return {
        containerClass:
          'border border-sky-300/25 bg-gradient-to-br from-sky-900 via-slate-950 to-slate-900 shadow-[0_24px_65px_rgba(14,165,233,0.38)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45"
            style={{
              background:
                'radial-gradient(130% 150% at 10% 10%, rgba(125,211,252,0.25), transparent 60%), radial-gradient(140% 160% at 90% 90%, rgba(14,116,144,0.28), transparent 65%)',
            }}
          />
        ),
      };
    case 'nebula':
      return {
        containerClass:
          'border border-purple-300/30 bg-gradient-to-br from-violet-800 via-indigo-950 to-slate-950 shadow-[0_26px_70px_rgba(124,58,237,0.36)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.5px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-55 mix-blend-screen"
            style={{
              background:
                'radial-gradient(140% 140% at 0% 0%, rgba(196,181,253,0.32), transparent 65%), radial-gradient(120% 130% at 100% 90%, rgba(167,139,250,0.28), transparent 60%)',
            }}
          />
        ),
      };
    case 'grid':
      return {
        containerClass:
          'border border-cyan-300/25 bg-slate-950 shadow-[0_24px_60px_rgba(8,47,73,0.5)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.5px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-60"
            style={{
              backgroundImage:
                'linear-gradient(rgba(148,163,184,0.18) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.14) 1px, transparent 1px)',
              backgroundSize: '32px 32px',
              backgroundPosition: 'center',
            }}
          />
        ),
      };
    case 'charcoal':
      return {
        containerClass: 'border border-slate-700/60 bg-slate-950 shadow-[0_18px_55px_rgba(15,23,42,0.55)]',
        contentClass: 'p-6 sm:p-7',
      };
    case 'steel':
      return {
        containerClass: 'border border-slate-600/60 bg-slate-800 shadow-[0_18px_52px_rgba(30,41,59,0.52)]',
        contentClass: 'p-6 sm:p-7',
      };
    case 'emerald':
      return {
        containerClass:
          'border border-emerald-300/30 bg-gradient-to-br from-emerald-900 via-emerald-950 to-slate-950 shadow-[0_22px_60px_rgba(16,185,129,0.35)]',
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1px]',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45"
            style={{
              background:
                'radial-gradient(140% 140% at 10% 15%, rgba(167,243,208,0.28), transparent 65%), radial-gradient(120% 130% at 90% 90%, rgba(45,197,253,0.22), transparent 60%)',
            }}
          />
        ),
      };
    case 'none':
      return {
        containerClass: 'border border-transparent bg-transparent shadow-none',
        contentClass: 'p-0',
      };
    case 'aurora':
    default:
      return {
        containerClass:
          'border border-white/10 bg-gradient-to-br from-slate-900 via-slate-950 to-black shadow-[0_20px_60px_rgba(15,23,42,0.4)]',
        contentClass: 'p-6 sm:p-7',
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-55"
            style={{
              background:
                'radial-gradient(130% 140% at 10% 10%, rgba(56,189,248,0.25), transparent 60%), radial-gradient(120% 140% at 90% 90%, rgba(129,140,248,0.23), transparent 60%)',
            }}
          />
        ),
      };
  }
};

type ChromeDecor = {
  frameClass: string;
  contentClass?: string;
  header: ReactNode | null;
};

const buildChromeDecor = (theme: ReplayChromeTheme, title: string): ChromeDecor => {
  switch (theme) {
    case 'chromium':
      return {
        frameClass: 'border border-[#d0d4dc] bg-[#e9ecef] shadow-[0_24px_70px_rgba(15,23,42,0.4)]',
        contentClass: 'bg-white',
        header: (
          <div className="bg-[#dee1e6] border-b border-[#c4c8ce]">
            <div className="flex items-end gap-2 px-4 pt-3">
              <div className="h-7 w-32 rounded-t-lg border border-[#c4c8ce] bg-white px-3 text-[11px] font-medium text-slate-700 shadow-sm">
                New Tab
              </div>
              <div className="ml-auto flex gap-2 pb-1 text-slate-500/70">
                <span className="h-2 w-2 rounded-full bg-slate-500/50" />
                <span className="h-2 w-2 rounded-full bg-slate-500/50" />
                <span className="h-2 w-2 rounded-full bg-slate-500/50" />
              </div>
            </div>
            <div className="flex items-center gap-3 px-4 pb-3">
              <div className="flex items-center gap-1 text-slate-600">
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[#c4c8ce] bg-white text-lg leading-none">‹</span>
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[#c4c8ce] bg-white text-lg leading-none">›</span>
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[#c4c8ce] bg-white text-base leading-none">↻</span>
              </div>
              <div className="flex-1 ml-2 rounded-full border border-[#c4c8ce] bg-white px-4 py-1 text-[11px] text-slate-600 shadow-inner">
                {title}
              </div>
              <div className="flex items-center gap-3 text-slate-600">
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[#c4c8ce] bg-white text-base leading-none">☆</span>
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-[#c4c8ce] bg-white text-sm leading-none font-medium">•••</span>
                <span className="inline-flex h-7 w-7 items-center justify-center rounded-full bg-gradient-to-br from-slate-300 to-slate-100 text-[10px] font-semibold text-slate-700">J</span>
              </div>
            </div>
          </div>
        ),
      };
    case 'midnight':
      return {
        frameClass: 'border border-indigo-500/15 bg-gradient-to-br from-indigo-950/85 via-slate-950/80 to-black/75 shadow-[0_32px_90px_rgba(79,70,229,0.35)]',
        contentClass: 'bg-slate-950/25',
        header: (
          <div className="flex items-center justify-between border-b border-indigo-400/15 bg-gradient-to-r from-indigo-950/50 via-slate-950/35 to-indigo-900/35 px-5 py-3 text-[11px] uppercase tracking-[0.22em] text-indigo-200/80">
            <div className="flex items-center gap-3 text-indigo-300/70">
              <span className="h-2 w-8 rounded-full bg-indigo-400/70" />
              <span className="h-2 w-4 rounded-full bg-indigo-400/40" />
            </div>
            <div className="max-w-sm truncate text-indigo-100/90">{title}</div>
            <div className="flex items-center gap-2 text-indigo-300/60">
              <span className="h-2 w-2 rounded-full bg-indigo-400/70" />
              <span className="h-2 w-2 rounded-full bg-indigo-400/70" />
            </div>
          </div>
        ),
      };
    case 'minimal':
      return {
        frameClass: 'border border-transparent bg-transparent shadow-none',
        contentClass: 'bg-transparent',
        header: null,
      };
    case 'aurora':
    default:
      return {
        frameClass: 'border border-white/10 bg-slate-950/60 backdrop-blur-sm shadow-[0_20px_60px_rgba(15,23,42,0.4)]',
        contentClass: 'bg-slate-950/35',
        header: (
          <div className="flex items-center gap-3 border-b border-white/5 bg-slate-950/40 px-5 py-3 text-xs text-slate-300">
            <div className="flex items-center gap-2">
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-rose-500" />
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-amber-400" />
              <span className="inline-flex h-2.5 w-2.5 rounded-full bg-emerald-400" />
            </div>
            <div className="truncate text-slate-200">{title}</div>
          </div>
        ),
      };
  }
};

export function ReplayPlayer({
  frames,
  autoPlay = true,
  loop = true,
  onFrameChange,
  executionStatus,
  chromeTheme = 'aurora',
  backgroundTheme = 'aurora',
}: ReplayPlayerProps) {
  const normalizedFrames = useMemo(() => {
    return frames
      .filter((frame): frame is ReplayFrame => Boolean(frame))
      .map((frame, index) => ({
        ...frame,
        id: frame.id || `${index}`,
      }));
  }, [frames]);

  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(autoPlay && normalizedFrames.length > 1);
  const [frameProgress, setFrameProgress] = useState(0);
  const [isMetadataCollapsed, setIsMetadataCollapsed] = useState(true);
  const rafRef = useRef<number | null>(null);
  const durationRef = useRef<number>(DEFAULT_DURATION);
  const backgroundDecor = useMemo(
    () => buildBackgroundDecor(backgroundTheme ?? 'aurora'),
    [backgroundTheme],
  );

  useEffect(() => {
    setCurrentIndex(0);
    setIsPlaying(autoPlay && normalizedFrames.length > 1);
  }, [autoPlay, normalizedFrames.length]);

  useEffect(() => {
    if (!normalizedFrames[currentIndex]) {
      return;
    }
    onFrameChange?.(normalizedFrames[currentIndex], currentIndex);
  }, [currentIndex, normalizedFrames, onFrameChange]);

  useEffect(() => {
    if (!isPlaying || normalizedFrames.length <= 1) {
      setFrameProgress(0);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
      return;
    }

    const playFrame = () => {
      const frame = normalizedFrames[currentIndex];
      const frameDuration = frame?.totalDurationMs ?? frame?.durationMs;
      durationRef.current = clampDuration(frameDuration);
      setFrameProgress(0);
      const start = performance.now();

      const step = (now: number) => {
        const elapsed = now - start;
        const progress = Math.min(1, elapsed / durationRef.current);
        setFrameProgress(progress);
        if (elapsed >= durationRef.current) {
          const atLastFrame = currentIndex >= normalizedFrames.length - 1;
          if (atLastFrame) {
            if (loop) {
              setCurrentIndex(0);
            } else {
              setIsPlaying(false);
            }
          } else {
            setCurrentIndex((prev) => Math.min(prev + 1, normalizedFrames.length - 1));
          }
          return;
        }
        rafRef.current = requestAnimationFrame(step);
      };

      rafRef.current = requestAnimationFrame(step);
    };

    playFrame();

    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, [isPlaying, currentIndex, normalizedFrames, loop]);

  useEffect(() => {
    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  // Compute current frame safely (use index 0 as fallback for empty arrays to satisfy hooks)
  const currentFrame = normalizedFrames.length > 0 ? normalizedFrames[currentIndex] : null;

  const headerTitle = currentFrame
    ? currentFrame.finalUrl || currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`
    : 'Replay';
  const chromeVariant = chromeTheme ?? 'aurora';
  const chromeDecor = useMemo(
    () => buildChromeDecor(chromeVariant, headerTitle),
    [chromeVariant, headerTitle],
  );

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

  // Early return AFTER all hooks
  if (normalizedFrames.length === 0 || !currentFrame) {
    return (
      <div
        className={clsx(
          'relative flex h-64 items-center justify-center overflow-hidden rounded-3xl text-sm text-slate-200/80 transition-all duration-500',
          backgroundDecor.containerClass,
        )}
      >
        {backgroundDecor.overlay}
        <div
          className={clsx(
            'relative z-[1] flex h-full w-full items-center justify-center text-center',
            backgroundDecor.contentClass,
          )}
        >
          <span className="max-w-sm text-xs text-slate-200/80 sm:text-sm">
            Replay timeline will appear once executions capture timeline artifacts.
          </span>
        </div>
      </div>
    );
  }

  // All remaining code can safely use currentFrame (guaranteed non-null here)
  const screenshot = currentFrame.screenshot;
  const displayDurationMs = currentFrame.totalDurationMs ?? currentFrame.durationMs;
  const hasResiliencyInfo = Boolean(
    (displayDurationMs && currentFrame.durationMs && Math.round(displayDurationMs) !== Math.round(currentFrame.durationMs)) ||
      (typeof currentFrame.retryAttempt === 'number' && currentFrame.retryAttempt > 1) ||
      (typeof currentFrame.retryMaxAttempts === 'number' && currentFrame.retryMaxAttempts > 1) ||
      (typeof currentFrame.retryConfigured === 'number' && currentFrame.retryConfigured > 0) ||
      (typeof currentFrame.retryDelayMs === 'number' && currentFrame.retryDelayMs > 0) ||
      (typeof currentFrame.retryBackoffFactor === 'number' && currentFrame.retryBackoffFactor !== 1 && currentFrame.retryBackoffFactor !== 0) ||
      (currentFrame.retryHistory && currentFrame.retryHistory.length > 0)
  );
  const dimensions: Dimensions = {
    width: screenshot?.width || FALLBACK_DIMENSIONS.width,
    height: screenshot?.height || FALLBACK_DIMENSIONS.height,
  };
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

  const pointerStyle = toPointStyle(currentFrame.clickPosition, dimensions);
  const cursorTrailPoints = toTrailPoints(currentFrame.cursorTrail, dimensions);

  const changeFrame = (index: number) => {
    if (index < 0 || index >= normalizedFrames.length) {
      return;
    }
    setCurrentIndex(index);
    setFrameProgress(0);
  };

  const handleNext = () => {
    if (currentIndex >= normalizedFrames.length - 1) {
      if (loop) {
        changeFrame(0);
      }
      return;
    }
    changeFrame(currentIndex + 1);
  };

  const handlePrevious = () => {
    if (currentIndex === 0) {
      if (loop) {
        changeFrame(normalizedFrames.length - 1);
      }
      return;
    }
    changeFrame(currentIndex - 1);
  };

  return (
    <div className="flex flex-col gap-4">
      <div
        className={clsx(
          'relative overflow-hidden rounded-3xl transition-all duration-500',
          backgroundDecor.containerClass,
        )}
      >
        {backgroundDecor.overlay}
        <div className={clsx('relative z-[1]', backgroundDecor.contentClass)}>
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-200/80">
            <span>Replay</span>
            <span>
              Step {currentIndex + 1} / {normalizedFrames.length}
            </span>
          </div>

          <div className="mt-4 space-y-3">
            <div className={clsx('overflow-hidden rounded-2xl transition-all duration-300', chromeDecor.frameClass)}>
            {chromeDecor.header}

            <div className={clsx('relative overflow-hidden', chromeDecor.contentClass)}>
              <div
                className="relative"
                style={{
                  paddingTop: `${aspectRatio}%`,
                }}
              >
                <div
                  className="absolute inset-0"
                  style={{
                    transform: `scale(${zoom})`,
                    transformOrigin: anchorStyle,
                    transition: 'transform 600ms ease, transform-origin 600ms ease',
                  }}
                >
                  {screenshot?.url ? (
                    <img
                      src={screenshot.url}
                      alt={currentFrame.nodeId || `Step ${currentFrame.stepIndex + 1}`}
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center bg-slate-900 text-slate-500">
                      Screenshot unavailable
                    </div>
                  )}

                  <div className="absolute inset-0">
                    {overlayRegions(currentFrame.maskRegions, 'mask')}
                    {overlayRegions(currentFrame.highlightRegions, 'highlight')}

                    {currentFrame.focusedElement?.boundingBox && (
                      <div
                        className="absolute rounded-2xl border border-sky-400/70 bg-sky-400/10 shadow-[0_0_60px_rgba(56,189,248,0.35)]"
                        style={toRectStyle(currentFrame.focusedElement.boundingBox, dimensions)}
                      />
                    )}

                    {cursorTrailPoints.length >= 2 && (
                      <svg
                        className="absolute inset-0 h-full w-full"
                        viewBox="0 0 100 100"
                        preserveAspectRatio="none"
                      >
                        <polyline
                          points={cursorTrailPoints.map((point) => `${point.x},${point.y}`).join(' ')}
                          stroke="rgba(56,189,248,0.6)"
                          strokeWidth={1.8}
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          fill="none"
                        />
                      </svg>
                    )}

                    {pointerStyle && (
                      <div
                        className="absolute -translate-x-1/2 -translate-y-1/2 transition-all duration-500 ease-out"
                        style={{
                          ...pointerStyle,
                          transitionProperty: 'left, top',
                        }}
                      >
                        <span className="absolute h-10 w-10 animate-ping rounded-full border border-white/40" />
                        <span className="relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-white/90 bg-white/80 shadow-lg" />
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>

            <div className="flex items-center gap-4">
              <button
                className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10"
                onClick={handlePrevious}
                aria-label="Previous frame"
              >
              <ChevronLeft size={16} />
            </button>

            <button
              className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-flow-accent text-white hover:bg-blue-500"
              onClick={() => setIsPlaying((prev) => !prev)}
              aria-label={isPlaying ? 'Pause replay' : 'Play replay'}
            >
              {isPlaying ? <Pause size={16} /> : <Play size={16} />}
            </button>

            <button
              className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10"
              onClick={handleNext}
              aria-label="Next frame"
            >
              <ChevronRight size={16} />
            </button>

              <div className="flex-1">
                <div className="relative h-1 overflow-hidden rounded-full bg-white/10">
                  <div
                    className="absolute inset-y-0 left-0 rounded-full bg-flow-accent transition-[width] duration-75"
                    style={{ width: `${frameProgress * 100}%` }}
                  />
                </div>
              </div>

              <div className="text-xs uppercase tracking-[0.2em] text-slate-200/80">
                {displayDurationMs ? formatDurationSeconds(displayDurationMs) : 'Auto'}
              </div>
            </div>
          </div>
        </div>
      </div>

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
            className="inline-flex items-center gap-1 rounded-full border border-white/10 bg-white/5 px-2.5 py-1 text-[11px] uppercase tracking-[0.2em] text-slate-200 hover:bg-white/10"
            aria-expanded={!isMetadataCollapsed}
          >
            {isMetadataCollapsed ? <ChevronDown size={12} /> : <ChevronUp size={12} />}
            {isMetadataCollapsed ? 'Expand' : 'Collapse'}
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
                  {typeof currentFrame.retryConfigured === 'number' && currentFrame.retryConfigured > 0 && (
                    <div>Configured retries: {currentFrame.retryConfigured}</div>
                  )}
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

      {normalizedFrames.length > 1 && (
        <div className="rounded-3xl border border-white/10 bg-slate-950/70 p-3">
          <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">Storyboard</div>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
            {normalizedFrames.map((frame, index) => (
              <button
                key={frame.id}
                className={clsx(
                  'group relative overflow-hidden rounded-2xl border border-white/5 bg-white/5 text-left transition-all duration-200',
                  currentIndex === index
                    ? 'border-flow-accent shadow-[0_12px_40px_rgba(37,99,235,0.35)]'
                    : 'hover:border-flow-accent/60 hover:shadow-[0_10px_30px_rgba(59,130,246,0.18)]'
                )}
                onClick={() => changeFrame(index)}
              >
                <div className="relative" style={{ paddingTop: `${aspectRatio}%` }}>
                  <img
                    src={frame.screenshot?.thumbnailUrl || frame.screenshot?.url}
                    alt={`Timeline frame ${index + 1}`}
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                  <div className={clsx('absolute inset-0 bg-black/40 transition-opacity duration-200', {
                    'opacity-0': currentIndex === index,
                    'opacity-70 group-hover:opacity-40': currentIndex !== index,
                  })}
                  />
                </div>
                <div className="px-3 py-2 text-[11px] text-slate-300">
                  <div className="truncate font-medium text-slate-100">
                    {frame.nodeId || frame.stepType || `Step ${frame.stepIndex + 1}`}
                  </div>
                  <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                    {formatDurationSeconds(clampDuration(frame.durationMs))}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      {executionStatus === 'failed' && frames.length > 0 && (
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
