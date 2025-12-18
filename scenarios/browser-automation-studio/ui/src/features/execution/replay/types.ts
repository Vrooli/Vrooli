/**
 * Type definitions for the ReplayPlayer component
 *
 * Contains all interfaces and type aliases used by the replay system:
 * - Frame and screenshot data structures
 * - Cursor animation types
 * - Theme configuration types
 * - Component props and controller interfaces
 */

import type { CSSProperties, ReactNode } from 'react';
import type { WatermarkSettings, IntroCardSettings, OutroCardSettings } from '@stores/settingsStore';

// =============================================================================
// Geometry Types
// =============================================================================

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

export interface NormalizedPoint {
  x: number;
  y: number;
}

export type Dimensions = { width: number; height: number };

// =============================================================================
// Cursor Animation Types
// =============================================================================

export type CursorSpeedProfile = 'instant' | 'linear' | 'easeIn' | 'easeOut' | 'easeInOut';
export type CursorPathStyle = 'linear' | 'parabolicUp' | 'parabolicDown' | 'cubic' | 'pseudorandom';

export interface CursorAnimationOverride {
  target?: NormalizedPoint;
  pathStyle?: CursorPathStyle;
  speedProfile?: CursorSpeedProfile;
  path?: ReplayPoint[];
}

export type CursorOverrideMap = Record<string, CursorAnimationOverride>;

export interface CursorPlan {
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

// =============================================================================
// Screenshot & Frame Types
// =============================================================================

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
  retryConfigured?: number | boolean;
  retryDelayMs?: number;
  retryBackoffFactor?: number;
  retryHistory?: ReplayRetryHistoryEntry[];
  domSnapshotHtml?: string;
  domSnapshotPreview?: string;
  domSnapshotArtifactId?: string;
}

// =============================================================================
// Theme Types
// =============================================================================

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

// =============================================================================
// Theme Decor Types (for theme builder return values)
// =============================================================================

export type BackgroundDecor = {
  containerClass: string;
  containerStyle?: CSSProperties;
  contentClass: string;
  baseLayer?: ReactNode;
  overlay?: ReactNode;
};

export type CursorDecor = {
  wrapperClass?: string;
  wrapperStyle?: CSSProperties;
  renderBase: ReactNode | null;
  trailColor: string;
  trailWidth: number;
  offset?: { x: number; y: number };
  transformOrigin?: string;
};

export type ChromeDecor = {
  frameClass: string;
  contentClass?: string;
  header: ReactNode | null;
};

// =============================================================================
// Component Props & Controller
// =============================================================================

export interface ReplayPlayerController {
  seek: (options: { frameIndex: number; progress?: number }) => void;
  play: () => void;
  pause: () => void;
  getViewportElement: () => HTMLElement | null;
  getPresentationElement: () => HTMLElement | null;
  getFrameCount: () => number;
}

export type ReplayPlayerPresentationMode = 'default' | 'export';

export interface ReplayPlayerPresentationDimensions {
  width?: number;
  height?: number;
  deviceScaleFactor?: number;
}

export interface ReplayPlayerProps {
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
  watermark?: WatermarkSettings;
  introCard?: IntroCardSettings;
  outroCard?: OutroCardSettings;
}

// =============================================================================
// Playback State Types
// =============================================================================

export type PlaybackPhase = 'intro' | 'frames' | 'outro';

// =============================================================================
// Option Types (for UI dropdowns)
// =============================================================================

export interface SpeedProfileOption {
  id: CursorSpeedProfile;
  label: string;
  description: string;
}

export interface CursorPathStyleOption {
  id: CursorPathStyle;
  label: string;
  description: string;
}
