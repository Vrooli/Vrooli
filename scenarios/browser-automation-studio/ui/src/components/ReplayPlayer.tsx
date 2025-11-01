import { useEffect, useMemo, useRef, useState, type ReactNode, type CSSProperties } from 'react';
import clsx from 'clsx';
import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Pause, Play } from 'lucide-react';

// Unsplash assets (IDs: m_7p45JfXQo, Tn29N3Hpf2E, KfFmwa7m5VQ) licensed for free use
const ABSOLUTE_URL_PATTERN = /^[a-zA-Z][a-zA-Z\d+.-]*:/;

const withReplayBasePath = (value: string) => {
  if (!value || ABSOLUTE_URL_PATTERN.test(value)) {
    return value;
  }
  const base = import.meta.env.BASE_URL || '/';
  if (base === '/' || value.startsWith(base)) {
    return value;
  }
  const normalizedBase = base.endsWith('/') ? base : `${base}/`;
  const normalizedValue = value.startsWith('/') ? value.slice(1) : value;
  return `${normalizedBase}${normalizedValue}`;
};

const resolveReplayAsset = (relativePath: string) => {
  const url = new URL(relativePath, import.meta.url);
  return withReplayBasePath(url.pathname || url.href);
};

const geometricPrismUrl = resolveReplayAsset('../assets/replay-backgrounds/geometric-prism.jpg');
const geometricOrbitUrl = resolveReplayAsset('../assets/replay-backgrounds/geometric-orbit.jpg');
const geometricMosaicUrl = resolveReplayAsset('../assets/replay-backgrounds/geometric-mosaic.jpg');

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
  | 'none'
  | 'geoPrism'
  | 'geoOrbit'
  | 'geoMosaic';

export type ReplayCursorTheme =
  | 'disabled'
  | 'white'
  | 'black'
  | 'aura'
  | 'arrowLight'
  | 'arrowDark'
  | 'arrowNeon';
export type ReplayCursorInitialPosition =
  | 'center'
  | 'top-left'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-right'
  | 'random';

interface ReplayPlayerProps {
  frames: ReplayFrame[];
  autoPlay?: boolean;
  loop?: boolean;
  onFrameChange?: (frame: ReplayFrame, index: number) => void;
  executionStatus?: 'pending' | 'running' | 'completed' | 'failed';
  chromeTheme?: ReplayChromeTheme;
  backgroundTheme?: ReplayBackgroundTheme;
  cursorTheme?: ReplayCursorTheme;
  cursorInitialPosition?: ReplayCursorInitialPosition;
  cursorScale?: number;
}

const DEFAULT_DURATION = 1600;
const MIN_DURATION = 800;
const MAX_DURATION = 6000;
const MIN_CURSOR_SCALE = 0.6;
const MAX_CURSOR_SCALE = 1.8;

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

const pointsEqual = (a?: ReplayPoint | null, b?: ReplayPoint | null, tolerance = 0.5) => {
  if (!a || !b) {
    return false;
  }
  if (typeof a.x !== 'number' || typeof a.y !== 'number' || typeof b.x !== 'number' || typeof b.y !== 'number') {
    return false;
  }
  return Math.abs(a.x - b.x) <= tolerance && Math.abs(a.y - b.y) <= tolerance;
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
  containerStyle?: CSSProperties;
  contentClass: string;
  baseLayer?: ReactNode;
  overlay?: ReactNode;
};

const buildBackgroundDecor = (theme: ReplayBackgroundTheme): BackgroundDecor => {
  switch (theme) {
    case 'sunset':
      return {
        containerClass:
          'border border-rose-200/35 shadow-[0_26px_70px_rgba(236,72,153,0.42)] bg-slate-950',
        containerStyle: {
          backgroundImage:
            'linear-gradient(135deg, rgba(244,114,182,0.92) 0%, rgba(251,191,36,0.88) 100%)',
          backgroundColor: '#43112d',
        },
        contentClass: 'p-6 sm:p-7 backdrop-blur-[1.2px]',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <div
              className="absolute inset-[-20%]"
              style={{
                background:
                  'radial-gradient(95% 80% at 0% 5%, rgba(255,245,235,0.42), transparent 65%), radial-gradient(105% 85% at 100% 100%, rgba(254,215,170,0.32), transparent 70%)',
              }}
            />
            <div
              className="absolute inset-0 opacity-28 mix-blend-soft-light"
              style={{
                backgroundImage:
                  'repeating-linear-gradient(145deg, rgba(254,226,226,0.45) 0, rgba(254,226,226,0.45) 1px, transparent 1px, transparent 16px)',
              }}
            />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-45 mix-blend-screen"
            style={{
              background:
                'radial-gradient(120% 130% at 15% 10%, rgba(236,72,153,0.3), transparent 65%), radial-gradient(125% 120% at 85% 90%, rgba(251,191,36,0.28), transparent 60%)',
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
        contentClass: 'p-6 sm:p-7',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0">
            <div
              className="absolute inset-0 opacity-75"
              style={{
                background:
                  'radial-gradient(120% 140% at 12% 12%, rgba(56,189,248,0.22), transparent 65%), radial-gradient(120% 130% at 88% 88%, rgba(14,116,144,0.28), transparent 70%)',
              }}
            />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0 opacity-82 mix-blend-screen"
            style={{
              backgroundImage:
                'linear-gradient(rgba(148,163,184,0.42) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.42) 1px, transparent 1px)',
              backgroundSize: '28px 28px',
              backgroundPosition: 'center',
            }}
          />
        ),
      };
    case 'geoPrism':
      return {
        containerClass:
          'border border-cyan-200/35 bg-slate-950 shadow-[0_32px_90px_rgba(56,189,248,0.36)]',
        containerStyle: {
          backgroundColor: '#0f172a',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/35',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricPrismUrl} alt="" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-br from-cyan-400/28 via-transparent to-indigo-500/24 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/42" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'radial-gradient(140% 120% at 12% 12%, rgba(56,189,248,0.25), transparent 65%), radial-gradient(140% 130% at 88% 88%, rgba(129,140,248,0.24), transparent 60%)',
            }}
          />
        ),
      };
    case 'geoOrbit':
      return {
        containerClass:
          'border border-sky-200/35 bg-slate-950 shadow-[0_30px_88px_rgba(14,165,233,0.38)]',
        containerStyle: {
          backgroundColor: '#0b1120',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/40',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricOrbitUrl} alt="" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-br from-sky-300/22 via-transparent to-amber-300/22 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/45" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'radial-gradient(130% 120% at 18% 15%, rgba(94,234,212,0.18), transparent 65%), radial-gradient(120% 120% at 82% 85%, rgba(250,204,21,0.16), transparent 60%)',
            }}
          />
        ),
      };
    case 'geoMosaic':
      return {
        containerClass:
          'border border-amber-200/30 bg-slate-950 shadow-[0_28px_80px_rgba(245,158,11,0.32)]',
        containerStyle: {
          backgroundColor: '#0b1526',
        },
        contentClass: 'p-6 sm:p-7 bg-slate-950/38',
        baseLayer: (
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <img src={geometricMosaicUrl} alt="" className="h-full w-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-tr from-sky-400/20 via-transparent to-indigo-400/22 mix-blend-screen" />
            <div className="absolute inset-0 bg-slate-950/45" />
          </div>
        ),
        overlay: (
          <div
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'linear-gradient(160deg, rgba(14,165,233,0.18) 0%, rgba(99,102,241,0.24) 45%, transparent 78%)',
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

type CursorDecor = {
  wrapperClass?: string;
  wrapperStyle?: CSSProperties;
  showPing: boolean;
  pingClass?: string;
  pingStyle?: CSSProperties;
  renderBase: ReactNode | null;
  trailColor: string;
  trailWidth: number;
  offset?: { x: number; y: number };
  transformOrigin?: string;
};

const buildCursorDecor = (theme: ReplayCursorTheme): CursorDecor => {
  const buildHalo = (config: {
    wrapperClass?: string;
    wrapperStyle?: CSSProperties;
    pingClass: string;
    pingStyle?: CSSProperties;
    baseClass: string;
    baseStyle?: CSSProperties;
    dotClass?: string;
    dotStyle?: CSSProperties;
    showDot?: boolean;
    trailColor: string;
    trailWidth: number;
  }): CursorDecor => ({
    wrapperClass: config.wrapperClass,
    wrapperStyle: config.wrapperStyle,
    showPing: true,
    pingClass: config.pingClass,
    pingStyle: config.pingStyle,
    renderBase: (
      <span className={config.baseClass} style={config.baseStyle}>
        {config.showDot === false || !config.dotClass ? null : (
          <span className={config.dotClass} style={config.dotStyle} />
        )}
      </span>
    ),
    trailColor: config.trailColor,
    trailWidth: config.trailWidth,
  });

  switch (theme) {
    case 'black':
      return buildHalo({
        pingClass: 'absolute h-8 w-8 animate-ping rounded-full border border-black/40 bg-black/10',
        baseClass:
          'relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-black bg-slate-900 shadow-[0_10px_28px_rgba(15,23,42,0.55)]',
        dotClass: 'h-2 w-2 rounded-full bg-white/80',
        trailColor: 'rgba(30,41,59,0.7)',
        trailWidth: 2,
      });
    case 'aura':
      return buildHalo({
        wrapperClass: 'drop-shadow-[0_18px_45px_rgba(14,165,233,0.45)]',
        pingClass: 'absolute h-12 w-12 animate-ping rounded-full border-2 border-cyan-300/35',
        pingStyle: {
          background:
            'radial-gradient(circle, rgba(56,189,248,0.25) 0%, rgba(14,116,144,0.08) 60%, transparent 80%)',
        },
        baseClass:
          'relative z-10 inline-flex h-6 w-6 items-center justify-center rounded-full border-2 border-cyan-200/80 bg-gradient-to-br from-sky-400 via-emerald-300 to-violet-400 shadow-[0_14px_38px_rgba(56,189,248,0.45)]',
        showDot: false,
        trailColor: 'rgba(14,165,233,0.8)',
        trailWidth: 2.3,
      });
    case 'disabled':
      return {
        showPing: false,
        renderBase: null,
        trailColor: 'rgba(0,0,0,0)',
        trailWidth: 0,
      };
    case 'arrowLight':
      return {
        showPing: false,
        wrapperClass: 'drop-shadow-[0_18px_32px_rgba(15,23,42,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d="M6 3L6.2 21.6L11.4 16.4L14.2 25.6L17.2 24.6L14.6 15.6L24 14.4L6 3Z"
                fill="rgba(255,255,255,0.96)"
                stroke="rgba(15,23,42,0.85)"
                strokeWidth={1.4}
                strokeLinejoin="round"
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(148,163,184,0.55)',
        trailWidth: 1.1,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'arrowDark':
      return {
        showPing: false,
        wrapperClass: 'drop-shadow-[0_20px_40px_rgba(15,23,42,0.6)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d="M6.2 3L6.4 21.2L11 16.8L14 26.2L17.7 24.8L14.7 15.2L24.2 14.1L6.2 3Z"
                fill="rgba(30,41,59,0.95)"
                stroke="rgba(226,232,240,0.92)"
                strokeWidth={1.3}
                strokeLinejoin="round"
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(51,65,85,0.65)',
        trailWidth: 1.2,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'arrowNeon':
      return {
        showPing: true,
        wrapperClass: 'drop-shadow-[0_22px_52px_rgba(59,130,246,0.55)]',
        pingClass: 'absolute h-12 w-12 animate-ping rounded-full border-2 border-cyan-200/40',
        pingStyle: {
          background:
            'radial-gradient(circle at 50% 50%, rgba(56,189,248,0.25) 0%, rgba(56,189,248,0.08) 55%, transparent 80%)',
        },
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <defs>
                <linearGradient id="replay-cursor-neon" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stopColor="#38bdf8" />
                  <stop offset="45%" stopColor="#34d399" />
                  <stop offset="100%" stopColor="#a855f7" />
                </linearGradient>
              </defs>
              <path
                d="M6 3L6.2 21.6L11.4 16.4L14.5 26.1L18 24.9L15 15.4L24.4 14.2L6 3Z"
                fill="url(#replay-cursor-neon)"
                stroke="rgba(191,219,254,0.9)"
                strokeWidth={1.2}
                strokeLinejoin="round"
              />
            </svg>
            <span className="pointer-events-none absolute -inset-1 rounded-full bg-cyan-300/25 blur-lg" />
          </span>
        ),
        trailColor: 'rgba(56,189,248,0.7)',
        trailWidth: 1.4,
        offset: { x: 8, y: 10 },
        transformOrigin: '18% 12%',
      };
    case 'white':
    default:
      return buildHalo({
        pingClass: 'absolute h-10 w-10 animate-ping rounded-full border border-white/35',
        baseClass:
          'relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-white/90 bg-white/85 shadow-[0_12px_32px_rgba(148,163,184,0.45)]',
        dotClass: 'h-1.5 w-1.5 rounded-full bg-slate-500/60',
        trailColor: 'rgba(59,130,246,0.65)',
        trailWidth: 1.8,
      });
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
  cursorTheme = 'white',
  cursorInitialPosition = 'center',
  cursorScale = 1,
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
  const cursorSeedRef = useRef<{ x: number; y: number }>({ x: Math.random(), y: Math.random() });
  const cursorSourceRef = useRef<'data' | 'fallback'>('fallback');
  const [cursorPosition, setCursorPosition] = useState<ReplayPoint | undefined>(undefined);
  const backgroundDecor = useMemo(
    () => buildBackgroundDecor(backgroundTheme ?? 'aurora'),
    [backgroundTheme],
  );
  const effectiveCursorTheme = cursorTheme ?? 'white';
  const isCursorEnabled = effectiveCursorTheme !== 'disabled';
  const cursorDecor = useMemo(() => buildCursorDecor(effectiveCursorTheme), [effectiveCursorTheme]);
  const pointerScale =
    typeof cursorScale === 'number' && !Number.isNaN(cursorScale)
      ? Math.min(MAX_CURSOR_SCALE, Math.max(MIN_CURSOR_SCALE, cursorScale))
      : 1;
  const cursorTrailStrokeWidth = cursorDecor.trailWidth * pointerScale;

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

  useEffect(() => {
    if (!isCursorEnabled) {
      cursorSourceRef.current = 'fallback';
      setCursorPosition(undefined);
    }
  }, [isCursorEnabled]);

  useEffect(() => {
    if (cursorInitialPosition === 'random') {
      cursorSeedRef.current = { x: Math.random(), y: Math.random() };
      if (isCursorEnabled) {
        cursorSourceRef.current = 'fallback';
        setCursorPosition(undefined);
      }
    }
  }, [cursorInitialPosition, isCursorEnabled]);

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

  const dimensions: Dimensions = {
    width: currentFrame?.screenshot?.width || FALLBACK_DIMENSIONS.width,
    height: currentFrame?.screenshot?.height || FALLBACK_DIMENSIONS.height,
  };

  useEffect(() => {
    if (!isCursorEnabled) {
      return;
    }
    if (!currentFrame) {
      return;
    }

    const extractPointFromFrame = (): ReplayPoint | undefined => {
      if (currentFrame.cursorTrail && currentFrame.cursorTrail.length > 0) {
        for (let index = currentFrame.cursorTrail.length - 1; index >= 0; index -= 1) {
          const point = currentFrame.cursorTrail[index];
          if (typeof point?.x === 'number' && typeof point?.y === 'number') {
            return point;
          }
        }
      }
      const click = currentFrame.clickPosition;
      if (click && typeof click.x === 'number' && typeof click.y === 'number') {
        return click;
      }
      return undefined;
    };

    const framePoint = extractPointFromFrame();
    if (framePoint) {
      cursorSourceRef.current = 'data';
      setCursorPosition((prev) => (pointsEqual(prev, framePoint) ? prev : framePoint));
      return;
    }

    if (cursorSourceRef.current === 'data' && cursorPosition) {
      return;
    }

    const { width, height } = dimensions;
    if (!width || !height) {
      return;
    }

    const padX = Math.max(12, width * 0.08);
    const padY = Math.max(12, height * 0.08);
    const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max);

    const fallbackPoint = (() => {
      switch (cursorInitialPosition) {
        case 'top-left':
          return { x: padX, y: padY };
        case 'top-right':
          return { x: width - padX, y: padY };
        case 'bottom-left':
          return { x: padX, y: height - padY };
        case 'bottom-right':
          return { x: width - padX, y: height - padY };
        case 'random':
          return {
            x: clamp(width * cursorSeedRef.current.x, padX, width - padX),
            y: clamp(height * cursorSeedRef.current.y, padY, height - padY),
          };
        case 'center':
        default:
          return { x: width / 2, y: height / 2 };
      }
    })();

    if (!fallbackPoint) {
      return;
    }

    if (cursorSourceRef.current !== 'fallback' || !pointsEqual(cursorPosition, fallbackPoint)) {
      cursorSourceRef.current = 'fallback';
      setCursorPosition(fallbackPoint);
    }
  }, [
    currentFrame,
    cursorInitialPosition,
    cursorPosition,
    dimensions.height,
    dimensions.width,
    isCursorEnabled,
  ]);

  // Early return AFTER all hooks
  if (normalizedFrames.length === 0 || !currentFrame) {
    return (
      <div
        className={clsx(
          'relative flex h-64 items-center justify-center overflow-hidden rounded-3xl text-sm text-slate-200/80 transition-all duration-500',
          backgroundDecor.containerClass,
        )}
        style={backgroundDecor.containerStyle}
      >
        {backgroundDecor.baseLayer}
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

  const cursorTrailPoints = isCursorEnabled
    ? toTrailPoints(currentFrame.cursorTrail, dimensions)
    : [];
  const pointerStyle = isCursorEnabled && cursorPosition
    ? toPointStyle(cursorPosition, dimensions)
    : undefined;
  const hasTrail = cursorTrailPoints.length >= 2 && cursorTrailStrokeWidth > 0.05;
  const pointerOffsetX = cursorDecor.offset?.x ?? 0;
  const pointerOffsetY = cursorDecor.offset?.y ?? 0;
  const basePointerTransform = `translate(calc(-50% + ${pointerOffsetX}px), calc(-50% + ${pointerOffsetY}px))${
    pointerScale !== 1 ? ` scale(${pointerScale})` : ''
  }`;
  const pointerWrapperStyle =
    pointerStyle && isCursorEnabled
      ? {
          ...pointerStyle,
          ...cursorDecor.wrapperStyle,
          transform: cursorDecor.wrapperStyle?.transform
            ? `${basePointerTransform} ${cursorDecor.wrapperStyle.transform}`
            : basePointerTransform,
          transformOrigin: cursorDecor.transformOrigin ?? '50% 50%',
          transitionProperty: 'left, top, transform',
        }
      : undefined;

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
        style={backgroundDecor.containerStyle}
      >
        {backgroundDecor.baseLayer}
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

                    {hasTrail && (
                      <svg
                        className="absolute inset-0 h-full w-full"
                        viewBox="0 0 100 100"
                        preserveAspectRatio="none"
                      >
                        <polyline
                          points={cursorTrailPoints.map((point) => `${point.x},${point.y}`).join(' ')}
                          stroke={cursorDecor.trailColor}
                          strokeWidth={cursorTrailStrokeWidth}
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          fill="none"
                        />
                      </svg>
                    )}

                    {pointerWrapperStyle && cursorDecor.renderBase && (
                      <div
                        className={clsx(
                          'absolute pointer-events-none transition-all duration-500 ease-out',
                          cursorDecor.wrapperClass,
                        )}
                        style={pointerWrapperStyle}
                      >
                        {cursorDecor.showPing && cursorDecor.pingClass && (
                          <span
                            className={cursorDecor.pingClass}
                            style={cursorDecor.pingStyle}
                          />
                        )}
                        {cursorDecor.renderBase}
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
