import { useEffect, useMemo, useRef, useState, useCallback, type ReactNode, type CSSProperties, type HTMLAttributes } from 'react';
import clsx from 'clsx';
import { ChevronDown, ChevronLeft, ChevronRight, Pause, Play } from 'lucide-react';

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

const REPLAY_ARROW_CURSOR_PATH = 'M6 3L6 22L10.4 18.1L13.1 26.4L15.9 25.2L13.1 17.5L22 17.5L6 3Z';


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

interface NormalizedPoint {
  x: number;
  y: number;
}

export type CursorSpeedProfile = 'instant' | 'linear' | 'easeIn' | 'easeOut' | 'easeInOut';
export type CursorPathStyle = 'linear' | 'parabolicUp' | 'parabolicDown' | 'cubic' | 'pseudorandom';

interface CursorAnimationOverride {
  target?: NormalizedPoint;
  pathStyle?: CursorPathStyle;
  speedProfile?: CursorSpeedProfile;
  path?: ReplayPoint[];
}

type CursorOverrideMap = Record<string, CursorAnimationOverride>;

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
  | 'arrowNeon'
  | 'handNeutral'
  | 'handAura';
export type ReplayCursorInitialPosition =
  | 'center'
  | 'top-left'
  | 'top-right'
  | 'bottom-left'
  | 'bottom-right'
  | 'random';

export type ReplayCursorClickAnimation = 'none' | 'pulse' | 'ripple';

export interface ReplayPlayerController {
  seek: (options: { frameIndex: number; progress?: number }) => void;
  play: () => void;
  pause: () => void;
  getViewportElement: () => HTMLElement | null;
  getPresentationElement: () => HTMLElement | null;
  getFrameCount: () => number;
}

type ReplayPlayerPresentationMode = 'default' | 'export';

interface ReplayPlayerPresentationDimensions {
  width?: number;
  height?: number;
  deviceScaleFactor?: number;
}

interface ReplayPlayerProps {
  frames: ReplayFrame[];
  autoPlay?: boolean;
  loop?: boolean;
  onFrameChange?: (frame: ReplayFrame, index: number) => void;
  onFrameProgressChange?: (frameIndex: number, progress: number) => void;
  executionStatus?: 'pending' | 'running' | 'completed' | 'failed';
  chromeTheme?: ReplayChromeTheme;
  backgroundTheme?: ReplayBackgroundTheme;
  cursorTheme?: ReplayCursorTheme;
  cursorInitialPosition?: ReplayCursorInitialPosition;
  cursorScale?: number;
  cursorClickAnimation?: ReplayCursorClickAnimation;
  cursorDefaultSpeedProfile?: CursorSpeedProfile;
  cursorDefaultPathStyle?: CursorPathStyle;
  exposeController?: (controller: ReplayPlayerController | null) => void;
  presentationMode?: ReplayPlayerPresentationMode;
  allowPointerEditing?: boolean;
  presentationDimensions?: ReplayPlayerPresentationDimensions;
}

const DEFAULT_DURATION = 1600;
const MIN_DURATION = 800;
const MAX_DURATION = 6000;
const MIN_CURSOR_SCALE = 0.6;
const MAX_CURSOR_SCALE = 1.8;
const DEFAULT_SPEED_PROFILE: CursorSpeedProfile = 'linear';
const DEFAULT_PATH_STYLE: CursorPathStyle = 'linear';
const SPEED_PROFILE_OPTIONS: Array<{ id: CursorSpeedProfile; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Consistent motion between frames' },
  { id: 'easeIn', label: 'Ease in', description: 'Begin slowly, accelerate toward the target' },
  { id: 'easeOut', label: 'Ease out', description: 'Move quickly at first, settle into the target' },
  { id: 'easeInOut', label: 'Ease in/out', description: 'Smooth acceleration and deceleration' },
  { id: 'instant', label: 'Instant', description: 'Jump directly at the end of the step' },
];

const CURSOR_PATH_STYLE_OPTIONS: Array<{ id: CursorPathStyle; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Direct line between previous and current positions' },
  { id: 'parabolicUp', label: 'Parabolic (up)', description: 'Arcs upward before easing into the target' },
  { id: 'parabolicDown', label: 'Parabolic (down)', description: 'Arcs downward before easing into the target' },
  { id: 'cubic', label: 'Cubic', description: 'Smooth S-curve with a gentle overshoot' },
  { id: 'pseudorandom', label: 'Pseudorandom', description: 'Organic path generated from deterministic random waypoints' },
];

const FALLBACK_CLICK_COLOR: [string, string, string] = ['59', '130', '246'];

const parseRgbaComponents = (value: string | undefined): [string, string, string] | null => {
  if (!value) {
    return null;
  }
  const match = value.match(/rgba?\(([^)]+)\)/i);
  if (!match) {
    return null;
  }
  const parts = match[1]
    .split(',')
    .map((part) => part.trim())
    .filter((part) => part.length > 0);
  if (parts.length < 3) {
    return null;
  }
  return [parts[0], parts[1], parts[2]];
};

const rgbaWithAlpha = (components: [string, string, string] | null, alpha: number) => {
  const [r, g, b] = components ?? FALLBACK_CLICK_COLOR;
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
};

const clamp01 = (value: number) => Math.min(1, Math.max(0, value));

const clampNormalizedPoint = (point: NormalizedPoint): NormalizedPoint => ({
  x: clamp01(point.x),
  y: clamp01(point.y),
});

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

const toNormalizedPoint = (
  point: ReplayPoint | null | undefined,
  dims: Dimensions,
): NormalizedPoint | undefined => {
  if (!point || typeof point.x !== 'number' || typeof point.y !== 'number') {
    return undefined;
  }
  const width = dims.width || FALLBACK_DIMENSIONS.width;
  const height = dims.height || FALLBACK_DIMENSIONS.height;
  if (!width || !height) {
    return undefined;
  }
  return clampNormalizedPoint({ x: point.x / width, y: point.y / height });
};

const toAbsolutePoint = (point: NormalizedPoint, dims: Dimensions): ReplayPoint => {
  const width = dims.width || FALLBACK_DIMENSIONS.width;
  const height = dims.height || FALLBACK_DIMENSIONS.height;
  return {
    x: clamp01(point.x) * width,
    y: clamp01(point.y) * height,
  };
};

const applySpeedProfile = (progress: number, profile: CursorSpeedProfile) => {
  const clamped = clamp01(progress);
  switch (profile) {
    case 'instant':
      return clamped >= 0.999 ? 1 : 0;
    case 'easeIn':
      return clamped * clamped;
    case 'easeOut':
      return 1 - (1 - clamped) * (1 - clamped);
    case 'easeInOut':
      if (clamped < 0.5) {
        return 2 * clamped * clamped;
      }
      return 1 - Math.pow(-2 * clamped + 2, 2) / 2;
    case 'linear':
    default:
      return clamped;
  }
};

const interpolatePath = (points: ReplayPoint[], progress: number): ReplayPoint => {
  if (points.length === 0) {
    return { x: 0, y: 0 };
  }
  if (points.length === 1) {
    return points[0];
  }

  let totalDistance = 0;
  const segments = [] as Array<{ start: ReplayPoint; end: ReplayPoint; length: number; cumulative: number }>;
  for (let index = 0; index < points.length - 1; index += 1) {
    const start = points[index];
    const end = points[index + 1];
    if (typeof start.x !== 'number' || typeof start.y !== 'number' || typeof end.x !== 'number' || typeof end.y !== 'number') {
      continue;
    }
    const length = Math.hypot(end.x - start.x, end.y - start.y);
    if (length <= 0.0001) {
      continue;
    }
    totalDistance += length;
    segments.push({ start, end, length, cumulative: totalDistance });
  }

  if (segments.length === 0 || totalDistance <= 0) {
    return points[points.length - 1];
  }

  const targetDistance = totalDistance * clamp01(progress);
  for (const segment of segments) {
    const prevCumulative = segment.cumulative - segment.length;
    if (targetDistance <= segment.cumulative) {
      const segmentProgress = segment.length > 0 ? (targetDistance - prevCumulative) / segment.length : 1;
      return {
        x: (segment.start.x ?? 0) + (segment.end.x ?? 0 - (segment.start.x ?? 0)) * segmentProgress,
        y: (segment.start.y ?? 0) + (segment.end.y ?? 0 - (segment.start.y ?? 0)) * segmentProgress,
      };
    }
  }

  return segments[segments.length - 1].end;
};

const formatCoordinate = (point?: ReplayPoint | null, dims?: Dimensions) => {
  if (!point || typeof point.x !== 'number' || typeof point.y !== 'number' || !dims?.width || !dims?.height) {
    return 'n/a';
  }
  const percentX = (point.x / dims.width) * 100;
  const percentY = (point.y / dims.height) * 100;
  const formatPercent = (value: number) => `${Math.round(value * 10) / 10}%`;
  return `${Math.round(point.x)}px, ${Math.round(point.y)}px (${formatPercent(percentX)}, ${formatPercent(percentY)})`;
};

const isSamePoint = (a: NormalizedPoint, b: NormalizedPoint, epsilon = 1e-4) =>
  Math.abs(a.x - b.x) < epsilon && Math.abs(a.y - b.y) < epsilon;

const createDeterministicRng = (seed: string) => {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i += 1) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return () => {
    // Xorshift32
    h ^= h << 13;
    h ^= h >>> 17;
    h ^= h << 5;
    return ((h >>> 0) % 1_000_000) / 1_000_000;
  };
};

const sampleQuadraticBezier = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  segments = 20,
): NormalizedPoint[] => {
  const points: NormalizedPoint[] = [];
  for (let i = 1; i < segments; i += 1) {
    const t = i / segments;
    const invT = 1 - t;
    const x = invT * invT * p0.x + 2 * invT * t * p1.x + t * t * p2.x;
    const y = invT * invT * p0.y + 2 * invT * t * p1.y + t * t * p2.y;
    points.push(clampNormalizedPoint({ x, y }));
  }
  return points;
};

const sampleCubicBezier = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  p3: NormalizedPoint,
  segments = 28,
): NormalizedPoint[] => {
  const points: NormalizedPoint[] = [];
  for (let i = 1; i < segments; i += 1) {
    const t = i / segments;
    const invT = 1 - t;
    const x =
      invT * invT * invT * p0.x +
      3 * invT * invT * t * p1.x +
      3 * invT * t * t * p2.x +
      t * t * t * p3.x;
    const y =
      invT * invT * invT * p0.y +
      3 * invT * invT * t * p1.y +
      3 * invT * t * t * p2.y +
      t * t * t * p3.y;
    points.push(clampNormalizedPoint({ x, y }));
  }
  return points;
};

const catmullRom = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  p3: NormalizedPoint,
  t: number,
): NormalizedPoint => {
  const t2 = t * t;
  const t3 = t2 * t;
  const x =
    0.5 *
    ((2 * p1.x) +
      (-p0.x + p2.x) * t +
      (2 * p0.x - 5 * p1.x + 4 * p2.x - p3.x) * t2 +
      (-p0.x + 3 * p1.x - 3 * p2.x + p3.x) * t3);
  const y =
    0.5 *
    ((2 * p1.y) +
      (-p0.y + p2.y) * t +
      (2 * p0.y - 5 * p1.y + 4 * p2.y - p3.y) * t2 +
      (-p0.y + 3 * p1.y - 3 * p2.y + p3.y) * t3);
  return { x, y };
};

const generateCatmullRomPath = (
  controlPoints: NormalizedPoint[],
  samplesPerSegment = 6,
): NormalizedPoint[] => {
  if (controlPoints.length <= 2) {
    return [];
  }
  const points: NormalizedPoint[] = [];
  for (let i = 0; i < controlPoints.length - 1; i += 1) {
    const p0 = controlPoints[i - 1] ?? controlPoints[i];
    const p1 = controlPoints[i];
    const p2 = controlPoints[i + 1];
    const p3 = controlPoints[i + 2] ?? controlPoints[i + 1];
    for (let step = 1; step <= samplesPerSegment; step += 1) {
      const t = step / (samplesPerSegment + 1);
      points.push(clampNormalizedPoint(catmullRom(p0, p1, p2, p3, t)));
    }
  }
  const first = controlPoints[0];
  const last = controlPoints[controlPoints.length - 1];
  return points.filter((point) => !isSamePoint(point, first) && !isSamePoint(point, last));
};

const generateStylizedPath = (
  style: CursorPathStyle,
  start: NormalizedPoint,
  target: NormalizedPoint,
  frameId: string,
): NormalizedPoint[] => {
  if (isSamePoint(start, target)) {
    return [];
  }

  const dx = target.x - start.x;
  const dy = target.y - start.y;
  const distance = Math.hypot(dx, dy);

  if (style === 'linear') {
    return [];
  }

  if (style === 'parabolicUp' || style === 'parabolicDown') {
    const mid = { x: (start.x + target.x) / 2, y: (start.y + target.y) / 2 };
    const curveStrength = Math.min(0.45, Math.max(0.08, distance * 0.45));
    const offsetY = style === 'parabolicUp' ? -curveStrength : curveStrength;
    const control = clampNormalizedPoint({ x: mid.x, y: mid.y + offsetY });
    return sampleQuadraticBezier(start, control, target);
  }

  if (style === 'cubic') {
    const magnitude = Math.min(0.45, distance * 0.35 + 0.08);
    const perpLength = Math.hypot(-dy, dx) || 1;
    const perp = { x: (-dy) / perpLength, y: dx / perpLength };
    const control1 = clampNormalizedPoint({
      x: start.x + dx * (1 / 3) + perp.x * magnitude,
      y: start.y + dy * (1 / 3) + perp.y * magnitude,
    });
    const control2 = clampNormalizedPoint({
      x: start.x + dx * (2 / 3) - perp.x * magnitude,
      y: start.y + dy * (2 / 3) - perp.y * magnitude,
    });
    return sampleCubicBezier(start, control1, control2, target);
  }

  const rng = createDeterministicRng(`${frameId}:${start.x.toFixed(4)}:${start.y.toFixed(4)}:${target.x.toFixed(4)}:${target.y.toFixed(4)}`);
  const waypointCount = 3;
  const controlPoints: NormalizedPoint[] = [start];
  for (let i = 1; i <= waypointCount; i += 1) {
    const t = i / (waypointCount + 1);
    const baseX = start.x + dx * t;
    const baseY = start.y + dy * t;
    const jitterRadius = Math.min(0.4, distance * 0.5 + 0.05);
    const angle = (rng() - 0.5) * Math.PI * 1.4;
    const offsetMagnitude = jitterRadius * (0.35 + rng() * 0.65);
    let candidate = clampNormalizedPoint({
      x: baseX + Math.cos(angle) * offsetMagnitude,
      y: baseY + Math.sin(angle) * offsetMagnitude,
    });
    const prev = controlPoints[controlPoints.length - 1];
    const prevDist = Math.hypot(prev.x - target.x, prev.y - target.y);
    const candidateDist = Math.hypot(candidate.x - target.x, candidate.y - target.y);
    if (candidateDist > prevDist) {
      const blend = 0.6;
      candidate = clampNormalizedPoint({
        x: candidate.x * blend + baseX * (1 - blend),
        y: candidate.y * blend + baseY * (1 - blend),
      });
    }
    controlPoints.push(candidate);
  }
  controlPoints.push(target);

  return generateCatmullRomPath(controlPoints, 6);
};

const normalizeOverride = (
  override: CursorAnimationOverride | null | undefined,
  defaultPathStyle: CursorPathStyle = DEFAULT_PATH_STYLE,
  defaultSpeedProfile: CursorSpeedProfile = DEFAULT_SPEED_PROFILE,
): CursorAnimationOverride | undefined => {
  if (!override) {
    return undefined;
  }
  const next: CursorAnimationOverride = {};
  if (override.target) {
    next.target = clampNormalizedPoint(override.target);
  }
  if (override.pathStyle && override.pathStyle !== defaultPathStyle) {
    next.pathStyle = override.pathStyle;
  }
  if (override.speedProfile && override.speedProfile !== defaultSpeedProfile) {
    next.speedProfile = override.speedProfile;
  }
  if (next.pathStyle || next.target || next.speedProfile) {
    return next;
  }
  return undefined;
};

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

interface CursorPlan {
  frameId: string;
  dims: Dimensions;
  startNormalized: NormalizedPoint;
  targetNormalized: NormalizedPoint;
  pathNormalized: NormalizedPoint[];
  speedProfile: CursorSpeedProfile;
  pathStyle: CursorPathStyle;
  hasRecordedTrail: boolean;
  previousTargetNormalized?: NormalizedPoint;
}

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
        baseClass:
          'relative z-10 inline-flex h-5 w-5 items-center justify-center rounded-full border-2 border-black bg-slate-900 shadow-[0_10px_28px_rgba(15,23,42,0.55)]',
        dotClass: 'h-2 w-2 rounded-full bg-white/80',
        trailColor: 'rgba(30,41,59,0.7)',
        trailWidth: 2,
      });
    case 'aura':
      return buildHalo({
        wrapperClass: 'drop-shadow-[0_18px_45px_rgba(14,165,233,0.45)]',
        baseClass:
          'relative z-10 inline-flex h-6 w-6 items-center justify-center rounded-full border-2 border-cyan-200/80 bg-gradient-to-br from-sky-400 via-emerald-300 to-violet-400 shadow-[0_14px_38px_rgba(56,189,248,0.45)]',
        showDot: false,
        trailColor: 'rgba(14,165,233,0.8)',
        trailWidth: 2.3,
      });
    case 'disabled':
      return {
        renderBase: null,
        trailColor: 'rgba(0,0,0,0)',
        trailWidth: 0,
      };
    case 'arrowLight':
      return {
        wrapperClass: 'drop-shadow-[0_18px_32px_rgba(15,23,42,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d={REPLAY_ARROW_CURSOR_PATH}
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
        wrapperClass: 'drop-shadow-[0_20px_40px_rgba(15,23,42,0.6)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center text-white">
            <svg viewBox="0 0 32 32" className="h-8 w-8">
              <path
                d={REPLAY_ARROW_CURSOR_PATH}
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
        wrapperClass: 'drop-shadow-[0_22px_52px_rgba(59,130,246,0.55)]',
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
                d={REPLAY_ARROW_CURSOR_PATH}
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
    case 'handAura':
      return {
        wrapperClass: 'drop-shadow-[0_20px_48px_rgba(56,189,248,0.4)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center">
            <svg viewBox="0 0 24 24" className="h-8 w-8" fill="none">
              <defs>
                <linearGradient id="replay-hand-gradient" x1="10%" y1="10%" x2="80%" y2="80%">
                  <stop offset="0%" stopColor="#38bdf8" />
                  <stop offset="55%" stopColor="#34d399" />
                  <stop offset="100%" stopColor="#a855f7" />
                </linearGradient>
              </defs>
              <g stroke="url(#replay-hand-gradient)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 14a8 8 0 0 1-8 8" />
                <path d="M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                <path d="M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1" />
                <path d="M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10" />
                <path d="M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
              </g>
              <circle
                cx={18.5}
                cy={12.5}
                r={3.8}
                fill="url(#replay-hand-gradient)"
                opacity={0.12}
              />
            </svg>
          </span>
        ),
        trailColor: 'rgba(56,189,248,0.72)',
        trailWidth: 1.3,
        offset: { x: 6, y: 12 },
        transformOrigin: '46% 18%',
      };
    case 'handNeutral':
      return {
        wrapperClass: 'drop-shadow-[0_16px_36px_rgba(15,23,42,0.55)]',
        renderBase: (
          <span className="relative inline-flex h-8 w-8 items-center justify-center">
            <svg viewBox="0 0 24 24" className="h-8 w-8" fill="none">
              <g stroke="rgba(241,245,249,0.92)" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 14a8 8 0 0 1-8 8" />
                <path d="M18 11v-1a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v0" />
                <path d="M14 10V9a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v1" />
                <path d="M10 9.5V4a2 2 0 0 0-2-2v0a2 2 0 0 0-2 2v10" />
                <path d="M18 11a2 2 0 1 1 4 0v3a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15" />
              </g>
            </svg>
          </span>
        ),
        trailColor: 'rgba(148,163,184,0.75)',
        trailWidth: 1.2,
        offset: { x: 6, y: 12 },
        transformOrigin: '46% 18%',
      };
    case 'white':
    default:
      return buildHalo({
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
      setIsPlaying(false);
      return;
    }
    setCurrentIndex(0);
    setIsPlaying(autoPlay && normalizedFrames.length > 1);
  }, [autoPlay, normalizedFrames.length, isExternallyControlled]);

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
  }, [isPlaying, currentIndex, normalizedFrames, loop, isExternallyControlled]);

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
        transform: cursorDecor.wrapperStyle?.transform
          ? `${basePointerTransform} ${cursorDecor.wrapperStyle.transform}`
          : basePointerTransform,
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
                </div>
              </div>
            </div>
          </div>

            {showInterfaceChrome && (
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
      )}

      {showInterfaceChrome && normalizedFrames.length > 1 && (
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
